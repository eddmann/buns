package proxy

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"strconv"
	"sync"
)

// SOCKS5 protocol constants
const (
	socks5Version = 0x05

	// Authentication methods
	authNone = 0x00

	// Commands
	cmdConnect = 0x01

	// Address types
	atypIPv4   = 0x01
	atypDomain = 0x03
	atypIPv6   = 0x04

	// Reply codes
	repSuccess         = 0x00
	repNotAllowed      = 0x02
	repHostUnreach     = 0x04
	repCmdNotSupported = 0x07
	repAddrNotSupp     = 0x08
)

// SOCKS5Proxy is a filtering SOCKS5 proxy server
type SOCKS5Proxy struct {
	listener net.Listener
	filter   *DomainFilter
	addr     string
	wg       sync.WaitGroup
	quit     chan struct{}
}

// NewSOCKS5Proxy creates a new SOCKS5 proxy server with domain filtering
func NewSOCKS5Proxy(filter *DomainFilter) (*SOCKS5Proxy, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("failed to create listener: %w", err)
	}

	return &SOCKS5Proxy{
		listener: listener,
		filter:   filter,
		addr:     listener.Addr().String(),
		quit:     make(chan struct{}),
	}, nil
}

// Addr returns the proxy's address (host:port)
func (p *SOCKS5Proxy) Addr() string {
	return p.addr
}

// Port returns just the port number
func (p *SOCKS5Proxy) Port() int {
	_, port, _ := net.SplitHostPort(p.addr)
	portNum, _ := strconv.Atoi(port)
	return portNum
}

// Start begins accepting connections
func (p *SOCKS5Proxy) Start() error {
	go p.acceptLoop()
	return nil
}

// Stop gracefully shuts down the proxy
func (p *SOCKS5Proxy) Stop() error {
	close(p.quit)
	err := p.listener.Close()
	p.wg.Wait()
	return err
}

func (p *SOCKS5Proxy) acceptLoop() {
	for {
		conn, err := p.listener.Accept()
		if err != nil {
			select {
			case <-p.quit:
				return
			default:
				continue
			}
		}

		p.wg.Add(1)
		go func() {
			defer p.wg.Done()
			p.handleConnection(conn)
		}()
	}
}

func (p *SOCKS5Proxy) handleConnection(conn net.Conn) {
	defer func() { _ = conn.Close() }()

	// Read version and auth methods
	header := make([]byte, 2)
	if _, err := io.ReadFull(conn, header); err != nil {
		return
	}

	if header[0] != socks5Version {
		return
	}

	// Read auth methods
	numMethods := int(header[1])
	methods := make([]byte, numMethods)
	if _, err := io.ReadFull(conn, methods); err != nil {
		return
	}

	// Accept no-auth only
	hasNoAuth := false
	for _, m := range methods {
		if m == authNone {
			hasNoAuth = true
			break
		}
	}

	if !hasNoAuth {
		_, _ = conn.Write([]byte{socks5Version, 0xFF}) // No acceptable methods
		return
	}

	// Send auth selection
	_, _ = conn.Write([]byte{socks5Version, authNone})

	// Read request
	request := make([]byte, 4)
	if _, err := io.ReadFull(conn, request); err != nil {
		return
	}

	if request[0] != socks5Version {
		return
	}

	// Only support CONNECT
	if request[1] != cmdConnect {
		p.sendReply(conn, repCmdNotSupported, nil)
		return
	}

	// Parse address
	host, port, err := p.readAddress(conn, request[3])
	if err != nil {
		p.sendReply(conn, repAddrNotSupp, nil)
		return
	}

	// Check domain filter
	if !p.filter.IsAllowed(host) {
		p.sendReply(conn, repNotAllowed, nil)
		return
	}

	// Connect to target - use net.JoinHostPort for IPv6 safety
	target := net.JoinHostPort(host, strconv.Itoa(int(port)))
	targetConn, err := net.Dial("tcp", target)
	if err != nil {
		p.sendReply(conn, repHostUnreach, nil)
		return
	}
	defer func() { _ = targetConn.Close() }()

	// Send success reply with bound address
	localAddr := targetConn.LocalAddr().(*net.TCPAddr)
	p.sendReply(conn, repSuccess, localAddr)

	// Tunnel data
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		_, _ = io.Copy(targetConn, conn)
	}()

	go func() {
		defer wg.Done()
		_, _ = io.Copy(conn, targetConn)
	}()

	wg.Wait()
}

func (p *SOCKS5Proxy) readAddress(conn net.Conn, addrType byte) (string, uint16, error) {
	var host string

	switch addrType {
	case atypIPv4:
		addr := make([]byte, 4)
		if _, err := io.ReadFull(conn, addr); err != nil {
			return "", 0, err
		}
		host = net.IP(addr).String()

	case atypDomain:
		lenByte := make([]byte, 1)
		if _, err := io.ReadFull(conn, lenByte); err != nil {
			return "", 0, err
		}
		domain := make([]byte, lenByte[0])
		if _, err := io.ReadFull(conn, domain); err != nil {
			return "", 0, err
		}
		host = string(domain)

	case atypIPv6:
		addr := make([]byte, 16)
		if _, err := io.ReadFull(conn, addr); err != nil {
			return "", 0, err
		}
		host = net.IP(addr).String()

	default:
		return "", 0, fmt.Errorf("unsupported address type: %d", addrType)
	}

	// Read port
	portBytes := make([]byte, 2)
	if _, err := io.ReadFull(conn, portBytes); err != nil {
		return "", 0, err
	}
	port := binary.BigEndian.Uint16(portBytes)

	return host, port, nil
}

func (p *SOCKS5Proxy) sendReply(conn net.Conn, rep byte, addr *net.TCPAddr) {
	reply := []byte{socks5Version, rep, 0x00}

	if addr == nil {
		// Send null address
		reply = append(reply, atypIPv4, 0, 0, 0, 0, 0, 0)
	} else if ip4 := addr.IP.To4(); ip4 != nil {
		reply = append(reply, atypIPv4)
		reply = append(reply, ip4...)
		reply = append(reply, byte(addr.Port>>8), byte(addr.Port))
	} else {
		reply = append(reply, atypIPv6)
		reply = append(reply, addr.IP.To16()...)
		reply = append(reply, byte(addr.Port>>8), byte(addr.Port))
	}

	_, _ = conn.Write(reply)
}

package proxy

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

// HTTPProxy is a filtering HTTP/HTTPS proxy server.
type HTTPProxy struct {
	listener net.Listener
	server   *http.Server
	filter   *DomainFilter
	addr     string
	wg       sync.WaitGroup
}

// NewHTTPProxy creates a new HTTP proxy server with domain filtering.
func NewHTTPProxy(filter *DomainFilter) (*HTTPProxy, error) {
	// Listen on random available port on localhost
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("failed to create listener: %w", err)
	}

	return NewHTTPProxyWithListener(listener, filter), nil
}

// NewHTTPProxyWithListener creates an HTTP proxy using an existing listener.
// If listener is nil, call StartUnix to create a Unix socket listener.
func NewHTTPProxyWithListener(listener net.Listener, filter *DomainFilter) *HTTPProxy {
	p := &HTTPProxy{
		filter: filter,
	}

	if listener != nil {
		p.listener = listener
		p.addr = listener.Addr().String()
	}

	p.server = &http.Server{
		Handler: http.HandlerFunc(p.handleRequest),
	}

	return p
}

// Addr returns the proxy's address (host:port).
func (p *HTTPProxy) Addr() string {
	return p.addr
}

// Port returns just the port number.
func (p *HTTPProxy) Port() int {
	_, port, _ := net.SplitHostPort(p.addr)
	portNum, _ := strconv.Atoi(port)
	return portNum
}

// Start begins accepting connections.
func (p *HTTPProxy) Start() error {
	p.wg.Add(1)
	go func() {
		defer p.wg.Done()
		_ = p.server.Serve(p.listener)
	}()
	return nil
}

// Stop gracefully shuts down the proxy.
func (p *HTTPProxy) Stop() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := p.server.Shutdown(ctx)
	p.wg.Wait()
	return err
}

// StartUnix starts the proxy on a Unix socket.
func (p *HTTPProxy) StartUnix(socketPath string) error {
	// Remove existing socket if present
	_ = os.Remove(socketPath)

	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		return fmt.Errorf("failed to create Unix socket: %w", err)
	}

	// Set restrictive permissions
	if err := os.Chmod(socketPath, 0600); err != nil {
		_ = listener.Close()
		return fmt.Errorf("failed to set socket permissions: %w", err)
	}

	p.listener = listener
	p.addr = socketPath

	return p.Start()
}

// handleRequest routes incoming proxy requests.
func (p *HTTPProxy) handleRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodConnect {
		p.handleConnect(w, r)
	} else {
		p.handleHTTP(w, r)
	}
}

// handleConnect handles HTTPS CONNECT requests (tunneling).
func (p *HTTPProxy) handleConnect(w http.ResponseWriter, r *http.Request) {
	host := r.Host

	// Check domain filter
	if !p.filter.IsAllowed(host) {
		http.Error(w, fmt.Sprintf("Access to %s is not allowed by sandbox policy", host), http.StatusForbidden)
		return
	}

	// Ensure host has a port
	if !strings.Contains(host, ":") {
		host = host + ":443"
	}

	// Connect to target with timeout
	targetConn, err := net.DialTimeout("tcp", host, 10*time.Second)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}

	// Hijack the connection
	hijacker, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "Hijacking not supported", http.StatusInternalServerError)
		_ = targetConn.Close()
		return
	}

	clientConn, _, err := hijacker.Hijack()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		_ = targetConn.Close()
		return
	}

	// Send success response
	_, _ = clientConn.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))

	// Tunnel data bidirectionally
	go func() {
		_, _ = io.Copy(targetConn, clientConn)
		_ = targetConn.Close()
	}()
	go func() {
		_, _ = io.Copy(clientConn, targetConn)
		_ = clientConn.Close()
	}()
}

// handleHTTP handles regular HTTP proxy requests.
func (p *HTTPProxy) handleHTTP(w http.ResponseWriter, r *http.Request) {
	host := r.Host
	if host == "" {
		host = r.URL.Host
	}

	// Check domain filter
	if !p.filter.IsAllowed(host) {
		http.Error(w, fmt.Sprintf("Access to %s is not allowed by sandbox policy", host), http.StatusForbidden)
		return
	}

	// Create outgoing request
	outReq := &http.Request{
		Method: r.Method,
		URL:    r.URL,
		Header: r.Header.Clone(),
		Body:   r.Body,
	}

	// Remove hop-by-hop headers
	outReq.Header.Del("Proxy-Connection")
	outReq.Header.Del("Proxy-Authenticate")
	outReq.Header.Del("Proxy-Authorization")

	// Make request with timeout
	client := &http.Client{
		Timeout: 60 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			// Don't follow redirects, let the client handle them
			return http.ErrUseLastResponse
		},
	}

	resp, err := client.Do(outReq)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	defer func() { _ = resp.Body.Close() }()

	// Copy response headers
	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}

	// Write status and body
	w.WriteHeader(resp.StatusCode)
	_, _ = io.Copy(w, resp.Body)
}

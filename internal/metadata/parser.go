package metadata

import (
	"bufio"
	"bytes"
	"strings"

	"github.com/BurntSushi/toml"
)

// Metadata represents the parsed // buns block from a script
type Metadata struct {
	Bun      string   `toml:"bun"`
	Packages []string `toml:"packages"`
}

// Parse extracts metadata from a script's // buns comment block
func Parse(content []byte) (*Metadata, error) {
	scanner := bufio.NewScanner(bytes.NewReader(content))
	var tomlLines []string
	inBlock := false

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		// Look for the // buns marker
		if !inBlock {
			if trimmed == "// buns" {
				inBlock = true
			}
			continue
		}

		// Inside the block - collect lines starting with //
		if strings.HasPrefix(trimmed, "//") {
			// Strip the // prefix and any single space after it
			content := strings.TrimPrefix(trimmed, "//")
			if strings.HasPrefix(content, " ") {
				content = content[1:]
			}
			tomlLines = append(tomlLines, content)
		} else {
			// First non-comment line ends the block
			break
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	// No metadata block found
	if len(tomlLines) == 0 {
		return &Metadata{}, nil
	}

	// Parse TOML
	tomlContent := strings.Join(tomlLines, "\n")
	var meta Metadata
	if _, err := toml.Decode(tomlContent, &meta); err != nil {
		return nil, err
	}

	return &meta, nil
}

// Package context provides functions for resolving the current bliss context
// by walking up the directory tree to find a .bliss-context marker file.
package context

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const markerFile = ".bliss-context"

// FindContext walks up from startDir looking for a .bliss-context file.
// Returns the context name (slug), the directory containing the marker, and any error.
func FindContext(startDir string) (name string, contextDir string, err error) {
	dir := startDir
	for {
		markerPath := filepath.Join(dir, markerFile)
		if _, statErr := os.Stat(markerPath); statErr == nil {
			name, err = ReadContextFile(dir)
			if err != nil {
				return "", "", err
			}
			return name, dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached filesystem root without finding marker
			return "", "", fmt.Errorf("no .bliss-context found (run 'bliss init' to create one)")
		}
		dir = parent
	}
}

// ReadContextFile reads the context name (slug) from the .bliss-context file in the given directory.
func ReadContextFile(dir string) (string, error) {
	markerPath := filepath.Join(dir, markerFile)
	data, err := os.ReadFile(markerPath)
	if err != nil {
		return "", fmt.Errorf("reading .bliss-context: %w", err)
	}
	name := strings.TrimSpace(string(data))
	if name == "" {
		return "", fmt.Errorf(".bliss-context is empty")
	}
	return name, nil
}

// WriteContextFile writes a context name (slug) to the .bliss-context file in the given directory.
func WriteContextFile(dir, name string) error {
	markerPath := filepath.Join(dir, markerFile)
	return os.WriteFile(markerPath, []byte(name+"\n"), 0644)
}

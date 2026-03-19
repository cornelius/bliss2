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
// Returns the UUID, the directory containing the marker, and any error.
func FindContext(startDir string) (uuid string, contextDir string, err error) {
	dir := startDir
	for {
		markerPath := filepath.Join(dir, markerFile)
		if _, statErr := os.Stat(markerPath); statErr == nil {
			uuid, err = ReadContextFile(dir)
			if err != nil {
				return "", "", err
			}
			return uuid, dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached filesystem root without finding marker
			return "", "", fmt.Errorf("no .bliss-context found (run 'bliss init' to create one)")
		}
		dir = parent
	}
}

// ReadContextFile reads the UUID from the .bliss-context file in the given directory.
func ReadContextFile(dir string) (string, error) {
	markerPath := filepath.Join(dir, markerFile)
	data, err := os.ReadFile(markerPath)
	if err != nil {
		return "", fmt.Errorf("reading .bliss-context: %w", err)
	}
	uuid := strings.TrimSpace(string(data))
	if uuid == "" {
		return "", fmt.Errorf(".bliss-context is empty")
	}
	return uuid, nil
}

// WriteContextFile writes a UUID to the .bliss-context file in the given directory.
func WriteContextFile(dir, uuid string) error {
	markerPath := filepath.Join(dir, markerFile)
	return os.WriteFile(markerPath, []byte(uuid+"\n"), 0644)
}

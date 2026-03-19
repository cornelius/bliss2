// Package todo provides types and functions for reading and writing todo files.
package todo

import (
	"fmt"
	"strings"
	"time"
)

// Todo represents a single todo item.
type Todo struct {
	UUID      string
	Title     string
	Body      string
	CreatedAt time.Time
}

// Parse parses the content of a todo file.
// The first line is the title. An optional body follows after a blank line.
func Parse(content string) (Todo, error) {
	content = strings.TrimRight(content, "\n")
	if content == "" {
		return Todo{}, fmt.Errorf("todo file is empty")
	}

	lines := strings.Split(content, "\n")
	title := strings.TrimSpace(lines[0])
	if title == "" {
		return Todo{}, fmt.Errorf("todo title is empty")
	}

	var body string
	if len(lines) > 1 {
		// Expect a blank line separator, then body
		rest := lines[1:]
		// Find the first blank line
		bodyStart := -1
		for i, line := range rest {
			if strings.TrimSpace(line) == "" {
				bodyStart = i + 1
				break
			}
		}
		if bodyStart >= 0 && bodyStart < len(rest) {
			body = strings.Join(rest[bodyStart:], "\n")
			body = strings.TrimRight(body, "\n")
		}
	}

	return Todo{
		Title: title,
		Body:  body,
	}, nil
}

// Format formats a todo for writing to a file.
func Format(t Todo) string {
	if t.Body == "" {
		return t.Title + "\n"
	}
	return t.Title + "\n\n" + t.Body + "\n"
}

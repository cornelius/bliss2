// Package list provides types and functions for reading and writing list files.
package list

import (
	"strings"
)

// Section represents a section within a list, with an optional name and ordered UUIDs.
type Section struct {
	Name  string
	Items []string
}

// List represents an ordered list of todos, optionally divided into sections.
type List struct {
	Sections []Section
}

// Parse parses the content of a list file.
// Lines starting with "---" are section separators.
// Other non-empty lines are UUIDs.
func Parse(content string) (List, error) {
	lines := strings.Split(strings.TrimRight(content, "\n"), "\n")

	var sections []Section
	current := Section{}

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if strings.HasPrefix(trimmed, "---") {
			// Section separator
			sections = append(sections, current)
			name := strings.TrimSpace(strings.TrimPrefix(trimmed, "---"))
			current = Section{Name: name}
		} else {
			current.Items = append(current.Items, trimmed)
		}
	}
	sections = append(sections, current)

	// If no content at all, return a list with a single empty section
	if len(sections) == 0 {
		sections = []Section{{}}
	}

	return List{Sections: sections}, nil
}

// Format formats a list for writing to a file.
func Format(l List) string {
	var sb strings.Builder
	for i, section := range l.Sections {
		if i > 0 {
			// Write separator
			if section.Name != "" {
				sb.WriteString("--- " + section.Name + "\n")
			} else {
				sb.WriteString("---\n")
			}
		}
		for _, item := range section.Items {
			sb.WriteString(item + "\n")
		}
	}
	return sb.String()
}

// Add adds a UUID to the list.
// If urgent is true, it is prepended to the first section.
// Otherwise it is appended to the last section.
func Add(l *List, uuid string, urgent bool) {
	if len(l.Sections) == 0 {
		l.Sections = []Section{{}}
	}
	if urgent {
		l.Sections[0].Items = append([]string{uuid}, l.Sections[0].Items...)
	} else {
		last := len(l.Sections) - 1
		l.Sections[last].Items = append(l.Sections[last].Items, uuid)
	}
}

// Remove removes a UUID from all sections in the list.
func Remove(l *List, uuid string) {
	for i := range l.Sections {
		items := l.Sections[i].Items[:0]
		for _, item := range l.Sections[i].Items {
			if item != uuid {
				items = append(items, item)
			}
		}
		l.Sections[i].Items = items
	}
}

// Contains returns true if the UUID is in any section of the list.
func Contains(l List, uuid string) bool {
	for _, section := range l.Sections {
		for _, item := range section.Items {
			if item == uuid {
				return true
			}
		}
	}
	return false
}

// AllUUIDs returns a flat list of all UUIDs in order across all sections.
func AllUUIDs(l List) []string {
	var result []string
	for _, section := range l.Sections {
		result = append(result, section.Items...)
	}
	return result
}

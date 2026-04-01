// Package slug provides context name slugification.
package slug

import (
	"regexp"
	"strings"
)

var (
	nonAlphanumHyphen = regexp.MustCompile(`[^a-z0-9-]+`)
	multipleHyphens   = regexp.MustCompile(`-{2,}`)
)

// Slugify converts a name to a URL-safe slug: lowercase, spaces and
// non-alphanumeric characters replaced with hyphens, collapsed and trimmed.
// Examples: "My Project" → "my-project", "bliss2" → "bliss2".
func Slugify(name string) string {
	s := strings.ToLower(name)
	s = nonAlphanumHyphen.ReplaceAllString(s, "-")
	s = multipleHyphens.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	return s
}

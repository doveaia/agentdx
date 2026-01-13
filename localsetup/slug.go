package localsetup

import (
	"regexp"
	"strings"
)

// ToSlug converts a project folder name to a database-safe slug.
//
// The conversion rules are:
//   - Convert to lowercase
//   - Replace hyphens and spaces with underscores
//   - Remove all characters except a-z, 0-9, and underscore
//   - Collapse multiple consecutive underscores to a single underscore
//   - Trim leading and trailing underscores
//
// Examples:
//   - "my-project" → "my_project"
//   - "My Project" → "my_project"
//   - "café-app" → "caf_app"
//   - "Test@#$Project" → "test_project"
func ToSlug(name string) string {
	// Convert to lowercase
	s := strings.ToLower(name)

	// Replace hyphens and spaces with underscores
	s = strings.ReplaceAll(s, "-", "_")
	s = strings.ReplaceAll(s, " ", "_")

	// Remove non-alphanumeric characters (except underscore)
	re := regexp.MustCompile(`[^a-z0-9_]`)
	s = re.ReplaceAllString(s, "")

	// Collapse multiple underscores to a single underscore
	re = regexp.MustCompile(`_+`)
	s = re.ReplaceAllString(s, "_")

	// Trim leading and trailing underscores
	s = strings.Trim(s, "_")

	return s
}

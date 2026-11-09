package theme

import (
	"regexp"
	"strings"
)

var themeNamePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.-]{0,63}$`)

// slugInvalid matches anything that is not a lowercase letter, digit, hyphen
// or underscore. Matches are replaced with a single hyphen.
var slugInvalid = regexp.MustCompile(`[^a-z0-9_-]+`)

// slugCollapse collapses consecutive hyphens into one.
var slugCollapse = regexp.MustCompile(`-+`)

// ValidThemeName reports whether name is safe as a theme directory name.
func ValidThemeName(name string) bool {
	return themeNamePattern.MatchString(name)
}

// SlugifyThemeName normalises a user-typed theme name into a lowercase value
// that is safe to use as a directory name. Spaces and punctuation become
// hyphens. Returns "" if the input has no usable characters, so callers can
// reject empty slugs.
func SlugifyThemeName(name string) string {
	s := strings.ToLower(strings.TrimSpace(name))
	s = slugInvalid.ReplaceAllString(s, "-")
	s = slugCollapse.ReplaceAllString(s, "-")
	return strings.Trim(s, "-_")
}

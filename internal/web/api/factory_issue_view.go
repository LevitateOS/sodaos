package api

import (
	"strings"
	"unicode"
)

// validNativeLinkPart reports whether a native owner or repository name
// can appear in a local link path. It rejects traversal, separators,
// query/fragment markers and whitespace/control characters.
func validNativeLinkPart(part string) bool {
	return part != "" && part != "." && part != ".." && !strings.ContainsAny(part, "/\\%?#") && !strings.ContainsFunc(part, func(c rune) bool { return unicode.IsSpace(c) || unicode.IsControl(c) })
}

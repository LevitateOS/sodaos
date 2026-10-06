package forgejo

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

const RepositoryPageSize = 12

func repositoryPart(value string) bool {
	return value != "" && value != "." && value != ".." && len(value) <= 255 && utf8.ValidString(value) && !strings.ContainsAny(value, "/\\") && strings.IndexFunc(value, unicode.IsControl) < 0
}

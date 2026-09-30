package auth

import (
	"strconv"
)

// PositiveID accepts one canonical positive decimal Forgejo identifier.
func PositiveID(value string) (int64, bool) {
	if len(value) == 0 || len(value) > 19 {
		return 0, false
	}
	id, err := strconv.ParseInt(value, 10, 64)
	return id, err == nil && id > 0 && strconv.FormatInt(id, 10) == value
}

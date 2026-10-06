package store

import (
	"fmt"

	"github.com/levitateos/sodaos/internal/identity"
)

func identityBinding(c identity.Connection) string {
	return fmt.Sprintf("soda/identity/%s/%d", c.ID, c.Generation)
}

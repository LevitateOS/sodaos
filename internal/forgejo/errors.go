package forgejo

import (
	"errors"
	"fmt"
)

// HTTPError preserves native denial/status without retaining provider bodies,
// request URLs or credentials. Callers must not retry with another authority.
type HTTPError struct{ Status int }

func (e *HTTPError) Error() string {
	return fmt.Sprintf("Forgejo rejected operation (HTTP %d)", e.Status)
}

var (
	ErrUnavailable      = errors.New("forgejo could not be reached")
	ErrInvalidResponse  = errors.New("invalid Forgejo response")
	ErrResponseTooLarge = errors.New("forgejo response exceeds the supported size")
)

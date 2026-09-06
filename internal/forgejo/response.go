package forgejo

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

// HTTPError preserves native denial/status without retaining provider bodies,
// request URLs or credentials. Callers must not retry with another authority.
type HTTPError struct{ Status int }

func (e *HTTPError) Error() string {
	return fmt.Sprintf("Forgejo rejected operation (HTTP %d)", e.Status)
}

var (
	ErrUnavailable      = errors.New("Forgejo could not be reached")
	ErrInvalidResponse  = errors.New("invalid Forgejo response")
	ErrResponseTooLarge = errors.New("Forgejo response exceeds the supported size")
)

func transportError(ctx context.Context) error {
	// Preserve cancellation without exposing http.Client's token-bearing URL errors.
	if err := ctx.Err(); err != nil {
		return err
	}
	return ErrUnavailable
}

func decodeResponse(body io.Reader, limit int64, out any) error {
	// Read one extra byte: LimitReader alone can accept a valid JSON prefix of
	// an oversized response. Bound the entire representation before decoding.
	data, err := io.ReadAll(io.LimitReader(body, limit+1))
	if err != nil {
		return ErrUnavailable
	}
	if int64(len(data)) > limit {
		return ErrResponseTooLarge
	}
	if bytes.Equal(bytes.TrimSpace(data), []byte("null")) {
		return ErrInvalidResponse
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	if err := decoder.Decode(out); err != nil {
		return ErrInvalidResponse
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return ErrInvalidResponse
	}
	return nil
}

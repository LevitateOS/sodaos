package host

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/levitateos/sodaos/internal/tailnet"
)

type tailnetRoundTrip func(*http.Request) (*http.Response, error)

func (f tailnetRoundTrip) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestTailnetClientRejectsMalformedAndSecretBearingErrorResponses(t *testing.T) {
	for _, tc := range []struct {
		status int
		body   string
		want   error
	}{
		{200, `{}`, tailnet.ErrUnavailable},
		{200, `null`, tailnet.ErrUnconfirmed},
		{200, strings.Repeat(" ", 65537), tailnet.ErrUnconfirmed},
		{409, `synthetic-private`, tailnet.ErrConflict},
		{422, `synthetic-private`, tailnet.ErrUnsupported},
		{503, `synthetic-private`, tailnet.ErrUnavailable},
		{502, `synthetic-private`, tailnet.ErrUnconfirmed},
	} {
		c := &Client{HTTP: &http.Client{Transport: tailnetRoundTrip(func(r *http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: tc.status, Body: io.NopCloser(strings.NewReader(tc.body)), Header: make(http.Header)}, nil
		})}}
		_, e := c.TailnetSettings(t.Context())
		if !errors.Is(e, tc.want) || strings.Contains(e.Error(), "synthetic") {
			t.Fatal(tc.status, e)
		}
	}
}

package host

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/coder/websocket"
)

func TestTerminalStatesPreservesSupportFailure(t *testing.T) {
	for _, code := range []string{"terminal_support_mismatch", "terminal_support_unavailable", "project_maintenance"} {
		t.Run(code, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				c, err := websocket.Accept(w, r, nil)
				if err != nil {
					return
				}
				defer c.CloseNow()
				if _, _, err := c.Read(r.Context()); err != nil {
					return
				}
				_ = Write(r.Context(), c, TerminalFrame{Type: "closed", Reason: code})
			}))
			defer server.Close()
			client := &Client{HTTP: server.Client()}
			// The transport deliberately routes the fixed private host authority to
			// this local HTTP fixture; production still uses its private Unix socket.
			client.HTTP.Transport = &terminalTestTransport{target: server.URL, base: http.DefaultTransport}
			_, err := client.TerminalStates(context.Background(), TerminalRequest{Action: "reserve", ID: "0123456789abcdef0123456789abcdef", Project: "p0123456789abcdef01234567", Login: "soda-tester", Identity: 1, Cols: 80, Rows: 24, Scope: string(makeHexScope())})
			var failure *TerminalError
			if !errors.As(err, &failure) || failure.Code != code {
				t.Fatalf("got %v, want %s", err, code)
			}
		})
	}
}

type terminalTestTransport struct {
	target string
	base   http.RoundTripper
}

func (t *terminalTestTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	target, err := http.NewRequestWithContext(r.Context(), r.Method, t.target+r.URL.RequestURI(), r.Body)
	if err != nil {
		return nil, err
	}
	target.Header = r.Header.Clone()
	return t.base.RoundTrip(target)
}

func makeHexScope() []byte {
	b := make([]byte, 64)
	for i := range b {
		b[i] = 'a'
	}
	return b
}

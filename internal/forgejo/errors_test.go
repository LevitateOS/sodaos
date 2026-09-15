package forgejo

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestResponseBoundary(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		limit      int64
		want       error
	}{
		{"valid", `{"id":12}`, 9, nil},
		{"oversized whitespace", `{"id":12} `, 9, ErrResponseTooLarge},
		{"second object", `{"id":12}{}`, 100, ErrInvalidResponse},
		{"trailing garbage", `{"id":12}private-data`, 100, ErrInvalidResponse},
		{"malformed", `{"id":"private-data"}`, 100, ErrInvalidResponse},
		{"null", `null`, 100, ErrInvalidResponse},
		{"empty", ``, 100, ErrInvalidResponse},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var user User
			err := decodeResponse(strings.NewReader(tc.body), tc.limit, &user)
			if !errors.Is(err, tc.want) {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestNativeStatusIsTypedAndSanitized(t *testing.T) {
	for _, status := range []int{301, 401, 403, 404, 409, 422, 429, 500, 503} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			requests := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests++
				w.WriteHeader(status)
				fmt.Fprint(w, `{"message":"private-provider-content"}`)
			}))
			defer server.Close()
			_, err := New(server.URL).Current(context.Background(), "private-token")
			var native *HTTPError
			if !errors.As(err, &native) || native.Status != status {
				t.Fatalf("wrong error: %v", err)
			}
			if strings.Contains(err.Error(), "private") || strings.Contains(err.Error(), server.URL) {
				t.Fatal("provider diagnostics leaked")
			}
			if requests != 1 {
				t.Fatalf("unexpected retry: %d requests", requests)
			}
		})
	}
}

func TestExchangeRejectsTrailingData(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"access_token":"private-token"}{}`)
	}))
	defer server.Close()
	token, err := New(server.URL).ExchangeGrant(context.Background(), "client", "secret", "code", "https://soda.example/oauth/callback", "verifier")
	if token.Access != "" || !errors.Is(err, ErrInvalidResponse) {
		t.Fatal("malformed token response accepted")
	}
}

func TestTransportCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := New("http://127.0.0.1:1").Current(ctx, "private-token")
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("cancellation lost: %v", err)
	}
}

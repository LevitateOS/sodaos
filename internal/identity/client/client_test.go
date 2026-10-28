package client

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMetadataArraysDecode(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/connections":
			_, _ = w.Write([]byte(`[{"id":"connection","owner_id":"9007199254740993","label":"subscription","email":"","plan":"plus","generation":1,"state":"ready"}]`))
		default:
			w.WriteHeader(http.StatusForbidden)
		}
	}))
	defer server.Close()
	c := &Client{HTTP: server.Client()}
	c.HTTP.Transport = &redirectTransport{server.URL, server.Client().Transport}
	connections, err := c.Connections(t.Context(), 9007199254740993)
	if err != nil || len(connections) != 1 || connections[0].OwnerID != 9007199254740993 {
		t.Fatal("metadata array or lossless identity failed", err)
	}
}

type redirectTransport struct {
	url  string
	next http.RoundTripper
}

func (t *redirectTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	r.URL.Scheme = "http"
	r.URL.Host = t.url[len("http://"):]
	return t.next.RoundTrip(r)
}

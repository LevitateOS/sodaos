package client

import (
	"context"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/levitateos/sodaos/internal/identity"
)

func (c *Client) AcquireGit(ctx context.Context, in identity.GitAcquireRequest) (identity.Lease, error) {
	var lease identity.Lease
	err := c.call(ctx, "/git-acquire", identity.Request{GitAcquire: &in}, &lease)
	return lease, err
}

func (c *Client) RegisterGit(ctx context.Context, id string, b identity.Binding) (identity.GitSession, error) {
	var session identity.GitSession
	err := c.call(ctx, "/git-register", identity.Request{ID: id, Binding: &b}, &session)
	return session, err
}

// GitProxy is private host-to-broker streaming, not a container interface.
// It reuses Unix transport without the metadata client's thirty-second timeout.
func (c *Client) GitProxy(id string) http.Handler {
	if !gitLeaseID(id) {
		return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { http.Error(w, "Git unavailable.", http.StatusForbidden) })
	}
	target := &url.URL{Scheme: "http", Host: "soda-identity", Path: "/git/" + id}
	return &httputil.ReverseProxy{
		Transport: c.HTTP.Transport,
		ErrorLog:  log.New(io.Discard, "", 0),
		Rewrite: func(r *httputil.ProxyRequest) {
			r.SetURL(target)
			r.Out.Header["Git-Protocol"] = append([]string(nil), r.In.Header.Values("Git-Protocol")...)
			r.Out.Header.Del("Authorization")
			r.Out.Header.Del("Proxy-Authorization")
			r.Out.Header.Del("Cookie")
		},
		ErrorHandler: func(w http.ResponseWriter, _ *http.Request, _ error) {
			http.Error(w, "Git relay unavailable.", http.StatusBadGateway)
		},
	}
}

func gitLeaseID(value string) bool {
	if len(value) != 32 {
		return false
	}
	for _, r := range value {
		if !(r >= '0' && r <= '9') && !(r >= 'a' && r <= 'f') {
			return false
		}
	}
	return true
}

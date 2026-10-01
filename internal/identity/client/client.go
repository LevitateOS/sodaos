// Package client calls the private identity broker over Unix HTTP.
package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/levitateos/sodaos/internal/identity"
)

type Client struct{ HTTP *http.Client }

func New(socket string) *Client {
	return &Client{HTTP: &http.Client{Timeout: 30 * time.Second, Transport: &http.Transport{DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, "unix", socket)
	}}, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}}
}

func (c *Client) call(ctx context.Context, path string, in identity.Request, out any) error {
	var body bytes.Buffer
	if err := json.NewEncoder(&body).Encode(in); err != nil {
		return err
	}
	r, err := http.NewRequestWithContext(ctx, "POST", "http://soda-identity"+path, &body)
	if err != nil {
		return err
	}
	r.Header.Set("Content-Type", "application/json")
	res, err := c.HTTP.Do(r)
	if err != nil {
		return errors.New("identity broker unavailable")
	}
	defer func() { _ = res.Body.Close() }()
	if res.StatusCode != http.StatusOK {
		return decodeError(res.Body)
	}
	return decodeResponse(res.Body, out)
}

func decodeError(body io.Reader) error {
	code, _ := io.ReadAll(io.LimitReader(body, 64))
	switch strings.TrimSpace(string(code)) {
	case "denied":
		return identity.ErrDenied
	case "busy":
		return identity.ErrBusy
	case "stale":
		return identity.ErrStale
	case "reauth":
		return identity.ErrUncertain
	case "missing":
		return identity.ErrNotFound
	}
	return errors.New("identity operation failed")
}

func decodeResponse(body io.Reader, out any) error {
	if out == nil {
		return nil
	}
	data, err := io.ReadAll(io.LimitReader(body, 512<<10+1))
	if err != nil {
		return err
	}
	if len(data) > 512<<10 {
		return errors.New("identity response exceeds limit")
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err = decoder.Decode(out); err != nil {
		return err
	}
	if decoder.Decode(new(any)) != io.EOF {
		return errors.New("invalid identity response")
	}
	return nil
}

func (c *Client) ReconcileLease(ctx context.Context, id string) error {
	return c.call(ctx, "/reconcile-lease", identity.Request{ID: id}, nil)
}

func (c *Client) Connections(ctx context.Context, owner int64) ([]identity.Connection, error) {
	var out []identity.Connection
	err := c.call(ctx, "/connections", identity.Request{OwnerID: owner}, &out)
	return out, err
}

func (c *Client) Available(ctx context.Context, actor int64, project string) ([]identity.Connection, error) {
	var out []identity.Connection
	err := c.call(ctx, "/available", identity.Request{OwnerID: actor, ProjectID: project}, &out)
	return out, err
}

func (c *Client) StartEnrollment(ctx context.Context, owner int64, providerID, label string) (identity.Enrollment, error) {
	var out identity.Enrollment
	err := c.call(ctx, "/enrollment/start", identity.Request{OwnerID: owner, ProviderID: providerID, Label: label}, &out)
	return out, err
}

func (c *Client) Enrollment(ctx context.Context, owner int64, id string) (identity.Enrollment, error) {
	var out identity.Enrollment
	err := c.call(ctx, "/enrollment/read", identity.Request{OwnerID: owner, ID: id}, &out)
	return out, err
}

func (c *Client) CancelEnrollment(ctx context.Context, owner int64, id string) error {
	return c.call(ctx, "/enrollment/cancel", identity.Request{OwnerID: owner, ID: id}, nil)
}

func (c *Client) Grants(ctx context.Context, owner int64, id string) ([]identity.Grant, error) {
	var out []identity.Grant
	err := c.call(ctx, "/grants", identity.Request{OwnerID: owner, ID: id}, &out)
	return out, err
}

func (c *Client) CreateGrant(ctx context.Context, owner int64, in identity.GrantRequest) (identity.Grant, error) {
	var out identity.Grant
	err := c.call(ctx, "/grant/create", identity.Request{OwnerID: owner, Grant: &in}, &out)
	return out, err
}

func (c *Client) RevokeGrant(ctx context.Context, owner int64, id string) error {
	return c.call(ctx, "/grant/revoke", identity.Request{OwnerID: owner, ID: id}, nil)
}

func (c *Client) Leases(ctx context.Context, owner int64, id string) ([]identity.Lease, error) {
	var out []identity.Lease
	err := c.call(ctx, "/leases", identity.Request{OwnerID: owner, ID: id}, &out)
	return out, err
}

func (c *Client) EndLease(ctx context.Context, owner int64, id string) error {
	return c.call(ctx, "/lease/end", identity.Request{OwnerID: owner, ID: id}, nil)
}

func (c *Client) Revoke(ctx context.Context, owner int64, id string) error {
	return c.call(ctx, "/revoke", identity.Request{OwnerID: owner, ID: id}, nil)
}

func (c *Client) Acquire(ctx context.Context, in identity.AcquireRequest) (identity.Lease, error) {
	var out identity.Lease
	err := c.call(ctx, "/acquire", identity.Request{Acquire: &in}, &out)
	return out, err
}

func (c *Client) Register(ctx context.Context, id string, b identity.Binding) (identity.Delivery, error) {
	var out identity.DeliveryWire
	err := c.call(ctx, "/register", identity.Request{ID: id, Binding: &b}, &out)
	return identity.Delivery(out), err
}

func (c *Client) Return(ctx context.Context, id string, b identity.Binding, data []byte) error {
	return c.call(ctx, "/return", identity.Request{ID: id, Binding: &b, Credential: data}, nil)
}

func (c *Client) Reject(ctx context.Context, id string, b identity.Binding) error {
	return c.call(ctx, "/reject", identity.Request{ID: id, Binding: &b}, nil)
}

func (c *Client) GetExecution(ctx context.Context, kind, executionID string) (identity.Execution, error) {
	var out identity.Execution
	err := c.call(ctx, "/execution/get", identity.Request{Kind: kind, ExecutionID: executionID}, &out)
	return out, err
}

func (c *Client) CloseExecution(ctx context.Context, kind, executionID string) error {
	return c.call(ctx, "/execution/close", identity.Request{Kind: kind, ExecutionID: executionID}, nil)
}

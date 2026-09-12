package host

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/netip"
	"time"

	"github.com/levitateos/sodaos/internal/projectos"
)

type Create struct {
	Profile *projectos.Profile `json:"profile,omitempty"`
	ID      string             `json:"id"`
	Owner   int64              `json:"owner"`
}
type Account struct {
	Project  string   `json:"project"`
	Login    string   `json:"login"`
	Identity int64    `json:"identity"`
	Keys     []string `json:"keys"`
}
type Environment struct {
	Image   string             `json:"image,omitempty"`
	Profile *projectos.Profile `json:"profile,omitempty"`
	ID      string             `json:"id"`
	IP      string             `json:"ip"`
	Running bool               `json:"running"`
}
type Connection struct {
	Environment Environment `json:"environment"`
	HostKey     string      `json:"host_key"`
	Fingerprint string      `json:"fingerprint"`
}
type Client struct{ HTTP *http.Client }
type nativeHTTPError struct{ status int }

func (e nativeHTTPError) Error() string {
	return fmt.Sprintf("native project operation failed (HTTP %d); operator should inspect soda-host journal", e.status)
}

func NewClient(socket string) *Client {
	return &Client{HTTP: &http.Client{Timeout: 4 * time.Minute, Transport: &http.Transport{DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, "unix", socket)
	}}, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}}
}
func (c *Client) call(ctx context.Context, path string, in, out any) error {
	var b bytes.Buffer
	if err := json.NewEncoder(&b).Encode(in); err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, "POST", "http://soda-host"+path, &b)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := c.HTTP.Do(req)
	if err != nil {
		return fmt.Errorf("host service unavailable")
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return nativeHTTPError{res.StatusCode}
	}
	if out != nil {
		body, err := io.ReadAll(io.LimitReader(res.Body, 65537))
		if err != nil || len(body) > 65536 || bytes.Equal(bytes.TrimSpace(body), []byte("null")) {
			return fmt.Errorf("invalid native response")
		}
		if err = json.Unmarshal(body, out); err != nil {
			return fmt.Errorf("invalid native response")
		}
	}
	return nil
}
func (c *Client) Create(ctx context.Context, in Create) (Environment, error) {
	var out Environment
	err := c.call(ctx, "/create", in, &out)
	if err == nil && (out.ID != in.ID || !out.Running || !validAddress(out.IP) || in.Profile == nil || out.Profile == nil || *out.Profile != *in.Profile) {
		err = fmt.Errorf("native creation did not return the expected running endpoint")
	}
	return out, err
}
func (c *Client) Inspect(ctx context.Context, id string) (Environment, error) {
	var out Environment
	err := c.call(ctx, "/inspect", Create{ID: id}, &out)
	if err == nil && (out.ID != id || (out.IP != "" && !validAddress(out.IP))) {
		err = fmt.Errorf("invalid native environment observation")
	}
	return out, err
}
func (c *Client) Join(ctx context.Context, in Account) error {
	var result struct {
		OK bool `json:"ok"`
	}
	err := c.call(ctx, "/account", in, &result)
	if err == nil && !result.OK {
		err = fmt.Errorf("native account result was not confirmed")
	}
	return err
}
func (c *Client) Connection(ctx context.Context, id string) (Connection, error) {
	var result Connection
	err := c.call(ctx, "/connection", Create{ID: id}, &result)
	if err == nil && (result.Environment.ID != id || (result.Environment.Running && (!validAddress(result.Environment.IP) || result.HostKey == "" || result.Fingerprint == ""))) {
		err = fmt.Errorf("native connection result incomplete")
	}
	return result, err
}
func validAddress(value string) bool {
	ip, err := netip.ParseAddr(value)
	return err == nil && !ip.IsUnspecified() && !ip.IsMulticast() && !ip.IsLoopback()
}

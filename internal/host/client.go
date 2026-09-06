package host

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"
)

type Create struct {
	ID    string `json:"id"`
	Owner int64  `json:"owner"`
}
type Account struct {
	Project  string   `json:"project"`
	Login    string   `json:"login"`
	Identity int64    `json:"identity"`
	Keys     []string `json:"keys"`
}
type Environment struct {
	ID      string `json:"id"`
	IP      string `json:"ip"`
	Running bool   `json:"running"`
}
type Client struct{ HTTP *http.Client }

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
		return fmt.Errorf("native project operation failed (HTTP %d); operator should inspect soda-host journal", res.StatusCode)
	}
	if out != nil {
		return json.NewDecoder(io.LimitReader(res.Body, 65536)).Decode(out)
	}
	return nil
}
func (c *Client) Create(ctx context.Context, in Create) (Environment, error) {
	var out Environment
	err := c.call(ctx, "/create", in, &out)
	return out, err
}
func (c *Client) Inspect(ctx context.Context, id string) (Environment, error) {
	var out Environment
	err := c.call(ctx, "/inspect", Create{ID: id}, &out)
	return out, err
}
func (c *Client) Join(ctx context.Context, in Account) error { return c.call(ctx, "/account", in, nil) }

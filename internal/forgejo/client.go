// Package forgejo is the HTTP client for Forgejo's supported APIs. It does not
// access Forgejo's database or copy upstream business rules into Soda.
package forgejo

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"
)

type Client struct {
	Base string
	HTTP *http.Client
}
type User struct {
	ID    int64  `json:"id"`
	Login string `json:"login"`
	Name  string `json:"full_name"`
	Admin bool   `json:"is_admin"`
}
type Repository struct {
	Private       bool                   `json:"private"`
	DefaultBranch string                 `json:"default_branch"`
	Permissions   *RepositoryPermissions `json:"permissions"`
	ID            int64                  `json:"id"`
	Name          string                 `json:"name"`
	FullName      string                 `json:"full_name"`
	Owner         User                   `json:"owner"`
}
type RepositoryPermissions struct {
	Admin bool `json:"admin"`
	Push  bool `json:"push"`
	Pull  bool `json:"pull"`
}

func New(base string) *Client {
	return &Client{Base: strings.TrimRight(base, "/"), HTTP: &http.Client{Timeout: 30 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}}
}

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

func (c *Client) request(ctx context.Context, method, path, token string, in, out any) error {
	var b bytes.Buffer
	if in != nil {
		if err := json.NewEncoder(&b).Encode(in); err != nil {
			return ErrInvalidResponse
		}
	}
	req, err := http.NewRequestWithContext(ctx, method, c.Base+"/api/v1"+path, &b)
	if err != nil {
		return ErrUnavailable
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "token "+token)
	res, err := c.HTTP.Do(req)
	if err != nil {
		return transportError(ctx)
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return &HTTPError{Status: res.StatusCode}
	}
	if out != nil {
		return decodeResponse(res.Body, 2<<20, out)
	}
	return nil
}

func (c *Client) Current(ctx context.Context, token string) (User, error) {
	var u User
	err := c.request(ctx, "GET", "/user", token, nil, &u)
	return u, err
}

// RevokeCurrentToken deletes the access token used for the request. Setup
// calls it once, after its configuration is written, so a failed setup keeps
// its retry token.
func (c *Client) RevokeCurrentToken(ctx context.Context, token string) error {
	return c.request(ctx, "DELETE", "/user/token", token, nil, nil)
}

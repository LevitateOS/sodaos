package forgejo

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/url"
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
	ID                  int64  `json:"id"`
	Name                string `json:"name"`
	FullName            string `json:"full_name"`
	Owner               User   `json:"owner"`
	Description         string `json:"description"`
	Private             bool   `json:"private"`
	DefaultBranch       string `json:"default_branch"`
	CloneURL            string `json:"clone_url"`
	SSHURL              string `json:"ssh_url"`
	Empty               bool   `json:"empty"`
	Fork                bool   `json:"fork"`
	Mirror              bool   `json:"mirror"`
	AllowMerge          bool   `json:"allow_merge_commits"`
	AllowSquash         bool   `json:"allow_squash_merge"`
	AllowRebase         bool   `json:"allow_rebase"`
	AllowRebaseExplicit bool   `json:"allow_rebase_explicit"`
	AllowFastForward    bool   `json:"allow_fast_forward_only_merge"`
}
type Application struct {
	ClientID string `json:"client_id"`
	Secret   string `json:"client_secret"`
}

func New(base string) *Client {
	return &Client{Base: strings.TrimRight(base, "/"), HTTP: &http.Client{Timeout: 30 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}}
}
func (c *Client) request(ctx context.Context, method, path, token string, in, out any) error {
	_, err := c.requestHeaders(ctx, method, path, token, in, out)
	return err
}
func (c *Client) requestHeaders(ctx context.Context, method, path, token string, in, out any) (http.Header, error) {
	var b bytes.Buffer
	if in != nil {
		if err := json.NewEncoder(&b).Encode(in); err != nil {
			return nil, ErrInvalidResponse
		}
	}
	req, err := http.NewRequestWithContext(ctx, method, c.Base+"/api/v1"+path, &b)
	if err != nil {
		return nil, ErrUnavailable
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "token "+token)
	res, err := c.HTTP.Do(req)
	if err != nil {
		return nil, transportError(ctx)
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, &HTTPError{Status: res.StatusCode}
	}
	if out != nil {
		if err := decodeResponse(res.Body, 2<<20, out); err != nil {
			return nil, err
		}
	}
	return res.Header, nil
}
func (c *Client) Current(ctx context.Context, token string) (User, error) {
	var u User
	err := c.request(ctx, "GET", "/user", token, nil, &u)
	return u, err
}
func (c *Client) CreateUser(ctx context.Context, token, login, email, password string) (User, error) {
	var u User
	err := c.request(ctx, "POST", "/admin/users", token, map[string]any{"username": login, "email": email, "password": password, "must_change_password": true, "send_notify": false}, &u)
	return u, err
}
func (c *Client) Repository(ctx context.Context, token, owner, name string) (Repository, error) {
	var r Repository
	err := c.request(ctx, "GET", "/repos/"+url.PathEscape(owner)+"/"+url.PathEscape(name), token, nil, &r)
	return r, err
}

func (c *Client) Application(ctx context.Context, token, redirect string) (Application, error) {
	var a Application
	err := c.request(ctx, "POST", "/user/applications/oauth2", token, map[string]any{"name": "SodaOS dashboard", "redirect_uris": []string{redirect}, "confidential_client": true}, &a)
	return a, err
}
func (c *Client) ExchangeGrant(ctx context.Context, clientID, secret, code, redirect, verifier string) (TokenResponse, error) {
	form := url.Values{"grant_type": {"authorization_code"}, "client_id": {clientID}, "client_secret": {secret}, "code": {code}, "redirect_uri": {redirect}, "code_verifier": {verifier}}
	return c.tokenRequest(ctx, form)
}

func (c *Client) RefreshGrant(ctx context.Context, clientID, secret, refresh string) (TokenResponse, error) {
	return c.tokenRequest(ctx, url.Values{"grant_type": {"refresh_token"}, "client_id": {clientID}, "client_secret": {secret}, "refresh_token": {refresh}})
}

type TokenResponse struct {
	Access    string `json:"access_token"`
	Refresh   string `json:"refresh_token"`
	Type      string `json:"token_type"`
	ExpiresIn int64  `json:"expires_in"`
	ExpiresAt int64  `json:"-"`
}

func (c *Client) tokenRequest(ctx context.Context, form url.Values) (TokenResponse, error) {
	started := time.Now()
	var token TokenResponse
	req, err := http.NewRequestWithContext(ctx, "POST", c.Base+"/login/oauth/access_token", strings.NewReader(form.Encode()))
	if err != nil {
		return token, ErrUnavailable
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	res, err := c.HTTP.Do(req)
	if err != nil {
		return token, transportError(ctx)
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return token, &HTTPError{Status: res.StatusCode}
	}
	if err = decodeResponse(res.Body, 65536, &token); err != nil {
		return TokenResponse{}, err
	}
	if token.Access == "" || token.Refresh == "" || !strings.EqualFold(token.Type, "bearer") || token.ExpiresIn <= 0 || token.ExpiresIn > 365*24*60*60 {
		return TokenResponse{}, ErrInvalidResponse
	}
	token.ExpiresAt = started.Add(time.Duration(token.ExpiresIn) * time.Second).Unix()
	return token, nil
}

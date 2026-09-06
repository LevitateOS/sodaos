package forgejo

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
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
	ID            int64  `json:"id"`
	Name          string `json:"name"`
	FullName      string `json:"full_name"`
	Owner         User   `json:"owner"`
	Description   string `json:"description"`
	Private       bool   `json:"private"`
	DefaultBranch string `json:"default_branch"`
	CloneURL      string `json:"clone_url"`
	SSHURL        string `json:"ssh_url"`
	Empty         bool   `json:"empty"`
}
type Application struct {
	ClientID string `json:"client_id"`
	Secret   string `json:"client_secret"`
}

func New(base string) *Client {
	return &Client{Base: strings.TrimRight(base, "/"), HTTP: &http.Client{Timeout: 30 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}}
}
func (c *Client) request(ctx context.Context, method, path, token string, in, out any) error {
	var b bytes.Buffer
	if in != nil {
		if err := json.NewEncoder(&b).Encode(in); err != nil {
			return err
		}
	}
	req, err := http.NewRequestWithContext(ctx, method, c.Base+"/api/v1"+path, &b)
	if err != nil {
		return err
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

// UserRepositories lists the requested user's repositories, not those of the
// operator whose token authorizes the request. Continue until an empty page so
// a server-side page-size cap cannot silently truncate the picker.
func (c *Client) UserRepositories(ctx context.Context, token, username string) ([]Repository, error) {
	var repositories []Repository
	for page := 1; ; page++ {
		var batch []Repository
		path := fmt.Sprintf("/users/%s/repos?page=%d&limit=50", url.PathEscape(username), page)
		if err := c.request(ctx, "GET", path, token, nil, &batch); err != nil {
			return nil, err
		}
		if len(batch) == 0 {
			return repositories, nil
		}
		repositories = append(repositories, batch...)
	}
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
}

func (c *Client) tokenRequest(ctx context.Context, form url.Values) (TokenResponse, error) {
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
	return token, nil
}

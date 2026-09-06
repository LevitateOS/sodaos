package forgejo

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
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
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	FullName string `json:"full_name"`
	Owner    User   `json:"owner"`
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
		return errors.New("Forgejo could not be reached")
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return fmt.Errorf("Forgejo rejected operation (HTTP %d); inspect native Forgejo administration", res.StatusCode)
	}
	if out != nil {
		return json.NewDecoder(io.LimitReader(res.Body, 2<<20)).Decode(out)
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
func (c *Client) Application(ctx context.Context, token, redirect string) (Application, error) {
	var a Application
	err := c.request(ctx, "POST", "/user/applications/oauth2", token, map[string]any{"name": "SodaOS dashboard", "redirect_uris": []string{redirect}, "confidential_client": true}, &a)
	return a, err
}
func (c *Client) Exchange(ctx context.Context, clientID, secret, code, redirect, verifier string) (string, error) {
	form := url.Values{"grant_type": {"authorization_code"}, "client_id": {clientID}, "client_secret": {secret}, "code": {code}, "redirect_uri": {redirect}, "code_verifier": {verifier}}
	req, err := http.NewRequestWithContext(ctx, "POST", c.Base+"/login/oauth/access_token", strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	res, err := c.HTTP.Do(req)
	if err != nil {
		return "", errors.New("Forgejo token exchange unavailable")
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return "", errors.New("Forgejo rejected token exchange")
	}
	var token struct {
		Access string `json:"access_token"`
	}
	err = json.NewDecoder(io.LimitReader(res.Body, 65536)).Decode(&token)
	if err != nil || token.Access == "" {
		return "", errors.New("invalid Forgejo token response")
	}
	return token.Access, nil
}

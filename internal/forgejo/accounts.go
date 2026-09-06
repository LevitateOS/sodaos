package forgejo

import (
	"context"
	"fmt"
)

type UserSettings struct {
	Name        string `json:"full_name"`
	Description string `json:"description"`
	Location    string `json:"location"`
	Pronouns    string `json:"pronouns"`
}

func (c *Client) Settings(ctx context.Context, token string) (UserSettings, error) {
	var result UserSettings
	err := c.request(ctx, "GET", "/user/settings", token, nil, &result)
	return result, err
}
func (c *Client) UpdateSettings(ctx context.Context, token string, input UserSettings) (UserSettings, error) {
	var result UserSettings
	err := c.request(ctx, "PATCH", "/user/settings", token, input, &result)
	return result, err
}

type PublicKey struct {
	ID          int64  `json:"id"`
	Title       string `json:"title"`
	Key         string `json:"key"`
	Fingerprint string `json:"fingerprint"`
}

func (c *Client) GitKeys(ctx context.Context, token string, page int) ([]PublicKey, error) {
	result := []PublicKey{}
	err := c.request(ctx, "GET", fmt.Sprintf("/user/keys?page=%d&limit=50", page), token, nil, &result)
	return result, err
}
func (c *Client) AddGitKey(ctx context.Context, token, title, key string) (PublicKey, error) {
	var result PublicKey
	err := c.request(ctx, "POST", "/user/keys", token, struct {
		Title    string `json:"title"`
		Key      string `json:"key"`
		ReadOnly bool   `json:"read_only"`
	}{title, key, false}, &result)
	return result, err
}
func (c *Client) People(ctx context.Context, token string, page int) ([]User, error) {
	result := []User{}
	err := c.request(ctx, "GET", fmt.Sprintf("/admin/users?page=%d&limit=50", page), token, nil, &result)
	return result, err
}

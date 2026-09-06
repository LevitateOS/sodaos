package forgejo

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

type CreateRepository struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Private     bool   `json:"private"`
	AutoInit    bool   `json:"auto_init"`
}

func (c *Client) MyRepositories(ctx context.Context, token string, page int) ([]Repository, error) {
	result := []Repository{}
	err := c.request(ctx, "GET", fmt.Sprintf("/user/repos?page=%d&limit=50", page), token, nil, &result)
	return result, err
}
func (c *Client) CreateRepository(ctx context.Context, token string, input CreateRepository) (Repository, error) {
	var result Repository
	err := c.request(ctx, "POST", "/user/repos", token, input, &result)
	return result, err
}

type Content struct {
	Name     string `json:"name"`
	Path     string `json:"path"`
	Type     string `json:"type"`
	SHA      string `json:"sha"`
	Size     int64  `json:"size"`
	Content  string `json:"content"`
	Encoding string `json:"encoding"`
}

func (c *Client) Contents(ctx context.Context, token, owner, repo, ref, file string) ([]Content, error) {
	path := "/repos/" + url.PathEscape(owner) + "/" + url.PathEscape(repo) + "/contents"
	if file != "" {
		segments := strings.Split(file, "/")
		for i, segment := range segments {
			segments[i] = url.PathEscape(segment)
		}
		path += "/" + strings.Join(segments, "/")
	}
	path += "?" + url.Values{"ref": {ref}}.Encode()
	var raw json.RawMessage
	if err := c.request(ctx, "GET", path, token, nil, &raw); err != nil {
		return nil, err
	}
	var items []Content
	if len(raw) > 0 && raw[0] == '[' {
		if json.Unmarshal(raw, &items) != nil {
			return nil, ErrInvalidResponse
		}
	} else {
		var item Content
		if json.Unmarshal(raw, &item) != nil {
			return nil, ErrInvalidResponse
		}
		items = []Content{item}
	}
	return items, nil
}

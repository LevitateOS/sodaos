package forgejo

import (
	"bytes"
	"context"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/url"
)

type Reaction struct {
	Content string `json:"content"`
	User    User   `json:"user"`
}

func (c *Client) IssueReactions(ctx context.Context, token, owner, repo string, index int64) ([]Reaction, error) {
	items := []Reaction{}
	err := c.request(ctx, "GET", fmt.Sprintf("%s/issues/%d/reactions", repoAPI(owner, repo), index), token, nil, &items)
	return items, err
}
func (c *Client) ReactToIssue(ctx context.Context, token, owner, repo string, index int64, content string, remove bool) error {
	method := "POST"
	if remove {
		method = "DELETE"
	}
	return c.request(ctx, method, fmt.Sprintf("%s/issues/%d/reactions", repoAPI(owner, repo), index), token, struct {
		Content string `json:"content"`
	}{content}, nil)
}

type Subscription struct {
	Subscribed bool `json:"subscribed"`
	Ignored    bool `json:"ignored"`
}

func (c *Client) IssueSubscription(ctx context.Context, token, owner, repo string, index int64) (Subscription, error) {
	var result Subscription
	err := c.request(ctx, "GET", fmt.Sprintf("%s/issues/%d/subscriptions/check", repoAPI(owner, repo), index), token, nil, &result)
	return result, err
}
func (c *Client) SetIssueSubscription(ctx context.Context, token, owner, repo string, index int64, login string, subscribe bool) error {
	method := "PUT"
	if !subscribe {
		method = "DELETE"
	}
	return c.request(ctx, method, fmt.Sprintf("%s/issues/%d/subscriptions/%s", repoAPI(owner, repo), index, url.PathEscape(login)), token, nil, nil)
}

type Attachment struct {
	Type string `json:"type"`
	ID   int64  `json:"id"`
	UUID string `json:"uuid"`
	Name string `json:"name"`
	Size int64  `json:"size"`
}

func (c *Client) IssueAttachments(ctx context.Context, token, owner, repo string, index int64) ([]Attachment, error) {
	items := []Attachment{}
	err := c.request(ctx, "GET", fmt.Sprintf("%s/issues/%d/assets", repoAPI(owner, repo), index), token, nil, &items)
	return items, err
}
func (c *Client) AddIssueAttachment(ctx context.Context, token, owner, repo string, index int64, name string, data []byte) (Attachment, error) {
	return c.uploadAttachment(ctx, token, fmt.Sprintf("%s/issues/%d/assets", repoAPI(owner, repo), index), name, data)
}
func (c *Client) uploadAttachment(ctx context.Context, token, path, name string, data []byte) (Attachment, error) {
	var result Attachment
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("attachment", name)
	if err != nil {
		return result, ErrInvalidResponse
	}
	if _, err = part.Write(data); err != nil {
		return result, ErrInvalidResponse
	}
	if err = writer.Close(); err != nil {
		return result, ErrInvalidResponse
	}
	req, err := http.NewRequestWithContext(ctx, "POST", c.Base+"/api/v1"+path, &body)
	if err != nil {
		return result, ErrUnavailable
	}
	req.Header.Set("Authorization", "token "+token)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	res, err := c.HTTP.Do(req)
	if err != nil {
		return result, transportError(ctx)
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return result, &HTTPError{Status: res.StatusCode}
	}
	if err = decodeResponse(res.Body, 65536, &result); err != nil {
		return Attachment{}, err
	}
	return result, nil
}

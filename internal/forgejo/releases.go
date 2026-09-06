package forgejo

import (
	"context"
	"fmt"
	"net/url"
)

type Release struct {
	ID         int64        `json:"id"`
	Name       string       `json:"name"`
	Tag        string       `json:"tag_name"`
	Target     string       `json:"target_commitish"`
	Body       string       `json:"body"`
	Draft      bool         `json:"draft"`
	Prerelease bool         `json:"prerelease"`
	Created    string       `json:"created_at"`
	Published  string       `json:"published_at"`
	Assets     []Attachment `json:"assets"`
}
type ReleaseFields struct {
	Name       *string `json:"name,omitempty"`
	Tag        *string `json:"tag_name,omitempty"`
	Target     *string `json:"target_commitish,omitempty"`
	Body       *string `json:"body,omitempty"`
	Draft      *bool   `json:"draft,omitempty"`
	Prerelease *bool   `json:"prerelease,omitempty"`
}

func (c *Client) Releases(ctx context.Context, token, owner, repo string, page int) ([]Release, Pagination, error) {
	items := []Release{}
	headers, err := c.requestHeaders(ctx, "GET", fmt.Sprintf("%s/releases?page=%d&limit=30", repoAPI(owner, repo), page), token, nil, &items)
	if err != nil {
		return nil, Pagination{}, err
	}
	metadata, err := pagination(headers, page)
	return items, metadata, err
}
func (c *Client) Release(ctx context.Context, token, owner, repo string, id int64) (Release, error) {
	var result Release
	err := c.request(ctx, "GET", fmt.Sprintf("%s/releases/%d", repoAPI(owner, repo), id), token, nil, &result)
	return result, err
}
func (c *Client) SaveRelease(ctx context.Context, token, owner, repo string, id int64, input ReleaseFields) (Release, error) {
	path := repoAPI(owner, repo) + "/releases"
	method := "POST"
	if id != 0 {
		path += fmt.Sprintf("/%d", id)
		method = "PATCH"
	}
	var result Release
	err := c.request(ctx, method, path, token, input, &result)
	return result, err
}
func (c *Client) AddReleaseAsset(ctx context.Context, token, owner, repo string, id int64, name string, data []byte) (Attachment, error) {
	return c.uploadAttachment(ctx, token, fmt.Sprintf("%s/releases/%d/assets", repoAPI(owner, repo), id), name, data)
}
func (c *Client) ReleaseAsset(ctx context.Context, token, owner, repo string, id, asset int64) (Attachment, error) {
	var result Attachment
	err := c.request(ctx, "GET", fmt.Sprintf("%s/releases/%d/assets/%d", repoAPI(owner, repo), id, asset), token, nil, &result)
	return result, err
}

// The UUID is resolved from native repository-scoped asset metadata by callers.
// No native response URL or redirect becomes a credential-forwarding target.
func (c *Client) AttachmentData(ctx context.Context, token, uuid string) ([]byte, error) {
	return c.readNativeBytes(ctx, "/attachments/"+url.PathEscape(uuid), token, DownloadLimit)
}

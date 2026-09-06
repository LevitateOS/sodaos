package forgejo

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"strings"
)

const DownloadLimit = 8 << 20

// RawFile uses only the fixed authenticated raw-file route, not provider-supplied
// download URLs or LFS redirects. Buffer at most 8 MiB before committing browser
// response headers, so cancellation/oversize cannot look like a complete download.
func (c *Client) RawFile(ctx context.Context, token, owner, repo, ref, file string) ([]byte, error) {
	parts := strings.Split(file, "/")
	for i, part := range parts {
		parts[i] = url.PathEscape(part)
	}
	endpoint := repoAPI(owner, repo) + "/raw/" + strings.Join(parts, "/") + "?" + url.Values{"ref": {ref}}.Encode()
	return c.readBytes(ctx, endpoint, token, DownloadLimit)
}

func (c *Client) readBytes(ctx context.Context, path, token string, limit int64) ([]byte, error) {
	return c.readNativeBytes(ctx, "/api/v1"+path, token, limit)
}
func (c *Client) readNativeBytes(ctx context.Context, path, token string, limit int64) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", c.Base+path, nil)
	if err != nil {
		return nil, ErrUnavailable
	}
	req.Header.Set("Authorization", "token "+token)
	req.Header.Set("Accept", "application/octet-stream")
	res, err := c.HTTP.Do(req)
	if err != nil {
		return nil, transportError(ctx)
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return nil, &HTTPError{Status: res.StatusCode}
	}
	if res.ContentLength > limit {
		return nil, ErrResponseTooLarge
	}
	body, err := io.ReadAll(io.LimitReader(res.Body, limit+1))
	if err != nil {
		return nil, ErrUnavailable
	}
	if int64(len(body)) > limit {
		return nil, ErrResponseTooLarge
	}
	return body, nil
}

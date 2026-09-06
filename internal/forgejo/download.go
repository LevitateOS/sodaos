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
	endpoint := c.Base + "/api/v1/repos/" + url.PathEscape(owner) + "/" + url.PathEscape(repo) + "/raw/" + strings.Join(parts, "/") + "?" + url.Values{"ref": {ref}}.Encode()
	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
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
	if res.ContentLength > DownloadLimit {
		return nil, ErrResponseTooLarge
	}
	body, err := io.ReadAll(io.LimitReader(res.Body, DownloadLimit+1))
	if err != nil {
		return nil, ErrUnavailable
	}
	if len(body) > DownloadLimit {
		return nil, ErrResponseTooLarge
	}
	return body, nil
}

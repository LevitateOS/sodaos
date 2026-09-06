package forgejo

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// Forgejo 15.0.7 omits scope in token responses and may reuse an older
// confidential-client consent. Introspection supplies the actual grant, not the
// requested scope. Never treat a requested scope as evidence of consent.
func (c *Client) GrantScopes(ctx context.Context, clientID, secret, access string, uid int64) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.Base+"/login/oauth/introspect", strings.NewReader(url.Values{"token": {access}}.Encode()))
	if err != nil {
		return "", ErrUnavailable
	}
	req.SetBasicAuth(clientID, secret)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	res, err := c.HTTP.Do(req)
	if err != nil {
		return "", transportError(ctx)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return "", &HTTPError{Status: res.StatusCode}
	}
	var result struct {
		Active   bool     `json:"active"`
		Scope    string   `json:"scope"`
		Subject  string   `json:"sub"`
		Audience []string `json:"aud"`
	}
	if err = decodeResponse(res.Body, 65536, &result); err != nil {
		return "", err
	}
	if !result.Active || result.Subject != strconv.FormatInt(uid, 10) || len(result.Audience) != 1 || result.Audience[0] != clientID {
		return "", ErrInvalidResponse
	}
	return result.Scope, nil
}

func HasScope(scopes, required string) bool {
	for _, scope := range strings.Fields(strings.ReplaceAll(scopes, ",", " ")) {
		if scope == "all" || scope == required || (strings.HasPrefix(required, "read:") && scope == "write:"+strings.TrimPrefix(required, "read:")) {
			return true
		}
	}
	return false
}

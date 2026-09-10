package forgejo

import (
	"context"
	"strconv"
)

// OwnPublicKey is the supported 15.0.7 own-user API projection. Never use its
// optional global fingerprint query or an arbitrary username/provider URL.
type OwnPublicKey struct {
	ID    int64  `json:"id"`
	Key   string `json:"key"`
	Title string `json:"title"`
	Owner User   `json:"user"`
	Type  string `json:"key_type"`
}

func (c *Client) OwnPublicKeys(ctx context.Context, token string, actor int64, page int) ([]OwnPublicKey, error) {
	if actor <= 0 || page < 1 || page > 8 {
		return nil, ErrInvalidResponse
	}
	var keys []OwnPublicKey
	if err := c.request(ctx, "GET", "/user/keys?limit=10&page="+strconv.Itoa(page), token, nil, &keys); err != nil {
		return nil, err
	}
	if keys == nil || len(keys) > 10 {
		return nil, ErrInvalidResponse
	}
	seen := map[int64]bool{}
	for _, key := range keys {
		if key.ID <= 0 || key.Owner.ID != actor || key.Type != "user" || key.Key == "" || len(key.Key) > 16384 || len(key.Title) > 800 || seen[key.ID] {
			return nil, ErrInvalidResponse
		}
		seen[key.ID] = true
	}
	return keys, nil
}

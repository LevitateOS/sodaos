package forgejo

import (
	"context"
	"net/mail"
	"strings"
)

type nativeEmail struct {
	Email    string `json:"email"`
	Verified bool   `json:"verified"`
	Primary  bool   `json:"primary"`
}

// GitIdentity resolves native attribution for the intended stable user. It never
// substitutes an unverified address or derives an address from a display login.
func (c *Client) GitIdentity(ctx context.Context, token string, expectedUserID int64) (User, string, error) {
	if expectedUserID <= 0 {
		return User{}, "", ErrInvalidResponse
	}
	user, err := c.Current(ctx, token)
	if err != nil {
		return User{}, "", err
	}
	if user.ID != expectedUserID || strings.TrimSpace(user.Login) == "" {
		return User{}, "", ErrInvalidResponse
	}
	var emails []nativeEmail
	if err = c.request(ctx, "GET", "/user/emails", token, nil, &emails); err != nil {
		return User{}, "", err
	}
	email, err := verifiedPrimaryEmail(emails)
	if err != nil {
		return User{}, "", err
	}
	return user, email, nil
}

func verifiedPrimaryEmail(emails []nativeEmail) (string, error) {
	var primary string
	for _, email := range emails {
		if !email.Primary {
			continue
		}
		if primary != "" || !email.Verified || !validNativeEmail(email.Email) {
			return "", ErrInvalidResponse
		}
		primary = email.Email
	}
	if primary == "" {
		return "", ErrInvalidResponse
	}
	return primary, nil
}

func validNativeEmail(value string) bool {
	address, err := mail.ParseAddress(value)
	return err == nil && address.Address == value && !strings.ContainsAny(value, "\r\n")
}

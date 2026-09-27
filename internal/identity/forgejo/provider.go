// Package forgejo owns broker-native OAuth grants, separate from browser grants.
package forgejo

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"net"
	"net/url"
	"strings"
	"time"

	upstream "github.com/levitateos/sodaos/internal/forgejo"
	"github.com/levitateos/sodaos/internal/identity"
)

const (
	requiredScopes     = "read:user write:repository"
	enrollmentLifetime = 10 * time.Minute
)

type Config struct {
	Base        string `json:"base"`
	ClientID    string `json:"client_id"`
	RedirectURL string `json:"redirect_url"`
}

type Provider struct {
	config Config
	secret string
	client *upstream.Client
}

func New(config Config, secret string) (*Provider, error) {
	if !validURL(config.Base) || !validURL(config.RedirectURL) {
		return nil, errors.New("invalid Forgejo provider URLs")
	}
	base, _ := url.Parse(config.Base)
	if base.RawQuery != "" || base.Fragment != "" || config.ClientID == "" || secret == "" {
		return nil, errors.New("invalid Forgejo provider configuration")
	}
	config.Base = strings.TrimRight(config.Base, "/")
	return &Provider{config: config, secret: secret, client: upstream.New(config.Base)}, nil
}

func validURL(value string) bool {
	u, err := url.Parse(value)
	if err != nil || u.Host == "" || u.User != nil || u.Fragment != "" {
		return false
	}
	if u.Scheme == "https" {
		return true
	}
	if u.Scheme != "http" {
		return false
	}
	ip := net.ParseIP(u.Hostname())
	return u.Hostname() == "localhost" || (ip != nil && ip.IsLoopback())
}

func randomToken() (string, error) {
	var bytes [32]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		return "", errors.New("forgejo enrollment randomness unavailable")
	}
	return base64.RawURLEncoding.EncodeToString(bytes[:]), nil
}

func (p *Provider) Start(ctx context.Context, ownerID int64) (identity.EnrollmentSession, error) {
	if ownerID <= 0 {
		return nil, identity.ErrDenied
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	state, err := randomToken()
	if err != nil {
		return nil, err
	}
	verifier, err := randomToken()
	if err != nil {
		return nil, err
	}
	id, err := randomToken()
	if err != nil {
		return nil, err
	}
	digest := sha256.Sum256([]byte(verifier))
	query := url.Values{
		"client_id": {p.config.ClientID}, "redirect_uri": {p.config.RedirectURL},
		"response_type": {"code"}, "scope": {requiredScopes}, "state": {state},
		"code_challenge": {base64.RawURLEncoding.EncodeToString(digest[:])}, "code_challenge_method": {"S256"},
	}
	session := &Session{
		provider: p, ownerID: ownerID, nonce: state, verifier: verifier,
		expires: time.Now().Add(enrollmentLifetime), enrollment: identity.Enrollment{
			ID:         id,
			ProviderID: identity.Forgejo, State: "pending", VerificationURL: p.config.Base + "/login/oauth/authorize?" + query.Encode(),
		},
	}
	session.timer = time.AfterFunc(time.Until(session.expires), session.expireEnrollment)
	return session, nil
}

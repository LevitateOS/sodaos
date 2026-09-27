package forgejo

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"

	upstream "github.com/levitateos/sodaos/internal/forgejo"
	"github.com/levitateos/sodaos/internal/identity"
	"github.com/levitateos/sodaos/internal/strictjson"
)

// Credential is encrypted broker custody data, never browser or worker delivery.
type Credential struct {
	Access  string `json:"access"`
	Refresh string `json:"refresh"`
	Expiry  int64  `json:"expiry"`
	UserID  int64  `json:"user_id,string"`
	Scopes  string `json:"scopes"`
}

func explicitScopes(value string) bool {
	scopes := strings.Fields(value)
	if len(scopes) != 2 {
		return false
	}
	return (scopes[0] == "read:user" && scopes[1] == "write:repository") ||
		(scopes[1] == "read:user" && scopes[0] == "write:repository")
}

func Decode(data []byte, expectedOwner int64) (Credential, error) {
	var credential Credential
	if len(data) > 65536 || strictjson.Decode(bytes.NewReader(data), &credential) != nil {
		return Credential{}, identity.ErrDenied
	}
	if !validCredential(credential, expectedOwner) {
		return Credential{}, identity.ErrDenied
	}
	return credential, nil
}

func validCredential(credential Credential, owner int64) bool {
	return owner > 0 && credential.UserID == owner && credential.Access != "" &&
		credential.Refresh != "" && credential.Expiry > 0 && explicitScopes(credential.Scopes)
}

func (p *Provider) verify(ctx context.Context, owner int64, token upstream.TokenResponse) (identity.Connection, Credential, error) {
	scopes, err := p.client.GrantScopes(ctx, p.config.ClientID, p.secret, token.Access, owner)
	if err != nil || !explicitScopes(scopes) {
		return identity.Connection{}, Credential{}, identity.ErrDenied
	}
	_, email, err := p.client.GitIdentity(ctx, token.Access, owner)
	if err != nil {
		return identity.Connection{}, Credential{}, identity.ErrDenied
	}
	return identity.Connection{ProviderID: identity.Forgejo, OwnerID: owner, Email: email, State: identity.Ready},
		Credential{Access: token.Access, Refresh: token.Refresh, Expiry: token.ExpiresAt, UserID: owner, Scopes: scopes}, nil
}

// Refresh performs one native renewal. Any uncertain or invalid renewed grant
// returns no credential; callers must retire the old seed instead of replaying it.
func (p *Provider) Refresh(ctx context.Context, owner int64, data []byte) (identity.Connection, []byte, error) {
	old, err := Decode(data, owner)
	if err != nil {
		return identity.Connection{}, nil, identity.ErrDenied
	}
	token, err := p.client.RefreshGrant(ctx, p.config.ClientID, p.secret, old.Refresh)
	if err != nil {
		return identity.Connection{}, nil, identity.ErrUncertain
	}
	connection, credential, err := p.verify(ctx, owner, token)
	if err != nil {
		return identity.Connection{}, nil, identity.ErrUncertain
	}
	encoded, err := json.Marshal(credential)
	if err != nil {
		return identity.Connection{}, nil, identity.ErrUncertain
	}
	return connection, encoded, nil
}

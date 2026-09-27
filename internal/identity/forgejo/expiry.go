package forgejo

import (
	"time"

	"github.com/levitateos/sodaos/internal/identity"
)

var _ identity.CredentialRefresher = (*Provider)(nil)

// CredentialExpiry validates native custody against its owner before admission.
func (p *Provider) CredentialExpiry(owner int64, data []byte) (time.Time, error) {
	credential, err := Decode(data, owner)
	if err != nil {
		return time.Time{}, err
	}
	return time.Unix(credential.Expiry, 0), nil
}

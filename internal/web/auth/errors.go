package auth

import "errors"

var (
	// ErrRepositoryConsent means the acting grant lacks user/repository scopes.
	ErrRepositoryConsent = errors.New("repository and user consent required")
	// ErrProviderIdentity means Forgejo's acting user differs from the Soda session.
	ErrProviderIdentity = errors.New("provider identity differs from Soda session")
)

package identity

import (
	"context"
	"time"
)

// EnrollmentSession confines provider-owned OAuth and cached token maintenance.
type EnrollmentSession interface {
	Snapshot() Enrollment
	Finish(context.Context) (Connection, []byte, error)
	Close() error
}

// CodeEnrollmentSession completes a native authorization-code flow. Completion
// stays on the private administration interface and retains no caller identity.
type CodeEnrollmentSession interface {
	EnrollmentSession
	Complete(context.Context, string, string) error
}

// Provider enrollment receives the trusted Soda owner, never a caller-selected
// upstream identity. Providers that bind a native user verify it on completion.
type Provider interface {
	Start(context.Context, int64) (EnrollmentSession, error)
}

// CredentialRefresher maintains broker-held custody only. Its credential bytes
// must never become browser responses or execution credential delivery.
type CredentialRefresher interface {
	CredentialExpiry(int64, []byte) (time.Time, error)
	Refresh(context.Context, int64, []byte) (Connection, []byte, error)
}

// Runtime confirms termination of the entire registered execution boundary.
type Runtime interface {
	Validate(context.Context, Lease) error
	Stop(context.Context, Lease) error
	Finish(context.Context, Lease) ([]byte, error)
}

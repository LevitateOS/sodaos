package identity

import "context"

// EnrollmentSession confines provider-owned OAuth and cached token maintenance.
type EnrollmentSession interface {
	Snapshot() Enrollment
	Finish(context.Context) (Connection, []byte, error)
	Close() error
}

type Provider interface {
	Start(context.Context, string) (EnrollmentSession, error)
}

// Runtime confirms termination of the entire registered execution boundary.
type Runtime interface {
	Validate(context.Context, Lease) error
	Stop(context.Context, Lease) error
	Finish(context.Context, Lease) ([]byte, error)
}

package identity

// GitAcquireRequest selects a native remote; caller authority comes only from
// the attested runtime and its trusted Acquire fields.
type GitAcquireRequest struct {
	// Factory reads stay within the authorizing run's native repository.
	ExpectedRepositoryID int64          `json:"expected_repository_id,string,omitempty"`
	Acquire              AcquireRequest `json:"acquire"`
	Owner                string         `json:"owner"`
	Repository           string         `json:"repository"`
}

// GitSession returns attribution and a lease, never an upstream credential.
type GitSession struct {
	Lease Lease  `json:"lease"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

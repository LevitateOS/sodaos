package forgejo

// ResolutionReadback is the exact observed resolution state of one native
// review thread: the anchored root, whether native shows it resolved, and the
// complete conversation digest behind that state. It is read-only evidence
// for reconciliation; reading never grants mutation authority, and resolved
// state alone is never semantic proof of an accepted fix.
type ResolutionReadback struct {
	RootCommentID int64
	Resolved      bool
	Digest        string
}

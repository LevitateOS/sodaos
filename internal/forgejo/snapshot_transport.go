package forgejo

import (
	"context"

	extensions "forgejo.org/extension-sdk"
)

// BackgroundSnapshotReader is the thin Soda transport for permission-checked
// native snapshots (FT10, F-read). It maps bounded Soda requests onto the
// Fountain SDK background call and maps its typed evidence onto the
// bracket-bound snapshot; BracketedRead validates every digest, locator and
// completeness flag before binding the bracket revision.
//
// ActorID is the bound native actor whose credential is presented privately
// alongside each read. The host verifies the credential belongs to that
// actor and intersects the enrolled binding with current native read
// authority; the read grants no mutation kind.
type BackgroundSnapshotReader struct {
	Client  extensions.BackgroundClient
	ActorID string
}

// ReadNativeRevision observes the atomic native revision and idle/busy
// state through owning-installation admission.
func (r *BackgroundSnapshotReader) ReadNativeRevision(ctx context.Context) (extensions.NativeRevisionObservation, error) {
	if r == nil || r.Client == nil {
		return extensions.NativeRevisionObservation{}, ErrInvalidSnapshot
	}
	return r.Client.ReadNativeRevision(ctx)
}

// ReadSnapshot reads one native snapshot through the background channel.
// The revision is bound by BracketedRead after the bracket holds, not here.
func (r *BackgroundSnapshotReader) ReadSnapshot(ctx context.Context, credential extensions.CredentialFile, req SnapshotRequest) (NativeSnapshot, error) {
	if r == nil || r.Client == nil {
		return NativeSnapshot{}, ErrInvalidSnapshot
	}
	if err := ValidateRequest(req); err != nil {
		return NativeSnapshot{}, err
	}
	if !decimalID(r.ActorID) {
		return NativeSnapshot{}, ErrInvalidSnapshot
	}
	mapped := extensions.SnapshotRequest{
		RepositoryID: req.RepositoryID,
		ActorID:      r.ActorID,
		Families:     make([]string, 0, len(req.Families)),
		IssueIndex:   req.IssueIndex,
		PullNumber:   req.PullNumber,
		CommentIDs:   req.CommentIDs,
		SHA:          req.SHA,
		Refs:         req.Refs,
		Limit:        req.Limit,
		Cursor:       req.Cursor,
	}
	for _, family := range req.Families {
		mapped.Families = append(mapped.Families, string(family))
	}
	answer, err := r.Client.ReadSnapshot(ctx, credential, mapped)
	if err != nil {
		return NativeSnapshot{}, err
	}
	return NativeSnapshot{
		RepositoryID: answer.RepositoryID,
		Issue:        answer.Issue,
		Comments:     answer.Comments,
		Dependencies: answer.Dependencies,
		Pull:         answer.Pull,
		Reviews:      answer.Reviews,
		Checks:       answer.Checks,
		Refs:         answer.Refs,
	}, nil
}

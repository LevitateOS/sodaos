package forgejo

import (
	"errors"
	"strconv"

	"github.com/levitateos/sodaos/internal/factory"
)

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

// ReadbackResolution verifies one expected conditional resolve request
// against its exact snapshot thread evidence. Visible, complete threads whose
// root and unchanged conversation digest match the request read back their
// native resolved flag; anything else refuses or reports unresolved:
//
//   - hidden or incomplete thread evidence blocks (never confirmed);
//   - a root, digest or scope mismatch refuses (changed conversation, wrong
//     target, deleted root recorded as digest mismatch);
//   - a matching but unresolved thread reads back unresolved (caller waits).
//
// The caller supplies the thread from the already-fetched FW17 snapshot
// pages for the request's PR; this adapter performs no fetch and no write.
func ReadbackResolution(expected factory.ResolveRequest, thread ThreadEvidence) (ResolutionReadback, error) {
	var empty ResolutionReadback
	if !expected.Resolved {
		return empty, errors.New("resolve readback needs a resolving request")
	}
	if !validThreadDigest(expected.Digest) {
		return empty, errors.New("resolve readback needs its expected conversation digest")
	}
	if !thread.Visible {
		return empty, errors.New("resolve readback blocked on hidden thread evidence")
	}
	if !thread.Complete {
		return empty, errors.New("resolve readback blocked on incomplete thread evidence")
	}
	if !decimalID(thread.RootID) {
		return empty, errors.New("resolve readback lacks its anchored root")
	}
	root, err := strconv.ParseInt(thread.RootID, 10, 64)
	if err != nil || root <= 0 {
		return empty, errors.New("resolve readback lacks its anchored root")
	}
	if root != expected.RootCommentID {
		return empty, errors.New("resolve readback target differs from its recorded root")
	}
	if len(thread.Members) != thread.MemberTotal {
		return empty, errors.New("resolve readback blocked on incomplete thread evidence")
	}
	if !validThreadDigest(thread.Digest) || thread.Digest != expected.Digest {
		return empty, errors.New("resolve conversation changed under its request")
	}
	return ResolutionReadback{RootCommentID: root, Resolved: thread.Resolved, Digest: thread.Digest}, nil
}

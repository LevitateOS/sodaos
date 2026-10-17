package api

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/forgejo"
)

// fakeCompletionSourceStoreNATIVE96 is the NATIVE96-local deterministic
// stand-in for the NATIVE95 store exports (CompletionSourceContext +
// CaptureCompletionLocalAtSource): it serves one fixed fixture local,
// checks the expected context by canonical validated JSON equality,
// and binds the returned local to the observed native tip exactly as
// the frozen source binding specifies. It is a test double, never a
// qualification of the real store authority (NATIVE95 proves that
// separately); it goes away once the sibling exports land.
type fakeCompletionSourceStoreNATIVE96 struct {
	local          factory.CompletionLocal
	rejectAtSource error
	contextCalls   int
	atSourceCalls  int
}

func newFakeCompletionSourceStoreNATIVE96(local factory.CompletionLocal) *fakeCompletionSourceStoreNATIVE96 {
	return &fakeCompletionSourceStoreNATIVE96{local: local}
}

func (f *fakeCompletionSourceStoreNATIVE96) CompletionSourceContext(context.Context, int64) (factory.CompletionAuthority, error) {
	f.contextCalls++
	if f.local.Material == nil {
		return factory.CompletionAuthority{}, errors.New("NATIVE96 fake store lacks its fixture local")
	}
	return f.local.Material.Authority, nil
}

func (f *fakeCompletionSourceStoreNATIVE96) CaptureCompletionLocalAtSource(_ context.Context, _ int64, expected factory.CompletionAuthority, tip string) (factory.CompletionLocal, error) {
	f.atSourceCalls++
	if f.rejectAtSource != nil {
		return factory.CompletionLocal{}, f.rejectAtSource
	}
	if err := expected.Validate(); err != nil {
		return factory.CompletionLocal{}, err
	}
	want, err := json.Marshal(f.local.Material.Authority)
	if err != nil {
		return factory.CompletionLocal{}, err
	}
	got, err := json.Marshal(expected)
	if err != nil {
		return factory.CompletionLocal{}, err
	}
	if string(got) != string(want) {
		return factory.CompletionLocal{}, errors.New("NATIVE96 fake store: context mismatch, no fallback")
	}
	if !factory.ValidCommit(tip) {
		return factory.CompletionLocal{}, errors.New("NATIVE96 fake store: invalid tip")
	}
	local := f.local
	local.SourceCommit = tip
	if err := local.Validate(); err != nil {
		return factory.CompletionLocal{}, err
	}
	return local, nil
}

// The serving capture binds the CURRENT native tip — a tip distinct
// from the accepted requirement source proves it is never the
// requirement or event commit — through exactly one refs-only read:
// at most 3 native calls with no page-counter charge.
func TestCaptureLocalBindsCurrentTip(t *testing.T) {
	ctx := context.Background()
	tip := strings.Repeat("d", 40)
	fake := completionTwoPageFake()
	fake.refTip = tip
	source := &CompletionSnapshotSource{Reader: fake, Store: newFakeCompletionSourceStoreNATIVE96(completionTestLocal())}
	local, err := source.CaptureLocal(ctx, 7)
	if err != nil {
		t.Fatalf("capture: %v", err)
	}
	if local.SourceCommit != tip {
		t.Fatalf("source=%q, want the exact current tip %q", local.SourceCommit, tip)
	}
	if local.Material.Authority.RequirementSourceCommit == tip {
		t.Fatal("capture bound the requirement source instead of the current tip")
	}
	if err := local.Validate(); err != nil {
		t.Fatalf("captured local invalid: %v", err)
	}
	req := fake.lastReq
	if len(req.Families) != 1 || req.Families[0] != forgejo.FamilyRefs ||
		len(req.Refs) != 1 || req.Refs[0] != "refs/heads/main" ||
		req.IssueIndex != "" || req.PullNumber != "" || len(req.CommentIDs) != 0 ||
		req.SHA != "" || req.Ancestry != nil || req.Limit != 0 || req.Cursor != "" {
		t.Fatalf("req=%+v, want the refs-only read for the target branch", req)
	}
	if fake.calls != 1 || fake.revs != 2 {
		t.Fatalf("calls=%d revs=%d, want exactly one refs snapshot and two observations", fake.calls, fake.revs)
	}
}

func TestCaptureLocalRequiresStoreAndReader(t *testing.T) {
	ctx := context.Background()
	var nilSource *CompletionSnapshotSource
	if _, err := nilSource.CaptureLocal(ctx, 7); !errors.Is(err, forgejo.ErrUnavailable) {
		t.Fatalf("err=%v, want the unavailable refusal without a source", err)
	}
	if _, err := (&CompletionSnapshotSource{Store: newFakeCompletionSourceStoreNATIVE96(completionTestLocal())}).CaptureLocal(ctx, 7); !errors.Is(err, forgejo.ErrUnavailable) {
		t.Fatalf("err=%v, want the unavailable refusal without a reader", err)
	}
	if _, err := (&CompletionSnapshotSource{Reader: completionTwoPageFake()}).CaptureLocal(ctx, 7); !errors.Is(err, forgejo.ErrUnavailable) {
		t.Fatalf("err=%v, want the unavailable refusal without a store", err)
	}
}

func TestCaptureLocalRefusesHiddenRef(t *testing.T) {
	fake := completionTwoPageFake()
	fake.refTip = strings.Repeat("d", 40)
	fake.hideRef = true
	source := &CompletionSnapshotSource{Reader: fake, Store: newFakeCompletionSourceStoreNATIVE96(completionTestLocal())}
	if _, err := source.CaptureLocal(context.Background(), 7); !errors.Is(err, forgejo.ErrIncompleteSnapshot) {
		t.Fatalf("err=%v, want the refusal for the redacted ref", err)
	}
}

func TestCaptureLocalRefusesMissingRef(t *testing.T) {
	fake := completionTwoPageFake()
	source := &CompletionSnapshotSource{Reader: fake, Store: newFakeCompletionSourceStoreNATIVE96(completionTestLocal())}
	if _, err := source.CaptureLocal(context.Background(), 7); !errors.Is(err, forgejo.ErrIncompleteSnapshot) {
		t.Fatalf("err=%v, want the refusal for the missing ref", err)
	}
}

func TestCaptureLocalRefusesBusyNative(t *testing.T) {
	fake := completionTwoPageFake()
	fake.refTip = strings.Repeat("d", 40)
	fake.idle = false
	source := &CompletionSnapshotSource{Reader: fake, Store: newFakeCompletionSourceStoreNATIVE96(completionTestLocal())}
	if _, err := source.CaptureLocal(context.Background(), 7); !errors.Is(err, forgejo.ErrNativeBusy) {
		t.Fatalf("err=%v, want the busy refusal without fallback", err)
	}
}

func TestCaptureLocalRefusesMalformedRef(t *testing.T) {
	fake := completionTwoPageFake()
	fake.refTip = "short-not-an-oid"
	source := &CompletionSnapshotSource{Reader: fake, Store: newFakeCompletionSourceStoreNATIVE96(completionTestLocal())}
	if _, err := source.CaptureLocal(context.Background(), 7); !errors.Is(err, forgejo.ErrInvalidSnapshot) {
		t.Fatalf("err=%v, want the malformed refusal without fallback", err)
	}
}

func TestCaptureLocalRefusesStaleRef(t *testing.T) {
	fake := completionTwoPageFake()
	fake.refTip = strings.Repeat("d", 40)
	fake.moveRev = func(calls int) int64 {
		if calls >= 1 {
			return 42
		}
		return 41
	}
	source := &CompletionSnapshotSource{Reader: fake, Store: newFakeCompletionSourceStoreNATIVE96(completionTestLocal())}
	if _, err := source.CaptureLocal(context.Background(), 7); !errors.Is(err, forgejo.ErrStaleSnapshot) {
		t.Fatalf("err=%v, want the ref-move refusal without fallback", err)
	}
}

// An authoritative-context mismatch propagates as the store's error
// with exactly one context read and one source read: no retry, no
// unqualified fallback capture.
func TestCaptureLocalPropagatesContextMismatchWithoutFallback(t *testing.T) {
	sentinel := errors.New("NATIVE96 fake authority moved after context")
	fake := completionTwoPageFake()
	fake.refTip = strings.Repeat("d", 40)
	bound := newFakeCompletionSourceStoreNATIVE96(completionTestLocal())
	bound.rejectAtSource = sentinel
	source := &CompletionSnapshotSource{Reader: fake, Store: bound}
	_, err := source.CaptureLocal(context.Background(), 7)
	if !errors.Is(err, sentinel) {
		t.Fatalf("err=%v, want the exact store mismatch error", err)
	}
	if bound.contextCalls != 1 || bound.atSourceCalls != 1 {
		t.Fatalf("context=%d atSource=%d, want exactly one attempt each, never a fallback", bound.contextCalls, bound.atSourceCalls)
	}
}

// A tip change across the capture pair surfaces as non-equal locals:
// the drive's confirm step defers on exactly this signal.
func TestCaptureLocalPairDetectsTipChange(t *testing.T) {
	ctx := context.Background()
	fake := completionTwoPageFake()
	bound := newFakeCompletionSourceStoreNATIVE96(completionTestLocal())
	source := &CompletionSnapshotSource{Reader: fake, Store: bound}
	fake.refTip = strings.Repeat("d", 40)
	first, err := source.CaptureLocal(ctx, 7)
	if err != nil {
		t.Fatalf("first capture: %v", err)
	}
	again, err := source.CaptureLocal(ctx, 7)
	if err != nil {
		t.Fatalf("repeat capture: %v", err)
	}
	if !again.Equal(first) {
		t.Fatal("repeat capture at a stable tip differs from its pair")
	}
	fake.refTip = strings.Repeat("e", 40)
	moved, err := source.CaptureLocal(ctx, 7)
	if err != nil {
		t.Fatalf("moved capture: %v", err)
	}
	if moved.Equal(first) {
		t.Fatal("capture pair across a tip change compares equal instead of deferring")
	}
}

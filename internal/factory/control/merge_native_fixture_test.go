package control_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	extensions "forgejo.org/extension-sdk"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/factory/control"
	"github.com/levitateos/sodaos/internal/forgejo"
	forgejopublish "github.com/levitateos/sodaos/internal/forgejo/publish"
)

type nativeST12Config struct {
	nativeST10Config
	MergeBound bool   `json:"merge_bound"`
	FountainDB string `json:"fountain_db"`
}

func loadNativeST12(t *testing.T) nativeST12Config {
	t.Helper()
	raw := os.Getenv("SODA_ST12_NATIVE")
	if raw == "" {
		t.Skip("native proof NOT RUN: SODA_ST12_NATIVE is not configured")
	}
	var cfg nativeST12Config
	nativeMust(t, json.Unmarshal([]byte(raw), &cfg))
	if cfg.FountainURL == "" || cfg.Socket == "" || cfg.TokenFile == "" || cfg.Repository <= 0 || cfg.ActorID <= 0 || cfg.TokenID <= 0 {
		t.Fatal("native fixture requires explicit repository and merger inputs")
	}
	if cfg.CreatorID <= 0 || cfg.CreatorID == cfg.ActorID || cfg.CreatorTokenFile == "" {
		t.Fatal("native fixture requires a distinct creator and restricted creator_token_file")
	}
	if cfg.ReviewerID <= 0 || cfg.ReviewerID == cfg.ActorID || cfg.ReviewerID == cfg.CreatorID || cfg.ReviewerTokenFile == "" {
		t.Fatal("native fixture requires a distinct reviewer and restricted reviewer_token_file")
	}
	if !factory.ValidTargetBranch(cfg.BaseBranch) {
		t.Fatal("native fixture requires exact base_branch")
	}
	if !cfg.MergeBound {
		t.Fatal("native proof suite requires the pull_request.merge binding")
	}
	if cfg.FountainDB == "" {
		t.Fatal("native fixture requires its fountain_db path")
	}
	return cfg
}

func nativeMergeReceipt(t *testing.T, label string, value any) {
	t.Helper()
	root := os.Getenv("ST12_RECEIPT_DIR")
	if root == "" {
		t.Fatal("ST12_RECEIPT_DIR is required to retain native proof")
	}
	if !filepath.IsAbs(root) {
		t.Fatal("native receipt directory must be absolute")
	}
	nativeMust(t, os.MkdirAll(root, 0o700))
	raw, err := json.MarshalIndent(value, "", "  ")
	nativeMust(t, err)
	nativeMust(t, os.WriteFile(filepath.Join(root, label+".json"), append(raw, '\n'), 0o600))
}

// nativeMergeDrain waits until the native revision is unchanged for a
// beat: the Idle flag does not cover async completion effects (one
// merge lands two advances within a second, all reported idle), and a
// conditioned submit opened during that churn is refused stale. Read
// errors count as motion, never as quiescence.
func nativeMergeDrain(t *testing.T, bg *forgejo.ServiceBackground) extensions.NativeRevisionObservation {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	var last extensions.NativeRevisionObservation
	stable := time.Now()
	first := true
	for {
		if err := nativeContextError(ctx); err != nil {
			t.Fatalf("native revision never settled: %v", err)
		}
		revision, err := bg.ReadNativeRevision(ctx)
		if ctxErr := nativeContextError(ctx); ctxErr != nil {
			t.Fatalf("native revision never settled: %v", ctxErr)
		}
		if err != nil || first || revision.Revision != last.Revision {
			last = revision
			stable = time.Now()
			first = false
			if !nativeContextWait(ctx, 100*time.Millisecond) {
				t.Fatalf("native revision never settled: %v", nativeContextError(ctx))
			}
			continue
		}
		if time.Since(stable) >= 1500*time.Millisecond {
			if err := nativeContextError(ctx); err != nil {
				t.Fatalf("native revision never settled: %v", err)
			}
			return revision
		}
		if !nativeContextWait(ctx, 100*time.Millisecond) {
			t.Fatalf("native revision never settled: %v", nativeContextError(ctx))
		}
	}
}

func nativeContextError(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if deadline, ok := ctx.Deadline(); ok && !time.Now().Before(deadline) {
		return context.DeadlineExceeded
	}
	return nil
}

func nativeContextWait(ctx context.Context, delay time.Duration) bool {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return nativeContextError(ctx) == nil
	}
}

// nativeMergeIdle waits for a quiescent host before a conditioned
// submit opens its bracket. Stability subsumes the Idle flag.
func nativeMergeIdle(t *testing.T, bg *forgejo.ServiceBackground) extensions.NativeRevisionObservation {
	t.Helper()
	return nativeMergeDrain(t, bg)
}

// nativeMergeStaleRefusal reports whether an outcome is the host-churn
// refusal (certain no-effect) rather than a scenario outcome.
func nativeMergeStaleRefusal(outcome factory.OperationOutcome) bool {
	return outcome.Effect == factory.OpEffectNotCommitted && outcome.Reason == "stale_native_revision"
}

// nativeMergeBusyError reports whether err is the transient busy-host
// signal on a direct native call: an HTTP 503 or a native_busy wait.
func nativeMergeBusyError(err error) bool {
	var status *forgejopublish.StatusError
	if errors.As(err, &status) && status.Status == http.StatusServiceUnavailable {
		return true
	}
	var wait *factory.PublicationWait
	return errors.As(err, &wait) && wait.Reason == "native_busy"
}

// nativeMergeCall retries one idempotent native read (or same-identity
// submit) until it answers; only the transient busy-host signal
// retries, everything else fails fast.
func nativeMergeCallContext[T any](ctx context.Context, call func(context.Context) (T, error)) (T, error) {
	var zero T
	for {
		if err := nativeContextError(ctx); err != nil {
			return zero, err
		}
		value, err := call(ctx)
		if ctxErr := nativeContextError(ctx); ctxErr != nil {
			return zero, ctxErr
		}
		if err == nil {
			return value, nil
		}
		if !nativeMergeBusyError(err) {
			return zero, err
		}
		if !nativeContextWait(ctx, 200*time.Millisecond) {
			return zero, nativeContextError(ctx)
		}
		if ctxErr := nativeContextError(ctx); ctxErr != nil {
			return zero, ctxErr
		}
	}
}

func nativeMergeCall[T any](t *testing.T, label string, call func(context.Context) (T, error)) T {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	value, err := nativeMergeCallContext(ctx, call)
	if err != nil {
		t.Fatalf("native %s unanswered: %v", label, err)
	}
	return value
}

// nativeMergeSubmitReviewCommitted submits one conditional review and
// requires its commit. An uncertain submit retries under its recorded
// identity; a certain stale refusal (host churn, no effect) reopens
// under a fresh identity.
func nativeMergeSubmitReviewCommitted(t *testing.T, r *forgejo.Reviewer, bg *forgejo.ServiceBackground, base factory.ReviewWork) factory.OperationOutcome {
	t.Helper()
	ctx := context.Background()
	for attempt := 0; attempt < 10; attempt++ {
		nativeMergeIdle(t, bg)
		w := base
		w.OperationID = "st12-" + factory.NewID()
		w = nativeReviewObserved(t, r, w)
		stale := false
		for i := 0; i < 5; i++ {
			outcome, err := r.SubmitReview(ctx, w)
			if err == nil {
				if outcome.Effect == factory.OpEffectCommitted {
					return outcome
				}
				if !nativeMergeStaleRefusal(outcome) {
					t.Fatalf("native review refused: %+v", outcome)
				}
				stale = true
				break
			}
			var refused *factory.PublicationRefusal
			if errors.As(err, &refused) {
				t.Fatalf("native review refused: %v", err)
			}
			time.Sleep(time.Duration(200*(i+1)) * time.Millisecond)
		}
		if !stale {
			t.Fatal("native review submit never confirmed")
		}
		t.Logf("native review reopened after host churn (attempt %d)", attempt+1)
	}
	t.Fatal("native review never committed")
	return factory.OperationOutcome{}
}

// nativeMergeSubmitMergeCommitted submits one conditional merge through
// the production client and requires its commit, reopening under a
// fresh identity only on certain stale refusal like the review path.
func nativeMergeSubmitMergeCommitted(t *testing.T, m control.MergeExecutor, bg *forgejo.ServiceBackground, base factory.MergeWork) factory.OperationOutcome {
	t.Helper()
	ctx := context.Background()
	for attempt := 0; attempt < 10; attempt++ {
		revision := nativeMergeIdle(t, bg)
		w := base
		w.OperationID = "st12-merge-" + factory.NewID()
		w.NativeRev = revision.Revision
		w.NotAfter = time.Now().Add(5 * time.Minute).Unix()
		stale := false
		for i := 0; i < 5; i++ {
			outcome, err := m.SubmitMerge(ctx, w)
			if err == nil {
				if outcome.Effect == factory.OpEffectCommitted {
					return outcome
				}
				if !nativeMergeStaleRefusal(outcome) {
					t.Fatalf("native merge refused: %+v", outcome)
				}
				stale = true
				break
			}
			var refused *factory.PublicationRefusal
			if errors.As(err, &refused) {
				t.Fatalf("native merge refused: %v", err)
			}
			time.Sleep(time.Duration(200*(i+1)) * time.Millisecond)
		}
		if !stale {
			t.Fatal("native merge submit never confirmed")
		}
		t.Logf("native merge reopened after host churn (attempt %d)", attempt+1)
	}
	t.Fatal("native merge never committed")
	return factory.OperationOutcome{}
}

// nativeMergeSubmitRawCommitted submits one merge intent directly and
// returns its committed record, reopening under a fresh identity only
// on certain stale refusal. Snapshot probes use the raw path to pin
// the exact receipt wire shape.
func nativeMergeSubmitRawCommitted(t *testing.T, bg *forgejo.ServiceBackground, tokenFile, authRev string, actorID, repoID int64, payload []byte) extensions.OperationRecord {
	t.Helper()
	for attempt := 0; attempt < 10; attempt++ {
		revision := nativeMergeIdle(t, bg)
		intent := extensions.OperationIntent{
			OperationID: "st12-raw-" + factory.NewID(), ActorID: strconv.FormatInt(actorID, 10),
			RepositoryID: strconv.FormatInt(repoID, 10), Kind: factory.OpMerge,
			AuthorizationRevision: authRev, ExpectedNativeRevision: revision.Revision,
			NotAfter: time.Now().Add(5 * time.Minute).Unix(), Payload: payload,
		}
		record := nativeMergeCall(t, "merge submit", func(callCtx context.Context) (extensions.OperationRecord, error) {
			return bg.SubmitOperation(callCtx, extensions.CredentialFile(tokenFile), intent)
		})
		if record.EffectState == factory.OpEffectCommitted {
			return record
		}
		if record.EffectState != factory.OpEffectNotCommitted || record.ReasonCode != "stale_native_revision" {
			t.Fatalf("native merge refused: effect=%s reason=%s completion=%s", record.EffectState, record.ReasonCode, record.CompletionState)
		}
		t.Logf("native merge reopened after host churn (attempt %d)", attempt+1)
	}
	t.Fatal("native merge never committed")
	return extensions.OperationRecord{}
}

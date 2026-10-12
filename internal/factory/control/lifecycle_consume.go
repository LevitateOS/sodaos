package control

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"time"

	sdk "forgejo.org/extension-sdk"
	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/store"
)

// LifecycleExecutor is the coordinator's narrow lifecycle surface: the
// enrolled native actor behind the lifecycle credential, a current
// native revision observation, closed-target resolution for completion
// notes, and the single conditional submit with lost-reply lookup per
// attempt. Implementations own all native effects; the consumer owns
// durable identity, the finite attempt budget and reconciliation.
// Lookup is read-only reconciliation by operation identity.
type LifecycleExecutor interface {
	EnrolledActor(ctx context.Context) (int64, error)
	ObserveLifecycleRevision(ctx context.Context, repositoryID int64) (int64, error)
	ResolveNoteTarget(ctx context.Context, repositoryID, issueIndex int64) (int64, error)
	SubmitLifecycle(ctx context.Context, work factory.LifecycleWork) (factory.OperationOutcome, error)
	LookupLifecycle(ctx context.Context, operationID string) (factory.OperationOutcome, error)
}

// LifecycleEventInput is one proven production completion that may fire
// a configured optional effect. SourceID is the durable event identity
// (revision plus task key, publication ID, merge ID); AuthRevision
// carries the event's current authorization facts. IssueIndex carries
// the original issue index for verified merges; every other event
// carries the exact native target in IssueID.
type LifecycleEventInput struct {
	Event        string
	SourceID     string
	RepositoryID int64
	IssueID      int64
	IssueIndex   int64
	AuthRevision string
	MergeRef     string
	Closed       bool
}

// lifecycleNotAfter bounds one lifecycle operation lifetime.
const lifecycleNotAfter = 5 * time.Minute

// lifecycleCallTimeout bounds one native call inside the consumer so
// optional work cannot stall default completion latency. Timeouts
// reconcile as uncertain under the persisted identity.
const lifecycleCallTimeout = 60 * time.Second

// withLifecycleTimeout scopes one native call.
func withLifecycleTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, lifecycleCallTimeout)
}

// LifecycleNewIssueKeys filters submitted task keys to actually newly
// created native issues: keys bound to reused task issues or delivered
// closed history are never creation events.
func LifecycleNewIssueKeys(keys []string, g factory.IntentGeneration) []string {
	var out []string
	for _, key := range keys {
		if _, reused := g.TaskIssueIDs[key]; reused {
			continue
		}
		if _, delivered := g.Delivered[key]; delivered {
			continue
		}
		out = append(out, key)
	}
	return out
}

// LifecycleActionID derives the durable action identity from the event
// identity so duplicate deliveries journal once and reconcile by ID.
func LifecycleActionID(in LifecycleEventInput) string {
	return "lifecycle:" + in.Event + ":" + in.SourceID
}

// LifecycleOperationID derives one attempt's native operation identity.
// Only native-identifier characters survive; anything else folds to '-'
// so task keys can never break the operation identity charset.
func LifecycleOperationID(actionID string, attempt int) string {
	var b strings.Builder
	b.WriteString(actionID)
	b.WriteString(":attempt-")
	b.WriteString(strconv.Itoa(attempt))
	raw := b.String()
	out := make([]byte, len(raw))
	for i := 0; i < len(raw); i++ {
		c := raw[i]
		switch {
		case c >= 'A' && c <= 'Z', c >= 'a' && c <= 'z', c >= '0' && c <= '9', c == '-', c == '_', c == '.', c == ':':
			out[i] = c
		default:
			out[i] = '-'
		}
	}
	return string(out)
}

// ClassifyLifecycleOutcome maps one reconciled native answer to its
// journal state: committed effects deliver; authority loss and ordered
// cancellation deny (no retry); other definitive refusals fail (one
// retry allowed after re-observation); absence and indeterminate
// answers stay uncertain.
func ClassifyLifecycleOutcome(outcome factory.OperationOutcome) string {
	if outcome.NotObserved {
		return factory.LifecycleResultUncertain
	}
	switch outcome.Effect {
	case factory.OpEffectCommitted:
		return factory.LifecycleResultDelivered
	case factory.OpEffectNotCommitted:
		if factory.LifecycleDenyReason(outcome.Reason) {
			return factory.LifecycleResultDenied
		}
		return factory.LifecycleResultFailed
	default:
		return factory.LifecycleResultUncertain
	}
}

// lifecycleAttempt submits attempt n for every unsettled target of one
// journaled action under a fresh operation identity and journals the
// reconciled result. Transport uncertainty is looked up under that
// same identity, never recreated under a new one.
func lifecycleAttempt(ctx context.Context, s *store.Store, ex LifecycleExecutor, a factory.LifecycleAction, n int) (factory.LifecycleAction, error) {
	obsCtx, obsCancel := withLifecycleTimeout(ctx)
	rev, err := ex.ObserveLifecycleRevision(obsCtx, a.RepositoryID)
	obsCancel()
	if err != nil {
		return a, err
	}
	opID := LifecycleOperationID(a.ID, n)
	next := a.RecordActionAttempt(opID)
	next, err = s.UpdateAction(ctx, next, a.Attempts)
	if err != nil {
		return a, err
	}
	a = next
	work := factory.LifecycleWork{
		OperationID: opID, Event: a.Event, ActorID: a.ActorID, RepositoryID: a.RepositoryID,
		IssueID: a.IssueID, LabelID: a.LabelID, AssigneeID: a.AssigneeID, ReviewerIDs: append([]int64(nil), a.ReviewerIDs...),
		Note: a.Note, AuthRevision: a.AuthRevision, MergeRef: a.MergeRef, Closed: a.Closed,
		NativeRev: rev, NotAfter: time.Now().Add(lifecycleNotAfter).Unix(),
	}
	subCtx, subCancel := withLifecycleTimeout(ctx)
	outcome, err := ex.SubmitLifecycle(subCtx, work)
	subCancel()
	if err != nil {
		lookupCtx, lookupCancel := withLifecycleTimeout(ctx)
		lookup, lookupErr := ex.LookupLifecycle(lookupCtx, opID)
		lookupCancel()
		if lookupErr != nil {
			journaled, journalErr := s.UpdateAction(ctx, a.RecordActionResult(factory.LifecycleResultUncertain), a.Attempts)
			if journalErr != nil {
				return a, journalErr
			}
			return journaled, nil
		}
		outcome = lookup
	}
	journaled, err := s.UpdateAction(ctx, a.RecordActionResult(ClassifyLifecycleOutcome(outcome)), a.Attempts)
	if err != nil {
		return a, err
	}
	return journaled, nil
}

// lifecycleRetryAllowed reports whether a journaled action may submit:
// a pending first attempt, or the single retry once every target
// shows definitive no-effect. Uncertain or denied targets never ride
// a new operation identity: uncertainty reconciles by lookup, denial
// stops. Budget remains, capability is still proved, configuration is
// still enabled, and the enrolled actor still matches the journaled
// actor. Reconciliation lookups need none of this.
func lifecycleRetryAllowed(ctx context.Context, s *store.Store, ex LifecycleExecutor, a factory.LifecycleAction) bool {
	if a.Attempts >= factory.MaxLifecycleAttempts || a.Revoked {
		return false
	}
	if a.Attempts > 0 {
		targets := a.Targets()
		if len(targets) == 0 {
			return false
		}
		for _, target := range targets {
			if a.Results[target] != factory.LifecycleResultFailed {
				return false
			}
		}
	}
	if RequireCurrentCapability(ctx, s, a.Event) != nil {
		return false
	}
	cfg, err := s.LifecycleConfig(ctx, a.Event)
	if err != nil || !cfg.Enabled || cfg.RepositoryID != a.RepositoryID {
		return false
	}
	actorCtx, actorCancel := withLifecycleTimeout(ctx)
	enrolled, err := ex.EnrolledActor(actorCtx)
	actorCancel()
	if err != nil || enrolled != a.ActorID {
		return false
	}
	return true
}

// ReconcileLifecycleAction settles one journaled action without
// starting new work blindly: uncertain operations are looked up under
// their own identity, and at most one retry follows a definitive
// no-effect outcome. It is safe to call from duplicate deliveries,
// later passes and post-restart recovery.
func ReconcileLifecycleAction(ctx context.Context, s *store.Store, ex LifecycleExecutor, id string) (factory.LifecycleAction, error) {
	if s == nil || ex == nil || id == "" {
		return factory.LifecycleAction{}, factory.ErrLifecycleAction
	}
	a, err := s.LifecycleAction(ctx, id)
	if err != nil {
		return factory.LifecycleAction{}, err
	}
	if a.Revoked {
		return a, nil
	}
	uncertain := map[string][]string{}
	for _, target := range a.Targets() {
		if a.Results[target] == factory.LifecycleResultUncertain {
			op := a.Operations[target]
			if op == "" {
				continue
			}
			uncertain[op] = append(uncertain[op], target)
		}
	}
	for op, targets := range uncertain {
		lookupCtx, lookupCancel := withLifecycleTimeout(ctx)
		outcome, err := ex.LookupLifecycle(lookupCtx, op)
		lookupCancel()
		if err != nil {
			continue
		}
		state := ClassifyLifecycleOutcome(outcome)
		if state == factory.LifecycleResultUncertain {
			continue
		}
		next := a
		for _, target := range targets {
			next = next.RecordResult(target, state)
		}
		updated, err := s.UpdateAction(ctx, next, a.Attempts)
		if err != nil {
			if errors.Is(err, store.ErrLifecycleActionStale) {
				return s.LifecycleAction(ctx, id)
			}
			return a, err
		}
		a = updated
	}
	if a.Revoked {
		return a, nil
	}
	if a.Attempts == 0 {
		if !lifecycleRetryAllowed(ctx, s, ex, a) {
			return a, nil
		}
		return lifecycleAttempt(ctx, s, ex, a, 1)
	}
	if lifecycleRetryAllowed(ctx, s, ex, a) {
		return lifecycleAttempt(ctx, s, ex, a, a.Attempts+1)
	}
	return a, nil
}

// ConsumeLifecycleEvent journals and submits the configured optional
// effect for one proven production completion. The inline call makes at
// most one attempt; the single retry after definitive no-effect runs
// on a later tick via ReconcileLifecycleAction so optional work stays
// out of default-path latency. Disabled events, unproved capability,
// repository mismatch, unresolvable targets and actor/enrollment
// mismatch journal nothing. Duplicate deliveries reconcile the
// journaled action instead of submitting twice. Every optional failure
// returns without touching default state; hook sites must never gate
// default delivery on this result.
func ConsumeLifecycleEvent(ctx context.Context, s *store.Store, ex LifecycleExecutor, in LifecycleEventInput) error {
	if s == nil || ex == nil {
		return factory.ErrLifecycleAction
	}
	if !factory.ValidLifecycleEvent(in.Event) || in.SourceID == "" || in.RepositoryID <= 0 || in.AuthRevision == "" {
		return factory.ErrLifecycleAction
	}
	cfg, err := s.LifecycleConfig(ctx, in.Event)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil
		}
		return err
	}
	if !cfg.Enabled || cfg.RepositoryID != in.RepositoryID {
		return nil
	}
	if err := RequireCurrentCapability(ctx, s, in.Event); err != nil {
		if errors.Is(err, store.ErrCapabilityUnproved) {
			return nil
		}
		return err
	}
	actorCtx, actorCancel := withLifecycleTimeout(ctx)
	enrolled, err := ex.EnrolledActor(actorCtx)
	actorCancel()
	if err != nil {
		return err
	}
	if enrolled <= 0 || enrolled != cfg.ActorID {
		return factory.ErrLifecycleAction
	}
	issueID := in.IssueID
	if in.Event == factory.LifecycleEventVerifiedMerge {
		if in.IssueIndex <= 0 || !in.Closed {
			return factory.ErrLifecycleAction
		}
		targetCtx, targetCancel := withLifecycleTimeout(ctx)
		resolved, err := ex.ResolveNoteTarget(targetCtx, in.RepositoryID, in.IssueIndex)
		targetCancel()
		if err != nil {
			return err
		}
		issueID = resolved
	}
	a := factory.LifecycleAction{
		ID: LifecycleActionID(in), Event: in.Event, Action: factory.EventAction(in.Event),
		ActorID: enrolled, RepositoryID: in.RepositoryID, IssueID: issueID,
		LabelID: cfg.LabelID, AssigneeID: cfg.AssigneeID, ReviewerIDs: append([]int64(nil), cfg.ReviewerIDs...),
		Note: cfg.Note, AuthRevision: in.AuthRevision, MergeRef: in.MergeRef, Closed: in.Closed,
	}
	if err := a.Validate(); err != nil {
		return err
	}
	recorded, err := s.RecordAction(ctx, a)
	if err != nil {
		if errors.Is(err, store.ErrLifecycleActionConflict) {
			_, reconcileErr := ReconcileLifecycleAction(ctx, s, ex, a.ID)
			return reconcileErr
		}
		return err
	}
	if recorded.Revoked {
		return nil
	}
	if recorded.Attempts > 0 {
		_, err = ReconcileLifecycleAction(ctx, s, ex, a.ID)
		return err
	}
	_, err = lifecycleAttempt(ctx, s, ex, recorded, 1)
	return err
}

// ConsumeGeneratedIssues fires the configured label/assignee effect for
// every actually newly created native issue behind one adopted intent
// generation. Reused and delivered bindings never fire. Call sites run
// this after the default adoption persists; every key is attempted and
// the first error returns.
func ConsumeGeneratedIssues(ctx context.Context, s *store.Store, ex LifecycleExecutor, r factory.IntentRevision, g factory.IntentGeneration) error {
	if s == nil || ex == nil {
		return factory.ErrLifecycleAction
	}
	var receipt sdk.IssueGraphReceipt
	if json.Unmarshal(g.Receipt, &receipt) != nil || receipt.ActorID != g.ActorID || receipt.RepositoryID != r.Repository {
		return factory.ErrLifecycleAction
	}
	issues, err := intentReceiptIssues(g.SubmittedTasks, g, receipt)
	if err != nil {
		return err
	}
	keys := make([]string, 0, len(g.SubmittedTasks))
	for _, task := range g.SubmittedTasks {
		keys = append(keys, task.Key)
	}
	var first error
	for _, key := range LifecycleNewIssueKeys(keys, g) {
		entry := issues[key]
		if entry.IssueID <= 0 {
			if first == nil {
				first = factory.ErrLifecycleAction
			}
			continue
		}
		in := LifecycleEventInput{
			Event: factory.LifecycleEventIssueCreated, SourceID: r.ID + ":" + key,
			RepositoryID: r.Repository, IssueID: entry.IssueID, AuthRevision: g.AuthRevision,
		}
		if err := ConsumeLifecycleEvent(ctx, s, ex, in); err != nil && first == nil {
			first = err
		}
	}
	return first
}

// ConsumePublishedPR fires the configured reviewer-request effect for
// one confirmed PR creation. The identity binds the publication plus
// the exact PR receipt so refires reconcile instead of duplicating.
func ConsumePublishedPR(ctx context.Context, s *store.Store, ex LifecycleExecutor, p factory.Publication) error {
	if s == nil || ex == nil {
		return factory.ErrLifecycleAction
	}
	if p.Stage != factory.PublicationPublished || p.PRNumber <= 0 || p.PRID <= 0 || p.PRCreate.IssueID <= 0 || p.PRCreate.Work == nil || p.PRCreate.Work.AuthRevision == "" {
		return factory.ErrLifecycleAction
	}
	source := p.ID + ":pr-" + strconv.FormatInt(p.PRNumber, 10) + "-" + strconv.FormatInt(p.PRID, 10)
	in := LifecycleEventInput{
		Event: factory.LifecycleEventPRCreated, SourceID: source,
		RepositoryID: p.Repository, IssueID: p.PRCreate.IssueID, AuthRevision: p.PRCreate.Work.AuthRevision,
	}
	return ConsumeLifecycleEvent(ctx, s, ex, in)
}

// ConsumeVerifiedMerge fires the configured completion-note effect for
// one verified merge with confirmed original closure. The identity
// binds the merge plus the exact merge and closure facts.
func ConsumeVerifiedMerge(ctx context.Context, s *store.Store, ex LifecycleExecutor, m factory.Merge) error {
	if s == nil || ex == nil {
		return factory.ErrLifecycleAction
	}
	if m.Stage != factory.MergeMerged || m.Issue <= 0 || m.MergedCommit == "" || m.OriginalClosingOccurrence == "" || m.Operation.Work == nil || m.Operation.Work.AuthRevision == "" {
		return factory.ErrLifecycleAction
	}
	source := m.ID + ":merge-" + m.MergedCommit + "-" + m.OriginalClosingOccurrence
	in := LifecycleEventInput{
		Event: factory.LifecycleEventVerifiedMerge, SourceID: source,
		RepositoryID: m.Repository, IssueIndex: m.Issue, AuthRevision: m.Operation.Work.AuthRevision,
		MergeRef: m.MergedCommit, Closed: true,
	}
	return ConsumeLifecycleEvent(ctx, s, ex, in)
}

// lifecycleDrainLimit bounds one recovery drain so a wedged native
// cannot stall the existing pass that hosts it.
const lifecycleDrainLimit = 8

// ReconcileUnsettledLifecycle settles journaled actions left pending or
// uncertain by earlier completions: duplicate-safe lookups first, then
// at most one retry per action after definitive no-effect. Existing
// passes call this after their default work; it starts no scheduler
// and submits nothing for disabled or unproved events.
func ReconcileUnsettledLifecycle(ctx context.Context, s *store.Store, ex LifecycleExecutor) error {
	if s == nil || ex == nil {
		return factory.ErrLifecycleAction
	}
	pending, err := s.UnsettledLifecycleActions(ctx, lifecycleDrainLimit)
	if err != nil {
		return err
	}
	for _, a := range pending {
		if _, err := ReconcileLifecycleAction(ctx, s, ex, a.ID); err != nil {
			return err
		}
	}
	return nil
}

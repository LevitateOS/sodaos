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
// (revision plus task key, publication plus PR receipt, merge plus
// merge/closure facts); AuthRevision carries the event's stable
// authorization basis (I-F5: generation authorization, PRCreate intent,
// merge intent — all immutable). IssueIndex is the work-issue read key
// (native index); TargetID is the exact native payload target, or zero
// for verified merges where the executor resolves and revalidates the
// closed target from the index before anything journals.
type LifecycleEventInput struct {
	Event        string
	SourceID     string
	RepositoryID int64
	IssueIndex   int64
	TargetID     int64
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
		if errors.Is(err, store.ErrLifecycleActionStale) {
			return ReconcileLifecycleAction(ctx, s, ex, a.ID)
		}
		return a, err
	}
	a = next
	work := factory.LifecycleWork{
		OperationID: opID, Event: a.Event, ActorID: a.ActorID, RepositoryID: a.RepositoryID,
		IssueID: a.TargetID, LabelID: a.LabelID, AssigneeID: a.AssigneeID, ReviewerIDs: append([]int64(nil), a.ReviewerIDs...),
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
				if errors.Is(journalErr, store.ErrLifecycleActionStale) {
					return ReconcileLifecycleAction(ctx, s, ex, a.ID)
				}
				return a, journalErr
			}
			return journaled, nil
		}
		outcome = lookup
	}
	journaled, err := s.UpdateAction(ctx, a.RecordActionResult(ClassifyLifecycleOutcome(outcome)), a.Attempts)
	if err != nil {
		if errors.Is(err, store.ErrLifecycleActionStale) {
			return ReconcileLifecycleAction(ctx, s, ex, a.ID)
		}
		return a, err
	}
	return journaled, nil
}

// lifecycleRetryAllowed reports whether a journaled action may submit:
// a pending first attempt, or the single retry once every target
// shows definitive no-effect. Uncertain or denied targets never ride
// a new operation identity: uncertainty reconciles by lookup, denial
// stops. Every submit revalidates CURRENT proof (I-F12): capability
// still proved, configuration still enabled for the same repository,
// and the enrolled actor still matching the journaled submitter. On
// any negative the action is left as-is (failed stays failed, pending
// stays pending) for a later tick; disable-mid-flight pauses, it does
// not cancel or revoke. Reconciliation lookups need none of this.
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

// claimLifecycleAction binds an unclaimed journaled action to current
// enrollment before its first attempt: the resolved enrolled UID
// becomes the submitter, and a missing completion-note target resolves
// from the journaled read-key index. Already-claimed rows pass
// through untouched. The claim persists without consuming attempt
// budget; failures leave the row unclaimed for a later tick.
func claimLifecycleAction(ctx context.Context, s *store.Store, ex LifecycleExecutor, a factory.LifecycleAction) (factory.LifecycleAction, error) {
	if a.Attempts > 0 || (a.ActorID > 0 && a.TargetID > 0) {
		return a, nil
	}
	next := a
	if next.ActorID <= 0 {
		actorCtx, actorCancel := withLifecycleTimeout(ctx)
		enrolled, err := ex.EnrolledActor(actorCtx)
		actorCancel()
		if err != nil || enrolled <= 0 {
			return a, nil
		}
		next.ActorID = enrolled
	}
	if next.TargetID <= 0 {
		if next.Event != factory.LifecycleEventVerifiedMerge {
			return a, nil
		}
		targetCtx, targetCancel := withLifecycleTimeout(ctx)
		resolved, err := ex.ResolveNoteTarget(targetCtx, next.RepositoryID, next.IssueID)
		targetCancel()
		if err != nil {
			return a, nil
		}
		next.TargetID = resolved
	}
	claimed, err := s.UpdateAction(ctx, next, a.Attempts)
	if err != nil {
		if errors.Is(err, store.ErrLifecycleActionStale) {
			return s.LifecycleAction(ctx, a.ID)
		}
		return a, err
	}
	return claimed, nil
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
		claimed, err := claimLifecycleAction(ctx, s, ex, a)
		if err != nil {
			return a, err
		}
		a = claimed
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
// out of default-path latency. The submitter is always the resolved
// enrolled lifecycle UID; the human configuration authorizer is
// journaled alongside, never equated (JEV 3/3 separate_roles plus
// native ActorMismatch evidence). Disabled events, unproved
// capability and repository mismatch journal nothing and perform no
// native work. A missing or unresolvable executor on an enabled,
// proved event journals an uncertain outcome so the no-op setup is
// visible instead of silent; the drain attempts it once an executor
// is available. Duplicate deliveries reconcile the journaled action
// instead of submitting twice. Every optional failure returns without
// touching default state; hook sites must never gate default delivery
// on this result.
// ErrLifecycleExecutorUnavailable reports an enabled, proved event
// that cannot submit because no lifecycle executor or enrollment is
// usable. The event journals an unclaimed uncertain row so the no-op
// setup stays visible; the drain claims and attempts it once an
// executor is available.
var ErrLifecycleExecutorUnavailable = errors.New("lifecycle executor unavailable")

// journalUnavailableExecutor records one unclaimed uncertain row for an
// enabled, proved event fired without a usable executor. Completion
// notes omit the target until the drain resolves it from the journaled
// read-key index. The row is always visible and never blocks the
// default path; errors return the unavailability sentinel.
func journalUnavailableExecutor(ctx context.Context, s *store.Store, cfg factory.LifecycleConfig, in LifecycleEventInput) error {
	a := factory.LifecycleAction{
		ID: LifecycleActionID(in), Event: in.Event, Action: factory.EventAction(in.Event),
		RepositoryID: in.RepositoryID, IssueID: in.IssueIndex, TargetID: in.TargetID,
		ConfigOwner: cfg.ActorID,
		LabelID:     cfg.LabelID, AssigneeID: cfg.AssigneeID, ReviewerIDs: append([]int64(nil), cfg.ReviewerIDs...),
		Note: cfg.Note, AuthRevision: in.AuthRevision, MergeRef: in.MergeRef, Closed: in.Closed,
	}
	if err := a.Validate(); err != nil {
		return err
	}
	recorded, err := s.RecordAction(ctx, a)
	if err != nil {
		if errors.Is(err, store.ErrLifecycleActionConflict) {
			return ErrLifecycleExecutorUnavailable
		}
		return err
	}
	if recorded.Attempts == 0 && len(recorded.Results) == 0 {
		if _, err := s.UpdateAction(ctx, recorded.RecordActionResult(factory.LifecycleResultUncertain), 0); err != nil && !errors.Is(err, store.ErrLifecycleActionStale) {
			return err
		}
	}
	return ErrLifecycleExecutorUnavailable
}

func ConsumeLifecycleEvent(ctx context.Context, s *store.Store, ex LifecycleExecutor, in LifecycleEventInput) error {
	if s == nil {
		return factory.ErrLifecycleAction
	}
	if !factory.ValidLifecycleEvent(in.Event) || in.SourceID == "" || in.RepositoryID <= 0 || in.IssueIndex <= 0 || in.AuthRevision == "" {
		return factory.ErrLifecycleAction
	}
	if in.Event != factory.LifecycleEventVerifiedMerge && in.TargetID <= 0 {
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
	if ex == nil {
		return journalUnavailableExecutor(ctx, s, cfg, in)
	}
	actorCtx, actorCancel := withLifecycleTimeout(ctx)
	enrolled, err := ex.EnrolledActor(actorCtx)
	actorCancel()
	if err != nil || enrolled <= 0 {
		return journalUnavailableExecutor(ctx, s, cfg, in)
	}
	target := in.TargetID
	if in.Event == factory.LifecycleEventVerifiedMerge {
		if !in.Closed {
			return factory.ErrLifecycleAction
		}
		targetCtx, targetCancel := withLifecycleTimeout(ctx)
		resolved, err := ex.ResolveNoteTarget(targetCtx, in.RepositoryID, in.IssueIndex)
		targetCancel()
		if err != nil {
			return err
		}
		target = resolved
	}
	a := factory.LifecycleAction{
		ID: LifecycleActionID(in), Event: in.Event, Action: factory.EventAction(in.Event),
		ActorID: enrolled, RepositoryID: in.RepositoryID, IssueID: in.IssueIndex, TargetID: target,
		ConfigOwner: cfg.ActorID,
		LabelID:     cfg.LabelID, AssigneeID: cfg.AssigneeID, ReviewerIDs: append([]int64(nil), cfg.ReviewerIDs...),
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
		if entry.IssueID <= 0 || entry.IssueIndex <= 0 {
			if first == nil {
				first = factory.ErrLifecycleAction
			}
			continue
		}
		in := LifecycleEventInput{
			Event: factory.LifecycleEventIssueCreated, SourceID: r.ID + ":" + key,
			RepositoryID: r.Repository, IssueIndex: entry.IssueIndex, TargetID: entry.IssueID, AuthRevision: g.AuthRevision,
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
	if p.Stage != factory.PublicationPublished || p.Issue <= 0 || p.PRNumber <= 0 || p.PRID <= 0 || p.PRCreate.IssueID <= 0 || p.PRCreate.Work == nil || p.PRCreate.Work.AuthRevision == "" {
		return factory.ErrLifecycleAction
	}
	source := p.ID + ":pr-" + strconv.FormatInt(p.PRNumber, 10) + "-" + strconv.FormatInt(p.PRID, 10)
	in := LifecycleEventInput{
		Event: factory.LifecycleEventPRCreated, SourceID: source,
		RepositoryID: p.Repository, IssueIndex: p.Issue, TargetID: p.PRCreate.IssueID, AuthRevision: p.PRCreate.Work.AuthRevision,
	}
	return ConsumeLifecycleEvent(ctx, s, ex, in)
}

// ConsumeVerifiedMerge fires the configured completion-note effect for
// one verified merge with confirmed original closure. Qualification
// is by facts, not stage: the merged commit, both closure timestamps,
// the original closing occurrence and the merge authorization. The
// call site runs only after finishMerge persists MergeMerged (which
// takes the merge by value, so the caller's copy still reads Open);
// facts set exclusively on that path are the stable gate. The
// identity binds the merge plus the exact merge and closure facts.
func ConsumeVerifiedMerge(ctx context.Context, s *store.Store, ex LifecycleExecutor, m factory.Merge) error {
	if s == nil || ex == nil {
		return factory.ErrLifecycleAction
	}
	if m.Issue <= 0 || m.PRNumber <= 0 || m.MergedCommit == "" || m.MergedUnix <= 0 || m.ClosedUnix <= 0 || m.OriginalClosingOccurrence == "" || m.OriginalClosedUnix <= 0 || m.Operation.Work == nil || m.Operation.Work.AuthRevision == "" {
		return factory.ErrLifecycleAction
	}
	source := m.ID + ":merge-" + m.MergedCommit + "-" + m.OriginalClosingOccurrence
	in := LifecycleEventInput{
		Event: factory.LifecycleEventVerifiedMerge, SourceID: source,
		RepositoryID: m.Repository, IssueIndex: m.Issue, AuthRevision: m.Operation.Work.AuthRevision,
		MergeRef: "pr/" + strconv.FormatInt(m.PRNumber, 10) + "@" + m.MergedCommit, Closed: true,
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

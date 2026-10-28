package control

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"

	sdk "forgejo.org/extension-sdk"
	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/store"
)

// scopedIntentTasksFromChosen adopts exactly the single planner candidate
// the Generate work command chose before serving: it binds the choice to
// the current accepted head and enforces it inside accepted scope before
// any native observation. No triplet judgment or template restructuring
// runs here: the choice arrives decided, so a second evaluation cannot
// rewrite it on the serving path.
func (c *Coordinator) scopedIntentTasksFromChosen(_ context.Context, r factory.IntentRevision, chosen *ChosenIntentPlan) ([]factory.IntentTask, error) {
	if chosen == nil {
		return nil, errors.New("chosen planner candidate is missing")
	}
	if chosen.Intent != r.Intent {
		return nil, errors.New("chosen planner candidate binds another intent")
	}
	if !factory.ValidIntentPlanningOperation(chosen.Operation) {
		return nil, errors.New("chosen planner candidate uses an unsupported operation")
	}
	// The planner runs against the reviewed draft; the accepted successor
	// adopts its choice. Either the choice binds this exact revision or
	// the draft this acceptance reviewed — never a third revision.
	if chosen.Revision != r.ID && chosen.Revision != r.Predecessor {
		return nil, errors.New("chosen planner revision mismatch")
	}
	if len(chosen.Tasks) == 0 {
		return nil, errors.New("chosen planner candidate carries no tasks")
	}
	work := r
	work.Tasks = append([]factory.IntentTask(nil), chosen.Tasks...)
	if err := work.Validate(); err != nil {
		return nil, errors.New("chosen planner candidate exceeds accepted scope")
	}
	return work.Tasks, nil
}

// intentDeltaEdge binds one accepted dependency to its native issue pair.
type intentDeltaEdge struct{ a, b int64 }

// intentDeltaBuild carries the shared state behind one intent-graph
// delta so each duty reads through one small check.
type intentDeltaBuild struct {
	c             *Coordinator
	ctx           context.Context
	r             factory.IntentRevision
	revision      int64
	work          factory.IntentRevision
	prevWork      factory.IntentRevision
	tasks         []factory.IntentTask
	current       map[string]factory.IntentTask
	reused        map[string]bool
	oldIssues     map[string]sdk.IssueGraphIssueReceipt
	generation    factory.IntentGeneration
	priorBindings []factory.IntentTaskBinding
	p             sdk.IssueGraphPayload
	ids           map[string]int64
	refs          map[string]sdk.IssueGraphRef
	delivered     map[string]factory.DeliveredIssue
	closed        map[string]factory.DeliveredIssue
	oldEdges      map[intentDeltaEdge]bool
}

// prepareIntentGraphWithChoice expresses a reviewed decomposition change
// through the existing native issue owner, adopting exactly the one
// pre-chosen planner candidate. Reused open keys retain issue IDs and
// history; delivered (closed) records are never edited. It returns the
// submitted task graph and delivered bindings alongside the payload.
func (c *Coordinator) prepareIntentGraphWithChoice(ctx context.Context, r factory.IntentRevision, revision int64, chosen *ChosenIntentPlan) (sdk.IssueGraphPayload, map[string]int64, []factory.IntentTask, map[string]factory.DeliveredIssue, error) {
	previous, err := c.Store.PreviousGeneratedIntent(ctx, r.Intent, r.ID)
	if errors.Is(err, store.ErrNotFound) {
		return c.prepareFirstIntentGraph(ctx, r, chosen)
	}
	if err != nil {
		return sdk.IssueGraphPayload{}, nil, nil, nil, err
	}
	b, err := c.loadIntentDeltaBuild(ctx, r, revision, chosen, previous)
	if err != nil {
		return sdk.IssueGraphPayload{}, nil, nil, nil, err
	}
	if err := b.verifyPreviousNatives(); err != nil {
		return b.p, nil, nil, nil, err
	}
	return b.finishIntentDelta()
}

// prepareFirstIntentGraph scopes and packs the first generation
// behind one accepted intent, which has no previous revision.
func (c *Coordinator) prepareFirstIntentGraph(ctx context.Context, r factory.IntentRevision, chosen *ChosenIntentPlan) (sdk.IssueGraphPayload, map[string]int64, []factory.IntentTask, map[string]factory.DeliveredIssue, error) {
	tasks, err := c.scopedIntentTasksFromChosen(ctx, r, chosen)
	if err != nil {
		return sdk.IssueGraphPayload{}, nil, nil, nil, err
	}
	work := r
	work.Tasks = tasks
	p, err := IntentGraphPayload(work)
	if err != nil {
		return sdk.IssueGraphPayload{}, nil, nil, nil, err
	}
	return p, nil, tasks, nil, nil
}

// loadIntentDeltaBuild resolves the previous generation, its receipt
// issues and the task diff behind one intent-graph delta.
func (c *Coordinator) loadIntentDeltaBuild(ctx context.Context, r factory.IntentRevision, revision int64, chosen *ChosenIntentPlan, previous factory.IntentRevision) (*intentDeltaBuild, error) {
	generation, err := c.Store.IntentGeneration(ctx, previous.ID)
	if err != nil {
		return nil, err
	}
	if len(generation.SubmittedTasks) == 0 {
		return nil, errors.New("previous generation predates submitted-task records")
	}
	priorSubmitted := previous
	priorSubmitted.Tasks = generation.SubmittedTasks
	tasks, err := c.scopedIntentTasksFromChosen(ctx, r, chosen)
	if err != nil {
		return nil, err
	}
	var receipt sdk.IssueGraphReceipt
	if generation.LocalOnly && generation.State == "completed" && generation.ProtectedGraphCycle() != "" {
		receipt.ActorID, receipt.RepositoryID = generation.ActorID, r.Repository
	} else if json.Unmarshal(generation.Receipt, &receipt) != nil || receipt.ActorID != generation.ActorID || receipt.RepositoryID != r.Repository {
		return nil, errors.New("previous graph receipt is unavailable")
	}
	oldIssues, err := intentReceiptIssues(generation.SubmittedTasks, generation, receipt)
	if err != nil {
		return nil, err
	}
	work := r
	work.Tasks = tasks
	prevWork := previous
	prevWork.Tasks = generation.SubmittedTasks
	movement, err := factory.DiffIntentTasks(prevWork, work)
	if err != nil {
		return nil, err
	}
	b := &intentDeltaBuild{c: c, ctx: ctx, r: r, revision: revision,
		work: work, prevWork: prevWork, tasks: tasks, generation: generation,
		oldIssues: oldIssues, reused: map[string]bool{}, current: map[string]factory.IntentTask{},
		ids: map[string]int64{}, refs: map[string]sdk.IssueGraphRef{},
		delivered: map[string]factory.DeliveredIssue{}, closed: map[string]factory.DeliveredIssue{},
		oldEdges: map[intentDeltaEdge]bool{}}
	for _, key := range movement.Reused {
		b.reused[key] = true
	}
	for _, task := range tasks {
		b.current[task.Key] = task
	}
	priorBindings, err := c.Store.IntentTaskBindings(ctx, previous.ID)
	if err != nil {
		return nil, err
	}
	b.priorBindings = priorBindings
	return b, nil
}

// verifyPreviousNatives verifies every previously submitted task
// against its native observation at this revision.
func (b *intentDeltaBuild) verifyPreviousNatives() error {
	for _, task := range b.generation.SubmittedTasks {
		if err := b.verifyPreviousNative(task); err != nil {
			return err
		}
	}
	return nil
}

// verifyPreviousNative verifies one previously submitted task,
// recording its edit or its delivered pin.
func (b *intentDeltaBuild) verifyPreviousNative(task factory.IntentTask) error {
	entry := b.oldIssues[task.Key]
	native, nativeRev, err := b.c.Graph.ReadGeneratedIssue(b.ctx, b.r.Repository, entry.IssueIndex, b.generation.ActorID)
	if err != nil {
		return err
	}
	if nativeRev != b.revision {
		return refuseAcceptance(RefusalStaleEvidence)
	}
	issue := native.Issue
	if issue == nil || !issue.Visible || !issue.Complete || issue.IsPull || issue.IsLocked || issue.ID != strconv.FormatInt(entry.IssueID, 10) {
		return refuseAcceptance(RefusalObjectiveChanged)
	}
	// Delivered (closed) records are verified but never edited: the
	// native writer refuses closed edits atomically. A delivered
	// record still carries the body its writing revision gave it, so
	// stability is checked against the previous acceptance pins; the
	// rebuilt previous-revision body remains the fallback when the
	// previous adoption never pinned the key. Open records are
	// rewritten by every delta, so the rebuild is exact for them.
	if issue.IsClosed {
		return b.pinDeliveredNative(task, entry, issue, native)
	}
	if issue.Title != intentTaskTitle(task) || issue.Content != intentTaskBody(b.prevWork, task) || issue.ContentVersion != entry.ContentVersion {
		return refuseAcceptance(RefusalObjectiveChanged)
	}
	title, body := issue.Title, issue.Content+"\nSuperseded by accepted product revision: "+b.r.ID+"\nThis task is retained as history and grants no current generated-work authority.\n"
	if next, ok := b.current[task.Key]; ok {
		title = intentTaskTitle(next)
		body = intentTaskBody(b.work, next)
	}
	b.p.Edits = append(b.p.Edits, sdk.IssueGraphEdit{IssueID: entry.IssueID, ExpectedTitle: issue.Title, ExpectedBody: issue.Content, ExpectedContentVersion: issue.ContentVersion, Title: title, Body: body})
	return b.verifyNativeEdges(task, entry, native)
}

// pinDeliveredNative pins one closed native record as delivered after
// proving it still carries its written content.
func (b *intentDeltaBuild) pinDeliveredNative(task factory.IntentTask, entry sdk.IssueGraphIssueReceipt, issue *sdk.SnapshotIssue, native sdk.NativeSnapshot) error {
	stable, err := b.c.deliveredStable(b.ctx, b.priorBindings, task, entry, issue, b.prevWork)
	if err != nil {
		return err
	}
	if !stable {
		return refuseAcceptance(RefusalObjectiveChanged)
	}
	b.closed[task.Key] = factory.DeliveredIssue{IssueID: entry.IssueID, IssueIndex: entry.IssueIndex, ContentVersion: entry.ContentVersion}
	return b.verifyNativeEdges(task, entry, native)
}

// verifyNativeEdges matches one native observation's dependencies
// against the accepted graph.
func (b *intentDeltaBuild) verifyNativeEdges(task factory.IntentTask, entry sdk.IssueGraphIssueReceipt, native sdk.NativeSnapshot) error {
	if native.Dependencies == nil || !native.Dependencies.Complete || len(native.Dependencies.Items) != len(task.DependsOn) {
		return refuseAcceptance(RefusalEdgeChanged)
	}
	expected := map[int64]bool{}
	for _, dep := range task.DependsOn {
		expected[b.oldIssues[dep].IssueID] = true
	}
	for _, d := range native.Dependencies.Items {
		dep, err := strconv.ParseInt(d.DependencyID, 10, 64)
		if err != nil || !d.Visible || !d.Complete || !expected[dep] {
			return refuseAcceptance(RefusalEdgeChanged)
		}
		delete(expected, dep)
		b.oldEdges[intentDeltaEdge{entry.IssueID, dep}] = true
	}
	if len(expected) > 0 {
		return refuseAcceptance(RefusalEdgeChanged)
	}
	return nil
}

// finishIntentDelta plans creates, dependencies and removals behind
// one verified delta and packs the immutable packet.
func (b *intentDeltaBuild) finishIntentDelta() (sdk.IssueGraphPayload, map[string]int64, []factory.IntentTask, map[string]factory.DeliveredIssue, error) {
	b.planDeltaCreates()
	b.planDeltaDependencies()
	if err := sdk.ValidateIssueGraphPayload(b.p); err != nil {
		return b.p, nil, nil, nil, fmt.Errorf("accepted decomposition exceeds native packet bounds: %w", err)
	}
	if len(b.delivered) == 0 {
		b.delivered = nil
	}
	return b.p, b.ids, b.tasks, b.delivered, nil
}

// planDeltaCreates plans created and rebound issues behind one delta.
func (b *intentDeltaBuild) planDeltaCreates() {
	for _, task := range b.tasks {
		old, ok := b.oldIssues[task.Key]
		if !ok {
			b.p.Creates = append(b.p.Creates, sdk.IssueGraphCreate{Key: task.Key, Title: intentTaskTitle(task), Body: intentTaskBody(b.work, task)})
			b.refs[task.Key] = sdk.IssueGraphRef{Key: task.Key}
			continue
		}
		if ref, closed := b.closed[task.Key]; closed {
			if b.reused[task.Key] {
				b.delivered[task.Key] = ref
				b.refs[task.Key] = sdk.IssueGraphRef{IssueID: ref.IssueID}
			} else {
				b.p.Creates = append(b.p.Creates, sdk.IssueGraphCreate{Key: task.Key, Title: intentTaskTitle(task), Body: intentTaskBody(b.work, task)})
				b.refs[task.Key] = sdk.IssueGraphRef{Key: task.Key}
			}
			continue
		}
		b.ids[task.Key] = old.IssueID
		b.refs[task.Key] = sdk.IssueGraphRef{IssueID: old.IssueID}
	}
}

// planDeltaDependencies plans added and removed dependencies behind
// one delta, keeping the retained native edges out of removals.
func (b *intentDeltaBuild) planDeltaDependencies() {
	retained := map[intentDeltaEdge]bool{}
	for _, task := range b.tasks {
		for _, dep := range task.DependsOn {
			a, c := b.refs[task.Key], b.refs[dep]
			key := intentDeltaEdge{a.IssueID, c.IssueID}
			if key.a > 0 && key.b > 0 && b.oldEdges[key] {
				retained[key] = true
				continue
			}
			b.p.Dependencies = append(b.p.Dependencies, sdk.IssueGraphDependency{Issue: a, Prerequisite: c})
		}
	}
	// Stable order follows the previous task/dependency order; the immutable
	// packet is deterministic even though membership checks use maps.
	for _, task := range b.generation.SubmittedTasks {
		for _, dep := range task.DependsOn {
			key := intentDeltaEdge{b.oldIssues[task.Key].IssueID, b.oldIssues[dep].IssueID}
			if !retained[key] {
				b.p.Dependencies = append(b.p.Dependencies, sdk.IssueGraphDependency{Issue: sdk.IssueGraphRef{IssueID: key.a}, Prerequisite: sdk.IssueGraphRef{IssueID: key.b}, Remove: true})
			}
		}
	}
}

// deliveredStable reports whether a closed native record still carries the
// exact content its writing revision gave it. The previous acceptance pins
// title and content digests at every adoption, so pins stay exact across
// any number of delivered hops with single-hop lookups. When the previous
// adoption never pinned the key, the rebuilt previous-revision body is the
// fallback; it is exact for records the previous submit wrote.
func (c *Coordinator) deliveredStable(ctx context.Context, prior []factory.IntentTaskBinding, task factory.IntentTask, entry sdk.IssueGraphIssueReceipt, issue *sdk.SnapshotIssue, prevWork factory.IntentRevision) (bool, error) {
	for _, b := range prior {
		if b.TaskKey != task.Key {
			continue
		}
		if b.IssueID != entry.IssueID || b.IssueIndex != entry.IssueIndex || b.ContentVersion != entry.ContentVersion {
			return false, errors.New("previous graph receipt does not bind submitted tasks")
		}
		decision, err := c.Store.AcceptanceDecision(ctx, b.AcceptanceID)
		if errors.Is(err, store.ErrNotFound) {
			break
		}
		if err != nil {
			return false, err
		}
		return issue.TitleDigest == decision.TitleDigest && issue.ContentDigest == decision.ContentDigest && issue.ContentVersion == decision.ContentVersion, nil
	}
	return issue.Title == intentTaskTitle(task) && issue.Content == intentTaskBody(prevWork, task) && issue.ContentVersion == entry.ContentVersion, nil
}

func intentReceiptIssues(tasks []factory.IntentTask, g factory.IntentGeneration, receipt sdk.IssueGraphReceipt) (map[string]sdk.IssueGraphIssueReceipt, error) {
	issues := map[string]sdk.IssueGraphIssueReceipt{}
	byID := map[int64]sdk.IssueGraphIssueReceipt{}
	for _, entry := range receipt.Issues {
		if entry.IssueID <= 0 || entry.IssueIndex <= 0 || byID[entry.IssueID].IssueID != 0 {
			return nil, errors.New("invalid native graph receipt")
		}
		byID[entry.IssueID] = entry
		if entry.Key != "" {
			if issues[entry.Key].IssueID != 0 {
				return nil, errors.New("duplicate native graph key")
			}
			issues[entry.Key] = entry
		}
	}
	for key, id := range g.TaskIssueIDs {
		entry, exists := byID[id]
		if !exists || entry.Key != "" {
			return nil, errors.New("native graph edit receipt missing reused task")
		}
		issues[key] = entry
	}
	for _, binding := range g.RetainedBindings {
		if g.ProtectedGraphCycle() == "" || binding.RevisionID != g.RevisionID || g.TaskIssueIDs[binding.TaskKey] != 0 {
			return nil, errors.New("retained native identity lacks its reassessment owner")
		}
		if observed, ok := byID[binding.IssueID]; ok && (observed.IssueIndex != binding.IssueIndex || observed.ContentVersion != binding.ContentVersion || (observed.Key != "" && observed.Key != binding.TaskKey)) {
			return nil, errors.New("native graph receipt changed a retained task")
		}
		if issues[binding.TaskKey].IssueID != 0 && issues[binding.TaskKey].IssueID != binding.IssueID {
			return nil, errors.New("retained task conflicts with native graph receipt")
		}
		issues[binding.TaskKey] = sdk.IssueGraphIssueReceipt{Key: binding.TaskKey, IssueID: binding.IssueID, IssueIndex: binding.IssueIndex, ContentVersion: binding.ContentVersion}
	}
	for key, ref := range g.Delivered {
		if issues[key].IssueID != 0 || byID[ref.IssueID].IssueID != 0 {
			return nil, errors.New("delivered history unexpectedly mutated")
		}
		issues[key] = sdk.IssueGraphIssueReceipt{Key: key, IssueID: ref.IssueID, IssueIndex: ref.IssueIndex, ContentVersion: ref.ContentVersion}
	}
	if len(issues) != len(tasks) {
		return nil, errors.New("native graph receipt does not bind submitted tasks")
	}
	for _, task := range tasks {
		if issues[task.Key].IssueID == 0 {
			return nil, errors.New("native graph receipt missing submitted task")
		}
	}
	return issues, nil
}

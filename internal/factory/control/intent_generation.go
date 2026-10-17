package control

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	sdk "forgejo.org/extension-sdk"
	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/store"
)

// IssueGraphExecutor supplies native mutation/permission observations only.
// Accepted product authority and exact child adoption remain Soda-owned.
type IssueGraphExecutor interface {
	CheckIntentOwner(context.Context, int64, int64, int64) (int64, error)
	SubmitGraph(context.Context, factory.IntentGeneration, int64) (sdk.OperationRecord, error)
	LookupGraph(context.Context, factory.IntentGeneration, int64) (sdk.OperationLookup, error)
	CancelGraph(context.Context, string) (sdk.OperationRecord, error)
	ReadGeneratedIssue(context.Context, int64, int64, int64) (sdk.NativeSnapshot, int64, error)
}

// IntentGraphPayload renders exact accepted tasks into native requirements.
// Provenance is visible in native issue content and separately retained in
// Soda's immutable generation ledger; it does not impersonate human authorship.
func IntentGraphPayload(r factory.IntentRevision) (sdk.IssueGraphPayload, error) {
	if err := r.Validate(); err != nil {
		return sdk.IssueGraphPayload{}, err
	}
	if r.State != "accepted" {
		return sdk.IssueGraphPayload{}, errors.New("product direction is not accepted")
	}
	p := sdk.IssueGraphPayload{}
	for _, task := range r.Tasks {
		body := intentTaskBody(r, task)
		p.Creates = append(p.Creates, sdk.IssueGraphCreate{Key: task.Key, Title: intentTaskTitle(task), Body: body})
		for _, dep := range task.DependsOn {
			p.Dependencies = append(p.Dependencies, sdk.IssueGraphDependency{Issue: sdk.IssueGraphRef{Key: task.Key}, Prerequisite: sdk.IssueGraphRef{Key: dep}})
		}
	}
	if err := sdk.ValidateIssueGraphPayload(p); err != nil {
		return sdk.IssueGraphPayload{}, err
	}
	return p, nil
}

func intentTaskBody(r factory.IntentRevision, t factory.IntentTask) string {
	var b strings.Builder
	b.WriteString("Accepted product objective:\n")
	b.WriteString(r.Objective)
	if t.Objective != r.Objective {
		b.WriteString("\n\nTask objective:\n")
		b.WriteString(t.Objective)
	}
	b.WriteString("\n\nAcceptance outcomes:\n")
	for _, outcome := range t.Outcomes {
		b.WriteString("- ")
		b.WriteString(outcome)
		b.WriteByte('\n')
	}
	selected := map[string]bool{}
	for _, id := range t.IdeaIDs {
		selected[id] = true
	}
	effective := map[string]bool{}
	for _, id := range r.EffectiveOutcomes {
		effective[id] = true
	}
	for _, idea := range r.Ideas {
		if !selected[idea.ID] && !idea.Assumption {
			continue
		}
		if len(effective) > 0 && !effective[idea.ID] {
			continue
		}
		if idea.Decision != "accept" && idea.Decision != "edit" {
			continue
		}
		label := "Accepted product choice"
		if idea.Assumption {
			label = "Accepted assumption"
		}
		if idea.Source != "" || idea.Provenance != "" {
			provenance := idea.Source
			if idea.Provenance != "" {
				if provenance != "" {
					provenance += ", "
				}
				provenance += idea.Provenance
			}
			if provenance != "" {
				label += " (" + provenance + ")"
			}
		}
		fmt.Fprintf(&b, "\n%s (%s):\n", label, idea.ID)
		if idea.Decision == "edit" {
			if idea.Description != r.Objective && idea.Description != t.Objective {
				fmt.Fprintf(&b, "Reviewed proposal: %s\n", idea.Description)
			}
			if idea.Correction != r.Objective && idea.Correction != t.Objective {
				fmt.Fprintf(&b, "Accepted correction: %s\n", idea.Correction)
			} else {
				if idea.Correction == r.Objective {
					b.WriteString("Accepted correction: the accepted product objective stated above\n")
				} else {
					b.WriteString("Accepted correction: the task objective stated above\n")
				}
			}
		} else if idea.Description != r.Objective && idea.Description != t.Objective {
			b.WriteString(idea.Description)
			b.WriteByte('\n')
		} else {
			if idea.Description == r.Objective {
				b.WriteString("The accepted product objective stated above\n")
			} else {
				b.WriteString("The task objective stated above\n")
			}
		}
		fmt.Fprintf(&b, "Consequence: %s\n", idea.Consequence)
		if idea.Evidence != "" {
			fmt.Fprintf(&b, "Evidence: %s\n", idea.Evidence)
		}
	}
	fmt.Fprintf(&b, "\nAccepted product intent: %s\nAccepted revision: %s\nIntent digest: %s\nTask: %s\n", r.Intent, r.ID, r.Digest(), t.Key)
	raw, _ := json.Marshal(t)
	digest := sha256.Sum256(raw)
	fmt.Fprintf(&b, "Task digest: %s\nProduct choices: %s\n", hex.EncodeToString(digest[:]), strings.Join(t.IdeaIDs, ", "))
	return b.String()
}

func generatedDecisionID(prefix, revision, key string) string {
	sum := sha256.Sum256([]byte(prefix + "\x00" + revision + "\x00" + key))
	if prefix == "command" {
		return hex.EncodeToString(sum[:16])
	}
	return "d" + hex.EncodeToString(sum[:12])
}

// GenerateAcceptedIntent writes only work bound to the current human-accepted
// direction. It persists the immutable operation before sending and adopts
// children only from its attributable receipt and exact current native evidence.
func (c *Coordinator) GenerateAcceptedIntent(ctx context.Context, intentID string, owner int64) (factory.IntentGeneration, error) {
	return c.generateAcceptedIntent(ctx, intentID, owner, nil)
}

// GenerateAcceptedIntentFromChoice writes only work bound to the current
// human-accepted direction, adopting exactly one pre-chosen planner
// candidate (see ChooseSingleCandidate and ChoosePlannedCandidate). The
// full Generate pipeline runs unchanged — durable operation, native
// submission, exact receipt adoption — except task derivation: the chosen
// tasks replace the serving advised graph, so no triplet judgment or
// template restructuring evaluates on this path. The choice binds the
// reviewed draft this acceptance carries as its predecessor (or this exact
// revision); a choice bound to another intent, another operation, a third
// revision or outside accepted scope refuses before any native observation.
func (c *Coordinator) GenerateAcceptedIntentFromChoice(ctx context.Context, intentID string, owner int64, chosen ChosenIntentPlan) (factory.IntentGeneration, error) {
	if chosen.Intent != intentID {
		return factory.IntentGeneration{}, errors.New("chosen planner candidate binds another intent")
	}
	return c.generateAcceptedIntent(ctx, intentID, owner, &chosen)
}

func (c *Coordinator) generateAcceptedIntent(ctx context.Context, intentID string, owner int64, chosen *ChosenIntentPlan) (factory.IntentGeneration, error) {
	ctx, unlock := c.progressPass(ctx)
	defer unlock()
	if c.Store == nil || c.Graph == nil || c.GraphActorID <= 0 {
		return factory.IntentGeneration{}, errors.New("native issue graph is unavailable")
	}
	r, err := c.Store.IntentRevision(ctx, intentID, owner)
	if err != nil {
		return factory.IntentGeneration{}, err
	}
	if r.State != "accepted" {
		return factory.IntentGeneration{}, errors.New("product direction is not accepted")
	}
	revision, err := c.Graph.CheckIntentOwner(ctx, r.Repository, r.Owner, c.GraphActorID)
	if err != nil {
		return factory.IntentGeneration{}, err
	}
	g, err := c.Store.IntentGeneration(ctx, r.ID)
	if errors.Is(err, store.ErrNotFound) {
		payload, reused, submitted, delivered, e := c.prepareIntentGraphWithChoice(ctx, r, revision, chosen)
		if e != nil {
			return factory.IntentGeneration{}, e
		}
		fresh, e := c.Graph.CheckIntentOwner(ctx, r.Repository, r.Owner, c.GraphActorID)
		if e != nil {
			return factory.IntentGeneration{}, e
		}
		if len(payload.Edits) > 0 && fresh != revision {
			return factory.IntentGeneration{}, refuseAcceptance(RefusalStaleEvidence)
		}
		revision = fresh
		if e = c.currentAcceptedIntent(ctx, r); e != nil {
			return factory.IntentGeneration{}, e
		}
		raw, e := json.Marshal(payload)
		if e != nil {
			return factory.IntentGeneration{}, e
		}
		g = factory.IntentGeneration{RevisionID: r.ID, IntentID: r.Intent, OperationID: "intent-" + r.ID, ActorID: c.GraphActorID, NativeRev: revision, AuthRevision: r.ID + ":" + r.Digest(), NotAfter: time.Now().Add(5 * time.Minute).Unix(), Payload: raw, TaskIssueIDs: reused, SubmittedTasks: submitted, Delivered: delivered, IntentDigest: r.Digest(), State: "pending"}
		if err = c.Store.BeginIntentGeneration(ctx, g); err != nil {
			// Concurrent callers must converge on the winner's exact native
			// revision and deadline rather than invent a second operation.
			_, e := c.Store.IntentGeneration(ctx, r.ID)
			if e != nil {
				return g, err
			}
		}
		// Read the database's exact JSON representation before submitting.
		// PostgreSQL JSONB normalizes object whitespace and key order.
		g, err = c.Store.IntentGeneration(ctx, r.ID)
		if err != nil {
			return g, err
		}
	} else if err != nil {
		return g, err
	}
	if g.IntentDigest != r.Digest() || g.ActorID != c.GraphActorID {
		return g, store.ErrCommandConflict
	}
	if g.State == "refused" {
		return g, errors.New("native graph operation was refused; revise accepted direction before retry")
	}
	if g.State == "completed" {
		return g, nil
	}
	if g.State == "pending" {
		lookup, e := c.Graph.LookupGraph(ctx, g, r.Repository)
		if e != nil {
			return g, e
		}
		var record sdk.OperationRecord
		if lookup.Record != nil {
			record = *lookup.Record
		} else {
			if lookup.Status != sdk.BackgroundOutcomeNotObserved {
				return g, errors.New("native graph outcome is unknown")
			}
			if err = c.currentAcceptedIntent(ctx, r); err != nil {
				g, _ = c.cancelIntentGeneration(ctx, g)
				return g, err
			}
			observed, e := c.Graph.CheckIntentOwner(ctx, r.Repository, r.Owner, g.ActorID)
			if e != nil {
				g, _ = c.cancelIntentGeneration(ctx, g)
				return g, e
			}
			if observed != g.NativeRev {
				g, _ = c.cancelIntentGeneration(ctx, g)
				return g, errors.New("accepted graph native observation is stale")
			}
			if g.NotAfter <= time.Now().Unix() {
				g, _ = c.cancelIntentGeneration(ctx, g)
				return g, errors.New("accepted graph operation expired before submission")
			}
			record, err = c.Graph.SubmitGraph(ctx, g, r.Repository)
			if err != nil {
				return g, err
			}
		}
		switch record.EffectState {
		case sdk.EffectCommitted:
			g.State = "committed"
			g.Receipt = record.Receipt
		case sdk.EffectNotCommitted:
			g.State = "refused"
			g.Receipt = record.Receipt
		default:
			return g, errors.New("native graph effect is unresolved")
		}
		if err = c.Store.RecordIntentGeneration(ctx, g); err != nil {
			return g, err
		}
		if g.State == "refused" {
			return g, errors.New("native graph mutation refused: " + record.ReasonCode)
		}
	}
	if err = c.adoptIntentGraph(ctx, r, g); err != nil {
		return g, err
	}
	// Optional lifecycle effect: new-issue label/assignee. Never gates
	// default delivery; duplicate runs reconcile by action identity.
	if c.Lifecycle != nil {
		_ = ConsumeGeneratedIssues(ctx, c.Store, c.Lifecycle, r, g)
	}
	g.State = "completed"
	if err = c.Store.RecordIntentGeneration(ctx, g); err != nil {
		return g, err
	}
	return g, nil
}

func (c *Coordinator) currentAcceptedIntent(ctx context.Context, r factory.IntentRevision) error {
	head, err := c.Store.IntentRevision(ctx, r.Intent, r.Owner)
	if err != nil {
		return err
	}
	if head.ID != r.ID || head.State != "accepted" || head.Digest() != r.Digest() {
		return store.ErrStaleRevision
	}
	return nil
}

func (c *Coordinator) adoptIntentGraph(ctx context.Context, r factory.IntentRevision, g factory.IntentGeneration) error {
	var receipt sdk.IssueGraphReceipt
	if json.Unmarshal(g.Receipt, &receipt) != nil || receipt.ActorID != g.ActorID || receipt.RepositoryID != r.Repository {
		return errors.New("native graph receipt does not bind submitted tasks")
	}
	if len(g.SubmittedTasks) == 0 {
		return errors.New("generation predates submitted-task records")
	}
	scoped := r
	scoped.Tasks = g.SubmittedTasks
	if err := scoped.Validate(); err != nil {
		return errors.New("submitted graph exceeds accepted scope")
	}
	issues, err := intentReceiptIssues(g.SubmittedTasks, g, receipt)
	if err != nil {
		return err
	}
	// Delivered natives are immutable history: verify them against the
	// previous acceptance pins, never the new scoped body.
	var prevWork *factory.IntentRevision
	priorTasks := map[string]factory.IntentTask{}
	var priorBindings []factory.IntentTaskBinding
	if len(g.Delivered) > 0 {
		if err := c.currentAcceptedIntent(ctx, r); err != nil {
			return err
		}
		history, err := c.Store.IntentHistory(ctx, r.Intent, r.Owner, 2)
		if err != nil {
			return err
		}
		if len(history) < 2 || history[0].ID != r.ID || history[1].ID != r.Predecessor {
			return errors.New("previous revision is unavailable")
		}
		priorGeneration, err := c.Store.IntentGeneration(ctx, history[1].ID)
		if err != nil {
			return err
		}
		if len(priorGeneration.SubmittedTasks) == 0 {
			return errors.New("previous generation predates submitted-task records")
		}
		pw := history[1]
		pw.Tasks = priorGeneration.SubmittedTasks
		prevWork = &pw
		for _, task := range pw.Tasks {
			priorTasks[task.Key] = task
		}
		for key := range g.Delivered {
			if _, ok := priorTasks[key]; !ok {
				return errors.New("delivered history unexpectedly mutated")
			}
		}
		priorBindings, err = c.Store.IntentTaskBindings(ctx, history[1].ID)
		if err != nil {
			return err
		}
	}
	tasks := map[string]factory.IntentTask{}
	for _, task := range g.SubmittedTasks {
		tasks[task.Key] = task
		if issues[task.Key].IssueID == 0 {
			return errors.New("native graph receipt missing submitted task")
		}
	}
	adopted := map[string]factory.IntentTaskBinding{}
	existing, err := c.Store.IntentTaskBindings(ctx, r.ID)
	if err != nil {
		return err
	}
	for _, b := range existing {
		decision, e := c.Store.AcceptanceDecision(ctx, b.AcceptanceID)
		if e == nil {
			entry := issues[b.TaskKey]
			if decision.ID != generatedDecisionID("acceptance", r.ID, b.TaskKey) || decision.Approver != r.Owner || decision.Repository != r.Repository || decision.IssueIndex != strconv.FormatInt(entry.IssueIndex, 10) || b.IssueID != entry.IssueID {
				return store.ErrCommandConflict
			}
			adopted[b.TaskKey] = b
		} else if !errors.Is(e, store.ErrNotFound) {
			return e
		}
	}
	active := map[string]bool{}
	var adopt func(string) error
	adopt = func(key string) error {
		if _, ok := adopted[key]; ok {
			return nil
		}
		if active[key] {
			return errors.New("cyclic accepted graph")
		}
		active[key] = true
		task := tasks[key]
		for _, dep := range task.DependsOn {
			if err := adopt(dep); err != nil {
				return err
			}
		}
		if err := c.currentAcceptedIntent(ctx, r); err != nil {
			return err
		}
		ownerRev, err := c.Graph.CheckIntentOwner(ctx, r.Repository, r.Owner, g.ActorID)
		if err != nil {
			return err
		}
		entry := issues[key]
		native, nativeRev, err := c.Graph.ReadGeneratedIssue(ctx, r.Repository, entry.IssueIndex, g.ActorID)
		if err != nil {
			return err
		}
		if nativeRev != ownerRev {
			return refuseAcceptance(RefusalStaleEvidence)
		}
		issue := native.Issue
		if issue == nil || !issue.Visible || !issue.Complete || issue.IsPull || issue.IsLocked || issue.ID != strconv.FormatInt(entry.IssueID, 10) || issue.Index != strconv.FormatInt(entry.IssueIndex, 10) || issue.Provenance.PosterID != strconv.FormatInt(g.ActorID, 10) {
			return refuseAcceptance(RefusalObjectiveChanged)
		}
		// Delivered records adopt without mutation: the native stays closed
		// with its pinned content while bindings and acceptances chain on.
		// A reopened or rewritten record refuses rather than rebinding
		// history it no longer matches.
		if ref, delivered := g.Delivered[key]; delivered {
			if entry.IssueID != ref.IssueID || entry.IssueIndex != ref.IssueIndex || entry.ContentVersion != ref.ContentVersion {
				return errors.New("delivered history unexpectedly mutated")
			}
			if !issue.IsClosed {
				return refuseAcceptance(RefusalObjectiveChanged)
			}
			stable, err := c.deliveredStable(ctx, priorBindings, priorTasks[key], entry, issue, *prevWork)
			if err != nil {
				return err
			}
			if !stable {
				return refuseAcceptance(RefusalObjectiveChanged)
			}
		} else if issue.Title != intentTaskTitle(task) || issue.Content != intentTaskBody(scoped, task) || issue.ContentVersion != entry.ContentVersion {
			return refuseAcceptance(RefusalObjectiveChanged)
		}
		if native.Dependencies == nil || !native.Dependencies.Complete || native.Dependencies.IssueID != issue.ID || len(native.Dependencies.Items) != len(task.DependsOn) {
			return refuseAcceptance(RefusalEdgeChanged)
		}
		expected := map[string]factory.IntentTaskBinding{}
		for _, dep := range task.DependsOn {
			b := adopted[dep]
			expected[strconv.FormatInt(b.IssueID, 10)] = b
		}
		decision := factory.Acceptance{ID: generatedDecisionID("acceptance", r.ID, key), Repository: r.Repository, IssueIndex: issue.Index, Approver: r.Owner, NativeRev: nativeRev, TitleDigest: issue.TitleDigest, TitleOccurrence: generatedTitleOccurrence(issue), ContentDigest: issue.ContentDigest, ContentVersion: issue.ContentVersion}
		head, headErr := c.Store.AcceptanceHead(ctx, r.Repository, entry.IssueIndex)
		if headErr == nil && head != decision.ID {
			decision.Predecessor = head
		} else if headErr != nil && !errors.Is(headErr, store.ErrNotFound) {
			return headErr
		}
		for _, edge := range native.Dependencies.Items {
			binding, ok := expected[edge.DependencyID]
			if !ok || !edge.Visible || !edge.Complete {
				return refuseAcceptance(RefusalEdgeChanged)
			}
			delete(expected, edge.DependencyID)
			decision.Prerequisites = append(decision.Prerequisites, factory.AcceptedPrerequisite{Occurrence: edge.OccurrenceID, DependsOn: edge.DependencyID, EndpointRepo: r.Repository, EndpointIssue: binding.IssueIndex, PrereqAcceptance: binding.AcceptanceID, Outcome: factory.PrereqCode})
		}
		if len(expected) > 0 {
			return refuseAcceptance(RefusalEdgeChanged)
		}
		// Interrupted adoption may have recorded acceptance before its binding.
		// Reuse only that exact deterministic decision, never a similar issue.
		createdUnix := issue.CreatedUnix
		if createdUnix <= 0 {
			createdUnix = issue.Provenance.CreatedUnix
		}
		binding := factory.IntentTaskBinding{RevisionID: r.ID, TaskKey: key, IssueID: entry.IssueID, IssueIndex: entry.IssueIndex, ContentVersion: entry.ContentVersion, AcceptanceID: decision.ID, CreatedUnix: createdUnix}
		// Persist authority provenance before acceptance can make the issue
		// eligible to any independently running coordinator pass.
		if err = c.Store.BindIntentTaskAcceptance(ctx, binding); err != nil {
			return err
		}

		old, e := c.Store.AcceptanceDecision(ctx, decision.ID)
		if errors.Is(e, store.ErrNotFound) {
			_, e = c.AdmitAcceptance(ctx, generatedDecisionID("command", r.ID, key), "accepted-intent:"+r.ID, decision)
		} else if e == nil {
			if old.Approver != decision.Approver || old.Repository != decision.Repository || old.IssueIndex != decision.IssueIndex || old.TitleDigest != decision.TitleDigest || old.ContentDigest != decision.ContentDigest {
				return store.ErrCommandConflict
			}
			if e = c.verifyAcceptance(ctx, old, AcceptanceEvidence{Revision: old.NativeRev, Issue: AcceptanceIssueView{Index: issue.Index, Visible: true, TitleDigest: issue.TitleDigest, TitleOccurrence: generatedTitleOccurrence(issue), ContentDigest: issue.ContentDigest, ContentVer: issue.ContentVersion}, Dependencies: generatedEdges(native.Dependencies)}); e != nil {
				return e
			}
		}
		if e != nil {
			return e
		}
		adopted[key] = binding
		active[key] = false
		return nil
	}
	for _, task := range g.SubmittedTasks {
		if err := adopt(task.Key); err != nil {
			return err
		}
	}
	return nil
}

func generatedEdges(page *sdk.SnapshotDependencyPage) []AcceptanceEdge {
	edges := []AcceptanceEdge{}
	for _, edge := range page.Items {
		edges = append(edges, AcceptanceEdge{Occurrence: edge.OccurrenceID, DependsOn: edge.DependencyID, Visible: edge.Visible})
	}
	return edges
}

func generatedTitleOccurrence(issue *sdk.SnapshotIssue) string {
	occurrence := factory.OriginalOccurrence
	for _, event := range issue.Lifecycle {
		if event.Kind == "retitled" {
			occurrence = event.OccurrenceID
		}
	}
	return occurrence
}

// Native titles summarize a task; the body retains its complete accepted
// objective and outcomes. UTF-8 byte width follows the native title column.
func intentTaskTitle(task factory.IntentTask) string {
	title := strings.TrimSpace(task.Objective)
	if line, _, ok := strings.Cut(title, "\n"); ok {
		title = strings.TrimSpace(line)
	}
	if len(title) > 255 {
		end := 0
		for index := range title {
			if index > 255 {
				break
			}
			end = index
		}
		if end == 0 {
			end = 255
		}
		title = strings.TrimSpace(title[:end])
	}
	return title
}

func (c *Coordinator) cancelIntentGeneration(ctx context.Context, g factory.IntentGeneration) (factory.IntentGeneration, error) {
	record, err := c.Graph.CancelGraph(ctx, g.OperationID)
	if err != nil {
		return g, err
	}
	if record.OperationID != g.OperationID {
		return g, errors.New("native graph cancellation identity differs")
	}
	switch record.EffectState {
	case sdk.EffectNotCommitted:
		g.State = "refused"
		g.Receipt = record.Receipt
	case sdk.EffectCommitted:
		g.State = "committed"
		g.Receipt = record.Receipt
	default:
		return g, nil
	}
	return g, c.Store.RecordIntentGeneration(context.WithoutCancel(ctx), g)
}

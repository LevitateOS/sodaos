package control

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	sdk "forgejo.org/extension-sdk"
	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/forgejo"
	"github.com/levitateos/sodaos/internal/store"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
	"unicode/utf8"
)

type generatedGraphFixture struct {
	mu           sync.Mutex
	revision     int64
	actor        int64
	owner        bool
	records      map[string]sdk.OperationRecord
	issues       map[int64]sdk.NativeSnapshot
	submissions  int
	beforeSubmit func(factory.IntentGeneration) error
}

func (f *generatedGraphFixture) CheckIntentOwner(context.Context, int64, int64, int64) (int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if !f.owner {
		return 0, errors.New("owner revoked")
	}
	return f.revision, nil
}
func (f *generatedGraphFixture) SubmitGraph(ctx context.Context, g factory.IntentGeneration, repo int64) (sdk.OperationRecord, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.beforeSubmit != nil {
		if err := f.beforeSubmit(g); err != nil {
			return sdk.OperationRecord{}, err
		}
	}
	if old, ok := f.records[g.OperationID]; ok {
		return old, nil
	}
	var p sdk.IssueGraphPayload
	if err := json.Unmarshal(g.Payload, &p); err != nil {
		return sdk.OperationRecord{}, err
	}
	if err := sdk.ValidateIssueGraphPayload(p); err != nil {
		return sdk.OperationRecord{}, err
	}
	f.submissions++
	f.revision++
	receipt := sdk.IssueGraphReceipt{ActorID: g.ActorID, RepositoryID: repo}
	ids := map[string]int64{}
	for _, c := range p.Creates {
		index := int64(len(f.issues) + 1)
		id := index + 100
		ids[c.Key] = id
		issue := &sdk.SnapshotIssue{ID: strconv.FormatInt(id, 10), Index: strconv.FormatInt(index, 10), Title: c.Title, Content: c.Body, TitleDigest: forgejo.ContentDigest(c.Title), ContentDigest: forgejo.ContentDigest(c.Body), Visible: true, Complete: true, Provenance: sdk.SnapshotCreationProvenance{PosterID: strconv.FormatInt(g.ActorID, 10)}}
		f.issues[index] = sdk.NativeSnapshot{RepositoryID: strconv.FormatInt(repo, 10), Issue: issue, Dependencies: &sdk.SnapshotDependencyPage{IssueID: issue.ID, Complete: true, Items: []sdk.SnapshotDependency{}}}
		receipt.Issues = append(receipt.Issues, sdk.IssueGraphIssueReceipt{Key: c.Key, IssueID: id, IssueIndex: index})
	}

	for _, edit := range p.Edits {
		index := edit.IssueID - 100
		snapshot := f.issues[index]
		issue := snapshot.Issue
		if issue == nil || issue.Title != edit.ExpectedTitle || issue.Content != edit.ExpectedBody || issue.ContentVersion != edit.ExpectedContentVersion {
			return sdk.OperationRecord{}, errors.New("stale fake edit")
		}
		if issue.Title != edit.Title {
			issue.Lifecycle = append(issue.Lifecycle, sdk.SnapshotLifecycleEvent{OccurrenceID: strconv.FormatInt(1000+f.revision, 10), Kind: "retitled", OldTitle: issue.Title, NewTitle: edit.Title})
		}
		issue.Title = edit.Title
		issue.TitleDigest = forgejo.ContentDigest(edit.Title)
		if issue.Content != edit.Body {
			issue.Content = edit.Body
			issue.ContentDigest = forgejo.ContentDigest(edit.Body)
			issue.ContentVersion++
		}
		receipt.Issues = append(receipt.Issues, sdk.IssueGraphIssueReceipt{IssueID: edit.IssueID, IssueIndex: index, ContentVersion: issue.ContentVersion})
	}
	resolve := func(r sdk.IssueGraphRef) int64 {
		if r.IssueID > 0 {
			return r.IssueID
		}
		return ids[r.Key]
	}
	for i, e := range p.Dependencies {
		id, dep := resolve(e.Issue), resolve(e.Prerequisite)
		snapshot := f.issues[id-100]
		if e.Remove {
			kept := []sdk.SnapshotDependency{}
			for _, old := range snapshot.Dependencies.Items {
				if old.DependencyID != strconv.FormatInt(dep, 10) {
					kept = append(kept, old)
				}
			}
			snapshot.Dependencies.Items = kept
		} else {
			snapshot.Dependencies.Items = append(snapshot.Dependencies.Items, sdk.SnapshotDependency{OccurrenceID: strconv.Itoa(i + 700 + int(f.revision)*100), IssueID: strconv.FormatInt(id, 10), DependencyID: strconv.FormatInt(dep, 10), Visible: true, Complete: true})
		}
		snapshot.Dependencies.Total = len(snapshot.Dependencies.Items)
		f.issues[id-100] = snapshot
		receipt.Dependencies = append(receipt.Dependencies, sdk.IssueGraphDependencyReceipt{IssueID: id, PrerequisiteID: dep, Removed: e.Remove})
	}
	raw, _ := json.Marshal(receipt)
	record := sdk.OperationRecord{OperationID: g.OperationID, ActorID: strconv.FormatInt(g.ActorID, 10), RepositoryID: strconv.FormatInt(repo, 10), Kind: sdk.OperationKindIssueGraph, Outcome: sdk.EffectCommitted, EffectState: sdk.EffectCommitted, CompletionState: sdk.CompletionComplete, Receipt: raw}
	f.records[g.OperationID] = record
	return record, nil
}
func (f *generatedGraphFixture) LookupGraph(_ context.Context, g factory.IntentGeneration, _ int64) (sdk.OperationLookup, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	r, ok := f.records[g.OperationID]
	if !ok {
		return sdk.OperationLookup{OperationID: g.OperationID, Status: sdk.BackgroundOutcomeNotObserved}, nil
	}
	return sdk.OperationLookup{OperationID: g.OperationID, Status: r.EffectState, Record: &r}, nil
}
func (f *generatedGraphFixture) CancelGraph(_ context.Context, id string) (sdk.OperationRecord, error) {
	return sdk.OperationRecord{OperationID: id, EffectState: sdk.EffectNotCommitted, CancellationStatus: sdk.CancellationCancelled}, nil
}
func (f *generatedGraphFixture) ReadGeneratedIssue(_ context.Context, _, index, _ int64) (sdk.NativeSnapshot, int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	s, ok := f.issues[index]
	if !ok {
		return sdk.NativeSnapshot{}, 0, errors.New("missing issue")
	}
	return s, f.revision, nil
}
func (f *generatedGraphFixture) ReadAcceptanceEvidence(_ context.Context, _, index string, _ []string) (AcceptanceEvidence, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	n, _ := strconv.ParseInt(index, 10, 64)
	s := f.issues[n]
	if s.Issue == nil {
		return AcceptanceEvidence{}, errors.New("missing issue")
	}
	i := s.Issue
	return AcceptanceEvidence{Revision: f.revision, Issue: AcceptanceIssueView{ID: i.ID, Index: i.Index, TitleDigest: i.TitleDigest, ContentDigest: i.ContentDigest, ContentVer: i.ContentVersion, TitleOccurrence: generatedTitleOccurrence(i), Visible: true, PosterID: i.Provenance.PosterID}, Dependencies: generatedEdges(s.Dependencies)}, nil
}
func acceptedGenerationFixture(t *testing.T) (*Coordinator, *generatedGraphFixture, factory.IntentRevision) {
	t.Helper()
	c := coordinatorFixture(t, nil, nil)
	r := factory.IntentRevision{ID: "d111111111111111111111111", Intent: "d222222222222222222222222", Owner: 7, Repository: 42, Objective: "Build a tracker", State: "accepted", MaxIssues: 10, PlanningCalls: 12, CreatedUnix: 1, Ideas: []factory.ProductIdea{{ID: "d333333333333333333333333", Description: "Track requests", Consequence: "Clear ownership", Decision: "accept"}}, Tasks: []factory.IntentTask{{Key: "d444444444444444444444444", Objective: "Request entry", Outcomes: []string{"Capture requests"}, IdeaIDs: []string{"d333333333333333333333333"}}, {Key: "d555555555555555555555555", Objective: "Request list", Outcomes: []string{"Show requests"}, IdeaIDs: []string{"d333333333333333333333333"}, DependsOn: []string{"d444444444444444444444444"}}}}
	if err := c.Store.SaveIntentRevision(context.Background(), r); err != nil {
		t.Fatal(err)
	}
	f := &generatedGraphFixture{revision: 10, actor: 91, owner: true, records: map[string]sdk.OperationRecord{}, issues: map[int64]sdk.NativeSnapshot{}}
	f.beforeSubmit = func(g factory.IntentGeneration) error {
		stored, err := c.Store.IntentGeneration(context.Background(), g.RevisionID)
		if err != nil {
			return err
		}
		var a, b any
		_ = json.Unmarshal(stored.Payload, &a)
		_ = json.Unmarshal(g.Payload, &b)
		aa, _ := json.Marshal(a)
		bb, _ := json.Marshal(b)
		if string(aa) != string(bb) || stored.OperationID != g.OperationID {
			return errors.New("native intent was not durable before send")
		}
		return nil
	}
	c.Graph, c.GraphActorID, c.AcceptanceReads = f, 91, f
	c.Store.PriorityCallBudget = 32
	c.Planning = &PlanningJudgments{Journal: c.Store, Funding: c.Store, Chooser: generatedGraphChooser{}}
	return c, f, r
}
func TestIntentGraphPayloadBindsProductDirection(t *testing.T) {
	r := factory.IntentRevision{ID: "d111111111111111111111111", Intent: "d222222222222222222222222", Owner: 7, Repository: 42, Objective: "Tracker", State: "accepted", MaxIssues: 5, CreatedUnix: 1, Ideas: []factory.ProductIdea{{ID: "d333333333333333333333333", Description: "Track", Consequence: "Visibility", Decision: "accept"}}, Tasks: []factory.IntentTask{{Key: "d444444444444444444444444", Objective: "Entry", Outcomes: []string{"Capture requests"}, IdeaIDs: []string{"d333333333333333333333333"}}}}
	p, err := IntentGraphPayload(r)
	if err != nil {
		t.Fatal(err)
	}
	if p.Creates[0].Body != intentTaskBody(r, r.Tasks[0]) || p.Creates[0].Key != r.Tasks[0].Key {
		t.Fatal("native provenance differs")
	}
	r.Tasks[0].Objective = strings.Repeat("é", 300) + "\nDetailed accepted objective"
	long, err := IntentGraphPayload(r)
	if err != nil {
		t.Fatal(err)
	}
	if len(long.Creates[0].Title) > 255 || !utf8.ValidString(long.Creates[0].Title) || !strings.Contains(long.Creates[0].Body, r.Tasks[0].Objective) {
		t.Fatal("concise native title lost full accepted objective")
	}
	r.State = "draft"
	if _, err = IntentGraphPayload(r); err == nil {
		t.Fatal("unaccepted direction generated")
	}
}
func TestIntentGraphGeneratesAndAdoptsOnceWithNativeProvenance(t *testing.T) {
	c, f, r := acceptedGenerationFixture(t)
	ctx := context.Background()
	g, err := c.GenerateAcceptedIntent(ctx, r.Intent, r.Owner)
	if err != nil {
		t.Fatal(err)
	}
	if g.State != "completed" {
		t.Fatalf("generation=%+v", g)
	}
	bindings, err := c.Store.IntentTaskBindings(ctx, r.ID)
	if err != nil || len(bindings) != 2 {
		t.Fatalf("bindings=%v err=%v", bindings, err)
	}
	decisions := map[int64]factory.Acceptance{}
	for _, b := range bindings {
		d, err := c.Store.AcceptanceDecision(ctx, b.AcceptanceID)
		if err != nil {
			t.Fatal(err)
		}
		if d.Approver != 7 || d.Initial {
			t.Fatal("service author impersonated human or creation adoption")
		}
		decisions[b.IssueIndex] = d
	}
	if len(decisions[2].Prerequisites) != 1 || decisions[2].Prerequisites[0].PrereqAcceptance != decisions[1].ID {
		t.Fatal("native dependency did not bind prerequisite acceptance")
	}
	if _, err = c.GenerateAcceptedIntent(ctx, r.Intent, r.Owner); err != nil {
		t.Fatal(err)
	}
	if f.submissions != 1 {
		t.Fatal("replay created another graph")
	}
	f.owner = false
	if _, err = c.GenerateAcceptedIntent(ctx, r.Intent, r.Owner); err == nil {
		t.Fatal("revoked owner retained generation authority")
	}
}
func TestIntentGraphServingOwnerRecoversDurableAcceptedDirection(t *testing.T) {
	c, f, r := acceptedGenerationFixture(t)
	ctx := context.Background()
	if err := c.Start(ctx, filepath.Join(t.TempDir(), "coordinator.lock")); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = c.Close() })
	g, err := c.Store.IntentGeneration(ctx, r.ID)
	if err != nil || g.State != "completed" {
		t.Fatalf("accepted graph not recovered by serving owner: %v %v", g.State, err)
	}
	if f.submissions != 1 {
		t.Fatal("serving recovery did not send exactly one graph")
	}
}
func TestIntentGraphInterruptedNativeResponseReusesStoredOperation(t *testing.T) {
	c, f, r := acceptedGenerationFixture(t)
	ctx := context.Background()
	p, err := IntentGraphPayload(r)
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(p)
	g := factory.IntentGeneration{RevisionID: r.ID, IntentID: r.Intent, OperationID: "intent-" + r.ID, ActorID: 91, NativeRev: 10, AuthRevision: r.ID + ":" + r.Digest(), NotAfter: time.Now().Add(time.Minute).Unix(), Payload: raw, IntentDigest: r.Digest(), State: "pending"}
	if err = c.Store.BeginIntentGeneration(ctx, g); err != nil {
		t.Fatal(err)
	}
	if _, err = f.SubmitGraph(ctx, g, r.Repository); err != nil {
		t.Fatal(err)
	}
	recovered, err := c.GenerateAcceptedIntent(ctx, r.Intent, r.Owner)
	if err != nil {
		t.Fatal(err)
	}
	if recovered.State != "completed" || f.submissions != 1 {
		t.Fatal("native receipt replay created new work")
	}
	revised := r
	revised.Predecessor = r.ID
	revised.ID = "d666666666666666666666666"
	revised.State = "draft"
	if err = c.Store.SaveIntentRevision(ctx, revised); err != nil {
		t.Fatal(err)
	}
	bindings, _ := c.Store.IntentTaskBindings(ctx, r.ID)
	for _, b := range bindings {
		current, generated, e := c.Store.GeneratedIntentCurrent(ctx, b.AcceptanceID)
		if e != nil || current || !generated {
			t.Fatal(fmt.Sprintf("old child authority survived correction: %v %v %v", current, generated, e))
		}
	}
	if _, err = c.GenerateAcceptedIntent(ctx, r.Intent, r.Owner); err == nil {
		t.Fatal("draft correction generated")
	}
	_ = store.ErrStaleRevision
}

func TestIntentGraphReviewedSplitAndConsolidationPreserveNativeHistory(t *testing.T) {
	c, f, r := acceptedGenerationFixture(t)
	ctx := context.Background()
	if _, err := c.GenerateAcceptedIntent(ctx, r.Intent, r.Owner); err != nil {
		t.Fatal(err)
	}
	original, _ := c.Store.IntentTaskBindings(ctx, r.ID)
	oldByKey := map[string]factory.IntentTaskBinding{}
	for _, b := range original {
		oldByKey[b.TaskKey] = b
	}
	revised := r
	revised.Predecessor = r.ID
	revised.ID = "d666666666666666666666666"
	revised.Tasks = append([]factory.IntentTask(nil), r.Tasks...)
	revised.Tasks[0].Objective = "Request entry and validation"
	revised.Tasks = append(revised.Tasks, factory.IntentTask{Key: "d777777777777777777777777", Objective: "Validate submitted requests", Outcomes: []string{"Show invalid input"}, IdeaIDs: r.Tasks[0].IdeaIDs, DependsOn: []string{r.Tasks[0].Key}})
	revised.Tasks[1].DependsOn = []string{revised.Tasks[2].Key}
	if err := c.Store.SaveIntentRevision(ctx, revised); err != nil {
		t.Fatal(err)
	}
	split, err := c.GenerateAcceptedIntent(ctx, revised.Intent, revised.Owner)
	if err != nil {
		t.Fatal(err)
	}
	if split.State != "completed" {
		t.Fatal("split adoption incomplete")
	}
	splitBindings, _ := c.Store.IntentTaskBindings(ctx, revised.ID)
	splitByKey := map[string]factory.IntentTaskBinding{}
	for _, b := range splitBindings {
		splitByKey[b.TaskKey] = b
	}
	for _, task := range r.Tasks {
		if splitByKey[task.Key].IssueID != oldByKey[task.Key].IssueID {
			t.Fatal("split replaced existing native issue")
		}
	}
	if f.issues[1].Issue.ContentVersion != 1 || len(f.issues[1].Issue.Lifecycle) != 1 {
		t.Fatal("split lost native body/title history")
	}
	consolidated := revised
	consolidated.Predecessor = revised.ID
	consolidated.ID = "d888888888888888888888888"
	consolidated.Tasks = []factory.IntentTask{revised.Tasks[0], revised.Tasks[1]}
	consolidated.Tasks[1].DependsOn = []string{consolidated.Tasks[0].Key}
	if err = c.Store.SaveIntentRevision(ctx, consolidated); err != nil {
		t.Fatal(err)
	}
	if _, err = c.GenerateAcceptedIntent(ctx, consolidated.Intent, consolidated.Owner); err != nil {
		t.Fatal(err)
	}
	retired := f.issues[splitByKey[revised.Tasks[2].Key].IssueIndex].Issue
	if retired.IsClosed || !strings.Contains(retired.Content, "Superseded by accepted product revision: "+consolidated.ID) {
		t.Fatal("consolidation erased or closed retired native work")
	}
	current, generated, err := c.Store.GeneratedIntentCurrent(ctx, splitByKey[revised.Tasks[2].Key].AcceptanceID)
	if err != nil || current || !generated {
		t.Fatal("retired task retained authority")
	}
}

type generatedGraphChooser struct{}

func (generatedGraphChooser) Select(_ context.Context, _ any, _ string, choices map[string]any) (factory.PriorityAnswer, error) {
	selected := "keep"
	if _, ok := choices[selected]; !ok {
		selected = "plan_stable_scope"
	}
	probabilities := map[string]float64{}
	for key := range choices {
		probabilities[key] = 0
	}
	probabilities[selected] = 1
	return factory.PriorityAnswer{Selected: selected, Model: "synthetic-model", Confidence: 1, Probabilities: probabilities}, nil
}

func TestIntentGraphStaleDurablePacketCancelsBeforeNativeEffects(t *testing.T) {
	c, f, r := acceptedGenerationFixture(t)
	ctx := context.Background()
	p, err := IntentGraphPayload(r)
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(p)
	g := factory.IntentGeneration{RevisionID: r.ID, IntentID: r.Intent, OperationID: "intent-" + r.ID, ActorID: 91, NativeRev: 9, AuthRevision: r.ID + ":" + r.Digest(), NotAfter: time.Now().Add(time.Minute).Unix(), Payload: raw, IntentDigest: r.Digest(), State: "pending"}
	if err = c.Store.BeginIntentGeneration(ctx, g); err != nil {
		t.Fatal(err)
	}
	_, err = c.GenerateAcceptedIntent(ctx, r.Intent, r.Owner)
	if err == nil {
		t.Fatal("stale native revision was submitted")
	}
	stored, err := c.Store.IntentGeneration(ctx, r.ID)
	if err != nil || stored.State != "refused" || f.submissions != 0 {
		t.Fatal("definite pre-send cancellation not retained", stored.State, err, f.submissions)
	}
}

func TestIntentGraphLongProposalRetainsAcceptedSemanticsWithinPacket(t *testing.T) {
	r := factory.IntentRevision{ID: "d111111111111111111111111", Intent: "d222222222222222222222222", Owner: 7, Repository: 42, Objective: strings.Repeat("x", 16000), State: "draft", MaxIssues: 10, CreatedUnix: 1}
	ids := make([]string, 104)
	for i := range ids {
		ids[i] = fmt.Sprintf("d%024x", 1000+i)
	}
	r = factory.ProposeIntent(r, ids)
	if err := r.Validate(); err != nil {
		t.Fatal(err)
	}
	r.State = "accepted"
	for i := range r.Ideas {
		r.Ideas[i].Decision = "accept"
	}
	payload, err := IntentGraphPayload(r)
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(payload)
	if len(raw) > sdk.MaxIssueGraphPayloadBytes {
		t.Fatal("long baseline exceeds native packet")
	}
	for _, issue := range payload.Creates {
		if strings.Count(issue.Body, r.Objective) != 1 {
			t.Fatal("native issue dropped or duplicated full accepted objective")
		}
		if !strings.Contains(issue.Body, r.Ideas[1].Description) || !strings.Contains(issue.Body, r.Ideas[1].Consequence) {
			t.Fatal("accepted access assumption absent from native requirements")
		}
	}
}

func TestIntentGraphBodyIncludesReviewedChoicesAndCorrections(t *testing.T) {
	r := factory.IntentRevision{ID: "d111111111111111111111111", Intent: "d222222222222222222222222", Owner: 7, Repository: 42, Objective: "Track team requests", State: "accepted", MaxIssues: 10, CreatedUnix: 1, Ideas: []factory.ProductIdea{{ID: "d333333333333333333333333", Description: "Ask for full address", Decision: "edit", Correction: "Ask only for a work email", Consequence: "Avoid collecting unnecessary personal data"}, {ID: "d444444444444444444444444", Description: "Stay within existing access", Assumption: true, Decision: "edit", Correction: "Permit only current project members", Consequence: "No new account enrollment"}}, Tasks: []factory.IntentTask{{Key: "d555555555555555555555555", Objective: "Ask only for a work email", Outcomes: []string{"Accept member request"}, IdeaIDs: []string{"d333333333333333333333333"}}}}
	payload, err := IntentGraphPayload(r)
	if err != nil {
		t.Fatal(err)
	}
	body := payload.Creates[0].Body
	for _, text := range []string{r.Objective, r.Ideas[0].Description, r.Ideas[0].Correction, r.Ideas[0].Consequence, r.Ideas[1].Correction, r.Ideas[1].Consequence} {
		if !strings.Contains(body, text) {
			t.Fatalf("native requirements lost %q", text)
		}
	}
	if strings.Count(body, r.Tasks[0].Objective) != 1 {
		t.Fatal("task and identical correction duplicated")
	}
}

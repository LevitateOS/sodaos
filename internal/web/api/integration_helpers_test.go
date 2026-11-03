// Helpers retained for the kept integration tests after unit-test removal.
package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	extensions "forgejo.org/extension-sdk"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/levitateos/factory/pkg/factory"
	"github.com/levitateos/factory/pkg/factory/control"
	"github.com/levitateos/factory/pkg/project"
	"github.com/levitateos/factory/pkg/store"
	"github.com/levitateos/sodaos/internal/forgejo"
)

// repository_issues page per cursor, and one body per issue index.
// Tests supply deterministic fixtures; production serves the
// background channel.
type completionReaderFake struct {
	revision int64
	idle     bool
	pages    map[string][]string
	bodies   map[string]forgejo.NativeSnapshot
	revErr   error
	snapErr  error
	moveRev  func(calls int) int64
	calls    int
	revs     int
	// refTip is the served tip for any requested refs-only read (""
	// serves no refs, proving the missing-refusal); refTipMove
	// scripts a tip change keyed on the snapshot-call count for the
	// capture-pair tests; hideRef serves the requested ref redacted.
	// lastReq records the most recent snapshot request so tests can
	// assert the refs-only shape.
	refTip     string
	refTipMove func(snapCalls int) string
	hideRef    bool
	lastReq    forgejo.SnapshotRequest
}

func (f *completionReaderFake) ReadNativeRevision(context.Context) (extensions.NativeRevisionObservation, error) {
	f.revs++
	if f.revErr != nil {
		return extensions.NativeRevisionObservation{}, f.revErr
	}
	revision := f.revision
	if f.moveRev != nil {
		revision = f.moveRev(f.calls)
	}
	return extensions.NativeRevisionObservation{Revision: revision, Idle: f.idle}, nil
}

func (f *completionReaderFake) ReadSnapshot(_ context.Context, _ extensions.CredentialFile, req forgejo.SnapshotRequest) (forgejo.NativeSnapshot, error) {
	f.calls++
	f.lastReq = req
	if f.snapErr != nil {
		return forgejo.NativeSnapshot{}, f.snapErr
	}
	revision := f.revision
	if f.moveRev != nil {
		revision = f.moveRev(f.calls)
	}
	if len(req.Families) == 1 && req.Families[0] == forgejo.FamilyRefs {
		tip := f.refTip
		if f.refTipMove != nil {
			tip = f.refTipMove(f.calls)
		}
		refs := []forgejo.RefEvidence{}
		for _, want := range req.Refs {
			if tip == "" {
				continue
			}
			ref := forgejo.RefEvidence{Ref: want, Complete: true, Visible: !f.hideRef}
			if !f.hideRef {
				ref.OID, ref.Exists = tip, true
			}
			refs = append(refs, ref)
		}
		return forgejo.NativeSnapshot{Revision: revision, RepositoryID: req.RepositoryID, Refs: refs}, nil
	}
	if len(req.Families) == 1 && req.Families[0] == forgejo.FamilyRepositoryIssues {
		indexes, ok := f.pages[req.Cursor]
		if !ok {
			return forgejo.NativeSnapshot{}, errors.New("unexpected cursor")
		}
		items := make([]forgejo.RepositoryIssueItem, 0, len(indexes))
		for _, index := range indexes {
			items = append(items, forgejo.RepositoryIssueItem{IssueID: index + "0", IssueIndex: index, Visible: true})
		}
		page := &forgejo.RepositoryIssuesPage{Items: items, Complete: true}
		if next, ok := completionNextCursor[req.Cursor]; ok {
			page.Complete = false
			page.NextCursor = next
		}
		return forgejo.NativeSnapshot{Revision: revision, RepositoryID: req.RepositoryID, RepositoryIssues: page}, nil
	}
	snapshot, ok := f.bodies[req.IssueIndex]
	if !ok {
		return forgejo.NativeSnapshot{}, errors.New("unexpected issue")
	}
	snapshot.Revision = revision
	snapshot.RepositoryID = req.RepositoryID
	return snapshot, nil
}

// completionNextCursor chains the scripted pages: cursor "" holds two
// issues with continuation "2", cursor "2" terminates.
var completionNextCursor = map[string]string{"": "2"}

func completionBody(index, title, body string, deps ...string) forgejo.NativeSnapshot {
	edges := make([]forgejo.DependencyEvidence, 0, len(deps))
	for i, dep := range deps {
		edges = append(edges, forgejo.DependencyEvidence{
			OccurrenceID: strconv.Itoa(700 + i), IssueID: index + "0",
			DependencyID: dep, Visible: true, Complete: true,
		})
	}
	return forgejo.NativeSnapshot{
		Issue: &forgejo.IssueEvidence{
			ID: index + "0", Index: index, Title: title, Content: body,
			TitleDigest: forgejo.ContentDigest(title), ContentDigest: forgejo.ContentDigest(body),
			ContentVersion: 1, Visible: true, Complete: true,
		},
		Dependencies: &forgejo.DependencyPage{
			IssueID: index + "0", Items: edges, Total: len(edges), Complete: true,
		},
	}
}

func completionTwoPageFake() *completionReaderFake {
	return &completionReaderFake{
		revision: 41, idle: true,
		pages: map[string][]string{"": {"1", "2"}, "2": {"3"}},
		bodies: map[string]forgejo.NativeSnapshot{
			"1": completionBody("1", "title 1", "body 1"),
			"2": completionBody("2", "title 2", "body 2", "10"),
			"3": completionBody("3", "title 3", "body 3"),
		},
	}
}

func completionTestLocal() factory.CompletionLocal {
	acceptance := factory.Acceptance{
		ID: "d" + strings.Repeat("0", 24), Repository: 7, IssueIndex: "3",
		Approver: 5, NativeRev: 41,
		TitleDigest: strings.Repeat("a", 64), TitleOccurrence: factory.OriginalOccurrence,
		ContentDigest: strings.Repeat("b", 64), ContentVersion: 2,
		Sources: []factory.SelectedSource{{ID: "11", ContentVersion: 0, Digest: strings.Repeat("c", 64)}},
	}
	material := factory.CompletionMaterial{
		Intents:     []factory.CompletionIntentScope{},
		Acceptances: []factory.Acceptance{acceptance},
		Withdrawals: []factory.CompletionWithdrawal{},
		Controls: []factory.IssueControl{{
			Repository: 7, Issue: 7,
			Readiness: factory.ReadinessQueued, Reason: factory.ReasonEligible,
			Fingerprint: strings.Repeat("1", 64), Authority: strings.Repeat("2", 64),
		}},
		Preparations: []factory.CompletionPreparation{},
		Plans:        []factory.InvestigationPlan{},
		PlanHeads:    []factory.CompletionPlanHead{},
		PlanTasks:    []factory.CompletionPlanTask{},
		Assignments:  []factory.CompletionAssignment{},
		Publications: []factory.CompletionPublication{},
		Merges:       []factory.CompletionMerge{},
		Checks:       []factory.CompletionCheck{},
		Authority: factory.CompletionAuthority{
			Effective: factory.EffectiveAuthority{
				Missing: []string{}, Effective: true, DispatchOpen: true,
				Authority: factory.AuthorityRef{
					RequirementsID: "d" + strings.Repeat("c", 24),
					ApprovalID:     "d" + strings.Repeat("e", 24),
				},
			},
			ProjectID:               "p" + strings.Repeat("d", 24),
			RequirementSourceCommit: strings.Repeat("c", 40),
			RequirementSetupDigest:  strings.Repeat("e", 64),
			RequirementInputsDigest: strings.Repeat("f", 64),
			ApprovalEffectsDigest:   strings.Repeat("0", 64),
			ApprovalReadinessDigest: strings.Repeat("1", 64),
			ApprovalVerified:        true,
			TargetBranch:            "refs/heads/main",
			Roles:                   []string{"soda-coder", "soda-reviewer"},
			ReassessmentMinutes:     factory.DefaultReassessmentMinutes,
		},
		Repository: 7,
	}
	digest, err := material.Digest()
	if err != nil {
		panic(err)
	}
	return factory.CompletionLocal{
		Material:             &material,
		SourceCommit:         strings.Repeat("c", 40),
		PreparationDigest:    strings.Repeat("d", 64),
		ScopeDigest:          digest,
		AuthorityFingerprint: "authority",
	}
}

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

func (f *fakeCompletionSourceStoreNATIVE96) CompletionSourceContext(_ context.Context, repository int64) (factory.CompletionAuthority, error) {
	f.contextCalls++
	if f.local.Material == nil {
		return factory.CompletionAuthority{}, errors.New("NATIVE96 fake store lacks its fixture local")
	}
	if f.local.Material.Repository != repository {
		return factory.CompletionAuthority{}, errors.New("NATIVE96 fake store: wrong canonical repository")
	}
	return f.local.Material.Authority, nil
}

func (f *fakeCompletionSourceStoreNATIVE96) CaptureCompletionLocalAtSource(_ context.Context, repository int64, expected factory.CompletionAuthority, tip string) (factory.CompletionLocal, error) {
	f.atSourceCalls++
	if f.rejectAtSource != nil {
		return factory.CompletionLocal{}, f.rejectAtSource
	}
	if f.local.Material == nil || f.local.Material.Repository != repository {
		return factory.CompletionLocal{}, errors.New("NATIVE96 fake store: wrong canonical capture repository")
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

type workerMCPTransport struct {
	id, secret string
}

func (t workerMCPTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	r = r.Clone(r.Context())
	r.Header.Set("X-Soda-Factory-Capability", t.id)
	r.Header.Set("Authorization", "Bearer "+t.secret)
	return http.DefaultTransport.RoundTrip(r)
}

// Declared MCP sources shared by the API integration journeys.
// fakeCatalogCapability is the one admitted test capability.
func fakeCatalogCapability() factory.MCPCapability {
	digest := strings.Repeat("a", 64)
	return factory.MCPCapability{
		ID: strings.Repeat("a", 32), AssignmentID: strings.Repeat("b", 32),
		RunID: "run-test", Generation: 1,
		Profile: factory.MCPProfileBuilder, Subject: "implement:42", Repository: 7,
		Project:           "p" + strings.Repeat("d", 24),
		PreparationDigest: digest, SourceDigest: "abc", AcceptedRevision: "rev-1",
		Authority: digest, DeadlineUnix: time.Now().Add(time.Hour).Unix(),
		State: factory.MCPCapabilityActive,
	}
}

type fakeCatalogRecord struct {
	receipt  factory.MCPWorkStatus
	terminal bool
}

// fakeCatalogSink mirrors store.RecordMCPSubmission semantics in
// memory: a repeated post key returns its original work, and a terminal
// result prevents new work posts.
type fakeCatalogSink struct {
	mu          sync.Mutex
	capability  factory.MCPCapability
	submissions map[string]fakeCatalogRecord
	posts       map[string]factory.MCPSubmission
	assignments map[string]factory.Assignment
	receipts    map[string]factory.PriorityReceipt
	runs        map[string]factory.Run
}

func newFakeCatalogSink() *fakeCatalogSink {
	capability := fakeCatalogCapability()
	sink := &fakeCatalogSink{
		capability:  capability,
		submissions: map[string]fakeCatalogRecord{},
		posts:       map[string]factory.MCPSubmission{},
		receipts:    map[string]factory.PriorityReceipt{},
		runs:        map[string]factory.Run{},
		assignments: map[string]factory.Assignment{
			capability.AssignmentID: {
				ID: capability.AssignmentID, Phase: string(factory.MCPPhaseOf(capability.Profile)),
				Stage: factory.AssignmentAssigned, Role: project.RoleCoder, Run: strings.Repeat("a", 32),
				SourceCommit: strings.Repeat("c", 40), Preparation: "prep-1",
				PromptSHA: strings.Repeat("d", 64),
			},
		},
	}
	assignment := sink.assignments[capability.AssignmentID]
	request, _ := control.ContextBundleRequest(capability, assignment, false, false)
	sink.capability.ContextSelectionSnapshot = request.Snapshot
	answer := factory.PriorityAnswer{Selected: factory.ContextBundleCurrent, Confidence: 1, Model: "qualified-http-fixture", Probabilities: map[string]float64{factory.ContextBundleCurrent: 1, factory.ContextBundleSnapshotHold: 0}}
	sink.receipts[request.Snapshot] = factory.PriorityReceipt{Snapshot: request.Snapshot, Purpose: factory.JudgmentContextBundle, Outcome: "context-bundle", Selected: answer.Selected, Calls: []factory.PriorityCall{{Request: request, Answer: answer}}}
	return sink
}

// HTTP catalog fixtures model an already-qualified upstream input owner.
// Real source/grant/producer cases use *store.Store in the bound PG fixture.
func (f *fakeCatalogSink) FactoryCurrentContext(_ context.Context, cap factory.MCPCapability, _ time.Time) (factory.CurrentContext, error) {
	return factory.CurrentContext{Version: "context-current-v1", Evidence: factory.CurrentContextEvidence{RequiredStreams: []string{"required_current"}, Availability: "complete", MissingReasons: []string{}}, Role: factory.CurrentRoleContext{Kind: cap.Profile}}, nil
}

func (f *fakeCatalogSink) FactoryRun(_ context.Context, id string) (factory.Run, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	run, ok := f.runs[id]
	if !ok {
		return factory.Run{}, store.ErrNotFound
	}
	return run, nil
}

func (f *fakeCatalogSink) PriorityJudgment(_ context.Context, snapshot string) (factory.PriorityReceipt, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	receipt, ok := f.receipts[snapshot]
	if !ok {
		return factory.PriorityReceipt{}, store.ErrNotFound
	}
	return receipt, nil
}

func (f *fakeCatalogSink) Assignment(_ context.Context, id string) (factory.Assignment, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	assignment, ok := f.assignments[id]
	if !ok {
		return factory.Assignment{}, store.ErrNotFound
	}
	return assignment, nil
}

func (f *fakeCatalogSink) GetMCPCapability(_ context.Context, id string) (factory.MCPCapability, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if id != f.capability.ID {
		return factory.MCPCapability{}, store.ErrNotFound
	}
	return f.capability, nil
}

func (f *fakeCatalogSink) record(submission factory.MCPSubmission, receipt factory.MCPWorkStatus) (factory.MCPWorkStatus, error) {
	key := submission.AssignmentID + "/" + submission.SubmissionID
	if current, ok := f.submissions[key]; ok {
		return current.receipt, nil
	}
	for _, current := range f.submissions {
		if current.terminal {
			return factory.MCPWorkStatus{}, store.ErrMCPSubmissionFrozen
		}
	}
	f.submissions[key] = fakeCatalogRecord{
		receipt:  receipt,
		terminal: factory.MCPIsTerminal(submission.Tool),
	}
	return receipt, nil
}

func (f *fakeCatalogSink) RecordMCPSubmission(_ context.Context, submission factory.MCPSubmission) (factory.MCPWorkStatus, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.record(submission, factory.MCPWorkStatus{SubmissionID: submission.SubmissionID, Tool: submission.Tool, State: factory.MCPWorkRecorded})
}

func (f *fakeCatalogSink) RecordWorkerPost(_ context.Context, submission factory.MCPSubmission) (store.FeedPostDetail, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	id := factory.MCPInputDigest([]byte(submission.AssignmentID + "\n" + submission.SubmissionID))[:32]
	if _, exists := f.posts[id]; !exists {
		f.posts[id] = submission
	}
	return store.FeedPostDetail{Record: factory.PostRecord{ID: id}, Assignment: submission.AssignmentID, Actor: string(f.capability.Profile)}, nil
}

func (f *fakeCatalogSink) TerminalMCPProposal(_ context.Context, assignmentID string) (factory.MCPWorkStatus, bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for key, current := range f.submissions {
		if strings.HasPrefix(key, assignmentID+"/") && current.terminal {
			return current.receipt, true, nil
		}
	}
	return factory.MCPWorkStatus{}, false, nil
}

func (f *fakeCatalogSink) MCPSubmissionCount(_ context.Context, assignmentID string) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	count := 0
	for key := range f.submissions {
		if strings.HasPrefix(key, assignmentID+"/") {
			count++
		}
	}
	return count, nil
}

func catalogRefused(res *mcp.CallToolResult, err error) bool {
	return err != nil || (res != nil && res.IsError)
}

func catalogText(t *testing.T, res *mcp.CallToolResult) string {
	t.Helper()
	if len(res.Content) != 1 {
		t.Fatalf("expected one content frame, got %d", len(res.Content))
	}
	text, ok := res.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("expected text content, got %T", res.Content[0])
	}
	return text.Text
}

func storeSelectionReceipt(t *testing.T, sink *fakeCatalogSink, assignment factory.Assignment, attemptEligible, lineageEligible bool, selected string) {
	t.Helper()
	request, err := control.ContextBundleRequest(sink.capability, assignment, attemptEligible, lineageEligible)
	if err != nil {
		t.Fatalf("selection fingerprint: %v", err)
	}
	fingerprint := request.Snapshot
	sink.capability.ContextSelectionSnapshot = fingerprint
	options := []string{factory.ContextBundleCurrent, factory.ContextBundleSnapshotHold}
	if attemptEligible {
		options = append(options, factory.ContextBundleAttempt)
	}
	if lineageEligible {
		options = append(options, factory.ContextBundleLineage)
	}
	answer := factory.PriorityAnswer{Selected: selected, Model: "fixture", Confidence: 1, Probabilities: map[string]float64{}}
	for _, option := range options {
		answer.Probabilities[option] = 0
	}
	answer.Probabilities[selected] = 1
	sink.receipts[fingerprint] = factory.PriorityReceipt{Snapshot: fingerprint, Purpose: factory.JudgmentContextBundle, Outcome: "context-bundle", Selected: selected, Calls: []factory.PriorityCall{{Request: request, Answer: answer}}}
}

// Helpers retained for the kept integration tests after unit-test removal.
package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	extensions "forgejo.org/extension-sdk"

	"github.com/levitateos/sodaos/internal/factory"
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

type workerMCPTransport struct {
	id, secret string
}

func (t workerMCPTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	r = r.Clone(r.Context())
	r.Header.Set("X-Soda-Factory-Capability", t.id)
	r.Header.Set("Authorization", "Bearer "+t.secret)
	return http.DefaultTransport.RoundTrip(r)
}

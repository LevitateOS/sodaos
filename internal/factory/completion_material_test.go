package factory

import (
	"fmt"
	"strings"
	"testing"
)

// FW32 material cutover: pure domain checks for CompletionMaterial,
// CompletionLocal and the safe projections. No storage, no provider.
func materialTestDigest(seed string) string { return strings.Repeat(seed, 64) }

func materialTestAcceptance(id, issue string) Acceptance {
	return Acceptance{
		ID: id, Repository: 7, IssueIndex: issue, Approver: 5, NativeRev: 41,
		TitleDigest: materialTestDigest("a"), TitleOccurrence: OriginalOccurrence,
		ContentDigest: materialTestDigest("b"), ContentVersion: 2,
		Sources: []SelectedSource{{ID: "11", ContentVersion: 0, Digest: materialTestDigest("c")}},
	}
}

func materialTestControl(issue int64, acceptance string) IssueControl {
	return IssueControl{
		Repository: issue, Issue: issue,
		Readiness: ReadinessQueued, Reason: ReasonEligible,
		Fingerprint: materialTestDigest("1"), Authority: materialTestDigest("2"),
		Acceptance: acceptance,
	}
}

func materialTestMaterial() CompletionMaterial {
	acceptance := materialTestAcceptance("d"+strings.Repeat("0", 24), "3")
	return CompletionMaterial{
		Intents:      []CompletionIntentScope{},
		Acceptances:  []Acceptance{acceptance},
		Withdrawals:  []CompletionWithdrawal{},
		Controls:     []IssueControl{materialTestControl(7, "")},
		Preparations: []CompletionPreparation{},
		Plans:        []InvestigationPlan{},
		PlanHeads:    []CompletionPlanHead{},
		PlanTasks:    []CompletionPlanTask{},
		Assignments:  []CompletionAssignment{},
		Publications: []CompletionPublication{},
		Merges:       []CompletionMerge{},
		Checks:       []CompletionCheck{},
		Authority: CompletionAuthority{
			Effective: EffectiveAuthority{
				Missing:      []string{},
				Effective:    true,
				DispatchOpen: true,
			},
			Roles:               []string{"soda-coder", "soda-reviewer"},
			ReassessmentMinutes: DefaultReassessmentMinutes,
		},
		Repository: 7,
	}
}

func materialTestLocal(t *testing.T) CompletionLocal {
	t.Helper()
	material := materialTestMaterial()
	digest, err := material.Digest()
	if err != nil {
		t.Fatal(err)
	}
	return CompletionLocal{
		Material:             &material,
		SourceCommit:         strings.Repeat("c", 40),
		PreparationDigest:    materialTestDigest("d"),
		ScopeDigest:          digest,
		AuthorityFingerprint: "authority",
	}
}

// Empty sections are explicit and valid; the digest is stable across
// encodings and moves with material content only.
func TestCompletionMaterialEmptyValid(t *testing.T) {
	material := materialTestMaterial()
	if err := material.Validate(); err != nil {
		t.Fatalf("empty material refused: %v", err)
	}
	first, err := material.Digest()
	if err != nil {
		t.Fatal(err)
	}
	second, err := material.Digest()
	if err != nil || first != second || !ValidDigest(first) {
		t.Fatalf("digest unstable: %q %v", second, err)
	}
}

// Nil sections are missing data and refuse; they are never confused
// with explicit empty arrays.
func TestCompletionMaterialNilRefuses(t *testing.T) {
	material := materialTestMaterial()
	material.Controls = nil
	if err := material.Validate(); err == nil {
		t.Fatal("nil controls accepted")
	}
	material = materialTestMaterial()
	material.Intents = nil
	if err := material.Validate(); err == nil {
		t.Fatal("nil intents accepted")
	}
}

// Withdrawals without their acceptance body refuse, and unordered
// sections refuse.
func TestCompletionMaterialReferencesAndOrder(t *testing.T) {
	material := materialTestMaterial()
	material.Withdrawals = []CompletionWithdrawal{{Issue: 3, AcceptanceID: "d" + strings.Repeat("9", 24)}}
	if err := material.Validate(); err == nil {
		t.Fatal("withdrawal without its body accepted")
	}
	material = materialTestMaterial()
	material.Withdrawals = []CompletionWithdrawal{}
	second := materialTestAcceptance("d"+strings.Repeat("1", 24), "4")
	material.Acceptances = append(material.Acceptances, second)
	if err := material.Validate(); err != nil {
		t.Fatalf("ordered acceptances refused: %v", err)
	}
	material.Acceptances[0], material.Acceptances[1] = material.Acceptances[1], material.Acceptances[0]
	if err := material.Validate(); err == nil {
		t.Fatal("unordered acceptances accepted")
	}
}

// Per-section overflow and the combined byte bound refuse whole,
// never a truncated prefix.
func TestCompletionMaterialBounds(t *testing.T) {
	material := materialTestMaterial()
	for i := 0; i <= MaxReassessmentIssues; i++ {
		material.Withdrawals = append(material.Withdrawals, CompletionWithdrawal{})
	}
	if err := material.Validate(); err == nil {
		t.Fatal("oversize section accepted")
	}
	sized := materialTestMaterial()
	for i := 0; i < MaxReassessmentIssues; i++ {
		sized.Acceptances = append(sized.Acceptances, materialTestAcceptance(fmt.Sprintf("d%024x", i+1), "3"))
	}
	if err := sized.Validate(); err == nil {
		t.Fatal("oversize encoding accepted")
	}
}

// The scope projection exports only effective approved ideas and
// authorized tasks: unaccepted ideas and private successor bodies stay
// out, while the predecessor chain finds the latest accepted revision.
func TestNewCompletionIntentScopeFilters(t *testing.T) {
	idea := func(id, decision string) ProductIdea {
		return ProductIdea{
			ID: id, Description: "idea " + id, Consequence: "consequence",
			Decision: decision, Source: "supported", Provenance: "autonomous", Evidence: "evidence",
		}
	}
	revision := IntentRevision{
		ID: "d" + strings.Repeat("a", 24), Intent: "d" + strings.Repeat("b", 24),
		Owner: 7, Repository: 7, Objective: "objective",
		Ideas: []ProductIdea{
			idea("d"+strings.Repeat("c", 24), "accept"),
			idea("d"+strings.Repeat("d", 24), "pending"),
		},
		Tasks: []IntentTask{{
			Key: "d" + strings.Repeat("e", 24), Objective: "task",
			Outcomes: []string{"done"}, IdeaIDs: []string{"d" + strings.Repeat("c", 24)},
		}},
		State:             "accepted",
		MaxIssues:         1,
		EffectiveOutcomes: []string{"d" + strings.Repeat("c", 24)},
		CreatedUnix:       1,
	}
	scope, err := NewCompletionIntentScope(revision, "draft", false, nil, []IntentTaskBinding{})
	if err != nil {
		t.Fatalf("scope refused: %v", err)
	}
	if len(scope.Ideas) != 1 || len(scope.Tasks) != 1 {
		t.Fatalf("scope=%+v, want only the effective idea and its task", scope)
	}
	if scope.CurrentState != "draft" || scope.LatestAccepted {
		t.Fatalf("scope=%+v, want draft head metadata without latest", scope)
	}
	if scope.RevisionDigest != revision.Digest() {
		t.Fatal("scope digest differs from its revision")
	}
	draft := revision
	draft.ID = "d" + strings.Repeat("f", 24)
	draft.Predecessor = revision.ID
	draft.State = "draft"
	if err := draft.Validate(); err != nil {
		t.Fatalf("draft revision invalid: %v", err)
	}
	if _, err := NewCompletionIntentScope(draft, "draft", false, nil, []IntentTaskBinding{}); err == nil {
		t.Fatal("private successor revision exported as scope")
	}
}

// Submitted generation tasks are authorized against the current
// accepted/effective ideas before export: derived keys/objectives are
// allowed, but unaccepted, unknown, or pending ideas, digest mismatch,
// and malformed graphs refuse the WHOLE capture. Persisted/direct
// DTOs face the same idea/graph bar before funding or use.
func TestNewCompletionIntentScopeAuthorizesSubmitted(t *testing.T) {
	idea := func(id, decision string) ProductIdea {
		return ProductIdea{
			ID: id, Description: "idea " + id, Consequence: "consequence",
			Decision: decision, Source: "supported", Provenance: "autonomous", Evidence: "evidence",
		}
	}
	acceptedID := "d" + strings.Repeat("c", 24)
	pendingID := "d" + strings.Repeat("d", 24)
	taskKey := "d" + strings.Repeat("e", 24)
	revision := IntentRevision{
		ID: "d" + strings.Repeat("a", 24), Intent: "d" + strings.Repeat("b", 24),
		Owner: 7, Repository: 7, Objective: "objective",
		Ideas: []ProductIdea{
			idea(acceptedID, "accept"),
			idea(pendingID, "pending"),
		},
		Tasks: []IntentTask{{
			Key: taskKey, Objective: "task",
			Outcomes: []string{"done"}, IdeaIDs: []string{acceptedID},
		}},
		State: "accepted", MaxIssues: 4,
		EffectiveOutcomes: []string{acceptedID},
		CreatedUnix:       1,
	}
	task := func(key string, ideas []string, deps []string) IntentTask {
		return IntentTask{
			Key: key, Objective: "objective " + key,
			Outcomes: []string{"done"}, IdeaIDs: ideas, DependsOn: deps,
		}
	}
	gen := func(tasks []IntentTask, digest string) *IntentGeneration {
		return &IntentGeneration{
			RevisionID: revision.ID, IntentID: revision.Intent,
			OperationID: "op-1", State: "complete",
			IntentDigest: digest, SubmittedTasks: tasks,
		}
	}
	derivedKey := "d" + strings.Repeat("f", 24)
	secondKey := "d" + strings.Repeat("0", 24)
	derived := []IntentTask{
		task(derivedKey, []string{acceptedID}, nil),
		task(secondKey, []string{acceptedID}, []string{derivedKey}),
	}
	scope, err := NewCompletionIntentScope(revision, "accepted", true,
		gen(derived, revision.Digest()), nil)
	if err != nil {
		t.Fatalf("derived submitted graph refused: %v", err)
	}
	if len(scope.SubmittedTasks) != 2 || scope.SubmittedTasks[1].Key != secondKey {
		t.Fatalf("submitted=%+v, want the derived graph exported", scope.SubmittedTasks)
	}
	unknownID := "d" + strings.Repeat("9", 24)
	for name, tasks := range map[string][]IntentTask{
		"pending idea":  {task("d"+strings.Repeat("1", 24), []string{pendingID}, nil)},
		"unknown idea":  {task("d"+strings.Repeat("2", 24), []string{unknownID}, nil)},
		"unknown dep":   {task("d"+strings.Repeat("3", 24), []string{acceptedID}, []string{unknownID})},
		"cyclic graph":  {task("d"+strings.Repeat("4", 24), []string{acceptedID}, []string{"d" + strings.Repeat("5", 24)}), task("d" + strings.Repeat("5", 24), []string{acceptedID}, []string{"d" + strings.Repeat("4", 24)})},
		"duplicate key": {task(taskKey, []string{acceptedID}, nil), task(taskKey, []string{acceptedID}, nil)},
	} {
		if _, err := NewCompletionIntentScope(revision, "accepted", true, gen(tasks, revision.Digest()), nil); err == nil {
			t.Fatalf("%s submitted graph accepted", name)
		}
	}
	if _, err := NewCompletionIntentScope(revision, "accepted", true,
		gen([]IntentTask{task(derivedKey, []string{acceptedID}, nil)}, strings.Repeat("0", 64)), nil); err == nil {
		t.Fatal("digest-mismatched generation accepted")
	}
	dto := scope
	dto.SubmittedTasks = []IntentTask{task("d"+strings.Repeat("6", 24), []string{pendingID}, nil)}
	if err := dto.Validate(); err == nil {
		t.Fatal("DTO submitted task with pending idea accepted")
	}
	dto.SubmittedTasks = []IntentTask{
		task("d"+strings.Repeat("7", 24), []string{acceptedID}, []string{"d" + strings.Repeat("8", 24)}),
		task("d"+strings.Repeat("8", 24), []string{acceptedID}, []string{"d" + strings.Repeat("7", 24)}),
	}
	if err := dto.Validate(); err == nil {
		t.Fatal("cyclic DTO submitted graph accepted")
	}
}

// The check projection validates its source before omitting observation
// metadata.
func TestNewCompletionCheckOmitsObservation(t *testing.T) {
	assessment := CheckAssessment{
		Results:    []CheckResult{{Context: "ci", State: CheckStateSuccess, Passed: true}},
		Checks:     []string{"ci"},
		Repository: 7, PRNumber: 9, PRID: 10, IssueID: 11,
		PolicyRevision: 1, NativeRev: 41, Revision: 2,
		AssessedUnix:     1720000000,
		ObservedContexts: 1,
		HeadRef:          "refs/heads/feature", BaseRef: "refs/heads/main",
		HeadOID: strings.Repeat("a", 40), BaseOID: strings.Repeat("b", 40),
		ChecksDigest: ChecksDigest([]string{"ci"}),
		Verdict:      CheckPass, Reason: CheckReasonPass,
	}
	projected, err := NewCompletionCheck(assessment)
	if err != nil {
		t.Fatalf("check refused: %v", err)
	}
	if err := projected.Validate(); err != nil {
		t.Fatalf("projected check refused: %v", err)
	}
	bad := assessment
	bad.Results[0].Passed = false
	if _, err := NewCompletionCheck(bad); err == nil {
		t.Fatal("incoherent assessment projected")
	}
}

// Pending and declined ideas never enter scope, even when named; the
// authorized-choice rule holds with and without a recorded effective
// subset.
func TestCompletionScopeRejectsPendingIdeas(t *testing.T) {
	pending := "d" + strings.Repeat("d", 24)
	revision := IntentRevision{
		ID: "d" + strings.Repeat("a", 24), Intent: "d" + strings.Repeat("b", 24),
		Owner: 7, Repository: 7, Objective: "objective",
		Ideas: []ProductIdea{{
			ID: "d" + strings.Repeat("c", 24), Description: "effective choice",
			Consequence: "consequence", Decision: "accept",
			Source: "supported", Provenance: "autonomous", Evidence: "evidence",
		}, {
			ID: pending, Description: "pending choice",
			Consequence: "consequence", Decision: "pending",
		}},
		Tasks: []IntentTask{{
			Key: "d" + strings.Repeat("e", 24), Objective: "task",
			Outcomes: []string{"done"}, IdeaIDs: []string{"d" + strings.Repeat("c", 24)},
		}},
		State: "accepted", MaxIssues: 2,
		EffectiveOutcomes: []string{"d" + strings.Repeat("c", 24)},
		CreatedUnix:       1,
	}
	scope, err := NewCompletionIntentScope(revision, "accepted", true, nil, []IntentTaskBinding{})
	if err != nil {
		t.Fatalf("scope refused: %v", err)
	}
	if len(scope.Ideas) != 1 {
		t.Fatalf("ideas=%d, want only the authorized choice", len(scope.Ideas))
	}
	tainted := scope
	tainted.Ideas = append(append([]ProductIdea(nil), scope.Ideas...), ProductIdea{
		ID: pending, Description: "pending choice",
		Consequence: "consequence", Decision: "pending",
	})
	if err := tainted.Validate(); err == nil {
		t.Fatal("pending idea validated as scope")
	}
}

// Permitted roles come from the active sponsorship restricted to
// configured valid roles: withdrawn or missing sponsorship permits
// nothing, and explicit empty is representable without authority.
func TestPermittedCompletionRoles(t *testing.T) {
	configured := map[string]RoleSelection{
		"soda-coder": {}, "soda-reviewer": {},
	}
	active := &Sponsorship{Active: true, Roles: []string{"soda-reviewer", "soda-coder", "soda-coder", "other-role"}}
	if roles := PermittedCompletionRoles(configured, active); len(roles) != 2 || roles[0] != "soda-coder" || roles[1] != "soda-reviewer" {
		t.Fatalf("roles=%v, want the sorted configured subset", roles)
	}
	if roles := PermittedCompletionRoles(configured, &Sponsorship{Active: false, Roles: []string{"soda-coder"}}); roles == nil || len(roles) != 0 {
		t.Fatalf("roles=%v, want explicit empty for withdrawn sponsorship", roles)
	}
	if roles := PermittedCompletionRoles(configured, nil); roles == nil || len(roles) != 0 {
		t.Fatalf("roles=%v, want explicit empty for missing sponsorship", roles)
	}
	empty := materialTestMaterial()
	empty.Authority.Roles = []string{}
	if err := empty.Validate(); err != nil {
		t.Fatalf("explicit empty roles refused: %v", err)
	}
	nilRoles := materialTestMaterial()
	nilRoles.Authority.Roles = nil
	if err := nilRoles.Validate(); err == nil {
		t.Fatal("missing role projection accepted")
	}
}

// CompletionLocal requires complete material with a matching digest;
// all-zero stays unsealed and Equal compares canonical content.
func TestCompletionLocalMaterialBinding(t *testing.T) {
	var zero CompletionLocal
	if zero.Sealed() {
		t.Fatal("all-zero local seals")
	}
	if err := zero.Validate(); err == nil {
		t.Fatal("nil material validated")
	}
	local := materialTestLocal(t)
	if err := local.Validate(); err != nil {
		t.Fatalf("complete local refused: %v", err)
	}
	tampered := local
	tampered.ScopeDigest = materialTestDigest("0")
	if err := tampered.Validate(); err == nil {
		t.Fatal("tampered digest validated")
	}
	missing := local
	missing.Material = nil
	if err := missing.Validate(); err == nil {
		t.Fatal("nil material validated")
	}
	clone := local
	other := *local.Material
	clone.Material = &other
	if !local.Equal(clone) {
		t.Fatal("equal content compares unequal")
	}
	moved := clone
	moved.SourceCommit = strings.Repeat("e", 40)
	if local.Equal(moved) {
		t.Fatal("moved source compares equal")
	}
	digest, err := local.Digest()
	if err != nil || !ValidDigest(digest) {
		t.Fatalf("local digest: %q %v", digest, err)
	}
	if again, err := local.Digest(); err != nil || again != digest {
		t.Fatalf("local digest unstable: %q %v", again, err)
	}
}

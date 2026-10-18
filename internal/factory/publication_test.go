package factory

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"
)

func testPublicationRun() Run {
	return Run{
		ID: "0123456789abcdef0123456789abcdef", ProjectID: "p" + strings.Repeat("d", 24),
		Role: "soda-coder", InputSHA: strings.Repeat("1", 40),
		Started: time.Unix(1000, 0), Deadline: time.Unix(2000, 0),
		Image: "sha256:" + strings.Repeat("b", 64), Harness: "codex", Model: "test",
	}
}

func testPublication() Publication {
	return Publication{
		Publish:      PublicationOperation{Kind: OpRefPublish},
		PRCreate:     PublicationOperation{Kind: OpPRCreate},
		ID:           "abcdef0123456789abcdef0123456789",
		AssignmentID: "0123456789abcdef0123456789abcdef",
		ProjectID:    "p" + strings.Repeat("d", 24),
		Role:         "soda-coder",
		Acceptance:   "d" + strings.Repeat("a", 24),
		Preparation:  "f" + strings.Repeat("b", 24),
		Run:          "0123456789abcdef0123456789abcdef",
		Candidate:    strings.Repeat("2", 40),
		BaseSHA:      strings.Repeat("1", 40),
		TargetBranch: "refs/heads/main",
		Stage:        PublicationOpen,
		Repository:   7,
		Issue:        3,
		CreatedUnix:  1100,
	}
}

func TestPublicationValidatesOpen(t *testing.T) {
	if err := testPublication().Validate(); err != nil {
		t.Fatalf("open publication refused: %v", err)
	}
}

func TestPublicationRejectsMalformed(t *testing.T) {
	cases := map[string]func(*Publication){
		"identity":    func(p *Publication) { p.ID = "short" },
		"assignment":  func(p *Publication) { p.AssignmentID = "" },
		"project":     func(p *Publication) { p.ProjectID = "nope" },
		"role":        func(p *Publication) { p.Role = "coder" },
		"scope":       func(p *Publication) { p.Issue = 0 },
		"acceptance":  func(p *Publication) { p.Acceptance = "nope" },
		"preparation": func(p *Publication) { p.Preparation = "nope" },
		"run":         func(p *Publication) { p.Run = "short" },
		"candidate":   func(p *Publication) { p.Candidate = "xyz" },
		"same":        func(p *Publication) { p.Candidate = p.BaseSHA },
		"target":      func(p *Publication) { p.TargetBranch = "main" },
		"comparison":  func(p *Publication) { p.Comparison = "xyz" },
		"tip":         func(p *Publication) { p.TargetTip = "xyz" },
		"observation": func(p *Publication) { p.NativeRev = 4 },
		"created":     func(p *Publication) { p.CreatedUnix = 0 },
		"publish":     func(p *Publication) { p.Publish.Kind = OpPRCreate },
		"prcreate":    func(p *Publication) { p.PRCreate.Kind = OpRefPublish },
		"stage":       func(p *Publication) { p.Stage = "publishing" },
		"open":        func(p *Publication) { p.Outcome = Succeeded },
		"openlink":    func(p *Publication) { p.PRNumber, p.PRID = 1, 1 },
	}
	for name, mutate := range cases {
		p := testPublication()
		mutate(&p)
		if err := p.Validate(); err == nil {
			t.Errorf("case %s accepted", name)
		}
	}
}

func testPublicationIntent(operationID string) *PublicationIntent {
	return &PublicationIntent{
		OperationID: operationID, AuthRevision: "accepted-revision",
		Candidate: strings.Repeat("2", 40), TargetBranch: "refs/heads/main",
		ExpectedOld: "absent", ComparisonRef: "refs/heads/main", ComparisonOID: strings.Repeat("1", 40),
		PRTitle: "Factory candidate for #3", Repository: 7, ActorID: 5, NativeRev: 9, NotAfter: 1300,
	}
}

func publishedPublication() Publication {
	p := testPublication()
	p.Stage, p.Outcome, p.Reason, p.FinishedUnix = PublicationPublished, Succeeded, PublishReasonLinked, 1200
	p.PRNumber, p.PRID = 9, 8
	p.Publish = PublicationOperation{
		Work:        testPublicationIntent("soda-abcdef-publish-1"),
		OperationID: "soda-abcdef-publish-1", Kind: OpRefPublish, Effect: OpEffectCommitted,
		Cancellation: OpCancelNone, Completion: OpCompletionComplete,
		Attempts: 1, UpdatedUnix: 1150,
	}
	p.PRCreate = PublicationOperation{
		Work:        testPublicationIntent("soda-abcdef-prcreate-1"),
		OperationID: "soda-abcdef-prcreate-1", Kind: OpPRCreate, Effect: OpEffectCommitted,
		Cancellation: OpCancelNone, Completion: OpCompletionComplete,
		HeadRef: "refs/heads/soda/factory/0123456789abcdef0123456789abcdef", BaseRef: "refs/heads/main",
		HeadOID: strings.Repeat("2", 40), BaseOID: strings.Repeat("1", 40),
		Attempts: 1, PRNumber: 9, PRID: 8, IssueID: 10, UpdatedUnix: 1180,
	}
	return p
}

func TestPublicationValidatesPublished(t *testing.T) {
	if err := publishedPublication().Validate(); err != nil {
		t.Fatalf("published publication refused: %v", err)
	}
}

func TestPublicationCorrectionRequiresRecordedRunAndMovesPublicationHead(t *testing.T) {
	makeCorrection := func(runID string) PublicationOperation {
		p := publishedPublication()
		id := PublicationOperationID(p.ID, OpRefPublish, 2)
		work := testPublicationIntent(id)
		work.ExpectedOld = p.Candidate
		work.Candidate = strings.Repeat("d", 40)
		work.CorrectionNumber, work.CorrectionAuthor = p.PRNumber, p.PRCreate.Work.ActorID
		return PublicationOperation{
			Work: work, RunID: runID, OperationID: id, Kind: OpRefPublish,
			Effect: OpEffectCommitted, Completion: OpCompletionComplete,
			Attempts: 1, UpdatedUnix: 1250,
		}
	}

	for name, runID := range map[string]string{"missing": "", "invalid": "not-a-run"} {
		t.Run(name, func(t *testing.T) {
			p := publishedPublication()
			p.Corrections = CorrectionOps{makeCorrection(runID)}
			p.Candidate = strings.Repeat("d", 40)
			if err := p.Validate(); err == nil {
				t.Fatalf("correction with %s RunID accepted", name)
			}
		})
	}

	t.Run("committed correction run becomes publication run", func(t *testing.T) {
		p := publishedPublication()
		correctionRun := "fedcba9876543210fedcba9876543210"
		p.Corrections = CorrectionOps{makeCorrection(correctionRun)}
		p.Candidate = strings.Repeat("d", 40)
		if err := p.Validate(); err == nil {
			t.Fatal("publication retaining the initial run after a committed correction was accepted")
		}
		p.Run = correctionRun
		if err := p.Validate(); err != nil {
			t.Fatalf("publication following committed correction run refused: %v", err)
		}
	})
}

func TestPublicationBoundsReviewHistoryAdmission(t *testing.T) {
	makeReviews := func(count, bodyBytes int) Publication {
		p := publishedPublication()
		for i := 0; i < count; i++ {
			runID := fmt.Sprintf("%032x", i+1)
			p.ReviewOperations = append(p.ReviewOperations, ReviewOperation{
				RunID: runID,
				Work: ReviewWork{
					OperationID: "review-" + runID, AuthRevision: ReviewAuthRevision(p.AssignmentID, runID),
					Repository: p.Repository, ActorID: 6, PRNumber: p.PRNumber, PRID: p.PRID,
					IssueID: p.PRCreate.IssueID, PRAuthorID: p.PRCreate.Work.ActorID,
					HeadRef: p.PRCreate.HeadRef, BaseRef: p.PRCreate.BaseRef,
					HeadOID: p.Publish.Work.Candidate, BaseOID: p.PRCreate.BaseOID,
					NativeRev: 1, NotAfter: 1300, Event: "APPROVED", Body: strings.Repeat("x", bodyBytes),
				},
			})
		}
		return p
	}
	if err := makeReviews(MaxPublicationReviewOperations, 0).Validate(); err != nil {
		t.Fatalf("history at record bound refused: %v", err)
	}
	if err := makeReviews(MaxPublicationReviewOperations+1, 0).Validate(); err == nil {
		t.Fatal("history above record bound accepted")
	}
	// Pending work reserves the largest permitted review-specific receipt
	// before native submission. Even when every operation later receives a
	// maximum-sized receipt, that already-admitted publication remains
	// persistable.
	perOperationAtBound := MaxPublicationReviewBytes / MaxPublicationReviewOperations
	maxPending := makeReviews(MaxPublicationReviewOperations, perOperationAtBound-MaxReviewReceiptBytes)
	if err := maxPending.Validate(); err != nil {
		t.Fatalf("pending history with maximum receipt reservations refused: %v", err)
	}
	for i := range maxPending.ReviewOperations {
		op := &maxPending.ReviewOperations[i]
		op.Outcome = OperationOutcome{
			OperationID: op.Work.OperationID, InstallationID: "install",
			Kind: OpReviewSubmit, ActorID: op.Work.ActorID,
			RepositoryID: maxPending.Repository, Receipt: bytes.Repeat([]byte{0xff}, MaxReviewReceiptBytes),
			Effect: OpEffectCommitted, Cancellation: OpCancelNone, Completion: OpCompletionComplete,
		}
	}
	if err := maxPending.Validate(); err != nil {
		t.Fatalf("maximum receipts made admitted history unpersistable: %v", err)
	}

	// A 63-operation pending history leaves 16384 bytes. One more body byte
	// than that permits, including its reserved receipt, must be rejected by
	// the same pre-submit admission used by the coordinator.
	nearBound := makeReviews(MaxPublicationReviewOperations-1, perOperationAtBound-MaxReviewReceiptBytes)
	nextID := fmt.Sprintf("%032x", MaxPublicationReviewOperations)
	next := ReviewOperation{RunID: nextID, Work: ReviewWork{
		OperationID: "review-" + nextID, AuthRevision: ReviewAuthRevision(nearBound.AssignmentID, nextID),
		Repository: nearBound.Repository, ActorID: 6, PRNumber: nearBound.PRNumber, PRID: nearBound.PRID,
		IssueID: nearBound.PRCreate.IssueID, PRAuthorID: nearBound.PRCreate.Work.ActorID,
		HeadRef: nearBound.PRCreate.HeadRef, BaseRef: nearBound.PRCreate.BaseRef,
		HeadOID: nearBound.Publish.Work.Candidate, BaseOID: nearBound.PRCreate.BaseOID,
		NativeRev: 1, NotAfter: 1300, Event: "APPROVED",
		Body: strings.Repeat("x", perOperationAtBound-MaxReviewReceiptBytes+1),
	}}
	if err := nearBound.CanAppendReviewOperation(next); err == nil {
		t.Fatal("review admission accepted body plus reserved receipt one byte over aggregate bound")
	}
	withReceipt := makeReviews(1, 0)
	withReceipt.ReviewOperations[0].Outcome = OperationOutcome{
		OperationID: withReceipt.ReviewOperations[0].Work.OperationID, InstallationID: "install",
		Kind: OpReviewSubmit, ActorID: withReceipt.ReviewOperations[0].Work.ActorID,
		RepositoryID: withReceipt.Repository, Receipt: []byte{0, 0xff, 0x80, 1},
		Effect: OpEffectCommitted, Cancellation: OpCancelNone, Completion: OpCompletionComplete,
	}
	serialized, err := json.Marshal(withReceipt)
	if err != nil {
		t.Fatal(err)
	}
	var restored Publication
	if err := json.Unmarshal(serialized, &restored); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(restored.ReviewOperations[0].Outcome.Receipt, withReceipt.ReviewOperations[0].Outcome.Receipt) {
		t.Fatal("review receipt bytes changed in publication storage encoding")
	}
}

func TestPublicationRejectsLookalikeLinks(t *testing.T) {
	p := publishedPublication()
	p.PRNumber = 10
	if err := p.Validate(); err == nil {
		t.Fatal("published PR differing from the recorded creation accepted")
	}
	p = publishedPublication()
	p.Publish.Effect = OpEffectNotCommitted
	if err := p.Validate(); err == nil {
		t.Fatal("published publication without a committed branch accepted")
	}
	p = publishedPublication()
	p.PRCreate.Effect = OpEffectPending
	p.PRCreate.PRNumber, p.PRCreate.PRID, p.PRCreate.IssueID = 0, 0, 0
	if err := p.Validate(); err == nil {
		t.Fatal("published publication with a pending creation accepted")
	}
}

func TestPublicationValidatesTerminalStages(t *testing.T) {
	for _, stage := range []struct {
		stage   string
		outcome Outcome
		reason  string
	}{
		{PublicationFailed, Failed, PublishReasonRefused},
		{PublicationWithdrawn, Cancelled, PublishReasonWithdrawn},
		{PublicationFenced, NeedsHuman, PublishReasonFenced},
	} {
		p := testPublication()
		p.Stage, p.Outcome, p.Reason, p.FinishedUnix = stage.stage, stage.outcome, stage.reason, 1200
		p.Publish = PublicationOperation{
			Work:        testPublicationIntent("soda-abcdef-publish-1"),
			OperationID: "soda-abcdef-publish-1", Kind: OpRefPublish,
			Effect: OpEffectCommitted, Attempts: 1, UpdatedUnix: 1150,
		}
		if err := p.Validate(); err != nil {
			t.Errorf("stage %s refused: %v", stage.stage, err)
		}
		p.Reason = PublishReasonLinked
		p.PRNumber, p.PRID = 1, 1
		if err := p.Validate(); err == nil {
			t.Errorf("stage %s with PR links accepted", stage.stage)
		}
	}
}

func TestPublicationOperationRejectsMalformed(t *testing.T) {
	cases := map[string]PublicationOperation{
		"kind":     {Kind: OpReviewSubmit},
		"attempts": {Kind: OpRefPublish, Attempts: 4},
		"unsubmitted": {
			Kind: OpRefPublish, Effect: OpEffectPending,
		},
		"identity": {
			Kind: OpRefPublish, Attempts: 1, Effect: OpEffectPending, UpdatedUnix: 1,
		},
		"effect": {
			Kind: OpRefPublish, OperationID: "soda-a-publish-1", Attempts: 1,
			Effect: "maybe", UpdatedUnix: 1,
		},
		"cancellation": {
			Kind: OpRefPublish, OperationID: "soda-a-publish-1", Attempts: 1,
			Effect: OpEffectPending, Cancellation: "maybe", UpdatedUnix: 1,
		},
		"completion": {
			Kind: OpRefPublish, OperationID: "soda-a-publish-1", Attempts: 1,
			Effect: OpEffectPending, Completion: "done", UpdatedUnix: 1,
		},
		"reason": {
			Kind: OpRefPublish, OperationID: "soda-a-publish-1", Attempts: 1,
			Effect: OpEffectPending, Reason: strings.Repeat("r", 65), UpdatedUnix: 1,
		},
		"branchlinks": {
			Kind: OpRefPublish, OperationID: "soda-a-publish-1", Attempts: 1,
			Effect: OpEffectCommitted, PRNumber: 1, PRID: 1, IssueID: 1, UpdatedUnix: 1,
		},
		"partiallinks": {
			Kind: OpPRCreate, OperationID: "soda-a-prcreate-1", Attempts: 1,
			Effect: OpEffectCommitted, PRNumber: 1, UpdatedUnix: 1,
		},
		"uncommittedlinks": {
			Kind: OpPRCreate, OperationID: "soda-a-prcreate-1", Attempts: 1,
			Effect: OpEffectPending, PRNumber: 1, PRID: 1, IssueID: 1, UpdatedUnix: 1,
		},
		"snapshot": {
			Kind: OpPRCreate, OperationID: "soda-a-prcreate-1", Attempts: 1,
			Effect: OpEffectCommitted, PRNumber: 1, PRID: 1, IssueID: 1, UpdatedUnix: 1,
		},
	}
	for name, op := range cases {
		if err := op.Validate(); err == nil {
			t.Errorf("case %s accepted", name)
		}
	}
}

func TestPublicationOperationIDShape(t *testing.T) {
	id := PublicationOperationID("abcdef0123456789abcdef0123456789", OpRefPublish, 1)
	if !ValidPublicationOperationID(id) {
		t.Fatalf("derived operation id refused: %s", id)
	}
	other := PublicationOperationID("abcdef0123456789abcdef0123456789", OpPRCreate, 1)
	if id == other {
		t.Fatal("publish and PR-create identities collide")
	}
	if PublicationOperationID("abcdef0123456789abcdef0123456789", OpRefPublish, 1) ==
		PublicationOperationID("abcdef0123456789abcdef0123456789", OpRefPublish, 2) {
		t.Fatal("attempt identities collide")
	}
	for _, bad := range []string{"", "has space", "semi;colon", strings.Repeat("a", 129)} {
		if ValidPublicationOperationID(bad) {
			t.Fatalf("operation id %q accepted", bad)
		}
	}
}

func TestPublicationBranchShape(t *testing.T) {
	ref := PublishBranchName("0123456789abcdef0123456789abcdef")
	if ref != "refs/heads/soda/factory/0123456789abcdef0123456789abcdef" {
		t.Fatalf("branch ref: %s", ref)
	}
	if !ValidTargetBranch(ref) {
		t.Fatalf("branch ref refused: %s", ref)
	}
	if RefHead(ref) != "soda/factory/0123456789abcdef0123456789abcdef" {
		t.Fatalf("branch head: %s", RefHead(ref))
	}
	if RefHead("refs/heads/a..b") != "" || RefHead("main") != "" {
		t.Fatal("malformed ref head accepted")
	}
}

func TestPublicationWorkValidates(t *testing.T) {
	work := PublicationWork{
		Bundle:        []byte("bundle"),
		AssignmentID:  "0123456789abcdef0123456789abcdef",
		Publication:   "abcdef0123456789abcdef0123456789",
		RunID:         "0123456789abcdef0123456789abcdef",
		Run:           testPublicationRun(),
		Candidate:     strings.Repeat("2", 40),
		BaseSHA:       strings.Repeat("3", 40),
		TargetBranch:  "refs/heads/main",
		OperationID:   "soda-abcdef-publish-1",
		AuthRevision:  AuthRevisionFor("0123456789abcdef0123456789abcdef", "abcdef0123456789abcdef0123456789", 0),
		ExpectedOld:   "absent",
		ComparisonRef: "refs/heads/main",
		ComparisonOID: strings.Repeat("1", 40),
		PRTitle:       PRTitleFor(3),
		PRBody:        PRBodyFor(3, "0123456789abcdef0123456789abcdef", "d"+strings.Repeat("a", 24), "0123456789abcdef0123456789abcdef", strings.Repeat("2", 40), strings.Repeat("1", 40)),
		Repository:    7,
		Issue:         3,
		ActorID:       5,
		NativeRev:     9,
		NotAfter:      1300,
	}
	if err := work.Validate(); err != nil {
		t.Fatalf("work refused: %v", err)
	}
	if len(work.AuthRevision) > 512 || work.AuthRevision == "" {
		t.Fatalf("auth revision: %q", work.AuthRevision)
	}
	work.Bundle = nil
	if err := work.Validate(); err == nil {
		t.Fatal("empty bundle accepted")
	}
}

func TestPublicationBodyClosesOriginalIssue(t *testing.T) {
	body := PRBodyFor(3, "0123456789abcdef0123456789abcdef", "d"+strings.Repeat("a", 24), "0123456789abcdef0123456789abcdef", strings.Repeat("2", 40), strings.Repeat("1", 40))
	if strings.Count(body, "Closes #3\n") != 1 || !strings.Contains(body, "\nCloses #3\n") {
		t.Fatalf("publication omitted its original issue closure: %q", body)
	}
	for _, issue := range []int64{0, -1} {
		if body := PRBodyFor(issue, "assignment", "acceptance", "run", "candidate", "source"); body != "" {
			t.Fatalf("invalid original issue emitted closure: %q", body)
		}
	}
}

func TestPublicationKeepsCommittedPRWhileCompletionWaits(t *testing.T) {
	p := publishedPublication()
	p.PRCreate.Completion = OpCompletionPending
	if err := p.Validate(); err == nil {
		t.Fatal("incomplete PR marked published")
	}
	p.Stage, p.Outcome, p.Reason, p.FinishedUnix = PublicationOpen, "", "", 0
	if err := p.Validate(); err != nil {
		t.Fatalf("committed partial link refused: %v", err)
	}
	p.PRCreate.Completion = OpCompletionNeedsIntervention
	p.Stage, p.Outcome, p.Reason, p.FinishedUnix = PublicationFenced, NeedsHuman, PublishReasonFenced, 1400
	if err := p.Validate(); err != nil {
		t.Fatalf("intervention hides committed link: %v", err)
	}
	p.Stage, p.Outcome, p.Reason = PublicationWithdrawn, Cancelled, PublishReasonWithdrawn
	if err := p.Validate(); err != nil {
		t.Fatalf("late withdrawal hides committed link: %v", err)
	}
}

func TestPublicationWithdrawalCannotGuessPendingEffect(t *testing.T) {
	p := testPublication()
	p.Publish = PublicationOperation{Work: testPublicationIntent("pending-op"), OperationID: "pending-op", Kind: OpRefPublish, Attempts: 1, UpdatedUnix: 1200, Effect: OpEffectPending}
	p.Stage, p.Outcome, p.Reason, p.FinishedUnix = PublicationWithdrawn, Cancelled, PublishReasonWithdrawn, 1400
	if err := p.Validate(); err == nil {
		t.Fatal("pending operation appeared cancelled")
	}
}

func TestPublicationCandidateTupleBindsAndAdvances(t *testing.T) {
	p := testPublication()
	tuple := CandidateTuple{Head: p.Candidate, Base: p.BaseSHA, DiffBase: p.BaseSHA, Revision: 0}
	if err := tuple.Validate(); err != nil {
		t.Fatalf("candidate tuple refused: %v", err)
	}
	p.Tuple = &tuple
	if err := p.Validate(); err != nil {
		t.Fatalf("publication with its current tuple refused: %v", err)
	}
	moved := tuple
	moved.Head = strings.Repeat("3", 40)
	p.Tuple = &moved
	if err := p.Validate(); err == nil {
		t.Fatal("tuple head outside publication history accepted")
	}
	p.Tuple = &tuple
	next, err := tuple.Advance(0, strings.Repeat("3", 40), p.BaseSHA, p.BaseSHA)
	if err != nil || next.Revision != 1 {
		t.Fatalf("tuple CAS advance refused: %v", err)
	}
	if _, err := tuple.Advance(1, strings.Repeat("3", 40), p.BaseSHA, p.BaseSHA); err == nil {
		t.Fatal("stale tuple revision overwrote a concurrent refresh")
	}
}

func TestPublicationCurrentDiffBasePrefersFreshTuple(t *testing.T) {
	p := testPublication()
	p.PRCreate.BaseOID = strings.Repeat("b", 40)
	if got := p.CurrentDiffBase(); got != p.PRCreate.BaseOID {
		t.Fatalf("unobserved publication lost its PR base: %q", got)
	}
	mergeBase := strings.Repeat("4", 40)
	p.Tuple = &CandidateTuple{Head: p.Candidate, Base: p.PRCreate.BaseOID, DiffBase: mergeBase}
	if got := p.CurrentDiffBase(); got != mergeBase {
		t.Fatalf("fresh tuple did not anchor the diff: %q", got)
	}
	stale := *p.Tuple
	stale.Head = strings.Repeat("3", 40)
	p.Tuple = &stale
	if got := p.CurrentDiffBase(); got != p.PRCreate.BaseOID {
		t.Fatalf("stale tuple anchored the diff: %q", got)
	}
	broken := stale
	broken.Head = p.Candidate
	broken.DiffBase = "not-a-commit"
	p.Tuple = &broken
	if got := p.CurrentDiffBase(); got != p.PRCreate.BaseOID {
		t.Fatalf("unverified diff base anchored the diff: %q", got)
	}
}

func TestReviewReceiptBoundPersistsMaximum(t *testing.T) {
	p := publishedPublication()
	runID := fmt.Sprintf("%032x", 1)
	op := ReviewOperation{RunID: runID, Work: ReviewWork{
		OperationID: "review-" + runID, AuthRevision: ReviewAuthRevision(p.AssignmentID, runID),
		Repository: p.Repository, ActorID: 6, PRNumber: p.PRNumber, PRID: p.PRID,
		IssueID: p.PRCreate.IssueID, PRAuthorID: p.PRCreate.Work.ActorID,
		HeadRef: p.PRCreate.HeadRef, BaseRef: p.PRCreate.BaseRef,
		HeadOID: p.Publish.Work.Candidate, BaseOID: p.PRCreate.BaseOID,
		NativeRev: 1, NotAfter: 1300, Event: "APPROVED",
	}, Outcome: OperationOutcome{
		OperationID: "review-" + runID, InstallationID: "install",
		Kind: OpReviewSubmit, ActorID: 6, RepositoryID: p.Repository,
		Receipt: bytes.Repeat([]byte{0xff}, MaxReviewReceiptBytes),
		Effect:  OpEffectCommitted, Cancellation: OpCancelNone, Completion: OpCompletionComplete,
	}}
	if err := op.ValidateForPublication(p); err != nil {
		t.Fatalf("maximum review receipt refused: %v", err)
	}
	tooLarge := op
	tooLarge.Outcome.Receipt = bytes.Repeat([]byte{0xff}, MaxReviewReceiptBytes+1)
	if err := tooLarge.ValidateForPublication(p); err == nil {
		t.Fatal("review receipt above its bound accepted")
	}
}

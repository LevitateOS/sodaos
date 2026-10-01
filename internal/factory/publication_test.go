package factory

import (
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
		PRBody:        PRBodyFor("0123456789abcdef0123456789abcdef", "d"+strings.Repeat("a", 24), "0123456789abcdef0123456789abcdef", strings.Repeat("2", 40), strings.Repeat("1", 40)),
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

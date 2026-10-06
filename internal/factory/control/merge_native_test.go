package control_test

// Native merge proof (ST12, S-merge). Fixture candidates, PRs, reviews and
// commit statuses are prepared through ordinary native Git/REST writes plus
// the production conditional review client; merges run through the
// production conditional merge client and MergePass. Absence of
// SODA_ST12_NATIVE skips native work and is never passing evidence.

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	extensions "forgejo.org/extension-sdk"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/factory/control"
	"github.com/levitateos/sodaos/internal/forgejo"
	forgejopublish "github.com/levitateos/sodaos/internal/forgejo/publish"
	"github.com/levitateos/sodaos/internal/project"
	"github.com/levitateos/sodaos/internal/store"
)

type nativeST12Config struct {
	nativeST10Config
	MergeBound bool   `json:"merge_bound"`
	FountainDB string `json:"fountain_db"`
}

func loadNativeST12(t *testing.T) nativeST12Config {
	t.Helper()
	raw := os.Getenv("SODA_ST12_NATIVE")
	if raw == "" {
		t.Skip("native proof NOT RUN: SODA_ST12_NATIVE is not configured")
	}
	var cfg nativeST12Config
	nativeMust(t, json.Unmarshal([]byte(raw), &cfg))
	if cfg.FountainURL == "" || cfg.Socket == "" || cfg.TokenFile == "" || cfg.Repository <= 0 || cfg.ActorID <= 0 || cfg.TokenID <= 0 {
		t.Fatal("native fixture requires explicit repository and merger inputs")
	}
	if cfg.CreatorID <= 0 || cfg.CreatorID == cfg.ActorID || cfg.CreatorTokenFile == "" {
		t.Fatal("native fixture requires a distinct creator and restricted creator_token_file")
	}
	if cfg.ReviewerID <= 0 || cfg.ReviewerID == cfg.ActorID || cfg.ReviewerID == cfg.CreatorID || cfg.ReviewerTokenFile == "" {
		t.Fatal("native fixture requires a distinct reviewer and restricted reviewer_token_file")
	}
	if !factory.ValidTargetBranch(cfg.BaseBranch) {
		t.Fatal("native fixture requires exact base_branch")
	}
	if !cfg.MergeBound {
		t.Fatal("native proof suite requires the pull_request.merge binding")
	}
	if cfg.FountainDB == "" {
		t.Fatal("native fixture requires its fountain_db path")
	}
	return cfg
}

func nativeMergeReceipt(t *testing.T, label string, value any) {
	t.Helper()
	root := os.Getenv("ST12_RECEIPT_DIR")
	if root == "" {
		t.Fatal("ST12_RECEIPT_DIR is required to retain native proof")
	}
	if !filepath.IsAbs(root) {
		t.Fatal("native receipt directory must be absolute")
	}
	nativeMust(t, os.MkdirAll(root, 0o700))
	raw, err := json.MarshalIndent(value, "", "  ")
	nativeMust(t, err)
	nativeMust(t, os.WriteFile(filepath.Join(root, label+".json"), append(raw, '\n'), 0o600))
}

// nativeMergeDrain waits until the native revision is unchanged for a
// beat: the Idle flag does not cover async completion effects (one
// merge lands two advances within a second, all reported idle), and a
// conditioned submit opened during that churn is refused stale. Read
// errors count as motion, never as quiescence.
func nativeMergeDrain(t *testing.T, bg *forgejo.ServiceBackground) extensions.NativeRevisionObservation {
	t.Helper()
	ctx := context.Background()
	deadline := time.Now().Add(60 * time.Second)
	var last extensions.NativeRevisionObservation
	stable := time.Now()
	first := true
	for time.Now().Before(deadline) {
		revision, err := bg.ReadNativeRevision(ctx)
		if err != nil || first || revision.Revision != last.Revision {
			last = revision
			stable = time.Now()
			first = false
			time.Sleep(100 * time.Millisecond)
			continue
		}
		if time.Since(stable) >= 1500*time.Millisecond {
			return revision
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatal("native revision never settled")
	return extensions.NativeRevisionObservation{}
}

// nativeMergeIdle waits for a quiescent host before a conditioned
// submit opens its bracket. Stability subsumes the Idle flag.
func nativeMergeIdle(t *testing.T, bg *forgejo.ServiceBackground) extensions.NativeRevisionObservation {
	t.Helper()
	return nativeMergeDrain(t, bg)
}

// nativeMergeStaleRefusal reports whether an outcome is the host-churn
// refusal (certain no-effect) rather than a scenario outcome.
func nativeMergeStaleRefusal(outcome factory.OperationOutcome) bool {
	return outcome.Effect == factory.OpEffectNotCommitted && outcome.Reason == "stale_native_revision"
}

// nativeMergeBusyError reports whether err is the transient busy-host
// signal on a direct native call: an HTTP 503 or a native_busy wait.
func nativeMergeBusyError(err error) bool {
	var status *forgejopublish.StatusError
	if errors.As(err, &status) && status.Status == http.StatusServiceUnavailable {
		return true
	}
	var wait *factory.PublicationWait
	return errors.As(err, &wait) && wait.Reason == "native_busy"
}

// nativeMergeCall retries one idempotent native read (or same-identity
// submit) until it answers; only the transient busy-host signal
// retries, everything else fails fast.
func nativeMergeCall[T any](t *testing.T, label string, call func() (T, error)) T {
	t.Helper()
	deadline := time.Now().Add(60 * time.Second)
	for {
		value, err := call()
		if err == nil {
			return value
		}
		if !nativeMergeBusyError(err) || !time.Now().Before(deadline) {
			t.Fatalf("native %s unanswered: %v", label, err)
		}
		time.Sleep(200 * time.Millisecond)
	}
}

// nativeMergeSubmitReviewCommitted submits one conditional review and
// requires its commit. An uncertain submit retries under its recorded
// identity; a certain stale refusal (host churn, no effect) reopens
// under a fresh identity.
func nativeMergeSubmitReviewCommitted(t *testing.T, r *forgejo.Reviewer, bg *forgejo.ServiceBackground, base factory.ReviewWork) factory.OperationOutcome {
	t.Helper()
	ctx := context.Background()
	for attempt := 0; attempt < 10; attempt++ {
		nativeMergeIdle(t, bg)
		w := base
		w.OperationID = "st12-" + factory.NewID()
		w = nativeReviewObserved(t, r, w)
		stale := false
		for i := 0; i < 5; i++ {
			outcome, err := r.SubmitReview(ctx, w)
			if err == nil {
				if outcome.Effect == factory.OpEffectCommitted {
					return outcome
				}
				if !nativeMergeStaleRefusal(outcome) {
					t.Fatalf("native review refused: %+v", outcome)
				}
				stale = true
				break
			}
			var refused *factory.PublicationRefusal
			if errors.As(err, &refused) {
				t.Fatalf("native review refused: %v", err)
			}
			time.Sleep(time.Duration(200*(i+1)) * time.Millisecond)
		}
		if !stale {
			t.Fatal("native review submit never confirmed")
		}
		t.Logf("native review reopened after host churn (attempt %d)", attempt+1)
	}
	t.Fatal("native review never committed")
	return factory.OperationOutcome{}
}

// nativeMergeSubmitMergeCommitted submits one conditional merge through
// the production client and requires its commit, reopening under a
// fresh identity only on certain stale refusal like the review path.
func nativeMergeSubmitMergeCommitted(t *testing.T, m control.MergeExecutor, bg *forgejo.ServiceBackground, base factory.MergeWork) factory.OperationOutcome {
	t.Helper()
	ctx := context.Background()
	for attempt := 0; attempt < 10; attempt++ {
		revision := nativeMergeIdle(t, bg)
		w := base
		w.OperationID = "st12-merge-" + factory.NewID()
		w.NativeRev = revision.Revision
		w.NotAfter = time.Now().Add(5 * time.Minute).Unix()
		stale := false
		for i := 0; i < 5; i++ {
			outcome, err := m.SubmitMerge(ctx, w)
			if err == nil {
				if outcome.Effect == factory.OpEffectCommitted {
					return outcome
				}
				if !nativeMergeStaleRefusal(outcome) {
					t.Fatalf("native merge refused: %+v", outcome)
				}
				stale = true
				break
			}
			var refused *factory.PublicationRefusal
			if errors.As(err, &refused) {
				t.Fatalf("native merge refused: %v", err)
			}
			time.Sleep(time.Duration(200*(i+1)) * time.Millisecond)
		}
		if !stale {
			t.Fatal("native merge submit never confirmed")
		}
		t.Logf("native merge reopened after host churn (attempt %d)", attempt+1)
	}
	t.Fatal("native merge never committed")
	return factory.OperationOutcome{}
}

// nativeMergeSubmitRawCommitted submits one merge intent directly and
// returns its committed record, reopening under a fresh identity only
// on certain stale refusal. Snapshot probes use the raw path to pin
// the exact receipt wire shape.
func nativeMergeSubmitRawCommitted(t *testing.T, bg *forgejo.ServiceBackground, tokenFile, authRev string, actorID, repoID int64, payload []byte) extensions.OperationRecord {
	t.Helper()
	ctx := context.Background()
	for attempt := 0; attempt < 10; attempt++ {
		revision := nativeMergeIdle(t, bg)
		intent := extensions.OperationIntent{
			OperationID: "st12-raw-" + factory.NewID(), ActorID: strconv.FormatInt(actorID, 10),
			RepositoryID: strconv.FormatInt(repoID, 10), Kind: factory.OpMerge,
			AuthorizationRevision: authRev, ExpectedNativeRevision: revision.Revision,
			NotAfter: time.Now().Add(5 * time.Minute).Unix(), Payload: payload,
		}
		record := nativeMergeCall(t, "merge submit", func() (extensions.OperationRecord, error) {
			return bg.SubmitOperation(ctx, extensions.CredentialFile(tokenFile), intent)
		})
		if record.EffectState == factory.OpEffectCommitted {
			return record
		}
		if record.EffectState != factory.OpEffectNotCommitted || record.ReasonCode != "stale_native_revision" {
			t.Fatalf("native merge refused: effect=%s reason=%s completion=%s", record.EffectState, record.ReasonCode, record.CompletionState)
		}
		t.Logf("native merge reopened after host churn (attempt %d)", attempt+1)
	}
	t.Fatal("native merge never committed")
	return extensions.OperationRecord{}
}

type nativeMergePR struct {
	branch   string
	head     string
	base     string
	prID     int64
	prNumber int64
	issueID  int64
}

// nativeMergeAPI posts one native mutation, retrying only the transient
// 503 a busy host returns while a native mutation reservation is held.
// Any other refusal fails fast; reads stay on the shared helper.
func nativeMergeAPI(t *testing.T, c nativeST12Config, method, path string, body, target any) {
	t.Helper()
	var raw []byte
	if body != nil {
		var err error
		raw, err = json.Marshal(body)
		nativeMust(t, err)
	}
	deadline := time.Now().Add(60 * time.Second)
	client := &http.Client{
		Timeout:       30 * time.Second,
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
	for {
		var input io.Reader
		if raw != nil {
			input = bytes.NewReader(raw)
		}
		req, err := http.NewRequestWithContext(context.Background(), method, strings.TrimRight(c.FountainURL, "/")+path, input)
		nativeMust(t, err)
		req.Header.Set("Authorization", "token "+nativeSecret(t, c.CreatorTokenFile))
		req.Header.Set("Content-Type", "application/json")
		response, err := client.Do(req)
		if err != nil {
			t.Fatal("native API transport unavailable")
		}
		if response.StatusCode != http.StatusServiceUnavailable || !time.Now().Before(deadline) {
			defer response.Body.Close()
			if response.StatusCode < 200 || response.StatusCode >= 300 {
				t.Fatalf("native fixture API %s %s: status %d", method, path, response.StatusCode)
			}
			if target != nil {
				nativeMust(t, json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(target))
			}
			return
		}
		response.Body.Close()
		time.Sleep(200 * time.Millisecond)
	}
}

func nativeMergePublish(t *testing.T, c nativeST12Config, label string) (nativeMergePR, nativeCandidate) {
	t.Helper()
	n := nativeNewCandidate(t, c.nativeST09Config)
	branch := "st12-" + label + "-" + factory.NewID()[:12]
	nativeGitOK(t, c.nativeST09Config, n.dir, "push", nativeRepoURL(c.nativeST09Config), n.head+":refs/heads/"+branch)
	var pr struct {
		ID     int64 `json:"id"`
		Number int64 `json:"number"`
	}
	nativeMergeAPI(t, c, http.MethodPost, "/api/v1/repos/"+c.Owner+"/"+c.Repo+"/pulls",
		map[string]any{"head": branch, "base": strings.TrimPrefix(c.BaseBranch, "refs/heads/"), "title": "ST12 " + label, "body": "Exact merge fixture", "allow_maintainer_edit": false}, &pr)
	if pr.ID <= 0 || pr.Number <= 0 {
		t.Fatal("native PR identity unconfirmed")
	}
	var issue struct {
		ID     int64 `json:"id"`
		Number int64 `json:"number"`
	}
	nativeAPI(t, c.nativeST09Config, c.CreatorTokenFile, http.MethodGet, "/api/v1/repos/"+c.Owner+"/"+c.Repo+"/issues/"+strconv.FormatInt(pr.Number, 10), nil, &issue)
	if issue.ID <= 0 || issue.Number != pr.Number {
		t.Fatal("native PR issue identity unconfirmed")
	}
	return nativeMergePR{branch: branch, head: n.head, base: n.base, prID: pr.ID, prNumber: pr.Number, issueID: issue.ID}, n
}

// TestNativeMergeWire pins the pull_request.merge receipt wire shape from
// the staged native producer. AdoptMerge must match this shape exactly.
func TestNativeMergeWire(t *testing.T) {
	c := loadNativeST12(t)
	p, _ := nativeMergePublish(t, c, "wire")
	bg := forgejo.NewServiceBackground(c.Socket, uint32(os.Getuid()), "")
	payload := extensions.MergePayload{
		PullRequestNumber: p.prNumber, HeadRepositoryID: c.Repository,
		HeadRef: "refs/heads/" + p.branch, BaseRef: c.BaseBranch,
		ExpectedHeadOID: p.head, ExpectedBaseOID: p.base,
		Method: extensions.MergeMethodFastForwardOnly,
	}
	raw, err := json.Marshal(payload)
	nativeMust(t, err)
	record := nativeMergeSubmitRawCommitted(t, bg, c.TokenFile, "st12-native-wire", c.ActorID, c.Repository, raw)
	nativeMergeReceipt(t, "merge-wire", record)
	var fields map[string]any
	nativeMust(t, json.Unmarshal(record.Receipt, &fields))
	t.Logf("receipt wire: %v", fields)
	if nativeTip(t, c.nativeST09Config, c.BaseBranch) != p.head {
		t.Fatal("base tip does not carry the merged head")
	}
}

// TestNativeMergeReviewSnapshot pins the reviews-family evidence Soda
// consumes as merge review evidence: the exact type strings and commit
// binding of conditional approvals and change requests.
func TestNativeMergeReviewSnapshot(t *testing.T) {
	c := loadNativeST12(t)
	ctx := context.Background()
	p, _ := nativeMergePublish(t, c, "reviewsnap")
	bg := forgejo.NewServiceBackground(c.Socket, uint32(os.Getuid()), "")
	r := forgejo.NewReviewer(bg, forgejo.New(c.FountainURL), c.ReviewerTokenFile)
	outcome := nativeMergeSubmitReviewCommitted(t, r, bg, factory.ReviewWork{
		AuthRevision: "st12-native-snapshot",
		Repository:   c.Repository, ActorID: c.ReviewerID, PRNumber: p.prNumber, PRID: p.prID,
		IssueID: p.issueID, PRAuthorID: c.CreatorID,
		HeadRef: "refs/heads/" + p.branch, BaseRef: c.BaseBranch, HeadOID: p.head, BaseOID: p.base,
		Event: "APPROVED", Body: "ST12 snapshot probe approval.",
	})
	nativeMergeReceipt(t, "reviewsnap-submit", outcome)
	for _, family := range []string{"reviews", "pull", "issue"} {
		req := extensions.SnapshotRequest{
			RepositoryID: strconv.FormatInt(c.Repository, 10), ActorID: strconv.FormatInt(c.ActorID, 10),
			Families: []string{family},
		}
		if family == "issue" {
			req.IssueIndex = strconv.FormatInt(p.prNumber, 10)
		} else {
			req.PullNumber = strconv.FormatInt(p.prNumber, 10)
		}
		observed := nativeMergeCall(t, family+" snapshot", func() (extensions.NativeSnapshot, error) {
			return bg.ReadSnapshot(ctx, extensions.CredentialFile(c.TokenFile), req)
		})
		nativeMergeReceipt(t, "reviewsnap-"+family, observed)
	}
	// Pin the change-request snapshot shape on a second PR.
	p2, _ := nativeMergePublish(t, c, "reqchanges")
	nativeMergeSubmitReviewCommitted(t, r, bg, factory.ReviewWork{
		AuthRevision: "st12-native-snapshot",
		Repository:   c.Repository, ActorID: c.ReviewerID, PRNumber: p2.prNumber, PRID: p2.prID,
		IssueID: p2.issueID, PRAuthorID: c.CreatorID,
		HeadRef: "refs/heads/" + p2.branch, BaseRef: c.BaseBranch, HeadOID: p2.head, BaseOID: p2.base,
		Event: "REQUEST_CHANGES", Body: "ST12 snapshot probe findings.",
	})
	observed := nativeMergeCall(t, "reviews snapshot", func() (extensions.NativeSnapshot, error) {
		return bg.ReadSnapshot(ctx, extensions.CredentialFile(c.TokenFile), extensions.SnapshotRequest{
			RepositoryID: strconv.FormatInt(c.Repository, 10), ActorID: strconv.FormatInt(c.ActorID, 10),
			PullNumber: strconv.FormatInt(p2.prNumber, 10), Families: []string{"reviews"},
		})
	})
	nativeMergeReceipt(t, "reviewsnap-reqchanges", observed)
}

// TestNativeMergeConfirmationSnapshot pins the post-merge pull and issue
// evidence Soda consumes as completion confirmation.
func TestNativeMergeConfirmationSnapshot(t *testing.T) {
	c := loadNativeST12(t)
	ctx := context.Background()
	p, _ := nativeMergePublish(t, c, "confsnap")
	bg := forgejo.NewServiceBackground(c.Socket, uint32(os.Getuid()), "")
	payload := extensions.MergePayload{
		PullRequestNumber: p.prNumber, HeadRepositoryID: c.Repository,
		HeadRef: "refs/heads/" + p.branch, BaseRef: c.BaseBranch,
		ExpectedHeadOID: p.head, ExpectedBaseOID: p.base,
		Method: extensions.MergeMethodFastForwardOnly,
	}
	raw, err := json.Marshal(payload)
	nativeMust(t, err)
	nativeMergeSubmitRawCommitted(t, bg, c.TokenFile, "st12-native-snapshot", c.ActorID, c.Repository, raw)
	nativeMergeIdle(t, bg)
	for _, family := range []string{"pull", "issue", "refs"} {
		req := extensions.SnapshotRequest{
			RepositoryID: strconv.FormatInt(c.Repository, 10), ActorID: strconv.FormatInt(c.ActorID, 10),
			Families: []string{family},
		}
		if family == "issue" {
			req.IssueIndex = strconv.FormatInt(p.prNumber, 10)
		} else {
			req.PullNumber = strconv.FormatInt(p.prNumber, 10)
		}
		if family == "refs" {
			req.Refs = []string{"refs/heads/" + p.branch, c.BaseBranch}
		}
		observed := nativeMergeCall(t, family+" snapshot", func() (extensions.NativeSnapshot, error) {
			return bg.ReadSnapshot(ctx, extensions.CredentialFile(c.TokenFile), req)
		})
		nativeMergeReceipt(t, "confsnap-"+family, observed)
	}
}

// nativeMergeSetup seeds one assignment with the reviewer binding
// separated from the author, and wires the production merger. It waits
// for native quiescence first: back-to-back tests share one fixture,
// and setup writes must not race the previous test's merge activity.
func nativeMergeSetup(t *testing.T, c nativeST12Config) (*nativeFixture, nativeCandidate) {
	t.Helper()
	nativeMergeIdle(t, forgejo.NewServiceBackground(c.Socket, uint32(os.Getuid()), ""))
	n := nativeNewCandidate(t, c.nativeST09Config)
	fx := nativeSeed(t, c.nativeST09Config, n)
	ctx := context.Background()
	policy, err := fx.db.RepositoryPolicy(ctx, c.Repository)
	nativeMust(t, err)
	policy.Review = factory.ActorBindingRef{TokenID: c.ReviewerTokenID, ActorID: c.ReviewerID, Kind: factory.OpReviewSubmit}
	nativeMust(t, fx.db.SaveRepositoryPolicy(ctx, policy))
	fx.coord.Merges = forgejo.NewMerger(fx.background, forgejo.New(c.FountainURL), c.TokenFile)
	return fx, n
}

// nativeMergeAssignment mirrors the shared fixture assignment, except the
// native issue write retries the transient busy-host 503 through
// nativeMergeAPI instead of failing fast.
func nativeMergeAssignment(t *testing.T, c nativeST12Config, fx *nativeFixture, n nativeCandidate) factory.Assignment {
	t.Helper()
	ctx := context.Background()
	cfg := fx.cfg
	rest := forgejo.New(cfg.FountainURL)
	creator, err := rest.Current(ctx, nativeSecret(t, cfg.CreatorTokenFile))
	nativeMust(t, err)
	repo, err := rest.RepositoryByID(ctx, nativeSecret(t, cfg.CreatorTokenFile), cfg.Repository)
	nativeMust(t, err)
	if creator.ID != cfg.CreatorID || repo.Permissions == nil || !repo.Permissions.Push {
		t.Fatal("fixture creator does not have verified native write authority")
	}
	var issue struct {
		Number int64 `json:"number"`
	}
	nativeMergeAPI(t, c, http.MethodPost, "/api/v1/repos/"+cfg.Owner+"/"+cfg.Repo+"/issues", map[string]string{"title": "ST12 " + factory.NewID(), "body": "Merge this fixture candidate."}, &issue)
	if issue.Number <= 0 {
		t.Fatal("native issue lacks its number")
	}
	evidence, err := fx.coord.AcceptanceReads.ReadAcceptanceEvidence(ctx, strconv.FormatInt(cfg.Repository, 10), strconv.FormatInt(issue.Number, 10), nil)
	nativeMust(t, err)
	if len(evidence.Dependencies) != 0 {
		t.Fatal("native fixture issue unexpectedly has prerequisites")
	}
	decision := factory.Acceptance{ID: "d" + factory.NewID()[:24], Repository: cfg.Repository, IssueIndex: strconv.FormatInt(issue.Number, 10), Approver: creator.ID, NativeRev: evidence.Revision, TitleDigest: evidence.Issue.TitleDigest, ContentDigest: evidence.Issue.ContentDigest, ContentVersion: evidence.Issue.ContentVer}
	receipt, err := fx.coord.AdmitAcceptance(ctx, factory.NewID(), "native:"+strconv.FormatInt(creator.ID, 10), decision)
	nativeMust(t, err)
	nativeReceipt(t, t.Name()+"-acceptance", struct {
		AuthenticatedActor int64
		CodeWrite          bool
		Evidence           control.AcceptanceEvidence
		Decision           factory.Acceptance
		Receipt            control.AcceptanceReceipt
	}{creator.ID, repo.Permissions.Push, evidence, decision, receipt})
	authority, err := fx.coord.EffectiveAuthority(ctx, cfg.Repository)
	nativeMust(t, err)
	if !authority.Effective {
		t.Fatalf("fixture authority ineffective: %+v", authority)
	}
	bound := authority.Authority
	bound.RequirementsID, err = fx.db.RequirementHead(ctx, fx.project)
	nativeMust(t, err)
	bound.ApprovalID, err = fx.db.ApprovalHead(ctx, fx.project)
	nativeMust(t, err)
	now := time.Now().Truncate(time.Second)
	prompt := []byte("ST12 recorded stopped-run fixture")
	hash := sha256.Sum256(prompt)
	a := factory.Assignment{Authority: bound, ID: factory.NewID(), ProjectID: fx.project, Role: project.RoleCoder, Repository: cfg.Repository, Issue: issue.Number, NativeRev: decision.NativeRev, Acceptance: decision.ID, Preparation: fx.preparation, Harness: "codex-1.2.3", HarnessVers: "1.2.3", Model: "fixture", Connection: "fixture-connection", SourceCommit: n.base, Prompt: prompt, PromptSHA: hex.EncodeToString(hash[:]), Run: factory.NewID(), Stage: factory.AssignmentAssigned, Attempts: 1, CreatedUnix: now.Unix()}
	a.RunHistory = []string{a.Run}
	run := factory.Run{ID: a.Run, ProjectID: a.ProjectID, Role: a.Role, InputSHA: n.base, Started: now, Deadline: now.Add(time.Hour), Image: "sha256:" + strings.Repeat("b", 64), Harness: a.Harness, Model: a.Model}
	nativeMust(t, fx.db.RecordDispatchPacket(ctx, factory.DispatchRegistration{ID: a.ID, Repository: a.Repository, Authority: a.Authority}, a, factory.Reservation{AssignmentID: a.ID, Repository: cfg.Repository, Connection: a.Connection, State: factory.ReservationHeld, PlannedMinutes: 30}, run, factory.RunView{RunID: a.Run, Repository: cfg.Repository, Issue: a.Issue, Attempt: a.ID}))
	run.Reconciled, run.Outcome, run.Summary = true, factory.Succeeded, "completed fixture candidate"
	nativeMust(t, fx.db.SaveFactoryRun(ctx, run))
	nativeMust(t, fx.db.ConsumeReservation(ctx, a.ID))
	a.Stage, a.Outcome, a.Reason, a.FinishedUnix = factory.AssignmentFinished, factory.Succeeded, factory.AssignReasonReported, now.Unix()
	a.Result = &factory.AssignmentResult{AssignmentID: a.ID, RunID: a.Run, Status: "completed", Summary: "fixture candidate", Candidate: n.head, Findings: []string{}, Reported: true, RecordedUnix: now.Unix()}
	nativeMust(t, fx.db.FinishAssignment(ctx, a))
	return a
}

// nativeMergeApprove submits one conditional review through the
// production reviewer and requires its native commit.
func nativeMergeApprove(t *testing.T, c nativeST12Config, p factory.Publication, event, body string) {
	t.Helper()
	if p.PRCreate.Work == nil {
		t.Fatalf("published publication lacks its recorded creation intent: stage=%s reason=%s publish=%+v prcreate=%+v", p.Stage, p.Reason, p.Publish, p.PRCreate)
	}
	bg := forgejo.NewServiceBackground(c.Socket, uint32(os.Getuid()), "")
	r := forgejo.NewReviewer(bg, forgejo.New(c.FountainURL), c.ReviewerTokenFile)
	nativeMergeSubmitReviewCommitted(t, r, bg, factory.ReviewWork{
		AuthRevision: "st12-native-proof",
		Repository:   c.Repository, ActorID: c.ReviewerID, PRNumber: p.PRNumber, PRID: p.PRID,
		IssueID: p.PRCreate.IssueID, PRAuthorID: p.PRCreate.Work.ActorID,
		HeadRef: p.PRCreate.HeadRef, BaseRef: p.PRCreate.BaseRef,
		HeadOID: p.PRCreate.HeadOID, BaseOID: p.PRCreate.BaseOID,
		Event: event, Body: body,
	})
}

// nativeMergeAssess runs the production ST11 read path and records the
// verdict: the exact record MergePass consumes.
func nativeMergeAssess(t *testing.T, c nativeST12Config, fx *nativeFixture, p factory.Publication) factory.CheckAssessment {
	t.Helper()
	assessor := forgejo.NewCheckAssessor(forgejo.NewServiceBackground(c.Socket, uint32(os.Getuid()), ""), forgejo.New(c.FountainURL), c.TokenFile)
	target := factory.CheckTarget{
		Repository: c.Repository, PRNumber: p.PRNumber, PRID: p.PRID, IssueID: p.PRCreate.IssueID,
		HeadRef: p.PRCreate.HeadRef, BaseRef: p.PRCreate.BaseRef,
		HeadOID: p.PRCreate.HeadOID, BaseOID: p.PRCreate.BaseOID,
	}
	observed := nativeObserveChecks(t, assessor, target, c.ActorID)
	policy, err := fx.db.RepositoryPolicy(context.Background(), c.Repository)
	nativeMust(t, err)
	adopted := factory.AdoptedChecks{Checks: append([]string(nil), policy.Checks...), PolicyRevision: policy.Revision, Digest: factory.ChecksDigest(policy.Checks)}
	assessment, err := factory.VerifyChecks(target, adopted, policy, observed, time.Now().Unix())
	nativeMust(t, err)
	stored, err := fx.db.RecordCheckAssessment(context.Background(), assessment)
	nativeMust(t, err)
	return stored
}

// nativeMergeOpenRow records an open merge row exactly like mergeOne's
// construction, for reconcile-gating tests: MergePass only opens rows
// behind a current pass, so tests that regress the evidence afterwards
// seed their row directly instead of driving the opening policy.
func nativeMergeOpenRow(t *testing.T, fx *nativeFixture, p factory.Publication) {
	t.Helper()
	ctx := context.Background()
	policy, err := fx.db.RepositoryPolicy(ctx, p.Repository)
	nativeMust(t, err)
	m := factory.Merge{
		Operation: factory.MergeOperation{Kind: factory.OpMerge},
		Authority: p.Authority, ID: factory.NewID(), PublicationID: p.ID,
		AssignmentID: p.AssignmentID, ProjectID: p.ProjectID, Role: p.Role,
		Acceptance: p.Acceptance, HeadRef: p.PRCreate.HeadRef, BaseRef: p.PRCreate.BaseRef,
		HeadOID: p.Candidate, BaseOID: p.PRCreate.BaseOID,
		Stage: factory.MergeOpen, Repository: p.Repository, Issue: p.Issue,
		PRNumber: p.PRNumber, PRID: p.PRID, IssueID: p.PRCreate.IssueID,
		PRAuthorID: p.PRCreate.Work.ActorID, ReviewerID: policy.Review.ActorID,
		CreatedUnix: time.Now().Unix(),
	}
	nativeMust(t, fx.db.RecordMerge(ctx, m))
}

// nativeMergeDrive runs MergePass until the merge leaves open/fenced or
// the deadline passes. Waits retry; transient native_unavailable errors
// retry too, since the fenced host refuses with 503 while a native
// mutation reservation is held. Any other error fails fast.
func nativeMergeDrive(t *testing.T, c nativeST12Config, fx *nativeFixture, publicationID string) (factory.Merge, control.MergeReport) {
	t.Helper()
	nativeMergeIdle(t, forgejo.NewServiceBackground(c.Socket, uint32(os.Getuid()), ""))
	ctx := context.Background()
	var last control.MergeReport
	for i := 0; i < 90; i++ {
		last = fx.coord.MergePass(ctx)
		for _, entry := range last.Errors {
			if entry.Reason != "native_unavailable" {
				t.Fatalf("merge errors: %+v", last)
			}
		}
		m, err := fx.db.MergeByPublication(ctx, publicationID)
		if err == nil && m.Stage != factory.MergeOpen && m.Stage != factory.MergeFenced {
			return m, last
		}
		time.Sleep(200 * time.Millisecond)
	}
	m, _ := fx.db.MergeByPublication(ctx, publicationID)
	t.Fatalf("merge did not settle: %+v last=%+v", m, last)
	return factory.Merge{}, last
}

// nativeMergePublishDrive runs PublishPass until one assignment's
// publication leaves open/fenced, tolerating only transient
// native_unavailable errors like nativeMergeDrive. Any terminal stage
// returns; the caller decides whether it is the scenario outcome.
func nativeMergePublishDrive(t *testing.T, c nativeST12Config, fx *nativeFixture, a factory.Assignment) factory.Publication {
	t.Helper()
	nativeMergeIdle(t, forgejo.NewServiceBackground(c.Socket, uint32(os.Getuid()), ""))
	ctx := context.Background()
	var p factory.Publication
	for i := 0; i < 120; i++ {
		report := fx.coord.PublishPass(ctx)
		for _, entry := range report.Errors {
			if entry.Reason != "native_unavailable" {
				t.Fatalf("publication errors: %+v", report)
			}
		}
		var err error
		p, err = fx.db.PublicationByAssignment(ctx, a.ID)
		nativeMust(t, err)
		if p.Stage != factory.PublicationOpen && p.Stage != factory.PublicationFenced {
			return p
		}
		time.Sleep(200 * time.Millisecond)
	}
	t.Fatalf("publication did not settle: stage=%s", p.Stage)
	return factory.Publication{}
}

// nativeMergeSeedPublication publishes one candidate, reseeding with a
// fresh assignment when host churn (certain stale refusal, no effect)
// burns the seeding submit. Anything else fails fast: only churn
// retries, never a scenario outcome.
func nativeMergeSeedPublication(t *testing.T, c nativeST12Config, fx *nativeFixture, n nativeCandidate) (factory.Publication, factory.Assignment) {
	t.Helper()
	for attempt := 0; attempt < 5; attempt++ {
		a := nativeMergeAssignment(t, c, fx, n)
		p := nativeMergePublishDrive(t, c, fx, a)
		if p.Stage == factory.PublicationPublished {
			return p, a
		}
		stale := p.Stage == factory.PublicationFailed &&
			(p.Publish.Reason == "stale_native_revision" || p.PRCreate.Reason == "stale_native_revision")
		if !stale {
			t.Fatalf("publication settled unpublished: stage=%s reason=%s publish=%+v prcreate=%+v", p.Stage, p.Reason, p.Publish, p.PRCreate)
		}
		t.Logf("publication reseeded after host churn (attempt %d)", attempt+1)
	}
	t.Fatal("publication never published")
	return factory.Publication{}, factory.Assignment{}
}

// nativeMergeInsertEdge records one blocked-by-blocker edge in the staged
// Forgejo database; Soda observes it back through production snapshots,
// so the proved observation path stays native.
func nativeMergeInsertEdge(t *testing.T, c nativeST12Config, blockedID, blockerID int64) {
	t.Helper()
	nativeMust(t, store.SeedStagedDependencyEdge(c.FountainDB, c.CreatorID, blockedID, blockerID))
}

// TestNativeMergeFullPass proves the complete ST12 path: a published PR
// with an independent approval and a recorded ST11 pass merges through
// MergePass, confirms its native completion, and links exactly.
func TestNativeMergeFullPass(t *testing.T) {
	c := loadNativeST12(t)
	for attempt := 0; attempt < 3; attempt++ {
		if nativeMergeFullPassAttempt(t, c) {
			return
		}
		t.Logf("full pass reseeded after host churn (attempt %d)", attempt+1)
	}
	t.Fatal("full pass never observed its scenario outcome")
}

// nativeMergeFullPassAttempt runs one full-pass scenario: false means
// host churn (certain stale refusal, no effect) burned the proof
// submit, and only then may the caller reseed.
func nativeMergeFullPassAttempt(t *testing.T, c nativeST12Config) bool {
	t.Helper()
	ctx := context.Background()
	fx, n := nativeMergeSetup(t, c)
	p, _ := nativeMergeSeedPublication(t, c, fx, n)
	nativeMergeApprove(t, c, p, "APPROVED", "ST12 proof approval.")
	nativePostStatus(t, c.nativeST09Config, p.PRCreate.HeadOID, "verify", "success")
	assessment := nativeMergeAssess(t, c, fx, p)
	if assessment.Verdict != factory.CheckPass {
		t.Fatalf("seed assessment did not pass: %+v", assessment)
	}
	m, report := nativeMergeDrive(t, c, fx, p.ID)
	if m.Stage == factory.MergeFailed && m.Reason == factory.MergeReasonRefused && m.Operation.Reason == "stale_native_revision" {
		return false
	}
	if m.Stage != factory.MergeMerged || len(report.Merged) != 1 || m.Operation.Attempts != 1 {
		t.Fatalf("merge unsettled: %+v %+v", m, report)
	}
	if m.MergedCommit != p.PRCreate.HeadOID || m.Operation.MergedCommit != p.PRCreate.HeadOID {
		t.Fatalf("merge realized the wrong commit: %+v", m)
	}
	if nativeTip(t, c.nativeST09Config, c.BaseBranch) != p.PRCreate.HeadOID {
		t.Fatal("base tip does not carry the merged head")
	}
	var issue struct {
		State string `json:"state"`
	}
	nativeAPI(t, c.nativeST09Config, c.CreatorTokenFile, http.MethodGet, "/api/v1/repos/"+c.Owner+"/"+c.Repo+"/issues/"+strconv.FormatInt(p.PRNumber, 10), nil, &issue)
	if issue.State != "closed" {
		t.Fatalf("merged PR issue is not closed: %q", issue.State)
	}
	// The confirmation stamps come from native evidence, not the submit:
	// the merger, merge stamp and close stamp must match a fresh read.
	bg := forgejo.NewServiceBackground(c.Socket, uint32(os.Getuid()), "")
	merger := forgejo.NewMerger(bg, forgejo.New(c.FountainURL), c.TokenFile)
	work := m.Operation.Work.Apply(factory.MergeWork{MergeID: m.ID, PublicationID: m.PublicationID, Issue: m.Issue})
	confirmation := nativeMergeCall(t, "completion observation", func() (factory.MergeConfirmation, error) {
		return merger.ObserveCompletion(ctx, work)
	})
	if m.MergedUnix != confirmation.MergedUnix || m.ClosedUnix != confirmation.ClosedUnix || confirmation.MergerID != c.ActorID {
		t.Fatalf("completion stamps are not native evidence: %+v vs %+v", m, confirmation)
	}
	nativeMergeReceipt(t, "fullpass-merge", m)
	nativeMergeReceipt(t, "fullpass-report", report)
	return true
}

// nativeMergeGit runs one git command with retries: the shared fixture
// serves rapid back-to-back tests, and a single busy response must not
// fail the proof. Failures name the command.
func nativeMergeGit(t *testing.T, c nativeST12Config, dir string, args ...string) string {
	t.Helper()
	var err error
	var out string
	for i := 0; i < 5; i++ {
		if out, err = nativeGit(t, c.nativeST09Config, dir, nil, args...); err == nil {
			return out
		}
		time.Sleep(time.Duration(200*(i+1)) * time.Millisecond)
	}
	t.Fatalf("native fixture git %q failed: %v", args, err)
	return ""
}

// nativeMergeAdvanceBranch appends one commit to an existing native
// branch and returns its tip.
func nativeMergeAdvanceBranch(t *testing.T, c nativeST12Config, branch string) string {
	t.Helper()
	var dir string
	var err error
	for i := 0; i < 5; i++ {
		dir = filepath.Join(t.TempDir(), "advance")
		if _, err = nativeGit(t, c.nativeST09Config, "", nil, "clone", nativeRepoURL(c.nativeST09Config), dir); err == nil {
			break
		}
		time.Sleep(time.Duration(200*(i+1)) * time.Millisecond)
	}
	if err != nil {
		t.Fatalf("native fixture git clone failed: %v", err)
	}
	nativeMergeGit(t, c, dir, "checkout", branch)
	nativeMust(t, os.WriteFile(filepath.Join(dir, "advance.txt"), []byte("ST12 advance "+factory.NewID()+"\n"), 0o600))
	nativeMergeGit(t, c, dir, "add", "advance.txt")
	nativeMergeGit(t, c, dir, "-c", "user.name=soda-tester", "-c", "user.email=soda-tester@localhost", "commit", "-m", "ST12 advance")
	nativeMergeGit(t, c, dir, "push", "origin", branch)
	return nativeMergeGit(t, c, dir, "rev-parse", "HEAD")
}

// TestNativeMergeStaleHead proves a moved candidate never merges: even
// with a valid approval on the new head, the merge refuses its stale
// binding without submitting anything.
func TestNativeMergeStaleHead(t *testing.T) {
	c := loadNativeST12(t)
	fx, n := nativeMergeSetup(t, c)
	p, a := nativeMergeSeedPublication(t, c, fx, n)
	before := nativeTip(t, c.nativeST09Config, c.BaseBranch)
	nativeMergeOpenRow(t, fx, p)
	head2 := nativeMergeAdvanceBranch(t, c, factory.PublicationBranch(a.ID))
	if head2 == p.PRCreate.HeadOID {
		t.Fatal("advance did not move the candidate")
	}
	// A fresh approval on the new head does not rescue the stale binding.
	bg := forgejo.NewServiceBackground(c.Socket, uint32(os.Getuid()), "")
	r := forgejo.NewReviewer(bg, forgejo.New(c.FountainURL), c.ReviewerTokenFile)
	nativeMergeSubmitReviewCommitted(t, r, bg, factory.ReviewWork{
		AuthRevision: "st12-native-proof",
		Repository:   c.Repository, ActorID: c.ReviewerID, PRNumber: p.PRNumber, PRID: p.PRID,
		IssueID: p.PRCreate.IssueID, PRAuthorID: p.PRCreate.Work.ActorID,
		HeadRef: p.PRCreate.HeadRef, BaseRef: p.PRCreate.BaseRef,
		HeadOID: head2, BaseOID: p.PRCreate.BaseOID,
		Event: "APPROVED", Body: "ST12 stale-head approval on the new head.",
	})
	m, _ := nativeMergeDrive(t, c, fx, p.ID)
	if m.Stage != factory.MergeFailed || m.Reason != factory.MergeReasonInvalid || m.Operation.Attempts != 0 {
		t.Fatalf("stale head misadvanced: %+v", m)
	}
	if nativeTip(t, c.nativeST09Config, c.BaseBranch) != before {
		t.Fatal("stale merge moved the base")
	}
	nativeMergeReceipt(t, "stalehead-merge", m)
}

// TestNativeMergeStaleBase proves a moved base never merges: the merge
// refuses its unverified base without submitting anything.
func TestNativeMergeStaleBase(t *testing.T) {
	c := loadNativeST12(t)
	fx, n := nativeMergeSetup(t, c)
	p, _ := nativeMergeSeedPublication(t, c, fx, n)
	nativeMergeOpenRow(t, fx, p)
	base2 := nativeMergeAdvanceBranch(t, c, strings.TrimPrefix(c.BaseBranch, "refs/heads/"))
	if base2 == p.PRCreate.BaseOID {
		t.Fatal("advance did not move the base")
	}
	m, _ := nativeMergeDrive(t, c, fx, p.ID)
	if m.Stage != factory.MergeFailed || m.Reason != factory.MergeReasonInvalid || m.Operation.Attempts != 0 {
		t.Fatalf("stale base misadvanced: %+v", m)
	}
	if nativeTip(t, c.nativeST09Config, c.BaseBranch) != base2 {
		t.Fatal("base tip is not the advanced base")
	}
	nativeMergeReceipt(t, "stalebase-merge", m)
}

// TestNativeMergeWithdrawBeforeSubmit proves withdrawal before any
// submit leaves no native effect: the merge withdraws, nothing commits.
func TestNativeMergeWithdrawBeforeSubmit(t *testing.T) {
	c := loadNativeST12(t)
	ctx := context.Background()
	fx, n := nativeMergeSetup(t, c)
	p, _ := nativeMergeSeedPublication(t, c, fx, n)
	before := nativeTip(t, c.nativeST09Config, c.BaseBranch)
	nativeMergeOpenRow(t, fx, p)
	m, err := fx.db.MergeByPublication(ctx, p.ID)
	nativeMust(t, err)
	if m.Stage != factory.MergeOpen || m.Operation.Work != nil {
		t.Fatalf("seed merge misadvanced: %+v", m)
	}
	if _, err := fx.db.WithdrawDispatch(ctx, c.Repository, "test withdrawal", "soda-tester"); err != nil {
		t.Fatal(err)
	}
	m, _ = nativeMergeDrive(t, c, fx, p.ID)
	if m.Stage != factory.MergeWithdrawn || m.Operation.Attempts != 0 {
		t.Fatalf("withdrawal unconfirmed: %+v", m)
	}
	if nativeTip(t, c.nativeST09Config, c.BaseBranch) != before {
		t.Fatal("withdrawn merge moved the base")
	}
	nativeMergeReceipt(t, "withdraw-merge", m)
}

// TestNativeMergeAuthorityChanged proves the merge stays under current
// authority: a rotated merge binding waits without submitting. The
// CAS policy revision pins recorded authority, so the rotated
// bracket stays orphaned; restoring the binding lets a fresh cycle
// proceed.
func TestNativeMergeAuthorityChanged(t *testing.T) {
	c := loadNativeST12(t)
	for attempt := 0; attempt < 3; attempt++ {
		if nativeMergeAuthorityAttempt(t, c) {
			return
		}
		t.Logf("authority reseeded after host churn (attempt %d)", attempt+1)
	}
	t.Fatal("authority never observed its scenario outcome")
}

func nativeMergeAuthorityAttempt(t *testing.T, c nativeST12Config) bool {
	t.Helper()
	ctx := context.Background()
	fx, n := nativeMergeSetup(t, c)
	p, _ := nativeMergeSeedPublication(t, c, fx, n)
	before := nativeTip(t, c.nativeST09Config, c.BaseBranch)
	nativeMergeOpenRow(t, fx, p)
	m, err := fx.db.MergeByPublication(ctx, p.ID)
	nativeMust(t, err)
	if m.Stage != factory.MergeOpen || m.Operation.Work != nil {
		t.Fatalf("seed merge misadvanced: %+v", m)
	}
	policy, err := fx.db.RepositoryPolicy(ctx, c.Repository)
	nativeMust(t, err)
	original := policy.Merge.ActorID
	policy.Merge.ActorID = original + 1000
	nativeMust(t, fx.db.SaveRepositoryPolicy(ctx, policy))
	sawWait := false
	for i := 0; i < 5; i++ {
		report := fx.coord.MergePass(ctx)
		for _, entry := range report.Errors {
			if entry.Reason != "native_unavailable" {
				t.Fatalf("merge errors: %+v", report)
			}
		}
		for _, wait := range report.Waits {
			if wait.Reason == "authority_changed" {
				sawWait = true
			}
		}
	}
	m, err = fx.db.MergeByPublication(ctx, p.ID)
	nativeMust(t, err)
	if m.Stage != factory.MergeOpen || m.Operation.Attempts != 0 || m.Operation.Work != nil || !sawWait {
		t.Fatalf("rotated authority misadvanced: %+v sawWait=%v", m, sawWait)
	}
	if nativeTip(t, c.nativeST09Config, c.BaseBranch) != before {
		t.Fatal("unauthorized merge moved the base")
	}
	nativeMergeReceipt(t, "authority-wait", m)
	policy, err = fx.db.RepositoryPolicy(ctx, c.Repository)
	nativeMust(t, err)
	policy.Merge.ActorID = original
	nativeMust(t, fx.db.SaveRepositoryPolicy(ctx, policy))
	// The fresh cycle needs its own fixture: the fixture host serves
	// the seed candidate's bundle, so a second candidate on the same
	// host cannot validate.
	fx2, n2 := nativeMergeSetup(t, c)
	p2, _ := nativeMergeSeedPublication(t, c, fx2, n2)
	nativeMergeApprove(t, c, p2, "APPROVED", "ST12 proof approval.")
	nativePostStatus(t, c.nativeST09Config, p2.PRCreate.HeadOID, "verify", "success")
	if assessment := nativeMergeAssess(t, c, fx2, p2); assessment.Verdict != factory.CheckPass {
		t.Fatalf("seed assessment did not pass: %+v", assessment)
	}
	m2, _ := nativeMergeDrive(t, c, fx2, p2.ID)
	if m2.Stage == factory.MergeFailed && m2.Reason == factory.MergeReasonRefused && m2.Operation.Reason == "stale_native_revision" {
		return false
	}
	if m2.Stage != factory.MergeMerged || m2.Operation.Attempts != 1 {
		t.Fatalf("restored authority unsettled: %+v", m2)
	}
	if nativeTip(t, c.nativeST09Config, c.BaseBranch) != p2.PRCreate.HeadOID {
		t.Fatal("restored merge did not move the base")
	}
	nativeMergeReceipt(t, "authority-merge", m2)
	return true
}

// TestNativeMergeCancelBeforeSubmit proves the first cancellation
// ordering through the production client: cancel before submit pins a
// tombstone, and the later submit stays cancelled without any effect.
func TestNativeMergeCancelBeforeSubmit(t *testing.T) {
	c := loadNativeST12(t)
	ctx := context.Background()
	p, _ := nativeMergePublish(t, c, "cancelbefore")
	before := nativeTip(t, c.nativeST09Config, c.BaseBranch)
	bg := forgejo.NewServiceBackground(c.Socket, uint32(os.Getuid()), "")
	merger := forgejo.NewMerger(bg, forgejo.New(c.FountainURL), c.TokenFile)
	id := "soda-st12-cancelbefore-" + factory.NewID()
	tombstone := nativeMergeCall(t, "cancel op", func() (factory.OperationOutcome, error) {
		return merger.CancelOp(ctx, id)
	})
	if tombstone.Effect != factory.OpEffectNotCommitted || tombstone.Cancellation != factory.OpCancelCancelled {
		t.Fatalf("cancel before submit is not a tombstone: %+v", tombstone)
	}
	revision := nativeMergeIdle(t, bg)
	w := factory.MergeWork{
		MergeID: factory.NewID(), PublicationID: factory.NewID(), OperationID: id, AuthRevision: "st12-native-proof",
		HeadRef: "refs/heads/" + p.branch, BaseRef: c.BaseBranch, HeadOID: p.head, BaseOID: p.base,
		Repository: c.Repository, Issue: p.prNumber, PRNumber: p.prNumber, PRID: p.prID, IssueID: p.issueID,
		PRAuthorID: c.CreatorID, ReviewerID: c.ReviewerID, ActorID: c.ActorID,
		NativeRev: revision.Revision, NotAfter: time.Now().Add(5 * time.Minute).Unix(),
		AssessmentRevision: 1, ReviewID: 1,
	}
	// The tombstoned identity answers cancelled without evaluation; a
	// stale answer (bracket evaluated first) resubmits the same
	// identity, never a fresh one.
	var outcome factory.OperationOutcome
	for i := 0; i < 10; i++ {
		outcome = nativeMergeCall(t, "submit after cancel", func() (factory.OperationOutcome, error) {
			return merger.SubmitMerge(ctx, w)
		})
		if !nativeMergeStaleRefusal(outcome) {
			break
		}
		nativeMergeIdle(t, bg)
	}
	if outcome.Effect != factory.OpEffectNotCommitted || outcome.Cancellation != factory.OpCancelCancelled {
		t.Fatalf("submit after cancel is not cancelled: %+v", outcome)
	}
	if nativeTip(t, c.nativeST09Config, c.BaseBranch) != before {
		t.Fatal("cancelled merge moved the base")
	}
	nativeMergeReceipt(t, "cancelbefore-submit", outcome)
	nativeMergeReceipt(t, "cancelbefore-tombstone", tombstone)
}

// TestNativeMergeCancelAfterCommit proves the second cancellation
// ordering through the production client: cancel after commit reports
// too-late, the committed effect stands, and lookup confirms both.
func TestNativeMergeCancelAfterCommit(t *testing.T) {
	c := loadNativeST12(t)
	ctx := context.Background()
	p, _ := nativeMergePublish(t, c, "cancelafter")
	bg := forgejo.NewServiceBackground(c.Socket, uint32(os.Getuid()), "")
	merger := forgejo.NewMerger(bg, forgejo.New(c.FountainURL), c.TokenFile)
	submitted := nativeMergeSubmitMergeCommitted(t, merger, bg, factory.MergeWork{
		MergeID: factory.NewID(), PublicationID: factory.NewID(),
		AuthRevision: "st12-native-proof",
		HeadRef:      "refs/heads/" + p.branch, BaseRef: c.BaseBranch, HeadOID: p.head, BaseOID: p.base,
		Repository: c.Repository, Issue: p.prNumber, PRNumber: p.prNumber, PRID: p.prID, IssueID: p.issueID,
		PRAuthorID: c.CreatorID, ReviewerID: c.ReviewerID, ActorID: c.ActorID,
		AssessmentRevision: 1, ReviewID: 1,
	})
	cancelled := nativeMergeCall(t, "cancel after commit", func() (factory.OperationOutcome, error) {
		return merger.CancelOp(ctx, submitted.OperationID)
	})
	if cancelled.Effect != factory.OpEffectCommitted || cancelled.Cancellation != factory.OpCancelTooLate {
		t.Fatalf("cancel after commit is not too-late: %+v", cancelled)
	}
	looked := nativeMergeCall(t, "cancel lookup", func() (factory.OperationOutcome, error) {
		return merger.LookupOp(ctx, submitted.OperationID)
	})
	if looked.Effect != factory.OpEffectCommitted || looked.Cancellation != factory.OpCancelTooLate {
		t.Fatalf("lookup does not confirm the standing effect: %+v", looked)
	}
	if nativeTip(t, c.nativeST09Config, c.BaseBranch) != p.head {
		t.Fatal("committed merge did not move the base")
	}
	nativeMergeReceipt(t, "cancelafter-cancel", cancelled)
}

// dropOnceMerger drops the first submit reply after the native side
// already committed it: the pass must reconcile by lookup instead of
// resubmitting.
type dropOnceMerger struct {
	control.MergeExecutor
	submits int
	dropped bool
}

func (d *dropOnceMerger) SubmitMerge(ctx context.Context, w factory.MergeWork) (factory.OperationOutcome, error) {
	d.submits++
	outcome, err := d.MergeExecutor.SubmitMerge(ctx, w)
	if err != nil {
		return outcome, err
	}
	if !d.dropped {
		d.dropped = true
		return factory.OperationOutcome{}, errors.New("ST12 proof dropped the submit reply")
	}
	return outcome, nil
}

// countMerger counts submits without changing any verdict.
type countMerger struct {
	control.MergeExecutor
	submits int
}

func (l *countMerger) SubmitMerge(ctx context.Context, w factory.MergeWork) (factory.OperationOutcome, error) {
	l.submits++
	return l.MergeExecutor.SubmitMerge(ctx, w)
}

// lagOnceMerger reports the first committed outcome as
// completion-pending, simulating the native lag between effect and
// completion through the exact production wait path. The submit,
// commit and confirmation stay native; only the timing is injected,
// and only when the host never exhibits the lag itself.
type lagOnceMerger struct {
	control.MergeExecutor
	submits int
	lagged  bool
}

func (l *lagOnceMerger) lag(outcome factory.OperationOutcome) factory.OperationOutcome {
	if !l.lagged && outcome.Effect == factory.OpEffectCommitted && outcome.Completion == factory.OpCompletionComplete {
		l.lagged = true
		outcome.Completion = ""
	}
	return outcome
}

func (l *lagOnceMerger) SubmitMerge(ctx context.Context, w factory.MergeWork) (factory.OperationOutcome, error) {
	l.submits++
	outcome, err := l.MergeExecutor.SubmitMerge(ctx, w)
	if err != nil {
		return outcome, err
	}
	return l.lag(outcome), nil
}

func (l *lagOnceMerger) LookupOp(ctx context.Context, id string) (factory.OperationOutcome, error) {
	outcome, err := l.MergeExecutor.LookupOp(ctx, id)
	if err != nil {
		return outcome, err
	}
	return l.lag(outcome), nil
}

// TestNativeMergeLostReply proves a lost submit reply reconciles by
// lookup: one submit, no replay, same intent, confirmed completion.
func TestNativeMergeLostReply(t *testing.T) {
	c := loadNativeST12(t)
	for attempt := 0; attempt < 3; attempt++ {
		if nativeMergeLostReplyAttempt(t, c) {
			return
		}
		t.Logf("lost reply reseeded after host churn (attempt %d)", attempt+1)
	}
	t.Fatal("lost reply never observed its scenario outcome")
}

// nativeMergeLostReplyAttempt runs one lost-reply scenario: false means
// host churn burned the proof submit, and only then may the caller
// reseed.
func nativeMergeLostReplyAttempt(t *testing.T, c nativeST12Config) bool {
	t.Helper()
	fx, n := nativeMergeSetup(t, c)
	p, _ := nativeMergeSeedPublication(t, c, fx, n)
	nativeMergeApprove(t, c, p, "APPROVED", "ST12 proof approval.")
	nativePostStatus(t, c.nativeST09Config, p.PRCreate.HeadOID, "verify", "success")
	if assessment := nativeMergeAssess(t, c, fx, p); assessment.Verdict != factory.CheckPass {
		t.Fatalf("seed assessment did not pass: %+v", assessment)
	}
	dropped := &dropOnceMerger{MergeExecutor: fx.coord.Merges}
	fx.coord.Merges = dropped
	m, _ := nativeMergeDrive(t, c, fx, p.ID)
	if m.Stage == factory.MergeFailed && m.Reason == factory.MergeReasonRefused && m.Operation.Reason == "stale_native_revision" {
		return false
	}
	if m.Stage != factory.MergeMerged || !dropped.dropped || dropped.submits != 1 {
		t.Fatalf("lost reply unrecovered: %+v submits=%d", m, dropped.submits)
	}
	if m.Operation.Work == nil || m.Operation.Work.OperationID != factory.MergeOperationID(p.ID, 1) {
		t.Fatalf("lost reply rewrote its intent: %+v", m)
	}
	if nativeTip(t, c.nativeST09Config, c.BaseBranch) != p.PRCreate.HeadOID {
		t.Fatal("reconciled merge did not move the base")
	}
	nativeMergeReceipt(t, "lostreply-merge", m)
	return true
}

// TestNativeMergeCommittedIncomplete proves a committed ref never
// finishes bookkeeping: the merge waits for native completion under
// its single identity, then confirms without resubmitting. It first
// polls for the natural lag window; only when the host reports
// completion synchronously does it inject one lagged pass through
// the exact production wait path.
func TestNativeMergeCommittedIncomplete(t *testing.T) {
	c := loadNativeST12(t)
	for attempt := 0; attempt < 3; attempt++ {
		if nativeMergeCommittedIncompleteAttempt(t, c) {
			return
		}
		t.Logf("committed-incomplete reseeded after host churn (attempt %d)", attempt+1)
	}
	t.Fatal("committed-incomplete never observed its scenario outcome")
}

func nativeMergeCommittedIncompleteAttempt(t *testing.T, c nativeST12Config) bool {
	t.Helper()
	ctx := context.Background()
	fx, n := nativeMergeSetup(t, c)
	p, _ := nativeMergeSeedPublication(t, c, fx, n)
	nativeMergeApprove(t, c, p, "APPROVED", "ST12 proof approval.")
	nativePostStatus(t, c.nativeST09Config, p.PRCreate.HeadOID, "verify", "success")
	if assessment := nativeMergeAssess(t, c, fx, p); assessment.Verdict != factory.CheckPass {
		t.Fatalf("seed assessment did not pass: %+v", assessment)
	}
	counted := &countMerger{MergeExecutor: fx.coord.Merges}
	fx.coord.Merges = counted
	for i := 0; i < 40; i++ {
		report := fx.coord.MergePass(ctx)
		for _, entry := range report.Errors {
			if entry.Reason != "native_unavailable" {
				t.Fatalf("merge errors: %+v", report)
			}
		}
		m, err := fx.db.MergeByPublication(ctx, p.ID)
		if err != nil {
			time.Sleep(50 * time.Millisecond)
			continue
		}
		if m.Operation.Effect == factory.OpEffectCommitted && m.Operation.Completion != factory.OpCompletionComplete &&
			(m.Stage == factory.MergeOpen || m.Stage == factory.MergeFenced) {
			t.Logf("natural committed-but-incomplete window observed")
			nativeMergeReceipt(t, "committedincomplete-caught", m)
			m, _ = nativeMergeDrive(t, c, fx, p.ID)
			if m.Stage != factory.MergeMerged || counted.submits != 1 || m.Operation.Attempts != 1 ||
				m.Operation.Work == nil || m.Operation.Work.OperationID != factory.MergeOperationID(p.ID, 1) {
				t.Fatalf("caught merge misreconciled: %+v submits=%d", m, counted.submits)
			}
			nativeMergeReceipt(t, "committedincomplete-merge", m)
			return true
		}
		if m.Stage == factory.MergeFailed && m.Reason == factory.MergeReasonRefused && m.Operation.Reason == "stale_native_revision" {
			return false
		}
		if m.Stage != factory.MergeOpen && m.Stage != factory.MergeFenced {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	m, err := fx.db.MergeByPublication(ctx, p.ID)
	nativeMust(t, err)
	if m.Stage == factory.MergeFailed && m.Reason == factory.MergeReasonRefused && m.Operation.Reason == "stale_native_revision" {
		return false
	}
	if m.Stage != factory.MergeMerged {
		t.Fatalf("natural scenario misadvanced: %+v", m)
	}
	t.Logf("completion synchronous; proving the wait path by injected lag")
	fx2, n2 := nativeMergeSetup(t, c)
	p2, _ := nativeMergeSeedPublication(t, c, fx2, n2)
	nativeMergeApprove(t, c, p2, "APPROVED", "ST12 proof approval.")
	nativePostStatus(t, c.nativeST09Config, p2.PRCreate.HeadOID, "verify", "success")
	if assessment := nativeMergeAssess(t, c, fx2, p2); assessment.Verdict != factory.CheckPass {
		t.Fatalf("seed assessment did not pass: %+v", assessment)
	}
	lagged := &lagOnceMerger{MergeExecutor: fx2.coord.Merges}
	fx2.coord.Merges = lagged
	sawWait := false
	for i := 0; i < 40; i++ {
		report := fx2.coord.MergePass(ctx)
		for _, entry := range report.Errors {
			if entry.Reason != "native_unavailable" {
				t.Fatalf("merge errors: %+v", report)
			}
		}
		for _, wait := range report.Waits {
			if wait.Reason == "native_completion_pending" {
				sawWait = true
			}
		}
		m2, err := fx2.db.MergeByPublication(ctx, p2.ID)
		if err != nil {
			time.Sleep(100 * time.Millisecond)
			continue
		}
		if m2.Stage != factory.MergeOpen && m2.Stage != factory.MergeFenced {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	m2, err := fx2.db.MergeByPublication(ctx, p2.ID)
	nativeMust(t, err)
	if m2.Stage == factory.MergeFailed && m2.Reason == factory.MergeReasonRefused && m2.Operation.Reason == "stale_native_revision" {
		return false
	}
	if m2.Stage != factory.MergeMerged || !sawWait || !lagged.lagged || lagged.submits != 1 ||
		m2.Operation.Attempts != 1 || m2.Operation.Work == nil || m2.Operation.Work.OperationID != factory.MergeOperationID(p2.ID, 1) {
		t.Fatalf("lagged merge misreconciled: %+v sawWait=%v lagged=%v submits=%d", m2, sawWait, lagged.lagged, lagged.submits)
	}
	if nativeTip(t, c.nativeST09Config, c.BaseBranch) != p2.PRCreate.HeadOID {
		t.Fatal("reconciled merge did not move the base")
	}
	nativeMergeReceipt(t, "committedincomplete-merge", m2)
	return true
}

// TestNativeMergeDependantRunnable proves confirmed completion releases
// an eligible dependant: B stays blocked on A's code until A's merge
// confirms, then B becomes runnable without any further input.
func TestNativeMergeDependantRunnable(t *testing.T) {
	c := loadNativeST12(t)
	for attempt := 0; attempt < 3; attempt++ {
		if nativeMergeDependantAttempt(t, c) {
			return
		}
		t.Logf("dependant reseeded after host churn (attempt %d)", attempt+1)
	}
	t.Fatal("dependant never observed its scenario outcome")
}

// nativeMergeDependantAttempt runs one dependant scenario: false means
// host churn burned a proof submit, and only then may the caller
// reseed. The edge wires exact issues, so the publish seeding stays on
// the wired assignment and churn reseeds the whole attempt instead.
func nativeMergeDependantAttempt(t *testing.T, c nativeST12Config) bool {
	t.Helper()
	ctx := context.Background()
	fx, nA := nativeMergeSetup(t, c)
	aA := nativeMergeAssignment(t, c, fx, nA)
	nB := nativeNewCandidate(t, c.nativeST09Config)
	aB := nativeMergeAssignment(t, c, fx, nB)
	var issueA, issueB struct {
		ID int64 `json:"id"`
	}
	nativeAPI(t, c.nativeST09Config, c.CreatorTokenFile, http.MethodGet, "/api/v1/repos/"+c.Owner+"/"+c.Repo+"/issues/"+strconv.FormatInt(aA.Issue, 10), nil, &issueA)
	nativeAPI(t, c.nativeST09Config, c.CreatorTokenFile, http.MethodGet, "/api/v1/repos/"+c.Owner+"/"+c.Repo+"/issues/"+strconv.FormatInt(aB.Issue, 10), nil, &issueB)
	if issueA.ID <= 0 || issueB.ID <= 0 {
		t.Fatal("native issue identities unconfirmed")
	}
	nativeMergeInsertEdge(t, c, issueB.ID, issueA.ID)
	var edges []struct {
		ID int64 `json:"id"`
	}
	nativeAPI(t, c.nativeST09Config, c.CreatorTokenFile, http.MethodGet, "/api/v1/repos/"+c.Owner+"/"+c.Repo+"/issues/"+strconv.FormatInt(aB.Issue, 10)+"/dependencies", nil, &edges)
	if len(edges) != 1 {
		t.Fatalf("native edge not served: %+v", edges)
	}
	evidence, err := fx.coord.AcceptanceReads.ReadAcceptanceEvidence(ctx, strconv.FormatInt(c.Repository, 10), strconv.FormatInt(aB.Issue, 10), nil)
	nativeMust(t, err)
	if len(evidence.Dependencies) != 1 {
		t.Fatalf("snapshot does not carry the edge: %+v", evidence.Dependencies)
	}
	edge := evidence.Dependencies[0]
	second := factory.Acceptance{
		ID: "d" + factory.NewID()[:24], Repository: c.Repository, IssueIndex: strconv.FormatInt(aB.Issue, 10),
		Approver: c.CreatorID, NativeRev: evidence.Revision,
		TitleDigest: evidence.Issue.TitleDigest, ContentDigest: evidence.Issue.ContentDigest, ContentVersion: evidence.Issue.ContentVer,
		Predecessor: aB.Acceptance,
		Prerequisites: []factory.AcceptedPrerequisite{{
			Occurrence: edge.Occurrence, DependsOn: edge.DependsOn,
			EndpointRepo: c.Repository, EndpointIssue: aA.Issue,
			Outcome: factory.PrereqCode, PrereqAcceptance: aA.Acceptance,
		}},
	}
	if _, err := fx.coord.AdmitAcceptance(ctx, factory.NewID(), "native:"+strconv.FormatInt(c.CreatorID, 10), second); err != nil {
		t.Fatalf("dependant acceptance refused: %v", err)
	}
	blocked, _, err := fx.coord.ObserveIssueEvent(ctx, control.IntakeHint{Delivery: "st12-dep-" + factory.NewID(), Repository: c.Repository, Issue: aB.Issue})
	nativeMust(t, err)
	if blocked.Readiness != factory.ReadinessBlocked {
		t.Fatalf("dependant runnable before completion: %+v", blocked)
	}
	pA := nativeMergePublishDrive(t, c, fx, aA)
	if pA.Stage == factory.PublicationFailed &&
		(pA.Publish.Reason == "stale_native_revision" || pA.PRCreate.Reason == "stale_native_revision") {
		return false
	}
	if pA.Stage != factory.PublicationPublished {
		t.Fatalf("seed publication unsettled: %+v", pA)
	}
	nativeMergeApprove(t, c, pA, "APPROVED", "ST12 proof approval.")
	nativePostStatus(t, c.nativeST09Config, pA.PRCreate.HeadOID, "verify", "success")
	if assessment := nativeMergeAssess(t, c, fx, pA); assessment.Verdict != factory.CheckPass {
		t.Fatalf("seed assessment did not pass: %+v", assessment)
	}
	mA, _ := nativeMergeDrive(t, c, fx, pA.ID)
	if mA.Stage == factory.MergeFailed && mA.Reason == factory.MergeReasonRefused && mA.Operation.Reason == "stale_native_revision" {
		return false
	}
	if mA.Stage != factory.MergeMerged {
		t.Fatalf("endpoint merge unsettled: %+v", mA)
	}
	released, err := fx.db.IssueControl(ctx, c.Repository, aB.Issue)
	nativeMust(t, err)
	if released.Readiness != factory.ReadinessQueued || released.Reason != factory.ReasonEligible {
		t.Fatalf("confirmed completion did not release the dependant: %+v", released)
	}
	nativeMergeReceipt(t, "dependant-merge", mA)
	nativeMergeReceipt(t, "dependant-control", released)
	return true
}

// TestNativeMergeProtectionRefuses runs LAST: native branch protection
// refuses the merge without any Soda bypass, and the refusal is
// recorded exactly. It removes its protection before finishing.
func TestNativeMergeProtectionRefuses(t *testing.T) {
	c := loadNativeST12(t)
	before := nativeTip(t, c.nativeST09Config, c.BaseBranch)
	var protection struct {
		BranchName        string `json:"branch_name"`
		RequiredApprovals int64  `json:"required_approvals"`
	}
	nativeMergeAPI(t, c, http.MethodPost, "/api/v1/repos/"+c.Owner+"/"+c.Repo+"/branch_protections",
		map[string]any{"branch_name": "main", "required_approvals": 2}, &protection)
	if protection.BranchName != "main" || protection.RequiredApprovals != 2 {
		t.Fatalf("native protection unconfirmed: %+v", protection)
	}
	t.Cleanup(func() {
		nativeMergeAPI(t, c, http.MethodDelete, "/api/v1/repos/"+c.Owner+"/"+c.Repo+"/branch_protections/main", nil, nil)
	})
	for attempt := 0; attempt < 3; attempt++ {
		if nativeMergeProtectionAttempt(t, c, before) {
			return
		}
		t.Logf("protection reseeded after host churn (attempt %d)", attempt+1)
	}
	t.Fatal("protection never observed its scenario outcome")
}

// nativeMergeProtectionAttempt runs one protection scenario: false
// means host churn burned the proof submit (stale instead of the
// protection refusal), and only then may the caller reseed.
func nativeMergeProtectionAttempt(t *testing.T, c nativeST12Config, before string) bool {
	t.Helper()
	fx, n := nativeMergeSetup(t, c)
	p, _ := nativeMergeSeedPublication(t, c, fx, n)
	nativeMergeApprove(t, c, p, "APPROVED", "ST12 proof approval.")
	nativePostStatus(t, c.nativeST09Config, p.PRCreate.HeadOID, "verify", "success")
	if assessment := nativeMergeAssess(t, c, fx, p); assessment.Verdict != factory.CheckPass {
		t.Fatalf("seed assessment did not pass: %+v", assessment)
	}
	m, _ := nativeMergeDrive(t, c, fx, p.ID)
	if m.Stage == factory.MergeFailed && m.Reason == factory.MergeReasonRefused && m.Operation.Reason == "stale_native_revision" {
		return false
	}
	if m.Stage != factory.MergeFailed || m.Reason != factory.MergeReasonRefused || m.Operation.Effect != factory.OpEffectNotCommitted {
		t.Fatalf("protected merge misadvanced: %+v", m)
	}
	if m.Operation.Reason != "native_refused" || m.Operation.Attempts != 1 {
		t.Fatalf("refusal not recorded exactly: %+v", m.Operation)
	}
	if nativeTip(t, c.nativeST09Config, c.BaseBranch) != before {
		t.Fatal("refused merge moved the base")
	}
	var issue struct {
		State string `json:"state"`
	}
	nativeAPI(t, c.nativeST09Config, c.CreatorTokenFile, http.MethodGet, "/api/v1/repos/"+c.Owner+"/"+c.Repo+"/issues/"+strconv.FormatInt(p.PRNumber, 10), nil, &issue)
	if issue.State != "open" {
		t.Fatalf("refused PR did not stay open: %q", issue.State)
	}
	nativeMergeReceipt(t, "protection-merge", m)
	return true
}

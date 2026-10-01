package control

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/store"
)

// Acceptance refusal reasons. Bounded codes: controls expose why a
// decision was refused or is no longer valid, never the evidence bytes.
const (
	RefusalStaleEvidence        = "stale_evidence"
	RefusalIncompleteEvidence   = "incomplete_evidence"
	RefusalNativeBusy           = "native_busy"
	RefusalSnapshotUnavailable  = "snapshot_unavailable"
	RefusalIssueHidden          = "issue_hidden"
	RefusalObjectiveChanged     = "objective_changed"
	RefusalSourceMissing        = "source_missing"
	RefusalSourceHidden         = "source_hidden"
	RefusalSourceChanged        = "source_changed"
	RefusalEdgeHidden           = "edge_hidden"
	RefusalEdgeChanged          = "edge_changed"
	RefusalPrereqUnknown        = "prereq_route_unknown"
	RefusalPrereqStale          = "prereq_route_stale"
	RefusalCreationUnverified   = "creation_unverified"
	RefusalCreationEdited       = "creation_edited"
	RefusalCreationBlocked      = "creation_blocked"
	RefusalCreationFactory      = "creation_factory_issue"
	RefusalCreationUnauthorized = "creation_unauthorized"
)

// AcceptanceRefusal reports why an acceptance decision was refused.
type AcceptanceRefusal struct {
	Reason string `json:"reason"`
}

func (e *AcceptanceRefusal) Error() string { return "issue acceptance refused: " + e.Reason }

func refuseAcceptance(reason string) *AcceptanceRefusal { return &AcceptanceRefusal{Reason: reason} }

// AcceptanceIssueView is the bracketed native objective: exact digests,
// content version, lifecycle count and creation provenance at one idle
// native revision.
type AcceptanceIssueView struct {
	Index         string
	TitleDigest   string
	ContentDigest string
	PosterID      string
	ContentVer    int
	Lifecycle     int
	Verified      bool
	FirstCreated  bool
	Visible       bool
}

// AcceptanceComment is one bracketed native comment revision.
type AcceptanceComment struct {
	ID         string
	Digest     string
	ContentVer int
	Visible    bool
}

// AcceptanceEdge is one direct blocked-by edge occurrence.
type AcceptanceEdge struct {
	Occurrence string
	DependsOn  string
	Visible    bool
}

// AcceptanceEvidence is one revision-bound native evidence set for an
// acceptance decision: the objective, the requested selected comments and
// the complete direct edge set.
type AcceptanceEvidence struct {
	Issue        AcceptanceIssueView
	Comments     []AcceptanceComment
	Dependencies []AcceptanceEdge
	Revision     int64
}

// AcceptanceSource brackets one native evidence read between equal idle
// native revision observations. Transport failures arrive as acceptance
// refusals with a stale, incomplete, busy or unavailable reason; any other
// error reports an unusable source.
type AcceptanceSource interface {
	ReadAcceptanceEvidence(ctx context.Context, repository, issue string, commentIDs []string) (AcceptanceEvidence, error)
}

// AcceptanceReceipt is the durable outcome of one acceptance decision:
// the admitted decision, the advanced head and its chain depth.
type AcceptanceReceipt struct {
	CommandID  string `json:"command_id"`
	DecisionID string `json:"decision_id"`
	Head       string `json:"head"`
	Depth      int64  `json:"depth"`
}

// WithdrawalReceipt is the durable outcome of one acceptance withdrawal.
type WithdrawalReceipt struct {
	CommandID string `json:"command_id"`
	Decision  string `json:"decision"`
	Withdrawn bool   `json:"withdrawn"`
}

// AdmitAcceptance records a maintainer's exact-inputs acceptance after
// verifying every selected revision against a fresh bracketed snapshot:
// the observed revision must equal the bracket, the objective digests and
// version must match, every selected source must exist visibly at its
// recorded revision, and the recorded prerequisite set must equal the
// complete current native edge set. Code prerequisite routes must name an
// existing acceptance that is still the endpoint's head. A repeated
// decision ID returns the same receipt; new content under it conflicts.
func (c *Coordinator) AdmitAcceptance(ctx context.Context, commandID, principal string, decision factory.Acceptance) (AcceptanceReceipt, error) {
	bounded, stop := context.WithTimeout(ctx, 30*time.Second)
	defer stop()
	if err := decision.Validate(); err != nil {
		return AcceptanceReceipt{}, err
	}
	if decision.Initial {
		return AcceptanceReceipt{}, refuseAcceptance(RefusalCreationUnverified)
	}
	// Verify before recording: refusals are common and must not poison
	// the command ledger with unfinished entries.
	evidence, err := c.readAcceptanceEvidence(bounded, decision)
	if err != nil {
		return AcceptanceReceipt{}, err
	}
	if err := c.verifyAcceptance(bounded, decision, evidence); err != nil {
		return AcceptanceReceipt{}, err
	}
	payload, err := json.Marshal(decision)
	if err != nil {
		return AcceptanceReceipt{}, err
	}
	target := "repository/" + strconv.FormatInt(decision.Repository, 10) + "/issue/" + decision.IssueIndex + "/acceptance"
	cmd := factory.Command{
		ID: commandID, Type: factory.CommandAcceptance, Target: target, Principal: principal,
		Payload: string(payload), Digest: factory.SettingsDigest(factory.CommandAcceptance, target, string(payload)),
	}
	if err = cmd.Validate(); err != nil {
		return AcceptanceReceipt{}, err
	}
	stored, created, err := c.Store.RecordFactoryCommand(bounded, cmd, time.Now())
	if err != nil {
		return AcceptanceReceipt{}, err
	}
	if !created {
		return replayAcceptance(stored)
	}
	if err = c.Store.AdmitAcceptanceDecision(bounded, decision); err != nil {
		if errors.Is(err, store.ErrStaleRevision) {
			_ = c.Store.FinishFactoryCommand(bounded, cmd.ID, `{"error":"stale_revision"}`, time.Now())
		}
		return AcceptanceReceipt{}, err
	}
	head, err := c.Store.AcceptanceHead(bounded, decision.Repository, mustIssueIndex(decision.IssueIndex))
	if err != nil {
		return AcceptanceReceipt{}, err
	}
	depth, err := c.Store.AcceptanceDepth(bounded, decision.Repository, mustIssueIndex(decision.IssueIndex))
	if err != nil {
		return AcceptanceReceipt{}, err
	}
	receipt := AcceptanceReceipt{CommandID: cmd.ID, DecisionID: decision.ID, Head: head, Depth: depth}
	outcome, err := json.Marshal(receipt)
	if err != nil {
		return AcceptanceReceipt{}, err
	}
	if err = c.Store.FinishFactoryCommand(bounded, cmd.ID, string(outcome), time.Now()); err != nil {
		return AcceptanceReceipt{}, err
	}
	return receipt, nil
}

func replayAcceptance(stored factory.Command) (AcceptanceReceipt, error) {
	if stored.Finished == "" {
		return AcceptanceReceipt{}, ErrCommandRunning
	}
	var failure struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal([]byte(stored.Outcome), &failure); err != nil {
		return AcceptanceReceipt{}, err
	}
	if failure.Error == "stale_revision" {
		return AcceptanceReceipt{}, store.ErrStaleRevision
	}
	var receipt AcceptanceReceipt
	if err := json.Unmarshal([]byte(stored.Outcome), &receipt); err != nil {
		return AcceptanceReceipt{}, err
	}
	return receipt, nil
}

func mustIssueIndex(index string) int64 {
	value, _ := strconv.ParseInt(index, 10, 64)
	return value
}

// readAcceptanceEvidence brackets the objective, every selected source and
// resolution, and the complete edge set at one native revision.
func (c *Coordinator) readAcceptanceEvidence(ctx context.Context, decision factory.Acceptance) (AcceptanceEvidence, error) {
	if c.AcceptanceReads == nil {
		return AcceptanceEvidence{}, refuseAcceptance(RefusalSnapshotUnavailable)
	}
	seen := make(map[string]bool)
	var commentIDs []string
	for _, section := range [][]factory.SelectedSource{decision.Sources, decision.Resolutions} {
		for _, source := range section {
			if !seen[source.ID] {
				seen[source.ID] = true
				commentIDs = append(commentIDs, source.ID)
			}
		}
	}
	evidence, err := c.AcceptanceReads.ReadAcceptanceEvidence(ctx, strconv.FormatInt(decision.Repository, 10), decision.IssueIndex, commentIDs)
	if err != nil {
		var refusal *AcceptanceRefusal
		if errors.As(err, &refusal) {
			return AcceptanceEvidence{}, refusal
		}
		return AcceptanceEvidence{}, err
	}
	return evidence, nil
}

// verifyAcceptance compares one decision against its bracketed evidence.
// A stale screen, hidden source or changed revision refuses instead of
// silently accepting newer material.
func (c *Coordinator) verifyAcceptance(ctx context.Context, decision factory.Acceptance, evidence AcceptanceEvidence) error {
	if evidence.Revision < 1 || decision.NativeRev != evidence.Revision {
		return refuseAcceptance(RefusalStaleEvidence)
	}
	issue := evidence.Issue
	if issue.Index != decision.IssueIndex {
		return refuseAcceptance(RefusalIncompleteEvidence)
	}
	if !issue.Visible {
		return refuseAcceptance(RefusalIssueHidden)
	}
	if issue.TitleDigest != decision.TitleDigest || issue.ContentDigest != decision.ContentDigest || issue.ContentVer != decision.ContentVersion {
		return refuseAcceptance(RefusalObjectiveChanged)
	}
	comments := make(map[string]AcceptanceComment, len(evidence.Comments))
	for _, comment := range evidence.Comments {
		comments[comment.ID] = comment
	}
	for _, section := range [][]factory.SelectedSource{decision.Sources, decision.Resolutions} {
		for _, source := range section {
			comment, ok := comments[source.ID]
			if !ok {
				return refuseAcceptance(RefusalSourceMissing)
			}
			if !comment.Visible {
				return refuseAcceptance(RefusalSourceHidden)
			}
			if comment.Digest != source.Digest || comment.ContentVer != source.ContentVersion {
				return refuseAcceptance(RefusalSourceChanged)
			}
		}
	}
	edges := make(map[string]AcceptanceEdge, len(evidence.Dependencies))
	for _, edge := range evidence.Dependencies {
		if !edge.Visible {
			return refuseAcceptance(RefusalEdgeHidden)
		}
		edges[edge.Occurrence] = edge
	}
	if len(edges) != len(decision.Prerequisites) {
		return refuseAcceptance(RefusalEdgeChanged)
	}
	for _, prereq := range decision.Prerequisites {
		edge, ok := edges[prereq.Occurrence]
		if !ok || edge.DependsOn != prereq.DependsOn {
			return refuseAcceptance(RefusalEdgeChanged)
		}
		if prereq.Outcome == factory.PrereqCode {
			if _, err := c.Store.AcceptanceDecision(ctx, prereq.PrereqAcceptance); err != nil {
				if errors.Is(err, store.ErrNotFound) {
					return refuseAcceptance(RefusalPrereqUnknown)
				}
				return err
			}
			head, err := c.Store.AcceptanceHead(ctx, prereq.EndpointRepo, prereq.EndpointIssue)
			if err != nil || head != prereq.PrereqAcceptance {
				if err != nil && !errors.Is(err, store.ErrNotFound) {
					return err
				}
				return refuseAcceptance(RefusalPrereqStale)
			}
		}
	}
	return nil
}

// AdmitInitialAcceptance records the narrowly verified original-creation
// path: a visible issue whose creation provenance is verified and
// first-created, posted by the named creator, with creation content
// version, no lifecycle events and no prerequisites. Factory-posted,
// imported, edited or prerequisite-bearing issues require explicit human
// adoption, as do creations without standing policy or a currently
// code-write-authorized creator. The decision ID is deterministic, so a
// duplicate creation observation replays one decision; it can neither
// admit another nor overwrite a later decision. Coordinator-internal:
// intake calls this on authenticated creation observations.
func (c *Coordinator) AdmitInitialAcceptance(ctx context.Context, repository int64, issue, creatorID string, creatorWriteAuthorized bool) (factory.Acceptance, error) {
	bounded, stop := context.WithTimeout(ctx, 30*time.Second)
	defer stop()
	if repository <= 0 || issue == "" || creatorID == "" {
		return factory.Acceptance{}, errors.New("invalid initial acceptance scope")
	}
	if c.AcceptanceReads == nil {
		return factory.Acceptance{}, refuseAcceptance(RefusalSnapshotUnavailable)
	}
	evidence, err := c.AcceptanceReads.ReadAcceptanceEvidence(bounded, strconv.FormatInt(repository, 10), issue, nil)
	if err != nil {
		var refusal *AcceptanceRefusal
		if errors.As(err, &refusal) {
			return factory.Acceptance{}, refusal
		}
		return factory.Acceptance{}, err
	}
	view := evidence.Issue
	if view.Index != issue {
		return factory.Acceptance{}, refuseAcceptance(RefusalIncompleteEvidence)
	}
	if !view.Visible {
		return factory.Acceptance{}, refuseAcceptance(RefusalIssueHidden)
	}
	if !view.Verified || !view.FirstCreated || view.PosterID != creatorID {
		return factory.Acceptance{}, refuseAcceptance(RefusalCreationUnverified)
	}
	if view.ContentVer != 0 || view.Lifecycle != 0 {
		return factory.Acceptance{}, refuseAcceptance(RefusalCreationEdited)
	}
	if len(evidence.Dependencies) != 0 {
		return factory.Acceptance{}, refuseAcceptance(RefusalCreationBlocked)
	}
	policy, err := c.Store.RepositoryPolicy(bounded, repository)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return factory.Acceptance{}, refuseAcceptance(RefusalCreationUnauthorized)
		}
		return factory.Acceptance{}, err
	}
	creator, err := strconv.ParseInt(creatorID, 10, 64)
	if err != nil || creator <= 0 {
		return factory.Acceptance{}, refuseAcceptance(RefusalCreationUnverified)
	}
	for _, ref := range []factory.ActorBindingRef{policy.Publish, policy.Create, policy.Review, policy.Merge} {
		if ref.ActorID == creator {
			return factory.Acceptance{}, refuseAcceptance(RefusalCreationFactory)
		}
	}
	if !creatorWriteAuthorized {
		return factory.Acceptance{}, refuseAcceptance(RefusalCreationUnauthorized)
	}
	decision := factory.Acceptance{
		ID: factory.InitialAcceptanceID(repository, issue), Repository: repository, IssueIndex: issue,
		Approver: creator, NativeRev: evidence.Revision, Initial: true,
		TitleDigest: view.TitleDigest, ContentDigest: view.ContentDigest, ContentVersion: view.ContentVer,
	}
	if err := decision.Validate(); err != nil {
		return factory.Acceptance{}, err
	}
	if err := c.Store.AdmitAcceptanceDecision(bounded, decision); err != nil {
		return factory.Acceptance{}, err
	}
	return decision, nil
}

// AcceptanceValidity is the assessed state of one issue's current
// acceptance: the head decision, whether it is still valid, and every
// reason it is not. Reasons name changed dimensions, never evidence bytes.
type AcceptanceValidity struct {
	Acceptance *factory.Acceptance `json:"acceptance,omitempty"`
	Reasons    []string            `json:"reasons,omitempty"`
	Revision   int64               `json:"revision"`
	Withdrawn  bool                `json:"withdrawn,omitempty"`
	Valid      bool                `json:"valid"`
}

// AcceptanceStatus assesses whether the current acceptance for one native
// issue is still valid: withdrawn heads fail, and a fresh bracket must
// still match every recorded objective revision, selected source and edge
// occurrence, with code prerequisite routes still at their recorded
// revision. Restored text does not reactivate: versions and occurrences
// retain the distinction. Read-only: it records no decision.
func (c *Coordinator) AcceptanceStatus(ctx context.Context, repository int64, issue string) (AcceptanceValidity, error) {
	bounded, stop := context.WithTimeout(ctx, 30*time.Second)
	defer stop()
	index := mustIssueIndex(issue)
	if repository <= 0 || index <= 0 {
		return AcceptanceValidity{}, errors.New("invalid acceptance scope")
	}
	head, err := c.Store.AcceptanceHead(bounded, repository, index)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return AcceptanceValidity{Reasons: []string{"no_acceptance"}}, nil
		}
		return AcceptanceValidity{}, err
	}
	decision, err := c.Store.AcceptanceDecision(bounded, head)
	if err != nil {
		return AcceptanceValidity{}, err
	}
	status := AcceptanceValidity{Acceptance: &decision}
	if withdrawn, _, err := c.Store.AcceptanceWithdrawn(bounded, repository, index, head); err != nil {
		return AcceptanceValidity{}, err
	} else if withdrawn {
		status.Withdrawn = true
		status.Reasons = []string{"withdrawn"}
		return status, nil
	}
	if c.AcceptanceReads == nil {
		return AcceptanceValidity{}, refuseAcceptance(RefusalSnapshotUnavailable)
	}
	evidence, err := c.readAcceptanceEvidence(bounded, decision)
	if err != nil {
		return AcceptanceValidity{}, err
	}
	status.Revision = evidence.Revision
	status.Reasons = assessAcceptance(decision, evidence)
	// Code prerequisite routes revalidate against the endpoint's current
	// head; a changed approved revision invalidates the dependent
	// relation even when the issue body is unchanged.
	for _, prereq := range decision.Prerequisites {
		if prereq.Outcome != factory.PrereqCode {
			continue
		}
		head, err := c.Store.AcceptanceHead(bounded, prereq.EndpointRepo, prereq.EndpointIssue)
		if err != nil || head != prereq.PrereqAcceptance {
			if err != nil && !errors.Is(err, store.ErrNotFound) {
				return AcceptanceValidity{}, err
			}
			status.Reasons = append(status.Reasons, RefusalPrereqStale)
		}
	}
	status.Valid = len(status.Reasons) == 0
	if status.Valid {
		status.Reasons = nil
	}
	return status, nil
}

// assessAcceptance compares one recorded decision against fresh evidence.
// Unlike admission, the global revision is an ordering guard only: a
// changed revision alone requires fresh reads, not human readoption.
func assessAcceptance(decision factory.Acceptance, evidence AcceptanceEvidence) []string {
	var reasons []string
	issue := evidence.Issue
	if issue.Index != decision.IssueIndex || evidence.Revision < 1 {
		return []string{RefusalIncompleteEvidence}
	}
	if !issue.Visible {
		return []string{RefusalIssueHidden}
	}
	if issue.TitleDigest != decision.TitleDigest || issue.ContentDigest != decision.ContentDigest || issue.ContentVer != decision.ContentVersion {
		reasons = append(reasons, RefusalObjectiveChanged)
	}
	comments := make(map[string]AcceptanceComment, len(evidence.Comments))
	for _, comment := range evidence.Comments {
		comments[comment.ID] = comment
	}
	for _, section := range [][]factory.SelectedSource{decision.Sources, decision.Resolutions} {
		for _, source := range section {
			comment, ok := comments[source.ID]
			switch {
			case !ok:
				reasons = append(reasons, RefusalSourceMissing)
			case !comment.Visible:
				reasons = append(reasons, RefusalSourceHidden)
			case comment.Digest != source.Digest || comment.ContentVer != source.ContentVersion:
				reasons = append(reasons, RefusalSourceChanged)
			}
		}
	}
	edges := make(map[string]AcceptanceEdge, len(evidence.Dependencies))
	for _, edge := range evidence.Dependencies {
		if !edge.Visible {
			reasons = append(reasons, RefusalEdgeHidden)
			continue
		}
		edges[edge.Occurrence] = edge
	}
	if len(edges) != len(decision.Prerequisites) {
		reasons = append(reasons, RefusalEdgeChanged)
	} else {
		for _, prereq := range decision.Prerequisites {
			edge, ok := edges[prereq.Occurrence]
			if !ok || edge.DependsOn != prereq.DependsOn {
				reasons = append(reasons, RefusalEdgeChanged)
				break
			}
		}
	}
	return reasons
}

// WithdrawAcceptance records a current maintainer's explicit withdrawal of
// the head acceptance for one native issue. The withdrawer is the
// host-admitted caller, recorded for attribution. Withdrawing a superseded
// decision is stale; a later acceptance advances past the latch.
func (c *Coordinator) WithdrawAcceptance(ctx context.Context, commandID, principal string, repository, issue int64, decision string, withdrawer int64) (WithdrawalReceipt, error) {
	bounded, stop := context.WithTimeout(ctx, 30*time.Second)
	defer stop()
	if repository <= 0 || issue <= 0 || decision == "" || withdrawer <= 0 {
		return WithdrawalReceipt{}, errors.New("invalid acceptance withdrawal")
	}
	target := "repository/" + strconv.FormatInt(repository, 10) + "/issue/" + strconv.FormatInt(issue, 10) + "/withdrawal"
	payload := `{"decision":` + strconv.Quote(decision) + `}`
	cmd := factory.Command{
		ID: commandID, Type: factory.CommandWithdrawal, Target: target, Principal: principal,
		Payload: payload, Digest: factory.SettingsDigest(factory.CommandWithdrawal, target, payload),
	}
	if err := cmd.Validate(); err != nil {
		return WithdrawalReceipt{}, err
	}
	stored, created, err := c.Store.RecordFactoryCommand(bounded, cmd, time.Now())
	if err != nil {
		return WithdrawalReceipt{}, err
	}
	if !created {
		return replayWithdrawal(stored)
	}
	if err = c.Store.WithdrawAcceptanceDecision(bounded, repository, issue, decision, withdrawer); err != nil {
		if errors.Is(err, store.ErrStaleRevision) {
			_ = c.Store.FinishFactoryCommand(bounded, cmd.ID, `{"error":"stale_revision"}`, time.Now())
		}
		return WithdrawalReceipt{}, err
	}
	receipt := WithdrawalReceipt{CommandID: cmd.ID, Decision: decision, Withdrawn: true}
	outcome, err := json.Marshal(receipt)
	if err != nil {
		return WithdrawalReceipt{}, err
	}
	if err = c.Store.FinishFactoryCommand(bounded, cmd.ID, string(outcome), time.Now()); err != nil {
		return WithdrawalReceipt{}, err
	}
	return receipt, nil
}

func replayWithdrawal(stored factory.Command) (WithdrawalReceipt, error) {
	if stored.Finished == "" {
		return WithdrawalReceipt{}, ErrCommandRunning
	}
	var failure struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal([]byte(stored.Outcome), &failure); err != nil {
		return WithdrawalReceipt{}, err
	}
	if failure.Error == "stale_revision" {
		return WithdrawalReceipt{}, store.ErrStaleRevision
	}
	var receipt WithdrawalReceipt
	if err := json.Unmarshal([]byte(stored.Outcome), &receipt); err != nil {
		return WithdrawalReceipt{}, err
	}
	return receipt, nil
}

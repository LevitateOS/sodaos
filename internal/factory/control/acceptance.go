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
	ClosedUnix    int64
	Verified      bool
	FirstCreated  bool
	Visible       bool
	Closed        bool
	IsPull        bool
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
	Publications factory.PublicationWithdrawal `json:"publications"`
	Merges       factory.MergeWithdrawal       `json:"merges"`
	CommandID    string                        `json:"command_id"`
	DecisionID   string                        `json:"decision_id"`
	Head         string                        `json:"head"`
	Depth        int64                         `json:"depth"`
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
	bounded, ownsPass, cancelPass := c.readinessPass(bounded)
	defer cancelPass()
	if err := decision.Validate(); err != nil {
		return AcceptanceReceipt{}, err
	}
	if decision.Initial {
		return AcceptanceReceipt{}, refuseAcceptance(RefusalCreationUnverified)
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
	// F05-F1: a stored matching admitted command replays its retained
	// decision under current visibility without requiring unchanged
	// native evidence. Only unrecorded commands take the fresh
	// stale-screen admission path below.
	if stored, err := c.Store.FactoryCommand(bounded, cmd.ID); err == nil {
		if stored.Digest != cmd.Digest {
			return AcceptanceReceipt{}, store.ErrCommandConflict
		}
		return c.replayAdmittedAcceptance(bounded, decision, stored)
	} else if !errors.Is(err, store.ErrNotFound) {
		return AcceptanceReceipt{}, err
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
	stored, created, err := c.Store.RecordFactoryCommand(bounded, cmd, time.Now())
	if err != nil {
		return AcceptanceReceipt{}, err
	}
	if !created {
		return c.replayAdmittedAcceptance(bounded, decision, stored)
	}
	if err = c.Store.AdmitAcceptanceDecision(bounded, decision); err != nil {
		if errors.Is(err, store.ErrStaleRevision) {
			_ = c.Store.FinishFactoryCommand(bounded, cmd.ID, `{"error":"stale_revision"}`, time.Now())
		} else if errors.Is(err, store.ErrReadinessCapacity) {
			_ = c.Store.FinishFactoryCommand(bounded, cmd.ID, `{"error":"readiness_capacity"}`, time.Now())
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
	if decision.Predecessor != "" {
		receipt.Publications = c.cancelAcceptancePublications(bounded, decision.Repository, mustIssueIndex(decision.IssueIndex), decision.Predecessor)
		receipt.Merges = c.cancelAcceptanceMerges(bounded, decision.Repository, mustIssueIndex(decision.IssueIndex), decision.Predecessor)
	}
	outcome, err := json.Marshal(receipt)
	if err != nil {
		return AcceptanceReceipt{}, err
	}
	if err = c.Store.FinishFactoryCommand(bounded, cmd.ID, string(outcome), time.Now()); err != nil {
		return AcceptanceReceipt{}, err
	}
	if ownsPass {
		_, _ = c.drainReadinessWork(bounded, "")
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
	if failure.Error == "readiness_capacity" {
		return AcceptanceReceipt{}, store.ErrReadinessCapacity
	}
	var receipt AcceptanceReceipt
	if err := json.Unmarshal([]byte(stored.Outcome), &receipt); err != nil {
		return AcceptanceReceipt{}, err
	}
	return receipt, nil
}

// replayAdmittedAcceptance returns the retained receipt for an
// already-recorded acceptance command after enforcing current
// visibility. Revision and content equality are not re-required: the
// decision was admitted under exact evidence, and replay neither
// re-authorizes native material nor admits anything new.
func (c *Coordinator) replayAdmittedAcceptance(ctx context.Context, decision factory.Acceptance, stored factory.Command) (AcceptanceReceipt, error) {
	evidence, err := c.readAcceptanceEvidence(ctx, decision)
	if err != nil {
		return AcceptanceReceipt{}, err
	}
	if err := verifyAcceptanceVisibility(decision, evidence); err != nil {
		return AcceptanceReceipt{}, err
	}
	return replayAcceptance(stored)
}

// verifyAcceptanceVisibility enforces the visibility subset of
// verifyAcceptance: the bracketed issue, every selected source and
// every current edge must still be visible. Established refusal
// codes; no revision, digest, version, edge-set or prerequisite
// equality.
func verifyAcceptanceVisibility(decision factory.Acceptance, evidence AcceptanceEvidence) error {
	issue := evidence.Issue
	if issue.Index != decision.IssueIndex {
		return refuseAcceptance(RefusalIncompleteEvidence)
	}
	if !issue.Visible {
		return refuseAcceptance(RefusalIssueHidden)
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
		}
	}
	for _, edge := range evidence.Dependencies {
		if !edge.Visible {
			return refuseAcceptance(RefusalEdgeHidden)
		}
	}
	return nil
}

func mustIssueIndex(index string) int64 {
	value, _ := strconv.ParseInt(index, 10, 64)
	return value
}

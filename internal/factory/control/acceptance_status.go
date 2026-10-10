package control

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/project"
	"github.com/levitateos/sodaos/internal/store"
)

// WithdrawalReceipt is the durable outcome of one acceptance withdrawal.
type WithdrawalReceipt struct {
	Publications factory.PublicationWithdrawal `json:"publications"`
	Merges       factory.MergeWithdrawal       `json:"merges"`
	SettledRuns  int                           `json:"settled_runs"`
	PendingRuns  []FencedRun                   `json:"pending_runs,omitempty"`
	CommandID    string                        `json:"command_id"`
	Decision     string                        `json:"decision"`
	Withdrawn    bool                          `json:"withdrawn"`
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
	status, _, err := c.acceptanceStatus(ctx, repository, issue)
	return status, err
}

func (c *Coordinator) acceptanceStatus(ctx context.Context, repository int64, issue string) (AcceptanceValidity, AcceptanceEvidence, error) {
	bounded, stop := context.WithTimeout(ctx, 30*time.Second)
	defer stop()
	index := mustIssueIndex(issue)
	if repository <= 0 || index <= 0 {
		return AcceptanceValidity{}, AcceptanceEvidence{}, errors.New("invalid acceptance scope")
	}
	head, err := c.Store.AcceptanceHead(bounded, repository, index)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return AcceptanceValidity{Reasons: []string{"no_acceptance"}}, AcceptanceEvidence{}, nil
		}
		return AcceptanceValidity{}, AcceptanceEvidence{}, err
	}
	decision, err := c.Store.AcceptanceDecision(bounded, head)
	if err != nil {
		return AcceptanceValidity{}, AcceptanceEvidence{}, err
	}
	status := AcceptanceValidity{Acceptance: &decision}
	if withdrawn, _, err := c.Store.AcceptanceWithdrawn(bounded, repository, index, head); err != nil {
		return AcceptanceValidity{}, AcceptanceEvidence{}, err
	} else if withdrawn {
		status.Withdrawn = true
		status.Reasons = []string{"withdrawn"}
		return status, AcceptanceEvidence{}, nil
	}
	if c.AcceptanceReads == nil {
		return AcceptanceValidity{}, AcceptanceEvidence{}, refuseAcceptance(RefusalSnapshotUnavailable)
	}
	evidence, err := c.readAcceptanceEvidence(bounded, decision)
	if err != nil {
		return AcceptanceValidity{}, AcceptanceEvidence{}, err
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
				return AcceptanceValidity{}, AcceptanceEvidence{}, err
			}
			status.Reasons = append(status.Reasons, RefusalPrereqStale)
		}
	}
	status.Valid = len(status.Reasons) == 0
	if status.Valid {
		status.Reasons = nil
	}
	return status, evidence, nil
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
	bounded, ownsPass, cancelPass := c.readinessPass(bounded)
	defer cancelPass()
	if repository <= 0 || issue <= 0 || decision == "" || withdrawer <= 0 {
		return WithdrawalReceipt{}, errors.New("invalid acceptance withdrawal")
	}
	target := "repository/" + strconv.FormatInt(repository, 10) + "/issue/" + strconv.FormatInt(issue, 10) + "/withdrawal"
	payload := `{"decision":` + strconv.Quote(decision) + `,"withdrawer":` + strconv.FormatInt(withdrawer, 10) + `}`
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
		if stored.Finished != "" {
			return replayWithdrawal(stored)
		}
		return c.resumeWithdrawal(bounded, stored, repository, issue, decision, withdrawer, ownsPass)
	}
	if err = c.Store.WithdrawAcceptanceDecision(bounded, repository, issue, decision, withdrawer); err != nil {
		if errors.Is(err, store.ErrStaleRevision) {
			_ = c.Store.FinishFactoryCommand(bounded, cmd.ID, `{"error":"stale_revision"}`, time.Now())
		} else if errors.Is(err, store.ErrReadinessCapacity) {
			_ = c.Store.FinishFactoryCommand(bounded, cmd.ID, `{"error":"readiness_capacity"}`, time.Now())
		}
		return WithdrawalReceipt{}, err
	}
	return c.finishWithdrawal(bounded, cmd.ID, repository, issue, decision, ownsPass)
}

func (c *Coordinator) resumeWithdrawal(ctx context.Context, cmd factory.Command, repository, issue int64, decision string, withdrawer int64, ownsPass bool) (WithdrawalReceipt, error) {
	if cmd.Type != factory.CommandWithdrawal || cmd.Target != "repository/"+strconv.FormatInt(repository, 10)+"/issue/"+strconv.FormatInt(issue, 10)+"/withdrawal" {
		return WithdrawalReceipt{}, errors.New("withdrawal command scope mismatch")
	}
	var payload struct {
		Decision   string `json:"decision"`
		Withdrawer int64  `json:"withdrawer"`
	}
	if err := json.Unmarshal([]byte(cmd.Payload), &payload); err != nil || payload.Decision != decision || payload.Withdrawer != withdrawer {
		return WithdrawalReceipt{}, errors.New("withdrawal command payload mismatch")
	}
	withdrawn, _, err := c.Store.AcceptanceWithdrawn(ctx, repository, issue, decision)
	if err != nil {
		return WithdrawalReceipt{}, err
	}
	if !withdrawn {
		err = c.Store.WithdrawAcceptanceDecision(ctx, repository, issue, decision, withdrawer)
		if err != nil {
			if errors.Is(err, store.ErrStaleRevision) || errors.Is(err, store.ErrReadinessCapacity) {
				code := "stale_revision"
				if errors.Is(err, store.ErrReadinessCapacity) {
					code = "readiness_capacity"
				}
				_ = c.Store.FinishFactoryCommand(ctx, cmd.ID, `{"error":"`+code+`"}`, time.Now())
			}
			return WithdrawalReceipt{}, err
		}
	}
	return c.finishWithdrawal(ctx, cmd.ID, repository, issue, decision, ownsPass)
}

func (c *Coordinator) recoverAbandonedWithdrawal(ctx context.Context, cmd factory.Command) error {
	bounded, stop := context.WithTimeout(ctx, 30*time.Second)
	defer stop()
	var payload struct {
		Decision   string `json:"decision"`
		Withdrawer int64  `json:"withdrawer"`
	}
	if err := json.Unmarshal([]byte(cmd.Payload), &payload); err != nil || payload.Decision == "" || payload.Withdrawer <= 0 {
		return errors.New("invalid abandoned withdrawal command")
	}
	var repository, issue int64
	if _, err := fmt.Sscanf(cmd.Target, "repository/%d/issue/%d/withdrawal", &repository, &issue); err != nil || repository <= 0 || issue <= 0 || cmd.Target != "repository/"+strconv.FormatInt(repository, 10)+"/issue/"+strconv.FormatInt(issue, 10)+"/withdrawal" {
		return errors.New("invalid abandoned withdrawal target")
	}
	_, err := c.resumeWithdrawal(bounded, cmd, repository, issue, payload.Decision, payload.Withdrawer, false)
	if errors.Is(err, ErrCommandRunning) || errors.Is(err, store.ErrStaleRevision) || errors.Is(err, store.ErrReadinessCapacity) {
		return nil
	}
	return err
}

func (c *Coordinator) finishWithdrawal(ctx context.Context, commandID string, repository, issue int64, decision string, ownsPass bool) (WithdrawalReceipt, error) {
	receipt := WithdrawalReceipt{
		CommandID: commandID, Decision: decision, Withdrawn: true,
		Publications: c.cancelAcceptancePublications(ctx, repository, issue, decision),
		Merges:       c.cancelAcceptanceMerges(ctx, repository, issue, decision),
	}
	cursor := ""
	for {
		page, next, err := c.Store.AcceptanceAssignmentsAfter(ctx, repository, issue, decision, cursor, 128)
		if err != nil {
			return receipt, err
		}
		for _, assignment := range page {
			if err = assignment.Validate(); err != nil {
				return receipt, err
			}
			for _, runID := range assignment.RunHistory {
				run, err := c.Store.FactoryRun(ctx, runID)
				if err != nil {
					return receipt, err
				}
				view, err := c.Store.FactoryRunView(ctx, runID)
				if err != nil {
					return receipt, err
				}
				if view.Repository != repository || view.Issue != issue || view.Attempt != assignment.ID || run.ProjectID != assignment.ProjectID {
					return receipt, errors.New("withdrawal run scope mismatch")
				}
				if !run.Reconciled && (c.Host == nil || c.Broker == nil) {
					return receipt, errors.New("withdrawal stop dependencies unavailable")
				}
				stopped := c.settleRun(ctx, run)
				if !stopped.Confirmed {
					receipt.PendingRuns = []FencedRun{{ID: runID, Reason: stopped.Reason}}
					return receipt, ErrCommandRunning
				}
				receipt.SettledRuns++
				if current, loadErr := c.Store.FactoryRun(ctx, runID); loadErr != nil {
					return receipt, loadErr
				} else {
					latest, lookupErr := c.Store.AssignmentByRun(ctx, runID)
					if lookupErr == nil && latest.Stage == factory.AssignmentAssigned {
						if c.Host == nil {
							receipt.PendingRuns = []FencedRun{{ID: runID, Reason: "accounting output unavailable"}}
							return receipt, ErrCommandRunning
						}
						state, inspectErr := c.Host.FactoryInspect(ctx, project.FactoryInspect{Project: current.ProjectID, ID: runID})
						if inspectErr != nil {
							receipt.PendingRuns = []FencedRun{{ID: runID, Reason: "run inspection unconfirmed"}}
							return receipt, ErrCommandRunning
						}
						if _, accounted := AccountSettledRun(ctx, c.Store, current, state.Output, time.Now()); !accounted {
							receipt.PendingRuns = []FencedRun{{ID: runID, Reason: "run accounting pending"}}
							return receipt, ErrCommandRunning
						}
					} else if lookupErr != nil && !errors.Is(lookupErr, sql.ErrNoRows) {
						return receipt, lookupErr
					}
				}
			}
		}
		if len(page) < 128 {
			break
		}
		if next == cursor {
			return receipt, errors.New("acceptance assignment cursor stalled")
		}
		cursor = next
	}
	outcome, err := json.Marshal(receipt)
	if err != nil {
		return WithdrawalReceipt{}, err
	}
	if err = c.Store.FinishFactoryCommand(ctx, commandID, string(outcome), time.Now()); err != nil {
		return WithdrawalReceipt{}, err
	}
	if ownsPass {
		_, _ = c.drainReadinessWork(ctx, "")
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
	if failure.Error == "readiness_capacity" {
		return WithdrawalReceipt{}, store.ErrReadinessCapacity
	}
	var receipt WithdrawalReceipt
	if err := json.Unmarshal([]byte(stored.Outcome), &receipt); err != nil {
		return WithdrawalReceipt{}, err
	}
	return receipt, nil
}

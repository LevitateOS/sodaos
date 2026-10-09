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

// WithdrawalReceipt is the durable outcome of one acceptance withdrawal.
type WithdrawalReceipt struct {
	Publications factory.PublicationWithdrawal `json:"publications"`
	Merges       factory.MergeWithdrawal       `json:"merges"`
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
		} else if errors.Is(err, store.ErrReadinessCapacity) {
			_ = c.Store.FinishFactoryCommand(bounded, cmd.ID, `{"error":"readiness_capacity"}`, time.Now())
		}
		return WithdrawalReceipt{}, err
	}
	receipt := WithdrawalReceipt{
		CommandID: cmd.ID, Decision: decision, Withdrawn: true,
		Publications: c.cancelAcceptancePublications(bounded, repository, issue, decision),
		Merges:       c.cancelAcceptanceMerges(bounded, repository, issue, decision),
	}
	outcome, err := json.Marshal(receipt)
	if err != nil {
		return WithdrawalReceipt{}, err
	}
	if err = c.Store.FinishFactoryCommand(bounded, cmd.ID, string(outcome), time.Now()); err != nil {
		return WithdrawalReceipt{}, err
	}
	if ownsPass {
		_, _ = c.drainReadinessWork(bounded, "")
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

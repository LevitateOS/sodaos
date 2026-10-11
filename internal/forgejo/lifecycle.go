package forgejo

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"unicode/utf8"

	sdk "forgejo.org/extension-sdk"
	"github.com/levitateos/sodaos/internal/factory"
)

// Optional lifecycle operation kinds. These strings must equal the
// extension SDK's OperationKindLifecycleIssue/Review/Note once lane C
// merges the native packet; the SDK is the canonical owner and the Soda
// copy here keeps the writer compiling against the current SDK.
const (
	LifecycleKindIssue  = "issue.lifecycle.apply"
	LifecycleKindReview = "pull_request.review.request"
	LifecycleKindNote   = "issue.comment.publish"
)

// Lifecycle denial reasons: native outcomes that stop further attempts
// rather than admitting the single retry.
const (
	lifecycleReasonAuthorityLost         = "authority_lost"
	lifecycleReasonCancelledBeforeSubmit = "cancelled_before_submit"
	lifecycleReasonCancelledBeforeAdmit  = "cancelled_before_admission"
	lifecycleEffectCommitted             = "committed"
	lifecycleEffectNotCommitted          = "not_committed"
)

// LifecycleIssuePayload builds one exact-issue label/assignee payload.
func LifecycleIssuePayload(issue, label, assignee int64) ([]byte, error) {
	if issue <= 0 || label < 0 || assignee < 0 || (label == 0 && assignee == 0) {
		return nil, ErrInvalidResponse
	}
	return json.Marshal(struct {
		IssueID    int64 `json:"issue_id"`
		LabelID    int64 `json:"label_id,omitempty"`
		AssigneeID int64 `json:"assignee_id,omitempty"`
	}{IssueID: issue, LabelID: label, AssigneeID: assignee})
}

// LifecycleReviewPayload builds one exact-PR reviewer-request payload.
func LifecycleReviewPayload(issue int64, reviewers []int64) ([]byte, error) {
	if issue <= 0 || len(reviewers) == 0 || len(reviewers) > factory.MaxLifecycleReviewers {
		return nil, ErrInvalidResponse
	}
	seen := map[int64]bool{}
	for _, id := range reviewers {
		if id <= 0 || seen[id] {
			return nil, ErrInvalidResponse
		}
		seen[id] = true
	}
	return json.Marshal(struct {
		IssueID     int64   `json:"issue_id"`
		ReviewerIDs []int64 `json:"reviewer_ids"`
	}{IssueID: issue, ReviewerIDs: reviewers})
}

// LifecycleNotePayload builds one bounded completion-note payload.
func LifecycleNotePayload(issue int64, content, mergeRef string) ([]byte, error) {
	if issue <= 0 || mergeRef == "" || len(mergeRef) > 512 {
		return nil, ErrInvalidResponse
	}
	if strings.TrimSpace(content) == "" || utf8.RuneCountInString(content) > factory.MaxCompletionNoteChars || strings.ContainsRune(content, 0) || !utf8.ValidString(content) {
		return nil, ErrInvalidResponse
	}
	return json.Marshal(struct {
		IssueID  int64  `json:"issue_id"`
		Content  string `json:"content"`
		MergeRef string `json:"merge_ref"`
	}{IssueID: issue, Content: content, MergeRef: mergeRef})
}

// LifecycleWriter submits bounded optional lifecycle operations through
// the shared service admission. Its secret stays outside Projects and
// agent processes, like the other native writers.
type LifecycleWriter struct {
	background sdk.BackgroundClient
	tokenFile  string
}

// NewLifecycleWriter builds a lifecycle writer over the shared service
// admission and its enrolled native actor credential file.
func NewLifecycleWriter(background sdk.BackgroundClient, tokenFile string) *LifecycleWriter {
	return &LifecycleWriter{background: background, tokenFile: tokenFile}
}

// SubmitAction sends one durable lifecycle operation under its persisted
// identity. An uncertain transport outcome is looked up under that same
// identity, never recreated under a new one.
func (w *LifecycleWriter) SubmitAction(ctx context.Context, operationID string, actorID, repositoryID int64, kind, authRevision string, nativeRev, notAfter int64, payload []byte) (sdk.OperationRecord, error) {
	if w == nil || w.background == nil || w.tokenFile == "" {
		return sdk.OperationRecord{}, ErrUnavailable
	}
	if operationID == "" || actorID <= 0 || repositoryID <= 0 || authRevision == "" || nativeRev < 1 || notAfter <= 0 || len(payload) == 0 {
		return sdk.OperationRecord{}, ErrInvalidResponse
	}
	switch kind {
	case LifecycleKindIssue, LifecycleKindReview, LifecycleKindNote:
	default:
		return sdk.OperationRecord{}, ErrInvalidResponse
	}
	intent := sdk.OperationIntent{
		OperationID: operationID, ActorID: strconv.FormatInt(actorID, 10),
		RepositoryID: strconv.FormatInt(repositoryID, 10), Kind: kind,
		AuthorizationRevision: authRevision, ExpectedNativeRevision: nativeRev,
		NotAfter: notAfter, Payload: payload,
	}
	record, err := w.background.SubmitOperation(ctx, sdk.CredentialFile(w.tokenFile), intent)
	if err != nil {
		lookup, readErr := w.background.GetOperation(ctx, operationID)
		if readErr != nil || lookup.Record == nil {
			return sdk.OperationRecord{}, err
		}
		record = *lookup.Record
	}
	return record, nil
}

// LookupAction reads one installation-owned lifecycle operation without
// mutating anything. Absence reports not_observed; it never proves that
// an earlier request cannot arrive.
func (w *LifecycleWriter) LookupAction(ctx context.Context, operationID string) (sdk.OperationLookup, error) {
	if w == nil || w.background == nil || operationID == "" {
		return sdk.OperationLookup{}, ErrUnavailable
	}
	return w.background.GetOperation(ctx, operationID)
}

// ClassifyLifecycleOutcome maps one native record to its per-target
// journal state: committed effects deliver; authority loss and ordered
// cancellation deny (no retry); other definitive refusals fail (one
// retry allowed after re-observation); anything else stays uncertain.
func ClassifyLifecycleOutcome(record sdk.OperationRecord) string {
	switch record.EffectState {
	case lifecycleEffectCommitted:
		return factory.LifecycleResultDelivered
	case lifecycleEffectNotCommitted:
		switch record.ReasonCode {
		case lifecycleReasonAuthorityLost, lifecycleReasonCancelledBeforeSubmit, lifecycleReasonCancelledBeforeAdmit:
			return factory.LifecycleResultDenied
		default:
			return factory.LifecycleResultFailed
		}
	default:
		return factory.LifecycleResultUncertain
	}
}

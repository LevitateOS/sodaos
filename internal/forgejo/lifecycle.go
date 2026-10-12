package forgejo

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"unicode/utf8"

	sdk "forgejo.org/extension-sdk"
	"github.com/levitateos/sodaos/internal/config"
	"github.com/levitateos/sodaos/internal/factory"
)

// Optional lifecycle operation kinds, aliased from the factory domain so
// event-to-kind mapping has one L-owned source. The extension SDK remains
// the canonical owner of these strings.
const (
	LifecycleKindIssue  = factory.LifecycleKindIssue
	LifecycleKindReview = factory.LifecycleKindReview
	LifecycleKindNote   = factory.LifecycleKindNote
)

// Lifecycle native effect states mirror the factory operation effects so
// both layers classify the same record identically.
const (
	lifecycleEffectCommitted    = factory.OpEffectCommitted
	lifecycleEffectNotCommitted = factory.OpEffectNotCommitted
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
		if factory.LifecycleDenyReason(record.ReasonCode) {
			return factory.LifecycleResultDenied
		}
		return factory.LifecycleResultFailed
	default:
		return factory.LifecycleResultUncertain
	}
}

// LifecycleExecutor adapts the lifecycle writer to the coordinator's
// narrow executor surface: enrolled-actor resolution, current revision
// observation, closed-target resolution for completion notes, and one
// conditional submit with lost-reply lookup per attempt. It implements
// the control LifecycleExecutor interface by matching methods; the
// control package owns that interface so the forgejo layer never
// imports it.
type LifecycleExecutor struct {
	background sdk.BackgroundClient
	rest       *Client
	tokenFile  string
}

// NewLifecycleExecutor builds a lifecycle executor over the shared
// service admission, the REST identity client and its enrolled native
// actor credential file.
func NewLifecycleExecutor(background sdk.BackgroundClient, rest *Client, tokenFile string) *LifecycleExecutor {
	return &LifecycleExecutor{background: background, rest: rest, tokenFile: tokenFile}
}

// EnrolledActor resolves the native actor behind the enrolled
// lifecycle credential. Configuration may declare the intended actor,
// but only this resolution proves enrollment.
func (e *LifecycleExecutor) EnrolledActor(ctx context.Context) (int64, error) {
	if e == nil || e.rest == nil || e.tokenFile == "" {
		return 0, ErrUnavailable
	}
	token, err := config.Secret(e.tokenFile)
	if err != nil {
		return 0, ErrUnavailable
	}
	current, err := loadActor(ctx, e.rest, token)
	if err != nil {
		return 0, err
	}
	if current.ID <= 0 {
		return 0, ErrInvalidResponse
	}
	return current.ID, nil
}

// ObserveLifecycleRevision observes the current native revision for one
// repository. Busy revisions are returned, not refused: a busy submit
// reconciles uncertain and native guards own admission.
func (e *LifecycleExecutor) ObserveLifecycleRevision(ctx context.Context, repositoryID int64) (int64, error) {
	if e == nil || e.background == nil || repositoryID <= 0 {
		return 0, ErrUnavailable
	}
	obs, err := e.background.ReadNativeRevision(ctx)
	if err != nil {
		return 0, err
	}
	if obs.Revision < 1 {
		return 0, ErrInvalidResponse
	}
	return obs.Revision, nil
}

// ResolveNoteTarget revalidates the completion-note target at submit
// time: the original issue index must resolve to a visible closed
// non-pull issue in the same repository. The merge confirmation proves
// closure; this read binds the exact native issue ID under current
// authority.
func (e *LifecycleExecutor) ResolveNoteTarget(ctx context.Context, repositoryID, issueIndex int64) (int64, error) {
	if e == nil || e.background == nil || e.tokenFile == "" || repositoryID <= 0 || issueIndex <= 0 {
		return 0, ErrUnavailable
	}
	actor, err := e.EnrolledActor(ctx)
	if err != nil {
		return 0, err
	}
	reader := &BackgroundSnapshotReader{Client: e.background, ActorID: strconv.FormatInt(actor, 10)}
	snapshot, err := BracketedRead(ctx, reader, sdk.CredentialFile(e.tokenFile), SnapshotRequest{
		RepositoryID: strconv.FormatInt(repositoryID, 10), IssueIndex: strconv.FormatInt(issueIndex, 10), Families: []SnapshotFamily{FamilyIssue},
	})
	if err != nil {
		return 0, err
	}
	issue := snapshot.Issue
	if issue == nil || !issue.Visible || !issue.Complete || issue.IsPull || !issue.IsClosed || issue.Index != strconv.FormatInt(issueIndex, 10) {
		return 0, ErrInvalidResponse
	}
	if snapshot.RepositoryID != strconv.FormatInt(repositoryID, 10) {
		return 0, ErrInvalidResponse
	}
	id, err := strconv.ParseInt(issue.ID, 10, 64)
	if err != nil || id <= 0 {
		return 0, ErrInvalidResponse
	}
	return id, nil
}

// lifecycleWorkPayload builds the exact native payload for one submit.
func lifecycleWorkPayload(work factory.LifecycleWork) ([]byte, error) {
	switch work.Event {
	case factory.LifecycleEventIssueCreated:
		return LifecycleIssuePayload(work.IssueID, work.LabelID, work.AssigneeID)
	case factory.LifecycleEventPRCreated:
		return LifecycleReviewPayload(work.IssueID, work.ReviewerIDs)
	case factory.LifecycleEventVerifiedMerge:
		return LifecycleNotePayload(work.IssueID, work.Note, work.MergeRef)
	default:
		return nil, ErrInvalidResponse
	}
}

// lifecycleOutcome maps one bound native record to the reconciled
// operation answer. Records under another identity are refused, never
// adopted.
func lifecycleOutcome(operationID string, record sdk.OperationRecord) (factory.OperationOutcome, error) {
	if record.OperationID != operationID {
		return factory.OperationOutcome{}, ErrInvalidResponse
	}
	effect := factory.OpEffectIndeterminate
	switch record.EffectState {
	case lifecycleEffectCommitted:
		effect = factory.OpEffectCommitted
	case lifecycleEffectNotCommitted:
		effect = factory.OpEffectNotCommitted
	}
	actor, _ := strconv.ParseInt(record.ActorID, 10, 64)
	repository, _ := strconv.ParseInt(record.RepositoryID, 10, 64)
	return factory.OperationOutcome{
		OperationID: operationID, InstallationID: record.InstallationID, Kind: record.Kind,
		ActorID: actor, RepositoryID: repository, Receipt: []byte(record.Receipt),
		Effect: effect, Cancellation: record.CancellationStatus, Completion: record.CompletionState, Reason: record.ReasonCode,
	}, nil
}

// SubmitLifecycle sends one durable lifecycle operation under its
// persisted identity. An uncertain transport outcome is looked up
// under that same identity, never recreated under a new one.
func (e *LifecycleExecutor) SubmitLifecycle(ctx context.Context, work factory.LifecycleWork) (factory.OperationOutcome, error) {
	if e == nil || e.background == nil || e.tokenFile == "" {
		return factory.OperationOutcome{}, ErrUnavailable
	}
	payload, err := lifecycleWorkPayload(work)
	if err != nil {
		return factory.OperationOutcome{}, err
	}
	writer := NewLifecycleWriter(e.background, e.tokenFile)
	record, err := writer.SubmitAction(ctx, work.OperationID, work.ActorID, work.RepositoryID, factory.LifecycleEventKind(work.Event), work.AuthRevision, work.NativeRev, work.NotAfter, payload)
	if err != nil {
		return factory.OperationOutcome{}, err
	}
	return lifecycleOutcome(work.OperationID, record)
}

// LookupLifecycle reads one installation-owned lifecycle operation
// without mutating anything. Absence reports not observed; it never
// proves that an earlier request cannot arrive.
func (e *LifecycleExecutor) LookupLifecycle(ctx context.Context, operationID string) (factory.OperationOutcome, error) {
	if e == nil || e.background == nil || operationID == "" {
		return factory.OperationOutcome{}, ErrUnavailable
	}
	writer := NewLifecycleWriter(e.background, e.tokenFile)
	lookup, err := writer.LookupAction(ctx, operationID)
	if err != nil {
		return factory.OperationOutcome{}, err
	}
	if lookup.OperationID != operationID {
		return factory.OperationOutcome{}, ErrInvalidResponse
	}
	if lookup.Record == nil || lookup.Status == sdk.BackgroundOutcomeNotObserved {
		return factory.OperationOutcome{OperationID: operationID, NotObserved: true}, nil
	}
	return lifecycleOutcome(operationID, *lookup.Record)
}

package publish

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	extensions "forgejo.org/extension-sdk"

	"github.com/levitateos/sodaos/internal/factory"
)

// BackgroundOperations is the publisher's narrow native-operation
// transport: bracketed revision reads plus submit, lookup and cancel for
// the installation's own conditional operations, and the current
// installation admission bound into push headers. One installation holds
// one current service admission: the transport owns its bootstrap and
// rebinds after revocation, so readers and publishers never split it.
type BackgroundOperations interface {
	ReadNativeRevision(ctx context.Context) (extensions.NativeRevisionObservation, error)
	SubmitOperation(ctx context.Context, credential extensions.CredentialFile, intent extensions.OperationIntent) (extensions.OperationRecord, error)
	GetOperation(ctx context.Context, operationID string) (extensions.OperationLookup, error)
	CancelOperation(ctx context.Context, operationID string) (extensions.OperationRecord, error)
	// PublishPushEnv binds one registered operation to its push headers
	// under the current admission. Rebind refreshes a revoked admission
	// first; a push that failed before launching retries under the new
	// admission only after a lookup still shows it pending.
	PublishPushEnv(ctx context.Context, operationID string, rebind bool) ([]string, error)
}

// Refusal is a terminal verdict: the operation will not proceed under
// its recorded identity. The controller records it and fails the
// publication instead of retrying an identical call.
type Refusal struct{ Reason string }

func (e *Refusal) Error() string { return "publication refused: " + e.Reason }

// Wait reports that no verdict exists yet: the call reached no native
// decision and the publication stays open for a later pass. It never
// advances the stage.
type Wait struct{ Reason string }

func (e *Wait) Error() string { return "publication waits: " + e.Reason }

// Operation intent lifetimes. Submit deadlines stay well inside the
// native one-hour maximum; host time authorizes them at submit.
const operationLifetime = 10 * time.Minute

// ObserveForPublish brackets one idle native revision and reads the exact
// target and comparison tips behind it. The tips are observed between two
// equal idle observations; a moved or busy revision, a missing comparison
// or an ambiguous advertisement refuses instead of binding a stale read.
// An empty target tip means the branch is absent.
func (c Config) ObserveForPublish(ctx context.Context, ops BackgroundOperations, targetRef, comparisonRef string) (factory.PublicationObservation, error) {
	var empty factory.PublicationObservation
	if err := c.Validate(); err != nil {
		return empty, err
	}
	if !validPublishRef(targetRef) || !validPublishRef(comparisonRef) || targetRef == comparisonRef {
		return empty, &Refusal{Reason: "invalid_refs"}
	}
	if ops == nil {
		return empty, &Wait{Reason: "operations_unavailable"}
	}
	first, err := bracketRevision(ctx, ops)
	if err != nil {
		return empty, err
	}
	target, comparison, err := c.observeTips(ctx, targetRef, comparisonRef)
	if err != nil {
		return empty, err
	}
	second, err := ops.ReadNativeRevision(ctx)
	if err != nil {
		return empty, mapRevisionError(err)
	}
	if !second.Idle || second.Revision != first {
		return empty, &Wait{Reason: "revision_moved"}
	}
	return factory.PublicationObservation{
		TargetRef: targetRef, TargetTip: target, Comparison: comparison,
		NativeRev: first, ObservedUnix: time.Now().Unix(),
	}, nil
}

func validPublishRef(ref string) bool {
	name, ok := strings.CutPrefix(ref, "refs/heads/")
	if !ok || name == "" || len(ref) > 512 || ref != strings.TrimSpace(ref) {
		return false
	}
	return !strings.Contains(ref, "..") && !strings.ContainsAny(ref, " ~^:?*\\")
}

func bracketRevision(ctx context.Context, ops BackgroundOperations) (int64, error) {
	observation, err := ops.ReadNativeRevision(ctx)
	if err != nil {
		return 0, mapRevisionError(err)
	}
	if !observation.Idle {
		return 0, &Wait{Reason: "native_busy"}
	}
	if observation.Revision < 1 {
		return 0, errors.New("invalid native revision observation")
	}
	return observation.Revision, nil
}

func mapRevisionError(err error) error {
	var wait *Wait
	if errors.As(err, &wait) {
		return err
	}
	var refusal *Refusal
	if errors.As(err, &refusal) {
		return err
	}
	return err
}

func (c Config) observeTips(ctx context.Context, targetRef, comparisonRef string) (target, comparison string, err error) {
	git, cleanup, err := c.sourceRepository(ctx)
	if err != nil {
		return "", "", err
	}
	defer cleanup()
	target, err = observeTip(ctx, git, c.Remote, targetRef, true)
	if err != nil {
		return "", "", err
	}
	comparison, err = observeTip(ctx, git, c.Remote, comparisonRef, false)
	if err != nil {
		return "", "", err
	}
	return target, comparison, nil
}

// observeTip reads one exact advertised tip. A missing target reads as
// empty for branch creation; any other absence, ambiguity or malformed
// advertisement refuses.
func observeTip(ctx context.Context, git *repository, remote, ref string, absentOK bool) (string, error) {
	out, err := git.run(ctx, "ls-remote", remote, ref)
	if err != nil {
		return "", err
	}
	text := strings.TrimSpace(string(out))
	if text == "" {
		if absentOK {
			return "", nil
		}
		return "", &Refusal{Reason: "comparison_missing"}
	}
	lines := strings.Split(text, "\n")
	if len(lines) != 1 {
		return "", &Refusal{Reason: "ambiguous_refs"}
	}
	fields := strings.Fields(lines[0])
	if len(fields) != 2 || fields[1] != ref || !factory.ValidCommit(fields[0]) {
		return "", &Refusal{Reason: "ambiguous_refs"}
	}
	return fields[0], nil
}

// PublishOpIntent is one branch-publication submit: the persisted
// operation identity, the policy-bound actor and authority, the bracketed
// observation and the exact ref tuple. A nonzero correction binds an
// update to its existing PR; zero means initial publication.
type PublishOpIntent struct {
	Credential            extensions.CredentialFile
	OperationID           string
	AuthRevision          string
	Ref                   string
	ExpectedOld           string
	NewOID                string
	ComparisonRef         string
	ExpectedComparisonOID string
	ActorID               int64
	RepositoryID          int64
	ExpectedRevision      int64
	NotAfter              int64
	CorrectionNumber      int64
	CorrectionAuthor      int64
}

// SubmitPublish submits one branch-publication intent under its persisted
// identity and reconciles the answer. A lost reply reconciles by lookup:
// the operation is adopted, never resubmitted blindly. A terminal record
// is adopted as well; only not_observed permits a later resubmit.
func (c Config) SubmitPublish(ctx context.Context, ops BackgroundOperations, in PublishOpIntent) (factory.OperationOutcome, error) {
	var empty factory.OperationOutcome
	payload, err := publishPayload(in)
	if err != nil {
		return empty, err
	}
	intent := extensions.OperationIntent{
		OperationID: in.OperationID, ActorID: strconv.FormatInt(in.ActorID, 10),
		RepositoryID: strconv.FormatInt(in.RepositoryID, 10), Kind: factory.OpRefPublish,
		AuthorizationRevision: in.AuthRevision, ExpectedNativeRevision: in.ExpectedRevision,
		NotAfter: in.NotAfter, Payload: payload,
	}
	return submitOperation(ctx, ops, in.Credential, intent)
}

func publishPayload(in PublishOpIntent) (json.RawMessage, error) {
	if !factory.ValidPublicationOperationID(in.OperationID) {
		return nil, &Refusal{Reason: "invalid_operation"}
	}
	if in.ActorID <= 0 || in.RepositoryID <= 0 {
		return nil, &Refusal{Reason: "invalid_operation"}
	}
	if in.AuthRevision == "" || len(in.AuthRevision) > 512 {
		return nil, &Refusal{Reason: "invalid_operation"}
	}
	if in.ExpectedRevision < 1 || in.NotAfter <= time.Now().Unix() {
		return nil, &Wait{Reason: "observation_expired"}
	}
	payload := extensions.PublishPayload{
		Ref: in.Ref, ExpectedOld: in.ExpectedOld, NewOID: in.NewOID,
		ComparisonRef: in.ComparisonRef, ExpectedComparisonOID: in.ExpectedComparisonOID,
	}
	if in.CorrectionNumber != 0 || in.CorrectionAuthor != 0 {
		payload.Correction = &extensions.PublishCorrection{Number: in.CorrectionNumber, ExpectedAuthorID: in.CorrectionAuthor}
	}
	if err := extensions.ValidatePublishPayload(payload); err != nil {
		return nil, &Refusal{Reason: "invalid_refs"}
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return raw, nil
}

// PRCreateOpIntent is one PR-creation submit: the persisted operation
// identity, the policy-bound actor and authority, the bracketed
// observation and the exact head/base snapshot with final title and body.
type PRCreateOpIntent struct {
	Credential       extensions.CredentialFile
	OperationID      string
	AuthRevision     string
	HeadRef          string
	BaseRef          string
	ExpectedHead     string
	ExpectedBase     string
	Title            string
	Body             string
	ActorID          int64
	RepositoryID     int64
	ExpectedRevision int64
	NotAfter         int64
}

// SubmitPRCreate submits one PR-creation intent under its persisted
// identity with the same lost-reply reconciliation as SubmitPublish.
func (c Config) SubmitPRCreate(ctx context.Context, ops BackgroundOperations, in PRCreateOpIntent) (factory.OperationOutcome, error) {
	var empty factory.OperationOutcome
	payload, err := prCreatePayload(in)
	if err != nil {
		return empty, err
	}
	intent := extensions.OperationIntent{
		OperationID: in.OperationID, ActorID: strconv.FormatInt(in.ActorID, 10),
		RepositoryID: strconv.FormatInt(in.RepositoryID, 10), Kind: factory.OpPRCreate,
		AuthorizationRevision: in.AuthRevision, ExpectedNativeRevision: in.ExpectedRevision,
		NotAfter: in.NotAfter, Payload: payload,
	}
	return submitOperation(ctx, ops, in.Credential, intent)
}

func prCreatePayload(in PRCreateOpIntent) (json.RawMessage, error) {
	if !factory.ValidPublicationOperationID(in.OperationID) {
		return nil, &Refusal{Reason: "invalid_operation"}
	}
	if in.ActorID <= 0 || in.RepositoryID <= 0 {
		return nil, &Refusal{Reason: "invalid_operation"}
	}
	if in.AuthRevision == "" || len(in.AuthRevision) > 512 {
		return nil, &Refusal{Reason: "invalid_operation"}
	}
	if in.ExpectedRevision < 1 || in.NotAfter <= time.Now().Unix() {
		return nil, &Wait{Reason: "observation_expired"}
	}
	payload := extensions.PRCreatePayload{
		HeadRepositoryID: in.RepositoryID, HeadRef: in.HeadRef, BaseRef: in.BaseRef,
		ExpectedHeadOID: in.ExpectedHead, ExpectedBaseOID: in.ExpectedBase,
		Title: in.Title, Body: in.Body,
	}
	if err := extensions.ValidatePRCreatePayload(in.RepositoryID, payload); err != nil {
		return nil, &Refusal{Reason: "invalid_pr"}
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return raw, nil
}

func submitOperation(ctx context.Context, ops BackgroundOperations, credential extensions.CredentialFile, intent extensions.OperationIntent) (factory.OperationOutcome, error) {
	var empty factory.OperationOutcome
	if ops == nil {
		return empty, &Wait{Reason: "operations_unavailable"}
	}
	record, err := ops.SubmitOperation(ctx, credential, intent)
	if err != nil {
		return empty, mapSubmitError(ctx, ops, intent.OperationID, err)
	}
	outcome, err := mapRecord(intent.OperationID, record)
	if err != nil {
		return empty, err
	}
	if record.Kind != intent.Kind || record.ActorID != intent.ActorID || record.RepositoryID != intent.RepositoryID {
		return empty, errors.New("operation record differs from submitted intent")
	}
	return outcome, nil
}

// mapSubmitError reconciles a failed submit before reporting it. A lost
// reply may hide a recorded operation: the lookup adopts it, and only a
// not_observed lookup leaves the identity unsubmitted. Terminal dispatch
// verdicts refuse or wait without another submit.
func mapSubmitError(ctx context.Context, ops BackgroundOperations, operationID string, err error) error {
	var status *StatusError
	if !errors.As(err, &status) {
		return reconcileAfterSubmitError(ctx, ops, operationID, err)
	}
	switch {
	case status.Status == 400:
		return &Refusal{Reason: "invalid_intent"}
	case status.Status == 401:
		return &Wait{Reason: "credential_invalid"}
	case status.Status == 403:
		return &Wait{Reason: "authority_lost"}
	case status.Status == 409 && status.Body == "intent_conflict":
		return &Refusal{Reason: "intent_conflict"}
	case status.Status == 409 && status.Body == "cancelled_before_submit":
		return reconcileAfterSubmitError(ctx, ops, operationID, err)
	default:
		return reconcileAfterSubmitError(ctx, ops, operationID, err)
	}
}

// reconcileAfterSubmitError adopts a possibly recorded operation after a
// submit call failed. NotObserved leaves the identity free for a later
// submit; any recorded state is adopted; a second failure reports the
// original error so the pass reconciles later.
func reconcileAfterSubmitError(ctx context.Context, ops BackgroundOperations, operationID string, submitErr error) error {
	outcome, err := lookupOperation(ctx, ops, operationID)
	if err != nil {
		return submitErr
	}
	return &Adopted{Outcome: outcome}
}

// Adopted carries a reconciled operation outcome discovered while
// handling a failed call. The caller records it instead of the error.
type Adopted struct{ Outcome factory.OperationOutcome }

func (e *Adopted) Error() string { return "publication operation reconciled after a lost reply" }

// LookupOperation reconciles one persisted operation identity against
// native state. NotObserved means no record exists yet; it never proves
// that an earlier request cannot still arrive.
func (c Config) LookupOperation(ctx context.Context, ops BackgroundOperations, operationID string) (factory.OperationOutcome, error) {
	if !factory.ValidPublicationOperationID(operationID) {
		return factory.OperationOutcome{}, &Refusal{Reason: "invalid_operation"}
	}
	if ops == nil {
		return factory.OperationOutcome{}, &Wait{Reason: "operations_unavailable"}
	}
	return lookupOperation(ctx, ops, operationID)
}

func lookupOperation(ctx context.Context, ops BackgroundOperations, operationID string) (factory.OperationOutcome, error) {
	var empty factory.OperationOutcome
	lookup, err := ops.GetOperation(ctx, operationID)
	if err != nil {
		return empty, mapLookupError(err)
	}
	if lookup.OperationID != operationID || lookup.InstallationID == "" {
		return empty, errors.New("operation lookup identity differs")
	}
	if lookup.Status == extensions.BackgroundOutcomeNotObserved {
		if lookup.Record != nil {
			return empty, errors.New("operation lookup contradicts its status")
		}
		return factory.OperationOutcome{NotObserved: true}, nil
	}
	if lookup.Record == nil {
		return empty, errors.New("operation lookup lacks its record")
	}
	if lookup.Record.InstallationID != lookup.InstallationID {
		return empty, errors.New("operation record installation differs from lookup")
	}
	return mapRecord(operationID, *lookup.Record)
}

func mapLookupError(err error) error {
	var status *StatusError
	if !errors.As(err, &status) {
		return err
	}
	switch status.Status {
	case 400:
		return &Refusal{Reason: "invalid_operation"}
	case 401, 403:
		return &Wait{Reason: "operations_unavailable"}
	default:
		return err
	}
}

// CancelOperation cancels one persisted operation identity and reconciles
// the answer. Cancellation contends on the same record as submission: a
// winning cancellation prevents the effect, a committed write makes it
// too late, and a lost reply reconciles by lookup.
func (c Config) CancelOperation(ctx context.Context, ops BackgroundOperations, operationID string) (factory.OperationOutcome, error) {
	var empty factory.OperationOutcome
	if !factory.ValidPublicationOperationID(operationID) {
		return empty, &Refusal{Reason: "invalid_operation"}
	}
	if ops == nil {
		return empty, &Wait{Reason: "operations_unavailable"}
	}
	record, err := ops.CancelOperation(ctx, operationID)
	if err != nil {
		var status *StatusError
		if errors.As(err, &status) && (status.Status == 400 || status.Status == 401 || status.Status == 403) {
			return empty, &Wait{Reason: "operations_unavailable"}
		}
		outcome, lookupErr := lookupOperation(ctx, ops, operationID)
		if lookupErr != nil {
			return empty, err
		}
		return outcome, nil
	}
	return mapRecord(operationID, record)
}

// mapRecord adopts one native operation record. Identity, kind and state
// vocabularies are verified; anything else refuses instead of adopting a
// lookalike. An unimplemented-kind outcome waits without a record.
func mapRecord(operationID string, record extensions.OperationRecord) (factory.OperationOutcome, error) {
	var empty factory.OperationOutcome
	if record.OperationID != operationID || record.InstallationID == "" {
		return empty, errors.New("operation record identity differs")
	}
	if record.Outcome == extensions.BackgroundOutcomeOperationsUnavailable {
		return empty, &Wait{Reason: "operations_unavailable"}
	}
	outcome := factory.OperationOutcome{
		OperationID: record.OperationID, InstallationID: record.InstallationID, Kind: record.Kind,
		Effect: record.EffectState, Cancellation: record.CancellationStatus,
		Completion: record.CompletionState, Reason: record.ReasonCode,
	}
	if !factory.ValidOpEffect(outcome.Effect) {
		return empty, errors.New("operation effect is unknown")
	}
	if outcome.Cancellation == "" {
		outcome.Cancellation = factory.OpCancelNone
	}
	if !factory.ValidOpCancellation(outcome.Cancellation) {
		return empty, errors.New("operation cancellation is unknown")
	}
	// Cancellation may win before submit. Its tombstone has no actor,
	// repository or kind; every submitted record must retain all three.
	tombstone := record.Kind == "" && record.ActorID == "" && record.RepositoryID == "" &&
		outcome.Effect == factory.OpEffectNotCommitted && outcome.Cancellation == factory.OpCancelCancelled
	if !tombstone {
		if record.Kind != factory.OpRefPublish && record.Kind != factory.OpPRCreate {
			return empty, errors.New("operation kind differs from publication")
		}
		actor, actorErr := strconv.ParseInt(record.ActorID, 10, 64)
		repository, repositoryErr := strconv.ParseInt(record.RepositoryID, 10, 64)
		if actorErr != nil || repositoryErr != nil || actor <= 0 || repository <= 0 {
			return empty, errors.New("operation record lacks its native scope")
		}
		outcome.ActorID, outcome.RepositoryID = actor, repository
	}
	if outcome.Completion != "" && !factory.ValidOpCompletion(outcome.Completion) {
		return empty, errors.New("operation completion is unknown")
	}
	if len(outcome.Reason) > 64 {
		return empty, errors.New("operation reason exceeds bounds")
	}
	if len(record.Receipt) > factory.MaxPublicationReceipt {
		return empty, errors.New("operation receipt exceeds bounds")
	}
	outcome.Receipt = append([]byte(nil), record.Receipt...)
	return outcome, nil
}

// runWithEnv runs one Git command with additional environment entries,
// binding a registered operation to its push headers without placing
// secrets in arguments. Diagnostics stay redacted like run.
func (g *repository) runWithEnv(ctx context.Context, extra []string, args ...string) error {
	command := g.command(ctx, args...)
	command.Env = append(command.Env, extra...)
	var output limitedOutput
	command.Stdout = &output
	if err := command.Run(); err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return errors.New("publication Git operation failed")
	}
	return nil
}

// ValidatedRepo is one validated candidate workspace: the verified bundle
// imported into a fresh bare repository with its exact candidate,
// ancestry, protected paths and credential material checked. It performs
// no push; PushBranch publishes exactly its candidate under a registered
// operation.
type ValidatedRepo struct {
	repo    *repository
	cleanup func()
	request Request
}

// PrepareValidated imports and validates one candidate bundle, returning
// the workspace for a registered push. The caller closes it. Validation
// verdicts refuse terminally; transport failures stay errors for a later
// pass.
func (c Config) PrepareValidated(ctx context.Context, r Request) (*ValidatedRepo, error) {
	if err := c.Validate(); err != nil {
		return nil, &Refusal{Reason: "invalid_publisher"}
	}
	if err := r.Validate(); err != nil {
		return nil, &Refusal{Reason: "candidate_invalid"}
	}
	git, cleanup, err := c.prepare(ctx, r)
	if err != nil {
		return nil, err
	}
	validated := &ValidatedRepo{repo: git, cleanup: cleanup, request: r}
	if err = verifyCandidateBundle(ctx, git); err != nil {
		cleanup()
		return nil, err
	}
	if err = git.validateCandidate(ctx, r, c.ProtectedPaths); err != nil {
		cleanup()
		return nil, validationVerdict(err)
	}
	if err = git.checkCredentials(ctx, r); err != nil {
		cleanup()
		return nil, validationVerdict(err)
	}
	return validated, nil
}

// Bundle verification is a local Git verdict on the supplied bytes.
// Startup failures and cancellation are infrastructure errors instead.
func verifyCandidateBundle(ctx context.Context, git *repository) error {
	err := git.command(ctx, "bundle", "verify", filepath.Join(git.dir, "candidate.bundle")).Run()
	if err == nil {
		return nil
	}
	if ctx.Err() != nil {
		return ctx.Err()
	}
	var exited *exec.ExitError
	if errors.As(err, &exited) && exited.ExitCode() == 1 {
		return &Refusal{Reason: "candidate_invalid"}
	}
	return errors.New("publication bundle verification unavailable")
}

// validationVerdict maps candidate checks to terminal refusals. Git
// transport failures and cancellation stay errors; every other check
// outcome is a verdict on the recorded bytes.
func validationVerdict(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	if err.Error() == "publication Git operation failed" || err.Error() == "publication credential scan failed" {
		return err
	}
	return &Refusal{Reason: "candidate_invalid"}
}

// Close removes the validated workspace.
func (v *ValidatedRepo) Close() {
	if v != nil && v.cleanup != nil {
		v.cleanup()
	}
}

// PushBranch pushes exactly the validated candidate to the registered
// target under its operation binding. The refspec carries one tuple and
// no options; anything else refuses before launching. A push failure
// reconciles by lookup: a committed, refused or still-pending operation
// is adopted, and only a pending one may push again under a rebound
// admission.
func (v *ValidatedRepo) PushBranch(ctx context.Context, c Config, ops BackgroundOperations, operationID, targetRef string) error {
	if v == nil || v.repo == nil {
		return errors.New("publication workspace is not validated")
	}
	if !factory.ValidPublicationOperationID(operationID) {
		return &Refusal{Reason: "invalid_operation"}
	}
	if !validPublishRef(targetRef) {
		return &Refusal{Reason: "invalid_refs"}
	}
	if ops == nil {
		return &Wait{Reason: "operations_unavailable"}
	}
	refspec, err := extensions.FormatPublishRefspec("refs/heads/candidate", targetRef)
	if err != nil {
		return &Refusal{Reason: "invalid_refs"}
	}
	env, err := ops.PublishPushEnv(ctx, operationID, false)
	if err != nil {
		return err
	}
	if err = v.repo.runWithEnv(ctx, env, "push", "--no-follow-tags", c.Remote, refspec); err != nil {
		return v.reconcilePush(ctx, c, ops, operationID, targetRef, refspec, err)
	}
	return nil
}

// reconcilePush adopts the operation behind a failed push. A still-pending
// operation retries once under a rebound admission: a revoked admission
// fails before launching, so the retry cannot duplicate a receiver. Any
// other state is adopted through the lookup.
func (v *ValidatedRepo) reconcilePush(ctx context.Context, c Config, ops BackgroundOperations, operationID, targetRef, refspec string, pushErr error) error {
	outcome, err := lookupOperation(ctx, ops, operationID)
	if err != nil {
		return pushErr
	}
	if outcome.Effect != factory.OpEffectPending || outcome.NotObserved {
		return &Adopted{Outcome: outcome}
	}
	env, err := ops.PublishPushEnv(ctx, operationID, true)
	if err != nil {
		return &Adopted{Outcome: outcome}
	}
	if err = v.repo.runWithEnv(ctx, env, "push", "--no-follow-tags", c.Remote, refspec); err != nil {
		reconciled, lookupErr := lookupOperation(ctx, ops, operationID)
		if lookupErr != nil {
			return pushErr
		}
		return &Adopted{Outcome: reconciled}
	}
	return nil
}

// PublishReceipt decodes one committed branch-publication receipt: the
// realized ref tuple plus the comparison tip behind it. Only the
// operation's own receipt proves its effect.
type PublishReceipt struct {
	Ref           string `json:"ref"`
	OldOID        string `json:"old_oid"`
	NewOID        string `json:"new_oid"`
	ComparisonRef string `json:"comparison_ref"`
	ComparisonOID string `json:"comparison_oid"`
	ActorID       int64  `json:"actor_id"`
	RepositoryID  int64  `json:"repository_id"`
}

// DecodePublishReceipt verifies one committed publish outcome against its
// submitted tuple and adopts the realized effect. A missing, oversized,
// malformed or mismatched receipt refuses: the branch cannot advance on
// an unattributed effect.
func DecodePublishReceipt(outcome factory.OperationOutcome, ref, expectedOld, newOID, comparisonRef, comparisonOID string, actorID, repositoryID int64) (factory.BranchOutcome, error) {
	branch := factory.BranchOutcome{Operation: outcome}
	if outcome.Effect != factory.OpEffectCommitted {
		return factory.BranchOutcome{}, errors.New("branch outcome is not committed")
	}
	if len(outcome.Receipt) == 0 || len(outcome.Receipt) > factory.MaxPublicationReceipt {
		return factory.BranchOutcome{}, errors.New("branch receipt is missing")
	}
	var receipt PublishReceipt
	decoder := json.NewDecoder(strings.NewReader(string(outcome.Receipt)))
	if err := decoder.Decode(&receipt); err != nil {
		return factory.BranchOutcome{}, errors.New("branch receipt is malformed")
	}
	if err := decoder.Decode(new(any)); !errors.Is(err, io.EOF) {
		return factory.BranchOutcome{}, errors.New("branch receipt is malformed")
	}
	if receipt.Ref != ref || receipt.NewOID != newOID || receipt.ComparisonRef != comparisonRef ||
		receipt.RepositoryID != repositoryID || receipt.ActorID != actorID || actorID <= 0 ||
		!factory.ValidCommit(receipt.ComparisonOID) || !strings.EqualFold(receipt.ComparisonOID, comparisonOID) {
		return factory.BranchOutcome{}, errors.New("branch receipt differs from its intent")
	}
	if expectedOld == extensions.PublishExpectedOldAbsent {
		if receipt.OldOID != strings.Repeat("0", 40) && receipt.OldOID != strings.Repeat("0", 64) {
			return factory.BranchOutcome{}, errors.New("branch receipt differs from its intent")
		}
	} else if !strings.EqualFold(receipt.OldOID, expectedOld) {
		return factory.BranchOutcome{}, errors.New("branch receipt differs from its intent")
	}
	branch.Ref, branch.OldOID, branch.NewOID, branch.Comparison =
		receipt.Ref, receipt.OldOID, receipt.NewOID, receipt.ComparisonOID
	return branch, nil
}

// PRCreateReceipt decodes one committed PR-creation receipt: the exact
// native PR identity plus the head/base snapshot behind it.
type PRCreateReceipt struct {
	HeadRef      string `json:"head_ref"`
	BaseRef      string `json:"base_ref"`
	HeadOID      string `json:"head_oid"`
	BaseOID      string `json:"base_oid"`
	AuthorID     int64  `json:"author_id"`
	RepositoryID int64  `json:"repository_id"`
	PRID         int64  `json:"pr_id"`
	IssueID      int64  `json:"issue_id"`
	PRNumber     int64  `json:"pr_number"`
}

// DecodePRCreateReceipt verifies one committed creation outcome against
// its submitted snapshot and adopts the exact PR. Only this operation's
// receipt proves creation: an existing or similar PR is never adopted.
func DecodePRCreateReceipt(outcome factory.OperationOutcome, headRef, baseRef, headOID, baseOID string, actorID, repositoryID int64) (factory.PRCreationOutcome, error) {
	created := factory.PRCreationOutcome{Operation: outcome}
	if outcome.Effect != factory.OpEffectCommitted {
		return factory.PRCreationOutcome{}, errors.New("PR creation outcome is not committed")
	}
	if len(outcome.Receipt) == 0 || len(outcome.Receipt) > factory.MaxPublicationReceipt {
		return factory.PRCreationOutcome{}, errors.New("PR creation receipt is missing")
	}
	var receipt PRCreateReceipt
	decoder := json.NewDecoder(strings.NewReader(string(outcome.Receipt)))
	if err := decoder.Decode(&receipt); err != nil {
		return factory.PRCreationOutcome{}, errors.New("PR creation receipt is malformed")
	}
	if err := decoder.Decode(new(any)); !errors.Is(err, io.EOF) {
		return factory.PRCreationOutcome{}, errors.New("PR creation receipt is malformed")
	}
	if receipt.HeadRef != headRef || receipt.BaseRef != baseRef ||
		!strings.EqualFold(receipt.HeadOID, headOID) || !strings.EqualFold(receipt.BaseOID, baseOID) ||
		receipt.RepositoryID != repositoryID || receipt.AuthorID != actorID || actorID <= 0 {
		return factory.PRCreationOutcome{}, errors.New("PR creation receipt differs from its intent")
	}
	if receipt.PRID <= 0 || receipt.IssueID <= 0 || receipt.PRNumber <= 0 {
		return factory.PRCreationOutcome{}, errors.New("PR creation receipt lacks its identity")
	}
	created.HeadRef, created.BaseRef, created.HeadOID, created.BaseOID =
		receipt.HeadRef, receipt.BaseRef, receipt.HeadOID, receipt.BaseOID
	created.PRNumber, created.PRID, created.IssueID, created.AuthorID =
		receipt.PRNumber, receipt.PRID, receipt.IssueID, receipt.AuthorID
	return created, nil
}

// OperationNotAfter returns the submit deadline for one operation: ten
// minutes of host time from now, inside the native maximum.
func OperationNotAfter(now time.Time) int64 {
	return now.Add(operationLifetime).Unix()
}

// StatusError reports a bounded native dispatch status with its bounded
// body. Callers map terminal statuses without retrying and reconcile
// after the rest.
type StatusError struct {
	Body   string
	Status int
}

func (e *StatusError) Error() string {
	return "background request returned status " + strconv.Itoa(e.Status)
}

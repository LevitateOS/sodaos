package forgejo

import (
	"context"
	"errors"
	"net/url"
	"strings"
	"sync"
	"time"

	extensions "forgejo.org/extension-sdk"

	"github.com/levitateos/sodaos/internal/config"
	"github.com/levitateos/sodaos/internal/factory"
	forgejopublish "github.com/levitateos/sodaos/internal/forgejo/publish"
)

// Publisher is the coordinator's conditional-publication executor: it
// validates candidates in the unprivileged publisher, binds fresh native
// observations, and drives the branch-publication and PR-creation
// operations through the shared service background transport. The native
// actor credential stays in its restricted file; lookups and
// cancellations use owning-installation admission even after actor
// authority loss. Methods implement the coordinator's publication
// executor structurally; faults arrive as factory refusal/wait verdicts.
type Publisher struct {
	background *ServiceBackground
	rest       *Client
	internal   string
	root       string
	tokenFile  string
	mu         sync.Mutex
	actorID    int64
	login      string
}

// NewPublisher builds the publication executor over the shared service
// background transport, the internal REST client, the internal native
// origin, the private publication root and the restricted native actor
// credential file. No I/O happens before the first call.
func NewPublisher(background *ServiceBackground, rest *Client, internalURL, root, tokenFile string) *Publisher {
	return &Publisher{background: background, rest: rest, internal: strings.TrimRight(internalURL, "/"), root: root, tokenFile: tokenFile}
}

// ObservePublication validates one candidate bundle and brackets its
// fresh native observation: the idle revision plus the exact target and
// comparison tips behind it.
func (p *Publisher) ObservePublication(ctx context.Context, work factory.PublicationWork) (factory.PublicationObservation, error) {
	var empty factory.PublicationObservation
	cfg, err := p.publisherConfig(ctx, work)
	if err != nil {
		return empty, err
	}
	secrets, err := p.protectedSecrets()
	if err != nil {
		return empty, err
	}
	validated, err := cfg.PrepareValidated(ctx, forgejopublish.Request{
		Run: work.Run, BaseSHA: work.BaseSHA, Commit: work.Candidate,
		Bundle: work.Bundle, ProtectedCredentials: secrets,
	})
	if err != nil {
		return empty, p.mapValidationError(err)
	}
	validated.Close()
	observation, err := cfg.ObserveForPublish(ctx, p.background, factory.PublishBranchName(work.AssignmentID), work.TargetBranch)
	if err != nil {
		return empty, p.flattenError(err)
	}
	return observation, nil
}

// SubmitPublish submits one branch-publication intent under its persisted
// identity with lost-reply reconciliation.
func (p *Publisher) SubmitPublish(ctx context.Context, work factory.PublicationWork) (factory.OperationOutcome, error) {
	var empty factory.OperationOutcome
	if err := work.Validate(); err != nil {
		return empty, &factory.PublicationRefusal{Reason: "invalid_work"}
	}
	cfg, err := p.publisherConfig(ctx, work)
	if err != nil {
		return empty, err
	}
	outcome, err := cfg.SubmitPublish(ctx, p.background, forgejopublish.PublishOpIntent{
		Credential: extensions.CredentialFile(p.tokenFile), OperationID: work.OperationID,
		AuthRevision: work.AuthRevision, Ref: factory.PublishBranchName(work.AssignmentID),
		ExpectedOld: work.ExpectedOld, NewOID: work.Candidate,
		ComparisonRef: work.ComparisonRef, ExpectedComparisonOID: work.ComparisonOID,
		ActorID: work.ActorID, RepositoryID: work.Repository,
		ExpectedRevision: work.NativeRev, NotAfter: work.NotAfter,
		CorrectionNumber: work.CorrectionNumber, CorrectionAuthor: work.CorrectionAuthor,
	})
	return p.flattenOutcome(outcome, err)
}

// PushBranch pushes exactly the validated candidate under its registered
// branch operation and adopts the post-push lookup.
func (p *Publisher) PushBranch(ctx context.Context, work factory.PublicationWork) (factory.OperationOutcome, error) {
	var empty factory.OperationOutcome
	if err := work.Validate(); err != nil {
		return empty, &factory.PublicationRefusal{Reason: "invalid_work"}
	}
	cfg, err := p.publisherConfig(ctx, work)
	if err != nil {
		return empty, err
	}
	secrets, err := p.protectedSecrets()
	if err != nil {
		return empty, err
	}
	validated, err := cfg.PrepareValidated(ctx, forgejopublish.Request{
		Run: work.Run, BaseSHA: work.BaseSHA, Commit: work.Candidate,
		Bundle: work.Bundle, ProtectedCredentials: secrets,
	})
	if err != nil {
		return empty, p.mapValidationError(err)
	}
	defer validated.Close()
	target := factory.PublishBranchName(work.AssignmentID)
	if err := validated.PushBranch(ctx, cfg, p.background, work.OperationID, target); err != nil {
		var adopted *forgejopublish.Adopted
		if errors.As(err, &adopted) {
			return adopted.Outcome, nil
		}
		return empty, p.flattenError(err)
	}
	outcome, err := cfg.LookupOperation(ctx, p.background, work.OperationID)
	if err != nil {
		return empty, p.flattenError(err)
	}
	return outcome, nil
}

// SubmitPRCreate submits one PR-creation intent under its persisted
// identity with lost-reply reconciliation.
func (p *Publisher) SubmitPRCreate(ctx context.Context, work factory.PublicationWork) (factory.OperationOutcome, error) {
	var empty factory.OperationOutcome
	if err := work.Validate(); err != nil {
		return empty, &factory.PublicationRefusal{Reason: "invalid_work"}
	}
	cfg, err := p.publisherConfig(ctx, work)
	if err != nil {
		return empty, err
	}
	outcome, err := cfg.SubmitPRCreate(ctx, p.background, forgejopublish.PRCreateOpIntent{
		Credential: extensions.CredentialFile(p.tokenFile), OperationID: work.OperationID,
		AuthRevision: work.AuthRevision, HeadRef: factory.PublishBranchName(work.AssignmentID),
		BaseRef: work.TargetBranch, ExpectedHead: work.Candidate, ExpectedBase: work.ComparisonOID,
		Title: work.PRTitle, Body: work.PRBody,
		ActorID: work.ActorID, RepositoryID: work.Repository,
		ExpectedRevision: work.NativeRev, NotAfter: work.NotAfter,
	})
	return p.flattenOutcome(outcome, err)
}

// LookupOp reconciles one persisted operation identity against native
// state.
func (p *Publisher) LookupOp(ctx context.Context, operationID string) (factory.OperationOutcome, error) {
	var empty factory.OperationOutcome
	if p.background == nil {
		return empty, &factory.PublicationWait{Reason: "operations_unavailable"}
	}
	outcome, err := forgejopublish.Config{}.LookupOperation(ctx, p.background, operationID)
	if err != nil {
		return empty, p.flattenError(err)
	}
	return outcome, nil
}

// CancelOp cancels one persisted operation identity and reconciles the
// answer.
func (p *Publisher) CancelOp(ctx context.Context, operationID string) (factory.OperationOutcome, error) {
	var empty factory.OperationOutcome
	if p.background == nil {
		return empty, &factory.PublicationWait{Reason: "operations_unavailable"}
	}
	outcome, err := forgejopublish.Config{}.CancelOperation(ctx, p.background, operationID)
	if err != nil {
		return empty, p.flattenError(err)
	}
	return outcome, nil
}

// AdoptBranch verifies one committed branch outcome against its submitted
// tuple and adopts the realized effect.
func (p *Publisher) AdoptBranch(work factory.PublicationWork, outcome factory.OperationOutcome) (factory.BranchOutcome, error) {
	if err := publicationOutcomeMatches(work, outcome, factory.OpRefPublish); err != nil {
		return factory.BranchOutcome{}, err
	}
	return forgejopublish.DecodePublishReceipt(outcome, factory.PublishBranchName(work.AssignmentID),
		work.ExpectedOld, work.Candidate, work.ComparisonRef, work.ComparisonOID, work.ActorID, work.Repository)
}

// AdoptPRCreation verifies one committed creation outcome against its
// submitted snapshot and adopts the exact PR.
func (p *Publisher) AdoptPRCreation(work factory.PublicationWork, outcome factory.OperationOutcome) (factory.PRCreationOutcome, error) {
	if err := publicationOutcomeMatches(work, outcome, factory.OpPRCreate); err != nil {
		return factory.PRCreationOutcome{}, err
	}
	return forgejopublish.DecodePRCreateReceipt(outcome, factory.PublishBranchName(work.AssignmentID),
		work.TargetBranch, work.Candidate, work.ComparisonOID, work.ActorID, work.Repository)
}

func publicationOutcomeMatches(work factory.PublicationWork, outcome factory.OperationOutcome, kind string) error {
	if outcome.OperationID != work.OperationID || outcome.InstallationID == "" || outcome.Kind != kind ||
		outcome.ActorID != work.ActorID || outcome.RepositoryID != work.Repository {
		return errors.New("publication outcome differs from its recorded intent")
	}
	return nil
}

// publisherConfig resolves one work item's publisher configuration: the
// repository's exact internal smart-HTTP remote plus the private root,
// bound actor login and restricted credential file. Every remote is
// derived from the verified repository identity, never caller-supplied.
func (p *Publisher) publisherConfig(ctx context.Context, work factory.PublicationWork) (forgejopublish.Config, error) {
	var empty forgejopublish.Config
	if err := work.ValidateObservation(); err != nil {
		return empty, &factory.PublicationRefusal{Reason: "invalid_work"}
	}
	if p.background == nil || p.rest == nil || p.internal == "" || p.root == "" || p.tokenFile == "" {
		return empty, &factory.PublicationWait{Reason: "operations_unavailable"}
	}
	if err := p.checkActor(ctx, work.ActorID); err != nil {
		return empty, err
	}
	token, err := config.Secret(p.tokenFile)
	if err != nil {
		return empty, &factory.PublicationWait{Reason: "credential_invalid"}
	}
	repo, err := p.rest.RepositoryByID(ctx, token, work.Repository)
	if err != nil {
		return empty, &factory.PublicationWait{Reason: "repository_unavailable"}
	}
	login := p.cachedLogin()
	cfg := forgejopublish.Config{
		Root: p.root, Remote: p.internal + "/" + url.PathEscape(repo.Owner.Login) + "/" + url.PathEscape(repo.Name) + ".git",
		Username: login, TokenFile: p.tokenFile,
	}
	if err := cfg.Validate(); err != nil {
		return empty, &factory.PublicationWait{Reason: "operations_unavailable"}
	}
	return cfg, nil
}

// checkActor binds the presented credential to the policy-bound actor:
// the credential's native owner must equal the enrolled binding. A
// mismatch waits instead of submitting as the wrong actor.
func (p *Publisher) checkActor(ctx context.Context, actorID int64) error {
	p.mu.Lock()
	cached := p.actorID
	p.mu.Unlock()
	if cached != 0 {
		if cached != actorID {
			return &factory.PublicationWait{Reason: "authority_lost"}
		}
		return nil
	}
	if p.rest == nil || p.tokenFile == "" {
		return &factory.PublicationWait{Reason: "operations_unavailable"}
	}
	token, err := config.Secret(p.tokenFile)
	if err != nil {
		return &factory.PublicationWait{Reason: "credential_invalid"}
	}
	bounded, stop := context.WithTimeout(ctx, 30*time.Second)
	defer stop()
	user, err := loadActor(bounded, p.rest, token)
	if err != nil {
		return &factory.PublicationWait{Reason: "repository_unavailable"}
	}
	if user.ID <= 0 || user.Login == "" {
		return &factory.PublicationWait{Reason: "repository_unavailable"}
	}
	p.mu.Lock()
	p.actorID, p.login = user.ID, user.Login
	p.mu.Unlock()
	if user.ID != actorID {
		return &factory.PublicationWait{Reason: "authority_lost"}
	}
	return nil
}

func (p *Publisher) cachedLogin() string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.login
}

// protectedSecrets supplies the candidate credential-leak scan with the
// publisher's own native secret: a candidate repeating it refuses.
// Broker provider credentials stay in broker custody and are unknown
// here; their exfiltration cannot be scanned by value.
func (p *Publisher) protectedSecrets() ([]string, error) {
	secret, err := config.Secret(p.tokenFile)
	if err != nil {
		return nil, &factory.PublicationWait{Reason: "credential_invalid"}
	}
	return []string{secret}, nil
}

// mapValidationError maps candidate validation failures to terminal
// refusals: the recorded bytes are wrong, and retrying identical bytes
// is pointless. Transport failures reaching validation stay errors.
func (p *Publisher) mapValidationError(err error) error {
	var refusal *forgejopublish.Refusal
	if errors.As(err, &refusal) {
		if refusal.Reason == "invalid_publisher" {
			return &factory.PublicationWait{Reason: "operations_unavailable"}
		}
		return &factory.PublicationRefusal{Reason: "candidate_invalid"}
	}
	var wait *forgejopublish.Wait
	if errors.As(err, &wait) {
		return &factory.PublicationWait{Reason: wait.Reason}
	}
	return err
}

// flattenOutcome flattens executor answers onto the coordinator
// contract: adopted outcomes return directly, verdicts translate, and
// transport errors pass through for later reconciliation.
func (p *Publisher) flattenOutcome(outcome factory.OperationOutcome, err error) (factory.OperationOutcome, error) {
	var empty factory.OperationOutcome
	if err == nil {
		return outcome, nil
	}
	var adopted *forgejopublish.Adopted
	if errors.As(err, &adopted) {
		return adopted.Outcome, nil
	}
	return empty, p.flattenError(err)
}

func (p *Publisher) flattenError(err error) error {
	var refusal *forgejopublish.Refusal
	if errors.As(err, &refusal) {
		return &factory.PublicationRefusal{Reason: refusal.Reason}
	}
	var wait *forgejopublish.Wait
	if errors.As(err, &wait) {
		return &factory.PublicationWait{Reason: wait.Reason}
	}
	return err
}

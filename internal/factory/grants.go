// Factory standing grants: separately recorded repository policy, appliance
// capacity, operator permission and provider sponsorship. Each authority is
// checked separately; one grant cannot substitute for another. Native actor
// references name administrator-enrolled bindings by ID only; they never
// enroll or carry credentials.
package factory

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"

	"github.com/levitateos/sodaos/internal/project"
)

// Operation kinds mirror the enrolled native binding kinds. Authorization
// for one kind never grants another.
const (
	OpRefPublish   = "git.ref.publish"
	OpPRCreate     = "pull_request.create"
	OpReviewSubmit = "pull_request.review.submit"
	OpMerge        = "pull_request.merge"
)

// ValidOperationKind reports whether kind is a supported native operation kind.
func ValidOperationKind(kind string) bool {
	switch kind {
	case OpRefPublish, OpPRCreate, OpReviewSubmit, OpMerge:
		return true
	default:
		return false
	}
}

// MergeFastForward is the only supported conditional merge method.
const MergeFastForward = "fast-forward-only"

// Settings command types carried on the shared command ledger. Each one
// binds a client-generated command ID to normalized admitted fields; the
// same ID with different content conflicts instead of executing twice.
const (
	CommandPolicy           = "policy"
	CommandOperatorGrant    = "operator-grant"
	CommandCapacity         = "capacity"
	CommandSponsorship      = "sponsorship"
	CommandEnvironmentGrant = "environment-grant"
	CommandRequirement      = "preparation-requirement"
	CommandApproval         = "preparation-approval"
	CommandReopen           = "reopen"
	CommandAcceptance       = "acceptance"
	CommandWithdrawal       = "withdrawal"
	CommandPause            = "pause"
	CommandResume           = "resume"
	CommandRetry            = "retry"
	CommandTakeover         = "takeover"
)

// SettingsCommandType reports whether typ is a grant/settings command. The
// operator stop/reconcile commands stay payload-free with their own digest.
// Lifecycle pause/resume/retry/takeover ride the same idempotent ledger
// through the admitted browser controls, never the operator endpoint.
func SettingsCommandType(typ string) bool {
	switch typ {
	case CommandPolicy, CommandOperatorGrant, CommandCapacity, CommandSponsorship,
		CommandEnvironmentGrant, CommandRequirement, CommandApproval, CommandReopen,
		CommandAcceptance, CommandWithdrawal,
		CommandPause, CommandResume, CommandRetry, CommandTakeover:
		return true
	default:
		return false
	}
}

// SettingsDigest binds a settings command to its exact normalized payload.
// Operator commands never use this digest.
func SettingsDigest(typ, target, payload string) string {
	sum := sha256.Sum256([]byte(typ + "\x00" + target + "\x00" + payload))
	return hex.EncodeToString(sum[:])
}

// ActorBindingRef references one administrator-enrolled native actor binding
// for the policy's repository. It holds IDs only, never secrets, and never
// selects an installation: the host derives installation identity at
// dispatch from authenticated transport.
type ActorBindingRef struct {
	TokenID int64  `json:"token_id,string"`
	ActorID int64  `json:"actor_id,string"`
	Kind    string `json:"kind"`
}

func (a ActorBindingRef) Validate() error {
	if a.TokenID <= 0 || a.ActorID <= 0 || !ValidOperationKind(a.Kind) {
		return errors.New("invalid native actor binding reference")
	}
	return nil
}

// RoleSelection pins the harness and model for one factory role. Only
// harnesses with their own proof are selectable at dispatch; the policy
// records the selection without proving it.
type RoleSelection struct {
	Harness string `json:"harness"`
	Model   string `json:"model"`
}

func (s RoleSelection) Validate() error {
	if !project.ValidHarnessVersion(s.Harness) {
		return errors.New("invalid role harness selection")
	}
	if s.Model == "" || len(s.Model) > 128 || strings.IndexFunc(s.Model, func(r rune) bool {
		return r < 0x20 || r == 0x7f
	}) >= 0 {
		return errors.New("invalid role model selection")
	}
	return nil
}

const (
	// MaxRequiredChecks bounds the nonempty native check set.
	MaxRequiredChecks = 32
	// MaxCheckName bounds one required check name.
	MaxCheckName = 128
	// MaxPolicyConcurrent bounds repository execution parallelism.
	MaxPolicyConcurrent = 32
)

// ValidTargetBranch reports whether branch is one full native branch ref.
func ValidTargetBranch(branch string) bool {
	name, ok := strings.CutPrefix(branch, "refs/heads/")
	if !ok || name == "" || len(branch) > 256 || branch != strings.TrimSpace(branch) {
		return false
	}
	return !strings.Contains(branch, "..") && strings.IndexFunc(branch, func(r rune) bool {
		return r <= 0x20 || r == 0x7f || strings.ContainsRune(" ~^:?*\\", r)
	}) < 0
}

// RepositoryPolicy is the repository owner's standing factory authorization:
// enablement, target, role selections, required native checks, merge method,
// repository limits and the enrolled native actor references. It grants no
// appliance capacity, provider sponsorship or environment permission.
type RepositoryPolicy struct {
	Roles         map[string]RoleSelection `json:"roles"`
	Publish       ActorBindingRef          `json:"publish_actor"`
	Create        ActorBindingRef          `json:"create_actor"`
	Review        ActorBindingRef          `json:"review_actor"`
	Merge         ActorBindingRef          `json:"merge_actor"`
	Repository    int64                    `json:"repository,string"`
	Revision      int64                    `json:"revision"`
	GrantedBy     int64                    `json:"granted_by,string"`
	TargetBranch  string                   `json:"target_branch"`
	Checks        []string                 `json:"required_checks"`
	MergeMethod   string                   `json:"merge_method"`
	MaxConcurrent int                      `json:"max_concurrent"`
	Enabled       bool                     `json:"enabled"`
	Paused        bool                     `json:"paused"`
}

func (p RepositoryPolicy) Validate() error {
	if p.Repository <= 0 || p.Revision < 0 || p.GrantedBy <= 0 {
		return errors.New("invalid repository policy identity")
	}
	if !ValidTargetBranch(p.TargetBranch) {
		return errors.New("invalid policy target branch")
	}
	if len(p.Roles) != 2 || p.Roles[project.RoleCoder].Validate() != nil || p.Roles[project.RoleReviewer].Validate() != nil {
		return errors.New("policy must select exactly the coding and review roles")
	}
	if len(p.Checks) == 0 || len(p.Checks) > MaxRequiredChecks {
		return errors.New("policy requires a nonempty bounded check set")
	}
	seen := make(map[string]bool, len(p.Checks))
	for _, check := range p.Checks {
		if check == "" || len(check) > MaxCheckName || check != strings.TrimSpace(check) ||
			strings.IndexFunc(check, func(r rune) bool { return r < 0x20 || r == 0x7f }) >= 0 || seen[check] {
			return errors.New("invalid required check name")
		}
		seen[check] = true
	}
	if p.MergeMethod != MergeFastForward {
		return errors.New("unsupported merge method")
	}
	for slot, ref := range map[string]ActorBindingRef{
		OpRefPublish: p.Publish, OpPRCreate: p.Create, OpReviewSubmit: p.Review, OpMerge: p.Merge,
	} {
		if err := ref.Validate(); err != nil || ref.Kind != slot {
			return errors.New("policy actor binding does not match its operation kind")
		}
	}
	if p.MaxConcurrent < 1 || p.MaxConcurrent > MaxPolicyConcurrent {
		return errors.New("invalid repository concurrency limit")
	}
	return nil
}

// Capacity is the appliance-wide factory execution budget, held separately
// from any repository grant. Only the configured operator changes it.
type Capacity struct {
	Revision          int64 `json:"revision"`
	UpdatedBy         int64 `json:"updated_by,string"`
	MaxConcurrentRuns int   `json:"max_concurrent_runs"`
	MaxQueued         int   `json:"max_queued"`
}

func (c Capacity) Validate() error {
	if c.Revision < 0 || c.UpdatedBy <= 0 {
		return errors.New("invalid capacity identity")
	}
	if c.MaxConcurrentRuns < 1 || c.MaxConcurrentRuns > 64 || c.MaxQueued < 0 || c.MaxQueued > 4096 {
		return errors.New("invalid appliance capacity limits")
	}
	return nil
}

// OperatorGrant permits one repository to use appliance capacity within its
// own limit. It grants no repository access or merge rights.
type OperatorGrant struct {
	Repository    int64 `json:"repository,string"`
	Revision      int64 `json:"revision"`
	GrantedBy     int64 `json:"granted_by,string"`
	MaxConcurrent int   `json:"max_concurrent"`
	Active        bool  `json:"active"`
}

func (g OperatorGrant) Validate() error {
	if g.Repository <= 0 || g.Revision < 0 || g.GrantedBy <= 0 {
		return errors.New("invalid operator grant identity")
	}
	if g.MaxConcurrent < 1 || g.MaxConcurrent > MaxPolicyConcurrent {
		return errors.New("invalid operator concurrency limit")
	}
	return nil
}

// AuthorityRef binds one dispatch decision to the exact grant, preparation
// and acceptance revisions behind it. Fountain treats these revisions as
// opaque; a changed grant closes dispatch rather than renewing old work.
type AuthorityRef struct {
	RequirementsID string `json:"requirements_id,omitempty"`
	ApprovalID     string `json:"approval_id,omitempty"`
	Policy         int64  `json:"policy"`
	Operator       int64  `json:"operator"`
	Capacity       int64  `json:"capacity"`
	Sponsorship    int64  `json:"sponsorship"`
	Environment    int64  `json:"environment"`
}

// Missing-authority reason codes. Bounded and credential-free: controls
// expose which authorization is missing or withdrawn, never secrets.
const (
	MissingPolicy            = "policy_missing"
	MissingPolicyDisabled    = "policy_disabled"
	MissingPolicyPaused      = "policy_paused"
	MissingOperatorGrant     = "operator_grant_missing"
	MissingOperatorWithdrawn = "operator_grant_withdrawn"
	MissingCapacity          = "capacity_missing"
	MissingSponsorship       = "sponsorship_missing"
	MissingSponsorshipGone   = "sponsorship_withdrawn"
	MissingEnvironment       = "environment_grant_missing"
	MissingEnvironmentGone   = "environment_grant_withdrawn"
	MissingPreparation       = "preparation_not_ready"
	MissingDispatch          = "dispatch_closed"
)

// EffectiveAuthority is the visible enablement verdict for one repository:
// the bound revisions, whether dispatch may proceed, and every missing or
// withdrawn authorization. It carries IDs and revisions only.
type EffectiveAuthority struct {
	Missing      []string     `json:"missing"`
	Authority    AuthorityRef `json:"authority"`
	Effective    bool         `json:"effective"`
	DispatchOpen bool         `json:"dispatch_open"`
}

// GrantReceipt is the immutable outcome of one grant command. Withdrawal
// fields report the durable requested state at commit; later external
// outcomes remain on their publication and merge records.
type GrantReceipt struct {
	PublicationsPending bool               `json:"publications_pending"`
	MergesPending       bool               `json:"merges_pending"`
	Effective           EffectiveAuthority `json:"effective"`
	Captured            []string           `json:"captured,omitempty"`
	CommandID           string             `json:"command_id"`
	Revision            int64              `json:"revision"`
	Withdrawn           bool               `json:"withdrawn"`
}

// AuthorityInput gathers the current records for one repository. A nil
// record means no grant was ever recorded. PreparationReady applies only
// when a Project exists for the repository.
type AuthorityInput struct {
	Policy           *RepositoryPolicy
	Operator         *OperatorGrant
	Appliance        *Capacity
	Sponsorship      *Sponsorship
	Environment      *project.EnvironmentGrant
	ProjectExists    bool
	PreparationReady bool
	DispatchOpen     bool
}

// DispatchRegistration is one outstanding dispatch captured by withdrawal in
// order. Registration is refused once dispatch closes; a delayed
// registration cannot escape the withdrawal set.
type DispatchRegistration struct {
	Authority  AuthorityRef `json:"authority"`
	ID         string       `json:"id"`
	Repository int64        `json:"repository,string"`
	Revision   int64        `json:"revision"`
}

func (d DispatchRegistration) Validate() error {
	if !ValidID(d.ID) || d.Repository <= 0 || d.Revision < 0 {
		return errors.New("invalid dispatch registration")
	}
	return nil
}

// MaxCapturedDispatch bounds the outstanding IDs one withdrawal reports.
const MaxCapturedDispatch = 1024

// MaxActiveDispatchCauses bounds the cause labels retained on one gate.
// Labels remain bounded strings because grant withdrawal callers own their
// vocabulary; lifecycle controls do not reinterpret grant causes.
const MaxActiveDispatchCauses = 16

// Withdrawal is the durable record of one dispatch closure: the cause, the
// closing principal and every outstanding ID captured in order. Local
// withdrawal acknowledges requested cancellation, not a won native race.
type Withdrawal struct {
	ActiveCauses []string              `json:"active_causes"`
	Publications PublicationWithdrawal `json:"publications"`
	Merges       MergeWithdrawal       `json:"merges"`
	Captured     []string              `json:"captured"`
	Repository   int64                 `json:"repository,string"`
	Revision     int64                 `json:"revision"`
	Cause        string                `json:"cause"`
	ClosedBy     string                `json:"closed_by"`
}

func (w Withdrawal) Validate() error {
	if w.Repository <= 0 || w.Revision < 0 || w.Cause == "" || len(w.Cause) > 256 || w.ClosedBy == "" || len(w.ClosedBy) > 128 {
		return errors.New("invalid dispatch withdrawal")
	}
	if len(w.Captured) > MaxCapturedDispatch {
		return errors.New("withdrawal captures too many outstanding dispatches")
	}
	if len(w.ActiveCauses) > MaxActiveDispatchCauses {
		return errors.New("dispatch has too many active closure causes")
	}
	seenCauses := make(map[string]struct{}, len(w.ActiveCauses))
	for _, cause := range w.ActiveCauses {
		if cause == "" || len(cause) > 256 {
			return errors.New("invalid active dispatch cause")
		}
		if _, exists := seenCauses[cause]; exists {
			return errors.New("duplicate active dispatch cause")
		}
		seenCauses[cause] = struct{}{}
	}
	for _, id := range w.Captured {
		if !ValidID(id) {
			return errors.New("withdrawal captures an invalid dispatch identity")
		}
	}
	return nil
}

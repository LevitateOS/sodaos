package factory

import (
	"errors"
	"regexp"
	"strings"
	"time"

	"github.com/levitateos/sodaos/internal/identity"
	"github.com/levitateos/sodaos/internal/project"
)

var digest = regexp.MustCompile(`^[a-f0-9]{64}$`)

func ValidDigest(value string) bool { return digest.MatchString(value) }

// Run is one supervised factory execution, keyed by its broker execution ID.
// The supervisor reconciles its lease/binding/credential facts against host
// and broker truth; Reconciled marks the settled run exactly once.
type Run struct {
	// ContextPending records a staged dispatch packet whose context has not
	// admitted its first native launch. It is cleared durably before that call.
	ContextPending       bool                                    `json:"context_pending,omitempty"`
	Admission            *RunAdmission                           `json:"admission,omitempty"`
	ExecutionManifest    []byte                                  `json:"execution_manifest,omitempty"`
	VerificationManifest []byte                                  `json:"verification_manifest,omitempty"`
	VerificationChecks   *project.FactoryVerificationChecksState `json:"verification_checks,omitempty"`
	FinalCheckpoint      *project.FactoryTaskCheckpointState     `json:"final_checkpoint,omitempty"`
	CredentialDelegated  bool                                    `json:"credential_delegated"`
	CredentialReturned   bool                                    `json:"credential_returned"`
	IdentityLeaseID      string                                  `json:"identity_lease_id,omitempty"`
	IdentityGeneration   int64                                   `json:"identity_generation,omitempty"`
	IdentityBinding      *identity.Binding                       `json:"identity_binding,omitempty"`
	ID                   string                                  `json:"id"`
	ProjectID            string                                  `json:"project_id"`
	Role                 string                                  `json:"role"`
	InputSHA             string                                  `json:"input_sha"`
	Started              time.Time                               `json:"started"`
	Deadline             time.Time                               `json:"deadline"`
	Outcome              Outcome                                 `json:"outcome,omitempty"`
	Summary              string                                  `json:"summary,omitempty"`
	Reconciled           bool                                    `json:"reconciled"`
	Image                string                                  `json:"image"`
	Harness              string                                  `json:"harness"`
	Model                string                                  `json:"model"`
}

// RunAdmission is the immutable authorization and execution profile captured
// when a dispatch packet admits this run. Standalone supervisor runs may omit
// it because they do not originate from factory dispatch.
//
// Tool is the exact tool bundle adopted at admission: the live staged host
// identity snapshotted when the packet recorded. Rows admitted before
// tool-identity proof existed leave it zero; those rows validate against
// the recorded policy version exactly and never adopt the current version.
type RunAdmission struct {
	Authority AuthorityRef              `json:"authority"`
	Policy    RepositoryPolicy          `json:"policy"`
	Profile   project.Profile           `json:"profile"`
	Tool      project.FactoryHarnessPin `json:"tool,omitempty"`
}

func (r Run) Validate() error {
	if !ValidID(r.ID) || !ValidCommit(r.InputSHA) {
		return errors.New("invalid factory execution identity")
	}
	if !ValidProjectID(r.ProjectID) || !role.MatchString(r.Role) {
		return errors.New("invalid factory execution address")
	}
	if err := r.validateDeadline(); err != nil {
		return err
	}
	if err := r.validateAdmission(); err != nil {
		return err
	}
	if len(r.ExecutionManifest) > 0 {
		manifest, err := project.ParseExecutionManifest(r.ExecutionManifest)
		if err != nil || manifest.Run != r.ID || !manifest.Deadline.Equal(r.Deadline) {
			return errors.New("execution manifest does not bind its recorded run")
		}
	}
	if r.FinalCheckpoint != nil {
		request := r.FinalCheckpoint.Request
		if request.ID != r.ID || request.Project != r.ProjectID || request.ManifestDigest != project.ExecutionManifestDigest(r.ExecutionManifest) || r.FinalCheckpoint.ValidateFor(request, true) != nil {
			return errors.New("final checkpoint does not bind its recorded run")
		}
	}
	if len(r.VerificationManifest) > 0 {
		manifest, err := project.ParseVerificationManifest(r.VerificationManifest)
		if err != nil || manifest.Run != r.ID || manifest.Project != r.ProjectID || manifest.SourceCommit != r.InputSHA || !manifest.Deadline.Equal(r.Deadline) || r.Role != project.RoleReviewer || len(r.ExecutionManifest) > 0 {
			return errors.New("verification manifest does not bind its recorded run")
		}
		if r.VerificationChecks != nil {
			state := r.VerificationChecks
			request := project.FactoryVerificationChecks{Project: r.ProjectID, Run: r.ID, Incarnation: state.Incarnation, ManifestDigest: project.ExecutionManifestDigest(r.VerificationManifest)}
			if state.ValidateFor(request) != nil || state.SourceCommit != manifest.SourceCommit || state.Preparation != manifest.Preparation || state.CheckDigest != manifest.CheckDigest {
				return errors.New("verification checks do not bind their recorded manifest")
			}
			if r.IdentityBinding != nil && r.IdentityBinding.InvocationID != state.Incarnation {
				return errors.New("verification checks differ from the native invocation")
			}
		}
	} else if r.VerificationChecks != nil {
		return errors.New("verification receipt has no recorded manifest")
	}
	if err := r.validateProvenance(); err != nil {
		return err
	}
	if err := r.validateOutcome(); err != nil {
		return err
	}
	return r.validateCredentials()
}

// AdoptedToolIdentity resolves the exact tool bundle a dispatch packet
// recorded: family and version from the assignment, executable digest and
// image from the assignment proof, cross-checked against the run image. It
// returns the zero pin when the packet predates recorded proof; callers keep
// those rows on the legacy policy-version rule and fail their recovery
// closed instead of falling back to the current selection.
func AdoptedToolIdentity(a Assignment, image string) (project.FactoryHarnessPin, error) {
	var none project.FactoryHarnessPin
	if a.AdoptedHarness == none {
		return none, nil
	}
	adopted := a.AdoptedHarness
	if err := adopted.Validate(); err != nil {
		return none, errors.New("recorded tool proof is invalid")
	}
	if a.Harness != adopted.Harness+"-"+adopted.Version || a.HarnessVers != adopted.Version {
		return none, errors.New("recorded tool proof does not match its assignment")
	}
	if image != adopted.Image {
		return none, errors.New("recorded tool proof does not match its run")
	}
	return adopted, nil
}

func (r Run) validateAdmission() error {
	if r.Admission == nil {
		return nil
	}
	a := r.Admission
	if err := a.Policy.Validate(); err != nil {
		return errors.New("invalid run admission policy")
	}
	if err := a.Profile.Validate(); err != nil {
		return errors.New("invalid run admission profile")
	}
	if a.Policy.Revision != a.Authority.Policy {
		return errors.New("run admission policy revision does not match authority")
	}
	selection, ok := a.Policy.Roles[r.Role]
	if !ok || r.Model != selection.Model {
		return errors.New("run does not match admitted role selection")
	}
	if a.Tool == (project.FactoryHarnessPin{}) {
		// Rows admitted before tool-identity proof existed: the recorded
		// policy version is the only recorded identity, so it must match
		// exactly. These rows never adopt the current version.
		if r.Harness != selection.Harness+"-"+selection.HarnessVers {
			return errors.New("run does not match admitted role selection")
		}
		return nil
	}
	// Rows with recorded proof bind it exactly: the adopted family must be
	// the approved selection family, and the run must carry the adopted
	// family-version and image byte-for-byte. Currency of the adopted
	// bundle is enforced at dispatch by the live host pin check (due-check
	// plus stale refusal); the executable digest itself is proved by the
	// host when it stages the bound bundle.
	if err := a.Tool.Validate(); err != nil {
		return errors.New("run does not match admitted role selection")
	}
	if a.Tool.Harness != selection.Harness {
		return errors.New("run does not match admitted role selection")
	}
	if r.Harness != a.Tool.Harness+"-"+a.Tool.Version || r.Image != a.Tool.Image {
		return errors.New("run does not match admitted role selection")
	}
	return nil
}

func (r Run) validateDeadline() error {
	if r.Started.IsZero() || !r.Deadline.After(r.Started) || r.Deadline.Sub(r.Started) > 3*time.Hour {
		return errors.New("invalid execution deadline")
	}
	return nil
}

func (r Run) validateProvenance() error {
	value, pinned := strings.CutPrefix(r.Image, "sha256:")
	if !pinned || !ValidDigest(value) {
		return errors.New("execution image is not pinned")
	}
	if r.Harness == "" || r.Model == "" || len(r.Harness) > 128 || len(r.Model) > 128 {
		return errors.New("execution harness/model provenance is incomplete")
	}
	return nil
}

func (r Run) validateOutcome() error {
	if !validOutcome(r.Outcome) || len(r.Summary) > 16<<10 || (r.Outcome == "" && r.Reconciled) {
		return errors.New("invalid execution outcome")
	}
	return nil
}

func (r Run) validateCredentials() error {
	if r.CredentialReturned && !r.CredentialDelegated {
		return errors.New("credential return has no delegation")
	}
	if r.IdentityLeaseID == "" {
		return r.validateUnleasedCredential()
	}
	if r.IdentityGeneration <= 0 {
		return errors.New("invalid credential generation")
	}
	if r.CredentialDelegated && r.IdentityBinding == nil {
		return errors.New("invalid credential delegation")
	}
	return r.validateIdentityBinding()
}

func (r Run) validateUnleasedCredential() error {
	if r.IdentityGeneration != 0 || r.IdentityBinding != nil || r.CredentialDelegated {
		return errors.New("credential authority has no lease")
	}
	return nil
}

func (r Run) validateIdentityBinding() error {
	b := r.IdentityBinding
	if b == nil {
		return nil
	}
	if b.Validate() != nil || b.Kind != identity.Factory || b.Generation != r.IdentityGeneration || b.ID != r.ID {
		return errors.New("invalid execution binding")
	}
	return nil
}

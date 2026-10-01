package factory

import (
	"errors"
	"regexp"
	"strings"
	"time"

	"github.com/levitateos/sodaos/internal/identity"
)

var digest = regexp.MustCompile(`^[a-f0-9]{64}$`)

func ValidDigest(value string) bool { return digest.MatchString(value) }

// Run is one supervised factory execution, keyed by its broker execution ID.
// The supervisor reconciles its lease/binding/credential facts against host
// and broker truth; Reconciled marks the settled run exactly once.
type Run struct {
	CredentialDelegated bool              `json:"credential_delegated"`
	CredentialReturned  bool              `json:"credential_returned"`
	IdentityLeaseID     string            `json:"identity_lease_id,omitempty"`
	IdentityGeneration  int64             `json:"identity_generation,omitempty"`
	IdentityBinding     *identity.Binding `json:"identity_binding,omitempty"`
	ID                  string            `json:"id"`
	ProjectID           string            `json:"project_id"`
	Role                string            `json:"role"`
	InputSHA            string            `json:"input_sha"`
	Started             time.Time         `json:"started"`
	Deadline            time.Time         `json:"deadline"`
	Outcome             Outcome           `json:"outcome,omitempty"`
	Summary             string            `json:"summary,omitempty"`
	Reconciled          bool              `json:"reconciled"`
	Image               string            `json:"image"`
	Harness             string            `json:"harness"`
	Model               string            `json:"model"`
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
	if err := r.validateProvenance(); err != nil {
		return err
	}
	if err := r.validateOutcome(); err != nil {
		return err
	}
	return r.validateCredentials()
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

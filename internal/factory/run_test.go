package factory

import (
	"strings"
	"testing"
	"time"

	"github.com/levitateos/sodaos/internal/identity"
	"github.com/levitateos/sodaos/internal/project"
)

func testRun() Run {
	return Run{ID: NewID(), ProjectID: "p123456789012345678901234", Role: "coder", InputSHA: strings.Repeat("a", 40), Started: time.Now(), Deadline: time.Now().Add(time.Hour), Image: "sha256:" + strings.Repeat("b", 64), Harness: "codex-0.157.1", Model: "test"}
}

func TestRunRequiresExecutionAddress(t *testing.T) {
	r := testRun()
	if err := r.Validate(); err != nil {
		t.Fatal(err)
	}
	r.Role = "Coder"
	if err := r.Validate(); err == nil {
		t.Fatal("unbounded role accepted")
	}
	r.Role = "coder"
	r.ProjectID = "../../other"
	if err := r.Validate(); err == nil {
		t.Fatal("non-project address accepted")
	}
}

func TestRunAdmissionBindsPolicyProfileAndRoleSelection(t *testing.T) {
	r := testRun()
	policy := grantTestPolicy()
	policy.Revision = 1
	r.Role, r.Model = project.RoleCoder, policy.Roles[project.RoleCoder].Model
	r.Admission = &RunAdmission{
		Authority: AuthorityRef{Policy: 1},
		Policy:    policy,
		Profile: project.Profile{
			ID: project.RockyHeadless, Distribution: "rocky", Version: "9.6", Interface: "headless",
			Architecture: "amd64", Image: "sha256:" + strings.Repeat("b", 64), Revision: strings.Repeat("c", 40),
		},
	}
	if err := r.Validate(); err != nil {
		t.Fatalf("valid admission rejected: %v", err)
	}

	changed := r
	changed.Admission = &RunAdmission{Authority: r.Admission.Authority, Policy: r.Admission.Policy, Profile: r.Admission.Profile}
	changed.Admission.Policy.Revision++
	if err := changed.Validate(); err == nil {
		t.Fatal("policy revision mismatch accepted")
	}
	changed.Admission = &RunAdmission{Authority: r.Admission.Authority, Policy: r.Admission.Policy, Profile: r.Admission.Profile}
	changed.Admission.Profile.Image = "unpinned"
	if err := changed.Validate(); err == nil {
		t.Fatal("invalid admitted profile accepted")
	}
	changed.Admission = &RunAdmission{Authority: r.Admission.Authority, Policy: r.Admission.Policy, Profile: r.Admission.Profile}
	changed.Model = "another-model"
	if err := changed.Validate(); err == nil {
		t.Fatal("run outside admitted role selection accepted")
	}
}

func TestCredentialAuthorityRequiresExactRunBinding(t *testing.T) {
	r := testRun()
	r.IdentityLeaseID, r.IdentityGeneration, r.CredentialDelegated = "lease", 1, true
	r.IdentityBinding = &identity.Binding{Kind: identity.Factory, ID: r.ID, Generation: 1}
	if err := r.Validate(); err != nil {
		t.Fatal(err)
	}
	r.IdentityBinding.ID = strings.Repeat("d", 64)
	if err := r.Validate(); err == nil {
		t.Fatal("binding for another execution accepted")
	}
	r.IdentityBinding.ID = r.ID
	r.IdentityBinding.Generation = 2
	if err := r.Validate(); err == nil {
		t.Fatal("different connection generation accepted")
	}
	r.IdentityBinding.Generation = 1
	r.IdentityLeaseID = ""
	if err := r.Validate(); err == nil {
		t.Fatal("delegation without reservation accepted")
	}
}

func TestReconciledRunIsTerminal(t *testing.T) {
	r := testRun()
	r.Reconciled = true
	if err := r.Validate(); err == nil {
		t.Fatal("active run settled")
	}
	r.Outcome, r.Summary = Cancelled, "stopped by operator"
	if err := r.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestCommandDigestBindsPayload(t *testing.T) {
	id := NewID()
	run := NewID()
	c := Command{ID: id, Type: CommandStop, Target: run, Principal: "os-uid:0", Digest: CommandDigest(CommandStop, run)}
	if err := c.Validate(); err != nil {
		t.Fatal(err)
	}
	c.Target = NewID()
	if err := c.Validate(); err == nil {
		t.Fatal("changed payload kept its digest")
	}
	c = Command{ID: id, Type: CommandStatus, Principal: "os-uid:0", Digest: CommandDigest(CommandStatus, "")}
	if err := c.Validate(); err == nil {
		t.Fatal("read recorded as durable command")
	}
}

package factory

import (
	"strings"
	"testing"
	"time"

	"github.com/levitateos/sodaos/internal/identity"
)

func TestCredentialAuthorityRequiresExactWorkspaceBinding(t *testing.T) {
	r := Run{ID: NewID(), AttemptID: NewID(), Role: Implementation, InputSHA: strings.Repeat("a", 40), Started: time.Now(), Deadline: time.Now().Add(time.Hour), Image: "sha256:" + strings.Repeat("b", 64), Harness: "test", Model: "test", IdentityLeaseID: "lease", IdentityGeneration: 1, CredentialDelegated: true}
	id := strings.Repeat("c", 64)
	r.Resources = []Resource{{Kind: "workspace", Name: ResourceName(r.ID, "workspace"), ID: id}}
	r.IdentityBinding = &identity.Binding{Kind: identity.Factory, ID: id, Generation: 1}
	if err := r.Validate(); err != nil {
		t.Fatal(err)
	}
	r.IdentityBinding.ID = strings.Repeat("d", 64)
	if err := r.Validate(); err == nil {
		t.Fatal("different execution binding accepted")
	}
	r.IdentityBinding.ID = id
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

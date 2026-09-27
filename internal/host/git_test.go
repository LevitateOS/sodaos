package host

import (
	"bytes"
	"encoding/json"
	"errors"
	"testing"

	"github.com/levitateos/sodaos/internal/host/terminal"
	"github.com/levitateos/sodaos/internal/identity"
)

func TestGitConfigurationSeparatesLaunchAndBrokerInterfaces(t *testing.T) {
	valid := Config{ForgejoURL: "https://forgejo.example.test", GitSocket: "/run/soda-git-interface/launch.sock", IdentitySocket: "/run/soda-identity/runtime.sock", MuseSocket: "/run/soda-muse-interface/launch.sock"}
	if err := validateGitRuntime(valid); err != nil {
		t.Fatal(err)
	}
	for _, socket := range []string{"relative/launch.sock", "/run/soda-identity/launch.sock", "/run/soda-muse-interface/launch.sock", "/run/soda-git-interface/admin.sock"} {
		invalid := valid
		invalid.GitSocket = socket
		if validateGitRuntime(invalid) == nil {
			t.Fatal("invalid launch interface admitted", socket)
		}
	}
}

func TestGitOriginRejectsUnrelatedRewriteAuthority(t *testing.T) {
	for _, origin := range []string{"", "http://forgejo.example.test", "https://secret@forgejo.example.test", "https://forgejo.example.test/other", "https://forgejo.example.test/?override=1", "https://forgejo.example.test/#fragment"} {
		if validGitOrigin(origin) {
			t.Fatal("invalid rewrite origin admitted", origin)
		}
	}
}

func TestGitDispatchDeniesFactoryAndCredentialCapture(t *testing.T) {
	executor := &terminalFake{}
	d := Daemon{Git: &terminal.GitRuntime{Exec: executor}, Terminal: &terminal.Service{Exec: executor}}
	for _, test := range []struct {
		action, kind, project string
	}{
		{"stop", identity.Factory, "project"},
		{"validate", identity.Terminal, "other-project"},
		{"finish", identity.Terminal, "project"},
		{"start", identity.Terminal, "project"},
	} {
		in := identity.DeliveryWire{Lease: identity.Lease{ProviderID: identity.Forgejo, Kind: test.kind, ProjectID: "project", Binding: &identity.Binding{Project: test.project, Scope: "git"}}}
		body, err := json.Marshal(in)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := d.identityOperation(t.Context(), "/identity/"+test.action, bytes.NewReader(body)); !errors.Is(err, identity.ErrDenied) {
			t.Fatal("invalid Git operation admitted", err)
		}
	}
	if executor.calls != 0 {
		t.Fatal("denied Git request reached native terminal or container")
	}
}

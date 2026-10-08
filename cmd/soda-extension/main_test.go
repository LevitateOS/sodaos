package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	extensions "forgejo.org/extension-sdk"
)

func TestReadOperatorIDRequiresPrivateCanonicalIdentity(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "operator-id")
	if err := os.WriteFile(path, []byte("42\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if got, err := readOperatorID(path); err != nil || got != "42" {
		t.Fatalf("readOperatorID() = %q, %v; want 42", got, err)
	}
	for name, value := range map[string]string{"zero": "0\n", "leading-zero": "042\n", "space": "42 \n", "double-newline": "42\n\n", "overflow": "9223372036854775808\n"} {
		t.Run(name, func(t *testing.T) {
			if err := os.WriteFile(path, []byte(value), 0o600); err != nil {
				t.Fatal(err)
			}
			if got, err := readOperatorID(path); err == nil {
				t.Fatalf("readOperatorID(%q) = %q, want rejection", name, got)
			}
		})
	}
	if err := os.WriteFile(path, []byte("42\n"), 0o666); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0o666); err != nil {
		t.Fatal(err)
	}
	if got, err := readOperatorID(path); err == nil {
		t.Fatalf("world-writable identity accepted as %q", got)
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(directory, "missing"), path); err != nil {
		t.Fatal(err)
	}
	if got, err := readOperatorID(path); err == nil {
		t.Fatalf("symlink identity accepted as %q", got)
	}
}

func TestSodaContributionPolicyKeepsOperatorPagesNarrow(t *testing.T) {
	authorize := contributionAuthorizer("42")
	for _, test := range []struct {
		name         string
		actor        string
		contribution extensions.Contribution
		want         bool
	}{
		{"spaces page", "99", extensions.Contribution{Kind: "page", ID: "spaces", Scope: "global"}, true},
		{"workspace panel", "99", extensions.Contribution{Kind: "panel", ID: "workspace", Scope: "panel"}, true},
		{"retired runners page", "42", extensions.Contribution{Kind: "page", ID: "runners", Scope: "global"}, false},
		{"operator tailnet admin page", "42", extensions.Contribution{Kind: "page", ID: "tailnet", Scope: "admin"}, true},
		{"other tailnet admin actor", "43", extensions.Contribution{Kind: "page", ID: "tailnet", Scope: "admin"}, false},
		{"padded tailnet admin actor", "042", extensions.Contribution{Kind: "page", ID: "tailnet", Scope: "admin"}, false},
		{"tailnet global page", "42", extensions.Contribution{Kind: "page", ID: "tailnet", Scope: "global"}, false},
		{"other runners actor", "43", extensions.Contribution{Kind: "page", ID: "runners", Scope: "global"}, false},
		{"wrong operator scope", "42", extensions.Contribution{Kind: "page", ID: "runners", Scope: "admin"}, false},
		{"unknown page", "42", extensions.Contribution{Kind: "page", ID: "unknown", Scope: "global"}, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			decision, err := authorize(context.Background(), extensions.ContributionRequest{
				ActorID: test.actor, Contribution: test.contribution,
			})
			if err != nil || decision.Allowed != test.want {
				t.Fatalf("authorize = %+v, %v; want allowed=%t", decision, err, test.want)
			}
		})
	}
}

func TestSodaContributionPolicyUsesDeclaredTailnetScope(t *testing.T) {
	manifest, err := extensions.LoadManifest("../../system/containers/extension")
	if err != nil {
		t.Fatal(err)
	}
	var tailnet *extensions.Page
	for i := range manifest.Pages {
		if manifest.Pages[i].ID == "tailnet" {
			tailnet = &manifest.Pages[i]
			break
		}
	}
	if tailnet == nil {
		t.Fatal("extension manifest has no Tailnet page")
	}
	contribution := extensions.Contribution{Kind: "page", ID: tailnet.ID, Scope: tailnet.Scope}
	authorize := contributionAuthorizer("42")
	for _, actor := range []struct {
		id   string
		want bool
	}{{"42", true}, {"43", false}} {
		decision, err := authorize(context.Background(), extensions.ContributionRequest{
			ActorID: actor.id, Contribution: contribution,
		})
		if err != nil || decision.Allowed != actor.want {
			t.Fatalf("authorize manifest Tailnet page for actor %s = %+v, %v; want allowed=%t", actor.id, decision, err, actor.want)
		}
	}
}

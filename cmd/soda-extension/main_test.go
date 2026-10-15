package main

import (
	"context"
	"testing"

	extensions "forgejo.org/extension-sdk"
)

func TestSodaContributionPolicyKeepsOperatorPagesNarrow(t *testing.T) {
	authorize := contributionAuthorizer()
	for _, test := range []struct {
		name         string
		actor        string
		contribution extensions.Contribution
		want         bool
	}{
		{"spaces page", "99", extensions.Contribution{Kind: "page", ID: "spaces", Scope: "global"}, true},
		{"workspace panel", "99", extensions.Contribution{Kind: "panel", ID: "workspace", Scope: "panel"}, true},
		{"retired runners page", "42", extensions.Contribution{Kind: "page", ID: "runners", Scope: "global"}, false},
		{"operator tailnet page", "42", extensions.Contribution{Kind: "page", ID: "tailnet", Scope: "global"}, false},
		{"other tailnet actor", "43", extensions.Contribution{Kind: "page", ID: "tailnet", Scope: "global"}, false},
		{"padded tailnet actor", "042", extensions.Contribution{Kind: "page", ID: "tailnet", Scope: "global"}, false},
		{"tailnet admin scope", "42", extensions.Contribution{Kind: "page", ID: "tailnet", Scope: "admin"}, false},
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

func TestPackageApplicationsSeparateControllerAuthority(t *testing.T) {
	for _, id := range []string{"soda", "tailnet"} {
		app, closeTransport, err := packageApplication(id, "/ipc/soda.sock")
		if err != nil {
			t.Fatal(err)
		}
		defer closeTransport()
		if app.HTTP == nil {
			t.Fatal("missing HTTP transport")
		}
		if id == "soda" {
			if app.Lifecycle == nil || app.AuthorizeContribution == nil || len(app.Policies) != 1 {
				t.Fatal("required controller missing")
			}
		} else if app.Lifecycle != nil || app.AuthorizeContribution != nil || len(app.Policies) != 0 {
			t.Fatal("optional package has controller authority")
		}
	}
	for _, id := range []string{"", "other", "tailnet-extension"} {
		if _, _, err := packageApplication(id, "/ipc/soda.sock"); err == nil {
			t.Fatalf("accepted %q", id)
		}
	}
}

func TestPackageManifestsAreAdmittedByNativeSDK(t *testing.T) {
	for _, path := range []string{"../../system/containers/extension", "../../system/containers/extension/tailnet-package"} {
		manifest, err := extensions.LoadManifest(path)
		if err != nil {
			t.Fatalf("native SDK rejected %s: %v", path, err)
		}
		if manifest.ID == "tailnet" && (len(manifest.Pages) != 1 || manifest.Pages[0].Scope != "admin" || manifest.Pages[0].Route != "" || len(manifest.Policies) != 0) {
			t.Fatal("optional package has workspace authority")
		}
	}
}

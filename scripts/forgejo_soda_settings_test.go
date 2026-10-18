package scripts

import (
	"encoding/json"
	"os"
	"testing"
)

func TestSodaNavigationComesFromForgejoExtensionManifest(t *testing.T) {
	readManifest := func(rel string) []byte {
		t.Helper()
		manifestBytes, err := os.ReadFile(rel)
		if err != nil {
			t.Fatal(err)
		}
		return manifestBytes
	}
	manifestBytes := readManifest("../system/containers/extension/extension.json")
	var manifest struct {
		Pages []struct {
			ID         string `json:"id"`
			Scope      string `json:"scope"`
			Permission string `json:"permission"`
		} `json:"pages"`
		Panels []struct {
			ID string `json:"id"`
		} `json:"panels"`
		PreferredWorkspace bool `json:"preferred_workspace"`
	}
	if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
		t.Fatal(err)
	}
	pages := make(map[string]struct {
		scope      string
		permission string
	}, len(manifest.Pages))
	for _, page := range manifest.Pages {
		pages[page.ID] = struct {
			scope      string
			permission string
		}{page.Scope, page.Permission}
	}
	if _, retired := pages["runners"]; retired {
		t.Fatalf("retired runners page still declared: %+v", pages)
	}
	if pages["spaces"].scope != "global" {
		t.Fatalf("unexpected core page declarations: %+v", pages)
	}
	if _, present := pages["tailnet"]; present {
		t.Fatalf("tailnet page rides the required manifest: %+v", pages)
	}
	// The tailnet page is contributed only by the separately activatable
	// optional package, under its admin-gated declaration.
	var optional struct {
		Pages []struct {
			ID         string `json:"id"`
			Scope      string `json:"scope"`
			Route      string `json:"route"`
			Permission string `json:"permission"`
		} `json:"pages"`
	}
	if err := json.Unmarshal(readManifest("../system/containers/extension/tailnet-package/extension.json"), &optional); err != nil {
		t.Fatal(err)
	}
	if len(optional.Pages) != 1 || optional.Pages[0].ID != "tailnet" || optional.Pages[0].Scope != "admin" || optional.Pages[0].Route != "" || optional.Pages[0].Permission == "" {
		t.Fatalf("unexpected optional tailnet page: %+v", optional.Pages)
	}
	workspacePanel := false
	for _, panel := range manifest.Panels {
		workspacePanel = workspacePanel || panel.ID == "workspace"
	}
	if !workspacePanel || !manifest.PreferredWorkspace {
		t.Fatal("workspace panel is not declared as the preferred workspace")
	}
}

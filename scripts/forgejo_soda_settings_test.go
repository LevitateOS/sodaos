package scripts

import (
	"encoding/json"
	"os"
	"testing"
)

func TestSodaNavigationComesFromForgejoExtensionManifest(t *testing.T) {
	manifestBytes, err := os.ReadFile("../appliance/soda-extension/extension.json")
	if err != nil {
		t.Fatal(err)
	}
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
	if pages["spaces"].scope != "global" || pages["tailnet"].scope != "admin" || pages["tailnet"].permission != "admin" {
		t.Fatalf("unexpected extension page declarations: %+v", pages)
	}
	workspacePanel := false
	for _, panel := range manifest.Panels {
		workspacePanel = workspacePanel || panel.ID == "workspace"
	}
	if !workspacePanel || !manifest.PreferredWorkspace {
		t.Fatal("workspace panel is not declared as the preferred workspace")
	}
}

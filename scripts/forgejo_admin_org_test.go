package scripts

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// These hashes pin the exact embedded Forgejo 15.0.7 templates inspected when
// the overrides were authored. The tests remove only Soda's presentation deltas
// before comparing, so native gates and hooks cannot drift unnoticed.
func TestForgejoAdminOrganizationOverridesRetain1507Source(t *testing.T) {
	tests := []struct {
		name         string
		path         []string
		upstreamHash string
		normalize    func(string) string
	}{
		{
			name:         "administrator layout",
			path:         []string{"admin", "layout_head.tmpl"},
			upstreamHash: "71fb21cdb74bdf890fdd0e0b56522437776cbddb7c289e8663ec3a00c3b37c0a",
			normalize: func(contents string) string {
				contents = strings.TrimPrefix(contents, `{{/* Soda presentation override of Forgejo 15.0.7 admin/layout_head.tmpl. */}}`+"\n")
				contents = strings.Replace(contents,
					`class="page-content {{.pageClass}} soda-page soda-admin soda-native-forms" data-signed="true"`,
					`class="page-content {{.pageClass}}"`, 1)
				contents = strings.Replace(contents,
					`class="ui container flex-container soda-page-container soda-admin-layout"`,
					`class="ui container flex-container"`, 1)
				contents = strings.Replace(contents,
					"\t\t"+`{{template "custom/soda/page_intro" dict "TitleID" "soda-admin-title" "Eyebrow" "Forgejo administration" "Title" .ctxData.Title "Description" "Manage Forgejo settings, identities, integrations, and service health." "Artwork" "dashboard-papercraft.png" "Class" "soda-page-intro--compact"}}`+"\n",
					"", 1)
				return contents
			},
		},
		{
			name:         "organization header",
			path:         []string{"org", "header.tmpl"},
			upstreamHash: "12daa767b4898ea9bbe2794fa2da3c04c17dc1ad4555c14d10219247198e3b29",
			normalize: func(contents string) string {
				contents = strings.TrimPrefix(contents, `{{/* Soda presentation override of Forgejo 15.0.7 org/header.tmpl. */}}`+"\n")
				return strings.Replace(contents,
					`class="ui container tw-flex tw-gap-x-4 soda-page-marker soda-org-header" data-signed="{{.IsSigned}}"`,
					`class="ui container tw-flex tw-gap-x-4"`, 1)
			},
		},
		{
			name:         "organization settings layout",
			path:         []string{"org", "settings", "layout_head.tmpl"},
			upstreamHash: "a24850ec5e6716926dcbe1ef777ed328af3a05ba7a918f162fdeebcdccd84672",
			normalize: func(contents string) string {
				contents = strings.TrimPrefix(contents, `{{/* Soda presentation override of Forgejo 15.0.7 org/settings/layout_head.tmpl. */}}`+"\n")
				contents = strings.Replace(contents,
					`class="page-content {{.pageClass}} soda-page soda-org-settings soda-native-forms" data-signed="true"`,
					`class="page-content {{.pageClass}}"`, 1)
				return strings.Replace(contents,
					`class="ui container flex-container soda-page-container soda-org-settings-layout"`,
					`class="ui container flex-container"`, 1)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			contents := readForgejoTemplate(t, tt.path...)
			normalized := tt.normalize(contents)
			got := fmt.Sprintf("%x", sha256.Sum256([]byte(normalized)))
			if got != tt.upstreamHash {
				t.Fatalf("override differs from pinned Forgejo 15.0.7 source beyond its reviewed Soda presentation delta: got %s, want %s", got, tt.upstreamHash)
			}
		})
	}
}

func TestForgejoAdminOrganizationCompositionBoundaries(t *testing.T) {
	admin := readForgejoTemplate(t, "admin", "layout_head.tmpl")
	for _, marker := range []string{
		`{{template "admin/navbar" .ctxData}}`,
		`{{template "base/alert" .ctxData}}`,
		`{{/* block: admin-setting-content */}}`,
		`soda-page soda-admin soda-native-forms`,
	} {
		if !strings.Contains(admin, marker) {
			t.Errorf("administrator layout lost native composition marker %q", marker)
		}
	}

	header := readForgejoTemplate(t, "org", "header.tmpl")
	for _, marker := range []string{
		`{{if .IsSigned}}`,
		`{{if .IsOrganizationMember}}`,
		`{{if $moderationEntryNeeded}}`,
		`{{if .Org.Email}}`,
		`{{template "org/follow_unfollow" .}}`,
		`{{template "org/menu" .}}`,
		`soda-page-marker soda-org-header`,
		`data-signed="{{.IsSigned}}"`,
	} {
		if !strings.Contains(header, marker) {
			t.Errorf("organization header lost native gate or composition marker %q", marker)
		}
	}

	settings := readForgejoTemplate(t, "org", "settings", "layout_head.tmpl")
	for _, marker := range []string{
		`{{template "org/header" .ctxData}}`,
		`{{template "org/settings/navbar" .ctxData}}`,
		`{{template "base/alert" .ctxData}}`,
		`{{/* block: org-setting-content */}}`,
		`soda-page soda-org-settings soda-native-forms`,
	} {
		if !strings.Contains(settings, marker) {
			t.Errorf("organization settings layout lost native composition marker %q", marker)
		}
	}
}

func TestForgejoAdminOrganizationStylesStayFamilyScoped(t *testing.T) {
	styles := []struct {
		name    string
		markers []string
	}{
		{name: "admin.css", markers: []string{".soda-admin-layout", ".soda-admin .admin-setting-content", "var(--soda-page-surface)", "@media (max-width: 700px)"}},
		{name: "organization.css", markers: []string{".soda-org-settings-layout", ".page-content.organization:has(.soda-org-header)", "> .ui.container:not(.fluid)", "var(--soda-page-surface)", "@media (max-width: 700px)"}},
	}
	for _, style := range styles {
		path := filepath.Join("..", "assets", "branding", "forgejo", style.name)
		contents, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		css := string(contents)
		for _, marker := range style.markers {
			if !strings.Contains(css, marker) {
				t.Errorf("%s lost %q", style.name, marker)
			}
		}
		for _, forbidden := range []string{".ui.red.button {", ".danger.button {", ".delete-button {"} {
			if strings.Contains(css, forbidden) {
				t.Errorf("%s must preserve native destructive control styling; found broad selector %q", style.name, forbidden)
			}
		}
	}
}

func TestForgejoSettingsComponentKeepsNativeBoundaries(t *testing.T) {
	path := filepath.Join("..", "assets", "branding", "forgejo", "components-settings.css")
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	css := string(contents)
	for _, marker := range []string{
		`:is(.soda-settings-layout, .soda-admin-layout, .soda-org-settings-layout) > .flex-container-nav > .ui.vertical.menu`,
		`details.item > summary`,
		`:is(.user-setting-content, .repo-setting-content, .user-main-content, .admin-setting-content, .org-setting-content) > .ui.top.attached.header`,
		`> .ui.attached.segment:not(table)`,
		`:is(.soda-config-heading, .soda-webhook-heading).ui.header`,
		`var(--soda-page-selected)`,
	} {
		if !strings.Contains(css, marker) {
			t.Errorf("shared settings stylesheet lost native composition scope %q", marker)
		}
	}
	for _, forbidden := range []string{
		".ui.form",
		".primary.button",
		".danger.button",
		".delete-button",
		"body:has(",
		".soda-config-list > .flex-list",
		".soda-blocked-users > .flex-item",
		".soda-quota-overview details.stats",
	} {
		if strings.Contains(css, forbidden) {
			t.Errorf("shared settings stylesheet exceeds its navigation/card contract with %q", forbidden)
		}
	}
}

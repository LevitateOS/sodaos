package scripts

import (
	"crypto/sha256"
	"fmt"
	"html/template"
	"strings"
	"testing"
)

func TestForgejoOrgDetailsSourceParityAndParse(t *testing.T) {
	cases := []struct {
		name, sha, provenance string
		edits                 [][2]string
	}{{name: "team/new.tmpl", sha: "eaaa7e2e9c1607b8022fa2cc121045f9a7836ce223f23ff4fc014ac370945c7a", provenance: "{{/* Soda presentation of Forgejo 15.0.7 templates/org/team/new.tmpl; GPL-3.0-or-later. Upstream SHA-256: eaaa7e2e9c1607b8022fa2cc121045f9a7836ce223f23ff4fc014ac370945c7a. */}}\n", edits: [][2]string{{"<div role=\"main\" aria-label=\"{{.Title}}\" class=\"page-content organization new team soda-page soda-org-details\" data-signed=\"{{if .IsSigned}}true{{else}}false{{end}}\">", "<div role=\"main\" aria-label=\"{{.Title}}\" class=\"page-content organization new team\">"},
		{"{{template \"org/header\" .}}\n\t<div class=\"ui container soda-page-container soda-org-details-intro\">{{template \"custom/soda/page_intro\" dict \"TitleID\" \"soda-org-detail-title\" \"Title\" .Title \"Class\" \"soda-page-intro--compact\"}}</div>", "{{template \"org/header\" .}}"},
		{"class=\"ui form soda-form\"", "class=\"ui form\""}}},
		{name: "team/teams.tmpl", sha: "9fd8917ee15655ca2be5184e08cf84be27beca6783e914d546b32c6d81715b79", provenance: "{{/* Soda presentation of Forgejo 15.0.7 templates/org/team/teams.tmpl; GPL-3.0-or-later. Upstream SHA-256: 9fd8917ee15655ca2be5184e08cf84be27beca6783e914d546b32c6d81715b79. */}}\n", edits: [][2]string{{"<div role=\"main\" aria-label=\"{{.Title}}\" class=\"page-content organization teams soda-page soda-org-details\" data-signed=\"{{if .IsSigned}}true{{else}}false{{end}}\">", "<div role=\"main\" aria-label=\"{{.Title}}\" class=\"page-content organization teams\">"},
			{"{{template \"org/header\" .}}\n\t<div class=\"ui container soda-page-container soda-org-details-intro\">{{template \"custom/soda/page_intro\" dict \"TitleID\" \"soda-org-detail-title\" \"Title\" .Title \"Class\" \"soda-page-intro--compact\"}}</div>", "{{template \"org/header\" .}}"}}},
		{name: "team/members.tmpl", sha: "f8240cd7264e2e37585336c6cd3bfbd9f73c0f9dba294c80f93df2e092e057e9", provenance: "{{/* Soda presentation of Forgejo 15.0.7 templates/org/team/members.tmpl; GPL-3.0-or-later. Upstream SHA-256: f8240cd7264e2e37585336c6cd3bfbd9f73c0f9dba294c80f93df2e092e057e9. */}}\n", edits: [][2]string{{"<div role=\"main\" aria-label=\"{{.Title}}\" class=\"page-content organization teams soda-page soda-org-details\" data-signed=\"{{if .IsSigned}}true{{else}}false{{end}}\">", "<div role=\"main\" aria-label=\"{{.Title}}\" class=\"page-content organization teams\">"},
			{"{{template \"org/header\" .}}\n\t<div class=\"ui container soda-page-container soda-org-details-intro\">{{template \"custom/soda/page_intro\" dict \"TitleID\" \"soda-org-detail-title\" \"Title\" .Title \"Class\" \"soda-page-intro--compact\"}}</div>", "{{template \"org/header\" .}}"}}},
		{name: "team/repositories.tmpl", sha: "945baec05eb8a32f4501bc5e1da240f40f25bff5bf2a67f094bf98f3f7d23947", provenance: "{{/* Soda presentation of Forgejo 15.0.7 templates/org/team/repositories.tmpl; GPL-3.0-or-later. Upstream SHA-256: 945baec05eb8a32f4501bc5e1da240f40f25bff5bf2a67f094bf98f3f7d23947. */}}\n", edits: [][2]string{{"<div role=\"main\" aria-label=\"{{.Title}}\" class=\"page-content organization teams soda-page soda-org-details\" data-signed=\"{{if .IsSigned}}true{{else}}false{{end}}\">", "<div role=\"main\" aria-label=\"{{.Title}}\" class=\"page-content organization teams\">"},
			{"{{template \"org/header\" .}}\n\t<div class=\"ui container soda-page-container soda-org-details-intro\">{{template \"custom/soda/page_intro\" dict \"TitleID\" \"soda-org-detail-title\" \"Title\" .Title \"Class\" \"soda-page-intro--compact\"}}</div>", "{{template \"org/header\" .}}"}}},
		{name: "member/members.tmpl", sha: "9422d47081088f224721b13574d0d2597f9e1ce809d20e43cb9ffbf9b56a9afa", provenance: "{{/* Soda presentation of Forgejo 15.0.7 templates/org/member/members.tmpl; GPL-3.0-or-later. Upstream SHA-256: 9422d47081088f224721b13574d0d2597f9e1ce809d20e43cb9ffbf9b56a9afa. */}}\n", edits: [][2]string{{"<div role=\"main\" aria-label=\"{{.Title}}\" class=\"page-content organization members soda-page soda-org-details\" data-signed=\"{{if .IsSigned}}true{{else}}false{{end}}\">", "<div role=\"main\" aria-label=\"{{.Title}}\" class=\"page-content organization members\">"},
			{"{{template \"org/header\" .}}\n\t<div class=\"ui container soda-page-container soda-org-details-intro\">{{template \"custom/soda/page_intro\" dict \"TitleID\" \"soda-org-detail-title\" \"Title\" .Title \"Class\" \"soda-page-intro--compact\"}}</div>", "{{template \"org/header\" .}}"}}},
		{name: "team/invite.tmpl", sha: "e5eb578f3ddf9e574099bc803a48bdba4582b0bf82f8644bd050e74b20506ba5", provenance: "{{/* Soda presentation of Forgejo 15.0.7 templates/org/team/invite.tmpl; GPL-3.0-or-later. Upstream SHA-256: e5eb578f3ddf9e574099bc803a48bdba4582b0bf82f8644bd050e74b20506ba5. */}}\n", edits: [][2]string{{"<div role=\"main\" aria-label=\"{{.Title}}\" class=\"page-content organization invite soda-page soda-org-details\" data-signed=\"{{if .IsSigned}}true{{else}}false{{end}}\">", "<div role=\"main\" aria-label=\"{{.Title}}\" class=\"page-content organization invite\">"},
			{"class=\"ui form soda-form\"", "class=\"ui form\""}}}}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			src := readForgejoTemplate(t, append([]string{"org"}, strings.Split(tt.name, "/")...)...)
			if !strings.HasPrefix(src, tt.provenance) {
				t.Fatal("missing exact attribution")
			}
			restored := strings.TrimPrefix(src, tt.provenance)
			for _, edit := range tt.edits {
				if strings.Count(restored, edit[0]) != 1 {
					t.Fatalf("unexpected presentation edit %q", edit[0])
				}
				restored = strings.Replace(restored, edit[0], edit[1], 1)
			}
			if got := fmt.Sprintf("%x", sha256.Sum256([]byte(restored))); got != tt.sha {
				t.Fatalf("native markup differs: %s", got)
			}
			funcs := template.FuncMap{"dict": forgejoTemplateDict, "ctx": func() any { return nil }, "AppSubUrl": func() string { return "/forge" }, "svg": func(_ string, _ ...any) string { return "icon" }, "TrustHTML": func(s string) template.HTML { return template.HTML(s) }, "PathEscape": func(s string) string { return s }, "QueryEscape": func(s string) string { return s }}
			if _, err := template.New(tt.name).Funcs(funcs).Parse(src); err != nil {
				t.Fatal(err)
			}
		})
	}
}

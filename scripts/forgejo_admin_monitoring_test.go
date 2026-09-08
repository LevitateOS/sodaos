package scripts

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"html/template"
	"strings"
	"testing"
)

func TestForgejoAdminMonitoringStockParityAndParse(t *testing.T) {
	cases := []struct {
		name, sha, provenance, stockSuffix string
		replacements                       [][2]string
	}{
		{name: "cron.tmpl", sha: "7469259693b0678847af6cb18b91e82b235fc64436917d71b31f5f15fcc9a996", provenance: "{{/* Soda presentation of Forgejo 15.0.7 templates/admin/cron.tmpl; GPL-3.0-or-later. Upstream SHA-256: 7469259693b0678847af6cb18b91e82b235fc64436917d71b31f5f15fcc9a996. */}}\n", replacements: [][2]string{{"class=\"admin-setting-content soda-admin-monitoring soda-admin-monitoring--cron\"", "class=\"admin-setting-content\""},
			{"class=\"ui attached table segment soda-monitoring-table\"", "class=\"ui attached table segment\""}}},
		{name: "queue.tmpl", stockSuffix: "\n", sha: "3c6cfd943d32b45d9f2c58633875d4ede46d14be1c31745720fe72b99fd5ed3f", provenance: "{{/* Soda presentation of Forgejo 15.0.7 templates/admin/queue.tmpl; GPL-3.0-or-later. Upstream SHA-256: 3c6cfd943d32b45d9f2c58633875d4ede46d14be1c31745720fe72b99fd5ed3f. */}}\n", replacements: [][2]string{{"class=\"admin-setting-content soda-admin-monitoring soda-admin-monitoring--queue\"", "class=\"admin-setting-content\""},
			{"class=\"ui attached table segment soda-monitoring-table\"", "class=\"ui attached table segment\""}}},
		{name: "queue_manage.tmpl", sha: "c65151cd142ef86bcd371f6b2fc25b19d6e74aec0924c092daa3c6296171977e", provenance: "{{/* Soda presentation of Forgejo 15.0.7 templates/admin/queue_manage.tmpl; GPL-3.0-or-later. Upstream SHA-256: c65151cd142ef86bcd371f6b2fc25b19d6e74aec0924c092daa3c6296171977e. */}}\n", replacements: [][2]string{{"class=\"admin-setting-content soda-admin-monitoring soda-admin-monitoring--queue-manage\"", "class=\"admin-setting-content\""},
			{"class=\"ui attached table segment soda-monitoring-table\"", "class=\"ui attached table segment\""},
			{"class=\"ui form soda-form\"", "class=\"ui form\""}}},
		{name: "notice.tmpl", sha: "5483e3fbd3133b12b8d1e901751e2a966a0340c2af572ca9176d52ed2b97b19f", provenance: "{{/* Soda presentation of Forgejo 15.0.7 templates/admin/notice.tmpl; GPL-3.0-or-later. Upstream SHA-256: 5483e3fbd3133b12b8d1e901751e2a966a0340c2af572ca9176d52ed2b97b19f. */}}\n", replacements: [][2]string{{"class=\"admin-setting-content soda-admin-monitoring soda-admin-monitoring--notice\"", "class=\"admin-setting-content\""}}},
		{name: "stacktrace.tmpl", sha: "f31011de3ff30d2e79702900e9271ec1bda0020cfe1c33841e50dac89168930e", provenance: "{{/* Soda presentation of Forgejo 15.0.7 templates/admin/stacktrace.tmpl; GPL-3.0-or-later. Upstream SHA-256: f31011de3ff30d2e79702900e9271ec1bda0020cfe1c33841e50dac89168930e. */}}\n", replacements: [][2]string{{"class=\"admin-setting-content soda-admin-monitoring soda-admin-monitoring--stacktrace\"", "class=\"admin-setting-content\""},
			{"class=\"ui form soda-form tw-flex tw-gap-3\"", "class=\"ui form tw-flex tw-gap-3\""}}},
		{name: "self_check.tmpl", sha: "d3160640928f95ef34c7a44767ffec4f1a8406ae78b67d3523f97d6820703c28", provenance: "{{/* Soda presentation of Forgejo 15.0.7 templates/admin/self_check.tmpl; GPL-3.0-or-later. Upstream SHA-256: d3160640928f95ef34c7a44767ffec4f1a8406ae78b67d3523f97d6820703c28. */}}\n", replacements: [][2]string{{"class=\"admin-setting-content soda-admin-monitoring soda-admin-monitoring--self-check\"", "class=\"admin-setting-content\""}}},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			src := readForgejoTemplate(t, "admin", tt.name)
			if !strings.HasPrefix(src, tt.provenance) {
				t.Fatal("missing version/license/source attribution")
			}
			restored := strings.TrimPrefix(src, tt.provenance)
			if tt.name == "queue.tmpl" || tt.name == "queue_manage.tmpl" {
				restored = strings.Replace(restored, ` "hideArtwork" true`, "", 1)
			}
			for _, r := range tt.replacements {
				if strings.Count(restored, r[0]) != 1 {
					t.Fatalf("unexpected presentation delta %q", r[0])
				}
				restored = strings.Replace(restored, r[0], r[1], 1)
			}
			restored += tt.stockSuffix
			if got := fmt.Sprintf("%x", sha256.Sum256([]byte(restored))); got != tt.sha {
				t.Fatalf("native markup changed outside declared presentation: %s", got)
			}
			if _, err := template.New(tt.name).Funcs(forgejoMonitoringFunctions()).Parse(src); err != nil {
				t.Fatal(err)
			}
		})
	}
}

type forgejoMonitoringLocale struct{}

func (forgejoMonitoringLocale) Tr(key string, _ ...any) string { return key }

type forgejoMonitoringContext struct{ Locale forgejoMonitoringLocale }

func forgejoMonitoringFunctions() template.FuncMap {
	return template.FuncMap{
		"ctx":       func() forgejoMonitoringContext { return forgejoMonitoringContext{} },
		"dict":      forgejoTemplateDict,
		"AppSubUrl": func() string { return "/forge" },
		"svg":       func(_ string, _ ...any) string { return "native-icon" },
		"TrustHTML": func(s string) template.HTML { return template.HTML(s) },
		"DateUtils": func() any { return nil },
	}
}

func TestForgejoAdminMonitoringPreservesNativeWarningsAndEmptyNotice(t *testing.T) {
	seams := `{{define "admin/layout_head"}}native-admin-head{{end}}{{define "admin/layout_footer"}}native-admin-footer{{end}}{{define "base/paginate"}}native-pagination{{end}}`
	for _, tt := range []struct {
		name   string
		data   map[string]any
		want   []string
		absent []string
	}{
		{name: "self_check", data: map[string]any{}, want: []string{"admin.self_check.no_problem_found"}, absent: []string{"ui red message", "ui warning message"}},
		{name: "self_check", data: map[string]any{"CacheError": "failed", "CacheSlow": true}, want: []string{"ui red message", "admin.config.cache_test_failed", "ui warning message", "admin.config.cache_test_slow"}},
		{name: "notice", data: map[string]any{}, want: []string{"repo.pulls.no_results", "native-pagination", `id="detail-modal"`}, absent: []string{`id="delete-selection"`, `action="/forge/admin/notices/empty"`}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			parsed, err := template.New("page").Funcs(forgejoMonitoringFunctions()).Parse(seams + readForgejoTemplate(t, "admin", tt.name+".tmpl"))
			if err != nil {
				t.Fatal(err)
			}
			var output bytes.Buffer
			if err := parsed.ExecuteTemplate(&output, "page", tt.data); err != nil {
				t.Fatal(err)
			}
			for _, want := range tt.want {
				if !strings.Contains(output.String(), want) {
					t.Errorf("missing native branch %q", want)
				}
			}
			for _, absent := range tt.absent {
				if strings.Contains(output.String(), absent) {
					t.Errorf("unexpected native action %q", absent)
				}
			}
		})
	}
}

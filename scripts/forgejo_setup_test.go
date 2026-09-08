package scripts

import (
	"bytes"
	"html/template"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type forgejoSetupLocale struct{}

func (forgejoSetupLocale) Tr(key string, _ ...any) string { return key }
func (forgejoSetupLocale) TrString(key string) string     { return key }

type forgejoSetupContext struct {
	Locale forgejoSetupLocale
}

func TestForgejoSetupKeepsNativeFieldAndHookContract(t *testing.T) {
	contents := readForgejoTemplate(t, "install.tmpl")

	fields := []struct {
		name     string
		contract string
	}{
		{"db_type", `type="hidden" id="db_type" name="db_type" value="{{.CurDbType}}"`},
		{"db_host", `id="db_host" name="db_host" value="{{.db_host}}"`},
		{"db_user", `id="db_user" name="db_user" value="{{.db_user}}"`},
		{"db_passwd", `id="db_passwd" name="db_passwd" type="password" value="{{.db_passwd}}"`},
		{"db_name", `id="db_name" name="db_name" value="{{.db_name}}"`},
		{"ssl_mode", `name="ssl_mode" value="{{if .ssl_mode}}{{.ssl_mode}}{{else}}disable{{end}}"`},
		{"db_schema", `id="db_schema" name="db_schema" value="{{.db_schema}}"`},
		{"db_path", `id="db_path" name="db_path" value="{{.db_path}}"`},
		{"app_name", `id="app_name" name="app_name" value="{{.app_name}}" required`},
		{"app_slogan", `id="app_slogan" name="app_slogan" value="{{.app_slogan}}"`},
		{"repo_root_path", `id="repo_root_path" name="repo_root_path" value="{{.repo_root_path}}" required`},
		{"lfs_root_path", `id="lfs_root_path" name="lfs_root_path" value="{{.lfs_root_path}}"`},
		{"run_user", `id="run_user" name="run_user" value="{{.run_user}}" readonly`},
		{"domain", `id="domain" name="domain" value="{{.domain}}" placeholder="next.forgejo.org" required`},
		{"ssh_port", `id="ssh_port" name="ssh_port" value="{{.ssh_port}}"`},
		{"http_port", `id="http_port" name="http_port" value="{{.http_port}}" required`},
		{"app_url", `id="app_url" name="app_url" value="{{.app_url}}" placeholder="https://next.forgejo.org" required`},
		{"log_root_path", `id="log_root_path" name="log_root_path" value="{{.log_root_path}}" placeholder="log" required`},
		{"disable_registration", `name="disable_registration" type="checkbox" {{if .disable_registration}}checked{{end}}`},
		{"enable_update_checker", `name="enable_update_checker" type="checkbox" {{if .enable_update_checker}}checked{{end}}`},
		{"smtp_addr", `id="smtp_addr" name="smtp_addr" value="{{.smtp_addr}}"`},
		{"smtp_port", `id="smtp_port" name="smtp_port" value="{{.smtp_port}}"`},
		{"smtp_from", `id="smtp_from" name="smtp_from" value="{{.smtp_from}}"`},
		{"smtp_user", `id="smtp_user" name="smtp_user" value="{{.smtp_user}}"`},
		{"smtp_passwd", `id="smtp_passwd" name="smtp_passwd" type="password" value="{{.smtp_passwd}}"`},
		{"register_confirm", `name="register_confirm" type="checkbox" {{if .register_confirm}}checked{{end}}`},
		{"mail_notify", `name="mail_notify" type="checkbox" {{if .mail_notify}}checked{{end}}`},
		{"offline_mode", `name="offline_mode" type="checkbox" {{if .offline_mode}}checked{{end}}`},
		{"disable_gravatar", `name="disable_gravatar" type="checkbox" {{if .disable_gravatar}}checked{{end}}`},
		{"enable_federated_avatar", `name="enable_federated_avatar" type="checkbox" {{if .enable_federated_avatar}}checked{{end}}`},
		{"enable_open_id_sign_in", `name="enable_open_id_sign_in" type="checkbox" {{if .enable_open_id_sign_in}}checked{{end}}`},
		{"allow_only_external_registration", `name="allow_only_external_registration" type="checkbox" {{if .allow_only_external_registration}}checked{{end}}`},
		{"enable_open_id_sign_up", `name="enable_open_id_sign_up" type="checkbox" {{if .enable_open_id_sign_up}}checked{{end}}`},
		{"enable_captcha", `name="enable_captcha" type="checkbox" {{if .enable_captcha}}checked{{end}}`},
		{"require_sign_in_view", `name="require_sign_in_view" type="checkbox" {{if .require_sign_in_view}}checked{{end}}`},
		{"default_keep_email_private", `name="default_keep_email_private" type="checkbox" {{if .default_keep_email_private}}checked{{end}}`},
		{"default_allow_create_organization", `name="default_allow_create_organization" type="checkbox" {{if .default_allow_create_organization}}checked{{end}}`},
		{"default_enable_timetracking", `name="default_enable_timetracking" type="checkbox" {{if .default_enable_timetracking}}checked{{end}}`},
		{"no_reply_address", `id="_no_reply_address" name="no_reply_address" value="{{.no_reply_address}}"`},
		{"password_algorithm", `id="password_algorithm" type="hidden" name="password_algorithm" value="{{.password_algorithm}}"`},
		{"admin_name", `id="admin_name" name="admin_name" value="{{.admin_name}}"`},
		{"admin_email", `id="admin_email" name="admin_email" type="email" value="{{.admin_email}}"`},
		{"admin_passwd", `id="admin_passwd" name="admin_passwd" type="password" autocomplete="new-password" value="{{.admin_passwd}}"`},
		{"admin_confirm_passwd", `id="admin_confirm_passwd" name="admin_confirm_passwd" autocomplete="new-password" type="password" value="{{.admin_confirm_passwd}}"`},
	}
	for _, field := range fields {
		t.Run(field.name, func(t *testing.T) {
			if !strings.Contains(contents, field.contract) {
				t.Errorf("install.tmpl lost native %s contract %q", field.name, field.contract)
			}
		})
	}

	for _, contract := range []string{
		`action="{{AppSubUrl}}/" method="post"`,
		`data-db-setting-for="common-host"`,
		`data-db-setting-for="postgres"`,
		`data-db-setting-for="sqlite3"`,
		`{{range .DbTypeNames}}`,
		`{{range .PasswordHashAlgorithms}}`,
		`{{if .Err_DbInstalledBefore}}`,
		`{{if .EnvConfigKeys}}`,
		`{{range .EnvConfigKeys}}`,
		`{{.CustomConfFile}}`,
		`src="{{AssetUrlPrefix}}/img/forgejo-loading.svg"`,
		`{{ctx.Locale.Tr "install.install_btn_confirm"}}`,
	} {
		if !strings.Contains(contents, contract) {
			t.Errorf("install.tmpl lost native setup hook %q", contract)
		}
	}
	for _, confirmation := range []string{"reinstall_confirm_first", "reinstall_confirm_second", "reinstall_confirm_third"} {
		if strings.Count(contents, `name="`+confirmation+`"`) != 1 {
			t.Errorf("install.tmpl must retain exactly one %s checkbox", confirmation)
		}
	}

	if !strings.Contains(contents, `class="page-content install soda-page soda-forgejo-setup" data-signed="false"`) {
		t.Error("install.tmpl is not an explicit anonymous full-page Soda shell")
	}
	if !strings.Contains(contents, `class="ui form soda-form"`) {
		t.Error("install.tmpl does not opt the native installer into shared form presentation")
	}
	if !templateCalls(contents)["custom/soda/page_intro"] {
		t.Error("install.tmpl does not compose the shared page intro")
	}
	if !templateCalls(contents)["custom/soda/theme_toggle"] || strings.Count(contents, `{{template "custom/soda/theme_toggle"`) != 1 {
		t.Error("install.tmpl must render exactly one shared guest theme toggle")
	}
}

func TestForgejoSetupRendersNativeConditionalBranches(t *testing.T) {
	contents := readForgejoTemplate(t, "install.tmpl")
	definitions := `
		{{define "base/head"}}native-head{{end}}
		{{define "base/footer"}}native-footer{{end}}
		{{define "base/alert"}}native-alert{{end}}
		{{define "custom/soda/page_intro"}}soda-intro/{{.Title}}{{end}}
		{{define "custom/soda/theme_toggle"}}soda-theme/{{.Class}}{{end}}
		{{define "page"}}` + contents + `{{end}}`
	functions := template.FuncMap{
		"AppSubUrl":      func() string { return "/forge" },
		"AssetUrlPrefix": func() string { return "/forge/assets" },
		"ctx":            func() forgejoSetupContext { return forgejoSetupContext{Locale: forgejoSetupLocale{}} },
		"dict":           forgejoTemplateDict,
		"svg": func(name string, _ ...any) template.HTML {
			return template.HTML(`<svg data-icon="` + template.HTMLEscapeString(name) + `"></svg>`)
		},
	}
	parsed, err := template.New("forgejo-setup").Funcs(functions).Parse(definitions)
	if err != nil {
		t.Fatalf("parse setup override: %v", err)
	}

	base := map[string]any{
		"Title":                  "Install Forgejo",
		"CurDbType":              "sqlite3",
		"DbTypeNames":            []map[string]string{{"type": "sqlite3", "name": "SQLite 3"}, {"type": "postgres", "name": "PostgreSQL"}},
		"PasswordHashAlgorithms": []string{"pbkdf2", "argon2"},
		"CustomConfFile":         "/safe/custom/app.ini",
	}
	render := func(t *testing.T, additions map[string]any) string {
		t.Helper()
		data := make(map[string]any, len(base)+len(additions))
		for key, value := range base {
			data[key] = value
		}
		for key, value := range additions {
			data[key] = value
		}
		var output bytes.Buffer
		if err := parsed.ExecuteTemplate(&output, "page", data); err != nil {
			t.Fatalf("render setup override: %v", err)
		}
		return output.String()
	}

	ordinary := render(t, nil)
	for _, want := range []string{
		"native-head", "native-alert", "native-footer", "soda-intro/install.title", "soda-theme/soda-setup-theme-toggle",
		`action="/forge/" method="post"`, `value="disable"`, "SQLite 3", "PostgreSQL",
		"pbkdf2", "argon2", "/safe/custom/app.ini", `/forge/assets/img/forgejo-loading.svg`,
	} {
		if !strings.Contains(ordinary, want) {
			t.Errorf("ordinary setup render does not contain %q", want)
		}
	}
	if strings.Contains(ordinary, "reinstall_confirm_first") {
		t.Error("ordinary setup render unexpectedly shows reinstall confirmations")
	}
	if strings.Contains(ordinary, "SODA_LOCKED_VALUE") {
		t.Error("ordinary setup render unexpectedly shows environment-owned keys")
	}

	reinstall := render(t, map[string]any{
		"Err_DbInstalledBefore": true,
		"EnvConfigKeys":         []string{"SODA_LOCKED_VALUE", "SODA_MANAGED_PATH"},
	})
	for _, want := range []string{
		"reinstall_confirm_first", "reinstall_confirm_second", "reinstall_confirm_third",
		"SODA_LOCKED_VALUE", "SODA_MANAGED_PATH", "install.env_config_keys_prompt",
	} {
		if !strings.Contains(reinstall, want) {
			t.Errorf("guarded setup render does not contain %q", want)
		}
	}
	if strings.Count(reinstall, "native-head") != 1 || strings.Count(reinstall, "native-footer") != 1 {
		t.Error("setup override must delegate to the native head and footer exactly once")
	}
}

func TestForgejoPostInstallKeepsNativeRedirectFallback(t *testing.T) {
	contents := readForgejoTemplate(t, "post-install.tmpl")
	for _, contract := range []string{
		`class="page-content install post-install soda-page soda-forgejo-setup soda-forgejo-setup-completing" data-signed="false"`,
		`src="{{AssetUrlPrefix}}/img/forgejo-loading.svg"`,
		`id="goto-user-login" href="{{AppSubUrl}}/user/login"`,
		`{{ctx.Locale.Tr "loading"}}`,
	} {
		if !strings.Contains(contents, contract) {
			t.Errorf("post-install.tmpl lost native completion contract %q", contract)
		}
	}
	if calls := templateCalls(contents); !calls["base/head"] || !calls["base/footer"] {
		t.Error("post-install.tmpl must delegate to the native head and footer")
	}
	if !templateCalls(contents)["custom/soda/theme_toggle"] || strings.Count(contents, `{{template "custom/soda/theme_toggle"`) != 1 {
		t.Error("post-install.tmpl must render exactly one shared guest theme toggle")
	}
}

func TestForgejoSetupStylesStayPageScoped(t *testing.T) {
	path := filepath.Join("..", "assets", "branding", "forgejo", "forgejo-setup.css")
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	css := string(contents)
	for _, selector := range []string{
		".soda-forgejo-setup .install-config-container.soda-page-container",
		".soda-forgejo-setup .soda-setup-panel.ui.segment",
		".soda-forgejo-setup .soda-form .optional.field",
		".soda-forgejo-setup-completing #goto-user-login",
	} {
		if !strings.Contains(css, selector) {
			t.Errorf("forgejo-setup.css lost scoped selector %q", selector)
		}
	}
}

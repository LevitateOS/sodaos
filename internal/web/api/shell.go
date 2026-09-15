package api

import (
	"bytes"
	"html/template"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"

	"github.com/levitateos/sodaos/internal/config"
	"github.com/levitateos/sodaos/internal/store"
)

// Must match appliance/forgejo/templates/custom/header.tmpl.
const workspacePresentation = "2026-09-15.workspace-review-1"

const workspaceShellCSP = "default-src 'none'; script-src 'self'; style-src 'self'; font-src 'self'; img-src 'self'; connect-src 'self'; frame-src 'self'; frame-ancestors 'none'; base-uri 'none'; form-action 'self'"

var workspaceShell = template.Must(template.New("workspace").Parse(`<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<meta name="soda-presentation-revision" content="{{.Revision}}">
<title>Spaces</title>
<link rel="stylesheet" href="/assets/soda/fonts/fonts.css">
<link rel="stylesheet" href="/assets/soda/theme/palette.css">
<link rel="stylesheet" href="/assets/soda/forgejo/css/theme-soda-auto.css">
<link rel="stylesheet" href="/assets/soda/forgejo/css/soda-controls.css">
<link rel="stylesheet" href="/assets/soda/forgejo/components.css?v={{.Revision}}">
<link rel="stylesheet" href="/assets/soda/forgejo/components-buttons.css?v={{.Revision}}">
<link rel="stylesheet" href="/assets/sodaspaces.css?v={{.Revision}}">
<link rel="stylesheet" href="/assets/sodaspaces-page.css?v={{.Revision}}">
<link rel="stylesheet" href="/assets/sodaspaces-drawer.css?v={{.Revision}}">
<link rel="stylesheet" href="/assets/sodaspaces-terminal.css?v={{.Revision}}">
<link rel="stylesheet" href="/assets/sodaspaces-shell.css?v={{.Revision}}">
<link rel="stylesheet" href="/assets/soda-terminal/xterm.css">
</head>
<body class="soda-workspace-shell">
<span class="sodaspaces-measure" aria-hidden="true">MMMMMMMMMMMMMMMM</span>
<nav id="sodaspaces-surfaces" hidden aria-label="Workspace surface">
<button type="button" class="ui button" id="soda-surface-forge">Forge</button>
<button type="button" class="ui button" id="soda-surface-terminal">Terminal</button>
</nav>
<iframe id="soda-forgejo-frame" title="Forgejo" src="{{.Frame}}"></iframe>
<div id="soda-workspace-divider" role="separator" tabindex="0" aria-orientation="vertical" aria-label="Workspace width"></div>
<div id="soda-workspace-root" data-actor="{{.Actor}}"></div>
<script type="module" src="/assets/sodaspaces-shell.js?v={{.Revision}}"></script>
</body>
</html>
`))

func workspaceFrame(r *http.Request) (string, bool) {
	if r.URL.ForceQuery && r.URL.RawQuery == "" {
		return "", false
	}
	query := r.URL.Query()
	if len(query) == 0 {
		return "/", true
	}
	values, ok := query["to"]
	if !ok || len(query) != 1 || len(values) != 1 {
		return "", false
	}
	return admitWorkspaceFrame(values[0])
}

func admitWorkspaceFrame(raw string) (string, bool) {
	u, ok := parseWorkspaceFrame(raw)
	if !ok || sodaFramePath(u.Path) || credentialFrameQuery(u.Query()) || outsideWorkspaceFrame(u) {
		return "", false
	}
	frame := u.Path
	if u.RawQuery != "" {
		frame += "?" + u.Query().Encode()
	}
	return frame, true
}

func parseWorkspaceFrame(raw string) (*url.URL, bool) {
	if blockedFrameRaw(raw) {
		return nil, false
	}
	u, err := url.Parse(raw)
	if err != nil || !relativeFrameURL(u) {
		return nil, false
	}
	if !strings.HasPrefix(u.Path, "/") || u.Path != path.Clean(u.Path) {
		return nil, false
	}
	return u, true
}

func blockedFrameRaw(raw string) bool {
	return raw == "" || len(raw) > 2048 || strings.Contains(raw, "..") || strings.Contains(raw, "//") || strings.Contains(raw, `\`)
}

func relativeFrameURL(u *url.URL) bool {
	if u.IsAbs() || u.Scheme != "" || u.Host != "" {
		return false
	}
	return u.Opaque == "" && u.User == nil && u.Fragment == ""
}

func sodaFramePath(p string) bool {
	return p == config.SodaPath || strings.HasPrefix(p, config.SodaPath+"/")
}

func credentialFrameQuery(q url.Values) bool {
	return q.Has("code") || q.Has("state") || q.Has("access_token") || q.Has("id_token")
}

func outsideWorkspaceFrame(u *url.URL) bool {
	if u.Query().Has("soda-connect") || sodaViewHost(u.Query()) {
		return true
	}
	return loginConsentInstall(u.Path)
}

// Any soda-view or soda-connect host is a native document, never framed
// content: valid settings hosts, failure displays, and unknown or duplicate
// selectors all stay top-level so the dashboard can answer them.
func sodaViewHost(q url.Values) bool {
	return q.Has("soda-view")
}

// Auth and account-recovery flows stay top-level. Keep in lockstep with
// authPaths in frontend/spaces/sodaspaces-frame.ts.
func loginConsentInstall(p string) bool {
	for _, prefix := range []string{
		"/user/login",
		"/user/logout",
		"/user/sign_up",
		"/user/activate",
		"/user/forgot_password",
		"/user/reset_password",
		"/user/two_factor",
		"/user/u2f",
		"/user/webauthn",
		"/user/oauth2",
		"/user/link_account",
		"/login/oauth",
		"/install",
	} {
		if p == prefix || strings.HasPrefix(p, prefix+"/") {
			return true
		}
	}
	return false
}

func (s *API) writeWorkspaceShell(w http.ResponseWriter, session store.Session, frame string) {
	var body bytes.Buffer
	if workspaceShell.Execute(&body, struct{ Revision, Actor, Frame string }{workspacePresentation, strconv.FormatInt(session.User.ID, 10), frame}) != nil {
		http.Error(w, "Workspace unavailable.", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "private, no-store")
	w.Header().Set("Referrer-Policy", "no-referrer")
	w.Header().Set("Content-Security-Policy", workspaceShellCSP)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body.Bytes())
}

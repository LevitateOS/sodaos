package api

import (
	"bytes"
	"html/template"
	"net/http"
	"strconv"

	"github.com/levitateos/sodaos/internal/store"
)

// Must match appliance/forgejo/templates/custom/header.tmpl.
const workspacePresentation = "2026-09-13.spaces-first-use-1"

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
<iframe id="soda-forgejo-frame" title="Forgejo" src="/"></iframe>
<div id="soda-workspace-root" data-actor="{{.Actor}}"></div>
<script type="module" src="/assets/sodaspaces-shell.js?v={{.Revision}}"></script>
</body>
</html>
`))

func (s *API) writeWorkspaceShell(w http.ResponseWriter, session store.Session) {
	var body bytes.Buffer
	if workspaceShell.Execute(&body, struct{ Revision, Actor string }{workspacePresentation, strconv.FormatInt(session.User.ID, 10)}) != nil {
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

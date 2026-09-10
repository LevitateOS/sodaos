package web

import (
	"bytes"
	"html/template"
	"net/http"
)

// Render completely before committing an HTML response. On template failure the
// caller still owns its error response; page authorization, CSP and data stay there.
func writePageTemplate(w http.ResponseWriter, tmpl *template.Template, data any, status int) error {
	var body bytes.Buffer
	if err := tmpl.Execute(&body, data); err != nil {
		return err
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_, _ = w.Write(body.Bytes())
	return nil
}

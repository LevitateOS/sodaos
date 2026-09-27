package muse

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNativeEnrollmentKeepsPresentationPrivate(t *testing.T) {
	root := t.TempDir()
	binary := filepath.Join(root, "muse-fixture")
	script := `#!/bin/sh
set -eu
[ "$1" = login ]
[ "$TBH_CREDENTIAL_BACKEND" = file ]
[ -z "${META_API_KEY:-}" ]
printf 'https://auth.meta.com/oauth/device/?code=ABCD-1234'
mkdir -p "$XDG_CONFIG_HOME/muse"
umask 077
cat > "$XDG_CONFIG_HOME/muse/auth.json" <<'JSON'
` + subscription + `
JSON
printf 'synthetic secret diagnostic' >&2
`
	if err := os.WriteFile(binary, []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	p := &Provider{config: Config{Binary: binary, Root: root}}
	session, err := p.Start(t.Context(), "subscription")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = session.Close() }()
	s := session.(*Session)
	select {
	case <-s.done:
	case <-time.After(5 * time.Second):
		t.Fatal("enrollment fixture did not stop")
	}
	snapshot := session.Snapshot()
	if snapshot.State != "completed" || snapshot.UserCode != "ABCD-1234" {
		t.Fatal("non-newline prompt was not consumed", snapshot.State)
	}
	conn, data, err := session.Finish(t.Context())
	if err != nil || !CredentialValid(data) {
		t.Fatal("native file not retained", err)
	}
	presentation, err := json.Marshal(conn)
	if err != nil || strings.Contains(string(presentation), "synthetic") {
		t.Fatal("account presentation or credentials escaped", err)
	}
	public, err := json.Marshal(snapshot)
	if err != nil || strings.Contains(string(public), "synthetic") {
		t.Fatal("native diagnostics escaped", err)
	}
}

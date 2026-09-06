package forgejo

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTailnetPreservesBrowserOrigin(t *testing.T) {
	p := filepath.Join(t.TempDir(), "forgejo.env")
	original := "FORGEJO__server__ROOT_URL=https://forgejo.example/\n"
	os.WriteFile(p, []byte(original), 0600)
	changed, err := UpdateSSHDomain(p, "100.100.0.1")
	if err != nil || !changed {
		t.Fatal(changed, err)
	}
	b, _ := os.ReadFile(p)
	if !strings.Contains(string(b), original) {
		t.Fatal("browser origin changed")
	}
	changed, err = UpdateSSHDomain(p, "100.100.0.1")
	if err != nil || changed {
		t.Fatal("unchanged native endpoint should not restart service")
	}
}

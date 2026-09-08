package web

import (
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"testing"

	"github.com/levitateos/sodaos/internal/avatar"
	"github.com/levitateos/sodaos/internal/config"
)

// This opt-in check uses the existing local Playwright/Chrome installation. It
// compares the actual HTTP response against the same SVG without response CSP,
// catching browser restrictions on DiceBear's internal <use> references.
func TestAvatarBrowserRendering(t *testing.T) {
	if os.Getenv("SODA_AVATAR_BROWSER_CHECK") != "1" {
		t.Skip("set SODA_AVATAR_BROWSER_CHECK=1 with local Playwright and Chrome")
	}
	mux := http.NewServeMux()
	mux.Handle(avatarPrefix, New(config.Config{}, nil))
	mux.HandleFunc("GET /reference/{hash}", func(w http.ResponseWriter, r *http.Request) {
		svg, err := avatar.Render(r.PathValue("hash"), 128)
		if err != nil {
			http.Error(w, "invalid test seed", 400)
			return
		}
		w.Header().Set("Content-Type", "image/svg+xml")
		w.Write([]byte(svg))
	})
	server := httptest.NewServer(mux)
	defer server.Close()
	cmd := exec.Command("node", "testdata/avatar-browser.cjs", server.URL)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("browser rendering: %v\n%s", err, out)
	} else {
		t.Log(string(out))
	}
}

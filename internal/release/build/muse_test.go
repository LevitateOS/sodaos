package build

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type museTransport func(*http.Request) (*http.Response, error)

func (f museTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestFetchMuseChecksBytesBeforePublishing(t *testing.T) {
	previous := http.DefaultClient
	defer func() { http.DefaultClient = previous }()
	payload := "verified native bytes"
	sum := sha256.Sum256([]byte(payload))
	root := t.TempDir()
	manifest, dest := filepath.Join(root, "release.json"), filepath.Join(root, "bin/muse")
	text := fmt.Sprintf(`{"version":"1.4.0-R4161.1","artifacts":{"x86_64":{"file":"muse-x86-linux","sha256":"%s","size":%d}}}`, hex.EncodeToString(sum[:]), len(payload))
	if err := os.WriteFile(manifest, []byte(text), 0o600); err != nil {
		t.Fatal(err)
	}
	received := "tampered native bytes"
	http.DefaultClient = &http.Client{Transport: museTransport(func(r *http.Request) (*http.Response, error) {
		if r.URL.Host != "lookaside.facebook.com" || r.URL.Query().Get("version") != "1.4.0-R4161.1" {
			t.Fatal("unpinned upstream request")
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(received))}, nil
	})}
	if err := FetchMuse(context.Background(), manifest, "x86_64", dest); err == nil {
		t.Fatal("accepted modified artifact")
	}
	if _, err := os.Stat(dest); !os.IsNotExist(err) {
		t.Fatal("published unverified bytes")
	}
	received = payload
	if err := FetchMuse(context.Background(), manifest, "x86_64", dest); err != nil {
		t.Fatal(err)
	}
	st, err := os.Stat(dest)
	if err != nil || st.Mode().Perm() != 0o755 {
		t.Fatal("missing executable")
	}
}

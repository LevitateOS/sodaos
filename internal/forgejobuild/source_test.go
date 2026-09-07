package forgejobuild

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type archiveEntry struct {
	name, body, target string
	kind               byte
	mode               int64
}

func archiveFixture(t *testing.T, extra ...archiveEntry) []byte {
	t.Helper()
	entries := []archiveEntry{}
	for _, name := range []string{"LICENSE", "go.mod", "go.sum", "package.json", "package-lock.json", "Makefile", "Dockerfile", "docker/root/usr/bin/entrypoint"} {
		entries = append(entries, archiveEntry{name: "forgejo/" + name, body: "base\n", kind: tar.TypeReg, mode: 0644})
	}
	entries = append(entries, extra...)
	var b bytes.Buffer
	gz := gzip.NewWriter(&b)
	tw := tar.NewWriter(gz)
	if err := tw.WriteHeader(&tar.Header{Name: "pax_global_header", Typeflag: tar.TypeXGlobalHeader, PAXRecords: map[string]string{"comment": strings.Repeat("a", 40)}}); err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		h := &tar.Header{Name: e.name, Typeflag: e.kind, Linkname: e.target, Mode: e.mode}
		if e.kind == tar.TypeReg {
			h.Size = int64(len(e.body))
		}
		if err := tw.WriteHeader(h); err != nil {
			t.Fatal(err)
		}
		if e.kind == tar.TypeReg {
			if _, err := io.WriteString(tw, e.body); err != nil {
				t.Fatal(err)
			}
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

func writeInputs(t *testing.T, archive []byte, patches ...string) (string, string, string) {
	t.Helper()
	dir := t.TempDir()
	lockPath, archivePath, out := filepath.Join(dir, "source.lock.json"), filepath.Join(dir, "archive.tar.gz"), filepath.Join(dir, "prepared")
	lock := Lock{Schema: 1, Version: "16.0.3", Commit: strings.Repeat("a", 40), ArchiveSHA256: hash(archive), Patches: []Patch{}}
	if err := os.Mkdir(filepath.Join(dir, "patches"), 0700); err != nil {
		t.Fatal(err)
	}
	for i, p := range patches {
		name := []string{"0001-first.patch", "0002-second.patch"}[i]
		if err := os.WriteFile(filepath.Join(dir, "patches", name), []byte(p), 0600); err != nil {
			t.Fatal(err)
		}
		lock.Patches = append(lock.Patches, Patch{File: name, SHA256: hash([]byte(p))})
	}
	b, err := json.Marshal(lock)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(lockPath, b, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(archivePath, archive, 0600); err != nil {
		t.Fatal(err)
	}
	return lockPath, archivePath, out
}

func TestPrepareRealTypesAndReceipt(t *testing.T) {
	archive := archiveFixture(t,
		archiveEntry{name: "forgejo/scripts/build", body: "#!/bin/sh\n", kind: tar.TypeReg, mode: 0755},
		archiveEntry{name: "forgejo/docs/license", target: "../LICENSE", kind: tar.TypeSymlink, mode: 0777},
	)
	lock, file, out := writeInputs(t, archive)
	if err := Prepare(context.Background(), lock, file, out); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(out, "source/Makefile"))
	if err != nil || string(got) != "base\n" {
		t.Fatalf("source: %q %v", got, err)
	}
	if target, err := os.Readlink(filepath.Join(out, "source/docs/license")); err != nil || target != "../LICENSE" {
		t.Fatalf("link: %q %v", target, err)
	}
	st, err := os.Stat(filepath.Join(out, "source/scripts/build"))
	if err != nil || st.Mode().Perm()&0111 == 0 {
		t.Fatalf("executable: %v", err)
	}
	b, err := os.ReadFile(filepath.Join(out, "prepared.json"))
	if err != nil {
		t.Fatal(err)
	}
	var receipt struct {
		LockSHA256   string `json:"lock_sha256"`
		SourceSHA256 string `json:"source_sha256"`
	}
	if err := json.Unmarshal(b, &receipt); err != nil {
		t.Fatal(err)
	}
	lockBytes, _ := os.ReadFile(lock)
	if receipt.LockSHA256 != hash(lockBytes) || !digestPattern.MatchString(receipt.SourceSHA256) {
		t.Fatalf("invalid receipt: %s", b)
	}
	if err := Prepare(context.Background(), lock, file, out); err == nil {
		t.Fatal("occupied output accepted")
	}
	after, _ := os.ReadFile(filepath.Join(out, "prepared.json"))
	if !bytes.Equal(b, after) {
		t.Fatal("existing receipt changed")
	}
}

func TestPrepareRejectsArchiveAttacks(t *testing.T) {
	for name, entries := range map[string][]archiveEntry{
		"escape":           {{name: "forgejo/../outside", body: "evil", kind: tar.TypeReg}},
		"absolute":         {{name: "/outside", body: "evil", kind: tar.TypeReg}},
		"wrong-root":       {{name: "other/file", body: "evil", kind: tar.TypeReg}},
		"duplicate":        {{name: "forgejo/Makefile", body: "evil", kind: tar.TypeReg}},
		"setuid":           {{name: "forgejo/setuid", body: "evil", kind: tar.TypeReg, mode: 04755}},
		"hardlink":         {{name: "forgejo/hard", target: "forgejo/LICENSE", kind: tar.TypeLink}},
		"device":           {{name: "forgejo/device", kind: tar.TypeChar}},
		"fifo":             {{name: "forgejo/fifo", kind: tar.TypeFifo}},
		"git-dir":          {{name: "forgejo/.git/config", body: "evil", kind: tar.TypeReg}},
		"absolute-link":    {{name: "forgejo/link", target: "/etc", kind: tar.TypeSymlink}},
		"escape-link":      {{name: "forgejo/link", target: "../outside", kind: tar.TypeSymlink}},
		"dangling-link":    {{name: "forgejo/link", target: "missing", kind: tar.TypeSymlink}},
		"cycle":            {{name: "forgejo/link", target: "link", kind: tar.TypeSymlink}},
		"child-of-link":    {{name: "forgejo/link", target: "docker", kind: tar.TypeSymlink}, {name: "forgejo/link/file", body: "evil", kind: tar.TypeReg}},
		"link-after-child": {{name: "forgejo/link/file", body: "evil", kind: tar.TypeReg}, {name: "forgejo/link", target: "docker", kind: tar.TypeSymlink}},
	} {
		t.Run(name, func(t *testing.T) {
			lock, file, out := writeInputs(t, archiveFixture(t, entries...))
			if err := Prepare(context.Background(), lock, file, out); err == nil {
				t.Fatal("unsafe archive accepted")
			}
			if _, err := os.Stat(filepath.Join(out, "prepared.json")); !os.IsNotExist(err) {
				t.Fatal("failure has completion receipt")
			}
			if _, err := os.Stat(filepath.Join(out, "outside")); !os.IsNotExist(err) {
				t.Fatal("archive escaped source")
			}
		})
	}
}

func TestDigestAndGzipIntegrityFailuresRetainAttempt(t *testing.T) {
	for _, name := range []string{"digest", "gzip-footer", "missing-input"} {
		t.Run(name, func(t *testing.T) {
			archive := archiveFixture(t)
			if name == "gzip-footer" {
				archive[len(archive)-8] ^= 0xff
			}
			lock, file, out := writeInputs(t, archive)
			if name == "digest" {
				if err := os.WriteFile(file, []byte("wrong bytes"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			if name == "missing-input" {
				file += ".absent"
			}
			if err := Prepare(context.Background(), lock, file, out); err == nil {
				t.Fatal("bad source accepted")
			}
			if _, err := os.Stat(out); err != nil {
				t.Fatal("failed attempt not retained")
			}
			if _, err := os.Stat(filepath.Join(out, "prepared.json")); !os.IsNotExist(err) {
				t.Fatal("failure has receipt")
			}
		})
	}
}

func TestOutputAndInputSymlinksRefused(t *testing.T) {
	lock, file, out := writeInputs(t, archiveFixture(t))
	alias := out + "-parent"
	if err := os.Symlink(filepath.Dir(out), alias); err != nil {
		t.Fatal(err)
	}
	if err := Prepare(context.Background(), lock, file, filepath.Join(alias, "new")); err == nil {
		t.Fatal("symlink parent accepted")
	}
	if err := os.Symlink(file, file+".link"); err != nil {
		t.Fatal(err)
	}
	if err := Prepare(context.Background(), lock, file+".link", out); err == nil {
		t.Fatal("symlink archive accepted")
	}
	if _, err := os.Stat(filepath.Join(out, "prepared.json")); !os.IsNotExist(err) {
		t.Fatal("failure has receipt")
	}
}

func TestInvalidLocksAndPatchDigests(t *testing.T) {
	for _, name := range []string{"unknown", "schema", "commit", "digest", "version", "null-patches", "unordered", "patch-path", "patch-digest", "trailing"} {
		t.Run(name, func(t *testing.T) {
			lock, file, out := writeInputs(t, archiveFixture(t), "patch bytes")
			b, _ := os.ReadFile(lock)
			var data map[string]any
			if err := json.Unmarshal(b, &data); err != nil {
				t.Fatal(err)
			}
			switch name {
			case "unknown":
				data["unexpected"] = true
			case "schema":
				data["schema"] = 2
			case "commit":
				data["commit"] = "../forgejo"
			case "digest":
				data["archive_sha256"] = "bad"
			case "version":
				data["version"] = "latest"
			case "null-patches":
				data["patches"] = nil
			case "unordered":
				data["patches"] = []Patch{{File: "0002-first.patch", SHA256: hash([]byte("patch bytes"))}}
			case "patch-path":
				data["patches"] = []Patch{{File: "../first.patch", SHA256: hash([]byte("patch bytes"))}}
			case "patch-digest":
				data["patches"] = []Patch{{File: "0001-first.patch", SHA256: strings.Repeat("0", 64)}}
			}
			b, _ = json.Marshal(data)
			if name == "trailing" {
				b = append(b, []byte(" {}")...)
			}
			if err := os.WriteFile(lock, b, 0600); err != nil {
				t.Fatal(err)
			}
			if err := Prepare(context.Background(), lock, file, out); err == nil {
				t.Fatal("invalid lock accepted")
			}
			if _, err := os.Stat(out); !os.IsNotExist(err) {
				t.Fatal("invalid lock wrote output")
			}
		})
	}
}

const firstPatch = "diff --git a/Makefile b/Makefile\n--- a/Makefile\n+++ b/Makefile\n@@ -1 +1 @@\n-base\n+first\n"
const secondPatch = "diff --git a/Makefile b/Makefile\n--- a/Makefile\n+++ b/Makefile\n@@ -1 +1 @@\n-first\n+second\n"

func TestOrderedPatchesAndConflictAreNativeGitOperations(t *testing.T) {
	for _, conflict := range []bool{false, true} {
		second := secondPatch
		if conflict {
			second = firstPatch
		}
		lock, file, out := writeInputs(t, archiveFixture(t), firstPatch, second)
		// Inherited Git repository/config state must not affect the source patcher.
		t.Setenv("GIT_DIR", filepath.Join(t.TempDir(), "not-a-repo"))
		err := Prepare(context.Background(), lock, file, out)
		if conflict {
			if err == nil {
				t.Fatal("conflicting patch accepted")
			}
			if _, err := os.Stat(filepath.Join(out, "prepared.json")); !os.IsNotExist(err) {
				t.Fatal("partial patch series has receipt")
			}
			b, _ := os.ReadFile(filepath.Join(out, "source/Makefile"))
			if string(b) != "first\n" {
				t.Fatal("partial attempt lost")
			}
		} else {
			if err != nil {
				t.Fatal(err)
			}
			b, _ := os.ReadFile(filepath.Join(out, "source/Makefile"))
			if string(b) != "second\n" {
				t.Fatalf("patches not applied in order: %q", b)
			}
		}
	}
}

func TestPatchCannotEscapeSource(t *testing.T) {
	patch := "diff --git a/../outside b/../outside\nnew file mode 100644\n--- /dev/null\n+++ b/../outside\n@@ -0,0 +1 @@\n+evil\n"
	lock, file, out := writeInputs(t, archiveFixture(t), patch)
	if err := Prepare(context.Background(), lock, file, out); err == nil {
		t.Fatal("escaping patch accepted")
	}
	if _, err := os.Stat(filepath.Join(out, "outside")); !os.IsNotExist(err) {
		t.Fatal("patch escaped")
	}
}

type roundTrip func(*http.Request) (*http.Response, error)

func (f roundTrip) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestDownloadUsesOnlyLockedPublicCommitAndNoCredentials(t *testing.T) {
	archive := archiveFixture(t)
	for _, status := range []int{200, 302, 404, 503} {
		lock, _, out := writeInputs(t, archive)
		calls := 0
		client := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }, Transport: roundTrip(func(r *http.Request) (*http.Response, error) {
			calls++
			if r.Method != "GET" || r.URL.String() != "https://codeberg.org/forgejo/forgejo/archive/"+strings.Repeat("a", 40)+".tar.gz" {
				t.Fatalf("unexpected request: %s", r.URL)
			}
			if r.Header.Get("Authorization") != "" || r.Header.Get("Cookie") != "" {
				t.Fatal("credentials in source request")
			}
			return &http.Response{StatusCode: status, Header: http.Header{"Location": []string{"https://other.invalid/private"}}, Body: io.NopCloser(bytes.NewReader(archive))}, nil
		})}
		err := prepare(context.Background(), lock, "", out, client)
		if (err == nil) != (status == 200) || calls != 1 {
			t.Fatalf("status %d: calls=%d err=%v", status, calls, err)
		}
	}
}

func TestCancelledPreparationHasNoReceipt(t *testing.T) {
	lock, file, out := writeInputs(t, archiveFixture(t))
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := Prepare(ctx, lock, file, out); !errors.Is(err, context.Canceled) {
		t.Fatalf("wanted cancellation, got %v", err)
	}
	if _, err := os.Stat(filepath.Join(out, "prepared.json")); !os.IsNotExist(err) {
		t.Fatal("cancelled preparation has receipt")
	}
}

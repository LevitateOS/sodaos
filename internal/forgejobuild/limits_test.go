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

func TestRelativeSymlinkChainIsRetained(t *testing.T) {
	lock, file, out := writeInputs(t, archiveFixture(t,
		archiveEntry{name: "forgejo/docs/first", target: "second", kind: tar.TypeSymlink, mode: 0777},
		archiveEntry{name: "forgejo/docs/second", target: "../LICENSE", kind: tar.TypeSymlink, mode: 0777},
	))
	if err := Prepare(context.Background(), lock, file, out); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(out, "source/docs/first"))
	if err != nil || string(b) != "base\n" {
		t.Fatalf("symlink chain: %q %v", b, err)
	}
}

func TestArchiveCommitMustMatchLock(t *testing.T) {
	lock, file, out := writeInputs(t, archiveFixture(t))
	b, err := os.ReadFile(lock)
	if err != nil {
		t.Fatal(err)
	}
	b = bytes.ReplaceAll(b, []byte(strings.Repeat("a", 40)), []byte(strings.Repeat("b", 40)))
	if err := os.WriteFile(lock, b, 0600); err != nil {
		t.Fatal(err)
	}
	if err := Prepare(context.Background(), lock, file, out); err == nil || !strings.Contains(err.Error(), "commit metadata") {
		t.Fatalf("wrong commit accepted: %v", err)
	}
	if _, err := os.Stat(filepath.Join(out, "prepared.json")); !os.IsNotExist(err) {
		t.Fatal("bad commit has receipt")
	}
}

type zeros struct{}

func (zeros) Read(b []byte) (int, error) { clear(b); return len(b), nil }

func TestExpansionLimitIncludesTarPadding(t *testing.T) {
	base := archiveFixture(t)
	reader, err := gzip.NewReader(bytes.NewReader(base))
	if err != nil {
		t.Fatal(err)
	}
	var compressed bytes.Buffer
	writer := gzip.NewWriter(&compressed)
	if _, err := io.Copy(writer, reader); err != nil {
		t.Fatal(err)
	}
	if err := reader.Close(); err != nil {
		t.Fatal(err)
	}
	// Highly compressed trailing padding must not bypass the expanded-byte cap.
	if _, err := io.CopyN(writer, zeros{}, maxSource+1); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	lock, file, out := writeInputs(t, compressed.Bytes())
	if err := Prepare(context.Background(), lock, file, out); err == nil || !strings.Contains(err.Error(), "expansion limit") {
		t.Fatalf("padding limit: %v", err)
	}
	if _, err := os.Stat(filepath.Join(out, "prepared.json")); !os.IsNotExist(err) {
		t.Fatal("oversize has receipt")
	}
}

func TestRequiredBuildInputCannotBeDeletedByPatch(t *testing.T) {
	patch := "diff --git a/LICENSE b/LICENSE\ndeleted file mode 100644\n--- a/LICENSE\n+++ /dev/null\n@@ -1 +0,0 @@\n-base\n"
	lock, file, out := writeInputs(t, archiveFixture(t), patch)
	if err := Prepare(context.Background(), lock, file, out); err == nil || !strings.Contains(err.Error(), "missing regular Forgejo build input") {
		t.Fatalf("missing license: %v", err)
	}
	if _, err := os.Stat(filepath.Join(out, "prepared.json")); !os.IsNotExist(err) {
		t.Fatal("missing license has receipt")
	}
}

func TestSourceTreeReceiptIsDeterministicAndContentBound(t *testing.T) {
	var hashes []string
	for _, patches := range [][]string{nil, nil, {firstPatch}} {
		lock, file, out := writeInputs(t, archiveFixture(t), patches...)
		if err := Prepare(context.Background(), lock, file, out); err != nil {
			t.Fatal(err)
		}
		b, err := os.ReadFile(filepath.Join(out, "prepared.json"))
		if err != nil {
			t.Fatal(err)
		}
		var receipt struct {
			SourceSHA256 string `json:"source_sha256"`
		}
		if err := json.Unmarshal(b, &receipt); err != nil {
			t.Fatal(err)
		}
		hashes = append(hashes, receipt.SourceSHA256)
	}
	if hashes[0] != hashes[1] || hashes[0] == hashes[2] {
		t.Fatalf("tree binding: %v", hashes)
	}
}

func TestDownloadFailureIsNotRetriedOrEchoed(t *testing.T) {
	lock, _, out := writeInputs(t, archiveFixture(t))
	calls := 0
	client := &http.Client{Transport: roundTrip(func(*http.Request) (*http.Response, error) {
		calls++
		return nil, errors.New("secret-bearing untrusted transport diagnostic")
	})}
	err := prepare(context.Background(), lock, "", out, client)
	if err == nil || err.Error() != "Forgejo source download failed" || calls != 1 {
		t.Fatalf("calls=%d error=%v", calls, err)
	}
	if _, err := os.Stat(filepath.Join(out, "prepared.json")); !os.IsNotExist(err) {
		t.Fatal("failed download has receipt")
	}
}

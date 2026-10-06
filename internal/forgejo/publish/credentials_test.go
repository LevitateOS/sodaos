package publish

import (
	"bufio"
	"bytes"
	"crypto/sha1"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func candidateGit(t *testing.T, c Config, args ...string) string {
	t.Helper()
	command := exec.Command("git", args...)
	command.Dir = filepath.Join(c.Root, "source")
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("candidate fixture Git: %v", err)
	}
	return strings.TrimSpace(string(output))
}

func candidateBundle(t *testing.T, c Config, r *Request) {
	t.Helper()
	r.Commit = candidateGit(t, c, "rev-parse", "HEAD")
	path := filepath.Join(c.Root, "candidate.bundle")
	candidateGit(t, c, "bundle", "create", path, "HEAD")
	var err error
	r.Bundle, err = os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
}

func TestPublisherRejectsCredentialHistoryBeforeAuthorization(t *testing.T) {
	for _, location := range []string{"deleted-blob", "commit-message", "tree-name"} {
		t.Run(location, func(t *testing.T) {
			c, r := candidateFixture(t, "README.md")
			secret := "synthetic-renewed-credential"
			name, data, message := "auth.json", secret, "credential-bearing change"
			if location == "commit-message" {
				data, message = "ordinary content", secret
			}
			if location == "tree-name" {
				name, data = secret, "ordinary content"
			}
			path := filepath.Join(c.Root, "source", name)
			if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
				t.Fatal(err)
			}
			candidateGit(t, c, "add", ".")
			candidateGit(t, c, "commit", "-m", message)
			candidateGit(t, c, "rm", name)
			candidateGit(t, c, "commit", "-m", "remove private input")
			candidateBundle(t, c, &r)
			r.ProtectedCredentials = []string{"synthetic-original-credential", secret}
			err := c.ValidateCandidate(t.Context(), r)
			if err == nil || !strings.Contains(err.Error(), "protected credential") {
				t.Fatalf("expected credential rejection, got %v", err)
			}
		})
	}
}

func TestPublisherAcceptsCleanCandidateWithCredentialDenylist(t *testing.T) {
	c, r := candidateFixture(t, "README.md")
	r.ProtectedCredentials = []string{"synthetic-original-credential", "synthetic-renewed-credential"}
	if err := c.ValidateCandidate(t.Context(), r); err != nil {
		t.Fatalf("clean candidate rejected: %v", err)
	}
}

func TestPublisherRejectsCompressedOversizedDecodedObject(t *testing.T) {
	c, r := candidateFixture(t, "README.md")
	if err := os.WriteFile(filepath.Join(c.Root, "source", "large.txt"), bytes.Repeat([]byte("x"), credentialObjectBytes+1), 0o600); err != nil {
		t.Fatal(err)
	}
	candidateGit(t, c, "add", ".")
	candidateGit(t, c, "commit", "-m", "compressible oversized object")
	candidateBundle(t, c, &r)
	err := c.ValidateCandidate(t.Context(), r)
	if err == nil || !strings.Contains(err.Error(), "decoded credential scan limit") {
		t.Fatalf("expected decoded limit rejection, got %v", err)
	}
}

func TestCredentialScanRejectsMismatchedContentIdentity(t *testing.T) {
	content := []byte("ordinary content")
	identity := sha1.Sum(append([]byte(fmt.Sprintf("blob %d\x00", len(content))), content...))
	object := fmt.Sprintf("%x", identity)
	batch := fmt.Sprintf("%s blob %d\n%s\n", object, len(content), bytes.Repeat([]byte("x"), len(content)))
	if err := scanCredentials(bufio.NewReader(strings.NewReader(batch)), []string{object}, nil); err == nil {
		t.Fatal("decoded content with different object identity accepted")
	}
}

func TestCredentialScanRejectsDecodedAggregateLimit(t *testing.T) {
	content := bytes.Repeat([]byte("x"), credentialObjectBytes)
	identity := sha1.Sum(append([]byte(fmt.Sprintf("blob %d\x00", len(content))), content...))
	object := fmt.Sprintf("%x", identity)
	header := fmt.Sprintf("%s blob %d\n", object, len(content))
	var readers []io.Reader
	var objects []string
	for range credentialTotalBytes/credentialObjectBytes + 1 {
		readers = append(readers, strings.NewReader(header), bytes.NewReader(content), strings.NewReader("\n"))
		objects = append(objects, object)
	}
	err := scanCredentials(bufio.NewReader(io.MultiReader(readers...)), objects, nil)
	if err == nil || !strings.Contains(err.Error(), "decoded credential scan limit") {
		t.Fatalf("expected total decoded limit rejection, got %v", err)
	}
}

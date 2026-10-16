package forgejo

import (
	"strings"
	"testing"

	"github.com/levitateos/sodaos/internal/factory"
)

func readbackTestThread(resolved bool, digest string) ThreadEvidence {
	resolver := ""
	if resolved {
		resolver = "6"
	}
	sha := strings.Repeat("2", 40)
	return ThreadEvidence{
		RootID: "11", IssueID: "11", ReviewID: "31",
		Path: "a.go", Side: "new", Line: 4,
		OriginalCommit: sha, HeadCommit: sha,
		ResolverID: resolver, Resolved: resolved,
		Digest: digest,
		Members: []ThreadMemberEvidence{
			{ID: "11", IssueID: "11", ContentDigest: ContentDigest("needs a guard"), ContentVersion: 1, CreatedUnix: 1700000000, UpdatedUnix: 1700000000, Visible: true, Complete: true},
			{ID: "12", IssueID: "11", ContentDigest: ContentDigest("guard added"), ContentVersion: 2, CreatedUnix: 1700000100, UpdatedUnix: 1700000100, Visible: true, Complete: true},
		},
		MemberTotal: 2, Visible: true, Complete: true,
	}
}

func readbackTestRequest(digest string) factory.ResolveRequest {
	return factory.ResolveRequest{
		HeadRef: "refs/heads/fix", BaseRef: "refs/heads/main",
		HeadOID: strings.Repeat("2", 40), BaseOID: strings.Repeat("1", 40),
		Digest: digest, Repository: 7, PRNumber: 3, PRID: 5,
		RootCommentID: 11, ActorID: 6, Resolved: true,
	}
}

func TestReadbackResolutionConfirmsExactResolve(t *testing.T) {
	digest := strings.Repeat("a", 64)
	readback, err := ReadbackResolution(readbackTestRequest(digest), readbackTestThread(true, digest))
	if err != nil {
		t.Fatalf("exact resolved readback refused: %v", err)
	}
	if !readback.Resolved || readback.RootCommentID != 11 || readback.Digest != digest {
		t.Fatalf("readback lost the resolution: %+v", readback)
	}
}

func TestReadbackResolutionReportsUnresolvedWithoutError(t *testing.T) {
	digest := strings.Repeat("a", 64)
	readback, err := ReadbackResolution(readbackTestRequest(digest), readbackTestThread(false, digest))
	if err != nil {
		t.Fatalf("matching unresolved readback refused: %v", err)
	}
	if readback.Resolved {
		t.Fatal("unresolved thread read back resolved")
	}
}

func TestReadbackResolutionRefuses(t *testing.T) {
	digest := strings.Repeat("a", 64)
	changed := readbackTestThread(true, strings.Repeat("b", 64))
	if _, err := ReadbackResolution(readbackTestRequest(digest), changed); err == nil {
		t.Fatal("changed conversation read back")
	}
	moved := readbackTestThread(true, digest)
	moved.RootID = "12"
	if _, err := ReadbackResolution(readbackTestRequest(digest), moved); err == nil {
		t.Fatal("wrong-root readback accepted")
	}
	hidden := readbackTestThread(true, digest)
	hidden.Visible = false
	hidden.Digest, hidden.Path, hidden.Members = "", "", nil
	if _, err := ReadbackResolution(readbackTestRequest(digest), hidden); err == nil {
		t.Fatal("hidden thread readback confirmed")
	}
	partial := readbackTestThread(true, digest)
	partial.Complete = false
	if _, err := ReadbackResolution(readbackTestRequest(digest), partial); err == nil {
		t.Fatal("incomplete thread readback confirmed")
	}
	truncated := readbackTestThread(true, digest)
	truncated.Members = truncated.Members[:1]
	if _, err := ReadbackResolution(readbackTestRequest(digest), truncated); err == nil {
		t.Fatal("truncated member list read back")
	}
	unresolving := readbackTestRequest(digest)
	unresolving.Resolved = false
	if _, err := ReadbackResolution(unresolving, readbackTestThread(true, digest)); err == nil {
		t.Fatal("non-resolving request read back")
	}
}

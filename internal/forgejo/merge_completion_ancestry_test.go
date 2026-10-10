package forgejo

import (
	"context"
	"os"
	"strconv"
	"strings"
	"testing"

	extensions "forgejo.org/extension-sdk"

	"github.com/levitateos/sodaos/internal/factory"
)

func TestMergerObserveCompletionUsesExactAncestrySnapshot(t *testing.T) {
	w := mergeTestWork()
	laterTip := strings.Repeat("c", 40)
	otherTip := strings.Repeat("d", 40)
	newMerger := func(t *testing.T, baseTip string, revision func() int64, mutateProof func(*extensions.NativeSnapshot, extensions.SnapshotRequest)) (*Merger, *[]extensions.SnapshotRequest) {
		t.Helper()
		var requests []extensions.SnapshotRequest
		fake := &fakeBackgroundServer{
			admission: strings.Repeat("a", 43),
			revision:  revision,
			snapshot: func(req extensions.SnapshotRequest) extensions.NativeSnapshot {
				requests = append(requests, req)
				answer := completionWireSnapshot(w, req, baseTip)
				if len(req.Families) == 2 && containsSnapshotFamily(req.Families, FamilyAncestry) && mutateProof != nil {
					mutateProof(&answer, req)
				}
				return answer
			},
		}
		background := NewServiceBackground(serveBackgroundSocket(t, fake), uint32(os.Getuid()), "")
		merger := NewMerger(background, observationREST(t, w.ActorID), observationCredential(t, "test-pat"))
		return merger, &requests
	}

	t.Run("later descendant accepted with attribution", func(t *testing.T) {
		merger, requests := newMerger(t, laterTip, nil, nil)
		confirmation, err := merger.ObserveCompletion(context.Background(), w)
		if err != nil {
			t.Fatalf("exact transitive witness refused: %v", err)
		}
		if confirmation.MergedCommit != w.HeadOID || confirmation.BaseTip != laterTip || !confirmation.BaseContainsMergedCommit ||
			confirmation.MergerID != w.ActorID || confirmation.MergedUnix <= 0 || confirmation.ClosedUnix <= 0 || !confirmation.IssueClosed {
			t.Fatalf("completion attribution changed: %+v", confirmation)
		}
		if got := ancestryRequests(*requests); len(got) != 1 || got[0].Ancestry == nil || got[0].Ancestry.AncestorOID != w.HeadOID || got[0].Ancestry.DescendantOID != laterTip {
			t.Fatalf("missing exact ancestry request: %+v", got)
		}
	})

	t.Run("equal head needs no extra witness", func(t *testing.T) {
		merger, requests := newMerger(t, w.HeadOID, nil, nil)
		confirmation, err := merger.ObserveCompletion(context.Background(), w)
		if err != nil || confirmation.BaseTip != w.HeadOID || !confirmation.BaseContainsMergedCommit {
			t.Fatalf("equal-head completion: %+v %v", confirmation, err)
		}
		if got := ancestryRequests(*requests); len(got) != 0 {
			t.Fatalf("equal head made an unnecessary ancestry read: %+v", got)
		}
	})

	for _, tc := range []struct {
		name   string
		mutate func(*extensions.NativeSnapshot, extensions.SnapshotRequest)
	}{
		{name: "false witness", mutate: func(s *extensions.NativeSnapshot, _ extensions.SnapshotRequest) { s.Ancestry.Reachable = false }},
		{name: "absent witness", mutate: func(s *extensions.NativeSnapshot, _ extensions.SnapshotRequest) { s.Ancestry = nil }},
		{name: "wrong ancestor", mutate: func(s *extensions.NativeSnapshot, _ extensions.SnapshotRequest) { s.Ancestry.AncestorOID = otherTip }},
		{name: "wrong descendant", mutate: func(s *extensions.NativeSnapshot, _ extensions.SnapshotRequest) { s.Ancestry.DescendantOID = otherTip }},
		{name: "base ref moved during witness", mutate: func(s *extensions.NativeSnapshot, _ extensions.SnapshotRequest) { s.Refs[0].OID = otherTip }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			merger, _ := newMerger(t, laterTip, nil, tc.mutate)
			if confirmation, err := merger.ObserveCompletion(context.Background(), w); err == nil {
				t.Fatalf("invalid ancestry evidence completed merge: %+v", confirmation)
			}
		})
	}

	t.Run("native revision moved", func(t *testing.T) {
		reads := 0
		merger, _ := newMerger(t, laterTip, func() int64 {
			reads++
			if reads >= 5 {
				return 13
			}
			return 12
		}, nil)
		if confirmation, err := merger.ObserveCompletion(context.Background(), w); err == nil {
			t.Fatalf("revision-moved witness completed merge: %+v", confirmation)
		}
	})
}

func completionWireSnapshot(w factory.MergeWork, req extensions.SnapshotRequest, baseTip string) extensions.NativeSnapshot {
	answer := extensions.NativeSnapshot{RepositoryID: strconv.FormatInt(w.Repository, 10)}
	for _, family := range req.Families {
		switch SnapshotFamily(family) {
		case FamilyIssue:
			answer.Issue = &extensions.SnapshotIssue{
				ID: strconv.FormatInt(w.IssueID, 10), Index: strconv.FormatInt(w.PRNumber, 10),
				Title: "merge objective", Content: "body", TitleDigest: ContentDigest("merge objective"), ContentDigest: ContentDigest("body"),
				Visible: true, Complete: true, IsPull: true,
				Provenance: extensions.SnapshotCreationProvenance{PosterID: strconv.FormatInt(w.PRAuthorID, 10)},
			}
		case FamilyPull:
			answer.Pull = &extensions.SnapshotPull{
				ID: strconv.FormatInt(w.PRID, 10), IssueID: strconv.FormatInt(w.IssueID, 10), Number: strconv.FormatInt(w.PRNumber, 10),
				HeadRepoID: strconv.FormatInt(w.Repository, 10), HeadBranch: factory.RefHead(w.HeadRef), HeadTip: w.HeadOID,
				BaseBranch: factory.RefHead(w.BaseRef), HasMerged: true, MergedCommit: w.HeadOID,
				MergerID: strconv.FormatInt(w.ActorID, 10), MergedUnix: 1100, Visible: true, Complete: true,
			}
		case FamilyRefs:
			for _, ref := range req.Refs {
				oid := w.HeadOID
				if ref == w.BaseRef {
					oid = baseTip
				}
				answer.Refs = append(answer.Refs, extensions.SnapshotRef{Ref: ref, OID: oid, Exists: true, Visible: true, Complete: true})
			}
		case FamilyAncestry:
			if req.Ancestry != nil {
				answer.Ancestry = &extensions.SnapshotAncestry{AncestorOID: req.Ancestry.AncestorOID, DescendantOID: req.Ancestry.DescendantOID, Reachable: true}
			}
		}
	}
	if answer.Issue != nil {
		answer.Issue.IsClosed = true
		answer.Issue.ClosedUnix = 1200
	}
	return answer
}

func containsSnapshotFamily(families []string, want SnapshotFamily) bool {
	for _, family := range families {
		if SnapshotFamily(family) == want {
			return true
		}
	}
	return false
}

func ancestryRequests(requests []extensions.SnapshotRequest) []extensions.SnapshotRequest {
	var out []extensions.SnapshotRequest
	for _, request := range requests {
		if containsSnapshotFamily(request.Families, FamilyAncestry) {
			out = append(out, request)
		}
	}
	return out
}

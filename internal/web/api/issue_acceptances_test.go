package api

import (
	"context"
	"errors"
	"testing"

	extensions "forgejo.org/extension-sdk"

	"github.com/levitateos/sodaos/internal/factory/control"
	"github.com/levitateos/sodaos/internal/forgejo"
)

type stubAcceptanceReader struct {
	before   extensions.NativeRevisionObservation
	after    extensions.NativeRevisionObservation
	snapshot forgejo.NativeSnapshot
	err      error
	requests []forgejo.SnapshotRequest
}

func (s *stubAcceptanceReader) ReadNativeRevision(context.Context) (extensions.NativeRevisionObservation, error) {
	if len(s.requests) == 0 {
		return s.before, nil
	}
	return s.after, nil
}

func (s *stubAcceptanceReader) ReadSnapshot(_ context.Context, _ extensions.CredentialFile, req forgejo.SnapshotRequest) (forgejo.NativeSnapshot, error) {
	s.requests = append(s.requests, req)
	return s.snapshot, s.err
}

func acceptanceWireSnapshot() forgejo.NativeSnapshot {
	return forgejo.NativeSnapshot{
		RepositoryID: "7",
		Issue: &forgejo.IssueEvidence{
			ID: "9", Index: "3", Title: "objective", Content: "body",
			TitleDigest: forgejo.ContentDigest("objective"), ContentDigest: forgejo.ContentDigest("body"),
			ContentVersion: 2, Complete: true, Visible: true,
			Provenance: forgejo.CreationProvenance{PosterID: "5", CreatedUnix: 41, FirstCreated: true, Verified: true},
			Lifecycle:  []forgejo.LifecycleEvent{{Kind: "retitle", AtUnix: 42, ActorID: "5"}},
		},
		Comments: &forgejo.CommentPage{
			IssueID: "9", Total: 2, Complete: true,
			Items: []forgejo.CommentEvidence{
				{ID: "11", IssueID: "9", Content: "answer",
					ContentDigest: forgejo.ContentDigest("answer"), ContentVersion: 0, Complete: true, Visible: true},
				{ID: "13", IssueID: "9", Content: "chatter",
					ContentDigest: forgejo.ContentDigest("chatter"), ContentVersion: 0, Complete: true, Visible: true},
			},
		},
		Dependencies: &forgejo.DependencyPage{
			IssueID: "9", Total: 1, Complete: true,
			Items: []forgejo.DependencyEvidence{{OccurrenceID: "21", IssueID: "9", DependencyID: "8", Complete: true, Visible: true}},
		},
	}
}

func idleAcceptanceObservation(rev int64) extensions.NativeRevisionObservation {
	return extensions.NativeRevisionObservation{Revision: rev, Idle: true}
}

func TestAcceptanceSnapshotSourceBracketsSelection(t *testing.T) {
	reader := &stubAcceptanceReader{before: idleAcceptanceObservation(41), after: idleAcceptanceObservation(41), snapshot: acceptanceWireSnapshot()}
	got, err := AcceptanceSnapshotSource{Reader: reader}.ReadAcceptanceEvidence(context.Background(), "7", "3", []string{"11"})
	if err != nil {
		t.Fatal(err)
	}
	if got.Revision != 41 || got.Issue.Index != "3" || got.Issue.ContentVer != 2 || got.Issue.Lifecycle != 1 {
		t.Fatalf("issue: %+v", got.Issue)
	}
	if !got.Issue.Verified || !got.Issue.FirstCreated || got.Issue.PosterID != "5" || !got.Issue.Visible {
		t.Fatalf("provenance: %+v", got.Issue)
	}
	if len(got.Comments) != 1 || got.Comments[0].ID != "11" || got.Comments[0].Digest != forgejo.ContentDigest("answer") {
		t.Fatalf("comments: %+v", got.Comments)
	}
	if len(got.Dependencies) != 1 || got.Dependencies[0].Occurrence != "21" || got.Dependencies[0].DependsOn != "8" {
		t.Fatalf("edges: %+v", got.Dependencies)
	}
	if len(reader.requests) != 1 {
		t.Fatalf("requests: %d", len(reader.requests))
	}
	req := reader.requests[0]
	if req.RepositoryID != "7" || req.IssueIndex != "3" || req.Limit != forgejo.SnapshotPageLimit || req.Cursor != "" {
		t.Fatalf("request: %+v", req)
	}
	if len(req.Families) != 3 || req.Families[0] != forgejo.FamilyIssue || req.Families[1] != forgejo.FamilyDependencies || req.Families[2] != forgejo.FamilyComments {
		t.Fatalf("families: %+v", req.Families)
	}
	if req.CommentIDs != nil {
		t.Fatalf("comment IDs: %+v", req.CommentIDs)
	}
}

func TestAcceptanceSnapshotSourceSkipsUnselectedComments(t *testing.T) {
	snapshot := acceptanceWireSnapshot()
	snapshot.Comments = nil
	reader := &stubAcceptanceReader{before: idleAcceptanceObservation(41), after: idleAcceptanceObservation(41), snapshot: snapshot}
	got, err := AcceptanceSnapshotSource{Reader: reader}.ReadAcceptanceEvidence(context.Background(), "7", "3", nil)
	if err != nil {
		t.Fatal(err)
	}
	if got.Comments != nil {
		t.Fatalf("comments: %+v", got.Comments)
	}
	req := reader.requests[0]
	for _, family := range req.Families {
		if family == forgejo.FamilyComments {
			t.Fatalf("families: %+v", req.Families)
		}
	}
	if req.CommentIDs != nil {
		t.Fatalf("comment IDs: %+v", req.CommentIDs)
	}
}

func TestAcceptanceSnapshotSourceMapsBracketFailures(t *testing.T) {
	busy := extensions.NativeRevisionObservation{Revision: 41, Idle: false}
	overLimit := acceptanceWireSnapshot()
	overLimit.Dependencies.Complete = false
	cases := map[string]struct {
		before, after extensions.NativeRevisionObservation
		snapshot      forgejo.NativeSnapshot
		err           error
		nilReader     bool
		reason        string
		raw           bool
	}{
		"stale":      {before: idleAcceptanceObservation(41), after: idleAcceptanceObservation(42), snapshot: acceptanceWireSnapshot(), reason: control.RefusalStaleEvidence},
		"busy":       {before: busy, after: idleAcceptanceObservation(41), snapshot: acceptanceWireSnapshot(), reason: control.RefusalNativeBusy},
		"over-limit": {before: idleAcceptanceObservation(41), after: idleAcceptanceObservation(41), snapshot: overLimit, reason: control.RefusalIncompleteEvidence},
		"invalid":    {before: idleAcceptanceObservation(41), after: idleAcceptanceObservation(41), err: forgejo.ErrInvalidSnapshot, reason: control.RefusalSnapshotUnavailable},
		"unwired":    {nilReader: true, reason: control.RefusalSnapshotUnavailable},
		"transport":  {before: idleAcceptanceObservation(41), after: idleAcceptanceObservation(41), err: errors.New("boom"), raw: true},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			var source AcceptanceSnapshotSource
			if !tc.nilReader {
				source = AcceptanceSnapshotSource{Reader: &stubAcceptanceReader{before: tc.before, after: tc.after, snapshot: tc.snapshot, err: tc.err}}
			}
			_, err := source.ReadAcceptanceEvidence(context.Background(), "7", "3", []string{"11"})
			if tc.raw {
				if err == nil || err.Error() != "boom" {
					t.Fatalf("transport error: %v", err)
				}
				return
			}
			var refusal *control.AcceptanceRefusal
			if !errors.As(err, &refusal) || refusal.Reason != tc.reason {
				t.Fatalf("mapped: %v", err)
			}
		})
	}
}

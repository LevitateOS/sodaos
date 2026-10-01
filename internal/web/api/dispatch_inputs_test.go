package api

import (
	"context"
	"errors"
	"strings"
	"testing"

	extensions "forgejo.org/extension-sdk"

	"github.com/levitateos/sodaos/internal/factory/control"
	"github.com/levitateos/sodaos/internal/forgejo"
)

type dispatchReaderFake struct {
	snapshot forgejo.NativeSnapshot
	revision int64
	idle     bool
	err      error
}

func (f *dispatchReaderFake) ReadNativeRevision(ctx context.Context) (extensions.NativeRevisionObservation, error) {
	return extensions.NativeRevisionObservation{Revision: f.revision, Idle: f.idle}, nil
}

func (f *dispatchReaderFake) ReadSnapshot(ctx context.Context, credential extensions.CredentialFile, req forgejo.SnapshotRequest) (forgejo.NativeSnapshot, error) {
	if f.err != nil {
		return forgejo.NativeSnapshot{}, f.err
	}
	return f.snapshot, nil
}

func dispatchTestSnapshot() forgejo.NativeSnapshot {
	return forgejo.NativeSnapshot{
		Revision: 41, RepositoryID: "7",
		Issue: &forgejo.IssueEvidence{
			ID: "3", Index: "3", Title: "title", Content: "body",
			TitleDigest: forgejo.ContentDigest("title"), ContentDigest: forgejo.ContentDigest("body"),
			ContentVersion: 2, Visible: true, Complete: true,
		},
		Comments: &forgejo.CommentPage{
			IssueID: "3", Total: 2, Complete: true,
			Items: []forgejo.CommentEvidence{
				{ID: "11", IssueID: "3", Content: "answer", ContentDigest: forgejo.ContentDigest("answer"), Visible: true, Complete: true},
				{ID: "12", IssueID: "3", Content: "chatter", ContentDigest: forgejo.ContentDigest("chatter"), Visible: true, Complete: true},
			},
		},
		Refs: []forgejo.RefEvidence{
			{Ref: "refs/heads/main", OID: strings.Repeat("e", 40), Exists: true, Visible: true, Complete: true},
		},
	}
}

func TestDispatchSnapshotSourceMapsInputs(t *testing.T) {
	source := DispatchSnapshotSource{Reader: &dispatchReaderFake{snapshot: dispatchTestSnapshot(), revision: 41, idle: true}}
	inputs, err := source.ReadDispatchInputs(context.Background(), "7", "3", []string{"11"}, "refs/heads/main")
	if err != nil {
		t.Fatalf("inputs refused: %v", err)
	}
	if inputs.Revision != 41 || inputs.Issue.Title != "title" || inputs.Issue.Body != "body" {
		t.Fatalf("issue = %+v", inputs.Issue)
	}
	if len(inputs.Comments) != 1 || inputs.Comments[0].ID != "11" || inputs.Comments[0].Content != "answer" {
		t.Fatalf("comments = %+v", inputs.Comments)
	}
	if inputs.Tip != strings.Repeat("e", 40) || inputs.TipRef != "refs/heads/main" {
		t.Fatalf("tip = %q ref = %q", inputs.Tip, inputs.TipRef)
	}
}

func TestDispatchSnapshotSourceMissingTipReadsEmpty(t *testing.T) {
	snapshot := dispatchTestSnapshot()
	snapshot.Refs = []forgejo.RefEvidence{
		{Ref: "refs/heads/main", Exists: false, Visible: true, Complete: true},
	}
	source := DispatchSnapshotSource{Reader: &dispatchReaderFake{snapshot: snapshot, revision: 41, idle: true}}
	inputs, err := source.ReadDispatchInputs(context.Background(), "7", "3", []string{"11"}, "refs/heads/main")
	if err != nil {
		t.Fatalf("inputs refused: %v", err)
	}
	if inputs.Tip != "" {
		t.Fatalf("tip = %q", inputs.Tip)
	}
}

func TestDispatchSnapshotSourceMapsTransportFailures(t *testing.T) {
	for err, reason := range map[error]string{
		forgejo.ErrStaleSnapshot:      control.RefusalStaleEvidence,
		forgejo.ErrIncompleteSnapshot: control.RefusalIncompleteEvidence,
		forgejo.ErrNativeBusy:         control.RefusalNativeBusy,
		forgejo.ErrInvalidSnapshot:    control.RefusalSnapshotUnavailable,
	} {
		source := DispatchSnapshotSource{Reader: &dispatchReaderFake{err: err, revision: 41, idle: true}}
		_, got := source.ReadDispatchInputs(context.Background(), "7", "3", nil, "refs/heads/main")
		var refusal *control.AcceptanceRefusal
		if !errors.As(got, &refusal) || refusal.Reason != reason {
			t.Fatalf("err %v -> %+v, want %s", err, got, reason)
		}
	}
	var _ control.DispatchReads = DispatchSnapshotSource{}
}

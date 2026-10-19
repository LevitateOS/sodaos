package api

import (
	"context"
	extensions "forgejo.org/extension-sdk"
	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/forgejo"
	"github.com/levitateos/sodaos/internal/store"
	"testing"
)

type commentContextReader struct{}

func (commentContextReader) ReadNativeRevision(context.Context) (extensions.NativeRevisionObservation, error) {
	return extensions.NativeRevisionObservation{Revision: 1, Idle: true}, nil
}
func (commentContextReader) ReadSnapshot(_ context.Context, _ extensions.CredentialFile, r forgejo.SnapshotRequest) (forgejo.NativeSnapshot, error) {
	s := forgejo.NativeSnapshot{RepositoryID: r.RepositoryID, Revision: 1}
	if r.Families[0] == forgejo.FamilyRepositoryIssues {
		s.RepositoryIssues = &forgejo.RepositoryIssuesPage{Items: []forgejo.RepositoryIssueItem{{IssueID: "11", IssueIndex: "1", Visible: true}}, Complete: true}
		return s, nil
	}
	title, body := "Add a greeting page", "The page displays HELLO."
	s.Issue = &forgejo.IssueEvidence{ID: "11", Index: "1", Title: title, Content: body, TitleDigest: forgejo.ContentDigest(title), ContentDigest: forgejo.ContentDigest(body), Visible: true, Complete: true}
	return s, nil
}
func TestCommentRoutingUsesNativeObjectivesAndIncludesUntouchedIssues(t *testing.T) {
	for _, candidates := range [][]factory.CommentWorkCandidate{nil, {{Issue: 1, Summary: "stopped", Status: "cancelled", Origin: factory.CommentCandidateRecent}}} {
		in := CommentJudgeInput{Comment: store.FeedCommentDetail{Record: factory.CommentRecord{Repository: 2}}, Candidates: candidates}
		got, err := prepareNativeCommentInput(context.Background(), commentContextReader{}, "", in)
		if err != nil {
			t.Fatal(err)
		}
		if len(got.Candidates) != 1 || got.Candidates[0].Title != "Add a greeting page" || got.Candidates[0].Objective != "The page displays HELLO." {
			t.Fatalf("missing objective: %+v", got.Candidates)
		}
		if len(candidates) > 0 && got.Candidates[0].WorkerStatus != "cancelled" {
			t.Fatal("worker status displaced or lost issue meaning")
		}
	}
}

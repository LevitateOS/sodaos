package api

import (
	"context"
	"errors"
	extensions "forgejo.org/extension-sdk"
	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/forgejo"
	"github.com/levitateos/sodaos/internal/store"
	"strconv"
)

// Comment matching reads issue meaning from the native issue owner. Worker
// summaries remain execution context and cannot substitute for an objective.
func (s *API) prepareCommentJudgeInput(ctx context.Context, in CommentJudgeInput) (CommentJudgeInput, error) {
	var observer *forgejo.ServiceObserver
	if s.Coordinator != nil {
		if source, ok := s.Coordinator.Readiness.(*ServiceReadinessSource); ok {
			observer = source.Observer
		}
	}
	if observer == nil {
		return in, errors.New("Issue context is unavailable. Reconnect Factory before retrying assessment.")
	}
	reader, credential := observer.SnapshotReader(), observer.Credential()
	return prepareNativeCommentInput(ctx, reader, credential, in)
}

func prepareNativeCommentInput(ctx context.Context, reader forgejo.SnapshotReader, credential extensions.CredentialFile, in CommentJudgeInput) (CommentJudgeInput, error) {
	repository := strconv.FormatInt(in.Comment.Record.Repository, 10)
	byIssue := map[int64]factory.CommentWorkCandidate{}
	var order []int64
	add := func(candidate factory.CommentWorkCandidate) {
		if _, ok := byIssue[candidate.Issue]; !ok {
			order = append(order, candidate.Issue)
		}
		byIssue[candidate.Issue] = candidate
	}
	for _, candidate := range in.Candidates {
		if candidate.Origin != factory.CommentCandidateRecent {
			add(candidate)
		}
	}
	cursor := ""
	for {
		page, err := reader.ReadSnapshot(ctx, credential, forgejo.SnapshotRequest{RepositoryID: repository, Families: []forgejo.SnapshotFamily{forgejo.FamilyRepositoryIssues}, Limit: 50, Cursor: cursor})
		if err != nil || page.RepositoryIssues == nil {
			return in, errors.New("Could not read this project's issues. Assessment will need a retry.")
		}
		if err := forgejo.ValidateRepositoryIssuesPage(page.RepositoryIssues, repository, cursor); err != nil {
			return in, err
		}
		for _, item := range page.RepositoryIssues.Items {
			if !item.Visible {
				continue
			}
			index, err := strconv.ParseInt(item.IssueIndex, 10, 64)
			if err != nil {
				return in, err
			}
			if _, ok := byIssue[index]; !ok {
				add(factory.CommentWorkCandidate{Issue: index, Origin: factory.CommentCandidateRecent})
			}
		}
		if len(order) > 250 {
			return in, errors.New("Choose the issue this request concerns; there are too many open issues to assess together.")
		}
		if page.RepositoryIssues.Complete {
			break
		}
		cursor = page.RepositoryIssues.NextCursor
	}
	candidates := make([]factory.CommentWorkCandidate, 0, len(order))
	for _, index := range order {
		snapshot, err := forgejo.BracketedRead(ctx, reader, credential, forgejo.SnapshotRequest{RepositoryID: repository, IssueIndex: strconv.FormatInt(index, 10), Families: []forgejo.SnapshotFamily{forgejo.FamilyIssue}, Limit: 50})
		if err != nil || snapshot.Issue == nil || !snapshot.Issue.Visible {
			return in, errors.New("Issue details changed or are unavailable. Retry assessment after refreshing.")
		}
		candidate := byIssue[index]
		candidate.Title = snapshot.Issue.Title
		body := []rune(snapshot.Issue.Content)
		if len(body) > 4000 {
			body = body[:4000]
		}
		candidate.Objective = string(body)
		for _, old := range in.Candidates {
			if old.Issue == index {
				candidate.WorkerStatus = old.Status
				break
			}
		}
		candidate.Summary = store.TruncateFeedSummary(snapshot.Issue.Title + "\n" + snapshot.Issue.Content)
		candidate.Status = "open"
		if snapshot.Issue.IsClosed {
			candidate.Status = "closed"
		}
		if err := candidate.Validate(); err != nil {
			return in, err
		}
		candidates = append(candidates, candidate)
	}
	in.Candidates = candidates
	return in, nil
}

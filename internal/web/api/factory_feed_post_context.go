package api

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/store"
)

var errPostIssueContext = errors.New("Refresh this request to read its issues using your current repository access.")

// Read issue meaning under the submitting human's live native admission. Save
// only bounded context for background routing; never retain the admission or
// enroll a Factory execution account as a side effect of posting.
func (s *API) capturePostIssueContext(r *http.Request, id string, revision int64) error {
	authority, ok := requestExtensionAuthority(r)
	if !ok {
		return errors.New("Native repository access is unavailable.")
	}
	detail, err := s.Store.FeedPost(r.Context(), id)
	if err != nil {
		return err
	}
	input, err := collectPostJudgeInput(r.Context(), s.Store, detail)
	if err != nil {
		return err
	}
	repository := strconv.FormatInt(detail.Record.Repository, 10)
	byIssue := map[int64]factory.PostWorkCandidate{}
	order := []int64{}
	add := func(index, title, body string, closed bool) error {
		number, err := strconv.ParseInt(index, 10, 64)
		if err != nil {
			return err
		}
		candidate := factory.PostWorkCandidate{Issue: number, Title: title, Objective: body, Summary: store.TruncateFeedSummary(title + "\n" + body), Status: "open", Origin: factory.PostCandidateRecent}
		if closed {
			candidate.Status = "closed"
		}
		for _, previous := range input.Candidates {
			if previous.Issue == number {
				candidate.Origin = previous.Origin
				candidate.WorkerStatus = previous.Status
				break
			}
		}
		if err := candidate.Validate(); err != nil {
			return err
		}
		if _, exists := byIssue[number]; !exists {
			order = append(order, number)
		}
		byIssue[number] = candidate
		return nil
	}
	cursor := ""
	for {
		page, err := authority.RepositoryIssues(r.Context(), repository, cursor, "", 50)
		if err != nil {
			return err
		}
		for _, issue := range page.Items {
			if err := add(issue.Index, issue.Title, issue.Body, issue.Closed); err != nil {
				return err
			}
		}
		if len(order) > 250 {
			return errors.New("Choose the issue this request concerns; there are too many open issues to assess together.")
		}
		if page.NextCursor == "" {
			break
		}
		if len(page.Items) == 0 || page.NextCursor == cursor {
			return errors.New("Issue context pagination did not advance.")
		}
		cursor = page.NextCursor
	}
	// A bound/source/parent issue can be closed and absent from the open list.
	for _, candidate := range input.Candidates {
		if candidate.Origin == factory.PostCandidateRecent {
			continue
		}
		if _, exists := byIssue[candidate.Issue]; exists {
			continue
		}
		page, err := authority.RepositoryIssues(r.Context(), repository, "", strconv.FormatInt(candidate.Issue, 10), 1)
		if err != nil {
			return err
		}
		if len(page.Items) != 1 {
			return errors.New("The selected issue is no longer visible.")
		}
		issue := page.Items[0]
		if err := add(issue.Index, issue.Title, issue.Body, issue.Closed); err != nil {
			return err
		}
	}
	candidates := make([]factory.PostWorkCandidate, 0, len(order))
	for _, index := range order {
		candidates = append(candidates, byIssue[index])
	}
	return s.Store.SavePostIssueContext(r.Context(), id, revision, candidates)
}

func (s *API) preparePostJudgeInput(ctx context.Context, in PostJudgeInput) (PostJudgeInput, error) {
	candidates, err := s.Store.PostIssueContext(ctx, in.SourcePost.Record.ID, in.SourcePost.Record.Revision)
	if err != nil {
		return in, errPostIssueContext
	}
	in.Candidates = candidates
	return in, nil
}

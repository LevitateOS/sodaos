package forgejo

import (
	"context"
	"net/url"
	"strconv"
	"strings"

	"github.com/levitateos/sodaos/internal/config"
)

// DeliverIssueFeedback uses Forgejo's supported comment API. The caller first
// checks the human's current issue-write authority. A source marker lets a lost
// response reconcile to the existing native comment without rewriting the issue.
func (w *IssueGraphWriter) DeliverIssueFeedback(ctx context.Context, repository, issue, actor int64, marker, body string) error {
	if repository <= 0 || issue <= 0 || marker == "" || strings.TrimSpace(body) == "" {
		return ErrInvalidResponse
	}
	if err := w.checkActor(ctx, actor); err != nil {
		return err
	}
	token, err := config.Secret(w.tokenFile)
	if err != nil {
		return ErrUnavailable
	}
	var repo Repository
	if err = w.rest.request(ctx, "GET", "/repositories/"+strconv.FormatInt(repository, 10), token, nil, &repo); err != nil {
		return err
	}
	if repo.ID != repository || repo.Owner.Login == "" || repo.Name == "" {
		return ErrInvalidResponse
	}
	path := "/repos/" + url.PathEscape(repo.Owner.Login) + "/" + url.PathEscape(repo.Name) + "/issues/" + strconv.FormatInt(issue, 10) + "/comments"
	type comment struct {
		ID   int64  `json:"id"`
		Body string `json:"body"`
		User User   `json:"user"`
	}
	for page := 1; ; page++ {
		var comments []comment
		if err = w.rest.request(ctx, "GET", path+"?limit=50&page="+strconv.Itoa(page), token, nil, &comments); err != nil {
			return err
		}
		for _, c := range comments {
			if c.ID > 0 && c.User.ID == actor && strings.HasPrefix(c.Body, marker+"\n") {
				return nil
			}
		}
		if len(comments) < 50 {
			break
		}
	}
	var saved comment
	if err = w.rest.request(ctx, "POST", path, token, map[string]string{"body": marker + "\n" + body}, &saved); err != nil {
		return err
	}
	if saved.ID <= 0 || saved.User.ID != actor || !strings.HasPrefix(saved.Body, marker+"\n") {
		return ErrInvalidResponse
	}
	return nil
}

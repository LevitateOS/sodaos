package forgejo

import (
	"context"
	"errors"
	"net/url"
	"strconv"
	"strings"

	"github.com/levitateos/factory/pkg/collaboration"
	"github.com/levitateos/factory/pkg/factory"
	"github.com/levitateos/sodaos/internal/config"
)

func (w *IssueGraphWriter) FeedbackHost() collaboration.HostID {
	if w == nil {
		return ""
	}
	return w.Identities.HostID()
}

// DeliverIssueFeedback consumes the Go journal's one send permit. Recovery uses
// an exact receipt or at most two legacy comment pages; absence never sends.
func (w *IssueGraphWriter) DeliverIssueFeedback(ctx context.Context, delivery factory.PostFeedbackDelivery, send bool, beforeSend func(context.Context) error) (factory.PostFeedbackDelivery, error) {
	if delivery.Validate() != nil || delivery.Intent.Host != string(w.FeedbackHost()) || send && (delivery.Legacy || delivery.Receipt != "" || delivery.Cursor != 1 || delivery.Complete) || beforeSend == nil {
		return delivery, ErrInvalidResponse
	}
	intent := delivery.Intent
	nativeRepository, err := w.Identities.NativeRepository(ctx, collaboration.RepositoryID(intent.Repository))
	if err != nil {
		return delivery, err
	}
	nativeActor, err := w.checkActor(ctx, intent.Actor)
	if err != nil {
		return delivery, err
	}
	token, err := config.Secret(w.tokenFile)
	if err != nil {
		return delivery, ErrUnavailable
	}
	var repo Repository
	if err = w.rest.request(ctx, "GET", "/repositories/"+nativeRepository, token, nil, &repo); err != nil {
		return delivery, err
	}
	if strconv.FormatInt(repo.ID, 10) != nativeRepository || repo.Owner.Login == "" || repo.Name == "" {
		return delivery, ErrInvalidResponse
	}
	base := "/repos/" + url.PathEscape(repo.Owner.Login) + "/" + url.PathEscape(repo.Name)
	path := base + "/issues/" + strconv.FormatInt(intent.Issue, 10) + "/comments"
	type comment struct {
		ID       int64  `json:"id"`
		Body     string `json:"body"`
		User     User   `json:"user"`
		IssueURL string `json:"issue_url"`
	}
	expected := intent.Marker + "\n" + intent.Body
	api, parseErr := url.Parse(w.rest.Base)
	if parseErr != nil {
		return delivery, ErrInvalidResponse
	}
	issuePath := strings.TrimSuffix(strings.TrimSuffix(api.Path, "/"), "/api/v1") + "/" + repo.Owner.Login + "/" + repo.Name + "/issues/" + strconv.FormatInt(intent.Issue, 10)
	accept := func(saved comment) (bool, error) {
		if saved.ID <= 0 || saved.User.ID != nativeActor || !strings.HasPrefix(saved.Body, intent.Marker+"\n") {
			return false, nil
		}
		observed, parseErr := url.Parse(saved.IssueURL)
		if parseErr != nil || observed.Scheme != api.Scheme || observed.Host != api.Host || observed.User != nil || observed.Path != issuePath || observed.RawQuery != "" || observed.Fragment != "" {
			return false, ErrInvalidResponse
		}
		delivery.Receipt = strconv.FormatInt(saved.ID, 10)
		if saved.Body != expected {
			delivery.Complete = true
			delivery.Attention = "content_changed"
			return true, errors.New("recorded feedback text changed; preserved")
		}
		delivery.Receipt = strconv.FormatInt(saved.ID, 10)
		delivery.Complete = true
		return true, nil
	}
	if send {
		// The enrolled service credential is the actor; no request-bound human SDK
		// or source author credential is substituted here.
		if repo.Permissions == nil || !repo.Permissions.Push && !repo.Permissions.Admin {
			return delivery, ErrUnavailable
		}
		if err = beforeSend(ctx); err != nil {
			delivery.Complete = true
			delivery.Attention = "source_changed_before_send"
			return delivery, err
		}
		var saved comment
		if err = w.rest.request(ctx, "POST", path, token, map[string]string{"body": expected}, &saved); err != nil {
			return delivery, err
		}
		matched, err := accept(saved)
		if !matched && err == nil {
			err = ErrInvalidResponse
		}
		return delivery, err
	}
	if delivery.Receipt != "" {
		var saved comment
		if err = w.rest.request(ctx, "GET", base+"/issues/comments/"+delivery.Receipt, token, nil, &saved); err != nil {
			return delivery, err
		}
		if strconv.FormatInt(saved.ID, 10) != delivery.Receipt {
			return delivery, ErrInvalidResponse
		}
		matched, err := accept(saved)
		if !matched && err == nil {
			err = ErrInvalidResponse
		}
		return delivery, err
	}
	if delivery.Complete {
		return delivery, nil
	}
	for step := 0; step < 2; step++ {
		var comments []comment
		if err = w.rest.request(ctx, "GET", path+"?limit=50&page="+strconv.Itoa(delivery.Cursor), token, nil, &comments); err != nil {
			return delivery, err
		}
		if len(comments) > 50 {
			return delivery, ErrInvalidResponse
		}
		for _, saved := range comments {
			matched, err := accept(saved)
			if matched || err != nil {
				return delivery, err
			}
		}
		if len(comments) < 50 {
			delivery.Complete = true
			delivery.Attention = "not_found"
			return delivery, nil
		}
		if delivery.Cursor == 1000000 {
			delivery.Complete = true
			delivery.Attention = "search_limit"
			return delivery, nil
		}
		delivery.Cursor++
	}
	return delivery, nil
}

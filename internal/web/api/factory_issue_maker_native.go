package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	sdk "forgejo.org/extension-sdk"
	"github.com/levitateos/sodaos/internal/factory"
)

type IssueMakerNative interface {
	ApplyIssueIntake(context.Context, sdk.IssueIntakeMutation) (sdk.IssueIntakeReceipt, error)
}

func (s *API) issueMakerNative() IssueMakerNative {
	if s.IssueMakerNative != nil {
		return s.IssueMakerNative
	}
	if s.ExtensionManager != nil {
		return s.ExtensionManager
	}
	return nil
}

func (s *API) captureIssueIntakeGrant(r *http.Request, id string, revision int64) error {
	comment, err := s.Store.FeedPost(r.Context(), id)
	if err != nil {
		return err
	}
	if grant, err := s.Store.IssueIntakeGrant(r.Context(), id, revision); err == nil && grant.Claims.Expires > time.Now().Unix() {
		return nil
	}
	authority, ok := requestExtensionAuthority(r)
	if !ok {
		return errors.New("native issue-write admission is unavailable")
	}
	grant, err := authority.AuthorizeIssueIntake(r.Context(), strconv.FormatInt(comment.Record.Repository, 10), sdk.IssueIntakeAuthorizationRequest{Source: id, Revision: revision, Digest: factory.MCPInputDigest([]byte(comment.Body))})
	if err != nil {
		return err
	}
	return s.Store.SaveIssueIntakeGrant(r.Context(), id, revision, grant)
}

type intakePostIssueCreator struct{ API *API }

func (c *intakePostIssueCreator) LookupPostIssue(context.Context, int64, string) (int64, error) {
	return 0, nil
}
func (c *intakePostIssueCreator) CreatePostIssue(ctx context.Context, draft factory.IssueDraft, repository int64) (int64, error) {
	grant, err := c.API.Store.IssueIntakeGrant(ctx, draft.SourcePost, draft.Revision)
	if err != nil {
		return 0, errors.New("retry the original request to authorize native issue creation")
	}
	if grant.Claims.Repository != strconv.FormatInt(repository, 10) {
		return 0, errors.New("issue intake repository changed")
	}
	writer := c.API.issueMakerNative()
	if writer == nil {
		return 0, errors.New("native issue intake is unavailable")
	}
	receipt, err := writer.ApplyIssueIntake(ctx, sdk.IssueIntakeMutation{Grant: grant.Token, Action: "create", Slot: 0, Step: 0, Title: draft.Title, Body: postIssueBody(draft)})
	return receipt.Index, err
}

func (s *API) mutateIssueMakerIssue(ctx context.Context, scope factory.IssueMakerScope, action, key, title, body, operation string) (int64, error) {
	if !factory.ValidPlanKey(key) || strings.TrimSpace(title) == "" || len(title) > 255 || strings.TrimSpace(body) == "" || len(body) > 16384 {
		return 0, errors.New("issue needs a stable key, title (255 bytes) and body (16384 bytes)")
	}
	comment, err := s.Store.FeedPost(ctx, scope.SourcePost)
	if err != nil || comment.Record.Revision != scope.Revision || comment.Record.Deleted || comment.Record.Revoked {
		return 0, errors.New("source request is no longer active")
	}
	progress, err := s.Store.PostIssueMaker(ctx, scope.SourcePost, scope.Revision)
	if err != nil || progress.Scope.Post != scope.Post || progress.State == "failed" || progress.State == "completed" {
		return 0, errors.New("Issue Maker is no longer active")
	}
	request := sdk.IssueIntakeMutation{Action: action, Title: title, Body: body}
	found := false
	for _, issue := range progress.Issues {
		if issue.Key == key {
			found = true
			request.ExpectedTitle, request.ExpectedBodyDigest = issue.Title, issue.BodyDigest
		}
	}
	if action == "refine" && !found {
		return 0, errors.New("Issue Maker may refine only issues it created for this request")
	}
	if action != "create" && action != "refine" {
		return 0, errors.New("unsupported issue action")
	}
	meaning, _ := json.Marshal([]string{action, key, title, body})
	request, err = s.Store.PrepareIssueMakerMutation(ctx, scope.Post, operation, factory.MCPInputDigest(meaning), key, request)
	if err != nil {
		return 0, err
	}
	grant, err := s.Store.IssueIntakeGrant(ctx, scope.SourcePost, scope.Revision)
	if err != nil {
		return 0, errors.New("native issue intake grant is unavailable")
	}
	if grant.Claims.Actor != comment.Record.Author || grant.Claims.Repository != strconv.FormatInt(comment.Record.Repository, 10) {
		return 0, errors.New("native issue intake scope changed")
	}
	request.Grant = grant.Token
	writer := s.issueMakerNative()
	if writer == nil {
		return 0, errors.New("native issue writer is unavailable")
	}
	receipt, err := writer.ApplyIssueIntake(ctx, request)
	if err != nil {
		return 0, err
	}
	if err = s.Store.RecordIssueMakerIssue(ctx, scope, key, request.Slot, receipt.Index, title, body); err != nil {
		return 0, err
	}
	return receipt.Index, nil
}

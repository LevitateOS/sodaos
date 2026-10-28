package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"

	"github.com/levitateos/sodaos/internal/factory"
)

// CommentProductChange is a read projection of the explicit review handoff.
// Accepted describes product scope only, not generation, verification or
// release; the build request it opens shows those.
//
// CommentProductStateUnavailable names an explicit unknown projection: the
// comment sits on the same-product lane but the read behind product_change
// failed.
const CommentProductStateUnavailable = "unavailable"

// UnavailableCommentProductChange projects an explicit unknown for a
// same-product comment whose read failed.
func UnavailableCommentProductChange(intent, acceptedRevision string) *CommentProductChange {
	return &CommentProductChange{Intent: intent, AcceptedRevision: acceptedRevision, State: CommentProductStateUnavailable}
}

type CommentProductChange struct {
	Intent           string `json:"intent"`
	AcceptedRevision string `json:"accepted_revision"`
	DraftRevision    string `json:"draft_revision,omitempty"`
	State            string `json:"state"`
	CanReview        bool   `json:"can_review"`
}

type commentProductBinding struct {
	Revision int64  `json:"revision"`
	Draft    string `json:"draft"`
}

// acceptedCommentProduct checks an immutable canonical source. It does not
// guess lineage from a headline, issue similarity or repository membership.
func (s *Store) acceptedCommentProduct(ctx context.Context, repository int64, intent, revision string) (*factory.CommentProductSource, error) {
	if intent == "" || revision == "" {
		return nil, nil
	}
	var matches bool
	err := s.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM factory_intent_revisions WHERE id=$1 AND intent=$2 AND data->>'repository'=$3 AND data->>'state'='accepted')`, revision, intent, strconv.FormatInt(repository, 10)).Scan(&matches)
	if err != nil || !matches {
		return nil, err
	}
	return &factory.CommentProductSource{Intent: intent, AcceptedRevision: revision}, nil
}

func (s *Store) assignmentCommentProduct(ctx context.Context, a factory.Assignment) (*factory.CommentProductSource, error) {
	if a.Subject.Delivery != nil {
		return s.acceptedCommentProduct(ctx, a.Repository, a.Subject.Delivery.IntentID, a.Subject.Delivery.AcceptedRevision)
	}
	if a.Subject.Intent != nil {
		return s.acceptedCommentProduct(ctx, a.Repository, a.Subject.Intent.IntentID, a.Subject.Intent.RevisionID)
	}
	return nil, nil
}

func (s *Store) postCommentProduct(ctx context.Context, detail FeedCommentDetail) (*factory.CommentProductSource, error) {
	if detail.Payload != nil && detail.Payload.Communication != nil {
		if source := detail.Payload.Communication.IntentSource; source != nil && source.State == "accepted" {
			return s.acceptedCommentProduct(ctx, detail.Record.Repository, source.Intent, source.Revision)
		}
	}
	if detail.Source != nil && detail.Source.Product != nil {
		p := detail.Source.Product
		return s.acceptedCommentProduct(ctx, detail.Record.Repository, p.Intent, p.AcceptedRevision)
	}
	if detail.Assignment != "" {
		a, err := s.Assignment(ctx, detail.Assignment)
		if err != nil {
			return nil, err
		}
		if a.Repository == detail.Record.Repository {
			return s.assignmentCommentProduct(ctx, a)
		}
	}
	return nil, nil
}

func (s *Store) CommentProductChange(ctx context.Context, detail FeedCommentDetail, pipe factory.CommentPipeline, actor int64) (*CommentProductChange, error) {
	if detail.Source == nil || detail.Source.Product == nil || pipe.Judgment != factory.CommentJudgmentApplied || pipe.Route != factory.CommentRouteNewIssue {
		return nil, nil
	}
	p := detail.Source.Product
	view := &CommentProductChange{Intent: p.Intent, AcceptedRevision: p.AcceptedRevision, State: "superseded"}
	var raw, headRaw []byte
	var binding commentProductBinding
	err := s.db.QueryRowContext(ctx, `SELECT COALESCE(c.data->'product_change','{}'::jsonb),r.data FROM factory_feed_comments c JOIN factory_intent_heads h ON h.id=$2 JOIN factory_intent_revisions r ON r.id=h.revision WHERE c.id=$1`, detail.Record.ID, p.Intent).Scan(&raw, &headRaw)
	if err != nil {
		return nil, err
	}
	var head factory.IntentRevision
	if err = json.Unmarshal(raw, &binding); err != nil {
		return nil, err
	}
	if err = json.Unmarshal(headRaw, &head); err != nil {
		return nil, err
	}
	if head.Repository != detail.Record.Repository {
		return nil, ErrNotFound
	}
	if binding.Draft != "" {
		view.DraftRevision = binding.Draft
	}
	if detail.Record.Deleted || detail.Record.Revoked || binding.Draft != "" && binding.Revision != detail.Record.Revision {
		return view, nil
	}
	if binding.Draft == "" {
		if head.ID == p.AcceptedRevision && head.State == "accepted" && pipe.Issue == 0 && pipe.Draft == factory.CommentDraftPending && pipe.Effect == factory.CommentEffectPending {
			view.State = "review_required"
			view.CanReview = head.Owner == actor
		}
		return view, nil
	}
	var descended bool
	err = s.db.QueryRowContext(ctx, `WITH RECURSIVE ancestry AS (SELECT id,data FROM factory_intent_revisions WHERE id=$1 AND intent=$3 UNION ALL SELECT r.id,r.data FROM factory_intent_revisions r JOIN ancestry a ON r.id=a.data->>'predecessor' WHERE r.intent=$3) SELECT EXISTS(SELECT 1 FROM ancestry WHERE id=$2)`, head.ID, binding.Draft, p.Intent).Scan(&descended)
	if err != nil {
		return nil, err
	}
	if descended && head.State == "accepted" && effectiveCommentProductIdea(head, detail) {
		view.State = "accepted"
		// Only the head owner may open the private build request behind
		// the accepted direction; this bit authorizes navigation only.
		view.CanReview = head.Owner == actor
	} else if descended && head.State == "draft" {
		view.State = "draft"
		view.CanReview = head.Owner == actor
	}
	if pipe.Issue != 0 {
		// The same-product lane never creates a native issue: a pipeline
		// that left it no longer speaks for the accepted direction.
		view.State = "superseded"
		view.CanReview = false
	}
	return view, nil
}

func revisionAdoptsCommentProductIdea(revision factory.IntentRevision, idea string) bool {
	decided := false
	for _, item := range revision.Ideas {
		if item.ID != idea || item.Decision != "accept" && item.Decision != "edit" {
			continue
		}
		decided = true
		break
	}
	if !decided {
		return false
	}
	if len(revision.EffectiveOutcomes) == 0 {
		return true
	}
	for _, outcome := range revision.EffectiveOutcomes {
		if outcome == idea {
			return true
		}
	}
	return false
}

func effectiveCommentProductIdea(head factory.IntentRevision, detail FeedCommentDetail) bool {
	return revisionAdoptsCommentProductIdea(head, factory.CommentDecisionID("product-idea", detail.Record.ID, detail.Record.Revision))
}

// ReviewCommentProductChange creates one same-intent successor draft and its
// source-comment receipt atomically. No planner, issue or acceptance is invoked.
func (s *Store) ReviewCommentProductChange(ctx context.Context, id string, revision, owner, at int64) (factory.IntentRevision, error) {
	var empty factory.IntentRevision
	if revision < 0 || owner <= 0 || at <= 0 {
		return empty, ErrCommandConflict
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return empty, err
	}
	defer func() { _ = tx.Rollback() }()
	detail, binding, err := lockCommentProductReview(ctx, tx, id, revision)
	if err != nil {
		return empty, err
	}
	p := detail.Source.Product
	var raw []byte
	if err = tx.QueryRowContext(ctx, `SELECT r.data FROM factory_intent_heads h JOIN factory_intent_revisions r ON r.id=h.revision WHERE h.id=$1 AND h.owner_id=$2 FOR UPDATE OF h`, p.Intent, owner).Scan(&raw); err != nil {
		return empty, err
	}
	var head factory.IntentRevision
	if err = json.Unmarshal(raw, &head); err != nil {
		return empty, err
	}
	if head.Repository != detail.Record.Repository {
		return empty, ErrNotFound
	}
	if binding.Draft != "" {
		if binding.Revision != revision {
			return empty, ErrStaleRevision
		}
		if err = tx.QueryRowContext(ctx, `SELECT data FROM factory_intent_revisions WHERE id=$1 AND intent=$2`, binding.Draft, p.Intent).Scan(&raw); err != nil {
			return empty, err
		}
		if err = json.Unmarshal(raw, &head); err != nil {
			return empty, err
		}
		return head, tx.Commit()
	}
	if head.ID != p.AcceptedRevision || head.State != "accepted" {
		return empty, ErrStaleRevision
	}
	draft, err := commentProductDraft(head, detail, at)
	if err != nil {
		return empty, err
	}
	if err = retainCommentGeneratedTasks(ctx, tx, head, &draft); err != nil {
		return empty, err
	}
	raw, err = json.Marshal(draft)
	if err != nil {
		return empty, err
	}
	if err = saveIntentRevisionTx(ctx, tx, draft, raw); err != nil {
		return empty, err
	}
	binding = commentProductBinding{Revision: revision, Draft: draft.ID}
	raw, err = json.Marshal(binding)
	if err != nil {
		return empty, err
	}
	if _, err = tx.ExecContext(ctx, `UPDATE factory_feed_comments SET data=data || jsonb_build_object('product_change',$2::jsonb) WHERE id=$1`, id, string(raw)); err != nil {
		return empty, err
	}
	if _, err = tx.ExecContext(ctx, `UPDATE factory_feed_comment_requests SET data=data || jsonb_build_object('detail','Product change opened for review. Planning and acceptance remain explicit.') WHERE comment=$1`, id); err != nil {
		return empty, err
	}
	return draft, tx.Commit()
}

// The accepted direction may contain a review baseline whose task keys differ
// from the graph actually generated. Carry the exact completed graph into the
// new planning draft; native issue identities and authority remain with the
// generation/adoption owners and are not copied into the draft.
func retainCommentGeneratedTasks(ctx context.Context, tx *sql.Tx, head factory.IntentRevision, draft *factory.IntentRevision) error {
	var raw []byte
	err := tx.QueryRowContext(ctx, `SELECT data FROM factory_intent_generations WHERE revision=$1 FOR UPDATE`, head.ID).Scan(&raw)
	if errors.Is(err, sql.ErrNoRows) {
		return nil // An accepted direction can be discussed before generation.
	}
	if err != nil {
		return err
	}
	var generation factory.IntentGeneration
	if err = json.Unmarshal(raw, &generation); err != nil {
		return err
	}
	if generation.State != "completed" || generation.RevisionID != head.ID || generation.IntentID != head.Intent || generation.IntentDigest != head.Digest() {
		return errors.New("product change awaits the exact accepted generation's completion")
	}
	if len(generation.SubmittedTasks) == 0 {
		return errors.New("completed generation has no submitted tasks")
	}
	submitted := head
	submitted.Tasks = generation.SubmittedTasks
	if err = submitted.Validate(); err != nil {
		return err
	}
	draft.Tasks = append([]factory.IntentTask(nil), generation.SubmittedTasks...)
	return draft.Validate()
}

func lockCommentProductReview(ctx context.Context, tx *sql.Tx, id string, revision int64) (FeedCommentDetail, commentProductBinding, error) {
	var d FeedCommentDetail
	var binding commentProductBinding
	var raw []byte
	err := tx.QueryRowContext(ctx, `SELECT repository,revision,author,body,data FROM factory_feed_comments WHERE id=$1 FOR UPDATE`, id).Scan(&d.Record.Repository, &d.Record.Revision, &d.Record.Author, &d.Body, &raw)
	if err != nil {
		return d, binding, err
	}
	d.Record.ID = id
	var data struct {
		Source        *factory.CommentSource `json:"source"`
		Deleted       bool                   `json:"deleted"`
		Revoked       bool                   `json:"revoked"`
		ProductChange commentProductBinding  `json:"product_change"`
	}
	if err = json.Unmarshal(raw, &data); err != nil {
		return d, binding, err
	}
	d.Source = data.Source
	binding = data.ProductChange
	if d.Record.Revision != revision || data.Deleted || data.Revoked || d.Source == nil || d.Source.Product == nil {
		return d, binding, ErrStaleRevision
	}
	var pipe factory.CommentPipeline
	err = tx.QueryRowContext(ctx, `SELECT revision,judgment,route,draft,effect,issue FROM factory_feed_comment_requests WHERE comment=$1 FOR UPDATE`, id).Scan(&pipe.Revision, &pipe.Judgment, &pipe.Route, &pipe.Draft, &pipe.Effect, &pipe.Issue)
	if err != nil {
		return d, binding, err
	}
	if pipe.Revision != revision || pipe.Judgment != factory.CommentJudgmentApplied || pipe.Route != factory.CommentRouteNewIssue || pipe.Draft != factory.CommentDraftPending || pipe.Effect != factory.CommentEffectPending || pipe.Issue != 0 {
		return d, binding, ErrStaleRevision
	}
	return d, binding, nil
}

func commentProductDraft(head factory.IntentRevision, detail FeedCommentDetail, at int64) (factory.IntentRevision, error) {
	if len(head.Ideas) >= 50 {
		return factory.IntentRevision{}, errors.New("product idea limit reached")
	}
	idea := factory.ProductIdea{ID: factory.CommentDecisionID("product-idea", detail.Record.ID, detail.Record.Revision), Description: detail.Body, Consequence: "The requested product change is reviewed, implemented and verified while preserving unaffected outcomes.", Decision: "accept", Source: "requested", Provenance: "human", Actor: head.Owner, Evidence: fmt.Sprintf("Feed comment %s revision %d by native actor %s; admitted for draft planning by the product owner.", detail.Record.ID, detail.Record.Revision, detail.Record.Author)}
	draft := head
	draft.Predecessor, draft.ID = head.ID, factory.CommentDecisionID("product-review", detail.Record.ID, detail.Record.Revision)
	draft.State, draft.CreatedUnix = "draft", at
	draft.Ideas = append(append([]factory.ProductIdea(nil), head.Ideas...), idea)
	draft.EffectiveOutcomes = append([]string(nil), head.EffectiveOutcomes...)
	if len(draft.EffectiveOutcomes) == 0 {
		for _, prior := range head.Ideas {
			if prior.Decision == "accept" || prior.Decision == "edit" {
				draft.EffectiveOutcomes = append(draft.EffectiveOutcomes, prior.ID)
			}
		}
	}
	draft.EffectiveOutcomes = append(draft.EffectiveOutcomes, idea.ID)
	return draft, draft.Validate()
}

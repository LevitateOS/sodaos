package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/project"
)

// CommentProductChange is a read projection of the explicit review handoff.
// Accepted describes product scope only, not generation, verification or release.
// The adopted lineage below never infers work from closed issues or worker
// exits: matched tasks reuse the submitted graph with its native bindings,
// completion is canonical by issue and acceptance, and verified names only
// the controller-owned F10 fact for the adopted revision and scope digest.
// Evidence and native issue identity stay separate fields; unknown or
// missing work reads as unavailable, never as success.
type CommentProductTaskEvidence struct {
	TaskKey    string `json:"task_key"`
	Issue      int64  `json:"issue"`
	Acceptance string `json:"acceptance"`
	Completion string `json:"completion,omitempty"`
	// Pending marks a bound matching task with no canonical completion
	// row yet: the native binding is exact, but no delivery or
	// completion is inferred. Pending never verifies the product.
	Pending bool `json:"pending,omitempty"`
}

const (
	// CommentProductEvidenceRecorded names resolved adopted lineage: the
	// deterministic comment idea was accepted in a descendant revision and
	// the canonical delivered graph behind it read back.
	CommentProductEvidenceRecorded = "recorded"
	// CommentProductEvidenceUnavailable names unknown work evidence: no
	// adopted descendant yet, or the canonical graph did not read back.
	// It mirrors the linked-work vocabulary and invents no new status.
	CommentProductEvidenceUnavailable = "unavailable"
)

// CommentProductStateUnavailable names an explicit unknown projection: the
// comment sits on the same-product lane but the read behind product_change
// failed. It never asserts absence and never reads as success; callers must
// render it as unavailable detail, not as detached generic progress.
const CommentProductStateUnavailable = "unavailable"

// UnavailableCommentProductChange projects an explicit unknown for a
// same-product comment whose read failed. Intent and accepted revision come
// from the accepted source behind the comment; adoption, verification and
// task evidence stay unknown, never success.
func UnavailableCommentProductChange(intent, acceptedRevision string) *CommentProductChange {
	return &CommentProductChange{
		Intent:           intent,
		AcceptedRevision: acceptedRevision,
		State:            CommentProductStateUnavailable,
		EvidenceStatus:   CommentProductEvidenceUnavailable,
	}
}

type CommentProductChange struct {
	Intent           string                       `json:"intent"`
	AcceptedRevision string                       `json:"accepted_revision"`
	DraftRevision    string                       `json:"draft_revision,omitempty"`
	State            string                       `json:"state"`
	CanReview        bool                         `json:"can_review"`
	AdoptedRevision  string                       `json:"adopted_revision,omitempty"`
	ScopeDigest      string                       `json:"scope_digest,omitempty"`
	Verified         bool                         `json:"verified"`
	EvidenceStatus   string                       `json:"evidence_status"`
	Evidence         []CommentProductTaskEvidence `json:"evidence,omitempty"`
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
	view := &CommentProductChange{Intent: p.Intent, AcceptedRevision: p.AcceptedRevision, State: "superseded", EvidenceStatus: CommentProductEvidenceUnavailable}
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
		// The accepted direction stays owner-openable: only the head
		// owner may open the private build request behind it. A
		// collaborator keeps the evidence link only. This bit authorizes
		// navigation, never a review write or inferred delivery.
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
	lineage, err := s.commentProductLineage(ctx, p.Intent, head, detail)
	if err != nil {
		return nil, err
	}
	view.AdoptedRevision = lineage.adopted
	view.ScopeDigest = lineage.digest
	view.Verified = lineage.verified
	view.EvidenceStatus = lineage.status
	view.Evidence = lineage.tasks
	return view, nil
}

// commentProductLineage resolves the deterministic comment idea through its
// descendant acceptance, then reuses the submitted graph with its native
// bindings for the matching task keys. Canonical completion comes from the
// delivered evidence join by issue and acceptance; a bound matching task
// with no completion row stays visible as pending without inferring
// delivery. Verification names only the controller-owned F10 fact for the
// adopted revision and scope digest. Adopted lineage survives a later
// superseding head as history. Unknown or unreadable work stays
// unavailable, never inferred from closed issues or worker exits.
func (s *Store) commentProductLineage(ctx context.Context, intent string, head factory.IntentRevision, detail FeedCommentDetail) (commentProductAdoption, error) {
	var out commentProductAdoption
	out.status = CommentProductEvidenceUnavailable
	idea := factory.CommentDecisionID("product-idea", detail.Record.ID, detail.Record.Revision)
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return out, err
	}
	defer func() { _ = tx.Rollback() }()
	adopted, err := adoptedCommentProductRevisionTx(ctx, tx, intent, head, idea)
	if err != nil {
		return out, err
	}
	if adopted == nil {
		return out, tx.Commit()
	}
	out.adopted, out.digest = adopted.ID, adopted.Digest()
	verified, err := verifiedAppRevisionTx(ctx, tx, *adopted)
	if err != nil {
		return out, err
	}
	out.verified = verified
	evidence, err := deliveredCommentProductTaskEvidenceTx(ctx, tx, *adopted, idea)
	if err != nil {
		return out, tx.Commit()
	}
	out.tasks = append(out.tasks, evidence...)
	out.status = CommentProductEvidenceRecorded
	return out, tx.Commit()
}

// deliveredCommentProductTaskEvidenceTx resolves the exact native
// deliveredCommentProductTaskEvidenceTx resolves the exact native
// bindings and optional canonical completions for only the adopted tasks
// carrying the deterministic comment idea. An unrelated pending task in the
// same scope never hides completed comment work; a matching task with a
// valid native binding but no completion row stays visible as pending,
// alongside completed matching tasks. A missing or invalid binding for the
// comment idea itself still reads as unavailable. Completion is canonical
// by issue and acceptance per matching task, never inferred from
// scope-level generation, closed issues or worker exits.
func deliveredCommentProductTaskEvidenceTx(ctx context.Context, tx *sql.Tx, revision factory.IntentRevision, idea string) ([]CommentProductTaskEvidence, error) {
	var raw []byte
	tasks := revision.Tasks
	err := tx.QueryRowContext(ctx, `SELECT data FROM factory_intent_generations WHERE revision=$1 FOR SHARE`, revision.ID).Scan(&raw)
	if err == nil {
		var generation factory.IntentGeneration
		if err := json.Unmarshal(raw, &generation); err != nil {
			return nil, err
		}
		if generation.State != "completed" || generation.RevisionID != revision.ID || generation.IntentID != revision.Intent || generation.IntentDigest != revision.Digest() {
			return nil, errors.New("accepted application generation is not complete and current")
		}
		if err := factory.AssessProposedTasks(revision, generation.SubmittedTasks, revision.MaxIssues); err != nil {
			return nil, err
		}
		// Native adoption binds the recorded assessed graph, whose task keys
		// can differ from the owner's accepted planning sketch.
		tasks = generation.SubmittedTasks
	} else if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	if len(tasks) < 1 || len(tasks) > 64 {
		return nil, errors.New("accepted application has no bounded delivered work")
	}
	var matched []factory.IntentTask
	for _, task := range tasks {
		for _, id := range task.IdeaIDs {
			if id == idea {
				matched = append(matched, task)
				break
			}
		}
	}
	evidence := make([]CommentProductTaskEvidence, 0, len(matched))
	for _, task := range matched {
		var binding factory.IntentTaskBinding
		if err := tx.QueryRowContext(ctx, `SELECT data FROM factory_intent_tasks WHERE revision=$1 AND task=$2`, revision.ID, task.Key).Scan(&raw); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(raw, &binding); err != nil {
			return nil, err
		}
		if binding.RevisionID != revision.ID || binding.TaskKey != task.Key || binding.IssueID <= 0 || binding.IssueIndex <= 0 || !project.ValidDecisionID(binding.AcceptanceID) {
			return nil, errors.New("accepted application task lacks its exact native binding")
		}
		var event factory.CompletionEvent
		var eventID, publicationID, occurrence string
		var generation int64
		if err := tx.QueryRowContext(ctx, `SELECT event,publication,occurrence,generation,data FROM factory_completions WHERE repository=$1 AND data->>'issue'=$2 AND data->>'acceptance_id'=$3 ORDER BY generation DESC LIMIT 1`, revision.Repository, fmt.Sprint(binding.IssueIndex), binding.AcceptanceID).Scan(&eventID, &publicationID, &occurrence, &generation, &raw); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				evidence = append(evidence, CommentProductTaskEvidence{TaskKey: task.Key, Issue: binding.IssueIndex, Acceptance: binding.AcceptanceID, Pending: true})
				continue
			}
			return nil, err
		}
		if err := json.Unmarshal(raw, &event); err != nil {
			return nil, err
		}
		if event.Validate() != nil || event.ID != eventID || event.PublicationID != publicationID || event.ClosingOccurrence != occurrence || event.Generation != generation || event.Repository != revision.Repository || event.Issue != binding.IssueIndex || event.AcceptanceID != binding.AcceptanceID {
			return nil, errors.New("accepted application task differs from its canonical completion")
		}
		evidence = append(evidence, CommentProductTaskEvidence{TaskKey: task.Key, Issue: binding.IssueIndex, Acceptance: binding.AcceptanceID, Completion: event.ID})
	}
	return evidence, nil
}

type commentProductAdoption struct {
	adopted  string
	digest   string
	verified bool
	status   string
	tasks    []CommentProductTaskEvidence
}

// adoptedCommentProductRevisionTx walks the immutable predecessor chain from
// the current head and returns the most recent accepted revision carrying the
// deterministic comment idea: the descendant acceptance. The walk is bounded;
// a broken or overlong chain reads as no adoption, never as an error.
func adoptedCommentProductRevisionTx(ctx context.Context, tx *sql.Tx, intent string, head factory.IntentRevision, idea string) (*factory.IntentRevision, error) {
	current := head
	for range 64 {
		if current.State == "accepted" && revisionAdoptsCommentProductIdea(current, idea) {
			out := current
			return &out, nil
		}
		if current.Predecessor == "" {
			return nil, nil
		}
		var raw []byte
		err := tx.QueryRowContext(ctx, `SELECT data FROM factory_intent_revisions WHERE id=$1 AND intent=$2`, current.Predecessor, intent).Scan(&raw)
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		if err != nil {
			return nil, err
		}
		var prev factory.IntentRevision
		if err := json.Unmarshal(raw, &prev); err != nil {
			return nil, err
		}
		current = prev
	}
	return nil, nil
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

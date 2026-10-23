package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
)

var ErrMCPCommunicationCap = errors.New("assignment communication bound reached; finish remains available")

// recordMCPCommunicationTx shares the immutable submission transaction, so a
// durable receipt can never claim a feed post that was not recorded. The
// canonical communication table already owns identity, thread and feed order.
func recordMCPCommunicationTx(ctx context.Context, tx *sql.Tx, capability factory.MCPCapability, submission factory.MCPSubmission, receipt *factory.MCPReceipt) error {
	if !factory.MCPIsCommunication(submission.Tool) {
		return nil
	}
	content, err := factory.DecodeMCPCommunication(submission.Tool, submission.Payload)
	if err != nil {
		return err
	}
	if !factory.MCPToolAdmitted(capability.Profile, submission.Tool) {
		return errors.New("communication outlet not admitted for this worker")
	}
	if err := factory.ValidateFeedComponent(content.ComponentID, capability.Profile, submission.Tool); err != nil {
		return err
	}
	var count int
	if err = tx.QueryRowContext(ctx, `SELECT count(*) FROM factory_mcp_submissions WHERE assignment=$1 AND tool = ANY($2)`, submission.AssignmentID, communicationToolNames()).Scan(&count); err != nil {
		return err
	}
	if count >= factory.MCPMaxCommunications {
		return ErrMCPCommunicationCap
	}
	if content.ReplyTo != "" {
		var parentKind string
		if err = tx.QueryRowContext(ctx, `SELECT COALESCE(data->'communication'->>'kind','') FROM factory_feed_comments WHERE id=$1 AND repository=$2 AND NOT COALESCE((data->>'deleted')::boolean,false) AND NOT COALESCE((data->>'revoked')::boolean,false)`, content.ReplyTo, capability.Repository).Scan(&parentKind); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return errors.New("reply target is not available in this project")
			}
			return err
		}
		if submission.Tool == factory.MCPToolAnswer && parentKind != "ask" {
			return errors.New("answer must refer to an ask post")
		}
	}
	if err := resolveCommunicationArtifactsTx(ctx, tx, capability.Repository, &content); err != nil {
		return err
	}
	return recordMCPFeedContentTx(ctx, tx, capability, submission, content, receipt)
}

func recordMCPFeedContentTx(ctx context.Context, tx *sql.Tx, capability factory.MCPCapability, submission factory.MCPSubmission, content factory.FeedCommunication, receipt *factory.MCPReceipt) error {
	var issue int64
	var err error
	if err = tx.QueryRowContext(ctx, `SELECT COALESCE((SELECT issue FROM factory_assignments WHERE id=$1 AND repository=$2),0)`, capability.AssignmentID, capability.Repository).Scan(&issue); err != nil {
		return err
	}
	postID := factory.MCPInputDigest([]byte(capability.AssignmentID + "\n" + submission.SubmissionID))[:32]
	if err = createCommunicationLibraryTx(ctx, tx, capability, submission, &content, postID, receipt); err != nil {
		return err
	}
	data, err := json.Marshal(map[string]any{
		"issue": issue, "reply_to": content.ReplyTo, "at": time.Now().Unix(), "author_name": recordPersona(capability.Profile),
		"actor": string(capability.Profile), "assignment": capability.AssignmentID, "run": capability.RunID,
		"project": capability.Project, "submission": submission.SubmissionID, "communication": content,
	})
	if err != nil {
		return err
	}
	// The full typed content lives in data; body is the concise textual equivalent
	// used by existing reply/JEV source readers, within their canonical body bound.
	body := content.Title + "\n\n" + content.Summary
	if content.Question != "" && !strings.Contains(body, content.Question) {
		body += "\n\n" + content.Question
	}
	if len([]rune(body)) > factory.CommentBodyLimit {
		body = string([]rune(body)[:factory.CommentBodyLimit])
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO factory_feed_comments(id,repository,revision,author,body,idempotency,data) VALUES($1,$2,0,$3,$4,$5,$6)`, postID, capability.Repository, "worker:"+capability.AssignmentID, body, submission.SubmissionID, string(data))
	if err != nil {
		return err
	}
	receipt.FeedPostID = postID
	return nil
}

func communicationToolNames() []string {
	names := make([]string, 0, len(factory.MCPCommunicationTools))
	for _, tool := range factory.MCPCommunicationTools {
		names = append(names, string(tool))
	}
	return names
}

func applyFeedCommunicationDetail(detail *FeedCommentDetail, raw []byte) error {
	var data struct {
		Actor         string                     `json:"actor"`
		Status        string                     `json:"status"`
		Assignment    string                     `json:"assignment"`
		Communication *factory.FeedCommunication `json:"communication"`
		Source        *factory.CommentSource     `json:"source"`
	}
	if err := json.Unmarshal(raw, &data); err != nil {
		return err
	}
	if data.Source != nil {
		detail.Source = data.Source
		if detail.Record.ReplyTo == "" && data.Source.Kind == factory.CommentSourcePost {
			detail.Record.ReplyTo = data.Source.ID
		}
	}
	if data.Communication != nil {
		detail.Actor = data.Actor
		detail.Assignment = data.Assignment
		detail.Status = data.Communication.Kind
		if data.Actor == "factory" && data.Status != "" {
			detail.Status = data.Status
		}
		detail.Payload = &factory.FeedPayload{Communication: data.Communication}
	}
	return nil
}

// feedSubmissionComponents keeps the worker's chosen presentation beside the
// canonical result, without replacing its observed lifecycle status or adding
// a second finish post. Only the role's admitted terminal tool can select it.
func (s *Store) feedSubmissionComponents(ctx context.Context, ids []string, out map[string]FeedRecordDetail) error {
	rows, err := s.db.QueryContext(ctx, `SELECT assignment,tool,payload FROM factory_mcp_submissions WHERE assignment = ANY($1)`, ids)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var assignment string
		var tool factory.MCPTool
		var raw []byte
		if err := rows.Scan(&assignment, &tool, &raw); err != nil {
			return err
		}
		if !factory.MCPIsTerminal(tool) {
			continue
		}
		var report struct {
			ComponentID string `json:"component_id"`
		}
		if err := json.Unmarshal(raw, &report); err != nil {
			return err
		}
		if report.ComponentID == "" {
			continue
		}
		detail, ok := out[assignment]
		if !ok {
			continue
		}
		if err := factory.ValidateFeedComponent(report.ComponentID, factory.MCPProfile(detail.Actor), tool); err != nil {
			continue
		}
		if detail.Payload == nil {
			detail.Payload = &factory.FeedPayload{}
		}
		detail.Payload.ComponentID = report.ComponentID
		out[assignment] = detail
	}
	return rows.Err()
}

// feedQuestionStates derives response arrival without mutating the original
// question or claiming that the answer was adopted by the worker.
func (s *Store) feedQuestionStates(ctx context.Context, details map[string]FeedCommentDetail) error {
	ids := []string{}
	for id, detail := range details {
		if detail.Payload != nil && detail.Payload.Communication != nil && detail.Payload.Communication.Kind == "ask" {
			detail.Payload.Communication.QuestionState = "open"
			ids = append(ids, id)
		}
	}
	if len(ids) == 0 {
		return nil
	}
	// The incorporated flag rides the same derivation: an answer the
	// worker adopted moves its question past answered without
	// claiming anything about unrecorded replies.
	rows, err := s.db.QueryContext(ctx, `SELECT p.id,bool_or(c.data->'communication'->>'kind'='answer'),
 bool_or(c.data->'communication'->>'kind'='answer' AND r.resolution='incorporated')
 FROM factory_feed_comments p JOIN factory_feed_comments c ON c.repository=p.repository
 AND (c.data->>'reply_to'=p.id OR (c.data->'source'->>'kind'='post' AND c.data->'source'->>'id'=p.id))
 LEFT JOIN factory_post_resolutions r ON r.post=c.id
 WHERE p.id=ANY($1) AND NOT COALESCE((c.data->>'deleted')::boolean,false) AND NOT COALESCE((c.data->>'revoked')::boolean,false)
 GROUP BY p.id`, ids)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		var answered, incorporated sql.NullBool
		if err := rows.Scan(&id, &answered, &incorporated); err != nil {
			return err
		}
		detail := details[id]
		state := "replied"
		if answered.Valid && answered.Bool {
			state = "answered"
		}
		if incorporated.Valid && incorporated.Bool {
			state = "incorporated"
		}
		detail.Payload.Communication.QuestionState = state
	}
	return rows.Err()
}

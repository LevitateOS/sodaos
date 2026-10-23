package store

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/project"
)

// MCPTaskOutputSubmissions reads the immutable authenticated parent proposals.
// Journal sequence, rather than submission identity, orders their checkpoints.
func (s *Store) MCPTaskOutputSubmissions(ctx context.Context, assignment string) ([]factory.MCPSubmission, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT submission,capability,input_digest,payload_hash,payload FROM factory_mcp_submissions WHERE assignment=$1 AND tool=$2 ORDER BY submission LIMIT 257`, assignment, string(factory.MCPToolSubmitTaskOutput))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []factory.MCPSubmission
	for rows.Next() {
		submission := factory.MCPSubmission{AssignmentID: assignment, Tool: factory.MCPToolSubmitTaskOutput}
		if err := rows.Scan(&submission.SubmissionID, &submission.CapabilityID, &submission.ExpectedInputDigest, &submission.PayloadHash, &submission.Payload); err != nil {
			return nil, err
		}
		if err := submission.Validate(); err != nil {
			return nil, err
		}
		result = append(result, submission)
	}
	if len(result) > 256 {
		return nil, errors.New("task output proposals exceed the admitted bound")
	}
	return result, rows.Err()
}

// SaveFactoryFinalCheckpoint retains only a confirmed native response. The
// request is immutable once reserved; terminal native evidence never changes.
func (s *Store) SaveFactoryFinalCheckpoint(ctx context.Context, run factory.Run, result project.FactoryTaskCheckpointState) error {
	if err := result.ValidateFor(result.Request, true); err != nil {
		return err
	}
	if result.Request.ID != run.ID || result.Request.Project != run.ProjectID || result.Request.ManifestDigest != project.ExecutionManifestDigest(run.ExecutionManifest) {
		return ErrStaleRevision
	}
	raw, err := json.Marshal(result)
	if err != nil {
		return err
	}
	manifest, err := json.Marshal(run.ExecutionManifest)
	if err != nil {
		return err
	}
	written, err := s.db.ExecContext(ctx, `UPDATE factory_runs SET data=jsonb_set(data,'{final_checkpoint}',$2::jsonb) WHERE id=$1 AND data->'execution_manifest'=$3::jsonb
AND (coalesce(data->'final_checkpoint','null'::jsonb)='null'::jsonb OR data->'final_checkpoint'=$2::jsonb
OR (data->'final_checkpoint'->'request'=$2::jsonb->'request' AND data->'final_checkpoint'->>'status' NOT IN ('committed','failed')))`, run.ID, string(raw), string(manifest))
	if err != nil {
		return err
	}
	n, err := written.RowsAffected()
	if err == nil && n != 1 {
		return ErrStaleRevision
	}
	return err
}

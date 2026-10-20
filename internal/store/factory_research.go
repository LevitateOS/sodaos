package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"

	"github.com/levitateos/sodaos/internal/factory"
)

// Progressive research records persist beside the assignment they
// belong to: one row per research assignment carrying the cumulative
// log. Reads never invent scope; only recorded batches accumulate.
func initializeResearchSchema(ctx context.Context, db *sql.DB) error {
	_, err := db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS factory_research_records(assignment TEXT PRIMARY KEY,repository BIGINT NOT NULL,data JSONB NOT NULL);`)
	return err
}

// recordMCPResearchTx persists one record_research batch into the
// assignment's cumulative log. Only the researcher profile records;
// the whole batch refuses when any cumulative bound or question
// index misses, so a partial append never lands.
func recordMCPResearchTx(ctx context.Context, tx *sql.Tx, cap factory.MCPCapability, submission factory.MCPSubmission, receipt *factory.MCPReceipt) error {
	if submission.Tool != factory.MCPToolRecordResearch {
		return nil
	}
	if string(cap.Profile) != string(factory.MCPProfileResearcher) || !factory.MCPToolAdmitted(cap.Profile, submission.Tool) {
		return errors.New("research recording requires the researcher profile")
	}
	record, err := factory.DecodeResearchRecord(submission.Tool, submission.Payload)
	if err != nil {
		return err
	}
	var raw []byte
	err = tx.QueryRowContext(ctx, `SELECT data FROM factory_research_records WHERE assignment=$1 FOR UPDATE`, submission.AssignmentID).Scan(&raw)
	log := factory.ResearchLog{}
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			return err
		}
	} else if err = json.Unmarshal(raw, &log); err != nil {
		return err
	}
	log, err = factory.AppendResearchRecord(log, record)
	if err != nil {
		return err
	}
	encoded, err := json.Marshal(log)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO factory_research_records(assignment,repository,data) VALUES($1,$2,$3) ON CONFLICT(assignment) DO UPDATE SET data=excluded.data`, submission.AssignmentID, cap.Repository, string(encoded))
	return err
}

// ResearchScope reads the researcher's declared scope: the admitted
// subject plus the cumulative log with per-question coverage. An
// assignment without records serves its subject with empty coverage;
// a non-research assignment refuses instead of inventing scope.
func (s *Store) ResearchScope(ctx context.Context, cap factory.MCPCapability) (factory.ResearchScopeView, error) {
	var out factory.ResearchScopeView
	if string(cap.Profile) != string(factory.MCPProfileResearcher) {
		return out, errors.New("research scope requires the researcher profile")
	}
	assigned, err := s.Assignment(ctx, cap.AssignmentID)
	if err != nil {
		return out, err
	}
	if assigned.Subject.Kind != factory.SubjectResearch || assigned.Subject.Research == nil {
		return out, errors.New("research scope needs a research assignment")
	}
	log := factory.ResearchLog{}
	var raw []byte
	if err = s.db.QueryRowContext(ctx, `SELECT data FROM factory_research_records WHERE assignment=$1`, cap.AssignmentID).Scan(&raw); err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			return out, err
		}
	} else if err = json.Unmarshal(raw, &log); err != nil {
		return out, err
	}
	return factory.CoverResearchLog(cap.AssignmentID, assigned.Subject.Research.Issue, assigned.Subject.Research.Acceptance, assigned.Subject.Research.Sources, log), nil
}

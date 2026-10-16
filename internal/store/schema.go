package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

const schemaFormatVersion = 52

// schemaStatements creates the current PostgreSQL schema in dependency
// order: tables, indexes, guard functions, then triggers. JSON payloads
// live in JSONB columns, which reject malformed JSON on write; small
// flags are BOOLEAN; insertion-ordered listings use explicit seq columns
// instead of the SQLite rowid.
var schemaStatements = []string{
	`CREATE TABLE factory_activity_settings(repository BIGINT PRIMARY KEY CHECK(repository>0), revision BIGINT NOT NULL CHECK(revision>0), actor_id BIGINT NOT NULL CHECK(actor_id>0), data JSONB NOT NULL)`,
	`CREATE TABLE schema_version(version INTEGER PRIMARY KEY)`,
	`INSERT INTO schema_version(version) VALUES(52)`,
	`CREATE TABLE applications(id TEXT PRIMARY KEY, administrator_id BIGINT NOT NULL CHECK(administrator_id>0), name TEXT NOT NULL, data_id TEXT NOT NULL UNIQUE, owner_team TEXT NOT NULL, origin TEXT NOT NULL UNIQUE, hostname TEXT NOT NULL UNIQUE, port INTEGER NOT NULL UNIQUE CHECK(port>=20000 AND port<30000), revision BIGINT NOT NULL CHECK(revision>0), observation JSONB CHECK(observation IS NULL OR octet_length(observation::text)<=2048))`,
	`CREATE TABLE application_revisions(application_id TEXT NOT NULL REFERENCES applications(id), revision BIGINT NOT NULL CHECK(revision>0), spec JSONB NOT NULL CHECK(octet_length(spec::text)<=4096), PRIMARY KEY(application_id,revision))`,
	`ALTER TABLE applications ADD FOREIGN KEY(id,revision) REFERENCES application_revisions(application_id,revision) DEFERRABLE INITIALLY DEFERRED`,
	`CREATE TABLE factory_departments(id TEXT PRIMARY KEY, owner_id BIGINT NOT NULL CHECK(owner_id>0), data JSONB NOT NULL)`,
	`CREATE TABLE factory_intent_heads(id TEXT PRIMARY KEY, owner_id BIGINT NOT NULL CHECK(owner_id>0), revision TEXT NOT NULL)`,
	`CREATE TABLE factory_intent_revisions(id TEXT PRIMARY KEY, intent TEXT NOT NULL REFERENCES factory_intent_heads(id), data JSONB NOT NULL)`,
	`CREATE TABLE factory_judgment_budget(id INTEGER PRIMARY KEY CHECK(id=1), spent INTEGER NOT NULL CHECK(spent>=0))`,
	`INSERT INTO factory_judgment_budget(id,spent) VALUES(1,0)`,
	`CREATE TABLE factory_priority_judgments(snapshot TEXT PRIMARY KEY, max_calls INTEGER NOT NULL CHECK(max_calls BETWEEN 0 AND 16), data JSONB)`,
	`CREATE TABLE factory_planning(repository BIGINT NOT NULL, issue BIGINT NOT NULL, snapshot TEXT NOT NULL, data JSONB NOT NULL, PRIMARY KEY(repository,issue))`,
	`CREATE TABLE factory_plan_revisions(attempt_root TEXT NOT NULL CHECK(attempt_root ~ '^[a-f0-9]{32}$'), plan_id TEXT NOT NULL CHECK(plan_id ~ '^[a-f0-9]{32}$'), generation BIGINT NOT NULL CHECK(generation>=0), digest TEXT NOT NULL CHECK(digest ~ '^[a-f0-9]{64}$'), data JSONB NOT NULL, PRIMARY KEY(attempt_root,plan_id))`,
	`CREATE TABLE factory_plan_heads(attempt_root TEXT PRIMARY KEY CHECK(attempt_root ~ '^[a-f0-9]{32}$'), plan_id TEXT NOT NULL CHECK(plan_id ~ '^[a-f0-9]{32}$'), generation BIGINT NOT NULL CHECK(generation>=0))`,
	`CREATE TABLE factory_plan_tasks(attempt_root TEXT NOT NULL CHECK(attempt_root ~ '^[a-f0-9]{32}$'), plan_id TEXT NOT NULL CHECK(plan_id ~ '^[a-f0-9]{32}$'), task_id TEXT NOT NULL CHECK(task_id ~ '^[a-z][a-z0-9_-]{0,31}$'), revision BIGINT NOT NULL CHECK(revision>=0), state TEXT NOT NULL, data JSONB NOT NULL, PRIMARY KEY(attempt_root,plan_id,task_id))`,
	`CREATE TABLE factory_intent_generations(revision TEXT PRIMARY KEY REFERENCES factory_intent_revisions(id), data JSONB NOT NULL)`,
	`CREATE TABLE factory_intent_tasks(revision TEXT NOT NULL REFERENCES factory_intent_revisions(id), task TEXT NOT NULL, data JSONB NOT NULL, PRIMARY KEY(revision,task))`,
	`CREATE TABLE factory_intent_planning_calls(intent TEXT NOT NULL REFERENCES factory_intent_heads(id), snapshot TEXT PRIMARY KEY REFERENCES factory_priority_judgments(snapshot))`,
	`CREATE TABLE factory_intent_allowance_heads(intent TEXT PRIMARY KEY REFERENCES factory_intent_heads(id), owner_id BIGINT NOT NULL CHECK(owner_id>0), revision TEXT NOT NULL)`,
	`CREATE TABLE factory_intent_allowances(id TEXT PRIMARY KEY, intent TEXT NOT NULL REFERENCES factory_intent_allowance_heads(intent), data JSONB NOT NULL)`,
	`CREATE TABLE factory_intent_reviews(result_revision TEXT PRIMARY KEY REFERENCES factory_intent_revisions(id), data JSONB NOT NULL)`,
	`CREATE TABLE users(id INTEGER PRIMARY KEY CHECK(id>0), login TEXT NOT NULL, name TEXT NOT NULL DEFAULT '')`,
	`CREATE TABLE keys(id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY, user_id INTEGER NOT NULL REFERENCES users(id), public TEXT NOT NULL, fingerprint TEXT NOT NULL, UNIQUE(user_id,fingerprint))`,
	`CREATE TABLE projects(id TEXT PRIMARY KEY, name TEXT NOT NULL, repository_id INTEGER NOT NULL UNIQUE, owner_id INTEGER NOT NULL REFERENCES users(id), repository TEXT NOT NULL, ip TEXT NOT NULL DEFAULT '', ready BOOLEAN NOT NULL DEFAULT FALSE, creation_profile JSONB CHECK(creation_profile IS NULL OR octet_length(creation_profile::text)<=1024))`,
	`CREATE TABLE memberships(project_id TEXT NOT NULL REFERENCES projects(id), user_id INTEGER NOT NULL REFERENCES users(id), login TEXT NOT NULL, PRIMARY KEY(project_id,user_id), UNIQUE(project_id,login))`,
	`CREATE TABLE grant_key_check(id INTEGER PRIMARY KEY CHECK(id=1), ciphertext BYTEA NOT NULL)`,
	`CREATE TABLE factory_runs(
seq BIGINT GENERATED ALWAYS AS IDENTITY,
id TEXT PRIMARY KEY,
active BOOLEAN NOT NULL, settled BOOLEAN NOT NULL,
data JSONB NOT NULL)`,
	`CREATE INDEX factory_unsettled_runs ON factory_runs(id) WHERE active OR NOT settled OR COALESCE(data->>'reconciled','false') <> 'true'`,
	`CREATE TABLE factory_commands(
id TEXT PRIMARY KEY, type TEXT NOT NULL, target TEXT NOT NULL DEFAULT '',
principal TEXT NOT NULL, digest TEXT NOT NULL, payload TEXT NOT NULL DEFAULT '' CHECK(char_length(payload)<=8192),
outcome TEXT NOT NULL DEFAULT '' CHECK(char_length(outcome)<=65536),
created TEXT NOT NULL, finished TEXT NOT NULL DEFAULT '')`,
	`CREATE INDEX factory_finished_commands_target ON factory_commands(target,type,created,id) WHERE finished<>''`,
	`CREATE TABLE factory_policies(repository INTEGER PRIMARY KEY CHECK(repository>0), revision INTEGER NOT NULL, data JSONB NOT NULL)`,
	`CREATE TABLE factory_capacity(id INTEGER PRIMARY KEY CHECK(id=1), revision INTEGER NOT NULL, data JSONB NOT NULL)`,
	`CREATE TABLE factory_operator_grants(repository INTEGER PRIMARY KEY CHECK(repository>0), revision INTEGER NOT NULL, data JSONB NOT NULL)`,
	`CREATE TABLE factory_sponsorships(repository INTEGER NOT NULL CHECK(repository>0), connection TEXT NOT NULL, revision INTEGER NOT NULL, data JSONB NOT NULL, PRIMARY KEY(repository,connection))`,
	`CREATE TABLE factory_connection_usage_budgets(connection TEXT PRIMARY KEY, revision INTEGER NOT NULL, data JSONB NOT NULL)`,
	`CREATE TABLE factory_dispatch(repository INTEGER PRIMARY KEY CHECK(repository>0), revision INTEGER NOT NULL, open BOOLEAN NOT NULL, data JSONB NOT NULL)`,
	`CREATE TABLE factory_dispatch_regs(seq BIGINT GENERATED ALWAYS AS IDENTITY, id TEXT PRIMARY KEY, repository INTEGER NOT NULL CHECK(repository>0), revision INTEGER NOT NULL, data JSONB NOT NULL)`,
	`CREATE INDEX factory_dispatch_regs_repository ON factory_dispatch_regs(repository)`,
	`CREATE TABLE factory_takeovers(run TEXT NOT NULL, member TEXT NOT NULL, project TEXT NOT NULL REFERENCES projects(id), data JSONB NOT NULL, PRIMARY KEY(run,member))`,
	`CREATE TABLE factory_lifecycle_actions(id TEXT PRIMARY KEY, event TEXT NOT NULL, action TEXT NOT NULL, repository BIGINT NOT NULL CHECK(repository>0), issue BIGINT NOT NULL CHECK(issue>0), attempts INTEGER NOT NULL DEFAULT 0 CHECK(attempts>=0), revoked BOOLEAN NOT NULL DEFAULT FALSE, data JSONB NOT NULL)`,
	`CREATE INDEX factory_lifecycle_actions_target ON factory_lifecycle_actions(repository,issue,event)`,
	`CREATE TABLE factory_lifecycle_capabilities(event TEXT PRIMARY KEY, revision TEXT NOT NULL, installation TEXT NOT NULL)`,
	`CREATE TABLE factory_lifecycle_configs(event TEXT PRIMARY KEY, revision BIGINT NOT NULL CHECK(revision>=0), enabled BOOLEAN NOT NULL, data JSONB NOT NULL)`,
	`CREATE TABLE factory_run_views(seq BIGINT GENERATED ALWAYS AS IDENTITY, run TEXT PRIMARY KEY REFERENCES factory_runs(id), repository INTEGER NOT NULL CHECK(repository>0), issue INTEGER NOT NULL DEFAULT 0 CHECK(issue>=0), attempt TEXT NOT NULL DEFAULT '' CHECK(char_length(attempt)<=64))`,
	`CREATE TABLE factory_assignments(seq BIGINT GENERATED ALWAYS AS IDENTITY, id TEXT PRIMARY KEY, repository INTEGER NOT NULL CHECK(repository>0), issue INTEGER NOT NULL CHECK(issue>0), run TEXT NOT NULL, stage TEXT NOT NULL, revision INTEGER NOT NULL, data JSONB NOT NULL)`,
	`CREATE TABLE factory_attempt_allowances(repository INTEGER NOT NULL CHECK(repository>0), issue INTEGER NOT NULL CHECK(issue>0), root_assignment TEXT PRIMARY KEY REFERENCES factory_assignments(id), revision BIGINT NOT NULL CHECK(revision>=0), data JSONB NOT NULL)`,
	`CREATE INDEX factory_attempt_allowance_issues ON factory_attempt_allowances(repository,issue)`,
	`CREATE INDEX factory_active_attempts ON factory_attempt_allowances(repository) WHERE data->>'active'='true'`,
	`CREATE UNIQUE INDEX factory_unfinished_assignment ON factory_assignments(repository,issue) WHERE stage='assigned'`,
	`CREATE INDEX factory_assignments_issue_history ON factory_assignments(repository,issue,seq DESC)`,
	`CREATE INDEX factory_finished_children ON factory_assignments(seq) WHERE stage='finished' AND data->>'publication_assignment'<>id`,
	`CREATE TABLE factory_reservations(seq BIGINT GENERATED ALWAYS AS IDENTITY, assignment TEXT PRIMARY KEY, repository INTEGER NOT NULL CHECK(repository>0), connection TEXT NOT NULL, state TEXT NOT NULL, data JSONB NOT NULL)`,
	`CREATE INDEX factory_reservations_connection_state ON factory_reservations(connection,state)`,
	`CREATE TABLE factory_usage(run TEXT PRIMARY KEY, repository INTEGER NOT NULL CHECK(repository>0), connection TEXT NOT NULL, minutes INTEGER NOT NULL CHECK(minutes>=0), started_at TIMESTAMPTZ NOT NULL, ended_at TIMESTAMPTZ NOT NULL CHECK(ended_at>=started_at), data JSONB NOT NULL)`,
	`CREATE INDEX factory_usage_connection_ended ON factory_usage(connection,ended_at)`,
	`CREATE TABLE issue_acceptance_decisions(id TEXT PRIMARY KEY, repository INTEGER NOT NULL CHECK(repository>0), issue INTEGER NOT NULL CHECK(issue>0), predecessor TEXT NOT NULL DEFAULT '', data JSONB NOT NULL)`,
	`CREATE TABLE issue_acceptance_heads(repository INTEGER NOT NULL CHECK(repository>0), issue INTEGER NOT NULL CHECK(issue>0), decision TEXT NOT NULL, PRIMARY KEY(repository,issue))`,
	`CREATE TABLE issue_acceptance_withdrawals(repository INTEGER NOT NULL CHECK(repository>0), issue INTEGER NOT NULL CHECK(issue>0), decision TEXT NOT NULL, withdrawer INTEGER NOT NULL CHECK(withdrawer>0), PRIMARY KEY(repository,issue,decision))`,
	`CREATE TABLE issue_controls(repository INTEGER NOT NULL CHECK(repository>0), issue INTEGER NOT NULL CHECK(issue>0), revision INTEGER NOT NULL CHECK(revision>0), data JSONB NOT NULL, PRIMARY KEY(repository,issue))`,
	`CREATE TABLE intake_deliveries(delivery TEXT PRIMARY KEY, repository INTEGER NOT NULL CHECK(repository>0), issue INTEGER NOT NULL CHECK(issue>0), kind TEXT NOT NULL, received_at INTEGER NOT NULL)`,
	`CREATE TABLE factory_publications(seq BIGINT GENERATED ALWAYS AS IDENTITY, assignment TEXT PRIMARY KEY, repository INTEGER NOT NULL CHECK(repository>0), issue INTEGER NOT NULL CHECK(issue>0), run TEXT NOT NULL, stage TEXT NOT NULL, revision INTEGER NOT NULL, data JSONB NOT NULL)`,
	`CREATE TABLE factory_check_assessments(repository INTEGER NOT NULL CHECK(repository>0), pr INTEGER NOT NULL CHECK(pr>0), revision INTEGER NOT NULL, data JSONB NOT NULL, PRIMARY KEY(repository,pr))`,
	`CREATE TABLE factory_merges(seq BIGINT GENERATED ALWAYS AS IDENTITY, publication TEXT PRIMARY KEY, repository INTEGER NOT NULL CHECK(repository>0), issue INTEGER NOT NULL CHECK(issue>0), pr INTEGER NOT NULL CHECK(pr>0), stage TEXT NOT NULL, revision INTEGER NOT NULL, data JSONB NOT NULL)`,
	`CREATE TABLE factory_readiness_budget(id INTEGER PRIMARY KEY CHECK(id=1), generation BIGINT NOT NULL CHECK(generation>=1), next_turn BIGINT NOT NULL CHECK(next_turn>=1))`,
	`INSERT INTO factory_readiness_budget(id,generation,next_turn) VALUES(1,1,1)`,
	`CREATE TABLE factory_readiness_sources(
id TEXT PRIMARY KEY CHECK(octet_length(id) BETWEEN 1 AND 160),
delivery TEXT NOT NULL DEFAULT '' CHECK(octet_length(delivery)<=128),
repository BIGINT NOT NULL CHECK(repository>0), issue BIGINT NOT NULL CHECK(issue>0),
creation_verified BOOLEAN NOT NULL DEFAULT FALSE,
creator_id BIGINT NOT NULL DEFAULT 0 CHECK(creator_id>=0), creator_write BOOLEAN NOT NULL DEFAULT FALSE,
native_issue_id BIGINT NOT NULL DEFAULT 0 CHECK(native_issue_id>=0), created_unix BIGINT NOT NULL DEFAULT 0 CHECK(created_unix>=0),
title_digest TEXT NOT NULL DEFAULT '' CHECK(octet_length(title_digest) IN (0,64)),
content_digest TEXT NOT NULL DEFAULT '' CHECK(octet_length(content_digest) IN (0,64)),
generation BIGINT NOT NULL DEFAULT 0 CHECK(generation>=0), turn BIGINT NOT NULL CHECK(turn>0),
retry_at TIMESTAMPTZ NOT NULL DEFAULT 'epoch', attempts INTEGER NOT NULL DEFAULT 0 CHECK(attempts BETWEEN 0 AND 6),
CHECK(NOT creation_verified OR (creator_id>0 AND native_issue_id>0 AND created_unix>0 AND octet_length(title_digest)=64 AND octet_length(content_digest)=64)))`,
	`CREATE TABLE factory_creation_cursors(repository BIGINT PRIMARY KEY CHECK(repository>0), task_id BIGINT NOT NULL CHECK(task_id>=0))`,
	`CREATE TABLE factory_readiness_nodes(
source TEXT NOT NULL REFERENCES factory_readiness_sources(id) ON DELETE CASCADE,
repository BIGINT NOT NULL CHECK(repository>0), issue BIGINT NOT NULL CHECK(issue>0),
state TEXT NOT NULL DEFAULT 'queued' CHECK(state IN ('queued','scanning','done')),
cursor_repository BIGINT NOT NULL DEFAULT 0 CHECK(cursor_repository>=0), cursor_issue BIGINT NOT NULL DEFAULT 0 CHECK(cursor_issue>=0),
PRIMARY KEY(source,repository,issue), CHECK((cursor_repository=0)=(cursor_issue=0)))`,
	`CREATE TABLE factory_priority_scans(
scan TEXT PRIMARY KEY, native_revision BIGINT NOT NULL CHECK(native_revision>=0),
local_digest TEXT NOT NULL, subjects JSONB NOT NULL, cursor INTEGER NOT NULL DEFAULT 0 CHECK(cursor>=0))`,
	`CREATE TABLE factory_priority_candidates(
scan TEXT NOT NULL REFERENCES factory_priority_scans(scan) ON DELETE CASCADE,
subject TEXT NOT NULL, repository BIGINT NOT NULL CHECK(repository>0), issue BIGINT NOT NULL CHECK(issue>0),
candidate JSONB, exclusion TEXT, fingerprint TEXT NOT NULL DEFAULT '',
PRIMARY KEY(scan,subject))`,
	FactoryCompletionsDDL,
	FactoryReassessmentsDDL,
	FactoryMCPCapabilitiesDDL,
	FactoryMCPSubmissionsDDL,
	`CREATE TABLE project_environment_grants(repository INTEGER PRIMARY KEY CHECK(repository>0), revision INTEGER NOT NULL, data JSONB NOT NULL)`,
	`CREATE TABLE project_requirement_decisions(id TEXT PRIMARY KEY, project_id TEXT NOT NULL REFERENCES projects(id), predecessor TEXT NOT NULL DEFAULT '', data JSONB NOT NULL)`,
	`CREATE TABLE project_requirement_heads(project_id TEXT PRIMARY KEY REFERENCES projects(id), decision TEXT NOT NULL)`,
	`CREATE TABLE project_approval_decisions(id TEXT PRIMARY KEY, project_id TEXT NOT NULL REFERENCES projects(id), predecessor TEXT NOT NULL DEFAULT '', data JSONB NOT NULL)`,
	`CREATE TABLE project_approval_heads(project_id TEXT PRIMARY KEY REFERENCES projects(id), decision TEXT NOT NULL)`,
	`CREATE TABLE identity_connections(id TEXT PRIMARY KEY, owner_id INTEGER NOT NULL, generation INTEGER NOT NULL, state TEXT NOT NULL, data JSONB NOT NULL, credential BYTEA NOT NULL)`,
	`CREATE TABLE identity_grants(id TEXT PRIMARY KEY, connection_id TEXT NOT NULL REFERENCES identity_connections(id), user_id INTEGER NOT NULL, project_id TEXT NOT NULL, revision INTEGER NOT NULL, revoked BOOLEAN NOT NULL, data JSONB NOT NULL)`,
	`CREATE UNIQUE INDEX identity_grant_recipient ON identity_grants(connection_id,user_id,project_id) WHERE revoked=FALSE`,
	`CREATE TABLE identity_leases(id TEXT PRIMARY KEY, connection_id TEXT NOT NULL REFERENCES identity_connections(id), data JSONB NOT NULL)`,
	`CREATE TABLE identity_events(id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY, owner_id INTEGER NOT NULL, connection_id TEXT NOT NULL, data JSONB NOT NULL)`,
	`CREATE TABLE project_lifecycle_grants(project_id TEXT PRIMARY KEY REFERENCES projects(id), revision INTEGER NOT NULL, data JSONB NOT NULL)`,
	`CREATE TABLE project_maintenance(project_id TEXT PRIMARY KEY REFERENCES projects(id), revision INTEGER NOT NULL, hold BOOLEAN NOT NULL, data JSONB NOT NULL)`,
	`CREATE TABLE project_preparations(id TEXT PRIMARY KEY, project_id TEXT NOT NULL REFERENCES projects(id), role TEXT NOT NULL, revision INTEGER NOT NULL, requirements TEXT NOT NULL, approval TEXT NOT NULL, data JSONB NOT NULL)`,
	`CREATE INDEX project_preparations_project ON project_preparations(project_id)`,
	`CREATE TABLE identity_executions(kind TEXT NOT NULL, execution_id TEXT NOT NULL, state TEXT NOT NULL, lease_id TEXT NOT NULL DEFAULT '', data JSONB NOT NULL, PRIMARY KEY(kind,execution_id))`,
	`CREATE OR REPLACE FUNCTION soda_reject_immutable() RETURNS trigger AS $$
BEGIN
  RAISE EXCEPTION 'soda: % is immutable', TG_TABLE_NAME;
END $$ LANGUAGE plpgsql`,
	`CREATE OR REPLACE FUNCTION soda_guard_application() RETURNS trigger AS $$
BEGIN
  IF NEW.id IS DISTINCT FROM OLD.id OR NEW.administrator_id IS DISTINCT FROM OLD.administrator_id
    OR NEW.name IS DISTINCT FROM OLD.name OR NEW.data_id IS DISTINCT FROM OLD.data_id
    OR NEW.owner_team IS DISTINCT FROM OLD.owner_team OR NEW.origin IS DISTINCT FROM OLD.origin
    OR NEW.hostname IS DISTINCT FROM OLD.hostname OR NEW.port IS DISTINCT FROM OLD.port
    OR (NEW.revision <> OLD.revision AND NEW.revision-1 <> OLD.revision)
  THEN RAISE EXCEPTION 'application identity is immutable'; END IF;
  RETURN NEW;
END $$ LANGUAGE plpgsql`,
	`CREATE OR REPLACE FUNCTION soda_guard_creation_profile() RETURNS trigger AS $$
BEGIN
  IF NEW.creation_profile IS DISTINCT FROM OLD.creation_profile THEN
    RAISE EXCEPTION 'creation profile is immutable';
  END IF;
  RETURN NEW;
END $$ LANGUAGE plpgsql`,
	`CREATE OR REPLACE FUNCTION soda_guard_run_binding() RETURNS trigger AS $$
BEGIN
  IF NEW.id IS DISTINCT FROM OLD.id
    OR (NEW.data->>'project_id') IS DISTINCT FROM (OLD.data->>'project_id')
    OR (NEW.data->>'role') IS DISTINCT FROM (OLD.data->>'role')
    OR (NEW.data->>'input_sha') IS DISTINCT FROM (OLD.data->>'input_sha')
    OR (NEW.data->>'started') IS DISTINCT FROM (OLD.data->>'started')
    OR (NEW.data->>'deadline') IS DISTINCT FROM (OLD.data->>'deadline')
    OR (NEW.data->>'image') IS DISTINCT FROM (OLD.data->>'image')
    OR (NEW.data->>'harness') IS DISTINCT FROM (OLD.data->>'harness')
    OR (NEW.data->>'model') IS DISTINCT FROM (OLD.data->>'model')
    OR (coalesce(OLD.data->>'outcome','')!='' AND coalesce(NEW.data->>'outcome','') IS DISTINCT FROM (OLD.data->>'outcome'))
    OR ((OLD.data->>'reconciled')::boolean IS TRUE AND (NEW.data->>'reconciled')::boolean IS NOT TRUE)
  THEN
    RAISE EXCEPTION 'factory run binding is immutable';
  END IF;
  RETURN NEW;
END $$ LANGUAGE plpgsql`,
	`CREATE OR REPLACE FUNCTION soda_guard_command() RETURNS trigger AS $$
BEGIN
  IF NEW.id IS DISTINCT FROM OLD.id OR NEW.type IS DISTINCT FROM OLD.type
    OR NEW.target IS DISTINCT FROM OLD.target OR NEW.principal IS DISTINCT FROM OLD.principal
    OR NEW.digest IS DISTINCT FROM OLD.digest OR NEW.payload IS DISTINCT FROM OLD.payload
    OR NEW.created IS DISTINCT FROM OLD.created
    OR (OLD.finished!='' AND (NEW.finished IS DISTINCT FROM OLD.finished OR NEW.outcome IS DISTINCT FROM OLD.outcome))
  THEN
    RAISE EXCEPTION 'factory command is immutable';
  END IF;
  RETURN NEW;
END $$ LANGUAGE plpgsql`,
	`CREATE OR REPLACE FUNCTION soda_guard_preparation_refs() RETURNS trigger AS $$
BEGIN
  IF NEW.id IS DISTINCT FROM OLD.id OR NEW.project_id IS DISTINCT FROM OLD.project_id
    OR NEW.role IS DISTINCT FROM OLD.role OR NEW.requirements IS DISTINCT FROM OLD.requirements
    OR NEW.approval IS DISTINCT FROM OLD.approval
  THEN
    RAISE EXCEPTION 'preparation references are immutable';
  END IF;
  RETURN NEW;
END $$ LANGUAGE plpgsql`,
	`CREATE OR REPLACE FUNCTION soda_guard_execution_identity() RETURNS trigger AS $$
BEGIN
  IF NEW.kind IS DISTINCT FROM OLD.kind OR NEW.execution_id IS DISTINCT FROM OLD.execution_id
    OR (NEW.data->>'digest') IS DISTINCT FROM (OLD.data->>'digest')
    OR (OLD.state='terminal' AND NEW.state IS DISTINCT FROM 'terminal')
  THEN
    RAISE EXCEPTION 'execution identity is immutable';
  END IF;
  RETURN NEW;
END $$ LANGUAGE plpgsql`,
	`CREATE TRIGGER immutable_creation_profile BEFORE UPDATE OF creation_profile ON projects FOR EACH ROW EXECUTE FUNCTION soda_guard_creation_profile()`,
	`CREATE TRIGGER application_identity_immutable BEFORE UPDATE ON applications FOR EACH ROW EXECUTE FUNCTION soda_guard_application()`,
	`CREATE TRIGGER application_revision_immutable_update BEFORE UPDATE ON application_revisions FOR EACH ROW EXECUTE FUNCTION soda_reject_immutable()`,
	`CREATE TRIGGER application_revision_immutable_delete BEFORE DELETE ON application_revisions FOR EACH ROW EXECUTE FUNCTION soda_reject_immutable()`,
	`CREATE TRIGGER factory_run_binding_immutable BEFORE UPDATE ON factory_runs FOR EACH ROW EXECUTE FUNCTION soda_guard_run_binding()`,
	`CREATE TRIGGER factory_command_immutable BEFORE UPDATE ON factory_commands FOR EACH ROW EXECUTE FUNCTION soda_guard_command()`,
	`CREATE TRIGGER factory_dispatch_reg_immutable_update BEFORE UPDATE ON factory_dispatch_regs FOR EACH ROW EXECUTE FUNCTION soda_reject_immutable()`,
	`CREATE TRIGGER factory_dispatch_reg_immutable_delete BEFORE DELETE ON factory_dispatch_regs FOR EACH ROW EXECUTE FUNCTION soda_reject_immutable()`,
	`CREATE TRIGGER factory_takeover_immutable_update BEFORE UPDATE ON factory_takeovers FOR EACH ROW EXECUTE FUNCTION soda_reject_immutable()`,
	`CREATE TRIGGER factory_takeover_immutable_delete BEFORE DELETE ON factory_takeovers FOR EACH ROW EXECUTE FUNCTION soda_reject_immutable()`,
	`CREATE TRIGGER factory_plan_revision_immutable_update BEFORE UPDATE ON factory_plan_revisions FOR EACH ROW EXECUTE FUNCTION soda_reject_immutable()`,
	`CREATE TRIGGER factory_plan_revision_immutable_delete BEFORE DELETE ON factory_plan_revisions FOR EACH ROW EXECUTE FUNCTION soda_reject_immutable()`,
	`CREATE TRIGGER factory_run_view_immutable_update BEFORE UPDATE ON factory_run_views FOR EACH ROW EXECUTE FUNCTION soda_reject_immutable()`,
	`CREATE TRIGGER factory_run_view_immutable_delete BEFORE DELETE ON factory_run_views FOR EACH ROW EXECUTE FUNCTION soda_reject_immutable()`,
	FactoryCompletionTriggers,
	FactoryCompletionDeleteTrigger,
	FactoryReassessmentDeleteTrigger,
	FactoryMCPCapabilityDeleteTrigger,
	FactoryMCPSubmissionTriggers,
	`CREATE TRIGGER issue_acceptance_decision_immutable_update BEFORE UPDATE ON issue_acceptance_decisions FOR EACH ROW EXECUTE FUNCTION soda_reject_immutable()`,
	`CREATE TRIGGER issue_acceptance_decision_immutable_delete BEFORE DELETE ON issue_acceptance_decisions FOR EACH ROW EXECUTE FUNCTION soda_reject_immutable()`,
	`CREATE TRIGGER issue_acceptance_withdrawal_immutable_update BEFORE UPDATE ON issue_acceptance_withdrawals FOR EACH ROW EXECUTE FUNCTION soda_reject_immutable()`,
	`CREATE TRIGGER issue_acceptance_withdrawal_immutable_delete BEFORE DELETE ON issue_acceptance_withdrawals FOR EACH ROW EXECUTE FUNCTION soda_reject_immutable()`,
	`CREATE TRIGGER project_requirement_decision_immutable_update BEFORE UPDATE ON project_requirement_decisions FOR EACH ROW EXECUTE FUNCTION soda_reject_immutable()`,
	`CREATE TRIGGER project_requirement_decision_immutable_delete BEFORE DELETE ON project_requirement_decisions FOR EACH ROW EXECUTE FUNCTION soda_reject_immutable()`,
	`CREATE TRIGGER project_approval_decision_immutable_update BEFORE UPDATE ON project_approval_decisions FOR EACH ROW EXECUTE FUNCTION soda_reject_immutable()`,
	`CREATE TRIGGER project_approval_decision_immutable_delete BEFORE DELETE ON project_approval_decisions FOR EACH ROW EXECUTE FUNCTION soda_reject_immutable()`,
	`CREATE TRIGGER identity_events_immutable_update BEFORE UPDATE ON identity_events FOR EACH ROW EXECUTE FUNCTION soda_reject_immutable()`,
	`CREATE TRIGGER identity_events_immutable_delete BEFORE DELETE ON identity_events FOR EACH ROW EXECUTE FUNCTION soda_reject_immutable()`,
	`CREATE TRIGGER preparation_refs_immutable BEFORE UPDATE ON project_preparations FOR EACH ROW EXECUTE FUNCTION soda_guard_preparation_refs()`,
	`CREATE TRIGGER identity_execution_immutable BEFORE UPDATE ON identity_executions FOR EACH ROW EXECUTE FUNCTION soda_guard_execution_identity()`,
}

// SchemaVersion is a format identifier. This unreleased database format has no
// compatibility migrations; older versioned files are refused without mutation.
func SchemaVersion() int { return schemaFormatVersion }

func schemaPresent(ctx context.Context, t *sql.Tx) (bool, error) {
	var hasVersion int
	if err := t.QueryRowContext(ctx, `SELECT count(*) FROM information_schema.tables WHERE table_schema='public' AND table_name='schema_version'`).Scan(&hasVersion); err != nil {
		return false, err
	}
	return hasVersion != 0, nil
}

func refuseUnversionedDatabase(ctx context.Context, t *sql.Tx) error {
	var objects int
	if err := t.QueryRowContext(ctx, `SELECT count(*) FROM information_schema.tables WHERE table_schema='public' AND table_type='BASE TABLE'`).Scan(&objects); err != nil {
		return err
	}
	if objects != 0 {
		return errors.New("refusing an unversioned nonempty database")
	}
	return nil
}

func storedSchemaVersion(ctx context.Context, t *sql.Tx) (int, error) {
	var count int
	var minimum, maximum sql.NullInt64
	if err := t.QueryRowContext(ctx, `SELECT count(*),min(version),max(version) FROM schema_version`).Scan(&count, &minimum, &maximum); err != nil {
		return 0, err
	}
	if count != 1 || !minimum.Valid || minimum.Int64 < 1 || minimum.Int64 != maximum.Int64 {
		return 0, errors.New("invalid database schema version record")
	}
	return int(minimum.Int64), nil
}

func verifyRequiredColumns(ctx context.Context, t *sql.Tx) error {
	for _, query := range []string{
		`SELECT id,administrator_id,name,data_id,owner_team,origin,hostname,port,revision,observation FROM applications LIMIT 0`,
		`SELECT application_id,revision,spec FROM application_revisions LIMIT 0`,
		`SELECT repository,revision,actor_id,data FROM factory_activity_settings LIMIT 0`,
		`SELECT id,login,name FROM users LIMIT 0`,
		`SELECT id,user_id,public,fingerprint FROM keys LIMIT 0`,
		`SELECT id,name,repository_id,owner_id,repository,ip,ready,creation_profile FROM projects LIMIT 0`,
		`SELECT project_id,user_id,login FROM memberships LIMIT 0`,
		`SELECT id,ciphertext FROM grant_key_check LIMIT 0`,
		`SELECT project_id,revision,data FROM project_lifecycle_grants LIMIT 0`,
		`SELECT project_id,revision,hold,data FROM project_maintenance LIMIT 0`,
		`SELECT id,project_id,role,revision,requirements,approval,data FROM project_preparations LIMIT 0`,
		`SELECT kind,execution_id,state,lease_id,data FROM identity_executions LIMIT 0`,
		`SELECT seq,id,active,settled,data FROM factory_runs LIMIT 0`,
		`SELECT id,type,target,principal,digest,payload,outcome,created,finished FROM factory_commands LIMIT 0`,
		`SELECT repository,revision,data FROM factory_policies LIMIT 0`,
		`SELECT id,revision,data FROM factory_capacity LIMIT 0`,
		`SELECT repository,revision,data FROM factory_operator_grants LIMIT 0`,
		`SELECT repository,connection,revision,data FROM factory_sponsorships LIMIT 0`,
		`SELECT connection,revision,data FROM factory_connection_usage_budgets LIMIT 0`,
		`SELECT repository,revision,open,data FROM factory_dispatch LIMIT 0`,
		`SELECT seq,id,repository,revision,data FROM factory_dispatch_regs LIMIT 0`,
		`SELECT seq,id,repository,issue,run,stage,revision,data FROM factory_assignments LIMIT 0`,
		`SELECT seq,assignment,repository,connection,state,data FROM factory_reservations LIMIT 0`,
		`SELECT run,repository,connection,minutes,started_at,ended_at,data FROM factory_usage LIMIT 0`,
		`SELECT run,member,project,data FROM factory_takeovers LIMIT 0`,
		`SELECT id,event,action,repository,issue,attempts,revoked,data FROM factory_lifecycle_actions LIMIT 0`,
		`SELECT event,revision,installation FROM factory_lifecycle_capabilities LIMIT 0`,
		`SELECT event,revision,enabled,data FROM factory_lifecycle_configs LIMIT 0`,
		`SELECT intent,owner_id,revision FROM factory_intent_allowance_heads LIMIT 0`,
		`SELECT id,intent,data FROM factory_intent_allowances LIMIT 0`,
		`SELECT result_revision,data FROM factory_intent_reviews LIMIT 0`,
		`SELECT id,repository,issue,predecessor,data FROM issue_acceptance_decisions LIMIT 0`,
		`SELECT repository,issue,decision FROM issue_acceptance_heads LIMIT 0`,
		`SELECT repository,issue,decision,withdrawer FROM issue_acceptance_withdrawals LIMIT 0`,
		`SELECT repository,revision,data FROM project_environment_grants LIMIT 0`,
		`SELECT id,project_id,predecessor,data FROM project_requirement_decisions LIMIT 0`,
		`SELECT project_id,decision FROM project_requirement_heads LIMIT 0`,
		`SELECT id,project_id,predecessor,data FROM project_approval_decisions LIMIT 0`,
		`SELECT project_id,decision FROM project_approval_heads LIMIT 0`,
		`SELECT seq,run,repository,issue,attempt FROM factory_run_views LIMIT 0`,
		`SELECT seq,assignment,repository,issue,run,stage,revision,data FROM factory_publications LIMIT 0`,
		`SELECT seq,publication,repository,issue,pr,stage,revision,data FROM factory_merges LIMIT 0`,
		`SELECT repository,generation,event,publication,occurrence,data FROM factory_completions LIMIT 0`,
		`SELECT cycle,repository,generation,event,state,data FROM factory_reassessments LIMIT 0`,
		`SELECT id,generation,next_turn FROM factory_readiness_budget LIMIT 0`,
		`SELECT id,delivery,repository,issue,creation_verified,creator_id,creator_write,native_issue_id,created_unix,title_digest,content_digest,generation,turn,retry_at,attempts FROM factory_readiness_sources LIMIT 0`,
		`SELECT repository,task_id FROM factory_creation_cursors LIMIT 0`,
		`SELECT source,repository,issue,state,cursor_repository,cursor_issue FROM factory_readiness_nodes LIMIT 0`,
	} {
		rows, err := t.QueryContext(ctx, query)
		if err != nil {
			return errors.New("database schema is incomplete")
		}
		if err = rows.Close(); err != nil {
			return err
		}
	}
	return nil
}

func verifyTrigger(ctx context.Context, t *sql.Tx, name, table string) error {
	var found int
	if err := t.QueryRowContext(ctx, `SELECT count(*) FROM information_schema.triggers WHERE trigger_name=$1 AND event_object_table=$2`, name, table).Scan(&found); err != nil {
		return err
	}
	if found != 1 {
		return errors.New("database schema is incomplete")
	}
	return nil
}

func verifyImmutableCreationProfile(ctx context.Context, t *sql.Tx) error {
	return verifyTrigger(ctx, t, "immutable_creation_profile", "projects")
}

func verifyImmutablePreparationRefs(ctx context.Context, t *sql.Tx) error {
	return verifyTrigger(ctx, t, "preparation_refs_immutable", "project_preparations")
}

func verifyImmutableExecutionIdentity(ctx context.Context, t *sql.Tx) error {
	return verifyTrigger(ctx, t, "identity_execution_immutable", "identity_executions")
}

func verifyImmutableApplications(ctx context.Context, t *sql.Tx) error {
	for _, guard := range [][2]string{
		{"application_identity_immutable", "applications"},
		{"application_revision_immutable_update", "application_revisions"},
		{"application_revision_immutable_delete", "application_revisions"},
	} {
		if err := verifyTrigger(ctx, t, guard[0], guard[1]); err != nil {
			return err
		}
	}
	return nil
}

func loadSchemaVersion(ctx context.Context, t *sql.Tx) (int, error) {
	present, err := schemaPresent(ctx, t)
	if err != nil {
		return 0, err
	}
	if !present {
		return 0, refuseUnversionedDatabase(ctx, t)
	}
	return storedSchemaVersion(ctx, t)
}

func initializeSchema(ctx context.Context, db *sql.DB) error {
	inner, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	t := inner
	defer func() { _ = t.Rollback() }()
	version, err := loadSchemaVersion(ctx, t)
	if err != nil {
		return err
	}
	if version == 0 {
		// DDL carries no parameters.
		for _, statement := range schemaStatements {
			if _, err = inner.ExecContext(ctx, statement); err != nil {
				return fmt.Errorf("create current database schema: %w", err)
			}
		}
	} else if version != SchemaVersion() {
		return errors.New("database schema differs from this application")
	}
	if err = verifyRequiredColumns(ctx, t); err != nil {
		return err
	}
	if err = verifyImmutableCreationProfile(ctx, t); err != nil {
		return err
	}
	if err = verifyImmutablePreparationRefs(ctx, t); err != nil {
		return err
	}
	if err = verifyImmutableExecutionIdentity(ctx, t); err != nil {
		return err
	}
	if err = verifyImmutableApplications(ctx, t); err != nil {
		return err
	}
	return t.Commit()
}

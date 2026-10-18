// Embedded PostgreSQL schema, mechanically copied from
// internal/store/schema.go (the single source of truth). The
// drift test below re-extracts the Go schemaStatements entries at
// test time — inline raw literals plus named raw-string constants
// from the four owner files — and fails on any difference, so a Go
// schema change forces a regeneration of this file in the same patch.
pub const SCHEMA_VERSION: i64 = 58;

pub const STATEMENTS: &[&str] = &[
    r#"CREATE TABLE factory_activity_settings(repository BIGINT PRIMARY KEY CHECK(repository>0), revision BIGINT NOT NULL CHECK(revision>0), actor_id BIGINT NOT NULL CHECK(actor_id>0), data JSONB NOT NULL)"#,
    r#"CREATE TABLE schema_version(version INTEGER PRIMARY KEY)"#,
    r#"INSERT INTO schema_version(version) VALUES(58)"#,
    r#"CREATE TABLE applications(id TEXT PRIMARY KEY, administrator_id BIGINT NOT NULL CHECK(administrator_id>0), name TEXT NOT NULL, data_id TEXT NOT NULL UNIQUE, owner_team TEXT NOT NULL, origin TEXT NOT NULL UNIQUE, hostname TEXT NOT NULL UNIQUE, port INTEGER NOT NULL UNIQUE CHECK(port>=20000 AND port<30000), revision BIGINT NOT NULL CHECK(revision>0), observation JSONB CHECK(observation IS NULL OR octet_length(observation::text)<=2048))"#,
    r#"CREATE TABLE application_revisions(application_id TEXT NOT NULL REFERENCES applications(id), revision BIGINT NOT NULL CHECK(revision>0), spec JSONB NOT NULL CHECK(octet_length(spec::text)<=4096), PRIMARY KEY(application_id,revision))"#,
    r#"ALTER TABLE applications ADD FOREIGN KEY(id,revision) REFERENCES application_revisions(application_id,revision) DEFERRABLE INITIALLY DEFERRED"#,
    r#"CREATE TABLE factory_departments(id TEXT PRIMARY KEY, owner_id BIGINT NOT NULL CHECK(owner_id>0), data JSONB NOT NULL)"#,
    r#"CREATE TABLE factory_intent_heads(id TEXT PRIMARY KEY, owner_id BIGINT NOT NULL CHECK(owner_id>0), revision TEXT NOT NULL)"#,
    r#"CREATE TABLE factory_intent_revisions(id TEXT PRIMARY KEY, intent TEXT NOT NULL REFERENCES factory_intent_heads(id), data JSONB NOT NULL)"#,
    r#"CREATE TABLE factory_judgment_budget(id INTEGER PRIMARY KEY CHECK(id=1), spent INTEGER NOT NULL CHECK(spent>=0))"#,
    r#"INSERT INTO factory_judgment_budget(id,spent) VALUES(1,0)"#,
    r#"CREATE TABLE factory_priority_judgments(snapshot TEXT PRIMARY KEY, max_calls INTEGER NOT NULL CHECK(max_calls BETWEEN 0 AND 16), data JSONB)"#,
    r#"CREATE TABLE factory_planning(repository BIGINT NOT NULL, issue BIGINT NOT NULL, snapshot TEXT NOT NULL, data JSONB NOT NULL, PRIMARY KEY(repository,issue))"#,
    r#"CREATE TABLE factory_plan_revisions(attempt_root TEXT NOT NULL CHECK(attempt_root ~ '^[a-f0-9]{32}$'), plan_id TEXT NOT NULL CHECK(plan_id ~ '^[a-f0-9]{32}$'), generation BIGINT NOT NULL CHECK(generation>=0), digest TEXT NOT NULL CHECK(digest ~ '^[a-f0-9]{64}$'), data JSONB NOT NULL, PRIMARY KEY(attempt_root,plan_id))"#,
    r#"CREATE TABLE factory_plan_heads(attempt_root TEXT PRIMARY KEY CHECK(attempt_root ~ '^[a-f0-9]{32}$'), plan_id TEXT NOT NULL CHECK(plan_id ~ '^[a-f0-9]{32}$'), generation BIGINT NOT NULL CHECK(generation>=0))"#,
    r#"CREATE TABLE factory_plan_tasks(attempt_root TEXT NOT NULL CHECK(attempt_root ~ '^[a-f0-9]{32}$'), plan_id TEXT NOT NULL CHECK(plan_id ~ '^[a-f0-9]{32}$'), task_id TEXT NOT NULL CHECK(task_id ~ '^[a-z][a-z0-9_-]{0,31}$'), revision BIGINT NOT NULL CHECK(revision>=0), state TEXT NOT NULL, data JSONB NOT NULL, PRIMARY KEY(attempt_root,plan_id,task_id))"#,
    r#"CREATE TABLE factory_intent_generations(revision TEXT PRIMARY KEY REFERENCES factory_intent_revisions(id), data JSONB NOT NULL)"#,
    r#"CREATE TABLE factory_intent_tasks(revision TEXT NOT NULL REFERENCES factory_intent_revisions(id), task TEXT NOT NULL, data JSONB NOT NULL, PRIMARY KEY(revision,task))"#,
    r#"CREATE TABLE factory_intent_planning_calls(intent TEXT NOT NULL REFERENCES factory_intent_heads(id), snapshot TEXT PRIMARY KEY REFERENCES factory_priority_judgments(snapshot))"#,
    r#"CREATE TABLE factory_intent_allowance_heads(intent TEXT PRIMARY KEY REFERENCES factory_intent_heads(id), owner_id BIGINT NOT NULL CHECK(owner_id>0), revision TEXT NOT NULL)"#,
    r#"CREATE TABLE factory_intent_allowances(id TEXT PRIMARY KEY, intent TEXT NOT NULL REFERENCES factory_intent_allowance_heads(intent), data JSONB NOT NULL)"#,
    r#"CREATE TABLE factory_intent_reviews(result_revision TEXT PRIMARY KEY REFERENCES factory_intent_revisions(id), data JSONB NOT NULL)"#,
    r#"CREATE TABLE users(id INTEGER PRIMARY KEY CHECK(id>0), login TEXT NOT NULL, name TEXT NOT NULL DEFAULT '')"#,
    r#"CREATE TABLE keys(id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY, user_id INTEGER NOT NULL REFERENCES users(id), public TEXT NOT NULL, fingerprint TEXT NOT NULL, UNIQUE(user_id,fingerprint))"#,
    r#"CREATE TABLE projects(id TEXT PRIMARY KEY, name TEXT NOT NULL, repository_id INTEGER NOT NULL UNIQUE, owner_id INTEGER NOT NULL REFERENCES users(id), repository TEXT NOT NULL, ip TEXT NOT NULL DEFAULT '', ready BOOLEAN NOT NULL DEFAULT FALSE, creation_profile JSONB CHECK(creation_profile IS NULL OR octet_length(creation_profile::text)<=1024))"#,
    r#"CREATE TABLE memberships(project_id TEXT NOT NULL REFERENCES projects(id), user_id INTEGER NOT NULL REFERENCES users(id), login TEXT NOT NULL, PRIMARY KEY(project_id,user_id), UNIQUE(project_id,login))"#,
    r#"CREATE TABLE grant_key_check(id INTEGER PRIMARY KEY CHECK(id=1), ciphertext BYTEA NOT NULL)"#,
    r#"CREATE TABLE factory_runs(
seq BIGINT GENERATED ALWAYS AS IDENTITY,
id TEXT PRIMARY KEY,
active BOOLEAN NOT NULL, settled BOOLEAN NOT NULL,
data JSONB NOT NULL)"#,
    r#"CREATE INDEX factory_unsettled_runs ON factory_runs(id) WHERE active OR NOT settled OR COALESCE(data->>'reconciled','false') <> 'true'"#,
    r#"CREATE TABLE factory_commands(
id TEXT PRIMARY KEY, type TEXT NOT NULL, target TEXT NOT NULL DEFAULT '',
principal TEXT NOT NULL, digest TEXT NOT NULL, payload TEXT NOT NULL DEFAULT '' CHECK(char_length(payload)<=8192),
outcome TEXT NOT NULL DEFAULT '' CHECK(char_length(outcome)<=65536),
created TEXT NOT NULL, finished TEXT NOT NULL DEFAULT '')"#,
    r#"CREATE INDEX factory_finished_commands_target ON factory_commands(target,type,created,id) WHERE finished<>''"#,
    r#"CREATE TABLE factory_policies(repository INTEGER PRIMARY KEY CHECK(repository>0), revision INTEGER NOT NULL, data JSONB NOT NULL)"#,
    r#"CREATE TABLE factory_capacity(id INTEGER PRIMARY KEY CHECK(id=1), revision INTEGER NOT NULL, data JSONB NOT NULL)"#,
    r#"CREATE TABLE factory_operator_grants(repository INTEGER PRIMARY KEY CHECK(repository>0), revision INTEGER NOT NULL, data JSONB NOT NULL)"#,
    r#"CREATE TABLE factory_sponsorships(repository INTEGER NOT NULL CHECK(repository>0), connection TEXT NOT NULL, revision INTEGER NOT NULL, data JSONB NOT NULL, PRIMARY KEY(repository,connection))"#,
    r#"CREATE TABLE factory_connection_usage_budgets(connection TEXT PRIMARY KEY, revision INTEGER NOT NULL, data JSONB NOT NULL)"#,
    r#"CREATE TABLE factory_dispatch(repository INTEGER PRIMARY KEY CHECK(repository>0), revision INTEGER NOT NULL, open BOOLEAN NOT NULL, data JSONB NOT NULL)"#,
    r#"CREATE TABLE factory_dispatch_regs(seq BIGINT GENERATED ALWAYS AS IDENTITY, id TEXT PRIMARY KEY, repository INTEGER NOT NULL CHECK(repository>0), revision INTEGER NOT NULL, data JSONB NOT NULL)"#,
    r#"CREATE INDEX factory_dispatch_regs_repository ON factory_dispatch_regs(repository)"#,
    r#"CREATE TABLE factory_takeovers(run TEXT NOT NULL, member TEXT NOT NULL, project TEXT NOT NULL REFERENCES projects(id), data JSONB NOT NULL, PRIMARY KEY(run,member))"#,
    r#"CREATE TABLE factory_lifecycle_actions(id TEXT PRIMARY KEY, event TEXT NOT NULL, action TEXT NOT NULL, repository BIGINT NOT NULL CHECK(repository>0), issue BIGINT NOT NULL CHECK(issue>0), attempts INTEGER NOT NULL DEFAULT 0 CHECK(attempts>=0), revoked BOOLEAN NOT NULL DEFAULT FALSE, data JSONB NOT NULL)"#,
    r#"CREATE INDEX factory_lifecycle_actions_target ON factory_lifecycle_actions(repository,issue,event)"#,
    r#"CREATE TABLE factory_lifecycle_capabilities(event TEXT PRIMARY KEY, revision TEXT NOT NULL, installation TEXT NOT NULL)"#,
    r#"CREATE TABLE factory_lifecycle_configs(event TEXT PRIMARY KEY, revision BIGINT NOT NULL CHECK(revision>=0), enabled BOOLEAN NOT NULL, data JSONB NOT NULL)"#,
    r#"CREATE TABLE factory_run_views(seq BIGINT GENERATED ALWAYS AS IDENTITY, run TEXT PRIMARY KEY REFERENCES factory_runs(id), repository INTEGER NOT NULL CHECK(repository>0), issue INTEGER NOT NULL DEFAULT 0 CHECK(issue>=0), attempt TEXT NOT NULL DEFAULT '' CHECK(char_length(attempt)<=64))"#,
    r#"CREATE TABLE factory_assignments(seq BIGINT GENERATED ALWAYS AS IDENTITY, id TEXT PRIMARY KEY, repository INTEGER NOT NULL CHECK(repository>0), issue INTEGER NOT NULL CHECK(issue>0 OR (issue=0 AND COALESCE(data->'subject'->>'kind','')='completion' AND COALESCE(data->'subject'->'completion'->>'completion_id','')~'^[a-f0-9]{32}$' AND COALESCE(data->'subject'->'completion'->>'cycle_id','')~'^[a-f0-9]{32}$' AND COALESCE(data->'subject'->'completion'->>'inventory_digest','')~'^[a-f0-9]{64}$' AND COALESCE(data->>'phase','')='reassess' AND COALESCE(data->>'role','')='soda-coder' AND COALESCE(data->>'parent_assignment','')='' AND COALESCE(data->>'publication_assignment','')='' AND COALESCE(data->>'id','')~'^[a-f0-9]{32}$' AND COALESCE(data->>'id','')=id AND COALESCE(data->>'attempt_root','')=id AND COALESCE(data->>'repository','')=repository::TEXT AND COALESCE(data->>'issue','')='0' AND COALESCE(data->>'acceptance','')='') OR (issue=0 AND COALESCE(data->'subject'->>'kind','')='intent' AND COALESCE(data->'subject'->'intent'->>'intent_id','')~'^d[0-9a-f]{24}$' AND COALESCE(data->'subject'->'intent'->>'owner_id','')~'^[1-9][0-9]*$' AND COALESCE(data->>'phase','')='decompose' AND COALESCE(data->>'role','')='soda-coder' AND COALESCE(data->>'parent_assignment','')='' AND COALESCE(data->>'publication_assignment','')='' AND COALESCE(data->>'id','')~'^[a-f0-9]{32}$' AND COALESCE(data->>'id','')=id AND COALESCE(data->>'attempt_root','')=id AND COALESCE(data->>'repository','')=repository::TEXT AND COALESCE(data->>'issue','')='0' AND COALESCE(data->>'acceptance','')='')), run TEXT NOT NULL, stage TEXT NOT NULL, revision INTEGER NOT NULL, data JSONB NOT NULL)"#,
    r#"CREATE TABLE factory_attempt_allowances(repository INTEGER NOT NULL CHECK(repository>0), issue INTEGER NOT NULL CHECK((issue>0 AND COALESCE(data->'completion','null'::JSONB)='null'::JSONB AND COALESCE(data->'intent','null'::JSONB)='null'::JSONB) OR (issue=0 AND COALESCE(data->'completion'->>'completion_id','')~'^[a-f0-9]{32}$' AND COALESCE(data->'completion'->>'cycle_id','')~'^[a-f0-9]{32}$' AND COALESCE(data->'completion'->>'inventory_digest','')~'^[a-f0-9]{64}$' AND COALESCE(data->>'repository','')=repository::TEXT AND COALESCE(data->>'issue','')='0' AND COALESCE(data->>'root_assignment','')=root_assignment) OR (issue=0 AND COALESCE(data->'intent'->>'intent_id','')~'^d[0-9a-f]{24}$' AND COALESCE(data->'intent'->>'owner_id','')~'^[1-9][0-9]*$' AND COALESCE(data->'completion','null'::JSONB)='null'::JSONB AND COALESCE(data->>'repository','')=repository::TEXT AND COALESCE(data->>'issue','')='0' AND COALESCE(data->>'root_assignment','')=root_assignment)), root_assignment TEXT PRIMARY KEY REFERENCES factory_assignments(id), revision BIGINT NOT NULL CHECK(revision>=0), data JSONB NOT NULL)"#,
    r#"CREATE INDEX factory_attempt_allowance_issues ON factory_attempt_allowances(repository,issue)"#,
    r#"CREATE INDEX factory_active_attempts ON factory_attempt_allowances(repository) WHERE data->>'active'='true'"#,
    r#"CREATE UNIQUE INDEX factory_unfinished_assignment ON factory_assignments(repository,issue) WHERE stage='assigned' AND issue>0"#,
    r#"CREATE UNIQUE INDEX factory_unfinished_completion_assignment ON factory_assignments(repository,(data->'subject'->'completion'->>'completion_id'),(data->'subject'->'completion'->>'cycle_id')) WHERE stage='assigned' AND issue=0"#,
    r#"CREATE INDEX factory_assignments_issue_history ON factory_assignments(repository,issue,seq DESC)"#,
    r#"CREATE INDEX factory_finished_children ON factory_assignments(seq) WHERE stage='finished' AND data->>'publication_assignment'<>id"#,
    r#"CREATE TABLE factory_reservations(seq BIGINT GENERATED ALWAYS AS IDENTITY, assignment TEXT PRIMARY KEY, repository INTEGER NOT NULL CHECK(repository>0), connection TEXT NOT NULL, state TEXT NOT NULL, data JSONB NOT NULL)"#,
    r#"CREATE INDEX factory_reservations_connection_state ON factory_reservations(connection,state)"#,
    r#"CREATE TABLE factory_usage(run TEXT PRIMARY KEY, repository INTEGER NOT NULL CHECK(repository>0), connection TEXT NOT NULL, minutes INTEGER NOT NULL CHECK(minutes>=0), started_at TIMESTAMPTZ NOT NULL, ended_at TIMESTAMPTZ NOT NULL CHECK(ended_at>=started_at), data JSONB NOT NULL)"#,
    r#"CREATE INDEX factory_usage_connection_ended ON factory_usage(connection,ended_at)"#,
    r#"CREATE TABLE issue_acceptance_decisions(id TEXT PRIMARY KEY, repository INTEGER NOT NULL CHECK(repository>0), issue INTEGER NOT NULL CHECK(issue>0), predecessor TEXT NOT NULL DEFAULT '', data JSONB NOT NULL)"#,
    r#"CREATE TABLE issue_acceptance_heads(repository INTEGER NOT NULL CHECK(repository>0), issue INTEGER NOT NULL CHECK(issue>0), decision TEXT NOT NULL, PRIMARY KEY(repository,issue))"#,
    r#"CREATE TABLE issue_acceptance_withdrawals(repository INTEGER NOT NULL CHECK(repository>0), issue INTEGER NOT NULL CHECK(issue>0), decision TEXT NOT NULL, withdrawer INTEGER NOT NULL CHECK(withdrawer>0), PRIMARY KEY(repository,issue,decision))"#,
    r#"CREATE TABLE issue_controls(repository INTEGER NOT NULL CHECK(repository>0), issue INTEGER NOT NULL CHECK(issue>0), revision INTEGER NOT NULL CHECK(revision>0), data JSONB NOT NULL, PRIMARY KEY(repository,issue))"#,
    r#"CREATE TABLE intake_deliveries(delivery TEXT PRIMARY KEY, repository INTEGER NOT NULL CHECK(repository>0), issue INTEGER NOT NULL CHECK(issue>0), kind TEXT NOT NULL, received_at INTEGER NOT NULL)"#,
    r#"CREATE TABLE factory_publications(seq BIGINT GENERATED ALWAYS AS IDENTITY, assignment TEXT PRIMARY KEY, repository INTEGER NOT NULL CHECK(repository>0), issue INTEGER NOT NULL CHECK(issue>0), run TEXT NOT NULL, stage TEXT NOT NULL, revision INTEGER NOT NULL, data JSONB NOT NULL)"#,
    r#"CREATE TABLE factory_check_assessments(repository INTEGER NOT NULL CHECK(repository>0), pr INTEGER NOT NULL CHECK(pr>0), revision INTEGER NOT NULL, data JSONB NOT NULL, PRIMARY KEY(repository,pr))"#,
    r#"CREATE TABLE factory_merges(seq BIGINT GENERATED ALWAYS AS IDENTITY, publication TEXT PRIMARY KEY, repository INTEGER NOT NULL CHECK(repository>0), issue INTEGER NOT NULL CHECK(issue>0), pr INTEGER NOT NULL CHECK(pr>0), stage TEXT NOT NULL, revision INTEGER NOT NULL, data JSONB NOT NULL)"#,
    r#"CREATE TABLE factory_readiness_budget(id INTEGER PRIMARY KEY CHECK(id=1), generation BIGINT NOT NULL CHECK(generation>=1), next_turn BIGINT NOT NULL CHECK(next_turn>=1))"#,
    r#"INSERT INTO factory_readiness_budget(id,generation,next_turn) VALUES(1,1,1)"#,
    r#"CREATE TABLE factory_readiness_sources(
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
CHECK(NOT creation_verified OR (creator_id>0 AND native_issue_id>0 AND created_unix>0 AND octet_length(title_digest)=64 AND octet_length(content_digest)=64)))"#,
    r#"CREATE TABLE factory_creation_cursors(repository BIGINT PRIMARY KEY CHECK(repository>0), task_id BIGINT NOT NULL CHECK(task_id>=0))"#,
    r#"CREATE TABLE factory_readiness_nodes(
source TEXT NOT NULL REFERENCES factory_readiness_sources(id) ON DELETE CASCADE,
repository BIGINT NOT NULL CHECK(repository>0), issue BIGINT NOT NULL CHECK(issue>0),
state TEXT NOT NULL DEFAULT 'queued' CHECK(state IN ('queued','scanning','done')),
cursor_repository BIGINT NOT NULL DEFAULT 0 CHECK(cursor_repository>=0), cursor_issue BIGINT NOT NULL DEFAULT 0 CHECK(cursor_issue>=0),
PRIMARY KEY(source,repository,issue), CHECK((cursor_repository=0)=(cursor_issue=0)))"#,
    r#"CREATE TABLE factory_priority_scans(
scan TEXT PRIMARY KEY, native_revision BIGINT NOT NULL CHECK(native_revision>=0),
local_digest TEXT NOT NULL, subjects JSONB NOT NULL, cursor INTEGER NOT NULL DEFAULT 0 CHECK(cursor>=0))"#,
    r#"CREATE TABLE factory_priority_candidates(
scan TEXT NOT NULL REFERENCES factory_priority_scans(scan) ON DELETE CASCADE,
subject TEXT NOT NULL, repository BIGINT NOT NULL CHECK(repository>0), issue BIGINT NOT NULL CHECK(issue>0),
candidate JSONB, exclusion TEXT, fingerprint TEXT NOT NULL DEFAULT '',
PRIMARY KEY(scan,subject))"#,
    r#"CREATE TABLE factory_completions(
repository BIGINT NOT NULL CHECK(repository>0), generation BIGINT NOT NULL CHECK(generation>=1),
event TEXT NOT NULL CHECK(event ~ '^[a-f0-9]{32}$'),
publication TEXT NOT NULL CHECK(publication ~ '^[a-f0-9]{32}$'),
occurrence TEXT NOT NULL CHECK(char_length(occurrence)>0 AND char_length(occurrence)<=256),
data JSONB NOT NULL,
PRIMARY KEY(repository,generation), UNIQUE(publication,occurrence))"#,
    r#"CREATE TABLE factory_reassessments(
cycle TEXT PRIMARY KEY CHECK(cycle ~ '^[a-f0-9]{32}$'),
repository BIGINT NOT NULL CHECK(repository>0), generation BIGINT NOT NULL CHECK(generation>=1),
event TEXT NOT NULL CHECK(event ~ '^[a-f0-9]{32}$'),
state TEXT NOT NULL, data JSONB NOT NULL,
FOREIGN KEY(repository,generation) REFERENCES factory_completions(repository,generation))"#,
    r#"CREATE TABLE factory_stall_events(
seq BIGSERIAL UNIQUE NOT NULL,
repository BIGINT NOT NULL CHECK(repository>0), generation BIGINT NOT NULL CHECK(generation>=0),
event TEXT NOT NULL CHECK(event ~ '^[a-f0-9]{32}$'),
episode TEXT NOT NULL CHECK(episode ~ '^[a-f0-9]{32}$'),
fingerprint TEXT NOT NULL CHECK(fingerprint ~ '^[a-f0-9]{64}$'),
data JSONB NOT NULL,
PRIMARY KEY(repository,generation,fingerprint))"#,
    r#"CREATE TABLE factory_mcp_capabilities(
id TEXT PRIMARY KEY CHECK(id ~ '^[a-f0-9]{32}$'),
assignment TEXT NOT NULL CHECK(assignment ~ '^[a-f0-9]{32}$'),
run TEXT NOT NULL CHECK(char_length(run)>0 AND char_length(run)<=256),
generation BIGINT NOT NULL CHECK(generation>=1),
profile TEXT NOT NULL, subject TEXT NOT NULL,
repository BIGINT NOT NULL CHECK(repository>0), project TEXT NOT NULL,
preparation_digest TEXT NOT NULL, authority_digest TEXT NOT NULL,
deadline BIGINT NOT NULL, state TEXT NOT NULL,
data JSONB NOT NULL)"#,
    r#"CREATE TABLE factory_mcp_submissions(
assignment TEXT NOT NULL CHECK(assignment ~ '^[a-f0-9]{32}$'),
submission TEXT NOT NULL CHECK(submission ~ '^[a-f0-9]{32}$'),
tool TEXT NOT NULL,
input_digest TEXT NOT NULL CHECK(input_digest ~ '^[a-f0-9]{64}$'),
payload_hash TEXT NOT NULL CHECK(payload_hash ~ '^[a-f0-9]{64}$'),
payload BYTEA NOT NULL CHECK(octet_length(payload)>0 AND octet_length(payload)<=65536),
state TEXT NOT NULL, reason TEXT NOT NULL DEFAULT '',
receipt JSONB NOT NULL,
PRIMARY KEY(assignment,submission))"#,
    r#"CREATE TABLE factory_mcp_submission_guards(assignment TEXT NOT NULL CHECK(assignment ~ '^[a-f0-9]{32}$'), PRIMARY KEY(assignment))"#,
    r#"CREATE TABLE project_environment_grants(repository INTEGER PRIMARY KEY CHECK(repository>0), revision INTEGER NOT NULL, data JSONB NOT NULL)"#,
    r#"CREATE TABLE project_requirement_decisions(id TEXT PRIMARY KEY, project_id TEXT NOT NULL REFERENCES projects(id), predecessor TEXT NOT NULL DEFAULT '', data JSONB NOT NULL)"#,
    r#"CREATE TABLE project_requirement_heads(project_id TEXT PRIMARY KEY REFERENCES projects(id), decision TEXT NOT NULL)"#,
    r#"CREATE TABLE project_approval_decisions(id TEXT PRIMARY KEY, project_id TEXT NOT NULL REFERENCES projects(id), predecessor TEXT NOT NULL DEFAULT '', data JSONB NOT NULL)"#,
    r#"CREATE TABLE project_approval_heads(project_id TEXT PRIMARY KEY REFERENCES projects(id), decision TEXT NOT NULL)"#,
    r#"CREATE TABLE identity_connections(id TEXT PRIMARY KEY, owner_id INTEGER NOT NULL, generation INTEGER NOT NULL, state TEXT NOT NULL, data JSONB NOT NULL, credential BYTEA NOT NULL)"#,
    r#"CREATE TABLE identity_grants(id TEXT PRIMARY KEY, connection_id TEXT NOT NULL REFERENCES identity_connections(id), user_id INTEGER NOT NULL, project_id TEXT NOT NULL, revision INTEGER NOT NULL, revoked BOOLEAN NOT NULL, data JSONB NOT NULL)"#,
    r#"CREATE UNIQUE INDEX identity_grant_recipient ON identity_grants(connection_id,user_id,project_id) WHERE revoked=FALSE"#,
    r#"CREATE TABLE identity_leases(id TEXT PRIMARY KEY, connection_id TEXT NOT NULL REFERENCES identity_connections(id), data JSONB NOT NULL)"#,
    r#"CREATE TABLE identity_events(id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY, owner_id INTEGER NOT NULL, connection_id TEXT NOT NULL, data JSONB NOT NULL)"#,
    r#"CREATE TABLE project_lifecycle_grants(project_id TEXT PRIMARY KEY REFERENCES projects(id), revision INTEGER NOT NULL, data JSONB NOT NULL)"#,
    r#"CREATE TABLE project_maintenance(project_id TEXT PRIMARY KEY REFERENCES projects(id), revision INTEGER NOT NULL, hold BOOLEAN NOT NULL, data JSONB NOT NULL)"#,
    r#"CREATE TABLE project_preparations(id TEXT PRIMARY KEY, project_id TEXT NOT NULL REFERENCES projects(id), role TEXT NOT NULL, revision INTEGER NOT NULL, requirements TEXT NOT NULL, approval TEXT NOT NULL, data JSONB NOT NULL)"#,
    r#"CREATE INDEX project_preparations_project ON project_preparations(project_id)"#,
    r#"CREATE TABLE identity_executions(kind TEXT NOT NULL, execution_id TEXT NOT NULL, state TEXT NOT NULL, lease_id TEXT NOT NULL DEFAULT '', data JSONB NOT NULL, PRIMARY KEY(kind,execution_id))"#,
    r#"CREATE OR REPLACE FUNCTION soda_reject_immutable() RETURNS trigger AS $$
BEGIN
  RAISE EXCEPTION 'soda: % is immutable', TG_TABLE_NAME;
END $$ LANGUAGE plpgsql"#,
    r#"CREATE OR REPLACE FUNCTION soda_guard_application() RETURNS trigger AS $$
BEGIN
  IF NEW.id IS DISTINCT FROM OLD.id OR NEW.administrator_id IS DISTINCT FROM OLD.administrator_id
    OR NEW.name IS DISTINCT FROM OLD.name OR NEW.data_id IS DISTINCT FROM OLD.data_id
    OR NEW.owner_team IS DISTINCT FROM OLD.owner_team OR NEW.origin IS DISTINCT FROM OLD.origin
    OR NEW.hostname IS DISTINCT FROM OLD.hostname OR NEW.port IS DISTINCT FROM OLD.port
    OR (NEW.revision <> OLD.revision AND NEW.revision-1 <> OLD.revision)
  THEN RAISE EXCEPTION 'application identity is immutable'; END IF;
  RETURN NEW;
END $$ LANGUAGE plpgsql"#,
    r#"CREATE OR REPLACE FUNCTION soda_guard_creation_profile() RETURNS trigger AS $$
BEGIN
  IF NEW.creation_profile IS DISTINCT FROM OLD.creation_profile THEN
    RAISE EXCEPTION 'creation profile is immutable';
  END IF;
  RETURN NEW;
END $$ LANGUAGE plpgsql"#,
    r#"CREATE OR REPLACE FUNCTION soda_guard_run_binding() RETURNS trigger AS $$
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
END $$ LANGUAGE plpgsql"#,
    r#"CREATE OR REPLACE FUNCTION soda_guard_command() RETURNS trigger AS $$
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
END $$ LANGUAGE plpgsql"#,
    r#"CREATE OR REPLACE FUNCTION soda_guard_preparation_refs() RETURNS trigger AS $$
BEGIN
  IF NEW.id IS DISTINCT FROM OLD.id OR NEW.project_id IS DISTINCT FROM OLD.project_id
    OR NEW.role IS DISTINCT FROM OLD.role OR NEW.requirements IS DISTINCT FROM OLD.requirements
    OR NEW.approval IS DISTINCT FROM OLD.approval
  THEN
    RAISE EXCEPTION 'preparation references are immutable';
  END IF;
  RETURN NEW;
END $$ LANGUAGE plpgsql"#,
    r#"CREATE OR REPLACE FUNCTION soda_guard_execution_identity() RETURNS trigger AS $$
BEGIN
  IF NEW.kind IS DISTINCT FROM OLD.kind OR NEW.execution_id IS DISTINCT FROM OLD.execution_id
    OR (NEW.data->>'digest') IS DISTINCT FROM (OLD.data->>'digest')
    OR (OLD.state='terminal' AND NEW.state IS DISTINCT FROM 'terminal')
  THEN
    RAISE EXCEPTION 'execution identity is immutable';
  END IF;
  RETURN NEW;
END $$ LANGUAGE plpgsql"#,
    r#"CREATE TRIGGER immutable_creation_profile BEFORE UPDATE OF creation_profile ON projects FOR EACH ROW EXECUTE FUNCTION soda_guard_creation_profile()"#,
    r#"CREATE TRIGGER application_identity_immutable BEFORE UPDATE ON applications FOR EACH ROW EXECUTE FUNCTION soda_guard_application()"#,
    r#"CREATE TRIGGER application_revision_immutable_update BEFORE UPDATE ON application_revisions FOR EACH ROW EXECUTE FUNCTION soda_reject_immutable()"#,
    r#"CREATE TRIGGER application_revision_immutable_delete BEFORE DELETE ON application_revisions FOR EACH ROW EXECUTE FUNCTION soda_reject_immutable()"#,
    r#"CREATE TRIGGER factory_run_binding_immutable BEFORE UPDATE ON factory_runs FOR EACH ROW EXECUTE FUNCTION soda_guard_run_binding()"#,
    r#"CREATE TRIGGER factory_command_immutable BEFORE UPDATE ON factory_commands FOR EACH ROW EXECUTE FUNCTION soda_guard_command()"#,
    r#"CREATE TRIGGER factory_dispatch_reg_immutable_update BEFORE UPDATE ON factory_dispatch_regs FOR EACH ROW EXECUTE FUNCTION soda_reject_immutable()"#,
    r#"CREATE TRIGGER factory_dispatch_reg_immutable_delete BEFORE DELETE ON factory_dispatch_regs FOR EACH ROW EXECUTE FUNCTION soda_reject_immutable()"#,
    r#"CREATE TRIGGER factory_takeover_immutable_update BEFORE UPDATE ON factory_takeovers FOR EACH ROW EXECUTE FUNCTION soda_reject_immutable()"#,
    r#"CREATE TRIGGER factory_takeover_immutable_delete BEFORE DELETE ON factory_takeovers FOR EACH ROW EXECUTE FUNCTION soda_reject_immutable()"#,
    r#"CREATE TRIGGER factory_plan_revision_immutable_update BEFORE UPDATE ON factory_plan_revisions FOR EACH ROW EXECUTE FUNCTION soda_reject_immutable()"#,
    r#"CREATE TRIGGER factory_plan_revision_immutable_delete BEFORE DELETE ON factory_plan_revisions FOR EACH ROW EXECUTE FUNCTION soda_reject_immutable()"#,
    r#"CREATE TRIGGER factory_run_view_immutable_update BEFORE UPDATE ON factory_run_views FOR EACH ROW EXECUTE FUNCTION soda_reject_immutable()"#,
    r#"CREATE TRIGGER factory_run_view_immutable_delete BEFORE DELETE ON factory_run_views FOR EACH ROW EXECUTE FUNCTION soda_reject_immutable()"#,
    r#"CREATE TRIGGER factory_completion_immutable_update BEFORE UPDATE ON factory_completions FOR EACH ROW EXECUTE FUNCTION soda_reject_immutable()"#,
    r#"CREATE TRIGGER factory_completion_immutable_delete BEFORE DELETE ON factory_completions FOR EACH ROW EXECUTE FUNCTION soda_reject_immutable()"#,
    r#"CREATE TRIGGER factory_reassessment_immutable_delete BEFORE DELETE ON factory_reassessments FOR EACH ROW EXECUTE FUNCTION soda_reject_immutable()"#,
    r#"CREATE TRIGGER factory_mcp_capability_no_delete BEFORE DELETE ON factory_mcp_capabilities FOR EACH ROW EXECUTE FUNCTION soda_reject_immutable()"#,
    r#"CREATE TRIGGER factory_mcp_submission_no_delete BEFORE DELETE ON factory_mcp_submissions FOR EACH ROW EXECUTE FUNCTION soda_reject_immutable()"#,
    r#"CREATE TRIGGER issue_acceptance_decision_immutable_update BEFORE UPDATE ON issue_acceptance_decisions FOR EACH ROW EXECUTE FUNCTION soda_reject_immutable()"#,
    r#"CREATE TRIGGER issue_acceptance_decision_immutable_delete BEFORE DELETE ON issue_acceptance_decisions FOR EACH ROW EXECUTE FUNCTION soda_reject_immutable()"#,
    r#"CREATE TRIGGER issue_acceptance_withdrawal_immutable_update BEFORE UPDATE ON issue_acceptance_withdrawals FOR EACH ROW EXECUTE FUNCTION soda_reject_immutable()"#,
    r#"CREATE TRIGGER issue_acceptance_withdrawal_immutable_delete BEFORE DELETE ON issue_acceptance_withdrawals FOR EACH ROW EXECUTE FUNCTION soda_reject_immutable()"#,
    r#"CREATE TRIGGER project_requirement_decision_immutable_update BEFORE UPDATE ON project_requirement_decisions FOR EACH ROW EXECUTE FUNCTION soda_reject_immutable()"#,
    r#"CREATE TRIGGER project_requirement_decision_immutable_delete BEFORE DELETE ON project_requirement_decisions FOR EACH ROW EXECUTE FUNCTION soda_reject_immutable()"#,
    r#"CREATE TRIGGER project_approval_decision_immutable_update BEFORE UPDATE ON project_approval_decisions FOR EACH ROW EXECUTE FUNCTION soda_reject_immutable()"#,
    r#"CREATE TRIGGER project_approval_decision_immutable_delete BEFORE DELETE ON project_approval_decisions FOR EACH ROW EXECUTE FUNCTION soda_reject_immutable()"#,
    r#"CREATE TRIGGER identity_events_immutable_update BEFORE UPDATE ON identity_events FOR EACH ROW EXECUTE FUNCTION soda_reject_immutable()"#,
    r#"CREATE TRIGGER identity_events_immutable_delete BEFORE DELETE ON identity_events FOR EACH ROW EXECUTE FUNCTION soda_reject_immutable()"#,
    r#"CREATE TRIGGER preparation_refs_immutable BEFORE UPDATE ON project_preparations FOR EACH ROW EXECUTE FUNCTION soda_guard_preparation_refs()"#,
    r#"CREATE TRIGGER identity_execution_immutable BEFORE UPDATE ON identity_executions FOR EACH ROW EXECUTE FUNCTION soda_guard_execution_identity()"#,
];

pub const VERIFY_QUERIES: &[&str] = &[
    r#"SELECT id,administrator_id,name,data_id,owner_team,origin,hostname,port,revision,observation FROM applications LIMIT 0"#,
    r#"SELECT application_id,revision,spec FROM application_revisions LIMIT 0"#,
    r#"SELECT repository,revision,actor_id,data FROM factory_activity_settings LIMIT 0"#,
    r#"SELECT id,login,name FROM users LIMIT 0"#,
    r#"SELECT id,user_id,public,fingerprint FROM keys LIMIT 0"#,
    r#"SELECT id,name,repository_id,owner_id,repository,ip,ready,creation_profile FROM projects LIMIT 0"#,
    r#"SELECT project_id,user_id,login FROM memberships LIMIT 0"#,
    r#"SELECT id,ciphertext FROM grant_key_check LIMIT 0"#,
    r#"SELECT project_id,revision,data FROM project_lifecycle_grants LIMIT 0"#,
    r#"SELECT project_id,revision,hold,data FROM project_maintenance LIMIT 0"#,
    r#"SELECT id,project_id,role,revision,requirements,approval,data FROM project_preparations LIMIT 0"#,
    r#"SELECT kind,execution_id,state,lease_id,data FROM identity_executions LIMIT 0"#,
    r#"SELECT seq,id,active,settled,data FROM factory_runs LIMIT 0"#,
    r#"SELECT id,type,target,principal,digest,payload,outcome,created,finished FROM factory_commands LIMIT 0"#,
    r#"SELECT repository,revision,data FROM factory_policies LIMIT 0"#,
    r#"SELECT id,revision,data FROM factory_capacity LIMIT 0"#,
    r#"SELECT repository,revision,data FROM factory_operator_grants LIMIT 0"#,
    r#"SELECT repository,connection,revision,data FROM factory_sponsorships LIMIT 0"#,
    r#"SELECT connection,revision,data FROM factory_connection_usage_budgets LIMIT 0"#,
    r#"SELECT repository,revision,open,data FROM factory_dispatch LIMIT 0"#,
    r#"SELECT seq,id,repository,revision,data FROM factory_dispatch_regs LIMIT 0"#,
    r#"SELECT seq,id,repository,issue,run,stage,revision,data FROM factory_assignments LIMIT 0"#,
    r#"SELECT seq,assignment,repository,connection,state,data FROM factory_reservations LIMIT 0"#,
    r#"SELECT run,repository,connection,minutes,started_at,ended_at,data FROM factory_usage LIMIT 0"#,
    r#"SELECT run,member,project,data FROM factory_takeovers LIMIT 0"#,
    r#"SELECT id,event,action,repository,issue,attempts,revoked,data FROM factory_lifecycle_actions LIMIT 0"#,
    r#"SELECT event,revision,installation FROM factory_lifecycle_capabilities LIMIT 0"#,
    r#"SELECT event,revision,enabled,data FROM factory_lifecycle_configs LIMIT 0"#,
    r#"SELECT intent,owner_id,revision FROM factory_intent_allowance_heads LIMIT 0"#,
    r#"SELECT id,intent,data FROM factory_intent_allowances LIMIT 0"#,
    r#"SELECT result_revision,data FROM factory_intent_reviews LIMIT 0"#,
    r#"SELECT id,repository,issue,predecessor,data FROM issue_acceptance_decisions LIMIT 0"#,
    r#"SELECT repository,issue,decision FROM issue_acceptance_heads LIMIT 0"#,
    r#"SELECT repository,issue,decision,withdrawer FROM issue_acceptance_withdrawals LIMIT 0"#,
    r#"SELECT repository,revision,data FROM project_environment_grants LIMIT 0"#,
    r#"SELECT id,project_id,predecessor,data FROM project_requirement_decisions LIMIT 0"#,
    r#"SELECT project_id,decision FROM project_requirement_heads LIMIT 0"#,
    r#"SELECT id,project_id,predecessor,data FROM project_approval_decisions LIMIT 0"#,
    r#"SELECT project_id,decision FROM project_approval_heads LIMIT 0"#,
    r#"SELECT seq,run,repository,issue,attempt FROM factory_run_views LIMIT 0"#,
    r#"SELECT seq,assignment,repository,issue,run,stage,revision,data FROM factory_publications LIMIT 0"#,
    r#"SELECT seq,publication,repository,issue,pr,stage,revision,data FROM factory_merges LIMIT 0"#,
    r#"SELECT repository,generation,event,publication,occurrence,data FROM factory_completions LIMIT 0"#,
    r#"SELECT cycle,repository,generation,event,state,data FROM factory_reassessments LIMIT 0"#,
    r#"SELECT id,generation,next_turn FROM factory_readiness_budget LIMIT 0"#,
    r#"SELECT id,delivery,repository,issue,creation_verified,creator_id,creator_write,native_issue_id,created_unix,title_digest,content_digest,generation,turn,retry_at,attempts FROM factory_readiness_sources LIMIT 0"#,
    r#"SELECT repository,task_id FROM factory_creation_cursors LIMIT 0"#,
    r#"SELECT source,repository,issue,state,cursor_repository,cursor_issue FROM factory_readiness_nodes LIMIT 0"#,
];

pub const VERIFY_TRIGGERS: &[(&str, &str)] = &[
    ("immutable_creation_profile", "projects"),
    ("preparation_refs_immutable", "project_preparations"),
    ("identity_execution_immutable", "identity_executions"),
    ("application_identity_immutable", "applications"),
    (
        "application_revision_immutable_update",
        "application_revisions",
    ),
    (
        "application_revision_immutable_delete",
        "application_revisions",
    ),
];

#[cfg(test)]
mod tests {
    use super::*;

    /// Go source owners for the drift test. `schema.go` owns the
    /// `schemaStatements` list (inline raw literals in order) and
    /// `schemaFormatVersion`; the other three files own the named
    /// raw-string DDL/trigger constants referenced bare inside that
    /// list. No other file is read; no external tool is invoked.
    const GO_OWNERS: &[&str] = &[
        "schema.go",
        "factory_completions.go",
        "factory_mcp_capability.go",
        "factory_mcp_submission.go",
    ];

    /// Directory holding the Go owners, derived from the existing
    /// canonical relative read (CARGO_MANIFEST_DIR is assigned the
    /// canonical identity crate dir; never repurposed).
    fn store_dir() -> String {
        let path = concat!(
            env!("CARGO_MANIFEST_DIR"),
            "/../../internal/store/schema.go"
        );
        let dir_end = path.rfind('/').expect("owner path has no dir");
        path[..dir_end].to_string()
    }

    fn read_owner(name: &str) -> String {
        std::fs::read_to_string(format!("{}/{}", store_dir(), name))
            .unwrap_or_else(|_| panic!("drift test: cannot read Go owner {name}"))
    }

    fn is_ident(s: &str) -> bool {
        let mut chars = s.chars();
        match chars.next() {
            Some(c) if c.is_ascii_alphabetic() || c == '_' => {}
            _ => return false,
        }
        s.chars().all(|c| c.is_ascii_alphanumeric() || c == '_')
    }

    /// Split a Go composite-literal body into top-level entries.
    /// Backtick raw-string spans are atomic (a Go raw string cannot
    /// contain a backtick); entries split on commas outside them.
    /// LIMIT: flat entries only. Nested composite literals,
    /// interpreted `"..."` strings, concatenations, or call
    /// expressions are unsupported and fail closed in `resolve_entry`.
    fn split_entries(block: &str) -> Vec<String> {
        let mut entries = Vec::new();
        let mut cur = String::new();
        let mut in_literal = false;
        for ch in block.chars() {
            if ch == '`' {
                in_literal = !in_literal;
                cur.push(ch);
            } else if ch == ',' && !in_literal {
                entries.push(std::mem::take(&mut cur));
            } else {
                cur.push(ch);
            }
        }
        if in_literal {
            panic!("drift test: unterminated Go raw literal");
        }
        entries.push(cur);
        entries
    }

    /// Look up a bare identifier as exactly one current
    /// `const NAME = ` + raw-string definition across GO_OWNERS.
    /// Zero, multiple, non-raw, or misplaced definitions fail
    /// closed instead of silently omitting the entry.
    /// LIMIT: `const NAME = ` (single spaces) starting its line, then
    /// a backtick (extra blanks allowed); parenthesized const blocks,
    /// comments between tokens, or interpreted strings are unsupported
    /// by design and fail here.
    fn find_raw_const(name: &str, owners: &[(&str, &str)]) -> String {
        let needle = format!("const {name} = ");
        let mut hits = 0;
        let mut value = String::new();
        for (file, src) in owners {
            let mut search = *src;
            while let Some(pos) = search.find(needle.as_str()) {
                let abs = src.len() - search.len() + pos;
                // Declaration must start its line (modulo indent):
                // rejects matches inside comments or other text.
                let line_start = src[..abs].rfind('\n').map_or(0, |p| p + 1);
                if !src[line_start..abs].trim().is_empty() {
                    panic!("drift test: {name} match in {file} is not a declaration");
                }
                let mut lit = &src[abs + needle.len()..];
                while lit.starts_with(' ') || lit.starts_with('\t') {
                    lit = &lit[1..];
                }
                if !lit.starts_with('`') {
                    panic!("drift test: {name} in {file} is not a raw-string const");
                }
                let close = lit[1..]
                    .find('`')
                    .unwrap_or_else(|| panic!("drift test: unterminated const {name} in {file}"));
                // NATIVE113: the closing raw literal must end its
                // declaration line (whitespace only may follow; EOL or
                // EOF permitted). Any trailing expression (`+ ...`,
                // `;`, comments, ...) is unsupported by the current
                // grammar and fails closed, never prefix-truncated.
                let after = &lit[close + 2..];
                let line_end = after.find('\n').map_or(after.len(), |p| p);
                let trailing = &after[..line_end];
                if !trailing.trim().is_empty() {
                    panic!("drift test: unsupported trailing tokens after const {name} raw literal in {file} (fail closed): {:?}", trailing);
                }
                hits += 1;
                value = lit[1..close + 1].to_string();
                search = &lit[close + 2..];
            }
        }
        if hits == 0 {
            panic!("drift test: no `const {name} = ` raw-string definition in Go owners");
        }
        if hits > 1 {
            panic!("drift test: ambiguous {name}: {hits} definitions in Go owners");
        }
        value
    }

    /// Resolve schemaStatements entries IN ORDER: an inline raw
    /// literal `` `...` `` yields its bytes; a bare identifier yields
    /// its current named-const bytes. Empty slots (trailing comma)
    /// are skipped; comments or any other form fail closed.
    fn go_statements(schema_src: &str, owners: &[(&str, &str)]) -> Vec<String> {
        let start_marker = "var schemaStatements = []string{";
        let end_marker = "\n}\n\n// SchemaVersion";
        let start = schema_src
            .find(start_marker)
            .expect("schemaStatements missing")
            + start_marker.len();
        let end = schema_src[start..]
            .find(end_marker)
            .expect("schemaStatements unterminated")
            + start;
        let mut out = Vec::new();
        for entry in split_entries(&schema_src[start..end]) {
            let e = entry.trim();
            if e.is_empty() {
                continue;
            }
            if e.starts_with("//") {
                panic!("drift test: unsupported comment entry (fail closed): {e:?}");
            }
            if e.starts_with('`') {
                if !(e.ends_with('`') && e.len() >= 2) {
                    panic!("drift test: malformed inline literal: {e:?}");
                }
                out.push(e[1..e.len() - 1].to_string());
            } else if is_ident(e) {
                out.push(find_raw_const(e, owners));
            } else {
                panic!("drift test: unsupported schemaStatements entry (fail closed): {e:?}");
            }
        }
        out
    }

    /// Verify-query block: inline raw literals only, in order. A bare
    /// identifier here fails closed (named-query support needs test
    /// maintenance, never silent omission).
    fn go_queries(schema_src: &str) -> Vec<String> {
        let start_marker = "for _, query := range []string{";
        let end_marker = "} {\n\t\trows, err";
        let start = schema_src
            .find(start_marker)
            .expect("verify query block missing")
            + start_marker.len();
        let end = schema_src[start..]
            .find(end_marker)
            .expect("verify query block unterminated")
            + start;
        let mut out = Vec::new();
        for entry in split_entries(&schema_src[start..end]) {
            let e = entry.trim();
            if e.is_empty() {
                continue;
            }
            if e.starts_with('`') && e.ends_with('`') && e.len() >= 2 {
                out.push(e[1..e.len() - 1].to_string());
            } else {
                panic!("drift test: unsupported verify-query entry (fail closed): {e:?}");
            }
        }
        out
    }

    /// Current `const schemaFormatVersion = N` from schema.go.
    fn go_schema_version(schema_src: &str) -> i64 {
        let needle = "const schemaFormatVersion = ";
        let pos = schema_src
            .find(needle)
            .expect("schemaFormatVersion missing");
        let mut digits = &schema_src[pos + needle.len()..];
        let mut len = 0;
        while digits[len..].starts_with(|c: char| c.is_ascii_digit()) {
            len += 1;
        }
        digits = &digits[..len];
        if digits.is_empty() {
            panic!("drift test: schemaFormatVersion has no digits");
        }
        // NATIVE113: the version digits must end their line
        // (whitespace only may follow; EOL or EOF permitted). A
        // trailing expression (`+ 1`, ...) is unsupported by the
        // current grammar and fails closed, never prefix-truncated.
        let rest = &schema_src[pos + needle.len() + len..];
        let rest_end = rest.find('\n').map_or(rest.len(), |p| p);
        let trailing = &rest[..rest_end];
        if !trailing.trim().is_empty() {
            panic!("drift test: unsupported trailing tokens after schemaFormatVersion digits (fail closed): {:?}", trailing);
        }
        digits.parse().expect("schemaFormatVersion unparsable")
    }

    #[test]
    fn schema_matches_go_source() {
        let schema_src = read_owner("schema.go");
        let owned: Vec<(String, String)> = GO_OWNERS
            .iter()
            .map(|n| ((*n).to_string(), read_owner(n)))
            .collect();
        let refs: Vec<(&str, &str)> = owned
            .iter()
            .map(|(n, s)| (n.as_str(), s.as_str()))
            .collect();
        let stmts = go_statements(&schema_src, &refs);
        assert_eq!(stmts.len(), STATEMENTS.len(), "statement count drifted");
        for (i, (go, rust)) in stmts.iter().zip(STATEMENTS.iter()).enumerate() {
            assert_eq!(go, rust, "statement {i} drifted");
        }
        let queries = go_queries(&schema_src);
        assert_eq!(queries.len(), VERIFY_QUERIES.len(), "verify count drifted");
        for (i, (go, rust)) in queries.iter().zip(VERIFY_QUERIES.iter()).enumerate() {
            assert_eq!(go, rust, "verify query {i} drifted");
        }
        // Seed INSERT is compared inside STATEMENTS above; the format
        // version is checked separately here.
        let go_version = go_schema_version(&schema_src);
        assert_eq!(go_version, SCHEMA_VERSION, "schema version drifted");
        println!(
            "drift check: go_statements={} rust_statements={} go_queries={} rust_queries={} go_version={} rust_version={}",
            stmts.len(),
            STATEMENTS.len(),
            queries.len(),
            VERIFY_QUERIES.len(),
            go_version,
            SCHEMA_VERSION,
        );
    }
}

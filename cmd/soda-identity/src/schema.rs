// Embedded PostgreSQL schema, mechanically copied from
// internal/store/schema.go (the single source of truth). The
// drift test below re-extracts the Go literals at test time and
// fails on any difference, so a Go schema change forces a
// regeneration of this file in the same patch.
pub const SCHEMA_VERSION: i64 = 28;

pub const STATEMENTS: &[&str] = &[
    r#"CREATE TABLE schema_version(version INTEGER PRIMARY KEY)"#,
    r#"INSERT INTO schema_version(version) VALUES(28)"#,
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
    r#"CREATE INDEX factory_unsettled_runs ON factory_runs(settled) WHERE settled=FALSE"#,
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
    r#"CREATE TABLE factory_run_views(seq BIGINT GENERATED ALWAYS AS IDENTITY, run TEXT PRIMARY KEY REFERENCES factory_runs(id), repository INTEGER NOT NULL CHECK(repository>0), issue INTEGER NOT NULL DEFAULT 0 CHECK(issue>=0), attempt TEXT NOT NULL DEFAULT '' CHECK(char_length(attempt)<=64))"#,
    r#"CREATE TABLE factory_assignments(seq BIGINT GENERATED ALWAYS AS IDENTITY, id TEXT PRIMARY KEY, repository INTEGER NOT NULL CHECK(repository>0), issue INTEGER NOT NULL CHECK(issue>0), run TEXT NOT NULL, stage TEXT NOT NULL, revision INTEGER NOT NULL, data JSONB NOT NULL)"#,
    r#"CREATE UNIQUE INDEX factory_unfinished_assignment ON factory_assignments(repository,issue) WHERE stage='assigned'"#,
    r#"CREATE INDEX factory_assignments_issue_history ON factory_assignments(repository,issue,seq DESC)"#,
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
    r#"CREATE TABLE factory_readiness_sweeps(repository INTEGER PRIMARY KEY CHECK(repository>0), revision INTEGER NOT NULL CHECK(revision>0), swept_at INTEGER NOT NULL)"#,
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
    r#"CREATE TRIGGER factory_run_binding_immutable BEFORE UPDATE ON factory_runs FOR EACH ROW EXECUTE FUNCTION soda_guard_run_binding()"#,
    r#"CREATE TRIGGER factory_command_immutable BEFORE UPDATE ON factory_commands FOR EACH ROW EXECUTE FUNCTION soda_guard_command()"#,
    r#"CREATE TRIGGER factory_dispatch_reg_immutable_update BEFORE UPDATE ON factory_dispatch_regs FOR EACH ROW EXECUTE FUNCTION soda_reject_immutable()"#,
    r#"CREATE TRIGGER factory_dispatch_reg_immutable_delete BEFORE DELETE ON factory_dispatch_regs FOR EACH ROW EXECUTE FUNCTION soda_reject_immutable()"#,
    r#"CREATE TRIGGER factory_takeover_immutable_update BEFORE UPDATE ON factory_takeovers FOR EACH ROW EXECUTE FUNCTION soda_reject_immutable()"#,
    r#"CREATE TRIGGER factory_takeover_immutable_delete BEFORE DELETE ON factory_takeovers FOR EACH ROW EXECUTE FUNCTION soda_reject_immutable()"#,
    r#"CREATE TRIGGER factory_run_view_immutable_update BEFORE UPDATE ON factory_run_views FOR EACH ROW EXECUTE FUNCTION soda_reject_immutable()"#,
    r#"CREATE TRIGGER factory_run_view_immutable_delete BEFORE DELETE ON factory_run_views FOR EACH ROW EXECUTE FUNCTION soda_reject_immutable()"#,
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
];

pub const VERIFY_TRIGGERS: &[(&str, &str)] = &[
    ("immutable_creation_profile", "projects"),
    ("preparation_refs_immutable", "project_preparations"),
    ("identity_execution_immutable", "identity_executions"),
];

#[cfg(test)]
mod tests {
    use super::*;

    fn go_literals(section_start: &str, section_end: &str) -> Vec<String> {
        let path = concat!(
            env!("CARGO_MANIFEST_DIR"),
            "/../../internal/store/schema.go"
        );
        let src = std::fs::read_to_string(path).unwrap();
        let start = src.find(section_start).unwrap();
        let end = src[start..].find(section_end).unwrap() + start;
        let block = &src[start..end];
        let mut out = Vec::new();
        let mut rest = block;
        while let Some(open) = rest.find('`') {
            let after = &rest[open + 1..];
            let close = after.find('`').unwrap();
            out.push(after[..close].to_string());
            rest = &after[close + 1..];
        }
        out
    }

    #[test]
    fn schema_matches_go_source() {
        let stmts = go_literals(
            "var schemaStatements = []string{",
            "\n}\n\n// SchemaVersion",
        );
        assert_eq!(stmts.len(), STATEMENTS.len(), "statement count drifted");
        for (i, (go, rust)) in stmts.iter().zip(STATEMENTS.iter()).enumerate() {
            assert_eq!(go, rust, "statement {i} drifted");
        }
        let queries = go_literals("for _, query := range []string{", "} {\n\t\trows, err");
        assert_eq!(queries.len(), VERIFY_QUERIES.len(), "verify count drifted");
        for (i, (go, rust)) in queries.iter().zip(VERIFY_QUERIES.iter()).enumerate() {
            assert_eq!(go, rust, "verify query {i} drifted");
        }
    }
}

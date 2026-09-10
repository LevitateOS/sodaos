-- Historical v7 schema, preserved from migrations 1–7 at 6e3b6e0.
-- Deliberately independent of the production migration slice.
CREATE TABLE schema_version(version INTEGER PRIMARY KEY);
INSERT INTO schema_version VALUES(7);
CREATE TABLE users(id INTEGER PRIMARY KEY CHECK(id>0), login TEXT NOT NULL, name TEXT NOT NULL DEFAULT '');
CREATE TABLE keys(id INTEGER PRIMARY KEY, user_id INTEGER NOT NULL REFERENCES users(id), public TEXT NOT NULL, fingerprint TEXT NOT NULL, UNIQUE(user_id,fingerprint));
CREATE TABLE projects(id TEXT PRIMARY KEY, name TEXT NOT NULL, repository_id INTEGER NOT NULL UNIQUE, owner_id INTEGER NOT NULL REFERENCES users(id), repository TEXT NOT NULL, ip TEXT NOT NULL DEFAULT '', ready INTEGER NOT NULL DEFAULT 0 CHECK(ready IN(0,1)));
CREATE TABLE memberships(project_id TEXT NOT NULL REFERENCES projects(id), user_id INTEGER NOT NULL REFERENCES users(id), login TEXT NOT NULL, PRIMARY KEY(project_id,user_id), UNIQUE(project_id,login));
CREATE TABLE sessions(token TEXT PRIMARY KEY, user_id INTEGER NOT NULL REFERENCES users(id), csrf TEXT NOT NULL, expires INTEGER NOT NULL);
CREATE TABLE oauth(state TEXT PRIMARY KEY, verifier TEXT NOT NULL, expires INTEGER NOT NULL);
ALTER TABLE oauth ADD COLUMN return_path TEXT NOT NULL DEFAULT '/projects' CHECK(return_path IN ('/projects','/app/'));
CREATE TABLE grant_key_check(id INTEGER PRIMARY KEY CHECK(id=1), ciphertext BLOB NOT NULL);
CREATE TABLE session_grants(session_token TEXT PRIMARY KEY REFERENCES sessions(token) ON DELETE CASCADE, ciphertext BLOB NOT NULL);
ALTER TABLE oauth ADD COLUMN repository_id INTEGER NOT NULL DEFAULT 0 CHECK(repository_id>=0);
ALTER TABLE oauth ADD COLUMN expected_user_id INTEGER NOT NULL DEFAULT 0 CHECK(expected_user_id>=0);
CREATE TABLE login_contexts(id TEXT PRIMARY KEY, pending TEXT UNIQUE, expires INTEGER NOT NULL);
ALTER TABLE sessions ADD COLUMN context_id TEXT REFERENCES login_contexts(id) ON DELETE CASCADE;
INSERT INTO login_contexts(id,expires) SELECT token,expires FROM sessions;
UPDATE sessions SET context_id=token;
CREATE UNIQUE INDEX sessions_context ON sessions(context_id);
ALTER TABLE oauth ADD COLUMN context_id TEXT REFERENCES login_contexts(id) ON DELETE CASCADE;
ALTER TABLE oauth ADD COLUMN spaces_return INTEGER NOT NULL DEFAULT 0 CHECK(spaces_return IN(0,1) AND (spaces_return=0 OR repository_id=0));
ALTER TABLE oauth ADD COLUMN settings_return TEXT NOT NULL DEFAULT '' CHECK(settings_return IN ('','runners') AND (settings_return='' OR (spaces_return=0 AND repository_id=0)));

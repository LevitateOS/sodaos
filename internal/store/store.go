// Package store owns the appliance PostgreSQL schema and row operations. It is not
// an HTTP API, host runtime or generic SQL escape hatch for other packages.
package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strconv"
	"strings"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/levitateos/sodaos/internal/project"
)

// ErrNotFound is returned when a looked-up row is absent. Prefer this sentinel
// over importing database/sql solely for sql.ErrNoRows.
var ErrNotFound = sql.ErrNoRows

// ErrCommandConflict reports a reused command identity with changed content.
var ErrCommandConflict = errors.New("command identity reused for different content")

type Store struct {
	db     *sql.DB
	grants *grantCipher
}
type User struct {
	ID          int64
	Login, Name string
}
type Key struct {
	ID                  int64
	Public, Fingerprint string
}
type Project struct {
	Profile               *project.Profile
	ID, Name              string
	RepositoryID, OwnerID int64
	Repository, IP        string
	Ready                 bool
}
type Session struct {
	User User
	CSRF string
	// ContextID binds a native request to its current Forgejo session.
	ContextID string
	Expires   int64
}

// bind rewrites ? placeholders to PostgreSQL $n parameters, skipping
// single-quoted literals. Every store query uses ? so call sites stay
// readable; the database sees only $n.
func bind(query string) string {
	var out strings.Builder
	out.Grow(len(query) + 8)
	n := 0
	inString := false
	for i := 0; i < len(query); i++ {
		c := query[i]
		if c == '\'' {
			// '' inside a literal is an escaped quote, not a terminator.
			if inString && i+1 < len(query) && query[i+1] == '\'' {
				out.WriteString("''")
				i++
				continue
			}
			inString = !inString
			out.WriteByte(c)
			continue
		}
		if c == '?' && !inString {
			n++
			out.WriteByte('$')
			out.WriteString(strconv.Itoa(n))
			continue
		}
		out.WriteByte(c)
	}
	return out.String()
}

func (s *Store) exec(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return s.db.ExecContext(ctx, bind(query), args...)
}

func (s *Store) query(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	return s.db.QueryContext(ctx, bind(query), args...)
}

func (s *Store) queryRow(ctx context.Context, query string, args ...any) *sql.Row {
	return s.db.QueryRowContext(ctx, bind(query), args...)
}

// tx is a rebinding transaction: its methods rewrite ? exactly like the
// Store methods. It deliberately does not embed *sql.Tx, so a call site
// that bypasses the binding fails to compile.
type tx struct{ inner *sql.Tx }

func (s *Store) begin(ctx context.Context) (*tx, error) {
	inner, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	return &tx{inner: inner}, nil
}

func (t *tx) Commit() error { return t.inner.Commit() }

func (t *tx) Rollback() error { return t.inner.Rollback() }

func (t *tx) exec(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return t.inner.ExecContext(ctx, bind(query), args...)
}

func (t *tx) query(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	return t.inner.QueryContext(ctx, bind(query), args...)
}

func (t *tx) queryRow(ctx context.Context, query string, args ...any) *sql.Row {
	return t.inner.QueryRowContext(ctx, bind(query), args...)
}

// sameJSONDocument reports whether two JSON documents carry the same
// content. JSONB storage normalizes key order, so byte comparison cannot
// tell an identical replay from a conflicting rewrite.
func sameJSONDocument(ctx context.Context, t *tx, stored, fresh []byte) (bool, error) {
	var same bool
	err := t.queryRow(ctx, `SELECT ?::jsonb = ?::jsonb`, string(stored), string(fresh)).Scan(&same)
	return same, err
}

// Open connects to the appliance PostgreSQL store, creating the current
// schema in an empty database and refusing anything else. dsn is a
// PostgreSQL connection URL; callers read it from a restricted secret
// file, never from configuration text.
func Open(dsn string) (*Store, error) { return open(dsn, nil) }

// OpenEncrypted is the production entrypoint. The operator-managed key protects
// native identity credentials and must be validated before schema initialization.
func OpenEncrypted(dsn string, key []byte) (*Store, error) {
	cipher, err := newGrantCipher(key)
	if err != nil {
		return nil, err
	}
	return open(dsn, cipher)
}

func configureStore(db *sql.DB, cipher *grantCipher) (*Store, error) {
	// Concurrent dispatch passes admit through separate connections;
	// the pool stays small because the appliance serves one backend.
	db.SetMaxOpenConns(16)
	db.SetMaxIdleConns(4)
	s := &Store{db: db, grants: cipher}
	err := s.checkGrantKey(context.Background())
	if err == nil {
		err = initializeSchema(context.Background(), db)
	}
	if err == nil && cipher != nil {
		err = s.initializeGrantKey(context.Background())
	}
	if err != nil {
		db.Close()
		return nil, err
	}
	return s, nil
}

func open(dsn string, cipher *grantCipher) (*Store, error) {
	if strings.TrimSpace(dsn) == "" {
		return nil, errors.New("postgres connection string is required")
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}
	return configureStore(db, cipher)
}
func (s *Store) Close() error { return s.db.Close() }

func (s *Store) UpsertUser(ctx context.Context, u User) error {
	if u.ID <= 0 || strings.TrimSpace(u.Login) == "" {
		return errors.New("invalid provider user")
	}
	_, err := s.exec(ctx, `INSERT INTO users(id,login,name) VALUES(?,?,?) ON CONFLICT(id) DO UPDATE SET login=excluded.login`, u.ID, u.Login, u.Name)
	return err
}

func (s *Store) User(ctx context.Context, id int64) (User, error) {
	var u User
	err := s.queryRow(ctx, `SELECT id,login,name FROM users WHERE id=?`, id).Scan(&u.ID, &u.Login, &u.Name)
	return u, err
}

func (s *Store) RenameProfile(ctx context.Context, id int64, name string) error {
	if len(name) > 200 {
		return errors.New("name too long")
	}
	_, err := s.exec(ctx, `UPDATE users SET name=? WHERE id=?`, name, id)
	return err
}

func (s *Store) AddKey(ctx context.Context, uid int64, public, fingerprint string) error {
	_, err := s.exec(ctx, `INSERT INTO keys(user_id,public,fingerprint) VALUES(?,?,?) ON CONFLICT(user_id,fingerprint) DO NOTHING`, uid, public, fingerprint)
	return err
}

func (s *Store) RemoveKey(ctx context.Context, uid, id int64) (bool, error) {
	result, err := s.exec(ctx, `DELETE FROM keys WHERE user_id=? AND id=?`, uid, id)
	if err != nil {
		return false, err
	}
	n, err := result.RowsAffected()
	return n == 1, err
}

func (s *Store) Keys(ctx context.Context, uid int64) ([]Key, error) {
	rows, err := s.query(ctx, `SELECT id,public,fingerprint FROM keys WHERE user_id=? ORDER BY id`, uid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Key{}
	for rows.Next() {
		var k Key
		if err = rows.Scan(&k.ID, &k.Public, &k.Fingerprint); err != nil {
			return nil, err
		}
		out = append(out, k)
	}
	return out, rows.Err()
}

func (s *Store) CreateProject(ctx context.Context, p Project) error {
	if p.Profile == nil {
		_, err := s.exec(ctx, `INSERT INTO projects(id,name,repository_id,owner_id,repository) VALUES(?,?,?,?,?)`, p.ID, p.Name, p.RepositoryID, p.OwnerID, p.Repository)
		return err
	}
	if err := p.Profile.Validate(); err != nil {
		return err
	}
	raw, err := json.Marshal(p.Profile)
	if err != nil {
		return err
	}
	_, err = s.exec(ctx, `INSERT INTO projects(id,name,repository_id,owner_id,repository,creation_profile) VALUES(?,?,?,?,?,?)`, p.ID, p.Name, p.RepositoryID, p.OwnerID, p.Repository, string(raw))
	return err
}

func (s *Store) MarkReady(ctx context.Context, id, ip string) error {
	_, err := s.exec(ctx, `UPDATE projects SET ip=?,ready=TRUE WHERE id=?`, ip, id)
	return err
}

func scanProject(row interface{ Scan(...any) error }) (Project, error) {
	var p Project
	var profile sql.NullString
	err := row.Scan(&p.ID, &p.Name, &p.RepositoryID, &p.OwnerID, &p.Repository, &p.IP, &p.Ready, &profile)
	if err == nil && profile.Valid {
		p.Profile, err = project.Decode(profile.String)
	}
	return p, err
}

const projectColumns = `id,name,repository_id,owner_id,repository,ip,ready,creation_profile`

func (s *Store) Project(ctx context.Context, id string) (Project, error) {
	return scanProject(s.queryRow(ctx, `SELECT `+projectColumns+` FROM projects WHERE id=?`, id))
}

// SpaceProjects is a bounded scan of Soda associations, not an authorized catalog.
// NULL deliberately rejects oversized stored labels instead of silently truncating
// them or allocating arbitrary DB text. Callers must authorize every returned row.
func (s *Store) SpaceProjects(ctx context.Context) ([]Project, error) {
	rows, err := s.query(ctx, `SELECT CASE WHEN octet_length(id)<=128 THEN id END,
 CASE WHEN octet_length(name)<=1024 THEN name END,repository_id,owner_id,
 CASE WHEN octet_length(repository)<=2048 THEN repository END,
 CASE WHEN octet_length(ip)<=128 THEN ip END,ready,creation_profile FROM projects ORDER BY id LIMIT 129`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Project{}
	for rows.Next() {
		p, err := scanProject(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (s *Store) ProjectByRepository(ctx context.Context, id int64) (Project, error) {
	return scanProject(s.queryRow(ctx, `SELECT `+projectColumns+` FROM projects WHERE repository_id=?`, id))
}

func (s *Store) Join(ctx context.Context, pid string, uid int64, login string) error {
	_, err := s.exec(ctx, `INSERT INTO memberships(project_id,user_id,login) VALUES(?,?,?)`, pid, uid, login)
	return err
}

func (s *Store) MemberLogin(ctx context.Context, pid string, uid int64) (string, error) {
	var login string
	err := s.queryRow(ctx, `SELECT login FROM memberships WHERE project_id=? AND user_id=?`, pid, uid).Scan(&login)
	return login, err
}

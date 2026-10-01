// Package store owns the appliance SQLite schema and row operations. It is not
// an HTTP API, host runtime or generic SQL escape hatch for other packages.
package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/levitateos/sodaos/internal/project"
	_ "modernc.org/sqlite"
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

func Open(path string) (*Store, error) { return open(path, nil) }

// OpenEncrypted is the production entrypoint. The operator-managed key protects
// native identity credentials and must be validated before schema initialization.
func OpenEncrypted(path string, key []byte) (*Store, error) {
	cipher, err := newGrantCipher(key)
	if err != nil {
		return nil, err
	}
	return open(path, cipher)
}

func configureStore(db *sql.DB, cipher *grantCipher) (*Store, error) {
	db.SetMaxOpenConns(1)
	s := &Store{db: db, grants: cipher}
	err := s.checkGrantKey(context.Background())
	if err == nil {
		err = initializeSchema(context.Background(), db)
	}
	if err == nil && cipher != nil {
		err = s.initializeGrantKey(context.Background())
	}
	if err == nil {
		_, err = db.Exec(`PRAGMA journal_mode=WAL;`)
	}
	if err != nil {
		db.Close()
		return nil, err
	}
	return s, nil
}

func open(path string, cipher *grantCipher) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, err
	}
	f.Close()
	absolute, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	// database/sql can replace connections after cancellation. Configure every
	// connection so foreign key enforcement remains active.
	dsn := url.URL{Scheme: "file", Path: absolute, RawQuery: url.Values{"_pragma": {"foreign_keys(1)", "busy_timeout(5000)"}}.Encode()}
	db, err := sql.Open("sqlite", dsn.String())
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
	_, err := s.db.ExecContext(ctx, `INSERT INTO users(id,login,name) VALUES(?,?,?) ON CONFLICT(id) DO UPDATE SET login=excluded.login`, u.ID, u.Login, u.Name)
	return err
}

func (s *Store) User(ctx context.Context, id int64) (User, error) {
	var u User
	err := s.db.QueryRowContext(ctx, `SELECT id,login,name FROM users WHERE id=?`, id).Scan(&u.ID, &u.Login, &u.Name)
	return u, err
}

func (s *Store) RenameProfile(ctx context.Context, id int64, name string) error {
	if len(name) > 200 {
		return errors.New("name too long")
	}
	_, err := s.db.ExecContext(ctx, `UPDATE users SET name=? WHERE id=?`, name, id)
	return err
}

func (s *Store) AddKey(ctx context.Context, uid int64, public, fingerprint string) error {
	_, err := s.db.ExecContext(ctx, `INSERT OR IGNORE INTO keys(user_id,public,fingerprint) VALUES(?,?,?)`, uid, public, fingerprint)
	return err
}

func (s *Store) RemoveKey(ctx context.Context, uid, id int64) (bool, error) {
	result, err := s.db.ExecContext(ctx, `DELETE FROM keys WHERE user_id=? AND id=?`, uid, id)
	if err != nil {
		return false, err
	}
	n, err := result.RowsAffected()
	return n == 1, err
}

func (s *Store) Keys(ctx context.Context, uid int64) ([]Key, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id,public,fingerprint FROM keys WHERE user_id=? ORDER BY id`, uid)
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
		_, err := s.db.ExecContext(ctx, `INSERT INTO projects(id,name,repository_id,owner_id,repository) VALUES(?,?,?,?,?)`, p.ID, p.Name, p.RepositoryID, p.OwnerID, p.Repository)
		return err
	}
	if err := p.Profile.Validate(); err != nil {
		return err
	}
	raw, err := json.Marshal(p.Profile)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, `INSERT INTO projects(id,name,repository_id,owner_id,repository,creation_profile) VALUES(?,?,?,?,?,?)`, p.ID, p.Name, p.RepositoryID, p.OwnerID, p.Repository, string(raw))
	return err
}

func (s *Store) MarkReady(ctx context.Context, id, ip string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE projects SET ip=?,ready=1 WHERE id=?`, ip, id)
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
	return scanProject(s.db.QueryRowContext(ctx, `SELECT `+projectColumns+` FROM projects WHERE id=?`, id))
}

// SpaceProjects is a bounded scan of Soda associations, not an authorized catalog.
// NULL deliberately rejects oversized stored labels instead of silently truncating
// them or allocating arbitrary DB text. Callers must authorize every returned row.
func (s *Store) SpaceProjects(ctx context.Context) ([]Project, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT CASE WHEN length(CAST(id AS BLOB))<=128 THEN id END,
 CASE WHEN length(CAST(name AS BLOB))<=1024 THEN name END,repository_id,owner_id,
 CASE WHEN length(CAST(repository AS BLOB))<=2048 THEN repository END,
 CASE WHEN length(CAST(ip AS BLOB))<=128 THEN ip END,ready,creation_profile FROM projects ORDER BY id LIMIT 129`)
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
	return scanProject(s.db.QueryRowContext(ctx, `SELECT `+projectColumns+` FROM projects WHERE repository_id=?`, id))
}

func (s *Store) Join(ctx context.Context, pid string, uid int64, login string) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO memberships(project_id,user_id,login) VALUES(?,?,?)`, pid, uid, login)
	return err
}

func (s *Store) MemberLogin(ctx context.Context, pid string, uid int64) (string, error) {
	var login string
	err := s.db.QueryRowContext(ctx, `SELECT login FROM memberships WHERE project_id=? AND user_id=?`, pid, uid).Scan(&login)
	return login, err
}

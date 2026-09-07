package store

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"errors"
	"strconv"
	"time"
)

var ErrGrantUnavailable = errors.New("provider grant unavailable; reauthentication required")
var ErrGrantKey = errors.New("provider grant encryption key missing or incorrect")

type Grant struct {
	Access  string `json:"access"`
	Refresh string `json:"refresh"`
	Scopes  string `json:"scopes"`
	Expires int64  `json:"expires"`
}

type grantCipher struct{ cipher.AEAD }

func newGrantCipher(key []byte) (*grantCipher, error) {
	if len(key) != 32 {
		return nil, ErrGrantKey
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, ErrGrantKey
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, ErrGrantKey
	}
	return &grantCipher{aead}, nil
}
func (c *grantCipher) seal(data []byte, binding string) []byte {
	nonce := make([]byte, c.NonceSize())
	rand.Read(nonce)
	return c.Seal(nonce, nonce, data, []byte(binding))
}
func (c *grantCipher) open(data []byte, binding string) ([]byte, error) {
	if c == nil || len(data) < c.NonceSize() {
		return nil, ErrGrantKey
	}
	plain, err := c.Open(nil, data[:c.NonceSize()], data[c.NonceSize():], []byte(binding))
	if err != nil {
		return nil, ErrGrantKey
	}
	return plain, nil
}

const keyBinding = "soda/session-grants/key-check/v1"

// Check an existing encrypted database before applying any schema changes.
func (s *Store) checkGrantKey(ctx context.Context) error {
	var exists int
	if err := s.db.QueryRowContext(ctx, `SELECT count(*) FROM sqlite_master WHERE type='table' AND name='grant_key_check'`).Scan(&exists); err != nil {
		return err
	}
	if exists == 0 {
		return nil
	}
	var ciphertext []byte
	err := s.db.QueryRowContext(ctx, `SELECT ciphertext FROM grant_key_check WHERE id=1`).Scan(&ciphertext)
	if errors.Is(err, sql.ErrNoRows) {
		var grants int
		if err = s.db.QueryRowContext(ctx, `SELECT count(*) FROM session_grants`).Scan(&grants); err != nil {
			return err
		}
		if grants != 0 {
			return ErrGrantKey
		}
		return nil
	}
	if err != nil {
		return err
	}
	plain, err := s.grants.open(ciphertext, keyBinding)
	if err != nil || string(plain) != keyBinding {
		return ErrGrantKey
	}
	return nil
}
func (s *Store) initializeGrantKey(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, `INSERT OR IGNORE INTO grant_key_check(id,ciphertext) VALUES(1,?)`, s.grants.seal([]byte(keyBinding), keyBinding))
	if err != nil {
		return err
	}
	return s.checkGrantKey(ctx)
}
func grantBinding(session string, uid int64) string {
	return "soda/session-grants/v1/" + hash(session) + "/" + strconv.FormatInt(uid, 10)
}

// CreateGrantedSession makes the session and encrypted grant visible together.
func (s *Store) CreateGrantedSession(ctx context.Context, session string, uid int64, csrf string, grant Grant) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, `INSERT INTO login_contexts(id,expires) VALUES(?,?)`, hash(session), time.Now().Add(12*time.Hour).Unix()); err != nil {
		return err
	}
	if err = s.insertGrantedSession(ctx, tx, hash(session), session, uid, csrf, grant); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) insertGrantedSession(ctx context.Context, tx *sql.Tx, contextID, session string, uid int64, csrf string, grant Grant) error {
	if s.grants == nil {
		return ErrGrantKey
	}
	if grant.Access == "" || grant.Refresh == "" || grant.Expires <= time.Now().Unix() {
		return ErrGrantUnavailable
	}
	plain, err := json.Marshal(grant)
	if err != nil {
		return ErrGrantUnavailable
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO sessions(token,user_id,csrf,expires,context_id) VALUES(?,?,?,?,?)`, hash(session), uid, csrf, time.Now().Add(12*time.Hour).Unix(), contextID); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO session_grants(session_token,ciphertext) VALUES(?,?)`, hash(session), s.grants.seal(plain, grantBinding(session, uid))); err != nil {
		return err
	}
	return nil
}
func (s *Store) Grant(ctx context.Context, session string, uid int64) (Grant, error) {
	var grant Grant
	var ciphertext []byte
	err := s.db.QueryRowContext(ctx, `SELECT g.ciphertext FROM session_grants g JOIN sessions s ON s.token=g.session_token WHERE s.token=? AND s.user_id=? AND s.expires>?`, hash(session), uid, time.Now().Unix()).Scan(&ciphertext)
	if errors.Is(err, sql.ErrNoRows) {
		return grant, ErrGrantUnavailable
	}
	if err != nil {
		return grant, err
	}
	plain, err := s.grants.open(ciphertext, grantBinding(session, uid))
	if err != nil {
		return grant, err
	}
	if json.Unmarshal(plain, &grant) != nil || grant.Access == "" || grant.Refresh == "" {
		return Grant{}, ErrGrantUnavailable
	}
	return grant, nil
}

// Update only: deletion on logout must win over an in-flight refresh. Never
// upsert a session/grant or resurrect it from an identity row.
func (s *Store) ReplaceGrant(ctx context.Context, session string, uid int64, grant Grant) error {
	if s.grants == nil {
		return ErrGrantKey
	}
	plain, err := json.Marshal(grant)
	if err != nil {
		return ErrGrantUnavailable
	}
	result, err := s.db.ExecContext(ctx, `UPDATE session_grants SET ciphertext=? WHERE session_token=? AND EXISTS(SELECT 1 FROM sessions WHERE token=? AND user_id=? AND expires>?)`, s.grants.seal(plain, grantBinding(session, uid)), hash(session), hash(session), uid, time.Now().Unix())
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows != 1 {
		return ErrGrantUnavailable
	}
	return nil
}
func (s *Store) DeleteGrant(ctx context.Context, session string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM session_grants WHERE session_token=?`, hash(session))
	return err
}

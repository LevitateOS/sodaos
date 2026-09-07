package store

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"
)

var ErrLoginContext = errors.New("sign-in cancelled, expired or superseded; start again")

// OAuthAttempt is a consumed, non-replayable provider exchange. Its cancellation
// marker survives consumption, but never supplies user authority.
type OAuthAttempt struct {
	OAuthLogin
	contextID, state string
	expires          int64
}

func (s *Store) BeginOAuth(ctx context.Context, state string, login OAuthLogin, session, previous string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	now, expires := time.Now().Unix(), time.Now().Add(10*time.Minute).Unix()
	// Expired contexts cannot authorize a callback; bounded lifetime, no worker.
	if _, err = tx.ExecContext(ctx, `DELETE FROM login_contexts WHERE expires<=?`, now); err != nil {
		return err
	}
	var id string
	switch {
	case session != "":
		err = tx.QueryRowContext(ctx, `SELECT context_id FROM sessions WHERE token=? AND expires>?`, hash(session), now).Scan(&id)
	case previous != "":
		// pending remains available even after the earlier callback claimed its state.
		err = tx.QueryRowContext(ctx, `SELECT id FROM login_contexts WHERE pending=? AND expires>?`, hash(previous), now).Scan(&id)
	default:
		id = hash(state)
		_, err = tx.ExecContext(ctx, `INSERT INTO login_contexts(id,expires) VALUES(?,?)`, id, expires)
	}
	if errors.Is(err, sql.ErrNoRows) {
		return ErrLoginContext
	}
	if err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `DELETE FROM oauth WHERE context_id=?`, id); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `UPDATE login_contexts SET pending=? WHERE id=?`, hash(state), id); err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO oauth(state,verifier,expires,repository_id,expected_user_id,context_id) VALUES(?,?,?,?,?,?)`, hash(state), login.Verifier, expires, login.RepositoryID, login.ExpectedUserID, id)
	if err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) ConsumeOAuth(ctx context.Context, state, session string) (OAuthAttempt, error) {
	var a OAuthAttempt
	// Legacy, unbound pending logins require a fresh start. Do not upgrade them to
	// anonymous authority. DELETE remains the single-use claim before provider I/O.
	err := s.db.QueryRowContext(ctx, `DELETE FROM oauth WHERE state=? AND expires>? AND context_id IN (SELECT id FROM login_contexts WHERE pending=? AND expires>?) RETURNING verifier,repository_id,expected_user_id,context_id,expires`, hash(state), time.Now().Unix(), hash(state), time.Now().Unix()).Scan(&a.Verifier, &a.RepositoryID, &a.ExpectedUserID, &a.contextID, &a.expires)
	if err != nil {
		return a, err
	}
	a.state = hash(state)
	// A callback cannot rotate a different browser's currently supplied session.
	var current string
	if session != "" {
		err = s.db.QueryRowContext(ctx, `SELECT context_id FROM sessions WHERE token=? AND expires>?`, hash(session), time.Now().Unix()).Scan(&current)
		if err == nil && current == a.contextID {
			return a, nil
		}
	} else {
		err = s.db.QueryRowContext(ctx, `SELECT context_id FROM sessions WHERE context_id=?`, a.contextID).Scan(&current)
		if errors.Is(err, sql.ErrNoRows) {
			return a, nil
		}
	}
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return a, err
	}
	return a, ErrLoginContext
}

// EndLoginContext uses the authenticated request's context, not a token which a
// racing callback may already have rotated. Foreign keys cancel pending OAuth,
// the replacement session and its grant together, but not other browser contexts.
func (s *Store) EndLoginContext(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM login_contexts WHERE id=?`, id)
	return err
}

// FinishOAuth is the only production callback persistence path. No provider I/O
// or cookie writes belong inside this short, conditional transaction.
func (s *Store) FinishOAuth(ctx context.Context, a OAuthAttempt, user User, session, csrf string, grant Grant) error {
	if user.ID <= 0 || strings.TrimSpace(user.Login) == "" || (a.ExpectedUserID != 0 && user.ID != a.ExpectedUserID) {
		return ErrLoginContext
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	now := time.Now().Unix()
	result, err := tx.ExecContext(ctx, `UPDATE login_contexts SET pending=NULL,expires=? WHERE id=? AND pending=? AND expires>? AND ?>?`, time.Now().Add(12*time.Hour).Unix(), a.contextID, a.state, now, a.expires, now)
	if err != nil {
		return err
	}
	count, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if count != 1 {
		return ErrLoginContext
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO users(id,login,name) VALUES(?,?,?) ON CONFLICT(id) DO UPDATE SET login=excluded.login`, user.ID, user.Login, user.Name); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `DELETE FROM sessions WHERE context_id=?`, a.contextID); err != nil {
		return err
	}
	if err = s.insertGrantedSession(ctx, tx, a.contextID, session, user.ID, csrf, grant); err != nil {
		return err
	}
	return tx.Commit()
}

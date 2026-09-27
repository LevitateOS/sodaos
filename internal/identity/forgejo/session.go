package forgejo

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"sync"
	"time"

	"github.com/levitateos/sodaos/internal/identity"
)

type Session struct {
	mu              sync.Mutex
	provider        *Provider
	ownerID         int64
	nonce, verifier string
	expires         time.Time
	timer           *time.Timer
	enrollment      identity.Enrollment
	connection      identity.Connection
	credential      Credential
}

var _ identity.CodeEnrollmentSession = (*Session)(nil)

func (s *Session) expire() {
	unretained := s.enrollment.State == "completed" && s.credential.Access != ""
	if (s.enrollment.State == "pending" || unretained) && !time.Now().Before(s.expires) {
		s.enrollment.State = "expired"
		s.nonce, s.verifier = "", ""
		s.credential = Credential{}
		s.connection = identity.Connection{}
		s.enrollment.VerificationURL = ""
		s.stopTimer()
	}
}

func (s *Session) expireEnrollment() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.expire()
}

func (s *Session) stopTimer() {
	if s.timer != nil {
		s.timer.Stop()
		s.timer = nil
	}
}

func (s *Session) Snapshot() identity.Enrollment {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.expire()
	return s.enrollment
}

func (s *Session) Complete(ctx context.Context, state, code string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.expire()
	if s.enrollment.State != "pending" || code == "" || subtle.ConstantTimeCompare([]byte(s.nonce), []byte(state)) != 1 {
		return identity.ErrDenied
	}
	verifier := s.verifier
	s.nonce, s.verifier = "", ""
	s.enrollment.State = "failed"
	s.enrollment.VerificationURL = ""
	ctx, cancel := context.WithDeadline(ctx, s.expires)
	defer cancel()
	token, err := s.provider.client.ExchangeGrant(ctx, s.provider.config.ClientID, s.provider.secret, code, s.provider.config.RedirectURL, verifier)
	if err != nil {
		return identity.ErrDenied
	}
	connection, credential, err := s.provider.verify(ctx, s.ownerID, token)
	if err != nil {
		return identity.ErrDenied
	}
	s.connection, s.credential = connection, credential
	s.enrollment.State = "completed"
	s.expire()
	if s.enrollment.State == "expired" {
		return identity.ErrDenied
	}
	return nil
}

func (s *Session) Finish(context.Context) (identity.Connection, []byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.expire()
	if s.enrollment.State != "completed" || !validCredential(s.credential, s.ownerID) {
		return identity.Connection{}, nil, identity.ErrDenied
	}
	encoded, err := json.Marshal(s.credential)
	if err != nil {
		return identity.Connection{}, nil, identity.ErrDenied
	}
	s.credential = Credential{}
	s.stopTimer()
	return s.connection, encoded, nil
}

func (s *Session) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.stopTimer()
	s.nonce, s.verifier = "", ""
	s.credential = Credential{}
	s.enrollment.VerificationURL = ""
	if s.enrollment.State == "pending" {
		s.enrollment.State = "canceled"
	}
	return nil
}

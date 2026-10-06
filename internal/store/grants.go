package store

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"database/sql"
	"errors"
)

var ErrGrantKey = errors.New("identity credential encryption key missing or incorrect")

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
	_, _ = rand.Read(nonce) // crypto/rand fills the buffer or terminates the process.
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

// Check an existing encrypted database before validating or creating its schema.
func (s *Store) checkGrantKey(ctx context.Context) error {
	var exists int
	if err := s.db.QueryRowContext(ctx, `SELECT count(*) FROM information_schema.tables WHERE table_schema='public' AND table_name='grant_key_check'`).Scan(&exists); err != nil {
		return err
	}
	if exists == 0 {
		return nil
	}
	var ciphertext []byte
	err := s.db.QueryRowContext(ctx, `SELECT ciphertext FROM grant_key_check WHERE id=1`).Scan(&ciphertext)
	if errors.Is(err, sql.ErrNoRows) {
		return s.rejectUnkeyedIdentityCredentials(ctx)
	}
	if err != nil {
		return err
	}
	return s.validateGrantKey(ciphertext)
}

func (s *Store) rejectUnkeyedIdentityCredentials(ctx context.Context) error {
	var identityTable int
	if err := s.db.QueryRowContext(ctx, `SELECT count(*) FROM information_schema.tables WHERE table_schema='public' AND table_name='identity_connections'`).Scan(&identityTable); err != nil {
		return err
	}
	if identityTable == 0 {
		return nil
	}
	var encrypted int
	if err := s.db.QueryRowContext(ctx, `SELECT count(*) FROM identity_connections WHERE octet_length(credential)>0`).Scan(&encrypted); err != nil {
		return err
	}
	if encrypted != 0 {
		return ErrGrantKey
	}
	return nil
}

func (s *Store) validateGrantKey(ciphertext []byte) error {
	plain, err := s.grants.open(ciphertext, keyBinding)
	if err != nil || string(plain) != keyBinding {
		return ErrGrantKey
	}
	return nil
}

func (s *Store) initializeGrantKey(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO grant_key_check(id,ciphertext) VALUES(1,$1) ON CONFLICT(id) DO NOTHING`, s.grants.seal([]byte(keyBinding), keyBinding))
	if err != nil {
		return err
	}
	return s.checkGrantKey(ctx)
}

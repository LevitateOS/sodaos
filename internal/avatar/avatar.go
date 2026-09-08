// Package avatar renders Soda's original robot artwork without identity lookups
// or network access. The v1 definition and renderer behavior are versioned assets.
package avatar

import (
	_ "embed"
	"encoding/hex"
	"errors"
	"strings"
	"sync"

	dicebear "github.com/dicebear/dicebear-go/v10"
)

const Version = "v1"

// Definition is the canonical editable artwork, embedded into the backend and
// available to the development preview. A string prevents caller mutation.
//
//go:embed style-v1.json
var definition string

func Definition() string { return definition }

var style = sync.OnceValues(func() (*dicebear.Style, error) {
	return dicebear.NewStyle([]byte(definition))
})

// Validate checks the embedded artwork before starting the production listener.
func Validate() error {
	_, err := style()
	return err
}

// NormalizeHash accepts only Forgejo's MD5 email identifier, not an email or a
// username. This is a public image seed, never an authentication credential.
func NormalizeHash(hash string) (string, error) {
	if len(hash) != 32 {
		return "", errors.New("avatar hash must contain 32 hexadecimal characters")
	}
	if _, err := hex.DecodeString(hash); err != nil {
		return "", errors.New("avatar hash must contain 32 hexadecimal characters")
	}
	return strings.ToLower(hash), nil
}

// Render returns the same SVG for a given hash and size, including across process
// restarts. Each call gets a separate DiceBear Avatar; the shared Style is immutable.
func Render(hash string, size int) (string, error) {
	hash, err := NormalizeHash(hash)
	if err != nil {
		return "", err
	}
	if size < 1 || size > 1024 {
		return "", errors.New("avatar size must be between 1 and 1024")
	}
	s, err := style()
	if err != nil {
		return "", err
	}
	a, err := dicebear.NewAvatar(s, map[string]any{
		"seed":            "soda-robot-" + Version + ":" + hash,
		"size":            size,
		"idRandomization": false,
	})
	if err != nil {
		return "", err
	}
	return a.SVG(), nil
}

//go:build linux

package main

import (
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
)

func executionState(root string) (string, error) {
	id := filepath.Base(root)
	if len(id) != 32 {
		return "", errors.New("invalid Muse execution ID")
	}
	if _, err := hex.DecodeString(id); err != nil {
		return "", err
	}
	state := "/tmp/soda-muse-state-" + id
	if err := os.Mkdir(state, 0o700); err != nil {
		return "", err
	}
	for _, name := range []string{"state", "cache", "data", "tmp"} {
		if err := os.Mkdir(filepath.Join(state, name), 0o700); err != nil {
			return "", err
		}
	}
	return state, nil
}

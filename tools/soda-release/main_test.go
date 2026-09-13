package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRejectsAmbiguousOperationsBeforeCredentialsOrNativeCommands(t *testing.T) {
	original := os.Args
	defer func() { os.Args = original }()
	for _, args := range [][]string{
		{"soda-release"},
		{"soda-release", "sign", "--out", filepath.Join(t.TempDir(), "out"), "--observe"},
		{"soda-release", "fetch", "--out", filepath.Join(t.TempDir(), "out"), "--auth-file", "/must-not-read"},
		{"soda-release", "publish", "--out", "relative"},
		{"soda-release", "reboot", "--out", filepath.Join(t.TempDir(), "out")},
	} {
		os.Args = args
		require.Error(t, run())
	}
}

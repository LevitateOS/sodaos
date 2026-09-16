package main

import (
	"errors"
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
		err := run()
		require.Error(t, err)
		require.Equal(t, 2, exitCode(err))
	}
}

func TestOperationalFailuresKeepExitOne(t *testing.T) {
	require.Equal(t, 1, exitCode(errors.New("registry unreachable")))
}

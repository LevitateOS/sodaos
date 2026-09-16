package main

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFinishButaneConversionRemovesPartialOutput(t *testing.T) {
	out := filepath.Join(t.TempDir(), "ignition.json")
	dest, err := os.OpenFile(out, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	require.NoError(t, err)
	err = finishButaneConversion(dest, out, func() error {
		_, _ = dest.WriteString("partial")
		return errors.New("butane boom")
	})
	require.ErrorContains(t, err, "partial output removed")
	_, statErr := os.Lstat(out)
	require.ErrorIs(t, statErr, os.ErrNotExist)
}

func TestFinishButaneConversionKeepsSuccessfulOutput(t *testing.T) {
	out := filepath.Join(t.TempDir(), "ignition.json")
	dest, err := os.OpenFile(out, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	require.NoError(t, err)
	require.NoError(t, finishButaneConversion(dest, out, func() error {
		_, err := dest.WriteString("{}")
		return err
	}))
	require.FileExists(t, out)
}

package main

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRefusesUnprivilegedInvocationBeforeReadingHost(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("unprivileged admission check")
	}
	require.ErrorContains(t, run(), "root and no arguments required")
}

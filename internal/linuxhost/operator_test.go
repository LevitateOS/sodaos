package linuxhost

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPKExecCallerRequiresBothRootIDsAndRootOriginalCaller(t *testing.T) {
	for _, test := range []struct {
		uid, euid int
		original  string
		allowed   bool
	}{
		{0, 0, "", true}, {0, 0, "0", true},
		{1000, 0, "", false}, {0, 1000, "", false}, {1000, 1000, "0", false},
		{0, 0, "1000", false}, {0, 0, "00", false}, {0, 0, "invalid", false},
	} {
		actor, err := pkexecCaller(test.uid, test.euid, test.original)
		if test.allowed {
			require.NoError(t, err)
			require.Equal(t, PKExecIdentity{Username: "root", UID: 0}, actor)
		} else {
			require.Error(t, err)
			require.Empty(t, actor)
		}
	}
}

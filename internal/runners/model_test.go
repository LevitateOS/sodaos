package runners

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCreateRequestRejectsRegistration(t *testing.T) {
	require.ErrorIs(t, (CreateRequest{}).Validate(), ErrUnavailable)
}

func TestRunnerIdentityHasAStableNarrowLinuxShape(t *testing.T) {
	account, err := AccountName("build-arm64")
	require.NoError(t, err)
	require.Equal(t, "soda-runner-build-arm64", account)
	require.LessOrEqual(t, len(account), 32)
	require.Error(t, ValidateID("Project/runner"))
}

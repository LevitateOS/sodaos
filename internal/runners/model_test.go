package runners

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCreateRequestAcceptsOnlyForgejo(t *testing.T) {
	forgejo := CreateRequest{
		ID: "forgejo-one", Provider: ProviderForgejo,
		RegistrationURL: "http://soda.example.test:30000",
		RegistrationID:  "33834eef-e758-48c4-a676-1745426747aa",
		Labels:          "soda-arm64:host", RegistrationToken: "provider-input",
	}
	require.NoError(t, forgejo.Validate())

	for _, provider := range []Provider{"github", "gitlab", ""} {
		rejected := forgejo
		rejected.Provider = provider
		require.ErrorContains(t, rejected.Validate(), "provider must be forgejo")
	}
	forgejo.Labels = "container:docker://example.test/image"
	require.ErrorContains(t, forgejo.Validate(), "name:host")
}

func TestRunnerIdentityHasAStableNarrowLinuxShape(t *testing.T) {
	account, err := AccountName("build-arm64")
	require.NoError(t, err)
	require.Equal(t, "soda-runner-build-arm64", account)
	require.LessOrEqual(t, len(account), 32)
	require.Error(t, ValidateID("Project/runner"))
}

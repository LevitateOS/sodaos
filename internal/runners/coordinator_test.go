package runners

import (
	"context"
	"errors"
	"strings"
	"testing"
	"testing/iotest"

	"github.com/levitateos/sodaos/internal/linuxhost"
	"github.com/stretchr/testify/require"
)

type fakeAuthorizer struct{ err error }

func (authorizer fakeAuthorizer) RequireAdministrator(context.Context, linuxhost.PKExecIdentity) error {
	return authorizer.err
}

var testAdministrator = linuxhost.PKExecIdentity{Username: "root", UID: 0}

type fakeLocal struct{ views []RunnerView }

func (local fakeLocal) List(context.Context) ([]RunnerView, error) { return local.views, nil }

type fakeLifecycle struct {
	action string
	create CreateRequest
	id     string
	err    error
}

func (lifecycle *fakeLifecycle) Create(ctx context.Context, request CreateRequest) error {
	lifecycle.action, lifecycle.create = "create", request
	return errors.Join(lifecycle.err, ctx.Err())
}
func (lifecycle *fakeLifecycle) Start(ctx context.Context, id string) error {
	lifecycle.action, lifecycle.id = "start", id
	return errors.Join(lifecycle.err, ctx.Err())
}
func (lifecycle *fakeLifecycle) Stop(ctx context.Context, id string) error {
	lifecycle.action, lifecycle.id = "stop", id
	return errors.Join(lifecycle.err, ctx.Err())
}
func (lifecycle *fakeLifecycle) Restart(ctx context.Context, id string) error {
	lifecycle.action, lifecycle.id = "restart", id
	return errors.Join(lifecycle.err, ctx.Err())
}
func (lifecycle *fakeLifecycle) Remove(ctx context.Context, id string) error {
	lifecycle.action, lifecycle.id = "remove", id
	return errors.Join(lifecycle.err, ctx.Err())
}

func TestCoordinatorRequiresNativeAdministratorBeforeReadingInputOrState(t *testing.T) {
	for _, action := range []string{"list", "create", "start", "stop", "restart", "remove"} {
		for _, test := range []struct {
			actor     linuxhost.PKExecIdentity
			account   linuxhost.Account
			lookupErr error
		}{
			{actor: linuxhost.PKExecIdentity{Username: "alice", UID: 1000}},
			{actor: testAdministrator, account: linuxhost.Account{Username: "root", UID: 1000}},
			{actor: testAdministrator, account: linuxhost.Account{Username: "other", UID: 0}},
			{actor: testAdministrator, lookupErr: errors.New("lookup unavailable")},
		} {
			coordinator := Coordinator{Authorizer: LinuxAuthorizer{Accounts: fakeLinuxAccounts{account: test.account, err: test.lookupErr}}}
			// Nil native owners panic if authorization ever reaches them.
			inputErr := errors.New("must not read input")
			_, err := coordinator.Execute(t.Context(), test.actor, action, iotest.ErrReader(inputErr))
			require.Error(t, err)
			require.NotErrorIs(t, err, inputErr)
		}
	}
}

func TestCoordinatorUsesOnlyTheConfiguredOrBundledForgejoEndpoint(t *testing.T) {
	for _, endpoint := range []string{"", "http://forgejo.fixture:3000"} {
		lifecycle := &fakeLifecycle{}
		coordinator := Coordinator{ForgejoURL: endpoint, Authorizer: fakeAuthorizer{}, Lifecycle: lifecycle}
		response, err := coordinator.Execute(t.Context(), testAdministrator, "create", strings.NewReader(`{
			"id":"forgejo-one","provider":"forgejo","registration_url":"https://external.invalid",
			"registration_id":"33834eef-e758-48c4-a676-1745426747aa",
			"labels":"soda-arm64:host","registration_token":"provider-input"
		}`))
		require.NoError(t, err)
		require.Equal(t, MutationResponse{OK: true}, response)
		if endpoint == "" {
			endpoint = BundledForgejoURL
		}
		require.Equal(t, endpoint, lifecycle.create.RegistrationURL)
		require.Equal(t, "provider-input", lifecycle.create.RegistrationToken)
	}
}

func TestCoordinatorReportsExactLocalListenerAndCapacityCounts(t *testing.T) {
	views := []RunnerView{
		{Descriptor: Descriptor{ID: "one"}, Capacity: 1, Service: ServiceState{Active: "active", Sub: "running"}},
		{Descriptor: Descriptor{ID: "two"}, Capacity: 1, Service: ServiceState{Active: "failed", Sub: "failed"}},
	}
	coordinator := Coordinator{ForgejoPublicURL: "https://forgejo.fixture", Authorizer: fakeAuthorizer{}, Local: fakeLocal{views: views}}
	response, err := coordinator.Execute(t.Context(), testAdministrator, "list", strings.NewReader(`{}`))
	require.NoError(t, err)
	require.Equal(t, ListResponse{ForgejoURL: "https://forgejo.fixture", Runners: views, RunnerCount: 2, ActiveListeners: 1, TotalCapacity: 2}, response)
}

func TestCoordinatorRetainsStrictRequestDecoding(t *testing.T) {
	for _, test := range []struct{ action, body string }{
		{"list", `{"path":"/tmp/other-client"}`},
		{"list", `{} {}`},
		{"start", `{"id":"one","id":"two"}`},
		{"start", `{"id":"one","project_id":"site"}`},
		{"remove", `{"id":"../one"}`},
		{"create", `{"provider":"unsupported","registration_token":"private-input"}`},
	} {
		coordinator := Coordinator{Authorizer: fakeAuthorizer{}}
		_, err := coordinator.Execute(t.Context(), testAdministrator, test.action, strings.NewReader(test.body))
		require.Error(t, err)
		require.NotContains(t, err.Error(), "private-input")
	}
}

func TestCoordinatorDispatchesExactNativeLifecycleAndPreservesFailure(t *testing.T) {
	for _, action := range []string{"start", "stop", "restart", "remove"} {
		for _, failure := range []error{nil, errors.New("native result unconfirmed"), context.Canceled} {
			lifecycle := &fakeLifecycle{err: failure}
			coordinator := Coordinator{Authorizer: fakeAuthorizer{}, Lifecycle: lifecycle}
			response, err := coordinator.Execute(t.Context(), testAdministrator, action, strings.NewReader(`{"id":"one"}`))
			require.Equal(t, action, lifecycle.action)
			require.Equal(t, "one", lifecycle.id)
			if failure == nil {
				require.NoError(t, err)
				require.Equal(t, MutationResponse{OK: true}, response)
			} else {
				require.ErrorIs(t, err, failure)
				require.Equal(t, MutationResponse{}, response)
			}
		}
	}
}

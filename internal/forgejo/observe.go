package forgejo

import (
	"context"
	"net/url"
	"strconv"
	"sync"

	extensions "forgejo.org/extension-sdk"
	"github.com/levitateos/sodaos/internal/config"
)

// Observation issue pages bound one reconciliation listing: 50 issues per
// page like native snapshot pages, oldest first so the oldest work is
// discovered before any sweep cap truncates the listing.
const ObservationIssuePageSize = 50

// ServiceObserver is the unattended native observation client: a lazily
// bootstrapped service background client for permission-checked snapshot
// reads, plus bounded REST enumeration over the same service credential.
// Bootstrap and actor resolution happen on first use and stay in memory;
// failures report per call without caching, so a later Fountain start or
// credential rotation recovers without a Soda restart.
type ServiceObserver struct {
	socket         string
	credentialFile string
	rest           *Client
	hostUID        uint32
	mu             sync.Mutex
	client         extensions.BackgroundClient
	actor          string
}

// NewServiceObserver builds the service observation client over the
// operator-configured shared service callback socket, expected native
// host peer UID, restricted PAT file and internal REST client. No I/O
// happens before the first observation.
func NewServiceObserver(socket string, hostUID uint32, credentialFile string, rest *Client) *ServiceObserver {
	return &ServiceObserver{socket: socket, hostUID: hostUID, credentialFile: credentialFile, rest: rest}
}

// Credential is the SDK-local restricted secret input for background
// reads: the configured PAT file, never the secret itself.
func (o *ServiceObserver) Credential() extensions.CredentialFile {
	return extensions.CredentialFile(o.credentialFile)
}

// SnapshotReader returns the thin snapshot transport over the lazily
// bootstrapped service client. The service credential file always
// supplies the secret; the read actor is the credential's native owner.
func (o *ServiceObserver) SnapshotReader() SnapshotReader {
	return &serviceSnapshotReader{observer: o}
}

func (o *ServiceObserver) ensureClient(ctx context.Context) (extensions.BackgroundClient, error) {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.client != nil {
		return o.client, nil
	}
	if o.socket == "" || o.credentialFile == "" {
		return nil, ErrUnavailable
	}
	client, err := extensions.BootstrapServiceBackground(ctx, extensions.ServiceBridgeOptions{
		SocketPath: o.socket, ExpectedHostUID: o.hostUID,
	})
	if err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, err
	}
	o.client = client
	return o.client, nil
}

func (o *ServiceObserver) ensureActor(ctx context.Context) (string, error) {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.actor != "" {
		return o.actor, nil
	}
	if o.rest == nil || o.credentialFile == "" {
		return "", ErrUnavailable
	}
	token, err := config.Secret(o.credentialFile)
	if err != nil {
		return "", err
	}
	user, err := o.rest.Current(ctx, token)
	if err != nil {
		return "", err
	}
	if user.ID <= 0 {
		return "", ErrInvalidResponse
	}
	o.actor = strconv.FormatInt(user.ID, 10)
	return o.actor, nil
}

// serviceSnapshotReader adapts the lazily bootstrapped service client to
// the snapshot transport. Reads present the service actor credential
// privately; revision observations use owning-installation admission.
type serviceSnapshotReader struct {
	observer *ServiceObserver
}

func (r *serviceSnapshotReader) ReadNativeRevision(ctx context.Context) (extensions.NativeRevisionObservation, error) {
	client, err := r.observer.ensureClient(ctx)
	if err != nil {
		return extensions.NativeRevisionObservation{}, err
	}
	return client.ReadNativeRevision(ctx)
}

func (r *serviceSnapshotReader) ReadSnapshot(ctx context.Context, _ extensions.CredentialFile, req SnapshotRequest) (NativeSnapshot, error) {
	client, err := r.observer.ensureClient(ctx)
	if err != nil {
		return NativeSnapshot{}, err
	}
	actor, err := r.observer.ensureActor(ctx)
	if err != nil {
		return NativeSnapshot{}, err
	}
	reader := &BackgroundSnapshotReader{Client: client, ActorID: actor}
	return reader.ReadSnapshot(ctx, r.observer.Credential(), req)
}

// ListIssuesPage enumerates one bounded page of a repository's native
// issue indexes, oldest first. Pull requests are excluded server-side;
// assessment re-verifies issue shape from authoritative snapshots.
func (o *ServiceObserver) ListIssuesPage(ctx context.Context, repository int64, page int) ([]int64, bool, error) {
	if o.rest == nil || o.credentialFile == "" || repository <= 0 || page < 1 {
		return nil, false, ErrUnavailable
	}
	token, err := config.Secret(o.credentialFile)
	if err != nil {
		return nil, false, err
	}
	repo, err := o.rest.RepositoryByID(ctx, token, repository)
	if err != nil {
		return nil, false, err
	}
	return o.rest.ListRepositoryIssuesPage(ctx, token, repo.Owner.Login, repo.Name, page)
}

// RepositoryByID resolves one repository's owner/name locator by ID.
func (c *Client) RepositoryByID(ctx context.Context, token string, id int64) (Repository, error) {
	var repo Repository
	if id <= 0 {
		return repo, ErrInvalidResponse
	}
	if err := c.request(ctx, "GET", "/repositories/"+strconv.FormatInt(id, 10), token, nil, &repo); err != nil {
		return Repository{}, err
	}
	if repo.ID != id || !repositoryPart(repo.Owner.Login) || !repositoryPart(repo.Name) ||
		repo.FullName != repo.Owner.Login+"/"+repo.Name {
		return Repository{}, ErrInvalidResponse
	}
	return repo, nil
}

// ListedIssue is the bounded native issue locator carried by list pages.
type ListedIssue struct {
	Index int64 `json:"index"`
}

// ListRepositoryIssuesPage lists one bounded page of a repository's issue
// indexes, oldest first. At most the page size is accepted; a full page
// reports more work until a short page ends the listing.
func (c *Client) ListRepositoryIssuesPage(ctx context.Context, token, owner, name string, page int) ([]int64, bool, error) {
	if !repositoryPart(owner) || !repositoryPart(name) || page < 1 {
		return nil, false, ErrInvalidResponse
	}
	query := url.Values{
		"state": {"all"}, "type": {"issues"}, "sort": {"oldest"},
		"limit": {strconv.Itoa(ObservationIssuePageSize)}, "page": {strconv.Itoa(page)},
	}
	var items []ListedIssue
	path := "/repos/" + url.PathEscape(owner) + "/" + url.PathEscape(name) + "/issues?" + query.Encode()
	if err := c.request(ctx, "GET", path, token, nil, &items); err != nil {
		return nil, false, err
	}
	if len(items) > ObservationIssuePageSize {
		return nil, false, ErrInvalidResponse
	}
	seen := make(map[int64]bool, len(items))
	var indexes []int64
	for _, item := range items {
		if item.Index <= 0 || seen[item.Index] {
			return nil, false, ErrInvalidResponse
		}
		seen[item.Index] = true
		indexes = append(indexes, item.Index)
	}
	return indexes, len(items) == ObservationIssuePageSize, nil
}

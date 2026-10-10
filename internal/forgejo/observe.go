package forgejo

import (
	"context"
	"crypto/tls"
	"net/http"
	"strconv"
	"sync"

	extensions "forgejo.org/extension-sdk"
	"github.com/levitateos/sodaos/internal/config"
)

// ServiceObserver is the unattended native observation client: a lazily
// bootstrapped service background client for permission-checked snapshot
// reads and actor resolution over the same service credential.
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
	background     *ServiceBackground
	actor          string
	actorCall      *serviceActorCall
}

type serviceActorCall struct {
	done  chan struct{}
	actor string
	err   error
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
	o.client = o.sharedBackgroundLocked()
	return o.client, nil
}

// sharedBackgroundLocked returns the observer's service transport,
// building an unshared one when the server did not share. Callers hold
// the observer mutex.
func (o *ServiceObserver) sharedBackgroundLocked() *ServiceBackground {
	if o.background == nil {
		o.background = NewServiceBackground(o.socket, o.hostUID, "")
	}
	return o.background
}

// ShareBackground shares one service admission between snapshot readers
// and conditional-operation publishers. The native host revokes the
// previous service admission on every bootstrap, so splitting readers
// and publishers across two bootstraps would invalidate each other.
func (o *ServiceObserver) ShareBackground(background *ServiceBackground) {
	if background == nil {
		return
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	o.background = background
	o.client = background
}

func (o *ServiceObserver) ensureActor(ctx context.Context) (string, error) {
	o.mu.Lock()
	if o.actor != "" {
		actor := o.actor
		o.mu.Unlock()
		return actor, nil
	}
	if o.rest == nil || o.credentialFile == "" {
		o.mu.Unlock()
		return "", ErrUnavailable
	}
	if call := o.actorCall; call != nil {
		o.mu.Unlock()
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-call.done:
			return call.actor, call.err
		}
	}
	call := &serviceActorCall{done: make(chan struct{})}
	o.actorCall = call
	rest, credentialFile := o.rest, o.credentialFile
	o.mu.Unlock()

	actor, lookupErr := loadServiceActor(ctx, rest, credentialFile)

	o.mu.Lock()
	if lookupErr == nil {
		o.actor = actor
	}
	call.actor = actor
	call.err = lookupErr
	o.actorCall = nil
	close(call.done)
	o.mu.Unlock()
	return actor, lookupErr
}

func loadServiceActor(ctx context.Context, rest *Client, credentialFile string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	token, err := config.Secret(credentialFile)
	if err != nil {
		return "", err
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	user, err := loadActor(ctx, rest, token)
	if err != nil {
		return "", err
	}
	if user.ID <= 0 {
		return "", ErrInvalidResponse
	}
	return strconv.FormatInt(user.ID, 10), nil
}

// loadActor resolves one credential through a private, single-use HTTP/1
// transport so concurrent shared-client traffic cannot replenish retries.
func loadActor(ctx context.Context, rest *Client, token string) (User, error) {
	if err := ctx.Err(); err != nil {
		return User{}, err
	}
	if rest == nil || rest.HTTP == nil || token == "" {
		return User{}, ErrUnavailable
	}
	configured := *rest.HTTP
	transport := configured.Transport
	if transport == nil {
		transport = http.DefaultTransport
	}
	standard, ok := transport.(*http.Transport)
	if !ok || standard == nil {
		return User{}, ErrUnavailable
	}
	lookupTransport := standard.Clone()
	protocols := new(http.Protocols)
	protocols.SetHTTP1(true)
	lookupTransport.Protocols = protocols
	if lookupTransport.TLSClientConfig != nil {
		lookupTransport.TLSClientConfig.NextProtos = []string{"http/1.1"}
	}
	lookupTransport.TLSNextProto = make(map[string]func(string, *tls.Conn) http.RoundTripper)
	defer lookupTransport.CloseIdleConnections()
	lookupHTTP := configured
	lookupHTTP.Transport = lookupTransport
	lookupHTTP.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	lookupClient := *rest
	lookupClient.HTTP = &lookupHTTP
	user, err := lookupClient.Current(ctx, token)
	if err != nil {
		return User{}, err
	}
	return user, nil
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

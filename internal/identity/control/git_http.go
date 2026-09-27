package control

import (
	"context"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/levitateos/sodaos/internal/forgejo"
	"github.com/levitateos/sodaos/internal/identity"
)

type gitRequest struct {
	lease      identity.Lease
	provider   gitProvider
	repository forgejo.Repository
	user       forgejo.User
	credential []byte
	request    *http.Request
	cancel     context.CancelFunc
}

func gitOperation(r *http.Request) (bool, error) {
	switch path.Base(r.URL.Path) {
	case "refs":
		return gitAdvertisement(r)
	case "git-upload-pack", "git-receive-pack":
		service := path.Base(r.URL.Path)
		if r.Method != http.MethodPost || r.URL.RawQuery != "" || r.URL.ForceQuery || r.Header.Get("Content-Type") != "application/x-"+service+"-request" {
			return false, identity.ErrDenied
		}
		return service == "git-receive-pack", nil
	default:
		return false, identity.ErrDenied
	}
}

func gitAdvertisement(r *http.Request) (bool, error) {
	q, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil || r.Method != http.MethodGet || len(q) != 1 || len(q["service"]) != 1 {
		return false, identity.ErrDenied
	}
	switch q.Get("service") {
	case "git-upload-pack":
		return false, nil
	case "git-receive-pack":
		return true, nil
	default:
		return false, identity.ErrDenied
	}
}

func gitPathValid(r *http.Request, repo forgejo.Repository) bool {
	base := "/" + repo.Owner.Login + "/" + repo.Name + ".git/"
	return r.URL.RawPath == "" && (r.URL.Path == base+"info/refs" || r.URL.Path == base+"git-upload-pack" || r.URL.Path == base+"git-receive-pack")
}

func (c *Controller) prepareGitRequest(r *http.Request, id string) (gitRequest, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	l, err := c.gitRequestLease(r, id)
	if err != nil {
		return gitRequest{}, err
	}
	ctx, cancel := context.WithDeadline(r.Context(), l.Deadline)
	r = r.WithContext(ctx)
	accepted := false
	defer func() {
		if !accepted {
			cancel()
		}
	}()
	if err := c.authorizeGitRuntime(ctx, l); err != nil {
		return gitRequest{}, err
	}
	write, _ := gitOperation(r)
	p, ok := c.providers[identity.Forgejo].(gitProvider)
	if !ok {
		return gitRequest{}, identity.ErrDenied
	}
	_, data, err := c.forgejoCredentialLocked(r.Context(), l.ActorID, l.ConnectionID)
	if err != nil {
		return gitRequest{}, err
	}
	repo, user, _, err := p.GitAuthority(r.Context(), l.ActorID, data, l.RepositoryID, write)
	if err != nil || !gitPathValid(r, repo) {
		clear(data)
		return gitRequest{}, identity.ErrDenied
	}
	c.trackGitRequest(l.ID, r, cancel)
	accepted = true
	return gitRequest{lease: l, provider: p, repository: repo, user: user, credential: data, request: r, cancel: cancel}, nil
}

func (c *Controller) trackGitRequest(id string, r *http.Request, cancel context.CancelFunc) {
	if c.gitRequests == nil {
		c.gitRequests = make(map[string]map[*http.Request]context.CancelFunc)
	}
	if c.gitRequests[id] == nil {
		c.gitRequests[id] = make(map[*http.Request]context.CancelFunc)
	}
	c.gitRequests[id][r] = cancel
}

// cancelGitRequests runs while the controller lock is held, before native stop.
func (c *Controller) cancelGitRequests(id string) {
	for _, cancel := range c.gitRequests[id] {
		cancel()
	}
}

func (c *Controller) releaseGitRequest(g gitRequest) {
	g.cancel()
	clear(g.credential)
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.gitRequests[g.lease.ID], g.request)
	if len(c.gitRequests[g.lease.ID]) == 0 {
		delete(c.gitRequests, g.lease.ID)
	}
}

func (c *Controller) serveGit(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	parts := strings.SplitN(strings.TrimPrefix(r.URL.Path, "/git/"), "/", 2)
	if len(parts) != 2 || parts[0] == "" || r.Header.Get("Origin") != "" || r.URL.RawPath != "" {
		http.Error(w, "denied", http.StatusForbidden)
		return
	}
	r = r.Clone(r.Context())
	r.URL.Path = "/" + parts[1]
	g, err := c.prepareGitRequest(r, parts[0])
	if err != nil {
		status, code := responseError(err)
		http.Error(w, code, status)
		return
	}
	defer c.releaseGitRequest(g)
	if g.provider.ProxyGit(w, g.request, g.lease.ActorID, g.credential, g.repository, g.user) {
		c.rejectGit(g.lease)
	}
}

func (c *Controller) rejectGit(l identity.Lease) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.uncertain(ctx, l) == nil {
		_ = c.retireConnection(ctx, l.ConnectionID)
	}
}

func (c *Controller) gitRequestLease(r *http.Request, id string) (identity.Lease, error) {
	l, err := c.store.IdentityLease(r.Context(), id)
	if err != nil || l.ProviderID != identity.Forgejo || l.Binding == nil || !l.Deadline.After(time.Now()) {
		return identity.Lease{}, identity.ErrDenied
	}
	write, err := gitOperation(r)
	if err != nil || (l.Kind == identity.Factory && write) {
		return identity.Lease{}, identity.ErrDenied
	}
	return l, nil
}

func (c *Controller) authorizeGitRuntime(ctx context.Context, l identity.Lease) error {
	conn, err := c.registrationAuthority(ctx, l)
	if err != nil || conn.OwnerID != l.ActorID || c.runtime.Validate(ctx, l) != nil {
		return identity.ErrDenied
	}
	return nil
}

package api

import (
	"net/http"

	"github.com/levitateos/factory/pkg/collaboration"
	factoryhttpapi "github.com/levitateos/factory/pkg/factory/httpapi"
	factoryservice "github.com/levitateos/factory/pkg/factory/service"
	"github.com/levitateos/factory/pkg/store"
	"github.com/levitateos/sodaos/internal/forgejo"
	"github.com/levitateos/sodaos/internal/web/auth"
)

func (s *API) collaborationReads(w http.ResponseWriter, r *http.Request, v store.Session) (factoryhttpapi.Product, forgejo.HumanReads, bool) {
	actor, ok := s.factoryCurrentActor(w, r, v)
	if !ok {
		return factoryhttpapi.Product{}, forgejo.HumanReads{}, false
	}
	authority, ok := requestExtensionAuthority(r)
	if !ok {
		auth.JSONError(w, 403, "native_authority_unavailable", "Current native authority is required.")
		return factoryhttpapi.Product{}, forgejo.HumanReads{}, false
	}
	current := func(w http.ResponseWriter, r *http.Request) bool { return s.checkLifecycleSession(w, r, v) }
	return s.factoryProduct(v, collaboration.ActorID(actor), current), forgejo.HumanReads{Authority: authority, Identities: s.FountainIdentities}, true
}
func (s *API) apiFactoryCollaborationIssues(w http.ResponseWriter, r *http.Request, v store.Session) {
	product, source, ok := s.collaborationReads(w, r, v)
	if !ok {
		return
	}
	product.CollaborationIssues = factoryhttpapi.CollaborationIssues{Service: factoryservice.CollaborationIssues{Source: source}}
	product.Handler().ServeHTTP(w, r)
}
func (s *API) apiFactoryCollaborationDiscussions(w http.ResponseWriter, r *http.Request, v store.Session) {
	product, source, ok := s.collaborationReads(w, r, v)
	if !ok {
		return
	}
	product.Discussions = factoryhttpapi.CollaborationDiscussions{Service: factoryservice.CollaborationDiscussions{Source: source}}
	product.Handler().ServeHTTP(w, r)
}

func (s *API) apiFactoryHumanComments(w http.ResponseWriter, r *http.Request, v store.Session) {
	product, source, ok := s.collaborationReads(w, r, v)
	if !ok {
		return
	}
	writer := forgejo.HumanComments{Authority: source.Authority, Identities: source.Identities}
	product.HumanComments = factoryhttpapi.HumanComments{Service: s.FactoryApplication().HumanComments(writer)}
	product.Handler().ServeHTTP(w, r)
}

func (s *API) apiFactoryHumanEdits(w http.ResponseWriter, r *http.Request, v store.Session) {
	product, source, ok := s.collaborationReads(w, r, v)
	if !ok {
		return
	}
	writer := forgejo.HumanEdits{Authority: source.Authority, Identities: source.Identities}
	product.HumanEdits = factoryhttpapi.HumanEdits{Service: s.FactoryApplication().HumanEdits(writer)}
	product.Handler().ServeHTTP(w, r)
}

func (s *API) apiFactoryHumanLocks(w http.ResponseWriter, r *http.Request, v store.Session) {
	product, source, ok := s.collaborationReads(w, r, v)
	if !ok {
		return
	}
	writer := forgejo.HumanLocks{Authority: source.Authority, Identities: source.Identities}
	product.HumanLocks = factoryhttpapi.HumanLocks{Service: s.FactoryApplication().HumanLocks(writer)}
	product.Handler().ServeHTTP(w, r)
}

func (s *API) apiFactoryHumanStates(w http.ResponseWriter, r *http.Request, v store.Session) {
	product, source, ok := s.collaborationReads(w, r, v)
	if !ok {
		return
	}
	writer := forgejo.HumanStates{Authority: source.Authority, Identities: source.Identities}
	product.HumanStates = factoryhttpapi.HumanStates{Service: s.FactoryApplication().HumanStates(writer)}
	product.Handler().ServeHTTP(w, r)
}

func (s *API) apiFactoryHumanIssueCreates(w http.ResponseWriter, r *http.Request, v store.Session) {
	product, source, ok := s.collaborationReads(w, r, v)
	if !ok {
		return
	}
	writer := forgejo.HumanIssueCreates{Authority: source.Authority, Identities: source.Identities}
	product.HumanIssueCreates = factoryhttpapi.HumanIssueCreates{Service: s.FactoryApplication().HumanIssueCreates(writer)}
	product.Handler().ServeHTTP(w, r)
}

func (s *API) apiFactoryCollaborationPulls(w http.ResponseWriter, r *http.Request, v store.Session) {
	product, source, ok := s.collaborationReads(w, r, v)
	if !ok {
		return
	}
	product.CollaborationPulls = factoryhttpapi.CollaborationPulls{Service: factoryservice.CollaborationPulls{Source: source}}
	product.Handler().ServeHTTP(w, r)
}

func (s *API) apiFactoryCollaborationPullFiles(w http.ResponseWriter, r *http.Request, v store.Session) {
	product, source, ok := s.collaborationReads(w, r, v)
	if !ok {
		return
	}
	product.PullFiles = factoryhttpapi.CollaborationPullFiles{Service: factoryservice.CollaborationPullFiles{Source: source}}
	product.Handler().ServeHTTP(w, r)
}

func (s *API) apiFactoryCollaborationPullCommits(w http.ResponseWriter, r *http.Request, v store.Session) {
	product, source, ok := s.collaborationReads(w, r, v)
	if !ok {
		return
	}
	product.PullCommits = factoryhttpapi.CollaborationPullCommits{Service: factoryservice.CollaborationPullCommits{Source: source}}
	product.Handler().ServeHTTP(w, r)
}

func (s *API) apiFactoryCollaborationReviews(w http.ResponseWriter, r *http.Request, v store.Session) {
	product, source, ok := s.collaborationReads(w, r, v)
	if !ok {
		return
	}
	product.Reviews = factoryhttpapi.CollaborationReviews{Service: factoryservice.CollaborationReviews{Source: source}}
	product.Handler().ServeHTTP(w, r)
}

func (s *API) apiFactoryCollaborationThreads(w http.ResponseWriter, r *http.Request, v store.Session) {
	product, source, ok := s.collaborationReads(w, r, v)
	if !ok {
		return
	}
	product.Threads = factoryhttpapi.CollaborationThreads{Service: factoryservice.CollaborationThreads{Source: source}}
	product.Handler().ServeHTTP(w, r)
}

func (s *API) apiFactoryCollaborationCapabilities(w http.ResponseWriter, r *http.Request, v store.Session) {
	product, source, ok := s.collaborationReads(w, r, v)
	if !ok {
		return
	}
	product.CapabilityReads = factoryhttpapi.CollaborationCapabilities{Service: factoryservice.CollaborationCapabilities{Source: source}}
	product.Handler().ServeHTTP(w, r)
}

func (s *API) apiFactoryCollaborationRepositoryCatalog(w http.ResponseWriter, r *http.Request, v store.Session) {
	product, source, ok := s.collaborationReads(w, r, v)
	if !ok {
		return
	}
	product.RepositoryCatalog = factoryhttpapi.CollaborationRepositories{Service: factoryservice.CollaborationRepositories{Source: source}}
	product.Handler().ServeHTTP(w, r)
}

func (s *API) apiFactoryCollaborationChecks(w http.ResponseWriter, r *http.Request, v store.Session) {
	product, source, ok := s.collaborationReads(w, r, v)
	if !ok {
		return
	}
	product.Checks = factoryhttpapi.CollaborationChecks{Service: factoryservice.CollaborationChecks{Source: source}}
	product.Handler().ServeHTTP(w, r)
}
func (s *API) apiFactoryHumanReviews(w http.ResponseWriter, r *http.Request, v store.Session) {
	product, source, ok := s.collaborationReads(w, r, v)
	if !ok {
		return
	}
	writer := forgejo.HumanReviews{Authority: source.Authority, Identities: source.Identities}
	product.HumanReviews = factoryhttpapi.HumanReviews{Service: s.FactoryApplication().HumanReviews(writer)}
	product.Handler().ServeHTTP(w, r)
}

func (s *API) apiFactoryHumanInlines(w http.ResponseWriter, r *http.Request, v store.Session) {
	product, source, ok := s.collaborationReads(w, r, v)
	if !ok {
		return
	}
	writer := forgejo.HumanInlines{Authority: source.Authority, Identities: source.Identities}
	product.HumanInlines = factoryhttpapi.HumanInlines{Service: s.FactoryApplication().HumanInlines(writer)}
	product.Handler().ServeHTTP(w, r)
}

func (s *API) apiFactoryHumanThreads(w http.ResponseWriter, r *http.Request, v store.Session) {
	product, source, ok := s.collaborationReads(w, r, v)
	if !ok {
		return
	}
	writer := forgejo.HumanThreads{Authority: source.Authority, Identities: source.Identities}
	threads := s.FactoryApplication().HumanThreads(writer)
	threads.Reader = writer
	product.HumanThreads = factoryhttpapi.HumanThreads{Service: threads}
	product.Handler().ServeHTTP(w, r)
}

func (s *API) apiFactoryHumanReviewRequests(w http.ResponseWriter, r *http.Request, v store.Session) {
	product, source, ok := s.collaborationReads(w, r, v)
	if !ok {
		return
	}
	writer := forgejo.HumanReviewRequests{Authority: source.Authority, Identities: source.Identities}
	product.HumanReviewRequests = factoryhttpapi.HumanReviewRequests{Service: s.FactoryApplication().HumanReviewRequests(writer)}
	product.Handler().ServeHTTP(w, r)
}

func (s *API) apiFactoryHumanMerges(w http.ResponseWriter, r *http.Request, v store.Session) {
	product, source, ok := s.collaborationReads(w, r, v)
	if !ok {
		return
	}
	writer := forgejo.HumanMerges{Authority: source.Authority, Identities: source.Identities}
	product.HumanMerges = factoryhttpapi.HumanMerges{Service: s.FactoryApplication().HumanMerges(writer)}
	product.Handler().ServeHTTP(w, r)
}

func (s *API) apiFactoryHumanPullCreates(w http.ResponseWriter, r *http.Request, v store.Session) {
	product, source, ok := s.collaborationReads(w, r, v)
	if !ok {
		return
	}
	writer := forgejo.HumanPullCreates{Authority: source.Authority, Identities: source.Identities}
	product.HumanPullCreates = factoryhttpapi.HumanPullCreates{Service: s.FactoryApplication().HumanPullCreates(writer)}
	product.Handler().ServeHTTP(w, r)
}

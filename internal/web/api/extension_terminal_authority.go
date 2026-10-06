package api

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strconv"
	"strings"
	"time"

	extensions "forgejo.org/extension-sdk"
	"github.com/levitateos/sodaos/internal/project"
	"github.com/levitateos/sodaos/internal/store"
	"github.com/levitateos/sodaos/internal/web/auth"
)

type extensionTerminalIdentity struct {
	authority extensions.Authority
	project   store.Project
	login     string
	actorID   int64
}

func nativeTerminalScope(generation string) string {
	sum := sha256.Sum256([]byte("native-terminal\x00" + generation))
	return hex.EncodeToString(sum[:])
}

func extensionTerminalGeneration(r *http.Request, authority extensions.Authority) bool {
	values := r.Header.Values(extensions.SessionGenerationHeader)
	return len(values) == 1 && values[0] != "" && values[0] == authority.SessionGeneration
}

func extensionTerminalOrigin(r *http.Request, origin string) bool {
	values := r.Header.Values("Origin")
	fetch := r.Header.Values("Sec-Fetch-Site")
	return len(values) == 1 && values[0] == origin && strings.HasPrefix(origin, "https://") &&
		len(fetch) <= 1 && (len(fetch) == 0 || fetch[0] == "same-origin")
}

func nativeTerminalContribution(authority extensions.Authority) bool {
	return extensionPageContribution(authority.Contribution, "spaces") || extensionWorkspacePanelContribution(authority.Contribution)
}

func nativeTerminalMember(p store.Project, login string, err error) bool {
	return err == nil && p.Ready && login != "root" && project.ValidLogin(login)
}

func (s *API) nativeTerminalActor(authority extensions.Authority) (int64, bool) {
	id, ok := auth.PositiveID(authority.Actor.ID)
	return id, ok && s.Store != nil
}

func (s *API) extensionTerminalAccount(w http.ResponseWriter, r *http.Request, generationHeader bool) (extensionTerminalIdentity, bool) {
	var identity extensionTerminalIdentity
	authority, err := auth.ExtensionAuthority(r)
	if err != nil || !nativeTerminalContribution(authority) {
		auth.JSONError(w, 403, "native_authority_unavailable", "Current native terminal authority is required.")
		return identity, false
	}
	if generationHeader && !extensionTerminalGeneration(r, authority) {
		auth.JSONError(w, 403, "session_changed", "Reload the current native session.")
		return identity, false
	}
	actorID, ok := s.nativeTerminalActor(authority)
	if !ok {
		auth.JSONError(w, 403, "invalid_actor", "Native actor is invalid.")
		return identity, false
	}
	p, err := s.Store.Project(r.Context(), r.PathValue("id"))
	if err != nil {
		auth.JSONError(w, 404, "not_found", "Environment not found.")
		return identity, false
	}
	login, err := s.Store.MemberLogin(r.Context(), p.ID, actorID)
	if !nativeTerminalMember(p, login, err) {
		auth.JSONError(w, 403, "membership_required", "An existing provisioned account is required.")
		return identity, false
	}
	identity = extensionTerminalIdentity{authority: authority, project: p, login: login, actorID: actorID}
	if !s.extensionTerminalRepository(r.Context(), identity) {
		auth.JSONError(w, 403, "repository_write_required", "Current repository write permission is required.")
		return extensionTerminalIdentity{}, false
	}
	return identity, true
}

func sameNativeTerminalAuthority(current, original extensions.Authority) bool {
	return current.Actor == original.Actor && current.SessionGeneration == original.SessionGeneration &&
		current.InstanceID == original.InstanceID && current.Contribution == original.Contribution
}

func nativeTerminalMembershipCurrent(ctx context.Context, login, expected string, err error) bool {
	return ctx.Err() == nil && err == nil && login == expected
}

func (s *API) extensionTerminalRepository(ctx context.Context, identity extensionTerminalIdentity) bool {
	check, done := context.WithTimeout(ctx, 5*time.Second)
	defer done()
	repo, err := identity.authority.Native().Repository(check, strconv.FormatInt(identity.project.RepositoryID, 10))
	return check.Err() == nil && err == nil && repo.ID == strconv.FormatInt(identity.project.RepositoryID, 10) &&
		(repo.Permission == "write" || repo.Permission == "admin")
}

func (s *API) extensionTerminalCurrent(ctx context.Context, r *http.Request, identity extensionTerminalIdentity) bool {
	if ctx.Err() != nil {
		return false
	}
	check, done := context.WithTimeout(ctx, 5*time.Second)
	defer done()
	current, err := auth.ExtensionAuthority(r.WithContext(check))
	if err != nil || !sameNativeTerminalAuthority(current, identity.authority) {
		return false
	}
	p, err := s.Store.Project(check, identity.project.ID)
	if err != nil || !p.Ready || p.RepositoryID != identity.project.RepositoryID {
		return false
	}
	login, err := s.Store.MemberLogin(check, p.ID, identity.actorID)
	return nativeTerminalMembershipCurrent(check, login, identity.login, err) && s.extensionTerminalRepository(check, identity)
}

func nativeTerminalSessionActor(authority extensions.Authority, present, valid bool, v store.Session) bool {
	return present && valid && authority.Actor.ID == strconv.FormatInt(v.User.ID, 10) && authority.Actor.Username == v.User.Login
}

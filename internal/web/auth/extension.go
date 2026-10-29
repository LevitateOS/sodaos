package auth

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	extensions "forgejo.org/extension-sdk"
)

// ExtensionAuthority verifies the SDK context against the live native admission
// through the host callback, including its method and current actor. Host
// admission and revocation remain host-owned.
func ExtensionAuthority(r *http.Request) (extensions.Authority, error) {
	if len(r.Header.Values(extensions.ContextHeader)) != 1 || len(r.Header.Values(extensions.AdmissionHeader)) != 1 {
		return extensions.Authority{}, errors.New("one native authority and admission are required")
	}
	authority, err := extensions.RequestContext(r)
	if err != nil {
		return extensions.Authority{}, fmt.Errorf("read native authority: %w", err)
	}
	if !ExtensionContribution(authority.ExtensionID, authority.Contribution) {
		return extensions.Authority{}, errors.New("native authority is for another extension")
	}
	if authority.Contribution.Action != strings.ToLower(r.Method) {
		return extensions.Authority{}, errors.New("native authority is for another method")
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	actor, err := authority.Native().CurrentActor(ctx)
	if err != nil {
		return extensions.Authority{}, fmt.Errorf("verify native actor: %w", err)
	}
	if actor != authority.Actor {
		return extensions.Authority{}, errors.New("native actor changed")
	}
	return authority, nil
}

// ExtensionContribution binds each declared contribution to its exact package.
func ExtensionContribution(packageID string, contribution extensions.Contribution) bool {
	if packageID == "tailnet" {
		return contribution.Kind == "page" && contribution.ID == "tailnet" && contribution.Scope == "admin"
	}
	if packageID != "soda" {
		return false
	}
	return contribution.Kind == "page" && (contribution.ID == "spaces" || contribution.ID == "feed" || contribution.ID == "applications") && contribution.Scope == "global" ||
		contribution.Kind == "page" && (contribution.ID == "repository-feed" || contribution.ID == "repository-settings") && contribution.Scope == "repository" ||
		contribution.Kind == "panel" && (contribution.ID == "workspace" || contribution.ID == "providers") && contribution.Scope == "panel"
}

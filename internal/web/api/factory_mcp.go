package api

// The appliance composes the shared role-filtered Factory MCP server with its
// persistent key, store and current runtime/service ports. The shared server
// owns protocol routing; durable work remains in canonical feed posts.

import (
	"net/http"

	"github.com/levitateos/factory/pkg/factory/control"
	factoryhttpapi "github.com/levitateos/factory/pkg/factory/httpapi"
	factorymcp "github.com/levitateos/factory/pkg/factory/mcpserver"
)

// FactoryMCPHandler exposes the catalog on the private listener.
// It refuses everything when the store is unavailable: transport
// without persistence must not admit calls.
func (s *API) FactoryMCPHandler() http.Handler {
	if s == nil || s.Store == nil {
		return factoryhttpapi.Unavailable()
	}
	if s.Config == nil {
		return factoryhttpapi.Unavailable()
	}
	key, err := control.LoadFactoryMCPKey(s.Config.FactoryPublicationRoot)
	if err != nil {
		return factoryhttpapi.Unavailable()
	}
	return factorymcp.NewHandler(factorymcp.Dependencies{
		IssueMaker:           s.factoryIssueMakerTool,
		Capabilities:         s.Store,
		Submissions:          s.Store,
		Assignments:          s.Store,
		PriorityJudgments:    s.Store,
		Runs:                 s.Store,
		Prepare:              s.prepareFactoryWorkerMCP,
		SystemOne:            &control.TypeSafeSystemOne{APIKeyFile: s.Config.TypeSafeAPIKeyFile, Model: s.Config.TypeSafeModel},
		VerificationEvidence: s.factoryVerificationEvidence,
		RepairPreview:        s.factoryRepairPreview,
		Catalog:              s.factoryApprovedVerificationCatalog,
		NativeTask:           s.factoryNativeTask,
	}, key)
}

package api

import (
	"context"
	"net/http"
	"time"

	"github.com/levitateos/sodaos/internal/project"
	"github.com/levitateos/sodaos/internal/web/auth"
)

const maintenanceInventoryLimit = 128

type maintenanceProject struct {
	ID    string `json:"id"`
	Ready bool   `json:"ready"`
}

type maintenanceInventory struct {
	Projects        []maintenanceProject `json:"projects"`
	FactoryWork     int                  `json:"factory_work"`
	PreparationWork int                  `json:"preparation_work"`
}

// ProjectMaintenanceHandler exposes read-only durable enrollment facts only to
// the private, OS-peer-gated operator listener. It is never on the public mux.
// Native identities and runtime work must be checked separately under admission
// exclusion; a database association is not evidence that its root exists.
func (s *API) ProjectMaintenanceHandler() http.Handler {
	return http.HandlerFunc(s.projectMaintenanceInventory)
}

func (s *API) projectMaintenanceInventory(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		auth.JSONError(w, http.StatusMethodNotAllowed, "method_not_allowed", "GET required.")
		return
	}
	if r.URL.RawQuery != "" || r.URL.RawPath != "" {
		auth.JSONError(w, http.StatusBadRequest, "invalid_inventory_request", "Inventory takes no parameters.")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	if s.Store == nil {
		auth.JSONError(w, http.StatusServiceUnavailable, "maintenance_inventory_unavailable", "Project enrollment could not be inspected.")
		return
	}
	projects, err := s.Store.SpaceProjectsAfter(ctx, "", maintenanceInventoryLimit+1)
	if err != nil {
		auth.JSONError(w, http.StatusServiceUnavailable, "maintenance_inventory_unavailable", "Project enrollment could not be inspected.")
		return
	}
	if len(projects) > maintenanceInventoryLimit {
		auth.JSONError(w, http.StatusServiceUnavailable, "maintenance_inventory_limit", "Project inventory exceeds the supported maintenance bound.")
		return
	}
	out := maintenanceInventory{Projects: make([]maintenanceProject, 0, len(projects))}
	for _, p := range projects {
		if !project.ValidID(p.ID) {
			auth.JSONError(w, http.StatusServiceUnavailable, "maintenance_inventory_unavailable", "Project enrollment could not be inspected.")
			return
		}
		out.Projects = append(out.Projects, maintenanceProject{ID: p.ID, Ready: p.Ready})
	}
	out.FactoryWork, out.PreparationWork, err = s.Store.ProjectMaintenanceWork(ctx)
	if err != nil {
		auth.JSONError(w, http.StatusServiceUnavailable, "maintenance_inventory_unavailable", "Managed work could not be inspected.")
		return
	}
	if out.FactoryWork > maintenanceInventoryLimit || out.PreparationWork > maintenanceInventoryLimit {
		auth.JSONError(w, http.StatusServiceUnavailable, "maintenance_inventory_limit", "Managed work exceeds the supported maintenance bound.")
		return
	}
	auth.JSONResponse(w, http.StatusOK, out)
}

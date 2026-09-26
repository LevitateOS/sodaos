// Package runners implements Soda's focused local CI runner composition.
// Providers remain authoritative for registration, labels, workflows,
// scheduling, results, and history.
package runners

import (
	"errors"
	"regexp"
)

const (
	DefaultRootPath   = "/var/lib/soda/runners"
	DefaultLockPath   = "/run/lock/soda/runners.lock"
	RunnerCapacity    = 1
	BundledForgejoURL = "http://127.0.0.1:3000"

	ProviderForgejo Provider = "forgejo"
)

var runnerIDPattern = regexp.MustCompile(`^[a-z][a-z0-9-]{0,15}$`)

var ErrUnavailable = errors.New("local CI execution is unavailable until isolated jobs are supported")

type Provider string

type CreateRequest struct {
	ID                string   `json:"id"`
	Provider          Provider `json:"provider"`
	RegistrationURL   string   `json:"registration_url"`
	RegistrationID    string   `json:"registration_id"`
	Labels            string   `json:"labels"`
	RegistrationToken string   `json:"registration_token"`
}

type RunnerRequest struct {
	ID string `json:"id"`
}

type EmptyRequest struct{}

type Descriptor struct {
	ID              string   `json:"id"`
	Provider        Provider `json:"provider"`
	RegistrationURL string   `json:"registration_url"`
	Account         string   `json:"account"`
	Architecture    string   `json:"architecture"`
}

type ServiceState struct {
	Load    string `json:"load"`
	Active  string `json:"active"`
	Sub     string `json:"sub"`
	Enabled string `json:"enabled"`
}

type RunnerView struct {
	Descriptor
	Version  string        `json:"version"`
	Capacity int           `json:"capacity"`
	Service  *ServiceState `json:"service"`
}

// Unavailable holds only validated directory IDs, never unsafe descriptors or diagnostics.
// A readable descriptor may still have an unknown service (nil) or version (empty).
type Inventory struct {
	Runners     []RunnerView `json:"runners"`
	Unavailable []string     `json:"unavailable"`
}

type ListResponse struct {
	Inventory
	ForgejoURL      string `json:"forgejo_url,omitempty"`
	Complete        bool   `json:"complete"`
	RunnerCount     int    `json:"runner_count"`
	ActiveListeners int    `json:"active_listeners"`
	TotalCapacity   int    `json:"total_capacity"`
}

func (inventory Inventory) Response(origin string) ListResponse {
	result := ListResponse{Inventory: inventory, ForgejoURL: origin, Complete: len(inventory.Unavailable) == 0, RunnerCount: len(inventory.Runners), TotalCapacity: len(inventory.Runners) * RunnerCapacity}
	for _, row := range inventory.Runners {
		if row.Service == nil || row.Version == "" {
			result.Complete = false
		}
		if row.Service != nil && row.Service.Active == "active" && row.Service.Sub == "running" {
			result.ActiveListeners++
		}
	}
	return result
}

type MutationResponse struct {
	OK bool `json:"ok"`
}

func (CreateRequest) Validate() error { return ErrUnavailable }

func ValidateID(id string) error {
	if !runnerIDPattern.MatchString(id) {
		return errors.New("runner id must match [a-z][a-z0-9-]{0,15}")
	}
	return nil
}

func AccountName(id string) (string, error) {
	if err := ValidateID(id); err != nil {
		return "", err
	}
	return "soda-runner-" + id, nil
}

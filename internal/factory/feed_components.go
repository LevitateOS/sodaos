package factory

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"slices"
)

// FeedComponent describes one semantic presentation from the design catalog.
// This catalog is shared with the browser. A component identifies the meaning
// of a report; it never grants permission to apply the reported change.
type FeedComponent struct {
	ID           string   `json:"id"`
	Label        string   `json:"label"`
	Family       string   `json:"family"`
	Presentation string   `json:"presentation"`
	Roles        []string `json:"roles"`
	Outlets      []string `json:"outlets"`
	Authority    string   `json:"authority"`
	Availability string   `json:"availability"`
}

//go:embed feed_components.json
var feedComponentJSON []byte

var feedComponents = readFeedComponents()

func readFeedComponents() []FeedComponent {
	var catalog []FeedComponent
	if err := json.Unmarshal(feedComponentJSON, &catalog); err != nil {
		panic(fmt.Sprintf("invalid embedded feed component catalog: %v", err))
	}
	return catalog
}

// FeedComponents returns the shared presentation catalog. The catalog includes
// controller-owned events and Library Admin presentations; their
// presence does not advertise an available worker capability.
func FeedComponents() []FeedComponent {
	out := make([]FeedComponent, len(feedComponents))
	for i, component := range feedComponents {
		component.Roles = slices.Clone(component.Roles)
		component.Outlets = slices.Clone(component.Outlets)
		out[i] = component
	}
	return out
}

func FeedComponentByID(id string) (FeedComponent, bool) {
	for _, component := range feedComponents {
		if component.ID == id {
			return component, true
		}
	}
	return FeedComponent{}, false
}

// ValidateFeedComponent checks the report's role and semantic outlet. Omitted
// IDs use the outlet's ordinary presentation. Factory effects and Library
// Admin mutations are never created by a worker choosing their card shape.
func ValidateFeedComponent(id string, profile MCPProfile, tool MCPTool) error {
	if id == "" {
		return nil
	}
	component, ok := FeedComponentByID(id)
	if !ok {
		return fmt.Errorf("unknown feed component %q", id)
	}
	worker := component.Authority == "worker" && component.Availability == "worker"
	library := component.Authority == "library" && component.Availability == "library" && profile == MCPProfile("library_admin")
	if !worker && !library {
		return fmt.Errorf("feed component %s requires its %s owner", id, component.Authority)
	}
	if !slices.Contains(MCPProfiles, profile) ||
		(!slices.Contains(component.Roles, "*") && !slices.Contains(component.Roles, string(profile))) {
		return fmt.Errorf("feed component %s is unavailable to %s", id, profile)
	}
	if !slices.Contains(component.Outlets, string(tool)) {
		return fmt.Errorf("feed component %s is not produced by %s", id, tool)
	}
	return nil
}

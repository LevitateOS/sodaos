package api

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"maps"
	"sort"
	"time"

	"github.com/levitateos/sodaos/internal/project"
	"github.com/levitateos/sodaos/internal/store"
)

type preparationInputView struct {
	Path   string `json:"path"`
	Digest string `json:"digest"`
}

type preparationRequirementPreview struct {
	SourceCommit string                 `json:"source_commit"`
	SetupDigest  string                 `json:"setup_digest"`
	InputsDigest string                 `json:"inputs_digest"`
	SourceInputs []preparationInputView `json:"source_inputs"`
	Predecessor  string                 `json:"predecessor"`
}

type preparationApprovalPreview struct {
	Requirement     string                `json:"requirement"`
	EffectsDigest   string                `json:"effects_digest"`
	ReadinessDigest string                `json:"readiness_digest"`
	Predecessor     string                `json:"predecessor"`
	Roles           []preparationItemView `json:"roles"`
}

// A preview observes a previously accepted selection; it neither selects new
// requirements nor verifies privileged effects on an administrator's behalf.
// Logs and repository contents stay outside the browser response.
type preparationDecisionPreview struct {
	Requirements *preparationRequirementPreview `json:"requirements,omitempty"`
	Approval     *preparationApprovalPreview    `json:"approval,omitempty"`
	Reason       string                         `json:"reason,omitempty"`
}

func matchPreparationRecord(record project.StoredPreparation, role string, d project.RequirementDecision) bool {
	prep := record.Preparation
	return prep.Role == role && record.State.Ready && !record.State.Stopped && prep.SourceCommit == d.SourceCommit &&
		prep.SetupDigest == d.SetupDigest && prep.Requirements.SourceCommit == d.SourceCommit &&
		prep.Requirements.Digest == d.InputsDigest && maps.Equal(prep.Requirements.SourceInputs, d.SourceInputs)
}

func matchPreparationState(state project.PrepareState, role string, prep project.Preparation) bool {
	return state.Role == role && state.SourceCommit == prep.SourceCommit && state.SetupDigest == prep.SetupDigest &&
		state.Ready && !state.Stopped && state.Phase == project.PrepareReady && state.Missing == "" &&
		state.SetupExit != nil && *state.SetupExit == 0 && state.CheckExit != nil && *state.CheckExit == 0
}

func (s *API) observePreparationRole(ctx context.Context, p store.Project, d project.RequirementDecision, inputs []preparationInputView, records []project.StoredPreparation, role string) *preparationItemView {
	for _, record := range records {
		if !matchPreparationRecord(record, role, d) {
			continue
		}
		prep := record.Preparation
		state, readErr := s.Host.InspectPreparation(ctx, project.PrepareInspect{Project: p.ID, ID: prep.ID})
		if readErr != nil || !matchPreparationState(state, role, prep) {
			continue
		}
		if !s.confirmPreparationInputs(ctx, prep, inputs) {
			continue
		}
		item := preparationItemDTO(project.StoredPreparation{Preparation: prep, State: state})
		item.Tools = append([]project.ResolvedTool(nil), state.Tools...)
		sort.Slice(item.Tools, func(i, j int) bool { return item.Tools[i].Name < item.Tools[j].Name })
		return &item
	}
	return nil
}

func (s *API) preparationDecisionPreview(ctx context.Context, p store.Project, requirement, approval string) preparationDecisionPreview {
	missing := func(reason string) preparationDecisionPreview { return preparationDecisionPreview{Reason: reason} }
	if requirement == "" {
		return missing("A maintainer must first accept the complete setup inputs through native Project preparation.")
	}
	d, err := s.Store.RequirementDecision(ctx, requirement)
	if err != nil || d.Project != p.ID || d.Validate() != nil || len(d.SourceInputs) == 0 {
		return missing("The accepted requirement does not identify a complete selection of repository inputs.")
	}
	if s.Host == nil {
		return missing("Native Project preparation is unavailable.")
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	records, err := s.Store.StandingProjectPreparations(ctx, p.ID)
	if err != nil || len(records) > 128 {
		return missing("Standing Project preparation could not be confirmed.")
	}
	inputs := make([]preparationInputView, 0, len(d.SourceInputs))
	for name, digest := range d.SourceInputs {
		inputs = append(inputs, preparationInputView{Path: name, Digest: digest})
	}
	sort.Slice(inputs, func(i, j int) bool { return inputs[i].Path < inputs[j].Path })
	roles := []preparationItemView{}
	for _, role := range []string{project.RoleCoder, project.RoleReviewer} {
		observed := s.observePreparationRole(ctx, p, d, inputs, records, role)
		if observed == nil {
			return missing("Ready native " + role + " preparation and its exact accepted input bytes must be confirmed before reviewing decisions.")
		}
		roles = append(roles, *observed)
	}
	// The administrator separately reviews live service/resource effects. This
	// fingerprint binds the selected setup, Project profile and observed tools;
	// Ready and successful exit codes form the distinct readiness fingerprint.
	effects := preparationPreviewDigest(struct {
		Project string                   `json:"project"`
		Profile *project.Profile         `json:"profile"`
		Inputs  []preparationInputView   `json:"inputs"`
		Setup   string                   `json:"setup"`
		Tools   [][]project.ResolvedTool `json:"tools"`
	}{p.ID, p.Profile, inputs, d.SetupDigest, [][]project.ResolvedTool{roles[0].Tools, roles[1].Tools}})
	readiness := preparationPreviewDigest(roles)
	return preparationDecisionPreview{
		Requirements: &preparationRequirementPreview{SourceCommit: d.SourceCommit, SetupDigest: d.SetupDigest, InputsDigest: d.InputsDigest, SourceInputs: inputs, Predecessor: requirement},
		Approval:     &preparationApprovalPreview{Requirement: requirement, EffectsDigest: effects, ReadinessDigest: readiness, Predecessor: approval, Roles: roles},
	}
}

func (s *API) confirmPreparationInputs(ctx context.Context, prep project.Preparation, inputs []preparationInputView) bool {
	// The native reader also selects AGENTS.md; leave a slot within its existing
	// bounded text context. Larger or non-text inputs require native inspection.
	const chunk = project.MaxFactoryContextFiles - 1
	for start := 0; start < len(inputs); start += chunk {
		end := min(start+chunk, len(inputs))
		paths := make([]string, 0, end-start)
		for _, input := range inputs[start:end] {
			paths = append(paths, input.Path)
		}
		deadline, _ := ctx.Deadline()
		request := project.FactoryPreparationContextRequest{Project: prep.Project, ID: prep.ID, SourceCommit: prep.SourceCommit, ApprovedBase: prep.SourceCommit, DiffBase: prep.SourceCommit, Candidate: prep.SourceCommit, Paths: paths, NotAfter: deadline}
		material, err := s.Host.ReadPreparationContext(ctx, request, prep.Role)
		if err != nil {
			return false
		}
		files := make(map[string][]byte, len(material.Files))
		for _, file := range material.Files {
			if _, duplicate := files[file.Path]; duplicate {
				return false
			}
			files[file.Path] = file.Content
		}
		for _, input := range inputs[start:end] {
			content, found := files[input.Path]
			if !found {
				return false
			}
			digest := sha256.Sum256(content)
			if hex.EncodeToString(digest[:]) != input.Digest {
				return false
			}
		}
	}
	return true
}

func preparationPreviewDigest(value any) string {
	data, _ := json.Marshal(value)
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:])
}

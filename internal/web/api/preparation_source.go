package api

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"maps"
	"net/http"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	extensions "forgejo.org/extension-sdk"
	"github.com/levitateos/sodaos/internal/project"
	"github.com/levitateos/sodaos/internal/store"
	"github.com/levitateos/sodaos/internal/web/auth"
)

type preparationSourceDraft struct {
	source            extensions.RepositorySource
	setup             project.ApprovedSetup
	requirements      preparationRequirementPreview
	setupFiles, tools []string
}

func uniquePreparationNames(names []string, max int, valid func(string) bool) ([]string, error) {
	if len(names) > max {
		return nil, errors.New("selection exceeds bounds")
	}
	result := append([]string{}, names...)
	sort.Strings(result)
	for i, name := range result {
		if !valid(name) || i > 0 && name == result[i-1] {
			return nil, errors.New("invalid selection")
		}
	}
	return result, nil
}

// preparationSourceDraft observes native Git objects, never a human checkout
// or caller-supplied digests. Protected setup files are an explicit subset of
// the maintainer's complete effective input selection.
func (s *API) readPreparationSource(ctx context.Context, r *http.Request, p store.Project, in preparationActionRequest, actors []string) (preparationSourceDraft, error) {
	var draft preparationSourceDraft
	setupFiles, err := uniquePreparationNames(in.SetupFiles, project.MaxApprovedFiles, project.ValidApprovedName)
	if err != nil || !slices.Contains(setupFiles, project.FactorySetupEntry) || !slices.Contains(setupFiles, project.FactoryCheckEntry) {
		return draft, errors.New("select protected setup.sh and check.sh inputs")
	}
	tools, err := uniquePreparationNames(in.RequiredTools, project.MaxPrepareTools, project.ValidToolName)
	if err != nil {
		return draft, err
	}
	paths := append([]string(nil), in.SourcePaths...)
	sort.Strings(paths)
	if len(paths) == 0 {
		return draft, errors.New("select complete effective repository inputs")
	}
	for _, name := range setupFiles {
		if !slices.Contains(paths, name) {
			return draft, errors.New("protected setup files must be selected repository inputs")
		}
	}
	authority, ok := requestExtensionAuthority(r)
	if !ok {
		return draft, errors.New("native source authority unavailable")
	}
	bounded, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	source, err := authority.RepositorySource(bounded, strconv.FormatInt(p.RepositoryID, 10), in.SourceCommit, paths, actors)
	if err != nil {
		return draft, err
	}
	setup := project.ApprovedSetup{Files: make(map[string][]byte), Bundle: source.Bundle}
	inputs := make([]preparationInputView, 0, len(paths))
	inputDigests := make(map[string]string, len(paths))
	for _, name := range paths {
		contents := source.Files[name]
		digest := sha256.Sum256(contents)
		inputs = append(inputs, preparationInputView{Path: name, Digest: hex.EncodeToString(digest[:])})
		inputDigests[name] = hex.EncodeToString(digest[:])
		if slices.Contains(setupFiles, name) {
			setup.Files[name] = contents
		}
	}
	if err = setup.Validate(); err != nil {
		return draft, err
	}
	if err = project.ValidateSourceInputs(inputDigests); err != nil {
		return draft, err
	}
	head, err := s.Store.RequirementHead(ctx, p.ID)
	if errors.Is(err, store.ErrNotFound) {
		head, err = "", nil
	}
	if err != nil {
		return draft, err
	}
	draft = preparationSourceDraft{source: source, setup: setup, setupFiles: setupFiles, tools: tools,
		requirements: preparationRequirementPreview{SourceCommit: source.Commit, SetupDigest: project.SetupDigestOf(setup.Files),
			InputsDigest: project.PreparationInputsDigestOf(setupFiles, inputDigests, tools), SourceInputs: inputs, Predecessor: head}}
	return draft, nil
}

func (s *API) apiPreparationSourcePreview(w http.ResponseWriter, r *http.Request, v store.Session, p store.Project, in preparationActionRequest) {
	if _, err := s.visibleRepository(r, v, p.RepositoryID); err != nil {
		auth.ProviderError(w, err)
		return
	}
	draft, err := s.readPreparationSource(r.Context(), r, p, in, nil)
	if err != nil {
		auth.JSONError(w, 422, "source_unavailable", "The selected approved source and protected setup inputs could not be confirmed.")
		return
	}
	if !s.checkLifecycleSession(w, r, v) {
		return
	}
	type sourceFile struct {
		Path    string `json:"path"`
		Content string `json:"content"`
		Bytes   int    `json:"bytes"`
	}
	files := make([]sourceFile, 0, len(draft.requirements.SourceInputs))
	for _, input := range draft.requirements.SourceInputs {
		contents := draft.source.Files[input.Path]
		file := sourceFile{Path: input.Path, Bytes: len(contents)}
		if utf8.Valid(contents) {
			file.Content = string(contents)
		}
		files = append(files, file)
	}
	auth.JSONResponse(w, 200, struct {
		Requirements  preparationRequirementPreview `json:"requirements"`
		Files         []sourceFile                  `json:"files"`
		RequiredTools []string                      `json:"required_tools"`
		SetupFiles    []string                      `json:"setup_files"`
	}{draft.requirements, files, draft.tools, draft.setupFiles})
}

// preparationAdministrator observes the current native Project privilege.
// The configured operator is the existing explicit appliance exception.
func (s *API) preparationAdministrator(ctx context.Context, p store.Project, actor int64) error {
	if actor == s.Config.OperatorID {
		return nil
	}
	login, err := s.Store.MemberLogin(ctx, p.ID, actor)
	if err != nil {
		return err
	}
	status, err := s.Host.ProjectAccess(ctx, project.ProjectAccessRequest{Project: p.ID, Login: login, Identity: actor})
	if err != nil {
		return err
	}
	if status.Project != p.ID || status.Login != login || status.Identity != actor || status.Administrator == nil || !*status.Administrator {
		return errors.New("native Project administrator required")
	}
	return nil
}

// Initial approval records an administrator's own observations of installed
// prerequisites and assigned resources. It does not claim role setup passed;
// the separate preparation action produces that native evidence.
func (s *API) apiPreparationInitialApproval(w http.ResponseWriter, r *http.Request, v store.Session, p store.Project, in preparationActionRequest) {
	if !in.Verified || strings.TrimSpace(in.EffectsEvidence) == "" || strings.TrimSpace(in.ReadinessEvidence) == "" || len(in.EffectsEvidence) > 8192 || len(in.ReadinessEvidence) > 8192 {
		auth.JSONError(w, 400, "inspection_required", "Describe the installed prerequisites, assigned resources and native readiness inspection you verified.")
		return
	}
	if !s.checkLifecycleSession(w, r, v) {
		return
	}
	if err := s.preparationAdministrator(r.Context(), p, v.User.ID); err != nil {
		auth.JSONError(w, 403, "project_administrator_required", "Current native Project administrator authority is required.")
		return
	}
	requirement, err := s.Store.RequirementDecision(r.Context(), in.Requirement)
	if err != nil || requirement.Project != p.ID {
		auth.JSONError(w, 409, "stale_requirement", "Refresh the accepted requirements before approving.")
		return
	}
	bind := struct {
		Project     string                      `json:"project"`
		Profile     *project.Profile            `json:"profile"`
		Requirement project.RequirementDecision `json:"requirement"`
		Kind        string                      `json:"kind"`
		Evidence    string                      `json:"evidence"`
	}{p.ID, p.Profile, requirement, "installed-effects", in.EffectsEvidence}
	in.EffectsDigest = preparationPreviewDigest(bind)
	bind.Kind, bind.Evidence = "native-readiness-inspection", in.ReadinessEvidence
	in.ReadinessDigest = preparationPreviewDigest(bind)
	s.apiPreparationActionApprove(w, r, v, p, in)
}

func draftSourceInputs(draft preparationSourceDraft) map[string]string {
	inputs := make(map[string]string, len(draft.requirements.SourceInputs))
	for _, input := range draft.requirements.SourceInputs {
		inputs[input.Path] = input.Digest
	}
	return inputs
}

func (s *API) apiPreparationStart(w http.ResponseWriter, r *http.Request, v store.Session, p store.Project, in preparationActionRequest) {
	if !project.ValidPreparationID(in.CoderID) || !project.ValidPreparationID(in.ReviewerID) || in.CoderID == in.ReviewerID {
		auth.JSONError(w, 400, "invalid_preparation", "Distinct coding and review preparation identities are required.")
		return
	}
	if err := s.preparationAdministrator(r.Context(), p, v.User.ID); err != nil {
		auth.JSONError(w, 403, "project_administrator_required", "Current native Project administrator authority is required.")
		return
	}
	requirement, err := s.Store.RequirementDecision(r.Context(), in.Requirement)
	if err != nil || requirement.Project != p.ID {
		auth.JSONError(w, 409, "stale_requirement", "Refresh the accepted requirements before preparing.")
		return
	}
	approval, err := s.Store.ApprovalDecision(r.Context(), in.Approval)
	if err != nil || approval.Project != p.ID || approval.Requirement != requirement.ID || !approval.Verified {
		auth.JSONError(w, 409, "stale_approval", "Refresh the separate administrator approval before preparing.")
		return
	}
	in.SourceCommit = requirement.SourceCommit
	actors := []string{strconv.FormatInt(requirement.Approver, 10)}
	draft, err := s.readPreparationSource(r.Context(), r, p, in, actors)
	if err != nil || requirement.SetupDigest != draft.requirements.SetupDigest || requirement.InputsDigest != draft.requirements.InputsDigest || !maps.Equal(requirement.SourceInputs, draftSourceInputs(draft)) {
		auth.JSONError(w, 409, "changed_inputs", "The selected source inputs differ from the accepted requirements. Refresh and accept them explicitly.")
		return
	}
	requirementDepth, err := s.Store.RequirementDepth(r.Context(), p.ID)
	if err != nil {
		auth.JSONError(w, 503, "store_unavailable", "Could not read accepted requirements.")
		return
	}
	approvalDepth, err := s.Store.ApprovalDepth(r.Context(), p.ID)
	if err != nil {
		auth.JSONError(w, 503, "store_unavailable", "Could not read administrator approval.")
		return
	}
	authorize := func(ctx context.Context) error {
		if !s.extensionSessionCurrent(ctx, r, v) {
			return errors.New("native session changed")
		}
		if err := s.preparationAdministrator(ctx, p, v.User.ID); err != nil {
			return err
		}
		if err := s.preparationAdministrator(ctx, p, approval.Approver); err != nil {
			return err
		}
		observed, err := s.readPreparationSource(ctx, r, p, in, actors)
		if err != nil {
			return err
		}
		if observed.source.Commit != requirement.SourceCommit || !observed.source.Writers[actors[0]] || observed.requirements.InputsDigest != requirement.InputsDigest || observed.requirements.SetupDigest != requirement.SetupDigest {
			return errors.New("accepted inputs or maintainer authority changed")
		}
		if !s.extensionSessionCurrent(ctx, r, v) {
			return errors.New("native session changed")
		}
		return nil
	}
	items := []preparationItemView{}
	for _, selected := range []struct{ id, role string }{{in.CoderID, project.RoleCoder}, {in.ReviewerID, project.RoleReviewer}} {
		prep := project.Prepare{Preparation: project.Preparation{ID: selected.id, Project: p.ID, Role: selected.role,
			Requirements: requirement.Ref(requirementDepth), Approval: approval.Ref(approvalDepth),
			SourceCommit: requirement.SourceCommit, SetupDigest: requirement.SetupDigest, Tools: draft.tools}, Setup: draft.setup}
		observed, err := s.Coordinator.StartPreparation(r.Context(), prep, authorize)
		if err != nil {
			auth.JSONError(w, 409, "preparation_unconfirmed", "Preparation was not confirmed. Refresh its retained state and retry these same identities; failed or interrupted identities need a new preparation.")
			return
		}
		items = append(items, preparationItemDTO(observed))
	}
	auth.JSONResponse(w, 200, struct {
		Preparations []preparationItemView `json:"preparations"`
	}{items})
}

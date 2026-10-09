package control

import (
	"context"
	"errors"
	"path"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/levitateos/sodaos/internal/project"
)

var repositoryPathReference = regexp.MustCompile(`(?:[A-Za-z0-9._-]+/)+[A-Za-z0-9._-]+|[A-Za-z0-9_-]+\.(?:go|rs|py|js|ts|tsx|jsx|java|kt|c|h|cpp|md|toml|yaml|yml|json|sh|sql|html|css|lock)`)

func readRepositoryContext(ctx context.Context, host DispatchHost, plan *attemptPlan, preparation project.Preparation, role string, approvedBase, diffBase, candidate string, requirePreparedCandidate bool) (*project.FactoryPreparationContext, error) {
	if host == nil || plan == nil || preparation.Validate() != nil || !time.Now().Before(plan.deadline) {
		return nil, errors.New("native preparation context reader unavailable")
	}
	if requirePreparedCandidate && role == project.RoleCoder && candidate != preparation.SourceCommit {
		return nil, errors.New("initial candidate differs from its prepared source")
	}
	paths, err := acceptedRepositoryPathReferences(plan)
	if err != nil {
		return nil, err
	}
	request := project.FactoryPreparationContextRequest{
		Project: plan.projectID, ID: preparation.ID,
		SourceCommit: preparation.SourceCommit, ApprovedBase: approvedBase,
		DiffBase: diffBase, Candidate: candidate,
		Paths: paths, NotAfter: plan.deadline,
	}
	if err := request.Validate(); err != nil {
		return nil, err
	}
	bounded, cancel := context.WithDeadline(ctx, plan.deadline)
	defer cancel()
	if err := bounded.Err(); err != nil {
		return nil, err
	}
	result, err := host.ReadPreparationContext(bounded, request, role)
	if err != nil {
		return nil, err
	}
	if err = result.ValidateFor(request, role); err != nil {
		return nil, err
	}
	return &result, nil
}

func acceptedRepositoryPathReferences(plan *attemptPlan) ([]string, error) {
	if plan == nil {
		return nil, errors.New("repository context path scope unavailable")
	}
	var texts []string
	texts = append(texts, plan.inputs.Issue.Title, plan.inputs.Issue.Body)
	for _, comment := range plan.inputs.Comments {
		texts = append(texts, comment.Content)
	}
	seen := make(map[string]bool)
	var paths []string
	for _, text := range texts {
		for _, candidate := range repositoryPathReference.FindAllString(text, -1) {
			if strings.HasPrefix(candidate, "../") || strings.HasPrefix(candidate, "./") || strings.HasPrefix(candidate, "/") {
				continue
			}
			clean := path.Clean(candidate)
			if clean == "." || strings.HasPrefix(clean, "../") || seen[clean] {
				continue
			}
			seen[clean] = true
			paths = append(paths, clean)
		}
	}
	sort.Strings(paths)
	if len(paths) > project.MaxFactoryContextPaths {
		return nil, errors.New("accepted repository path references exceed the context bound")
	}
	return paths, nil
}

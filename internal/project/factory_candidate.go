package project

import "errors"

// FactoryCandidate prepares a fresh reviewer checkout of one exact candidate,
// reusing only a ready preparation's protected approved setup. The coordinator
// binds the current requirement/admin decisions before requesting preparation.
// Neither role Git configuration nor its private home crosses this boundary.
// Coder corrections continue in their original checkout.
type FactoryCandidate struct {
	Preparation       Preparation `json:"preparation"`
	SourcePreparation string      `json:"source_preparation"`
	Bundle            []byte      `json:"bundle"`
}

func (p FactoryCandidate) Validate() error {
	if err := p.Preparation.Validate(); err != nil {
		return err
	}
	if p.Preparation.Role != RoleReviewer {
		return errors.New("only review receives a fresh candidate preparation")
	}
	if !ValidPreparationID(p.SourcePreparation) || p.SourcePreparation == p.Preparation.ID {
		return errors.New("candidate preparation requires a fresh identity")
	}
	if len(p.Bundle) == 0 || len(p.Bundle) > MaxSourceBundle {
		return errors.New("candidate source bundle exceeds preparation bounds")
	}
	return nil
}

// FactoryCandidateInspect addresses the checkout through a settled recorded
// run. Role, preparation and container incarnation come from the host receipt.
type FactoryCandidateInspect struct {
	Project string `json:"project"`
	ID      string `json:"id"`
}

func (p FactoryCandidateInspect) Validate() error {
	if !ValidID(p.Project) || !ValidFactoryRunID(p.ID) {
		return errors.New("invalid candidate inspection address")
	}
	return nil
}

// FactoryCandidateState identifies the observed checkout HEAD and whether
// tracked, staged or untracked work differs. Untracked build output may follow
// worktree .gitignore rules; tracked ignore-file changes still count as edits.
// The private .soda-home is excluded; role Git configuration and info/exclude
// cannot hide edits.
type FactoryCandidateState struct {
	ID        string `json:"id"`
	Project   string `json:"project"`
	Container string `json:"container"`
	Candidate string `json:"candidate"`
	Dirty     bool   `json:"dirty"`
}

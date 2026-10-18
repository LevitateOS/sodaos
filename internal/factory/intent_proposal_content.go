package factory

import (
	"errors"

	"github.com/levitateos/sodaos/internal/project"
)

// Exclusion preserves one declined idea's bounded meaning and attributable
// basis in proposal evidence. Facts keep sourced facts only; exclusions
// carry decided rejections so judgments can recognize a declined behavior
// reintroduced by an authored walkthrough without re-reading the revision.
type Exclusion struct {
	Idea        string `json:"idea"`
	Description string `json:"description"`
	Consequence string `json:"consequence"`
	Provenance  string `json:"provenance,omitempty"`
}

// Validate checks exclusion binding, bounded meaning and known provenance.
func (e Exclusion) Validate() error {
	if !project.ValidDecisionID(e.Idea) || !intentText(e.Description, 4000) || !intentText(e.Consequence, 4000) {
		return errors.New("invalid exclusion meaning")
	}
	if e.Provenance != "" && e.Provenance != "human" && e.Provenance != "autonomous" {
		return errors.New("invalid exclusion provenance")
	}
	return nil
}

// ProposalContent binds authored alternatives and walkthroughs to one
// effective intent revision. Facts are sourced from the person's objective
// and corrections; Alternatives are suggested choices; Unknowns stay labeled.
// Declined ideas are excluded by ID and never enter outcomes; Exclusions
// preserves each decline's meaning alongside its ID.
type ProposalContent struct {
	Producer     string               `json:"producer"`
	Supported    bool                 `json:"supported"`
	Intent       string               `json:"intent"`
	Revision     string               `json:"revision"`
	Objective    string               `json:"objective"`
	Facts        []string             `json:"facts"`
	Alternatives []ProductAlternative `json:"alternatives,omitempty"`
	Excluded     []string             `json:"excluded,omitempty"`
	Exclusions   []Exclusion          `json:"exclusions,omitempty"`
	Unknowns     []string             `json:"unknowns"`
}

// ComposeProposalContent authors bounded review content for a draft revision.
// It preserves the full objective, applies corrections to outcomes and keeps
// declined outcomes out of every alternative.
func ComposeProposalContent(r IntentRevision) (ProposalContent, error) {
	if err := r.Validate(); err != nil {
		return ProposalContent{}, err
	}
	if r.State != "draft" {
		return ProposalContent{}, errors.New("proposal content needs a draft revision")
	}
	content := ProposalContent{Producer: AuthoredProducer, Intent: r.Intent, Revision: r.ID, Objective: r.Objective, Facts: []string{r.Objective}}
	for _, idea := range r.Ideas {
		switch idea.Decision {
		case "edit":
			content.Facts = append(content.Facts, idea.Correction)
		case "decline":
			content.Excluded = append(content.Excluded, idea.ID)
			content.Exclusions = append(content.Exclusions, Exclusion{Idea: idea.ID, Description: idea.Description, Consequence: idea.Consequence, Provenance: idea.Provenance})
		}
	}
	set := DescribeAlternatives(r.Objective)
	content.Supported, content.Unknowns = set.Supported, set.Unknowns
	if !set.Supported {
		return content, content.Validate()
	}
	for _, alternative := range set.Alternatives {
		for _, idea := range r.Ideas {
			if idea.Decision == "edit" && len(alternative.Outcomes) < 20 {
				alternative.Outcomes = append(alternative.Outcomes, proposalObjective(idea.Correction))
			}
		}
		content.Alternatives = append(content.Alternatives, alternative)
	}
	return content, content.Validate()
}

// Validate checks revision binding, preserved facts and alternative bounds.
func (c ProposalContent) Validate() error {
	if c.Producer != AuthoredProducer || !project.ValidDecisionID(c.Intent) || !project.ValidDecisionID(c.Revision) || !intentText(c.Objective, 16000) {
		return errors.New("invalid proposal content identity or objective")
	}
	if len(c.Facts) == 0 || len(c.Facts) > 51 || len(c.Excluded) > 50 || len(c.Exclusions) > 50 || len(c.Unknowns) == 0 || len(c.Unknowns) > 8 {
		return errors.New("invalid proposal content bounds")
	}
	if c.Facts[0] != c.Objective {
		return errors.New("proposal content must lead with the full objective")
	}
	if err := checkProposalFacts(c); err != nil {
		return err
	}
	return checkProposalAlternatives(c)
}

func checkProposalFacts(c ProposalContent) error {
	for _, fact := range c.Facts {
		if !intentText(fact, 16000) {
			return errors.New("invalid proposal fact")
		}
	}
	for _, id := range c.Excluded {
		if !project.ValidDecisionID(id) {
			return errors.New("invalid excluded idea")
		}
	}
	if len(c.Exclusions) != len(c.Excluded) {
		return errors.New("exclusion meanings must match excluded ideas")
	}
	for i, exclusion := range c.Exclusions {
		if err := exclusion.Validate(); err != nil {
			return err
		}
		if exclusion.Idea != c.Excluded[i] {
			return errors.New("exclusion meanings must match excluded ideas")
		}
	}
	return nil
}

func checkProposalAlternatives(c ProposalContent) error {
	for _, unknown := range c.Unknowns {
		if !intentText(unknown, 4000) {
			return errors.New("invalid proposal unknown")
		}
	}
	if !c.Supported {
		if len(c.Alternatives) != 0 {
			return errors.New("unsupported content must not carry alternatives")
		}
		return nil
	}
	if len(c.Alternatives) != 2 {
		return errors.New("supported content needs two alternatives")
	}
	for _, alternative := range c.Alternatives {
		if err := alternative.Validate(); err != nil {
			return err
		}
	}
	return nil
}

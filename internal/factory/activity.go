package factory

// ActivitySettings controls the local repository feed independently of execution authority.
type ActivitySettings struct {
	IssueCreated  bool `json:"issue_created"`
	PRCreated     bool `json:"pr_created"`
	VerifiedMerge bool `json:"verified_merge"`
}

func DefaultActivitySettings() ActivitySettings { return ActivitySettings{true, true, true} }

type ActivityItem struct {
	ID          string `json:"id"`
	Kind        string `json:"kind"`
	Issue       string `json:"issue"`
	PR          string `json:"pr,omitempty"`
	CreatedUnix int64  `json:"created_unix"`
	Summary     string `json:"summary"`
	URL         string `json:"url"`
}
type RepositoryActivity struct {
	Repository     string           `json:"repository"`
	RepositoryPath string           `json:"repository_path"`
	Revision       int64            `json:"revision"`
	Settings       ActivitySettings `json:"settings"`
	Items          []ActivityItem   `json:"items"`
}

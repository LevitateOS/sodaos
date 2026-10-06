package forgejo

// LifecycleEvent is one title/lifecycle transition.
type LifecycleEvent struct {
	Kind     string `json:"kind"`
	AtUnix   int64  `json:"at_unix"`
	ActorID  string `json:"actor_id"`
	OldTitle string `json:"old_title,omitempty"`
	NewTitle string `json:"new_title,omitempty"`
}

// CreationProvenance binds verified creation evidence. Verified requires
// visibility plus the native first-created flag; cached text alone never
// verifies provenance.
type CreationProvenance struct {
	PosterID     string `json:"poster_id,omitempty"`
	CreatedUnix  int64  `json:"created_unix,omitempty"`
	FirstCreated bool   `json:"first_created,omitempty"`
	Verified     bool   `json:"verified,omitempty"`
}

// IssueEvidence is the permission-checked view of one native issue. Hidden
// issues preserve only their locator IDs with Visible=false.
type IssueEvidence struct {
	ID             string             `json:"id"`
	Index          string             `json:"index"`
	Title          string             `json:"title,omitempty"`
	Content        string             `json:"content,omitempty"`
	TitleDigest    string             `json:"title_digest,omitempty"`
	ContentDigest  string             `json:"content_digest,omitempty"`
	ContentVersion int                `json:"content_version,omitempty"`
	NumComments    int                `json:"num_comments,omitempty"`
	IsClosed       bool               `json:"is_closed,omitempty"`
	IsLocked       bool               `json:"is_locked,omitempty"`
	IsPull         bool               `json:"is_pull,omitempty"`
	Provenance     CreationProvenance `json:"provenance"`
	Lifecycle      []LifecycleEvent   `json:"lifecycle,omitempty"`
	CreatedUnix    int64              `json:"created_unix,omitempty"`
	UpdatedUnix    int64              `json:"updated_unix,omitempty"`
	ClosedUnix     int64              `json:"closed_unix,omitempty"`
	Visible        bool               `json:"visible"`
	HiddenReason   string             `json:"hidden_reason,omitempty"`
	Complete       bool               `json:"complete"`
}

// CommentEvidence is the permission-checked view of one native comment.
type CommentEvidence struct {
	ID             string `json:"id"`
	IssueID        string `json:"issue_id"`
	Type           string `json:"type,omitempty"`
	PosterID       string `json:"poster_id,omitempty"`
	Content        string `json:"content,omitempty"`
	ContentDigest  string `json:"content_digest,omitempty"`
	ContentVersion int    `json:"content_version,omitempty"`
	ReviewID       string `json:"review_id,omitempty"`
	Invalidated    bool   `json:"invalidated,omitempty"`
	CreatedUnix    int64  `json:"created_unix,omitempty"`
	UpdatedUnix    int64  `json:"updated_unix,omitempty"`
	Visible        bool   `json:"visible"`
	HiddenReason   string `json:"hidden_reason,omitempty"`
	Complete       bool   `json:"complete"`
}

// CommentPage carries one bounded comment list with completeness evidence.
type CommentPage struct {
	IssueID  string            `json:"issue_id"`
	Items    []CommentEvidence `json:"items"`
	Total    int               `json:"total"`
	Complete bool              `json:"complete"`
}

// DependencyEvidence is one edge occurrence. OccurrenceID distinguishes
// removal plus readdition of the same edge; hidden occurrences redact the
// target so an inaccessible prerequisite blocks without leaking it.
type DependencyEvidence struct {
	OccurrenceID string `json:"occurrence_id"`
	IssueID      string `json:"issue_id"`
	DependencyID string `json:"dependency_id,omitempty"`
	CreatedUnix  int64  `json:"created_unix,omitempty"`
	UpdatedUnix  int64  `json:"updated_unix,omitempty"`
	Visible      bool   `json:"visible"`
	HiddenReason string `json:"hidden_reason,omitempty"`
	Complete     bool   `json:"complete"`
}

// DependencyPage carries one bounded edge list with completeness evidence.
type DependencyPage struct {
	IssueID  string               `json:"issue_id"`
	Items    []DependencyEvidence `json:"items"`
	Total    int                  `json:"total"`
	Complete bool                 `json:"complete"`
}

func validIssue(issue *IssueEvidence, req SnapshotRequest) error {
	if !decimalID(issue.ID) || !decimalID(issue.Index) {
		return ErrInvalidSnapshot
	}
	if req.IssueIndex != "" && issue.Index != req.IssueIndex {
		return ErrInvalidSnapshot
	}
	if !issue.Visible {
		if issue.Title != "" || issue.Content != "" || issue.TitleDigest != "" || issue.ContentDigest != "" {
			return ErrInvalidSnapshot
		}
		return nil
	}
	if issue.TitleDigest == "" || issue.ContentDigest == "" {
		return ErrInvalidSnapshot
	}
	if issue.TitleDigest != ContentDigest(issue.Title) || issue.ContentDigest != ContentDigest(issue.Content) {
		return ErrInvalidSnapshot
	}
	return nil
}

func validCommentPage(page *CommentPage) error {
	if page.Total < 0 || len(page.Items) != page.Total {
		return ErrIncompleteSnapshot
	}
	for _, item := range page.Items {
		if !decimalID(item.ID) || !item.Complete {
			return ErrInvalidSnapshot
		}
		if !item.Visible {
			if item.Content != "" || item.ContentDigest != "" {
				return ErrInvalidSnapshot
			}
			continue
		}
		if item.ContentDigest == "" || item.ContentDigest != ContentDigest(item.Content) {
			return ErrInvalidSnapshot
		}
	}
	return nil
}

func validDependencyPage(page *DependencyPage) error {
	if page.Total < 0 || len(page.Items) != page.Total {
		return ErrIncompleteSnapshot
	}
	for _, item := range page.Items {
		if !decimalID(item.OccurrenceID) || !item.Complete {
			return ErrInvalidSnapshot
		}
		if !item.Visible && item.DependencyID != "" {
			return ErrInvalidSnapshot
		}
		if item.Visible && !decimalID(item.DependencyID) {
			return ErrInvalidSnapshot
		}
	}
	return nil
}

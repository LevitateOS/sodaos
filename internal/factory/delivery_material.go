package factory

// DeliveryNativeMaterial is complete permission-checked PR discussion at one
// idle native revision. Raw bodies accompany their native digests; callers
// never ask a semantic judge to infer a review from an identifier or hash.
// Revision stays outside the semantic fingerprint.
type DeliveryNativeMaterial struct {
	Revision          int64                  `json:"native_revision"`
	Repository        int64                  `json:"repository,string"`
	PRID              int64                  `json:"pr_id,string"`
	IssueID           int64                  `json:"issue_id,string"`
	Head              string                 `json:"head"`
	Base              string                 `json:"base"`
	Title             string                 `json:"title"`
	Body              string                 `json:"body"`
	Comments          []DeliveryComment      `json:"comments"`
	Reviews           []DeliveryNativeReview `json:"reviews"`
	Threads           []DeliveryThread       `json:"threads"`
	Eligible          bool                   `json:"eligible"`
	Blocker           string                 `json:"blocker"`
	BlockingReviewIDs []string               `json:"blocking_review_ids"`
}

type DeliveryComment struct {
	ID             string `json:"id"`
	PosterID       string `json:"poster_id"`
	ReviewID       string `json:"review_id"`
	Type           string `json:"type"`
	Content        string `json:"content"`
	Digest         string `json:"digest"`
	ContentVersion int    `json:"content_version"`
	Invalidated    bool   `json:"invalidated"`
}

type DeliveryNativeReview struct {
	ID        string `json:"id"`
	ActorID   string `json:"actor_id"`
	Type      string `json:"type"`
	Commit    string `json:"commit"`
	Content   string `json:"content"`
	Digest    string `json:"digest"`
	Official  bool   `json:"official"`
	Stale     bool   `json:"stale"`
	Dismissed bool   `json:"dismissed"`
}

type DeliveryThread struct {
	RootID         string   `json:"root_id"`
	ReviewID       string   `json:"review_id"`
	Path           string   `json:"path"`
	Side           string   `json:"side"`
	Line           int64    `json:"line"`
	OriginalCommit string   `json:"original_commit"`
	HeadCommit     string   `json:"head_commit"`
	Digest         string   `json:"digest"`
	Resolved       bool     `json:"resolved"`
	ResolverID     string   `json:"resolver_id"`
	Members        []string `json:"members"`
}

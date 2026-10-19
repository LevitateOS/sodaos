package factory

// MCPFeedMessage is an assignment-visible communication or human reply. Inbox
// contains concise summaries; discussion contains full stored bodies/content.
type MCPFeedMessage struct {
	ID            string             `json:"id"`
	ReplyTo       string             `json:"reply_to,omitempty"`
	Actor         string             `json:"actor"`
	Author        string             `json:"author"`
	At            int64              `json:"at"`
	Status        string             `json:"status"`
	Summary       string             `json:"summary"`
	Body          string             `json:"body,omitempty"`
	Communication *FeedCommunication `json:"communication,omitempty"`
}

type MCPFeedPage struct {
	Items      []MCPFeedMessage `json:"items"`
	NextCursor string           `json:"next_cursor,omitempty"`
	Incomplete bool             `json:"incomplete"`
}

package forgejo

import (
	"context"
	"fmt"
	"net/url"
)

func (c *Client) SearchRepositories(ctx context.Context, token, search string, ownerID int64, page int) ([]Repository, Pagination, error) {
	query := url.Values{"q": {search}, "includeDesc": {"true"}, "page": {fmt.Sprint(page)}, "limit": {"30"}}
	if ownerID > 0 {
		query.Set("uid", fmt.Sprint(ownerID))
		query.Set("exclusive", "true")
	}
	var result struct {
		OK   bool         `json:"ok"`
		Data []Repository `json:"data"`
	}
	headers, err := c.requestHeaders(ctx, "GET", "/repos/search?"+query.Encode(), token, nil, &result)
	if err != nil {
		return nil, Pagination{}, err
	}
	if !result.OK {
		return nil, Pagination{}, ErrInvalidResponse
	}
	metadata, err := pagination(headers, page)
	if result.Data == nil {
		result.Data = []Repository{}
	}
	return result.Data, metadata, err
}

type RepositoryMeta struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	Owner    string `json:"owner"`
	FullName string `json:"full_name"`
}
type WorkItem struct {
	Issue
	Repository RepositoryMeta `json:"repository"`
	Pull       *struct {
		Merged bool `json:"merged"`
		Draft  bool `json:"draft"`
	} `json:"pull_request"`
}
type WorkQuery struct {
	Search, State, Kind, Owner, Labels, Milestones          string
	Assigned, Created, Mentioned, ReviewRequested, Reviewed bool
	Page                                                    int
}

func (c *Client) SearchWork(ctx context.Context, token string, q WorkQuery) ([]WorkItem, Pagination, error) {
	query := url.Values{"q": {q.Search}, "state": {q.State}, "type": {q.Kind}, "owner": {q.Owner}, "labels": {q.Labels}, "milestones": {q.Milestones}, "assigned": {fmt.Sprint(q.Assigned)}, "created": {fmt.Sprint(q.Created)}, "mentioned": {fmt.Sprint(q.Mentioned)}, "review_requested": {fmt.Sprint(q.ReviewRequested)}, "reviewed": {fmt.Sprint(q.Reviewed)}, "page": {fmt.Sprint(q.Page)}, "limit": {"30"}}
	items := []WorkItem{}
	headers, err := c.requestHeaders(ctx, "GET", "/repos/issues/search?"+query.Encode(), token, nil, &items)
	if err != nil {
		return nil, Pagination{}, err
	}
	metadata, err := pagination(headers, q.Page)
	return items, metadata, err
}

type Notification struct {
	ID         int64      `json:"id"`
	Unread     bool       `json:"unread"`
	Pinned     bool       `json:"pinned"`
	Updated    string     `json:"updated_at"`
	Repository Repository `json:"repository"`
	Subject    struct {
		Title   string `json:"title"`
		Type    string `json:"type"`
		HTMLURL string `json:"html_url"`
		State   string `json:"state"`
	} `json:"subject"`
}

func (c *Client) Notifications(ctx context.Context, token string, all bool, page int) ([]Notification, Pagination, error) {
	items := []Notification{}
	headers, err := c.requestHeaders(ctx, "GET", fmt.Sprintf("/notifications?all=%t&page=%d&limit=30", all, page), token, nil, &items)
	if err != nil {
		return nil, Pagination{}, err
	}
	metadata, err := pagination(headers, page)
	return items, metadata, err
}
func (c *Client) SetNotification(ctx context.Context, token string, id int64, state string) error {
	return c.request(ctx, "PATCH", fmt.Sprintf("/notifications/threads/%d?to-status=%s", id, url.QueryEscape(state)), token, nil, nil)
}

type UserProfile struct {
	User
	Description string `json:"description"`
	Location    string `json:"location"`
	Pronouns    string `json:"pronouns"`
	Created     string `json:"created"`
}

func (c *Client) UserProfile(ctx context.Context, token, login string) (UserProfile, error) {
	var result UserProfile
	err := c.request(ctx, "GET", "/users/"+url.PathEscape(login), token, nil, &result)
	return result, err
}

type Activity struct {
	ID         int64      `json:"id"`
	Actor      User       `json:"act_user"`
	Repository Repository `json:"repo"`
	Operation  string     `json:"op_type"`
	Created    string     `json:"created"`
	Content    string     `json:"content"`
	Ref        string     `json:"ref_name"`
}

func (c *Client) UserActivity(ctx context.Context, token, login, date string, page int) ([]Activity, Pagination, error) {
	items := []Activity{}
	query := url.Values{"only-performed-by": {"true"}, "date": {date}, "page": {fmt.Sprint(page)}, "limit": {"30"}}
	headers, err := c.requestHeaders(ctx, "GET", "/users/"+url.PathEscape(login)+"/activities/feeds?"+query.Encode(), token, nil, &items)
	if err != nil {
		return nil, Pagination{}, err
	}
	metadata, err := pagination(headers, page)
	return items, metadata, err
}

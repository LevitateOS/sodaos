package forgejo

import (
	"context"
	"fmt"
	"net/url"
)

// Pointer fields retain the native PATCH distinction between omitted and false.
type ProtectionFields struct {
	EnablePush               *bool     `json:"enable_push,omitempty"`
	EnablePushWhitelist      *bool     `json:"enable_push_whitelist,omitempty"`
	PushWhitelistDeployKeys  *bool     `json:"push_whitelist_deploy_keys,omitempty"`
	PushUsers                *[]string `json:"push_whitelist_usernames,omitempty"`
	PushTeams                *[]string `json:"push_whitelist_teams,omitempty"`
	EnableMergeWhitelist     *bool     `json:"enable_merge_whitelist,omitempty"`
	MergeUsers               *[]string `json:"merge_whitelist_usernames,omitempty"`
	MergeTeams               *[]string `json:"merge_whitelist_teams,omitempty"`
	EnableApprovalsWhitelist *bool     `json:"enable_approvals_whitelist,omitempty"`
	ApprovalUsers            *[]string `json:"approvals_whitelist_username,omitempty"`
	ApprovalTeams            *[]string `json:"approvals_whitelist_teams,omitempty"`
	EnableStatusCheck        *bool     `json:"enable_status_check,omitempty"`
	StatusChecks             *[]string `json:"status_check_contexts,omitempty"`
	RequiredApprovals        *int64    `json:"required_approvals,omitempty"`
	BlockRejected            *bool     `json:"block_on_rejected_reviews,omitempty"`
	BlockReviewRequests      *bool     `json:"block_on_official_review_requests,omitempty"`
	DismissStale             *bool     `json:"dismiss_stale_approvals,omitempty"`
	IgnoreStale              *bool     `json:"ignore_stale_approvals,omitempty"`
	Signed                   *bool     `json:"require_signed_commits,omitempty"`
	ProtectedFiles           *string   `json:"protected_file_patterns,omitempty"`
	UnprotectedFiles         *string   `json:"unprotected_file_patterns,omitempty"`
	BlockOutdated            *bool     `json:"block_on_outdated_branch,omitempty"`
	ApplyToAdmins            *bool     `json:"apply_to_admins,omitempty"`
}
type BranchProtection struct {
	Name string `json:"rule_name"`
	ProtectionFields
}

func (c *Client) BranchProtections(ctx context.Context, token, owner, repo string, page int) ([]BranchProtection, Pagination, error) {
	items := []BranchProtection{}
	headers, err := c.requestHeaders(ctx, "GET", fmt.Sprintf("%s/branch_protections?page=%d&limit=30", repoAPI(owner, repo), page), token, nil, &items)
	if err != nil {
		return nil, Pagination{}, err
	}
	metadata, err := pagination(headers, page)
	return items, metadata, err
}
func (c *Client) CreateBranchProtection(ctx context.Context, token, owner, repo string, input BranchProtection) (BranchProtection, error) {
	var result BranchProtection
	err := c.request(ctx, "POST", repoAPI(owner, repo)+"/branch_protections", token, input, &result)
	return result, err
}
func (c *Client) EditBranchProtection(ctx context.Context, token, owner, repo, name string, input ProtectionFields) (BranchProtection, error) {
	var result BranchProtection
	err := c.request(ctx, "PATCH", repoAPI(owner, repo)+"/branch_protections/"+url.PathEscape(name), token, input, &result)
	return result, err
}

type TagProtection struct {
	ID      int64    `json:"id"`
	Pattern string   `json:"name_pattern"`
	Users   []string `json:"whitelist_usernames"`
	Teams   []string `json:"whitelist_teams"`
}
type TagProtectionFields struct {
	Pattern *string   `json:"name_pattern,omitempty"`
	Users   *[]string `json:"whitelist_usernames,omitempty"`
	Teams   *[]string `json:"whitelist_teams,omitempty"`
}

func (c *Client) TagProtections(ctx context.Context, token, owner, repo string, page int) ([]TagProtection, Pagination, error) {
	items := []TagProtection{}
	headers, err := c.requestHeaders(ctx, "GET", fmt.Sprintf("%s/tag_protections?page=%d&limit=30", repoAPI(owner, repo), page), token, nil, &items)
	if err != nil {
		return nil, Pagination{}, err
	}
	metadata, err := pagination(headers, page)
	return items, metadata, err
}
func (c *Client) CreateTagProtection(ctx context.Context, token, owner, repo string, input TagProtectionFields) (TagProtection, error) {
	var result TagProtection
	err := c.request(ctx, "POST", repoAPI(owner, repo)+"/tag_protections", token, input, &result)
	return result, err
}
func (c *Client) EditTagProtection(ctx context.Context, token, owner, repo string, id int64, input TagProtectionFields) (TagProtection, error) {
	var result TagProtection
	err := c.request(ctx, "PATCH", fmt.Sprintf("%s/tag_protections/%d", repoAPI(owner, repo), id), token, input, &result)
	return result, err
}

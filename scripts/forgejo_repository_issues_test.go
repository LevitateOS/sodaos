package scripts

import (
	"os"
	"strings"
	"testing"
)

func TestForgejoRepositoryIssuePagesKeepNativeWorkflows(t *testing.T) {
	cases := map[string][]string{
		"repo/issue/list.tmpl": {
			`template "repo/header" .`, `id="issue-pins"`, `data-move-url="{{$.Link}}/move_pin"`,
			`template "repo/issue/navbar" .`, `template "repo/issue/search" .`,
			`{{if not .Repository.IsArchived}}`, `{{if .PageIsIssueList}}`,
			`template "repo/issue/filters" .`, `id="issue-actions"`,
			`template "repo/issue/openclose" .`, `template "repo/issue/filter_actions" .`,
			`template "shared/issuelist" dict "." . "listType" "repo"`,
		},
		"repo/issue/new.tmpl": {
			`template "repo/header" .`, `template "repo/issue/new_form" .`,
		},
		"repo/issue/choose.tmpl": {
			`{{range .IssueTemplates}}`, `{{range .IssueConfig.ContactLinks}}`, `{{if .IssueConfig.BlankIssuesEnabled}}`,
			`template={{.FileName}}`, `{{if $.milestone}}`, `{{if $.project}}`, `{{- if .IssueConfigError}}`,
		},
		"repo/issue/labels.tmpl": {
			`{{if and (or .CanWriteIssues .CanWritePulls) (not .Repository.IsArchived)}}`,
			`template "repo/issue/labels/label_new" .`, `template "repo/issue/labels/label_list" .`,
			`template "repo/issue/labels/edit_delete_label" .`,
		},
		"repo/issue/view.tmpl": {
			`template "repo/header" .`, `template "repo/issue/view_title" .`,
			`{{if .Issue.IsPull}}`, `template "repo/pulls/tab_menu" .`, `template "repo/issue/view_content" .`,
		},
		"repo/issue/milestone_new.tmpl": {
			`template "repo/issue/navbar" .`, `{{if and (or .CanWriteIssues .CanWritePulls) .PageIsEditMilestone}}`,
			`class="field {{if .Err_Title}}error{{end}}"`, `class="field {{if .Err_Deadline}}error{{end}}"`,
			`id="clear-date"`, `template "shared/combomarkdowneditor"`, `"MarkdownPreviewUrl"`, `"EasyMDE" true`,
		},
		"repo/issue/milestone_issues.tmpl": {
			`{{if not .Repository.IsArchived}}`, `{{if or .CanWriteIssues .CanWritePulls}}`,
			`data-url="{{$.RepoLink}}/milestones/{{.MilestoneID}}/open"`,
			`data-url="{{$.RepoLink}}/milestones/{{.MilestoneID}}/close"`,
			`{{.Milestone.RenderedContent}}`, `template "repo/issue/filters" .`,
			`template "shared/issuelist" dict "." . "listType" "milestone"`,
		},
		"repo/pulls/commits.tmpl": {
			`template "repo/issue/view_title" .`, `template "repo/pulls/tab_menu" .`, `template "repo/commits_table" .`,
		},
		"repo/pulls/files.tmpl": {
			`id="repolink"`, `id="issueIndex"`, `template "repo/issue/view_title" .`,
			`template "repo/pulls/tab_menu" .`, `template "repo/diff/box" .`,
		},
	}
	for name, markers := range cases {
		contents := readForgejoTemplate(t, strings.Split(name, "/")...)
		for _, marker := range markers {
			if !strings.Contains(contents, marker) {
				t.Errorf("%s lost native workflow marker %q", name, marker)
			}
		}
	}
}

func TestForgejoPullFragmentsKeepNativeHooks(t *testing.T) {
	tabs := readForgejoTemplate(t, "repo", "pulls", "tab_menu.tmpl")
	for _, marker := range []string{
		`.PageIsPullConversation`, `.PageIsPullCommits`, `.PageIsPullFiles`,
		`{{if .NumCommits}}href="{{.Issue.Link}}/commits"{{end}}`,
		`{{if or .Diff.TotalAddition .Diff.TotalDeletion}}`,
	} {
		if !strings.Contains(tabs, marker) {
			t.Errorf("pull tabs lost native marker %q", marker)
		}
	}

	status := readForgejoTemplate(t, "repo", "pulls", "status.tmpl")
	for _, marker := range []string{
		`{{if .CommitStatus}}`, `data-show-all=`, `data-hide-all=`,
		`{{range .CommitStatuses}}`, `template "repo/commit_status" .`,
		`{{if $.is_context_required}}`, `{{range .MissingRequiredChecks}}`,
	} {
		if !strings.Contains(status, marker) {
			t.Errorf("pull status lost native marker %q", marker)
		}
	}

	trust := readForgejoTemplate(t, "repo", "pulls", "trust.tmpl")
	for _, marker := range []string{
		`{{if .CanReadUnitActions}}`, `{{if and .UserCanDelegateTrustWithPullRequest .PullRequestPosterIsExplicitlyTrustedWithActions}}`,
		`{{else if .SomePullRequestRunsNeedApproval}}`, `{{if .UserCanDelegateTrustWithPullRequest}}`,
		`id="pull-request-trust-panel-revoke"`, `name="trust" value="revoke"`,
		`id="pull-request-trust-panel-deny"`, `name="trust" value="deny"`,
		`id="pull-request-trust-panel-once"`, `name="trust" value="once"`,
		`id="pull-request-trust-panel-always"`, `name="trust" value="always"`,
	} {
		if !strings.Contains(trust, marker) {
			t.Errorf("pull trust panel lost native marker %q", marker)
		}
	}
}

func TestForgejoRepositoryIssueStylesStayScoped(t *testing.T) {
	contents, err := os.ReadFile("../assets/branding/forgejo/repository-issues.css")
	if err != nil {
		t.Fatal(err)
	}
	css := string(contents)
	for _, marker := range []string{
		`.soda-repo-work-items`, `.soda-repo-issue-editor`, `.soda-repo-thread`,
		`.soda-pull-tabs`, `.soda-commit-status-panel`, `.soda-pull-diff`,
		`.page-content.repository.milestones`, `.soda-milestone-editor`,
	} {
		if !strings.Contains(css, marker) {
			t.Errorf("repository issue stylesheet lost scope %q", marker)
		}
	}
	for _, forbidden := range []string{"body:has(", "#navbar", ".page-footer", "overflow: hidden", "overflow: clip"} {
		if strings.Contains(css, forbidden) {
			t.Errorf("repository issue stylesheet contains unsafe shared/clipping rule %q", forbidden)
		}
	}
}

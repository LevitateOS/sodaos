package scripts

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestForgejoProjectBoardKeepsNativeBehaviorHooks(t *testing.T) {
	board := readForgejoTemplate(t, "projects", "view.tmpl")
	for _, want := range []string{
		`$canWriteProject := and .CanWriteProjects (or (not .Repository) (not .Repository.IsArchived))`,
		`id="project-board" class="soda-project-board"`, `class="board {{if $canWriteProject}}sortable{{end}}"`,
		`data-url="{{$.Link}}/move"`, `data-id="{{.ID}}" data-sorting="{{.Sorting}}" data-url="{{$.Link}}/{{.ID}}"`,
		`data-project="{{$.Project.ID}}" data-board="{{.ID}}" id="board_{{.ID}}"`,
		`data-issue="{{.ID}}"`, `template "repo/issue/card" (dict "Issue" . "Page" $)`,
		`data-url="{{.Link}}/open"`, `data-url="{{.Link}}/close"`, `data-url="{{.Link}}/delete"`,
		`id="new-project-column-item"`, `id="new_project_column_submit"`, `class="ui primary button edit-project-column-button"`,
		`id="default-project-column-modal-{{.ID}}"`, `id="delete-project-column-modal-{{.ID}}"`, `id="delete-project"`,
		`template "base/modal_actions_confirm"`,
	} {
		if !strings.Contains(board, want) {
			t.Errorf("project board lost native behavior hook %q", want)
		}
	}
	if strings.Count(board, `{{if $canWriteProject}}`) < 5 {
		t.Error("project board no longer retains the native write-permission gates")
	}
}

func TestForgejoProjectBoardStylesDoNotOwnDragState(t *testing.T) {
	contents, err := os.ReadFile(filepath.Join("..", "assets", "branding", "forgejo", "packages.css"))
	if err != nil {
		t.Fatalf("read package stylesheet: %v", err)
	}
	css := string(contents)
	for _, want := range []string{".soda-project-board-page", ".soda-project-board-toolbar", ".soda-project-board .project-column", ".soda-project-board .issue-card"} {
		if !strings.Contains(css, want) {
			t.Errorf("project board stylesheet lacks scoped owner %q", want)
		}
	}
	for _, forbidden := range []string{".soda-project-board .sortable", ".soda-project-board .hidden"} {
		if strings.Contains(css, forbidden) {
			t.Errorf("project board stylesheet may override native state with %q", forbidden)
		}
	}
}

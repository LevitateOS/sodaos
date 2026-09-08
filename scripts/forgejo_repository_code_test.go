package scripts

import (
	"crypto/sha256"
	"fmt"
	"strings"
	"testing"
)

// Exact upstream recovery guards every native action, gate, input and JS delegate.
func TestForgejoCodeAndWorkflowOverridesKeepNativeBodies(t *testing.T) {
	cases := []struct {
		path, hash string
		undo       [][2]string
	}{
		{"moderation/new_abuse_report.tmpl", "552b7c046569127ade1615566065123fcf7f45d41072affea8d16765515d35af", [][2]string{{"class=\"page-content moderation new-report soda-page soda-moderation\" data-signed=\"{{if .IsSigned}}true{{else}}false{{end}}\"", "class=\"page-content moderation new-report\""}, {"class=\"ui form soda-form\"", "class=\"ui form\""}}},
		{"repo/actions/dispatch.tmpl", "7d7626ef9a37196c4d1c14621430a1046c10063939c95375b4b2f116609f6c6d", [][2]string{{"class=\"ui info message tw-flex tw-items-center soda-code-dispatch\"", "class=\"ui info message tw-flex tw-items-center\""}}},
		{"repo/actions/list.tmpl", "5088c8639e883d60c2bf1b310cc772355caa2298415a38591570197d6d1d6c76", [][2]string{{"data-signed=\"{{if .IsSigned}}true{{else}}false{{end}}\" class=\"soda-page soda-code soda-code-actions page-content ", "class=\"page-content "}}},
		{"repo/actions/list_inner.tmpl", "bad8a6689e3c76d2dcbc61e8012771f55321d585eca5bc767565ef3bc8a608ff", [][2]string{{"class=\"ui stackable grid soda-code-workflows\"", "class=\"ui stackable grid\""}}},
		{"repo/actions/no_workflows.tmpl", "91b7fc0d341981a988ef5ee4d31a4c4e5ff16eeea4bf1405c9114c86819609f4", [][2]string{{"class=\"empty-placeholder soda-empty soda-code-no-workflows\"", "class=\"empty-placeholder\""}, {"<h2 class=\"soda-empty-title\">", "<h2>"}, {"<p class=\"soda-empty-description\">", "<p>"}}},
		{"repo/actions/runs_list.tmpl", "e5bb7de6c4912d8ff3fdce9274bbf3cac0f35a19efcaf8d77f62ae1fbf3fd29b", [][2]string{{"class=\"flex-list run-list soda-code-runs\"", "class=\"flex-list run-list\""}, {"class=\"empty-placeholder soda-empty soda-empty--inset\"", "class=\"empty-placeholder\""}, {"<h2 class=\"soda-empty-title\">", "<h2>"}}},
		{"repo/actions/view.tmpl", "9afaab79b413f2478f3ef08b5199485bf329edb9e1d35afb0f8245380cf4a4e2", [][2]string{{"data-signed=\"{{if .IsSigned}}true{{else}}false{{end}}\" class=\"soda-page soda-code soda-code-actions page-content ", "class=\"page-content "}}},
		{"repo/branch/list.tmpl", "18c2141e9532650aba2e9693decb85512484cc657f14fcd22f244c87c09c7e60", [][2]string{{"data-signed=\"{{if .IsSigned}}true{{else}}false{{end}}\" class=\"soda-page soda-code soda-code-browser page-content ", "class=\"page-content "}}},
		{"repo/commit_page.tmpl", "8c1aa5b0ff14c880558f6776ec1b8a1c56c13ab674395eb9ca699775fd8701fc", [][2]string{{"data-signed=\"{{if .IsSigned}}true{{else}}false{{end}}\" class=\"soda-page soda-code soda-code-compare page-content ", "class=\"page-content "}}},
		{"repo/commits.tmpl", "69bdb04a868d29059ecec2a95675d46c66879ebec15e9f33c6b85b8bc95e648b", [][2]string{{"data-signed=\"{{if .IsSigned}}true{{else}}false{{end}}\" class=\"soda-page soda-code soda-code-browser page-content ", "class=\"page-content "}, {"class=\"repo-button-row soda-toolbar\"", "class=\"repo-button-row\""}}},
		{"repo/commits_list.tmpl", "fa3576dec061568018f2622aa4e6197cce15122ac0f2a6947306e533553132b7", [][2]string{{"class=\"ui attached table segment commit-table soda-code-history\"", "class=\"ui attached table segment commit-table\""}}},
		{"repo/commits_table.tmpl", "4ee088c88faa1fe4c5e9fd825d3374b2cc09df41f4e0bb67a138cddaff419a4d", [][2]string{{"class=\"soda-code-commits-heading ui top attached header commits-table", "class=\"ui top attached header commits-table"}, {"class=\"ui attached segment soda-code-commit-search\"", "class=\"ui attached segment\""}}},
		{"repo/diff/compare.tmpl", "786dd644d7eff69a920177d08e8b25eac7c4fc8cfdeb19d7b3b7e0b232b7fbe4", [][2]string{{"data-signed=\"{{if .IsSigned}}true{{else}}false{{end}}\" class=\"soda-page soda-code soda-code-compare page-content ", "class=\"page-content "}, {"class=\"ui segment choose branch soda-code-branch-picker\"", "class=\"ui segment choose branch\""}}},
		{"repo/editor/cherry_pick.tmpl", "4217cced23d2fb55f828520f1f3154d12db1983d19772ef587bc90e0dda890a6", [][2]string{{"data-signed=\"{{if .IsSigned}}true{{else}}false{{end}}\" class=\"soda-page soda-code soda-code-editor page-content ", "class=\"page-content "}, {"<form class=\"soda-form ", "<form class=\""}}},
		{"repo/editor/commit_form.tmpl", "035bfa6d45ab6af8988c5695b74aa332e723f300d237ca8ff0c184a19793fb68", [][2]string{{"class=\"commit-form soda-form-section\"", "class=\"commit-form\""}, {"<h3 class=\"soda-code-commit-title\">", "<h3>"}}},
		{"repo/editor/delete.tmpl", "b5532ab7e69d0933722e60f6c9b1b443bc4d72f180197faaa52c46d34e3d5fc0", [][2]string{{"data-signed=\"{{if .IsSigned}}true{{else}}false{{end}}\" class=\"soda-page soda-code soda-code-editor page-content ", "class=\"page-content "}, {"<form class=\"soda-form ", "<form class=\""}}},
		{"repo/editor/edit.tmpl", "7f2eeb1a0fd496451a69fb3313c748b9c0e40ac1adf836e33dc605f5f7c4df7f", [][2]string{{"data-signed=\"{{if .IsSigned}}true{{else}}false{{end}}\" class=\"soda-page soda-code soda-code-editor page-content ", "class=\"page-content "}, {"<form class=\"soda-form ", "<form class=\""}}},
		{"repo/editor/patch.tmpl", "a128e7c2f41a6e7199ceb7e6f9609fb6e983eece71f66195d40fd932772c0fd7", [][2]string{{"data-signed=\"{{if .IsSigned}}true{{else}}false{{end}}\" class=\"soda-page soda-code soda-code-editor page-content ", "class=\"page-content "}, {"<form class=\"soda-form ", "<form class=\""}}},
		{"repo/editor/upload.tmpl", "db1ae7fb8e2a14503a2511d6d0b69bee462faf9bc5d2144cb2169716709f7258", [][2]string{{"data-signed=\"{{if .IsSigned}}true{{else}}false{{end}}\" class=\"soda-page soda-code soda-code-editor page-content ", "class=\"page-content "}, {"<form class=\"soda-form ", "<form class=\""}}},
		{"repo/empty.tmpl", "8795f36b8ffcc6ff7f81714f7aed2d21a1579468992407a554ad7f38fdffdda0", [][2]string{{"data-signed=\"{{if .IsSigned}}true{{else}}false{{end}}\" class=\"soda-page soda-code soda-code-browser page-content ", "class=\"page-content "}}},
		{"repo/find/files.tmpl", "7b8bb3d319d0509f3d31df2bf27e5eac4ac8a0410c7f0a8c5ca74c2b30c319ec", [][2]string{{"data-signed=\"{{if .IsSigned}}true{{else}}false{{end}}\" class=\"soda-page soda-code soda-code-browser page-content ", "class=\"page-content "}, {"class=\"tw-flex tw-items-center soda-code-finder\"", "class=\"tw-flex tw-items-center\""}}},
		{"repo/forks.tmpl", "a519029ae63d7da5a6ad3e601bb6571b8f7fb9c79ea3a9a67b05713f32936e34", [][2]string{{"data-signed=\"{{if .IsSigned}}true{{else}}false{{end}}\" class=\"soda-page soda-code soda-code-browser page-content ", "class=\"page-content "}}},
		{"repo/home.tmpl", "41cf3643a2247475e9634369c657fffa7f2af18f94735a9c1bf37147a2846393", [][2]string{{"data-signed=\"{{if .IsSigned}}true{{else}}false{{end}}\" class=\"soda-page soda-code soda-code-browser page-content ", "class=\"page-content "}, {"class=\"repo-button-row soda-toolbar\"", "class=\"repo-button-row\""}}},
		{"repo/search.tmpl", "16aa2ff93305811a67b7f3ac76c8f7acf6dbf8a0bafe8cdef7356e1b06a035c5", [][2]string{{"data-signed=\"{{if .IsSigned}}true{{else}}false{{end}}\" class=\"soda-page soda-code soda-code-browser page-content ", "class=\"page-content "}}},
		{"repo/tag/list.tmpl", "e150f972d21c6ea11bb7b150ad5c4cf638dcee28ee08f9283b3e0f867d3f17f1", [][2]string{{"data-signed=\"{{if .IsSigned}}true{{else}}false{{end}}\" class=\"soda-page soda-code soda-code-browser page-content ", "class=\"page-content "}}},
		{"repo/user_cards.tmpl", "42bd9f68ed92ae17e31fa07237f18f1e8fd2b767e5fc5da86e83addf082c0744", [][2]string{{"class=\"user-cards soda-code-people\"", "class=\"user-cards\""}}},
		{"repo/view_file.tmpl", "e2a59d99fab215d2a1b94def6b97380ff220c6aef7789d939ef372fbf2cd9af1", [][2]string{{" non-diff-file-content soda-code-file\"", " non-diff-file-content\""}}},
		{"repo/view_list.tmpl", "f16a26c269c8bcd8699971cca3de2197b3d407bf9bb94a46b7030f410b253dc0", [][2]string{{"class=\"ui single line table tw-mt-0 soda-code-files\"", "class=\"ui single line table tw-mt-0\""}}},
		{"repo/watchers.tmpl", "19350d7226014e63bfa7b9d12e48128e1f3d84ee8686f7603bc56e2c0f0b53e8", [][2]string{{"data-signed=\"{{if .IsSigned}}true{{else}}false{{end}}\" class=\"soda-page soda-code soda-code-browser page-content ", "class=\"page-content "}}},
		{"shared/secrets/add_list.tmpl", "c5c7115161b6a2c3a21aab85385bbfed2214e73cd0dac3417a53b6ae50674a04", [][2]string{{"class=\"ui top attached header soda-config-heading\"", "class=\"ui top attached header\""}, {"class=\"ui attached segment soda-config-list\"", "class=\"ui attached segment\""}}},
		{"shared/variables/variable_list.tmpl", "53adaccdeba06289d589f2d0da64e1acc382febd598c84043305302c7e88091d", [][2]string{{"class=\"ui top attached header soda-config-heading\"", "class=\"ui top attached header\""}, {"class=\"ui attached segment soda-config-list\"", "class=\"ui attached segment\""}}},
		{"webhook/new.tmpl", "581de797cb9a2f83193985ec51e712f15fdd3e7c5008512ecfa114035623e73a", [][2]string{{"class=\"ui top attached header soda-webhook-heading\"", "class=\"ui top attached header\""}, {"class=\"ui attached segment soda-webhook-provider\"", "class=\"ui attached segment\""}}},
		{"webhook/shared-settings.tmpl", "06576d36e3b23e919aebbae72cefa8743492d8478040928bf7ce47d0622a3fc6", [][2]string{{"class=\"event type soda-webhook-events\"", "class=\"event type\""}}},
		{"repo/sub_menu.tmpl", "e9c1b52e5f0a8f766d4bec6897926db92a05f6c2f9147bc704ddf43664518b9e", [][2]string{{"class=\"soda-code-summary ui segments repository-summary", "class=\"ui segments repository-summary"}}},
		{"repo/branch_dropdown.tmpl", "0d3a657b58fa06b24c459854d60cc49bf9a85b7fe0f167236321748d1c93cf41", [][2]string{{"class=\"soda-code-ref-selector js-branch-tag-selector ", "class=\"js-branch-tag-selector "}}},
		{"repo/commit_header.tmpl", "83e7c16cf1701969474fe47c62c9f86c717d1611e0373c5b50ac48534f13a194", [][2]string{{"class=\"soda-code-commit-heading ui top attached header clearing segment tw-relative commit-header", "class=\"ui top attached header clearing segment tw-relative commit-header"}}},
		{"repo/clone_buttons.tmpl", "909f391790b470a2156f0747dd3cb77da5a6702d41f675a3bc4394ea6e30f0bf", [][2]string{{"class=\"js-clone-url soda-code-clone-url\"", "class=\"js-clone-url\""}}},
		{"repo/diff/box.tmpl", "6a252c2fb44ea9c1dfa0b4b9d22335598b96d560b6bdc8d6285314ba36723eb4", [][2]string{{"<div class=\"soda-code-diff\">", "<div>"}}},
		{"shared/blocked_users_list.tmpl", "a677d1c59c5c158916f007da2a43161f5b4f615e75a540e8e75d38b92ea385e8", [][2]string{{"class=\"flex-list soda-blocked-users\"", "class=\"flex-list\""}}},
		{"shared/quota_overview.tmpl", "c408c33f3919a7a90724320b35b750b79261adff505fd1b6f2a1b9a4fd8909d1", [][2]string{{"class=\"ui top attached header soda-config-heading\"", "class=\"ui top attached header\""}, {"class=\"ui attached segment soda-quota-overview\"", "class=\"ui attached segment\""}}},
	}
	for _, tc := range cases {
		t.Run(tc.path, func(t *testing.T) {
			contents := readForgejoTemplate(t, tc.path)
			_, native, ok := strings.Cut(contents, "\n")
			if !ok || !strings.HasPrefix(contents, "{{/* Adapted from Forgejo 15.0.7 templates/") {
				t.Fatal("missing pinned upstream attribution")
			}
			for _, pair := range tc.undo {
				if !strings.Contains(native, pair[0]) {
					t.Fatalf("missing presentation seam %q", pair[0])
				}
				native = strings.ReplaceAll(native, pair[0], pair[1])
			}
			if got := fmt.Sprintf("%x", sha256.Sum256([]byte(native))); got != tc.hash {
				t.Fatalf("native template changed outside declared presentation deltas: %s", got)
			}
		})
	}
}

func TestForgejoCodeEditorsKeepExplicitFormAndNativeRoot(t *testing.T) {
	for _, name := range []string{"edit", "delete", "upload", "patch", "cherry_pick"} {
		contents := readForgejoTemplate(t, "repo/editor/"+name+".tmpl")
		if strings.Count(contents, `<form class="soda-form `) != 1 {
			t.Errorf("%s: expected one opted-in native editor form", name)
		}
		if !strings.Contains(contents, `data-signed="{{if .IsSigned}}true{{else}}false{{end}}"`) {
			t.Errorf("%s: missing explicit guest/signed presentation", name)
		}
	}
}

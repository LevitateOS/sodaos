# Form page redesign

Implemented the existing repository concept and all 40 additional mockup families through official Forgejo 15.0.7 template overrides. The coverage below contains 55 distinct template paths, including native wrappers reached through shared form partials. This is the scope of the mockup set; unrelated settings and authentication workflows are not additional redesign projects.

## Shared design

- `assets/branding/forgejo/form-pages.css` owns the story/form columns, wide authoring layout, inline creation panels, responsive spacing and cobalt submit actions. The existing shared form component supplies control styling; its optional control-background token lets these forms use white surfaces.
- `custom/soda/page_intro.tmpl` remains the single heading/artwork component. Existing canonical illustrations are reused without image generation, editing or new downloads.
- Repository creation pairs owner/name, moves import guidance into the story column, groups description/visibility and provides native template and advanced disclosures. Its private checkbox and all native defaults remain intact.
- Nine provider form bodies share the migration layout, with source/destination groups and native migration options. Forgejo continues to use its upstream wrapper delegating to the Gitea form. Provider fields, authorization help and supported import units remain distinct.
- Editors keep native Markdown/code widgets, metadata, attachments, draft/publish actions and commit choices. Settings keep their real navigation and authorization. Keys, OAuth registration, tags, secrets and variables retain their native panel/dialog/list placement.

## Native contracts

Mockup text is a visual reference, not an authority for fields or permissions. Native translated permission descriptions, required fields, defaults, ownership, error conditions, widget IDs/classes, form methods/actions and capability gates are preserved. No fake board data, successful transfer state, invitation field, role selector or provider authentication was added.

`form-native-contracts.json` records submission-control hashes and capability gates from `794f86a` and the embedded 15.0.7 export before this pass. The source test checks the actual redesigned files. Older tests still recover their exact upstream source by undoing only reviewed presentation fragments in `form-presentation-deltas.json`; that normalization is test-only. Native browser checks use the actual served templates and styles.

## Coverage

| Mockup family | Native template paths |
| --- | --- |
| New repository | `repo/create.tmpl` |
| Fork repository | `repo/pulls/fork.tmpl` |
| New organization | `org/create.tmpl` |
| New team | `org/team/new.tmpl` |
| New repository project | `repo/projects/new.tmpl` |
| New personal or organization project | `org/projects/new.tmpl` |
| New issue | `repo/issue/new.tmpl` |
| New pull request | `repo/diff/compare.tmpl` |
| New milestone | `repo/issue/milestone_new.tmpl` |
| New release | `repo/release/new.tmpl` |
| New wiki page | `repo/wiki/new.tmpl` |
| New file | `repo/editor/edit.tmpl` |
| Upload files | `repo/editor/upload.tmpl` |
| Create your account | `user/auth/signup.tmpl` |
| Register with OpenID | `user/auth/signup_openid_register.tmpl` |
| Create account as administrator | `admin/user/new.tmpl` |
| New authentication source | `admin/auth/new.tmpl` |
| New access token | `user/settings/access_token_edit.tmpl` |
| Register OAuth application | `user/settings/applications_oauth2_list.tmpl`, `org/settings/applications.tmpl`, `admin/applications/list.tmpl` |
| New Actions runner | `repo/settings/runner_create.tmpl`, `user/settings/runner_create.tmpl`, `org/settings/runners_create.tmpl`, `admin/runners/create.tmpl` |
| New protected branch rule | `repo/settings/protected_branch.tmpl` |
| New protected tag rule | `repo/settings/tags.tmpl` |
| New package cleanup rule | `user/settings/packages_cleanup_rules_edit.tmpl`, `org/settings/packages_cleanup_rules_edit.tmpl` |
| New webhook | `repo/settings/webhook/new.tmpl`, `user/settings/hook_new.tmpl`, `org/settings/hook_new.tmpl`, `admin/hook_new.tmpl` |
| Add deploy key | `repo/settings/deploy_keys.tmpl` |
| Add SSH key | `user/settings/keys.tmpl`, `user/settings/keys_ssh.tmpl` |
| Add GPG key | `user/settings/keys.tmpl`, `user/settings/keys_gpg.tmpl` |
| Add SSH principal | `user/settings/keys.tmpl`, `user/settings/keys_principal.tmpl` |
| Add Actions secret | `shared/secrets/add_list.tmpl`, `repo/settings/actions.tmpl`, `user/settings/actions.tmpl`, `org/settings/actions.tmpl` |
| Add Actions variable | `shared/variables/variable_list.tmpl`, `repo/settings/actions.tmpl`, `user/settings/actions.tmpl`, `org/settings/actions.tmpl`, `admin/actions.tmpl` |
| New abuse report | `moderation/new_abuse_report.tmpl` |
| Import from Git | `repo/migrate/git.tmpl` |
| Import from GitHub | `repo/migrate/github.tmpl` |
| Import from GitLab | `repo/migrate/gitlab.tmpl` |
| Import from Forgejo | `repo/migrate/forgejo.tmpl` |
| Import from Gitea | `repo/migrate/gitea.tmpl` |
| Import from Gogs | `repo/migrate/gogs.tmpl` |
| Import from OneDev | `repo/migrate/onedev.tmpl` |
| Import from GitBucket | `repo/migrate/gitbucket.tmpl` |
| Import from Codebase | `repo/migrate/codebase.tmpl` |
| Import from Pagure | `repo/migrate/pagure.tmpl` |

Shared implementation also touches `repo/create_basic.tmpl`, `shared/actions/runner_create.tmpl`, `package/shared/cleanup_rules/edit.tmpl` and the three layout heads needed to avoid duplicate page titles. Unchanged stock wrappers inherit the redesigned partials; they are not copied into the override tree.

## Validation and limits

### Consistency follow-up

The follow-up review removes repeated form titles from projects, forks, milestones,
releases, team creation, administrator creation, reports and OpenID registration.
Project/milestone/release intros now use the native create/edit title and help text;
the project and fork fieldsets reference their page heading. Runner, webhook and
cleanup variants use the same existing presentation path across owner scopes.

One explicit action-row role now controls spacing, separators, alignment and mobile
button widths across full forms and inline creation panels. Native action order,
submit attributes and destructive-action distinctions remain intact. Dialogs keep
their native footer and receive consistent content padding and heading typography.
The tag deletion row no longer opts into creation-panel styling. Section headings
share one font token, project/milestone page widths agree with other creation pages,
and form settings clear the older body inset so headings and fields align.

Follow-up evidence is in `.artifacts/form-consistency-20260910/`. The existing
72-case native browser check also asserts no repeated title, consistent action-row
geometry and webhook heading/field alignment. Go template contracts, required
typecheck/analyzer/fixtures, the Forgejo suite (32 passed, 19 opt-in skips) and the
isolated preview build passed. Desktop and mobile/dark captures were inspected.
The fixture's secret/variable routes returned 404, so their dialog refinements have
source review only; the prior administrator/owner/runner limits still apply.
No forms were submitted, images generated, services restarted or appliances deployed.

### Initial implementation evidence

Evidence is retained in `.artifacts/form-redesign-20260910/`. Local fixture validation is separate from appliance deployment.

- Passed `go test ./scripts`, `bun run typecheck` (including analyzer/fixtures), `bun run test:forgejo` (31 passed, 19 opt-in skips), and an isolated `bun run build:preview`. Logs retain the earlier failures and the passing retries.
- The explicit `SODA_FORGEJO_FORM_REVIEW=1` browser test passed 72 rendered cases: 18 accessible routes, each at 1440px and 390px in light and dark themes. It checks live template landmarks, heading count, artwork loading, horizontal bounds and the shared layout. Read-only interactions verify native required-field validity, checkbox submission values, keyboard-operated disclosures, initialization fields, visibility radios and SSH/GPG panel opening. The test blocks non-GET/HEAD requests and confirms none were attempted.
- Local preview assets are backed up under `served-before/`. The affected stylesheets were refreshed and the supported `forgejo manager reload-templates` command reloaded mounted templates. A missing existing `sodaspaces-page.css` was supplied from the canonical preview build to let verified captures complete. No container restart, configuration change or appliance rollout occurred.
- Verified desktop captures cover repository creation, GitHub import, organization creation, access-token creation, personal project creation, issue creation and webhook creation. The guest registration capture shows the real disabled-registration state.
- Final version-7 captures were visually inspected: repository/GitHub forms at desktop width (`review-desktop-v7/`), repository/project/issue forms on mobile (`review-mobile-v7/`), and organization/token forms in mobile dark mode (`review-dark-v7/`). These captures verify the served template revision and stylesheet bytes; the earlier desktop captures retain the preceding typography/spacing state.
- The screenshot account cannot access repository-owner or administrator-only forms; those requests returned native 404/403 responses. Runner creation is unavailable to this local fixture, and registration remains disabled. These branches retain source-contract coverage but are not claimed as individually verified native workflows.
- No forms were submitted and no repositories, accounts, tokens, runners, integrations, keys or provider resources were created. No new image generation was requested.

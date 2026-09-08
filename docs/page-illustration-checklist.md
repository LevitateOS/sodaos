# Page illustration checklist

Active goal: assess each page, decide whether art helps, record its scene prompt, generate and integrate missing appropriate art, inspect the native rendering, then advance to the next pending page.

## Workflow and status

- `Pending`: page purpose/layout still needs individual review; no blanket no-image decisions.
- `Existing — verify`: artwork exists; confirm that it fits this page and renders correctly.
- `Generate`: distinct artwork is appropriate; prompt/output/integration/checks remain.
- `Done`: appropriate artwork integrated and inspected, with evidence below.
- `Integrated — verify`: asset and source are ready; native page verification remains pending.
- `No image`: individually reviewed; record the concrete reason.
- `Partial — trace caller`: not assumed to be a page; account for its actual page callers before closing.

Preserve native headings, forms, permissions and scripts. Use the fixed references and material/character rules in [settings-art-prompts.md](../assets/branding/forgejo/settings-art-prompts.md). Decorative art must not replace meaningful status, avatars or repository content. Each generated page needs its own recorded scene; shared native components do not each need an image.

Scope starts with every current override below. Shared layout coverage can include native pages without a leaf override; add those route variants as their callers are traced. This is an inventory, not a claim that every route has been visited. Existing staged and unstaged work predates this goal and must be preserved.

## Current page

Next: administrator landing page and remaining admin callers; resolve unconditional inherited dashboard artwork before closing the admin batch. Authentication, Actions and remaining shared callers still require assessment. Track the issue/PR create-button text visibility defect observed in capture-h1PW7Q separately from artwork decisions. Enrollment artwork suppression has source/test evidence; native enrollment capture remains unperformed. Personal registry settings artwork is complete; organization registry settings was assessed without extra art. Keep cleanup add/edit/preview pages free of inherited decorative art. Organization registry artwork is integrated, but native verification awaits an existing accessible organization.

## Per-template inventory

| Status | Template / page entry | Current artwork or caller | Decision / evidence |
| --- | --- | --- | --- |
| Integrated — verify | [`admin/applications/list.tmpl`](../appliance/forgejo/templates/admin/applications/list.tmpl) | `admin/layout_head` | Shared OAuth application inventory/create caller reviewed. Do not add another image; current inherited dashboard art needs an explicit page decision after reviewing the admin landing page. Source assessment only; current fixture is non-admin. |
| Integrated — verify | [`admin/applications/oauth2_edit.tmpl`](../appliance/forgejo/templates/admin/applications/oauth2_edit.tmpl) | `admin/layout_head` | Focused client configuration/credential form should have no decorative art. Inherited dashboard image is now suppressed through an explicit hideArtwork input; native verification remains pending; leaf reviewed, shared form previously assessed. Source assessment only; current fixture is non-admin. |
| Pending | [`admin/auth/edit.tmpl`](../appliance/forgejo/templates/admin/auth/edit.tmpl) | `admin/layout_head` | Not yet reviewed in this goal. |
| Integrated — verify | [`admin/auth/list.tmpl`](../appliance/forgejo/templates/admin/auth/list.tmpl) | `admin/layout_head`, artwork suppressed | Authentication-source names, types and enabled indicators are the meaningful visuals; retain the compact configuration inventory. Full leaf reviewed; inherited dashboard art suppressed explicitly. Native administrator capture pending. |
| Pending | [`admin/auth/new.tmpl`](../appliance/forgejo/templates/admin/auth/new.tmpl) | `admin/layout_head` | Not yet reviewed in this goal. |
| Pending | [`admin/config.tmpl`](../appliance/forgejo/templates/admin/config.tmpl) | `admin/layout_head` | Not yet reviewed in this goal. |
| Integrated — verify | [`admin/cron.tmpl`](../appliance/forgejo/templates/admin/cron.tmpl) | `admin/layout_head`, artwork suppressed | Schedules, execution timestamps and actual result icons are the relevant visuals. Full leaf reviewed; inherited artwork suppressed explicitly. Native admin capture pending with the non-admin fixture. |
| Integrated — verify | [`admin/dashboard.tmpl`](../appliance/forgejo/templates/admin/dashboard.tmpl) | `admin/layout_head`, artwork suppressed | Maintenance operations and live system-status content need no decorative scene. Full wrapper reviewed; explicit hideArtwork input now suppresses inherited home-dashboard image. Native admin capture pending; focused admin parity and presentation tests passed. |
| Integrated — verify | [`admin/emails/list.tmpl`](../appliance/forgejo/templates/admin/emails/list.tmpl) | `admin/layout_head`, artwork suppressed | Email ownership, primary/activated state and confirmation dialogs need focused attention; no decorative email scene. Full leaf reviewed; inherited dashboard art suppressed explicitly. Native administrator capture pending. |
| Integrated — verify | [`admin/layout_head.tmpl`](../appliance/forgejo/templates/admin/layout_head.tmpl) | `dashboard-papercraft.png` inherited by every caller | Audit found unconditional home-dashboard artwork across administration, including credential editors and queue diagnostics. Dashboard, OAuth editing and both queue pages now opt out explicitly. Unfinished: review remaining callers and verify native suppression. Existing image is not verified as appropriate for this entire layout. |
| Integrated — verify | [`admin/notice.tmpl`](../appliance/forgejo/templates/admin/notice.tmpl) | `admin/layout_head`, artwork suppressed | Actual system notices and their detail dialog must remain prominent; an empty list is not proof of system health. Full leaf reviewed; inherited artwork suppressed explicitly. Native admin capture pending with the non-admin fixture. |
| Integrated — verify | [`admin/queue.tmpl`](../appliance/forgejo/templates/admin/queue.tmpl) | `admin/layout_head` | Actual queue types, worker counts and pending counts provide the useful visual information. No decorative art is appropriate; inherited dashboard image is now suppressed through an explicit hideArtwork input; native verification remains pending. Full leaf reviewed. Source assessment only; current fixture is non-admin. |
| Integrated — verify | [`admin/queue_manage.tmpl`](../appliance/forgejo/templates/admin/queue_manage.tmpl) | `admin/layout_head` | Actual queue counts, removal control and maximum-worker form need focused diagnostics. No decorative art is appropriate; inherited dashboard image is now suppressed through an explicit hideArtwork input; native verification remains pending. Full leaf reviewed. Source assessment only; current fixture is non-admin. |
| Integrated — verify | [`admin/repo/list.tmpl`](../appliance/forgejo/templates/admin/repo/list.tmpl) | `admin/layout_head`, artwork suppressed | Repository visibility, archive/template/mirror flags, sizes and ownership need table space and exact native labels. Full leaf reviewed; inherited dashboard art suppressed explicitly. Native administrator capture pending. |
| Integrated — verify | [`admin/self_check.tmpl`](../appliance/forgejo/templates/admin/self_check.tmpl) | `admin/layout_head`, artwork suppressed | Database and cache warnings or the actual no-problem result communicate the diagnostic outcome; decoration should not imply a result. Full leaf reviewed; inherited artwork suppressed explicitly. Native admin capture pending with the non-admin fixture. |
| Integrated — verify | [`admin/stacktrace.tmpl`](../appliance/forgejo/templates/admin/stacktrace.tmpl) | `admin/layout_head`, artwork suppressed | Process/stacktrace tabs, counts and actual stack content require working space; no decorative scene is useful. Full leaf reviewed; inherited artwork suppressed explicitly. Native admin capture pending with the non-admin fixture. |
| Pending | [`admin/user/edit.tmpl`](../appliance/forgejo/templates/admin/user/edit.tmpl) | `admin/layout_head` | Not yet reviewed in this goal. |
| Integrated — verify | [`admin/user/list.tmpl`](../appliance/forgejo/templates/admin/user/list.tmpl) | `admin/layout_head`, artwork suppressed | Account types, activation, restrictions, 2FA and last login require a scannable status table; decorative people art adds no account-specific information. Full leaf reviewed; inherited dashboard art suppressed explicitly. Native administrator capture pending. |
| Integrated — verify | [`admin/user/new.tmpl`](../appliance/forgejo/templates/admin/user/new.tmpl) | `admin/layout_head` | Full account-creation form reviewed. A compact robot preparing one blank identity card can orient onboarding without depicting permissions or completion. Distinct admin-new-account-papercraft.png generated with built-in image_gen and integrated via an explicit artwork input. Exact prompt/provenance: assets/branding/forgejo/admin-new-account-prompt.md. RGBA inspected; focused admin parity/presentation tests passed. Native admin capture remains pending. |
| Partial — trace caller | [`custom/explore_empty.tmpl`](../appliance/forgejo/templates/custom/explore_empty.tmpl) | — | Not yet reviewed in this goal. |
| Partial — trace caller | [`custom/explore_navbar.tmpl`](../appliance/forgejo/templates/custom/explore_navbar.tmpl) | — | Not yet reviewed in this goal. |
| Partial — trace caller | [`custom/extra_links.tmpl`](../appliance/forgejo/templates/custom/extra_links.tmpl) | — | Not yet reviewed in this goal. |
| Partial — trace caller | [`custom/footer.tmpl`](../appliance/forgejo/templates/custom/footer.tmpl) | — | Not yet reviewed in this goal. |
| Partial — trace caller | [`custom/header.tmpl`](../appliance/forgejo/templates/custom/header.tmpl) | — | Not yet reviewed in this goal. |
| Partial — trace caller | [`custom/soda/empty_content.tmpl`](../appliance/forgejo/templates/custom/soda/empty_content.tmpl) | — | Not yet reviewed in this goal. |
| Partial — trace caller | [`custom/soda/guest_theme.tmpl`](../appliance/forgejo/templates/custom/soda/guest_theme.tmpl) | — | Not yet reviewed in this goal. |
| Partial — trace caller | [`custom/soda/notification_preview.tmpl`](../appliance/forgejo/templates/custom/soda/notification_preview.tmpl) | — | Not yet reviewed in this goal. |
| Partial — trace caller | [`custom/soda/page_intro.tmpl`](../appliance/forgejo/templates/custom/soda/page_intro.tmpl) | — | Not yet reviewed in this goal. |
| Partial — trace caller | [`custom/soda/theme_toggle.tmpl`](../appliance/forgejo/templates/custom/soda/theme_toggle.tmpl) | — | Not yet reviewed in this goal. |
| Existing — verify | [`explore/code.tmpl`](../appliance/forgejo/templates/explore/code.tmpl) | `explore-papercraft.png` | Not yet reviewed in this goal. |
| Existing — verify | [`explore/repos.tmpl`](../appliance/forgejo/templates/explore/repos.tmpl) | `explore-papercraft.png` | Not yet reviewed in this goal. |
| Existing — verify | [`explore/users.tmpl`](../appliance/forgejo/templates/explore/users.tmpl) | `orgs-papercraft.png`, `users-papercraft.png` | Not yet reviewed in this goal. |
| Existing — verify | [`home.tmpl`](../appliance/forgejo/templates/home.tmpl) | `home-papercraft.png` | Not yet reviewed in this goal. |
| Existing — verify | [`install.tmpl`](../appliance/forgejo/templates/install.tmpl) | `home-papercraft.png` | Not yet reviewed in this goal. |
| Pending | [`moderation/new_abuse_report.tmpl`](../appliance/forgejo/templates/moderation/new_abuse_report.tmpl) | — | Not yet reviewed in this goal. |
| Existing — verify | [`org/create.tmpl`](../appliance/forgejo/templates/org/create.tmpl) | `new-org-papercraft.png` | Not yet reviewed in this goal. |
| Partial — trace caller | [`org/header.tmpl`](../appliance/forgejo/templates/org/header.tmpl) | — | Not yet reviewed in this goal. |
| No image | [`org/home.tmpl`](../appliance/forgejo/templates/org/home.tmpl) | `org/header` | Organization identity, authored README, repositories and real member avatars take priority; source-reviewed. |
| No image | [`org/member/members.tmpl`](../appliance/forgejo/templates/org/member/members.tmpl) | `org/header` | Actual avatars, roles, visibility and owner-only security indicators take priority; source-reviewed. |
| No image | [`org/projects/list.tmpl`](../appliance/forgejo/templates/org/projects/list.tmpl) | Native owner identity and project summaries | Organization/personal variants assessed. Keep real project descriptions, open/closed counts and search primary; personal empty list inspected at capture-bepvu4/001.png. Organization/populated variants source-only. |
| Integrated — verify | [`org/projects/new.tmpl`](../appliance/forgejo/templates/org/projects/new.tmpl) | `new-project-papercraft.png`, creation only | Personal creation verified at desktop/mobile; organization capture and edit-state capture pending. Edit uses no artwork. |
| No image | [`org/projects/view.tmpl`](../appliance/forgejo/templates/org/projects/view.tmpl) | Native project columns and issue cards | Personal/organization board wrappers and full shared board source reviewed. Drag/drop columns, card previews, counts and column dialogs already provide the meaningful visuals; preserve working space. Native board capture unavailable without an existing project. |
| Partial — trace caller | [`org/settings/layout_head.tmpl`](../appliance/forgejo/templates/org/settings/layout_head.tmpl) | `org/header` | Registry/cleanup callers assessed; remaining settings routes explicitly queued below. No shared artwork decision imposed on callers. |
| No image | [`org/team/invite.tmpl`](../appliance/forgejo/templates/org/team/invite.tmpl) | Real organization avatar | Invitation card identifies organization, team and inviter before the join action; decoration could imply already-confirmed membership. Full template source reviewed; no invitation available for native capture. |
| No image | [`org/team/members.tmpl`](../appliance/forgejo/templates/org/team/members.tmpl) | `org/header` | Actual member identities, owner controls and pending invitations take priority. Source-reviewed. |
| Integrated — verify | [`org/team/new.tmpl`](../appliance/forgejo/templates/org/team/new.tmpl) | `new-team-papercraft.png` on creation only | Robot assembles member cards in a shared holder. Edit/Owners-team permission forms need no illustration. Native capture pending existing accessible organization. |
| No image | [`org/team/repositories.tmpl`](../appliance/forgejo/templates/org/team/repositories.tmpl) | `org/header` | Exact repository membership and bulk access controls take priority. Source-reviewed. |
| No image | [`org/team/teams.tmpl`](../appliance/forgejo/templates/org/team/teams.tmpl) | `org/header` | Team cards already show real members and counts; no added header art. Source-reviewed. |
| No image | [`package/settings.tmpl`](../appliance/forgejo/templates/package/settings.tmpl) | `org/header`, `user/overview/header` | Repository association and version deletion need package identity and clear warnings, not a decorative header; both owner branches source-reviewed. |
| No image | [`package/shared/cargo.tmpl`](../appliance/forgejo/templates/package/shared/cargo.tmpl) | — | Inline initialize/rebuild controls and explanatory text within owner registry settings; no independent image. |
| No image | [`package/shared/cleanup_rules/edit.tmpl`](../appliance/forgejo/templates/package/shared/cleanup_rules/edit.tmpl) | — | Add/edit retention form: keep/remove criteria and actions take priority; no decorative art. |
| No image | [`package/shared/cleanup_rules/list.tmpl`](../appliance/forgejo/templates/package/shared/cleanup_rules/list.tmpl) | — | Inline rule summaries and add/edit/preview controls; owner settings header owns any artwork. |
| No image | [`package/shared/cleanup_rules/preview.tmpl`](../appliance/forgejo/templates/package/shared/cleanup_rules/preview.tmpl) | — | Affected-version count and six-column results table are the meaningful visuals; no artwork. |
| Partial — trace caller | [`package/shared/list.tmpl`](../appliance/forgejo/templates/package/shared/list.tmpl) | — | Not yet reviewed in this goal. |
| No image | [`package/shared/versionlist.tmpl`](../appliance/forgejo/templates/package/shared/versionlist.tmpl) | `user/overview/package_versions` | Search, sort, container tag filter, version rows and pagination; no independent decorative header. Source assessment. |
| No image | [`package/view.tmpl`](../appliance/forgejo/templates/package/view.tmpl) | `user/overview/header` | Package/version identity, protocol content, files and metadata take priority; source assessment below. |
| Pending | [`post-install.tmpl`](../appliance/forgejo/templates/post-install.tmpl) | — | Not yet reviewed in this goal. |
| No image | [`projects/list.tmpl`](../appliance/forgejo/templates/projects/list.tmpl) | Repository, organization and personal callers traced | Full shared partial reviewed; decoration belongs only in the selected personal/organization creation wrapper. No independent partial artwork. |
| No image | [`projects/new.tmpl`](../appliance/forgejo/templates/projects/new.tmpl) | Repository, organization and personal callers traced | Full shared partial reviewed; decoration belongs only in the selected personal/organization creation wrapper. No independent partial artwork. |
| No image | [`projects/view.tmpl`](../appliance/forgejo/templates/projects/view.tmpl) | Repository, organization and personal callers traced | Full shared partial reviewed; decoration belongs only in the selected personal/organization creation wrapper. No independent partial artwork. |
| Partial — trace caller | [`repo/actions/dispatch.tmpl`](../appliance/forgejo/templates/repo/actions/dispatch.tmpl) | — | Not yet reviewed in this goal. |
| Pending | [`repo/actions/list.tmpl`](../appliance/forgejo/templates/repo/actions/list.tmpl) | `repo/header` | Not yet reviewed in this goal. |
| Partial — trace caller | [`repo/actions/list_inner.tmpl`](../appliance/forgejo/templates/repo/actions/list_inner.tmpl) | — | Not yet reviewed in this goal. |
| Partial — trace caller | [`repo/actions/no_workflows.tmpl`](../appliance/forgejo/templates/repo/actions/no_workflows.tmpl) | — | Not yet reviewed in this goal. |
| Partial — trace caller | [`repo/actions/runs_list.tmpl`](../appliance/forgejo/templates/repo/actions/runs_list.tmpl) | — | Not yet reviewed in this goal. |
| Pending | [`repo/actions/view.tmpl`](../appliance/forgejo/templates/repo/actions/view.tmpl) | `repo/header` | Not yet reviewed in this goal. |
| No image | [`repo/activity.tmpl`](../appliance/forgejo/templates/repo/activity.tmpl) | Native pulse and analytics | All four callers assessed: pulse summary/activity lists, contributor chart, code-frequency chart, recent-commit chart. Native data visuals are primary; loading/failed/empty feedback must remain unambiguous. Source and desktop captures below. |
| No image | [`repo/branch/list.tmpl`](../appliance/forgejo/templates/repo/branch/list.tmpl) | Native branch state and divergence | Default/protected/deleted branch states, commit status, ahead/behind bars and PR state labels already encode meaning. Create/rename/delete dialogs are focused operations. Full source reviewed; read-only default branch captured. |
| Partial — trace caller | [`repo/branch_dropdown.tmpl`](../appliance/forgejo/templates/repo/branch_dropdown.tmpl) | — | Not yet reviewed in this goal. |
| Partial — trace caller | [`repo/clone_buttons.tmpl`](../appliance/forgejo/templates/repo/clone_buttons.tmpl) | — | Not yet reviewed in this goal. |
| No image | [`repo/commit_header.tmpl`](../appliance/forgejo/templates/repo/commit_header.tmpl) | Commit detail and single-commit diff callers | Full source now reviewed across bounded reads. Real author/committer/signature, ancestry, Git notes and focused branch/tag/cherry-pick dialogs are meaningful content; no decorative image. |
| No image | [`repo/commit_page.tmpl`](../appliance/forgejo/templates/repo/commit_page.tmpl) | Native commit metadata and diff | Commit identity/status and actual changed lines are the relevant visuals. Wrapper reviewed and real commit captured; shared header/diff internals remain separately tracked. |
| No image | [`repo/commits.tmpl`](../appliance/forgejo/templates/repo/commits.tmpl) | Native commit table | History is author/message/SHA/date and reference navigation with search, not an introductory page. Wrapper and commits_table reviewed; native two-commit history captured. Comparison and PR commit callers are now assessed. |
| Partial — trace caller | [`repo/commits_list.tmpl`](../appliance/forgejo/templates/repo/commits_list.tmpl) | — | Not yet reviewed in this goal. |
| No image | [`repo/commits_table.tmpl`](../appliance/forgejo/templates/repo/commits_table.tmpl) | History, comparison and PR commits callers | All override callers traced and assessed. Search/counts/refs/renamed-file feedback and commit rows need no independent artwork. |
| Existing — verify | [`repo/create.tmpl`](../appliance/forgejo/templates/repo/create.tmpl) | `new-repo-papercraft.png` | Not yet reviewed in this goal. |
| No image | [`repo/diff/box.tmpl`](../appliance/forgejo/templates/repo/diff/box.tmpl) | Commit detail, comparison and PR files callers | Full 267-line partial reviewed. Actual text/image/CSV changes, viewed progress, review controls and unavailable/truncated feedback must remain primary. No independent artwork. |
| No image | [`repo/diff/compare.tmpl`](../appliance/forgejo/templates/repo/diff/compare.tmpl) | Native reference picker and diff | Full wrapper reviewed: branch/tag/fork selectors, comparison direction/type, nothing-to-compare, existing PR, new PR and archived/sign-in states. Decorative branching or success imagery would compete with actual comparison state. Commit comparison captured. |
| No image | [`repo/editor/cherry_pick.tmpl`](../appliance/forgejo/templates/repo/editor/cherry_pick.tmpl) | Exact commit and destination branch | Cherry-pick/revert variants show the source SHA, operation and target. Preserve this precise context without decorative branching/success imagery. Full template reviewed. |
| No image | [`repo/editor/commit_form.tmpl`](../appliance/forgejo/templates/repo/editor/commit_form.tmpl) | Five authoring callers traced | Real author avatar, signing eligibility/warnings, signoff, direct/new-branch choice and commit email are meaningful controls. Full partial and every override caller reviewed; no independent artwork. |
| No image | [`repo/editor/delete.tmpl`](../appliance/forgejo/templates/repo/editor/delete.tmpl) | Native deletion commit form | Destructive intent is expressed through target-specific commit details; keep author, branch choice and confirmation prominent. Full wrapper and shared form reviewed; no illustration. |
| No image | [`repo/editor/edit.tmpl`](../appliance/forgejo/templates/repo/editor/edit.tmpl) | CodeMirror and native preview | New/edit variants need code, filename/path, rendered preview and change diff space. Empty-content confirmation stays focused. Full template reviewed; no decorative image. |
| No image | [`repo/editor/patch.tmpl`](../appliance/forgejo/templates/repo/editor/patch.tmpl) | Native patch editor | Patch text and branch context must remain the focus; no decorative patching metaphor. Full template reviewed, including empty-content confirmation and commit form. |
| No image | [`repo/editor/upload.tmpl`](../appliance/forgejo/templates/repo/editor/upload.tmpl) | Upload area and commit form | Destination path, actual upload feedback and commit target are primary. No decorative file/upload scene suggesting completion. Full wrapper reviewed; upload component remains separately tracked. |
| No image | [`repo/empty.tmpl`](../appliance/forgejo/templates/repo/empty.tmpl) | Native quickstart/status | Full source reviewed. Writer quickstart prioritizes clone/new-file/upload and exact Git commands; read-only empty state has no setup action; archived/broken states need direct warnings. No image for any branch. No empty repository in current public inventory; source-only assessment. |
| No image | [`repo/find/files.tmpl`](../appliance/forgejo/templates/repo/find/files.tmpl) | Native file finder | Focused path input and matching file rows support quick navigation. No-results text should stay direct, without another illustration. Full template reviewed and populated finder captured. |
| No image | [`repo/forks.tmpl`](../appliance/forgejo/templates/repo/forks.tmpl) | Real fork owner avatars | Full template reviewed. Actual owner/repository links explain the fork network; decorative branching adds no information. Populated native list inspected. |
| No image | [`repo/graph.tmpl`](../appliance/forgejo/templates/repo/graph.tmpl) | Actual Git graph | The graph itself is the illustration of ancestry. Ref selection, monochrome/color modes and loading state need no decorative scene. Full wrapper source reviewed; populated native graph captured. |
| Partial — trace caller | [`repo/header.tmpl`](../appliance/forgejo/templates/repo/header.tmpl) | — | Not yet reviewed in this goal. |
| No image | [`repo/home.tmpl`](../appliance/forgejo/templates/repo/home.tmpl) | Repository-owned content | Full wrapper reviewed across home/directory/file/blame, branch/tag/commit context, topics, flags/archive messages, template use and clone controls. No decorative hero competing with the repository description, file tree or README. Native home/file/blame captures below. |
| No image | [`repo/issue/choose.tmpl`](../appliance/forgejo/templates/repo/issue/choose.tmpl) | Repository-authored template choices | Full chooser reviewed: template names/descriptions, external contact links, optional blank issue and invalid-config warning. No decorative scene competing with the repository-specific choices. Source-based; chooser state not captured. |
| No image | [`repo/issue/labels.tmpl`](../appliance/forgejo/templates/repo/issue/labels.tmpl) | `repo/header` | Native label colors and descriptions are the meaningful visuals; preserve the compact management list. Empty read-only list inspected at capture-TADFh7/002.png (1440px); populated and writer controls assessed from source, not rendered. |
| No image | [`repo/issue/list.tmpl`](../appliance/forgejo/templates/repo/issue/list.tmpl) | Issue and pull-request lists | Preserve native status icons, labels, review metadata, pinned cards and filters. Shared empty result also represents filtered-out items, not a successful completion. Full override and upstream 15.0.7 shared/issuelist reviewed. Native populated issues and empty pulls inspected at capture-h1PW7Q/001.png and 002.png (1440px); populated PR review states source-only. Separate observed defect: create-button text is invisible in both captures; needs CSS investigation. |
| No image | [`repo/issue/milestone_issues.tmpl`](../appliance/forgejo/templates/repo/issue/milestone_issues.tmpl) | `repo/header` | Milestone detail uses actual progress, deadline, issue counts and filtered issues; decorative art would compete with those signals. Full override reviewed; native detail capture remains unperformed. |
| No image | [`repo/issue/milestone_new.tmpl`](../appliance/forgejo/templates/repo/issue/milestone_new.tmpl) | `repo/header` | Focused create/edit title, deadline and Markdown-description form under repository navigation; no extra artwork. Full override reviewed; writer form not rendered with this read-only fixture. |
| No image | [`repo/issue/milestones.tmpl`](../appliance/forgejo/templates/repo/issue/milestones.tmpl) | `repo/header` | Actual progress bars, overdue dates and open/closed counts carry the visual hierarchy. Native populated list inspected at capture-TADFh7/001.png (1440px); full override reviewed, writer actions source-only. |
| Partial — trace caller | [`repo/issue/navbar.tmpl`](../appliance/forgejo/templates/repo/issue/navbar.tmpl) | — | Not yet reviewed in this goal. |
| No image | [`repo/issue/new.tmpl`](../appliance/forgejo/templates/repo/issue/new.tmpl) | Author and report form | Full wrapper and stock new_form reviewed: blank/template fields, report text, attachments and permission-gated metadata. Keep writing space and actual author avatar; no decorative image. Native blank form inspected. |
| No image | [`repo/issue/view.tmpl`](../appliance/forgejo/templates/repo/issue/view.tmpl) | Authored conversation and timeline | Full wrapper and stock view_content reviewed. Real author avatars, state, timeline, attachments and comment controls supply the visuals. Issue/PR discussion uses no extra art; native closed issue inspected, PR-specific state source-only. |
| Done | [`repo/migrate/migrate.tmpl`](../appliance/forgejo/templates/repo/migrate/migrate.tmpl) | `migrate-papercraft.png` | Dedicated history/import scene; desktop/mobile native captures inspected. See record below. |
| No image | [`repo/migrate/migrating.tmpl`](../appliance/forgejo/templates/repo/migrate/migrating.tmpl) | Native loading/error visuals | Live progress, failure and retry/cancel controls already communicate operation state. Extra decorative art would compete with those signals; source reviewed, no operation launched. |
| Partial — no image | [`repo/migrate/options.tmpl`](../appliance/forgejo/templates/repo/migrate/options.tmpl) | Native provider-form callers | Reusable inline mirror/LFS fields, not a page or introduction. Do not insert decorative artwork among native provider options; provider page assessment remains separate. |
| No image | [`repo/projects/list.tmpl`](../appliance/forgejo/templates/repo/projects/list.tmpl) | Native repository context | Project summaries/search/counts are the purpose; no separate introductory hero. Empty read-only list inspected at capture-4u3Imx/001.png; populated/write states source-only. |
| No image | [`repo/projects/new.tmpl`](../appliance/forgejo/templates/repo/projects/new.tmpl) | Native shared form | Creation/edit opens directly beneath repository navigation in a focused title/description/template/card-preview form. No extra intro or decorative block; preserve consistency with adjacent repository editing tools. Source assessment; no writable fixture repository. |
| No image | [`repo/projects/view.tmpl`](../appliance/forgejo/templates/repo/projects/view.tmpl) | Native board and issue navigation | Full wrapper reviewed: issue navigation, archive-gated new issue action and full-width shared board. Cards/columns carry actual work and need the available space. No existing board capture. |
| No image | [`repo/pulls/commits.tmpl`](../appliance/forgejo/templates/repo/pulls/commits.tmpl) | Native PR title/tabs and commit table | Full wrapper reviewed. The PR identity and actual commit history are the subject, with no separate decorative intro. Source-based assessment; inspected public repository has no PRs. |
| No image | [`repo/pulls/files.tmpl`](../appliance/forgejo/templates/repo/pulls/files.tmpl) | Native PR title/tabs and diff box | Full wrapper reviewed. File review needs space for actual differences, comments and viewed progress. No decorative image; source-based assessment without an existing PR fixture. |
| Done | [`repo/pulls/fork.tmpl`](../appliance/forgejo/templates/repo/pulls/fork.tmpl) | `fork-papercraft.png` | Independent-copy scene; desktop/mobile native captures inspected. See record below. |
| Partial — trace caller | [`repo/pulls/status.tmpl`](../appliance/forgejo/templates/repo/pulls/status.tmpl) | — | Not yet reviewed in this goal. |
| Partial — trace caller | [`repo/pulls/tab_menu.tmpl`](../appliance/forgejo/templates/repo/pulls/tab_menu.tmpl) | — | Not yet reviewed in this goal. |
| Partial — trace caller | [`repo/pulls/trust.tmpl`](../appliance/forgejo/templates/repo/pulls/trust.tmpl) | — | Not yet reviewed in this goal. |
| No image | [`repo/release/list.tmpl`](../appliance/forgejo/templates/repo/release/list.tmpl) | Real release state, notes and downloads | Keep draft/prerelease/stable labels, verification, publisher, notes and assets primary. Full source reviewed; existing prerelease list captured at capture-id7a6e/001.png. |
| No image | [`repo/release/new.tmpl`](../appliance/forgejo/templates/repo/release/new.tmpl) | Native release editor | Create/tag-only/draft/edit/publish variants need clear tag target, notes, attachments and prerelease choices. Decorative shipment/success imagery would suggest a state not yet established. Full source reviewed; no release submitted. |
| No image | [`repo/release_tag_header.tmpl`](../appliance/forgejo/templates/repo/release_tag_header.tmpl) | Release/tag list toolbar | Both callers assessed; count tabs, conditional search/RSS/create and code-only submenu need no independent image. Full partial reviewed. |
| Pending | [`repo/search.tmpl`](../appliance/forgejo/templates/repo/search.tmpl) | `repo/header` | Not yet reviewed in this goal. |
| No image | [`repo/settings/actions.tmpl`](../appliance/forgejo/templates/repo/settings/actions.tmpl) | `repo/settings/layout_head` | Reviewed all three branches: runner inventory uses live status indicators and owner/label columns; secrets and variables use compact configuration lists and native add/edit dialogs. No decorative header is needed in these repository-scoped settings. Full leaf reviewed; runner/configuration shared bodies reviewed where called. Source assessment only: current fixture has no repository settings access. |
| No image | [`repo/settings/branches.tmpl`](../appliance/forgejo/templates/repo/settings/branches.tmpl) | `repo/settings/layout_head` | Default-branch selection and named protection rules form a compact management page. Keep the archived-unavailable warning and empty-rule feedback prominent; no decorative scene. Full override reviewed; source-only assessment, no native settings capture available to the current fixture. |
| No image | [`repo/settings/collaboration.tmpl`](../appliance/forgejo/templates/repo/settings/collaboration.tmpl) | `repo/settings/layout_head` | Real collaborator avatars, per-person access levels and organization-team permissions are the relevant visuals. Keep add/remove controls adjacent to their targets. Full leaf reviewed; runner/configuration shared bodies reviewed where called. Source assessment only: current fixture has no repository settings access. |
| No image | [`repo/settings/deploy_keys.tmpl`](../appliance/forgejo/templates/repo/settings/deploy_keys.tmpl) | `repo/settings/layout_head` | Key fingerprints, recent-use indicators and read/write access must remain easy to scan; the inline add-key panel is a focused credential form. Full leaf reviewed; runner/configuration shared bodies reviewed where called. Source assessment only: current fixture has no repository settings access. |
| No image | [`repo/settings/githook_edit.tmpl`](../appliance/forgejo/templates/repo/settings/githook_edit.tmpl) | `repo/settings/layout_head` | The actual hook filename and CodeMirror script editor need working space; an illustration adds no useful context to editing executable code. Full leaf reviewed; runner/configuration shared bodies reviewed where called. Source assessment only: current fixture has no repository settings access. |
| No image | [`repo/settings/githooks.tmpl`](../appliance/forgejo/templates/repo/settings/githooks.tmpl) | `repo/settings/layout_head` | Native active/inactive dots and hook filenames communicate configuration state; keep this compact executable-hook inventory free of decoration. Full leaf reviewed; runner/configuration shared bodies reviewed where called. Source assessment only: current fixture has no repository settings access. |
| No image | [`repo/settings/layout_head.tmpl`](../appliance/forgejo/templates/repo/settings/layout_head.tmpl) | `repo/header` | Shared repository header/sidebar/alert wrapper, not a separate page. All 22 current override callers traced and individually assessed above; do not inject shared decorative art into their forms, previews or diagnostics. Native non-overridden callers are not claimed as separately audited. |
| No image | [`repo/settings/lfs.tmpl`](../appliance/forgejo/templates/repo/settings/lfs.tmpl) | `repo/settings/layout_head` | Object IDs, sizes, age, commit lookup and deletion warnings are the useful content; preserve a compact inventory, including its no-files row. Full override reviewed; source-only assessment, no native settings capture available to the current fixture. |
| No image | [`repo/settings/lfs_file.tmpl`](../appliance/forgejo/templates/repo/settings/lfs_file.tmpl) | `repo/settings/layout_head` | The actual LFS object is the visual: image, video, audio, PDF, model, markup or source preview. Decorative art would compete with file content and Unicode/size warnings. Full override reviewed; source-only assessment, no native settings capture available to the current fixture. |
| No image | [`repo/settings/lfs_file_find.tmpl`](../appliance/forgejo/templates/repo/settings/lfs_file_find.tmpl) | `repo/settings/layout_head` | This is an object-to-commit lookup result table, with paths, branches, parent hashes and dates. Keep results and no-commits feedback unobstructed. Full override reviewed; source-only assessment, no native settings capture available to the current fixture. |
| No image | [`repo/settings/lfs_locks.tmpl`](../appliance/forgejo/templates/repo/settings/lfs_locks.tmpl) | `repo/settings/layout_head` | Actual lock-owner avatars, file paths and missing-file/attribute warnings communicate state. Preserve the path entry and force-unlock controls without a decorative lock illustration. Full override reviewed; source-only assessment, no native settings capture available to the current fixture. |
| No image | [`repo/settings/lfs_pointers.tmpl`](../appliance/forgejo/templates/repo/settings/lfs_pointers.tmpl) | `repo/settings/layout_head` | Pointer diagnostics compare actual association, existence and accessibility flags. Keep the status matrix and association action visible; artwork would not explain individual object results. Full override reviewed; source-only assessment, no native settings capture available to the current fixture. |
| No image | [`repo/settings/options.tmpl`](../appliance/forgejo/templates/repo/settings/options.tmpl) | `repo/settings/layout_head` | General settings combines identity/avatar editing, federation, mirror direction/error state, signing trust, indexing and owner-only destructive actions. Preserve section headings and warnings without an unrelated illustrated intro. Full 790-line override reviewed; nested push-mirror dialog not re-reviewed. Source-only assessment; no native repository settings capture with the current fixture. |
| No image | [`repo/settings/protected_branch.tmpl`](../appliance/forgejo/templates/repo/settings/protected_branch.tmpl) | `repo/settings/layout_head` | Create/edit rule form needs space for path patterns, push allowlists, signatures, approvals, status checks, merge restrictions and admin enforcement. An illustration cannot accurately summarize these independently selected policies. Full override reviewed; source-only assessment, no native settings capture available to the current fixture. |
| No image | [`repo/settings/runner_create.tmpl`](../appliance/forgejo/templates/repo/settings/runner_create.tmpl) | `repo/settings/layout_head` | A short name/description form creates a runner record; avoid artwork suggesting an already connected or working runner. Full leaf reviewed; runner/configuration shared bodies reviewed where called. Source assessment only: current fixture has no repository settings access. |
| No image | [`repo/settings/runner_details.tmpl`](../appliance/forgejo/templates/repo/settings/runner_details.tmpl) | `repo/settings/layout_head` | Actual active/idle/offline indicators, last-online time, version, labels and task history supply the meaningful visual content. Full leaf reviewed; runner/configuration shared bodies reviewed where called. Source assessment only: current fixture has no repository settings access. |
| No image | [`repo/settings/runner_edit.tmpl`](../appliance/forgejo/templates/repo/settings/runner_edit.tmpl) | `repo/settings/layout_head` | Name/description and token-regeneration controls are a focused edit task; preserve the regeneration explanation without an illustrated intro. Full leaf reviewed; runner/configuration shared bodies reviewed where called. Source assessment only: current fixture has no repository settings access. |
| No image | [`repo/settings/runner_setup.tmpl`](../appliance/forgejo/templates/repo/settings/runner_setup.tmpl) | `repo/settings/layout_head` | The one-time token warning, UUID and exact configuration instructions need uninterrupted attention; no decorative art. Full leaf reviewed; runner/configuration shared bodies reviewed where called. Source assessment only: current fixture has no repository settings access. |
| No image | [`repo/settings/secrets.tmpl`](../appliance/forgejo/templates/repo/settings/secrets.tmpl) | `repo/settings/layout_head` | Masked values, names and mutation dialogs form a focused configuration screen; a decorative key scene would duplicate the existing key icon without explaining scope. Full leaf reviewed; runner/configuration shared bodies reviewed where called. Source assessment only: current fixture has no repository settings access. |
| No image | [`repo/settings/tags.tmpl`](../appliance/forgejo/templates/repo/settings/tags.tmpl) | `repo/settings/layout_head` | Combined create/edit pattern form and rule table uses actual allowed-user avatars and team labels. Preserve the explicit no-one/empty and archived states without decorative protection imagery. Full override reviewed; source-only assessment, no native settings capture available to the current fixture. |
| No image | [`repo/settings/units.tmpl`](../appliance/forgejo/templates/repo/settings/units.tmpl) | `repo/settings/layout_head` | Feature toggles, internal/external tracker and wiki choices, merge methods and dependent fields require precise reading. No decorative scene adds useful context. Leaf plus all four upstream 15.0.7 overview/issues/pulls/wiki sections reviewed; source-only, no settings access with the fixture. |
| No image | [`repo/settings/webhook/base.tmpl`](../appliance/forgejo/templates/repo/settings/webhook/base.tmpl) | `repo/settings/layout_head` | Repository-scoped endpoint inventory uses actual delivery-status dots and endpoint links. Keep its compact list and provider chooser without decorative art. Leaf, upstream list wrapper and shared base_list reviewed. Source-only assessment; no native repository settings capture with the current fixture. |
| Partial — trace caller | [`repo/settings/webhook/base_list.tmpl`](../appliance/forgejo/templates/repo/settings/webhook/base_list.tmpl) | — | Not yet reviewed in this goal. |
| Partial — trace caller | [`repo/settings/webhook/history.tmpl`](../appliance/forgejo/templates/repo/settings/webhook/history.tmpl) | — | Not yet reviewed in this goal. |
| No image | [`repo/settings/webhook/new.tmpl`](../appliance/forgejo/templates/repo/settings/webhook/new.tmpl) | `repo/settings/layout_head` | Create/edit webhook configuration uses the selected provider icon and native provider-specific form, followed by delivery history. No extra illustration is needed above those controls. Leaf and shared dispatcher reviewed; individual provider bodies not re-reviewed in this pass. Source-only assessment; no native repository settings capture with the current fixture. |
| Partial — trace caller | [`repo/sub_menu.tmpl`](../appliance/forgejo/templates/repo/sub_menu.tmpl) | — | Not yet reviewed in this goal. |
| No image | [`repo/tag/list.tmpl`](../appliance/forgejo/templates/repo/tag/list.tmpl) | Native tags and verification | Names, commit references, verification and permission-gated archives/actions are the useful visuals. Full source reviewed; existing tag list captured at capture-0nKZ6d/001.png. |
| Partial — no image | [`repo/user_cards.tmpl`](../appliance/forgejo/templates/repo/user_cards.tmpl) | Real user avatars | Avatar/name cards and pagination for profile followers/following and repository watchers. Preserve identity imagery; page-specific watcher assessment remains separate. |
| No image | [`repo/view_file.tmpl`](../appliance/forgejo/templates/repo/view_file.tmpl) | File preview and README callers | Full partial reviewed. Authored markup/text/images/video/audio/PDF/3D or raw fallback are the content; keep warnings, source/rendered toggle and file controls distinct. No added art for previews or too-large/error states. |
| No image | [`repo/view_list.tmpl`](../appliance/forgejo/templates/repo/view_list.tmpl) | Home/directory caller | Full partial reviewed: parent path, folders/files/submodules, lazy latest-commit metadata and optional README. These content/navigation rows need no independent image. |
| No image | [`repo/watchers.tmpl`](../appliance/forgejo/templates/repo/watchers.tmpl) | Real watcher/stargazer cards | Full wrapper and stock user_cards reviewed. People identity is the visual subject; empty stargazer feedback remains direct. Both native route variants inspected. |
| No image | [`repo/wiki/new.tmpl`](../appliance/forgejo/templates/repo/wiki/new.tmpl) | Native Markdown editor | Create/edit both need space for authored title/content/preview and commit message. No decorative image; full source reviewed, no writable wiki fixture. |
| No image | [`repo/wiki/pages.tmpl`](../appliance/forgejo/templates/repo/wiki/pages.tmpl) | Page names and modification times | This is a document index with original Git-entry links and writer-gated new-page action. Keep scan density; full source reviewed, populated index not captured. |
| No image | [`repo/wiki/revision.tmpl`](../appliance/forgejo/templates/repo/wiki/revision.tmpl) | Native commit history | Author/time, revision count, clone controls and paginated commits provide the relevant evidence. No decorative timeline; full source reviewed, native history not captured. |
| No image | [`repo/wiki/search.tmpl`](../appliance/forgejo/templates/repo/wiki/search.tmpl) | Wiki reader search dropdown | Caller traced through the reader HTMX target. Result titles/snippets or explicit no-results feedback need no decorative image; full fragment reviewed. |
| Done | [`repo/wiki/start.tmpl`](../appliance/forgejo/templates/repo/wiki/start.tmpl) | `wiki-welcome-papercraft.png` | Robot opens blank reference book in dedicated welcome state; native translated text and writer/mirror gate preserved. Desktop/mobile inspected; see evidence below. |
| No image | [`repo/wiki/view.tmpl`](../appliance/forgejo/templates/repo/wiki/view.tmpl) | Authored document, sidebar and footer | Reader includes author/history, search, table of contents, format warnings and Unicode escape controls. Preserve authored visual hierarchy. Full source reviewed including sidebar/footer edit and delete controls; populated native page not captured. |
| Partial — trace caller | [`shared/actions/runner_create.tmpl`](../appliance/forgejo/templates/shared/actions/runner_create.tmpl) | — | Personal, organization and repository pages assessed without artwork; admin owner layout remains pending. |
| Partial — trace caller | [`shared/actions/runner_details.tmpl`](../appliance/forgejo/templates/shared/actions/runner_details.tmpl) | — | Personal, organization and repository pages assessed without artwork; admin owner layout remains pending. |
| Partial — trace caller | [`shared/actions/runner_edit.tmpl`](../appliance/forgejo/templates/shared/actions/runner_edit.tmpl) | — | Personal, organization and repository pages assessed without artwork; admin owner layout remains pending. |
| Partial — trace caller | [`shared/actions/runner_list.tmpl`](../appliance/forgejo/templates/shared/actions/runner_list.tmpl) | — | Personal caller reviewed; status table needs no art. Other owner pages remain pending. |
| Partial — trace caller | [`shared/actions/runner_setup.tmpl`](../appliance/forgejo/templates/shared/actions/runner_setup.tmpl) | — | Personal, organization and repository pages assessed without artwork; admin owner layout remains pending. |
| Partial — trace caller | [`shared/blocked_users_list.tmpl`](../appliance/forgejo/templates/shared/blocked_users_list.tmpl) | — | Not yet reviewed in this goal. |
| Partial — trace caller | [`shared/quota_overview.tmpl`](../appliance/forgejo/templates/shared/quota_overview.tmpl) | — | Not yet reviewed in this goal. |
| Partial — trace caller | [`shared/secrets/add_list.tmpl`](../appliance/forgejo/templates/shared/secrets/add_list.tmpl) | — | Personal caller reviewed; inline list and add/edit dialogs need no art. Other owner pages remain pending. |
| Partial — trace caller | [`shared/variables/variable_list.tmpl`](../appliance/forgejo/templates/shared/variables/variable_list.tmpl) | — | Personal caller reviewed; exact configuration and dialogs need no art. Other owner pages remain pending. |
| Done | [`status/404.tmpl`](../appliance/forgejo/templates/status/404.tmpl) | `not-found-papercraft.png` | Small neutral wayfinding scene; native general/repository 404 captures inspected at desktop/mobile widths. |
| No image | [`status/413.tmpl`](../appliance/forgejo/templates/status/413.tmpl) | — | Payload-too-large diagnostic: keep the brief error code/explanation immediately visible after a failed request. Additional artwork would lengthen this corrective interruption; source reviewed, no oversized request submitted. |
| Pending | [`user/auth/activate.tmpl`](../appliance/forgejo/templates/user/auth/activate.tmpl) | — | Not yet reviewed in this goal. |
| Pending | [`user/auth/change_passwd.tmpl`](../appliance/forgejo/templates/user/auth/change_passwd.tmpl) | — | Not yet reviewed in this goal. |
| Pending | [`user/auth/forgot_passwd.tmpl`](../appliance/forgejo/templates/user/auth/forgot_passwd.tmpl) | — | Not yet reviewed in this goal. |
| Pending | [`user/auth/grant.tmpl`](../appliance/forgejo/templates/user/auth/grant.tmpl) | — | Not yet reviewed in this goal. |
| Pending | [`user/auth/grant_error.tmpl`](../appliance/forgejo/templates/user/auth/grant_error.tmpl) | `repo/header` | Not yet reviewed in this goal. |
| Pending | [`user/auth/link_account.tmpl`](../appliance/forgejo/templates/user/auth/link_account.tmpl) | — | Not yet reviewed in this goal. |
| Pending | [`user/auth/prohibit_login.tmpl`](../appliance/forgejo/templates/user/auth/prohibit_login.tmpl) | — | Not yet reviewed in this goal. |
| Pending | [`user/auth/reset_passwd.tmpl`](../appliance/forgejo/templates/user/auth/reset_passwd.tmpl) | — | Not yet reviewed in this goal. |
| Existing — verify | [`user/auth/signin.tmpl`](../appliance/forgejo/templates/user/auth/signin.tmpl) | `login-papercraft.png` | Not yet reviewed in this goal. |
| Pending | [`user/auth/signin_openid.tmpl`](../appliance/forgejo/templates/user/auth/signin_openid.tmpl) | — | Not yet reviewed in this goal. |
| Pending | [`user/auth/signup.tmpl`](../appliance/forgejo/templates/user/auth/signup.tmpl) | — | Not yet reviewed in this goal. |
| Pending | [`user/auth/signup_openid_connect.tmpl`](../appliance/forgejo/templates/user/auth/signup_openid_connect.tmpl) | — | Not yet reviewed in this goal. |
| Pending | [`user/auth/signup_openid_register.tmpl`](../appliance/forgejo/templates/user/auth/signup_openid_register.tmpl) | — | Not yet reviewed in this goal. |
| Pending | [`user/auth/twofa.tmpl`](../appliance/forgejo/templates/user/auth/twofa.tmpl) | — | Not yet reviewed in this goal. |
| Pending | [`user/auth/twofa_scratch.tmpl`](../appliance/forgejo/templates/user/auth/twofa_scratch.tmpl) | — | Not yet reviewed in this goal. |
| Pending | [`user/auth/webauthn.tmpl`](../appliance/forgejo/templates/user/auth/webauthn.tmpl) | — | Not yet reviewed in this goal. |
| Pending | [`user/code.tmpl`](../appliance/forgejo/templates/user/code.tmpl) | `org/header`, `user/overview/header` | Not yet reviewed in this goal. |
| Existing — verify | [`user/dashboard/dashboard.tmpl`](../appliance/forgejo/templates/user/dashboard/dashboard.tmpl) | `dashboard-papercraft.png` | Not yet reviewed in this goal. |
| Existing — verify | [`user/dashboard/issues.tmpl`](../appliance/forgejo/templates/user/dashboard/issues.tmpl) | `issues-papercraft.png`, `pulls-papercraft.png` | Not yet reviewed in this goal. |
| Existing — verify | [`user/dashboard/milestones.tmpl`](../appliance/forgejo/templates/user/dashboard/milestones.tmpl) | `milestones-papercraft.png` | Not yet reviewed in this goal. |
| Existing — verify | [`user/notification/notification_div.tmpl`](../appliance/forgejo/templates/user/notification/notification_div.tmpl) | `notifications-papercraft.png` | Not yet reviewed in this goal. |
| Done | [`user/notification/notification_subscriptions.tmpl`](../appliance/forgejo/templates/user/notification/notification_subscriptions.tmpl) | `subscriptions-papercraft.png`, `watching-papercraft.png` | Both page variants have separate integrated artwork and inspected native captures. |
| Partial — no image | [`user/overview/header.tmpl`](../appliance/forgejo/templates/user/overview/header.tmpl) | Native tab navigation | Shared across profile, package, code and project pages; only navigation/permission gates/counts. Art belongs to an assessed page intro, never this repeated tab strip. |
| No image | [`user/overview/package_versions.tmpl`](../appliance/forgejo/templates/user/overview/package_versions.tmpl) | `org/header`, `user/overview/header` | Both owner branches are focused version lookup pages; preserve compact intro and filters without decoration. Source assessment. |
| Variants — see below | [`user/overview/packages.tmpl`](../appliance/forgejo/templates/user/overview/packages.tmpl) | Personal: `personal-packages-papercraft.png`; organization: `organization-packages-papercraft.png` | Personal complete; organization integrated, native verification pending. |
| No image | [`user/profile.tmpl`](../appliance/forgejo/templates/user/profile.tmpl) | Native avatar, README and activity chart | Individually assessed repositories, activity, stars, followers, following and README/conditional-watching variants below; identity and authored content take priority. |
| Done | [`user/settings/account.tmpl`](../appliance/forgejo/templates/user/settings/account.tmpl) | `settings-account-papercraft.png` via layout | Existing mailbox/account scene retained; native desktop/mobile introduction inspected. |
| Done | [`user/settings/appearance.tmpl`](../appliance/forgejo/templates/user/settings/appearance.tmpl) | `settings-appearance-papercraft.png` via layout | Existing swatch scene retained; native desktop/mobile introduction verified. |
| No image | [`user/settings/access_token_edit.tmpl`](../appliance/forgejo/templates/user/settings/access_token_edit.tmpl) | `hideArtwork` | Native resource/scope selection takes priority; desktop creation form checked. |
| Done | [`user/settings/applications.tmpl`](../appliance/forgejo/templates/user/settings/applications.tmpl) | `settings-applications-papercraft.png` via layout | Existing application-connection scene retained; native desktop/mobile landing captures inspected. |
| No image | [`user/settings/applications_oauth2_edit.tmpl`](../appliance/forgejo/templates/user/settings/applications_oauth2_edit.tmpl) | `hideArtwork` | Credential and redirect configuration; source reviewed, native edit verification pending. |
| No image | [`user/settings/applications_oauth2_list.tmpl`](../appliance/forgejo/templates/user/settings/applications_oauth2_list.tmpl) | Personal, organization, admin application pages | Shared list/create section; native app identity, client IDs, locked state and configuration controls take priority. Page owners assessed separately. |
| Done | [`user/settings/hooks.tmpl`](../appliance/forgejo/templates/user/settings/hooks.tmpl) | `settings-webhooks-papercraft.png` | Personal webhook list only; desktop/mobile and new-form isolation checked. |
| Done | [`user/settings/keys.tmpl`](../appliance/forgejo/templates/user/settings/keys.tmpl) | `settings-keys-papercraft.png` via layout | Existing key-rack scene retained; native desktop/mobile landing captures inspected. |
| Partial — trace caller | [`user/settings/layout_footer.tmpl`](../appliance/forgejo/templates/user/settings/layout_footer.tmpl) | — | Not yet reviewed in this goal. |
| Existing — verify | [`user/settings/layout_head.tmpl`](../appliance/forgejo/templates/user/settings/layout_head.tmpl) | `settings-account-papercraft.png`, `settings-appearance-papercraft.png`, `settings-applications-papercraft.png`, `settings-keys-papercraft.png`, `settings-profile-papercraft.png`, `settings-security-papercraft.png` | Not yet reviewed in this goal. |
| Done | [`user/settings/organization.tmpl`](../appliance/forgejo/templates/user/settings/organization.tmpl) | `settings-organizations-papercraft.png` | Distinct membership-card scene; desktop/mobile empty-state captures checked. |
| Done | [`user/settings/packages.tmpl`](../appliance/forgejo/templates/user/settings/packages.tmpl) | `settings-packages-papercraft.png` | Native landing-page content preserved; explicit artwork input excludes cleanup pages. |
| Done | [`user/settings/profile.tmpl`](../appliance/forgejo/templates/user/settings/profile.tmpl) | `settings-profile-papercraft.png` via layout | Existing portrait-frame artwork retained; native desktop/mobile header captures inspected. |
| Done | [`user/settings/security/security.tmpl`](../appliance/forgejo/templates/user/settings/security/security.tmpl) | `settings-security-papercraft.png` | Existing security scene retained; native desktop/mobile landing captures inspected. |
| No image | [`user/settings/security/twofa_enroll.tmpl`](../appliance/forgejo/templates/user/settings/security/twofa_enroll.tmpl) | `user/settings/layout_head` | Explicitly suppress decoration for enrollment/re-enrollment; QR and passcode are the task. Native capture unperformed. |
| Partial — trace caller | [`webhook/new.tmpl`](../appliance/forgejo/templates/webhook/new.tmpl) | — | Not yet reviewed in this goal. |
| Partial — trace caller | [`webhook/shared-settings.tmpl`](../appliance/forgejo/templates/webhook/shared-settings.tmpl) | — | Not yet reviewed in this goal. |

## Additional page variants

| Status | Organization home/member variant | Decision |
| --- | --- | --- |
| No image | Home with rendered or plain README | Organization-authored content may provide its own imagery; no competing illustration. |
| No image | Home without README | Native organization identity, repository list and conditional member/team sidebar supply context. |
| No image | Public-only members | Real avatars/names and membership visibility are the visual information. |
| No image | Internal/owner member view | Real roles, owner-visible two-factor status and membership actions take priority. |
| No image | Leave/remove member confirmations | Named person/organization and exact confirmation text; no decoration. |

| Status | Organization runner page | Decision |
| --- | --- | --- |
| No image | `org/settings/runners_create.tmpl` | Two-property creation form beneath organization identity. |
| No image | `org/settings/runners_setup.tmpl` | One-time token and setup commands take priority. |
| No image | `org/settings/runners_edit.tmpl` | Properties and token regeneration need direct attention. |
| No image | `org/settings/runners_details.tmpl` | Native runner status, metadata and organization task history supply visuals. |

| Status | Personal runner page | Decision |
| --- | --- | --- |
| No image | `user/settings/runner_create.tmpl` | Short name/description form; keep creation direct without a second visual section. |
| No image | `user/settings/runner_setup.tmpl` | One-time token, UUID and configuration/command instructions are the essential content. |
| No image | `user/settings/runner_edit.tmpl` | Properties and explicit token-regeneration option need clear form hierarchy. |
| No image | `user/settings/runner_details.tmpl` | Actual status, labels, metadata and task history supply the meaningful visuals. |

| Status | OAuth caller / inline state | Decision / remaining work |
| --- | --- | --- |
| No image | Personal OAuth grants section and revoke confirmation | App names, authorization dates and revoke action; no additional decoration within the applications landing page. |
| No image | Shared OAuth create form and application-delete confirmation | Name, redirect URIs, confidentiality choice and exact deletion notice; owner page supplies any header art. |
| No image | `org/settings/applications_oauth2_edit.tmpl` | Organization identity plus credential/redirect form; no additional art. Source assessment, runtime unverified. |
| Pending | `admin/applications/list.tmpl` | Shared list/create caller traced; admin layout/page decision remains pending. |
| Pending | `admin/applications/oauth2_edit.tmpl` | Shared credential form traced; admin layout/page decision remains pending. |

| Status | Personal settings route | Decision / remaining work |
| --- | --- | --- |
| No image | `/user/settings/blocked_users` | Actual identities, block dates and unblock actions; native empty-state desktop capture inspected. |
| No image | `/user/settings/actions/runners` | Runner names, UUIDs, labels, ownership and live status are the relevant visuals; source-reviewed. |
| No image | `/user/settings/actions/secrets` | Names, masked values and edit controls; no decorative scene. Source-reviewed. |
| No image | `/user/settings/actions/variables` | Exact names/values and mutation controls; no decorative scene. Source-reviewed. |
| No image | `/user/settings/storage_overview` | Native quota bars, subject totals and exceeded-limit indicators are the page visuals; source-reviewed. |

| Status | Personal repository settings variant | Decision |
| --- | --- | --- |
| No image | `user/settings/repos.tmpl` — standard inventory | Repository type, owner/name, size and fork origin are the relevant visual information. Preserve compact rows. |
| No image | `user/settings/repos.tmpl` — directory inventory with adoption/deletion enabled | Registered repositories and unadopted directories must remain visually distinguishable beside their permission-gated actions. |
| No image | Adopt pre-existing repository confirmation | Exact directory and confirmation text take priority; no decorative image. |
| No image | Delete pre-existing repository confirmation | Preserve the directory-specific warning and action without added decoration. |

| Status | Personal webhook variant | Decision |
| --- | --- | --- |
| Done | `/user/settings/hooks` | Dedicated connection illustration in list introduction. |
| No image | `user/settings/hook_new.tmpl` — new | Provider icon and endpoint/event fields take precedence. Native Forgejo-provider desktop capture inspected. |
| No image | `user/settings/hook_new.tmpl` — edit | Same provider form plus delivery history; source-based decision, no existing webhook edited or captured. |

Upstream-only page callers traced from the running stock binary with `forgejo embedded view`:

| Status | Native page template / variant | Decision |
| --- | --- | --- |
| Done | `user/settings/packages.tmpl` (now overridden) | Registry-maintenance scene integrated; native desktop/mobile captures and cleanup-page isolation checked. |
| No image | `org/settings/packages.tmpl` | Organization identity header and navigation lead directly to cleanup/Cargo controls; no added decorative intro. Source assessment. |
| No image | `user/settings/packages_cleanup_rules_edit.tmpl` — add | Long retention form; prioritize criteria. Native desktop capture inspected. |
| No image | `user/settings/packages_cleanup_rules_edit.tmpl` — edit | Same criteria plus remove/preview actions; source-reviewed. |
| No image | `org/settings/packages_cleanup_rules_edit.tmpl` — add | Same shared retention fields under organization navigation; source-reviewed. |
| No image | `org/settings/packages_cleanup_rules_edit.tmpl` — edit | Existing-rule editing and removal; source-reviewed. |
| No image | `user/settings/packages_cleanup_rules_preview.tmpl` | Exact affected versions and count, including zero-result text; source-reviewed. |
| No image | `org/settings/packages_cleanup_rules_preview.tmpl` | Same preview table under organization navigation; source-reviewed. |


- [x] Personal package registry: dedicated wrapping scene integrated and native desktop/mobile captures inspected.
- [ ] Organization package registry: shared-shelf scene integrated; native verification pending.

| Status | Page variant | Decision / evidence |
| --- | --- | --- |
| Done | `/notifications/subscriptions` (`Status == 1`) | Dedicated conversation-bookmark scene integrated; desktop/mobile captures inspected. |
| Done | `/notifications/watching` (`Status == 2`) | Dedicated binoculars/repositories scene; desktop/mobile native captures inspected. |
| No image | Personal package version list | Existing profile identity plus package name gives context; keep version search and chronology prominent. |
| No image | Organization package version list | Organization identity already supplies context; version and tag selection are the task, not registry discovery. |
| No image | Package version detail, individual or organization owner | Preserve package name/version, install content, native assets and metadata; no added illustration in the common shell. Protocol-specific content remains upstream-owned. |
| No image | Contributor repositories | Actual avatar plus searchable repository list; a decorative header would delay browsing and compete with identity. |
| No image | Contributor public activity | Native heatmap/feed already provide relevant visuals; a scene would push activity below the fold. |
| No image | Contributor starred repositories | Personal selection list belongs directly below profile navigation; retain actual repository entries and truthful empty result. |
| No image | Contributor followers | Preserve real people’s avatars and names; do not substitute a robot crowd, including in the empty state. |
| No image | Contributor following | Same identity-card responsibility but a separate relationship list; no decorative scene above the people. |
| No image | Contributor README overview | The contributor authors this content and may supply their own artwork. Source-reviewed; no generated illustration ahead of it. |
| No image | Contributor watched repositories (stars-disabled navigation variant) | Source-reviewed profile list under the same identity shell; distinct from the separate `/notifications/watching` introduction. |

Organization settings routes discovered in the native navbar (each still needs its own source/layout assessment):

| Status | Organization-relative route | Page |
| --- | --- | --- |
| No image | `/settings` | Organization identity/avatar already supplies visual context; prioritize profile, visibility and permission fields. |
| No image | `/settings/hooks` | Organization identity plus native endpoint/status list; new/edit provider controls remain undecorated. Source-reviewed. |
| No image | `/settings/labels` | Real label colors, names, counts and editing previews are the visual content; source-reviewed. |
| No image | `/settings/applications` | Organization identity plus actual app names/client IDs and creation controls; source-reviewed. |
| No image | `/settings/actions/runners` | Actual runner status/labels and registration controls; source-reviewed under organization identity. |
| No image | `/settings/actions/secrets` | Secret names, masked values and mutation dialogs; source-reviewed under organization identity. |
| No image | `/settings/actions/variables` | Exact names/values and mutation dialogs; source-reviewed under organization identity. |
| No image | `/settings/blocked_users` | Organization blocked users: actual identities, search and block/unblock controls take priority; source-reviewed. |
| No image | `/settings/storage_overview` | Organization quota bars/totals/status are the relevant visuals; source-reviewed. |
| No image | `/settings/delete` | Warning and exact organization-name confirmation take priority; no decorative image in page or modal. |

- [ ] Split combined Explore users/organizations, dashboard issues/pulls and personal/organization page branches into individual decisions.
- [ ] Trace account, organization, repository and administrator shared layouts for native routes without leaf overrides.
- [ ] Distinguish new/edit/detail states where one template serves multiple page purposes.

## Completed page records

### Package version lists and detail

- Source inspected: both owner branches in `user/overview/package_versions.tmpl`, the full `package/shared/versionlist.tmpl`, and full `package/view.tmpl`.
- Personal version lookup already includes the owner's identity card; organization lookup includes its header. Both add a compact package-name introduction, breadcrumb, search/sort and version rows. Container versions also have a tagged/untagged selector. A decorative scene would add height before the release-selection task; neither branch receives one. Empty filtered results retain their truthful text.
- Detail uses the actual package/version heading and publication information above protocol content, with metadata, downloads, files and latest versions alongside. Its 23 native content includes and 22 metadata includes remain unchanged. No decorative image in this shared detail shell for either owner type; this is not a claim to have reviewed or rendered every upstream protocol partial.
- Read-only public package inventories for Alice and Vince returned empty lists, in addition to the previously captured empty Bob registry. No package was published to manufacture a screenshot. These decisions are based on source/layout responsibilities; no populated version/detail browser capture is claimed. No source behavior changed, so no new tests or template reload were needed.

### Organization package registry — integrated, native verification pending

- Appropriate in the existing compact introduction. Sorting packages onto a shared shelf distinguishes the team collection from personal package wrapping. [Exact prompts and provenance](../assets/branding/forgejo/organization-packages-art-prompt.md).
- Initial RGB checkerboard rejected; built-in cutout correction produced visually inspected 1536×1024 RGBA with transparent corners. Only the organization branch's Artwork field changed; native owner navigation and shared list remain intact.
- Focused offline `TestForgejoPackages` tests passed, including personal/organization owner branches.
- Read-only `/api/v1/orgs?limit=20` returned an empty list. No organization fixture was created, so native desktop/mobile verification remains pending and this page is not marked Done. Proceeding to independent pages.

### Repository migration chooser

- Decision: appropriate in the existing intro; replace reused new-repository art with a distinct import/history scene.
- Prompt and rationale: [migrate-art-prompt.md](../assets/branding/forgejo/migrate-art-prompt.md). Built-in image generator; fixed Dashboard/Notifications/New-repository references.
- Integration: only the existing `Artwork` filename changed; native provider controls/routes were preserved. PNG verified as 1536×1024 RGBA with transparent corners.
- Local stock Forgejo template reload returned `Reloaded`. Focused offline `go test -count=1 -mod=readonly ./scripts -run TestForgejoOnboarding` passed.
- Actual authorized screenshot-helper captures inspected: `.artifacts/screenshots/capture-LnR3TI/001.png` (1440×1000), `.artifacts/screenshots/capture-gYsMJg/001.png` (390×844). The new illustration displays cleanly on the fixture's light theme, stays within the mobile header, and does not cover provider controls. Baseline: `capture-QmgQHa/001.png`.
- No migration submission or fixture/account change. Dark-theme rendering and provider-specific form states were not newly exercised. Template reload refreshes the local cache globally; these captures validate this page, not every unrelated override.

### Repository fork

- Decision: appropriate in the existing intro. Two notebooks connected by a branching ribbon show a new independent copy with a retained connection.
- [Exact prompt/provenance](../assets/branding/forgejo/fork-art-prompt.md); selected transparent 1536×1024 PNG uses the same three style references as Migration and Settings.
- Only the existing Artwork filename changed. Focused offline `TestForgejoOnboarding` tests passed; local templates reloaded.
- Actual `/repo/fork/23` screenshots inspected using the authorized fixture: `.artifacts/screenshots/capture-Ja64Ea/001.png` (1440×1000) and `capture-fK2bfz/001.png` (390×844). Art is contained, cream surfaces remain intact, and native ownership/source/name/visibility/branch controls remain visible without overlap. Baseline: `capture-SBm2fy/001.png`.
- No fork submitted and no fixture/account preferences changed. Dark appearance and POST validation were not exercised.

### Not-found page (general and repository context)

- Appropriate as a small decorative map/compass scene; it does not assert that a resource was deleted or exists behind a permission boundary. The native 404 heading, translated/custom explanation, conditional recovery link and version remain intact. [Exact prompt](../assets/branding/forgejo/not-found-art-prompt.md).
- Transparent RGBA 1536×1024 image; local status CSS limits display to 220px and 65% of the card. Empty alt text preserves the accessible diagnostic text.
- Focused offline `TestForgejoStatus` tests passed, including escaped custom messages. Test context gained only the native asset-prefix function required by the decorative image.
- Initial captures exposed cached status CSS and oversized artwork (`capture-bFddDz`, `capture-ziCjI7`); rejected. Bumped status.css to v3 and reloaded local templates.
- Final native screenshot-helper captures inspected: `.artifacts/screenshots/capture-wbiIF7/{001,002}.png` at 1440×1000 and `capture-geC5uV/{001,002}.png` at 390×844. The two routes are `/soda-art-missing-page` and `/bob/activity-field-notes/src/branch/soda-art-missing-branch`. Error text, illustration and repository navigation fit with no visible clipping. Baseline: `capture-D1CnJg/001.png`.
- No data mutation. Light theme inspected; dark theme and conditional recovery-link runtime state not exercised. The 413 decision is a source/layout judgment, not newly triggered runtime evidence.

### Notification subscriptions

- Appropriate in the existing intro. The robot bookmarks conversation cards, distinguishing followed discussions from inbox delivery. [Exact generation/correction prompts](../assets/branding/forgejo/subscriptions-art-prompt.md).
- Initial RGB/checkerboard output rejected; one built-in cutout edit produced verified 1536×1024 RGBA with transparent corners. The existing `Status == 1` native branch selects the new asset; no native controls, filtering, list logic or status mutations changed.
- Focused offline shared-page-boundary and notification-preview Go tests passed. Local templates reloaded.
- Actual fixture screenshots inspected: `.artifacts/screenshots/capture-Pe7Zti/001.png` (1440×1000 subscriptions), `capture-MrHjH1/001.png` (390×844 subscriptions), and `capture-Pe7Zti/002.png` (Watching isolation check). The decorative scene is contained, native tabs/filters and truthful empty state remain. Watching still shows the original inbox art. Baseline `capture-fZfjV8/001.png`.
- Initial Chrome launch failed transiently; read-only process/lock inspection found no live fixture browser or lock, and a second launch succeeded without deleting profile data. No subscription, account or fixture changes. Populated list, dark appearance and bulk actions were not exercised.

### Watched repositories

- Appropriate in the existing header. Binoculars and repository folders distinguish watching whole repositories from inbox messages and bookmarked discussions. [Exact prompt](../assets/branding/forgejo/watching-art-prompt.md).
- Verified 1536×1024 RGBA with transparent corners. The shared template defaults to Watching artwork and retains its existing subscriptions-specific override; no native controls or watch records changed.
- Focused offline shared presentation-boundary and notification-preview Go tests passed. Local templates reloaded.
- Native screenshot-helper evidence inspected: `.artifacts/screenshots/capture-kRrPCS/001.png` (1440×1000 Watching), `capture-aNBSUP/001.png` (390×844 Watching), `capture-kRrPCS/002.png` (Subscriptions isolation check). Image, search and filter controls fit; the original truthful empty result remains. Baseline: `capture-Pe7Zti/002.png`.
- No account/fixture/watch mutation. Populated repositories, dark appearance and filter interactions were not newly exercised.

### Public contributor profile and tabs

- Source inspected: `user/profile.tmpl` (all tab/visibility branches), `user/overview/header.tmpl` (conditional navigation and native counts), and `repo/user_cards.tmpl` (real avatar/name cards plus pagination). Caller search traced the navigation into packages, code and projects, which remain separate pending pages.
- Decision: no generated image for the seven profile variants listed above. The large native identity card is already the main visual; repositories, an authored README, the activity heatmap/feed and actual social identities should receive the remaining space. Mobile captures confirm the identity card already consumes much of the first viewport.
- Native screenshot-helper captures inspected: `.artifacts/screenshots/capture-v555UO/{001,002,003}.png` at 1440×1000 for Bob’s repository, activity and starred tabs; `capture-mmWzCs/{001,002,003,004}.png` at 390×844 for repository, activity, followers and following tabs. Real repository/activity content and empty star/social states were visible. No personal credential content was captured; the fixture’s public `.invalid` contact is test data.
- README and stars-disabled variants are source-based decisions, not claimed runtime captures. No templates, assets, profile content or relationships changed. No tests/reload needed for this documentation-only assessment.

### Personal package registry

- Appropriate in its dedicated compact introduction, distinct from basic profile tabs. A small wrapping scene identifies package publishing while the real owner avatar and native list remain separate. [Exact prompt](../assets/branding/forgejo/personal-packages-art-prompt.md).
- Selected 1536×1024 RGBA image with transparent corners; no alpha repair. Added Artwork only to the individual-owner branch of `user/overview/packages.tmpl`, retaining the organization branch and shared package renderer.
- Focused offline `TestForgejoPackages` tests passed; local templates reloaded.
- Native `/bob/-/packages` screenshot-helper captures inspected: `.artifacts/screenshots/capture-idaJqm/001.png` (1440×1000), `capture-xqKhxu/001.png` (390×844). The illustration fits the introduction without overlapping identity/navigation/text. Desktop shows the unchanged truthful empty registry and documentation link. Baseline `capture-4tPRwq/001.png`.
- No packages, account preferences or fixtures changed. Populated registry, dark appearance and package operations were not exercised; this is illustration/layout evidence.

### Package settings and cleanup pages

- Inspected complete package settings, Cargo and all three cleanup overrides. Read the six native user/organization settings callers directly from the running Forgejo binary via its read-only `embedded view` command, rather than assuming they were absent because no leaf override exists.
- Package-version settings retain association controls and the native deletion warning/modal without decoration, for both owner types. The cleanup list and Cargo are inline sections, not separate image destinations. Add/edit cleanup rules retain clear keep/remove semantics; previews retain exact counts and version rows without a decorative scene.
- Native authorized helper captures inspected: `.artifacts/screenshots/capture-AyNtZk/001.png` for personal registry settings and `002.png` for add cleanup rule, both 1440×1000. The settings introduction has room for a compact scene. The lengthy cleanup form already extends below the viewport; no illustration is appropriate there. Edit/preview and organization states are source assessments, not runtime claims.
- No forms submitted, keys generated, Cargo index initialized, cleanup rules created or package state changed. No source changes or tests required for these decisions. The next image must target only the settings landing page, not every page sharing its settings flag/layout.

### Personal registry settings illustration

- [Exact prompts/provenance](../assets/branding/forgejo/settings-packages-art-prompt.md). Robot adjusts the gear on a package organizer, distinguishing maintenance from public registry publishing. Initial RGB output rejected; built-in correction yielded visually inspected 1536×1024 RGBA with transparent corners.
- Added the stock 15.0.7 personal settings landing template as an official override, changing only its layout call to supply artwork. The shared layout accepts that explicit input; existing artwork selection remains. Cleanup callers do not supply it, so they remain undecorated. Inventory now contains 204 overrides/helpers.
- Focused offline `TestForgejoPackages` and `TestForgejoPagesComposeSharedPresentationWithNativeBoundaries` passed. Local templates reloaded.
- Inspected native captures: `.artifacts/screenshots/capture-gzvM6u/001.png` (1440×1000 landing), `002.png` (add-cleanup isolation), and `capture-X8BBed/001.png` (390×844 landing). The image composites cleanly and fits beside the heading without overlap. Mobile settings navigation still precedes the form below the viewport. Baseline `capture-AyNtZk/001.png`.
- No forms submitted or registry state changed. Dark theme, initialized Cargo, populated cleanup lists and edit/preview runtime states were not exercised.

### Organization registry settings

- Inspected the complete overridden organization settings layout and organization identity header, plus stock `org/settings/packages.tmpl` and `org/settings/navbar.tmpl` through the running binary's read-only embedded-resource viewer.
- No image: unlike the personal landing page, this layout has no dedicated page introduction. The native organization avatar/name/description and horizontal navigation already precede a settings sidebar and the cleanup/Cargo sections. Adding a robot would create another header before two operational sections. Retain the real organization identity and direct access to those controls.
- This decision is specific to registry settings; other organization settings pages remain pending individually. The native navbar exposed ten more route groups, now explicitly listed, including configuration-dependent pages. No blanket no-image rule was assigned to the shared layout.
- Source-based assessment only: no accessible organization fixture is available for native capture. No templates, organization records, Cargo index or cleanup rules changed. No tests were needed for this documentation-only decision.

### Personal webhook list

- Inspected native `user/settings/hooks.tmpl` and `hook_new.tmpl` via the running binary's embedded viewer, the shared provider-dispatch override and initial shared event controls. Personal new/edit forms retain provider identity, endpoint configuration and delivery history without decorative art. Other owners/providers remain pending; this does not close all shared webhook partial callers.
- Generated a distinct endpoint-connection scene using the fixed references; [exact prompts/provenance](../assets/branding/forgejo/settings-webhooks-art-prompt.md). Initial RGB/checkerboard rejected; built-in correction produced visually inspected 1536×1024 RGBA with transparent corners.
- Added a stock personal list override, changing only its layout call to pass the existing explicit artwork input. Inventory now contains 205 overrides/helpers. Native list/actions remain unchanged.
- Focused offline `TestForgejoWebhookPartialsRetain1507Source` and `TestForgejoPagesComposeSharedPresentationWithNativeBoundaries` passed; local templates reloaded.
- Native captures inspected: `.artifacts/screenshots/capture-AsQZYE/001.png` (1440×1000 list), `002.png` (new Forgejo webhook form isolation), and `capture-86gUS7/001.png` (390×844 list). Art fits and composites cleanly; new form remains undecorated. Baselines `capture-v83uiR/{001,002}.png`.
- No webhook created, edited, tested or delivered. Empty list/light theme captured; populated list, edit history, other providers and dark theme were not exercised.

### Personal organization memberships

- Read the complete native `user/settings/organization.tmpl` from the running stock binary. The introduction can carry a small membership-card scene while real organization avatars/names and leave controls remain in the list. [Exact prompt/provenance](../assets/branding/forgejo/settings-organizations-art-prompt.md).
- Selected 1536×1024 RGBA with transparent corners; visually inspected, no repair needed. New official override changes the layout call only. Direct line comparison confirmed all subsequent membership list, conditional create permission, pagination, empty state and leave-confirmation content matches stock. Inventory now has 206 overrides/helpers.
- Focused offline `TestForgejoPagesComposeSharedPresentationWithNativeBoundaries` passed; local templates reloaded.
- Native captures inspected: `.artifacts/screenshots/capture-3Mk8rM/001.png` (1440×1000) and `capture-rAjYlf/001.png` (390×844), baseline `capture-m6PFNf/001.png`. Image fits the introduction without overlapping text and preserves the truthful no-memberships result. Mobile navigation remains ahead of the list below the viewport.
- No organization created or membership changed. Populated membership list, leave modal/action and dark theme were not exercised.

### Personal repository settings

- Inspected the full native `user/settings/repos.tmpl` through the running stock binary's read-only embedded viewer. This is a repository inventory rather than collaboration/transfer management. Both main branches and their permission-dependent confirmation modals are now explicit checklist entries.
- No image for this inventory: repository type icons, size, name and fork origin supply precise context. In the directory branch, the difference between registered repositories and unadopted directories, and available adopt/delete actions, is the primary information. Keep confirmation modals focused on the named directory. Empty states retain the same undecorated layout.
- Native `/user/settings/repos` desktop capture inspected at `.artifacts/screenshots/capture-2Mi7ht/001.png` (1440×1000). The authorized fixture owns no repositories; no populated or adoption-enabled runtime evidence is claimed. Those decisions are based on source review.
- No repository, directory, permissions or configuration changed. No new asset or override needed; no tests or reload required for this documentation-only assessment.

### Blocked users and personal navbar coverage

- Read complete native personal and organization blocked-user templates, shared blocked-user list and personal settings navbar through the running binary's read-only embedded viewer.
- No decorative image on either blocked-user page: the real avatar/name and block date identify the account affected by each unblock action. Organization settings additionally has native user search and a block button; those controls and the organization identity remain the relevant context. Empty states retain concise native feedback without invented character scenes.
- Native personal empty-state screenshot inspected: `.artifacts/screenshots/capture-1OqIQs/001.png` (1440×1000). Organization and populated states are source-based assessments, not runtime claims. No blocks/unblocks or other account mutations performed; no source changes or tests needed.
- Personal navbar coverage now explicitly includes conditional Actions runners/secrets/variables and quota storage overview. Profile/account/appearance/security/applications/keys remain covered by existing pending or verify entries; registry/hooks/memberships/repository inventory have individual records above. Navbar inspection does not prove complete coverage of hidden subroutes.

### Existing profile settings artwork

- Read the full profile override, including profile/privacy fields and the separate avatar upload/delete section. Retain the approved portrait-frame illustration selected by `PageIsSettingsProfile`; it identifies profile editing without replacing the actual avatar control. Original prompt and alpha provenance remain in `settings-art-prompts.md`. No asset or native template changed.
- Native desktop capture inspected: `.artifacts/screenshots/capture-Q1c7e7/001.png` (1440×1000). Image is contained beside the intro; profile fields stay separate. Initial mobile capture `capture-6HUSt7/001.png` shows native autofocus scrolling to the username field, so it is form evidence, not header evidence.
- Added optional `--scroll-top` to the authorized screenshot helper and documented it. Default behavior remains unchanged. The flag scrolls after settling without disabling autofocus or changing native DOM/content. Actual 390×844 run inspected at `capture-gmGqrm/001.png`: header illustration fits without overlap. Successful run validates the added capture option; no synthetic tests added.
- No profile, privacy, avatar or account preference was changed. Dark theme, avatar upload/delete, validation errors and reverse-proxy/rename-disabled variants were not exercised. Those source branches retain the same decorative header and native controls.

### Existing account settings artwork

- Read the complete account override: password availability/local/OAuth conditions, email addresses and activation states, notification preferences, add/delete email controls and account deletion warning/confirmation. Retain the approved mailbox/account illustration in the common introduction; email management is a central part of this page. No extra art inside password, email or destructive confirmation sections.
- Native captures inspected with `--scroll-top`: `.artifacts/screenshots/capture-ZmbJ3J/001.png` (1440×1000) and `capture-Tn2wG8/001.png` (390×844). Art fits beside the heading and remains distinct from the forms. Desktop password fields are empty; the displayed email belongs to the authorized test fixture. Mobile navigation still precedes the forms below the viewport.
- Existing prompt/provenance stays in `settings-art-prompts.md`; no asset/template change or regeneration. No tests/reload needed for this verification-only work.
- No passwords entered, emails sent/changed, notifications changed or deletion attempted. Disabled-password, pending activation, deletion-disabled, validation-error and dark-theme states were not exercised; their source conditions remain native.

### Existing appearance settings artwork

- Read the complete appearance override: theme and language choices, repository-unit hints and hidden comment-event groups. The approved swatch scene directly represents visual preferences; retain it in the introduction without duplicating art within the individual forms. Original prompt/provenance remains in `settings-art-prompts.md`.
- Native captures inspected with `--scroll-top`: `.artifacts/screenshots/capture-WrtaPc/001.png` (1440×1000) and `capture-vGSLyp/001.png` (390×844). The swatch illustration fits beside the heading; theme/language controls remain separate. Mobile settings navigation precedes the forms below the viewport.
- No asset, template, theme, language, hints or comment preferences changed. No tests/reload needed for verification-only work. Light rendering with the fixture's follow-system preference was observed; dark rendering, dropdown interaction and submitted preferences were not exercised.

### Security landing and two-factor enrollment

- Read complete security landing and enrollment overrides. Retain the existing security scene on the landing page, separate from truthful native enrollment status and WebAuthn controls. Native captures inspected: `.artifacts/screenshots/capture-lVl9Ud/001.png` (1440×1000) and `capture-0XYdZL/001.png` (390×844); artwork fits without overlap. Existing prompt/provenance remains unchanged.
- Enrollment and re-enrollment need the actual QR code, secret instructions and passcode field as their visual focus. Added an explicit `hideArtwork` layout input on that shared enrollment template, applied after default artwork selection. Native enrollment content, QR source and form remain untouched. No new decorative image generated.
- Focused offline shared-presentation tests passed (command also selected matching security tests if present); local templates reloaded. This is not native enrollment verification. Enrollment was not opened or submitted, avoiding creation/capture of a live enrollment secret; its render check remains outstanding. Landing capture shows the fixture's unenrolled state only.
- No two-factor setting, security key or account link changed. Forced enrollment, enabled TOTP, re-enrollment, OpenID and dark theme runtime states were not exercised.

### Existing SSH/GPG keys artwork

- Read the key landing override and complete native SSH, principal and GPG partials through the running binary's embedded viewer. Retain the approved key-rack introduction; native fingerprints, verification status and activity icons remain distinct within the lists.
- Add-key panels, principal controls, SSH/GPG signature verification blocks and deletion confirmations need no independent decorative art: key material, command/token instructions and exact actions are their content. These are inline states under the existing page header, not new image destinations. Principal availability and disabled-management branches remain native.
- Native captures inspected with `--scroll-top`: `.artifacts/screenshots/capture-iM0k3F/001.png` (1440×1000) and `capture-6KKAv1/001.png` (390×844). Illustration fits without overlap; actual empty lists and the local SSH-disabled/signing-only explanation remain intact. Mobile navigation precedes the key sections below the viewport.
- No keys, signatures, principals or verification challenges were submitted/generated; no configuration changed. Populated lists, add/verify states, deletion modals and dark theme are source assessments/unexercised runtime states. Existing asset and prompt retained; no tests/reload needed for verification-only work.

### Existing applications landing artwork

- Read the complete applications landing override: access-token resources/scopes, activity metadata and regeneration/deletion confirmations, plus native OAuth grants and application includes. Retain the approved application-connection illustration in the introduction; exact token permissions and native grant/application content remain separate below.
- Native captures inspected with `--scroll-top`: `.artifacts/screenshots/capture-flZti2/001.png` (1440×1000) and `capture-fc5zay/001.png` (390×844). Art fits beside the heading; desktop shows empty token/grant lists and the start of the native OAuth creation form. Mobile navigation remains before those sections below the viewport.
- OAuth edit wrapper was inspected, but its included edit form and separate token-creation page remain queued; this landing verification does not close those variants or shared partials.
- No tokens/applications created, credentials regenerated, grants revoked or forms submitted. Populated lists, secret display, OAuth-disabled and dark-theme states were not exercised. Existing asset/prompt unchanged; no tests/reload required for verification-only work.

### OAuth editing and access-token creation

- Read the complete native OAuth edit form and access-token creation template. OAuth editing presents client identity/secret, regeneration and redirect/confidential-client configuration. Token creation presents resource boundaries, repository selection with state-preserving GET pagination and per-category permissions. These controls need no decorative scene.
- Both personal wrappers now explicitly pass `hideArtwork` to the existing layout. Updated the existing token override; its presentation classes and attribution are retained. Inventory remains 206 overrides/helpers. No native form content or handler changed.
- Focused offline shared-presentation test passed; local templates reloaded. Initial token-creation desktop capture (before restoring its existing presentation classes) inspected at `.artifacts/screenshots/capture-Adtqrc/001.png` (1440×1000): no artwork, resource and scope controls visible. No token generated.
- OAuth edit has source evidence only: no existing fixture application is available, and none was created. Native edit verification remains pending. Secret regeneration/display, form submission and mobile token rendering were not exercised.

- Follow-up diff review caught accidental replacement of existing token-page presentation classes while reading stock content. Restored those classes and attribution; final native captures are recorded below.

### Final token creation capture after presentation restoration

- Re-ran the focused shared-presentation test successfully and reloaded local templates after restoring existing token-page classes.
- Inspected `.artifacts/screenshots/capture-Rtrb3T/001.png` (1440×1000) and `capture-GCYOXS/001.png` (390×844), using `--scroll-top`. Decorative artwork is absent; desktop resource boundaries and permission rows retain their original Soda styling. Mobile header/navigation fit; the form remains below the viewport. These supersede the earlier token-page capture for final presentation evidence.
- No token name, scope, resource selection or account state changed; no token generated. Native OAuth edit verification remains pending independently.

### OAuth shared-section caller audit

- Read complete overridden `applications_oauth2_list` and native `applications_oauth2`, `grants_oauth2`, organization application list/edit and administrator application list/edit wrappers via the running binary's embedded viewer. Correct admin path is `admin/applications/list.tmpl`; an initial lookup at `admin/applications.tmpl` returned no match, then the embedded inventory resolved it.
- Shared list/create partial: no independent image. Application names/client IDs, locked built-ins, edit/delete controls and redirect/confidentiality fields are the content. Existing personal page artwork remains its sole decorative introduction. Grants and revoke confirmation likewise retain actual application identity and dates without extra art.
- Organization/admin caller entries are explicit and remain pending where their full page assessment/rendering is incomplete. No new runtime evidence or mutation; no source changes or tests needed for this caller audit.

### Personal Actions lists

- Read native `user/settings/actions.tmpl` from the running binary and complete overridden runner list, secret add/list and variable list. The native `PageType` dispatch explicitly maps all three personal routes; no new owner or backend inferred.
- Runners: no decorative image. The table's native active/idle/offline indicators, UUIDs, labels and ownership are the relevant visuals. Keep search, registration controls and edit/delete actions direct. Registration-token dropdown was not opened.
- Secrets: no decorative image. Masked values, names, dates and exact add/edit help take priority. Variables: no decorative image. Actual configuration values and mutation controls take priority. Their add/edit/delete dialogs also need no independent artwork.
- These are source assessments. The fixture navbar did not expose Actions, and no configuration was changed to enable it. No runner registration, token reset, secret/variable mutation or native runtime capture is claimed. Repository, organization and administrator callers remain pending separately; runner create/setup/edit/detail pages are next.
- No source change or tests required for this assessment.

### Personal runner subpages

- Read all four full shared runner overrides and the exact stock personal create/setup/edit/details wrappers through the running binary's embedded viewer. Each wrapper uses the personal settings layout without an artwork-specific input.
- Create: no image for a two-property form. Setup: no image ahead of the one-time credential warning, UUID/token and copyable configuration/daemon command. Edit: no image beside properties and the token-regeneration choice. Details: preserve actual active/idle/offline indicators, ownership, labels, version and task run/status/repository/commit history as the visual information.
- These four decisions are source-based. Actions was not enabled and no runner or token was created to manufacture a screenshot. Registration/setup commands were read as source only, not executed. Other owner layouts remain pending; shared partial rows retain that distinction.
- No source changes or tests needed for these assessments.

### Personal and organization storage overview

- Read both native owner wrappers and the complete `shared/quota_overview.tmpl` from the running binary. Each page uses its existing owner settings layout and the same quota renderer, with owner-specific explanatory text.
- No generated illustration for either owner. Quota group/rule names, used/limit values, native acceptable/exceeded icons, colored subject bars and expandable per-subject totals directly explain storage. Additional decoration would compete with those meaningful visuals.
- No separate cleanup link or cleanup subpage appears in these inspected templates; no such route is inferred. Quota-disabled and empty-group cases retain the same source layout without invented state graphics.
- Source-based assessment only: the fixture navbar did not expose quota storage and no configuration was changed to enable it. No usage, quota, file or account mutation performed; no runtime quota rendering or passing runtime validation claimed. No source changes/tests needed.

### Organization general settings and deletion

- Read complete stock `org/settings/options.tmpl` and `delete.tmpl` through the running binary's embedded viewer. Their previously inspected shared layout includes the real organization identity header and navigation.
- General options: no extra image. The page edits that same identity, description/contact fields, public/limited/private visibility, repository-admin team-access option and avatar; the real organization header is the appropriate visual. The administrator-only repository-creation limit remains part of this same undecorated form. No separate illustration for validation or visibility variants.
- Deletion: no image on the warning page or final modal. Keep the exact organization-name confirmation, alert and native destructive action prominent.
- Both decisions are source-based; no organization fixture is available for native capture. No organization/avatar/visibility/permission change or deletion attempted. No source changes/tests required.

### Organization labels and webhooks

- Read complete stock organization labels/hooks/hook-new wrappers, label list/new/edit/delete partials, and webhook list/base-list/delete-modal via the running binary. Initial local-file lookups confirmed these leaf partials are upstream-owned, then the embedded viewer supplied exact content.
- Labels: no extra art in the page, new/edit dialogs or deletion confirmation. Actual rendered labels, colors, descriptions, counts and color selectors convey the task. Preserve exclusive/archive controls and their warnings. The empty-label template-loader include remains native; no claim that its downstream controls were newly exercised.
- Organization webhooks: no additional introduction below the existing organization identity header. Endpoint URLs and native last-status dots are the relevant list visuals. New/edit wrappers use the previously reviewed provider dispatcher and form/history structure; keep those controls and deletion confirmation undecorated. This does not close unreviewed provider-specific partials or other owner pages.
- These decisions are source-based only. No organization fixture, label, webhook or delivery was created/changed. No source changes or tests required; native organization rendering remains unavailable.

### Organization applications and Actions

- Inspected exact stock organization Actions dispatch and all four organization runner wrappers through the embedded viewer; rechecked the organization settings layout. Application list/edit wrappers were read in the earlier OAuth caller audit and use this same layout.
- Application list/create: no added illustration below the real organization header; client identity and configuration controls are the focus. OAuth edit: no decoration beside credentials, regeneration and redirect settings. Unlike the personal edit wrapper, this organization layout never selected applications artwork, so no suppression change is necessary.
- Runners list: native status/labels/ownership table is the visual information. Secrets list: names and masked values. Variables list: exact configuration values. Each retains its native dialogs without extra images.
- Runner creation, setup, editing and detail decisions are individually listed above. They use the previously fully reviewed shared forms/token instructions/status-history content under organization identity; no replacement runner authority or execution is implied.
- Source evidence only; no organization/runner/secret/variable/application was created, modified or captured. Other owner contexts remain pending independently. No source edits/tests required.

### Organization home and members

- Inspected complete current `org/home.tmpl` and `org/member/members.tmpl`, including README plain/rendered states, visibility-conditioned member/team sidebar, repository creation/migration links and member-role/security/action branches.
- Home needs no illustration above its real organization header and authored README/repository content. Members needs no illustration despite its existing compact title: real people, roles and public/private status are the relevant visual content. Owner-only two-factor status stays meaningful and native; it is not replaced by an illustrative security symbol.
- Per-variant decisions are recorded above. All are source assessments; no organization fixture or populated member screenshot is claimed. No membership, visibility, role, README or repository change performed. No source edits/tests needed.

### Team list and detail pages

- Read complete current team list, members and repositories overrides. Each already has organization identity and a compact title; detail pages also include a team sidebar and native tab navigation.
- Team list: no additional artwork. Real member avatars, team names and member/repository counts define each card; owner create/join and member leave actions remain clear. Team members: no decoration beside actual identities, owner-only add/remove controls, last-owner protection and pending email invitations. Team repositories: no decoration beside exact access membership, owner-only search/add/remove and all-repository restrictions.
- Leave/remove confirmations and add-all/remove-all repository dialogs remain text/action focused with no independent image. Empty lists keep their native feedback.
- Source-based decisions only. No organization/team fixture, invitation, membership or repository association changed. Native team screenshots remain unavailable; no source changes/tests required. Creation/edit and invitation pages remain pending separately.

## Team creation and invitation assessment

New-team creation gets its own member-card assembly scene, recorded in [new-team-art-prompt.md](../assets/branding/forgejo/new-team-art-prompt.md). The `PageIsOrgTeamsNew` branch supplies artwork to the existing intro; edit and protected Owners-team forms receive none. All repository scope, administrative/general access, unit permission matrix, disabled units, update/delete controls and native form markup remain intact. The source parity/parse test and shared presentation boundary test passed. Selected PNG was visually inspected and has RGBA transparency. Native organization screenshots remain pending; no organization or invitation was created. Invitation uses its real organization avatar and explicit join action without decorative art.

## Personal/organization project workflow

Creation uses a robot placing the first card on a paper planning board; exact prompt and transparency edit are in [new-project-art-prompt.md](../assets/branding/forgejo/new-project-art-prompt.md). Selected image is RGBA 1536×1024, alpha 0–254. Only the existing wrapper intro changes; shared title/description/template/card-preview fields and native form actions are untouched. `PageIsEditProjects` excludes art during editing. Personal native creation was inspected at 1440×1000 (`capture-4gebxg/001.png`) and 390×844 (`capture-HnZ2xF/001.png`), using the authorized fixture and `--scroll-top`; transparent composition is clean and the form remains reachable. Baseline creation is `capture-bepvu4/002.png`. No project was submitted. Organization creation and edit-state rendering remain pending existing accessible data.

Personal empty project list was inspected at `capture-bepvu4/001.png`; organization/personal list source and full board source were reviewed, including write/archive gating, open/close/delete, sorting/search, authored descriptions, column colors/defaults, issue cards and dialogs. These operational pages need no additional illustration. Organization project context and shared presentation tests passed; `git diff --check` passed. Repository wrappers remain next.

## Repository projects and wiki welcome

Repository project wrappers were individually reviewed against the previously inspected full shared list/form/board partials. Caller search found the repository and personal/organization wrappers; all are now assessed. Repository list capture: `.artifacts/screenshots/capture-4u3Imx/001.png` (1440×1000, existing public Bob repository, non-writer empty state). Repository creation/edit and board decisions are source-based, with no project or permission mutation.

Wiki welcome now replaces the decorative book icon with the distinct [wiki-welcome scene](../assets/branding/forgejo/wiki-welcome-art-prompt.md), RGBA 1536×1024, alpha 0–254. Scoped sizing replaces the obsolete icon styling. Native translated heading/description and `CanWriteWiki`/mirror gate are unchanged; the adjusted source parity test restores only this exact image substitution before checking the pinned upstream hash. `go test ./scripts -run TestForgejoRepositoryContent -count=1` passed. Templates reloaded locally. Desktop native capture: `capture-Mt5rd1/001.png`; mobile: `capture-jIRhpd/001.png` (390×844), inspected with clean transparency and readable native text. Baseline: `capture-6uTxx1/001.png`. Fixture is read-only for this repository, so the create-page button is correctly absent; writer rendering remains unobserved, with gate source preserved.

## Wiki content and release/tag assessment

The wiki editor (new/edit), index, revision history, reader (including authored sidebar/footer and table of contents), and search result/no-result fragment were individually read in full. They retain authored content and native metadata without additional artwork. The illustrated welcome remains the entry point for an empty wiki. These content-state judgments are source-based: the inspected public wiki is empty and no pages were created.

Release creation, tag-only submission, draft editing/publishing, published editing, prerelease selection, attachments/external assets and deletion confirmation were individually considered from the full form. All remain focused native editing states with no decorative illustration. Release list and tag list were read in full, including code/release permission differences, signing indicators, archive suppression, publisher and attachment variants. Native desktop captures show the existing prerelease list (`capture-id7a6e/001.png`) and tag list (`capture-0nKZ6d/001.png`); no downloads, submissions or mutations were performed. Single-release detail was inspected at 390×844 in `capture-SXnGuE/001.png` (`/bob/activity-field-notes/releases/tag/v0.2.0-preview`): prerelease label, publisher, notes and archive links remain readable; no additional artwork needed. No implementation changed; tests were not rerun for checklist decisions.

## Repository activity and history

All captures below used the authorized fixture at 1440×1000 on existing `/bob/activity-field-notes` pages; no repository changes, branch dialogs or downloads were performed. Full branch and graph wrappers, history wrapper and commits_table were read. The selected Forgejo 15.0.7 embedded pulse, contributor, code-frequency, recent-commit and navigation templates were also read to establish the activity variants without adding overrides.

| Route suffix | Decision | Native evidence under `.artifacts/screenshots/` |
| --- | --- | --- |
| `/activity` | No image: period-specific counts, bars, authors and actual issue/release activity are the visual content. | `capture-ExrVGn/001.png` |
| `/activity/contributors` | No image: contribution-type selector, real contributor identity and chart need prominence. | `capture-poIZvf/001.png` |
| `/activity/code-frequency` | No image: native addition/deletion time series already explains the page. | `capture-poIZvf/002.png` |
| `/activity/recent-commits` | No image: native annual commit chart is the subject. | `capture-poIZvf/003.png` |
| `/branches` | No image: repository branches and their actual status/divergence. | `capture-ExrVGn/002.png` |
| `/commits/branch/main` | No image: searchable revision evidence and author identity. | `capture-ExrVGn/003.png` |
| `/graph` | No image: actual ancestry, branch/tag references and commit labels. | `capture-ExrVGn/004.png` |

All seven screenshots were inspected. Default chart state and colored Git graph were observed; loading/error/empty, other contribution types, monochrome and branch mutation states were source-assessed, not exercised. No implementation changed and tests were not rerun for checklist decisions.

## Commit detail, comparison and file finder

Full commit-page wrapper, full comparison wrapper and full file-finder template were reviewed. No decorative image is appropriate: each page presents code evidence or direct navigation. The comparison assessment includes branch/tag and cross-fork selection, swapping head/base, comparison type, no changes (including allowed empty PR), existing PR status, new PR form visibility, archive and sign-in messages. These secondary branches were source-assessed, not submitted.

Authorized native desktop captures, all inspected at 1440×1000, under `.artifacts/screenshots/capture-RMoZvK/`: `001.png` shows `/bob/activity-field-notes/commit/f89919ac0e`; `002.png` shows `/bob/activity-field-notes/compare/5de700b51b...f89919ac0e`; `003.png` shows `/bob/activity-field-notes/find/branch/main`. Real commit metadata, a three-line addition, comparison reference controls and the two-file finder list are visible. No code, note, PR or branch was changed. The shared commit header read was truncated, so its full review remains pending alongside diff-box/pull-request callers. Tests were not rerun for these documentation-only decisions.

## Shared commit/diff and pull-request tabs

Completed the previously truncated commit-header review with bounded reads, covering author/committer, parent links, trusted/untrusted/unmatched signature states, GPG/SSH metadata, Git notes and create-branch/tag/cherry-pick/revert dialogs. Read the entire diff-box partial in two chunks: tree/data loading, viewed-file progress, review controls, selected commit/range notices, unavailable diffs, generated/vendored/protected/LFS/renamed files, text/image/CSV rendering, binary/truncated feedback and comment editor. Each is native evidence or a focused control, not a place for decorative art. Nested upstream renderers were not individually audited as source changes; none were changed.

Read both PR commits/files wrappers and traced every override caller of commits_table, commit_header and diff/box. The no-image decision applies to these tabs and their shared visual components. Prior `capture-RMoZvK/001.png` and `002.png` show ordinary commit/comparison rendering; they are not PR evidence. The public API for `bob/activity-field-notes/pulls?state=all&limit=50` returned `[]`; no PR was created, and PR-specific rendering remains unobserved. These are source-based illustration assessments, with no behavior changes or tests rerun.

## Repository code browser variants

Read the full home wrapper, file preview partial and directory list partial; traced file preview from both home and optional README in the directory list. Also read the selected Forgejo 15.0.7 embedded blame template. No new illustration is appropriate: repository descriptions, authored README/markup/media, actual file names and line authorship supply their own identity and meaning. Source-assessed variants include nested directories, branch/tag/commit context, template repository action, archived/flagged notices, missing description/README, symlink following, submodules, plain text/source, images/video/audio/PDF/3D/raw fallback, citation files, too-large/error/warning feedback and ignored-revision blame notices. Those are not separate decorative empty states.

Native 1440×1000 captures were inspected under `.artifacts/screenshots/capture-ZiPugy/`: `001.png` repository home with file list and README; `002.png` `/src/branch/main/getting-started.md` rendered Markdown; `003.png` `/blame/branch/main/getting-started.md` with line authorship. All use existing `/bob/activity-field-notes` and the authorized read-only fixture context. Other file/media/state variants are source-assessed, not captured. No repository actions or preferences were changed; tests were not rerun for documentation-only decisions.

## Code authoring assessment

Read in full the new/edit file, upload, patch, deletion and cherry-pick/revert templates plus their shared commit form; caller search found exactly those five wrappers. Each has a specific no-image reason in the inventory. The shared form includes actual author identity, signing state/reasons, signoff, permission-dependent direct commit versus new branch, empty-repository restrictions and commit-email selection. These are not decorative setup pages. Native preview/diff and empty-content dialogs likewise retain focused feedback.

All decisions here are source-based. The authorized screenshot fixture has no writable repository; no repository, fork, upload, patch, commit, deletion or cherry-pick was performed to obtain captures. No source implementation changed and tests were not rerun for checklist decisions. The empty repository page remains the next independent assessment.

## Empty repository and social lists

Empty repository was individually assessed across writer quickstart, reader empty message, archived restrictions and broken feedback. No decorative illustration: the writer needs executable Git instructions and the reader has no initialization action; broken/archived states must not suggest successful setup or recovery. The public API repository inventory returned 25 repositories, all `empty: false`; this is no native empty-state proof and no fixture was created.

Fork and watcher wrappers plus stock Forgejo 15.0.7 user_cards were read in full. Watchers and stargazers use actual identity cards and explicit empty feedback; fork rows use actual owner avatars/repository links. Native desktop captures under `.artifacts/screenshots/capture-JgGgfp/` were inspected: `001.png` forks, `002.png` watchers, `003.png` empty stargazers, all on existing `/bob/activity-field-notes`. No stars, watches or forks were changed. No-image decisions recorded for each; no implementation changes or test runs.

## Issue creation and discussion

Read full chooser/new/view overrides and selected Forgejo 15.0.7 embedded new_form and view_content. The blank editor and structured-template dispatch, title defaults, metadata permission gates, maintainer-edit choice for cross-repository PRs, authored conversation, timeline, archived/blocked/locked/sign-in feedback and comment controls all benefit from focused native content. No decorative artwork is appropriate. Individual nested field/comment/sidebar implementations were not audited as code changes; none were changed.

Native desktop screenshots inspected: `capture-QF2Yhi/001.png` blank `/bob/activity-field-notes/issues/new` form and `capture-pf4GoU/001.png` closed issue `/issues/9`, showing actual author, timeline and comment area. No issue/comment/upload/reaction/subscription was submitted. Template chooser, structured issue forms and PR conversation branches remain source-based decisions. Tests were not rerun for documentation-only assessment.

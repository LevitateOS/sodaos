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

Next: resolve native verification gaps for administrator, organization, setup, enabled registration, Actions and code-search pages. All 206 current override files have an artwork assessment. Existing screenshot evidence and limits are recorded per page below; the goal is not complete. Button-text visibility defects remain recorded separately.

## Per-template inventory

| Status | Template / page entry | Current artwork or caller | Decision / evidence |
| --- | --- | --- | --- |
| Integrated — verify | [`admin/applications/list.tmpl`](../appliance/forgejo/templates/admin/applications/list.tmpl) | `admin/layout_head` | Application IDs, built-in locked status and inline redirect-URI/client-type form are the useful content. Leaf and complete shared list/create form reviewed; inherited dashboard artwork now suppressed explicitly. Native capture pending. Source assessment only; current fixture is non-admin. |
| Integrated — verify | [`admin/applications/oauth2_edit.tmpl`](../appliance/forgejo/templates/admin/applications/oauth2_edit.tmpl) | `admin/layout_head` | Focused client configuration/credential form should have no decorative art. Inherited dashboard image is now suppressed by the opt-in layout; native verification remains pending; leaf reviewed, shared form previously assessed. Source assessment only; current fixture is non-admin. |
| Integrated — verify | [`admin/auth/edit.tmpl`](../appliance/forgejo/templates/admin/auth/edit.tmpl) | `admin/layout_head`, artwork suppressed | Full 449-line override reviewed, including LDAP/DLDAP, SMTP, PAM and OAuth branches. Credentials, claims/group mappings, TLS options, activation and deletion instructions need uninterrupted form space. No decorative art; inherited dashboard image suppressed explicitly. Native administrator capture pending. |
| Integrated — verify | [`admin/auth/list.tmpl`](../appliance/forgejo/templates/admin/auth/list.tmpl) | `admin/layout_head`, artwork suppressed | Authentication-source names, types and enabled indicators are the meaningful visuals; retain the compact configuration inventory. Full leaf reviewed; inherited dashboard art suppressed explicitly. Native administrator capture pending. |
| Integrated — verify | [`admin/auth/new.tmpl`](../appliance/forgejo/templates/admin/auth/new.tmpl) | `admin/layout_head`, artwork suppressed | Provider configuration, authentication options and callback instructions need focused form space. Full leaf reviewed, including its LDAP/SMTP/OAuth delegation and inline PAM controls; provider partial bodies not separately reviewed in this pass. Inherited dashboard art suppressed explicitly. Native admin capture pending. |
| Integrated — verify | [`admin/config.tmpl`](../appliance/forgejo/templates/admin/config.tmpl) | `admin/layout_head`, artwork suppressed | Full override reviewed: actual server/security/SSH/LFS/database/service/mail/cache/session/Git/log configuration is a diagnostic reference. Keep native values and test controls prominent; no decorative art. Inherited dashboard image suppressed explicitly. Native admin capture pending. |
| Integrated — verify | [`admin/cron.tmpl`](../appliance/forgejo/templates/admin/cron.tmpl) | `admin/layout_head`, artwork suppressed | Schedules, execution timestamps and actual result icons are the relevant visuals. Full leaf reviewed; inherited artwork suppressed explicitly. Native admin capture pending with the non-admin fixture. |
| Integrated — verify | [`admin/dashboard.tmpl`](../appliance/forgejo/templates/admin/dashboard.tmpl) | `admin/layout_head`, artwork suppressed | Maintenance operations and live system-status content need no decorative scene. Full wrapper reviewed; opt-in layout excludes the dashboard image. Native admin capture pending; focused admin parity and presentation tests passed. |
| Integrated — verify | [`admin/emails/list.tmpl`](../appliance/forgejo/templates/admin/emails/list.tmpl) | `admin/layout_head`, artwork suppressed | Email ownership, primary/activated state and confirmation dialogs need focused attention; no decorative email scene. Full leaf reviewed; inherited dashboard art suppressed explicitly. Native administrator capture pending. |
| Integrated — verify | [`admin/layout_head.tmpl`](../appliance/forgejo/templates/admin/layout_head.tmpl) | Explicit optional artwork input | Full shared layout and current callers assessed. No default artwork or suppression flags remain; admin/user/new alone selects its account illustration. Native administrator rendering remains pending. |
| Integrated — verify | [`admin/notice.tmpl`](../appliance/forgejo/templates/admin/notice.tmpl) | `admin/layout_head`, artwork suppressed | Actual system notices and their detail dialog must remain prominent; an empty list is not proof of system health. Full leaf reviewed; inherited artwork suppressed explicitly. Native admin capture pending with the non-admin fixture. |
| Integrated — verify | [`admin/queue.tmpl`](../appliance/forgejo/templates/admin/queue.tmpl) | `admin/layout_head` | Actual queue types, worker counts and pending counts provide the useful visual information. No decorative art is appropriate; inherited dashboard image is now suppressed by the opt-in layout; native verification remains pending. Full leaf reviewed. Source assessment only; current fixture is non-admin. |
| Integrated — verify | [`admin/queue_manage.tmpl`](../appliance/forgejo/templates/admin/queue_manage.tmpl) | `admin/layout_head` | Actual queue counts, removal control and maximum-worker form need focused diagnostics. No decorative art is appropriate; inherited dashboard image is now suppressed by the opt-in layout; native verification remains pending. Full leaf reviewed. Source assessment only; current fixture is non-admin. |
| Integrated — verify | [`admin/repo/list.tmpl`](../appliance/forgejo/templates/admin/repo/list.tmpl) | `admin/layout_head`, artwork suppressed | Repository visibility, archive/template/mirror flags, sizes and ownership need table space and exact native labels. Full leaf reviewed; inherited dashboard art suppressed explicitly. Native administrator capture pending. |
| Integrated — verify | [`admin/self_check.tmpl`](../appliance/forgejo/templates/admin/self_check.tmpl) | `admin/layout_head`, artwork suppressed | Database and cache warnings or the actual no-problem result communicate the diagnostic outcome; decoration should not imply a result. Full leaf reviewed; inherited artwork suppressed explicitly. Native admin capture pending with the non-admin fixture. |
| Integrated — verify | [`admin/stacktrace.tmpl`](../appliance/forgejo/templates/admin/stacktrace.tmpl) | `admin/layout_head`, artwork suppressed | Process/stacktrace tabs, counts and actual stack content require working space; no decorative scene is useful. Full leaf reviewed; inherited artwork suppressed explicitly. Native admin capture pending with the non-admin fixture. |
| Integrated — verify | [`admin/user/edit.tmpl`](../appliance/forgejo/templates/admin/user/edit.tmpl) | `admin/layout_head`, artwork suppressed | Full override reviewed: identity, authentication source, account permissions, password/2FA changes, avatar and deletion controls need focused editing space. No decorative art; opt-in layout excludes the dashboard image. Native admin capture pending. |
| Integrated — verify | [`admin/user/list.tmpl`](../appliance/forgejo/templates/admin/user/list.tmpl) | `admin/layout_head`, artwork suppressed | Account types, activation, restrictions, 2FA and last login require a scannable status table; decorative people art adds no account-specific information. Full leaf reviewed; inherited dashboard art suppressed explicitly. Native administrator capture pending. |
| Integrated — verify | [`admin/user/new.tmpl`](../appliance/forgejo/templates/admin/user/new.tmpl) | `admin/layout_head` | Full account-creation form reviewed. A compact robot preparing one blank identity card can orient onboarding without depicting permissions or completion. Distinct admin-new-account-papercraft.png generated with built-in image_gen and integrated via an explicit artwork input. Exact prompt/provenance: assets/branding/forgejo/admin-new-account-prompt.md. RGBA inspected; focused admin parity/presentation tests passed. Native admin capture remains pending. |
| Partial — no image | [`custom/explore_empty.tmpl`](../appliance/forgejo/templates/custom/explore_empty.tmpl) | — | Explore repository/people/organization callers traced and reviewed previously. Keep native-purpose icons and caller-specific feedback/actions; existing page intro already owns artwork. No additional image. |
| Partial — no image | [`custom/explore_navbar.tmpl`](../appliance/forgejo/templates/custom/explore_navbar.tmpl) | — | Explore repository, code and people/organization callers traced. Wraps native navigation only; no decorative image belongs in tab navigation. |
| Partial — no image | [`custom/extra_links.tmpl`](../appliance/forgejo/templates/custom/extra_links.tmpl) | — | Native base/head_navbar hook supplies theme/appearance controls. Full override and upstream call site reviewed; preserve meaningful SVG controls, no raster artwork. |
| Partial — no image | [`custom/footer.tmpl`](../appliance/forgejo/templates/custom/footer.tmpl) | — | Global base/footer hook hosts authenticated notification popover and its script. Full override reviewed; compact status/retry/navigation UI needs no decorative artwork. |
| Partial — no image | [`custom/header.tmpl`](../appliance/forgejo/templates/custom/header.tmpl) | — | Global base/head hook loads shared/page styles and guest-theme script gate. Full override and upstream hook reviewed; no visible illustration owned here. |
| Partial — no image | [`custom/soda/empty_content.tmpl`](../appliance/forgejo/templates/custom/soda/empty_content.tmpl) | — | Reviewed full helper and local callers: Explore, package lists, notifications/subscriptions and milestones. Renders optional icon/text only; each page owns its artwork decision. No independently generated image. |
| Partial — no image | [`custom/soda/guest_theme.tmpl`](../appliance/forgejo/templates/custom/soda/guest_theme.tmpl) | — | Full head/navbar route gate reviewed and traced to custom/header and custom/extra_links. Loads theme script or control only; no independent artwork. |
| Partial — no image | [`custom/soda/notification_preview.tmpl`](../appliance/forgejo/templates/custom/soda/notification_preview.tmpl) | — | Traced from notification_div div-only/soda-preview branch, requested by custom/footer. Full fragment reviewed: repository/title/time/status icons and empty message only. Keep compact preview free of decoration; native popover rendering remains unverified here. |
| Partial — no image | [`custom/soda/page_intro.tmpl`](../appliance/forgejo/templates/custom/soda/page_intro.tmpl) | — | Reviewed full helper and all local call sites. Artwork is optional explicit caller input, with empty alt text; no default image or page classification inside the helper. Caller rows retain their native verification gaps. |
| Partial — no image | [`custom/soda/theme_toggle.tmpl`](../appliance/forgejo/templates/custom/soda/theme_toggle.tmpl) | — | Full control and local callers traced across guest shell/auth/setup. Sun/moon SVGs convey theme choice; no raster decoration appropriate. |
| Integrated — verify | [`explore/code.tmpl`](../appliance/forgejo/templates/explore/code.tmpl) | `explore-papercraft.png` | Retain approved discovery artwork shared with repository Explore; no replacement of approved art. Full wrapper and shared search/results reviewed. Native /explore/code displays repository directory instead; code-search rendering remains pending. |
| Done | [`explore/repos.tmpl`](../appliance/forgejo/templates/explore/repos.tmpl) | `explore-papercraft.png` | Retain approved code-folder discovery scene. Full wrapper reviewed; populated repository directory artwork inspected at desktop/mobile. Native search/list/pagination delegates preserved. |
| Done | [`explore/users.tmpl`](../appliance/forgejo/templates/explore/users.tmpl) | `orgs-papercraft.png`, `users-papercraft.png` | Retain distinct approved people and organization scenes. Both wrapper branches reviewed; populated people and empty organization artwork inspected at desktop/mobile. Organization empty-state primary button has invisible text, recorded separately. |
| Done | [`home.tmpl`](../appliance/forgejo/templates/home.tmpl) | `home-papercraft.png` | Retain approved shared-workbench scene: directly supports public collaboration welcome. Full template reviewed; desktop and mobile artwork inspected, including a taller mobile viewport to see the entire scene. Registration-enabled branch remains unobserved. |
| Integrated — verify | [`install.tmpl`](../appliance/forgejo/templates/install.tmpl) | `home-papercraft.png` | Retain approved shared-workspace scene in setup introduction; no additional art in configuration sections. Entire form reviewed. Native setup rendering remains pending an authorized uninstalled instance; current instance is installed. |
| No image | [`moderation/new_abuse_report.tmpl`](../appliance/forgejo/templates/moderation/new_abuse_report.tmpl) | — | A serious reporting form should prioritize category, required remarks, validation and cancel/submit controls. Full template including missing-content disabled gate reviewed; no decorative image. No report submitted; native form capture unperformed. |
| Done | [`org/create.tmpl`](../appliance/forgejo/templates/org/create.tmpl) | `new-org-papercraft.png` | Retain approved collaborative assembly scene; matches creating a shared organization. Full name/visibility/permission form reviewed; desktop/mobile header artwork inspected. No organization created. |
| Partial — no image | [`org/header.tmpl`](../appliance/forgejo/templates/org/header.tmpl) | — | Full identity header reviewed; local home/member/team/project/package/code/settings callers traced. Preserve actual organization avatar, visibility, description and gated actions; no decorative image in this repeated header. Native organization verification remains pending. |
| No image | [`org/home.tmpl`](../appliance/forgejo/templates/org/home.tmpl) | `org/header` | Organization identity, authored README, repositories and real member avatars take priority; source-reviewed. |
| No image | [`org/member/members.tmpl`](../appliance/forgejo/templates/org/member/members.tmpl) | `org/header` | Actual avatars, roles, visibility and owner-only security indicators take priority; source-reviewed. |
| No image | [`org/projects/list.tmpl`](../appliance/forgejo/templates/org/projects/list.tmpl) | Native owner identity and project summaries | Organization/personal variants assessed. Keep real project descriptions, open/closed counts and search primary; personal empty list inspected at capture-bepvu4/001.png. Organization/populated variants source-only. |
| Integrated — verify | [`org/projects/new.tmpl`](../appliance/forgejo/templates/org/projects/new.tmpl) | `new-project-papercraft.png`, creation only | Personal creation verified at desktop/mobile; organization capture and edit-state capture pending. Edit uses no artwork. |
| No image | [`org/projects/view.tmpl`](../appliance/forgejo/templates/org/projects/view.tmpl) | Native project columns and issue cards | Personal/organization board wrappers and full shared board source reviewed. Drag/drop columns, card previews, counts and column dialogs already provide the meaningful visuals; preserve working space. Native board capture unavailable without an existing project. |
| Partial — no image | [`org/settings/layout_head.tmpl`](../appliance/forgejo/templates/org/settings/layout_head.tmpl) | `org/header` | Full wrapper reviewed: organization identity header, native settings navigation and alerts only. No inherited decorative image. Native registry/cleanup and other settings caller decisions are recorded separately; organization rendering remains unverified. |
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
| Partial — no image | [`package/shared/list.tmpl`](../appliance/forgejo/templates/package/shared/list.tmpl) | Personal/organization and native repository registries | Full helper reviewed: type/query filters, publication metadata, repository-access gate and empty/no-match branches. Owner pages supply their own art; repository list needs no second illustration. Native repository caller added below. |
| No image | [`package/shared/versionlist.tmpl`](../appliance/forgejo/templates/package/shared/versionlist.tmpl) | `user/overview/package_versions` | Search, sort, container tag filter, version rows and pagination; no independent decorative header. Source assessment. |
| No image | [`package/view.tmpl`](../appliance/forgejo/templates/package/view.tmpl) | `user/overview/header` | Package/version identity, protocol content, files and metadata take priority; source assessment below. |
| No image | [`post-install.tmpl`](../appliance/forgejo/templates/post-install.tmpl) | — | Native loading animation and login handoff communicate ongoing setup. A static decorative image would compete with that state feedback. Full template reviewed; preserve loading SVG and goto-user-login hook. No installation run or native capture. |
| No image | [`projects/list.tmpl`](../appliance/forgejo/templates/projects/list.tmpl) | Repository, organization and personal callers traced | Full shared partial reviewed; decoration belongs only in the selected personal/organization creation wrapper. No independent partial artwork. |
| No image | [`projects/new.tmpl`](../appliance/forgejo/templates/projects/new.tmpl) | Repository, organization and personal callers traced | Full shared partial reviewed; decoration belongs only in the selected personal/organization creation wrapper. No independent partial artwork. |
| No image | [`projects/view.tmpl`](../appliance/forgejo/templates/projects/view.tmpl) | Repository, organization and personal callers traced | Full shared partial reviewed; decoration belongs only in the selected personal/organization creation wrapper. No independent partial artwork. |
| No image | [`repo/actions/dispatch.tmpl`](../appliance/forgejo/templates/repo/actions/dispatch.tmpl) | Native repository Actions | Workflow branch selection and typed inputs are the task; retain the compact native run form without decoration. Full partial reviewed; called by list_inner. No workflow dispatched. |
| Integrated — verify | [`repo/actions/list.tmpl`](../appliance/forgejo/templates/repo/actions/list.tmpl) | Native repository Actions | Populated workflow/status inventory needs no decorative header; no-workflows branch selects distinct workflow art. Full wrapper reviewed including native polling. Native Actions route unavailable; verification pending. |
| Integrated — verify | [`repo/actions/list_inner.tmpl`](../appliance/forgejo/templates/repo/actions/list_inner.tmpl) | Native repository Actions | Traced list caller, workflow/actor/status filters, enable/disable gate and dispatch/runs/empty branches. Art confined to no_workflows; native verification pending. |
| Integrated — verify | [`repo/actions/no_workflows.tmpl`](../appliance/forgejo/templates/repo/actions/no_workflows.tmpl) | Native repository Actions | Distinct workflow-tile illustration supports first workflow setup. Native writer/non-writer guidance preserved. Exact prompt: workflows-prompt.md. Focused parity tests passed; native route returned 404, so rendering remains pending. |
| No image | [`repo/actions/runs_list.tmpl`](../appliance/forgejo/templates/repo/actions/runs_list.tmpl) | Native repository Actions | Run status, commit/actor/ref and timing are meaningful visuals. Keep filtered/no-run messages compact without added artwork. Full partial reviewed and list_inner caller traced; native run capture unavailable. |
| No image | [`repo/actions/view.tmpl`](../appliance/forgejo/templates/repo/actions/view.tmpl) | Native repository Actions | Native JS run/log/artifact viewer needs available space for execution evidence and controls. Full template data/locale contract reviewed; no decorative art added. Native run capture unavailable. |
| No image | [`repo/activity.tmpl`](../appliance/forgejo/templates/repo/activity.tmpl) | Native pulse and analytics | All four callers assessed: pulse summary/activity lists, contributor chart, code-frequency chart, recent-commit chart. Native data visuals are primary; loading/failed/empty feedback must remain unambiguous. Source and desktop captures below. |
| No image | [`repo/branch/list.tmpl`](../appliance/forgejo/templates/repo/branch/list.tmpl) | Native branch state and divergence | Default/protected/deleted branch states, commit status, ahead/behind bars and PR state labels already encode meaning. Create/rename/delete dialogs are focused operations. Full source reviewed; read-only default branch captured. |
| Partial — no image | [`repo/branch_dropdown.tmpl`](../appliance/forgejo/templates/repo/branch_dropdown.tmpl) | Home/history/commit header, Actions dispatch, search and releases | Full template reviewed and six local call sites traced. Native branch/tag/commit icons and selected ref identify the control; no decorative raster image. Preserve Vue data contract, creation gate and caller-specific URLs/form options. |
| Partial — no image | [`repo/clone_buttons.tmpl`](../appliance/forgejo/templates/repo/clone_buttons.tmpl) | — | Full control reviewed; local repository home/empty and wiki view/revision callers traced. Protocol buttons, readonly URL and copy icon are functional content; no decorative illustration. Preserve native protocol visibility and clipboard hooks. |
| No image | [`repo/commit_header.tmpl`](../appliance/forgejo/templates/repo/commit_header.tmpl) | Commit detail and single-commit diff callers | Full source now reviewed across bounded reads. Real author/committer/signature, ancestry, Git notes and focused branch/tag/cherry-pick dialogs are meaningful content; no decorative image. |
| No image | [`repo/commit_page.tmpl`](../appliance/forgejo/templates/repo/commit_page.tmpl) | Native commit metadata and diff | Commit identity/status and actual changed lines are the relevant visuals. Wrapper reviewed and real commit captured; shared header/diff internals remain separately tracked. |
| No image | [`repo/commits.tmpl`](../appliance/forgejo/templates/repo/commits.tmpl) | Native commit table | History is author/message/SHA/date and reference navigation with search, not an introductory page. Wrapper and commits_table reviewed; native two-commit history captured. Comparison and PR commit callers are now assessed. |
| Partial — no image | [`repo/commits_list.tmpl`](../appliance/forgejo/templates/repo/commits_list.tmpl) | Commit table and wiki revision | Full helper reviewed; commit history, pull commits and compare callers traced through commits_table, plus direct wiki revision caller. Avatars, signatures, statuses, tags and hashes are meaningful visuals; no decorative image. |
| No image | [`repo/commits_table.tmpl`](../appliance/forgejo/templates/repo/commits_table.tmpl) | History, comparison and PR commits callers | All override callers traced and assessed. Search/counts/refs/renamed-file feedback and commit rows need no independent artwork. |
| Done | [`repo/create.tmpl`](../appliance/forgejo/templates/repo/create.tmpl) | `new-repo-papercraft.png` | Retain approved robot filing a project sheet; matches repository creation. Wrapper and creation gates reviewed; desktop/mobile header artwork inspected. Delegated form bodies not re-audited here; no form submitted. |
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
| Partial — no image | [`repo/header.tmpl`](../appliance/forgejo/templates/repo/header.tmpl) | Shared repository identity/navigation | Full header reviewed and local caller families traced across repository pages, settings and contextual errors. Native identity/icon, visibility/archive/template badges, fork/mirror provenance, gated actions and tabs are the useful visuals; no repeated decorative image. |
| No image | [`repo/home.tmpl`](../appliance/forgejo/templates/repo/home.tmpl) | Repository-owned content | Full wrapper reviewed across home/directory/file/blame, branch/tag/commit context, topics, flags/archive messages, template use and clone controls. No decorative hero competing with the repository description, file tree or README. Native home/file/blame captures below. |
| No image | [`repo/issue/choose.tmpl`](../appliance/forgejo/templates/repo/issue/choose.tmpl) | Repository-authored template choices | Full chooser reviewed: template names/descriptions, external contact links, optional blank issue and invalid-config warning. No decorative scene competing with the repository-specific choices. Source-based; chooser state not captured. |
| No image | [`repo/issue/labels.tmpl`](../appliance/forgejo/templates/repo/issue/labels.tmpl) | `repo/header` | Native label colors and descriptions are the meaningful visuals; preserve the compact management list. Empty read-only list inspected at capture-TADFh7/002.png (1440px); populated and writer controls assessed from source, not rendered. |
| No image | [`repo/issue/list.tmpl`](../appliance/forgejo/templates/repo/issue/list.tmpl) | Issue and pull-request lists | Preserve native status icons, labels, review metadata, pinned cards and filters. Shared empty result also represents filtered-out items, not a successful completion. Full override and upstream 15.0.7 shared/issuelist reviewed. Native populated issues and empty pulls inspected at capture-h1PW7Q/001.png and 002.png (1440px); populated PR review states source-only. Separate observed defect: create-button text is invisible in both captures; needs CSS investigation. |
| No image | [`repo/issue/milestone_issues.tmpl`](../appliance/forgejo/templates/repo/issue/milestone_issues.tmpl) | `repo/header` | Milestone detail uses actual progress, deadline, issue counts and filtered issues; decorative art would compete with those signals. Full override reviewed; native detail capture remains unperformed. |
| No image | [`repo/issue/milestone_new.tmpl`](../appliance/forgejo/templates/repo/issue/milestone_new.tmpl) | `repo/header` | Focused create/edit title, deadline and Markdown-description form under repository navigation; no extra artwork. Full override reviewed; writer form not rendered with this read-only fixture. |
| No image | [`repo/issue/milestones.tmpl`](../appliance/forgejo/templates/repo/issue/milestones.tmpl) | `repo/header` | Actual progress bars, overdue dates and open/closed counts carry the visual hierarchy. Native populated list inspected at capture-TADFh7/001.png (1440px); full override reviewed, writer actions source-only. |
| Partial — no image | [`repo/issue/navbar.tmpl`](../appliance/forgejo/templates/repo/issue/navbar.tmpl) | — | Full helper reviewed; local issue list/chooser, labels, milestone list/new and project-board callers traced. Labels/milestones navigation needs no independently generated image. |
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
| Partial — no image | [`repo/pulls/status.tmpl`](../appliance/forgejo/templates/repo/pulls/status.tmpl) | — | Full helper reviewed; native pull view_content/pull caller located. Check states, missing required checks and target links are meaningful visuals; no decorative image. Native status rendering unverified. |
| Partial — no image | [`repo/pulls/tab_menu.tmpl`](../appliance/forgejo/templates/repo/pulls/tab_menu.tmpl) | — | Full helper reviewed; conversation, commits and files callers traced. Navigation counts and addition/deletion bar already convey actual content; no decorative image. Native populated pull rendering remains unverified. |
| Partial — no image | [`repo/pulls/trust.tmpl`](../appliance/forgejo/templates/repo/pulls/trust.tmpl) | — | Full helper reviewed; native pull view_content/pull callers located. Permission-gated deny/once/always/revoke decisions and warning must stay prominent; no decorative image. No trust action executed. |
| No image | [`repo/release/list.tmpl`](../appliance/forgejo/templates/repo/release/list.tmpl) | Real release state, notes and downloads | Keep draft/prerelease/stable labels, verification, publisher, notes and assets primary. Full source reviewed; existing prerelease list captured at capture-id7a6e/001.png. |
| No image | [`repo/release/new.tmpl`](../appliance/forgejo/templates/repo/release/new.tmpl) | Native release editor | Create/tag-only/draft/edit/publish variants need clear tag target, notes, attachments and prerelease choices. Decorative shipment/success imagery would suggest a state not yet established. Full source reviewed; no release submitted. |
| No image | [`repo/release_tag_header.tmpl`](../appliance/forgejo/templates/repo/release_tag_header.tmpl) | Release/tag list toolbar | Both callers assessed; count tabs, conditional search/RSS/create and code-only submenu need no independent image. Full partial reviewed. |
| No image | [`repo/search.tmpl`](../appliance/forgejo/templates/repo/search.tmpl) | `repo/header` | Native branch/query/mode controls and code snippets are the useful content. Keep empty/no-result feedback compact without decorative art. Full wrapper plus shared search/results reviewed; desktop no-results capture inspected (capture-tQQ0un/001.png). |
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
| Partial — no image | [`repo/settings/webhook/base_list.tmpl`](../appliance/forgejo/templates/repo/settings/webhook/base_list.tmpl) | Native webhook list wrapper | Full helper, upstream list wrapper and provider menu reviewed; local personal/repository callers traced. Last-delivery status, destination URL and provider icons are the relevant visuals. No extra decorative image; personal page intro retains its own art. |
| Partial — no image | [`repo/settings/webhook/history.tmpl`](../appliance/forgejo/templates/repo/settings/webhook/history.tmpl) | — | Full helper reviewed; webhook/new caller traced. Delivery state, UUID, request/response tabs and permission-gated test/replay controls need no decorative image. No delivery or replay executed; native history unverified. |
| No image | [`repo/settings/webhook/new.tmpl`](../appliance/forgejo/templates/repo/settings/webhook/new.tmpl) | `repo/settings/layout_head` | Create/edit webhook configuration uses the selected provider icon and native provider-specific form, followed by delivery history. No extra illustration is needed above those controls. Leaf and shared dispatcher reviewed; individual provider bodies not re-reviewed in this pass. Source-only assessment; no native repository settings capture with the current fixture. |
| Partial — no image | [`repo/sub_menu.tmpl`](../appliance/forgejo/templates/repo/sub_menu.tmpl) | Repository home/history/branches and release-tag header | Full helper reviewed and local callers traced. Real commit/branch/tag counts, size and language percentages are meaningful visuals; no decorative image. Preserve code-read, empty-repository, hidden-info and blame gates. |
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
| Partial — no image | [`shared/actions/runner_create.tmpl`](../appliance/forgejo/templates/shared/actions/runner_create.tmpl) | — | Full helper reviewed. Name/description form and create/cancel controls should remain focused. Personal/organization/repository/admin page assessments remain recorded separately; native verification gaps remain open. |
| Partial — no image | [`shared/actions/runner_details.tmpl`](../appliance/forgejo/templates/shared/actions/runner_details.tmpl) | — | Full helper reviewed. Actual status, labels, ownership and task history are the relevant visuals. Personal/organization/repository/admin page assessments remain recorded separately; native verification gaps remain open. |
| Partial — no image | [`shared/actions/runner_edit.tmpl`](../appliance/forgejo/templates/shared/actions/runner_edit.tmpl) | — | Full helper reviewed. Property editing and token regeneration choice should remain focused. Personal/organization/repository/admin page assessments remain recorded separately; native verification gaps remain open. |
| Partial — no image | [`shared/actions/runner_list.tmpl`](../appliance/forgejo/templates/shared/actions/runner_list.tmpl) | — | Full helper reviewed. Actual status indicators, UUIDs, owner/labels and gated management controls are the relevant visuals. Personal/organization/repository/admin page assessments remain recorded separately; native verification gaps remain open. |
| Partial — no image | [`shared/actions/runner_setup.tmpl`](../appliance/forgejo/templates/shared/actions/runner_setup.tmpl) | — | Full helper reviewed. UUID/token and setup snippets are the relevant content; no decoration or generated success imagery. Personal/organization/repository/admin page assessments remain recorded separately; native verification gaps remain open. |
| Partial — no image | [`shared/blocked_users_list.tmpl`](../appliance/forgejo/templates/shared/blocked_users_list.tmpl) | — | Full helper and native personal/organization settings callers reviewed. Real avatars, names, blocked dates and unblock actions are the relevant visuals; empty state stays compact. No decorative image. |
| Partial — no image | [`shared/quota_overview.tmpl`](../appliance/forgejo/templates/shared/quota_overview.tmpl) | — | Full helper and native personal/organization storage_overview callers reviewed. Actual rule status, usage/limit totals and subject breakdown bars are the meaningful visuals; no decorative image. |
| Partial — no image | [`shared/secrets/add_list.tmpl`](../appliance/forgejo/templates/shared/secrets/add_list.tmpl) | — | Full list and add/edit dialogs reviewed; local repository secrets/Actions callers traced, other owner assessments recorded separately. Masked values, names, dates and focused input help need no decoration. No secrets changed. |
| Partial — no image | [`shared/variables/variable_list.tmpl`](../appliance/forgejo/templates/shared/variables/variable_list.tmpl) | — | Full list and create/edit dialog reviewed; local repository Actions caller traced, other owner assessments recorded separately. Exact name/value data and validation help take precedence; no decorative image. No variables changed. |
| Done | [`status/404.tmpl`](../appliance/forgejo/templates/status/404.tmpl) | `not-found-papercraft.png` | Small neutral wayfinding scene; native general/repository 404 captures inspected at desktop/mobile widths. |
| No image | [`status/413.tmpl`](../appliance/forgejo/templates/status/413.tmpl) | — | Payload-too-large diagnostic: keep the brief error code/explanation immediately visible after a failed request. Additional artwork would lengthen this corrective interruption; source reviewed, no oversized request submitted. |
| No image | [`user/auth/activate.tmpl`](../appliance/forgejo/templates/user/auth/activate.tmpl) | Native account recovery/activation | State-dependent password confirmation, invalid-code/password feedback, manual approval, resend limits and unconfirmed-email controls need exact native messages. No common decorative scene fits those outcomes; keep the compact form/message layout. Full override reviewed; state-specific native captures unperformed. No emails requested or account state changed. |
| No image | [`user/auth/change_passwd.tmpl`](../appliance/forgejo/templates/user/auth/change_passwd.tmpl) | Native credential form | Password and confirmation fields are a focused credential update, also embedded during account linking. No decorative art; wrapper and complete upstream change_passwd_inner reviewed. Source assessment only; no reset codes requested or passwords changed, native reset-session capture unperformed. |
| No image | [`user/auth/forgot_passwd.tmpl`](../appliance/forgejo/templates/user/auth/forgot_passwd.tmpl) | Native account recovery/activation | A short email-entry form transitions to sent, disabled-service or resend-limited feedback. Keep recovery focused on the actual instruction; no decorative mail-delivery scene that could imply delivery before it occurs. Full override reviewed; state-specific native captures unperformed. No emails requested or account state changed. |
| No image | [`user/auth/grant.tmpl`](../appliance/forgejo/templates/user/auth/grant.tmpl) | — | Actual application name/creator, scopes, redirect destination and authorize/cancel controls must drive consent. Decorative Soda imagery could distract from that identity; no added art. Full override reviewed; native consent/error-session capture unperformed. No grants or account state changed. |
| No image | [`user/auth/grant_error.tmpl`](../appliance/forgejo/templates/user/auth/grant_error.tmpl) | `repo/header` | Actual authorization error and explanation carry the outcome, with optional repository context. Keep this concise diagnostic free of decorative art. Full override reviewed; native consent/error-session capture unperformed. No grants or account state changed. |
| No image | [`user/auth/link_account.tmpl`](../appliance/forgejo/templates/user/auth/link_account.tmpl) | Native federated authentication | Account-linking tabs distinguish creating an account from signing into an existing one. Keep that choice and native delegated forms prominent; no additional wrapper art. Inner signup/signin bodies remain part of the separate authentication review. Full leaf reviewed; provider-session native capture unperformed. No account links or registrations changed. |
| No image | [`user/auth/prohibit_login.tmpl`](../appliance/forgejo/templates/user/auth/prohibit_login.tmpl) | — | The account-access restriction and its explanation are the entire message. Avoid a playful illustration that would soften or misrepresent that state. Full override reviewed; native consent/error-session capture unperformed. No grants or account state changed. |
| No image | [`user/auth/reset_passwd.tmpl`](../appliance/forgejo/templates/user/auth/reset_passwd.tmpl) | Native credential form | New password, optional 2FA/recovery code, remember-me and invalid-link feedback must remain clear. No decorative art; full override reviewed across valid/invalid reset branches. Source assessment only; no reset codes requested or passwords changed, native reset-session capture unperformed. |
| Done | [`user/auth/signin.tmpl`](../appliance/forgejo/templates/user/auth/signin.tmpl) | `login-papercraft.png` | Retain approved welcome/workstation scene. Native standalone desktop artwork and intentionally artwork-free mobile form inspected; full wrapper and stock signin_inner reviewed. Provider/account-linking states remain unobserved separately. |
| No image | [`user/auth/signin_openid.tmpl`](../appliance/forgejo/templates/user/auth/signin_openid.tmpl) | Native federated authentication | The existing site logo and OpenID mark identify the sign-in method; URI, remember-me and return controls need no additional generated illustration. Full leaf reviewed; provider-session native capture unperformed. No account links or registrations changed. |
| Integrated — verify | [`user/auth/signup.tmpl`](../appliance/forgejo/templates/user/auth/signup.tmpl) | `signup-papercraft.png` | Distinct welcome-folder scene for enabled standalone registration only; disabled registration and account linking exclude it. Full native signup body reviewed; focused parity/presentation tests passed. Desktop/mobile disabled-state captures inspected; enabled native rendering remains pending. |
| No image | [`user/auth/signup_openid_connect.tmpl`](../appliance/forgejo/templates/user/auth/signup_openid_connect.tmpl) | Native federated authentication | Existing-account credentials and the read-only OpenID URI define the linking task; an illustration adds no useful identity information. Full leaf reviewed; provider-session native capture unperformed. No account links or registrations changed. |
| No image | [`user/auth/signup_openid_register.tmpl`](../appliance/forgejo/templates/user/auth/signup_openid_register.tmpl) | Native federated authentication | The provider URI, username/email and CAPTCHA define this provider-linked registration step. Keep identity confirmation focused, with no additional artwork. Full leaf reviewed; provider-session native capture unperformed. No account links or registrations changed. |
| No image | [`user/auth/twofa.tmpl`](../appliance/forgejo/templates/user/auth/twofa.tmpl) | Native authentication challenge | Single one-time-code entry with verification and fallback link: keep the time-sensitive challenge compact, with no decorative art. Full override reviewed; challenge-session native capture unperformed. No authentication state changed. |
| No image | [`user/auth/twofa_scratch.tmpl`](../appliance/forgejo/templates/user/auth/twofa_scratch.tmpl) | Native authentication challenge | Single recovery-code entry with verification and return link: keep recovery focused, with no decorative art. Full override reviewed; challenge-session native capture unperformed. No authentication state changed. |
| No image | [`user/auth/webauthn.tmpl`](../appliance/forgejo/templates/user/auth/webauthn.tmpl) | Native authentication challenge | Native key icon, loading indicator and device instructions communicate the interaction. A generated device scene could misrepresent the actual authenticator; no extra art. Full override reviewed; challenge-session native capture unperformed. No authentication state changed. |
| No image | [`user/code.tmpl`](../appliance/forgejo/templates/user/code.tmpl) | `org/header`, `user/overview/header` | Contributor avatar or organization identity already establishes scope; query controls and code snippets need the remaining space. Both owner branches and shared search/results reviewed without extra art. Attempted owner-code capture displayed repository profile instead; native code-search variant unverified. |
| Done | [`user/dashboard/dashboard.tmpl`](../appliance/forgejo/templates/user/dashboard/dashboard.tmpl) | `dashboard-papercraft.png` | Retain approved workbench illustration in repository sidebar. Full wrapper and stock empty-feed guide reviewed; personal empty dashboard artwork inspected at desktop/mobile. Organization context and populated feed remain unobserved in this check. |
| Done | [`user/dashboard/issues.tmpl`](../appliance/forgejo/templates/user/dashboard/issues.tmpl) | `issues-papercraft.png`, `pulls-papercraft.png` | Retain distinct approved issue-checklist and collaborative-review scenes. Both branches fully reviewed; personal empty issue and pull dashboards inspected at desktop/mobile. Organization/populated variants remain unobserved. |
| Done | [`user/dashboard/milestones.tmpl`](../appliance/forgejo/templates/user/dashboard/milestones.tmpl) | `milestones-papercraft.png` | Retain approved stepping-stone/flag scene. Full filter, repository selector and milestone-content template reviewed; personal empty page artwork inspected at desktop/mobile. Populated/organization variants remain unobserved. |
| Done | [`user/notification/notification_div.tmpl`](../appliance/forgejo/templates/user/notification/notification_div.tmpl) | `notifications-papercraft.png` | Retain approved inbox-sorting scene. Full template reviewed, including preview dispatch and native status controls. Empty unread page artwork inspected at desktop/mobile; populated/read/pinned and preview rendering remain unverified here. |
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
| Partial — no image | [`user/settings/layout_footer.tmpl`](../appliance/forgejo/templates/user/settings/layout_footer.tmpl) | — | Closes the shared settings layout and delegates base/footer. Full helper reviewed; no independently generated artwork. Header/caller verification tracked separately. |
| Integrated — verify | [`user/settings/layout_head.tmpl`](../appliance/forgejo/templates/user/settings/layout_head.tmpl) | `settings-account-papercraft.png`, `settings-appearance-papercraft.png`, `settings-applications-papercraft.png`, `settings-keys-papercraft.png`, `settings-profile-papercraft.png`, `settings-security-papercraft.png` | Full layout reviewed; 12 local callers traced. Six category illustrations plus explicit package/webhook/organization landing art; enrollment and OAuth/token editing suppress art. Landing-page evidence recorded below; sensitive detail/native upstream caller verification remains incomplete. |
| Done | [`user/settings/organization.tmpl`](../appliance/forgejo/templates/user/settings/organization.tmpl) | `settings-organizations-papercraft.png` | Distinct membership-card scene; desktop/mobile empty-state captures checked. |
| Done | [`user/settings/packages.tmpl`](../appliance/forgejo/templates/user/settings/packages.tmpl) | `settings-packages-papercraft.png` | Native landing-page content preserved; explicit artwork input excludes cleanup pages. |
| Done | [`user/settings/profile.tmpl`](../appliance/forgejo/templates/user/settings/profile.tmpl) | `settings-profile-papercraft.png` via layout | Existing portrait-frame artwork retained; native desktop/mobile header captures inspected. |
| Done | [`user/settings/security/security.tmpl`](../appliance/forgejo/templates/user/settings/security/security.tmpl) | `settings-security-papercraft.png` | Existing security scene retained; native desktop/mobile landing captures inspected. |
| No image | [`user/settings/security/twofa_enroll.tmpl`](../appliance/forgejo/templates/user/settings/security/twofa_enroll.tmpl) | `user/settings/layout_head` | Explicitly suppress decoration for enrollment/re-enrollment; QR and passcode are the task. Native capture unperformed. |
| Partial — no image | [`webhook/new.tmpl`](../appliance/forgejo/templates/webhook/new.tmpl) | — | Full provider dispatcher reviewed; repository settings caller traced. Selected provider icon identifies the integration; native provider form and delivery history need no added illustration. Individual upstream provider bodies not re-audited here. |
| Partial — no image | [`webhook/shared-settings.tmpl`](../appliance/forgejo/templates/webhook/shared-settings.tmpl) | Native provider forms | Full helper reviewed: event groups, branch filter, provider-specific authorization-header gate, active state and create/update/delete controls. Native Forgejo provider caller inspected. No decorative art among configuration fields; other provider bodies remain separately scoped. |

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
| Integrated — verify | `admin/applications/list.tmpl` | Shared list/create reviewed; no decorative art, inherited image suppressed. Native admin capture pending. |
| Integrated — verify | `admin/applications/oauth2_edit.tmpl` | Credential editor assessed; inherited image suppressed. Native admin capture pending. |

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

## Additional administrator layout callers

Read-only enumeration of Forgejo 15.0.7 embedded admin templates found these 15 stock leaf callers of the overridden `admin/layout_head`. They currently inherit its fallback dashboard artwork. Caller discovery is not a page-content assessment or native rendering check. No new local leaf overrides were added.

| Status | Upstream template | Remaining work |
| --- | --- | --- |
| Integrated — verify | `admin/actions.tmpl` | Full stock wrapper reviewed; delegates to previously assessed runner/variable forms and status tables. No decorative art; opt-in shared layout has no fallback. Native admin capture pending. |
| Integrated — verify | `admin/config_settings.tmpl` | Avatar-service toggles and editor-app configuration are exact settings, not an illustration gallery. Full stock wrapper reviewed. No decorative artwork; opt-in admin layout removes fallback. Native capture pending. |
| Integrated — verify | `admin/hook_new.tmpl` | Provider forms and system/default scope headings should remain focused; shared provider dispatcher previously reviewed. Full stock wrapper reviewed. No decorative artwork; opt-in admin layout removes fallback. Native capture pending. |
| Integrated — verify | `admin/hooks.tmpl` | Separate system/default endpoint lists use delivery status and exact scope. Full stock wrapper reviewed. No decorative artwork; opt-in admin layout removes fallback. Native capture pending. |
| Integrated — verify | `admin/moderation/report_details.tmpl` | Full stock page reviewed: content reference, reporter/date/category, remarks and retained shadow-copy details form an evidence review. No decorative illustration is appropriate. The opt-in admin layout removes fallback artwork for this caller. No native admin capture. |
| Integrated — verify | `admin/moderation/reports.tmpl` | Full stock page reviewed: real content-type icons, report counts/categories/remarks and handled/ignored/content actions are the useful information. No decorative image is appropriate, including the no-open-reports state. The opt-in admin layout removes fallback artwork for this caller. No native admin capture. |
| Integrated — verify | `admin/org/list.tmpl` | Real organization names, visibility and team/member/repository counts carry the context. Full stock wrapper reviewed. No decorative artwork; opt-in admin layout removes fallback. Native capture pending. |
| Integrated — verify | `admin/packages/list.tmpl` | Version/owner/size records, cleanup and delete controls need a compact operational table. Full stock wrapper reviewed. No decorative artwork; opt-in admin layout removes fallback. Native capture pending. |
| Integrated — verify | `admin/repo/unadopted.tmpl` | Actual directory search results and adopt/delete confirmations must remain prominent. Full stock wrapper reviewed. No decorative artwork; opt-in admin layout removes fallback. Native capture pending. |
| Integrated — verify | `admin/runners/create.tmpl` | Full stock wrapper reviewed; delegates to previously assessed runner/variable forms and status tables. No decorative art; opt-in shared layout has no fallback. Native admin capture pending. |
| Integrated — verify | `admin/runners/details.tmpl` | Full stock wrapper reviewed; delegates to previously assessed runner/variable forms and status tables. No decorative art; opt-in shared layout has no fallback. Native admin capture pending. |
| Integrated — verify | `admin/runners/edit.tmpl` | Full stock wrapper reviewed; delegates to previously assessed runner/variable forms and status tables. No decorative art; opt-in shared layout has no fallback. Native admin capture pending. |
| Integrated — verify | `admin/runners/setup.tmpl` | Full stock wrapper reviewed; delegates to previously assessed runner/variable forms and status tables. No decorative art; opt-in shared layout has no fallback. Native admin capture pending. |
| Integrated — verify | `admin/stats.tmpl` | Actual named statistics need no decorative image. Full stock wrapper reviewed. No decorative artwork; opt-in admin layout removes fallback. Native capture pending. |
| Integrated — verify | `admin/user/view.tmpl` | Real avatar, account status, emails, repositories and organizations provide the visuals; detail/email partials also reviewed. Full stock wrapper reviewed. No decorative artwork; opt-in admin layout removes fallback. Native capture pending. |

Admin layout consolidation: all 18 local leaf callers and 15 additional stock admin callers are assessed. Artwork is now opt-in through `.artwork`; only new-account creation selects an illustration. Earlier explicit suppression inputs were removed as redundant. This supersedes references above to pending fallback suppression or hideArtwork inputs; native admin verification remains pending.

### Public registration artwork

- Generated `signup-papercraft.png`: the established cream cardstock robot opens a mint welcome folder with a blank page and small doorway symbol. Exact prompt and original output: [signup-prompt.md](../assets/branding/forgejo/signup-prompt.md). Selected unchanged, 1536×1024 RGBA with transparency.
- The standalone signup wrapper selects the image only when registration is enabled and account-linking mode is absent. Native fields, CAPTCHA, OAuth choices and disabled-registration explanation remain upstream-owned.
- `go test ./scripts -run 'TestForgejoSecondaryAuth|TestForgejoPagesComposeSharedPresentationWithNativeBoundaries' -count=1` passed. Templates reloaded successfully.
- Native guest captures inspected: `.artifacts/screenshots/capture-sQUJUE/001.png` (1440×1000) and `.artifacts/screenshots/capture-ZgJTAd/001.png` (390×844). Both display the disabled-registration explanation without artwork. Enabled rendering remains pending; registration was not enabled and no account was created.
- Next: finish repository Actions list/empty-state/dispatch caller assessment, then remaining pending pages. The signup item stays Integrated — verify.

### Repository Actions artwork

- Reviewed all six local Actions templates and their caller chain. Workflow selection/dispatch, populated or filtered run lists and the native JS detail mount remain free of decorative art. Only the no-workflows branch gains a distinct robot assembling workflow tiles.
- Built-in generation retained unchanged as `assets/branding/forgejo/workflows-papercraft.png`, 1536×1024 RGBA, alpha range 0–254; visually inspected against the dashboard identity/material reference. Exact prompt/original path: [workflows-prompt.md](../assets/branding/forgejo/workflows-prompt.md).
- Focused native-body parity and shared presentation tests passed; templates reloaded. Native capture `.artifacts/screenshots/capture-rxLOx6/001.png` shows 404 at `/bob/activity-field-notes/actions`, not the empty state. Rendering verification remains pending an accessible Actions page. No Actions configuration or workflow execution was changed.
- Next: repository search, user code search and remaining unchecked pages/shared callers.

### Search, reporting and setup transition assessment

- Repository and both owner-scoped code-search wrappers were reviewed together with stock Forgejo 15.0.7 `shared/search/code/search` and `results`. Branch/query/mode controls, index-unavailable feedback, path breadcrumbs, language filters and file results should remain the focus. No additional art selected.
- Native desktop capture `.artifacts/screenshots/capture-tQQ0un/001.png` confirms repository no-results rendering. The second capture, requested at `/bob/-/code?q=README`, displays the repository profile, so it does not verify owner code search. Organization code search and populated results remain unobserved.
- Abuse-report creation is a focused category/remarks form; post-install is a loading/handoff state with its own native animation. Full templates reviewed; neither receives decorative art. No report or installation was executed.
- Documentation-only assessment; no source behavior changes or tests required. Next: verify existing approved illustrations and trace remaining shared callers.

### Approved login illustration verification

- Retain the existing `login-papercraft.png` welcome/workstation scene. It is approved historical artwork with a different robot face; preserve it rather than regenerate it to match newer assets. New assets continue to use the dashboard identity reference.
- Full wrapper and Forgejo 15.0.7 `signin_inner` reviewed: internal-sign-in, password, remember-me, CAPTCHA, OAuth delegate, registration and recovery branches remain native. Delegated provider bodies were not re-audited here.
- Native guest captures inspected: `.artifacts/screenshots/capture-l1irJf/001.png` (1440×1000) displays the complete scene beside the form; `.artifacts/screenshots/capture-qkriM7/001.png` (390×844) displays the complete form with artwork intentionally hidden by `login.css`. No clipped fields or image detected in these captures.
- This verifies standalone login presentation only; account-linking/provider/error branches remain unobserved. No login submitted or credentials entered. Documentation-only change, no tests rerun.

### Approved Explore illustration verification

- Retain the existing repository/code-folder, people and organization-workspace illustrations. Reviewed all three Explore wrappers, people/organization branches, native navbar delegation and custom empty-state branches. Empty states need no second generated image beneath the existing page artwork.
- Native desktop captures inspected: `.artifacts/screenshots/capture-0c67pa/001.png` repositories, `002.png` people and `003.png` organizations. Matching mobile captures: `.artifacts/screenshots/capture-nhV5sL/001.png` through `003.png` at 390×844. Each header image fits without clipping; directory/empty-state content stays separate.
- Desktop `capture-0c67pa/004.png`, requested at `/explore/code`, displays the repository directory. It is not code-search verification; that row remains Integrated — verify.
- Separate observed defect: organization empty-state primary button text is invisible on the desktop capture, consistent with the previously recorded issue/PR button-text defect. This artwork check does not claim that button styling is correct or fixed.
- No source changes, actions submitted or new assets needed; documentation-only verification. Remaining queue: home/setup, creation pages, dashboard/notifications and shared callers.

### Approved creation-page illustration verification

- Preserve both distinct approved creation images. Repository creation uses the project-sheet/folder scene; organization creation uses the collaborative assembly scene. They fit their page purposes and do not require replacements.
- Native desktop captures `.artifacts/screenshots/capture-syiyXr/001.png` (repository) and `002.png` (organization), plus mobile `.artifacts/screenshots/capture-zuLx0z/001.png` and `002.png` at 390×844, were inspected. Header artwork fits beside titles without overlapping the visible form fields. Lower form sections extend beyond these viewport captures; this is not a complete form-interaction test.
- Full organization template reviewed, including visibility and repository-admin team-access input. Repository wrapper reviewed, including creation-limit branches and native helper/basic/template/init/advanced delegates; delegate bodies were not re-audited in this verification step.
- No forms submitted or resources created. Documentation-only evidence update; no tests rerun. Next: home/setup and dashboard/notification artwork.

### Approved public-home artwork verification

- Retain the distinct shared-workbench illustration. Full home template reviewed: native sign-in/explore destinations, registration-availability message branch, theme control and feature content. No new image needed.
- Native guest desktop `.artifacts/screenshots/capture-G7cFAz/001.png` (1440×1000) shows the full scene beside welcome text. Mobile `.artifacts/screenshots/capture-wwIDbQ/001.png` (390×844) places art below the account/navigation content; the viewport cuts through the scene naturally. Additional `.artifacts/screenshots/capture-gA6bYa/001.png` (390×1200) confirms the entire mobile illustration fits without horizontal clipping or overlap.
- Local registration-disabled guidance is visible; enabled-registration branch was not exercised. No settings changed or forms submitted. Documentation-only evidence update; no tests rerun. Next: setup and dashboard/notification artwork.

### Approved dashboard artwork verification

- Retain the workbench scene in the dashboard sidebar; no second illustration is needed inside the activity guide. Reviewed the full wrapper and stock Forgejo 15.0.7 empty-feed guide. Heatmap, feed, context selector and repository inventory remain native delegates.
- Native personal empty-dashboard captures inspected: `.artifacts/screenshots/capture-LIvjYo/001.png` (1440×1000) and `.artifacts/screenshots/capture-VBZh2Y/001.png` (390×1600). The complete illustration fits beside desktop activity and below mobile activity, above repository selection, without clipping or overlap.
- Organization context, populated feed and heatmap states were not exercised by these captures. No resources or preferences changed; documentation-only update, no tests rerun. Next: dashboard issue/pull/milestone views, notifications and setup.

### Approved issue, pull and milestone dashboard artwork

- Preserve the distinct issue-checklist, collaborative code-review and milestone-path scenes. Their header placement identifies each overview without replacing native counts, status or progress. No second empty-state image needed.
- Full issue/pull wrapper reviewed, including branch-specific review filters, organization-label gate, search, sort and shared issue-list delegation. Full milestone template reviewed, including repository selection, progress, dates, tracked time and rendered content.
- Native desktop `.artifacts/screenshots/capture-11DuBL/001.png` through `003.png` and matching mobile `.artifacts/screenshots/capture-K1SESK/001.png` through `003.png` (390×844) inspected in issue/pull/milestone order. All three images fit their headers without overlap; personal empty-state controls remain visible. The milestone empty card extends below the mobile viewport.
- These captures do not verify populated lists or organization contexts. No records or settings changed; documentation-only update, no tests rerun. Next: notifications, setup and remaining shared callers.

### Approved notifications artwork verification

- Retain the inbox-sorting scene in the page introduction; no second decorative empty-state image needed. Full notification template reviewed: unread/read navigation, conditional purge control, sequence marker, row links/status/time, pin/read/unread forms and preview dispatch remain intact.
- Native empty-unread captures inspected: `.artifacts/screenshots/capture-MXKbXK/001.png` (1440×1000) and `.artifacts/screenshots/capture-pk1mQ3/001.png` (390×844). Illustration fits the header without overlap and native empty-state text stays prominent.
- No notification status changed. Populated/read/pinned and preview rendering are not proven by these captures. Documentation-only verification; no tests rerun. Next: setup and remaining shared callers.

### Setup illustration source assessment

- Read the complete setup override: database variants and reinstall confirmations; paths, listeners and advertised URL; registration; email; optional service/privacy/authentication controls; initial administrator fields; environment-config notice; submit and preloaded native loading image.
- Retain the already approved shared-workspace artwork in the introduction. It illustrates the workspace being configured. No additional decorative images belong inside database, credential, networking or policy sections; their labels/help/error states are the relevant information.
- This is source assessment only. Home-page verification of the same asset does not establish setup-layout rendering. Native setup verification remains pending an authorized uninstalled instance. The existing instance was not unlocked, reset or reinstalled; no setup form was submitted.
- Documentation-only change; no tests rerun. Next: remaining shared callers and outstanding native artwork verification.

### Presentation helper ownership

- Traced local callers of page_intro, empty_content, Explore empty/navbar and guest theme/toggle helpers. Full shared helper bodies inspected; Explore bodies had already been inspected with their directory pages.
- These helpers need no separately generated illustration. Page intro renders an explicitly selected optional image; empty content renders caller-selected icon/text; navigation/theme controls retain meaningful native/SVG controls. Page-specific imagery and verification remain in each caller row.
- This source trace does not close outstanding native caller verification. No implementation changed; documentation-only update, no tests rerun.

### Global hooks and notification preview

- Full custom header/footer/extra-links and notification-preview overrides inspected. Verified upstream base/head, base/footer and base/head_navbar extension call sites in Forgejo 15.0.7.
- These own stylesheet loading, theme/appearance controls or compact notification UI, not page illustrations. Keep their meaningful SVG/status content; no independently generated raster image needed.
- Notification fragment caller chain: custom/footer GET request → notification_div preview branch → notification_preview. This source trace does not establish native popover rendering or populated notification behavior.
- Documentation-only assessment; no source changes, tests or notification mutations.

### Personal settings layout ownership trace

- Read complete layout_head/layout_footer and traced all 12 local header callers. Six native page flags select profile/account/appearance/security/keys/applications artwork. Packages, webhooks and organization memberships pass explicit landing artwork; enrollment and OAuth/token editing pass hideArtwork, applied last.
- Previously recorded landing-page screenshots establish the six approved settings images and three new landing images. This trace does not turn those into verification of enrollment, credential editing or every upstream settings caller.
- Footer owns closing markup and native footer delegation only. No new illustration belongs in it. Header remains Integrated — verify until the outstanding native caller states are checked; no duplicate artwork generated.
- Documentation-only assessment; no settings or source changed, no tests rerun.

### Native repository package caller

| Status | Native template / page | Decision / evidence |
| --- | --- | --- |
| No image | Forgejo 15.0.7 `repo/packages.tmpl` — repository Packages | Full native wrapper reviewed. Repository header establishes scope; package-type icons, publication metadata and private-repository/public-owner visibility warning take precedence. No decorative image selected. Native rendering not captured in this assessment. |

- Full `package/shared/list` inspected and traced to both owner branches in `user/overview/packages` and the native repository wrapper. Preserve native package-access checks and distinguish an empty registry from a filtered no-match result.
- Existing owner registry artwork stays in its caller introduction. Adding artwork to this shared list would duplicate that art and impose it on repository content. No new image generated; no packages or settings changed. Documentation-only assessment, no tests rerun.

### Pull-request shared status and navigation

- Full status, trust and tab-menu overrides reviewed. Local conversation/commits/files tab callers traced; stock Forgejo 15.0.7 pull-content call sites confirm status/trust integration.
- Preserve real check states, missing required contexts, trust warnings/permission-gated controls, tab counts and diff statistics. None needs decorative artwork. This call-site trace is not a new full audit of the upstream merge panel.
- No accessible populated pull fixture was created, no checks triggered and no trust state changed. Native rendering remains unverified; documentation-only assessment, no tests rerun.

### Repository utility controls

- Full clone-buttons and issue-navbar helpers inspected; local callers traced. Clone controls serve repository home/empty and wiki view/revision. Issue navigation serves lists, chooser, labels, milestone creation/list and project board.
- Keep protocol/URL/copy controls and labels/milestones links free of decorative artwork. Page-level art decisions remain with their assessed callers; this trace does not add native state coverage.
- Documentation-only change, no commands copied/executed or resources changed; no tests rerun.

### Blocked-user and quota helper callers

- Read both complete helpers and all four native personal/organization blocked_users/storage_overview wrappers in Forgejo 15.0.7. Native caller paths are explicit; these are settings content rather than separate illustration owners.
- Preserve actual identity imagery, unblock targets, quota acceptance/exceeded indicators, totals and breakdown bars. No additional decorative art is appropriate. Existing page-level assessments and their native limitations still apply.
- No users blocked/unblocked or quota settings changed. Source-only caller trace; documentation-only update, no tests rerun.

### Shared commit list

- Full commit-list body reviewed, including author identity, wiki/pull/regular links, signature verification delegate, multiline messages, status/tag delegates, dates, copy-hash and file/path actions.
- Local caller chain: commits/pull commits/compare → commits_table → commits_list; wiki revision calls it directly. Keep this data table free of decorative illustration; existing page-specific decisions remain applicable.
- Source-only caller trace, not new populated-pull/wiki evidence. No Git operations or UI actions executed. Documentation-only update; no tests rerun.

### Organization identity and settings wrappers

- Full org/header and settings/layout_head inspected. Local caller families traced across organization home, member/team/project views, package/code views and settings. Actual avatar/name/visibility and permission-gated actions remain the header's visual content.
- No decorative artwork belongs in this repeated identity wrapper or settings navigation. Individual team/project/registry introductions retain their explicit artwork decisions and outstanding native checks.
- Source-only assessment; no organization created, membership changed or settings submitted. Documentation-only update, no tests rerun.

### Webhook form dispatcher and delivery history

- Full dispatcher and history helper inspected. Dispatcher selects the native provider form and includes history; repository settings wrapper calls the dispatcher. Other owner-page decisions remain recorded separately.
- Provider identity, delivery status and request/response evidence are the meaningful visuals. No decorative art added to these shared sections. Preserve active-state and permission gates around test/replay actions.
- Source-only assessment; no payload delivered, webhook changed or history opened in the UI. Native history verification remains unperformed. Documentation-only update; no tests rerun.

### Shared runner presentation assessment

- Rechecked all five complete runner helpers together: creation, editing, setup, details and list. No decorative artwork is appropriate in these focused forms, setup credentials or operational tables.
- Existing owner-page assessments cover personal, organization, repository and administrator contexts; this helper review does not add native runtime evidence. Admin artwork is opt-in, so these native callers do not inherit the dashboard scene.
- No runner/token created, regenerated, registered or deleted; source placeholders were inspected only. Documentation-only update; no tests rerun.

### Override inventory reconciliation

- Compared all current `.tmpl` files under `appliance/forgejo/templates` against linked checklist entries: **206 actual files, zero missing entries, zero stale file links**. This verifies file coverage, not every upstream route or native state.
- Full repository header reviewed; local callers traced across code, issues, pulls, wiki, projects, releases, Actions, settings, migration and contextual errors. No additional decorative image belongs in its identity/navigation strip.
- All local file entries now have an artwork assessment. Integrated — verify entries still require native rendering checks; previous no-image source decisions do not imply runtime coverage. Restricted administrator/organization/setup/Actions/registration states remain outstanding.

### Accumulated-change regression check

- Checked all literal papercraft references in current overrides: 34 distinct referenced illustration filenames, all present in the workspace. This is file-existence evidence, not visual verification.
- Ran `go test ./scripts -run TestForgejo -count=1`. Initial run exposed three stale account-detail parity normalizers for the intentional hideArtwork input (token editing, OAuth editing, two-factor enrollment). Added exact one-occurrence normalization only for those three fixtures; retained original native-body hashes.
- Re-ran the same Forgejo-focused suite: passed. No runtime page or security state changed. Outstanding native artwork captures still require the requested accessible test states.

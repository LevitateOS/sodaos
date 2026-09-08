# Page illustration checklist

Active goal: assess each page, decide whether art helps, record its scene prompt, generate and integrate missing appropriate art, inspect the native rendering, then advance to the next pending page.

## Workflow and status

- `Pending`: page purpose/layout still needs individual review; no blanket no-image decisions.
- `Existing — verify`: artwork exists; confirm that it fits this page and renders correctly.
- `Generate`: distinct artwork is appropriate; prompt/output/integration/checks remain.
- `Done`: appropriate artwork integrated and inspected, with evidence below.
- `No image`: individually reviewed; record the concrete reason.
- `Partial — trace caller`: not assumed to be a page; account for its actual page callers before closing.

Preserve native headings, forms, permissions and scripts. Use the fixed references and material/character rules in [settings-art-prompts.md](../assets/branding/forgejo/settings-art-prompts.md). Decorative art must not replace meaningful status, avatars or repository content. Each generated page needs its own recorded scene; shared native components do not each need an image.

Scope starts with every current override below. Shared layout coverage can include native pages without a leaf override; add those route variants as their callers are traced. This is an inventory, not a claim that every route has been visited. Existing staged and unstaged work predates this goal and must be preserved.

## Current page

Next: notification subscriptions — `user/notification/notification_subscriptions.tmpl`. It reuses inbox artwork; inspect whether a distinct watching/subscription scene suits the page.

## Per-template inventory

| Status | Template / page entry | Current artwork or caller | Decision / evidence |
| --- | --- | --- | --- |
| Pending | [`admin/applications/list.tmpl`](../appliance/forgejo/templates/admin/applications/list.tmpl) | `admin/layout_head` | Not yet reviewed in this goal. |
| Pending | [`admin/applications/oauth2_edit.tmpl`](../appliance/forgejo/templates/admin/applications/oauth2_edit.tmpl) | `admin/layout_head` | Not yet reviewed in this goal. |
| Pending | [`admin/auth/edit.tmpl`](../appliance/forgejo/templates/admin/auth/edit.tmpl) | `admin/layout_head` | Not yet reviewed in this goal. |
| Pending | [`admin/auth/list.tmpl`](../appliance/forgejo/templates/admin/auth/list.tmpl) | `admin/layout_head` | Not yet reviewed in this goal. |
| Pending | [`admin/auth/new.tmpl`](../appliance/forgejo/templates/admin/auth/new.tmpl) | `admin/layout_head` | Not yet reviewed in this goal. |
| Pending | [`admin/config.tmpl`](../appliance/forgejo/templates/admin/config.tmpl) | `admin/layout_head` | Not yet reviewed in this goal. |
| Pending | [`admin/cron.tmpl`](../appliance/forgejo/templates/admin/cron.tmpl) | `admin/layout_head` | Not yet reviewed in this goal. |
| Pending | [`admin/dashboard.tmpl`](../appliance/forgejo/templates/admin/dashboard.tmpl) | `admin/layout_head` | Not yet reviewed in this goal. |
| Pending | [`admin/emails/list.tmpl`](../appliance/forgejo/templates/admin/emails/list.tmpl) | `admin/layout_head` | Not yet reviewed in this goal. |
| Existing — verify | [`admin/layout_head.tmpl`](../appliance/forgejo/templates/admin/layout_head.tmpl) | `dashboard-papercraft.png` | Not yet reviewed in this goal. |
| Pending | [`admin/notice.tmpl`](../appliance/forgejo/templates/admin/notice.tmpl) | `admin/layout_head` | Not yet reviewed in this goal. |
| Pending | [`admin/queue.tmpl`](../appliance/forgejo/templates/admin/queue.tmpl) | `admin/layout_head` | Not yet reviewed in this goal. |
| Pending | [`admin/queue_manage.tmpl`](../appliance/forgejo/templates/admin/queue_manage.tmpl) | `admin/layout_head` | Not yet reviewed in this goal. |
| Pending | [`admin/repo/list.tmpl`](../appliance/forgejo/templates/admin/repo/list.tmpl) | `admin/layout_head` | Not yet reviewed in this goal. |
| Pending | [`admin/self_check.tmpl`](../appliance/forgejo/templates/admin/self_check.tmpl) | `admin/layout_head` | Not yet reviewed in this goal. |
| Pending | [`admin/stacktrace.tmpl`](../appliance/forgejo/templates/admin/stacktrace.tmpl) | `admin/layout_head` | Not yet reviewed in this goal. |
| Pending | [`admin/user/edit.tmpl`](../appliance/forgejo/templates/admin/user/edit.tmpl) | `admin/layout_head` | Not yet reviewed in this goal. |
| Pending | [`admin/user/list.tmpl`](../appliance/forgejo/templates/admin/user/list.tmpl) | `admin/layout_head` | Not yet reviewed in this goal. |
| Pending | [`admin/user/new.tmpl`](../appliance/forgejo/templates/admin/user/new.tmpl) | `admin/layout_head` | Not yet reviewed in this goal. |
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
| Pending | [`org/home.tmpl`](../appliance/forgejo/templates/org/home.tmpl) | `org/header` | Not yet reviewed in this goal. |
| Pending | [`org/member/members.tmpl`](../appliance/forgejo/templates/org/member/members.tmpl) | `org/header` | Not yet reviewed in this goal. |
| Pending | [`org/projects/list.tmpl`](../appliance/forgejo/templates/org/projects/list.tmpl) | `org/header`, `user/overview/header` | Not yet reviewed in this goal. |
| Pending | [`org/projects/new.tmpl`](../appliance/forgejo/templates/org/projects/new.tmpl) | `user/overview/header` | Not yet reviewed in this goal. |
| Pending | [`org/projects/view.tmpl`](../appliance/forgejo/templates/org/projects/view.tmpl) | `org/header`, `user/overview/header` | Not yet reviewed in this goal. |
| Pending | [`org/settings/layout_head.tmpl`](../appliance/forgejo/templates/org/settings/layout_head.tmpl) | `org/header` | Not yet reviewed in this goal. |
| Pending | [`org/team/invite.tmpl`](../appliance/forgejo/templates/org/team/invite.tmpl) | — | Not yet reviewed in this goal. |
| Pending | [`org/team/members.tmpl`](../appliance/forgejo/templates/org/team/members.tmpl) | `org/header` | Not yet reviewed in this goal. |
| Pending | [`org/team/new.tmpl`](../appliance/forgejo/templates/org/team/new.tmpl) | `org/header` | Not yet reviewed in this goal. |
| Pending | [`org/team/repositories.tmpl`](../appliance/forgejo/templates/org/team/repositories.tmpl) | `org/header` | Not yet reviewed in this goal. |
| Pending | [`org/team/teams.tmpl`](../appliance/forgejo/templates/org/team/teams.tmpl) | `org/header` | Not yet reviewed in this goal. |
| Pending | [`package/settings.tmpl`](../appliance/forgejo/templates/package/settings.tmpl) | `org/header`, `user/overview/header` | Not yet reviewed in this goal. |
| Partial — trace caller | [`package/shared/cargo.tmpl`](../appliance/forgejo/templates/package/shared/cargo.tmpl) | — | Not yet reviewed in this goal. |
| Partial — trace caller | [`package/shared/cleanup_rules/edit.tmpl`](../appliance/forgejo/templates/package/shared/cleanup_rules/edit.tmpl) | — | Not yet reviewed in this goal. |
| Partial — trace caller | [`package/shared/cleanup_rules/list.tmpl`](../appliance/forgejo/templates/package/shared/cleanup_rules/list.tmpl) | — | Not yet reviewed in this goal. |
| Partial — trace caller | [`package/shared/cleanup_rules/preview.tmpl`](../appliance/forgejo/templates/package/shared/cleanup_rules/preview.tmpl) | — | Not yet reviewed in this goal. |
| Partial — trace caller | [`package/shared/list.tmpl`](../appliance/forgejo/templates/package/shared/list.tmpl) | — | Not yet reviewed in this goal. |
| Partial — trace caller | [`package/shared/versionlist.tmpl`](../appliance/forgejo/templates/package/shared/versionlist.tmpl) | — | Not yet reviewed in this goal. |
| Pending | [`package/view.tmpl`](../appliance/forgejo/templates/package/view.tmpl) | `user/overview/header` | Not yet reviewed in this goal. |
| Pending | [`post-install.tmpl`](../appliance/forgejo/templates/post-install.tmpl) | — | Not yet reviewed in this goal. |
| Partial — trace caller | [`projects/list.tmpl`](../appliance/forgejo/templates/projects/list.tmpl) | — | Not yet reviewed in this goal. |
| Partial — trace caller | [`projects/new.tmpl`](../appliance/forgejo/templates/projects/new.tmpl) | — | Not yet reviewed in this goal. |
| Partial — trace caller | [`projects/view.tmpl`](../appliance/forgejo/templates/projects/view.tmpl) | — | Not yet reviewed in this goal. |
| Partial — trace caller | [`repo/actions/dispatch.tmpl`](../appliance/forgejo/templates/repo/actions/dispatch.tmpl) | — | Not yet reviewed in this goal. |
| Pending | [`repo/actions/list.tmpl`](../appliance/forgejo/templates/repo/actions/list.tmpl) | `repo/header` | Not yet reviewed in this goal. |
| Partial — trace caller | [`repo/actions/list_inner.tmpl`](../appliance/forgejo/templates/repo/actions/list_inner.tmpl) | — | Not yet reviewed in this goal. |
| Partial — trace caller | [`repo/actions/no_workflows.tmpl`](../appliance/forgejo/templates/repo/actions/no_workflows.tmpl) | — | Not yet reviewed in this goal. |
| Partial — trace caller | [`repo/actions/runs_list.tmpl`](../appliance/forgejo/templates/repo/actions/runs_list.tmpl) | — | Not yet reviewed in this goal. |
| Pending | [`repo/actions/view.tmpl`](../appliance/forgejo/templates/repo/actions/view.tmpl) | `repo/header` | Not yet reviewed in this goal. |
| Pending | [`repo/activity.tmpl`](../appliance/forgejo/templates/repo/activity.tmpl) | `repo/header` | Not yet reviewed in this goal. |
| Pending | [`repo/branch/list.tmpl`](../appliance/forgejo/templates/repo/branch/list.tmpl) | `repo/header` | Not yet reviewed in this goal. |
| Partial — trace caller | [`repo/branch_dropdown.tmpl`](../appliance/forgejo/templates/repo/branch_dropdown.tmpl) | — | Not yet reviewed in this goal. |
| Partial — trace caller | [`repo/clone_buttons.tmpl`](../appliance/forgejo/templates/repo/clone_buttons.tmpl) | — | Not yet reviewed in this goal. |
| Partial — trace caller | [`repo/commit_header.tmpl`](../appliance/forgejo/templates/repo/commit_header.tmpl) | — | Not yet reviewed in this goal. |
| Pending | [`repo/commit_page.tmpl`](../appliance/forgejo/templates/repo/commit_page.tmpl) | `repo/commit_header`, `repo/header` | Not yet reviewed in this goal. |
| Pending | [`repo/commits.tmpl`](../appliance/forgejo/templates/repo/commits.tmpl) | `repo/header` | Not yet reviewed in this goal. |
| Partial — trace caller | [`repo/commits_list.tmpl`](../appliance/forgejo/templates/repo/commits_list.tmpl) | — | Not yet reviewed in this goal. |
| Partial — trace caller | [`repo/commits_table.tmpl`](../appliance/forgejo/templates/repo/commits_table.tmpl) | — | Not yet reviewed in this goal. |
| Existing — verify | [`repo/create.tmpl`](../appliance/forgejo/templates/repo/create.tmpl) | `new-repo-papercraft.png` | Not yet reviewed in this goal. |
| Partial — trace caller | [`repo/diff/box.tmpl`](../appliance/forgejo/templates/repo/diff/box.tmpl) | `repo/commit_header` | Not yet reviewed in this goal. |
| Pending | [`repo/diff/compare.tmpl`](../appliance/forgejo/templates/repo/diff/compare.tmpl) | `repo/header` | Not yet reviewed in this goal. |
| Pending | [`repo/editor/cherry_pick.tmpl`](../appliance/forgejo/templates/repo/editor/cherry_pick.tmpl) | `repo/header` | Not yet reviewed in this goal. |
| Partial — trace caller | [`repo/editor/commit_form.tmpl`](../appliance/forgejo/templates/repo/editor/commit_form.tmpl) | — | Not yet reviewed in this goal. |
| Pending | [`repo/editor/delete.tmpl`](../appliance/forgejo/templates/repo/editor/delete.tmpl) | `repo/header` | Not yet reviewed in this goal. |
| Pending | [`repo/editor/edit.tmpl`](../appliance/forgejo/templates/repo/editor/edit.tmpl) | `repo/header` | Not yet reviewed in this goal. |
| Pending | [`repo/editor/patch.tmpl`](../appliance/forgejo/templates/repo/editor/patch.tmpl) | `repo/header` | Not yet reviewed in this goal. |
| Pending | [`repo/editor/upload.tmpl`](../appliance/forgejo/templates/repo/editor/upload.tmpl) | `repo/header` | Not yet reviewed in this goal. |
| Pending | [`repo/empty.tmpl`](../appliance/forgejo/templates/repo/empty.tmpl) | `repo/header` | Not yet reviewed in this goal. |
| Pending | [`repo/find/files.tmpl`](../appliance/forgejo/templates/repo/find/files.tmpl) | `repo/header` | Not yet reviewed in this goal. |
| Pending | [`repo/forks.tmpl`](../appliance/forgejo/templates/repo/forks.tmpl) | `repo/header` | Not yet reviewed in this goal. |
| Pending | [`repo/graph.tmpl`](../appliance/forgejo/templates/repo/graph.tmpl) | `repo/header` | Not yet reviewed in this goal. |
| Partial — trace caller | [`repo/header.tmpl`](../appliance/forgejo/templates/repo/header.tmpl) | — | Not yet reviewed in this goal. |
| Pending | [`repo/home.tmpl`](../appliance/forgejo/templates/repo/home.tmpl) | `repo/header` | Not yet reviewed in this goal. |
| Pending | [`repo/issue/choose.tmpl`](../appliance/forgejo/templates/repo/issue/choose.tmpl) | `repo/header` | Not yet reviewed in this goal. |
| Pending | [`repo/issue/labels.tmpl`](../appliance/forgejo/templates/repo/issue/labels.tmpl) | `repo/header` | Not yet reviewed in this goal. |
| Pending | [`repo/issue/list.tmpl`](../appliance/forgejo/templates/repo/issue/list.tmpl) | `repo/header` | Not yet reviewed in this goal. |
| Pending | [`repo/issue/milestone_issues.tmpl`](../appliance/forgejo/templates/repo/issue/milestone_issues.tmpl) | `repo/header` | Not yet reviewed in this goal. |
| Pending | [`repo/issue/milestone_new.tmpl`](../appliance/forgejo/templates/repo/issue/milestone_new.tmpl) | `repo/header` | Not yet reviewed in this goal. |
| Pending | [`repo/issue/milestones.tmpl`](../appliance/forgejo/templates/repo/issue/milestones.tmpl) | `repo/header` | Not yet reviewed in this goal. |
| Partial — trace caller | [`repo/issue/navbar.tmpl`](../appliance/forgejo/templates/repo/issue/navbar.tmpl) | — | Not yet reviewed in this goal. |
| Pending | [`repo/issue/new.tmpl`](../appliance/forgejo/templates/repo/issue/new.tmpl) | `repo/header` | Not yet reviewed in this goal. |
| Pending | [`repo/issue/view.tmpl`](../appliance/forgejo/templates/repo/issue/view.tmpl) | `repo/header` | Not yet reviewed in this goal. |
| Done | [`repo/migrate/migrate.tmpl`](../appliance/forgejo/templates/repo/migrate/migrate.tmpl) | `migrate-papercraft.png` | Dedicated history/import scene; desktop/mobile native captures inspected. See record below. |
| No image | [`repo/migrate/migrating.tmpl`](../appliance/forgejo/templates/repo/migrate/migrating.tmpl) | Native loading/error visuals | Live progress, failure and retry/cancel controls already communicate operation state. Extra decorative art would compete with those signals; source reviewed, no operation launched. |
| Partial — no image | [`repo/migrate/options.tmpl`](../appliance/forgejo/templates/repo/migrate/options.tmpl) | Native provider-form callers | Reusable inline mirror/LFS fields, not a page or introduction. Do not insert decorative artwork among native provider options; provider page assessment remains separate. |
| Pending | [`repo/projects/list.tmpl`](../appliance/forgejo/templates/repo/projects/list.tmpl) | `repo/header` | Not yet reviewed in this goal. |
| Pending | [`repo/projects/new.tmpl`](../appliance/forgejo/templates/repo/projects/new.tmpl) | `repo/header` | Not yet reviewed in this goal. |
| Pending | [`repo/projects/view.tmpl`](../appliance/forgejo/templates/repo/projects/view.tmpl) | `repo/header` | Not yet reviewed in this goal. |
| Pending | [`repo/pulls/commits.tmpl`](../appliance/forgejo/templates/repo/pulls/commits.tmpl) | `repo/header` | Not yet reviewed in this goal. |
| Pending | [`repo/pulls/files.tmpl`](../appliance/forgejo/templates/repo/pulls/files.tmpl) | `repo/header` | Not yet reviewed in this goal. |
| Done | [`repo/pulls/fork.tmpl`](../appliance/forgejo/templates/repo/pulls/fork.tmpl) | `fork-papercraft.png` | Independent-copy scene; desktop/mobile native captures inspected. See record below. |
| Partial — trace caller | [`repo/pulls/status.tmpl`](../appliance/forgejo/templates/repo/pulls/status.tmpl) | — | Not yet reviewed in this goal. |
| Partial — trace caller | [`repo/pulls/tab_menu.tmpl`](../appliance/forgejo/templates/repo/pulls/tab_menu.tmpl) | — | Not yet reviewed in this goal. |
| Partial — trace caller | [`repo/pulls/trust.tmpl`](../appliance/forgejo/templates/repo/pulls/trust.tmpl) | — | Not yet reviewed in this goal. |
| Pending | [`repo/release/list.tmpl`](../appliance/forgejo/templates/repo/release/list.tmpl) | `repo/header`, `repo/release_tag_header` | Not yet reviewed in this goal. |
| Pending | [`repo/release/new.tmpl`](../appliance/forgejo/templates/repo/release/new.tmpl) | `repo/header` | Not yet reviewed in this goal. |
| Partial — trace caller | [`repo/release_tag_header.tmpl`](../appliance/forgejo/templates/repo/release_tag_header.tmpl) | — | Not yet reviewed in this goal. |
| Pending | [`repo/search.tmpl`](../appliance/forgejo/templates/repo/search.tmpl) | `repo/header` | Not yet reviewed in this goal. |
| Pending | [`repo/settings/actions.tmpl`](../appliance/forgejo/templates/repo/settings/actions.tmpl) | `repo/settings/layout_head` | Not yet reviewed in this goal. |
| Pending | [`repo/settings/branches.tmpl`](../appliance/forgejo/templates/repo/settings/branches.tmpl) | `repo/settings/layout_head` | Not yet reviewed in this goal. |
| Pending | [`repo/settings/collaboration.tmpl`](../appliance/forgejo/templates/repo/settings/collaboration.tmpl) | `repo/settings/layout_head` | Not yet reviewed in this goal. |
| Pending | [`repo/settings/deploy_keys.tmpl`](../appliance/forgejo/templates/repo/settings/deploy_keys.tmpl) | `repo/settings/layout_head` | Not yet reviewed in this goal. |
| Pending | [`repo/settings/githook_edit.tmpl`](../appliance/forgejo/templates/repo/settings/githook_edit.tmpl) | `repo/settings/layout_head` | Not yet reviewed in this goal. |
| Pending | [`repo/settings/githooks.tmpl`](../appliance/forgejo/templates/repo/settings/githooks.tmpl) | `repo/settings/layout_head` | Not yet reviewed in this goal. |
| Pending | [`repo/settings/layout_head.tmpl`](../appliance/forgejo/templates/repo/settings/layout_head.tmpl) | `repo/header` | Not yet reviewed in this goal. |
| Pending | [`repo/settings/lfs.tmpl`](../appliance/forgejo/templates/repo/settings/lfs.tmpl) | `repo/settings/layout_head` | Not yet reviewed in this goal. |
| Pending | [`repo/settings/lfs_file.tmpl`](../appliance/forgejo/templates/repo/settings/lfs_file.tmpl) | `repo/settings/layout_head` | Not yet reviewed in this goal. |
| Pending | [`repo/settings/lfs_file_find.tmpl`](../appliance/forgejo/templates/repo/settings/lfs_file_find.tmpl) | `repo/settings/layout_head` | Not yet reviewed in this goal. |
| Pending | [`repo/settings/lfs_locks.tmpl`](../appliance/forgejo/templates/repo/settings/lfs_locks.tmpl) | `repo/settings/layout_head` | Not yet reviewed in this goal. |
| Pending | [`repo/settings/lfs_pointers.tmpl`](../appliance/forgejo/templates/repo/settings/lfs_pointers.tmpl) | `repo/settings/layout_head` | Not yet reviewed in this goal. |
| Pending | [`repo/settings/options.tmpl`](../appliance/forgejo/templates/repo/settings/options.tmpl) | `repo/settings/layout_head` | Not yet reviewed in this goal. |
| Pending | [`repo/settings/protected_branch.tmpl`](../appliance/forgejo/templates/repo/settings/protected_branch.tmpl) | `repo/settings/layout_head` | Not yet reviewed in this goal. |
| Pending | [`repo/settings/runner_create.tmpl`](../appliance/forgejo/templates/repo/settings/runner_create.tmpl) | `repo/settings/layout_head` | Not yet reviewed in this goal. |
| Pending | [`repo/settings/runner_details.tmpl`](../appliance/forgejo/templates/repo/settings/runner_details.tmpl) | `repo/settings/layout_head` | Not yet reviewed in this goal. |
| Pending | [`repo/settings/runner_edit.tmpl`](../appliance/forgejo/templates/repo/settings/runner_edit.tmpl) | `repo/settings/layout_head` | Not yet reviewed in this goal. |
| Pending | [`repo/settings/runner_setup.tmpl`](../appliance/forgejo/templates/repo/settings/runner_setup.tmpl) | `repo/settings/layout_head` | Not yet reviewed in this goal. |
| Pending | [`repo/settings/secrets.tmpl`](../appliance/forgejo/templates/repo/settings/secrets.tmpl) | `repo/settings/layout_head` | Not yet reviewed in this goal. |
| Pending | [`repo/settings/tags.tmpl`](../appliance/forgejo/templates/repo/settings/tags.tmpl) | `repo/settings/layout_head` | Not yet reviewed in this goal. |
| Pending | [`repo/settings/units.tmpl`](../appliance/forgejo/templates/repo/settings/units.tmpl) | `repo/settings/layout_head` | Not yet reviewed in this goal. |
| Pending | [`repo/settings/webhook/base.tmpl`](../appliance/forgejo/templates/repo/settings/webhook/base.tmpl) | `repo/settings/layout_head` | Not yet reviewed in this goal. |
| Partial — trace caller | [`repo/settings/webhook/base_list.tmpl`](../appliance/forgejo/templates/repo/settings/webhook/base_list.tmpl) | — | Not yet reviewed in this goal. |
| Partial — trace caller | [`repo/settings/webhook/history.tmpl`](../appliance/forgejo/templates/repo/settings/webhook/history.tmpl) | — | Not yet reviewed in this goal. |
| Pending | [`repo/settings/webhook/new.tmpl`](../appliance/forgejo/templates/repo/settings/webhook/new.tmpl) | `repo/settings/layout_head` | Not yet reviewed in this goal. |
| Partial — trace caller | [`repo/sub_menu.tmpl`](../appliance/forgejo/templates/repo/sub_menu.tmpl) | — | Not yet reviewed in this goal. |
| Pending | [`repo/tag/list.tmpl`](../appliance/forgejo/templates/repo/tag/list.tmpl) | `repo/header`, `repo/release_tag_header` | Not yet reviewed in this goal. |
| Partial — trace caller | [`repo/user_cards.tmpl`](../appliance/forgejo/templates/repo/user_cards.tmpl) | — | Not yet reviewed in this goal. |
| Partial — trace caller | [`repo/view_file.tmpl`](../appliance/forgejo/templates/repo/view_file.tmpl) | — | Not yet reviewed in this goal. |
| Partial — trace caller | [`repo/view_list.tmpl`](../appliance/forgejo/templates/repo/view_list.tmpl) | — | Not yet reviewed in this goal. |
| Pending | [`repo/watchers.tmpl`](../appliance/forgejo/templates/repo/watchers.tmpl) | `repo/header` | Not yet reviewed in this goal. |
| Pending | [`repo/wiki/new.tmpl`](../appliance/forgejo/templates/repo/wiki/new.tmpl) | `repo/header` | Not yet reviewed in this goal. |
| Pending | [`repo/wiki/pages.tmpl`](../appliance/forgejo/templates/repo/wiki/pages.tmpl) | `repo/header` | Not yet reviewed in this goal. |
| Pending | [`repo/wiki/revision.tmpl`](../appliance/forgejo/templates/repo/wiki/revision.tmpl) | `repo/header` | Not yet reviewed in this goal. |
| Partial — trace caller | [`repo/wiki/search.tmpl`](../appliance/forgejo/templates/repo/wiki/search.tmpl) | — | Not yet reviewed in this goal. |
| Pending | [`repo/wiki/start.tmpl`](../appliance/forgejo/templates/repo/wiki/start.tmpl) | `repo/header` | Not yet reviewed in this goal. |
| Pending | [`repo/wiki/view.tmpl`](../appliance/forgejo/templates/repo/wiki/view.tmpl) | `repo/header` | Not yet reviewed in this goal. |
| Partial — trace caller | [`shared/actions/runner_create.tmpl`](../appliance/forgejo/templates/shared/actions/runner_create.tmpl) | — | Not yet reviewed in this goal. |
| Partial — trace caller | [`shared/actions/runner_details.tmpl`](../appliance/forgejo/templates/shared/actions/runner_details.tmpl) | — | Not yet reviewed in this goal. |
| Partial — trace caller | [`shared/actions/runner_edit.tmpl`](../appliance/forgejo/templates/shared/actions/runner_edit.tmpl) | — | Not yet reviewed in this goal. |
| Partial — trace caller | [`shared/actions/runner_list.tmpl`](../appliance/forgejo/templates/shared/actions/runner_list.tmpl) | — | Not yet reviewed in this goal. |
| Partial — trace caller | [`shared/actions/runner_setup.tmpl`](../appliance/forgejo/templates/shared/actions/runner_setup.tmpl) | — | Not yet reviewed in this goal. |
| Partial — trace caller | [`shared/blocked_users_list.tmpl`](../appliance/forgejo/templates/shared/blocked_users_list.tmpl) | — | Not yet reviewed in this goal. |
| Partial — trace caller | [`shared/quota_overview.tmpl`](../appliance/forgejo/templates/shared/quota_overview.tmpl) | — | Not yet reviewed in this goal. |
| Partial — trace caller | [`shared/secrets/add_list.tmpl`](../appliance/forgejo/templates/shared/secrets/add_list.tmpl) | — | Not yet reviewed in this goal. |
| Partial — trace caller | [`shared/variables/variable_list.tmpl`](../appliance/forgejo/templates/shared/variables/variable_list.tmpl) | — | Not yet reviewed in this goal. |
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
| Existing — verify | [`user/notification/notification_subscriptions.tmpl`](../appliance/forgejo/templates/user/notification/notification_subscriptions.tmpl) | `notifications-papercraft.png` | Not yet reviewed in this goal. |
| Partial — trace caller | [`user/overview/header.tmpl`](../appliance/forgejo/templates/user/overview/header.tmpl) | — | Not yet reviewed in this goal. |
| Pending | [`user/overview/package_versions.tmpl`](../appliance/forgejo/templates/user/overview/package_versions.tmpl) | `org/header`, `user/overview/header` | Not yet reviewed in this goal. |
| Pending | [`user/overview/packages.tmpl`](../appliance/forgejo/templates/user/overview/packages.tmpl) | `org/header`, `user/overview/header` | Not yet reviewed in this goal. |
| Pending | [`user/profile.tmpl`](../appliance/forgejo/templates/user/profile.tmpl) | `user/overview/header` | Not yet reviewed in this goal. |
| Pending | [`user/settings/access_token_edit.tmpl`](../appliance/forgejo/templates/user/settings/access_token_edit.tmpl) | `user/settings/layout_head` | Not yet reviewed in this goal. |
| Pending | [`user/settings/account.tmpl`](../appliance/forgejo/templates/user/settings/account.tmpl) | `user/settings/layout_head` | Not yet reviewed in this goal. |
| Pending | [`user/settings/appearance.tmpl`](../appliance/forgejo/templates/user/settings/appearance.tmpl) | `user/settings/layout_head` | Not yet reviewed in this goal. |
| Pending | [`user/settings/applications.tmpl`](../appliance/forgejo/templates/user/settings/applications.tmpl) | `user/settings/layout_head` | Not yet reviewed in this goal. |
| Pending | [`user/settings/applications_oauth2_edit.tmpl`](../appliance/forgejo/templates/user/settings/applications_oauth2_edit.tmpl) | `user/settings/layout_head` | Not yet reviewed in this goal. |
| Partial — trace caller | [`user/settings/applications_oauth2_list.tmpl`](../appliance/forgejo/templates/user/settings/applications_oauth2_list.tmpl) | — | Not yet reviewed in this goal. |
| Pending | [`user/settings/keys.tmpl`](../appliance/forgejo/templates/user/settings/keys.tmpl) | `user/settings/layout_head` | Not yet reviewed in this goal. |
| Partial — trace caller | [`user/settings/layout_footer.tmpl`](../appliance/forgejo/templates/user/settings/layout_footer.tmpl) | — | Not yet reviewed in this goal. |
| Existing — verify | [`user/settings/layout_head.tmpl`](../appliance/forgejo/templates/user/settings/layout_head.tmpl) | `settings-account-papercraft.png`, `settings-appearance-papercraft.png`, `settings-applications-papercraft.png`, `settings-keys-papercraft.png`, `settings-profile-papercraft.png`, `settings-security-papercraft.png` | Not yet reviewed in this goal. |
| Pending | [`user/settings/profile.tmpl`](../appliance/forgejo/templates/user/settings/profile.tmpl) | `user/settings/layout_head` | Not yet reviewed in this goal. |
| Pending | [`user/settings/security/security.tmpl`](../appliance/forgejo/templates/user/settings/security/security.tmpl) | `user/settings/layout_head` | Not yet reviewed in this goal. |
| Pending | [`user/settings/security/twofa_enroll.tmpl`](../appliance/forgejo/templates/user/settings/security/twofa_enroll.tmpl) | `user/settings/layout_head` | Not yet reviewed in this goal. |
| Partial — trace caller | [`webhook/new.tmpl`](../appliance/forgejo/templates/webhook/new.tmpl) | — | Not yet reviewed in this goal. |
| Partial — trace caller | [`webhook/shared-settings.tmpl`](../appliance/forgejo/templates/webhook/shared-settings.tmpl) | — | Not yet reviewed in this goal. |

## Additional page variants

- [ ] Split combined Explore users/organizations, dashboard issues/pulls and personal/organization page branches into individual decisions.
- [ ] Trace account, organization, repository and administrator shared layouts for native routes without leaf overrides.
- [ ] Distinguish new/edit/detail states where one template serves multiple page purposes.

## Completed page records

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

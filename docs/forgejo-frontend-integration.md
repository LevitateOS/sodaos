# Forgejo frontend integration

Use **official template overrides and native Forgejo rendering**, not a separate
frontend that requires exposing every workflow as JSON. The
[leading plan](dashboard-implementation-plan.md) owns implementation and acceptance;
[H01](forgejo-api-coverage.md) retains source/API/workflow findings, not a patch backlog.
Only bounded U08 is accepted; U01's architecture acceptance was withdrawn.

## Supported lanes before architectural escalation

1. Existing native workflows and configuration for the desired behavior.
2. Custom themes/CSS/assets for presentation.
3. Dedicated template extension points for small additions.
4. Whole-template overrides only where extension points are insufficient.
5. Existing REST/OAuth/webhook integrations for concrete Soda-owned callers.
6. Native Git/SSH/HTTPS/LFS/package protocols and Actions for their intended uses.

These are version-specific interfaces, not a claim that every arbitrary change is
supported. A missing JSON endpoint does not imply that the native page is missing.
Forgejo's web handlers call internal services and render templates independently
of its public API. Do not extract/scrape complete HTML documents into React, borrow
native cookies or use a privileged token to bypass a missing interface.

## Actual template mechanism and limits

In inspected source, `modules/templates/base.go::AssetFS` layers custom templates
from `<CustomPath>/templates/` before built-in templates. Shared native templates
call extension points such as `custom/header.tmpl`, `custom/footer.tmpl`,
`custom/extra_links.tmpl`, `custom/extra_links_footer.tmpl` and `custom/extra_tabs.tmpl`.
Preserve native template context, form fields/actions, scripts/assets and security
checks. This supported customization does not require compiling Forgejo.

U01 must inspect the selected stock version/configuration, exact shared shell and
extension points, custom-path delivery, reload/restart behavior and browser/Soda
integration before changes. Do not assume a template can arbitrarily call Soda's
backend or that matching branding establishes a shared authenticated session.
Prove native form/script compatibility and enrolled-key/origin preservation.

Keep ordinary native pages—including login/password/MFA/consent/admin—working.
Soda's environment UI and legitimate stock API clients remain separate integration
responsibilities. Preserve complete user workflows and all conditional-enabled
features, not necessarily the old duplicate React screens. Retire those callers
only after verified replacements.

## Hard stopping point

A downstream Forgejo source patch/custom executable is a **failure path**. Stop
and explain the exact limitation; revisit the architecture with the user rather
than invent an API, actor/grant engine, patch series or build pipeline. Separately
scoped upstream contributions are not permission to depend on an unmerged fork.
Do not weaken native security or silently remove a required workflow to avoid
raising the conflict.

The fork-specific source preparer, development lock and native auth/read/build
proposals have been removed. Git and ignored evidence retain their history; there
is no dormant alternate backend. See [licensing](licensing.md) for retained notice/
source obligations, which apply to stock-image/template distribution too.

No template override, native upgrade, ingress change or deployment is claimed by
this guide. Working login and all four persistent environments remain untouched.

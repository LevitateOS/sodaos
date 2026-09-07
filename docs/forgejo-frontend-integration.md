# Forgejo customization contract

Use stock Forgejo's frontend throughout; Soda adds the **Sodaspaces** repository
tab/environment drawer. See the [short plan](sodaspaces-plan.md) for pending work and
the [handoff](implementation-status.md) for source versus installed evidence. No tab,
drawer or authenticated native-page → Soda connection is implemented yet.

## Verified source surface

The inspected **15.0.7** source establishes:

- `modules/templates/base.go::AssetFS` layers `<CustomPath>/templates/` ahead of
  built-ins; absent overrides fall back to native templates.
- `templates/base/head_navbar.tmpl` calls empty `custom/extra_links.tmpl`.
- `templates/repo/header.tmpl` calls empty `custom/extra_tabs.tmpl` with repository
  context. Native permission/unit gates remain upstream-owned.
- Header/footer/body hooks also exist. Prefer dedicated hooks; copy an entire
  template only for a concrete unmet requirement, not the unchanged reference tree.
- The root image/wrapper defaults CustomPath to `/data/gitea`. Soda mounts
  `/var/lib/soda/forgejo` as `/data`; the candidate override location is therefore
  `/var/lib/soda/forgejo/gitea/templates/`. Verify effective configuration and
  reload/restart requirements before any rollout.

Staging already supplies native themes/logos under `gitea/public/assets`, **not** a
new template delivery/render path. Forgejo's customized Fomantic subset has native
accessibility/initialization adaptations; it is not a standalone Soda component kit.
Preserve native markup, scripts, form behavior and branding. The source reference
is retained under `.artifacts/research/h01-0f43b9f/v15.0.7/forgejo/`; inspect the exact
selected version/configuration when changing an override.

## Integration limits

A template cannot directly call Soda's database/backend. An OAuth grant is not a
native web session, and different ports do not isolate cookies. Preserve actual
actor/repository identity, origin/CSRF and native WebAuthn/session boundaries; the
[existing API](dashboard-api.md) is not proof of authenticated embedding.

If supported configuration/templates/assets/APIs cannot meet the requirement,
explain the concrete constraint and return for a decision. Do not fork Forgejo,
ship a custom executable, scrape/relay HTML, borrow credentials or introduce a
replacement frontend/framework. Keep native Git/LFS/package and administrator flows.

Overrides/assets require [matching notices and source obligations](licensing.md).
No image upgrade, ingress change, service restart or provider mutation is authorized
by this guide. All retained project state, private credentials and evidence stay intact.

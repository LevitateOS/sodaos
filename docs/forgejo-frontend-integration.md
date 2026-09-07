# Forgejo frontend and Sodaspaces integration

**Use Forgejo's native frontend throughout.** Official template overrides add
Soda-specific functionality rather than rebuilding the forge. The selected product
name is **Sodaspaces**: a repository tab and proposed right-side environment drawer.
No Bootstrap, replacement component library, React shell, iframe or HTML-scraping
proxy is selected. Forgejo's native scripts/styles and separate Cockpit React/
PatternFly pages remain. The [leading plan](dashboard-implementation-plan.md) owns
implementation/acceptance; only bounded U08 is accepted.

## Deliberately supported customization

The retained H01 **15.0.7** source establishes:

- `modules/templates/base.go::AssetFS` layers `<CustomPath>/templates/` before
  built-in templates. Missing overrides fall back to native templates.
- `templates/base/head_navbar.tmpl` calls the empty `custom/extra_links.tmpl` hook.
- `templates/repo/header.tmpl` calls the empty `custom/extra_tabs.tmpl` hook with
  repository context; shared native code retains its permission/unit gates.
- `custom/header.tmpl`, `custom/footer.tmpl` and other body hooks provide additional
  extension points. Copy a whole upstream template only for a concrete unmet need.
- The root image/wrapper defaults CustomPath to `/data/gitea`. The appliance mounts
  `/var/lib/soda/forgejo` as `/data`; the candidate override destination is therefore
  `/var/lib/soda/forgejo/gitea/templates/`. Verify effective runtime configuration
  and reload/restart requirements before rollout.

Current staging already supplies native themes/logos under `gitea/public/assets`.
Forgejo uses a customized Fomantic subset with native accessibility/initialization
adaptations; neither wholesale Fomantic transplantation nor replacing native scripts
is selected. Use native markup/styles for additions, with focused JavaScript only
where needed. Do not globally inject another CSS framework or redraw branding.

## Sodaspaces source still to implement

1. Add a repository Sodaspaces navigation control through the verified hook.
2. Implement a focused right-side drawer using inspected native page conventions.
   Preserve required native DOM/scripts, responsive/keyboard/focus behavior and
   ordinary navigation. Opening a drawer must not create/join/start an environment.
3. Establish a supported authenticated path from that native page to Soda's backend.
   A Forgejo template cannot call Soda's database, and an OAuth grant is not an
   assumed native browser session. Different ports do not isolate cookies. Review
   configured origins, native/Soda identity mismatch, CSRF, return navigation,
   expiry/revocation and actual logout effects before wiring any mutation.
4. Resolve repository context by stable native ID, under current actor permissions,
   and show real existing/incomplete/absent environment state. Preserve trusted
   memberships and all existing Linux data. No role inventory or automatic remapping.
5. Wire explicit create/join, own public development keys and connection details to
   the retained Go/native operations; then implement the U07 own-account terminal.
   No host-root shell, implicit lifecycle, key upload or general workload controller.
6. Prove one full native-page/Soda journey before extending coverage. Exact template/
   asset/browser tests and approved live integration remain due; no drawer has been
   delivered by deleting the React frontend.

If the desired authenticated integration cannot be achieved through supported
mechanisms, explain the exact conflict and revisit it with the user. **A downstream
Forgejo patch/custom executable is an architectural failure path**, not permission
to recreate the removed build/API platform or weaken native security.

## What remains after frontend removal

Both standalone React and original Go/HTMX pages/forms/static assets are removed.
The Go service retains OAuth and the protected
[environment/access API](dashboard-api.md), SQLite/grants, restricted helper,
setup/advertisement clients and native tests remain. The root React directory,
SPA serving/validation, duplicate collaboration clients/handlers and their dedicated
build/browser support are removed. `/app/`, old HTML/form paths and duplicate forge
API routes are no longer application routes; deployed bytes/evidence are historical.
Root and completed OAuth redirect only to configured native Forgejo. This is not
native session transfer or a working Sodaspaces entry; no Soda browser UI remains.

Template/source notices remain required; see [licensing](licensing.md). No template
rollout, image upgrade, ingress change or service restart is authorized by this
guide. Preserve all four project environments, credentials, backups and evidence.

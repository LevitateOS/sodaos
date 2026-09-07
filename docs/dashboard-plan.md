# Soda frontend inventory

**Selected: Forgejo's native frontend plus Sodaspaces.** Only bounded U08 is accepted;
U01's architecture acceptance remains withdrawn. Follow the
[leading plan](dashboard-implementation-plan.md), [native integration guide](forgejo-frontend-integration.md),
[179-group workflow register](forgejo-api-coverage.md) and [handoff](implementation-status.md).

## UI and authority

| Area | UI owner | Soda responsibility |
| --- | --- | --- |
| Login/password/MFA/consent/recovery/IdP | Native Forgejo | Supported customization and protected Soda OAuth integration |
| Repositories/code/history/blame/diffs | Native Forgejo | Preserve scripts/forms; add Sodaspaces entry |
| Issues/PRs/boards/settings/organizations | Native Forgejo | Native permissions/outcomes and customization compatibility |
| Work/search/notifications/Actions/publishing/wiki/packages/admin | Native Forgejo | Preserve complete conditional-enabled native workflows/protocols |
| Environments/membership/development keys/connections/terminal | Sodaspaces addition to native Forgejo, not yet implemented | Existing Go/SQLite/restricted helper with real provisioning and explicit actor actions |
| Host administration/Tailnet/local runner capacity | Separate stock Cockpit and retained pages | Preserve native operator boundary and backing logic/tests |

Forgejo owns business rules, identity, permissions and native administration. Soda
owns only its extension records and operations. Site admin, organization/repository
owner/admin, Soda operator and host root are not interchangeable. Native Git keys
and development-access keys remain distinct. A tab's visibility never authorizes
provisioning, a shell or cross-project access.

Sodaspaces is a UI name, not Codespaces' per-user/disposable lifecycle or a database/
container rename. Shared persistent environments, project-local human accounts,
ordinary Git/mise/workloads and direct-IP SSH/SCP/SFTP remain the product.

## Removed versus retained source

- Root `dashboard/`, React/router/Zustand/PatternFly dashboard support, SPA serving/
  validation, duplicate Forgejo workflow handlers/clients and their exclusive tests
  are removed. No Bootstrap or replacement component library is selected.
- Old Go/HTMX templates, static assets, forms and dedicated clients/tests are also
  removed. `internal/web/` retains OAuth and [Soda APIs](dashboard-api.md) for real
  preferences/keys/environments. There is no standalone Soda browser UI. Root and
  OAuth completion return to configured native Forgejo, not removed pages.
- Go config/store/grants/native helper, stock Forgejo API/OAuth/setup/advertisement
  and native repository lookup/ownership remain. Native account administration
  stays wholly in Forgejo; its former Soda People/picker clients are removed. No
  provider-role inventory, replacement Git engine or generalized recovery subsystem.
- `cockpit/` retains React/PatternFly/Vite+, its own lock and license collector;
  shared native/branding dependencies remain. Node is still a Cockpit build tool,
  not a new production service. Go dependency metadata is unchanged by removal.
- Existing stock image/IID/OCI/build/stage/install consumers remain. Soda is an
  API/OAuth command without HTML/static payload. New bundles reject stale SPA output.
- Canonical assets, existing notices and [licensing obligations](licensing.md) stay.
  Old source/artifact evidence retains its original license/version attribution.

## Remaining integration and evidence

The exact tab/drawer/authenticated request path needs bounded review and implemented
source/tests. Supported navigation hooks are verified in 15.0.7 source; that is not
a working drawer, shared session or hot-reload guarantee. Keep native forms/scripts,
CSRF, cookie/origin/RP-ID and Git/protocol endpoints intact. A missing JSON route is
not a reason to patch Forgejo or recreate a native workflow in Soda.

The retained `fed66cb` authority boundary remains: acting grants and current-owner
environment visibility. The old account forms are removed, not replaced with an
operator-token or local-validation fallback. No Linux remapping/offboarding is promised.
Terminal remains U07; optional lifecycle/profiles/limits remain unselected E tracks.

Installed components still `8b823db` with stock 15.0.7 and the historical React preview;
source removal is not a deployment. Four environments and all identities, keys,
roots/workloads/dirty data/backups/evidence stay preserved. U17 rehearses actual
Sodaspaces integration before separately approved U18 cutover; U20 owns final
fresh/populated/both-architecture/operator/product proof.

# Appliance definitions

[Responsibility map index](README.md) · [Coverage scope](../README.md).

Current responsibility accounting at `519b76bd` (2026-10-07).
One slice owns each named duty; disjoint complete symbols may share an owner.
Compound fields/clauses may share a physical line with distinct selectors.
Disposition concerns the duty, not source validity or installed qualification.

<a id="coverage-b39bd042512f"></a>

## [frontend/forgejo/README.md](../../../../../frontend/forgejo/README.md)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–782; file scaffold; and | [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) | retained | Current scaffold duty: file scaffold; Current declaration duty: and — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-7739cb525c8e"></a>

## [frontend/forgejo/templates/repo/actions/list.tmpl](../../../../../frontend/forgejo/templates/repo/actions/list.tmpl)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–42; file scaffold; pollingOk; noActiveDropdowns | [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) | retained | Current scaffold duty: file scaffold; Current declaration duty: pollingOk; Current declaration duty: noActiveDropdowns — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-21ba044ba45b"></a>
<a id="coverage-13a20fd79ea1"></a>

## [system/containers/extension/extension.json](../../../../../system/containers/extension/extension.json)

Current configuration/service body read and reconciled to line-bounded responsibility units

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–7, 10, 14–26; current configuration/service block | [G01](../../slices/forgejo-integration.md#g01-browser-authority-and-contributions) | retained | Native extension context and browser contribution admission — system/containers/extension/extension.json:1-7,10-10,14-26; current body read; consumer/selector is the service, rootfs copy or extension manifest named by the file |
| 8, 11–13; current configuration/service block | [S02](../../slices/spaces-and-terminals.md#s02-workspace-lifetime-and-navigation) | retained | Spaces page contribution; Persistent Workspace panel contribution — system/containers/extension/extension.json:8-8; current body read; consumer/selector is the service, rootfs copy or extension manifest named by the file; system/containers/extension/extension.json:11-13; current body read; consumer/selector is the service, rootfs copy or extension manifest named by the file |
| 9; current configuration/service block | [N03](../../slices/networking.md#n03-host-tailnet-control) | retained | Operator Tailnet page contribution — system/containers/extension/extension.json:9-9; current body read; consumer/selector is the service, rootfs copy or extension manifest named by the file |

<a id="coverage-2ff14f94cf6d"></a>
<a id="coverage-16534b6a88eb"></a>

## [system/host/config/forgejo.env](../../../../../system/host/config/forgejo.env)

Current configuration/service body read and reconciled to line-bounded responsibility units

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–14; current configuration/service block | [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) | retained | Maintained native Forgejo presentation and browser enhancement — system/host/config/forgejo.env:1-14; current body read; consumer/selector is the service, rootfs copy or extension manifest named by the file |
| 15–19; current configuration/service block | [O04](../../slices/operator-administration.md#o04-native-host-administration-and-updates) | retained | Native service diagnostic policy: console method/EscapedPath/status only, exclude request query/referrer fields — system/host/config/forgejo.env:15-19; current body read; consumer/selector is the service, rootfs copy or extension manifest named by the file |

<a id="coverage-d6987f7d8f3c"></a>
<a id="coverage-49e48bbaf0a9"></a>

## [system/host/config/proxy.Caddyfile](../../../../../system/host/config/proxy.Caddyfile)

Current configuration/service body read and reconciled to line-bounded responsibility units

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–8, 16–19; current configuration/service block | [N01](../../slices/networking.md#n01-private-origins-tls-and-activation) | retained | Private origin, TLS, activation and proxy routing — system/host/config/proxy.Caddyfile:1-8,16-19; current body read; consumer/selector is the service, rootfs copy or extension manifest named by the file |
| 9–15; current configuration/service block | [H05](../../slices/shared-supporting-slices.md#h05-branding-avatars-and-attribution) | retained | Public avatar proxy strips browser credentials — system/host/config/proxy.Caddyfile:9-15; current body read; consumer/selector is the service, rootfs copy or extension manifest named by the file |

<a id="coverage-b5b1e03b7afd"></a>
<a id="coverage-6a08317f1040"></a>

## [system/host/config/soda.sysusers](../../../../../system/host/config/soda.sysusers)

Current configuration/service body read and reconciled to line-bounded responsibility units

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–2, 4; current configuration/service block | [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) | retained | Private IPC/service process lifetime — system/host/config/soda.sysusers:1-2,4-4; current body read; consumer/selector is the service, rootfs copy or extension manifest named by the file |
| 3; current configuration/service block | [I02](../../slices/identity-brokering.md#i02-encrypted-credential-custody) | retained | Dedicated broker OS identity — system/host/config/soda.sysusers:3-3; current body read; consumer/selector is the service, rootfs copy or extension manifest named by the file |

<a id="coverage-6f9dcc209ec6"></a>
<a id="coverage-c70c6576ca83"></a>

## [system/host/config/soda.tmpfiles](../../../../../system/host/config/soda.tmpfiles)

Current configuration/service body read and reconciled to line-bounded responsibility units

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1; current configuration/service block | [H04](../../slices/shared-supporting-slices.md#h04-configuration-and-filesystem-primitives) | retained | Configuration/filesystem primitives — system/host/config/soda.tmpfiles:1-1; current body read; consumer/selector is the service, rootfs copy or extension manifest named by the file |
| 2, 16–21; current configuration/service block | [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) | retained | Dashboard private service-state parent; Native extension shared IPC root — system/host/config/soda.tmpfiles:2-2; current body read; consumer/selector is the service, rootfs copy or extension manifest named by the file; system/host/config/soda.tmpfiles:16-21; current body read; consumer/selector is the service, rootfs copy or extension manifest named by the file |
| 3; current configuration/service block | [F09](../../slices/factory-coordination.md#f09-publication-progression) | retained | Dashboard/private publication roots — system/host/config/soda.tmpfiles:3-3; current body read; consumer/selector is the service, rootfs copy or extension manifest named by the file |
| 4–5; current configuration/service block | [N01](../../slices/networking.md#n01-private-origins-tls-and-activation) | retained | Native forge/proxy persistent roots — system/host/config/soda.tmpfiles:4-5; current body read; consumer/selector is the service, rootfs copy or extension manifest named by the file |
| 6–15; current configuration/service block | [O01](../../slices/operator-administration.md#o01-first-boot-database-provisioning) | retained | PostgreSQL data, backup and socket roots — system/host/config/soda.tmpfiles:6-15; current body read; consumer/selector is the service, rootfs copy or extension manifest named by the file |
| 22–23; current configuration/service block | [I02](../../slices/identity-brokering.md#i02-encrypted-credential-custody) | retained | Broker home/private runtime roots — system/host/config/soda.tmpfiles:22-23; current body read; consumer/selector is the service, rootfs copy or extension manifest named by the file |
| 24; current configuration/service block | [N06](../../slices/networking.md#n06-project-companion-lifecycle) | retained | Companion private runtime root — system/host/config/soda.tmpfiles:24-24; current body read; consumer/selector is the service, rootfs copy or extension manifest named by the file |
| 25–26; current configuration/service block | [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) | retained | Private operator and factory runtime roots — system/host/config/soda.tmpfiles:25-26; current body read; consumer/selector is the service, rootfs copy or extension manifest named by the file |

<a id="coverage-d4613f4f5c25"></a>
<a id="coverage-1e2a7106265c"></a>

## [system/host/provisioning/base.json](../../../../../system/host/provisioning/base.json)

Current configuration/service body read and reconciled to line-bounded responsibility units

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–15, 19–22; current configuration/service block | [D11](../../slices/release-and-installation.md#d11-host-installation-and-payload-application) | retained | Disk installation and retained image import — system/host/provisioning/base.json:1-15,19-22; current body read; consumer/selector is the service, rootfs copy or extension manifest named by the file |
| 16–18; current configuration/service block | [O04](../../slices/operator-administration.md#o04-native-host-administration-and-updates) | retained | Boot-time native rpm-ostree extension provisioning and request marker; declarations/fields: `soda-extensions.service` — system/host/provisioning/base.json:16-18; current body read; consumer/selector is the service, rootfs copy or extension manifest named by the file |

<a id="coverage-32654a4b2cac"></a>
<a id="coverage-0a787ad15dff"></a>

## [system/host/services/forgejo.container](../../../../../system/host/services/forgejo.container)

Current configuration/service body read and reconciled to line-bounded responsibility units

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–21, 23–25, 32–40; current configuration/service block | [G02](../../slices/forgejo-integration.md#g02-background-service-admission) | retained | Background native service admission wiring — system/host/services/forgejo.container:1-21,23-25,32-40; current body read; consumer/selector is the service, rootfs copy or extension manifest named by the file |
| 22, 26–31; current configuration/service block | [O01](../../slices/operator-administration.md#o01-first-boot-database-provisioning) | retained | Native database password-file mount; Native PostgreSQL role/connection settings — system/host/services/forgejo.container:22-22; current body read; consumer/selector is the service, rootfs copy or extension manifest named by the file; system/host/services/forgejo.container:26-31; current body read; consumer/selector is the service, rootfs copy or extension manifest named by the file |

<a id="coverage-a74a2fbcc6f0"></a>
<a id="coverage-1cfac032d995"></a>

## [system/host/services/soda-dashboard.container](../../../../../system/host/services/soda-dashboard.container)

Current configuration/service body read and reconciled to line-bounded responsibility units

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–15, 20–25, 27, 29, 35–44; current configuration/service block | [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) | retained | Private IPC/service process lifetime; 4 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |
| 16–17; current configuration/service block | [O02](../../slices/operator-administration.md#o02-operator-identity-bootstrap) | retained | Operator configuration and grant-key mount — system/host/services/soda-dashboard.container:16-17; current body read; consumer/selector is the service, rootfs copy or extension manifest named by the file |
| 18–19; current configuration/service block | [H02](../../slices/shared-supporting-slices.md#h02-storage-mechanics) | retained | PostgreSQL input/socket mounts — system/host/services/soda-dashboard.container:18-19; current body read; consumer/selector is the service, rootfs copy or extension manifest named by the file |
| 26; current configuration/service block | [H04](../../slices/shared-supporting-slices.md#h04-configuration-and-filesystem-primitives) | retained | Shared protected dashboard state parent — system/host/services/soda-dashboard.container:26-26; current body read; consumer/selector is the service, rootfs copy or extension manifest named by the file |
| 28; current configuration/service block | [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) | retained | Dashboard/operator durable/private runtime mounts — system/host/services/soda-dashboard.container:28-28; current body read; consumer/selector is the service, rootfs copy or extension manifest named by the file |
| 30–34; current configuration/service block | [G02](../../slices/forgejo-integration.md#g02-background-service-admission) | retained | Native service callback peer and private operator wiring — system/host/services/soda-dashboard.container:30-34; current body read; consumer/selector is the service, rootfs copy or extension manifest named by the file |

<a id="coverage-7204b9c0af73"></a>
<a id="coverage-3ce544336cc7"></a>

## [system/host/services/soda-project@.service](../../../../../system/host/services/soda-project@.service)

Current configuration/service body read and reconciled to line-bounded responsibility units

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–8, 12–18; current configuration/service block | [P05](../../slices/projects.md#p05-project-startstop) | retained | Explicit Project Start/Stop confirmation and outcome — system/host/services/soda-project@.service:1-8,12-18; current body read; consumer/selector is the service, rootfs copy or extension manifest named by the file |
| 9; current configuration/service block | [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) | retained | Muse bind-only execution maintenance — system/host/services/soda-project@.service:9-9; current body read; consumer/selector is the service, rootfs copy or extension manifest named by the file |
| 10–11; current configuration/service block | [N06](../../slices/networking.md#n06-project-companion-lifecycle) | retained | Asynchronous companion lifecycle hook — system/host/services/soda-project@.service:10-11; current body read; consumer/selector is the service, rootfs copy or extension manifest named by the file |

<a id="coverage-fc99ee28b044"></a>

## [system/licenses/forgejo-LICENSE](../../../../../system/licenses/forgejo-LICENSE)

Inherited prior inventory row

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–674; lines 1–674 (all top-level items and imports) | [H05](../../slices/shared-supporting-slices.md#h05-branding-avatars-and-attribution) | retained | Current artifact-or-source scope; split declarations retain the path's release/install/qualification duty. — system/licenses/forgejo-LICENSE; prior ledger docs/development/ideal-filetree-plan/coverage/inventory/appliance.md row:  /  [system/licenses/forgejo-LICENSE](../../../../../system/licenses/forgejo-LICENSE)  /  documentation / active  /  [H05](../../slices/shared-supporting-slices.md#h05-branding-avatars-and-attribution)  /  |

## Removed predecessor navigation

These anchors identify earlier mapped source. Their paths no longer define this current map;
the current inventory and normalized predecessor ledger preserve replacement/retirement evidence.
No historical source body is counted as current coverage.

<a id="coverage-60a3ac729dc2"></a>
<a id="applianceforgejotemplatesadminauthedittmpl-1"></a>

Former source `appliance/forgejo/templates/admin/auth/edit.tmpl`; consult its pinned earlier Git source and the current coverage disposition.

<a id="coverage-49bbbf5cbe78"></a>
<a id="applianceforgejotemplatesreposettingsoptionstmpl-1"></a>

Former source `appliance/forgejo/templates/repo/settings/options.tmpl`; consult its pinned earlier Git source and the current coverage disposition.

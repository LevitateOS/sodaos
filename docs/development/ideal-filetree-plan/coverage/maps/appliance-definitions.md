# Appliance definitions

[Responsibility map index](README.md) · [Coverage snapshot and limits](../README.md).
Page grouping is navigation; the slice IDs retain their individual review ownership.

R02 reconciliation @HEAD `74a6a1b7` (post-C09): nine sections moved
`appliance/config/*`→`system/host/config/*`,
`appliance/provisioning/base.json`→`system/host/provisioning/base.json`,
`appliance/services/*`→`system/host/services/*`,
`appliance/soda-extension/extension.json`→`system/containers/extension/extension.json`.
Line ranges re-verified against moved files (contents intact); audit baseline
paths retained in this note. `appliance/forgejo/**` sections unmoved.

<a id="coverage-2ff14f94cf6d"></a>

## [system/host/config/forgejo.env](../../../../../system/host/config/forgejo.env)

Authored installed configuration/build input; service dependency wiring is not a second authority owner.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) / active | 1–14 | Maintained native Forgejo presentation and browser enhancement |
| [O04](../../slices/operator-administration.md#o04-native-host-administration-and-updates) / active | 15–19 | Native service diagnostic policy: console method/EscapedPath/status only, exclude request query/referrer fields |

<a id="coverage-d6987f7d8f3c"></a>

## [system/host/config/proxy.Caddyfile](../../../../../system/host/config/proxy.Caddyfile)

Authored installed configuration/build input; service dependency wiring is not a second authority owner.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [N01](../../slices/networking.md#n01-private-origins-tls-and-activation) / active | 1–8, 16–19 | Private origin, TLS, activation and proxy routing |
| [H05](../../slices/shared-supporting-slices.md#h05-branding-avatars-and-attribution) / active | 9–15 | Public avatar proxy strips browser credentials |

<a id="coverage-b5b1e03b7afd"></a>

## [system/host/config/soda.sysusers](../../../../../system/host/config/soda.sysusers)

Authored installed configuration/build input; service dependency wiring is not a second authority owner.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 1–2, 4 | Private IPC/service process lifetime |
| [I02](../../slices/identity-brokering.md#i02-encrypted-credential-custody) / active | 3 | Dedicated broker OS identity |

<a id="coverage-6f9dcc209ec6"></a>

## [system/host/config/soda.tmpfiles](../../../../../system/host/config/soda.tmpfiles)

Authored installed configuration/build input; service dependency wiring is not a second authority owner.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [H04](../../slices/shared-supporting-slices.md#h04-configuration-and-filesystem-primitives) / active | 1 | Configuration/filesystem primitives |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 2 | Dashboard private service-state parent |
| [F09](../../slices/factory-coordination.md#f09-publication-progression) / active | 3 | Dashboard/private publication roots |
| [N01](../../slices/networking.md#n01-private-origins-tls-and-activation) / active | 4–5 | Native forge/proxy persistent roots |
| [O01](../../slices/operator-administration.md#o01-first-boot-database-provisioning) / active | 6–15 | PostgreSQL data, backup and socket roots |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 16–21 | Native extension shared IPC root |
| [I02](../../slices/identity-brokering.md#i02-encrypted-credential-custody) / active | 22–23 | Broker home/private runtime roots |
| [N06](../../slices/networking.md#n06-project-companion-lifecycle) / active | 24 | Companion private runtime root |
| [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) / active | 25–26 | Private operator and factory runtime roots |

<a id="coverage-60a3ac729dc2"></a>

<a id="applianceforgejotemplatesadminauthedittmpl-1"></a>

## [appliance/forgejo/templates/admin/auth/edit.tmpl](../../../../../appliance/forgejo/templates/admin/auth/edit.tmpl)

Maintained native template, not generated code. Native Forgejo owns handler/session/data effects; G08 owns its customization surface.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) / active | 1–9, 424–449 | Maintained native Forgejo presentation and browser enhancement |
| [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) / active | 10–423 | Native form {{.Link}}; native handler/CSRF/authorization owns the effect |

<a id="coverage-49bbbf5cbe78"></a>

<a id="applianceforgejotemplatesreposettingsoptionstmpl-1"></a>

## [appliance/forgejo/templates/repo/settings/options.tmpl](../../../../../appliance/forgejo/templates/repo/settings/options.tmpl)

Maintained native template, not generated code. Native Forgejo owns handler/session/data effects; G08 owns its customization surface.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) / active | 1–7, 52–57, 69–77, 92–169, 174–177, 232–267, 278–287, 339–353, 391–399, 413, 461–507, 512–587, 606–617, 636–648, 671–685, 704–716, 735–748, 767–788, 803–808 | Maintained native Forgejo presentation and browser enhancement |
| [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) / active | 8–51 | Native form update; native handler/CSRF/authorization owns the effect |
| [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) / active | 58–68 | Native form {{.Link}}/avatar; native handler/CSRF/authorization owns the effect |
| [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) / active | 78–91 | Native form federation; native handler/CSRF/authorization owns the effect |
| [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) / active | 170–173 | Native form mirror-sync; native handler/CSRF/authorization owns the effect |
| [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) / active | 178–231 | Native form mirror; native handler/CSRF/authorization owns the effect |
| [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) / active | 268–272 | Native form push-mirror-sync; native handler/CSRF/authorization owns the effect |
| [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) / active | 273–277 | Native form push-mirror-remove; native handler/CSRF/authorization owns the effect |
| [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) / active | 288–338 | Native form push-mirror-add; native handler/CSRF/authorization owns the effect |
| [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) / active | 354–390 | Native form signing; native handler/CSRF/authorization owns the effect |
| [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) / active | 400–412 | Native form admin; native handler/CSRF/authorization owns the effect |
| [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) / active | 414–460 | Native form admin_index; native handler/CSRF/authorization owns the effect |
| [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) / active | 508–511 | Native form cancel_transfer; native handler/CSRF/authorization owns the effect |
| [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) / active | 588–605 | Native form convert; native handler/CSRF/authorization owns the effect |
| [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) / active | 618–635 | Native form convert_fork; native handler/CSRF/authorization owns the effect |
| [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) / active | 649–670 | Native form transfer; native handler/CSRF/authorization owns the effect |
| [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) / active | 686–703 | Native form delete; native handler/CSRF/authorization owns the effect |
| [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) / active | 717–734 | Native form delete-wiki; native handler/CSRF/authorization owns the effect |
| [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) / active | 749–766 | Native form rename-wiki-branch; native handler/CSRF/authorization owns the effect |
| [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) / active | 789–802 | Native form {{if .Repository.IsArchived}}unarchive{{else}}archive{{end}}; native handler/CSRF/authorization owns the effect |

<a id="coverage-d4613f4f5c25"></a>

## [system/host/provisioning/base.json](../../../../../system/host/provisioning/base.json)

Authored installed configuration/build input; service dependency wiring is not a second authority owner.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [D11](../../slices/release-and-installation.md#d11-host-installation-and-payload-application) / active | 1–15, 19–22 | Disk installation and retained image import |
| [O04](../../slices/operator-administration.md#o04-native-host-administration-and-updates) / active | 16–18 | Boot-time native rpm-ostree extension provisioning and request marker; declarations/fields: `soda-extensions.service` |

<a id="coverage-32654a4b2cac"></a>

## [system/host/services/forgejo.container](../../../../../system/host/services/forgejo.container)

Authored installed configuration/build input; service dependency wiring is not a second authority owner.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [G02](../../slices/forgejo-integration.md#g02-background-service-admission) / active | 1–21, 23–25, 32–40 | Background native service admission wiring |
| [O01](../../slices/operator-administration.md#o01-first-boot-database-provisioning) / active | 22 | Native database password-file mount |
| [O01](../../slices/operator-administration.md#o01-first-boot-database-provisioning) / active | 26–31 | Native PostgreSQL role/connection settings |

<a id="coverage-a74a2fbcc6f0"></a>

## [system/host/services/soda-dashboard.container](../../../../../system/host/services/soda-dashboard.container)

Authored installed configuration/build input; service dependency wiring is not a second authority owner.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 1–15, 20–23, 29, 36–44 | Private IPC/service process lifetime |
| [O02](../../slices/operator-administration.md#o02-operator-identity-bootstrap) / active | 16–17 | Operator configuration and grant-key mount |
| [H02](../../slices/shared-supporting-slices.md#h02-storage-mechanics) / active | 18–19 | PostgreSQL input/socket mounts |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 24–25 | Host and broker administration IPC mounts; dashboard is not a runtime credential consumer |
| [H04](../../slices/shared-supporting-slices.md#h04-configuration-and-filesystem-primitives) / active | 26 | Shared protected dashboard state parent |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 27 | Native extension IPC mount |
| [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) / active | 28 | Dashboard/operator durable/private runtime mounts |
| [G02](../../slices/forgejo-integration.md#g02-background-service-admission) / active | 30–34 | Native service callback peer and private operator wiring |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 35 | Native extension/private operator listener entrypoint |

<a id="coverage-7204b9c0af73"></a>

## [system/host/services/soda-project@.service](../../../../../system/host/services/soda-project@.service)

Authored installed configuration/build input; service dependency wiring is not a second authority owner.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [P05](../../slices/projects.md#p05-project-startstop) / active | 1–8, 12–18 | Explicit Project Start/Stop confirmation and outcome |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 9 | Muse bind-only execution maintenance |
| [N06](../../slices/networking.md#n06-project-companion-lifecycle) / active | 10–11 | Asynchronous companion lifecycle hook |

<a id="coverage-21ba044ba45b"></a>

## [system/containers/extension/extension.json](../../../../../system/containers/extension/extension.json)

Authored installed configuration/build input; service dependency wiring is not a second authority owner.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [G01](../../slices/forgejo-integration.md#g01-browser-authority-and-contributions) / active | 1–7, 10, 14–26 | Native extension context and browser contribution admission |
| [S02](../../slices/spaces-and-terminals.md#s02-workspace-lifetime-and-navigation) / active | 8 | Spaces page contribution |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 9 | Operator Tailnet page contribution |
| [S02](../../slices/spaces-and-terminals.md#s02-workspace-lifetime-and-navigation) / active | 11–13 | Persistent Workspace panel contribution |


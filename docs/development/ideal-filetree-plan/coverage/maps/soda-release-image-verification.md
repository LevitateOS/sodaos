# Soda release image verification

[Responsibility map index](README.md) · [Coverage snapshot and limits](../README.md).
Page grouping is navigation; the slice IDs retain their individual review ownership.

R02 @HEAD `d5012d10` (C08): oracle.rs verified byte-identical move; intervals kept.

<a id="coverage-16cef9a7838a"></a>

<a id="rustsoda-release-imagetestsoraclers-1"></a>

## [lib/soda-release-image/tests/oracle.rs](../../../../../lib/soda-release-image/tests/oracle.rs)

Source assertions/fixtures were inspected for mapping only; no test execution, runtime or installed proof claimed. Large-file ledger retains committed declaration/member spans rather than grouping methods into one whole-file unit; responsibility follows the verified concern/branch mapping, not declaration count.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 1–10, 116–185, 219–236, 292–324, 393–418, 649–684 | Complete native candidate image assembly and authenticated live media; declarations/fields: `b64`, `check_ok`, `check_err`, `oracle_local_quadlet`, `oracle_package_inputs`, `oracle_rpm_inventory`, `oracle_extension_asset_names`, `oracle_candidate_live_config` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 11–14 | Complete native candidate image assembly and authenticated live media; declaration/member b64; declarations/fields: `b64` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 15–18 | Complete native candidate image assembly and authenticated live media; declaration/member check_ok; declarations/fields: `check_ok` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 19–23 | Complete native candidate image assembly and authenticated live media; declaration/member check_err; declarations/fields: `check_err` |
| [D04](../../slices/release-and-installation.md#d04-authenticated-installation-media) / active | 24–115, 186–218, 237–291, 325–392 | Source assertion of Authenticated installation media; declarations/fields: `oracle_media_base_url`, `oracle_live_ignition`, `oracle_media_compression`, `oracle_media_log_events` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 419–442 | Complete native candidate image assembly and authenticated live media; declaration/member oracle_qualify_reason; declarations/fields: `oracle_qualify_reason` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 443–444 | Complete native candidate image assembly and authenticated live media; declaration/member Stub; declarations/fields: `Stub` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 445–447 | Complete native candidate image assembly and authenticated live media; declaration/member source; declarations/fields: `source` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 448–450 | Complete native candidate image assembly and authenticated live media; declaration/member forgejo_source; declarations/fields: `forgejo_source` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 451–453 | Complete native candidate image assembly and authenticated live media; declaration/member forgejo_revision; declarations/fields: `forgejo_revision` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 454–456 | Complete native candidate image assembly and authenticated live media; declaration/member native; declarations/fields: `native` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 457–459 | Complete native candidate image assembly and authenticated live media; declaration/member out; declarations/fields: `out` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 460–462 | Complete native candidate image assembly and authenticated live media; declaration/member arch; declarations/fields: `arch` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 463–465 | Complete native candidate image assembly and authenticated live media; declaration/member revision; declarations/fields: `revision` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 466–468 | Complete native candidate image assembly and authenticated live media; declaration/member live_inputs; declarations/fields: `live_inputs` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 469–476 | Complete native candidate image assembly and authenticated live media; declaration/member execute; declarations/fields: `execute` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 477–484 | Complete native candidate image assembly and authenticated live media; declaration/member capture; declarations/fields: `capture` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 485–487 | Complete native candidate image assembly and authenticated live media; declaration/member next; declarations/fields: `next` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 488–490 | Complete native candidate image assembly and authenticated live media; declaration/member resolve_inputs; declarations/fields: `resolve_inputs` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 491–493 | Complete native candidate image assembly and authenticated live media; declaration/member dependencies; declarations/fields: `dependencies` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 494–496 | Complete native candidate image assembly and authenticated live media; declaration/member compile; declarations/fields: `compile` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 497–504 | Complete native candidate image assembly and authenticated live media; declaration/member compile_rust; declarations/fields: `compile_rust` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 505–507 | Complete native candidate image assembly and authenticated live media; declaration/member stage_fork_binary; declarations/fields: `stage_fork_binary` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 508–510 | Complete native candidate image assembly and authenticated live media; declaration/member assets; declarations/fields: `assets` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 511–519 | Complete native candidate image assembly and authenticated live media; declaration/member images; declarations/fields: `images` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 520–527 | Complete native candidate image assembly and authenticated live media; declaration/member inspect_oci; declarations/fields: `inspect_oci` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 528–535 | Complete native candidate image assembly and authenticated live media; declaration/member verify_content; declarations/fields: `verify_content` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 536–538 | Complete native candidate image assembly and authenticated live media; declaration/member resolve_core_os; declarations/fields: `resolve_core_os` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 539–544 | Complete native candidate image assembly and authenticated live media; declaration/member read_live_inputs; declarations/fields: `read_live_inputs` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 545–547 | Complete native candidate image assembly and authenticated live media; declaration/member check_native; declarations/fields: `check_native` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 548–559 | Complete native candidate image assembly and authenticated live media; declaration/member sign_media; declarations/fields: `sign_media` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 560–569 | Complete native candidate image assembly and authenticated live media; declaration/member verify_copy; declarations/fields: `verify_copy` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 570–579 | Complete native candidate image assembly and authenticated live media; declaration/member write_document; declarations/fields: `write_document` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 580–612 | Complete native candidate image assembly and authenticated live media; declaration/member oracle_stage_layout; declarations/fields: `oracle_stage_layout` |
| [D05](../../slices/release-and-installation.md#d05-artifact-verification) / active | 613–648 | Source assertion of Artifact verification; declarations/fields: `oracle_rootfs_chunks` |


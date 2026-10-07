# Server entrypoints

[Responsibility map index](README.md) · [Coverage scope](../README.md).

Current responsibility accounting at `519b76bd` (2026-10-07).
One slice owns each named duty; disjoint complete symbols may share an owner.
Compound fields/clauses may share a physical line with distinct selectors.
Disposition concerns the duty, not source validity or installed qualification.

<a id="coverage-72702733d2bf"></a>

## [cmd/soda-dashboard/extension.go](../../../../../cmd/soda-dashboard/extension.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–53; file scaffold; extensionSocketLive; extensionListener | [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) | retained | Current scaffold duty: file scaffold; Current declaration duty: extensionSocketLive; Current declaration duty: extensionListener — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-360556c13d40"></a>

## [cmd/soda-dashboard/extension_test.go](../../../../../cmd/soda-dashboard/extension_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–75; file scaffold; TestExtensionListenerUsesPrivateSocketWithoutReplacingFiles; TestExtensionListenerReclaimsStaleSocket | [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) | retained | Current scaffold duty: file scaffold; Current declaration duty: TestExtensionListenerUsesPrivateSocketWithoutReplacingFiles; Current declaration duty: TestExtensionListenerReclaimsStaleSocket — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-ec73c0fae138"></a>

## [cmd/soda-dashboard/main.go](../../../../../cmd/soda-dashboard/main.go)

Current defining methods and mixed responsibilities inspected at `71cfb075` (2026-10-08); the preceding snapshot is retained for unrelated entries.

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–28, 33–37, 44–79, 95–105, 128–154, 156–159, 161, 163–191; scaffold, main, run, listener error reporting, extensionHTTPServer, serveExtensionSocket, beginShutdown, shutdownServers | [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) | retained | Public/private listener ownership; join shutdown results, preserve Serve/Shutdown/Close failures and force transport closure after the shared HTTP deadline. Actual dashboard checks pass; no installed service qualification. |
| 29, 110–113; run configuration flag and openDashboard/config.Load | [H04](../../slices/shared-supporting-slices.md#h04-configuration-and-filesystem-primitives) | retained | Load configured service inputs under the existing configuration profile. |
| 30; run extension-socket flag | [G01](../../slices/forgejo-integration.md#g01-browser-authority-and-contributions) | retained | Optional private native-extension endpoint selection. |
| 31–32, 39–43, 81–94, 155, 162; StartCoordinator, admission wrappers, operatorEndpoint, StopAdmission, CloseCoordinator | [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) | retained | Require an explicit OS operator principal; retain coordinator ownership through admitted callbacks and startup/error/normal shutdown. |
| 38, 118–121; run database.Close and openDashboard/config.Secret | [H02](../../slices/shared-supporting-slices.md#h02-storage-mechanics) | retained | Read the restricted PostgreSQL DSN and preserve database-close failures after coordinator retirement. |
| 106–109; openDashboard/avatar.Validate | [H05](../../slices/shared-supporting-slices.md#h05-branding-avatars-and-attribution) | retained | Validate canonical avatar definitions before serving. |
| 114–117, 122–127; openDashboard/config.GrantKey and store.OpenEncrypted | [I02](../../slices/identity-brokering.md#i02-encrypted-credential-custody) | retained | Read the provisioned encryption key and open the existing encrypted Store. |
| 160; beginShutdown/CloseTerminals | [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) | retained | Cancel and join tracked hijacked terminal streams before releasing Store/coordinator ownership. |

<a id="coverage-dashboard-main-test"></a>

## [cmd/soda-dashboard/main_test.go](../../../../../cmd/soda-dashboard/main_test.go)

Current source test declaration and local listener behavior inspected.

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–56; `TestShutdownServersClosesConnectionsAndReturnsTimeout` | [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) | retained | Exercise shutdown of an admitted held HTTP connection, confirm deadline propagation, then release and join the request and server goroutines. — Included in the seven-check dashboard ownership packet |

<a id="coverage-e446e6589b09"></a>

## [cmd/soda-dashboard/operator.go](../../../../../cmd/soda-dashboard/operator.go)

exact-blob current maintained map; full spans retained

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–18; whole file | [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) | retained | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 19–49; operatorHTTPServer | [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) | retained | Admitted operator-only factory command endpoint; declarations/fields: `operatorHTTPServer` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |

<a id="coverage-370aad109b4e"></a>

## [cmd/soda-dashboard/operator_linux.go](../../../../../cmd/soda-dashboard/operator_linux.go)

exact-blob current maintained map; full spans retained

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–10; whole file | [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) | retained | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 11–58; operatorPeerListener, gateOperatorPeers, operatorPeerListener.Accept, operatorPeerOK | [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) | retained | Operator principal boundary for private factory commands; declarations/fields: `operatorPeerListener`, `gateOperatorPeers`, `operatorPeerListener.Accept`, `operatorPeerOK` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |

<a id="coverage-eb2c8e223d64"></a>

## [cmd/soda-dashboard/operator_linux_test.go](../../../../../cmd/soda-dashboard/operator_linux_test.go)

exact-blob current maintained map; full spans retained

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–17; whole file | [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) | retained | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 18–74; operatorRoundTrip, TestOperatorSocketAdmitsConfiguredPeer, TestOperatorSocketRebindsStalePath, TestOperatorSocketRefusesForeignPeer | [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) | retained | Operator principal boundary for private factory commands; declarations/fields: `operatorRoundTrip`, `TestOperatorSocketAdmitsConfiguredPeer`, `TestOperatorSocketRebindsStalePath`, `TestOperatorSocketRefusesForeignPeer` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |

<a id="coverage-978604ee0fa6"></a>

## [cmd/soda-dashboard/operator_other.go](../../../../../cmd/soda-dashboard/operator_other.go)

exact-blob current maintained map; full spans retained

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–9; whole file | [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) | retained | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 10–12; gateOperatorPeers | [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) | retained | Operator principal boundary for private factory commands; declarations/fields: `gateOperatorPeers` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |

<a id="coverage-6105744cc963"></a>

## [cmd/soda-dashboard/operator_test.go](../../../../../cmd/soda-dashboard/operator_test.go)

exact-blob current maintained map; full spans retained

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–9; whole file | [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) | retained | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 10–19; TestOperatorEndpointRequiresExplicitPeer | [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) | retained | Operator principal boundary for private factory commands; declarations/fields: `TestOperatorEndpointRequiresExplicitPeer` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |

<a id="coverage-81292ae984ec"></a>

## [cmd/soda-dashboard/postgres_fixture_test.go](../../../../../cmd/soda-dashboard/postgres_fixture_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–24; file scaffold; postgresFixture | [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) | retained | Current scaffold duty: file scaffold; Current declaration duty: postgresFixture — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-0c5003e81409"></a>

## [cmd/soda-extension/main.go](../../../../../cmd/soda-extension/main.go)

exact-blob current maintained map; full spans retained

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–41, 58–62, 72–95; whole file; run; operatorIDFileAvailable; contributionAuthorizer; allowedContribution; main | [G01](../../slices/forgejo-integration.md#g01-browser-authority-and-contributions) | retained | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility; 6 named units assigned here; remaining selectors preserve each duty — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 42–57, 63–71; readOperatorID, canonicalOperatorID | [H04](../../slices/shared-supporting-slices.md#h04-configuration-and-filesystem-primitives) | retained | Restricted configured operator identity file boundary; declarations/fields: `readOperatorID`, `canonicalOperatorID` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |

<a id="coverage-dae71a5724d8"></a>

## [cmd/soda-extension/main_test.go](../../../../../cmd/soda-extension/main_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–77; file scaffold; TestReadOperatorIDRequiresPrivateCanonicalIdentity; TestSodaContributionPolicyKeepsOperatorPagesNarrow | [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) | retained | Current scaffold duty: file scaffold; Current declaration duty: TestReadOperatorIDRequiresPrivateCanonicalIdentity; Current declaration duty: TestSodaContributionPolicyKeepsOperatorPagesNarrow — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-b55d6e144ade"></a>

## [cmd/soda-forgejo-tailnet/main.rs](../../../../../cmd/soda-forgejo-tailnet/main.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–14; lines 1–7: mod forgejo and attached body; lines 8–14: fn main and attached body | [D03](../../slices/release-and-installation.md#d03-candidate-production) | retained | Declaration block for mod forgejo in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; Declaration block for fn main in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context. — cmd/soda-forgejo-tailnet/main.rs:1-7; current source declaration and body; cmd/soda-forgejo-tailnet/main.rs:8-14; current source declaration and body |

<a id="coverage-8d7acd4446f3"></a>

## [cmd/soda-install/src/buildx.rs](../../../../../cmd/soda-install/src/buildx.rs)

Inherited prior inventory row; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–253; lines 1–5: use std and attached body; lines 6–6: use std and attached body; lines 7–8: use crate and attached body; lines 9–10: const JSON_LIMIT and attached body; lines 11–17: fn oci_architecture and attached body; lines 18–21: fn digest and attached body; lines 22–25: fn revision and attached body; lines 26–29: fn file_is_regular and attached body; lines 30–37: fn hash_file and attached body; lines 38–51: use sha2 and attached body; lines 52–53: fn hex_encode and attached body; lines 54–61: const HEX and attached body; lines 62–63: fn sha256_hex and attached body; lines 64–68: use sha2 and attached body; lines 69–69: fn write_new and attached body; lines 70–82: use std and attached body; lines 83–100: use std and attached body; lines 101–108: fn open_dir_nofollow and attached body; lines 109–120: fn open_at_file and attached body; lines 121–123: use std and attached body; lines 124–125: use std and attached body; lines 126–139: fn fstatat_nofollow and attached body; lines 140–146: fn same_file and attached body; lines 147–160: fn read_json_bytes and attached body; lines 161–194: use std and attached body; lines 195–195: mod tests and attached body; lines 196–198: use super and attached body; lines 199–212: fn shapes_and_arch and attached body; lines 213–239: fn hash_and_write_new and attached body; lines 240–253: fn confined_json_reads and attached body | [H04](../../slices/shared-supporting-slices.md#h04-configuration-and-filesystem-primitives) | retained | Declaration block for use std in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 30 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-402abdae95f9"></a>

## [cmd/soda-install/src/candidate.rs](../../../../../cmd/soda-install/src/candidate.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–155; lines 1–6: use crate and attached body; lines 7–7: use crate and attached body; lines 8–8: use crate and attached body; lines 9–9: use crate and attached body; lines 10–10: use crate and attached body; lines 11–11: use crate and attached body; lines 12–13: const CANDIDATE_INSTALLER_BINARY and attached body; lines 14–24: fn candidate_identity_matches and attached body; lines 25–31: fn candidate_file_matches and attached body; lines 32–47: fn candidate_release_matches and attached body; lines 48–70: fn candidate_requirement and attached body; lines 71–76: struct MachineDefaults and attached body; lines 77–102: fn rewrite_candidate_host_file and attached body; lines 103–154: fn candidate_destination and attached body; lines 155–155: mod tests and attached body | [D11](../../slices/release-and-installation.md#d11-host-installation-and-payload-application) | retained | Declaration block for use crate in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 15 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-509d0fb0f434"></a>

## [cmd/soda-install/src/candidate/tests.rs](../../../../../cmd/soda-install/src/candidate/tests.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–220; lines 1–1: use super and attached body; lines 2–2: use crate and attached body; lines 3–3: use std and attached body; lines 4–88: fn candidate_root_fixture and attached body; lines 89–124: fn requirement_authenticates_all_images and attached body; lines 125–220: fn destination_keeps_password_only_provisioning and attached body | [D11](../../slices/release-and-installation.md#d11-host-installation-and-payload-application) | retained | Declaration block for use super in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 6 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-e57151376c7c"></a>

## [cmd/soda-install/src/command.rs](../../../../../cmd/soda-install/src/command.rs)

Current cohesive body inspected; single defining duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–401; whole file: bounded privileged command runner used by the interactive installer; owns command argv execution, timeout/cancellation, output limits, and exit mapping | [D11](../../slices/release-and-installation.md#d11-host-installation-and-payload-application) | retained | Bounded privileged command runner used by the interactive installer; owns command argv execution, timeout/cancellation, output limits, and exit mapping. — cmd/soda-install/src/command.rs module docs and RealRunner; consumers are installer run/execute modules |

<a id="coverage-3ae2cc6a01b6"></a>

## [cmd/soda-install/src/console/mod.rs](../../../../../cmd/soda-install/src/console/mod.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–109; lines 1–4: use std and attached body; lines 5–13: struct Console and attached body; lines 14–36: fn go_space_len and attached body; lines 37–75: fn trim_space_bytes and attached body; lines 76–85: fn errno_name and attached body; lines 86–100: fn errno_text and attached body; lines 101–102: mod network and attached body; lines 103–105: mod terminal and attached body; lines 106–108: mod test_support and attached body; lines 109–109: mod tests and attached body | [D11](../../slices/release-and-installation.md#d11-host-installation-and-payload-application) | retained | Declaration block for use std in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 10 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-d0b626d58601"></a>

## [cmd/soda-install/src/console/network.rs](../../../../../cmd/soda-install/src/console/network.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–171; lines 1–1: use std and attached body; lines 2–2: use std and attached body; lines 3–4: use crate and attached body; lines 5–5: use crate and attached body; lines 6–6: use crate and attached body; lines 7–8: use super and attached body; lines 9–10: impl Console and attached body; lines 11–16: fn choose_network_action and attached body; lines 17–55: fn run_network_editor and attached body; lines 56–64: fn reopen_stdio and attached body; lines 65–70: fn from_file and attached body; lines 71–84: fn confirm_live_addresses and attached body; lines 85–94: fn print_live_addresses and attached body; lines 95–114: fn apply_network_choice and attached body; lines 115–138: fn inspect_and_confirm_network and attached body; lines 139–171: fn network_with and attached body | [D11](../../slices/release-and-installation.md#d11-host-installation-and-payload-application) | retained | Declaration block for use std in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 16 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-61f6fb7aeb94"></a>

## [cmd/soda-install/src/console/terminal.rs](../../../../../cmd/soda-install/src/console/terminal.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–208; lines 1–2: use std and attached body; lines 3–3: use std and attached body; lines 4–4: use std and attached body; lines 5–6: use crate and attached body; lines 7–7: use crate and attached body; lines 8–9: use super and attached body; lines 10–12: impl Console and attached body; lines 13–23: fn open and attached body; lines 24–29: fn fd and attached body; lines 30–34: fn tty_fd and attached body; lines 35–39: fn print and attached body; lines 40–60: fn tty_write and attached body; lines 61–67: fn page and attached body; lines 68–72: fn ask and attached body; lines 73–95: fn poll and attached body; lines 96–116: fn read_byte and attached body; lines 117–142: fn line and attached body; lines 143–158: fn hide_echo and attached body; lines 159–170: fn restore_echo and attached body; lines 171–208: fn secret and attached body | [D11](../../slices/release-and-installation.md#d11-host-installation-and-payload-application) | retained | Declaration block for use std in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 20 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-a697ee5f6749"></a>

## [cmd/soda-install/src/console/test_support.rs](../../../../../cmd/soda-install/src/console/test_support.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–106; lines 1–1: use std and attached body; lines 2–2: use std and attached body; lines 3–3: use std and attached body; lines 4–8: struct Pty and attached body; lines 9–41: fn open_pty and attached body; lines 42–66: fn poll_read and attached body; lines 67–86: fn read_until and attached body; lines 87–106: fn drain_idle and attached body | [D11](../../slices/release-and-installation.md#d11-host-installation-and-payload-application) | retained | Declaration block for use std in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 8 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-9bda78b32af4"></a>

## [cmd/soda-install/src/console/tests.rs](../../../../../cmd/soda-install/src/console/tests.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–107; lines 1–1: use super and attached body; lines 2–2: use super and attached body; lines 3–3: use crate and attached body; lines 4–4: use crate and attached body; lines 5–5: use crate and attached body; lines 6–8: use std and attached body; lines 9–37: fn ask_trims_and_refuses_controls and attached body; lines 38–66: fn secret_hides_input_and_restores_echo and attached body; lines 67–75: fn cancelled_line_returns_ctx_error and attached body; lines 76–107: fn network_step_flows and attached body | [D11](../../slices/release-and-installation.md#d11-host-installation-and-payload-application) | retained | Declaration block for use super in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 10 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-5bfe0a36f02c"></a>

## [cmd/soda-install/src/deliver.rs](../../../../../cmd/soda-install/src/deliver.rs)

Current cohesive body inspected; single defining duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–432; whole file: installer payload manifest and image inventory decoding/admission; confirms release identity, host version, image names and digest bindings before import | [D11](../../slices/release-and-installation.md#d11-host-installation-and-payload-application) | retained | Installer payload manifest and image inventory decoding/admission; confirms release identity, host version, image names and digest bindings before import. — cmd/soda-install/src/deliver.rs; consumed by installer payload import |

<a id="coverage-1058494584bb"></a>

## [cmd/soda-install/src/deliver/tests.rs](../../../../../cmd/soda-install/src/deliver/tests.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–195; lines 1–1: use super and attached body; lines 2–39: fn fixture and attached body; lines 40–84: fn payload_validation_matrix and attached body; lines 85–95: fn load_refuses_trailing_data and attached body; lines 96–118: fn payload_chooses_raw_alias_before_type_conversion and attached body; lines 119–131: fn payload_validates_each_duplicate_image_value and attached body; lines 132–139: fn payload_json and attached body; lines 140–177: fn payload_fixture and attached body; lines 178–195: fn verify_content_binds_layout and attached body | [D11](../../slices/release-and-installation.md#d11-host-installation-and-payload-application) | retained | Declaration block for use super in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 9 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-89671170eb0f"></a>

## [cmd/soda-install/src/disks/tests.rs](../../../../../cmd/soda-install/src/disks/tests.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–204; lines 1–1: use super and attached body; lines 2–2: use crate and attached body; lines 3–24: fn device and attached body; lines 25–63: fn unused_taxonomy and attached body; lines 64–85: fn live_media_parsing and attached body; lines 86–109: fn parent_disk_sysfs_walk and attached body; lines 110–135: fn summary_and_same_disk and attached body; lines 136–144: fn summary_escapes_external_terminal_controls and attached body; lines 145–193: fn scan_decodes_and_blocks and attached body; lines 194–204: fn lsblk_size_requires_unsigned_integer_token_but_accepts_negative_zero and attached body | [D11](../../slices/release-and-installation.md#d11-host-installation-and-payload-application) | retained | Declaration block for use super in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 10 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-0db284564e6d"></a>

## [cmd/soda-install/src/enroll/arm.rs](../../../../../cmd/soda-install/src/enroll/arm.rs)

Current cohesive body inspected; single defining duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–617; whole file: operator ssh enrollment arming and private enrollment window state, with address/fingerprint confirmation and protected expiry storage | [O07](../../slices/operator-administration.md#o07-operator-ssh-enrollment) | retained | Operator SSH enrollment arming and private enrollment window state, with address/fingerprint confirmation and protected expiry storage. — cmd/soda-install/src/enroll/arm.rs; consumed by enrollment action dispatch |

<a id="coverage-26a3b0e74f23"></a>

## [cmd/soda-install/src/enroll/arm_tests.rs](../../../../../cmd/soda-install/src/enroll/arm_tests.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–285; lines 1–1: use super and attached body; lines 2–2: use super and attached body; lines 3–3: use super and attached body; lines 4–5: use crate and attached body; lines 6–6: use crate and attached body; lines 7–7: use crate and attached body; lines 8–10: use crate and attached body; lines 11–30: fn enrollment_choice_matrix and attached body; lines 31–58: fn enrollment_state_publishes_whole_and_preserves_existing and attached body; lines 59–77: fn guard_runner and attached body; lines 78–111: fn guard_template_absent and attached body; lines 112–149: fn guard_template_refused and attached body; lines 150–150: fn address_selection_retries_typos and attached body; lines 151–194: use std and attached body; lines 195–203: fn direct_server_refused and attached body; lines 204–222: fn receive_commands_refused_before_state and attached body; lines 223–241: fn pty_proceeds_past_console_check and attached body; lines 242–260: fn enrollment_write_requires_complete_count and attached body; lines 261–285: fn enrollment_live_binding_requires_current_interface_address and attached body | [D11](../../slices/release-and-installation.md#d11-host-installation-and-payload-application) | retained | Declaration block for use super in the current test-or-fixture source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 19 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-b223a65b18b3"></a>

## [cmd/soda-install/src/enroll/keys/authorized_keys.rs](../../../../../cmd/soda-install/src/enroll/keys/authorized_keys.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–329; lines 1–1: use std and attached body; lines 2–2: use std and attached body; lines 3–3: use std and attached body; lines 4–4: use std and attached body; lines 5–5: use std and attached body; lines 6–9: use super and attached body; lines 10–10: use super and attached body; lines 11–11: use crate and attached body; lines 12–12: use crate and attached body; lines 13–25: fn has_authorized_key and attached body; lines 26–45: fn inspect_existing_authorized_keys and attached body; lines 46–70: fn verify_authorized_keys_unchanged and attached body; lines 71–105: fn append_existing_key and attached body; lines 106–141: fn write_temp_key_file and attached body; lines 142–171: fn link_and_confirm_key_file and attached body; lines 172–176: struct TempKey and attached body; lines 177–178: impl Drop and attached body; lines 179–200: fn drop and attached body; lines 201–223: fn create_exclusive_key_file and attached body; lines 224–252: fn append_enrollment_key_with_writer and attached body; lines 253–274: fn confirm_key_file_contents and attached body; lines 275–293: fn validate_key_file_match and attached body; lines 294–320: fn confirm_key_file_stats and attached body; lines 321–329: fn enrollment_confirm_key_file and attached body | [D11](../../slices/release-and-installation.md#d11-host-installation-and-payload-application) | retained | Declaration block for use std in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 24 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-a369030d29e5"></a>

## [cmd/soda-install/src/enroll/keys/directory.rs](../../../../../cmd/soda-install/src/enroll/keys/directory.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–142; lines 1–1: use std and attached body; lines 2–2: use std and attached body; lines 3–3: use std and attached body; lines 4–5: use crate and attached body; lines 6–9: fn errno_error and attached body; lines 10–17: fn fstat_fd and attached body; lines 18–19: fn open_dir_nofollow and attached body; lines 20–20: use rustix and attached body; lines 21–41: use std and attached body; lines 42–57: fn openat_file and attached body; lines 58–60: struct SshDir and attached body; lines 61–62: impl SshDir and attached body; lines 63–66: fn raw and attached body; lines 67–68: impl Drop and attached body; lines 69–72: fn drop and attached body; lines 73–102: fn open_and_lock_ssh_directory and attached body; lines 103–116: fn validate_authorized_keys_stat and attached body; lines 117–127: fn enrollment_safe_directory and attached body; lines 128–142: fn enrollment_root_home and attached body | [D11](../../slices/release-and-installation.md#d11-host-installation-and-payload-application) | retained | Declaration block for use std in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 19 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-e6cd4820c41f"></a>

## [cmd/soda-install/src/enroll/keys/mod.rs](../../../../../cmd/soda-install/src/enroll/keys/mod.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–52; lines 1–4: mod authorized_keys and attached body; lines 5–6: mod directory and attached body; lines 7–7: mod tests and attached body; lines 8–9: use std and attached body; lines 10–10: use std and attached body; lines 11–12: use crate and attached body; lines 13–13: use crate and attached body; lines 14–15: use authorized_keys and attached body; lines 16–19: use directory and attached body; lines 20–33: fn single_write and attached body; lines 34–37: fn append_enrollment_key and attached body; lines 38–43: fn random_hex and attached body; lines 44–52: fn random_hex_with and attached body | [D11](../../slices/release-and-installation.md#d11-host-installation-and-payload-application) | retained | Declaration block for mod authorized_keys in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 13 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-9ba8a0ac5691"></a>

## [cmd/soda-install/src/enroll/keys/tests.rs](../../../../../cmd/soda-install/src/enroll/keys/tests.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–349; lines 1–1: use super and attached body; lines 2–2: use super and attached body; lines 3–3: use super and attached body; lines 4–4: use std and attached body; lines 5–7: use std and attached body; lines 8–17: fn random_hex_entropy_failure_returns_without_issuing_a_value and attached body; lines 18–24: fn make_ssh_dir and attached body; lines 25–30: fn test_ctx and attached body; lines 31–70: fn preserves_authorized_keys and attached body; lines 71–85: fn creates_authorized_keys and attached body; lines 86–132: fn failed_new_key_write_cleans_only_the_owned_temporary_inode and attached body; lines 133–206: fn refuses_unsafe_key_paths and attached body; lines 207–218: fn cancelled_append_preserves_key and attached body; lines 219–234: fn concurrent_writer_preserves_key and attached body; lines 235–285: fn native_editor_race_preserves_newer_file and attached body; lines 286–312: fn concurrent_creation_is_never_replaced and attached body; lines 313–341: fn partial_append_does_not_roll_back_native_data and attached body; lines 342–349: use std and attached body | [D11](../../slices/release-and-installation.md#d11-host-installation-and-payload-application) | retained | Declaration block for use super in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 18 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-332f68bd4871"></a>

## [cmd/soda-install/src/enroll/mod.rs](../../../../../cmd/soda-install/src/enroll/mod.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–38; lines 1–6: mod arm and attached body; lines 7–7: mod arm_tests and attached body; lines 8–8: mod keys and attached body; lines 9–9: mod native_config and attached body; lines 10–10: mod peer and attached body; lines 11–11: mod receive and attached body; lines 12–12: mod selinux and attached body; lines 13–13: mod serve and attached body; lines 14–15: mod session and attached body; lines 16–17: mod test_support and attached body; lines 18–18: mod tests and attached body; lines 19–24: use native_config and attached body; lines 25–25: use peer and attached body; lines 26–26: use receive and attached body; lines 27–27: use serve and attached body; lines 28–28: use session and attached body; lines 29–30: const ENROLLMENT_DIR and attached body; lines 31–31: const ENROLLMENT_UNIT and attached body; lines 32–32: const ENROLLMENT_SOCKET_UNIT and attached body; lines 33–33: const ENROLLMENT_TEMPLATE_UNIT and attached body; lines 34–34: const ENROLLMENT_UNIT_DIRECTORY and attached body; lines 35–35: const ENROLLMENT_PORT and attached body; lines 36–36: const ENROLLMENT_SOCKET and attached body; lines 37–37: const ENROLLMENT_CONFIG_PATH and attached body; lines 38–38: const ENROLLMENT_KEY_LIMIT and attached body | [D11](../../slices/release-and-installation.md#d11-host-installation-and-payload-application) | retained | Declaration block for mod arm in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 25 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-072923d81a78"></a>

## [cmd/soda-install/src/enroll/native_config.rs](../../../../../cmd/soda-install/src/enroll/native_config.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–177; lines 1–1: use crate and attached body; lines 2–2: use crate and attached body; lines 3–7: use super and attached body; lines 8–18: fn enrollment_binary and attached body; lines 19–62: fn enrollment_config and attached body; lines 63–75: fn enrollment_public_key and attached body; lines 76–89: fn read_bounded and attached body; lines 90–104: fn enrollment_private_address and attached body; lines 105–109: fn enrollment_client_command and attached body; lines 110–136: fn enrollment_start_args and attached body; lines 137–151: fn enrollment_socket_unit_config and attached body; lines 152–177: fn enrollment_template_unit_config and attached body | [D11](../../slices/release-and-installation.md#d11-host-installation-and-payload-application) | retained | Declaration block for use crate in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 12 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-68248ea706b1"></a>

## [cmd/soda-install/src/enroll/peer.rs](../../../../../cmd/soda-install/src/enroll/peer.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–56; lines 1–1: use crate and attached body; lines 2–3: fn enrollment_peer_unit and attached body; lines 4–14: const PREFIX and attached body; lines 15–16: fn enrollment_receiver_unit and attached body; lines 17–17: const PREFIX and attached body; lines 18–28: const SUFFIX and attached body; lines 29–31: fn enrollment_connection_unit and attached body; lines 32–56: use std and attached body | [D11](../../slices/release-and-installation.md#d11-host-installation-and-payload-application) | retained | Declaration block for use crate in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 8 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-3bb183996527"></a>

## [cmd/soda-install/src/enroll/receive.rs](../../../../../cmd/soda-install/src/enroll/receive.rs)

Inherited prior inventory row; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–150; lines 1–5: use std and attached body; lines 6–6: use std and attached body; lines 7–8: use crate and attached body; lines 9–9: use crate and attached body; lines 10–11: use super and attached body; lines 12–15: use super and attached body; lines 16–36: fn admit_receive_enrollment and attached body; lines 37–48: fn confirm_enrollment_import and attached body; lines 49–62: fn submit_enrollment_key and attached body; lines 63–80: use std and attached body; lines 81–91: use std and attached body; lines 92–98: fn receive_enrollment and attached body; lines 99–106: use std and attached body; lines 107–150: fn enrollment_stdin and attached body | [O07](../../slices/operator-administration.md#o07-operator-ssh-enrollment) | retained | Declaration block for use std in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 14 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-79374d7b4c97"></a>

## [cmd/soda-install/src/enroll/selinux.rs](../../../../../cmd/soda-install/src/enroll/selinux.rs)

Inherited prior inventory row; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–203; lines 1–10: const ENROLLMENT_SELINUX_PORT_TYPE and attached body; lines 11–28: fn enrollment_port_labeled and attached body; lines 29–33: fn ensure_enrollment_port_label and attached body; lines 34–76: use crate and attached body; lines 77–81: fn label_enrollment_config and attached body; lines 82–101: use crate and attached body; lines 102–102: mod tests and attached body; lines 103–103: use super and attached body; lines 104–104: use crate and attached body; lines 105–105: use crate and attached body; lines 106–108: use crate and attached body; lines 109–127: fn port_labeled_matrix and attached body; lines 128–177: fn ensure_port_label_flows and attached body; lines 178–203: fn label_config_flows and attached body | [O07](../../slices/operator-administration.md#o07-operator-ssh-enrollment) | retained | Declaration block for const ENROLLMENT_SELINUX_PORT_TYPE in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 14 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-ecd587154af5"></a>

## [cmd/soda-install/src/enroll/serve.rs](../../../../../cmd/soda-install/src/enroll/serve.rs)

Inherited prior inventory row; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–219; lines 1–5: use std and attached body; lines 6–6: use std and attached body; lines 7–7: use std and attached body; lines 8–9: use crate and attached body; lines 10–10: use crate and attached body; lines 11–11: use crate and attached body; lines 12–15: use super and attached body; lines 16–16: use super and attached body; lines 17–21: use super and attached body; lines 22–38: fn validate_enrollment_server_environment and attached body; lines 39–46: fn start_enrollment_broker and attached body; lines 47–66: use std and attached body; lines 67–78: fn set_enrollment_connection_deadline and attached body; lines 79–99: fn read_enrollment_key_from_connection and attached body; lines 100–110: use std and attached body; lines 111–118: fn enrollment_commit_status and attached body; lines 119–156: fn commit_enrollment_key and attached body; lines 157–164: use std and attached body; lines 165–213: fn serve_enrollment_loop and attached body; lines 214–219: fn serve_enrollment and attached body | [O07](../../slices/operator-administration.md#o07-operator-ssh-enrollment) | retained | Declaration block for use std in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 20 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-c2d6ce709412"></a>

## [cmd/soda-install/src/enroll/session.rs](../../../../../cmd/soda-install/src/enroll/session.rs)

Inherited prior inventory row; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–358; lines 1–4: use std and attached body; lines 5–5: use std and attached body; lines 6–6: use std and attached body; lines 7–8: use crate and attached body; lines 9–9: use crate and attached body; lines 10–10: use crate and attached body; lines 11–11: use crate and attached body; lines 12–15: use super and attached body; lines 16–16: use super and attached body; lines 17–21: use super and attached body; lines 22–29: struct EnrollmentSession and attached body; lines 30–36: fn new and attached body; lines 37–91: fn stop_units and attached body; lines 92–121: fn cleanup_units and attached body; lines 122–129: fn close and attached body; lines 130–162: fn write_enrollment_config and attached body; lines 163–202: fn publish_enrollment_units and attached body; lines 203–216: fn start_enrollment_service and attached body; lines 217–227: fn publish_enrollment_state and attached body; lines 228–246: fn check_enrollment_result and attached body; lines 247–260: fn notify_enrollment_ready and attached body; lines 261–284: fn poll_console_cancel and attached body; lines 285–310: fn check_enrollment_service_active and attached body; lines 311–342: fn wait_enrollment_result and attached body; lines 343–358: fn arm_enrollment and attached body | [O07](../../slices/operator-administration.md#o07-operator-ssh-enrollment) | retained | Declaration block for use std in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 25 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-293731d2e5b7"></a>

## [cmd/soda-install/src/enroll/test_support.rs](../../../../../cmd/soda-install/src/enroll/test_support.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–97; lines 1–2: const TEST_KEY and attached body; lines 3–3: const TEST_KEY_2 and attached body; lines 4–7: const TEST_KEY_3 and attached body; lines 8–8: static ENV_LOCK and attached body; lines 9–12: struct TempDir and attached body; lines 13–14: impl Drop and attached body; lines 15–20: fn drop and attached body; lines 21–30: fn temp_dir and attached body; lines 31–34: fn test_uid and attached body; lines 35–39: struct TestPty and attached body; lines 40–41: fn open_test_pty and attached body; lines 42–75: use std and attached body; lines 76–76: fn drain_available and attached body; lines 77–77: use std and attached body; lines 78–97: use std and attached body | [D11](../../slices/release-and-installation.md#d11-host-installation-and-payload-application) | retained | Declaration block for const TEST_KEY in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 15 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-c2eb5dbab112"></a>

## [cmd/soda-install/src/enroll/tests.rs](../../../../../cmd/soda-install/src/enroll/tests.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–170; lines 1–1: use super and attached body; lines 2–2: use super and attached body; lines 3–5: use super and attached body; lines 6–36: fn enrollment_input_matrix and attached body; lines 37–63: fn enrollment_private_address_and_client and attached body; lines 64–109: fn enrollment_native_policy and attached body; lines 110–144: fn enrollment_native_socket_ownership and attached body; lines 145–170: fn enrollment_peer_service_provenance and attached body | [D11](../../slices/release-and-installation.md#d11-host-installation-and-payload-application) | retained | Declaration block for use super in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 8 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-c77152f20702"></a>

## [cmd/soda-install/src/errors.rs](../../../../../cmd/soda-install/src/errors.rs)

Inherited prior inventory row; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–105; lines 1–7: use std and attached body; lines 8–30: enum Error and attached body; lines 31–32: impl Error and attached body; lines 33–36: fn msg and attached body; lines 37–38: impl fmt and attached body; lines 39–57: fn fmt and attached body; lines 58–62: impl std and attached body; lines 63–81: fn errno_text and attached body; lines 82–89: fn os_error and attached body; lines 90–98: fn path_error and attached body; lines 99–105: fn link_error and attached body | [D11](../../slices/release-and-installation.md#d11-host-installation-and-payload-application) | retained | Declaration block for use std in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 11 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-1cedb82a8efa"></a>

## [cmd/soda-install/src/execute.rs](../../../../../cmd/soda-install/src/execute.rs)

Current cohesive body inspected; single defining duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–404; whole file: native disk installation execution, media verification, retry/diagnostic console handling, and attempt ignition lifecycle | [D11](../../slices/release-and-installation.md#d11-host-installation-and-payload-application) | retained | Native disk installation execution, media verification, retry/diagnostic console handling, and attempt ignition lifecycle. — cmd/soda-install/src/execute.rs; invoked by soda-install disk action |

<a id="coverage-20069380e830"></a>

## [cmd/soda-install/src/execute/tests.rs](../../../../../cmd/soda-install/src/execute/tests.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–298; lines 1–1: use super and attached body; lines 2–2: use crate and attached body; lines 3–3: use std and attached body; lines 4–4: use std and attached body; lines 5–47: fn fixture_disk and attached body; lines 48–123: fn execution_boundary and attached body; lines 124–185: fn removable_and_blocked_gates and attached body; lines 186–190: fn pipe_console and attached body; lines 191–203: use std and attached body; lines 204–209: struct PowerRecorder and attached body; lines 210–211: impl Runner and attached body; lines 212–230: fn run and attached body; lines 231–293: fn landing_outcomes and attached body; lines 294–298: fn media_verification_fails_without_identity and attached body | [D11](../../slices/release-and-installation.md#d11-host-installation-and-payload-application) | retained | Declaration block for use super in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 14 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-3077d08d6fa5"></a>

## [cmd/soda-install/src/hostadmit.rs](../../../../../cmd/soda-install/src/hostadmit.rs)

Inherited prior inventory row; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–121; lines 1–5: use std and attached body; lines 6–7: use crate and attached body; lines 8–8: use crate and attached body; lines 9–9: use crate and attached body; lines 10–10: use crate and attached body; lines 11–22: fn live_iso_from_cmdline and attached body; lines 23–49: fn admit_live_installer and attached body; lines 50–61: fn selinux_enforcing and attached body; lines 62–74: fn os_release and attached body; lines 75–101: fn core_os_host and attached body; lines 102–102: mod tests and attached body; lines 103–105: use super and attached body; lines 106–121: fn os_release_and_live_detection and attached body | [D11](../../slices/release-and-installation.md#d11-host-installation-and-payload-application) | retained | Declaration block for use std in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 13 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-85c23fa204be"></a>

## [cmd/soda-install/src/inputs.rs](../../../../../cmd/soda-install/src/inputs.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–218; lines 1–4: use crate and attached body; lines 5–5: use crate and attached body; lines 6–6: use crate and attached body; lines 7–7: use serde_json and attached body; lines 8–8: use serde_json and attached body; lines 9–12: use std and attached body; lines 13–31: fn hostname and attached body; lines 32–37: fn project_subnet and attached body; lines 38–56: fn valid_password_hash and attached body; lines 57–60: fn valid_provisioning_inputs and attached body; lines 61–99: fn template_storage and attached body; lines 100–121: fn admit_provisioning_files and attached body; lines 122–125: fn file_entry and attached body; lines 126–131: fn set_field and attached body; lines 132–133: struct GoHtmlFormatter and attached body; lines 134–134: impl Formatter and attached body; lines 135–157: fn write_string_fragment and attached body; lines 158–171: fn serialize_ignition and attached body; lines 172–217: fn destination and attached body; lines 218–218: mod tests and attached body | [D11](../../slices/release-and-installation.md#d11-host-installation-and-payload-application) | retained | Declaration block for use crate in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 20 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-f94e80267daf"></a>

## [cmd/soda-install/src/inputs/tests.rs](../../../../../cmd/soda-install/src/inputs/tests.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–250; lines 1–3: use super and attached body; lines 4–45: fn hostname_vectors and attached body; lines 46–66: fn subnet_vectors and attached body; lines 67–89: fn password_hash_vectors and attached body; lines 90–92: const TEMPLATE and attached body; lines 93–94: const KEY and attached body; lines 95–100: fn hash and attached body; lines 101–122: fn destination_assembles_config and attached body; lines 123–130: fn destination_without_key_omits_authorized_keys and attached body; lines 131–140: fn destination_keeps_go_safe_compact_json_shape and attached body; lines 141–250: fn destination_rejects_bad_inputs and attached body | [D11](../../slices/release-and-installation.md#d11-host-installation-and-payload-application) | retained | Declaration block for use super in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 11 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-e362a49b8491"></a>

## [cmd/soda-install/src/main.rs](../../../../../cmd/soda-install/src/main.rs)

Inherited prior inventory row; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–47; lines 1–3: mod buildx and attached body; lines 4–4: mod candidate and attached body; lines 5–5: mod command and attached body; lines 6–6: mod console and attached body; lines 7–7: mod deliver and attached body; lines 8–8: mod disks and attached body; lines 9–9: mod enroll and attached body; lines 10–10: mod errors and attached body; lines 11–11: mod execute and attached body; lines 12–12: mod hostadmit and attached body; lines 13–13: mod inputs and attached body; lines 14–14: mod netip and attached body; lines 15–15: mod oci and attached body; lines 16–16: mod pathx and attached body; lines 17–17: mod pemx and attached body; lines 18–18: mod run and attached body; lines 19–19: mod setup and attached body; lines 20–20: mod signal and attached body; lines 21–21: mod sshkey and attached body; lines 22–22: mod wizard and attached body; lines 23–23: mod x509 and attached body; lines 24–40: fn main and attached body; lines 41–47: fn action and attached body | [D11](../../slices/release-and-installation.md#d11-host-installation-and-payload-application) | retained | Declaration block for mod buildx in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 23 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-87ac45cb01a0"></a>

## [cmd/soda-install/src/netip/address.rs](../../../../../cmd/soda-install/src/netip/address.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–64; lines 1–4: use std and attached body; lines 5–10: struct Addr and attached body; lines 11–13: struct NetipError and attached body; lines 14–15: impl NetipError and attached body; lines 16–20: fn addr_bytes and attached body; lines 21–26: fn prefix and attached body; lines 27–27: impl std and attached body; lines 28–31: fn fmt and attached body; lines 32–32: impl std and attached body; lines 33–36: fn parse_addr and attached body; lines 37–64: fn parse_addr_bytes and attached body | [D11](../../slices/release-and-installation.md#d11-host-installation-and-payload-application) | retained | Declaration block for use std in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 11 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-d39b3dcc7a2a"></a>

## [cmd/soda-install/src/netip/address_format.rs](../../../../../cmd/soda-install/src/netip/address_format.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–68; lines 1–1: use super and attached body; lines 2–2: use std and attached body; lines 3–4: impl Addr and attached body; lines 5–7: fn is4 and attached body; lines 8–10: fn is6 and attached body; lines 11–13: fn is4_in6 and attached body; lines 14–16: fn zone and attached body; lines 17–25: fn is_private and attached body; lines 26–34: fn to_string_go and attached body; lines 35–41: fn bits and attached body; lines 42–50: fn masked_bytes and attached body; lines 51–55: fn is_private_v4 and attached body; lines 56–68: fn mask and attached body | [D11](../../slices/release-and-installation.md#d11-host-installation-and-payload-application) | retained | Declaration block for use super in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 13 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-eb65218110d3"></a>

## [cmd/soda-install/src/netip/mod.rs](../../../../../cmd/soda-install/src/netip/mod.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–12; lines 1–4: use self and attached body; lines 5–5: use self and attached body; lines 6–7: mod address and attached body; lines 8–8: mod address_format and attached body; lines 9–11: mod prefix and attached body; lines 12–12: mod tests and attached body | [D11](../../slices/release-and-installation.md#d11-host-installation-and-payload-application) | retained | Declaration block for use self in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 6 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-badc330165bb"></a>

## [cmd/soda-install/src/netip/prefix.rs](../../../../../cmd/soda-install/src/netip/prefix.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–69; lines 1–3: use super and attached body; lines 4–7: struct Prefix and attached body; lines 8–33: fn parse_prefix and attached body; lines 34–35: impl Prefix and attached body; lines 36–38: fn addr and attached body; lines 39–52: fn masked and attached body; lines 53–69: fn contains and attached body | [D11](../../slices/release-and-installation.md#d11-host-installation-and-payload-application) | retained | Declaration block for use super in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 7 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-7ca38279c508"></a>

## [cmd/soda-install/src/netip/tests/address.rs](../../../../../cmd/soda-install/src/netip/tests/address.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–50; lines 1–3: use super and attached body; lines 4–28: fn standard_ip_grammar_and_display and attached body; lines 29–36: fn zone_adapter_splits_first_percent_and_requires_nonempty_ipv6_zone and attached body; lines 37–50: fn private_policy_unmaps_ipv4_mapped_values and attached body | [D11](../../slices/release-and-installation.md#d11-host-installation-and-payload-application) | retained | Declaration block for use super in the current test-or-fixture source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 4 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-40bad251ed56"></a>

## [cmd/soda-install/src/netip/tests/mod.rs](../../../../../cmd/soda-install/src/netip/tests/mod.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–2; lines 1–1: mod address and attached body; lines 2–2: mod prefix and attached body | [D11](../../slices/release-and-installation.md#d11-host-installation-and-payload-application) | retained | Declaration block for mod address in the current test-or-fixture source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; Declaration block for mod prefix in the current test-or-fixture source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context. — cmd/soda-install/src/netip/tests/mod.rs:1-1; current source declaration and body; cmd/soda-install/src/netip/tests/mod.rs:2-2; current source declaration and body |

<a id="coverage-d3518eb82d22"></a>

## [cmd/soda-install/src/netip/tests/prefix.rs](../../../../../cmd/soda-install/src/netip/tests/prefix.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–45; lines 1–3: use super and attached body; lines 4–25: fn prefix_masks_and_contains_by_family and attached body; lines 26–45: fn prefix_admission_checks_length_family_zone_and_mask and attached body | [D11](../../slices/release-and-installation.md#d11-host-installation-and-payload-application) | retained | Declaration block for use super in the current test-or-fixture source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; Declaration block for fn prefix_masks_and_contains_by_family in the current test-or-fixture source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; Declaration block for fn prefix_admission_checks_length_family_zone_and_mask in the current test-or-fixture source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context. — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-2e26e04830cf"></a>

## [cmd/soda-install/src/oci/inspection.rs](../../../../../cmd/soda-install/src/oci/inspection.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–259; lines 1–1: use std and attached body; lines 2–3: use super and attached body; lines 4–4: use super and attached body; lines 5–5: use super and attached body; lines 6–6: use super and attached body; lines 7–7: use crate and attached body; lines 8–12: use crate and attached body; lines 13–28: fn fetch_oci_blob and attached body; lines 29–39: fn validate_oci_layers and attached body; lines 40–56: fn validate_oci_rootfs and attached body; lines 57–92: fn validate_oci_attribution and attached body; lines 93–139: fn inspect_oci_config and attached body; lines 140–159: fn inspect_oci_image and attached body; lines 160–179: fn validate_oci_layout_inputs and attached body; lines 180–210: fn inspect_layout_descriptor and attached body; lines 211–230: fn inspect_layout_images and attached body; lines 231–259: fn inspect_oci_layout and attached body | [D11](../../slices/release-and-installation.md#d11-host-installation-and-payload-application) | retained | Declaration block for use std in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 17 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-8abb6b9cbcb6"></a>

## [cmd/soda-install/src/oci/layout.rs](../../../../../cmd/soda-install/src/oci/layout.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–240; lines 1–1: use std and attached body; lines 2–2: use std and attached body; lines 3–3: use std and attached body; lines 4–4: use std and attached body; lines 5–5: use std and attached body; lines 6–7: use crate and attached body; lines 8–10: use crate and attached body; lines 11–21: struct OciImage and attached body; lines 22–28: struct OciLayout and attached body; lines 29–35: struct Blob and attached body; lines 36–42: struct Descriptor and attached body; lines 43–52: struct Loader and attached body; lines 53–68: fn open_root and attached body; lines 69–69: fn open_layout_file and attached body; lines 70–119: use std and attached body; lines 120–142: fn lstat_layout_file and attached body; lines 143–148: fn copy_oci_blob and attached body; lines 149–190: use sha2 and attached body; lines 191–215: fn read_oci_blob and attached body; lines 216–240: fn load_layout_file and attached body | [D11](../../slices/release-and-installation.md#d11-host-installation-and-payload-application) | retained | Declaration block for use std in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 20 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-0707ceadbd3d"></a>

## [cmd/soda-install/src/oci/metadata.rs](../../../../../cmd/soda-install/src/oci/metadata.rs)

Current cohesive body inspected; single defining duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–559; whole file: installer-side oci metadata and descriptor parsing used to validate the release image layout before payload application | [D11](../../slices/release-and-installation.md#d11-host-installation-and-payload-application) | retained | Installer-side OCI metadata and descriptor parsing used to validate the release image layout before payload application. — cmd/soda-install/src/oci/metadata.rs; consumed by installer OCI import and content validation |

<a id="coverage-729f20197d20"></a>

## [cmd/soda-install/src/oci/mod.rs](../../../../../cmd/soda-install/src/oci/mod.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–25; lines 1–7: use self and attached body; lines 8–8: use self and attached body; lines 9–10: const MANIFEST_MEDIA_TYPE and attached body; lines 11–11: const CONFIG_MEDIA_TYPE and attached body; lines 12–12: const INDEX_MEDIA_TYPE and attached body; lines 13–13: const LAYER_TAR and attached body; lines 14–14: const LAYER_GZIP and attached body; lines 15–15: const LAYER_ZSTD and attached body; lines 16–17: mod inspection and attached body; lines 18–18: mod layout and attached body; lines 19–21: mod metadata and attached body; lines 22–24: mod test_support and attached body; lines 25–25: mod tests and attached body | [D11](../../slices/release-and-installation.md#d11-host-installation-and-payload-application) | retained | Declaration block for use self in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 13 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-127ab6ca6386"></a>

## [cmd/soda-install/src/oci/test_support.rs](../../../../../cmd/soda-install/src/oci/test_support.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–99; lines 1–1: use super and attached body; lines 2–2: use std and attached body; lines 3–3: use std and attached body; lines 4–5: use crate and attached body; lines 6–6: use serde_json and attached body; lines 7–8: static COUNTER and attached body; lines 9–23: fn temp_dir and attached body; lines 24–35: fn write_blob and attached body; lines 36–43: fn desc and attached body; lines 44–87: fn add_image and attached body; lines 88–99: fn layout_fixture and attached body | [D11](../../slices/release-and-installation.md#d11-host-installation-and-payload-application) | retained | Declaration block for use super in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 11 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-7e6c8bfa62f4"></a>

## [cmd/soda-install/src/oci/tests.rs](../../../../../cmd/soda-install/src/oci/tests.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–151; lines 1–1: use super and attached body; lines 2–2: use super and attached body; lines 3–3: use std and attached body; lines 4–7: use crate and attached body; lines 8–42: fn layout_preserves_identities_and_counts_once and attached body; lines 43–133: fn layout_refuses_substitution and attached body; lines 134–151: fn descriptor_size_accepts_negative_zero_but_rejects_noninteger_tokens and attached body | [D11](../../slices/release-and-installation.md#d11-host-installation-and-payload-application) | retained | Declaration block for use super in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 7 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-0b369eb80558"></a>

## [cmd/soda-install/src/pathx.rs](../../../../../cmd/soda-install/src/pathx.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–92; lines 1–5: use std and attached body; lines 6–14: fn join and attached body; lines 15–30: fn dir and attached body; lines 31–38: fn base and attached body; lines 39–52: fn eval_symlinks and attached body; lines 53–53: mod tests and attached body; lines 54–56: use super and attached body; lines 57–74: fn join_dir_base_vectors and attached body; lines 75–92: fn symlinks_resolve and attached body | [D11](../../slices/release-and-installation.md#d11-host-installation-and-payload-application) | retained | Declaration block for use std in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 9 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-a43afcb47dce"></a>

## [cmd/soda-install/src/pemx.rs](../../../../../cmd/soda-install/src/pemx.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–115; lines 1–3: use base64 and attached body; lines 4–5: const BEGIN and attached body; lines 6–6: const END and attached body; lines 7–7: const MAX_INPUT and attached body; lines 8–68: fn decode_certificate and attached body; lines 69–69: mod tests and attached body; lines 70–70: use super and attached body; lines 71–76: fn pem and attached body; lines 77–115: fn accepts_one_public_certificate_and_refuses_other_content and attached body | [D11](../../slices/release-and-installation.md#d11-host-installation-and-payload-application) | retained | Declaration block for use base64 in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 9 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-e59905a3fbe2"></a>

## [cmd/soda-install/src/setup/access.rs](../../../../../cmd/soda-install/src/setup/access.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–174; lines 1–1: use super and attached body; lines 2–2: use super and attached body; lines 3–4: use crate and attached body; lines 5–5: use crate and attached body; lines 6–6: use crate and attached body; lines 7–7: use crate and attached body; lines 8–8: use url and attached body; lines 9–43: fn valid_installed_https_origin and attached body; lines 44–62: fn valid_percent_escapes and attached body; lines 63–83: fn installed_forgejo_origin and attached body; lines 84–91: fn uses_internal_tls and attached body; lines 92–116: fn confirm_active_browser_units and attached body; lines 117–142: fn print_local_ca_guidance and attached body; lines 143–174: fn configured_access and attached body | [D11](../../slices/release-and-installation.md#d11-host-installation-and-payload-application) | retained | Declaration block for use super in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 14 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-9ef677bda77d"></a>

## [cmd/soda-install/src/setup/address.rs](../../../../../cmd/soda-install/src/setup/address.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–139; lines 1–1: use crate and attached body; lines 2–4: use crate and attached body; lines 5–8: struct SetupAddress and attached body; lines 9–16: fn skip_setup_interface and attached body; lines 17–44: fn append_setup_address and attached body; lines 45–63: fn private_setup_origin and attached body; lines 64–131: fn setup_addresses and attached body; lines 132–139: fn optional_string and attached body | [D11](../../slices/release-and-installation.md#d11-host-installation-and-payload-application) | retained | Declaration block for use crate in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 8 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-c75267d40678"></a>

## [cmd/soda-install/src/setup/configure.rs](../../../../../cmd/soda-install/src/setup/configure.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–295; current setup configuration/install declarations including write_setup_file | [D11](../../slices/release-and-installation.md#d11-host-installation-and-payload-application) | retained | `write_setup_file` uses complete `write_all` semantics while preserving exclusive creation, private file mode and close-error precedence; actual installer checks cover complete writes and custody. |

<a id="coverage-baa827087b33"></a>

## [cmd/soda-install/src/setup/local_ca.rs](../../../../../cmd/soda-install/src/setup/local_ca.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–12; lines 1–1: use crate and attached body; lines 2–12: fn local_ca_fingerprint and attached body | [D11](../../slices/release-and-installation.md#d11-host-installation-and-payload-application) | retained | Declaration block for use crate in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; Declaration block for fn local_ca_fingerprint in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context. — cmd/soda-install/src/setup/local_ca.rs:1-1; current source declaration and body; cmd/soda-install/src/setup/local_ca.rs:2-12; current source declaration and body |

<a id="coverage-cb2e6f03e11b"></a>

## [cmd/soda-install/src/setup/mod.rs](../../../../../cmd/soda-install/src/setup/mod.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–36; lines 1–5: use crate and attached body; lines 6–6: use crate and attached body; lines 7–7: use crate and attached body; lines 8–8: use crate and attached body; lines 9–10: use self and attached body; lines 11–11: use self and attached body; lines 12–12: use self and attached body; lines 13–15: use self and attached body; lines 16–16: use self and attached body; lines 17–19: const LOCAL_CA_PATH and attached body; lines 20–25: const SBIN and attached body; lines 26–28: fn configure_install and attached body; lines 29–30: mod access and attached body; lines 31–31: mod address and attached body; lines 32–32: mod configure and attached body; lines 33–35: mod local_ca and attached body; lines 36–36: mod tests and attached body | [D11](../../slices/release-and-installation.md#d11-host-installation-and-payload-application) | retained | Declaration block for use crate in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 17 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-8a1b0e06babc"></a>

## [cmd/soda-install/src/setup/tests/access.rs](../../../../../cmd/soda-install/src/setup/tests/access.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–154; lines 1–1: use super and attached body; lines 2–2: use super and attached body; lines 3–4: use crate and attached body; lines 5–6: use crate and attached body; lines 7–25: fn tlv and attached body; lines 26–27: fn ca_cert_pem and attached body; lines 28–82: use ed25519_dalek and attached body; lines 83–96: fn local_certificate_export_requires_public_ca and attached body; lines 97–128: fn local_trust_guidance_rejects_untrusted_destination and attached body; lines 129–154: fn configured_guidance_reports_inactive_service_without_replay and attached body | [D11](../../slices/release-and-installation.md#d11-host-installation-and-payload-application) | retained | Declaration block for use super in the current test-or-fixture source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 10 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-97dae69c1f38"></a>

## [cmd/soda-install/src/setup/tests/configure.rs](../../../../../cmd/soda-install/src/setup/tests/configure.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–368; lines 1–1: use super and attached body; lines 2–2: use super and attached body; lines 3–4: use crate and attached body; lines 5–8: use crate and attached body; lines 9–53: fn private_setup_addresses and attached body; lines 54–79: fn private_setup_does_not_replay_existing_state and attached body; lines 80–98: fn configure_runs_without_laptop_terminal and attached body; lines 99–169: fn setup_rollback_keeps_referenced_state and attached body; lines 170–183: fn operator_token_validation and attached body; lines 184–185: fn read_until and attached body; lines 186–186: use std and attached body; lines 187–209: use std and attached body; lines 210–210: fn private_setup_keeps_credential_out_of_commands_and_transcript and attached body; lines 211–211: use std and attached body; lines 212–220: use std and attached body; lines 221–368: const TOKEN and attached body | [D11](../../slices/release-and-installation.md#d11-host-installation-and-payload-application) | retained | Declaration block for use super in the current test-or-fixture source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 16 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-c5633f85a396"></a>

## [cmd/soda-install/src/setup/tests/fixtures.rs](../../../../../cmd/soda-install/src/setup/tests/fixtures.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–5; lines 1–1: use super and attached body; lines 2–5: fn null_console and attached body | [D11](../../slices/release-and-installation.md#d11-host-installation-and-payload-application) | retained | Declaration block for use super in the current fixture-or-asset source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; Declaration block for fn null_console in the current fixture-or-asset source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context. — cmd/soda-install/src/setup/tests/fixtures.rs:1-1; current source declaration and body; cmd/soda-install/src/setup/tests/fixtures.rs:2-5; current source declaration and body |

<a id="coverage-d611b3f1b48a"></a>

## [cmd/soda-install/src/setup/tests/mod.rs](../../../../../cmd/soda-install/src/setup/tests/mod.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–3; lines 1–1: mod access and attached body; lines 2–2: mod configure and attached body; lines 3–3: mod fixtures and attached body | [D11](../../slices/release-and-installation.md#d11-host-installation-and-payload-application) | retained | Declaration block for mod access in the current test-or-fixture source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; Declaration block for mod configure in the current test-or-fixture source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; Declaration block for mod fixtures in the current test-or-fixture source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context. — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-5feb75a83f03"></a>

## [cmd/soda-install/src/signal.rs](../../../../../cmd/soda-install/src/signal.rs)

Inherited prior inventory row; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–192; lines 1–5: use std and attached body; lines 6–6: use std and attached body; lines 7–7: use std and attached body; lines 8–9: use crate and attached body; lines 10–11: static TERMINATED and attached body; lines 12–27: static SIGINT_COUNT and attached body; lines 28–40: fn install_handlers and attached body; lines 41–49: enum CancelSource and attached body; lines 50–55: struct Ctx and attached body; lines 56–59: impl Ctx and attached body; lines 60–72: fn root and attached body; lines 73–85: fn test and attached body; lines 86–97: fn interrupt_scope and attached body; lines 98–107: fn detached and attached body; lines 108–119: fn with_timeout and attached body; lines 120–128: fn terminated and attached body; lines 129–133: fn deadline and attached body; lines 134–150: fn err and attached body; lines 151–155: fn test_deliver_sigint and attached body; lines 156–156: mod tests and attached body; lines 157–157: use super and attached body; lines 158–158: use std and attached body; lines 159–162: static SERIAL and attached body; lines 163–192: fn scopes_observe_cancel_deadline_and_sigint and attached body | [D11](../../slices/release-and-installation.md#d11-host-installation-and-payload-application) | retained | Declaration block for use std in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 24 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-56e2ba74e5f2"></a>

## [cmd/soda-install/src/sshkey/authorized_keys.rs](../../../../../cmd/soda-install/src/sshkey/authorized_keys.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–88; lines 1–1: use super and attached body; lines 2–2: use super and attached body; lines 3–3: use super and attached body; lines 4–4: use sha2 and attached body; lines 5–12: struct AuthorizedKey and attached body; lines 13–37: fn parse_authorized_key and attached body; lines 38–63: fn parse_authorized_key_bytes and attached body; lines 64–84: fn public_key and attached body; lines 85–88: fn fingerprint_sha256_wire and attached body | [D11](../../slices/release-and-installation.md#d11-host-installation-and-payload-application) | retained | Declaration block for use super in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 9 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-f2f9f215bbd6"></a>

## [cmd/soda-install/src/sshkey/base64.rs](../../../../../cmd/soda-install/src/sshkey/base64.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–30; lines 1–1: use base64 and attached body; lines 2–2: use base64 and attached body; lines 3–18: fn go_std_decode and attached body; lines 19–22: fn b64_decode_go and attached body; lines 23–26: fn b64_encode and attached body; lines 27–30: fn b64_encode_raw and attached body | [D11](../../slices/release-and-installation.md#d11-host-installation-and-payload-application) | retained | Declaration block for use base64 in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 6 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-8ef09b44a221"></a>

## [cmd/soda-install/src/sshkey/mod.rs](../../../../../cmd/soda-install/src/sshkey/mod.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–18; lines 1–4: use self and attached body; lines 5–5: use self and attached body; lines 6–8: use self and attached body; lines 9–12: use self and attached body; lines 13–13: mod authorized_keys and attached body; lines 14–14: mod base64 and attached body; lines 15–17: mod wire and attached body; lines 18–18: mod tests and attached body | [D11](../../slices/release-and-installation.md#d11-host-installation-and-payload-application) | retained | Declaration block for use self in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 8 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-84c441cbf0d4"></a>

## [cmd/soda-install/src/sshkey/tests.rs](../../../../../cmd/soda-install/src/sshkey/tests.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–226; lines 1–1: use super and attached body; lines 2–2: use super and attached body; lines 3–7: fn put_string and attached body; lines 8–13: fn marshal_string and attached body; lines 14–22: fn mpint and attached body; lines 23–29: fn rsa_line and attached body; lines 30–32: const ED25519 and attached body; lines 33–33: const RSA and attached body; lines 34–34: const ECDSA256 and attached body; lines 35–35: const ECDSA384 and attached body; lines 36–36: const ECDSA521 and attached body; lines 37–44: fn sk_ed25519_wire and attached body; lines 45–52: fn sk_ecdsa_wire and attached body; lines 53–63: fn ordinary_p256_point and attached body; lines 64–83: fn real_keys_parse_and_normalize and attached body; lines 84–92: fn rsa_exponent_and_modulus_bounds_remain_installer_policy and attached body; lines 93–154: fn rejects_match_go_taxonomy and attached body; lines 155–173: fn unsupported_types_reported and attached body; lines 174–185: fn sk_keys_match_go_truncation_rule and attached body; lines 186–201: fn sk_ecdsa_keeps_uncompressed_valid_point_policy and attached body; lines 202–218: fn base64_matches_go_decode and attached body; lines 219–226: fn malformed_mpint_and_trailing_wire_bytes_are_rejected and attached body | [D11](../../slices/release-and-installation.md#d11-host-installation-and-payload-application) | retained | Declaration block for use super in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 22 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-12f259082594"></a>

## [cmd/soda-install/src/sshkey/wire.rs](../../../../../cmd/soda-install/src/sshkey/wire.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–135; lines 1–1: use elliptic_curve and attached body; lines 2–2: use ssh_key and attached body; lines 3–5: use ssh_key and attached body; lines 6–16: enum Kind and attached body; lines 17–21: struct Key and attached body; lines 22–23: impl Key and attached body; lines 24–26: fn key_type and attached body; lines 27–29: fn kind and attached body; lines 30–33: fn marshal and attached body; lines 34–87: fn validate_point and attached body; lines 88–93: fn bit_len and attached body; lines 94–135: fn parse_public_key and attached body | [D11](../../slices/release-and-installation.md#d11-host-installation-and-payload-application) | retained | Declaration block for use elliptic_curve in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 12 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-2b367fa388f6"></a>

## [cmd/soda-install/src/wizard/mod.rs](../../../../../cmd/soda-install/src/wizard/mod.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–114; lines 1–4: use crate and attached body; lines 5–5: use crate and attached body; lines 6–6: use crate and attached body; lines 7–7: use crate and attached body; lines 8–8: use crate and attached body; lines 9–10: use self and attached body; lines 11–14: use self and attached body; lines 15–18: type DiskInspector and attached body; lines 19–27: struct DiskInstallChoices and attached body; lines 28–89: fn dispatch_install_step and attached body; lines 90–108: fn collect_disk_install_choices and attached body; lines 109–110: mod review and attached body; lines 111–113: mod steps and attached body; lines 114–114: mod tests and attached body | [D11](../../slices/release-and-installation.md#d11-host-installation-and-payload-application) | retained | Declaration block for use crate in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 14 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-83139f87e96e"></a>

## [cmd/soda-install/src/wizard/review.rs](../../../../../cmd/soda-install/src/wizard/review.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–81; lines 1–1: use super and attached body; lines 2–2: use super and attached body; lines 3–4: use crate and attached body; lines 5–5: use crate and attached body; lines 6–6: use crate and attached body; lines 7–7: use crate and attached body; lines 8–8: use crate and attached body; lines 9–31: fn print_final_review and attached body; lines 32–38: fn erase_phrase and attached body; lines 39–53: fn confirm_final_review and attached body; lines 54–81: fn step_subnet_and_review and attached body | [D11](../../slices/release-and-installation.md#d11-host-installation-and-payload-application) | retained | Declaration block for use super in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 11 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-bc0a5fc329a9"></a>

## [cmd/soda-install/src/wizard/steps.rs](../../../../../cmd/soda-install/src/wizard/steps.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–249; lines 1–1: use super and attached body; lines 2–3: use crate and attached body; lines 4–4: use crate and attached body; lines 5–5: use crate and attached body; lines 6–6: use crate and attached body; lines 7–7: use crate and attached body; lines 8–8: use crate and attached body; lines 9–17: fn check_nav and attached body; lines 18–25: fn ask_nav and attached body; lines 26–35: fn ask_secret_nav and attached body; lines 36–43: fn step_network and attached body; lines 44–51: fn handle_disk_inspect_failure and attached body; lines 52–78: fn print_disk_list and attached body; lines 79–96: fn select_disk and attached body; lines 97–122: fn step_disk and attached body; lines 123–157: fn step_hostname and attached body; lines 158–159: const MIN_PASSWORD_RUNES and attached body; lines 160–166: fn valid_password and attached body; lines 167–176: fn password_feedback and attached body; lines 177–190: fn hash_password and attached body; lines 191–217: fn step_password and attached body; lines 218–249: fn step_subnet and attached body | [D11](../../slices/release-and-installation.md#d11-host-installation-and-payload-application) | retained | Declaration block for use super in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 22 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-186cdea7e948"></a>

## [cmd/soda-install/src/wizard/tests.rs](../../../../../cmd/soda-install/src/wizard/tests.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–176; lines 1–1: use super and attached body; lines 2–2: use crate and attached body; lines 3–5: use std and attached body; lines 6–75: fn selections_and_phrases and attached body; lines 76–176: fn full_wizard_flow and attached body | [D11](../../slices/release-and-installation.md#d11-host-installation-and-payload-application) | retained | Declaration block for use super in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 5 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-d0f7199abd11"></a>

## [cmd/soda-install/src/x509/algorithms.rs](../../../../../cmd/soda-install/src/x509/algorithms.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–129; lines 1–19: enum SignatureAlgorithm and attached body; lines 20–21: impl SignatureAlgorithm and attached body; lines 22–44: fn name and attached body; lines 45–51: enum PublicKeyAlgorithm and attached body; lines 52–53: impl PublicKeyAlgorithm and attached body; lines 54–63: fn name and attached body; lines 64–67: fn signature_algorithm and attached body; lines 68–129: use x509_cert and attached body | [D11](../../slices/release-and-installation.md#d11-host-installation-and-payload-application) | retained | Declaration block for enum SignatureAlgorithm in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 8 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-433ea76efbac"></a>

## [cmd/soda-install/src/x509/certificate.rs](../../../../../cmd/soda-install/src/x509/certificate.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–201; lines 1–1: use std and attached body; lines 2–7: use x509_cert and attached body; lines 8–14: use super and attached body; lines 15–124: fn parse_certificate and attached body; lines 125–129: fn parse_public_key and attached body; lines 130–195: use rsa and attached body; lines 196–201: fn rsa_parameters_supported and attached body | [D11](../../slices/release-and-installation.md#d11-host-installation-and-payload-application) | retained | Declaration block for use std in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 7 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-eaf2ab1ccf85"></a>

## [cmd/soda-install/src/x509/mod.rs](../../../../../cmd/soda-install/src/x509/mod.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–17; lines 1–8: mod algorithms and attached body; lines 9–9: mod certificate and attached body; lines 10–10: mod types and attached body; lines 11–13: mod verify and attached body; lines 14–14: mod tests and attached body; lines 15–16: use certificate and attached body; lines 17–17: use verify and attached body | [D11](../../slices/release-and-installation.md#d11-host-installation-and-payload-application) | retained | Declaration block for mod algorithms in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 7 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-fb33470bfa0d"></a>

## [cmd/soda-install/src/x509/tests/mod.rs](../../../../../cmd/soda-install/src/x509/tests/mod.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–347; lines 1–6: use super and attached body; lines 7–8: struct DeterministicTestRng and attached body; lines 9–10: impl rsa and attached body; lines 11–13: fn next_u32 and attached body; lines 14–19: fn next_u64 and attached body; lines 20–25: fn fill_bytes and attached body; lines 26–30: fn try_fill_bytes and attached body; lines 31–32: impl rsa and attached body; lines 33–34: const CADDY_ROOT and attached body; lines 35–37: const CADDY_DER_SHA256 and attached body; lines 38–48: fn pinned_caddy_root_admits_and_verifies_original_tbs and attached body; lines 49–61: fn pinned_root_refuses_signature_tampering_and_trailing_der and attached body; lines 62–62: fn ecdsa_signature_der_stays_strict_for_every_supported_curve and attached body; lines 63–84: use ecdsa and attached body; lines 85–105: fn signature_policy_refuses_weak_and_unknown_algorithms and attached body; lines 106–106: fn rsa_spki_parameters_allow_only_absent_or_null and attached body; lines 107–116: use x509_cert and attached body; lines 117–117: fn rsa_pss_requires_matching_hash_mgf_and_hash_length_salt and attached body; lines 118–121: use x509_cert and attached body; lines 122–140: fn identifier and attached body; lines 141–141: fn signature_algorithm_parameters_follow_the_typed_der_profile and attached body; lines 142–148: use x509_cert and attached body; lines 149–190: fn algorithm and attached body; lines 191–191: fn duplicate_and_malformed_used_extensions_are_refused and attached body; lines 192–192: use x509_cert and attached body; lines 193–228: use x509_cert and attached body; lines 229–229: fn retained_ecdsa_curve_verifiers_check_original_tbs and attached body; lines 230–230: use ecdsa and attached body; lines 231–231: use sha2 and attached body; lines 232–271: macro_rules! verify_curve and attached body; lines 272–296: use ecdsa and attached body; lines 297–297: fn retained_ed25519_verifier_checks_original_tbs and attached body; lines 298–313: use ed25519_dalek and attached body; lines 314–314: fn retained_rsa_pkcs1_and_pss_verifiers_check_original_tbs and attached body; lines 315–347: use sha2 and attached body | [D11](../../slices/release-and-installation.md#d11-host-installation-and-payload-application) | retained | Declaration block for use super in the current test-or-fixture source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 35 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-257e4ba9e3e5"></a>

## [cmd/soda-install/src/x509/types.rs](../../../../../cmd/soda-install/src/x509/types.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–21; lines 1–3: use super and attached body; lines 4–13: enum PublicKeyData and attached body; lines 14–21: struct Certificate and attached body | [D11](../../slices/release-and-installation.md#d11-host-installation-and-payload-application) | retained | Declaration block for use super in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; Declaration block for enum PublicKeyData in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; Declaration block for struct Certificate in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context. — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-ec6326cd1bae"></a>

## [cmd/soda-install/src/x509/verify.rs](../../../../../cmd/soda-install/src/x509/verify.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–228; lines 1–1: use super and attached body; lines 2–6: use super and attached body; lines 7–8: const ERR_UNSUPPORTED_ALGORITHM and attached body; lines 9–9: const ERR_RSA_VERIFICATION and attached body; lines 10–10: const ERR_ECDSA_FAILURE and attached body; lines 11–11: const ERR_ED25519_FAILURE and attached body; lines 12–18: fn insecure_algorithm_error and attached body; lines 19–28: fn mismatch_error and attached body; lines 29–39: enum Hash and attached body; lines 40–59: fn algorithm_details and attached body; lines 60–61: fn digest and attached body; lines 62–69: use sha2 and attached body; lines 70–99: fn verify_rsa and attached body; lines 100–106: fn verify_ecdsa_256 and attached body; lines 107–115: use signature and attached body; lines 116–122: fn verify_ecdsa_384 and attached body; lines 123–131: use signature and attached body; lines 132–138: fn verify_ecdsa_521 and attached body; lines 139–147: use signature and attached body; lines 148–154: fn verify_ecdsa_224 and attached body; lines 155–163: use signature and attached body; lines 164–170: fn verify_ed25519 and attached body; lines 171–183: use ed25519_dalek and attached body; lines 184–228: fn verify_self_signature and attached body | [D11](../../slices/release-and-installation.md#d11-host-installation-and-payload-application) | retained | Declaration block for use super in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 24 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-f1e982151d2f"></a>

## [cmd/soda-tailnet/command.go](../../../../../cmd/soda-tailnet/command.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–53; file scaffold; execute; enrollmentMessage; connectionDescription | [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) | retained | Current scaffold duty: file scaffold; 4 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-7f053f567b04"></a>

## [cmd/soda-tailnet/command_test.go](../../../../../cmd/soda-tailnet/command_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–79; file scaffold; TestEnrollmentMessage; TestConnectedMessageDoesNotInventServiceURLs; TestExecuteKeepsUnavailableStatusNonFatal; failingWriter; Write; TestExecuteReportsWriteFailure; TestExecutePreservesParentCancellationAsGuidance | [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) | retained | Current scaffold duty: file scaffold; 8 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-992ff36cb36f"></a>

## [cmd/soda-tailnet/main.go](../../../../../cmd/soda-tailnet/main.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–24; file scaffold; main; run | [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) | retained | Current scaffold duty: file scaffold; Current declaration duty: main; Current declaration duty: run — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

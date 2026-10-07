# Soda setup

[Responsibility map index](README.md) · [Coverage scope](../README.md).

Current responsibility accounting at `519b76bd` (2026-10-07).
One slice owns each named duty; disjoint complete symbols may share an owner.
Compound fields/clauses may share a physical line with distinct selectors.
Disposition concerns the duty, not source validity or installed qualification.

<a id="coverage-62bde0c1dda9"></a>

## [cmd/soda-setup/src/cli.rs](../../../../../cmd/soda-setup/src/cli.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–171; current module/import/attribute shell; declaration run; declaration SetupOpts; fields forgejo_url, forgejo_internal_url, token_file, out, provision_db_only; declaration ParseOutcome; declaration print_setup_usage; declaration parse_bool_flag; declaration parse_go_bool; declaration looks_like_flag; declaration parse_args | [O01](../../slices/operator-administration.md#o01-first-boot-database-provisioning) | retained | Imports and module declarations wire cmd/soda-setup/src/cli.rs into its current native target.; 9 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-f6899bb14730"></a>

## [cmd/soda-setup/src/config.rs](../../../../../cmd/soda-setup/src/config.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–60; current module/import/attribute shell; declaration DashboardConfig; fields listen, forgejo_url, forgejo_internal_url, database_dsn_file, host_socket, identity_socket, grant_key_file, operator_id, forgejo_background_socket, forgejo_background_host_uid, forgejo_background_credential_file, forgejo_review_credential_file, forgejo_merge_credential_file, factory_intake_secret_file, factory_publication_root; declaration encode_dashboard_config | [O01](../../slices/operator-administration.md#o01-first-boot-database-provisioning) | retained | Imports and module declarations wire cmd/soda-setup/src/config.rs into its current native target.; DashboardConfig: implement the current native setup and operator enrollment duty in config.rs.; encode_dashboard_config: implement the current native setup and operator enrollment duty in config.rs. — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-a775768ce0ee"></a>

## [cmd/soda-setup/src/forgejo.rs](../../../../../cmd/soda-setup/src/forgejo.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–380; current module/import/attribute shell; declaration ForgejoUser; fields id, admin; declaration ForgejoUserResponse; fields id, admin; declaration deserialize; declaration UserVisitor; declaration Value; declaration expecting; declaration visit_map; declaration forgejo_get_user; declaration forgejo_revoke_token; declaration forgejo_request; declaration forgejo_request_with_timeout; declaration decode_user; declaration http_tests; declaration read_request; declaration serve; declaration request_reads_chunked_and_eof_bodies; declaration request_returns_error_status_body_and_does_not_follow_redirects; declaration request_sends_the_literal_authorization_scheme; declaration request_caps_success_and_error_status_bodies; declaration request_rejects_malformed_and_truncated_responses_neutrally; declaration request_timeout_covers_headers_and_body_together; declaration request_rejects_plaintext_from_https_peer_neutrally | [O01](../../slices/operator-administration.md#o01-first-boot-database-provisioning) | retained | Imports and module declarations wire cmd/soda-setup/src/forgejo.rs into its current native target.; 23 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-736833f76716"></a>

## [cmd/soda-setup/src/json.rs](../../../../../cmd/soda-setup/src/json.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–31; current module/import/attribute shell; declaration GoFormatter; declaration write_string_fragment | [O01](../../slices/operator-administration.md#o01-first-boot-database-provisioning) | retained | Imports and module declarations wire cmd/soda-setup/src/json.rs into its current native target.; GoFormatter: implement the current native setup and operator enrollment duty in json.rs.; write_string_fragment: implement the current native setup and operator enrollment duty in json.rs. — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-55de8158374d"></a>

## [cmd/soda-setup/src/main.rs](../../../../../cmd/soda-setup/src/main.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–3, 5–8, 10; current module/import/attribute shell; declaration cli; declaration forgejo; declaration json; declaration origin; declaration secrets; declaration system | [O01](../../slices/operator-administration.md#o01-first-boot-database-provisioning) | retained | Imports and module declarations wire cmd/soda-setup/src/main.rs into its current native target.; 7 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |
| 4, 11–15; declaration config; declaration tests | [H04](../../slices/shared-supporting-slices.md#h04-configuration-and-filesystem-primitives) | retained | config: implement the current configuration and safe filesystem primitives duty in main.rs.; Collect embedded tests for main.rs. — current source cmd/soda-setup/src/main.rs; lines 4-4; module/caller wiring inspected; current source cmd/soda-setup/src/main.rs; lines 11-15; module/caller wiring inspected |
| 9, 16–18; declaration setup; declaration main | [O02](../../slices/operator-administration.md#o02-operator-identity-bootstrap) | retained | setup: implement the current operator identity bootstrap duty in main.rs.; Compose the current main.rs command/module and its native consumers. — current source cmd/soda-setup/src/main.rs; lines 9-9; module/caller wiring inspected; current source cmd/soda-setup/src/main.rs; lines 16-18; module/caller wiring inspected |

<a id="coverage-0dd230cff76a"></a>

## [cmd/soda-setup/src/origin.rs](../../../../../cmd/soda-setup/src/origin.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–134; current module/import/attribute shell; declaration base_url; declaration raw_port; declaration valid_percent_escapes; declaration credential; declaration admit_setup_paths | [O01](../../slices/operator-administration.md#o01-first-boot-database-provisioning) | retained | Imports and module declarations wire cmd/soda-setup/src/origin.rs into its current native target.; 6 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-21dac083007f"></a>

## [cmd/soda-setup/src/secrets.rs](../../../../../cmd/soda-setup/src/secrets.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–146; current module/import/attribute shell; declaration read_random_32; declaration hex_encode; declaration HEX; declaration base64_encode; declaration write_secret_file; declaration reuse_postgres_secrets; declaration provision_postgres_secrets | [O01](../../slices/operator-administration.md#o01-first-boot-database-provisioning) | retained | Imports and module declarations wire cmd/soda-setup/src/secrets.rs into its current native target.; 8 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-20e1a42fb6b9"></a>

## [cmd/soda-setup/src/setup.rs](../../../../../cmd/soda-setup/src/setup.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–107; current module/import/attribute shell; declaration setup; declaration read_random_32_inner; declaration write_setup_secret; declaration write_setup_config | [O01](../../slices/operator-administration.md#o01-first-boot-database-provisioning) | retained | Setup publication uses complete writes with honest close/error precedence. After publication, one revocation attempt whose response is lost is reported as unconfirmed; the published config remains and is neither automatically retried nor activated. Ten actual installer/setup checks passed for this bounded source scope; no installed qualification is claimed. |

<a id="coverage-4c136f5c5d6f"></a>

## [cmd/soda-setup/src/system.rs](../../../../../cmd/soda-setup/src/system.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–33; current module/import/attribute shell; declaration POSTGRES_SECRET_DIR; declaration FORGEJO_DB_USER; declaration SODA_SERVICE_GROUP; declaration POSTGRES_SOCKET_DIR; declaration FORGEJO_TIMEOUT; declaration FORGEJO_RESPONSE_LIMIT; declaration euid; declaration chown_path | [O01](../../slices/operator-administration.md#o01-first-boot-database-provisioning) | retained | Imports and module declarations wire cmd/soda-setup/src/system.rs into its current native target.; 9 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-ac10276ffbf0"></a>

## [cmd/soda-setup/src/tests/admission.rs](../../../../../cmd/soda-setup/src/tests/admission.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–73; current module/import/attribute shell; declaration admits_only_valid_setup_paths; declaration non_root_run_reports_operator_access; declaration flag_parsing_matches_go_setup | [O01](../../slices/operator-administration.md#o01-first-boot-database-provisioning) | retained | Imports and module declarations wire cmd/soda-setup/src/tests/admission.rs into its current native target.; 4 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-105d5aff9072"></a>

## [cmd/soda-setup/src/tests/encoding.rs](../../../../../cmd/soda-setup/src/tests/encoding.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–76; current module/import/attribute shell; declaration dashboard_config_bytes_match_go_encoder; declaration forgejo_user_decode_matches_client_rules; declaration config_strings_keep_go_html_and_line_separator_escapes; declaration base64_and_hex_vectors | [O01](../../slices/operator-administration.md#o01-first-boot-database-provisioning) | retained | Imports and module declarations wire cmd/soda-setup/src/tests/encoding.rs into its current native target.; 5 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-eaa5ea51a3d5"></a>

## [cmd/soda-setup/src/tests/fixtures.rs](../../../../../cmd/soda-setup/src/tests/fixtures.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–112; current module/import/attribute shell; declaration TEST_SEQ; declaration test_root; declaration Stub; fields url, calls; declaration stub_server; declaration write_token | [O01](../../slices/operator-administration.md#o01-first-boot-database-provisioning) | retained | Imports and module declarations wire cmd/soda-setup/src/tests/fixtures.rs into its current native target.; 6 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-51a4cab40ff0"></a>

## [cmd/soda-setup/src/tests/mod.rs](../../../../../cmd/soda-setup/src/tests/mod.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–5; declaration admission; declaration encoding; declaration fixtures; declaration postgres; declaration setup | [O01](../../slices/operator-administration.md#o01-first-boot-database-provisioning) | retained | admission: implement the current native setup and operator enrollment duty in mod.rs.; 5 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-424e1d37fa2a"></a>

## [cmd/soda-setup/src/tests/postgres.rs](../../../../../cmd/soda-setup/src/tests/postgres.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–131; current module/import/attribute shell; declaration provisions_all_four_postgres_files; declaration reuses_complete_pre_existing_secrets; declaration disagreeing_secrets_are_reported; declaration provision_only_reports_dsn_path | [O01](../../slices/operator-administration.md#o01-first-boot-database-provisioning) | retained | Imports and module declarations wire cmd/soda-setup/src/tests/postgres.rs into its current native target.; 5 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-02a7b4f50f9d"></a>

## [cmd/soda-setup/src/tests/setup.rs](../../../../../cmd/soda-setup/src/tests/setup.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–188; current module/import/attribute shell; declaration success_revokes_and_failure_keeps_retry_token; declaration preserves_pre_existing_secrets_and_cleans_own_key; declaration credential_boundary_matrix | [O01](../../slices/operator-administration.md#o01-first-boot-database-provisioning) | retained | Imports and module declarations wire cmd/soda-setup/src/tests/setup.rs into its current native target.; 4 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

## Removed predecessor navigation

These anchors identify earlier mapped source. Their paths no longer define this current map;
the current inventory and normalized predecessor ledger preserve replacement/retirement evidence.
No historical source body is counted as current coverage.

<a id="coverage-bcf0fc800d74"></a>
<a id="rustsoda-setupsrcmainrs-1"></a>

Former source `rust/soda-setup/src/main.rs`; consult its pinned earlier Git source and the current coverage disposition.

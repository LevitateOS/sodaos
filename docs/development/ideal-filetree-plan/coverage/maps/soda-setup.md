# Soda setup

[Responsibility map index](README.md) · [Coverage snapshot and limits](../README.md).
Page grouping is navigation; the slice IDs retain their individual review ownership.

<a id="coverage-bcf0fc800d74"></a>

<a id="rustsoda-setupsrcmainrs-1"></a>

## [rust/soda-setup/src/main.rs](../../../../../rust/soda-setup/src/main.rs)

Source assertions/fixtures were inspected for mapping only; no test execution, runtime or installed proof claimed. Large-file ledger retains committed declaration/member spans rather than grouping methods into one whole-file unit; responsibility follows the verified concern/branch mapping, not declaration count.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [O01](../../slices/operator-administration.md#o01-first-boot-database-provisioning) / active | 1–13 | First-boot PostgreSQL protected credential paths/constants; declarations/fields: `POSTGRES_SECRET_DIR`, `FORGEJO_DB_USER`, `SODA_SERVICE_GROUP`, `POSTGRES_SOCKET_DIR`, `FORGEJO_TIMEOUT`, `FORGEJO_RESPONSE_LIMIT`, `geteuid`, `chown`, `__errno_location`, `euid`, `chown_path`, `errno_message` |
| [O01](../../slices/operator-administration.md#o01-first-boot-database-provisioning) / active | 14–18 | First-boot PostgreSQL protected credential paths/constants; declaration/member POSTGRES_SECRET_DIR; declarations/fields: `POSTGRES_SECRET_DIR` |
| [O01](../../slices/operator-administration.md#o01-first-boot-database-provisioning) / active | 19–22 | First-boot PostgreSQL protected credential paths/constants; declaration/member FORGEJO_DB_USER; declarations/fields: `FORGEJO_DB_USER` |
| [O01](../../slices/operator-administration.md#o01-first-boot-database-provisioning) / active | 23–26 | First-boot PostgreSQL protected credential paths/constants; declaration/member SODA_SERVICE_GROUP; declarations/fields: `SODA_SERVICE_GROUP` |
| [O01](../../slices/operator-administration.md#o01-first-boot-database-provisioning) / active | 27 | First-boot PostgreSQL protected credential paths/constants; declaration/member POSTGRES_SOCKET_DIR; declarations/fields: `POSTGRES_SOCKET_DIR` |
| [O01](../../slices/operator-administration.md#o01-first-boot-database-provisioning) / active | 28 | First-boot PostgreSQL protected credential paths/constants; declaration/member FORGEJO_TIMEOUT; declarations/fields: `FORGEJO_TIMEOUT` |
| [O01](../../slices/operator-administration.md#o01-first-boot-database-provisioning) / active | 29–35 | First-boot PostgreSQL protected credential paths/constants; declaration/member FORGEJO_RESPONSE_LIMIT; declarations/fields: `FORGEJO_RESPONSE_LIMIT` |
| [O01](../../slices/operator-administration.md#o01-first-boot-database-provisioning) / active | 36 | First-boot PostgreSQL protected credential paths/constants; declaration/member geteuid; declarations/fields: `geteuid` |
| [O01](../../slices/operator-administration.md#o01-first-boot-database-provisioning) / active | 37–38 | First-boot PostgreSQL protected credential paths/constants; declaration/member chown; declarations/fields: `chown` |
| [O01](../../slices/operator-administration.md#o01-first-boot-database-provisioning) / active | 39–41 | First-boot PostgreSQL protected credential paths/constants; declaration/member __errno_location; declarations/fields: `__errno_location` |
| [O01](../../slices/operator-administration.md#o01-first-boot-database-provisioning) / active | 42–45 | First-boot PostgreSQL protected credential paths/constants; declaration/member euid; declarations/fields: `euid` |
| [O01](../../slices/operator-administration.md#o01-first-boot-database-provisioning) / active | 46–62 | First-boot PostgreSQL protected credential paths/constants; declaration/member chown_path; declarations/fields: `chown_path` |
| [O01](../../slices/operator-administration.md#o01-first-boot-database-provisioning) / active | 63–74 | First-boot PostgreSQL protected credential paths/constants; declaration/member errno_message; declarations/fields: `errno_message` |
| [O02](../../slices/operator-administration.md#o02-operator-identity-bootstrap) / active | 75–78 | Explicit operator setup entrypoint; declarations/fields: `main` |
| [O02](../../slices/operator-administration.md#o02-operator-identity-bootstrap) / active | 79–120 | Explicit operator setup entrypoint; declaration/member run; declarations/fields: `run` |
| [O02](../../slices/operator-administration.md#o02-operator-identity-bootstrap) / active | 121 | Operator commissioning input/options; declarations/fields: `SetupOpts` |
| [O02](../../slices/operator-administration.md#o02-operator-identity-bootstrap) / active | 122 | Operator commissioning input/options; declaration/member SetupOpts.forgejo_url; declarations/fields: `SetupOpts.forgejo_url` |
| [O02](../../slices/operator-administration.md#o02-operator-identity-bootstrap) / active | 123 | Operator commissioning input/options; declaration/member SetupOpts.forgejo_internal_url; declarations/fields: `SetupOpts.forgejo_internal_url` |
| [O02](../../slices/operator-administration.md#o02-operator-identity-bootstrap) / active | 124 | Operator commissioning input/options; declaration/member SetupOpts.token_file; declarations/fields: `SetupOpts.token_file` |
| [O02](../../slices/operator-administration.md#o02-operator-identity-bootstrap) / active | 125 | Operator commissioning input/options; declaration/member SetupOpts.out; declarations/fields: `SetupOpts.out` |
| [O02](../../slices/operator-administration.md#o02-operator-identity-bootstrap) / active | 126–129 | Operator commissioning input/options; declaration/member SetupOpts.provision_db_only; declarations/fields: `SetupOpts.provision_db_only` |
| [O02](../../slices/operator-administration.md#o02-operator-identity-bootstrap) / active | 130–134 | Operator commissioning input/options; declaration/member ParseOutcome; declarations/fields: `ParseOutcome` |
| [O02](../../slices/operator-administration.md#o02-operator-identity-bootstrap) / active | 135–140 | Operator commissioning input/options; declaration/member print_setup_usage; declarations/fields: `print_setup_usage` |
| [O02](../../slices/operator-administration.md#o02-operator-identity-bootstrap) / active | 141–161 | Operator commissioning input/options; declaration/member parse_bool_flag; declarations/fields: `parse_bool_flag` |
| [O02](../../slices/operator-administration.md#o02-operator-identity-bootstrap) / active | 162–174 | Operator commissioning input/options; declaration/member parse_go_bool; declarations/fields: `parse_go_bool` |
| [O02](../../slices/operator-administration.md#o02-operator-identity-bootstrap) / active | 175–178 | Operator commissioning input/options; declaration/member looks_like_flag; declarations/fields: `looks_like_flag` |
| [O02](../../slices/operator-administration.md#o02-operator-identity-bootstrap) / active | 179–246 | Operator commissioning input/options; declaration/member parse_args; declarations/fields: `parse_args` |
| [O02](../../slices/operator-administration.md#o02-operator-identity-bootstrap) / active | 247–288 | Operator commissioning input/options; declaration/member base_url; declarations/fields: `base_url` |
| [O02](../../slices/operator-administration.md#o02-operator-identity-bootstrap) / active | 289–309 | Operator commissioning input/options; declaration/member valid_percent_escapes; declarations/fields: `valid_percent_escapes` |
| [O02](../../slices/operator-administration.md#o02-operator-identity-bootstrap) / active | 310–333 | Native Forgejo operator credential/token admission and revocation; declarations/fields: `credential` |
| [O02](../../slices/operator-administration.md#o02-operator-identity-bootstrap) / active | 334 | Native Forgejo operator credential/token admission and revocation; declaration/member ForgejoUser; declarations/fields: `ForgejoUser` |
| [O02](../../slices/operator-administration.md#o02-operator-identity-bootstrap) / active | 335 | Native Forgejo operator credential/token admission and revocation; declaration/member ForgejoUser.id; declarations/fields: `ForgejoUser.id` |
| [O02](../../slices/operator-administration.md#o02-operator-identity-bootstrap) / active | 336–338 | Native Forgejo operator credential/token admission and revocation; declaration/member ForgejoUser.admin; declarations/fields: `ForgejoUser.admin` |
| [O02](../../slices/operator-administration.md#o02-operator-identity-bootstrap) / active | 339–346 | Native Forgejo operator credential/token admission and revocation; declaration/member forgejo_get_user; declarations/fields: `forgejo_get_user` |
| [O02](../../slices/operator-administration.md#o02-operator-identity-bootstrap) / active | 347–354 | Native Forgejo operator credential/token admission and revocation; declaration/member forgejo_revoke_token; declarations/fields: `forgejo_revoke_token` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 355–399 | Bounded native HTTP bootstrap transport; declarations/fields: `forgejo_request` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 400–407 | Bounded native HTTP bootstrap transport; declaration/member dns_lookup; declarations/fields: `dns_lookup` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 408–465 | Bounded native HTTP bootstrap transport; declaration/member read_http_response; declarations/fields: `read_http_response` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 466–486 | Bounded native HTTP bootstrap transport; declaration/member read_to_end_limited; declarations/fields: `read_to_end_limited` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 487–509 | Bounded native HTTP bootstrap transport; declaration/member read_chunked; declarations/fields: `read_chunked` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 510–532 | Bounded native HTTP bootstrap transport; declaration/member read_line; declarations/fields: `read_line` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 533–566 | Native bootstrap JSON/response parser; declarations/fields: `decode_user` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 567–576 | Native bootstrap JSON/response parser; declaration/member JsonValue; declarations/fields: `JsonValue` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 577–590 | Native bootstrap JSON/response parser; declaration/member parse; declarations/fields: `parse` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 591–597 | Native bootstrap JSON/response parser; declaration/member as_object; declarations/fields: `as_object` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 598–604 | Native bootstrap JSON/response parser; declaration/member as_bool; declarations/fields: `as_bool` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 605–618 | Native bootstrap JSON/response parser; declaration/member as_i64; declarations/fields: `as_i64` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 619 | Native bootstrap JSON/response parser; declaration/member JsonParser; declarations/fields: `JsonParser` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 620 | Native bootstrap JSON/response parser; declaration/member JsonParser.bytes; declarations/fields: `JsonParser.bytes` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 621–624 | Native bootstrap JSON/response parser; declaration/member JsonParser.pos; declarations/fields: `JsonParser.pos` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 625–632 | Native bootstrap JSON/response parser; declaration/member skip_ws; declarations/fields: `skip_ws` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 633–647 | Native bootstrap JSON/response parser; declaration/member parse_value; declarations/fields: `parse_value` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 648–656 | Native bootstrap JSON/response parser; declaration/member parse_literal; declarations/fields: `parse_literal` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 657–693 | Native bootstrap JSON/response parser; declaration/member parse_object; declarations/fields: `parse_object` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 694–719 | Native bootstrap JSON/response parser; declaration/member parse_array; declarations/fields: `parse_array` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 720–775 | Native bootstrap JSON/response parser; declaration/member parse_string; declarations/fields: `parse_string` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 776–819 | Native bootstrap JSON/response parser; declaration/member parse_number; declarations/fields: `parse_number` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 820–844 | Native bootstrap JSON/response parser; declaration/member push_json_string; declarations/fields: `push_json_string` |
| [O02](../../slices/operator-administration.md#o02-operator-identity-bootstrap) / active | 845–906 | Initial service configuration rendering; declarations/fields: `encode_dashboard_config` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 907–915 | Random secret/base64 primitives; declarations/fields: `read_random_32` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 916 | Random secret/base64 primitives; declaration/member hex_encode; declarations/fields: `hex_encode` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 917–925 | Random secret/base64 primitives; declaration/member HEX; declarations/fields: `HEX` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 926 | Random secret/base64 primitives; declaration/member base64_encode; declarations/fields: `base64_encode` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 927–949 | Random secret/base64 primitives; declaration/member ALPHABET; declarations/fields: `ALPHABET` |
| [H04](../../slices/shared-supporting-slices.md#h04-configuration-and-filesystem-primitives) / active | 950–969 | Protected secret writes; declarations/fields: `write_secret_file` |
| [O01](../../slices/operator-administration.md#o01-first-boot-database-provisioning) / active | 970–996 | Preserve existing complete database secrets; initial provisioning only; declarations/fields: `reuse_postgres_secrets` |
| [O01](../../slices/operator-administration.md#o01-first-boot-database-provisioning) / active | 997–1068 | Preserve existing complete database secrets; initial provisioning only; declaration/member provision_postgres_secrets; declarations/fields: `provision_postgres_secrets` |
| [O02](../../slices/operator-administration.md#o02-operator-identity-bootstrap) / active | 1069–1089 | Initial operator setup path admission and commissioning; declarations/fields: `admit_setup_paths` |
| [O02](../../slices/operator-administration.md#o02-operator-identity-bootstrap) / active | 1090–1156 | Initial operator setup path admission and commissioning; declaration/member setup; declarations/fields: `setup` |
| [O02](../../slices/operator-administration.md#o02-operator-identity-bootstrap) / active | 1157–1163 | Initial operator setup path admission and commissioning; declaration/member read_random_32_inner; declarations/fields: `read_random_32_inner` |
| [H04](../../slices/shared-supporting-slices.md#h04-configuration-and-filesystem-primitives) / active | 1164–1176 | Protected native configuration file operations; declarations/fields: `write_setup_secret` |
| [H04](../../slices/shared-supporting-slices.md#h04-configuration-and-filesystem-primitives) / active | 1177–1189 | Protected native configuration file operations; declaration/member write_setup_config; declarations/fields: `write_setup_config` |
| [H04](../../slices/shared-supporting-slices.md#h04-configuration-and-filesystem-primitives) / active | 1190–1197 | Protected native configuration file operations; declaration/member tests; declarations/fields: `tests` |
| [H04](../../slices/shared-supporting-slices.md#h04-configuration-and-filesystem-primitives) / active | 1198–1199 | Protected native configuration file operations; declaration/member TEST_SEQ; declarations/fields: `TEST_SEQ` |
| [O02](../../slices/operator-administration.md#o02-operator-identity-bootstrap) / active | 1200–1207, 1462–1582, 1678–1686 | Initial setup source fixtures/assertions; declarations/fields: `test_root`, `credential_boundary_matrix`, `non_root_run_reports_operator_access` |
| [O02](../../slices/operator-administration.md#o02-operator-identity-bootstrap) / active | 1208 | Initial setup source fixtures/assertions; declaration/member Stub; declarations/fields: `Stub` |
| [O02](../../slices/operator-administration.md#o02-operator-identity-bootstrap) / active | 1209 | Initial setup source fixtures/assertions; declaration/member Stub.url; declarations/fields: `Stub.url` |
| [O02](../../slices/operator-administration.md#o02-operator-identity-bootstrap) / active | 1210–1212 | Initial setup source fixtures/assertions; declaration/member Stub.calls; declarations/fields: `Stub.calls` |
| [O02](../../slices/operator-administration.md#o02-operator-identity-bootstrap) / active | 1213–1294 | Initial setup source fixtures/assertions; declaration/member stub_server; declarations/fields: `stub_server` |
| [O02](../../slices/operator-administration.md#o02-operator-identity-bootstrap) / active | 1295–1302 | Initial setup source fixtures/assertions; declaration/member write_token; declarations/fields: `write_token` |
| [O02](../../slices/operator-administration.md#o02-operator-identity-bootstrap) / active | 1303–1331 | Initial setup source fixtures/assertions; declaration/member success_revokes_and_failure_keeps_retry_token; declarations/fields: `success_revokes_and_failure_keeps_retry_token` |
| [O01](../../slices/operator-administration.md#o01-first-boot-database-provisioning) / active | 1332–1382, 1664–1677 | Source assertion of First-boot database provisioning; declarations/fields: `provisions_all_four_postgres_files`, `provision_only_reports_dsn_path` |
| [O01](../../slices/operator-administration.md#o01-first-boot-database-provisioning) / active | 1383–1416 | Source assertion of First-boot database provisioning; declaration/member preserves_pre_existing_secrets_and_cleans_own_key; declarations/fields: `preserves_pre_existing_secrets_and_cleans_own_key` |
| [O01](../../slices/operator-administration.md#o01-first-boot-database-provisioning) / active | 1417–1461 | Source assertion of First-boot database provisioning; declaration/member reuses_complete_pre_existing_secrets; declarations/fields: `reuses_complete_pre_existing_secrets` |
| [O02](../../slices/operator-administration.md#o02-operator-identity-bootstrap) / active | 1583–1600 | Initial setup source fixtures/assertions; declaration/member dashboard_config_bytes_match_go_encoder; declarations/fields: `dashboard_config_bytes_match_go_encoder` |
| [O02](../../slices/operator-administration.md#o02-operator-identity-bootstrap) / active | 1601–1626 | Initial setup source fixtures/assertions; declaration/member admits_only_valid_setup_paths; declarations/fields: `admits_only_valid_setup_paths` |
| [O02](../../slices/operator-administration.md#o02-operator-identity-bootstrap) / active | 1627–1647 | Initial setup source fixtures/assertions; declaration/member forgejo_user_decode_matches_client_rules; declarations/fields: `forgejo_user_decode_matches_client_rules` |
| [O02](../../slices/operator-administration.md#o02-operator-identity-bootstrap) / active | 1648–1663 | Initial setup source fixtures/assertions; declaration/member disagreeing_secrets_are_reported; declarations/fields: `disagreeing_secrets_are_reported` |
| [O02](../../slices/operator-administration.md#o02-operator-identity-bootstrap) / active | 1687–1714 | Initial setup source fixtures/assertions; declaration/member flag_parsing_matches_go_setup; declarations/fields: `flag_parsing_matches_go_setup` |
| [O02](../../slices/operator-administration.md#o02-operator-identity-bootstrap) / active | 1715–1721 | Initial setup source fixtures/assertions; declaration/member json_string_escapes_match_go; declarations/fields: `json_string_escapes_match_go` |
| [O02](../../slices/operator-administration.md#o02-operator-identity-bootstrap) / active | 1722–1731 | Initial setup source fixtures/assertions; declaration/member base64_and_hex_vectors; declarations/fields: `base64_and_hex_vectors` |


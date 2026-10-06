# Identity providers

[Responsibility map index](README.md) · [Coverage snapshot and limits](../README.md).
Page grouping is navigation; the slice IDs retain their individual review ownership.

R02 re-audit COMPLETE @HEAD: codex.rs split re-mapped (mod+config+protocol+tests), muse.rs re-mapped (mod+tests), providers/mod.rs verified clean. All rows machine-verified against current bytes.

<a id="coverage-533cd08fa569"></a>

<a id="rustidentity-providerssrccodexrs-1"></a>

## [cmd/soda-identity/src/providers/codex/mod.rs](../../../../../cmd/soda-identity/src/providers/codex/mod.rs)

Re-audit @HEAD: pre-A05/A06 `codex.rs` split into `mod.rs` + `config.rs` + `protocol.rs` + `tests.rs`; rows re-mapped declaration-by-declaration to current bytes.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [I07](../../slices/identity-brokering.md#i07-codex-adapter) / active | 1–20 | Codex device enrollment, account discovery and auth-file capture; declarations/fields: `VERSION`, `Provider`, `Provider.config`, `new`, `start`, `create_session`, `split_env`, `Inner`, `Inner.state`, `Inner.write`, `Inner.child`, `Inner.replies`, `Inner.next`, `Inner.root`, `Inner.done`, `Inner.finished`, `Inner.cancelled`, `Inner.closed`, `Session`, `Session.inner`, `snapshot`, `cancel`, `initialize`, `start_device`, `Wire`, `Wire.wire_type`, `Wire.login_id`, `Wire.verification_url`, `Wire.user_code`, `finish`, `account`, `Response`, `Response.account`, `Account`, `Account.account_type`, `Account.email`, `Account.plan`, `credential_file`, `stop`, `close`, `tests` |
| [I07](../../slices/identity-brokering.md#i07-codex-adapter) / active | 21 | Codex device enrollment, account discovery and auth-file capture; declaration/member VERSION; declarations/fields: `VERSION` |
| [I07](../../slices/identity-brokering.md#i07-codex-adapter) / active | 23–25 | Codex device enrollment, account discovery and auth-file capture; declaration/member Provider; declarations/fields: `Provider` |
| [I07](../../slices/identity-brokering.md#i07-codex-adapter) / active | 24 | Codex device enrollment, account discovery and auth-file capture; declaration/member Provider.config; declarations/fields: `Provider.config` |
| [I07](../../slices/identity-brokering.md#i07-codex-adapter) / active | 28–40 | Codex device enrollment, account discovery and auth-file capture; declaration/member new; declarations/fields: `new` |
| [I07](../../slices/identity-brokering.md#i07-codex-adapter) / active | 45–63 | Codex device enrollment, account discovery and auth-file capture; declaration/member start; declarations/fields: `start` |
| [I07](../../slices/identity-brokering.md#i07-codex-adapter) / active | 65–150 | Codex device enrollment, account discovery and auth-file capture; declaration/member create_session; declarations/fields: `create_session` |
| [I07](../../slices/identity-brokering.md#i07-codex-adapter) / active | 152–154 | Codex device enrollment, account discovery and auth-file capture; declaration/member split_env; declarations/fields: `split_env` |
| [I07](../../slices/identity-brokering.md#i07-codex-adapter) / active | 156–167 | Codex device enrollment, account discovery and auth-file capture; declaration/member Inner; declarations/fields: `Inner` |
| [I07](../../slices/identity-brokering.md#i07-codex-adapter) / active | 157 | Codex device enrollment, account discovery and auth-file capture; declaration/member Inner.state; declarations/fields: `Inner.state` |
| [I07](../../slices/identity-brokering.md#i07-codex-adapter) / active | 158 | Codex device enrollment, account discovery and auth-file capture; declaration/member Inner.write; declarations/fields: `Inner.write` |
| [I07](../../slices/identity-brokering.md#i07-codex-adapter) / active | 159 | Codex device enrollment, account discovery and auth-file capture; declaration/member Inner.child; declarations/fields: `Inner.child` |
| [I07](../../slices/identity-brokering.md#i07-codex-adapter) / active | 160 | Codex device enrollment, account discovery and auth-file capture; declaration/member Inner.replies; declarations/fields: `Inner.replies` |
| [I07](../../slices/identity-brokering.md#i07-codex-adapter) / active | 161 | Codex device enrollment, account discovery and auth-file capture; declaration/member Inner.next; declarations/fields: `Inner.next` |
| [I07](../../slices/identity-brokering.md#i07-codex-adapter) / active | 162 | Codex device enrollment, account discovery and auth-file capture; declaration/member Inner.root; declarations/fields: `Inner.root` |
| [I07](../../slices/identity-brokering.md#i07-codex-adapter) / active | 163 | Codex device enrollment, account discovery and auth-file capture; declaration/member Inner.done; declarations/fields: `Inner.done` |
| [I07](../../slices/identity-brokering.md#i07-codex-adapter) / active | 164 | Codex device enrollment, account discovery and auth-file capture; declaration/member Inner.finished; declarations/fields: `Inner.finished` |
| [I07](../../slices/identity-brokering.md#i07-codex-adapter) / active | 165 | Codex device enrollment, account discovery and auth-file capture; declaration/member Inner.cancelled; declarations/fields: `Inner.cancelled` |
| [I07](../../slices/identity-brokering.md#i07-codex-adapter) / active | 166 | Codex device enrollment, account discovery and auth-file capture; declaration/member Inner.closed; declarations/fields: `Inner.closed` |
| [I07](../../slices/identity-brokering.md#i07-codex-adapter) / active | 169–172 | Codex device enrollment, account discovery and auth-file capture; declaration/member Session; declarations/fields: `Session` |
| [I07](../../slices/identity-brokering.md#i07-codex-adapter) / active | 171 | Codex device enrollment, account discovery and auth-file capture; declaration/member Session.inner; declarations/fields: `Session.inner` |
| [I07](../../slices/identity-brokering.md#i07-codex-adapter) / active | 175–177 | Codex device enrollment, account discovery and auth-file capture; declaration/member snapshot; declarations/fields: `snapshot` |
| [I07](../../slices/identity-brokering.md#i07-codex-adapter) / active | 180–182 | Codex device enrollment, account discovery and auth-file capture; declaration/member cancel; declarations/fields: `cancel` |
| [I07](../../slices/identity-brokering.md#i07-codex-adapter) / active | 184–189 | Codex device enrollment, account discovery and auth-file capture; declaration/member initialize; declarations/fields: `initialize` |
| [I07](../../slices/identity-brokering.md#i07-codex-adapter) / active | 191–196 | Codex device enrollment, account discovery and auth-file capture; declaration/member start_device; declarations/fields: `start_device` |
| [I07](../../slices/identity-brokering.md#i07-codex-adapter) / active | 197–222 | Codex device enrollment, account discovery and auth-file capture; declaration/member Wire; declarations/fields: `Wire` |
| [I07](../../slices/identity-brokering.md#i07-codex-adapter) / active | 200 | Codex device enrollment, account discovery and auth-file capture; declaration/member Wire.wire_type; declarations/fields: `Wire.wire_type` |
| [I07](../../slices/identity-brokering.md#i07-codex-adapter) / active | 202 | Codex device enrollment, account discovery and auth-file capture; declaration/member Wire.login_id; declarations/fields: `Wire.login_id` |
| [I07](../../slices/identity-brokering.md#i07-codex-adapter) / active | 204 | Codex device enrollment, account discovery and auth-file capture; declaration/member Wire.verification_url; declarations/fields: `Wire.verification_url` |
| [I07](../../slices/identity-brokering.md#i07-codex-adapter) / active | 206 | Codex device enrollment, account discovery and auth-file capture; declaration/member Wire.user_code; declarations/fields: `Wire.user_code` |
| [I07](../../slices/identity-brokering.md#i07-codex-adapter) / active | 224–232 | Codex device enrollment, account discovery and auth-file capture; declaration/member finish; declarations/fields: `finish` |
| [I07](../../slices/identity-brokering.md#i07-codex-adapter) / active | 234–239 | Codex device enrollment, account discovery and auth-file capture; declaration/member account; declarations/fields: `account` |
| [I07](../../slices/identity-brokering.md#i07-codex-adapter) / active | 240–243 | Codex device enrollment, account discovery and auth-file capture; declaration/member Response; declarations/fields: `Response` |
| [I07](../../slices/identity-brokering.md#i07-codex-adapter) / active | 242 | Codex device enrollment, account discovery and auth-file capture; declaration/member Response.account; declarations/fields: `Response.account` |
| [I07](../../slices/identity-brokering.md#i07-codex-adapter) / active | 244–265 | Codex device enrollment, account discovery and auth-file capture; declaration/member Account; declarations/fields: `Account` |
| [I07](../../slices/identity-brokering.md#i07-codex-adapter) / active | 247 | Codex device enrollment, account discovery and auth-file capture; declaration/member Account.account_type; declarations/fields: `Account.account_type` |
| [I07](../../slices/identity-brokering.md#i07-codex-adapter) / active | 248 | Codex device enrollment, account discovery and auth-file capture; declaration/member Account.email; declarations/fields: `Account.email` |
| [I07](../../slices/identity-brokering.md#i07-codex-adapter) / active | 250 | Codex device enrollment, account discovery and auth-file capture; declaration/member Account.plan; declarations/fields: `Account.plan` |
| [I07](../../slices/identity-brokering.md#i07-codex-adapter) / active | 267–288 | Codex device enrollment, account discovery and auth-file capture; declaration/member credential_file; declarations/fields: `credential_file` |
| [I07](../../slices/identity-brokering.md#i07-codex-adapter) / active | 290–303 | Codex device enrollment, account discovery and auth-file capture; declaration/member stop; declarations/fields: `stop` |
| [I07](../../slices/identity-brokering.md#i07-codex-adapter) / active | 305–318 | Codex device enrollment, account discovery and auth-file capture; declaration/member close; declarations/fields: `close` |
| [I07](../../slices/identity-brokering.md#i07-codex-adapter) / active | 320–322 | Codex device enrollment, account discovery and auth-file capture; declaration/member tests; declarations/fields: `tests` |

## [cmd/soda-identity/src/providers/codex/config.rs](../../../../../cmd/soda-identity/src/providers/codex/config.rs)

Re-audit @HEAD: split sibling; spans re-audited against current bytes.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [I07](../../slices/identity-brokering.md#i07-codex-adapter) / active | 1–7 | Codex device enrollment, account discovery and auth-file capture; declarations/fields: `Config`, `Config.binary`, `Config.version`, `Config.sha256`, `Config.root`, `environment`, `filter_env`, `validate_config`, `check_binary`, `check_version`, `from` |
| [I07](../../slices/identity-brokering.md#i07-codex-adapter) / active | 8–14 | Codex device enrollment, account discovery and auth-file capture; declaration/member Config; declarations/fields: `Config` |
| [I07](../../slices/identity-brokering.md#i07-codex-adapter) / active | 10 | Codex device enrollment, account discovery and auth-file capture; declaration/member Config.binary; declarations/fields: `Config.binary` |
| [I07](../../slices/identity-brokering.md#i07-codex-adapter) / active | 11 | Codex device enrollment, account discovery and auth-file capture; declaration/member Config.version; declarations/fields: `Config.version` |
| [I07](../../slices/identity-brokering.md#i07-codex-adapter) / active | 12 | Codex device enrollment, account discovery and auth-file capture; declaration/member Config.sha256; declarations/fields: `Config.sha256` |
| [I07](../../slices/identity-brokering.md#i07-codex-adapter) / active | 13 | Codex device enrollment, account discovery and auth-file capture; declaration/member Config.root; declarations/fields: `Config.root` |
| [I07](../../slices/identity-brokering.md#i07-codex-adapter) / active | 16–18 | Codex device enrollment, account discovery and auth-file capture; declaration/member environment; declarations/fields: `environment` |
| [I07](../../slices/identity-brokering.md#i07-codex-adapter) / active | 20–38 | Codex device enrollment, account discovery and auth-file capture; declaration/member filter_env; declarations/fields: `filter_env` |
| [I07](../../slices/identity-brokering.md#i07-codex-adapter) / active | 40–49 | Codex device enrollment, account discovery and auth-file capture; declaration/member validate_config; declarations/fields: `validate_config` |
| [I07](../../slices/identity-brokering.md#i07-codex-adapter) / active | 51–57 | Codex device enrollment, account discovery and auth-file capture; declaration/member check_binary; declarations/fields: `check_binary` |
| [I07](../../slices/identity-brokering.md#i07-codex-adapter) / active | 59–72 | Codex device enrollment, account discovery and auth-file capture; declaration/member check_version; declarations/fields: `check_version` |
| [I07](../../slices/identity-brokering.md#i07-codex-adapter) / active | 75–78 | Codex device enrollment, account discovery and auth-file capture; declaration/member from; declarations/fields: `from` |

## [cmd/soda-identity/src/providers/codex/protocol.rs](../../../../../cmd/soda-identity/src/providers/codex/protocol.rs)

Re-audit @HEAD: split sibling; spans re-audited against current bytes.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [I07](../../slices/identity-brokering.md#i07-codex-adapter) / active | 1–9 | Codex device enrollment, account discovery and auth-file capture; declarations/fields: `Message`, `Message.id`, `Message.method`, `Message.result`, `Message.error`, `Message.params`, `send`, `call`, `protocol_error`, `Detail`, `Detail.message`, `read_loop`, `notify`, `Event`, `Event.login_id`, `Event.success` |
| [I07](../../slices/identity-brokering.md#i07-codex-adapter) / active | 10–22 | Codex device enrollment, account discovery and auth-file capture; declaration/member Message; declarations/fields: `Message` |
| [I07](../../slices/identity-brokering.md#i07-codex-adapter) / active | 13 | Codex device enrollment, account discovery and auth-file capture; declaration/member Message.id; declarations/fields: `Message.id` |
| [I07](../../slices/identity-brokering.md#i07-codex-adapter) / active | 15 | Codex device enrollment, account discovery and auth-file capture; declaration/member Message.method; declarations/fields: `Message.method` |
| [I07](../../slices/identity-brokering.md#i07-codex-adapter) / active | 17 | Codex device enrollment, account discovery and auth-file capture; declaration/member Message.result; declarations/fields: `Message.result` |
| [I07](../../slices/identity-brokering.md#i07-codex-adapter) / active | 19 | Codex device enrollment, account discovery and auth-file capture; declaration/member Message.error; declarations/fields: `Message.error` |
| [I07](../../slices/identity-brokering.md#i07-codex-adapter) / active | 21 | Codex device enrollment, account discovery and auth-file capture; declaration/member Message.params; declarations/fields: `Message.params` |
| [I07](../../slices/identity-brokering.md#i07-codex-adapter) / active | 25–34 | Codex device enrollment, account discovery and auth-file capture; declaration/member send; declarations/fields: `send` |
| [I07](../../slices/identity-brokering.md#i07-codex-adapter) / active | 36 | Codex device enrollment, account discovery and auth-file capture; declaration/member call; declarations/fields: `call` |
| [I07](../../slices/identity-brokering.md#i07-codex-adapter) / active | 87–105 | Codex device enrollment, account discovery and auth-file capture; declaration/member protocol_error; declarations/fields: `protocol_error` |
| [I07](../../slices/identity-brokering.md#i07-codex-adapter) / active | 88–105 | Codex device enrollment, account discovery and auth-file capture; declaration/member Detail; declarations/fields: `Detail` |
| [I07](../../slices/identity-brokering.md#i07-codex-adapter) / active | 91 | Codex device enrollment, account discovery and auth-file capture; declaration/member Detail.message; declarations/fields: `Detail.message` |
| [I07](../../slices/identity-brokering.md#i07-codex-adapter) / active | 107–162 | Codex device enrollment, account discovery and auth-file capture; declaration/member read_loop; declarations/fields: `read_loop` |
| [I07](../../slices/identity-brokering.md#i07-codex-adapter) / active | 164–187 | Codex device enrollment, account discovery and auth-file capture; declaration/member notify; declarations/fields: `notify` |
| [I07](../../slices/identity-brokering.md#i07-codex-adapter) / active | 168–187 | Codex device enrollment, account discovery and auth-file capture; declaration/member Event; declarations/fields: `Event` |
| [I07](../../slices/identity-brokering.md#i07-codex-adapter) / active | 171 | Codex device enrollment, account discovery and auth-file capture; declaration/member Event.login_id; declarations/fields: `Event.login_id` |
| [I07](../../slices/identity-brokering.md#i07-codex-adapter) / active | 172 | Codex device enrollment, account discovery and auth-file capture; declaration/member Event.success; declarations/fields: `Event.success` |

## [cmd/soda-identity/src/providers/codex/tests.rs](../../../../../cmd/soda-identity/src/providers/codex/tests.rs)

Re-audit @HEAD: split sibling; spans re-audited against current bytes.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [I07](../../slices/identity-brokering.md#i07-codex-adapter) / active | 1–6 | Codex device enrollment, account discovery and auth-file capture; declarations/fields: `environment_filters_provider_credentials`, `config_validation_matches_go`, `protocol_fixture`, `fixture_provider`, `managed_enrollment_persists_only_after_process_stop`, `cancel_removes_unfinished_enrollment`, `device_disabled_maps_to_actionable_error` |
| [I07](../../slices/identity-brokering.md#i07-codex-adapter) / active | 8–28 | Codex device enrollment, account discovery and auth-file capture; declaration/member environment_filters_provider_credentials; declarations/fields: `environment_filters_provider_credentials` |
| [I07](../../slices/identity-brokering.md#i07-codex-adapter) / active | 30–62 | Codex device enrollment, account discovery and auth-file capture; declaration/member config_validation_matches_go; declarations/fields: `config_validation_matches_go` |
| [I07](../../slices/identity-brokering.md#i07-codex-adapter) / active | 64–76 | Codex device enrollment, account discovery and auth-file capture; declaration/member protocol_fixture; declarations/fields: `protocol_fixture` |
| [I07](../../slices/identity-brokering.md#i07-codex-adapter) / active | 78–90 | Codex device enrollment, account discovery and auth-file capture; declaration/member fixture_provider; declarations/fields: `fixture_provider` |
| [I07](../../slices/identity-brokering.md#i07-codex-adapter) / active | 92–116 | Codex device enrollment, account discovery and auth-file capture; declaration/member managed_enrollment_persists_only_after_process_stop; declarations/fields: `managed_enrollment_persists_only_after_process_stop` |
| [I07](../../slices/identity-brokering.md#i07-codex-adapter) / active | 118–127 | Codex device enrollment, account discovery and auth-file capture; declaration/member cancel_removes_unfinished_enrollment; declarations/fields: `cancel_removes_unfinished_enrollment` |
| [I07](../../slices/identity-brokering.md#i07-codex-adapter) / active | 129–137 | Codex device enrollment, account discovery and auth-file capture; declaration/member device_disabled_maps_to_actionable_error; declarations/fields: `device_disabled_maps_to_actionable_error` |

## [cmd/soda-identity/src/providers/mod.rs](../../../../../cmd/soda-identity/src/providers/mod.rs)

Re-audit @HEAD: every row verified declaration-by-declaration against current bytes; no drift.

Source assertions/fixtures were inspected for mapping only; no test execution, runtime or installed proof claimed. Public module wiring is mapped separately; module exposure does not create a new process boundary.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [I01](../../slices/identity-brokering.md#i01-enrollment-and-owner-consent) / active | 1–7, 25–238 | Provider enrollment interface and private native enrollment sessions; declarations/fields: `Kind`, `Error`, `denied`, `uncertain`, `failed`, `kind`, `is_denied`, `is_uncertain`, `fmt`, `from`, `enrollment_tempdir`, `random_bytes`, `private_tmpfs`, `TestDir`, `new`, `path`, `drop`, `run_capture` |
| [I07](../../slices/identity-brokering.md#i07-codex-adapter) / active | 8 | Codex native enrollment adapter module; declarations/fields: `codex` |
| [I08](../../slices/identity-brokering.md#i08-muse-adapter) / active | 9 | Muse native enrollment adapter module; declarations/fields: `muse` |
| [I01](../../slices/identity-brokering.md#i01-enrollment-and-owner-consent) / active | 10 | Module wiring: sha256; Enrollment and owner consent; declarations/fields: `sha256` |
| [I01](../../slices/identity-brokering.md#i01-enrollment-and-owner-consent) / active | 11–24 | Module wiring: types; Enrollment and owner consent; declarations/fields: `types` |

<a id="coverage-e008c55b400c"></a>

<a id="rustidentity-providerssrcmusers-1"></a>

## [cmd/soda-identity/src/providers/muse/mod.rs](../../../../../cmd/soda-identity/src/providers/muse/mod.rs)

Re-audit @HEAD: pre-A05/A06 `muse.rs` moved to `muse/mod.rs` + `muse/tests.rs`; rows re-mapped declaration-by-declaration to current bytes.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [I08](../../slices/identity-brokering.md#i08-muse-adapter) / active | 1–16 | Muse device enrollment and immutable subscription extraction; declarations/fields: `VERSION`, `VERSION_LINE`, `DEVICE_PREFIX`, `Config`, `Config.binary`, `Config.version`, `Config.sha256`, `Config.root`, `Provider`, `Provider.config`, `new`, `start`, `split_env`, `Inner`, `Inner.state`, `Inner.child`, `Inner.root`, `Inner.done`, `Inner.finished`, `Session`, `Session.inner`, `snapshot`, `finish`, `close`, `read_loop`, `complete`, `scan_device_url`, `environment`, `validate_config`, `check_binary`, `check_version`, `credential_valid`, `Wire`, `Wire.schema_version`, `Wire.providers`, `Providers`, `Providers.meta`, `Meta`, `Meta.access_token`, `Meta.api_key`, `Meta.api_base_url`, `Meta.mechanism`, `Meta.obtained_via`, `credential_file`, `tests` |
| [I08](../../slices/identity-brokering.md#i08-muse-adapter) / active | 18 | Muse device enrollment and immutable subscription extraction; declaration/member VERSION; declarations/fields: `VERSION` |
| [I08](../../slices/identity-brokering.md#i08-muse-adapter) / active | 19 | Muse device enrollment and immutable subscription extraction; declaration/member VERSION_LINE; declarations/fields: `VERSION_LINE` |
| [I08](../../slices/identity-brokering.md#i08-muse-adapter) / active | 20 | Muse device enrollment and immutable subscription extraction; declaration/member DEVICE_PREFIX; declarations/fields: `DEVICE_PREFIX` |
| [I08](../../slices/identity-brokering.md#i08-muse-adapter) / active | 22–28 | Muse device enrollment and immutable subscription extraction; declaration/member Config; declarations/fields: `Config` |
| [I08](../../slices/identity-brokering.md#i08-muse-adapter) / active | 24 | Muse device enrollment and immutable subscription extraction; declaration/member Config.binary; declarations/fields: `Config.binary` |
| [I08](../../slices/identity-brokering.md#i08-muse-adapter) / active | 25 | Muse device enrollment and immutable subscription extraction; declaration/member Config.version; declarations/fields: `Config.version` |
| [I08](../../slices/identity-brokering.md#i08-muse-adapter) / active | 26 | Muse device enrollment and immutable subscription extraction; declaration/member Config.sha256; declarations/fields: `Config.sha256` |
| [I08](../../slices/identity-brokering.md#i08-muse-adapter) / active | 27 | Muse device enrollment and immutable subscription extraction; declaration/member Config.root; declarations/fields: `Config.root` |
| [I08](../../slices/identity-brokering.md#i08-muse-adapter) / active | 30–32 | Muse device enrollment and immutable subscription extraction; declaration/member Provider; declarations/fields: `Provider` |
| [I08](../../slices/identity-brokering.md#i08-muse-adapter) / active | 31 | Muse device enrollment and immutable subscription extraction; declaration/member Provider.config; declarations/fields: `Provider.config` |
| [I08](../../slices/identity-brokering.md#i08-muse-adapter) / active | 35–40 | Muse device enrollment and immutable subscription extraction; declaration/member new; declarations/fields: `new` |
| [I08](../../slices/identity-brokering.md#i08-muse-adapter) / active | 45–105 | Muse device enrollment and immutable subscription extraction; declaration/member start; declarations/fields: `start` |
| [I08](../../slices/identity-brokering.md#i08-muse-adapter) / active | 107–109 | Muse device enrollment and immutable subscription extraction; declaration/member split_env; declarations/fields: `split_env` |
| [I08](../../slices/identity-brokering.md#i08-muse-adapter) / active | 111–117 | Muse device enrollment and immutable subscription extraction; declaration/member Inner; declarations/fields: `Inner` |
| [I08](../../slices/identity-brokering.md#i08-muse-adapter) / active | 112 | Muse device enrollment and immutable subscription extraction; declaration/member Inner.state; declarations/fields: `Inner.state` |
| [I08](../../slices/identity-brokering.md#i08-muse-adapter) / active | 113 | Muse device enrollment and immutable subscription extraction; declaration/member Inner.child; declarations/fields: `Inner.child` |
| [I08](../../slices/identity-brokering.md#i08-muse-adapter) / active | 114 | Muse device enrollment and immutable subscription extraction; declaration/member Inner.root; declarations/fields: `Inner.root` |
| [I08](../../slices/identity-brokering.md#i08-muse-adapter) / active | 115 | Muse device enrollment and immutable subscription extraction; declaration/member Inner.done; declarations/fields: `Inner.done` |
| [I08](../../slices/identity-brokering.md#i08-muse-adapter) / active | 116 | Muse device enrollment and immutable subscription extraction; declaration/member Inner.finished; declarations/fields: `Inner.finished` |
| [I08](../../slices/identity-brokering.md#i08-muse-adapter) / active | 119–122 | Muse device enrollment and immutable subscription extraction; declaration/member Session; declarations/fields: `Session` |
| [I08](../../slices/identity-brokering.md#i08-muse-adapter) / active | 121 | Muse device enrollment and immutable subscription extraction; declaration/member Session.inner; declarations/fields: `Session.inner` |
| [I08](../../slices/identity-brokering.md#i08-muse-adapter) / active | 125–127 | Muse device enrollment and immutable subscription extraction; declaration/member snapshot; declarations/fields: `snapshot` |
| [I08](../../slices/identity-brokering.md#i08-muse-adapter) / active | 129–141 | Muse device enrollment and immutable subscription extraction; declaration/member finish; declarations/fields: `finish` |
| [I08](../../slices/identity-brokering.md#i08-muse-adapter) / active | 143–164 | Muse device enrollment and immutable subscription extraction; declaration/member close; declarations/fields: `close` |
| [I08](../../slices/identity-brokering.md#i08-muse-adapter) / active | 166–186 | Muse device enrollment and immutable subscription extraction; declaration/member read_loop; declarations/fields: `read_loop` |
| [I08](../../slices/identity-brokering.md#i08-muse-adapter) / active | 188–199 | Muse device enrollment and immutable subscription extraction; declaration/member complete; declarations/fields: `complete` |
| [I08](../../slices/identity-brokering.md#i08-muse-adapter) / active | 203–228 | Muse device enrollment and immutable subscription extraction; declaration/member scan_device_url; declarations/fields: `scan_device_url` |
| [I08](../../slices/identity-brokering.md#i08-muse-adapter) / active | 231–243 | Muse device enrollment and immutable subscription extraction; declaration/member environment; declarations/fields: `environment` |
| [I08](../../slices/identity-brokering.md#i08-muse-adapter) / active | 245–259 | Muse device enrollment and immutable subscription extraction; declaration/member validate_config; declarations/fields: `validate_config` |
| [I08](../../slices/identity-brokering.md#i08-muse-adapter) / active | 261–268 | Muse device enrollment and immutable subscription extraction; declaration/member check_binary; declarations/fields: `check_binary` |
| [I08](../../slices/identity-brokering.md#i08-muse-adapter) / active | 270–283 | Muse device enrollment and immutable subscription extraction; declaration/member check_version; declarations/fields: `check_version` |
| [I08](../../slices/identity-brokering.md#i08-muse-adapter) / active | 287–325 | Muse device enrollment and immutable subscription extraction; declaration/member credential_valid; declarations/fields: `credential_valid` |
| [I08](../../slices/identity-brokering.md#i08-muse-adapter) / active | 291–295 | Muse device enrollment and immutable subscription extraction; declaration/member Wire; declarations/fields: `Wire` |
| [I08](../../slices/identity-brokering.md#i08-muse-adapter) / active | 293 | Muse device enrollment and immutable subscription extraction; declaration/member Wire.schema_version; declarations/fields: `Wire.schema_version` |
| [I08](../../slices/identity-brokering.md#i08-muse-adapter) / active | 294 | Muse device enrollment and immutable subscription extraction; declaration/member Wire.providers; declarations/fields: `Wire.providers` |
| [I08](../../slices/identity-brokering.md#i08-muse-adapter) / active | 296–299 | Muse device enrollment and immutable subscription extraction; declaration/member Providers; declarations/fields: `Providers` |
| [I08](../../slices/identity-brokering.md#i08-muse-adapter) / active | 298 | Muse device enrollment and immutable subscription extraction; declaration/member Providers.meta; declarations/fields: `Providers.meta` |
| [I08](../../slices/identity-brokering.md#i08-muse-adapter) / active | 300–325 | Muse device enrollment and immutable subscription extraction; declaration/member Meta; declarations/fields: `Meta` |
| [I08](../../slices/identity-brokering.md#i08-muse-adapter) / active | 303 | Muse device enrollment and immutable subscription extraction; declaration/member Meta.access_token; declarations/fields: `Meta.access_token` |
| [I08](../../slices/identity-brokering.md#i08-muse-adapter) / active | 305 | Muse device enrollment and immutable subscription extraction; declaration/member Meta.api_key; declarations/fields: `Meta.api_key` |
| [I08](../../slices/identity-brokering.md#i08-muse-adapter) / active | 307 | Muse device enrollment and immutable subscription extraction; declaration/member Meta.api_base_url; declarations/fields: `Meta.api_base_url` |
| [I08](../../slices/identity-brokering.md#i08-muse-adapter) / active | 309 | Muse device enrollment and immutable subscription extraction; declaration/member Meta.mechanism; declarations/fields: `Meta.mechanism` |
| [I08](../../slices/identity-brokering.md#i08-muse-adapter) / active | 311 | Muse device enrollment and immutable subscription extraction; declaration/member Meta.obtained_via; declarations/fields: `Meta.obtained_via` |
| [I08](../../slices/identity-brokering.md#i08-muse-adapter) / active | 327–347 | Muse device enrollment and immutable subscription extraction; declaration/member credential_file; declarations/fields: `credential_file` |
| [I08](../../slices/identity-brokering.md#i08-muse-adapter) / active | 349–351 | Muse device enrollment and immutable subscription extraction; declaration/member tests; declarations/fields: `tests` |

## [cmd/soda-identity/src/providers/muse/tests.rs](../../../../../cmd/soda-identity/src/providers/muse/tests.rs)

Re-audit @HEAD: split sibling; spans re-audited against current bytes.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [I08](../../slices/identity-brokering.md#i08-muse-adapter) / active | 1–4 | Muse device enrollment and immutable subscription extraction; declarations/fields: `SUBSCRIPTION`, `native_subscription_and_private_file`, `enrollment_environment_and_device_prompt`, `config_validation_matches_go`, `binary_digest_checked_before_version`, `version_line_must_match_exactly`, `native_enrollment_keeps_presentation_private` |
| [I08](../../slices/identity-brokering.md#i08-muse-adapter) / active | 6 | Muse device enrollment and immutable subscription extraction; declaration/member SUBSCRIPTION; declarations/fields: `SUBSCRIPTION` |
| [I08](../../slices/identity-brokering.md#i08-muse-adapter) / active | 8–34 | Muse device enrollment and immutable subscription extraction; declaration/member native_subscription_and_private_file; declarations/fields: `native_subscription_and_private_file` |
| [I08](../../slices/identity-brokering.md#i08-muse-adapter) / active | 36–75 | Muse device enrollment and immutable subscription extraction; declaration/member enrollment_environment_and_device_prompt; declarations/fields: `enrollment_environment_and_device_prompt` |
| [I08](../../slices/identity-brokering.md#i08-muse-adapter) / active | 77–120 | Muse device enrollment and immutable subscription extraction; declaration/member config_validation_matches_go; declarations/fields: `config_validation_matches_go` |
| [I08](../../slices/identity-brokering.md#i08-muse-adapter) / active | 122–143 | Muse device enrollment and immutable subscription extraction; declaration/member binary_digest_checked_before_version; declarations/fields: `binary_digest_checked_before_version` |
| [I08](../../slices/identity-brokering.md#i08-muse-adapter) / active | 145–163 | Muse device enrollment and immutable subscription extraction; declaration/member version_line_must_match_exactly; declarations/fields: `version_line_must_match_exactly` |
| [I08](../../slices/identity-brokering.md#i08-muse-adapter) / active | 165–212 | Muse device enrollment and immutable subscription extraction; declaration/member native_enrollment_keeps_presentation_private; declarations/fields: `native_enrollment_keeps_presentation_private` |

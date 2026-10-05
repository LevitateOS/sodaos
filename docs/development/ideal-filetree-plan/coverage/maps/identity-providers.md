# Identity providers

[Responsibility map index](README.md) · [Coverage snapshot and limits](../README.md).
Page grouping is navigation; the slice IDs retain their individual review ownership.

<a id="coverage-533cd08fa569"></a>

<a id="rustidentity-providerssrccodexrs-1"></a>

## [rust/identity-providers/src/codex.rs](../../../../../rust/identity-providers/src/codex.rs)

Source assertions/fixtures were inspected for mapping only; no test execution, runtime or installed proof claimed. Large-file ledger retains committed declaration/member spans rather than grouping methods into one whole-file unit; responsibility follows the verified concern/branch mapping, not declaration count.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [I07](../../slices/identity-brokering.md#i07-codex-adapter) / active | 1–17 | Codex device enrollment, account discovery and auth-file capture; declarations/fields: `VERSION`, `Config`, `Provider`, `new`, `start`, `create_session`, `split_env`, `Inner`, `Session`, `Message`, `snapshot`, `cancel`, `initialize`, `start_device`, `Wire`, `send`, `call`, `finish`, `account`, `Response`, `Account`, `credential_file`, `stop`, `close`, `protocol_error`, `Detail`, `read_loop`, `notify`, `Event`, `environment`, `filter_env`, `validate_config`, `check_binary`, `check_version`, `from`, `tests`, `environment_filters_provider_credentials`, `config_validation_matches_go`, `protocol_fixture`, `fixture_provider`, `managed_enrollment_persists_only_after_process_stop`, `cancel_removes_unfinished_enrollment`, `device_disabled_maps_to_actionable_error` |
| [I07](../../slices/identity-brokering.md#i07-codex-adapter) / active | 18–20 | Codex device enrollment, account discovery and auth-file capture; declaration/member VERSION; declarations/fields: `VERSION` |
| [I07](../../slices/identity-brokering.md#i07-codex-adapter) / active | 21 | Codex device enrollment, account discovery and auth-file capture; declaration/member Config; declarations/fields: `Config` |
| [I07](../../slices/identity-brokering.md#i07-codex-adapter) / active | 22 | Codex device enrollment, account discovery and auth-file capture; declaration/member Config.binary; declarations/fields: `Config.binary` |
| [I07](../../slices/identity-brokering.md#i07-codex-adapter) / active | 23 | Codex device enrollment, account discovery and auth-file capture; declaration/member Config.version; declarations/fields: `Config.version` |
| [I07](../../slices/identity-brokering.md#i07-codex-adapter) / active | 24 | Codex device enrollment, account discovery and auth-file capture; declaration/member Config.sha256; declarations/fields: `Config.sha256` |
| [I07](../../slices/identity-brokering.md#i07-codex-adapter) / active | 25–27 | Codex device enrollment, account discovery and auth-file capture; declaration/member Config.root; declarations/fields: `Config.root` |
| [I07](../../slices/identity-brokering.md#i07-codex-adapter) / active | 28 | Codex device enrollment, account discovery and auth-file capture; declaration/member Provider; declarations/fields: `Provider` |
| [I07](../../slices/identity-brokering.md#i07-codex-adapter) / active | 29–32 | Codex device enrollment, account discovery and auth-file capture; declaration/member Provider.config; declarations/fields: `Provider.config` |
| [I07](../../slices/identity-brokering.md#i07-codex-adapter) / active | 33–49 | Codex device enrollment, account discovery and auth-file capture; declaration/member new; declarations/fields: `new` |
| [I07](../../slices/identity-brokering.md#i07-codex-adapter) / active | 50–69 | Codex device enrollment, account discovery and auth-file capture; declaration/member start; declarations/fields: `start` |
| [I07](../../slices/identity-brokering.md#i07-codex-adapter) / active | 70–156 | Codex device enrollment, account discovery and auth-file capture; declaration/member create_session; declarations/fields: `create_session` |
| [I07](../../slices/identity-brokering.md#i07-codex-adapter) / active | 157–160 | Codex device enrollment, account discovery and auth-file capture; declaration/member split_env; declarations/fields: `split_env` |
| [I07](../../slices/identity-brokering.md#i07-codex-adapter) / active | 161 | Codex device enrollment, account discovery and auth-file capture; declaration/member Inner; declarations/fields: `Inner` |
| [I07](../../slices/identity-brokering.md#i07-codex-adapter) / active | 162 | Codex device enrollment, account discovery and auth-file capture; declaration/member Inner.state; declarations/fields: `Inner.state` |
| [I07](../../slices/identity-brokering.md#i07-codex-adapter) / active | 163 | Codex device enrollment, account discovery and auth-file capture; declaration/member Inner.write; declarations/fields: `Inner.write` |
| [I07](../../slices/identity-brokering.md#i07-codex-adapter) / active | 164 | Codex device enrollment, account discovery and auth-file capture; declaration/member Inner.child; declarations/fields: `Inner.child` |
| [I07](../../slices/identity-brokering.md#i07-codex-adapter) / active | 165 | Codex device enrollment, account discovery and auth-file capture; declaration/member Inner.replies; declarations/fields: `Inner.replies` |
| [I07](../../slices/identity-brokering.md#i07-codex-adapter) / active | 166 | Codex device enrollment, account discovery and auth-file capture; declaration/member Inner.next; declarations/fields: `Inner.next` |
| [I07](../../slices/identity-brokering.md#i07-codex-adapter) / active | 167 | Codex device enrollment, account discovery and auth-file capture; declaration/member Inner.root; declarations/fields: `Inner.root` |
| [I07](../../slices/identity-brokering.md#i07-codex-adapter) / active | 168 | Codex device enrollment, account discovery and auth-file capture; declaration/member Inner.done; declarations/fields: `Inner.done` |
| [I07](../../slices/identity-brokering.md#i07-codex-adapter) / active | 169 | Codex device enrollment, account discovery and auth-file capture; declaration/member Inner.finished; declarations/fields: `Inner.finished` |
| [I07](../../slices/identity-brokering.md#i07-codex-adapter) / active | 170 | Codex device enrollment, account discovery and auth-file capture; declaration/member Inner.cancelled; declarations/fields: `Inner.cancelled` |
| [I07](../../slices/identity-brokering.md#i07-codex-adapter) / active | 171–174 | Codex device enrollment, account discovery and auth-file capture; declaration/member Inner.closed; declarations/fields: `Inner.closed` |
| [I07](../../slices/identity-brokering.md#i07-codex-adapter) / active | 175 | Codex device enrollment, account discovery and auth-file capture; declaration/member Session; declarations/fields: `Session` |
| [I07](../../slices/identity-brokering.md#i07-codex-adapter) / active | 176–179 | Codex device enrollment, account discovery and auth-file capture; declaration/member Session.inner; declarations/fields: `Session.inner` |
| [I07](../../slices/identity-brokering.md#i07-codex-adapter) / active | 180–181 | Codex device enrollment, account discovery and auth-file capture; declaration/member Message; declarations/fields: `Message` |
| [I07](../../slices/identity-brokering.md#i07-codex-adapter) / active | 182–183 | Codex device enrollment, account discovery and auth-file capture; declaration/member Message.id; declarations/fields: `Message.id` |
| [I07](../../slices/identity-brokering.md#i07-codex-adapter) / active | 184–185 | Codex device enrollment, account discovery and auth-file capture; declaration/member Message.method; declarations/fields: `Message.method` |
| [I07](../../slices/identity-brokering.md#i07-codex-adapter) / active | 186–187 | Codex device enrollment, account discovery and auth-file capture; declaration/member Message.result; declarations/fields: `Message.result` |
| [I07](../../slices/identity-brokering.md#i07-codex-adapter) / active | 188–189 | Codex device enrollment, account discovery and auth-file capture; declaration/member Message.error; declarations/fields: `Message.error` |
| [I07](../../slices/identity-brokering.md#i07-codex-adapter) / active | 190–193 | Codex device enrollment, account discovery and auth-file capture; declaration/member Message.params; declarations/fields: `Message.params` |
| [I07](../../slices/identity-brokering.md#i07-codex-adapter) / active | 194–198 | Codex device enrollment, account discovery and auth-file capture; declaration/member snapshot; declarations/fields: `snapshot` |
| [I07](../../slices/identity-brokering.md#i07-codex-adapter) / active | 199–202 | Codex device enrollment, account discovery and auth-file capture; declaration/member cancel; declarations/fields: `cancel` |
| [I07](../../slices/identity-brokering.md#i07-codex-adapter) / active | 203–209 | Codex device enrollment, account discovery and auth-file capture; declaration/member initialize; declarations/fields: `initialize` |
| [I07](../../slices/identity-brokering.md#i07-codex-adapter) / active | 210–216 | Codex device enrollment, account discovery and auth-file capture; declaration/member start_device; declarations/fields: `start_device` |
| [I07](../../slices/identity-brokering.md#i07-codex-adapter) / active | 217–218 | Codex device enrollment, account discovery and auth-file capture; declaration/member Wire; declarations/fields: `Wire` |
| [I07](../../slices/identity-brokering.md#i07-codex-adapter) / active | 219–220 | Codex device enrollment, account discovery and auth-file capture; declaration/member Wire.wire_type; declarations/fields: `Wire.wire_type` |
| [I07](../../slices/identity-brokering.md#i07-codex-adapter) / active | 221–222 | Codex device enrollment, account discovery and auth-file capture; declaration/member Wire.login_id; declarations/fields: `Wire.login_id` |
| [I07](../../slices/identity-brokering.md#i07-codex-adapter) / active | 223–224 | Codex device enrollment, account discovery and auth-file capture; declaration/member Wire.verification_url; declarations/fields: `Wire.verification_url` |
| [I07](../../slices/identity-brokering.md#i07-codex-adapter) / active | 225–242 | Codex device enrollment, account discovery and auth-file capture; declaration/member Wire.user_code; declarations/fields: `Wire.user_code` |
| [I07](../../slices/identity-brokering.md#i07-codex-adapter) / active | 243–253 | Codex device enrollment, account discovery and auth-file capture; declaration/member send; declarations/fields: `send` |
| [I07](../../slices/identity-brokering.md#i07-codex-adapter) / active | 254–303 | Codex device enrollment, account discovery and auth-file capture; declaration/member call; declarations/fields: `call` |
| [I07](../../slices/identity-brokering.md#i07-codex-adapter) / active | 304–313 | Codex device enrollment, account discovery and auth-file capture; declaration/member finish; declarations/fields: `finish` |
| [I07](../../slices/identity-brokering.md#i07-codex-adapter) / active | 314–320 | Codex device enrollment, account discovery and auth-file capture; declaration/member account; declarations/fields: `account` |
| [I07](../../slices/identity-brokering.md#i07-codex-adapter) / active | 321 | Codex device enrollment, account discovery and auth-file capture; declaration/member Response; declarations/fields: `Response` |
| [I07](../../slices/identity-brokering.md#i07-codex-adapter) / active | 322–324 | Codex device enrollment, account discovery and auth-file capture; declaration/member Response.account; declarations/fields: `Response.account` |
| [I07](../../slices/identity-brokering.md#i07-codex-adapter) / active | 325–326 | Codex device enrollment, account discovery and auth-file capture; declaration/member Account; declarations/fields: `Account` |
| [I07](../../slices/identity-brokering.md#i07-codex-adapter) / active | 327 | Codex device enrollment, account discovery and auth-file capture; declaration/member Account.account_type; declarations/fields: `Account.account_type` |
| [I07](../../slices/identity-brokering.md#i07-codex-adapter) / active | 328–329 | Codex device enrollment, account discovery and auth-file capture; declaration/member Account.email; declarations/fields: `Account.email` |
| [I07](../../slices/identity-brokering.md#i07-codex-adapter) / active | 330–346 | Codex device enrollment, account discovery and auth-file capture; declaration/member Account.plan; declarations/fields: `Account.plan` |
| [I07](../../slices/identity-brokering.md#i07-codex-adapter) / active | 347–369 | Codex device enrollment, account discovery and auth-file capture; declaration/member credential_file; declarations/fields: `credential_file` |
| [I07](../../slices/identity-brokering.md#i07-codex-adapter) / active | 370–384 | Codex device enrollment, account discovery and auth-file capture; declaration/member stop; declarations/fields: `stop` |
| [I07](../../slices/identity-brokering.md#i07-codex-adapter) / active | 385–399 | Codex device enrollment, account discovery and auth-file capture; declaration/member close; declarations/fields: `close` |
| [I07](../../slices/identity-brokering.md#i07-codex-adapter) / active | 400–401 | Codex device enrollment, account discovery and auth-file capture; declaration/member protocol_error; declarations/fields: `protocol_error` |
| [I07](../../slices/identity-brokering.md#i07-codex-adapter) / active | 402–403 | Codex device enrollment, account discovery and auth-file capture; declaration/member Detail; declarations/fields: `Detail` |
| [I07](../../slices/identity-brokering.md#i07-codex-adapter) / active | 404–419 | Codex device enrollment, account discovery and auth-file capture; declaration/member Detail.message; declarations/fields: `Detail.message` |
| [I07](../../slices/identity-brokering.md#i07-codex-adapter) / active | 420–476 | Codex device enrollment, account discovery and auth-file capture; declaration/member read_loop; declarations/fields: `read_loop` |
| [I07](../../slices/identity-brokering.md#i07-codex-adapter) / active | 477–481 | Codex device enrollment, account discovery and auth-file capture; declaration/member notify; declarations/fields: `notify` |
| [I07](../../slices/identity-brokering.md#i07-codex-adapter) / active | 482–483 | Codex device enrollment, account discovery and auth-file capture; declaration/member Event; declarations/fields: `Event` |
| [I07](../../slices/identity-brokering.md#i07-codex-adapter) / active | 484 | Codex device enrollment, account discovery and auth-file capture; declaration/member Event.login_id; declarations/fields: `Event.login_id` |
| [I07](../../slices/identity-brokering.md#i07-codex-adapter) / active | 485–501 | Codex device enrollment, account discovery and auth-file capture; declaration/member Event.success; declarations/fields: `Event.success` |
| [I07](../../slices/identity-brokering.md#i07-codex-adapter) / active | 502–505 | Codex device enrollment, account discovery and auth-file capture; declaration/member environment; declarations/fields: `environment` |
| [I07](../../slices/identity-brokering.md#i07-codex-adapter) / active | 506–525 | Codex device enrollment, account discovery and auth-file capture; declaration/member filter_env; declarations/fields: `filter_env` |
| [I07](../../slices/identity-brokering.md#i07-codex-adapter) / active | 526–536 | Codex device enrollment, account discovery and auth-file capture; declaration/member validate_config; declarations/fields: `validate_config` |
| [I07](../../slices/identity-brokering.md#i07-codex-adapter) / active | 537–544 | Codex device enrollment, account discovery and auth-file capture; declaration/member check_binary; declarations/fields: `check_binary` |
| [I07](../../slices/identity-brokering.md#i07-codex-adapter) / active | 545–560 | Codex device enrollment, account discovery and auth-file capture; declaration/member check_version; declarations/fields: `check_version` |
| [I07](../../slices/identity-brokering.md#i07-codex-adapter) / active | 561–566 | Codex device enrollment, account discovery and auth-file capture; declaration/member from; declarations/fields: `from` |
| [I07](../../slices/identity-brokering.md#i07-codex-adapter) / active | 567–571 | Codex device enrollment, account discovery and auth-file capture; declaration/member tests; declarations/fields: `tests` |
| [I07](../../slices/identity-brokering.md#i07-codex-adapter) / active | 572–593 | Codex device enrollment, account discovery and auth-file capture; declaration/member environment_filters_provider_credentials; declarations/fields: `environment_filters_provider_credentials` |
| [I07](../../slices/identity-brokering.md#i07-codex-adapter) / active | 594–626 | Codex device enrollment, account discovery and auth-file capture; declaration/member config_validation_matches_go; declarations/fields: `config_validation_matches_go` |
| [I07](../../slices/identity-brokering.md#i07-codex-adapter) / active | 627–640 | Codex device enrollment, account discovery and auth-file capture; declaration/member protocol_fixture; declarations/fields: `protocol_fixture` |
| [I07](../../slices/identity-brokering.md#i07-codex-adapter) / active | 641–655 | Codex device enrollment, account discovery and auth-file capture; declaration/member fixture_provider; declarations/fields: `fixture_provider` |
| [I07](../../slices/identity-brokering.md#i07-codex-adapter) / active | 656–681 | Codex device enrollment, account discovery and auth-file capture; declaration/member managed_enrollment_persists_only_after_process_stop; declarations/fields: `managed_enrollment_persists_only_after_process_stop` |
| [I07](../../slices/identity-brokering.md#i07-codex-adapter) / active | 682–692 | Codex device enrollment, account discovery and auth-file capture; declaration/member cancel_removes_unfinished_enrollment; declarations/fields: `cancel_removes_unfinished_enrollment` |
| [I07](../../slices/identity-brokering.md#i07-codex-adapter) / active | 693–702 | Codex device enrollment, account discovery and auth-file capture; declaration/member device_disabled_maps_to_actionable_error; declarations/fields: `device_disabled_maps_to_actionable_error` |

<a id="coverage-a9d928996350"></a>

## [rust/identity-providers/src/lib.rs](../../../../../rust/identity-providers/src/lib.rs)

Source assertions/fixtures were inspected for mapping only; no test execution, runtime or installed proof claimed. Public module wiring is mapped separately; module exposure does not create a new process boundary.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [I01](../../slices/identity-brokering.md#i01-enrollment-and-owner-consent) / active | 1–5, 23–236 | Provider enrollment interface and private native enrollment sessions; declarations/fields: `Kind`, `Error`, `denied`, `uncertain`, `failed`, `kind`, `is_denied`, `is_uncertain`, `fmt`, `from`, `enrollment_tempdir`, `random_bytes`, `private_tmpfs`, `TestDir`, `new`, `path`, `drop`, `run_capture` |
| [I07](../../slices/identity-brokering.md#i07-codex-adapter) / active | 6 | Codex native enrollment adapter module; declarations/fields: `codex` |
| [I08](../../slices/identity-brokering.md#i08-muse-adapter) / active | 7 | Muse native enrollment adapter module; declarations/fields: `muse` |
| [I01](../../slices/identity-brokering.md#i01-enrollment-and-owner-consent) / active | 8 | Module wiring: sha256; Enrollment and owner consent; declarations/fields: `sha256` |
| [I01](../../slices/identity-brokering.md#i01-enrollment-and-owner-consent) / active | 9–22 | Module wiring: types; Enrollment and owner consent; declarations/fields: `types` |

<a id="coverage-e008c55b400c"></a>

<a id="rustidentity-providerssrcmusers-1"></a>

## [rust/identity-providers/src/muse.rs](../../../../../rust/identity-providers/src/muse.rs)

Source assertions/fixtures were inspected for mapping only; no test execution, runtime or installed proof claimed. Large-file ledger retains committed declaration/member spans rather than grouping methods into one whole-file unit; responsibility follows the verified concern/branch mapping, not declaration count.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [I08](../../slices/identity-brokering.md#i08-muse-adapter) / active | 1–16 | Muse device enrollment and immutable subscription extraction; declarations/fields: `VERSION`, `VERSION_LINE`, `DEVICE_PREFIX`, `Config`, `Provider`, `new`, `start`, `split_env`, `Inner`, `Session`, `snapshot`, `finish`, `close`, `read_loop`, `complete`, `scan_device_url`, `environment`, `validate_config`, `check_binary`, `check_version`, `credential_valid`, `Wire`, `Providers`, `Meta`, `credential_file`, `tests`, `SUBSCRIPTION`, `native_subscription_and_private_file`, `enrollment_environment_and_device_prompt`, `config_validation_matches_go`, `binary_digest_checked_before_version`, `version_line_must_match_exactly`, `native_enrollment_keeps_presentation_private` |
| [I08](../../slices/identity-brokering.md#i08-muse-adapter) / active | 17 | Muse device enrollment and immutable subscription extraction; declaration/member VERSION; declarations/fields: `VERSION` |
| [I08](../../slices/identity-brokering.md#i08-muse-adapter) / active | 18 | Muse device enrollment and immutable subscription extraction; declaration/member VERSION_LINE; declarations/fields: `VERSION_LINE` |
| [I08](../../slices/identity-brokering.md#i08-muse-adapter) / active | 19–21 | Muse device enrollment and immutable subscription extraction; declaration/member DEVICE_PREFIX; declarations/fields: `DEVICE_PREFIX` |
| [I08](../../slices/identity-brokering.md#i08-muse-adapter) / active | 22 | Muse device enrollment and immutable subscription extraction; declaration/member Config; declarations/fields: `Config` |
| [I08](../../slices/identity-brokering.md#i08-muse-adapter) / active | 23 | Muse device enrollment and immutable subscription extraction; declaration/member Config.binary; declarations/fields: `Config.binary` |
| [I08](../../slices/identity-brokering.md#i08-muse-adapter) / active | 24 | Muse device enrollment and immutable subscription extraction; declaration/member Config.version; declarations/fields: `Config.version` |
| [I08](../../slices/identity-brokering.md#i08-muse-adapter) / active | 25 | Muse device enrollment and immutable subscription extraction; declaration/member Config.sha256; declarations/fields: `Config.sha256` |
| [I08](../../slices/identity-brokering.md#i08-muse-adapter) / active | 26–28 | Muse device enrollment and immutable subscription extraction; declaration/member Config.root; declarations/fields: `Config.root` |
| [I08](../../slices/identity-brokering.md#i08-muse-adapter) / active | 29 | Muse device enrollment and immutable subscription extraction; declaration/member Provider; declarations/fields: `Provider` |
| [I08](../../slices/identity-brokering.md#i08-muse-adapter) / active | 30–33 | Muse device enrollment and immutable subscription extraction; declaration/member Provider.config; declarations/fields: `Provider.config` |
| [I08](../../slices/identity-brokering.md#i08-muse-adapter) / active | 34–43 | Muse device enrollment and immutable subscription extraction; declaration/member new; declarations/fields: `new` |
| [I08](../../slices/identity-brokering.md#i08-muse-adapter) / active | 44–105 | Muse device enrollment and immutable subscription extraction; declaration/member start; declarations/fields: `start` |
| [I08](../../slices/identity-brokering.md#i08-muse-adapter) / active | 106–109 | Muse device enrollment and immutable subscription extraction; declaration/member split_env; declarations/fields: `split_env` |
| [I08](../../slices/identity-brokering.md#i08-muse-adapter) / active | 110 | Muse device enrollment and immutable subscription extraction; declaration/member Inner; declarations/fields: `Inner` |
| [I08](../../slices/identity-brokering.md#i08-muse-adapter) / active | 111 | Muse device enrollment and immutable subscription extraction; declaration/member Inner.state; declarations/fields: `Inner.state` |
| [I08](../../slices/identity-brokering.md#i08-muse-adapter) / active | 112 | Muse device enrollment and immutable subscription extraction; declaration/member Inner.child; declarations/fields: `Inner.child` |
| [I08](../../slices/identity-brokering.md#i08-muse-adapter) / active | 113 | Muse device enrollment and immutable subscription extraction; declaration/member Inner.root; declarations/fields: `Inner.root` |
| [I08](../../slices/identity-brokering.md#i08-muse-adapter) / active | 114 | Muse device enrollment and immutable subscription extraction; declaration/member Inner.done; declarations/fields: `Inner.done` |
| [I08](../../slices/identity-brokering.md#i08-muse-adapter) / active | 115–118 | Muse device enrollment and immutable subscription extraction; declaration/member Inner.finished; declarations/fields: `Inner.finished` |
| [I08](../../slices/identity-brokering.md#i08-muse-adapter) / active | 119 | Muse device enrollment and immutable subscription extraction; declaration/member Session; declarations/fields: `Session` |
| [I08](../../slices/identity-brokering.md#i08-muse-adapter) / active | 120–123 | Muse device enrollment and immutable subscription extraction; declaration/member Session.inner; declarations/fields: `Session.inner` |
| [I08](../../slices/identity-brokering.md#i08-muse-adapter) / active | 124–127 | Muse device enrollment and immutable subscription extraction; declaration/member snapshot; declarations/fields: `snapshot` |
| [I08](../../slices/identity-brokering.md#i08-muse-adapter) / active | 128–141 | Muse device enrollment and immutable subscription extraction; declaration/member finish; declarations/fields: `finish` |
| [I08](../../slices/identity-brokering.md#i08-muse-adapter) / active | 142–164 | Muse device enrollment and immutable subscription extraction; declaration/member close; declarations/fields: `close` |
| [I08](../../slices/identity-brokering.md#i08-muse-adapter) / active | 165–186 | Muse device enrollment and immutable subscription extraction; declaration/member read_loop; declarations/fields: `read_loop` |
| [I08](../../slices/identity-brokering.md#i08-muse-adapter) / active | 187–201 | Muse device enrollment and immutable subscription extraction; declaration/member complete; declarations/fields: `complete` |
| [I08](../../slices/identity-brokering.md#i08-muse-adapter) / active | 202–229 | Muse device enrollment and immutable subscription extraction; declaration/member scan_device_url; declarations/fields: `scan_device_url` |
| [I08](../../slices/identity-brokering.md#i08-muse-adapter) / active | 230–243 | Muse device enrollment and immutable subscription extraction; declaration/member environment; declarations/fields: `environment` |
| [I08](../../slices/identity-brokering.md#i08-muse-adapter) / active | 244–259 | Muse device enrollment and immutable subscription extraction; declaration/member validate_config; declarations/fields: `validate_config` |
| [I08](../../slices/identity-brokering.md#i08-muse-adapter) / active | 260–268 | Muse device enrollment and immutable subscription extraction; declaration/member check_binary; declarations/fields: `check_binary` |
| [I08](../../slices/identity-brokering.md#i08-muse-adapter) / active | 269–285 | Muse device enrollment and immutable subscription extraction; declaration/member check_version; declarations/fields: `check_version` |
| [I08](../../slices/identity-brokering.md#i08-muse-adapter) / active | 286–290 | Muse device enrollment and immutable subscription extraction; declaration/member credential_valid; declarations/fields: `credential_valid` |
| [I08](../../slices/identity-brokering.md#i08-muse-adapter) / active | 291 | Muse device enrollment and immutable subscription extraction; declaration/member Wire; declarations/fields: `Wire` |
| [I08](../../slices/identity-brokering.md#i08-muse-adapter) / active | 292 | Muse device enrollment and immutable subscription extraction; declaration/member Wire.schema_version; declarations/fields: `Wire.schema_version` |
| [I08](../../slices/identity-brokering.md#i08-muse-adapter) / active | 293–295 | Muse device enrollment and immutable subscription extraction; declaration/member Wire.providers; declarations/fields: `Wire.providers` |
| [I08](../../slices/identity-brokering.md#i08-muse-adapter) / active | 296 | Muse device enrollment and immutable subscription extraction; declaration/member Providers; declarations/fields: `Providers` |
| [I08](../../slices/identity-brokering.md#i08-muse-adapter) / active | 297–299 | Muse device enrollment and immutable subscription extraction; declaration/member Providers.meta; declarations/fields: `Providers.meta` |
| [I08](../../slices/identity-brokering.md#i08-muse-adapter) / active | 300–301 | Muse device enrollment and immutable subscription extraction; declaration/member Meta; declarations/fields: `Meta` |
| [I08](../../slices/identity-brokering.md#i08-muse-adapter) / active | 302–303 | Muse device enrollment and immutable subscription extraction; declaration/member Meta.access_token; declarations/fields: `Meta.access_token` |
| [I08](../../slices/identity-brokering.md#i08-muse-adapter) / active | 304–305 | Muse device enrollment and immutable subscription extraction; declaration/member Meta.api_key; declarations/fields: `Meta.api_key` |
| [I08](../../slices/identity-brokering.md#i08-muse-adapter) / active | 306–307 | Muse device enrollment and immutable subscription extraction; declaration/member Meta.api_base_url; declarations/fields: `Meta.api_base_url` |
| [I08](../../slices/identity-brokering.md#i08-muse-adapter) / active | 308–309 | Muse device enrollment and immutable subscription extraction; declaration/member Meta.mechanism; declarations/fields: `Meta.mechanism` |
| [I08](../../slices/identity-brokering.md#i08-muse-adapter) / active | 310–325 | Muse device enrollment and immutable subscription extraction; declaration/member Meta.obtained_via; declarations/fields: `Meta.obtained_via` |
| [I08](../../slices/identity-brokering.md#i08-muse-adapter) / active | 326–348 | Muse device enrollment and immutable subscription extraction; declaration/member credential_file; declarations/fields: `credential_file` |
| [I08](../../slices/identity-brokering.md#i08-muse-adapter) / active | 349–353 | Muse device enrollment and immutable subscription extraction; declaration/member tests; declarations/fields: `tests` |
| [I08](../../slices/identity-brokering.md#i08-muse-adapter) / active | 354–356 | Muse device enrollment and immutable subscription extraction; declaration/member SUBSCRIPTION; declarations/fields: `SUBSCRIPTION` |
| [I08](../../slices/identity-brokering.md#i08-muse-adapter) / active | 357–384 | Muse device enrollment and immutable subscription extraction; declaration/member native_subscription_and_private_file; declarations/fields: `native_subscription_and_private_file` |
| [I08](../../slices/identity-brokering.md#i08-muse-adapter) / active | 385–425 | Muse device enrollment and immutable subscription extraction; declaration/member enrollment_environment_and_device_prompt; declarations/fields: `enrollment_environment_and_device_prompt` |
| [I08](../../slices/identity-brokering.md#i08-muse-adapter) / active | 426–470 | Muse device enrollment and immutable subscription extraction; declaration/member config_validation_matches_go; declarations/fields: `config_validation_matches_go` |
| [I08](../../slices/identity-brokering.md#i08-muse-adapter) / active | 471–493 | Muse device enrollment and immutable subscription extraction; declaration/member binary_digest_checked_before_version; declarations/fields: `binary_digest_checked_before_version` |
| [I08](../../slices/identity-brokering.md#i08-muse-adapter) / active | 494–513 | Muse device enrollment and immutable subscription extraction; declaration/member version_line_must_match_exactly; declarations/fields: `version_line_must_match_exactly` |
| [I08](../../slices/identity-brokering.md#i08-muse-adapter) / active | 514–561 | Muse device enrollment and immutable subscription extraction; declaration/member native_enrollment_keeps_presentation_private; declarations/fields: `native_enrollment_keeps_presentation_private` |


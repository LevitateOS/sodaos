# Identity providers

[Responsibility map index](README.md) · [Coverage scope](../README.md).

Current responsibility accounting at `519b76bd` (2026-10-07).
One slice owns each named duty; disjoint complete symbols may share an owner.
Compound fields/clauses may share a physical line with distinct selectors.
Disposition concerns the duty, not source validity or installed qualification.

<a id="coverage-884fc1ea2739"></a>

## [cmd/soda-identity/src/providers/codex/config.rs](../../../../../cmd/soda-identity/src/providers/codex/config.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–78; current module/import/attribute shell; declaration Config; fields binary, version, sha256, root; declaration environment; declaration filter_env; declaration validate_config; declaration check_binary; declaration check_version; declaration from | [I07](../../slices/identity-brokering.md#i07-codex-adapter) | retained | Imports and module declarations wire cmd/soda-identity/src/providers/codex/config.rs into its current native target.; 8 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-533cd08fa569"></a>
<a id="rustidentity-providerssrccodexrs-1"></a>
<a id="coverage-f94b7d150404"></a>

## [cmd/soda-identity/src/providers/codex/mod.rs](../../../../../cmd/soda-identity/src/providers/codex/mod.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–341; current module/import/attribute shell; declaration config; declaration protocol; declaration VERSION; declaration Provider; fields config; declaration new; declaration start; declaration create_session; declaration split_env; declaration Inner; fields state, write, child, replies, next, root, done, finished, cancelled, closed; declaration drop; declaration Session; fields inner; declaration snapshot; declaration cancel; declaration initialize; declaration start_device; declaration Wire; fields wire_type, login_id, verification_url, user_code; declaration finish; declaration account; declaration Response; fields account; declaration Account; fields account_type, email, plan; declaration credential_file; declaration stop; declaration close; declaration tests | [I07](../../slices/identity-brokering.md#i07-codex-adapter) | retained | Imports and module declarations wire cmd/soda-identity/src/providers/codex/mod.rs into its current native target.; 25 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-24c39b793775"></a>

## [cmd/soda-identity/src/providers/codex/protocol.rs](../../../../../cmd/soda-identity/src/providers/codex/protocol.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–187; current module/import/attribute shell; declaration Message; fields id, method, result, error, params; declaration send; declaration call; declaration protocol_error; declaration Detail; fields message, Error; declaration read_loop; declaration notify; declaration Event; fields login_id, success | [I07](../../slices/identity-brokering.md#i07-codex-adapter) | retained | Imports and module declarations wire cmd/soda-identity/src/providers/codex/protocol.rs into its current native target.; 9 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-2d75179f9461"></a>

## [cmd/soda-identity/src/providers/codex/tests.rs](../../../../../cmd/soda-identity/src/providers/codex/tests.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–153; current module/import/attribute shell; declaration environment_filters_provider_credentials; declaration config_validation_matches_go; declaration protocol_fixture; declaration fixture_provider; declaration managed_enrollment_persists_only_after_process_stop; declaration cancel_removes_unfinished_enrollment; declaration device_disabled_maps_to_actionable_error | [I07](../../slices/identity-brokering.md#i07-codex-adapter) | retained | Imports and module declarations wire cmd/soda-identity/src/providers/codex/tests.rs into its current native target.; 8 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-d5a0df56eddd"></a>

## [cmd/soda-identity/src/providers/mod.rs](../../../../../cmd/soda-identity/src/providers/mod.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–7, 10–234; current module/import/attribute shell; declaration sha256; declaration types; declaration Kind; declaration Error; fields kind, message; declaration denied; declaration uncertain; declaration failed; declaration kind; declaration is_denied; declaration is_uncertain; declaration fmt; declaration from; declaration enrollment_tempdir; declaration enrollment_tempdir_with; declaration private_tmpfs; declaration TestDir; fields dir; declaration new; declaration path; declaration entropy_tests; declaration enrollment_tempdir_propagates_partial_entropy_failure; declaration run_capture | [I01](../../slices/identity-brokering.md#i01-enrollment-and-owner-consent) | retained | Imports and module declarations wire cmd/soda-identity/src/providers/mod.rs into its current native target.; 22 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |
| 8; declaration codex | [I07](../../slices/identity-brokering.md#i07-codex-adapter) | retained | codex: implement the current Codex provider protocol adapter duty in mod.rs. — current source cmd/soda-identity/src/providers/mod.rs; lines 8-8; module/caller wiring inspected |
| 9; declaration muse | [I08](../../slices/identity-brokering.md#i08-muse-adapter) | retained | muse: implement the current Muse provider protocol adapter duty in mod.rs. — current source cmd/soda-identity/src/providers/mod.rs; lines 9-9; module/caller wiring inspected |

<a id="coverage-e008c55b400c"></a>
<a id="rustidentity-providerssrcmusers-1"></a>
<a id="coverage-37502c6f2904"></a>

## [cmd/soda-identity/src/providers/muse/mod.rs](../../../../../cmd/soda-identity/src/providers/muse/mod.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–369; current module/import/attribute shell; declaration VERSION; declaration VERSION_LINE; declaration DEVICE_PREFIX; declaration Config; fields binary, version, sha256, root; declaration Provider; fields config; declaration new; declaration start; declaration split_env; declaration Inner; fields state, child, root, done, finished; declaration drop; declaration Session; fields inner; declaration snapshot; declaration finish; declaration close; declaration read_loop; declaration complete; declaration scan_device_url; declaration environment; declaration validate_config; declaration check_binary; declaration check_version; declaration credential_valid; declaration Wire; fields schema_version, providers; declaration Providers; fields meta; declaration Meta; fields access_token, api_key, api_base_url, mechanism, obtained_via; declaration credential_file; declaration tests | [I08](../../slices/identity-brokering.md#i08-muse-adapter) | retained | Imports and module declarations wire cmd/soda-identity/src/providers/muse/mod.rs into its current native target.; 28 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-49feb356e396"></a>

## [cmd/soda-identity/src/providers/muse/tests.rs](../../../../../cmd/soda-identity/src/providers/muse/tests.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–220; current module/import/attribute shell; declaration SUBSCRIPTION; declaration native_subscription_and_private_file; declaration enrollment_environment_and_device_prompt; declaration config_validation_matches_go; declaration binary_digest_checked_before_version; declaration version_line_must_match_exactly; declaration native_enrollment_keeps_presentation_private | [I08](../../slices/identity-brokering.md#i08-muse-adapter) | retained | Imports and module declarations wire cmd/soda-identity/src/providers/muse/tests.rs into its current native target.; 8 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-24f1ac6675d6"></a>

## [cmd/soda-identity/src/providers/sha256.rs](../../../../../cmd/soda-identity/src/providers/sha256.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–83; current module/import/attribute shell; declaration digest; declaration digest_reader; declaration hex; declaration HEX; declaration tests; declaration standard_vectors; declaration reader_streams_and_propagates_errors; declaration FailingReader; declaration read | [I01](../../slices/identity-brokering.md#i01-enrollment-and-owner-consent) | retained | Imports and module declarations wire cmd/soda-identity/src/providers/sha256.rs into its current native target.; 10 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-9b8f23916018"></a>

## [cmd/soda-identity/src/providers/types.rs](../../../../../cmd/soda-identity/src/providers/types.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–105; current module/import/attribute shell; declaration MAX_CREDENTIAL_BYTES; declaration credential_valid; declaration Connection; fields provider_id, id, owner_id, label, email, plan, generation, state; declaration Enrollment; fields provider_id, id, verification_url, user_code, state, error, connection; declaration i64_string; declaration serialize; declaration deserialize; declaration tests; declaration connection_wire_shape_matches_go; declaration enrollment_omits_empty_error; declaration credential_bounds_match_go | [I01](../../slices/identity-brokering.md#i01-enrollment-and-owner-consent) | retained | Imports and module declarations wire cmd/soda-identity/src/providers/types.rs into its current native target.; 12 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

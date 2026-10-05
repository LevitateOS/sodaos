# Host projects

[Responsibility map index](README.md) · [Coverage snapshot and limits](../README.md).
Page grouping is navigation; the slice IDs retain their individual review ownership.

<a id="coverage-7417f4912a73"></a>

<a id="rustsoda-hostsrcaccountrs-1"></a>

## [rust/soda-host/src/account.rs](../../../../../rust/soda-host/src/account.rs)

Source assertions/fixtures were inspected for mapping only; no test execution, runtime or installed proof claimed. Large-file ledger retains committed declaration/member spans rather than grouping methods into one whole-file unit; responsibility follows the verified concern/branch mapping, not declaration count.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 1–15 | Canonical SSH keys and revision comparison; declarations/fields: `AGENT_PROGRAM`, `valid_key_revision`, `canonicalize_account_keys`, `CONFIRM_SPECS` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 16–18 | Canonical SSH keys and revision comparison; declaration/member AGENT_PROGRAM; declarations/fields: `AGENT_PROGRAM` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 19–27 | Canonical SSH keys and revision comparison; declaration/member valid_key_revision; declarations/fields: `valid_key_revision` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 28–42 | Canonical SSH keys and revision comparison; declaration/member canonicalize_account_keys; declarations/fields: `canonicalize_account_keys` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 43–53 | Canonical SSH keys and revision comparison; declaration/member CONFIRM_SPECS; declarations/fields: `CONFIRM_SPECS` |
| [P03](../../slices/projects.md#p03-human-membership-and-accounts) / active | 54 | Account confirmation; declarations/fields: `confirm_account` |
| [P03](../../slices/projects.md#p03-human-membership-and-accounts) / active | 55–70 | Account confirmation; declaration/member ERR; declarations/fields: `ERR` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 71–98 | Explicit access-key request/state validation; declarations/fields: `canonical_keys` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 99–111 | Explicit access-key request/state validation; declaration/member valid_access_keys_request; declarations/fields: `valid_access_keys_request` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 112–122 | Explicit access-key request/state validation; declaration/member KEY_STATE_SPECS; declarations/fields: `KEY_STATE_SPECS` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 123–126 | Explicit access-key request/state validation; declaration/member decode_access_key_state; declarations/fields: `decode_access_key_state` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 127–145 | Explicit access-key request/state validation; declaration/member ERR; declarations/fields: `ERR` |
| [P03](../../slices/projects.md#p03-human-membership-and-accounts) / active | 146–190 | Privileged human account helper operation; declarations/fields: `account` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 191–255 | Explicit key observe/apply and stale revision refusal; declarations/fields: `access_keys` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 256–261, 371–424, 471–476 | Account and key source tests; declarations/fields: `tests`, `development_keys_require_canonical_unique_lines`, `agent_program_path_matches_go` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 262–264 | Account and key source tests; declaration/member ED; declarations/fields: `ED` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 265–266 | Account and key source tests; declaration/member MockCall; declarations/fields: `MockCall` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 267 | Account and key source tests; declaration/member Mock; declarations/fields: `Mock` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 268 | Account and key source tests; declaration/member Mock.calls; declarations/fields: `Mock.calls` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 269–272 | Account and key source tests; declaration/member Mock.script; declarations/fields: `Mock.script` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 273–281 | Account and key source tests; declaration/member new; declarations/fields: `new` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 282–300 | Account and key source tests; declaration/member run; declarations/fields: `run` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 301–304 | Account and key source tests; declaration/member deadline; declarations/fields: `deadline` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 305–314 | Account and key source tests; declaration/member test_config; declarations/fields: `test_config` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 315–318 | Account and key source tests; declaration/member pid; declarations/fields: `pid` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 319–329 | Account and key source tests; declaration/member inspect_payload; declarations/fields: `inspect_payload` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 330–338 | Account and key source tests; declaration/member container_payload; declarations/fields: `container_payload` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 339–346, 425–470, 599–641 | Source assertion of Development SSH access; declarations/fields: `key_revision_shape`, `access_key_state_decode_uses_plain_json_semantics`, `access_keys_preview_observes_without_applying` |
| [P03](../../slices/projects.md#p03-human-membership-and-accounts) / active | 347–370, 477–516 | Source assertion of Human membership and accounts; declarations/fields: `account_keys_drop_comments_but_keep_newline`, `account_provisions_with_exact_body_and_argv` |
| [P03](../../slices/projects.md#p03-human-membership-and-accounts) / active | 517–598 | Source assertion of Human membership and accounts; declaration/member account_rejects_bad_requests_and_unconfirmed_helpers; declarations/fields: `account_rejects_bad_requests_and_unconfirmed_helpers` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 642–673 | Source assertion of Development SSH access; declaration/member access_keys_apply_round_trips_preview_and_confirms_set; declarations/fields: `access_keys_apply_round_trips_preview_and_confirms_set` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 674–764 | Source assertion of Development SSH access; declaration/member access_keys_rejects_drift_and_bad_requests; declarations/fields: `access_keys_rejects_drift_and_bad_requests` |

<a id="coverage-43e7d0b067b8"></a>

<a id="rustsoda-hostsrcdomainrs-1"></a>

## [rust/soda-host/src/domain.rs](../../../../../rust/soda-host/src/domain.rs)

Source assertions/fixtures were inspected for mapping only; no test execution, runtime or installed proof claimed. Large-file ledger retains committed declaration/member spans rather than grouping methods into one whole-file unit; responsibility follows the verified concern/branch mapping, not declaration count.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) / active | 1–9 | Project identifiers and runtime predicates; declarations/fields: `ROCKY_HEADLESS`, `GO_CC`, `GO_CF`, `in_ranges`, `is_hex_lower`, `valid_version`, `valid_id`, `valid_login`, `valid_image_ref`, `valid_container_id`, `PROFILE_SPECS` |
| [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) / active | 10–13 | Project identifiers and runtime predicates; declaration/member ROCKY_HEADLESS; declarations/fields: `ROCKY_HEADLESS` |
| [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) / active | 14 | Project identifiers and runtime predicates; declaration/member GO_CC; declarations/fields: `GO_CC` |
| [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) / active | 15–38 | Project identifiers and runtime predicates; declaration/member GO_CF; declarations/fields: `GO_CF` |
| [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) / active | 39–43 | Project identifiers and runtime predicates; declaration/member in_ranges; declarations/fields: `in_ranges` |
| [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) / active | 44–49 | Project identifiers and runtime predicates; declaration/member is_hex_lower; declarations/fields: `is_hex_lower` |
| [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) / active | 50–68 | Project identifiers and runtime predicates; declaration/member valid_version; declarations/fields: `valid_version` |
| [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) / active | 69–73 | Project identifiers and runtime predicates; declaration/member valid_id; declarations/fields: `valid_id` |
| [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) / active | 74–84 | Project identifiers and runtime predicates; declaration/member valid_login; declarations/fields: `valid_login` |
| [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) / active | 85–90 | Project identifiers and runtime predicates; declaration/member valid_image_ref; declarations/fields: `valid_image_ref` |
| [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) / active | 91–94 | Project identifiers and runtime predicates; declaration/member valid_container_id; declarations/fields: `valid_container_id` |
| [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) / active | 95–126 | Project identifiers and runtime predicates; declaration/member PROFILE_SPECS; declarations/fields: `PROFILE_SPECS` |
| [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) / active | 127 | Immutable profile and runtime readiness; declarations/fields: `Profile` |
| [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) / active | 128 | Immutable profile and runtime readiness; declaration/member Profile.id; declarations/fields: `Profile.id` |
| [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) / active | 129 | Immutable profile and runtime readiness; declaration/member Profile.distribution; declarations/fields: `Profile.distribution` |
| [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) / active | 130 | Immutable profile and runtime readiness; declaration/member Profile.version; declarations/fields: `Profile.version` |
| [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) / active | 131 | Immutable profile and runtime readiness; declaration/member Profile.interface; declarations/fields: `Profile.interface` |
| [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) / active | 132 | Immutable profile and runtime readiness; declaration/member Profile.architecture; declarations/fields: `Profile.architecture` |
| [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) / active | 133 | Immutable profile and runtime readiness; declaration/member Profile.image; declarations/fields: `Profile.image` |
| [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) / active | 134–137 | Immutable profile and runtime readiness; declaration/member Profile.revision; declarations/fields: `Profile.revision` |
| [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) / active | 138–156 | Immutable profile and runtime readiness; declaration/member validate; declarations/fields: `validate` |
| [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) / active | 157–161 | Immutable profile and runtime readiness; declaration/member from_value; declarations/fields: `from_value` |
| [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) / active | 162–175 | Immutable profile and runtime readiness; declaration/member from_map; declarations/fields: `from_map` |
| [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) / active | 176–194 | Immutable profile and runtime readiness; declaration/member encode_into; declarations/fields: `encode_into` |
| [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) / active | 195–202 | Immutable profile and runtime readiness; declaration/member encode; declarations/fields: `encode` |
| [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) / active | 203–212 | Immutable profile and runtime readiness; declaration/member decode_profile; declarations/fields: `decode_profile` |
| [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) / active | 213–232 | Immutable profile and runtime readiness; declaration/member CREATE_SPECS; declarations/fields: `CREATE_SPECS` |
| [P01](../../slices/projects.md#p01-repository-association-and-creation) / active | 233 | Native creation request validation; declarations/fields: `Create` |
| [P01](../../slices/projects.md#p01-repository-association-and-creation) / active | 234 | Native creation request validation; declaration/member Create.profile; declarations/fields: `Create.profile` |
| [P01](../../slices/projects.md#p01-repository-association-and-creation) / active | 235 | Native creation request validation; declaration/member Create.id; declarations/fields: `Create.id` |
| [P01](../../slices/projects.md#p01-repository-association-and-creation) / active | 236–239 | Native creation request validation; declaration/member Create.owner; declarations/fields: `Create.owner` |
| [P01](../../slices/projects.md#p01-repository-association-and-creation) / active | 240–246 | Native creation request validation; declaration/member validate; declarations/fields: `validate` |
| [P01](../../slices/projects.md#p01-repository-association-and-creation) / active | 247–251 | Native creation request validation; declaration/member from_value; declarations/fields: `from_value` |
| [P01](../../slices/projects.md#p01-repository-association-and-creation) / active | 252–261 | Native creation request validation; declaration/member from_map; declarations/fields: `from_map` |
| [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) / active | 262, 267–270, 294–298 | Environment observation; declarations/fields: `Environment`, `Environment.running`, `encode_into`, `encode` |
| [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) / active | 263 | Environment observation; declaration/member Environment.image; declarations/fields: `Environment.image` |
| [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) / active | 264 | Environment observation; declaration/member Environment.profile; declarations/fields: `Environment.profile` |
| [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) / active | 265 | Environment observation; declaration/member Environment.id; declarations/fields: `Environment.id` |
| [N02](../../slices/networking.md#n02-project-lan-access) / active | 266 | Native LAN address observation field within Environment; declarations/fields: `Environment.ip` |
| [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) / active | 271–291 | Environment observation; declaration/member encode_into; declarations/fields: `encode_into` |
| [N02](../../slices/networking.md#n02-project-lan-access) / active | 292–293 | Encode observed native Project LAN address; declarations/fields: `Environment.ip` |
| [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) / active | 299–306 | Environment observation; declaration/member encode; declarations/fields: `encode` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 307 | Developer SSH endpoint contract; declarations/fields: `Connection` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 308 | Developer SSH endpoint contract; declaration/member Connection.environment; declarations/fields: `Connection.environment` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 309 | Developer SSH endpoint contract; declaration/member Connection.host_key; declarations/fields: `Connection.host_key` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 310–313 | Developer SSH endpoint contract; declaration/member Connection.fingerprint; declarations/fields: `Connection.fingerprint` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 314–326 | Developer SSH endpoint contract; declaration/member encode; declarations/fields: `encode` |
| [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) / active | 327 | Native OS release observation; declarations/fields: `OsRelease` |
| [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) / active | 328 | Native OS release observation; declaration/member OsRelease.id; declarations/fields: `OsRelease.id` |
| [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) / active | 329 | Native OS release observation; declaration/member OsRelease.version; declarations/fields: `OsRelease.version` |
| [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) / active | 330–333 | Native OS release observation; declaration/member OsRelease.name; declarations/fields: `OsRelease.name` |
| [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) / active | 334–345 | Native OS release observation; declaration/member encode_into; declarations/fields: `encode_into` |
| [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) / active | 346 | Native OS release observation; declaration/member OsObservation; declarations/fields: `OsObservation` |
| [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) / active | 347 | Native OS release observation; declaration/member OsObservation.environment; declarations/fields: `OsObservation.environment` |
| [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) / active | 348 | Native OS release observation; declaration/member OsObservation.release; declarations/fields: `OsObservation.release` |
| [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) / active | 349–352 | Native OS release observation; declaration/member OsObservation.unavailable; declarations/fields: `OsObservation.unavailable` |
| [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) / active | 353–367 | Native OS release observation; declaration/member encode; declarations/fields: `encode` |
| [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) / active | 368–378 | Native OS release observation; declaration/member valid_os_id; declarations/fields: `valid_os_id` |
| [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) / active | 379–390 | Native OS release observation; declaration/member valid_os_version; declarations/fields: `valid_os_version` |
| [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) / active | 391–410 | Native OS release observation; declaration/member valid_os_release; declarations/fields: `valid_os_release` |
| [P03](../../slices/projects.md#p03-human-membership-and-accounts) / active | 411 | Human account association request; declarations/fields: `Account` |
| [P03](../../slices/projects.md#p03-human-membership-and-accounts) / active | 412 | Human account association request; declaration/member Account.project; declarations/fields: `Account.project` |
| [P03](../../slices/projects.md#p03-human-membership-and-accounts) / active | 413 | Human account association request; declaration/member Account.login; declarations/fields: `Account.login` |
| [P03](../../slices/projects.md#p03-human-membership-and-accounts) / active | 414 | Human account association request; declaration/member Account.identity; declarations/fields: `Account.identity` |
| [P03](../../slices/projects.md#p03-human-membership-and-accounts) / active | 415–417 | Human account association request; declaration/member Account.keys; declarations/fields: `Account.keys` |
| [P03](../../slices/projects.md#p03-human-membership-and-accounts) / active | 418–437 | Human account association request; declaration/member ACCOUNT_SPECS; declarations/fields: `ACCOUNT_SPECS` |
| [P03](../../slices/projects.md#p03-human-membership-and-accounts) / active | 438–442 | Human account association request; declaration/member from_value; declarations/fields: `from_value` |
| [P03](../../slices/projects.md#p03-human-membership-and-accounts) / active | 443–454 | Human account association request; declaration/member from_map; declarations/fields: `from_map` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 455 | Managed access keys and revision-fenced state; declarations/fields: `AccessKeys` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 456 | Managed access keys and revision-fenced state; declaration/member AccessKeys.project; declarations/fields: `AccessKeys.project` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 457 | Managed access keys and revision-fenced state; declaration/member AccessKeys.login; declarations/fields: `AccessKeys.login` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 458 | Managed access keys and revision-fenced state; declaration/member AccessKeys.identity; declarations/fields: `AccessKeys.identity` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 459 | Managed access keys and revision-fenced state; declaration/member AccessKeys.revision; declarations/fields: `AccessKeys.revision` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 460 | Managed access keys and revision-fenced state; declaration/member AccessKeys.keys; declarations/fields: `AccessKeys.keys` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 461–463 | Managed access keys and revision-fenced state; declaration/member AccessKeys.apply; declarations/fields: `AccessKeys.apply` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 464–491 | Managed access keys and revision-fenced state; declaration/member ACCESS_KEYS_SPECS; declarations/fields: `ACCESS_KEYS_SPECS` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 492–496 | Managed access keys and revision-fenced state; declaration/member from_value; declarations/fields: `from_value` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 497–510 | Managed access keys and revision-fenced state; declaration/member from_map; declarations/fields: `from_map` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 511 | Managed access keys and revision-fenced state; declaration/member AccessKeyState; declarations/fields: `AccessKeyState` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 512 | Managed access keys and revision-fenced state; declaration/member AccessKeyState.revision; declarations/fields: `AccessKeyState.revision` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 513–517 | Managed access keys and revision-fenced state; declaration/member AccessKeyState.keys; declarations/fields: `AccessKeyState.keys` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 518–533 | Managed access keys and revision-fenced state; declaration/member encode; declarations/fields: `encode` |
| [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) / active | 534–536, 607–628 | Pure Project source tests; declarations/fields: `tests`, `environment_omits_empty_like_go` |
| [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) / active | 537–549 | Pure Project source tests; declaration/member sample_profile; declarations/fields: `sample_profile` |
| [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) / active | 550–566 | Pure Project source tests; declaration/member validators_match_go_regexps; declarations/fields: `validators_match_go_regexps` |
| [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) / active | 567–580 | Source assertion of Profile and runtime readiness; declarations/fields: `profile_validation_matches` |
| [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) / active | 581–591 | Source assertion of Profile and runtime readiness; declaration/member profile_wire_round_trip; declarations/fields: `profile_wire_round_trip` |
| [P01](../../slices/projects.md#p01-repository-association-and-creation) / active | 592–606 | Source assertion of Repository association and creation; declarations/fields: `create_decode_and_validate` |
| [I04](../../slices/identity-brokering.md#i04-execution-admission-and-lease-fencing) / active | 629–662 | Source assertion of Execution admission and lease fencing; declarations/fields: `os_release_validation_matches_go` |
| [P03](../../slices/projects.md#p03-human-membership-and-accounts) / active | 663–687 | Source assertion of Human membership and accounts; declarations/fields: `account_dto_strict_shape` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 688–701 | Source assertion of Development SSH access; declarations/fields: `access_keys_dto_strict_shape` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 702–709 | Source assertion of Development SSH access; declaration/member access_key_state_encodes_struct_order; declarations/fields: `access_key_state_encodes_struct_order` |

<a id="coverage-4263fb98abf4"></a>

## [rust/soda-host/src/pops.rs](../../../../../rust/soda-host/src/pops.rs)



| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 1–125 | Bounded strict operation input decoding |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 126–137 | Managed SSH key request decode; declarations/fields: `AccessKeysReq`, `decode` |
| [P03](../../slices/projects.md#p03-human-membership-and-accounts) / active | 138–149 | Human account request decode; declarations/fields: `AccountReq`, `decode` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 150–197 | Preparation request and result adapters; declarations/fields: `PrepareReq`, `decode`, `PrepareCandidateReq`, `InspectPreparationReq`, `StopPreparationReq` |
| [P12](../../slices/projects.md#p12-maintenance-holds) / active | 198–211 | Maintenance hold request/result; declarations/fields: `HoldPreparationReq`, `decode` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 212–226 | Native preparation Ops construction; declarations/fields: `Ops`, `runtime` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 227–232 | Native key operation; declarations/fields: `access_keys` |
| [P03](../../slices/projects.md#p03-human-membership-and-accounts) / active | 233–238 | Native account operation; declarations/fields: `account` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 239–278 | Preparation/candidate/inspect/stop operations; declarations/fields: `prepare`, `prepare_candidate`, `inspect_preparation`, `stop_preparation` |
| [P12](../../slices/projects.md#p12-maintenance-holds) / active | 279–286 | Maintenance hold operation; declarations/fields: `hold_preparation` |

<a id="coverage-6695c792cf9e"></a>

<a id="rustsoda-hostsrcprojectrs-1"></a>

## [rust/soda-host/src/project.rs](../../../../../rust/soda-host/src/project.rs)

Source assertions/fixtures were inspected for mapping only; no test execution, runtime or installed proof claimed. Large-file ledger retains committed declaration/member spans rather than grouping methods into one whole-file unit; responsibility follows the verified concern/branch mapping, not declaration count.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) / active | 1–27 | Project profile/native execution metadata; declarations/fields: `PROFILE_INSPECT_FORMAT`, `INSPECTION_SPECS`, `PROJECT_INSPECT_FORMAT` |
| [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) / active | 28–30 | Project profile/native execution metadata; declaration/member PROFILE_INSPECT_FORMAT; declarations/fields: `PROFILE_INSPECT_FORMAT` |
| [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) / active | 31–74 | Project profile/native execution metadata; declaration/member INSPECTION_SPECS; declarations/fields: `INSPECTION_SPECS` |
| [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) / active | 75–77 | Project profile/native execution metadata; declaration/member PROJECT_INSPECT_FORMAT; declarations/fields: `PROJECT_INSPECT_FORMAT` |
| [H04](../../slices/shared-supporting-slices.md#h04-configuration-and-filesystem-primitives) / active | 78 | Executor configuration; declarations/fields: `Config` |
| [H04](../../slices/shared-supporting-slices.md#h04-configuration-and-filesystem-primitives) / active | 79 | Executor configuration; declaration/member Config.muse_socket; declarations/fields: `Config.muse_socket` |
| [H04](../../slices/shared-supporting-slices.md#h04-configuration-and-filesystem-primitives) / active | 80 | Executor configuration; declaration/member Config.image; declarations/fields: `Config.image` |
| [H04](../../slices/shared-supporting-slices.md#h04-configuration-and-filesystem-primitives) / active | 81 | Executor configuration; declaration/member Config.network; declarations/fields: `Config.network` |
| [H04](../../slices/shared-supporting-slices.md#h04-configuration-and-filesystem-primitives) / active | 82 | Executor configuration; declaration/member Config.subnet; declarations/fields: `Config.subnet` |
| [H04](../../slices/shared-supporting-slices.md#h04-configuration-and-filesystem-primitives) / active | 83–87 | Executor configuration; declaration/member Config.bridge; declarations/fields: `Config.bridge` |
| [P05](../../slices/projects.md#p05-project-startstop) / active | 88 | Project Start/Stop lifecycle contract; declarations/fields: `Lifecycle` |
| [P05](../../slices/projects.md#p05-project-startstop) / active | 89 | Project Start/Stop lifecycle contract; declaration/member Lifecycle.project; declarations/fields: `Lifecycle.project` |
| [P05](../../slices/projects.md#p05-project-startstop) / active | 90–92 | Project Start/Stop lifecycle contract; declaration/member Lifecycle.action; declarations/fields: `Lifecycle.action` |
| [P05](../../slices/projects.md#p05-project-startstop) / active | 93–106 | Project Start/Stop lifecycle contract; declaration/member LIFECYCLE_SPECS; declarations/fields: `LIFECYCLE_SPECS` |
| [P05](../../slices/projects.md#p05-project-startstop) / active | 107–111 | Project Start/Stop lifecycle contract; declaration/member from_value; declarations/fields: `from_value` |
| [P05](../../slices/projects.md#p05-project-startstop) / active | 112–121 | Project Start/Stop lifecycle contract; declaration/member from_map; declarations/fields: `from_map` |
| [P05](../../slices/projects.md#p05-project-startstop) / active | 122 | Project Start/Stop lifecycle contract; declaration/member LifecycleState; declarations/fields: `LifecycleState` |
| [P05](../../slices/projects.md#p05-project-startstop) / active | 123 | Project Start/Stop lifecycle contract; declaration/member LifecycleState.environment; declarations/fields: `LifecycleState.environment` |
| [P05](../../slices/projects.md#p05-project-startstop) / active | 124–127 | Project Start/Stop lifecycle contract; declaration/member LifecycleState.boot_enabled; declarations/fields: `LifecycleState.boot_enabled` |
| [P05](../../slices/projects.md#p05-project-startstop) / active | 128–138 | Project Start/Stop lifecycle contract; declaration/member encode; declarations/fields: `encode` |
| [P05](../../slices/projects.md#p05-project-startstop) / active | 139–140 | Project Start/Stop lifecycle contract; declaration/member PROJECT_UNIT_PATH; declarations/fields: `PROJECT_UNIT_PATH` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 141 | Native privileged command executor mechanics; declarations/fields: `Executor` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 142–150, 157–166, 178–238 | Native privileged command executor mechanics; declaration/member run; declarations/fields: `run` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 151–156, 167–174, 239–244 | Native privileged command executor mechanics; declaration/member is_host_native; declarations/fields: `is_host_native` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 175–177 | Native privileged command executor mechanics; declaration/member Native; declarations/fields: `Native` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 245–269 | Native privileged command executor mechanics; declaration/member exit_text; declarations/fields: `exit_text` |
| [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) / active | 270 | Project runtime, inspect and isolation/readiness admission; declarations/fields: `Runtime` |
| [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) / active | 271 | Project runtime, inspect and isolation/readiness admission; declaration/member Runtime.exec; declarations/fields: `Runtime.exec` |
| [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) / active | 272–275 | Project runtime, inspect and isolation/readiness admission; declaration/member Runtime.config; declarations/fields: `Runtime.config` |
| [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) / active | 276–285 | Project runtime, inspect and isolation/readiness admission; declaration/member podman; declarations/fields: `podman` |
| [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) / active | 286–361 | Project runtime, inspect and isolation/readiness admission; declaration/member resolve_profile; declarations/fields: `resolve_profile` |
| [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) / active | 362–462 | Project runtime, inspect and isolation/readiness admission; declaration/member inspect; declarations/fields: `inspect` |
| [P01](../../slices/projects.md#p01-repository-association-and-creation) / active | 463–472 | Reserved Project creation and native unit materialization; declarations/fields: `create` |
| [P01](../../slices/projects.md#p01-repository-association-and-creation) / active | 473–538 | Reserved Project creation and native unit materialization; declaration/member create_container; declarations/fields: `create_container` |
| [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) / active | 539–565 | Creation startup, health and ready observations; declarations/fields: `start_created` |
| [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) / active | 566–594 | Creation startup, health and ready observations; declaration/member wait_project_ready; declarations/fields: `wait_project_ready` |
| [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) / active | 595–648 | Creation startup, health and ready observations; declaration/member project_container; declarations/fields: `project_container` |
| [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) / active | 649–699 | Native OS observation; declarations/fields: `observe_os` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 700–736 | Developer SSH connection/host-key observation; declarations/fields: `connection` |
| [P05](../../slices/projects.md#p05-project-startstop) / active | 737–763 | Lifecycle inspection and explicit Start/Stop; declarations/fields: `lifecycle` |
| [P05](../../slices/projects.md#p05-project-startstop) / active | 764–782 | Lifecycle inspection and explicit Start/Stop; declaration/member read_project_unit; declarations/fields: `read_project_unit` |
| [P05](../../slices/projects.md#p05-project-startstop) / active | 783–807 | Lifecycle inspection and explicit Start/Stop; declaration/member apply_lifecycle_action; declarations/fields: `apply_lifecycle_action` |
| [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) / active | 808–818 | Native architecture gate; declarations/fields: `go_arch` |
| [H04](../../slices/shared-supporting-slices.md#h04-configuration-and-filesystem-primitives) / active | 819–837 | Native protected state directories; declarations/fields: `go_dir` |
| [P05](../../slices/projects.md#p05-project-startstop) / active | 838–843 | Native Project unit rendering and action invocation; declarations/fields: `valid_lifecycle_action` |
| [P05](../../slices/projects.md#p05-project-startstop) / active | 844–868 | Native Project unit rendering and action invocation; declaration/member parse_unit_show_properties; declarations/fields: `parse_unit_show_properties` |
| [P05](../../slices/projects.md#p05-project-startstop) / active | 869 | Native Project unit rendering and action invocation; declaration/member validate_unit_properties; declarations/fields: `validate_unit_properties` |
| [P05](../../slices/projects.md#p05-project-startstop) / active | 870–887 | Native Project unit rendering and action invocation; declaration/member ERR; declarations/fields: `ERR` |
| [P05](../../slices/projects.md#p05-project-startstop) / active | 888–907 | Native Project unit rendering and action invocation; declaration/member verify_lifecycle_outcome; declarations/fields: `verify_lifecycle_outcome` |
| [P05](../../slices/projects.md#p05-project-startstop) / active | 908–914 | Native Project unit rendering and action invocation; declaration/member valid_address; declarations/fields: `valid_address` |
| [P05](../../slices/projects.md#p05-project-startstop) / active | 915–923 | Native Project unit rendering and action invocation; declaration/member lifecycle_action_confirmed; declarations/fields: `lifecycle_action_confirmed` |
| [P05](../../slices/projects.md#p05-project-startstop) / active | 924–939 | Native Project unit rendering and action invocation; declaration/member confirm_lifecycle; declarations/fields: `confirm_lifecycle` |
| [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) / active | 940–947 | Profile/create/OS result confirmation; declarations/fields: `confirm_resolve_profile` |
| [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) / active | 948–962 | Profile/create/OS result confirmation; declaration/member valid_os_environment; declarations/fields: `valid_os_environment` |
| [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) / active | 963–976 | Profile/create/OS result confirmation; declaration/member valid_os_observation; declarations/fields: `valid_os_observation` |
| [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) / active | 977–987 | Profile/create/OS result confirmation; declaration/member confirm_observe_os; declarations/fields: `confirm_observe_os` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 988–1001 | Developer SSH endpoint confirmation; declarations/fields: `confirm_access_keys` |
| [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) / active | 1002–1009 | Container profile labels and isolation/native OS parsing; declarations/fields: `tolerant_str` |
| [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) / active | 1010–1032 | Container profile labels and isolation/native OS parsing; declaration/member labels_of; declarations/fields: `labels_of` |
| [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) / active | 1033–1067 | Container profile labels and isolation/native OS parsing; declaration/member apply_creation_profile; declarations/fields: `apply_creation_profile` |
| [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) / active | 1068–1080 | Container profile labels and isolation/native OS parsing; declaration/member project_id_map; declarations/fields: `project_id_map` |
| [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) / active | 1081–1097 | Container profile labels and isolation/native OS parsing; declaration/member is_python_space; declarations/fields: `is_python_space` |
| [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) / active | 1098–1142 | Container profile labels and isolation/native OS parsing; declaration/member split_lines; declarations/fields: `split_lines` |
| [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) / active | 1143–1208 | Container profile labels and isolation/native OS parsing; declaration/member shlex_posix; declarations/fields: `shlex_posix` |
| [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) / active | 1209–1218 | Container profile labels and isolation/native OS parsing; declaration/member valid_release_id; declarations/fields: `valid_release_id` |
| [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) / active | 1219–1228 | Container profile labels and isolation/native OS parsing; declaration/member valid_release_version; declarations/fields: `valid_release_version` |
| [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) / active | 1229–1277 | Container profile labels and isolation/native OS parsing; declaration/member parse_os_release; declarations/fields: `parse_os_release` |
| [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) / active | 1278–1282 | Container profile labels and isolation/native OS parsing; declaration/member tests; declarations/fields: `tests` |
| [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) / active | 1283, 1392–1414, 1439–1505, 1522–1562, 1645–1673 | Project source fixtures and assertions; declarations/fields: `Mock`, `container_inspect`, `inspect_rejects_mismatch_and_bad_network`, `project_container_binds_isolated_identity`, `connection_returns_ed25519_material` |
| [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) / active | 1284 | Project source fixtures and assertions; declaration/member Mock.calls; declarations/fields: `Mock.calls` |
| [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) / active | 1285–1288 | Project source fixtures and assertions; declaration/member Mock.script; declarations/fields: `Mock.script` |
| [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) / active | 1289–1297 | Project source fixtures and assertions; declaration/member new; declarations/fields: `new` |
| [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) / active | 1298–1315 | Project source fixtures and assertions; declaration/member run; declarations/fields: `run` |
| [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) / active | 1316–1319 | Project source fixtures and assertions; declaration/member deadline; declarations/fields: `deadline` |
| [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) / active | 1320–1329 | Project source fixtures and assertions; declaration/member test_config; declarations/fields: `test_config` |
| [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) / active | 1330–1341 | Project source fixtures and assertions; declaration/member sample_profile; declarations/fields: `sample_profile` |
| [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) / active | 1342–1350 | Project source fixtures and assertions; declaration/member image_inspect; declarations/fields: `image_inspect` |
| [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) / active | 1351–1364, 1415–1438, 2053–2204 | Source assertion of Profile and runtime readiness; declarations/fields: `resolve_profile_accepts_native_image`, `inspect_observes_identity_and_profile`, `address_profile_os_key_confirmations` |
| [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) / active | 1365–1391 | Source assertion of Profile and runtime readiness; declaration/member resolve_profile_refuses_foreign_or_invalid; declarations/fields: `resolve_profile_refuses_foreign_or_invalid` |
| [P01](../../slices/projects.md#p01-repository-association-and-creation) / active | 1506–1521 | Source assertion of Repository association and creation; declarations/fields: `create_validates_before_exec` |
| [I04](../../slices/identity-brokering.md#i04-execution-admission-and-lease-fencing) / active | 1563–1605 | Source assertion of Execution admission and lease fencing; declarations/fields: `os_release_parsing_mirrors_python_cases` |
| [I03](../../slices/identity-brokering.md#i03-delegation-and-connection-availability) / active | 1606–1644 | Source assertion of Delegation and connection availability; declarations/fields: `observe_os_reports_unavailable_without_starting` |
| [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) / active | 1674–1680 | Project source fixtures and assertions; declaration/member format_inspect; declarations/fields: `format_inspect` |
| [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) / active | 1681–1688 | Project source fixtures and assertions; declaration/member unit_show; declarations/fields: `unit_show` |
| [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) / active | 1689–1743 | Project source fixtures and assertions; declaration/member unit_show_parsing_matrix; declarations/fields: `unit_show_parsing_matrix` |
| [P05](../../slices/projects.md#p05-project-startstop) / active | 1744–1842 | Source assertion of Project Start/Stop; declarations/fields: `lifecycle_start_stop_inspect_flows` |
| [P05](../../slices/projects.md#p05-project-startstop) / active | 1843–1967 | Source assertion of Project Start/Stop; declaration/member lifecycle_rejects_identity_change_and_bad_outcome; declarations/fields: `lifecycle_rejects_identity_change_and_bad_outcome` |
| [P05](../../slices/projects.md#p05-project-startstop) / active | 1968–2052 | Source assertion of Project Start/Stop; declaration/member lifecycle_facade_confirmations; declarations/fields: `lifecycle_facade_confirmations` |

<a id="coverage-c2acb1beeebb"></a>

<a id="rustsoda-hostsrcsshrs-1"></a>

## [rust/soda-host/src/ssh.rs](../../../../../rust/soda-host/src/ssh.rs)

Source assertions/fixtures were inspected for mapping only; no test execution, runtime or installed proof claimed. Large-file ledger retains committed declaration/member spans rather than grouping methods into one whole-file unit; responsibility follows the verified concern/branch mapping, not declaration count.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 1–25, 1022–1028 | SSH public-key parsing, host key validation and canonical developer-key forms; declarations/fields: `trim_ws`, `b64_value`, `b64_decode`, `decode_quantum_go`, `b64_decode_go`, `b64_corrupt`, `B64_STD`, `b64_encode`, `b64_encode_raw`, `read_string`, `read_u32`, `read_u64`, `put_string`, `Mpint`, `strip_zeros`, `parse_mpint`, `mpint_bitlen`, `mpint_cmp`, `mpint_to_i64`, `marshal_mpint`, `ALGO_RSA`, `ALGO_DSS`, `ALGO_ECDSA256`, `ALGO_ECDSA384`, `ALGO_ECDSA521`, `ALGO_SKECDSA`, `ALGO_ED25519`, `ALGO_SKED25519`, `cert_inner`, `is_cert_algo`, `ParsedKey`, `KeyMaterial`, `CertMaterial`, `curve_for`, `parse_key_fields`, `parse_tuples`, `parse_cert`, `marshal_tuples`, `marshal_material`, `marshal_fields`, `parse_public_key`, `parse_key_text`, `scan_options`, `parse_authorized_key`, `marshal_authorized_key`, `fingerprint_sha256`, `tests`, `RSA`, `ECDSA256`, `ECDSA384`, `ECDSA521`, `ED`, `DSA`, `CERT`, `canonical`, `all_types_round_trip_canonically`, `sk_types_parse`, `comments_and_whitespace_tolerated_but_not_canonical`, `options_detected_and_multiline_rules_match`, `invalid_keys_rejected`, `fingerprint_shape` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 26–64 | SSH public-key parsing, host key validation and canonical developer-key forms; declaration/member trim_ws; declarations/fields: `trim_ws` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 65–75 | SSH public-key parsing, host key validation and canonical developer-key forms; declaration/member b64_value; declarations/fields: `b64_value` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 76–121 | SSH public-key parsing, host key validation and canonical developer-key forms; declaration/member b64_decode; declarations/fields: `b64_decode` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 122–186 | SSH public-key parsing, host key validation and canonical developer-key forms; declaration/member decode_quantum_go; declarations/fields: `decode_quantum_go` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 187–199 | SSH public-key parsing, host key validation and canonical developer-key forms; declaration/member b64_decode_go; declarations/fields: `b64_decode_go` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 200–203 | SSH public-key parsing, host key validation and canonical developer-key forms; declaration/member b64_corrupt; declarations/fields: `b64_corrupt` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 204–205 | SSH public-key parsing, host key validation and canonical developer-key forms; declaration/member B64_STD; declarations/fields: `B64_STD` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 206–224 | SSH public-key parsing, host key validation and canonical developer-key forms; declaration/member b64_encode; declarations/fields: `b64_encode` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 225–230 | SSH public-key parsing, host key validation and canonical developer-key forms; declaration/member b64_encode_raw; declarations/fields: `b64_encode_raw` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 231–241 | SSH public-key parsing, host key validation and canonical developer-key forms; declaration/member read_string; declarations/fields: `read_string` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 242–251 | SSH public-key parsing, host key validation and canonical developer-key forms; declaration/member read_u32; declarations/fields: `read_u32` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 252–258 | SSH public-key parsing, host key validation and canonical developer-key forms; declaration/member read_u64; declarations/fields: `read_u64` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 259–265 | SSH public-key parsing, host key validation and canonical developer-key forms; declaration/member put_string; declarations/fields: `put_string` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 266 | SSH public-key parsing, host key validation and canonical developer-key forms; declaration/member Mpint; declarations/fields: `Mpint` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 267 | SSH public-key parsing, host key validation and canonical developer-key forms; declaration/member Mpint.negative; declarations/fields: `Mpint.negative` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 268–270 | SSH public-key parsing, host key validation and canonical developer-key forms; declaration/member Mpint.mag; declarations/fields: `Mpint.mag` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 271–275 | SSH public-key parsing, host key validation and canonical developer-key forms; declaration/member strip_zeros; declarations/fields: `strip_zeros` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 276–314 | SSH public-key parsing, host key validation and canonical developer-key forms; declaration/member parse_mpint; declarations/fields: `parse_mpint` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 315–322 | SSH public-key parsing, host key validation and canonical developer-key forms; declaration/member mpint_bitlen; declarations/fields: `mpint_bitlen` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 323–357 | SSH public-key parsing, host key validation and canonical developer-key forms; declaration/member mpint_cmp; declarations/fields: `mpint_cmp` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 358–372 | SSH public-key parsing, host key validation and canonical developer-key forms; declaration/member mpint_to_i64; declarations/fields: `mpint_to_i64` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 373–403 | SSH public-key parsing, host key validation and canonical developer-key forms; declaration/member marshal_mpint; declarations/fields: `marshal_mpint` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 404 | SSH public-key parsing, host key validation and canonical developer-key forms; declaration/member ALGO_RSA; declarations/fields: `ALGO_RSA` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 405 | SSH public-key parsing, host key validation and canonical developer-key forms; declaration/member ALGO_DSS; declarations/fields: `ALGO_DSS` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 406 | SSH public-key parsing, host key validation and canonical developer-key forms; declaration/member ALGO_ECDSA256; declarations/fields: `ALGO_ECDSA256` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 407 | SSH public-key parsing, host key validation and canonical developer-key forms; declaration/member ALGO_ECDSA384; declarations/fields: `ALGO_ECDSA384` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 408 | SSH public-key parsing, host key validation and canonical developer-key forms; declaration/member ALGO_ECDSA521; declarations/fields: `ALGO_ECDSA521` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 409 | SSH public-key parsing, host key validation and canonical developer-key forms; declaration/member ALGO_SKECDSA; declarations/fields: `ALGO_SKECDSA` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 410 | SSH public-key parsing, host key validation and canonical developer-key forms; declaration/member ALGO_ED25519; declarations/fields: `ALGO_ED25519` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 411–412 | SSH public-key parsing, host key validation and canonical developer-key forms; declaration/member ALGO_SKED25519; declarations/fields: `ALGO_SKED25519` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 413–426 | SSH public-key parsing, host key validation and canonical developer-key forms; declaration/member cert_inner; declarations/fields: `cert_inner` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 427–435 | SSH public-key parsing, host key validation and canonical developer-key forms; declaration/member is_cert_algo; declarations/fields: `is_cert_algo` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 436 | SSH public-key parsing, host key validation and canonical developer-key forms; declaration/member ParsedKey; declarations/fields: `ParsedKey` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 437 | SSH public-key parsing, host key validation and canonical developer-key forms; declaration/member ParsedKey.key_type; declarations/fields: `ParsedKey.key_type` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 438–441 | SSH public-key parsing, host key validation and canonical developer-key forms; declaration/member ParsedKey.blob; declarations/fields: `ParsedKey.blob` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 442–471 | SSH public-key parsing, host key validation and canonical developer-key forms; declaration/member KeyMaterial; declarations/fields: `KeyMaterial` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 472 | SSH public-key parsing, host key validation and canonical developer-key forms; declaration/member CertMaterial; declarations/fields: `CertMaterial` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 473 | SSH public-key parsing, host key validation and canonical developer-key forms; declaration/member CertMaterial.nonce; declarations/fields: `CertMaterial.nonce` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 474 | SSH public-key parsing, host key validation and canonical developer-key forms; declaration/member CertMaterial.inner; declarations/fields: `CertMaterial.inner` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 475 | SSH public-key parsing, host key validation and canonical developer-key forms; declaration/member CertMaterial.serial; declarations/fields: `CertMaterial.serial` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 476 | SSH public-key parsing, host key validation and canonical developer-key forms; declaration/member CertMaterial.cert_type_num; declarations/fields: `CertMaterial.cert_type_num` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 477 | SSH public-key parsing, host key validation and canonical developer-key forms; declaration/member CertMaterial.key_id; declarations/fields: `CertMaterial.key_id` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 478 | SSH public-key parsing, host key validation and canonical developer-key forms; declaration/member CertMaterial.principals; declarations/fields: `CertMaterial.principals` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 479 | SSH public-key parsing, host key validation and canonical developer-key forms; declaration/member CertMaterial.valid_after; declarations/fields: `CertMaterial.valid_after` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 480 | SSH public-key parsing, host key validation and canonical developer-key forms; declaration/member CertMaterial.valid_before; declarations/fields: `CertMaterial.valid_before` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 481 | SSH public-key parsing, host key validation and canonical developer-key forms; declaration/member CertMaterial.critical_options; declarations/fields: `CertMaterial.critical_options` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 482 | SSH public-key parsing, host key validation and canonical developer-key forms; declaration/member CertMaterial.extensions; declarations/fields: `CertMaterial.extensions` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 483 | SSH public-key parsing, host key validation and canonical developer-key forms; declaration/member CertMaterial.reserved; declarations/fields: `CertMaterial.reserved` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 484 | SSH public-key parsing, host key validation and canonical developer-key forms; declaration/member CertMaterial.sig_key; declarations/fields: `CertMaterial.sig_key` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 485 | SSH public-key parsing, host key validation and canonical developer-key forms; declaration/member CertMaterial.sig_format; declarations/fields: `CertMaterial.sig_format` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 486–488 | SSH public-key parsing, host key validation and canonical developer-key forms; declaration/member CertMaterial.sig_blob; declarations/fields: `CertMaterial.sig_blob` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 489–497 | SSH public-key parsing, host key validation and canonical developer-key forms; declaration/member curve_for; declarations/fields: `curve_for` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 498–594 | SSH public-key parsing, host key validation and canonical developer-key forms; declaration/member parse_key_fields; declarations/fields: `parse_key_fields` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 595–596 | SSH public-key parsing, host key validation and canonical developer-key forms; declaration/member ParsedTuples; declarations/fields: `ParsedTuples` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 597–622 | SSH public-key parsing, host key validation and canonical developer-key forms; declaration/member parse_tuples; declarations/fields: `parse_tuples` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 623–680 | SSH public-key parsing, host key validation and canonical developer-key forms; declaration/member parse_cert; declarations/fields: `parse_cert` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 681–697 | SSH public-key parsing, host key validation and canonical developer-key forms; declaration/member marshal_tuples; declarations/fields: `marshal_tuples` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 698–702 | SSH public-key parsing, host key validation and canonical developer-key forms; declaration/member marshal_material; declarations/fields: `marshal_material` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 703–755 | SSH public-key parsing, host key validation and canonical developer-key forms; declaration/member marshal_fields; declarations/fields: `marshal_fields` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 756–775 | SSH public-key parsing, host key validation and canonical developer-key forms; declaration/member parse_public_key; declarations/fields: `parse_public_key` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 776–790 | SSH public-key parsing, host key validation and canonical developer-key forms; declaration/member parse_key_text; declarations/fields: `parse_key_text` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 791–820 | SSH public-key parsing, host key validation and canonical developer-key forms; declaration/member scan_options; declarations/fields: `scan_options` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 821–897 | SSH public-key parsing, host key validation and canonical developer-key forms; declaration/member parse_authorized_key; declarations/fields: `parse_authorized_key` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 898–902 | SSH public-key parsing, host key validation and canonical developer-key forms; declaration/member marshal_authorized_key; declarations/fields: `marshal_authorized_key` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 903–908 | SSH public-key parsing, host key validation and canonical developer-key forms; declaration/member fingerprint_sha256; declarations/fields: `fingerprint_sha256` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 909–913 | SSH public-key parsing, host key validation and canonical developer-key forms; declaration/member tests; declarations/fields: `tests` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 914 | SSH public-key parsing, host key validation and canonical developer-key forms; declaration/member RSA; declarations/fields: `RSA` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 915 | SSH public-key parsing, host key validation and canonical developer-key forms; declaration/member ECDSA256; declarations/fields: `ECDSA256` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 916 | SSH public-key parsing, host key validation and canonical developer-key forms; declaration/member ECDSA384; declarations/fields: `ECDSA384` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 917 | SSH public-key parsing, host key validation and canonical developer-key forms; declaration/member ECDSA521; declarations/fields: `ECDSA521` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 918–919 | SSH public-key parsing, host key validation and canonical developer-key forms; declaration/member ED; declarations/fields: `ED` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 920 | SSH public-key parsing, host key validation and canonical developer-key forms; declaration/member DSA; declarations/fields: `DSA` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 921–922 | SSH public-key parsing, host key validation and canonical developer-key forms; declaration/member CERT; declarations/fields: `CERT` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 923–928 | SSH public-key parsing, host key validation and canonical developer-key forms; declaration/member canonical; declarations/fields: `canonical` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 929–937 | SSH public-key parsing, host key validation and canonical developer-key forms; declaration/member all_types_round_trip_canonically; declarations/fields: `all_types_round_trip_canonically` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 938–951 | SSH public-key parsing, host key validation and canonical developer-key forms; declaration/member sk_types_parse; declarations/fields: `sk_types_parse` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 952–968 | SSH public-key parsing, host key validation and canonical developer-key forms; declaration/member comments_and_whitespace_tolerated_but_not_canonical; declarations/fields: `comments_and_whitespace_tolerated_but_not_canonical` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 969–985 | SSH public-key parsing, host key validation and canonical developer-key forms; declaration/member options_detected_and_multiline_rules_match; declarations/fields: `options_detected_and_multiline_rules_match` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 986–1011 | SSH public-key parsing, host key validation and canonical developer-key forms; declaration/member invalid_keys_rejected; declarations/fields: `invalid_keys_rejected` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 1012–1021 | Source assertion of Encoding and parsing; declarations/fields: `base64_vectors` |

<a id="coverage-71b62fe672bf"></a>

## [rust/soda-host/tests/pops_oracle.rs](../../../../../rust/soda-host/tests/pops_oracle.rs)

Source assertions/fixtures were inspected for mapping only; no test execution, runtime or installed proof claimed. Large-file ledger retains committed declaration/member spans rather than grouping methods into one whole-file unit; responsibility follows the verified concern/branch mapping, not declaration count.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 1–2, 5–13, 330–342, 573–582, 584, 648, 853–870 | Scripted native account/key/preparation/hold operation and byte oracle; declarations/fields: `pops`, `PID`, `FID`, `REV`, `COMMIT`, `ED`, `GO_ACCESS_KEY_STATE`, `GO_ACCESS_KEY_STATE_EMPTY`, `GO_HOLD_TRUE`, `GO_HOLD_FALSE`, `GO_PREPARE_FULL`, `GO_PREPARE_READY`, `GO_PREPARE_STOPPED`, `GO_PREPARE_MISSING`, `GO_KEYS_BODY_APPLY`, `GO_ACCOUNT_BODY`, `GO_HELPER_INSPECT`, `GO_UNKNOWN_FIELD`, `GO_NESTED_TYPE_ERROR`, `GO_BAD_BASE64`, `Mock`, `new`, `run`, `deadline`, `test_config`, `ops`, `format_inspect`, `container_inspect`, `helper_state`, `approve_response`, `fixture_files`, `fixture_digest`, `prepare_body`, `oversize_body_rejected_before_shape`, `candidate_body`, `SRC`, `oracle_decode_errors_match_go_verbatim` |
| Historical; no active owner / obsolete | 3–4 | Superseded wiring/predecessor-owner narrative; current module/daemon/native release callers contradict it |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 14–25 | Scripted native account/key/preparation/hold operation and byte oracle; declaration/member pops; declarations/fields: `pops` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 26 | Scripted native account/key/preparation/hold operation and byte oracle; declaration/member PID; declarations/fields: `PID` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 27 | Scripted native account/key/preparation/hold operation and byte oracle; declaration/member FID; declarations/fields: `FID` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 28 | Scripted native account/key/preparation/hold operation and byte oracle; declaration/member REV; declarations/fields: `REV` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 29 | Scripted native account/key/preparation/hold operation and byte oracle; declaration/member COMMIT; declarations/fields: `COMMIT` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 30–31 | Scripted native account/key/preparation/hold operation and byte oracle; declaration/member ED; declarations/fields: `ED` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 32 | Scripted native account/key/preparation/hold operation and byte oracle; declaration/member GO_ACCESS_KEY_STATE; declarations/fields: `GO_ACCESS_KEY_STATE` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 33 | Scripted native account/key/preparation/hold operation and byte oracle; declaration/member GO_ACCESS_KEY_STATE_EMPTY; declarations/fields: `GO_ACCESS_KEY_STATE_EMPTY` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 34 | Scripted native account/key/preparation/hold operation and byte oracle; declaration/member GO_HOLD_TRUE; declarations/fields: `GO_HOLD_TRUE` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 35 | Scripted native account/key/preparation/hold operation and byte oracle; declaration/member GO_HOLD_FALSE; declarations/fields: `GO_HOLD_FALSE` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 36 | Scripted native account/key/preparation/hold operation and byte oracle; declaration/member GO_PREPARE_FULL; declarations/fields: `GO_PREPARE_FULL` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 37 | Scripted native account/key/preparation/hold operation and byte oracle; declaration/member GO_PREPARE_READY; declarations/fields: `GO_PREPARE_READY` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 38 | Scripted native account/key/preparation/hold operation and byte oracle; declaration/member GO_PREPARE_STOPPED; declarations/fields: `GO_PREPARE_STOPPED` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 39–40 | Scripted native account/key/preparation/hold operation and byte oracle; declaration/member GO_PREPARE_MISSING; declarations/fields: `GO_PREPARE_MISSING` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 41 | Scripted native account/key/preparation/hold operation and byte oracle; declaration/member GO_KEYS_BODY_APPLY; declarations/fields: `GO_KEYS_BODY_APPLY` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 42 | Scripted native account/key/preparation/hold operation and byte oracle; declaration/member GO_ACCOUNT_BODY; declarations/fields: `GO_ACCOUNT_BODY` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 43–44 | Scripted native account/key/preparation/hold operation and byte oracle; declaration/member GO_HELPER_INSPECT; declarations/fields: `GO_HELPER_INSPECT` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 45 | Scripted native account/key/preparation/hold operation and byte oracle; declaration/member GO_UNKNOWN_FIELD; declarations/fields: `GO_UNKNOWN_FIELD` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 46 | Scripted native account/key/preparation/hold operation and byte oracle; declaration/member GO_NESTED_TYPE_ERROR; declarations/fields: `GO_NESTED_TYPE_ERROR` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 47–50 | Scripted native account/key/preparation/hold operation and byte oracle; declaration/member GO_BAD_BASE64; declarations/fields: `GO_BAD_BASE64` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 51–52 | Scripted native account/key/preparation/hold operation and byte oracle; declaration/member MockCall; declarations/fields: `MockCall` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 53 | Scripted native account/key/preparation/hold operation and byte oracle; declaration/member Mock; declarations/fields: `Mock` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 54 | Scripted native account/key/preparation/hold operation and byte oracle; declaration/member Mock.calls; declarations/fields: `Mock.calls` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 55–58 | Scripted native account/key/preparation/hold operation and byte oracle; declaration/member Mock.script; declarations/fields: `Mock.script` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 59–67 | Scripted native account/key/preparation/hold operation and byte oracle; declaration/member new; declarations/fields: `new` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 68–86 | Scripted native account/key/preparation/hold operation and byte oracle; declaration/member run; declarations/fields: `run` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 87–90 | Scripted native account/key/preparation/hold operation and byte oracle; declaration/member deadline; declarations/fields: `deadline` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 91–100 | Scripted native account/key/preparation/hold operation and byte oracle; declaration/member test_config; declarations/fields: `test_config` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 101–111 | Scripted native account/key/preparation/hold operation and byte oracle; declaration/member ops; declarations/fields: `ops` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 112–119 | Scripted native account/key/preparation/hold operation and byte oracle; declaration/member format_inspect; declarations/fields: `format_inspect` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 120–128 | Scripted native account/key/preparation/hold operation and byte oracle; declaration/member container_inspect; declarations/fields: `container_inspect` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 129–154 | Scripted native account/key/preparation/hold operation and byte oracle; declaration/member helper_state; declarations/fields: `helper_state` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 155–162 | Scripted native account/key/preparation/hold operation and byte oracle; declaration/member approve_response; declarations/fields: `approve_response` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 163–169 | Scripted native account/key/preparation/hold operation and byte oracle; declaration/member fixture_files; declarations/fields: `fixture_files` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 170–175 | Scripted native account/key/preparation/hold operation and byte oracle; declaration/member fixture_digest; declarations/fields: `fixture_digest` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 176–189 | Scripted native account/key/preparation/hold operation and byte oracle; declaration/member prepare_body; declarations/fields: `prepare_body` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 190–224, 387–418, 837–852 | Source assertion of Development SSH access; declarations/fields: `access_keys_req_strict_shape`, `access_keys_observe_matches_golden`, `oracle_access_key_state_empty_encoding` |
| [P03](../../slices/projects.md#p03-human-membership-and-accounts) / active | 225–242, 480–504 | Source assertion of Human membership and accounts; declarations/fields: `account_req_strict_shape`, `account_provisions_login` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 243–272, 525–553, 583, 647, 673–699, 788–812 | Source assertion of Checkout allocation and preparation; declarations/fields: `prepare_req_strict_shape`, `prepare_ready_short_circuit_matches_golden`, `prepare_candidate_reuses_protected_snapshot`, `prepare_candidate_rejects_unready_source`, `inspect_preparation_matches_golden`, `oracle_prepare_state_full_encoding` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 273–289 | Source assertion of Checkout allocation and preparation; declaration/member prepare_candidate_req_strict_shape; declarations/fields: `prepare_candidate_req_strict_shape` |
| [P12](../../slices/projects.md#p12-maintenance-holds) / active | 290–329, 735–787 | Source assertion of Maintenance holds; declarations/fields: `inspect_stop_hold_req_shapes`, `hold_preparation_confirms_marker` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 343–351 | Scripted native account/key/preparation/hold operation and byte oracle; declaration/member ops_constructor_mirrors_runtime_fields; declarations/fields: `ops_constructor_mirrors_runtime_fields` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 352–361 | Scripted native account/key/preparation/hold operation and byte oracle; declaration/member binding_argv; declarations/fields: `binding_argv` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 362–371 | Scripted native account/key/preparation/hold operation and byte oracle; declaration/member helper_argv; declarations/fields: `helper_argv` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 372–386 | Scripted native account/key/preparation/hold operation and byte oracle; declaration/member helper_ops; declarations/fields: `helper_ops` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 419–439 | Source assertion of Development SSH access; declaration/member access_keys_apply_rechecks_revision; declarations/fields: `access_keys_apply_rechecks_revision` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 440–455 | Source assertion of Development SSH access; declaration/member access_keys_rejects_revision_drift; declarations/fields: `access_keys_rejects_revision_drift` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 456–479 | Source assertion of Development SSH access; declaration/member access_keys_validates_before_exec; declarations/fields: `access_keys_validates_before_exec` |
| [P03](../../slices/projects.md#p03-human-membership-and-accounts) / active | 505–524 | Source assertion of Human membership and accounts; declaration/member account_refuses_stopped_and_root; declarations/fields: `account_refuses_stopped_and_root` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 554–572 | Source assertion of Checkout allocation and preparation; declaration/member prepare_validates_before_exec; declarations/fields: `prepare_validates_before_exec` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 585–646, 649–672 | Scripted native account/key/preparation/hold operation and byte oracle; declaration/member NEW; declarations/fields: `NEW` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 700–734 | Source assertion of Checkout allocation and preparation; declaration/member stop_preparation_confirms_retirement; declarations/fields: `stop_preparation_confirms_retirement` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 813–836 | Source assertion of Checkout allocation and preparation; declaration/member oracle_prepare_state_missing_encoding; declarations/fields: `oracle_prepare_state_missing_encoding` |


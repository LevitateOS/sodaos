# Soda muse maintain

[Responsibility map index](README.md) · [Coverage scope](../README.md).

Current responsibility accounting at `519b76bd` (2026-10-07).
One slice owns each named duty; disjoint complete symbols may share an owner.
Compound fields/clauses may share a physical line with distinct selectors.
Disposition concerns the duty, not source validity or installed qualification.

<a id="coverage-dcae12a59b98"></a>

## [cmd/soda-muse-maintain/src/archive.rs](../../../../../cmd/soda-muse-maintain/src/archive.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–188; current module/import/attribute shell; declaration feed_archive; declaration ExactPread; fields fd, name, offset, remaining; declaration read; declaration EmitWriter; fields emit, error; declaration write; declaration flush; declaration archive_error; declaration tool_header; declaration emit_archive; declaration archive_tests | [D11](../../slices/release-and-installation.md#d11-host-installation-and-payload-application) | retained | Imports and module declarations wire cmd/soda-muse-maintain/src/archive.rs into its current native target.; 11 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-e141f4bb2460"></a>

## [cmd/soda-muse-maintain/src/archive_tests.rs](../../../../../cmd/soda-muse-maintain/src/archive_tests.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–194; current module/import/attribute shell; declaration ustar_header_metadata_is_explicit; declaration archive_is_deterministic_and_uses_declared_entries; declaration tool_replacement_preserves_other_files_and_refuses_symlinks; declaration streamed_stage_delivery; declaration wait_output_reports_status; declaration short_tool_file_ends_copy_with_eof | [D11](../../slices/release-and-installation.md#d11-host-installation-and-payload-application) | retained | Imports and module declarations wire cmd/soda-muse-maintain/src/archive_tests.rs into its current native target.; 7 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-9aa2aaa855cf"></a>

## [cmd/soda-muse-maintain/src/command.rs](../../../../../cmd/soda-muse-maintain/src/command.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–321; current module/import/attribute shell; declaration podman; declaration podman_streamed; declaration WriteHalfOnDrop; declaration drop; declaration FeederOwner; fields interrupt_fd, thread; declaration interrupt; declaration finish; declaration wait_output; declaration OUTPUT_LIMIT; declaration custody_tests; declaration child; declaration continuous_output_cannot_starve_deadline; declaration exited_child_with_inherited_pipe_is_failure; declaration silent_child_is_killed_at_absolute_deadline; declaration feeder_shutdown_interrupts_blocked_write_and_joins | [D11](../../slices/release-and-installation.md#d11-host-installation-and-payload-application) | retained | Imports and module declarations wire cmd/soda-muse-maintain/src/command.rs into its current native target.; 17 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-795fa6e15310"></a>

## [cmd/soda-muse-maintain/src/config.rs](../../../../../cmd/soda-muse-maintain/src/config.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–3, 21–35; current module/import/attribute shell; declaration load_config; declaration config_tests | [D11](../../slices/release-and-installation.md#d11-host-installation-and-payload-application) | retained | Imports and module declarations wire cmd/soda-muse-maintain/src/config.rs into its current native target.; load_config: implement the current host installation and payload application duty in config.rs.; Exercise the named native behavior in config.rs. — Current named units/source consumers; retained normalized source evidence records each selector |
| 4–20; declaration Config; fields muse_sha256, muse_version, muse_socket, identity_socket, codex_harness, codex_harness_sha256, codex_harness_version, tailnet_management, tailnet_image, image, network, subnet, bridge | [H04](../../slices/shared-supporting-slices.md#h04-configuration-and-filesystem-primitives) | retained | Config: implement the current configuration and safe filesystem primitives duty in config.rs. — current source cmd/soda-muse-maintain/src/config.rs; lines 4-20; module/caller wiring inspected |

<a id="coverage-53b3244b9f45"></a>

## [cmd/soda-muse-maintain/src/config_tests.rs](../../../../../cmd/soda-muse-maintain/src/config_tests.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–120; current module/import/attribute shell; declaration go_quoted_vectors; declaration config_read_error_shapes; declaration host_config_decode_vectors; declaration runtime_config_validation_order | [D11](../../slices/release-and-installation.md#d11-host-installation-and-payload-application) | retained | Imports and module declarations wire cmd/soda-muse-maintain/src/config_tests.rs into its current native target.; 5 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-242704bc518f"></a>

## [cmd/soda-muse-maintain/src/config_validation.rs](../../../../../cmd/soda-muse-maintain/src/config_validation.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–102; current module/import/attribute shell; declaration validate_runtime_config; declaration valid_tailnet_config; declaration validate_muse_runtime; declaration validate_identity_runtime; declaration valid_harness_version; declaration valid_network_names; declaration valid_network_name; declaration valid_tailnet_image | [D11](../../slices/release-and-installation.md#d11-host-installation-and-payload-application) | retained | Imports and module declarations wire cmd/soda-muse-maintain/src/config_validation.rs into its current native target.; 9 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-eea851854be2"></a>

## [cmd/soda-muse-maintain/src/config_wire.rs](../../../../../cmd/soda-muse-maintain/src/config_wire.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–166; current module/import/attribute shell; declaration decode_host_config; declaration deserialize; declaration ConfigVisitor; declaration Value; declaration expecting; declaration visit_map; declaration visit_unit; declaration set_string; declaration set_bool; declaration HOST_FIELDS; declaration host_field_slot; declaration go_quoted; declaration is_go_nonprint | [D11](../../slices/release-and-installation.md#d11-host-installation-and-payload-application) | retained | Imports and module declarations wire cmd/soda-muse-maintain/src/config_wire.rs into its current native target.; 14 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-476e3ce1f41b"></a>

## [cmd/soda-muse-maintain/src/filesystem.rs](../../../../../cmd/soda-muse-maintain/src/filesystem.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–160; current module/import/attribute shell; declaration go_base; declaration go_dir; declaration path_error; declaration go_errno; declaration last_errno; declaration Tool; fields name, fd, size; declaration drop; declaration load_tools; declaration open_tool; declaration trusted_tool; declaration verify_native; declaration filesystem_tests | [D11](../../slices/release-and-installation.md#d11-host-installation-and-payload-application) | retained | Imports and module declarations wire cmd/soda-muse-maintain/src/filesystem.rs into its current native target.; 13 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-98056e47cce4"></a>

## [cmd/soda-muse-maintain/src/filesystem_tests.rs](../../../../../cmd/soda-muse-maintain/src/filesystem_tests.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–124; current module/import/attribute shell; declaration sha256_known_answers; declaration native_errno_keeps_operation_and_path_context; declaration native_path_helpers_keep_fixed_tool_location; declaration tool_source_and_digest_admission | [D11](../../slices/release-and-installation.md#d11-host-installation-and-payload-application) | retained | Imports and module declarations wire cmd/soda-muse-maintain/src/filesystem_tests.rs into its current native target.; 5 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-0bb3ef5abfb1"></a>

## [cmd/soda-muse-maintain/src/interface.rs](../../../../../cmd/soda-muse-maintain/src/interface.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–172; current module/import/attribute shell; declaration INTERFACE_SCRIPT; declaration prepare_interface; declaration FdGuard; declaration drop; declaration attach_interface; declaration syscall_open_tree; declaration restrict_interface_mount; declaration attach_project_mount; declaration interface_tests | [D11](../../slices/release-and-installation.md#d11-host-installation-and-payload-application) | retained | Imports and module declarations wire cmd/soda-muse-maintain/src/interface.rs into its current native target.; 10 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-7f975165dd39"></a>

## [cmd/soda-muse-maintain/src/interface_admission.rs](../../../../../cmd/soda-muse-maintain/src/interface_admission.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–55; current module/import/attribute shell; declaration public_socket_directory; declaration validate_public_socket; declaration validate_interface_directory | [D11](../../slices/release-and-installation.md#d11-host-installation-and-payload-application) | retained | Imports and module declarations wire cmd/soda-muse-maintain/src/interface_admission.rs into its current native target.; 4 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-0f023d7a5011"></a>

## [cmd/soda-muse-maintain/src/interface_tests.rs](../../../../../cmd/soda-muse-maintain/src/interface_tests.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–56; current module/import/attribute shell; declaration public_interface_admission | [D11](../../slices/release-and-installation.md#d11-host-installation-and-payload-application) | retained | Imports and module declarations wire cmd/soda-muse-maintain/src/interface_tests.rs into its current native target.; public_interface_admission: implement the current host installation and payload application duty in interface_tests.rs. — current source cmd/soda-muse-maintain/src/interface_tests.rs; Cargo target and callers; current source cmd/soda-muse-maintain/src/interface_tests.rs; lines 6-56; module/caller wiring inspected |

<a id="coverage-07e542a2ef71"></a>

## [cmd/soda-muse-maintain/src/main.rs](../../../../../cmd/soda-muse-maintain/src/main.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–5, 7–29; current module/import/attribute shell; declaration archive; declaration command; declaration config_validation; declaration config_wire; declaration filesystem; declaration interface; declaration interface_admission; declaration network; declaration options; declaration project; declaration release; declaration release_validation; declaration release_wire; declaration sha256; declaration stage; declaration test_support | [D11](../../slices/release-and-installation.md#d11-host-installation-and-payload-application) | retained | Imports and module declarations wire cmd/soda-muse-maintain/src/main.rs into its current native target.; 17 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |
| 6; declaration config | [H04](../../slices/shared-supporting-slices.md#h04-configuration-and-filesystem-primitives) | retained | config: implement the current configuration and safe filesystem primitives duty in main.rs. — current source cmd/soda-muse-maintain/src/main.rs; lines 6-6; module/caller wiring inspected |
| 30–32; declaration MUSE_VERSION; declaration RELEASE_PATH | [P10](../../slices/projects.md#p10-shared-tools-and-packages) | retained | MUSE_VERSION: implement the current Project shared tools duty in main.rs.; RELEASE_PATH: implement the current Project shared tools duty in main.rs. — current source cmd/soda-muse-maintain/src/main.rs; lines 30-30; module/caller wiring inspected; current source cmd/soda-muse-maintain/src/main.rs; lines 31-32; module/caller wiring inspected |
| 33–74; declaration main; declaration run; declaration is_lower_hex; declaration maintain | [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) | retained | Compose the current main.rs command/module and its native consumers.; 4 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-db7ff0f1159b"></a>

## [cmd/soda-muse-maintain/src/network.rs](../../../../../cmd/soda-muse-maintain/src/network.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–35; current module/import/attribute shell; declaration parse_prefix | [D11](../../slices/release-and-installation.md#d11-host-installation-and-payload-application) | retained | Imports and module declarations wire cmd/soda-muse-maintain/src/network.rs into its current native target.; parse_prefix: implement the current host installation and payload application duty in network.rs. — current source cmd/soda-muse-maintain/src/network.rs; Cargo target and callers; current source cmd/soda-muse-maintain/src/network.rs; lines 5-35; module/caller wiring inspected |

<a id="coverage-f65ed9ad49cd"></a>

## [cmd/soda-muse-maintain/src/network_tests.rs](../../../../../cmd/soda-muse-maintain/src/network_tests.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–37; current module/import/attribute shell; declaration prefix_grammar_family_and_bounds | [D11](../../slices/release-and-installation.md#d11-host-installation-and-payload-application) | retained | Imports and module declarations wire cmd/soda-muse-maintain/src/network_tests.rs into its current native target.; prefix_grammar_family_and_bounds: implement the current host installation and payload application duty in network_tests.rs. — current source cmd/soda-muse-maintain/src/network_tests.rs; Cargo target and callers; current source cmd/soda-muse-maintain/src/network_tests.rs; lines 3-37; module/caller wiring inspected |

<a id="coverage-e9a367d5dec7"></a>

## [cmd/soda-muse-maintain/src/options.rs](../../../../../cmd/soda-muse-maintain/src/options.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–131; declaration Options; fields config, project, tools, bind_only; declaration default_tools; declaration parse; declaration print_usage; declaration usage_text; declaration flag_error; declaration parse_bool_flag; declaration valid_project_id; declaration options_tests | [D11](../../slices/release-and-installation.md#d11-host-installation-and-payload-application) | retained | Options: implement the current host installation and payload application duty in options.rs.; 9 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-03d910657770"></a>

## [cmd/soda-muse-maintain/src/options_tests.rs](../../../../../cmd/soda-muse-maintain/src/options_tests.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–75; current module/import/attribute shell; declaration args; declaration flag_parsing_vectors | [D11](../../slices/release-and-installation.md#d11-host-installation-and-payload-application) | retained | Imports and module declarations wire cmd/soda-muse-maintain/src/options_tests.rs into its current native target.; args: implement the current host installation and payload application duty in options_tests.rs.; flag_parsing_vectors: implement the current host installation and payload application duty in options_tests.rs. — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-a0ddbc725cb6"></a>

## [cmd/soda-muse-maintain/src/project.rs](../../../../../cmd/soda-muse-maintain/src/project.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–166; current module/import/attribute shell; declaration INSPECT_FORMAT; declaration Observation; fields id, project, owner, pid, running; declaration inspect_project; declaration validate_observation; declaration decode_observation; declaration Field; declaration Pairs; declaration deserialize; declaration PairsVisitor; declaration Value; declaration expecting; declaration visit_map; declaration observation_slot; declaration KEYS; declaration wait_project; declaration confirm_project; declaration project_tests | [D11](../../slices/release-and-installation.md#d11-host-installation-and-payload-application) | retained | Imports and module declarations wire cmd/soda-muse-maintain/src/project.rs into its current native target.; 18 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-de9d24148010"></a>

## [cmd/soda-muse-maintain/src/project_tests.rs](../../../../../cmd/soda-muse-maintain/src/project_tests.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–100; current module/import/attribute shell; declaration observation_decode_and_validate | [D11](../../slices/release-and-installation.md#d11-host-installation-and-payload-application) | retained | Imports and module declarations wire cmd/soda-muse-maintain/src/project_tests.rs into its current native target.; observation_decode_and_validate: implement the current host installation and payload application duty in project_tests.rs. — current source cmd/soda-muse-maintain/src/project_tests.rs; Cargo target and callers; current source cmd/soda-muse-maintain/src/project_tests.rs; lines 5-100; module/caller wiring inspected |

<a id="coverage-b2679ef268d8"></a>

## [cmd/soda-muse-maintain/src/release.rs](../../../../../cmd/soda-muse-maintain/src/release.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–139; current module/import/attribute shell; declaration apply_release_images; declaration ReleasePayload; fields architecture, images; declaration ReleaseImage; fields reference, config, manifest, archive_sha256; declaration image_config; declaration load_release_payload; declaration Guard; declaration drop; declaration release_tests | [D11](../../slices/release-and-installation.md#d11-host-installation-and-payload-application) | retained | Imports and module declarations wire cmd/soda-muse-maintain/src/release.rs into its current native target.; 9 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-6d111180bd63"></a>

## [cmd/soda-muse-maintain/src/release_tests.rs](../../../../../cmd/soda-muse-maintain/src/release_tests.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–189; current module/import/attribute shell; declaration valid_release_json; declaration release_payload_accept_and_reject; declaration release_images_apply_and_conflict | [D11](../../slices/release-and-installation.md#d11-host-installation-and-payload-application) | retained | Imports and module declarations wire cmd/soda-muse-maintain/src/release_tests.rs into its current native target.; 4 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-7d7127cb58ef"></a>

## [cmd/soda-muse-maintain/src/release_validation.rs](../../../../../cmd/soda-muse-maintain/src/release_validation.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–116; current module/import/attribute shell; declaration validate_release_payload; declaration NAMES; declaration is_hex_string; declaration is_digest; declaration valid_coreos_version; declaration valid_repository_prefix | [D11](../../slices/release-and-installation.md#d11-host-installation-and-payload-application) | retained | Imports and module declarations wire cmd/soda-muse-maintain/src/release_validation.rs into its current native target.; 7 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-87732be3f5b9"></a>

## [cmd/soda-muse-maintain/src/release_wire.rs](../../../../../cmd/soda-muse-maintain/src/release_wire.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–238; current module/import/attribute shell; declaration decode_release_payload; declaration PayloadDto; fields format, id, revision, architecture, coreos, base, repository_prefix, schema, presentation, host_packages, images, upgrade_from; declaration deserialize; declaration PayloadVisitor; declaration Value; declaration expecting; declaration visit_map; declaration Nullable; fields T; declaration NullableVisitor; fields T; declaration visit_none; declaration visit_unit; declaration visit_some; declaration Images; declaration ImagesVisitor; declaration ImageDto; declaration ImageVisitor; declaration PAYLOAD_FIELDS; declaration IMAGE_FIELDS; declaration slot; declaration payload_slot; declaration image_slot | [D11](../../slices/release-and-installation.md#d11-host-installation-and-payload-application) | retained | Imports and module declarations wire cmd/soda-muse-maintain/src/release_wire.rs into its current native target.; 33 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-91f19be11588"></a>

## [cmd/soda-muse-maintain/src/sha256.rs](../../../../../cmd/soda-muse-maintain/src/sha256.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–13; current module/import/attribute shell; declaration hex_encode; declaration HEX | [D11](../../slices/release-and-installation.md#d11-host-installation-and-payload-application) | retained | Imports and module declarations wire cmd/soda-muse-maintain/src/sha256.rs into its current native target.; hex_encode: implement the current host installation and payload application duty in sha256.rs.; HEX: implement the current host installation and payload application duty in sha256.rs. — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-5bf33bf3b0bd"></a>

## [cmd/soda-muse-maintain/src/stage.rs](../../../../../cmd/soda-muse-maintain/src/stage.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–87; current module/import/attribute shell; declaration BUS_SCRIPT; declaration DESTINATIONS; declaration INSTALL_SCRIPT; declaration ensure_system_bus; declaration stage_tools | [D11](../../slices/release-and-installation.md#d11-host-installation-and-payload-application) | retained | Imports and module declarations wire cmd/soda-muse-maintain/src/stage.rs into its current native target.; 6 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-f9b2a30636f3"></a>

## [cmd/soda-muse-maintain/src/test_support.rs](../../../../../cmd/soda-muse-maintain/src/test_support.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–100; current module/import/attribute shell; declaration TEST_SEQ; declaration TestDir; declaration make; declaration path; declaration dir_str; declaration drop; declaration is_root; declaration PROJECT; declaration hex_string; declaration synthetic_feeds; declaration emit_synthetic; declaration run_install_script | [D11](../../slices/release-and-installation.md#d11-host-installation-and-payload-application) | retained | Imports and module declarations wire cmd/soda-muse-maintain/src/test_support.rs into its current native target.; 13 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

## Removed predecessor navigation

These anchors identify earlier mapped source. Their paths no longer define this current map;
the current inventory and normalized predecessor ledger preserve replacement/retirement evidence.
No historical source body is counted as current coverage.

<a id="coverage-4d99badbce97"></a>
<a id="rustsoda-muse-maintainsrcmainrs-1"></a>

Former source `rust/soda-muse-maintain/src/main.rs`; consult its pinned earlier Git source and the current coverage disposition.

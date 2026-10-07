# Soda project factory roles

[Responsibility map index](README.md) · [Coverage scope](../README.md).

Current responsibility accounting at `519b76bd` (2026-10-07).
One slice owns each named duty; disjoint complete symbols may share an owner.
Compound fields/clauses may share a physical line with distinct selectors.
Disposition concerns the duty, not source validity or installed qualification.

<a id="coverage-f0029f61a5aa"></a>
<a id="coverage-2059e018b3d9"></a>

## [cmd/soda-project-terminal/src/factory_roles/accounts.rs](../../../../../cmd/soda-project-terminal/src/factory_roles/accounts.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–13; current module/import/attribute shell | [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) | retained | Imports and module declarations wire cmd/soda-project-terminal/src/factory_roles/accounts.rs into its current native target. — current source cmd/soda-project-terminal/src/factory_roles/accounts.rs; Cargo target and callers |
| 14–304; declaration NOLOGIN; declaration SYSTEM_KEYDIR; declaration Account; fields name, dir, uid, gid, shell; declaration useradd_argv; declaration run_check; declaration bytes_to_string; declaration system_account; declaration member_groups; declaration primary_group; declaration test_account; declaration role_record; declaration check_home; declaration ensure_checkouts; declaration ensure_creds; declaration key_path; declaration refuse_keyfile; declaration ensure_role_production; declaration ensure_role_test; declaration ensure_role; declaration tests | [P06](../../slices/projects.md#p06-factory-role-accounts) | retained | NOLOGIN: implement the current factory role accounts duty in accounts.rs.; 20 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-fba4211e4aab"></a>

## [cmd/soda-project-terminal/src/factory_roles/accounts_tests.rs](../../../../../cmd/soda-project-terminal/src/factory_roles/accounts_tests.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–6; current module/import/attribute shell | [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) | retained | Imports and module declarations wire cmd/soda-project-terminal/src/factory_roles/accounts_tests.rs into its current native target. — current source cmd/soda-project-terminal/src/factory_roles/accounts_tests.rs; Cargo target and callers |
| 7–112; declaration mode; declaration useradd_recipe_is_exact; declaration ensure_provisions_locked_roles; declaration ensure_refuses_interactive_accounts; declaration ensure_refuses_external_keys_and_links; declaration ensure_refuses_bad_roles | [P06](../../slices/projects.md#p06-factory-role-accounts) | retained | mode: exercise fixed non-login role account lookup, creation, group membership, and home custody.; 6 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-4a90aea52f4d"></a>

## [cmd/soda-project-terminal/src/factory_roles/b64.rs](../../../../../cmd/soda-project-terminal/src/factory_roles/b64.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–43; current module/import/attribute shell; declaration python_validate; declaration decode; declaration tests; declaration vectors | [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) | retained | Imports and module declarations wire cmd/soda-project-terminal/src/factory_roles/b64.rs into its current native target.; 5 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-12b41c06de77"></a>

## [cmd/soda-project-terminal/src/factory_roles/emit.rs](../../../../../cmd/soda-project-terminal/src/factory_roles/emit.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–190; current module/import/attribute shell; declaration obj; declaration str_value; declaration dumps_default; declaration json_equal; declaration unique_keys; declaration is_int_literal; declaration norm_int; declaration numbers_equal; declaration tests; declaration parse; declaration emit_matches_reference; declaration emit_state_files; declaration equality_matches_reference | [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) | retained | Imports and module declarations wire cmd/soda-project-terminal/src/factory_roles/emit.rs into its current native target.; 14 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-3bf014d79d52"></a>

## [cmd/soda-project-terminal/src/factory_roles/error.rs](../../../../../cmd/soda-project-terminal/src/factory_roles/error.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–56; current module/import/attribute shell; declaration FIXED_MESSAGE; declaration Error; declaration fail; declaration io; declaration io_msg; declaration classify; declaration from | [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) | retained | Imports and module declarations wire cmd/soda-project-terminal/src/factory_roles/error.rs into its current native target.; 9 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-6be2561b7359"></a>

## [cmd/soda-project-terminal/src/factory_roles/execution.rs](../../../../../cmd/soda-project-terminal/src/factory_roles/execution.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–24, 95–98, 120–123, 198–204, 212–555, 881–883; current module/import/attribute shell; declaration KILL_LOG; declaration TERM_GRACE_MS; declaration KILL_GRACE_MS; declaration POLL_SLICE_MS; declaration log_failure_cleanup; declaration run_as_role; declaration run_preparation; declaration spawn_detached; declaration grandchild_body; declaration do_start; declaration tests | [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) | retained | Imports and module declarations wire cmd/soda-project-terminal/src/factory_roles/execution.rs into its current native target.; 12 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |
| 25–94, 205–211; declaration child_environ; declaration parent_of; declaration exit_code; declaration wait_child; declaration to_cstring | [P06](../../slices/projects.md#p06-factory-role-accounts) | retained | Run the fixed Project role process with bounded environment, output, and status handling (child_environ).; 5 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |
| 99–119, 124–197, 556–880; declaration signal_child; declaration ReapOutcome; declaration reap_once; declaration reap_until; declaration terminate_child; declaration proc_state_group; declaration group_alive; declaration probe_group_quiet; declaration preparation_blocks_release; declaration leader_owned_by; declaration killpg; declaration signal_group; declaration retire_group; declaration do_stop; declaration any_running; declaration hold_revision; declaration do_hold; declaration do_release | [P12](../../slices/projects.md#p12-maintenance-holds) | retained | signal_child: observe, stop, or reconcile a preparation process group and its hold/release state.; 18 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-a58d3a28eb02"></a>

## [cmd/soda-project-terminal/src/factory_roles/execution_tests.rs](../../../../../cmd/soda-project-terminal/src/factory_roles/execution_tests.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–19, 44–63, 293–348, 460–483, 737–752; current module/import/attribute shell; declaration parent_matrix; declaration write_proot_stat; declaration dead_pgid; declaration metadata_hardening_refuses_links_and_aliases; declaration own_child_pids; declaration proc_comm; declaration log_failure_cleanup_reports_unconfirmed_when_child_gone; declaration proc_startup | [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) | retained | Imports and module declarations wire cmd/soda-project-terminal/src/factory_roles/execution_tests.rs into its current native target.; 9 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |
| 20–43; declaration child_environment_is_fixed_and_ordered | [P06](../../slices/projects.md#p06-factory-role-accounts) | retained | child_environment_is_fixed_and_ordered: exercise the current preparation process supervisor, fixed-role launch, or hold/stop lifecycle case. — current source cmd/soda-project-terminal/src/factory_roles/execution_tests.rs; lines 20-43; module/caller wiring inspected |
| 64–292, 349–459, 484–736, 753–786; declaration proc_group_reads_pgrp_not_session; declaration signal_group_confirms_dead_groups; declaration hold_request; declaration release_request; declaration hold_bars_preparation_and_releases_by_revision; declaration stop_bars_unknown_identity_and_retires_known; declaration inspect_reports_phases_and_hold; declaration log_io_failure_reaps_child_and_reports_unconfirmed; declaration log_io_failure_terminates_silent_child_bounded; declaration terminate_child_never_signals_after_ownership_loss; declaration reap_until_distinguishes_reaped_alive_and_gone; declaration uncertain_stop_bars_hold_release; declaration release_probe_maps_native_states; declaration release_after_quiesced_preparation_succeeds; declaration stop_launch_error_with_live_foreign_group_is_uncertain | [P12](../../slices/projects.md#p12-maintenance-holds) | retained | proc_group_reads_pgrp_not_session: exercise the current preparation process supervisor, fixed-role launch, or hold/stop lifecycle case.; 15 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-bda10213e476"></a>

## [cmd/soda-project-terminal/src/factory_roles/inputs.rs](../../../../../cmd/soda-project-terminal/src/factory_roles/inputs.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–19, 23–37, 131–189, 218–225, 314–464, 508–510; current module/import/attribute shell; declaration MAX_SOURCE_BUNDLE; declaration ReqFields; fields id, role, setup_digest, source_commit, credential; declaration git_clone_argv; declaration git_catfile_argv; declaration verify_bundle; declaration ApprovedInputs; fields fields, files, bundle; declaration checkout_cname; declaration mkdirat_exclusive; declaration openat_dir_no_follow; declaration CHECKOUT_STAGING; declaration publish_checkout; declaration write_snapshot; declaration tests | [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) | retained | Imports and module declarations wire cmd/soda-project-terminal/src/factory_roles/inputs.rs into its current native target.; 14 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |
| 20–22, 38–66, 94–130, 226–286; declaration MAX_APPROVED_FILES; declaration MAX_APPROVED_FILE_SIZE; declaration MAX_APPROVED_TOTAL; declaration from_stored; declaration decode_files; declaration approve_inputs | [P08](../../slices/projects.md#p08-preparation-requirements-acceptance) | retained | Bound the accepted setup/check file count.; 6 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |
| 67–79; declaration do_ensure | [P06](../../slices/projects.md#p06-factory-role-accounts) | retained | Ensure the fixed non-login factory role accounts exist. — current source cmd/soda-project-terminal/src/factory_roles/inputs.rs; lines 67-79; module/caller wiring inspected |
| 80–93; declaration collapse | [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) | retained | Apply last-wins duplicate-key handling to decoded JSON objects. — current source cmd/soda-project-terminal/src/factory_roles/inputs.rs; lines 80-93; module/caller wiring inspected |
| 190–217, 465–507; declaration check_credential_file; declaration do_approve | [P09](../../slices/projects.md#p09-privileged-preparation-approval) | retained | Bind an approved preparation to its role-private credential file.; Apply the privileged preparation approval and preserve its checkout custody boundary. — current source cmd/soda-project-terminal/src/factory_roles/inputs.rs; lines 190-217; module/caller wiring inspected; current source cmd/soda-project-terminal/src/factory_roles/inputs.rs; lines 465-507; module/caller wiring inspected |
| 287–313; declaration open_dir_no_follow; declaration fchmod; declaration fchown | [H04](../../slices/shared-supporting-slices.md#h04-configuration-and-filesystem-primitives) | retained | Use descriptor-bound filesystem primitives for checkout custody (open_dir_no_follow).; Use descriptor-bound filesystem primitives for checkout custody (fchmod).; Use descriptor-bound filesystem primitives for checkout custody (fchown). — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-62ae59369601"></a>

## [cmd/soda-project-terminal/src/factory_roles/inputs_tests.rs](../../../../../cmd/soda-project-terminal/src/factory_roles/inputs_tests.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–9, 317–765, 815–844; current module/import/attribute shell; declaration preexisting_checkout; declaration approve_refusal_preserves_preexisting_checkout; declaration approve_bundle_failure_preserves_preexisting_checkout; declaration claimed_checkout; declaration approve_refusal_preserves_checkout_planted_during_verification; declaration approve_refusal_preserves_dangling_checkout_symlink; declaration approve_refusal_preserves_checkout_path_file; declaration approve_credential_failure_removes_owned_checkout; declaration approve_refuses_when_checkout_parent_unwritable; declaration approve_failure_preserves_checkout_swapped_during_cleanup; declaration checkout_identity_primitives_refuse_and_preserve; declaration checkout_publication_refuses_without_replacement; declaration checkout_publication_across_filesystems_fails_explicitly; declaration failed_publication_leaves_no_final_receipt | [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) | retained | Imports and module declarations wire cmd/soda-project-terminal/src/factory_roles/inputs_tests.rs into its current native target.; 15 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |
| 10–316; declaration mode; declaration git_recipes_are_exact; declaration ensure_reports_fixed_roles; declaration approve_writes_protected_snapshot_and_verifies_bundle; declaration approve_rejects_untrusted_inputs_before_effects; declaration Mutate; declaration approve_enforces_all_bounds; declaration approve_failure_removes_everything | [P08](../../slices/projects.md#p08-preparation-requirements-acceptance) | retained | mode: exercise accepted preparation field bounds, source/checkout custody, or privileged credential binding as named.; 8 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |
| 766–814; declaration approve_binds_role_private_credentials | [P09](../../slices/projects.md#p09-privileged-preparation-approval) | retained | approve_binds_role_private_credentials: exercise accepted preparation field bounds, source/checkout custody, or privileged credential binding as named. — current source cmd/soda-project-terminal/src/factory_roles/inputs_tests.rs; lines 766-814; module/caller wiring inspected |

<a id="coverage-3b6f93dd9410"></a>
<a id="coverage-d3ef4dadb095"></a>

## [cmd/soda-project-terminal/src/factory_roles/layout.rs](../../../../../cmd/soda-project-terminal/src/factory_roles/layout.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–15, 151–158; current module/import/attribute shell; declaration prep_dir | [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) | retained | Imports and module declarations wire cmd/soda-project-terminal/src/factory_roles/layout.rs into its current native target.; prep_dir: implement the current Project preparation and checkout allocation duty in layout.rs. — current source cmd/soda-project-terminal/src/factory_roles/layout.rs; Cargo target and callers; current source cmd/soda-project-terminal/src/factory_roles/layout.rs; lines 151-158; module/caller wiring inspected |
| 16–150; declaration lexists; declaration file_name; declaration mkdir_p; declaration chown; declaration chmod; declaration open_ro; declaration read_json; declaration write_new; declaration write_new_file; declaration owned_dir; declaration ensure_layout | [H04](../../slices/shared-supporting-slices.md#h04-configuration-and-filesystem-primitives) | retained | lexists: implement the current configuration and safe filesystem primitives duty in layout.rs.; 11 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |
| 159–204; declaration Hold; fields active, revision; declaration hold_json; declaration hold_state; declaration refuse_barred | [P12](../../slices/projects.md#p12-maintenance-holds) | retained | Hold: implement the current Project maintenance holds duty in layout.rs.; 4 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-bbb0cfc6ed28"></a>

## [cmd/soda-project-terminal/src/factory_roles/mod.rs](../../../../../cmd/soda-project-terminal/src/factory_roles/mod.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–12, 15–19, 31–51, 53–105, 122–140, 144–145, 153–220; current module/import/attribute shell; declaration b64; declaration emit; declaration error; declaration fsx; declaration execution; declaration proc; declaration records; declaration sha; declaration testutil; declaration validate; declaration SETUP_ENTRY; declaration CHECK_ENTRY; declaration SHELL; declaration GIT; declaration LOG_CAP; declaration FACTORY_DEFAULT; declaration TestCtx; fields git; declaration Ctx; fields factory, preparations, credentials, hold, lock, test; declaration production; declaration test_at; declaration at; declaration from_env; declaration git; declaration read_stdin_capped; declaration main_inner; declaration run; dispatch_op signature and match shell; start operation arm; inspect operation arm | [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) | retained | Imports and module declarations wire cmd/soda-project-terminal/src/factory_roles/mod.rs into its current native target.; 30 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |
| 13–14, 106–121, 141; declaration account; declaration priv_uid; declaration priv_gid; ensure operation arm | [P06](../../slices/projects.md#p06-factory-role-accounts) | retained | Wire the fixed role account module; account creation and custody belong to P06.; 4 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |
| 20–30; declaration ops_approve; declaration ops_inspect; declaration ops_record | [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) | retained | Wire the shared helper dispatcher to ensure/approval input operations; operation handlers carry P06/P08/P09 responsibility.; Wire inspection, stop, and maintenance hold handlers; operation handlers carry P07/P12 responsibility.; Wire tool-evidence recording and preparation launch handlers; operation handlers carry P08/P07 responsibility. — Current named units/source consumers; retained normalized source evidence records each selector |
| 52, 143; declaration INPUT_CAP; record operation arm | [P08](../../slices/projects.md#p08-preparation-requirements-acceptance) | retained | INPUT_CAP: implement the current Project preparation requirement acceptance duty in mod.rs.; Route bounded tool and verification evidence for a preparation. — current source cmd/soda-project-terminal/src/factory_roles/mod.rs; lines 52-52; module/caller wiring inspected; current source cmd/soda-project-terminal/src/factory_roles/mod.rs: dispatch_op operation arms and submodule targets |
| 142; approve operation arm | [P09](../../slices/projects.md#p09-privileged-preparation-approval) | retained | Route the privileged approval request that applies accepted preparation inputs. — current source cmd/soda-project-terminal/src/factory_roles/mod.rs: dispatch_op operation arms and submodule targets |
| 146–148; stop operation arm; hold/release operation arms | [P12](../../slices/projects.md#p12-maintenance-holds) | retained | Route preparation stop and process retirement.; Route maintenance hold creation and revision-bound release. — current source cmd/soda-project-terminal/src/factory_roles/mod.rs: dispatch_op operation arms and submodule targets |
| 149–152; unknown operation arm and match close | [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) | retained | Return the fixed unsupported-operation wire error and close dispatch. — current source cmd/soda-project-terminal/src/factory_roles/mod.rs: dispatch_op operation arms and submodule targets |

<a id="coverage-534c44049c23"></a>

## [cmd/soda-project-terminal/src/factory_roles/records.rs](../../../../../cmd/soda-project-terminal/src/factory_roles/records.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–21, 124–345; current module/import/attribute shell; declaration MAX_TOOLS; declaration MAX_TOOL_TEXT; declaration char_len; declaration Prestate; fields fields, tools; declaration start_prestate; declaration read_log; declaration started_pgid; declaration started_pid; declaration exit_is_zero; declaration phase_of; declaration do_inspect; declaration tests | [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) | retained | Imports and module declarations wire cmd/soda-project-terminal/src/factory_roles/records.rs into its current native target.; 13 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |
| 22–123; declaration check_tool_entry; declaration check_verified; declaration do_record | [P08](../../slices/projects.md#p08-preparation-requirements-acceptance) | retained | Validate one bounded tool/version evidence record.; Validate the preparation launcher evidence fields.; Persist immutable tool and verification evidence for the accepted preparation. — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-dc1e8fd25ecd"></a>

## [cmd/soda-project-terminal/src/factory_roles/records_tests.rs](../../../../../cmd/soda-project-terminal/src/factory_roles/records_tests.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–161; current module/import/attribute shell; declaration tool; declaration tool_entry_matrix; declaration verified_matrix; declaration record_reports_waiting_and_locks_evidence; declaration record_after_finish_is_refused; declaration prestate_bars_launch_blockers; declaration prestate_unknown_identity | [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) | retained | Imports and module declarations wire cmd/soda-project-terminal/src/factory_roles/records_tests.rs into its current native target.; 8 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-05c42acbf0c2"></a>

## [cmd/soda-project-terminal/src/factory_roles/sha.rs](../../../../../cmd/soda-project-terminal/src/factory_roles/sha.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–4, 17–31; current module/import/attribute shell; declaration approved_digest; declaration tests | [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) | retained | Imports and module declarations wire cmd/soda-project-terminal/src/factory_roles/sha.rs into its current native target.; approved_digest: implement the current Project preparation and checkout allocation duty in sha.rs.; Collect embedded tests for sha.rs. — Current named units/source consumers; retained normalized source evidence records each selector |
| 5–16; declaration hex; declaration HEX | [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) | retained | hex: implement the current wire decoding, encoding, and representation conversion duty in sha.rs.; HEX: implement the current wire decoding, encoding, and representation conversion duty in sha.rs. — current source cmd/soda-project-terminal/src/factory_roles/sha.rs; lines 5-5; module/caller wiring inspected; current source cmd/soda-project-terminal/src/factory_roles/sha.rs; lines 6-16; module/caller wiring inspected |

<a id="coverage-82149808f59d"></a>

## [cmd/soda-project-terminal/src/factory_roles/sha_tests.rs](../../../../../cmd/soda-project-terminal/src/factory_roles/sha_tests.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–33; current module/import/attribute shell; declaration hex_of; declaration nist_vectors | [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) | retained | Imports and module declarations wire cmd/soda-project-terminal/src/factory_roles/sha_tests.rs into its current native target.; hex_of: implement the current Project preparation and checkout allocation duty in sha_tests.rs.; nist_vectors: implement the current Project preparation and checkout allocation duty in sha_tests.rs. — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-a445c358fe44"></a>

## [cmd/soda-project-terminal/src/factory_roles/tests.rs](../../../../../cmd/soda-project-terminal/src/factory_roles/tests.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–86, 132–201; current module/import/attribute shell; declaration NEXT; declaration Scratch; fields root, factory; declaration fresh; declaration git_script; declaration ctx; declaration record_text; declaration drop; declaration PID; declaration PID2; declaration COMMIT; declaration b64_encode; declaration ALPHA; declaration fixture_files; declaration record_value; declaration op_value; declaration assert_fail; declaration b64_round_trip | [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) | retained | Imports and module declarations wire cmd/soda-project-terminal/src/factory_roles/tests.rs into its current native target.; 18 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |
| 87–131; declaration approve_value; declaration approve_default | [P08](../../slices/projects.md#p08-preparation-requirements-acceptance) | retained | approve_value: implement the current Project preparation requirement acceptance duty in tests.rs.; approve_default: implement the current Project preparation requirement acceptance duty in tests.rs. — current source cmd/soda-project-terminal/src/factory_roles/tests.rs; lines 87-127; module/caller wiring inspected; current source cmd/soda-project-terminal/src/factory_roles/tests.rs; lines 128-131; module/caller wiring inspected |

<a id="coverage-1e734e642cd1"></a>

## [cmd/soda-project-terminal/src/factory_roles/validate.rs](../../../../../cmd/soda-project-terminal/src/factory_roles/validate.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–134; current module/import/attribute shell; declaration ROLES; declaration is_lower_hex; declaration is_id; declaration is_role; declaration is_name; declaration is_digest; declaration is_commit; declaration check_id; declaration check_role; declaration check_role_str; declaration check_name_str; declaration check_digest; declaration check_commit; declaration key_set; declaration as_object; declaration as_int_text; declaration as_i64; declaration tests | [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) | retained | Imports and module declarations wire cmd/soda-project-terminal/src/factory_roles/validate.rs into its current native target.; 19 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-0f99b3442164"></a>

## [cmd/soda-project-terminal/src/factory_roles/validate_tests.rs](../../../../../cmd/soda-project-terminal/src/factory_roles/validate_tests.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–21, 31–87; current module/import/attribute shell; declaration val; declaration id_matrix; declaration name_matrix; declaration digest_commit_matrix; declaration key_sets_collapse_duplicates; declaration int_text_matches_type_is_int | [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) | retained | Imports and module declarations wire cmd/soda-project-terminal/src/factory_roles/validate_tests.rs into its current native target.; 7 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |
| 22–30; declaration role_matrix | [P06](../../slices/projects.md#p06-factory-role-accounts) | retained | role_matrix: implement the current factory role accounts duty in validate_tests.rs. — current source cmd/soda-project-terminal/src/factory_roles/validate_tests.rs; lines 22-30; module/caller wiring inspected |

<a id="coverage-138af891676d"></a>
<a id="coverage-ca824fa18e03"></a>

## [cmd/soda-project-terminal/tests/factory_roles_oracle.rs](../../../../../cmd/soda-project-terminal/tests/factory_roles_oracle.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–210, 225–399, 420–468; current module/import/attribute shell; declaration NEXT; declaration PID; declaration PID2; declaration COMMIT; declaration FIXED_STDERR; declaration binary; declaration Scratch; fields root, factory, git, record; declaration fresh; declaration run; declaration ok; declaration refused; declaration drop; declaration b64_encode; declaration ALPHA; declaration canonical_digest; declaration approve_request; declaration record_request; declaration fixture_request; declaration field; declaration wait_for; declaration oracle_approve_record_inspect_exact_bytes; declaration oracle_start_runs_detached_to_ready; declaration oracle_setup_failure_skips_check; declaration oracle_output_truncates_at_cap; declaration oracle_running_stop_lifecycle; declaration Guard; declaration oracle_dead_supervisor_is_interrupted; declaration oracle_refusal_contract | [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) | retained | Imports and module declarations wire cmd/soda-project-terminal/tests/factory_roles_oracle.rs into its current native target.; 30 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |
| 211–224; declaration oracle_ensure_exact_bytes | [P06](../../slices/projects.md#p06-factory-role-accounts) | retained | oracle_ensure_exact_bytes: source assertion of factory role accounts. — current source cmd/soda-project-terminal/tests/factory_roles_oracle.rs; lines 211-224; module/caller wiring inspected; exact byte-identical source map rust/soda-project-factory-roles/tests/oracle.rs; exact byte-identical prior map docs/development/ideal-filetree-plan/coverage/maps/soda-project-factory-roles.md at the same current line |
| 400–419; declaration oracle_stop_hold_release_bytes | [P12](../../slices/projects.md#p12-maintenance-holds) | retained | oracle_stop_hold_release_bytes: source assertion of maintenance holds. — current source cmd/soda-project-terminal/tests/factory_roles_oracle.rs; lines 400-419; module/caller wiring inspected; exact byte-identical source map rust/soda-project-factory-roles/tests/oracle.rs; exact byte-identical prior map docs/development/ideal-filetree-plan/coverage/maps/soda-project-factory-roles.md at the same current line |

## Removed predecessor navigation

These anchors identify earlier mapped source. Their paths no longer define this current map;
the current inventory and normalized predecessor ledger preserve replacement/retirement evidence.
No historical source body is counted as current coverage.

<a id="coverage-8d1e30c52682"></a>

Former source `rust/soda-project-factory-roles/src/main.rs`; consult its pinned earlier Git source and the current coverage disposition.

<a id="coverage-bf1ad14564e0"></a>

Former source `rust/soda-project-factory-roles/src/ops_approve.rs`; consult its pinned earlier Git source and the current coverage disposition.

<a id="coverage-bbd6009d9c3a"></a>

Former source `rust/soda-project-factory-roles/src/ops_inspect.rs`; consult its pinned earlier Git source and the current coverage disposition.

<a id="coverage-cbcfc57d4b50"></a>

Former source `rust/soda-project-factory-roles/src/proc.rs`; consult its pinned earlier Git source and the current coverage disposition.

<a id="coverage-f843e3e6b032"></a>

Former source `rust/soda-project-factory-roles/src/sha.rs`; consult its pinned earlier Git source and the current coverage disposition.

<a id="coverage-28bdb9bbdb55"></a>

Former source `rust/soda-project-factory-roles/src/validate.rs`; consult its pinned earlier Git source and the current coverage disposition.

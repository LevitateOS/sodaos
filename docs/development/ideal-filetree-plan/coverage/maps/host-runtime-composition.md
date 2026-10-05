# Host runtime composition

[Responsibility map index](README.md) · [Coverage snapshot and limits](../README.md).
Page grouping is navigation; the slice IDs retain their individual review ownership.

<a id="coverage-46b7e6567367"></a>

## [rust/soda-host/src/dbackend.rs](../../../../../rust/soda-host/src/dbackend.rs)

Source assertions/fixtures were inspected for mapping only; no test execution, runtime or installed proof claimed. Large-file ledger retains committed declaration/member spans rather than grouping methods into one whole-file unit; responsibility follows the verified concern/branch mapping, not declaration count.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 1–27 | Privileged daemon composition and bounded operation errors; declarations/fields: `HAS_SCAFFOLDS`, `SCAFFOLD_PREFIX`, `internal`, `decode_native`, `decode_empty`, `map_factory_err`, `native_deadline`, `tailnet_deadline` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 28–29 | Privileged daemon composition and bounded operation errors; declaration/member HAS_SCAFFOLDS; declarations/fields: `HAS_SCAFFOLDS` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 30–35 | Privileged daemon composition and bounded operation errors; declaration/member SCAFFOLD_PREFIX; declarations/fields: `SCAFFOLD_PREFIX` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 36–45 | Privileged daemon composition and bounded operation errors; declaration/member internal; declarations/fields: `internal` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 46–52 | Privileged daemon composition and bounded operation errors; declaration/member decode_native; declarations/fields: `decode_native` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 53–63 | Privileged daemon composition and bounded operation errors; declaration/member decode_empty; declarations/fields: `decode_empty` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 64–77 | Privileged daemon composition and bounded operation errors; declaration/member map_factory_err; declarations/fields: `map_factory_err` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 78–81 | Privileged daemon composition and bounded operation errors; declaration/member native_deadline; declarations/fields: `native_deadline` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 82–89 | Privileged daemon composition and bounded operation errors; declaration/member tailnet_deadline; declarations/fields: `tailnet_deadline` |
| [H04](../../slices/shared-supporting-slices.md#h04-configuration-and-filesystem-primitives) / active | 90 | Native backend configuration; declarations/fields: `BackendConfig` |
| [H04](../../slices/shared-supporting-slices.md#h04-configuration-and-filesystem-primitives) / active | 91 | Native backend configuration; declaration/member BackendConfig.project; declarations/fields: `BackendConfig.project` |
| [H04](../../slices/shared-supporting-slices.md#h04-configuration-and-filesystem-primitives) / active | 92 | Native backend configuration; declaration/member BackendConfig.codex_harness; declarations/fields: `BackendConfig.codex_harness` |
| [H04](../../slices/shared-supporting-slices.md#h04-configuration-and-filesystem-primitives) / active | 93 | Native backend configuration; declaration/member BackendConfig.codex_harness_sha256; declarations/fields: `BackendConfig.codex_harness_sha256` |
| [H04](../../slices/shared-supporting-slices.md#h04-configuration-and-filesystem-primitives) / active | 94 | Native backend configuration; declaration/member BackendConfig.codex_harness_version; declarations/fields: `BackendConfig.codex_harness_version` |
| [H04](../../slices/shared-supporting-slices.md#h04-configuration-and-filesystem-primitives) / active | 95 | Native backend configuration; declaration/member BackendConfig.muse_harness; declarations/fields: `BackendConfig.muse_harness` |
| [H04](../../slices/shared-supporting-slices.md#h04-configuration-and-filesystem-primitives) / active | 96 | Native backend configuration; declaration/member BackendConfig.muse_harness_sha256; declarations/fields: `BackendConfig.muse_harness_sha256` |
| [H04](../../slices/shared-supporting-slices.md#h04-configuration-and-filesystem-primitives) / active | 97 | Native backend configuration; declaration/member BackendConfig.muse_harness_version; declarations/fields: `BackendConfig.muse_harness_version` |
| [H04](../../slices/shared-supporting-slices.md#h04-configuration-and-filesystem-primitives) / active | 98 | Native backend configuration; declaration/member BackendConfig.broker_socket; declarations/fields: `BackendConfig.broker_socket` |
| [H04](../../slices/shared-supporting-slices.md#h04-configuration-and-filesystem-primitives) / active | 99–101 | Native backend configuration; declaration/member BackendConfig.tailnet_image; declarations/fields: `BackendConfig.tailnet_image` |
| [H04](../../slices/shared-supporting-slices.md#h04-configuration-and-filesystem-primitives) / active | 102–104 | Native backend configuration; declaration/member BackendConfig.muse; declarations/fields: `BackendConfig.muse` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 105 | Muse execution config and provider seam; declarations/fields: `MuseConfig` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 106 | Muse execution config and provider seam; declaration/member MuseConfig.version; declarations/fields: `MuseConfig.version` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 107–111 | Muse execution config and provider seam; declaration/member MuseConfig.sha256; declarations/fields: `MuseConfig.sha256` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 112 | Muse execution config and provider seam; declaration/member TerminalSeam; declarations/fields: `TerminalSeam` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 113 | Muse execution config and provider seam; declaration/member TerminalSeam.harness; declarations/fields: `TerminalSeam.harness` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 114 | Muse execution config and provider seam; declaration/member TerminalSeam.harness_version; declarations/fields: `TerminalSeam.harness_version` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 115 | Muse execution config and provider seam; declaration/member TerminalSeam.harness_sha256; declarations/fields: `TerminalSeam.harness_sha256` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 116 | Muse execution config and provider seam; declaration/member TerminalSeam.muse_harness; declarations/fields: `TerminalSeam.muse_harness` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 117 | Muse execution config and provider seam; declaration/member TerminalSeam.muse_harness_version; declarations/fields: `TerminalSeam.muse_harness_version` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 118–123 | Muse execution config and provider seam; declaration/member TerminalSeam.muse_harness_sha256; declarations/fields: `TerminalSeam.muse_harness_sha256` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 124 | Muse execution config and provider seam; declaration/member BrokerSeam; declarations/fields: `BrokerSeam` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 125–128 | Muse execution config and provider seam; declaration/member BrokerSeam.broker; declarations/fields: `BrokerSeam.broker` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 129 | Muse execution config and provider seam; declaration/member HooksSeam; declarations/fields: `HooksSeam` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 130–134 | Muse execution config and provider seam; declaration/member HooksSeam.broker; declarations/fields: `HooksSeam.broker` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 135–141 | Muse execution config and provider seam; declaration/member new; declarations/fields: `new` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 142–146 | Muse execution config and provider seam; declaration/member Factory; declarations/fields: `Factory` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 147 | Native backend lifetime/composition; declarations/fields: `DaemonBackend` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 148 | Native backend lifetime/composition; declaration/member DaemonBackend.project; declarations/fields: `DaemonBackend.project` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 149 | Native backend lifetime/composition; declaration/member DaemonBackend.terminal; declarations/fields: `DaemonBackend.terminal` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 150 | Native backend lifetime/composition; declaration/member DaemonBackend.factory; declarations/fields: `DaemonBackend.factory` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 151 | Native backend lifetime/composition; declaration/member DaemonBackend.broker; declarations/fields: `DaemonBackend.broker` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 152 | Native backend lifetime/composition; declaration/member DaemonBackend.tailnet; declarations/fields: `DaemonBackend.tailnet` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 153–154 | Native backend lifetime/composition; declaration/member DaemonBackend.companion; declarations/fields: `DaemonBackend.companion` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 155 | Native backend lifetime/composition; declaration/member DaemonBackend.pops; declarations/fields: `DaemonBackend.pops` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 156 | Native backend lifetime/composition; declaration/member DaemonBackend.muse; declarations/fields: `DaemonBackend.muse` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 157 | Native backend lifetime/composition; declaration/member DaemonBackend.codex_harness; declarations/fields: `DaemonBackend.codex_harness` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 158 | Native backend lifetime/composition; declaration/member DaemonBackend.broker_socket; declarations/fields: `DaemonBackend.broker_socket` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 159 | Native backend lifetime/composition; declaration/member DaemonBackend.factory_state_dir; declarations/fields: `DaemonBackend.factory_state_dir` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 160–163 | Native backend lifetime/composition; declaration/member DaemonBackend.next_session; declarations/fields: `DaemonBackend.next_session` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 164–214 | Native backend lifetime/composition; declaration/member open; declarations/fields: `open` |
| [N06](../../slices/networking.md#n06-project-companion-lifecycle) / active | 215–218 | Companion service lifetime callbacks; declarations/fields: `companion_start` |
| [N06](../../slices/networking.md#n06-project-companion-lifecycle) / active | 219–222 | Companion service lifetime callbacks; declaration/member companion_wait; declarations/fields: `companion_wait` |
| [N06](../../slices/networking.md#n06-project-companion-lifecycle) / active | 223–226 | Companion service lifetime callbacks; declaration/member companion_stop; declarations/fields: `companion_stop` |
| [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) / active | 227–240 | Factory service accessor; declarations/fields: `factory` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 241–266 | Factory provider setup |
| [I04](../../slices/identity-brokering.md#i04-execution-admission-and-lease-fencing) / active | 267–286 | Broker lease conversion; declarations/fields: `cv_lease_to_texec` |
| [I05](../../slices/identity-brokering.md#i05-native-binding-and-private-delivery) / active | 287–302 | Native binding conversion; declarations/fields: `cv_binding_to_texec` |
| [I05](../../slices/identity-brokering.md#i05-native-binding-and-private-delivery) / active | 303–318 | Native binding conversion; declaration/member cv_binding_to_pfactory; declarations/fields: `cv_binding_to_pfactory` |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 319–335 | Native factory run conversion; declarations/fields: `cv_run_to_tcodex` |
| [I04](../../slices/identity-brokering.md#i04-execution-admission-and-lease-fencing) / active | 336–351 | Acquire value conversion; declarations/fields: `cv_acquire_to_texec` |
| [I04](../../slices/identity-brokering.md#i04-execution-admission-and-lease-fencing) / active | 352–370 | Lease response conversion; declarations/fields: `cv_lease_to_pfactory` |
| [S06](../../slices/spaces-and-terminals.md#s06-factory-activity-presentation) / active | 371–382 | Factory output conversion; declarations/fields: `cv_slice_to_pfactory` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 383 | Provider native harness/reserve/start adapter; declarations/fields: `harness_family`, `harness_version`, `harness_sha256`, `reserve`, `start` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 384–386 | Provider native harness/reserve/start adapter; declaration/member harness_family; declarations/fields: `harness_family` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 387–393 | Provider native harness/reserve/start adapter; declaration/member harness_version; declarations/fields: `harness_version` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 394–400 | Provider native harness/reserve/start adapter; declaration/member harness_sha256; declarations/fields: `harness_sha256` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 401–423 | Provider native harness/reserve/start adapter; declaration/member reserve; declarations/fields: `reserve` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 424–442 | Provider native harness/reserve/start adapter; declaration/member start; declarations/fields: `start` |
| [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) / active | 443–461 | Native wait adapter; declarations/fields: `wait` |
| [I06](../../slices/identity-brokering.md#i06-completion-revocation-and-reconciliation) / active | 462–478 | Stop/retire/final credential adapters; declarations/fields: `stop` |
| [I06](../../slices/identity-brokering.md#i06-completion-revocation-and-reconciliation) / active | 479–495 | Stop/retire/final credential adapters; declaration/member stop_unbound; declarations/fields: `stop_unbound` |
| [I06](../../slices/identity-brokering.md#i06-completion-revocation-and-reconciliation) / active | 496–512 | Stop/retire/final credential adapters; declaration/member capture; declarations/fields: `capture` |
| [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) / active | 513–521 | Native liveness adapter; declarations/fields: `live` |
| [S06](../../slices/spaces-and-terminals.md#s06-factory-activity-presentation) / active | 522–543 | Factory output adapter; declarations/fields: `output` |
| [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) / active | 544–558 | Human takeover adapter; declarations/fields: `takeover_copy` |
| [F09](../../slices/factory-coordination.md#f09-publication-progression) / active | 559–577 | Native candidate export adapter; declarations/fields: `export_bundle` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 578–588 | Provider-specific Muse/Codex selection and immutable pin; declarations/fields: `muse_scoped_lease` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 589–602 | Provider-specific Muse/Codex selection and immutable pin; declaration/member service; declarations/fields: `service` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 603–613 | Provider-specific Muse/Codex selection and immutable pin; declaration/member pin_family; declarations/fields: `pin_family` |
| [I04](../../slices/identity-brokering.md#i04-execution-admission-and-lease-fencing) / active | 614 | Broker acquire adapter; declarations/fields: `acquire` |
| [I04](../../slices/identity-brokering.md#i04-execution-admission-and-lease-fencing) / active | 615–626 | Broker acquire adapter; declaration/member acquire; declarations/fields: `acquire` |
| [I05](../../slices/identity-brokering.md#i05-native-binding-and-private-delivery) / active | 627–640 | Broker native registration adapter; declarations/fields: `register` |
| [I06](../../slices/identity-brokering.md#i06-completion-revocation-and-reconciliation) / active | 641–652 | Broker credential return/reconcile/close adapters; declarations/fields: `return_lease` |
| [I06](../../slices/identity-brokering.md#i06-completion-revocation-and-reconciliation) / active | 653 | Broker credential return/reconcile/close adapters; declaration/member reconcile_lease; declarations/fields: `reconcile_lease` |
| [I06](../../slices/identity-brokering.md#i06-completion-revocation-and-reconciliation) / active | 654–664 | Broker credential return/reconcile/close adapters; declaration/member execution_is_terminal; declarations/fields: `execution_is_terminal` |
| [I06](../../slices/identity-brokering.md#i06-completion-revocation-and-reconciliation) / active | 665–676 | Broker credential return/reconcile/close adapters; declaration/member close_execution; declarations/fields: `close_execution` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 677 | Muse/native execution hook adapters; declarations/fields: `acquire`, `attach`, `end`, `authorize`, `nested_authorize`, `select` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 678–684 | Muse/native execution hook adapters; declaration/member acquire; declarations/fields: `acquire` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 685–692 | Muse/native execution hook adapters; declaration/member attach; declarations/fields: `attach` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 693–695 | Muse/native execution hook adapters; declaration/member end; declarations/fields: `end` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 696–700 | Muse/native execution hook adapters; declaration/member authorize; declarations/fields: `authorize` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 701–704 | Muse/native execution hook adapters; declaration/member nested_authorize; declarations/fields: `nested_authorize` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 705–716 | Muse/native execution hook adapters; declaration/member select; declarations/fields: `select` |
| [I04](../../slices/identity-brokering.md#i04-execution-admission-and-lease-fencing) / active | 717 | Human execution broker acquisition adapter; declarations/fields: `acquire` |
| [I04](../../slices/identity-brokering.md#i04-execution-admission-and-lease-fencing) / active | 718–724 | Human execution broker acquisition adapter; declaration/member acquire; declarations/fields: `acquire` |
| [I05](../../slices/identity-brokering.md#i05-native-binding-and-private-delivery) / active | 725–732 | Native registration adapter; declarations/fields: `register` |
| [I06](../../slices/identity-brokering.md#i06-completion-revocation-and-reconciliation) / active | 733–746 | Human execution reconciliation adapter; declarations/fields: `reconcile_lease` |
| [I06](../../slices/identity-brokering.md#i06-completion-revocation-and-reconciliation) / active | 747–758 | Human execution reconciliation adapter; declaration/member targeted_id; declarations/fields: `targeted_id` |
| [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) / active | 759 | Profile observation backend; declarations/fields: `profile` |
| [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) / active | 760–768 | Profile observation backend; declaration/member profile; declarations/fields: `profile` |
| [P01](../../slices/projects.md#p01-repository-association-and-creation) / active | 769–784 | Native Project create backend; declarations/fields: `create` |
| [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) / active | 785–794 | Environment/native OS observation; declarations/fields: `inspect` |
| [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) / active | 795–803 | Environment/native OS observation; declaration/member observe_os; declarations/fields: `observe_os` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 804–812 | SSH connection backend; declarations/fields: `connection` |
| [P05](../../slices/projects.md#p05-project-startstop) / active | 813–822 | Start/Stop backend; declarations/fields: `lifecycle` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 823–830 | Developer keys backend; declarations/fields: `access_keys` |
| [P03](../../slices/projects.md#p03-human-membership-and-accounts) / active | 831–839 | Human account backend; declarations/fields: `account` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 840–847 | Native preparation backend; declarations/fields: `prepare` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 848–855 | Native preparation backend; declaration/member prepare_candidate; declarations/fields: `prepare_candidate` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 856–863 | Native preparation backend; declaration/member inspect_preparation; declarations/fields: `inspect_preparation` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 864–871 | Native preparation backend; declaration/member stop_preparation; declarations/fields: `stop_preparation` |
| [P12](../../slices/projects.md#p12-maintenance-holds) / active | 872–879 | Hold backend; declarations/fields: `hold_preparation` |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 880–892 | Factory launch backend; declarations/fields: `factory_launch` |
| [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) / active | 893–903 | Run inspect/stop/takeover backend; declarations/fields: `factory_inspect` |
| [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) / active | 904–914 | Run inspect/stop/takeover backend; declaration/member factory_stop; declarations/fields: `factory_stop` |
| [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) / active | 915–925 | Run inspect/stop/takeover backend; declaration/member factory_takeover; declarations/fields: `factory_takeover` |
| [S06](../../slices/spaces-and-terminals.md#s06-factory-activity-presentation) / active | 926–936 | Factory output backend; declarations/fields: `factory_output` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 937–946 | Harness/native identity execution backend; declarations/fields: `factory_harness` |
| [F09](../../slices/factory-coordination.md#f09-publication-progression) / active | 947–957 | Candidate export backend; declarations/fields: `factory_export` |
| [F11](../../slices/factory-coordination.md#f11-candidate-verification-assessment) / active | 958–968 | Candidate verification backend; declarations/fields: `factory_candidate_inspect` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 969–983 | Provider launch and callback backend; declarations/fields: `identity_launch` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 984–1016 | Provider launch and callback backend; declaration/member identity_action; declarations/fields: `identity_action` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 1017–1040 | Tailnet operation backend; declarations/fields: `tailnet` |
| [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) / active | 1041–1051 | Attach registration and pump backend; declarations/fields: `terminal_accept` |
| [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) / active | 1052–1064 | Attach registration and pump backend; declaration/member pump_terminal; declarations/fields: `pump_terminal` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 1065–1076 | Tailnet wire emission adapters; declarations/fields: `decode_tailnet_empty` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 1077–1096 | Tailnet wire emission adapters; declaration/member map_tailnet_err; declarations/fields: `map_tailnet_err` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 1097–1103 | Tailnet wire emission adapters; declaration/member valid_revision; declarations/fields: `valid_revision` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 1104–1127 | Tailnet wire emission adapters; declaration/member TAILNET_PROJECT_SPECS; declarations/fields: `TAILNET_PROJECT_SPECS` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 1128–1143 | Tailnet wire emission adapters; declaration/member decode_project_request; declarations/fields: `decode_project_request` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 1144–1180 | Tailnet wire emission adapters; declaration/member validate_project_request; declarations/fields: `validate_project_request` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 1181–1263 | Tailnet wire emission adapters; declaration/member encode_project_view; declarations/fields: `encode_project_view` |
| [N05](../../slices/networking.md#n05-project-tailnet-selection) / active | 1264–1301 | Project selection and policy native observation fencing; declarations/fields: `tailnet_project_or_policy` |
| [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) / active | 1302–1310 | WebSocket parsing and interactive native pump; declarations/fields: `WsIn` |
| [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) / active | 1311–1379 | WebSocket parsing and interactive native pump; declaration/member ws_read_message; declarations/fields: `ws_read_message` |
| [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) / active | 1380–1390 | WebSocket parsing and interactive native pump; declaration/member read_exact; declarations/fields: `read_exact` |
| [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) / active | 1391–1401 | WebSocket parsing and interactive native pump; declaration/member set_deadline; declarations/fields: `set_deadline` |
| [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) / active | 1402–1411 | WebSocket parsing and interactive native pump; declaration/member ws_write_text; declarations/fields: `ws_write_text` |
| [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) / active | 1412–1444 | WebSocket parsing and interactive native pump; declaration/member ws_write_frame; declarations/fields: `ws_write_frame` |
| [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) / active | 1445–1459 | WebSocket parsing and interactive native pump; declaration/member closed_frame; declarations/fields: `closed_frame` |
| [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) / active | 1460–1612 | WebSocket parsing and interactive native pump; declaration/member pump_terminal; declarations/fields: `pump_terminal` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 1613–1616, 1691–1702, 1738–1752, 1793–1818, 1994–2007 | Backend composition and domain adapter source assertions; declarations/fields: `tests`, `targeted_routes_validate_id_only`, `backend_with_scratch_tailnet`, `tailnet_rejects_bad_input_as_invalid`, `masked_frame` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 1617–1643 | Backend composition and domain adapter source assertions; declaration/member test_config; declarations/fields: `test_config` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 1644–1647 | Backend composition and domain adapter source assertions; declaration/member backend; declarations/fields: `backend` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 1648–1659 | Backend composition and domain adapter source assertions; declaration/member backend_with_muse; declarations/fields: `backend_with_muse` |
| [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) / active | 1660–1671 | Source assertion of Profile and runtime readiness; declarations/fields: `profile_rejects_unknown_fields` |
| [P01](../../slices/projects.md#p01-repository-association-and-creation) / active | 1672–1690 | Source assertion of Repository association and creation; declarations/fields: `create_checks_identity_before_validate` |
| [P05](../../slices/projects.md#p05-project-startstop) / active | 1703–1719 | Source assertion of Project Start/Stop; declarations/fields: `lifecycle_rejects_bad_action_without_exec` |
| [I03](../../slices/identity-brokering.md#i03-delegation-and-connection-availability) / active | 1720–1737 | Source assertion of Delegation and connection availability; declarations/fields: `factory_routes_report_unavailable_root` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 1753–1764 | Backend composition and domain adapter source assertions; declaration/member tailnet_settings_and_options_answer_without_appliance_state; declarations/fields: `tailnet_settings_and_options_answer_without_appliance_state` |
| [N04](../../slices/networking.md#n04-project-enrollment-policy) / active | 1765–1777 | Source assertion of Project enrollment policy; declarations/fields: `tailnet_host_and_enrollment_reject_malformed_bodies` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 1778–1792 | Source assertion of Provider execution integration; declarations/fields: `identity_launch_without_broker_is_internal` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 1819–1864 | Backend composition and domain adapter source assertions; declaration/member tailnet_validate_rules_match_go; declarations/fields: `tailnet_validate_rules_match_go` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 1865–1889 | Backend composition and domain adapter source assertions; declaration/member tailnet_view_encode_matches_go_omitempty; declarations/fields: `tailnet_view_encode_matches_go_omitempty` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 1890–1928 | Backend composition and domain adapter source assertions; declaration/member factory_error_mapping; declarations/fields: `factory_error_mapping` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 1929–1958 | Backend composition and domain adapter source assertions; declaration/member tailnet_error_mapping; declarations/fields: `tailnet_error_mapping` |
| [I05](../../slices/identity-brokering.md#i05-native-binding-and-private-delivery) / active | 1959–1993 | Source assertion of Native binding and private delivery; declarations/fields: `lease_binding_round_trip` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 2008–2024 | Backend composition and domain adapter source assertions; declaration/member read_server_frame; declarations/fields: `read_server_frame` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 2025–2080 | Backend composition and domain adapter source assertions; declaration/member ws_codec_round_trip_ping_fragmentation_close; declarations/fields: `ws_codec_round_trip_ping_fragmentation_close` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 2081–2087 | Backend composition and domain adapter source assertions; declaration/member terminal_accept_mints_unique_sessions; declarations/fields: `terminal_accept_mints_unique_sessions` |

<a id="coverage-6adfee929431"></a>

## [rust/soda-host/src/iconfig.rs](../../../../../rust/soda-host/src/iconfig.rs)



| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [H04](../../slices/shared-supporting-slices.md#h04-configuration-and-filesystem-primitives) / active | 1–23 | Protected host config and known field slots; declarations/fields: `Config` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 24–33 | Provider binary pins and private execution sockets |
| [N06](../../slices/networking.md#n06-project-companion-lifecycle) / active | 34–35 | Companion management/image settings |
| [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) / active | 36 | Project image setting |
| [N02](../../slices/networking.md#n02-project-lan-access) / active | 37–39 | Project LAN settings |
| [H04](../../slices/shared-supporting-slices.md#h04-configuration-and-filesystem-primitives) / active | 40–112 | Strict config binding slots; declarations/fields: `CONFIG_SPECS` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 113–176 | First JSON value boundary parsing; declarations/fields: `first_value_end` |
| [D11](../../slices/release-and-installation.md#d11-host-installation-and-payload-application) / active | 177–214 | Installed payload image binding overlay; declarations/fields: `apply_release_images`, `UNAVAILABLE` |
| [H04](../../slices/shared-supporting-slices.md#h04-configuration-and-filesystem-primitives) / active | 215–226 | Config path basename helper; declarations/fields: `base_name` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 227–279 | Muse and Codex execution runtime admission; declarations/fields: `validate_muse_runtime`, `validate_identity_runtime` |
| [N06](../../slices/networking.md#n06-project-companion-lifecycle) / active | 280–289 | Companion image setting validation; declarations/fields: `valid_tailnet_image` |
| [N02](../../slices/networking.md#n02-project-lan-access) / active | 290–298 | Project network-name validation; declarations/fields: `valid_network_names` |
| [H04](../../slices/shared-supporting-slices.md#h04-configuration-and-filesystem-primitives) / active | 299–356 | Ordered cross-runtime config validation and load; declarations/fields: `validate_runtime_config`, `load_config` |

<a id="coverage-e886011dfdad"></a>

## [rust/soda-host/src/lib.rs](../../../../../rust/soda-host/src/lib.rs)

Public module wiring is mapped separately; module exposure does not create a new process boundary.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 1–6 | Privileged host daemon composition, private native operations and listener lifetime |
| [P03](../../slices/projects.md#p03-human-membership-and-accounts) / active | 7 | Module wiring: account; Human membership and accounts; declarations/fields: `account` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 8 | Module wiring: dbackend; Private IPC and service lifetime; declarations/fields: `dbackend` |
| [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) / active | 9 | Module wiring: domain; Profile and runtime readiness; declarations/fields: `domain` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 10 | Module wiring: gmux_admission; Private IPC and service lifetime; declarations/fields: `gmux_admission` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 11 | Module wiring: gmux_backend; Private IPC and service lifetime; declarations/fields: `gmux_backend` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 12 | Module wiring: gmux_routes; Private IPC and service lifetime; declarations/fields: `gmux_routes` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 13 | Module wiring: gmux_server; Private IPC and service lifetime; declarations/fields: `gmux_server` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 14 | Module wiring: iclient; Private IPC and service lifetime; declarations/fields: `iclient` |
| [H04](../../slices/shared-supporting-slices.md#h04-configuration-and-filesystem-primitives) / active | 15 | Module wiring: iconfig; Configuration and filesystem primitives; declarations/fields: `iconfig` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 16 | Module wiring: json; Encoding and parsing; declarations/fields: `json` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 17 | Module wiring: muse; Provider execution integration; declarations/fields: `muse` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 18 | Module wiring: muse_serve; Provider execution integration; declarations/fields: `muse_serve` |
| [N02](../../slices/networking.md#n02-project-lan-access) / active | 19 | Module wiring: net; Project LAN access; declarations/fields: `net` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 20 | Module wiring: nist; Encoding and parsing; declarations/fields: `nist` |
| [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) / active | 21 | Module wiring: pfactory; Run lifecycle and intervention; declarations/fields: `pfactory` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 22 | Module wiring: pops; Checkout allocation and preparation; declarations/fields: `pops` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 23 | Module wiring: preparation; Checkout allocation and preparation; declarations/fields: `preparation` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 24 | Module wiring: prepare; privileged checkout preparation; declarations/fields: `prepare` |
| [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) / active | 25 | Module wiring: project; Profile and runtime readiness; declarations/fields: `project` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 26 | Module wiring: sha256; Encoding and parsing; declarations/fields: `sha256` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 27 | Module wiring: ssh; Development SSH access; declarations/fields: `ssh` |
| [N06](../../slices/networking.md#n06-project-companion-lifecycle) / active | 28 | Module wiring: tailnet_companion; Project companion lifecycle; declarations/fields: `tailnet_companion` |
| [N06](../../slices/networking.md#n06-project-companion-lifecycle) / active | 29 | Module wiring: tailnet_domain; Project companion lifecycle; declarations/fields: `tailnet_domain` |
| [N06](../../slices/networking.md#n06-project-companion-lifecycle) / active | 30 | Module wiring: tailnet_files; Project companion lifecycle; declarations/fields: `tailnet_files` |
| [N06](../../slices/networking.md#n06-project-companion-lifecycle) / active | 31 | Module wiring: tailnet_runtime; Project companion lifecycle; declarations/fields: `tailnet_runtime` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 32 | Module wiring: tcodex; Provider execution integration; declarations/fields: `tcodex` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 33 | Module wiring: tcontrol; Host Tailnet control; declarations/fields: `tcontrol` |
| [N04](../../slices/networking.md#n04-project-enrollment-policy) / active | 34 | Module wiring: tcontrol_enroll; Project enrollment policy; declarations/fields: `tcontrol_enroll` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 35 | Module wiring: tcontrol_native; Host Tailnet control; declarations/fields: `tcontrol_native` |
| [N04](../../slices/networking.md#n04-project-enrollment-policy) / active | 36 | Module wiring: tcontrol_policy; Project enrollment policy; declarations/fields: `tcontrol_policy` |
| [N04](../../slices/networking.md#n04-project-enrollment-policy) / active | 37 | Module wiring: tcontrol_provider; Project enrollment policy; declarations/fields: `tcontrol_provider` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 38 | Module wiring: tcontrol_wire; Encoding and parsing; declarations/fields: `tcontrol_wire` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 39 | Module wiring: texec; Human terminal lifecycle; declarations/fields: `texec` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 40 | Module wiring: tfactory; Provider execution integration; declarations/fields: `tfactory` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 41 | Module wiring: tmuse; Provider execution integration; declarations/fields: `tmuse` |

<a id="coverage-f793d9450ffe"></a>

## [rust/soda-host/src/main.rs](../../../../../rust/soda-host/src/main.rs)

Source assertions/fixtures were inspected for mapping only; no test execution, runtime or installed proof claimed. Large-file ledger retains committed declaration/member spans rather than grouping methods into one whole-file unit; responsibility follows the verified concern/branch mapping, not declaration count.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 1–20 | Host binary flags and root/private listener startup; declarations/fields: `DEFAULT_CONFIG`, `RELEASE_CONFIG`, `FACTORY_STATE_DIR`, `MUSE_JOIN_TIMEOUT`, `SHUTDOWN_DRAIN`, `SHUTDOWN`, `handle_signal`, `install_signal_handlers`, `MainError`, `message`, `exit_code`, `Flags`, `usage`, `parse_flags`, `is_root` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 21 | Host binary flags and root/private listener startup; declaration/member DEFAULT_CONFIG; declarations/fields: `DEFAULT_CONFIG` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 22 | Host binary flags and root/private listener startup; declaration/member RELEASE_CONFIG; declarations/fields: `RELEASE_CONFIG` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 23 | Host binary flags and root/private listener startup; declaration/member FACTORY_STATE_DIR; declarations/fields: `FACTORY_STATE_DIR` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 24 | Host binary flags and root/private listener startup; declaration/member MUSE_JOIN_TIMEOUT; declarations/fields: `MUSE_JOIN_TIMEOUT` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 25–26 | Host binary flags and root/private listener startup; declaration/member SHUTDOWN_DRAIN; declarations/fields: `SHUTDOWN_DRAIN` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 27–28 | Host binary flags and root/private listener startup; declaration/member SHUTDOWN; declarations/fields: `SHUTDOWN` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 29–32 | Host binary flags and root/private listener startup; declaration/member handle_signal; declarations/fields: `handle_signal` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 33–46 | Host binary flags and root/private listener startup; declaration/member install_signal_handlers; declarations/fields: `install_signal_handlers` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 47–52 | Host binary flags and root/private listener startup; declaration/member MainError; declarations/fields: `MainError` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 53–58 | Host binary flags and root/private listener startup; declaration/member message; declarations/fields: `message` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 59–66 | Host binary flags and root/private listener startup; declaration/member exit_code; declarations/fields: `exit_code` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 67 | Host binary flags and root/private listener startup; declaration/member Flags; declarations/fields: `Flags` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 68 | Host binary flags and root/private listener startup; declaration/member Flags.config; declarations/fields: `Flags.config` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 69 | Host binary flags and root/private listener startup; declaration/member Flags.release; declarations/fields: `Flags.release` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 70 | Host binary flags and root/private listener startup; declaration/member Flags.listen_path; declarations/fields: `Flags.listen_path` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 71 | Host binary flags and root/private listener startup; declaration/member Flags.tailnet_action; declarations/fields: `Flags.tailnet_action` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 72 | Host binary flags and root/private listener startup; declaration/member Flags.project; declarations/fields: `Flags.project` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 73–75 | Host binary flags and root/private listener startup; declaration/member Flags.positionals; declarations/fields: `Flags.positionals` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 76–88 | Host binary flags and root/private listener startup; declaration/member usage; declarations/fields: `usage` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 89–141 | Host binary flags and root/private listener startup; declaration/member parse_flags; declarations/fields: `parse_flags` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 142–148 | Host binary flags and root/private listener startup; declaration/member is_root; declarations/fields: `is_root` |
| [N06](../../slices/networking.md#n06-project-companion-lifecycle) / active | 149–192 | Explicit companion start/stop/reconcile CLI action; declarations/fields: `run_tailnet_action` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 193–223 | Privileged backend construction and service lifetime; declarations/fields: `open_backend` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 224–341 | Privileged backend construction and service lifetime; declaration/member serve_host_socket; declarations/fields: `serve_host_socket` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 342–372 | Privileged backend construction and service lifetime; declaration/member run; declarations/fields: `run` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 373–380 | Privileged backend construction and service lifetime; declaration/member main; declarations/fields: `main` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 381–383 | Startup/flag source tests; declarations/fields: `tests` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 384–388 | Startup/flag source tests; declaration/member args; declarations/fields: `args` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 389–411 | Startup/flag source tests; declaration/member flag_forms; declarations/fields: `flag_forms` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 412–421 | Startup/flag source tests; declaration/member bind_listener_keeps_root_gate; declarations/fields: `bind_listener_keeps_root_gate` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 422–431 | Startup/flag source tests; declaration/member tailnet_action_args_rejected_without_root_or_shape; declarations/fields: `tailnet_action_args_rejected_without_root_or_shape` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 432–448 | Startup/flag source tests; declaration/member exit_code_contract_and_systemd_retry; declarations/fields: `exit_code_contract_and_systemd_retry` |


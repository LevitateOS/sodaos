# Identity broker state and entrypoints

[Responsibility map index](README.md) · [Coverage snapshot and limits](../README.md).
Page grouping is navigation; the slice IDs retain their individual review ownership.

<a id="coverage-9d94b3328ec0"></a>

## [rust/soda-identity/src/main.rs](../../../../../rust/soda-identity/src/main.rs)



| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 1–106 | Broker service settings and process startup; declarations/fields: `SETTINGS_FIELDS`, `PROVIDER_FIELDS`, `ProviderSettings`, `Settings`, `SHUTDOWN`, `handle_signal`, `main`, `run`, `parse_args` |
| [H04](../../slices/shared-supporting-slices.md#h04-configuration-and-filesystem-primitives) / active | 107–132 | Protected configuration input; declarations/fields: `load` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 133–197 | Distinct admin/runtime socket activation and listener admission; declarations/fields: `service_listeners`, `listen`, `activated_listeners` |
| [H02](../../slices/shared-supporting-slices.md#h02-storage-mechanics) / active | 198–203 | Open shared broker PostgreSQL Store with protected configured DSN/key; declarations/fields: `open_store` |
| [I02](../../slices/identity-brokering.md#i02-encrypted-credential-custody) / active | 204–244 | Broker key/DSN secret input and protected encryption custody; declarations/fields: `open_store`, `grant_key`, `secret` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 245–246, 276–280 | Broker assembly and provider/controller initialization; declarations/fields: `open_broker` |
| [I07](../../slices/identity-brokering.md#i07-codex-adapter) / active | 247–261 | Construct configured Codex enrollment adapter and private native enrollment root; declarations/fields: `identity_providers::codex::Provider::new` |
| [I08](../../slices/identity-brokering.md#i08-muse-adapter) / active | 262–275 | Construct configured Muse enrollment adapter and private native enrollment root; declarations/fields: `identity_providers::muse::Provider::new` |
| [H04](../../slices/shared-supporting-slices.md#h04-configuration-and-filesystem-primitives) / active | 281–316 | Protected directory creation; declarations/fields: `mkdir_all_mode` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 317–344, 347–356, 363–372 | Private server worker lifetime and periodic reconciliation; declarations/fields: `serve` |
| [I06](../../slices/identity-brokering.md#i06-completion-revocation-and-reconciliation) / active | 345–346 | Schedule broker periodic exact lease reconciliation; declarations/fields: `broker.sweep` |
| [I06](../../slices/identity-brokering.md#i06-completion-revocation-and-reconciliation) / active | 357–362 | Drive periodic lease retirement/reconciliation and preserve unconfirmed termination result; declarations/fields: `broker.sweep` |

<a id="coverage-00ed7b9f06a9"></a>

## [rust/soda-identity/src/schema.rs](../../../../../rust/soda-identity/src/schema.rs)

Embedded generated compatibility copy names internal/store/schema.go as the source of truth; embedded drift test is structural source coverage, not native PostgreSQL execution. The users SQL lines mix native actor mirror id/login (G01) and local display-name preference name (G09); Project declaration mixes association/owner (P01), readiness/profile (P02) and LAN ip (N02). These are generated same-line schema references, not broker domain mutation ownership. Named generated SQL subunits on lines 11/13/149/151 share physical lines because the declaration/query is dense; symbols identify disjoint fields, not competing ownership of a field or duplicate production operations.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [H02](../../slices/shared-supporting-slices.md#h02-storage-mechanics) / generated | 1–8, 148, 183–229 | Embedded compatibility copy of the shared Go PostgreSQL schema; declarations/fields: `SCHEMA_VERSION`, `STATEMENTS`, `VERIFY_QUERIES`, `VERIFY_TRIGGERS`, `tests`, `go_literals`, `schema_matches_go_source` |
| [H02](../../slices/shared-supporting-slices.md#h02-storage-mechanics) / generated | 9 | Generated shared SQL declaration/constraint/query for schema_version; declarations/fields: `schema_version` |
| [H02](../../slices/shared-supporting-slices.md#h02-storage-mechanics) / generated | 10, 64–67 | Generated shared SQL schema/query compatibility mechanics; declarations/fields: `INSERT INTO schema_version`, `CREATE OR REPLACE FUNCTION soda_reject_immutable` |
| [G01](../../slices/forgejo-integration.md#g01-browser-authority-and-contributions) / generated | 11, 149 | Generated native user mirror fields users.id/users.login; browser actor identity reference, not Project native-account provisioning; declarations/fields: `users` |
| [G09](../../slices/forgejo-integration.md#g09-local-profile-preferences) / generated | 11 | Generated users.name display-name preference field/default, separate from id/login on the same SQL line; declarations/fields: `users.name` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / generated | 12, 150 | Generated saved developer public-key fields keys.id/user_id/public/fingerprint and uniqueness constraint; declarations/fields: `keys` |
| [N02](../../slices/networking.md#n02-project-lan-access) / generated | 13 | Generated projects.ip native LAN observation field/default; separate from association and readiness on same SQL line; declarations/fields: `projects.ip` |
| [P01](../../slices/projects.md#p01-repository-association-and-creation) / generated | 13, 125, 151 | Generated repository association fields projects.id/name/repository_id/owner_id/repository; additional readiness and LAN field seams listed separately; declarations/fields: `projects` |
| [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) / generated | 13 | Generated projects.ready/creation_profile readiness and immutable profile fields/check; separate from repository association on same SQL line; declarations/fields: `projects.ready`, `projects.creation_profile`, `octet_length(creation_profile::text)<=1024` |
| [P03](../../slices/projects.md#p03-human-membership-and-accounts) / generated | 14, 152 | Generated shared SQL declaration/constraint/query for memberships; declarations/fields: `memberships` |
| [I02](../../slices/identity-brokering.md#i02-encrypted-credential-custody) / generated | 15, 153 | Generated shared SQL declaration/constraint/query for grant_key_check; declarations/fields: `grant_key_check` |
| [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) / generated | 16–21, 158 | Generated shared SQL declaration/constraint/query for factory_runs; declarations/fields: `factory_runs` |
| [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) / generated | 22–26, 159 | Generated shared SQL declaration/constraint/query for factory_commands; declarations/fields: `factory_commands` |
| [F01](../../slices/factory-coordination.md#f01-repository-factory-policy) / generated | 27, 160 | Generated shared SQL declaration/constraint/query for factory_policies; declarations/fields: `factory_policies` |
| [F03](../../slices/factory-coordination.md#f03-capacity-reservations-and-accounting) / generated | 28, 161 | Generated shared SQL declaration/constraint/query for factory_capacity; declarations/fields: `factory_capacity` |
| [F02](../../slices/factory-coordination.md#f02-operator-execution-grants) / generated | 29, 162 | Generated shared SQL declaration/constraint/query for factory_operator_grants; declarations/fields: `factory_operator_grants` |
| [F04](../../slices/factory-coordination.md#f04-connection-sponsorship) / generated | 30, 163 | Generated shared SQL declaration/constraint/query for factory_sponsorships; declarations/fields: `factory_sponsorships` |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / generated | 31, 164 | Generated shared SQL declaration/constraint/query for factory_dispatch; declarations/fields: `factory_dispatch` |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / generated | 32–33, 128–129, 165 | Generated shared SQL declaration/constraint/query for factory_dispatch_regs; declarations/fields: `factory_dispatch_regs` |
| [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) / generated | 34, 130–131, 169 | Generated shared SQL declaration/constraint/query for factory_takeovers; declarations/fields: `factory_takeovers` |
| [S06](../../slices/spaces-and-terminals.md#s06-factory-activity-presentation) / generated | 35, 132–133, 178 | Generated shared SQL declaration/constraint/query for factory_run_views; declarations/fields: `factory_run_views` |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / generated | 36–37, 166 | Generated shared SQL declaration/constraint/query for factory_assignments; declarations/fields: `factory_assignments` |
| [F03](../../slices/factory-coordination.md#f03-capacity-reservations-and-accounting) / generated | 38, 167 | Generated shared SQL declaration/constraint/query for factory_reservations; declarations/fields: `factory_reservations` |
| [F03](../../slices/factory-coordination.md#f03-capacity-reservations-and-accounting) / generated | 39, 168 | Generated shared SQL declaration/constraint/query for factory_usage; declarations/fields: `factory_usage` |
| [F05](../../slices/factory-coordination.md#f05-accepted-requirements-and-invalidation) / generated | 40, 134–135, 170 | Generated shared SQL declaration/constraint/query for issue_acceptance_decisions; declarations/fields: `issue_acceptance_decisions` |
| [F05](../../slices/factory-coordination.md#f05-accepted-requirements-and-invalidation) / generated | 41, 171 | Generated shared SQL declaration/constraint/query for issue_acceptance_heads; declarations/fields: `issue_acceptance_heads` |
| [F05](../../slices/factory-coordination.md#f05-accepted-requirements-and-invalidation) / generated | 42, 136–137, 172 | Generated shared SQL declaration/constraint/query for issue_acceptance_withdrawals; declarations/fields: `issue_acceptance_withdrawals` |
| [F06](../../slices/factory-coordination.md#f06-issue-intake-and-readiness) / generated | 43 | Generated shared SQL declaration/constraint/query for issue_controls; declarations/fields: `issue_controls` |
| [F06](../../slices/factory-coordination.md#f06-issue-intake-and-readiness) / generated | 44 | Generated shared SQL declaration/constraint/query for intake_deliveries; declarations/fields: `intake_deliveries` |
| [F09](../../slices/factory-coordination.md#f09-publication-progression) / generated | 45, 179 | Generated shared SQL declaration/constraint/query for factory_publications; declarations/fields: `factory_publications` |
| [F11](../../slices/factory-coordination.md#f11-candidate-verification-assessment) / generated | 46 | Generated shared SQL declaration/constraint/query for factory_check_assessments; declarations/fields: `factory_check_assessments` |
| [F12](../../slices/factory-coordination.md#f12-merge-eligibility-and-completion) / generated | 47, 180–182 | Generated shared SQL declaration/constraint/query for factory_merges; declarations/fields: `factory_merges` |
| [F06](../../slices/factory-coordination.md#f06-issue-intake-and-readiness) / generated | 48 | Generated shared SQL declaration/constraint/query for factory_readiness_sweeps; declarations/fields: `factory_readiness_sweeps` |
| [P10](../../slices/projects.md#p10-shared-tools-and-packages) / generated | 49, 173 | Generated shared SQL declaration/constraint/query for project_environment_grants; declarations/fields: `project_environment_grants` |
| [P08](../../slices/projects.md#p08-preparation-requirements-acceptance) / generated | 50, 138–139, 174 | Generated shared SQL declaration/constraint/query for project_requirement_decisions; declarations/fields: `project_requirement_decisions` |
| [P08](../../slices/projects.md#p08-preparation-requirements-acceptance) / generated | 51, 175 | Generated shared SQL declaration/constraint/query for project_requirement_heads; declarations/fields: `project_requirement_heads` |
| [P09](../../slices/projects.md#p09-privileged-preparation-approval) / generated | 52, 140–141, 176 | Generated shared SQL declaration/constraint/query for project_approval_decisions; declarations/fields: `project_approval_decisions` |
| [P09](../../slices/projects.md#p09-privileged-preparation-approval) / generated | 53, 177 | Generated shared SQL declaration/constraint/query for project_approval_heads; declarations/fields: `project_approval_heads` |
| [I02](../../slices/identity-brokering.md#i02-encrypted-credential-custody) / generated | 54 | Generated shared SQL declaration/constraint/query for identity_connections; declarations/fields: `identity_connections` |
| [I03](../../slices/identity-brokering.md#i03-delegation-and-connection-availability) / generated | 55–56 | Generated shared SQL declaration/constraint/query for identity_grants; declarations/fields: `identity_grants` |
| [I04](../../slices/identity-brokering.md#i04-execution-admission-and-lease-fencing) / generated | 57 | Generated shared SQL declaration/constraint/query for identity_leases; declarations/fields: `identity_leases` |
| [I10](../../slices/identity-brokering.md#i10-identity-audit-history) / generated | 58, 142–143 | Generated shared SQL declaration/constraint/query for identity_events; declarations/fields: `identity_events` |
| [P05](../../slices/projects.md#p05-project-startstop) / generated | 59, 154 | Generated shared SQL declaration/constraint/query for project_lifecycle_grants; declarations/fields: `project_lifecycle_grants` |
| [P12](../../slices/projects.md#p12-maintenance-holds) / generated | 60, 155 | Generated shared SQL declaration/constraint/query for project_maintenance; declarations/fields: `project_maintenance` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / generated | 61–62, 156 | Generated shared SQL declaration/constraint/query for project_preparations; declarations/fields: `project_preparations` |
| [I04](../../slices/identity-brokering.md#i04-execution-admission-and-lease-fencing) / generated | 63, 157 | Generated shared SQL declaration/constraint/query for identity_executions; declarations/fields: `identity_executions` |
| [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) / generated | 68–74 | Generated immutable creation-profile SQL constraint; declarations/fields: `CREATE OR REPLACE FUNCTION soda_guard_creation_profile` |
| [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) / generated | 75–92, 126 | Generated immutable factory run identity/final outcome SQL constraint; declarations/fields: `CREATE OR REPLACE FUNCTION soda_guard_run_binding`, `factory_runs` |
| [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) / generated | 93–104, 127 | Generated immutable factory command receipt SQL constraint; declarations/fields: `CREATE OR REPLACE FUNCTION soda_guard_command`, `factory_commands` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / generated | 105–114, 144 | Generated immutable preparation decision-reference SQL constraint; declarations/fields: `CREATE OR REPLACE FUNCTION soda_guard_preparation_refs`, `project_preparations` |
| [I04](../../slices/identity-brokering.md#i04-execution-admission-and-lease-fencing) / generated | 115–124, 145–147 | Generated immutable execution fence identity/terminal SQL constraint; declarations/fields: `CREATE OR REPLACE FUNCTION soda_guard_execution_identity`, `identity_executions` |
| [G09](../../slices/forgejo-integration.md#g09-local-profile-preferences) / generated | 149 | Generated schema verification query names users.name preference field alongside native actor mirror id/login; declarations/fields: `users.name` |
| [N02](../../slices/networking.md#n02-project-lan-access) / generated | 151 | Generated schema verification query names projects.ip native LAN observation field; declarations/fields: `projects.ip` |
| [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) / generated | 151 | Generated schema verification query names projects.ready/creation_profile alongside association fields; declarations/fields: `projects.ready`, `projects.creation_profile` |

<a id="coverage-984632265c1d"></a>

<a id="rustsoda-identitysrcstorers-1"></a>

## [rust/soda-identity/src/store.rs](../../../../../rust/soda-identity/src/store.rs)

Source assertions/fixtures were inspected for mapping only; no test execution, runtime or installed proof claimed. Large-file ledger retains committed declaration/member spans rather than grouping methods into one whole-file unit; responsibility follows the verified concern/branch mapping, not declaration count.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [H02](../../slices/shared-supporting-slices.md#h02-storage-mechanics) / active | 1–12 | Transactional PostgreSQL Store and shared schema admission; declarations/fields: `identity_binding`, `bind`, `Store`, `Tx`, `open_encrypted`, `open`, `query`, `exec`, `query_row`, `simple`, `transaction` |
| [H02](../../slices/shared-supporting-slices.md#h02-storage-mechanics) / active | 13–18 | Transactional PostgreSQL Store and shared schema admission; declaration/member identity_binding; declarations/fields: `identity_binding` |
| [H02](../../slices/shared-supporting-slices.md#h02-storage-mechanics) / active | 19–50 | Transactional PostgreSQL Store and shared schema admission; declaration/member bind; declarations/fields: `bind` |
| [H02](../../slices/shared-supporting-slices.md#h02-storage-mechanics) / active | 51 | Transactional PostgreSQL Store and shared schema admission; declaration/member Store; declarations/fields: `Store` |
| [H02](../../slices/shared-supporting-slices.md#h02-storage-mechanics) / active | 52 | Transactional PostgreSQL Store and shared schema admission; declaration/member Store.client; declarations/fields: `Store.client` |
| [H02](../../slices/shared-supporting-slices.md#h02-storage-mechanics) / active | 53–55 | Transactional PostgreSQL Store and shared schema admission; declaration/member Store.grants; declarations/fields: `Store.grants` |
| [H02](../../slices/shared-supporting-slices.md#h02-storage-mechanics) / active | 56 | Transactional PostgreSQL Store and shared schema admission; declaration/member Tx; declarations/fields: `Tx` |
| [H02](../../slices/shared-supporting-slices.md#h02-storage-mechanics) / active | 57–60 | Transactional PostgreSQL Store and shared schema admission; declaration/member Tx.store; declarations/fields: `Tx.store` |
| [H02](../../slices/shared-supporting-slices.md#h02-storage-mechanics) / active | 61–65 | Transactional PostgreSQL Store and shared schema admission; declaration/member open_encrypted; declarations/fields: `open_encrypted` |
| [H02](../../slices/shared-supporting-slices.md#h02-storage-mechanics) / active | 66–80 | Transactional PostgreSQL Store and shared schema admission; declaration/member open; declarations/fields: `open` |
| [H02](../../slices/shared-supporting-slices.md#h02-storage-mechanics) / active | 81–86 | Transactional PostgreSQL Store and shared schema admission; declaration/member query; declarations/fields: `query` |
| [H02](../../slices/shared-supporting-slices.md#h02-storage-mechanics) / active | 87–90 | Transactional PostgreSQL Store and shared schema admission; declaration/member exec; declarations/fields: `exec` |
| [H02](../../slices/shared-supporting-slices.md#h02-storage-mechanics) / active | 91–95 | Transactional PostgreSQL Store and shared schema admission; declaration/member query_row; declarations/fields: `query_row` |
| [H02](../../slices/shared-supporting-slices.md#h02-storage-mechanics) / active | 96–99 | Transactional PostgreSQL Store and shared schema admission; declaration/member simple; declarations/fields: `simple` |
| [H02](../../slices/shared-supporting-slices.md#h02-storage-mechanics) / active | 100–117 | Transactional PostgreSQL Store and shared schema admission; declaration/member transaction; declarations/fields: `transaction` |
| [I02](../../slices/identity-brokering.md#i02-encrypted-credential-custody) / active | 118–132 | Grant encryption-key verification and custody init; declarations/fields: `check_grant_key` |
| [I02](../../slices/identity-brokering.md#i02-encrypted-credential-custody) / active | 133–152 | Grant encryption-key verification and custody init; declaration/member reject_unkeyed_identity_credentials; declarations/fields: `reject_unkeyed_identity_credentials` |
| [I02](../../slices/identity-brokering.md#i02-encrypted-credential-custody) / active | 153–165 | Grant encryption-key verification and custody init; declaration/member validate_grant_key; declarations/fields: `validate_grant_key` |
| [I02](../../slices/identity-brokering.md#i02-encrypted-credential-custody) / active | 166–177 | Grant encryption-key verification and custody init; declaration/member initialize_grant_key; declarations/fields: `initialize_grant_key` |
| [H02](../../slices/shared-supporting-slices.md#h02-storage-mechanics) / active | 178–200 | Shared schema version admission; declarations/fields: `initialize_schema` |
| [I03](../../slices/identity-brokering.md#i03-delegation-and-connection-availability) / active | 201–206 | Delegation value validation; declarations/fields: `grants` |
| [I02](../../slices/identity-brokering.md#i02-encrypted-credential-custody) / active | 207–254 | Encrypted connection credential save/read; declarations/fields: `save_connection` |
| [I03](../../slices/identity-brokering.md#i03-delegation-and-connection-availability) / active | 255–262 | Connection metadata read; declarations/fields: `connection` |
| [I02](../../slices/identity-brokering.md#i02-encrypted-credential-custody) / active | 263–273 | Protected credential read; declarations/fields: `credential` |
| [I03](../../slices/identity-brokering.md#i03-delegation-and-connection-availability) / active | 274–283 | Connection availability; declarations/fields: `connections` |
| [I03](../../slices/identity-brokering.md#i03-delegation-and-connection-availability) / active | 284–300 | Connection availability; declaration/member available; declarations/fields: `available` |
| [I06](../../slices/identity-brokering.md#i06-completion-revocation-and-reconciliation) / active | 301–333 | Connection state/generation mutation; declarations/fields: `set_state` |
| [I03](../../slices/identity-brokering.md#i03-delegation-and-connection-availability) / active | 334–365 | Grant save/read/revoke authority; declarations/fields: `save_grant` |
| [I03](../../slices/identity-brokering.md#i03-delegation-and-connection-availability) / active | 366–373 | Grant save/read/revoke authority; declaration/member grant; declarations/fields: `grant` |
| [I03](../../slices/identity-brokering.md#i03-delegation-and-connection-availability) / active | 374–383 | Grant save/read/revoke authority; declaration/member grants_for; declarations/fields: `grants_for` |
| [I03](../../slices/identity-brokering.md#i03-delegation-and-connection-availability) / active | 384–412 | Grant save/read/revoke authority; declaration/member revoke_grant; declarations/fields: `revoke_grant` |
| [I04](../../slices/identity-brokering.md#i04-execution-admission-and-lease-fencing) / active | 413–419 | Lease read/reserve and exact execution fences; declarations/fields: `leases` |
| [I04](../../slices/identity-brokering.md#i04-execution-admission-and-lease-fencing) / active | 420–427 | Lease read/reserve and exact execution fences; declaration/member lease; declarations/fields: `lease` |
| [I04](../../slices/identity-brokering.md#i04-execution-admission-and-lease-fencing) / active | 428–458 | Lease read/reserve and exact execution fences; declaration/member reserve; declarations/fields: `reserve` |
| [I05](../../slices/identity-brokering.md#i05-native-binding-and-private-delivery) / active | 459–471 | Native binding registration persistence; declarations/fields: `register` |
| [I06](../../slices/identity-brokering.md#i06-completion-revocation-and-reconciliation) / active | 472–488 | Lease return and credential maintenance; declarations/fields: `return_lease` |
| [I04](../../slices/identity-brokering.md#i04-execution-admission-and-lease-fencing) / active | 489–499 | Execution fence read/admission; declarations/fields: `execution` |
| [I04](../../slices/identity-brokering.md#i04-execution-admission-and-lease-fencing) / active | 500–524 | Execution fence read/admission; declaration/member admit_execution; declarations/fields: `admit_execution` |
| [I06](../../slices/identity-brokering.md#i06-completion-revocation-and-reconciliation) / active | 525–543 | Terminal execution observation and forgotten lease reconciliation; declarations/fields: `observe_execution` |
| [I06](../../slices/identity-brokering.md#i06-completion-revocation-and-reconciliation) / active | 544–555 | Terminal execution observation and forgotten lease reconciliation; declaration/member forget_lease; declarations/fields: `forget_lease` |
| [I10](../../slices/identity-brokering.md#i10-identity-audit-history) / unknown | 556–572 | Bounded owner/connection immutable event read helper; runtime read exposure/retention unresolved; declarations/fields: `events` |
| [H02](../../slices/shared-supporting-slices.md#h02-storage-mechanics) / active | 573–575 | Transaction query and shared schema mechanics; declarations/fields: `query`, `exec`, `query_row`, `load_schema_version`, `verify_required_columns`, `verify_trigger` |
| [H02](../../slices/shared-supporting-slices.md#h02-storage-mechanics) / active | 576–579 | Transaction query and shared schema mechanics; declaration/member query; declarations/fields: `query` |
| [H02](../../slices/shared-supporting-slices.md#h02-storage-mechanics) / active | 580–583 | Transaction query and shared schema mechanics; declaration/member exec; declarations/fields: `exec` |
| [H02](../../slices/shared-supporting-slices.md#h02-storage-mechanics) / active | 584–587 | Transaction query and shared schema mechanics; declaration/member query_row; declarations/fields: `query_row` |
| [H02](../../slices/shared-supporting-slices.md#h02-storage-mechanics) / active | 588–620 | Transaction query and shared schema mechanics; declaration/member load_schema_version; declarations/fields: `load_schema_version` |
| [H02](../../slices/shared-supporting-slices.md#h02-storage-mechanics) / active | 621–628 | Transaction query and shared schema mechanics; declaration/member verify_required_columns; declarations/fields: `verify_required_columns` |
| [H02](../../slices/shared-supporting-slices.md#h02-storage-mechanics) / active | 629–639 | Transaction query and shared schema mechanics; declaration/member verify_trigger; declarations/fields: `verify_trigger` |
| [I10](../../slices/identity-brokering.md#i10-identity-audit-history) / active | 640–667 | Credential-free immutable event append inside domain transaction; declarations/fields: `append_event` |
| [I02](../../slices/identity-brokering.md#i02-encrypted-credential-custody) / active | 668–704 | Provider credential maintenance under current exact lease; declarations/fields: `maintain_credential` |
| [I10](../../slices/identity-brokering.md#i10-identity-audit-history) / active | 705–711 | Connection change and lease audit event construction; declarations/fields: `changed` |
| [I10](../../slices/identity-brokering.md#i10-identity-audit-history) / active | 712–729 | Connection change and lease audit event construction; declaration/member lease_event; declarations/fields: `lease_event` |
| [H02](../../slices/shared-supporting-slices.md#h02-storage-mechanics) / active | 730–737 | PostgreSQL parameter formatting; declarations/fields: `Param` |
| [H02](../../slices/shared-supporting-slices.md#h02-storage-mechanics) / active | 738–741 | PostgreSQL parameter formatting; declaration/member text; declarations/fields: `text` |
| [H02](../../slices/shared-supporting-slices.md#h02-storage-mechanics) / active | 742–745 | PostgreSQL parameter formatting; declaration/member int; declarations/fields: `int` |
| [H02](../../slices/shared-supporting-slices.md#h02-storage-mechanics) / active | 746–749 | PostgreSQL parameter formatting; declaration/member boolean; declarations/fields: `boolean` |
| [H02](../../slices/shared-supporting-slices.md#h02-storage-mechanics) / active | 750–753 | PostgreSQL parameter formatting; declaration/member bytea; declarations/fields: `bytea` |
| [H02](../../slices/shared-supporting-slices.md#h02-storage-mechanics) / active | 754–774 | PostgreSQL parameter formatting; declaration/member encode; declarations/fields: `encode` |
| [H02](../../slices/shared-supporting-slices.md#h02-storage-mechanics) / active | 775–778 | SQL parameter/source assertions; declarations/fields: `tests` |
| [H02](../../slices/shared-supporting-slices.md#h02-storage-mechanics) / active | 779–795 | SQL parameter/source assertions; declaration/member bind_rewrites_placeholders_outside_literals; declarations/fields: `bind_rewrites_placeholders_outside_literals` |
| [H02](../../slices/shared-supporting-slices.md#h02-storage-mechanics) / active | 796–799 | SQL parameter/source assertions; declaration/member bytea_params_use_hex_text_form; declarations/fields: `bytea_params_use_hex_text_form` |


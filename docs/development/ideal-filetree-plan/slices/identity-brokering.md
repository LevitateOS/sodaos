# Identity brokering

[Slice catalog and evidence scope](README.md). The cards below span current
source owners; they do not create packages, services or an approved intended model.

## I01 Enrollment and owner consent

Start, observe, cancel and complete an explicit selected-provider enrollment with owner-bound metadata and exposure consent.

- **Entrypoints:** POST/GET/DELETE identity enrollments; Controller.start_enrollment/enrollment/cancel_enrollment.
- **Owned data:** Transient enrollment ID/session presentation and completion result; owner-bound connection metadata references.
- **Authority:** Verified native owner; explicit credential-exposure consent; selected provider only.
- **Dependencies:** [I02](#i02-encrypted-credential-custody); [I07](#i07-codex-adapter); [I08](#i08-muse-adapter); Native session authority.
- **Source files:** [internal/web/api/identity.go:78](../../../../internal/web/api/identity.go#L78); [rust/soda-identity/src/control.rs:166](../../../../rust/soda-identity/src/control.rs#L166); [rust/soda-identity/src/main.rs:245](../../../../rust/soda-identity/src/main.rs#L245).
- **Tests:** [internal/web/identity_native_test.go:46](../../../../internal/web/identity_native_test.go#L46) — Source-only API fixture assertions: owner/session binding and consent required before broker call; changed actor/generation/origin refused.
- **Unclear boundaries:** Owns enrollment lifecycle, while I02 owns retained encrypted credential bytes and generations. Existing providers are Codex and Muse; current registry is not authority to invent additional providers or universal enrollment semantics.
- **Evidence status:** Representative source catalog at committed 26d420f2; assertions inspected only, no tests or native execution performed.

## I02 Encrypted credential custody

Retain native credential bytes encrypted with connection/generation binding under the dedicated broker's private custody.

- **Entrypoints:** Store.open_encrypted/save_connection/credential; GrantCipher.seal/open; broker startup key input.
- **Owned data:** Encrypted connection credential ciphertext; connection credential generation; protected encryption key input; PostgreSQL persistence.
- **Authority:** Dedicated broker process; plaintext excluded from owner/browser metadata and ordinary application APIs.
- **Dependencies:** [I01](#i01-enrollment-and-owner-consent); [I06](#i06-completion-revocation-and-reconciliation); PostgreSQL; AES-GCM; Restricted key file.
- **Source files:** [rust/soda-identity/src/store.rs:61](../../../../rust/soda-identity/src/store.rs#L61); [rust/soda-identity/src/store.rs:224](../../../../rust/soda-identity/src/store.rs#L224); [rust/soda-identity/src/crypto.rs:25](../../../../rust/soda-identity/src/crypto.rs#L25).
- **Tests:** [rust/soda-identity/src/crypto.rs:84](../../../../rust/soda-identity/src/crypto.rs#L84) — Source-only unit assertions: authenticated round-trip; wrong binding, short ciphertext and tampering refused; [rust/soda-identity/tests/broker.rs:205](../../../../rust/soda-identity/tests/broker.rs#L205) — Source-only PostgreSQL broker fixture: retained credential round-trip; fixture may return early when unavailable.
- **Unclear boundaries:** Owns ciphertext and generation, not grants or native launch authorization. Current implementation uses PostgreSQL; credentials guide still names SQLite in runtime details, so documentary storage wording needs reconciliation. No encryption or database test executed.
- **Evidence status:** Representative source catalog at committed 26d420f2; assertions inspected only, no tests or native execution performed.

## I03 Delegation and connection availability

Authorize explicit subscription sharing to a named member/Project and expose only the connection metadata available to that actor.

- **Entrypoints:** GET available-connections; POST identity grants; Controller.available/create_grant/revoke_grant.
- **Owned data:** Grant ID, named user/Project, revision and revocation; derived availability metadata.
- **Authority:** Connection owner consent plus current requester/recipient Project membership and repository rights.
- **Dependencies:** [I02](#i02-encrypted-credential-custody); [P03](projects.md#p03-human-membership-and-accounts); Native Forgejo repository callback; [I06](#i06-completion-revocation-and-reconciliation).
- **Source files:** [internal/web/api/identity_grants.go:47](../../../../internal/web/api/identity_grants.go#L47); [rust/soda-identity/src/control.rs:159](../../../../rust/soda-identity/src/control.rs#L159); [rust/soda-identity/src/control.rs:279](../../../../rust/soda-identity/src/control.rs#L279).
- **Tests:** [internal/web/identity_native_test.go:92](../../../../internal/web/identity_native_test.go#L92) — Source-only API fixture: named members, write authority and both confirmations required; [rust/soda-identity/tests/broker.rs:205](../../../../rust/soda-identity/tests/broker.rs#L205) — Source-only broker fixture: foreign actor sees nothing without grant; shared metadata omits email; revoke removes availability.
- **Unclear boundaries:** Owns delegation state, not custody or execution reservations. Availability is derived, not evidence that execution is currently admissible. I06 performs grant revocation and resulting retirement against these owned grants.
- **Evidence status:** Representative source catalog at committed 26d420f2; assertions inspected only, no tests or native execution performed.

## I04 Execution admission and lease fencing

Admit one logical execution against current connection/grant authority and fence replays, conflicting reservations and late registration.

- **Entrypoints:** Runtime acquire/get_execution/close_execution; Controller.acquire/admit_execution/reserve.
- **Owned data:** Execution identity/digest/state/tombstone; lease reservation/deadline and captured connection/grant generations.
- **Authority:** Runtime-only broker API; authenticated native execution consumer; current grant and provider-specific concurrency constraints.
- **Dependencies:** [I02](#i02-encrypted-credential-custody); [I03](#i03-delegation-and-connection-availability); [I05](#i05-native-binding-and-private-delivery); [I06](#i06-completion-revocation-and-reconciliation); PostgreSQL.
- **Source files:** [rust/soda-identity/src/control.rs:317](../../../../rust/soda-identity/src/control.rs#L317); [rust/soda-identity/src/control.rs:439](../../../../rust/soda-identity/src/control.rs#L439); [rust/soda-identity/src/store.rs:435](../../../../rust/soda-identity/src/store.rs#L435).
- **Tests:** [rust/soda-identity/tests/broker.rs:239](../../../../rust/soda-identity/tests/broker.rs#L239) — Source-only PostgreSQL broker fixture: identical acquisition replays same lease; [rust/soda-identity/tests/broker.rs:359](../../../../rust/soda-identity/tests/broker.rs#L359) — Source-only broker fixture: closed execution denies late register and reacquire; fixture may skip without database.
- **Unclear boundaries:** Owns execution/reservation records. I05 supplies the attested binding; I06 closes the same records. Codex serialization and Muse concurrency are existing policies, not a universal-provider rule or new generic scheduler.
- **Evidence status:** Representative source catalog at committed 26d420f2; assertions inspected only, no tests or native execution performed.

## I05 Native binding and private delivery

Validate the exact native account/container/process/unit binding before credentials cross the private runtime interface.

- **Entrypoints:** Runtime register; Controller.register; host validate callbacks; Codex/Muse native adapter validation.
- **Owned data:** Attested lease binding fields; private delivery payload and protected native credential destination.
- **Authority:** Dedicated broker runtime plane; privileged host verifies native incarnation/invocation; browser input cannot attest authority.
- **Dependencies:** [I04](#i04-execution-admission-and-lease-fencing); [I02](#i02-encrypted-credential-custody); [I09](#i09-provider-execution-integration); [P02](projects.md#p02-profile-and-runtime-readiness); [P03](projects.md#p03-human-membership-and-accounts); [P06](projects.md#p06-factory-role-accounts); systemd; Podman.
- **Source files:** [rust/soda-identity/src/control.rs:502](../../../../rust/soda-identity/src/control.rs#L502); [rust/soda-identity/src/runtime.rs](../../../../rust/soda-identity/src/runtime.rs); [rust/soda-host/src/tcodex.rs](../../../../rust/soda-host/src/tcodex.rs); [docs/reference/credentials.md:112](../../../reference/credentials.md#L112).
- **Tests:** [rust/soda-host/tests/tcodex_ops_oracle.rs:206](../../../../rust/soda-host/tests/tcodex_ops_oracle.rs#L206) — Oracle source assertions with fake executor: exact container, role UID/GID and systemd invocation checks precede delivery; [rust/soda-host/tests/muse_serve_oracle.rs:426](../../../../rust/soda-host/tests/muse_serve_oracle.rs#L426) — Oracle source assertions: mismatched invocation returns stale and stops further native calls.
- **Unclear boundaries:** Owns binding attestation/delivery transition, not lease reservation or credential storage. Shared lease table is one state machine, not duplicate ownership. Native oracle tests prove authored call contracts only; installed attestation was not exercised.
- **Evidence status:** Representative source catalog at committed 26d420f2; assertions inspected only, no tests or native execution performed.

## I06 Completion, revocation and reconciliation

Retire leases and logical executions through verified stop/finish/return, explicit revocation and deadline reconciliation without prematurely releasing custody.

- **Entrypoints:** Controller.return_lease/end_lease/revoke/revoke_grant/reconcile/sweep; host stop/finish callbacks.
- **Owned data:** Lease terminal/returned state; execution closure; revocation transitions; custody-return result and updated generation.
- **Authority:** Owner may revoke own connection/grant; native runtime confirms retirement; unknown termination cannot imply successful return.
- **Dependencies:** [I02](#i02-encrypted-credential-custody); [I03](#i03-delegation-and-connection-availability); [I04](#i04-execution-admission-and-lease-fencing); [I05](#i05-native-binding-and-private-delivery); [I07](#i07-codex-adapter); [I08](#i08-muse-adapter); [I09](#i09-provider-execution-integration).
- **Source files:** [rust/soda-identity/src/control.rs:610](../../../../rust/soda-identity/src/control.rs#L610); [rust/soda-identity/src/control.rs:705](../../../../rust/soda-identity/src/control.rs#L705).
- **Tests:** [rust/soda-identity/tests/broker.rs:390](../../../../rust/soda-identity/tests/broker.rs#L390) — Source-only PostgreSQL broker fixture: revoke changes connection state and retires live leases; [rust/soda-identity/tests/broker.rs:306](../../../../rust/soda-identity/tests/broker.rs#L306) — Source-only broker fixture: Muse return forgets lease without rotating connection generation.
- **Unclear boundaries:** Owns closure transitions over I04's lease/execution records, not a second ledger. Mutable Codex return and immutable Muse retirement already diverge inside core code; review those concrete policies before extracting abstractions.
- **Evidence status:** Representative source catalog at committed 26d420f2; assertions inspected only, no tests or native execution performed.

## I07 Codex adapter

Adapt the pinned native Codex enrollment protocol and mutable subscription credentials to the broker's existing provider interface.

- **Entrypoints:** Codex Provider.new/start; Session.snapshot/finish/close; native device enrollment subprocess.
- **Owned data:** Transient private Codex enrollment root/process; native auth bytes; provider-specific public enrollment presentation.
- **Authority:** Explicit selected Codex owner; configured pinned binary/version/hash; inherited credential environment filtered.
- **Dependencies:** [I01](#i01-enrollment-and-owner-consent); [I02](#i02-encrypted-credential-custody); [I06](#i06-completion-revocation-and-reconciliation); Native Codex executable.
- **Source files:** [rust/identity-providers/src/codex.rs:50](../../../../rust/identity-providers/src/codex.rs#L50); [rust/identity-providers/src/codex.rs:304](../../../../rust/identity-providers/src/codex.rs#L304); [docs/reference/credentials.md:109](../../../reference/credentials.md#L109).
- **Tests:** [rust/identity-providers/src/codex.rs:656](../../../../rust/identity-providers/src/codex.rs#L656) — Source-only native-protocol fixture assertions: credential retention after process stop; close removes private enrollment root; [rust/identity-providers/src/codex.rs:682](../../../../rust/identity-providers/src/codex.rs#L682) — Source-only fixture assertions: unfinished enrollment canceled and root removed.
- **Unclear boundaries:** Adapter owns Codex protocol/auth representation; core owns grants, custody and lease records. Execution-side integration is I09. Browser launch's current Codex scope does not redefine the entire broker as Codex-only.
- **Evidence status:** Representative source catalog at committed 26d420f2; assertions inspected only, no tests or native execution performed.

## I08 Muse adapter

Adapt pinned Muse subscription device enrollment and native auth validation without exposing credential or diagnostic contents in presentation.

- **Entrypoints:** Muse Provider.new/start; Session.finish/close; credential_valid; native login subprocess.
- **Owned data:** Transient Muse enrollment root/process; native subscription auth representation and bounded public device presentation.
- **Authority:** Explicit selected Muse owner; pinned native binary/version/hash; subscription routing rather than fallback billing.
- **Dependencies:** [I01](#i01-enrollment-and-owner-consent); [I02](#i02-encrypted-credential-custody); [I06](#i06-completion-revocation-and-reconciliation); Native Muse executable.
- **Source files:** [rust/identity-providers/src/muse.rs:44](../../../../rust/identity-providers/src/muse.rs#L44); [rust/identity-providers/src/muse.rs:286](../../../../rust/identity-providers/src/muse.rs#L286); [docs/reference/credentials.md:134](../../../reference/credentials.md#L134).
- **Tests:** [rust/identity-providers/src/muse.rs:514](../../../../rust/identity-providers/src/muse.rs#L514) — Source-only native fixture assertions: file credential backend, environment filtering, completed enrollment and no credentials/diagnostics in public presentation; [rust/identity-providers/src/muse.rs:494](../../../../rust/identity-providers/src/muse.rs#L494) — Source-only pinned-version fixture assertions: exact native version required.
- **Unclear boundaries:** Adapter owns Muse enrollment/auth semantics, while concurrency and immutable lease closure belong to I04/I06. This is an existing adapter child, not evidence that Codex assumptions can be copied to every provider.
- **Evidence status:** Representative source catalog at committed 26d420f2; assertions inspected only, no tests or native execution performed.

## I09 Provider execution integration

Connect human terminal, factory and explicitly registered nested native execution consumers to broker admission and attested private delivery.

- **Entrypoints:** Codex identity launch; host factory Codex/Muse adapters; soda-muse Project launch; soda-identity-compose register.
- **Owned data:** Native execution unit/process metadata and private runtime files; consumer-side execution binding; no separate connection/grant ledger.
- **Authority:** Native actor/account or fixed factory role; exact invocation; nested registration explicitly opted in.
- **Dependencies:** [I04](#i04-execution-admission-and-lease-fencing); [I05](#i05-native-binding-and-private-delivery); [I06](#i06-completion-revocation-and-reconciliation); [I07](#i07-codex-adapter); [I08](#i08-muse-adapter); [P02](projects.md#p02-profile-and-runtime-readiness); [P03](projects.md#p03-human-membership-and-accounts); [P06](projects.md#p06-factory-role-accounts); [P07](projects.md#p07-checkout-allocation-and-preparation); [S04](spaces-and-terminals.md#s04-human-terminal-lifecycle); systemd; Podman.
- **Source files:** [internal/web/api/identity_launch.go:13](../../../../internal/web/api/identity_launch.go#L13); [rust/soda-host/src/muse.rs:1523](../../../../rust/soda-host/src/muse.rs#L1523); [rust/soda-host/src/tmuse.rs:275](../../../../rust/soda-host/src/tmuse.rs#L275); [rust/soda-identity-compose/src/main.rs:27](../../../../rust/soda-identity-compose/src/main.rs#L27); [rust/soda-host/src/pfactory.rs:2035](../../../../rust/soda-host/src/pfactory.rs#L2035).
- **Tests:** [rust/soda-host/tests/muse_serve_oracle.rs:416](../../../../rust/soda-host/tests/muse_serve_oracle.rs#L416) — Oracle source assertions with fake executor/hooks: delivery echo after valid native invocation; stale invocation refused; [rust/soda-host/tests/tcodex_ops_oracle.rs:206](../../../../rust/soda-host/tests/tcodex_ops_oracle.rs#L206) — Oracle source assertions: factory-native binding checks use exact role/container/unit; [rust/soda-host/src/pfactory.rs:4427](../../../../rust/soda-host/src/pfactory.rs#L4427) — Unit regression with fake broker/terminal: Muse harness sends one acquire request with provider_id=muse and the original execution ID. The scripted fixture still returns a Codex lease; completed phase does not establish real Muse authorization, reservation, delivery or retirement.
- **Unclear boundaries:** Dispatch supplies the selected harness and connection; host execution owns provider-matched lease acquisition and native reservation/delivery. Broker core remains the sole lease/custody owner. Current harness names also identify broker providers; that name coupling needs intended-model review for the broader provider catalog. Reservation refusal now logs its cause before the existing abandonment path; native Muse execution remains unqualified here.
- **Evidence status:** Request-provider regression and host launch source inspected at committed 26d420f2; no tests or native execution performed.

## I10 Identity audit history

Append credential-free identity events atomically with broker state changes and preserve their immutable owner/connection history.

- **Entrypoints:** Broker/store mutations append audit events transactionally; Store.events bounded history reader has no current Rust caller or exposed route found.
- **Owned data:** identity_events append-only rows with actor, owner, connection, generation, action and time; no retained credential contents.
- **Authority:** Broker mutation authority supplies event identity; database triggers refuse UPDATE/DELETE; history read intent is owner/connection scoped in storage code.
- **Dependencies:** [I01](#i01-enrollment-and-owner-consent); [I02](#i02-encrypted-credential-custody); [I03](#i03-delegation-and-connection-availability); [I04](#i04-execution-admission-and-lease-fencing); [I06](#i06-completion-revocation-and-reconciliation); [H02](shared-supporting-slices.md#h02-storage-mechanics).
- **Source files:** [rust/soda-identity/src/schema.rs:58](../../../../rust/soda-identity/src/schema.rs#L58); [rust/soda-identity/src/store.rs:556](../../../../rust/soda-identity/src/store.rs#L556); [rust/soda-identity/src/store.rs:640](../../../../rust/soda-identity/src/store.rs#L640); [rust/soda-identity/src/wire.rs:531](../../../../rust/soda-identity/src/wire.rs#L531).
- **Tests:** [internal/store/identity_test.go:79](../../../../internal/store/identity_test.go#L79) — Go PostgreSQL shared-schema assertions: ordered audit actions, immutable trigger and credential absence; not direct Rust runtime proof; [internal/store/identity_test.go:101](../../../../internal/store/identity_test.go#L101) — Go PostgreSQL fixture: failed audit INSERT rolls back grant mutation; tests not run.
- **Unclear boundaries:** Audit history has its own state and append integrity; it does not acquire credential custody, grant authorization or retention policy from neighboring slices. The unused Rust history reader is an unresolved applicability seam; no public audit feature is inferred. Shared-schema/predecessor Go tests require separate port/caller classification.
- **Evidence status:** Existing append state mapped; read exposure, intended audit requirements and direct Rust proof remain unresolved.



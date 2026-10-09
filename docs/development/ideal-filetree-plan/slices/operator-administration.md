# Operator administration

[Slice catalog and evidence scope](README.md). The cards below span current
source owners; they do not create packages, services or an approved intended model.

## O01 First-boot database provisioning

Seed appliance database credentials and initialize the standard PostgreSQL roles before a Forgejo operator identity exists.

- **Entrypoints:** soda-pg-provision.service -> soda-setup --provision-db-only; soda-pg-init-roles.
- **Owned data:** /etc/soda/installed admission marker; /etc/soda/postgres/{super,forgejo,soda}.passwd and soda.dsn; PostgreSQL roles/databases.
- **Authority:** Native root for setup; installed-host service conditions; Existing complete credentials are adopted; incomplete/preexisting state is preserved or refused.
- **Dependencies:** [D11](release-and-installation.md#d11-host-installation-and-payload-application); PostgreSQL; Podman/systemd.
- **Source files:** `appliance/services/soda-pg-provision.service:3 (historical source locator)`; `rust/soda-setup/src/main.rs:79 (historical source locator)`; `rust/soda-setup/src/main.rs:992 (historical source locator)`; `rust/soda-pg-maintenance/src/bin/soda-pg-init-roles.rs:28 (historical source locator)`.
- **Tests:** `rust/soda-setup/src/main.rs:1332 (historical source locator)` — Source fixture: password/DSN shape and file modes, not native PostgreSQL initialization; `rust/soda-pg-maintenance/src/lib.rs:95 (historical source locator)` — Unit: generated role/database SQL shape; [scripts/pg_backup_test.go:209](../../../../scripts/pg_backup_test.go#L209) — Native Rust-command/PostgreSQL fixture driver: role/database provisioning, idempotent repeat and quoted password handling; inspected only, not run.
- **Unclear boundaries:** Database boot prerequisites precede O02. O03 owns later operator maintenance; this is not a password-rotation path.
- **Evidence status:** Current source/service path; tests inspected, not executed.

- **Validity review:** [O01 audit record](../reviews/O01.md).

## O02 Operator identity bootstrap

Bind the installed appliance to its native Forgejo operator and create initial protected application configuration.

- **Entrypoints:** soda-setup --forgejo-url ... --token-file ; Forgejo /api/v1/user and bootstrap-token revocation.
- **Owned data:** /etc/soda/dashboard.json; Stable operator_id; /etc/soda/grant-key; Transient restricted bootstrap token file.
- **Authority:** Native root plus an administrator-owned read:user token; Fresh configuration only; bootstrap token revoked after successful configuration publication.
- **Dependencies:** [O01](#o01-first-boot-database-provisioning); [N01](networking.md#n01-private-origins-tls-and-activation); Native Forgejo API.
- **Source files:** [docs/guides/operator-setup.md:19 (historical line locator)](../../../guides/operator-setup.md); `rust/soda-setup/src/main.rs:1099 (historical source locator)`; [internal/web/api/operator.go:12](../../../../internal/web/api/operator.go#L12).
- **Tests:** `rust/soda-setup/src/main.rs:1462 (historical source locator)` — Source fixture matrix: restricted credentials, existing configuration, preserved files and failed/successful setup; successful assertions include operator_id and key/config modes.
- **Unclear boundaries:** Private origin/TLS activation is N01 despite shared setup/activation files. This initial application grant key is not evidence that bootstrap configures the separate identity-broker database/key.
- **Evidence status:** Current bootstrap path; broker commissioning seam remains explicit; no execution claim.

- **Validity review:** [O02 audit record](../reviews/O02.md).

## O03 Existing-install credential maintenance

Maintain protected appliance service credential configuration against existing live state, including removal of legacy inline database passwords before native Forgejo starts.

- **Entrypoints:** soda-forgejo-migrate.service -> soda-forgejo-migrate before forgejo.service; Explicit operator maintenance procedures; no general credential-rotation CLI identified.
- **Owned data:** Existing operator-controlled service configuration and secret files; Preserved live database/service state; Existing Forgejo app.ini [database] PASSWD entries; configured PASSWD_URI and other sections/file permissions are preserved by the current migration path.
- **Authority:** Installed native service/operator filesystem authority over existing configuration; Provider credential rotation/revocation and application commissioning have their own decisions and owners.
- **Dependencies:** [O01](#o01-first-boot-database-provisioning); [O02](#o02-operator-identity-bootstrap); [I02](identity-brokering.md#i02-encrypted-credential-custody); [I06](identity-brokering.md#i06-completion-revocation-and-reconciliation); Native Forgejo/PostgreSQL maintenance.
- **Source files:** [docs/reference/credentials.md:43 (historical line locator)](../../../reference/credentials.md); `rust/soda-setup/src/main.rs:965 (historical source locator)`; `rust/soda-forgejo-migrate/src/main.rs:22 (historical source locator)`; `appliance/services/soda-forgejo-migrate.service:9 (historical source locator)`.
- **Tests:** `rust/soda-setup/src/main.rs:1417 (historical source locator)` — Source fixture: initial setup reuses complete existing PostgreSQL secrets unchanged; preservation guard only, not rotation coverage; `rust/soda-forgejo-migrate/src/main.rs:151 (historical source locator)` — Password-assignment shape vectors; `rust/soda-forgejo-migrate/src/main.rs:167 (historical source locator)` — Byte record/trailing-newline splitting vectors. These do not assert section-preserving migration, secret rotation or installed service behavior.
- **Unclear boundaries:** The one-shot database configuration repair is shipped and selected, so its legacy input does not make the Rust migration dead code. General secret rotation/deletion remains unresolved; runtime provider enrollment/custody/revocation stays I01/I02/I06. Initial setup is O01/O02.
- **Evidence status:** Current shipped maintenance entrypoint plus documented operator responsibility; general rotation and installed execution remain unverified.

- **Validity review:** [O03 audit record](../reviews/O03.md).

## O04 Native host administration and updates

Keep root host diagnosis, service maintenance and native update ownership distinct from product controls; provide bounded whole-domain quiescence for Fountain native-operation recovery.

- **Entrypoints:** Stock Cockpit root administration; Fedora CoreOS rpm-ostree/Zincati; soda-forgejo-domain stop|inhibit|status|lift|start; Fountain admin native-operation reconciliation.
- **Owned data:** Host journals, systemd units and deployment state; Runtime forgejo.service mask; nativeop-offline marker under the configured Forgejo AppDataPath; Native Forgejo console access-log fields/configuration; request query/referrer content is excluded by the selected template.
- **Authority:** Native host operator/root; Fountain owns native-operation reconciliation; helper has no force-unlock verb.
- **Dependencies:** [N01](networking.md#n01-private-origins-tls-and-activation); Fedora CoreOS; systemd/Podman; Fountain.
- **Source files:** [docs/development/cockpit.md:12 (historical line locator)](../../cockpit.md); [docs/architecture/release.md:24 (historical line locator)](../../../architecture/release.md); `rust/soda-forgejo-domain/src/main.rs:1 (historical source locator)`; [docs/development/cockpit.md:21 (historical line locator)](../../cockpit.md); `appliance/config/forgejo.env:15 (historical source locator)`.
- **Tests:** [tests/build/forgejo_domain_test.go:39](../../../../tests/build/forgejo_domain_test.go#L39) — CLI source test: nonroot refusal before effects; `rust/soda-forgejo-domain/src/main.rs:895 (historical source locator)` — Unit with fake system operations: inhibition requires quiescence before masking/marking; [scripts/sodaspaces_templates_test.go:58](../../../../scripts/sodaspaces_templates_test.go#L58) — Configuration/template assertion: method, escaped path and status only, with query/referrer markers excluded; no installed log retrieval/retention/visibility proof.
- **Unclear boundaries:** No parallel Soda host updater or general recovery engine. Disk installation is D11; Tailnet/Git-advertisement helper is Networking, not soda-forgejo-domain. Native journal diagnostics and current service logging configuration share operator administration; desired retention, visibility and installed behavior remain unresolved. This does not add a Soda log store or log service.
- **Evidence status:** Existing native administration/helper source; actual update/recovery qualification not established by these tests.

- **Validity review:** [O04 audit record](../reviews/O04.md).

## O05 Database backup and retention

Create verified, timestamped PostgreSQL backup runs and retain the configured number of completed runs.

- **Entrypoints:** soda-postgres-backup.timer/service; soda-pg-backup.
- **Owned data:** /var/lib/soda/backups/postgres; New .in-progress run -> completed timestamp directory; globals.sql and forgejo/soda custom-format dumps.
- **Authority:** Operator-controlled container-local PostgreSQL peer access; New backup publication precedes rotation; retention does not delete the newly published run.
- **Dependencies:** [O01](#o01-first-boot-database-provisioning); PostgreSQL pg_dump/pg_dumpall; Podman; systemd timer.
- **Source files:** `rust/soda-pg-maintenance/src/bin/soda-pg-backup.rs:26 (historical source locator)`; `rust/soda-pg-maintenance/src/bin/soda-pg-backup.rs:59 (historical source locator)`; `appliance/services/soda-postgres-backup.timer:6 (historical source locator)`.
- **Tests:** `rust/soda-pg-maintenance/src/bin/soda-pg-backup.rs:181 (historical source locator)` — Unit: completed-run filename recognition; not backup/rotation execution against PostgreSQL; [scripts/pg_backup_test.go:110](../../../../scripts/pg_backup_test.go#L110) — Native Rust-command/PostgreSQL fixture driver: backup output, retained run publication and rotation assertions; inspected only, not run.
- **Unclear boundaries:** Each database has its own consistent snapshot. This implementation does not establish whole-appliance, project-files or separate broker-state backup coverage.
- **Evidence status:** Current command/schedule and native fixture driver source inspected; no backup/restore execution or installed qualification performed.

- **Validity review:** [O05 audit record](../reviews/O05.md).

## O06 Database restore

Restore selected appliance databases, or explicitly restore globals on a fresh cluster, from an existing backup run.

- **Entrypoints:** soda-pg-restore --yes <backup-dir> [db ...]; soda-pg-restore --yes --globals <backup-dir>.
- **Owned data:** Existing globals.sql/*.dump inputs; Target PostgreSQL roles/databases; Temporary container restore staging files.
- **Authority:** Explicit --yes before mutation and operator-controlled Podman access; Caller must stop Forgejo/soda-host; globals restoration is documented for a fresh cluster.
- **Dependencies:** [O05](#o05-database-backup-and-retention); [O04](#o04-native-host-administration-and-updates); PostgreSQL psql/createdb/pg_restore; Podman.
- **Source files:** `rust/soda-pg-maintenance/src/bin/soda-pg-restore.rs:1 (historical source locator)`; `rust/soda-pg-maintenance/src/bin/soda-pg-restore.rs:26 (historical source locator)`.
- **Tests:** `rust/soda-pg-maintenance/src/lib.rs:79 (historical source locator)` — Unit: database-name allow/refusal cases; `rust/soda-pg-maintenance/src/lib.rs:89 (historical source locator)` — Unit: SQL literal quoting; neither test executes restoration; [scripts/pg_backup_test.go:175](../../../../scripts/pg_backup_test.go#L175) — Native Rust-command/PostgreSQL fixture driver: restore --yes recovers exact rows; 259–274 covers missing-confirmation and backup-retention refusals; inspected only, not run.
- **Unclear boundaries:** Separate from nondestructive backup creation. Service stopping is a documented caller precondition, not demonstrated automatic quiescence in this command.
- **Evidence status:** Current command and native round-trip/refusal fixture source inspected; globals-only, operational quiescence and installed qualification remain unverified.

- **Validity review:** [O06 audit record](../reviews/O06.md).

## O07 Operator SSH enrollment

Import one operator public SSH key through an explicitly armed, bounded installed-host enrollment window.

- **Entrypoints:** soda-install enroll-key; internal enrollment-serve/enrollment-receive actions; temporary native socket/receiver units.
- **Owned data:** Armed selected interface/address and boot-clock expiry; temporary socket/unit/sshd configuration; root authorized_keys; bounded submitted public key.
- **Authority:** Operator explicitly arms the installed-host window; native root password/forced-command and receiver/peer provenance checks govern key receipt.
- **Dependencies:** [D11](release-and-installation.md#d11-host-installation-and-payload-application); [O04](#o04-native-host-administration-and-updates); [H04](shared-supporting-slices.md#h04-configuration-and-filesystem-primitives); Native OpenSSH/systemd/SELinux.
- **Source files:** `rust/soda-install/src/run.rs:111 (historical source locator)`; `rust/soda-install/src/enroll/mod.rs:19 (historical source locator)`; [docs/operator/enroll-key.md:6 (historical line locator)](../../../operator/enroll-key.md).
- **Tests:** `rust/soda-install/src/enroll/mod.rs:357 (historical source locator)` — Bounded public-key input matrix; `rust/soda-install/src/enroll/mod.rs:415 (historical source locator)` — Native authentication/forced-command configuration assertions; `rust/soda-install/src/enroll/mod.rs:461 (historical source locator)` — Temporary native socket ownership assertions; `rust/soda-install/src/enroll/mod.rs:496 (historical source locator)` — Peer service-provenance vectors; `rust/soda-install/src/enroll/keys.rs:512 (historical source locator)` — Existing authorized-key preservation fixture. These are inspected assertions, not executed installed enrollment.
- **Unclear boundaries:** Installed operator/root SSH enrollment has its own authority, temporary listener and expiry/cleanup lifecycle. It is distinct from Project developer keys (P04), Forgejo operator identity (O02) and disk installation (D11).
- **Evidence status:** Existing enrollment implementation mapped as a separate candidate; no keys, services or host state changed.


- **Validity review:** [O07 audit record](../reviews/O07.md).

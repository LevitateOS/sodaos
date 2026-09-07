# Dashboard credential migration and rollback

**Rehearsed and executed for dashboard candidate `35df189` on the existing
`soda-test` guest; not a general installer or final cutover proof.** See
[exact execution evidence](implementation-status.md#native-react-preview-migration-and-operator-browser-proof).
U03/U04 own the schema/key
contract; U18 owns later default-SPA cutover. This is a bounded dashboard upgrade,
not first-install/bootstrap, an updater or a whole-appliance recovery system.
Require explicit target/deployment permission before executing any step.

## New installations

`soda-setup` now creates an exclusive `grant-key` file alongside OAuth credentials,
containing 32 random bytes encoded as standard base64. It writes
`grant_key_file` in the new config. `soda-activate` sets root:soda 0640 ownership
alongside existing credentials. No key is bundled into images or generated on
application startup. Setup/activation retain first-install refusal.

## Controlled existing-state rehearsal, before live deployment

1. Record the exact prior and candidate backend/frontend image/artifact identities.
   Preserve a matching prior artifact; do not rely on a mutable image tag.
   Inventory native consumers of the configuration too: `soda-runners` uses the
   same strict loader. A pre-grant-key binary rejects `grant_key_file`; back up
   and include its compatible candidate binary in an approved config migration.
   Do not weaken unknown-field validation or create a second runner config.
2. With permission, stop **only** the target's dashboard service to quiesce its
   writes. Do not stop/delete Forgejo or project containers. Back up the configured
   Soda database using SQLite's backup API, including any committed WAL data;
   copying the main `.db` file alone is not a valid backup. Use a new exclusive
   mode-0700 private directory and a mode-0600 backup destination. Verify the
   backup's integrity with SQLite before using it. Keep the source DB/WAL intact.
3. Preserve the matching dashboard config, OAuth client credentials, existing
   grant key if any, file owners/modes and prior application artifact in that
   same private backup set. Never put secrets in argv, terminal output, logs,
   browser captures or distributable artifacts. This backup does not need or
   authorize reading Forgejo's database or project private SSH keys.
4. On the controlled **copy**, supply a new exclusive restricted file containing
   standard-base64 encoding of 32 cryptographically random bytes if upgrading
   from v1/v2. Add its absolute path as `grant_key_file` in the copied config.
   For an encrypted v3 database, reuse its exact existing key—never generate a
   replacement. Place the final key under `/etc/soda` with root:soda 0640 and a
   restricted parent, as the native service recipe expects.
5. Validate the matching frontend bundle and config/key before opening the copied
   database through the candidate. Exercise preservation, missing/wrong key,
   grant binding, callback/consent, refresh/logout and rollback tests against
   controlled fixtures. No live schema change is authorized by this document.
6. Only after successful rehearsal and separately approved target rollout,
   install the matching dashboard artifacts/config/key and any affected native
   config-consumer binaries, preserving their owners/modes/SELinux labels; restart
   only the dashboard. Check runner `list` through its native root boundary without
   enrollment, registration or job execution. Replacing its CLI does not require
   restarting runner services. Do not rerun `soda-setup`, `soda-activate` or `install-native.sh`
   against the existing installation. Preserve the OAuth application/callback,
   service UID/socket and all project identities/state.

A maintenance tool must refuse occupied backup/key destinations and verify
ownership before mutation. A reusable dashboard-only deployment tool is still pending. Exact-target
private operator recipes executed the recorded `35df189` rehearsal/rollout;
the steps above do not establish deployment proof for other revisions/targets.

## Compatibility and rollback

Schema v3 adds `grant_key_check` and `session_grants`; v2 already appended an OAuth
column incompatible with the old binary's positional inserts. Never assume an
older binary can run against the upgraded DB, even if some queries still work.

Rollback means stopping the dashboard and restoring a **matching consistent
pre-change Soda DB/config/key/artifact set**, including correct ownership, through
an explicitly authorized procedure. Do not restore an old DB while retaining
new WAL files. Preserve the failed candidate's private state/evidence first.
Any post-backup Soda writes require an explicit preservation decision before
rollback; never silently discard newly registered keys, memberships or project
reservations. Do not revert unrelated Forgejo changes, replace project writable
roots or erase disks to make rollback succeed.

Retain the key for as long as its database or encrypted backups may be needed.
Missing/wrong key startup fails closed. Encryption protects a copied DB without
its key; it does not protect tokens from an already-compromised running dashboard.
The dashboard remains one process; multi-replica refresh coordination is not
implemented or implied.

## Native consent caveats

Forgejo 15.0.7 reuses confidential-client grants and does not return scopes in the
token response. Soda verifies actual scopes by the supported introspection
endpoint. For older `read:user` consent, users must revoke that native application
grant in Forgejo Applications settings and consent again for repository writes.
Administrators separately request administrator consent. This is not a request
to replace the OAuth application or use the bootstrap token for a denied user.

Native refresh counters belong to a user/application grant. Another session's
login or refresh can invalidate an older refresh token when upstream invalidation
is enabled. The affected Soda session reauthenticates; no credential sharing or
upstream setting change hides this limitation. Local sign-out deletes the local
grant only; native Git/Forgejo/SSH authority and access lifetimes remain separate.

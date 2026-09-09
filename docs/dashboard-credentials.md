# Dashboard credential migration and rollback

**Rehearsed and executed for dashboard candidate `35df189` on the existing
`soda-test` guest; not a general installer or final cutover proof.** See
[revision-specific evidence](implementation-status.md#accepted-native-evidence);
the complete historical migration record remains in Git at `9f3baa7`.
Both standalone frontends are now removed, but the Go API's schema/key contract
remains. Candidate `bdbce8e` subsequently passed fresh and copied retained-v3 → v5
native rehearsal, including rejection and paired rollback cases, in an isolated
networkless fixture. The user then separately approved retained cutover: a new matching backup preceded
live v3 → v5 migration and native callback/config/proxy delivery. Native browser and
existing-account access observations passed; see the [cutover handoff](implementation-status.md#approved-retained-cutover).
This is a bounded service upgrade, not bootstrap, an updater or appliance recovery.
Require explicit target/deployment permission before executing any step.

## New installations

`soda-setup` now creates an exclusive `grant-key` file alongside OAuth credentials,
containing 32 random bytes encoded as standard base64. It writes
`grant_key_file` in the new config. `soda-activate` sets root:soda 0640 ownership
alongside existing credentials. No key is bundled into images or generated on
application startup. Setup/activation retain first-install refusal.

## Controlled existing-state rehearsal, before live deployment

1. Record the exact prior and candidate application image/artifact identities.
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
5. Validate the matching backend/native payload and config/key before opening the copied
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

## Sodaspaces namespace transition

The current source removes `public_url` / `--public-url` and uses the unchanged
`forgejo_url` origin with `/-/soda/` API/login/callback routes. The strict loader
rejects old configuration; this is not an automatic migration or permission to
edit the retained VM. Routing commit `6deaf9a` left schema v3 unchanged; current
source appends v4/v5 as described below, with unchanged encrypted grant binding/key.
The integrated drawer and bounded native browser/access proof passed. Copied private
state rehearsal also passed; see the [handoff](implementation-status.md#phase-6-preserved-state-rehearsal).
Those proofs did not authorize silent cutover; separate approval was subsequently
obtained and the bounded retained-target transition passed.

For a later approved transition, include the matching backend and strict-config
consumers (notably `soda-runners`), copied configuration without `public_url`, Caddy
recipe and `proxy.env` using only `FORGEJO_ORIGIN` plus the existing private bind.
Preserve native Forgejo's origin/client identity, all credential files and project
state. Back up proxy configuration as well as the consistent Soda DB/config/key/
artifact set. Do not rerun first-install setup/activation on an existing target.

The actual OAuth application's owner must register
`FORGEJO_ORIGIN/-/soda/oauth/callback` through native Applications settings in the
approved transition sequence. That native edit preserves the client secret;
Forgejo 15.0.7's API PATCH regenerates it, so do not substitute a blind API update.
Preserve the prior callback/configuration for the explicitly reviewed transition
and rollback scope rather than deleting upstream state automatically.

New host-only Secure/HttpOnly/SameSite=Lax cookies use unique names and Path
`/-/soda/`. Old standalone cookies are ignored, not borrowed or automatically
expired across ports; users explicitly sign in again. Old pending browser flows
restart, with no unprefixed callback alias or rewriting of stored destinations.
Soda-only logout remains distinct from native Forgejo/SSH logout. Real browser/proxy
and copied populated-state rehearsal passed independently, followed by separately
approved retained transition and native browser/own-access observations.

## Schema v4 OAuth context

Version 4 appends `oauth.repository_id` and `oauth.expected_user_id`, each a
nonnegative integer defaulting to zero (absent). These are short-lived navigation/
identity-consistency hints, not foreign keys to Soda users/projects or a permission
inventory. The same atomic consume returns them with the PKCE verifier. Legacy
pending states retain their verifier/expiry and get no repository/expected-user
context; the old `return_path` column stays unused. Callback query parameters cannot
supply missing historical context or override stored IDs.

The existing grant key is checked **before** migration. Local tests exercise a
real v3-schema fixture containing profiles, keys, project/membership/session rows,
pending OAuth and encrypted grants. Wrong/missing keys leave version/columns
unchanged; the correct key preserves grant/key-check ciphertext and product records
while migrating. Those local tests are separate from the later successful native
`bdbce8e` rehearsal on copied private v3 installation data.

A pre-v4 binary rejects the newer schema version. Do not downgrade the marker or
assume the extra columns make mixed binaries safe. Rehearse a matching candidate
on authorized fresh/copied state, including context expiry/replay and rollback
preservation, before any live configuration/database change.

## Schema v5 login cancellation

The append-only migration adds internal login contexts and session/OAuth
references. Each existing Soda session gets its own context; user IDs, token hashes,
CSRF, expiry and encrypted grants/key-check bytes are preserved. Old pending OAuth
has no cancellation binding and must restart, including a v4 pending login. This
supersedes the v4-only pending-state compatibility above, not historical evidence.
Wrong/missing keys still fail before migration. Pre-v5 binaries reject schema v5;
rehearse matching DB/config/key/artifacts before separately authorized deployment.
The fresh fixture runs v5. Copied private v3 → v5 preserved every original column,
profile/key/project/membership/session/grant row and ciphertext, with integrity,
foreign keys and migrated contexts checked. The rehearsal did not mutate live retained data. The later separately approved
`soda-test` transition migrated v3 → v5, preserving all original rows/ciphertext before
login. Normal subsequent expiry/login/logout changed session/grant rows; original
profiles, keys, projects and memberships remained unchanged.

## Schema v6 Spaces return

The append-only migration adds `oauth.spaces_return`, default false, constrained
against simultaneous repository intent. Only `destination=spaces` selects it;
callback maps it to configured-origin `/-/soda/spaces`, never a caller URL. Existing
pending v5 transactions keep their original intent and cancellation binding.
Populated synthetic v5 migration tests preserve sessions, memberships, keys, projects,
and exact encrypted grant/key-check bytes; wrong keys fail before migration. Logout
still wins finalization. No OAuth client edit or new secret/consent is required.
Pre-v6 binaries reject schema v6. Matching backend/assets and backed-up state require
separately authorized delivery; these local tests are not a retained-state rollout.

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
endpoint. For an older insufficient grant, the acting user may need to explicitly revoke
that unique Soda application grant in native Applications settings and consent
again. Current Soda requests only read user/repository/organization scopes; the
retired admin form no longer requests administrator consent. Do not automatically
revoke grants, replace the OAuth application or use the bootstrap token for a denial.

Native refresh counters belong to a user/application grant. Another session's
login or refresh can invalidate an older refresh token when upstream invalidation
is enabled. The affected Soda session reauthenticates; no credential sharing or
upstream setting change hides this limitation. Local sign-out deletes the local
grant only; native Git/Forgejo/SSH authority and access lifetimes remain separate.

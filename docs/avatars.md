# Soda robot avatars

Soda's Go backend renders an original DiceBear style through Forgejo's supported
`GRAVATAR_SOURCE` configuration. No Forgejo, Bottts or DiceBear fork is involved.
The source and local artwork preview are implemented; live Forgejo activation
and appliance deployment require their separately scoped actions.

## Behavior and ownership

The immutable v1 style contains 44 component variants: six heads, six face panels,
ten eye sets, eight mouths, six earcap sets and eight accessories, including plain.
Eight backgrounds and four accent colors come from Soda's existing palette.
The editable source, provenance and preview command are in
[`internal/avatar/`](../internal/avatar/README.md).

Forgejo supplies its normalized email MD5 hash. Soda uses
`soda-robot-v1:<lowercase hash>` as the seed. Identical input and size produce
identical SVG bytes across requests and restarts. Changing the avatar email can
change the robot. Distinct hashes do not guarantee visually unique images.
Shipped v1 artwork/rendering changes require an explicit version decision;
snapshots detect accidental changes during dependency upgrades.

Uploaded photos retain native priority. Provider-backed user/commit-author
images use Soda robots rather than external Gravatar photos. Repository avatars,
ghost users and other native fallback cases keep upstream behavior. No avatar
files or database rows are deleted, migrated or synchronized. Photo upload and
deletion remain Forgejo's native controls; there is no Soda editor or reroll state.

## Public image contract

`GET` and `HEAD /-/soda/avatars/v1/{hash}` accept exactly 32 hexadecimal characters.
The optional `s` parameter is an integer from 1 through 1024; default 128. The
native `d=identicon` parameter is accepted without changing the output. Duplicate,
malformed and unknown options, including remote default-image URLs, return 400.
Unknown versions, extra path segments and encoded/noncanonical avatar paths
return 404. Unsupported methods return 405 with `Allow: GET, HEAD`.

Successful responses are `image/svg+xml` with a content-derived ETag,
`Cache-Control: public, max-age=86400`, conditional GET/HEAD support and the
backend's restrictive CSP/nosniff headers. Validation errors and rendering
failures are bounded plain text with `no-store`; render failures return 503.
The renderer uses embedded artwork/schema, no disk cache or network fetches.
It does not inspect cookies, credentials, account existence or either database.
The identifier is still an email hash, not a promise of email anonymization.

Caddy sends `/-/soda/avatars/*` on the configured Forgejo origin to the
existing backend at `127.0.0.1:8080`, preserving the path and dropping Cookie and
Authorization for that route. The rest of `/-/soda/*` also reaches Soda under the
same origin, retaining the credentials required by protected API/OAuth routes.
Other Forgejo routes still reach port 3000. Public image generation does not
establish or bypass the authenticated Sodaspaces session contract.

## First activation

The existing first-activation command derives this environment entry from the
validated `forgejo_url`, never the internal listener or unrelated Soda origin:

```text
FORGEJO__picture__GRAVATAR_SOURCE=https://<configured-forgejo-origin>/-/soda/avatars/v1/
```

Configure Forgejo's **native administration → configuration settings** to allow
provider avatars (`DisableGravatar=false`) and disable federated avatars
(`EnableFederatedAvatar=false`). Those settings are database-backed in Forgejo
15.0.7; old environment/app.ini entries are not a reliable override of saved
values. Soda does not write Forgejo's database or gain administrator authority.

`[server] OFFLINE_MODE=true` bypasses providers, including this local provider.
Leave explicit offline policy unchanged unless the operator separately selects
provider mode. No Internet connection is needed for Soda avatar generation once
the build's dependencies are present.

## Existing-instance activation and restoration

Prepare the candidate backend, Caddyfile and exact configuration diff before
requesting activation. Do not rerun first-install, `soda-setup`, or `soda-activate`
against an already activated appliance. Keep the matching prior artifact/config
set and native setting values in a restricted, new backup directory; preserve
all later writes and existing avatar storage.

The intended change set is:

1. Deliver the candidate backend containing the embedded style and its dependency
   notices. Keep the current backend configuration, grants, keys and database.
2. Deliver the avatar-only Caddy route and set the provider entry above to the
   actual existing Forgejo browser origin, retaining every other environment entry.
3. Apply the two native avatar settings through an authorized Forgejo administrator.
   Record the prior values. Do not silently change offline mode.
4. With target-specific service permission, restart the affected backend/Forgejo
   and reload or restart Caddy using the target's existing service mechanism.
   Check a synthetic hash URL before inspecting native pages.
5. Use an explicitly approved disposable avatar account to test photo upload and
   deletion. Inspect profile, contributor-list and discussion avatars, preserving
   unrelated fixture data. Confirm avatar requests stay on the configured origin.

For the retained development instance, the target is only
`sodaos-local-forgejo` at `http://localhost:3300`. It currently exposes Forgejo
directly, so applying a template reload alone cannot exercise this Caddy/backend
route. A local integration rehearsal must explicitly provide a loopback candidate
backend and proxy, name their ports/origin, preserve that container's data/mounts,
and approve the native configuration/restart and avatar-fixture changes together.
Do not put an unreachable appliance/private origin into the laptop preview.
Use [`scripts/screenshot.ts`](screenshot-capture.md) and its retained fixture
profile for real Forgejo captures; artwork catalogs are not native UI evidence.

To restore, first restore the previous provider URL/native setting values and
restart Forgejo as authorized; then restore the matching Caddy/backend artifact
and configuration set. No avatar-file deletion or database rollback is needed
for this feature. Do not use old whole-database backups to erase later writes.

## Checks and packaging

```sh
go test ./internal/avatar ./internal/web ./tools/soda-avatars ./internal/nativebuild
go test -race ./internal/avatar ./internal/web -run 'Avatar|Render|Definition|Versioned|Stable|Offline|InputBounds'
python3 -m unittest discover -s tests/build -p test_avatar_integration.py
SODA_CADDY_BINARY=/absolute/path/to/caddy python3 -m unittest discover -s tests/build -p test_avatar_integration.py
SODA_AVATAR_BROWSER_CHECK=1 go test ./internal/web -run TestAvatarBrowserRendering -v
go run ./tools/soda-avatars
```

The optional Caddy test adapts the real shipped Caddyfile, then uses test-owned
loopback listeners and synthetic upstreams. It verifies routing and header
isolation, not TLS or installed Forgejo behavior. Without the variable it reports
a skip. The optional browser check uses local Playwright and Chrome to compare
production HTTP/CSP image pixels with the same SVG served without CSP, using only
test-owned loopback requests. On macOS, use a real nonsymlinked `TMPDIR` for native bundle tests.

Artwork is embedded in the backend binary, so there is no new runtime asset path
or container. Native metadata collection includes
[`avatar-dependencies.txt`](../appliance/licenses/avatar-dependencies.txt); the
bundle verifier requires it for new bundles. Older bundles retain their matching
verifier. The preview tool and `.artifacts/` are not appliance payloads.

The notices contain actual license texts for the pinned DiceBear core/schema,
JSON Schema validator and Go text dependency. This focused addition does not
claim closure of the appliance's broader [licensing obligations](licensing.md).

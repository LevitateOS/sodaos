# Updating the VM with the latest front-end changes

**Legacy local VM procedure.** This page describes the older package-in-image
layout. For the current native-catalog appliance on `soda-metal` (`ssh soda`),
use [Forge development](forge-development.md).

How to ship front-end source changes from this repository to the running
site of the retained local VM `sodaos-local` at https://192.168.2.152/,
verified live. Written 2026-10-19 after the feed-restyle, providers-button,
header-removal and feed-icons deploys. This is a development-VM update
procedure, not a release path; instance identity, credentials and history
live in `~/VirtualMachines/sodaos-local/README.md` on the host. Worked
examples of every step are the (host-local, untracked) install scripts in
`~/vm-web-probe/build/install2.sh` … `install5.sh`; guest receipts live in
`/var/home/soda-*/` on the VM.

## The one-paragraph model

The site is Fountain (Forgejo fork) plus the Soda extension. There is no
"copy files to a web root" step. Almost every front-end change ships inside
**one derived Forgejo container image** that carries three payloads:
(1) the extension package `soda.tar.gz` + its `catalog.json` under
`/usr/share/soda/packages/`, (2) the same extension assets unpacked under
`/usr/share/soda/extension/assets/`, and (3) native branding files
(templates/CSS/JS) under `/usr/share/soda/forgejo/`. On boot, a systemd
drop-in seeds the package into the extension cache, Forgejo starts from that
image, and its extension manager installs the catalog package whose SHA256
differs from the installed one. So an update is always: build assets →
repack the package → rebuild the image from the *currently running* image →
point the two pins (container image + cache-seed) at it → restart → verify
in a real browser.

## Change surfaces: what to build for what you edited

| You edited | Build | Ships as |
| --- | --- | --- |
| `frontend/spaces/*.ts`, `frontend/spaces/*.css`, `frontend/tailnet/*` | Extension assets (§2) | `soda.tar.gz` assets + `/usr/share/soda/extension/assets/` |
| `assets/branding/forgejo/*.css` | Both: extension build compiles `components.css` into the package; the raw file is also a Forgejo public asset | Package assets **and** `public/assets/soda/forgejo/<name>.css` |
| `assets/branding/forgejo/*.ts` (native JS modules) | `bun scripts/build-forgejo.ts` (§2) | `public/assets/soda/forgejo/<name>.js` |
| `frontend/forgejo/templates/**` | Nothing (copied raw) | `templates/…` in the image, per `frontend/forgejo/payload.json` |
| Fountain Go code (sibling `forgejo-ext` checkout) or the dashboard (`cmd/soda-dashboard`) | Full Fountain/dashboard image builds | **Out of scope here** — much bigger; do not improvise it as a file copy |

The authoritative map from image paths to source files is
`frontend/forgejo/payload.json` (keys are paths under
`/usr/share/soda/forgejo/`, values are repo sources; `@build/forgejo-js/…`
means "output of `scripts/build-forgejo.ts`").

## 0. Preflight — derive the current pins, never copy old ones

Several sessions deploy to this VM and layer new drop-ins on top of old
ones. Values in old receipts (and in this document) go stale within hours.
Always read the live state first:

```sh
SSH='ssh -F /home/vince/VirtualMachines/sodaos-local/private/ssh.config sodaos-local'
curl --cacert ~/VirtualMachines/sodaos-local/appliance-root-ca.crt \
  -o /dev/null -w '%{http_code}\n' https://192.168.2.152/        # site up?
$SSH 'podman inspect soda-forgejo --format "{{.Image}}"'          # RUNNING image id = your FROM base
$SSH 'systemctl cat forgejo.service | grep -E "Image=|CATALOG" ' # which drop-in wins (highest number)
$SSH 'systemctl cat soda-extension-install.service | grep ExecStart='  # cache-seed pin + package sha
$SSH 'podman run --rm --pull=never --network=none --entrypoint /bin/sh <RUNNING_IMAGE> \
  -ec "cat /usr/share/soda/packages/catalog.json; \
       sha256sum /usr/share/soda/extension/backend /usr/share/soda/extension/extension.json /usr/share/soda/extension/run"'
```

As of writing (2026-10-19, evening) the winning layer is
`96-role-portraits.conf` in both drop-in dirs, running image
`6e4155f8…`, package `08703724…`, backend `4e384474…`. By the time you read
this they will be different — the commands above are the source of truth.

Also: the `sodaos` checkout is usually dirty with other work. That is
expected — preserve it, edit only your files, and expect the asset build to
include whatever the working tree contains. Before deploying, diff your
built CSS against the installed one to confirm the delta is really just
your change (§5 shows how).

## 1. Make and test the source change

From the repository root, run the focused checks for what you touched:

```sh
bun --bun node_modules/typescript/bin/tsc -p tsconfig.browser.json
bun --bun node_modules/typescript/bin/tsc -p tsconfig.tests.json
SODA_TAILNET_COMPONENT=1 bun test --isolate --timeout 60000 tests/frontend/<area>.test.ts
go test ./scripts/ -run <TestName> -count=1     # for template changes
```

## 2. Build the assets

Extension assets (Spaces/Feed/Tailnet JS + CSS). Output must be a **fresh**
directory; terminal assets come from the prepared vendor dir:

```sh
bun scripts/build-soda-extension.ts --package=soda \
  --out ~/vm-web-probe/build/sodaN \
  --terminal-assets .artifacts/browser-terminal/vendor
```

Track 2 (Fountain F1-F4 named repository routes: `/feed`, `/requests`,
`/settings/sodaos/…`) builds from the same source with `--track=2`. It uses
`system/containers/extension/extension.track2.json` and also writes it to the
output as `extension.json`; that manifest replaces the package's one, so it
only installs together with a Forgejo image on that Fountain (the pinned
loader rejects `routes`). Its new `repository-settings` page also needs the
`soda-extension` backend and the Soda dashboard built from the same commit.

Only the `soda` package is installed on this VM (there is no tailnet
extension directory). Do not merge tailnet assets in — a 2026-10-19-morning
recipe did that for an older manifest; it no longer applies.

Native JS modules (only if you edited `assets/branding/forgejo/*.ts` or
added one — new modules also need a `payload.json` entry):

```sh
bun scripts/build-forgejo.ts --out ~/vm-web-probe/build/forgejo-js
```

Assemble the asset tree and its hash list (paths relative to the installed
extension dir, e.g. `assets/sodaspaces-page.css`):

```sh
cd ~/vm-web-probe/build
mkdir packageN && cp -r sodaN/assets packageN/assets
(cd packageN && find assets -type f | sort | xargs sha256sum) > assetsN.sha256
tar -cf packageN.tar -C packageN . && sha256sum packageN.tar > packageN.sha256
```

## 3. Repack the extension package and rebuild the image (on the guest)

Do this on the guest so the new image derives offline from the running one
(`--pull=never --network=none`). Transfer `packageN.tar`, `packageN.sha256`,
`assetsN.sha256`, your install script, and any native payload files
(templates/CSS/JS) into a fresh guest task dir, e.g.
`/var/home/soda-<change-name>/`, then:

1. Verify `sha256sum -c packageN.sha256`, unpack to `build/`.
2. Extract the **current** package and catalog from the running image:
   `podman create` + `podman cp` of
   `/usr/share/soda/packages/{soda.tar.gz,catalog.json}`. Assert the tarball
   SHA256 equals the catalog's `sha256` and the cache-seed pin from §0 — if
   they disagree, someone else deployed in between; stop and re-derive.
3. Unpack it, record `sha256sum backend extension.json run`, then replace
   **only** `assets/` with your built assets. Backend, manifest and
   launcher must stay byte-identical (asserted again after install).
4. Repack with **`tar --format=ustar -czf`** — this is load-bearing.
   Fountain's `extractArchive` (sibling checkout, `forgejo-ext/modules/extensions/catalog.go`)
   accepts USTAR regular-file/directory entries only and fatals at startup
   on PAX/GNU extensions ("package archive uses an unsupported format").
   A GNU-tar default archive took the site down once; the rollback in §6
   fixed it. Entry order in the tar: `extension.json backend run assets`.
5. Compute the new SHA256 + size and rewrite `catalog.json` (`sed` the old
   `size`/`sha256` values; keep everything else identical). Note the
   tarball SHA changes on every repack (gzip header), so always recompute.
6. Build the replacement image:

   ```dockerfile
   FROM sha256:<RUNNING_IMAGE_FROM_§0>
   LABEL org.soda.presentation.patch="<change-name>"
   COPY --chown=0:0 soda-new.tar.gz /usr/share/soda/packages/soda.tar.gz
   COPY --chown=0:0 catalog.json    /usr/share/soda/packages/catalog.json
   RUN rm -rf /usr/share/soda/extension/assets
   COPY --chown=0:0 build/assets    /usr/share/soda/extension/assets
   # only the native payload files you changed, at their payload.json paths:
   COPY --chown=0:0 payload/extra_links.tmpl /usr/share/soda/forgejo/templates/custom/extra_links.tmpl
   COPY --chown=0:0 payload/components.css   /usr/share/soda/forgejo/public/assets/soda/forgejo/components.css
   ```

   `podman build --pull=never --network=none --iidfile forgejo-image.id -f Containerfile.forgejo .`

## 4. Deploy: repoint the pins and restart

The two pins that matter are the **highest-numbered** drop-ins (later files
override earlier ones; that is also the trap — see §7):

- `/etc/containers/systemd/forgejo.container.d/NN-*.conf` — `Image=` for
  Forgejo (plus `FORGEJO__extensions__CATALOG_PATH`).
- `/etc/systemd/system/soda-extension-install.service.d/NN-*.conf` —
  `ExecStart=` that seeds `.cache/packages/<package-sha>.tar.gz` from the
  image before Forgejo starts.

Either edit the current winning pair (backups first) or add your own
higher-numbered pair. Then:

```sh
# snapshot what must not move
podman inspect --format '{{.Name}} {{.Id}} {{.State.StartedAt}}' soda-p<projects…> > projects.before
sha256sum /var/lib/soda/forgejo/gitea/extensions/soda/{backend,extension.json,run} > preserved.before

systemctl stop forgejo.service
install -m 0644 <new confs> <drop-in dirs>
systemctl daemon-reload
systemctl restart soda-extension-install.service   # seeds the new package cache
systemctl start forgejo.service
systemctl is-active forgejo.service soda-extension-install.service
sha256sum -c preserved.before                      # backend unchanged
# projects.after must cmp-identical to projects.before
```

If the seed step fails, restore the backed-up confs, `daemon-reload`,
restart the seed and Forgejo, and stop — you are back on the old image.

## 5. Verify — in this order, every time

1. **Installed hashes, but not immediately.** The extension manager swaps
   assets shortly *after* Forgejo starts; the assets directory can be
   missing for a while (observed >60 s once). Load a page first, then:
   `cd /var/lib/soda/forgejo/gitea/extensions/soda && sha256sum -c <task-dir>/assetsN.sha256`
   — retry until all OK before concluding anything.
2. **Backend/manifest/run hashes** unchanged, **Project containers**
   unchanged (step 4 snapshots).
3. **Scope check:** your installed CSS diff vs the pre-deploy one should be
   exactly your rules. (Download the installed `sodaspaces-page.css` before
   deploying if you want this comparison.)
4. **Real browser.** `node ~/vm-web-probe/probe.mjs` logs in as
   `soda-operator` (password from `private/administrator-password`), walks
   12 routes, and reports HTTP status, console/page errors and mobile
   overflow; screenshots land in `~/vm-web-probe/out/`. Look at the
   screenshots — DOM-clean is not the same as correct. Then exercise the
   actual changed behavior through the UI (click the thing you changed),
   like `probe4.mjs`/`probe6.mjs` do for recent changes.
5. **Record it:** append a short section to
   `~/VirtualMachines/sodaos-local/README.md` (what changed, new
   image/package SHAs, guest task dir, rollback) and leave the same facts
   in `<task-dir>/deployment.txt` on the guest.

## 6. Rollback

Everything needed is the previous image id + previous confs (your backups):

```sh
# restore the two winning drop-ins to their .before copies (old image + old package sha)
systemctl daemon-reload
systemctl stop forgejo.service
systemctl restart soda-extension-install.service
systemctl start forgejo.service
```

If you deployed by moving the `localhost/soda-ux-forgejo:oct19` tag (older
style), retag the previous image id back to that tag instead. The database
is never touched by any step here; do not restore database backups for a
front-end rollback.

## 7. Traps learned the hard way

- **Shadowed drop-ins.** `90-*.conf` files look authoritative but lose to
  `94-`/`95-`/`96-*.conf`. The 90-series installer pin once pointed at an
  image with an *older extension backend* — deploying "by the old recipe"
  would have silently downgraded the backend and manifest. Always
  `systemctl cat` and identify the last `ExecStart=`/`Image=`.
- **Derive, don't copy.** Image ids and package SHAs in receipts rot within
  hours (another session retags/repins `oct19` frequently). Your `FROM`
  line must be the image `podman inspect soda-forgejo` reports *now*.
- **USTAR or death.** §3 step 4. GNU tar's default format bricks startup.
- **The old feed-scroll recipe is obsolete.** `.artifacts/feed-scroll-vm/install.sh`
  installs via `gitea extensions install` from the 90-series
  drop-in; that path is shadowed and its base image has the wrong backend.
  Use §3–§4 (or `install5.sh` in `~/vm-web-probe/build/`) instead.
- **`components.css` exists twice.** Raw under
  `/usr/share/soda/forgejo/public/assets/soda/forgejo/` for native pages,
  compiled inside the extension package for islands. Editing the source
  usually means shipping both (§2 table).
- **Cache-busting is manual.** Native CSS/JS URLs carry `?v=…` pins in
  `frontend/forgejo/templates/custom/header.tmpl` and in templates that load
  scripts. If you change a native CSS/JS file's contents, bump its `?v=`
  or browsers keep the old copy for hours.
- **Asset verification races the manager.** §5 step 1. A failed hash check
  in the first minute is not a failed deploy; a page load appears to be
  what settles it.
- **Never hand-copy files into the installed extension dir** to "fix" a
  failed install. The catalog package is the source of truth; a hand copy
  desynchronises the receipt and is overwritten (or rejected) next boot.
- **Secrets:** `private/administrator-password` stays out of commands,
  logs and screenshots. Probes read it from the file at runtime.
- **Git:** these deploys ship the dirty working tree on purpose and are not
  commits. Do not commit/push as part of a VM update unless asked.

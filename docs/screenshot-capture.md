# Screenshot capture brief — later real evidence

Adapted from `soda-os`'s handbook capture rules, not its old page list or release claims. Local preview captures are separate from installed-product evidence. Do not add broken image links or substitute generated UI, mockups or the Forgejo component sheet for an installed product screenshot.

## Quick local page screenshots

### Existing development fixture login

The local development container `sodaos-local-forgejo` at
`http://localhost:3300` has the user-authorized, non-admin `soda-screenshot`
fixture account. Its authenticated Chrome profile is already saved in ignored
`.local/screenshot-fixture-profile/`. Reuse it directly from the repository root:

```sh
bun scripts/screenshot.ts --profile .local/screenshot-fixture-profile \
  http://localhost:3300/ \
  http://localhost:3300/user/settings
```

If the session expires, log in through Forgejo's normal login form using that
same profile. The generated credential is retained privately in ignored
`.local/screenshot-fixture/create-output.txt`; automation can read it internally
to fill the form. Never print that file or expose the password in command
arguments, logs or screenshots. Select “Remember me” and close the browser
cleanly before capturing. This account and profile belong to this local
development instance and may not exist in another checkout or on another machine.

### Setup and manual login

Use the pinned Bun, installed Google Chrome and the root Playwright dependency.
Run `bun install --frozen-lockfile` from the repository root to prepare the workspace.
The helper uses a persistent browser context so mobile captures use the requested
CSS viewport rather than cropping a desktop window. See [tooling](typescript.md).

Log in once in its dedicated browser window, then press Enter in the terminal:

```sh
bun scripts/screenshot.ts --login http://localhost:3300/user/login
```

Choose “Remember me” if offered. The script keeps that browser profile in ignored
`.local/screenshot-profile/`; it contains session cookies and stays private.
It does not copy your normal browser's cookies or store a password in source.
If the session expires, run `--login` again.

Capture one or more URLs using that session:

```sh
bun scripts/screenshot.ts \
  http://localhost:3300/ \
  http://localhost:3300/user/settings

bun scripts/screenshot.ts --width 390 --height 844 \
  http://localhost:3300/explore/repos
```

Each run prints its PNG paths under a fresh `.artifacts/screenshots/capture-*`
directory. Files are numbered in URL order. These are viewport screenshots by default. Use `--full-page` to include offscreen
content while preserving the requested CSS viewport and record that mode in the
verification sidecar. Each capture starts from a fresh document so a fragment-only
URL still receives an actual HTTP response for verification. `--wait 3000` gives JavaScript three extra seconds to
settle. Inspect the result: a PNG can still show an expired login or an error page.
Use `--scroll-top` to inspect the header when native autofocus scrolls to a form
field. This scrolls after settling; it does not disable focus or change the page.

Keep the dedicated profile closed between runs. Use `--profile DIR` for a separate
guest/account profile, `--out DIR` for a new output directory, or `CHROME` for a
different Chrome executable. Run `--help` for defaults. Fixtures are added and
removed manually through Forgejo; the script has no fixture management.

### Reviewing CSS from an isolated worktree

The running preview may bind another checkout. `--local-css` serves this
checkout's Soda stylesheets to the capture browser in its `custom/header.tmpl`
order, including added or removed files. It accepts only `http://localhost:3300`
capture URLs and cannot be combined with `--login`:

```sh
bun scripts/screenshot.ts --local-css \
  --profile /absolute/path/to/existing/.local/screenshot-fixture-profile \
  --width 390 --height 844 http://localhost:3300/user/settings
```

This changes only the capture browser's CSS. Native server HTML, scripts, account
preferences and the live checkout remain unchanged. Label these as candidate CSS
captures; they do not prove that edited templates rendered on the server. If the
profile lives in another checkout, use its absolute `--profile` path. Install the
root workspace dependencies in the checkout running the script; no profile copy
or cross-workspace dependency lookup is needed.

### Generated scripts in the local preview

Run `bun run build:preview` after browser TypeScript or branding changes. It emits
minified scripts and projects the production branding payload into
`.artifacts/forgejo-preview/branding/` and the complete canonical public tree at
`.artifacts/forgejo-preview/public/` (including root Sodaspaces modules/CSS and locked
xterm assets). `--out DIR/branding` places the complete tree beside it by default;
`--public-out` selects it explicitly. Use an isolated output for validation rather
than refreshing a live mount without scope. Serving the complete tree still needs a
separately authorized preview mount/configuration change; it is not another app or
listener, and a stock preview alone does not supply the Go API/OAuth backend.
The local Compose mount for
`/data/gitea/public/assets/soda/forgejo` must use that directory instead of the
TypeScript source directory. Other source, font and theme mounts remain as configured.
Changing the mount needs a separately authorized recreation of the existing preview
container while preserving its data volume. Subsequent asset rebuilds use the same
mount and do not require a container restart. This is local preview wiring, not an
appliance deployment or permission to change retained projects.

## Conditions

Use an explicitly authorized matching-native installation/browser and disposable representative identities, projects and repositories. Hide tokens, private keys, passwords, authentication URLs, personal email, private repository names and sensitive terminal details **before capture**. Do not crop away a warning or alter a control to imply a capability.

Capture readable controls with enough navigation to identify the page, not a full desktop. PNG suits UI text. Record actual source revision, target/architecture, viewport and performed action in the review/handoff—not a fabricated PASS manifest. A screenshot proves only what is visible, not that the corresponding provisioning, access or persistence action worked.

Use a fresh private evidence directory under `.artifacts/`; publish selected images into documentation only after explicit review. There are intentionally no published image placeholders below.

## Useful captures

| View | What should be visible |
| --- | --- |
| Soda sign-in | Actual Forgejo sign-in entry, without credentials or OAuth query data |
| Soda Profile | Public-key fingerprints and registration control; never a private key |
| Soda People | Operator create-person flow with empty password input and disposable identities |
| Soda Projects | Real environment list and native provisioning/error state |
| Soda project detail | Explicit join or joined account, actual project IP and SSH guidance; no invented DNS route |
| Project terminal | Alice/Bob identity and matching shared install paths, without secrets or fabricated output |
| Cockpit | Stock operator navigation with Tailnet/Runners; no old Projects/People/Updates pages |
| Tailnet | Connected native state/addresses and relevant preference; authentication URL and sensitive peers hidden |
| Runners | Disposable local capacity and native service state, with no registration token |
| Forgejo repository | Native repository view and actual clone control |
| Forgejo keys | Native public-key registration interface, separate from Soda project-access keys |

## Public handbook placement

The release-day handbook source is `docs/public/`; its
[authoring contract](public/README.md) defines ingestion and image syntax.
Promote only reviewed real captures into `assets/dashboard/`, `assets/cockpit/`
or `assets/forgejo/` under that directory. Reference them beside the relevant
instruction, after the page's description. Keep incomplete capture work and
release-interface review in [the internal editorial checklist](public-docs-review.md),
not in published pages. Do not reuse predecessor Cockpit Projects/People/Updates
captures for the new dashboard or stage a terminal screenshot before its real
own-user/project-boundary behavior is verified.

## Review

Compare each caption and alt text with the actual image. Use only images that clarify an instruction; inspect narrow/wide layouts and relevant light/dark modes before publishing. Keep required warnings and distinguish observed state from unverified actions. The [branding component review](branding-review.md) is a separate visual/style check, not a screenshot source for claiming the full appliance works.

### Verified presentation captures

For the existing local Forgejo preview, use `--verify` to reject HTTP errors,
redirects (including login), unexpected status pages, missing visible main
landmarks, stale template revision/stylesheet registry or bytes, and browser
errors before saving a PNG. `--landmark CSS` tightens the expected page landmark.
`--theme light` or `--theme dark` changes only the capture browser's stylesheet
and document theme; it does not submit or persist an account preference.

```sh
bun scripts/screenshot.ts --verify --landmark .soda-repo-issue-editor \
  --profile .local/screenshot-fixture-profile --theme dark --scroll-top \
  http://localhost:3300/vince/activity-playground/issues/new
```

Every accepted verified PNG has a JSON sidecar with requested/actual URL,
HTTP status, visible-landmark selector, viewport, theme, template revision,
registry hash, per-stylesheet hashes and timestamp. Bump the presentation meta
revision and changed stylesheet versions in `custom/header.tmpl` when activating
a new candidate. Rejected runs preserve earlier captures and report the reason;
a failed route is not coverage. Verification requires real server assets and
cannot be combined with `--local-css`. A valid capture still requires visual
inspection, and a viewport image does not prove offscreen content or interactions.

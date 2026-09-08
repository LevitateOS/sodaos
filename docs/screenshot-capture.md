# Screenshot capture brief — later real evidence

Adapted from `soda-os`'s handbook capture rules, not its old page list or release claims. Local preview captures are separate from installed-product evidence. Do not add broken image links or substitute generated UI, mockups or the Forgejo component sheet for an installed product screenshot.

## Quick local page screenshots

### Existing development fixture login

The local development container `sodaos-local-forgejo` at
`http://localhost:3300` has the user-authorized, non-admin `soda-screenshot`
fixture account. Its authenticated Chrome profile is already saved in ignored
`.local/screenshot-fixture-profile/`. Reuse it directly from the repository root:

```sh
node scripts/screenshot.mjs --profile .local/screenshot-fixture-profile \
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

Use Node.js, installed Google Chrome and the Playwright dependency already declared
in `cockpit/package.json`. The helper uses a persistent browser context so mobile
captures use the requested CSS viewport rather than cropping a desktop window.
On the current development machine, ignored `node_modules/playwright` links to the
already installed desktop runtime package; no package installation was needed.
Other machines can use the existing Cockpit development dependencies or `NODE_PATH`
pointing to an installed Playwright package directory.

Log in once in its dedicated browser window, then press Enter in the terminal:

```sh
node scripts/screenshot.mjs --login http://localhost:3300/user/login
```

Choose “Remember me” if offered. The script keeps that browser profile in ignored
`.local/screenshot-profile/`; it contains session cookies and stays private.
It does not copy your normal browser's cookies or store a password in source.
If the session expires, run `--login` again.

Capture one or more URLs using that session:

```sh
node scripts/screenshot.mjs \
  http://localhost:3300/ \
  http://localhost:3300/user/settings

node scripts/screenshot.mjs --width 390 --height 844 \
  http://localhost:3300/explore/repos
```

Each run prints its PNG paths under a fresh `.artifacts/screenshots/capture-*`
directory. Files are numbered in URL order. These are viewport screenshots,
not full scrolling pages. `--wait 3000` gives JavaScript three extra seconds to
settle. Inspect the result: a PNG can still show an expired login or an error page.

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
node scripts/screenshot.mjs --local-css \
  --profile /absolute/path/to/existing/.local/screenshot-fixture-profile \
  --width 390 --height 844 http://localhost:3300/user/settings
```

This changes only the capture browser's CSS. Native server HTML, scripts, account
preferences and the live checkout remain unchanged. Label these as candidate CSS
captures; they do not prove that edited templates rendered on the server. If the
profile or existing Playwright dependency lives in another checkout, use its
absolute `--profile` path and `NODE_PATH` as needed. No profile copy is necessary.

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

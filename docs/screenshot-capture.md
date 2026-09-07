# Screenshot capture brief — later real evidence

Adapted from `soda-os`'s handbook capture rules, not its old page list or release claims. No screenshots have been captured for this implementation. Do not add broken image links or substitute generated UI, mockups or the Forgejo component sheet for an installed product screenshot.

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

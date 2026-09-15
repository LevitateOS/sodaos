# Stock Cockpit branding

The active adapter is `branding.css` plus `theme.css`. It is independent of the
retired custom-page workspace. [The Cockpit guide](../../../docs/development/cockpit.md)
owns stock-page, security and installed acceptance requirements.

## Current design and supported hook

The adapter follows the current [Forgejo design](../../../docs/reference/forgejo.md):
red actions, white/near-black surfaces, square controls, Barlow body text,
Barlow Condensed headings and IBM Plex Mono controls. It reuses the canonical
brutalist light/dark symbols. Sign-in/administration stays restrained; the marketing
welcome page's subway backgrounds are not substituted into native administration.

Cockpit 366's configuration branding root, login CSS variables and native
`#brand::before` content hook were reviewed, along with its shell and stock Overview
stylesheet imports. Native login code consumes the branding content into its existing
heading. No replacement form, injected application script or custom page is installed.
Native theme activation, controls, error handling, authorization and status/terminal
colors remain owned by Cockpit.

## Staging

`scripts/stage.py` creates `/etc/cockpit/branding/` from:

- This directory's active CSS and the canonical `../theme/palette.css` (local import).
- `../source/soda-symbol-brutalist.svg` and its dark variant.
- The canonical font inventory and licenses in `../fonts/`.
- `../forgejo/apple-touch-icon.png` and an ICO container of the existing canonical
  16px/32px favicon PNGs. The pixels are copied, not redrawn.

Older kit SVG backgrounds, wordmarks, favicon proofs and noninteractive previews
remain preserved source/provenance, not active staging inputs or current screenshots.
No retired workspace output is staged. The top-level `cockpit/` directory is
removed; [`provenance/`](provenance/README.md) preserves its upstream inputs and
licenses without making them active branding/build inputs.

## Checks and limits

`tests/forgejo/cockpit-branding.test.ts` checks workspace independence and resolves
CSS/fonts/native theme tokens in a deliberately small browser styling fixture.
Temporary-filesystem staging tests check canonical icon/font bytes and reject stale
custom-page output. These are not native Cockpit HTML, login/PAM, keyboard/error,
mobile or visual parity acceptance. The [fresh VM receipt](../../../docs/development/cockpit.md)
separately records native root login, stock administration and logout; wider
visual/theme/keyboard acceptance remains pending.

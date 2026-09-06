# Soda OS Cockpit identity kit

The SodaOS staging recipe selects the shared palette, Cockpit adapter, two login
backgrounds, favicon, Apple touch icon and canonical SVGs for `/etc/cockpit/branding/`.
The retained Tailnet and Runners pages bundle the same palette and symbol. The
preview and individual favicon PNG proofs are not installed. This is unbuilt
source composition, not deployment to an existing server; the predecessor's
Projects RPM and four-page Cockpit layout are not carried over.

Open `preview.html` locally to review both themes at 100% browser zoom. Its login
panels are non-interactive placement studies, not stock Cockpit screenshots or
replacement login pages. The source is self-contained and makes no network requests.

## Artwork inventory

Paths below are relative to this directory. Reuse approved SVG masters directly;
do not create a second copy of the symbol or wordmark for Cockpit.

| Placement | Source / deliverable | Treatment |
| --- | --- | --- |
| Login badge or square page identity | `../source/soda-symbol.svg` | Approved full-colour SVG, unchanged |
| Horizontal identity on light surfaces | `../source/soda-logo-horizontal.svg` | Approved lowercase outlined wordmark, unchanged |
| Horizontal identity on dark surfaces | `../source/soda-logo-horizontal-dark.svg` | Approved outlined wordmark and white symbol keyline, unchanged |
| Browser favicon | `favicon.ico` | PNG-encoded 16, 32, 48 px frames, each rendered directly from the symbol |
| Native-size favicon proofs | `favicon-16.png`, `favicon-32.png`, `favicon-48.png` | Transparent PNGs; also inputs to the ICO bundle |
| Apple shortcut | `apple-touch-icon.png` | 180 x 180 px, opaque approved navy; no baked-in corner mask |
| Login background, light | `login-background-light.svg` | Static 1920 x 1080 vector canvas, cover-cropped |
| Login background, dark | `login-background-dark.svg` | Static 1920 x 1080 vector canvas, cover-cropped |
| Shared palette | `../theme/palette.css` | Color values shared with Forgejo |
| Cockpit adapter | `../../../cockpit/src/cockpit/theme.css` | Native theme activation and PatternFly mappings |
| Asset and placement proof | `preview.html`, `preview.css` | Offline responsive review sheet, not runtime UI |

The background waves and bubbles echo the existing symbol without modifying it.
Use these images decoratively, behind an opaque login card. There is no motion,
so no reduced-motion alternative is needed. A plain canvas color is a sufficient
fallback. No animated-background experiment is part of this native source selection.

## Color and type

| Role | Daylight | Midnight |
| --- | --- | --- |
| Canvas | `#FFFDF8` | `#0C1017` |
| Surface | `#FFFFFF` | `#141A24` |
| Main text | `#1C1917` | `#F0F4F8` |
| Secondary text | `#57524E` | `#94A3B8` |
| Input boundary | `#8F857D` | `#64748B` |
| Primary action | `#155EEF` | `#2563EB` |
| Action hover | `#0E56DB` | `#1D4ED8` |
| Action pressed | `#0B46B3` | `#1E40AF` |
| Link / focus | `#155EEF` | `#60A5FA` |
| Text / icon on primary action | `#FFFFFF` | `#FFFFFF` |

`../theme/palette.css` owns the values; this table documents their intended roles.
Keep native warning, danger, success, and disabled-state semantics. Do not recolor
the approved logo to match control colors. The preview's system font is only for
offline review: integration should retain Cockpit's native PatternFly fonts.

The palette targets at least 4.5:1 for main/secondary text, link interaction
states and button labels, and 3:1 for input boundaries and focus colors.
Token-pair measurements are not a complete UI accessibility audit. Current
SodaOS native contrast and keyboard behavior remain unvalidated.
Use underlines for text links. Use a 2px focus outline with 3px offset; do not
replace native keyboard focus with color alone.

Retain the established one-bubble-diameter clear space. The horizontal mark must
not be used below 114px wide. The proof uses 204px login lockups and up to 380px
identity samples. Symbol sizes are placement-dependent; inspect the favicon at
actual 16px, not just enlarged. No new font files or duplicate logo masters.

## Proposed copy and documentation

- Product identity: **Soda OS** (the approved artwork uses lowercase lettering).
- Login tagline: **Your development home.**
- Login instruction: retain stock Cockpit authentication copy; actual host access is root/operator-only.
- Operator extensions: **Runners** and **Tailscale**. Projects/People belong in the separate Soda dashboard; Updates is excluded.
- Review-sheet headline: **Your workspace. Your machine.** This is proposed copy,
  not a new product guarantee or an instruction to add it to every page.

No documentation URL has been invented. Selecting a stable published Soda
handbook destination and contextual page links remains an integration decision.
The preview's documentation link opens this local asset guide, not a purported
public support site. No optional empty-state, social, or About artwork is needed
for this kit; the stock About dialog should remain intact.

## Reproduction and verification

The canonical files are retained unchanged. The predecessor's
`tools/render-cockpit-branding` generator/tests have not been ported; its old
commands and package entry-point patching are not SodaOS build steps. PNG-encoded
ICO frames target modern browsers, not legacy Windows icon consumers.

Assets are architecture-independent, but their presence does not establish a
working native integration. Before shipping, validate the actually installed
Cockpit version in both themes, keyboard/error states, narrow screens and favicon
caching. Preserve stock authentication and operator navigation. Follow the
[current Cockpit port](../../../docs/cockpit-port.md) and
[native validation guide](../../../docs/native-validation.md), not predecessor
RPM/release instructions. Execution remains held until explicitly authorized.

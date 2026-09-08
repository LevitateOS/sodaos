# Reusable Soda assets

Imported from `~/Projects/soda-os` (source HEAD `bc1d3e0dbec48dfaa6a20f9d0453ad3e4cdf353c`). Artwork and color values are unchanged; stylesheet paths were adjusted for this layout. These are design resources, not an implementation of the old product architecture.

## Inventory

| Path | Contents |
| --- | --- |
| `branding/source/` | Nine canonical SVG logos and symbols: full-color, dark-background, white, navy, and black variants |
| `branding/soda-os-logo-concept-v3.png` | Approved visual reference |
| `branding/web/` | Web favicons and Apple touch icon |
| `branding/icons/hicolor/` | System icons from 16px to 512px |
| `branding/installer/` | Raster logo variants, export manifest, and Anaconda CSS |
| `branding/cockpit/` | Light/dark login backgrounds, ICO/PNG icons, offline preview, and Cockpit/PatternFly styles |
| `branding/fonts/` | Self-hosted Fraunces, Barlow and IBM Plex Mono WOFF2 faces, CSS, exact-version provenance and OFL licenses |
| `branding/forgejo/` | App/favicon exports, manifest, artwork preview, native component preview, and light/dark/automatic themes in `css/` |
| `branding/theme/palette.css` | Shared light/dark semantic color tokens and control ramps |
| `branding/terminal/sodaos.txt` | Terminal wordmark |
| `animated-wave-background/` | Animated SVG background with standalone HTML/CSS demo |

Start with the SVG masters and `branding/theme/palette.css` for new UI work. The palette exposes mode-qualified tokens such as `--soda-light-canvas` and `--soda-dark-action`; it does not select a theme or define typography/spacing. Preserve the distinction between link and filled-action colors.

## Preview locally

Open these files directly in a browser:

- `branding/cockpit/preview.html`
- `branding/forgejo/preview.html`
- `animated-wave-background/index.html`

`branding/forgejo/theme-preview.html` is **not standalone**: it requires a matching Forgejo instance and its native CSS/image routes.

## Application-specific styles

These adapters are retained for reuse/reference, not globally applied to the new dashboard:

- `branding/cockpit/theme.css`: semantic roles and PatternFly mappings, originally from `cockpit/src/cockpit/theme.css`.
- `branding/cockpit/soda.css`: small Soda page utility styles from the same source directory.
- `branding/cockpit/branding.css`: native Cockpit 366 login adapter, from `packaging/rpm/projects/sources/branding/sodaos/branding.css`.
- `branding/forgejo/css/*.css`: Forgejo 15 themes and controls, from `packaging/rpm/forgejo/sources/custom/public/assets/css/`. Native `theme-forgejo-light.css` / `theme-forgejo-dark.css` are supplied by Forgejo, not this repository.
- `branding/installer/soda.css`: Anaconda integration from `packaging/installer/branding/soda.css`; absolute image URLs refer to installed Anaconda paths.

Palette imports now resolve to the single `branding/theme/palette.css` source. Future packaging must preserve these relative paths or rewrite imports when staging. Native Cockpit branding also expects installed wordmark paths; this import does not install or activate any themes.

The READMEs in `branding/cockpit/` and `branding/theme/` retain original design context. Their descriptions of RPM staging and source-repository tests are historical, not claims about this project.

## Scope and maintenance

No application components, services, OS configuration, dependencies, caches, credentials, or build artifacts were imported. No standalone font files existed outside dependencies/build outputs; use the chosen frontend's fonts rather than copying vendored font infrastructure.

Raster exports and manifests are retained unchanged. Forgejo-only regeneration/check source is now ported as `scripts/render-forgejo-branding.sh` and `tools/png-equal/`, together with `scripts/check-forgejo-branding.mjs` for the native component sheet. See [branding review](../docs/branding-review.md); execution is held and no derivatives were regenerated. Other renderer source (`scripts/render-branding.*`, `scripts/render-installer-branding.sh`, `tools/render-cockpit-branding/`) remains in the predecessor; do not execute or modify that repository without separate authorization, or hand-edit derivative PNGs.

See [branding usage](../docs/branding.md) for mark selection and constraints.

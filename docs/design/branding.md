# Soda branding assets

The current identity is the offset-core emblem shared with the website. Its outer plate is red (#df001b), its middle plate is black (#101010) on light surfaces or white on dark surfaces, and its core is transparent. There are no monochrome variants.

## Logo usage

- Use [soda-symbol.svg](../../assets/branding/source/soda-symbol.svg) on light surfaces and [soda-symbol-dark.svg](../../assets/branding/source/soda-symbol-dark.svg) on dark surfaces.
- Use [soda-logo-horizontal.svg](../../assets/branding/source/soda-logo-horizontal.svg) or [soda-logo-horizontal-dark.svg](../../assets/branding/source/soda-logo-horizontal-dark.svg) when the wordmark is needed.
- The existing `soda-symbol-brutalist.svg` and dark counterpart contain the same current emblem and are consumed by Forgejo, Cockpit and terminal rendering.
- Keep the core transparent. Do not recolor, stretch, rotate or add effects to the emblem.

SVGs in `assets/branding/source/` are editable masters, synchronized with the website public assets.

## Raster exports

Run `scripts/render-branding.sh` to regenerate installer logos, desktop icons and web/Cockpit/Forgejo PNGs. Use `scripts/render-branding.sh --check` to compare their pixels with the current SVGs. Export sizes and backgrounds live in `assets/branding/manifest.tsv` and the Forgejo manifest. Cockpit staging constructs its ICO from the current Forgejo PNGs.

## Color and backgrounds

[palette.css](../../assets/branding/theme/palette.css) owns shared light/dark colors. Preserve distinct link and filled-action roles. See the [asset inventory](../../assets/README.md), [Forgejo customization](../reference/forgejo.md) and [Cockpit](../development/cockpit.md) for integration.

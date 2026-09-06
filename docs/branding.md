# Soda branding assets

The reusable design library lives in [`assets/`](../assets/README.md), imported from the previous `soda-os` repository. Existing artwork still spells the product “Soda OS”; the migration does not redraw or rename the wordmark.

## Logo usage

- Use [`soda-logo-horizontal.svg`](../assets/branding/source/soda-logo-horizontal.svg) on white/light backgrounds.
- Use [`soda-logo-horizontal-dark.svg`](../assets/branding/source/soda-logo-horizontal-dark.svg) on navy/dark backgrounds; its white keyline protects the symbol.
- Use [`soda-symbol.svg`](../assets/branding/source/soda-symbol.svg) for square placements or where the product name is already visible.
- White, navy, and black variants are for restricted-color reproduction on contrasting backgrounds.
- Keep one bubble diameter of clear space. Do not display the horizontal wordmark below 114px wide.
- Do not stretch, rotate, recolor, add effects/outlines, or put the full-color mark on cyan/busy backgrounds.

SVGs in `assets/branding/source/` are canonical. PNGs and the ICO are derivatives, not editable masters.

## Color and backgrounds

[`palette.css`](../assets/branding/theme/palette.css) is the shared color source. Light mode uses warm ivory, white surfaces, dark ink, and primary blue; dark mode uses navy surfaces, light ink, brighter blue links, and darker filled actions. Preserve separate action/link roles for contrast.

Static login backgrounds are in `assets/branding/cockpit/`; the animated wave demo is in `assets/animated-wave-background/`. The application-specific themes are reference assets until explicitly integrated and tested against their host application versions.

See the [asset inventory](../assets/README.md) for previews, source locations, dependencies, and regeneration notes. The adapted [Forgejo branding review](branding-review.md) describes installed configuration, optional renderer/browser checks and their held execution requirements.

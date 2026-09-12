# Forgejo image assets

Forgejo delivers six Soda branding images plus six welcome-page backgrounds:

| Asset | Purpose |
| --- | --- |
| `../source/soda-symbol-brutalist.svg` | Light-surface symbol |
| `../source/soda-symbol-brutalist-dark.svg` | Dark-surface symbol |
| `logo.png` | 512px raster logo/default share image |
| `favicon.png` | 32px browser icon |
| `favicon-16.png` | 16px browser icon |
| `apple-touch-icon.png` | 180px symbol on black |

`manifest.tsv` owns the PNG exports. Native `/assets/img/` aliases reuse the
canonical exports; they are not separate artwork. The native symbol/core remains
open, with a red outer plate and contrasting middle plate.

The delivery manifest is `internal/nativebuild/forgejo-payload.json`. Its source
test rejects extra branding images. `build:preview` removes unlisted top-level
images from its generated Soda image directories after successful staging; native
icons and uploaded content are outside that cleanup.

The 34 papercraft illustrations, their 29 prompt/review documents, the obsolete
standalone artwork preview and two unused Forgejo wordmarks have been removed.
History and original provenance remain in Git before this cleanup. Nine shared
legacy SVG sources remain outside Forgejo because Cockpit/installer consumers
still use them; none is included in Forgejo's payload.

Native interface/provider icons, QR codes and actual user/repository/organization
images remain. The functional `theme-preview.html` uses the native canonical logo.
See [redesign status](../../../docs/forgejo-redesign-status.md) for live/component
review evidence and [branding review](../../../docs/branding-review.md) for tools.

The public front page uses the website’s six approved soda-bar WebPs: mobile,
tablet and desktop, each in day/night. They live in `backgrounds/`; its README
records provenance. The native page and footer form one centered opaque frame.
Other pages do not request or display these backgrounds.

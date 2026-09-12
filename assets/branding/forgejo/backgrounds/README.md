# Welcome-page backgrounds

The six optimized WebP files are copied without modification from the website's
`src/app/assets/soda-bar-backgrounds` at commit `4ff11a5` in
[LevitateOS/soda-os-website](https://github.com/LevitateOS/soda-os-website/tree/4ff11a5/src/app/assets/soda-bar-backgrounds).
That repository owns the original PNG masters and generation prompts.

`home.css` selects mobile below 48rem, tablet from 48rem and desktop from 75rem,
with a separate day/night exposure following the guest theme. Only the public
welcome page paints a background; other Forgejo pages retain their native layout.
`forgejo-payload.json` delivers all six relative CSS assets, including subpath installs.

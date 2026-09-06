# Branding review — execution held

The Forgejo browser check, SVG-to-PNG renderer and pixel-comparison utility are adapted from `soda-os` commit `bc1d3e0dbec48dfaa6a20f9d0453ad3e4cdf353c`, with their focused tests. No rendering, browser execution, tests or artwork changes were performed during the port.

## Installed configuration

`appliance/config/forgejo.env` carries the predecessor's app metadata, stock/accessibility theme choices and static-cache revalidation setting into the container deployment. It does not bring over the old PAM identity, host paths, fixed ports, package restrictions or generated secrets. The selected Forgejo 15.0.7 environment-to-INI source supports the empty/default section (`FORGEJO____APP_NAME`) and escaped section dot (`ui_0x2E_meta`). Setup still owns browser/SSH configuration; existing installations are not silently migrated by this source edit.

Staging supplies the canonical symbols at Forgejo's native `/assets/img/` paths and relocates the palette imports consistently. The application keeps its native diff, error and ANSI colors. These remain unvalidated installed assumptions.

## Later component-sheet check

This needs an explicitly approved **disposable matching-native Linux Forgejo deployment** with the same native version and staged Soda styles. It is not a fake replacement for Forgejo CSS or an authenticated production workflow test.

1. Complete the later native dependency/browser prerequisites. The check uses the pinned Playwright in `cockpit/package.json`; installation of that package/browser is a separately authorized operation.
2. On that disposable target, copy `assets/branding/forgejo/theme-preview.html` to `/var/lib/soda/forgejo/gitea/public/assets/soda-theme-preview.html`, with the same readability/ownership as adjacent custom assets. This is an explicit target mutation, not part of ordinary packaging. The review sheet is intentionally excluded from the appliance stage.
3. Inspect it at the target's actual `/assets/soda-theme-preview.html` URL. Use the operator's trusted private route or a loopback tunnel; do not change listener exposure or disable TLS verification for the check.
4. From the authorized matching-native Linux browser environment, invoke:

```sh
SODA_NATIVE_VALIDATE=DISPOSABLE_TARGET node scripts/check-forgejo-branding.mjs \
  https://FORGEJO_ORIGIN/assets/soda-theme-preview.html \
  .artifacts/branding/FRESH_REVIEW_DIRECTORY
```

Use a fresh evidence directory; existing evidence is not overwritten. The checker opens fresh unauthenticated browser contexts, changes only the review sheet's query-selected theme, checks controls/contrast/focus/native semantic colors, captures narrow/wide/2x images and reports page/resource errors. It does not change anyone's saved Forgejo preference or create repositories. Do not point it at another page.

Review the resulting images and remove only the explicitly run-owned review sheet when authorized. Component checks and screenshots do **not** establish full accessibility conformance, login/CI behavior or installed appliance acceptance. Follow [native validation](native-validation.md) for those separate journeys.

## Later raster consistency/regeneration

Canonical SVGs and the existing export manifest remain unchanged. With Go and native `rsvg-convert` explicitly available:

```sh
scripts/render-forgejo-branding.sh --check
# Equivalent focused Go wrapper; renderer tests are opt-in, not silently skipped:
go test -tags=branding ./scripts
```

The default source-test run includes the PNG pixel-comparison utility, placement checks and renderer argument checks. The `branding` test tag additionally invokes the actual renderer. Report whether that separate check ran.

Only when derivative regeneration is requested, run `scripts/render-forgejo-branding.sh` without `--check`. It writes the manifest-listed PNGs from the canonical SVG; it does not redraw the logo or update other application assets. Inspect the diff and preserve attribution before committing changes.

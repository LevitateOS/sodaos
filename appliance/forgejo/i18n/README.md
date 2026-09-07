# Soda translation scaffold

`en-US.ini` reserves the `[soda]` namespace for Soda-owned text. It is intentionally
empty. Add other language files only when translation work is selected.
Existing templates and JavaScript remain unchanged and continue using their current
English copy and native Forgejo translations.

These files are **additions**, not deployable replacement catalogs. Forgejo 15.0.7
loads a custom locale file ahead of its built-in equivalent without merging keys.
Do not mount this directory directly into the running container.

When localization is implemented:

1. Read the exact selected Forgejo version's built-in `options/locale/locale_*.ini`.
2. Combine each native catalog with the corresponding Soda additions in an ignored
   generated output directory, rejecting duplicate keys rather than silently
   replacing native translations. Keep native catalogs out of authored source.
3. Deliver the complete files to `<CustomPath>/options/locale/locale_<language>.ini`
   (`/data/gitea/options/locale/` in our preview), with matching upstream notices.
4. Use `ctx.Locale.Tr "soda.<key>"` in templates and pass translated JavaScript
   labels through data attributes. Keep native strings on their existing keys.
5. Restart the selected instance to load locales in production mode, then check
   native translations, Soda translations, fallback behavior and text wrapping.

No merge script, staging integration, locale mount, translations or restart are
part of this scaffold. JSON catalogs under `options/locale_next/` also exist in
Forgejo 15; this scaffold uses INI for simple text additions.

Verified source: [locale loading](https://codeberg.org/forgejo/forgejo/src/tag/v15.0.7/modules/translation/translation.go)
and [custom file precedence](https://codeberg.org/forgejo/forgejo/src/tag/v15.0.7/modules/assetfs/layered.go).

# Soda translations

`en-US.ini` contains Soda-owned additions in `[soda]`. Existing controls and
warnings retain native translations. Untranslated Soda additions use Forgejo's
English fallback; existing native languages and JSON catalogs remain untouched.

These additions are **not a deployable replacement catalog**. Forgejo 15.0.7
loads a custom INI ahead of its embedded equivalent without merging its keys.
Never mount this directory directly into a running instance.

For the authorized local preview, extract the exact existing image's English INI
with `forgejo embedded view options/locale/locale_en-US.ini`, then run:

```sh
python3 scripts/forgejo-locales.py \
  --native .artifacts/forgejo-presentation/upstream/locale_en-US.ini \
  --out .artifacts/personal-settings/locales/locale_en-US.ini
```

The standard-library helper validates the complete native input and rejects
repeated keys and namespace collisions. It concatenates the original INI bytes
without reserializing native values, interpolation, escapes or translation keys.
Native JSON catalogs under `options/locale_next/` remain under native ownership.
Generated native catalogs belong under ignored `.artifacts/`, with Forgejo's
GPL-3.0-or-later attribution, never in authored source.

Use `ctx.Locale.Tr "soda.<key>"` in templates. JavaScript uses rendered labels
rather than maintaining a parallel translation dictionary.

The complete generated English catalog was copied to the existing preview's
`/data/gitea/options/locale/locale_en-US.ini` and activated with the single
user-authorized restart. The existing image, configuration and data volume were
retained. Further locale changes require another authorized activation; template
reloads do not refresh production locale caches. Appliance staging is separate.

Verified upstream source:
[locale loading](https://codeberg.org/forgejo/forgejo/src/tag/v15.0.7/modules/translation/translation.go)
and [custom file precedence](https://codeberg.org/forgejo/forgejo/src/tag/v15.0.7/modules/assetfs/layered.go).

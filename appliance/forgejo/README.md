# Soda Forgejo login override

Targets stock Forgejo **15.0.7**. The login wrapper is adapted from the embedded
`templates/user/auth/signin.tmpl`; upstream project: https://codeberg.org/forgejo/forgejo.
Forgejo's GPL-3.0-or-later terms apply to the adapted template; see
[licensing obligations](../../docs/licensing.md). Its original embedded wrapper SHA-256:
`b19374666dc4bff231aa49d60d2459cfebf565028edf67b439b3d093fc3a7547`.

Soda changes the wrapper into a branded story/form layout and adds a scoped
stylesheet via the official `custom/header` hook. The existing
`user/auth/signin_inner` template is invoked unchanged, including alerts, internal
sign-in, account-link mode, password settings, CAPTCHA, providers, WebAuthn errors,
registration and password-recovery controls. Native head/footer and scripts remain.
No authentication handlers, form fields or security middleware are replaced.

## Asset mapping

| Source | Forgejo custom destination |
| --- | --- |
| `templates/` | `/data/gitea/templates/` |
| `assets/branding/forgejo/` (repository root) | `/data/gitea/public/assets/soda/forgejo/` |
| `assets/branding/fonts/` | `/data/gitea/public/assets/soda/fonts/` |
| `assets/branding/theme/` | `/data/gitea/public/assets/soda/theme/` |
| `assets/branding/source/` | `/data/gitea/public/assets/soda/source/` |

The preview's `.artifacts/local-forgejo/compose.yaml` binds these source directories
read-only; Docker volume `sodaos-local-forgejo_data` retains its users/repositories.
CSS and artwork changes need a browser refresh; template changes need:

```sh
docker exec --user git sodaos-local-forgejo forgejo manager reload-templates
```

The stylesheet uses `AssetUrlPrefix` through the template and relative CSS imports.
Login presentation is the approved **light** concept; other pages retain their native
theme. Below 900px the illustration is hidden to prioritize signing in. The English
brand copy is authored here; native form labels retain Forgejo localization.
No password-manager replacement, fake theme switch or unsupported sign-in option
is added. Forgejo's native footer retains language/license access.

The papercraft PNG is the user-approved original generated illustration, copied
unchanged from `.artifacts/design/login-papercraft-v1.png`. Canonical logo artwork
and the shared palette are reused unchanged. Fonts retain their family OFL notices.

## Scope and checks

Only the isolated local preview was updated. Appliance staging does **not** yet
ship these overrides/assets; template allowlisting, conflict refusal, full notice
packaging and separately authorized deployment remain required before rollout.
See the implementation handoff for actual checks and remaining validation.

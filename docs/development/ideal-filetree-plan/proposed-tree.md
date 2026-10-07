# Complete desired repository tree

## Complete proposed tree

Every tracked source has a destination or an explicit disposition. The tree applies the [package consolidations](package-ownership.md#recommended-package-changes) and the decided language policy: it shows the post-cutover target, so pre-port Go/Python implementations are omitted even where their cutover commit is still pending. It retains the host crate at `lib/host/`; the decomposition section's host entries are now the retained plan, not a conditional alternative. Existing small files keep their names unless package consolidation requires a namespace. The three release-assets integration-test namespaces retain distinct support modules.

The existing `sodaspaces-factory-view.ts` and `sodaspaces-terminal-view.ts` receive
view concerns from their larger state owners; their combined size still needs
implementation review. Providers and assets merge into their existing owners.
Shared HTTP/test support and native assertion destinations also intentionally
consolidate existing concerns. Each shared leaf has one implementation owner;
duplicate file leaves or competing Rust module roots are not intended.

Initial selective adoption update at source `72e4bb9015b6d6a622b45638104c74851a137473`,
refreshed for completed L04 at `229e9cce` and L08/L09 at `21387814`:
this remains the desired application tree, with superseded generic-engine leaves
removed below. Retained names describe application policy or library adapters,
not a requirement to recreate their old implementation. The
[library chapter](library-adoption.md#execution-packets) controls exact boundaries
and gated deletion. Full inventory/count regeneration waits for implementation;
the earlier exact-match/delta counts below are historical observations.

| Retained target family | Application responsibility after adoption |
| --- | --- |
| Identity `http*`, `pg.rs`, Store/Tx and `strict*` | Hyper listener/admission and tokio-postgres deadline/typed transaction adapters; Serde profile/domain rules. No HTTP frame, PG wire/DSN or SQL translator engine |
| A-owned `lib/unix-http/{Cargo.toml,src/lib.rs}` | Shared bounded Hyper Unix client and driver/deadline custody; callers retain socket, status and credential policy |
| Host `daemon/{http,response,websocket}`, `json/{mod,number}`, `ssh/material` | Routes/body limits, single upgrade/pump lifecycle, Serde schemas and ssh-key algorithm/fingerprint policy |
| Installer `netip`, `sshkey/authorized_keys`, `pemx`, `x509` | std IP prefix/admission; ssh-key policy; bounded PEM envelope and typed local-CA/raw-TBS verification. Certificate-only URL/calendar grammar is retired after L06 gates |
| Release `json*`, Compose/Muse/guest wire and Acceptance structured data | Concrete Serde admission/emission and bounded ordered/raw application data; `lib/json` is retired with no target allocation |
| Release/build/import OCI adapters | Delivery owns low-level scanning; callers retain admitted content/layout/descriptor policy and original blob custody |
| `lib/release-inputs/src/trust_key.rs` and release trust callers | Shared typed P-256 admission; image/delivery retain role authority and original-DER fingerprints |
| Release/terminal/acceptance process, file and evidence modules | Existing authority, bounded input/output, cancellation, cleanup and narrow library adapters; no new framework |

Hash/codec engines disappear into selected library calls and existing fingerprint
recipes. Tests of retained policy remain; grammar-only equivalence tests are
replaced with actual admitted/rejected producer/caller profiles, not copied as
new library modules. The coordinator updates dependency/bin/test selectors with
the affected implementation, rather than treating this tree as source evidence.

Historical R02 reconciliation @HEAD `216cad1b` (landed-scope check of the
then-proposed tree; the selective target changes above do not refresh this evidence):

- EXACT MATCH (every HEAD file proposed, 0 missing): `lib/json` (2),
  `lib/soda-release-build` (21), `lib/soda-release-image` (33),
  `cmd/soda-identity` (42), `cmd/soda-project-terminal` (20), `system/` (64).
- FUTURE-SPLIT DELTAS (current coarse file, proposed dir exists — pending
  decomposition, not misplacement; 10 files): deliver `src/{buildx,fetch,
  jsonx,model,native,oci,publish}.rs` → same-stem dirs, `tests/oracle.rs` →
  `tests/oracle/`; tools `tests/cli.rs` → `tests/cli/`; host `src/domain.rs`
  → `src/domain/`.
- UNMAPPED (no same-stem proposed destination; dispositions belong to
  slice/decomposition upkeep — recorded here, not silently carried; 37
  files): host `GMUX_PATCHES.md`, `src/{dbackend,gmux_admission,
  gmux_backend,gmux_routes,gmux_server,iclient,iconfig,json,main,muse,
  muse_serve,pfactory,pops,tailnet_companion,tailnet_domain,tailnet_files,
  tailnet_runtime,tcontrol,tcontrol_enroll,tcontrol_native,tcontrol_policy,
  tcontrol_provider,tcontrol_wire}.rs`,
  `src/terminal/factory/{tcodex,tfactory,tmuse}.rs`,
  `tests/{gmux_smoke,iclient_oracle,muse_serve_oracle,pops_oracle,
  tcodex_ops_oracle,tcontrol_oracle}.rs` (33); tools `src/{check_cli,
  pipeline}.rs` (2); assets `tests/fetchers_cli.rs` +
  `tests/fixtures/prov-root/.../forgejo-payload.json` (2, C09-created,
  post-proposal).

```text
.
├── .agents/
│   └── plans/
│       └── 2026-09-16-remove-rpm-pinning.md
├── .githooks/
│   └── pre-commit
├── assets/
│   ├── animated-wave-background/
│   │   ├── animated-waves.svg
│   │   ├── index.html
│   │   └── styles.css
│   ├── branding/
│   │   ├── cockpit/
│   │   │   ├── provenance/
│   │   │   │   ├── LICENSES.txt
│   │   │   │   ├── README.md
│   │   │   │   ├── patternfly-MIT.txt
│   │   │   │   ├── patternfly-react-MIT.txt
│   │   │   │   └── redhat-fonts-OFL.txt
│   │   │   ├── README.md
│   │   │   ├── apple-touch-icon.png
│   │   │   ├── branding.css
│   │   │   ├── favicon-16.png
│   │   │   ├── favicon-32.png
│   │   │   ├── favicon-48.png
│   │   │   ├── favicon.ico
│   │   │   ├── login-background-dark.svg
│   │   │   ├── login-background-light.svg
│   │   │   ├── preview.css
│   │   │   ├── preview.html
│   │   │   ├── soda.css
│   │   │   └── theme.css
│   │   ├── fonts/
│   │   │   ├── barlow/
│   │   │   │   ├── LICENSE
│   │   │   │   ├── barlow-latin-400-normal.woff2
│   │   │   │   └── barlow-latin-600-normal.woff2
│   │   │   ├── barlow-condensed/
│   │   │   │   ├── LICENSE
│   │   │   │   └── barlow-condensed-latin-800-normal.woff2
│   │   │   ├── fraunces/
│   │   │   │   ├── LICENSE
│   │   │   │   └── fraunces-latin-wght-normal.woff2
│   │   │   ├── ibm-plex-mono/
│   │   │   │   ├── LICENSE
│   │   │   │   ├── ibm-plex-mono-latin-400-normal.woff2
│   │   │   │   └── ibm-plex-mono-latin-500-normal.woff2
│   │   │   ├── README.md
│   │   │   ├── fonts.css
│   │   │   └── sources.json
│   │   ├── forgejo/
│   │   │   ├── backgrounds/
│   │   │   │   ├── masters/
│   │   │   │   │   ├── desktop-day.png
│   │   │   │   │   ├── desktop-night.png
│   │   │   │   │   ├── mobile-day.png
│   │   │   │   │   ├── mobile-night.png
│   │   │   │   │   ├── tablet-day.png
│   │   │   │   │   └── tablet-night.png
│   │   │   │   ├── README.md
│   │   │   │   ├── manifest.json
│   │   │   │   ├── subway-desktop-day.webp
│   │   │   │   ├── subway-desktop-night.webp
│   │   │   │   ├── subway-mobile-day.webp
│   │   │   │   ├── subway-mobile-night.webp
│   │   │   │   ├── subway-tablet-day.webp
│   │   │   │   └── subway-tablet-night.webp
│   │   │   ├── css/
│   │   │   │   ├── soda-controls.css
│   │   │   │   ├── theme-soda-auto.css
│   │   │   │   ├── theme-soda-dark.css
│   │   │   │   └── theme-soda-light.css
│   │   │   ├── login-station/
│   │   │   │   ├── masters/
│   │   │   │   │   └── approaching-train.png
│   │   │   │   ├── README.md
│   │   │   │   ├── approaching-train.webp
│   │   │   │   └── manifest.json
│   │   │   ├── README.md
│   │   │   ├── account-details.css
│   │   │   ├── account-settings.css
│   │   │   ├── admin-details.css
│   │   │   ├── admin-monitoring.css
│   │   │   ├── admin.css
│   │   │   ├── apple-touch-icon.png
│   │   │   ├── auth.css
│   │   │   ├── code-search.css
│   │   │   ├── components-buttons.css
│   │   │   ├── components-empty.css
│   │   │   ├── components-forms.css
│   │   │   ├── components-guest.css
│   │   │   ├── components-intro.css
│   │   │   ├── components-list.css
│   │   │   ├── components-navigation.css
│   │   │   ├── components-repository-toolbar.css
│   │   │   ├── components-settings.css
│   │   │   ├── components-toolbar-layout.css
│   │   │   ├── components-toolbar.css
│   │   │   ├── components.css
│   │   │   ├── configuration.css
│   │   │   ├── create.css
│   │   │   ├── dashboard.css
│   │   │   ├── explore.css
│   │   │   ├── favicon-16.png
│   │   │   ├── favicon.png
│   │   │   ├── federated-auth.css
│   │   │   ├── forgejo-events.d.ts
│   │   │   ├── forgejo-setup.css
│   │   │   ├── form-pages.css
│   │   │   ├── home.css
│   │   │   ├── insights.css
│   │   │   ├── lit.ts
│   │   │   ├── login-theme.ts
│   │   │   ├── login.css
│   │   │   ├── logo.png
│   │   │   ├── manifest.tsv
│   │   │   ├── milestones.css
│   │   │   ├── moderation.css
│   │   │   ├── notification-preview.css
│   │   │   ├── notification-preview.ts
│   │   │   ├── notifications.css
│   │   │   ├── onboarding.css
│   │   │   ├── org-create.css
│   │   │   ├── org-details.css
│   │   │   ├── org-home.css
│   │   │   ├── organization.css
│   │   │   ├── packages.css
│   │   │   ├── personal-settings.ts
│   │   │   ├── profiles.css
│   │   │   ├── projects.css
│   │   │   ├── quota.css
│   │   │   ├── repository-actions.ts
│   │   │   ├── repository-code-actions.css
│   │   │   ├── repository-code-browser.css
│   │   │   ├── repository-code-editing.css
│   │   │   ├── repository-code.css
│   │   │   ├── repository-content.css
│   │   │   ├── repository-issues.css
│   │   │   ├── repository-settings-details.css
│   │   │   ├── repository-switcher.css
│   │   │   ├── repository-switcher.ts
│   │   │   ├── repository.css
│   │   │   ├── runners.css
│   │   │   ├── status.css
│   │   │   ├── theme-preview.html
│   │   │   └── webhooks.css
│   │   ├── host/
│   │   │   └── README.md
│   │   ├── icons/
│   │   │   ├── hicolor/
│   │   │   │   ├── 128x128/
│   │   │   │   │   └── apps/
│   │   │   │   │       └── soda-os.png
│   │   │   │   ├── 16x16/
│   │   │   │   │   └── apps/
│   │   │   │   │       └── soda-os.png
│   │   │   │   ├── 24x24/
│   │   │   │   │   └── apps/
│   │   │   │   │       └── soda-os.png
│   │   │   │   ├── 256x256/
│   │   │   │   │   └── apps/
│   │   │   │   │       └── soda-os.png
│   │   │   │   ├── 32x32/
│   │   │   │   │   └── apps/
│   │   │   │   │       └── soda-os.png
│   │   │   │   ├── 48x48/
│   │   │   │   │   └── apps/
│   │   │   │   │       └── soda-os.png
│   │   │   │   ├── 512x512/
│   │   │   │   │   └── apps/
│   │   │   │   │       └── soda-os.png
│   │   │   │   └── 64x64/
│   │   │   │       └── apps/
│   │   │   │           └── soda-os.png
│   │   │   └── octicons/
│   │   │       ├── LICENSE
│   │   │       ├── README.md
│   │   │       ├── repo-19.14.0.svg
│   │   │       └── terminal-19.14.0.svg
│   │   ├── installer/
│   │   │   ├── manifest.tsv
│   │   │   ├── soda-logo-black.png
│   │   │   ├── soda-logo-horizontal-dark.png
│   │   │   ├── soda-logo-horizontal.png
│   │   │   ├── soda-logo-navy.png
│   │   │   ├── soda-logo-white.png
│   │   │   ├── soda-symbol-black.png
│   │   │   ├── soda-symbol-navy.png
│   │   │   ├── soda-symbol-white.png
│   │   │   ├── soda-symbol.png
│   │   │   └── soda.css
│   │   ├── source/
│   │   │   ├── soda-logo-black.svg
│   │   │   ├── soda-logo-horizontal-dark.svg
│   │   │   ├── soda-logo-horizontal.svg
│   │   │   ├── soda-logo-navy.svg
│   │   │   ├── soda-logo-white.svg
│   │   │   ├── soda-symbol-black.svg
│   │   │   ├── soda-symbol-brutalist-dark.svg
│   │   │   ├── soda-symbol-brutalist.svg
│   │   │   ├── soda-symbol-navy.svg
│   │   │   ├── soda-symbol-white.svg
│   │   │   └── soda-symbol.svg
│   │   ├── terminal/
│   │   │   ├── README.md
│   │   │   ├── fastfetch.jsonc
│   │   │   ├── motd.txt
│   │   │   └── sodaos.txt
│   │   ├── theme/
│   │   │   ├── README.md
│   │   │   └── palette.css
│   │   ├── web/
│   │   │   ├── apple-touch-icon.png
│   │   │   ├── favicon-16.png
│   │   │   └── favicon-32.png
│   │   └── soda-os-logo-concept-v3.png
│   └── README.md
├── cmd/
│   ├── soda-activate/
│   │   ├── src/
│   │   │   ├── tests/
│   │   │   │   ├── activation.rs
│   │   │   │   ├── cli.rs
│   │   │   │   ├── fixtures.rs
│   │   │   │   ├── mod.rs
│   │   │   │   └── origin.rs
│   │   │   ├── activation.rs
│   │   │   ├── cli.rs
│   │   │   ├── forgejo_env.rs
│   │   │   ├── main.rs
│   │   │   ├── origin.rs
│   │   │   └── system.rs
│   │   └── Cargo.toml
│   ├── soda-console-welcome/
│   │   ├── src/
│   │   │   ├── config.rs
│   │   │   ├── main.rs
│   │   │   ├── origin.rs
│   │   │   ├── tests.rs
│   │   │   └── welcome.rs
│   │   └── Cargo.toml
│   ├── soda-dashboard/
│   │   ├── extension.go
│   │   ├── extension_test.go
│   │   ├── main.go
│   │   ├── operator.go
│   │   ├── operator_linux.go
│   │   ├── operator_linux_test.go
│   │   ├── operator_other.go
│   │   ├── operator_test.go
│   │   └── postgres_fixture_test.go
│   ├── soda-extension/
│   │   ├── main.go
│   │   └── main_test.go
│   ├── soda-factory/
│   │   ├── src/
│   │   │   ├── main.rs
│   │   │   └── operator_tests.rs
│   │   └── Cargo.toml
│   ├── soda-forgejo-domain/
│   │   ├── src/
│   │   │   ├── tests/
│   │   │   │   ├── cli.rs
│   │   │   │   ├── config.rs
│   │   │   │   ├── domain.rs
│   │   │   │   ├── fixtures.rs
│   │   │   │   └── mod.rs
│   │   │   ├── cli.rs
│   │   │   ├── config.rs
│   │   │   ├── domain.rs
│   │   │   ├── main.rs
│   │   │   └── system.rs
│   │   └── Cargo.toml
│   ├── soda-forgejo-migrate/
│   │   ├── src/
│   │   │   └── main.rs
│   │   └── Cargo.toml
│   ├── soda-forgejo-tailnet/
│   │   └── main.rs
│   ├── soda-host/
│   │   └── main.rs
│   ├── soda-identity/
│   │   ├── src/
│   │   │   ├── providers/
│   │   │   │   ├── codex/
│   │   │   │   │   ├── config.rs
│   │   │   │   │   ├── mod.rs
│   │   │   │   │   ├── protocol.rs
│   │   │   │   │   └── tests.rs
│   │   │   │   ├── muse/
│   │   │   │   │   ├── mod.rs
│   │   │   │   │   └── tests.rs
│   │   │   │   ├── mod.rs
│   │   │   │   └── types.rs
│   │   │   ├── acquisition.rs
│   │   │   ├── control.rs
│   │   │   ├── crypto.rs
│   │   │   ├── enrollment.rs
│   │   │   ├── grants.rs
│   │   │   ├── http.rs
│   │   │   ├── http_routes.rs
│   │   │   ├── http_tests.rs
│   │   │   ├── lib.rs
│   │   │   ├── main.rs
│   │   │   ├── pg.rs
│   │   │   ├── pg_dsn.rs
│   │   │   ├── pg_query.rs
│   │   │   ├── pg_tests.rs
│   │   │   ├── registration.rs
│   │   │   ├── retirement.rs
│   │   │   ├── runtime.rs
│   │   │   ├── schema.rs
│   │   │   ├── store.rs
│   │   │   ├── store_connections.rs
│   │   │   ├── store_events.rs
│   │   │   ├── store_executions.rs
│   │   │   ├── store_grants.rs
│   │   │   ├── store_leases.rs
│   │   │   ├── store_schema.rs
│   │   │   ├── store_tests.rs
│   │   │   ├── strict.rs
│   │   │   ├── strict_tests.rs
│   │   │   ├── wire.rs
│   │   │   ├── wire_errors.rs
│   │   │   ├── wire_execution.rs
│   │   │   ├── wire_grants.rs
│   │   │   ├── wire_scalars.rs
│   │   │   ├── wire_tests.rs
│   │   │   └── wire_time.rs
│   │   ├── tests/
│   │   │   ├── common/
│   │   │   │   └── mod.rs
│   │   │   ├── broker.rs
│   │   │   ├── enrollment.rs
│   │   │   └── http.rs
│   │   └── Cargo.toml
│   ├── soda-identity-compose/
│   │   ├── src/
│   │   │   ├── compose.rs
│   │   │   ├── compose_tests.rs
│   │   │   ├── launch_wire.rs
│   │   │   ├── main.rs
│   │   │   ├── options.rs
│   │   │   └── registration.rs
│   │   └── Cargo.toml
│   ├── soda-image-import/
│   │   ├── src/
│   │   │   ├── oci/
│   │   │   │   ├── inspection.rs
│   │   │   │   ├── layout.rs
│   │   │   │   ├── metadata.rs
│   │   │   │   └── mod.rs
│   │   │   ├── tests/
│   │   │   │   ├── fixtures.rs
│   │   │   │   ├── import.rs
│   │   │   │   ├── mod.rs
│   │   │   │   ├── oci.rs
│   │   │   │   ├── payload.rs
│   │   │   │   └── primitives.rs
│   │   │   ├── context.rs
│   │   │   ├── import.rs
│   │   │   ├── json_binding.rs
│   │   │   ├── main.rs
│   │   │   ├── payload.rs
│   │   │   └── platform.rs
│   │   └── Cargo.toml
│   ├── soda-install/
│   │   ├── src/
│   │   │   ├── candidate/
│   │   │   │   └── tests.rs
│   │   │   ├── console/
│   │   │   │   ├── mod.rs
│   │   │   │   ├── network.rs
│   │   │   │   ├── terminal.rs
│   │   │   │   ├── test_support.rs
│   │   │   │   └── tests.rs
│   │   │   ├── deliver/
│   │   │   │   └── tests.rs
│   │   │   ├── disks/
│   │   │   │   └── tests.rs
│   │   │   ├── enroll/
│   │   │   │   ├── keys/
│   │   │   │   │   ├── authorized_keys.rs
│   │   │   │   │   ├── directory.rs
│   │   │   │   │   ├── mod.rs
│   │   │   │   │   └── tests.rs
│   │   │   │   ├── arm.rs
│   │   │   │   ├── arm_tests.rs
│   │   │   │   ├── mod.rs
│   │   │   │   ├── native_config.rs
│   │   │   │   ├── peer.rs
│   │   │   │   ├── receive.rs
│   │   │   │   ├── selinux.rs
│   │   │   │   ├── serve.rs
│   │   │   │   ├── session.rs
│   │   │   │   ├── test_support.rs
│   │   │   │   └── tests.rs
│   │   │   ├── execute/
│   │   │   │   └── tests.rs
│   │   │   ├── inputs/
│   │   │   │   └── tests.rs
│   │   │   ├── netip/
│   │   │   │   ├── tests/
│   │   │   │   │   ├── address.rs
│   │   │   │   │   ├── mod.rs
│   │   │   │   │   └── prefix.rs
│   │   │   │   └── mod.rs
│   │   │   ├── oci/
│   │   │   │   ├── inspection.rs
│   │   │   │   ├── layout.rs
│   │   │   │   ├── metadata.rs
│   │   │   │   ├── mod.rs
│   │   │   │   ├── test_support.rs
│   │   │   │   └── tests.rs
│   │   │   ├── setup/
│   │   │   │   ├── tests/
│   │   │   │   │   ├── access.rs
│   │   │   │   │   ├── configure.rs
│   │   │   │   │   ├── fixtures.rs
│   │   │   │   │   └── mod.rs
│   │   │   │   ├── access.rs
│   │   │   │   ├── address.rs
│   │   │   │   ├── configure.rs
│   │   │   │   ├── local_ca.rs
│   │   │   │   └── mod.rs
│   │   │   ├── sshkey/
│   │   │   │   ├── authorized_keys.rs
│   │   │   │   ├── mod.rs
│   │   │   │   └── tests.rs
│   │   │   ├── wizard/
│   │   │   │   ├── mod.rs
│   │   │   │   ├── review.rs
│   │   │   │   ├── steps.rs
│   │   │   │   └── tests.rs
│   │   │   ├── x509/
│   │   │   │   ├── tests/
│   │   │   │   │   ├── algorithms.rs
│   │   │   │   │   ├── fixtures.rs
│   │   │   │   │   ├── mod.rs
│   │   │   │   │   ├── public_key.rs
│   │   │   │   │   ├── structure.rs
│   │   │   │   │   └── verify.rs
│   │   │   │   ├── mod.rs
│   │   │   │   └── verify.rs
│   │   │   ├── buildx.rs
│   │   │   ├── candidate.rs
│   │   │   ├── command.rs
│   │   │   ├── deliver.rs
│   │   │   ├── disks.rs
│   │   │   ├── errors.rs
│   │   │   ├── execute.rs
│   │   │   ├── hostadmit.rs
│   │   │   ├── inputs.rs
│   │   │   ├── jsongo.rs
│   │   │   ├── main.rs
│   │   │   ├── pemx.rs
│   │   │   ├── run.rs
│   │   │   └── signal.rs
│   │   └── Cargo.toml
│   ├── soda-muse/
│   │   ├── src/
│   │   │   ├── account.rs
│   │   │   ├── config.rs
│   │   │   ├── config_tests.rs
│   │   │   ├── execution.rs
│   │   │   ├── launch.rs
│   │   │   ├── launch_tests.rs
│   │   │   ├── launch_wire.rs
│   │   │   ├── main.rs
│   │   │   ├── paths.rs
│   │   │   ├── runtime.rs
│   │   │   ├── shell.rs
│   │   │   └── shell_tests.rs
│   │   └── Cargo.toml
│   ├── soda-muse-maintain/
│   │   ├── src/
│   │   │   ├── archive.rs
│   │   │   ├── archive_tests.rs
│   │   │   ├── command.rs
│   │   │   ├── config.rs
│   │   │   ├── config_tests.rs
│   │   │   ├── config_validation.rs
│   │   │   ├── config_wire.rs
│   │   │   ├── filesystem.rs
│   │   │   ├── filesystem_tests.rs
│   │   │   ├── interface.rs
│   │   │   ├── interface_admission.rs
│   │   │   ├── interface_tests.rs
│   │   │   ├── json.rs
│   │   │   ├── main.rs
│   │   │   ├── network.rs
│   │   │   ├── network_tests.rs
│   │   │   ├── options.rs
│   │   │   ├── options_tests.rs
│   │   │   ├── project.rs
│   │   │   ├── project_tests.rs
│   │   │   ├── release.rs
│   │   │   ├── release_tests.rs
│   │   │   ├── release_validation.rs
│   │   │   ├── release_wire.rs
│   │   │   ├── stage.rs
│   │   │   └── test_support.rs
│   │   └── Cargo.toml
│   ├── soda-pg-maintenance/
│   │   ├── src/
│   │   │   ├── bin/
│   │   │   │   ├── soda-pg-backup.rs
│   │   │   │   ├── soda-pg-init-roles.rs
│   │   │   │   └── soda-pg-restore.rs
│   │   │   └── lib.rs
│   │   └── Cargo.toml
│   ├── soda-project-terminal/
│   │   ├── src/
│   │   │   ├── bin/
│   │   │   │   ├── project-account.rs
│   │   │   │   └── project-factory-roles.rs
│   │   │   ├── factory_roles/
│   │   │   │   ├── accounts.rs
│   │   │   │   ├── accounts_tests.rs
│   │   │   │   ├── error.rs
│   │   │   │   ├── execution.rs
│   │   │   │   ├── execution_tests.rs
│   │   │   │   ├── inputs.rs
│   │   │   │   ├── inputs_tests.rs
│   │   │   │   ├── layout.rs
│   │   │   │   ├── mod.rs
│   │   │   │   ├── records.rs
│   │   │   │   ├── records_tests.rs
│   │   │   │   ├── sha_tests.rs
│   │   │   │   ├── tests.rs
│   │   │   │   ├── validate.rs
│   │   │   │   └── validate_tests.rs
│   │   │   ├── account.rs
│   │   │   ├── broker.rs
│   │   │   ├── broker_tests.rs
│   │   │   ├── cgroup.rs
│   │   │   ├── fs.rs
│   │   │   ├── fs_tests.rs
│   │   │   ├── key_lines.rs
│   │   │   ├── key_request.rs
│   │   │   ├── keys.rs
│   │   │   ├── keys_tests.rs
│   │   │   ├── lib.rs
│   │   │   ├── main.rs
│   │   │   ├── project_account.rs
│   │   │   ├── project_account_tests.rs
│   │   │   ├── proto.rs
│   │   │   ├── pty.rs
│   │   │   ├── pty_io.rs
│   │   │   ├── pty_process.rs
│   │   │   ├── pty_relay.rs
│   │   │   ├── pty_tests.rs
│   │   │   ├── socket.rs
│   │   │   ├── state_json.rs
│   │   │   ├── subscription_cgroup.rs
│   │   │   ├── subscription_credentials.rs
│   │   │   ├── subscription_prepare.rs
│   │   │   ├── subscription_profile.rs
│   │   │   ├── subscription_retire.rs
│   │   │   ├── subscription_start.rs
│   │   │   ├── subscription_wire.rs
│   │   │   ├── svc.rs
│   │   │   ├── svc_tests.rs
│   │   │   ├── sys.rs
│   │   │   ├── term.rs
│   │   │   ├── term_attach.rs
│   │   │   ├── term_binding.rs
│   │   │   ├── term_binding_tests.rs
│   │   │   ├── term_collect.rs
│   │   │   ├── term_create.rs
│   │   │   ├── term_native_tests.rs
│   │   │   ├── term_paths.rs
│   │   │   ├── term_prepare.rs
│   │   │   ├── term_protocol_tests.rs
│   │   │   ├── term_status.rs
│   │   │   ├── timex.rs
│   │   │   ├── timex_tests.rs
│   │   │   └── tmux.rs
│   │   ├── tests/
│   │   │   ├── cli.rs
│   │   │   ├── factory_roles_oracle.rs
│   │   │   └── project_account.rs
│   │   └── Cargo.toml
│   ├── soda-setup/
│   │   ├── src/
│   │   │   ├── tests/
│   │   │   │   ├── admission.rs
│   │   │   │   ├── encoding.rs
│   │   │   │   ├── fixtures.rs
│   │   │   │   ├── mod.rs
│   │   │   │   ├── postgres.rs
│   │   │   │   └── setup.rs
│   │   │   ├── cli.rs
│   │   │   ├── config.rs
│   │   │   ├── forgejo.rs
│   │   │   ├── json.rs
│   │   │   ├── main.rs
│   │   │   ├── origin.rs
│   │   │   ├── secrets.rs
│   │   │   ├── setup.rs
│   │   │   └── system.rs
│   │   └── Cargo.toml
│   └── soda-tailnet/
│       ├── command.go
│       ├── command_test.go
│       └── main.go
├── docs/
│   ├── architecture/
│   │   ├── factory-interfaces.md
│   │   ├── networking.md
│   │   ├── overview.md
│   │   ├── release.md
│   │   └── trust.md
│   ├── design/
│   │   ├── spaces/
│   │   │   ├── README.md
│   │   │   ├── drawer-compact.svg
│   │   │   ├── drawer-sessions.svg
│   │   │   ├── drawer-work.svg
│   │   │   ├── focus.svg
│   │   │   ├── mobile-states.svg
│   │   │   ├── render-sheets.ts
│   │   │   ├── sheets.css
│   │   │   ├── split.svg
│   │   │   └── tsconfig.tools.json
│   │   ├── avatars.md
│   │   ├── branding.md
│   │   ├── console-welcome.md
│   │   ├── forms.md
│   │   ├── screenshot-capture.md
│   │   └── spaces-ux.md
│   ├── development/
│   │   ├── README.md
│   │   ├── cockpit.md
│   │   ├── factory-implementation-plan.md
│   │   ├── forgejo-extensions-plan.md
│   │   ├── go-packages.md
│   │   ├── go.md
│   │   ├── ideal-filetree-plan/
│   │   │   ├── coverage/
│   │   │   │   ├── README.md
│   │   │   │   ├── inventory/
│   │   │   │   │   ├── README.md
│   │   │   │   │   ├── agent-plans.md
│   │   │   │   │   ├── appliance.md
│   │   │   │   │   ├── assets.md
│   │   │   │   │   ├── backend.md
│   │   │   │   │   ├── cmd.md
│   │   │   │   │   ├── docs.md
│   │   │   │   │   ├── factory-os.md
│   │   │   │   │   ├── frontend.md
│   │   │   │   │   ├── git-hooks.md
│   │   │   │   │   ├── native-packages.md
│   │   │   │   │   ├── project-os.md
│   │   │   │   │   ├── root-files.md
│   │   │   │   │   ├── scripts.md
│   │   │   │   │   ├── tests.md
│   │   │   │   │   └── tools.md
│   │   │   │   └── maps/
│   │   │   │       ├── README.md
│   │   │   │       ├── acceptance-execution-and-evidence.md
│   │   │   │       ├── acceptance-native-qualification.md
│   │   │   │       ├── appliance-definitions.md
│   │   │   │       ├── assets.md
│   │   │   │       ├── backend-acceptance.md
│   │   │   │       ├── backend-config.md
│   │   │   │       ├── backend-factory.md
│   │   │   │       ├── backend-forgejo.md
│   │   │   │       ├── backend-host.md
│   │   │   │       ├── backend-identity.md
│   │   │   │       ├── backend-project.md
│   │   │   │       ├── backend-store.md
│   │   │   │       ├── backend-tailnet.md
│   │   │   │       ├── backend-web-api.md
│   │   │   │       ├── backend-web-auth-and-composition.md
│   │   │   │       ├── browser-spaces.md
│   │   │   │       ├── browser-tailnet.md
│   │   │   │       ├── developer-scripts.md
│   │   │   │       ├── developer-tools.md
│   │   │   │       ├── documentation-contracts.md
│   │   │   │       ├── documentation-development-and-design.md
│   │   │   │       ├── factory-control-admission-and-dispatch.md
│   │   │   │       ├── factory-control-native-fixtures.md
│   │   │   │       ├── factory-control-publication-and-review.md
│   │   │   │       ├── host-factory-runs.md
│   │   │   │       ├── host-muse-execution.md
│   │   │   │       ├── host-preparation.md
│   │   │   │       ├── host-projects.md
│   │   │   │       ├── host-protocols-and-clients.md
│   │   │   │       ├── host-provider-execution.md
│   │   │   │       ├── host-runtime-composition.md
│   │   │   │       ├── host-service-admission.md
│   │   │   │       ├── host-tailnet-companions.md
│   │   │   │       ├── host-tailnet-control.md
│   │   │   │       ├── host-terminals.md
│   │   │   │       ├── identity-broker-protocol-and-policy.md
│   │   │   │       ├── identity-broker-state-and-entrypoints.md
│   │   │   │       ├── identity-providers.md
│   │   │   │       ├── installation-input-parsers.md
│   │   │   │       ├── installation-native-workflow.md
│   │   │   │       ├── installation-operator-enrollment.md
│   │   │   │       ├── project-system-definitions.md
│   │   │   │       ├── public-handbook.md
│   │   │   │       ├── root-and-retired-definitions.md
│   │   │   │       ├── server-entrypoints.md
│   │   │   │       ├── soda-activate.md
│   │   │   │       ├── soda-asset-fetchers.md
│   │   │   │       ├── soda-candidate-setup.md
│   │   │   │       ├── soda-console-welcome.md
│   │   │   │       ├── soda-factory.md
│   │   │   │       ├── soda-forgejo-domain.md
│   │   │   │       ├── soda-forgejo-locales.md
│   │   │   │       ├── soda-forgejo-migrate.md
│   │   │   │       ├── soda-identity-compose.md
│   │   │   │       ├── soda-image-import.md
│   │   │   │       ├── soda-muse-maintain.md
│   │   │   │       ├── soda-muse.md
│   │   │   │       ├── soda-pg-maintenance.md
│   │   │   │       ├── soda-project-account.md
│   │   │   │       ├── soda-project-factory-roles.md
│   │   │   │       ├── soda-project-terminal.md
│   │   │   │       ├── soda-release-build-implementation.md
│   │   │   │       ├── soda-release-build-verification.md
│   │   │   │       ├── soda-release-deliver-implementation.md
│   │   │   │       ├── soda-release-deliver-verification.md
│   │   │   │       ├── soda-release-image-implementation.md
│   │   │   │       ├── soda-release-image-verification.md
│   │   │   │       ├── soda-release-tools-implementation.md
│   │   │   │       ├── soda-release-tools-verification.md
│   │   │   │       ├── soda-rotate-lab-creds.md
│   │   │   │       ├── soda-setup.md
│   │   │   │       ├── soda-stage-render.md
│   │   │   │       ├── soda-test-vm.md
│   │   │   │       ├── tests-build.md
│   │   │   │       ├── tests-forgejo.md
│   │   │   │       ├── tests-frontend.md
│   │   │   │       └── tests-installed.md
│   │   │   ├── decomposition/
│   │   │   │   ├── README.md
│   │   │   │   ├── browser-and-design.md
│   │   │   │   ├── factory-coordination.md
│   │   │   │   ├── host-runtime.md
│   │   │   │   ├── identity-brokering.md
│   │   │   │   ├── installation-and-operations.md
│   │   │   │   ├── project-runtime.md
│   │   │   │   ├── release-production.md
│   │   │   │   ├── server-and-state.md
│   │   │   │   └── verification-and-support.md
│   │   │   ├── ideal-filetree-plan.md
│   │   │   ├── implementation-lanes.md
│   │   │   ├── implementation-tasks.md
│   │   │   ├── integration.md
│   │   │   ├── library-adoption.md
│   │   │   ├── maintenance.md
│   │   │   ├── package-ownership.md
│   │   │   ├── placement.md
│   │   │   ├── port-assessment.md
│   │   │   ├── proposed-tree.md
│   │   │   ├── review-assignments.md
│   │   │   ├── review-baseline.md
│   │   │   ├── review-format.md
│   │   │   ├── reviews/
│   │   │   │   ├── README.md
│   │   │   │   ├── D01.md
│   │   │   │   ├── D02.md
│   │   │   │   ├── D03.md
│   │   │   │   ├── D04.md
│   │   │   │   ├── D05.md
│   │   │   │   ├── D06.md
│   │   │   │   ├── D07.md
│   │   │   │   ├── D08.md
│   │   │   │   ├── D09.md
│   │   │   │   ├── D10.md
│   │   │   │   ├── D11.md
│   │   │   │   ├── F01.md
│   │   │   │   ├── F02.md
│   │   │   │   ├── F03.md
│   │   │   │   ├── F04.md
│   │   │   │   ├── F05.md
│   │   │   │   ├── F06.md
│   │   │   │   ├── F07.md
│   │   │   │   ├── F08.md
│   │   │   │   ├── F09.md
│   │   │   │   ├── F10.md
│   │   │   │   ├── F11.md
│   │   │   │   ├── F12.md
│   │   │   │   ├── G01.md
│   │   │   │   ├── G02.md
│   │   │   │   ├── G03.md
│   │   │   │   ├── G04.md
│   │   │   │   ├── G05.md
│   │   │   │   ├── G06.md
│   │   │   │   ├── G07.md
│   │   │   │   ├── G08.md
│   │   │   │   ├── G09.md
│   │   │   │   ├── H01.md
│   │   │   │   ├── H02.md
│   │   │   │   ├── H03.md
│   │   │   │   ├── H04.md
│   │   │   │   ├── H05.md
│   │   │   │   ├── H06.md
│   │   │   │   ├── I01.md
│   │   │   │   ├── I02.md
│   │   │   │   ├── I03.md
│   │   │   │   ├── I04.md
│   │   │   │   ├── I05.md
│   │   │   │   ├── I06.md
│   │   │   │   ├── I07.md
│   │   │   │   ├── I08.md
│   │   │   │   ├── I09.md
│   │   │   │   ├── I10.md
│   │   │   │   ├── N01.md
│   │   │   │   ├── N02.md
│   │   │   │   ├── N03.md
│   │   │   │   ├── N04.md
│   │   │   │   ├── N05.md
│   │   │   │   ├── N06.md
│   │   │   │   ├── N07.md
│   │   │   │   ├── O01.md
│   │   │   │   ├── O02.md
│   │   │   │   ├── O03.md
│   │   │   │   ├── O04.md
│   │   │   │   ├── O05.md
│   │   │   │   ├── O06.md
│   │   │   │   ├── O07.md
│   │   │   │   ├── P01.md
│   │   │   │   ├── P02.md
│   │   │   │   ├── P03.md
│   │   │   │   ├── P04.md
│   │   │   │   ├── P05.md
│   │   │   │   ├── P06.md
│   │   │   │   ├── P07.md
│   │   │   │   ├── P08.md
│   │   │   │   ├── P09.md
│   │   │   │   ├── P10.md
│   │   │   │   ├── P11.md
│   │   │   │   ├── P12.md
│   │   │   │   ├── S01.md
│   │   │   │   ├── S02.md
│   │   │   │   ├── S03.md
│   │   │   │   ├── S04.md
│   │   │   │   ├── S05.md
│   │   │   │   └── S06.md
│   │   │   └── slices/
│   │   │       ├── README.md
│   │   │       ├── factory-coordination.md
│   │   │       ├── forgejo-integration.md
│   │   │       ├── identity-brokering.md
│   │   │       ├── networking.md
│   │   │       ├── operator-administration.md
│   │   │       ├── projects.md
│   │   │       ├── release-and-installation.md
│   │   │       ├── shared-supporting-slices.md
│   │   │       └── spaces-and-terminals.md
│   │   ├── lit.md
│   │   ├── native-support.md
│   │   ├── python.md
│   │   ├── release.md
│   │   ├── testing.md
│   │   └── typescript.md
│   ├── factory/
│   │   └── decision-gate.md
│   ├── guides/
│   │   ├── develop.md
│   │   ├── installation.md
│   │   ├── local-testing.md
│   │   ├── media.md
│   │   ├── operator-setup.md
│   │   ├── project-clis.md
│   │   └── project-services.md
│   ├── operator/
│   │   └── enroll-key.md
│   ├── ops/
│   │   ├── gc-policy.md
│   │   └── retention.md
│   ├── product/
│   │   ├── overview.md
│   │   ├── projects.md
│   │   ├── scope.md
│   │   └── spaces.md
│   ├── public/
│   │   ├── 10-Start-here/
│   │   │   ├── 10-index.md
│   │   │   └── 20-product-model.md
│   │   ├── 20-Deploy/
│   │   │   ├── 05-verify-downloads.md
│   │   │   ├── 10-deploy-to-cloud.md
│   │   │   ├── 20-install-on-premises.md
│   │   │   ├── 25-operator-setup.md
│   │   │   └── 30-first-connection.md
│   │   ├── 30-Use-Soda/
│   │   │   ├── 05-dashboard.md
│   │   │   ├── 10-cockpit.md
│   │   │   ├── 15-software-factory.md
│   │   │   ├── 20-projects-and-workspaces.md
│   │   │   ├── 30-forgejo.md
│   │   │   ├── 35-collaboration.md
│   │   │   ├── 40-tailscale.md
│   │   │   ├── 50-ci-runners.md
│   │   │   └── 60-updates-and-fallback.md
│   │   ├── 40-Develop/
│   │   │   ├── 10-connect-and-develop.md
│   │   │   ├── 20-shared-tools-and-files.md
│   │   │   └── 30-project-services.md
│   │   ├── 50-Operate/
│   │   │   ├── 10-people-and-access.md
│   │   │   ├── 20-administration.md
│   │   │   ├── 30-backups-and-restoration.md
│   │   │   └── 40-data-safety-and-removal.md
│   │   └── README.md
│   ├── reference/
│   │   ├── api.md
│   │   ├── configuration.md
│   │   ├── credentials.md
│   │   ├── factory.md
│   │   ├── forgejo.md
│   │   ├── project-os.md
│   │   └── terminal.md
│   ├── research/
│   │   ├── factory-capability-map.md
│   │   ├── host-strategy.md
│   │   ├── library-reuse-coverage.md
│   │   ├── library-reuse-investigation.md
│   │   ├── licensing.md
│   │   ├── notices.md
│   │   ├── onedev.md
│   │   └── predecessor-reuse.md
│   └── README.md
├── frontend/
│   ├── forgejo/
│   │   ├── locales/
│   │   │   ├── README.md
│   │   │   └── en-US.ini
│   │   ├── templates/
│   │   │   ├── admin/
│   │   │   │   ├── applications/
│   │   │   │   │   ├── list.tmpl
│   │   │   │   │   └── oauth2_edit.tmpl
│   │   │   │   ├── auth/
│   │   │   │   │   ├── edit.tmpl
│   │   │   │   │   ├── edit_ldap.tmpl
│   │   │   │   │   ├── edit_oauth.tmpl
│   │   │   │   │   ├── edit_pam.tmpl
│   │   │   │   │   ├── edit_smtp.tmpl
│   │   │   │   │   ├── list.tmpl
│   │   │   │   │   └── new.tmpl
│   │   │   │   ├── emails/
│   │   │   │   │   └── list.tmpl
│   │   │   │   ├── repo/
│   │   │   │   │   └── list.tmpl
│   │   │   │   ├── runners/
│   │   │   │   │   └── create.tmpl
│   │   │   │   ├── user/
│   │   │   │   │   ├── edit.tmpl
│   │   │   │   │   ├── list.tmpl
│   │   │   │   │   └── new.tmpl
│   │   │   │   ├── config.tmpl
│   │   │   │   ├── cron.tmpl
│   │   │   │   ├── dashboard.tmpl
│   │   │   │   ├── hook_new.tmpl
│   │   │   │   ├── layout_head.tmpl
│   │   │   │   ├── notice.tmpl
│   │   │   │   ├── queue.tmpl
│   │   │   │   ├── queue_manage.tmpl
│   │   │   │   ├── self_check.tmpl
│   │   │   │   └── stacktrace.tmpl
│   │   │   ├── custom/
│   │   │   │   ├── soda/
│   │   │   │   │   ├── empty_content.tmpl
│   │   │   │   │   ├── guest_theme.tmpl
│   │   │   │   │   ├── milestone_row.tmpl
│   │   │   │   │   ├── notification_preview.tmpl
│   │   │   │   │   ├── page_intro.tmpl
│   │   │   │   │   ├── profile_block_dialog.tmpl
│   │   │   │   │   ├── profile_repositories.tmpl
│   │   │   │   │   └── theme_toggle.tmpl
│   │   │   │   ├── explore_empty.tmpl
│   │   │   │   ├── explore_navbar.tmpl
│   │   │   │   ├── extra_links.tmpl
│   │   │   │   ├── footer.tmpl
│   │   │   │   └── header.tmpl
│   │   │   ├── explore/
│   │   │   │   ├── code.tmpl
│   │   │   │   ├── repos.tmpl
│   │   │   │   └── users.tmpl
│   │   │   ├── moderation/
│   │   │   │   └── new_abuse_report.tmpl
│   │   │   ├── org/
│   │   │   │   ├── member/
│   │   │   │   │   └── members.tmpl
│   │   │   │   ├── projects/
│   │   │   │   │   ├── list.tmpl
│   │   │   │   │   ├── new.tmpl
│   │   │   │   │   └── view.tmpl
│   │   │   │   ├── settings/
│   │   │   │   │   ├── hook_new.tmpl
│   │   │   │   │   ├── layout_head.tmpl
│   │   │   │   │   ├── packages_cleanup_rules_edit.tmpl
│   │   │   │   │   └── runners_create.tmpl
│   │   │   │   ├── team/
│   │   │   │   │   ├── invite.tmpl
│   │   │   │   │   ├── members.tmpl
│   │   │   │   │   ├── new.tmpl
│   │   │   │   │   ├── repositories.tmpl
│   │   │   │   │   └── teams.tmpl
│   │   │   │   ├── create.tmpl
│   │   │   │   ├── header.tmpl
│   │   │   │   └── home.tmpl
│   │   │   ├── package/
│   │   │   │   ├── shared/
│   │   │   │   │   ├── cleanup_rules/
│   │   │   │   │   │   ├── edit.tmpl
│   │   │   │   │   │   ├── list.tmpl
│   │   │   │   │   │   └── preview.tmpl
│   │   │   │   │   ├── cargo.tmpl
│   │   │   │   │   ├── list.tmpl
│   │   │   │   │   └── versionlist.tmpl
│   │   │   │   ├── settings.tmpl
│   │   │   │   └── view.tmpl
│   │   │   ├── projects/
│   │   │   │   ├── list.tmpl
│   │   │   │   ├── new.tmpl
│   │   │   │   └── view.tmpl
│   │   │   ├── repo/
│   │   │   │   ├── actions/
│   │   │   │   │   ├── dispatch.tmpl
│   │   │   │   │   ├── list.tmpl
│   │   │   │   │   ├── list_inner.tmpl
│   │   │   │   │   ├── no_workflows.tmpl
│   │   │   │   │   ├── runs_list.tmpl
│   │   │   │   │   └── view.tmpl
│   │   │   │   ├── branch/
│   │   │   │   │   └── list.tmpl
│   │   │   │   ├── diff/
│   │   │   │   │   ├── box.tmpl
│   │   │   │   │   └── compare.tmpl
│   │   │   │   ├── editor/
│   │   │   │   │   ├── cherry_pick.tmpl
│   │   │   │   │   ├── commit_form.tmpl
│   │   │   │   │   ├── delete.tmpl
│   │   │   │   │   ├── edit.tmpl
│   │   │   │   │   ├── patch.tmpl
│   │   │   │   │   └── upload.tmpl
│   │   │   │   ├── find/
│   │   │   │   │   └── files.tmpl
│   │   │   │   ├── issue/
│   │   │   │   │   ├── choose.tmpl
│   │   │   │   │   ├── labels.tmpl
│   │   │   │   │   ├── list.tmpl
│   │   │   │   │   ├── milestone_issues.tmpl
│   │   │   │   │   ├── milestone_new.tmpl
│   │   │   │   │   ├── milestones.tmpl
│   │   │   │   │   ├── navbar.tmpl
│   │   │   │   │   ├── new.tmpl
│   │   │   │   │   └── view.tmpl
│   │   │   │   ├── migrate/
│   │   │   │   │   ├── codebase.tmpl
│   │   │   │   │   ├── git.tmpl
│   │   │   │   │   ├── gitbucket.tmpl
│   │   │   │   │   ├── gitea.tmpl
│   │   │   │   │   ├── github.tmpl
│   │   │   │   │   ├── gitlab.tmpl
│   │   │   │   │   ├── gogs.tmpl
│   │   │   │   │   ├── migrate.tmpl
│   │   │   │   │   ├── migrating.tmpl
│   │   │   │   │   ├── onedev.tmpl
│   │   │   │   │   ├── options.tmpl
│   │   │   │   │   └── pagure.tmpl
│   │   │   │   ├── projects/
│   │   │   │   │   ├── list.tmpl
│   │   │   │   │   ├── new.tmpl
│   │   │   │   │   └── view.tmpl
│   │   │   │   ├── pulls/
│   │   │   │   │   ├── commits.tmpl
│   │   │   │   │   ├── files.tmpl
│   │   │   │   │   ├── fork.tmpl
│   │   │   │   │   ├── status.tmpl
│   │   │   │   │   ├── tab_menu.tmpl
│   │   │   │   │   └── trust.tmpl
│   │   │   │   ├── release/
│   │   │   │   │   ├── list.tmpl
│   │   │   │   │   └── new.tmpl
│   │   │   │   ├── settings/
│   │   │   │   │   ├── units/
│   │   │   │   │   │   ├── issues.tmpl
│   │   │   │   │   │   ├── overview.tmpl
│   │   │   │   │   │   ├── pulls.tmpl
│   │   │   │   │   │   └── wiki.tmpl
│   │   │   │   │   ├── webhook/
│   │   │   │   │   │   ├── base.tmpl
│   │   │   │   │   │   ├── base_list.tmpl
│   │   │   │   │   │   ├── history.tmpl
│   │   │   │   │   │   └── new.tmpl
│   │   │   │   │   ├── actions.tmpl
│   │   │   │   │   ├── branches.tmpl
│   │   │   │   │   ├── collaboration.tmpl
│   │   │   │   │   ├── deploy_keys.tmpl
│   │   │   │   │   ├── githook_edit.tmpl
│   │   │   │   │   ├── githooks.tmpl
│   │   │   │   │   ├── layout_footer.tmpl
│   │   │   │   │   ├── layout_head.tmpl
│   │   │   │   │   ├── lfs.tmpl
│   │   │   │   │   ├── lfs_file.tmpl
│   │   │   │   │   ├── lfs_file_find.tmpl
│   │   │   │   │   ├── lfs_locks.tmpl
│   │   │   │   │   ├── lfs_pointers.tmpl
│   │   │   │   │   ├── navbar.tmpl
│   │   │   │   │   ├── options.tmpl
│   │   │   │   │   ├── options_danger.tmpl
│   │   │   │   │   ├── options_federation.tmpl
│   │   │   │   │   ├── options_maintenance.tmpl
│   │   │   │   │   ├── options_mirrors.tmpl
│   │   │   │   │   ├── options_modals.tmpl
│   │   │   │   │   ├── options_repository.tmpl
│   │   │   │   │   ├── options_trust.tmpl
│   │   │   │   │   ├── protected_branch.tmpl
│   │   │   │   │   ├── runner_create.tmpl
│   │   │   │   │   ├── runner_details.tmpl
│   │   │   │   │   ├── runner_edit.tmpl
│   │   │   │   │   ├── runner_setup.tmpl
│   │   │   │   │   ├── secrets.tmpl
│   │   │   │   │   ├── tags.tmpl
│   │   │   │   │   └── units.tmpl
│   │   │   │   ├── tag/
│   │   │   │   │   └── list.tmpl
│   │   │   │   ├── wiki/
│   │   │   │   │   ├── new.tmpl
│   │   │   │   │   ├── pages.tmpl
│   │   │   │   │   ├── revision.tmpl
│   │   │   │   │   ├── search.tmpl
│   │   │   │   │   ├── start.tmpl
│   │   │   │   │   └── view.tmpl
│   │   │   │   ├── activity.tmpl
│   │   │   │   ├── branch_dropdown.tmpl
│   │   │   │   ├── clone_buttons.tmpl
│   │   │   │   ├── commit_header.tmpl
│   │   │   │   ├── commit_page.tmpl
│   │   │   │   ├── commits.tmpl
│   │   │   │   ├── commits_list.tmpl
│   │   │   │   ├── commits_table.tmpl
│   │   │   │   ├── create.tmpl
│   │   │   │   ├── create_basic.tmpl
│   │   │   │   ├── empty.tmpl
│   │   │   │   ├── forks.tmpl
│   │   │   │   ├── graph.tmpl
│   │   │   │   ├── header.tmpl
│   │   │   │   ├── home.tmpl
│   │   │   │   ├── release_tag_header.tmpl
│   │   │   │   ├── search.tmpl
│   │   │   │   ├── sub_menu.tmpl
│   │   │   │   ├── user_cards.tmpl
│   │   │   │   ├── view_file.tmpl
│   │   │   │   ├── view_list.tmpl
│   │   │   │   └── watchers.tmpl
│   │   │   ├── shared/
│   │   │   │   ├── actions/
│   │   │   │   │   ├── runner_create.tmpl
│   │   │   │   │   ├── runner_details.tmpl
│   │   │   │   │   ├── runner_edit.tmpl
│   │   │   │   │   ├── runner_list.tmpl
│   │   │   │   │   └── runner_setup.tmpl
│   │   │   │   ├── secrets/
│   │   │   │   │   └── add_list.tmpl
│   │   │   │   ├── user/
│   │   │   │   │   └── profile_big_avatar.tmpl
│   │   │   │   ├── variables/
│   │   │   │   │   └── variable_list.tmpl
│   │   │   │   ├── blocked_users_list.tmpl
│   │   │   │   └── quota_overview.tmpl
│   │   │   ├── status/
│   │   │   │   ├── 404.tmpl
│   │   │   │   └── 413.tmpl
│   │   │   ├── user/
│   │   │   │   ├── auth/
│   │   │   │   │   ├── activate.tmpl
│   │   │   │   │   ├── change_passwd.tmpl
│   │   │   │   │   ├── forgot_passwd.tmpl
│   │   │   │   │   ├── grant.tmpl
│   │   │   │   │   ├── grant_error.tmpl
│   │   │   │   │   ├── link_account.tmpl
│   │   │   │   │   ├── prohibit_login.tmpl
│   │   │   │   │   ├── reset_passwd.tmpl
│   │   │   │   │   ├── signin.tmpl
│   │   │   │   │   ├── signin_openid.tmpl
│   │   │   │   │   ├── signup.tmpl
│   │   │   │   │   ├── signup_openid_connect.tmpl
│   │   │   │   │   ├── signup_openid_register.tmpl
│   │   │   │   │   ├── twofa.tmpl
│   │   │   │   │   ├── twofa_scratch.tmpl
│   │   │   │   │   └── webauthn.tmpl
│   │   │   │   ├── dashboard/
│   │   │   │   │   ├── dashboard.tmpl
│   │   │   │   │   ├── issues.tmpl
│   │   │   │   │   └── milestones.tmpl
│   │   │   │   ├── notification/
│   │   │   │   │   ├── notification_div.tmpl
│   │   │   │   │   └── notification_subscriptions.tmpl
│   │   │   │   ├── overview/
│   │   │   │   │   ├── header.tmpl
│   │   │   │   │   ├── package_versions.tmpl
│   │   │   │   │   └── packages.tmpl
│   │   │   │   ├── settings/
│   │   │   │   │   ├── security/
│   │   │   │   │   │   ├── accountlinks.tmpl
│   │   │   │   │   │   ├── openid.tmpl
│   │   │   │   │   │   ├── security.tmpl
│   │   │   │   │   │   ├── twofa.tmpl
│   │   │   │   │   │   ├── twofa_enroll.tmpl
│   │   │   │   │   │   └── webauthn.tmpl
│   │   │   │   │   ├── access_token_edit.tmpl
│   │   │   │   │   ├── account.tmpl
│   │   │   │   │   ├── actions.tmpl
│   │   │   │   │   ├── appearance.tmpl
│   │   │   │   │   ├── applications.tmpl
│   │   │   │   │   ├── applications_oauth2.tmpl
│   │   │   │   │   ├── applications_oauth2_edit.tmpl
│   │   │   │   │   ├── applications_oauth2_edit_form.tmpl
│   │   │   │   │   ├── applications_oauth2_list.tmpl
│   │   │   │   │   ├── blocked_users.tmpl
│   │   │   │   │   ├── grants_oauth2.tmpl
│   │   │   │   │   ├── hook_new.tmpl
│   │   │   │   │   ├── hooks.tmpl
│   │   │   │   │   ├── keys.tmpl
│   │   │   │   │   ├── keys_gpg.tmpl
│   │   │   │   │   ├── keys_principal.tmpl
│   │   │   │   │   ├── keys_ssh.tmpl
│   │   │   │   │   ├── layout_footer.tmpl
│   │   │   │   │   ├── layout_head.tmpl
│   │   │   │   │   ├── navbar.tmpl
│   │   │   │   │   ├── organization.tmpl
│   │   │   │   │   ├── packages.tmpl
│   │   │   │   │   ├── packages_cleanup_rules_edit.tmpl
│   │   │   │   │   ├── packages_cleanup_rules_preview.tmpl
│   │   │   │   │   ├── profile.tmpl
│   │   │   │   │   ├── repos.tmpl
│   │   │   │   │   ├── runner_create.tmpl
│   │   │   │   │   ├── runner_details.tmpl
│   │   │   │   │   ├── runner_edit.tmpl
│   │   │   │   │   ├── runner_setup.tmpl
│   │   │   │   │   └── storage_overview.tmpl
│   │   │   │   ├── code.tmpl
│   │   │   │   └── profile.tmpl
│   │   │   ├── webhook/
│   │   │   │   ├── new.tmpl
│   │   │   │   └── shared-settings.tmpl
│   │   │   ├── home.tmpl
│   │   │   ├── install.tmpl
│   │   │   └── post-install.tmpl
│   │   ├── README.md
│   │   ├── locale.lock.json
│   │   └── payload.json
│   ├── spaces/
│   │   ├── README.md
│   │   ├── soda-extension.ts
│   │   ├── soda-identity-response.ts
│   │   ├── soda-identity.ts
│   │   ├── soda-native-paths.ts
│   │   ├── soda-spaces-entry.ts
│   │   ├── soda-workspace-panel-entry.ts
│   │   ├── sodaspaces-api.ts
│   │   ├── sodaspaces-attention.ts
│   │   ├── sodaspaces-environment-view.ts
│   │   ├── sodaspaces-factory-navigation-view.ts
│   │   ├── sodaspaces-factory-response.ts
│   │   ├── sodaspaces-factory-screen.ts
│   │   ├── sodaspaces-factory-stream-response.ts
│   │   ├── sodaspaces-factory-view.ts
│   │   ├── sodaspaces-factory.ts
│   │   ├── sodaspaces-inventory-response.ts
│   │   ├── sodaspaces-keys-response.ts
│   │   ├── sodaspaces-layout.ts
│   │   ├── sodaspaces-network.ts
│   │   ├── sodaspaces-page.ts
│   │   ├── sodaspaces-project-access.ts
│   │   ├── sodaspaces-project-connection.ts
│   │   ├── sodaspaces-project-journey-view.ts
│   │   ├── sodaspaces-project-mutations.ts
│   │   ├── sodaspaces-project-network.ts
│   │   ├── sodaspaces-project-refresh.ts
│   │   ├── sodaspaces-project-request.ts
│   │   ├── sodaspaces-project-response.ts
│   │   ├── sodaspaces-project-runtime.ts
│   │   ├── sodaspaces-project-settings-view.ts
│   │   ├── sodaspaces-project-view.ts
│   │   ├── sodaspaces-project.css
│   │   ├── sodaspaces-project.ts
│   │   ├── sodaspaces-repository-picker-view.ts
│   │   ├── sodaspaces-repository-response.ts
│   │   ├── sodaspaces-session-navigation-view.ts
│   │   ├── sodaspaces-terminal-actions.ts
│   │   ├── sodaspaces-terminal-attachment.ts
│   │   ├── sodaspaces-terminal-dialog-view.ts
│   │   ├── sodaspaces-terminal-response.ts
│   │   ├── sodaspaces-terminal-screen.ts
│   │   ├── sodaspaces-terminal-view.ts
│   │   ├── sodaspaces-terminal.css
│   │   ├── sodaspaces-terminal.ts
│   │   ├── sodaspaces-workspace-drawer.ts
│   │   ├── sodaspaces-workspace-factory.ts
│   │   ├── sodaspaces-workspace-focus.ts
│   │   ├── sodaspaces-workspace-inventory.ts
│   │   ├── sodaspaces-workspace-layout.ts
│   │   ├── sodaspaces-workspace-measurement.ts
│   │   ├── sodaspaces-workspace-navigation.ts
│   │   ├── sodaspaces-workspace-pane-view.ts
│   │   ├── sodaspaces-workspace-setup.ts
│   │   ├── sodaspaces-workspace-shell-view.ts
│   │   ├── sodaspaces-workspace-terminal-hosts.ts
│   │   ├── sodaspaces-workspace-terminals.ts
│   │   ├── sodaspaces-workspace-toolbar-view.ts
│   │   ├── sodaspaces-workspace-types.ts
│   │   ├── sodaspaces-workspace-view.ts
│   │   ├── sodaspaces-workspace.css
│   │   ├── sodaspaces-workspace.ts
│   │   ├── sodaspaces.css
│   │   └── terminal-vendor.d.ts
│   └── tailnet/
│       ├── soda-settings.css
│       ├── soda-tailnet-actions.ts
│       ├── soda-tailnet-confirmation-view.ts
│       ├── soda-tailnet-enrollment-view.ts
│       ├── soda-tailnet-entry.ts
│       ├── soda-tailnet-host-view.ts
│       ├── soda-tailnet-observation.ts
│       ├── soda-tailnet-page.ts
│       ├── soda-tailnet-response.ts
│       └── soda-tailnet.css
├── internal/
│   ├── acceptance/
│   │   ├── developer_access.go
│   │   ├── developer_access_journey_test.go
│   │   ├── developer_access_request_test.go
│   │   ├── developer_access_session.go
│   │   ├── developer_access_test.go
│   │   ├── developer_access_transfer.go
│   │   ├── developer_access_users.go
│   │   ├── installed.go
│   │   ├── lifecycle_state.go
│   │   ├── lifecycle_state_test.go
│   │   ├── personal_git.go
│   │   ├── personal_git_exercise.go
│   │   ├── personal_git_keys.go
│   │   ├── personal_git_test.go
│   │   ├── personal_git_transport.go
│   │   ├── service_https.go
│   │   ├── service_https_test.go
│   │   ├── workload_access.go
│   │   ├── workload_access_test.go
│   │   ├── workload_exec.go
│   │   └── workload_exec_test.go
│   ├── archcheck/
│   │   └── arch_test.go
│   ├── avatar/
│   │   ├── testdata/
│   │   │   └── v1-snapshots.json
│   │   ├── README.md
│   │   ├── avatar.go
│   │   ├── avatar_test.go
│   │   └── style-v1.json
│   ├── config/
│   │   ├── background_test.go
│   │   ├── config.go
│   │   ├── config_test.go
│   │   ├── grant_key_test.go
│   │   ├── intake_test.go
│   │   ├── load_test.go
│   │   └── review_credential_test.go
│   ├── factory/
│   │   ├── control/
│   │   │   ├── acceptance.go
│   │   │   ├── acceptance_evidence.go
│   │   │   ├── acceptance_initial.go
│   │   │   ├── acceptance_initial_test.go
│   │   │   ├── acceptance_status.go
│   │   │   ├── acceptance_status_test.go
│   │   │   ├── acceptance_test.go
│   │   │   ├── checks_native_fixture_test.go
│   │   │   ├── checks_native_stale_test.go
│   │   │   ├── checks_native_test.go
│   │   │   ├── checks_pass.go
│   │   │   ├── checks_pass_test.go
│   │   │   ├── coordinator.go
│   │   │   ├── coordinator_test.go
│   │   │   ├── correction.go
│   │   │   ├── correction_test.go
│   │   │   ├── cycles.go
│   │   │   ├── dispatch.go
│   │   │   ├── dispatch_accounting_test.go
│   │   │   ├── dispatch_attempt.go
│   │   │   ├── dispatch_concurrency_test.go
│   │   │   ├── dispatch_fixture_test.go
│   │   │   ├── dispatch_inputs.go
│   │   │   ├── dispatch_launch.go
│   │   │   ├── dispatch_occupancy.go
│   │   │   ├── dispatch_recovery.go
│   │   │   ├── dispatch_recovery_test.go
│   │   │   ├── dispatch_registration.go
│   │   │   ├── dispatch_result.go
│   │   │   ├── dispatch_selection.go
│   │   │   ├── dispatch_test.go
│   │   │   ├── dispatch_wait_test.go
│   │   │   ├── grant_authority.go
│   │   │   ├── grants.go
│   │   │   ├── grants_test.go
│   │   │   ├── inventory_test.go
│   │   │   ├── lifecycle.go
│   │   │   ├── lifecycle_retry.go
│   │   │   ├── lifecycle_retry_test.go
│   │   │   ├── lifecycle_takeover.go
│   │   │   ├── lifecycle_takeover_test.go
│   │   │   ├── lifecycle_test.go
│   │   │   ├── merge.go
│   │   │   ├── merge_effect_test.go
│   │   │   ├── merge_evidence.go
│   │   │   ├── merge_fixture_test.go
│   │   │   ├── merge_native_completion_test.go
│   │   │   ├── merge_native_effect_test.go
│   │   │   ├── merge_native_fixture_test.go
│   │   │   ├── merge_native_setup_test.go
│   │   │   ├── merge_native_stale_test.go
│   │   │   ├── merge_native_test.go
│   │   │   ├── merge_reconcile.go
│   │   │   ├── merge_test.go
│   │   │   ├── merge_withdraw.go
│   │   │   ├── operator.go
│   │   │   ├── operator_test.go
│   │   │   ├── postgres_fixture_external_test.go
│   │   │   ├── postgres_fixture_test.go
│   │   │   ├── preparation_decisions.go
│   │   │   ├── prerequisites.go
│   │   │   ├── publication.go
│   │   │   ├── publication_authority.go
│   │   │   ├── publication_effect_test.go
│   │   │   ├── publication_export.go
│   │   │   ├── publication_fixture_test.go
│   │   │   ├── publication_hooks_test.go
│   │   │   ├── publication_native_effect_test.go
│   │   │   ├── publication_native_fixture_test.go
│   │   │   ├── publication_native_test.go
│   │   │   ├── publication_reconcile.go
│   │   │   ├── publication_recovery_test.go
│   │   │   ├── publication_test.go
│   │   │   ├── publication_withdraw.go
│   │   │   ├── readiness.go
│   │   │   ├── readiness_fixture_test.go
│   │   │   ├── readiness_prerequisite_test.go
│   │   │   ├── readiness_sweep.go
│   │   │   ├── readiness_sweep_test.go
│   │   │   ├── readiness_test.go
│   │   │   ├── readiness_visibility_test.go
│   │   │   ├── review_cycle.go
│   │   │   ├── review_cycle_test.go
│   │   │   ├── review_executor.go
│   │   │   ├── review_native_primitive_test.go
│   │   │   ├── settle.go
│   │   │   ├── settle_test.go
│   │   │   ├── st15_demo_accept_test.go
│   │   │   ├── st15_demo_broker_test.go
│   │   │   ├── st15_demo_coding_test.go
│   │   │   ├── st15_demo_completion_test.go
│   │   │   ├── st15_demo_coordinator_test.go
│   │   │   ├── st15_demo_journey_test.go
│   │   │   ├── st15_demo_native_test.go
│   │   │   ├── st15_demo_project_test.go
│   │   │   ├── st15_demo_review_test.go
│   │   │   ├── st15_demo_runs_test.go
│   │   │   ├── st15_demo_seed_test.go
│   │   │   ├── st15_demo_stack_test.go
│   │   │   ├── traversal.go
│   │   │   └── traversal_test.go
│   │   ├── acceptance.go
│   │   ├── acceptance_test.go
│   │   ├── allowance.go
│   │   ├── allowance_test.go
│   │   ├── assignment.go
│   │   ├── assignment_resources.go
│   │   ├── assignment_result.go
│   │   ├── assignment_test.go
│   │   ├── checks.go
│   │   ├── checks_assessment.go
│   │   ├── checks_observation.go
│   │   ├── checks_test.go
│   │   ├── dispatch_prompt.go
│   │   ├── effective_authority.go
│   │   ├── grants.go
│   │   ├── grants_test.go
│   │   ├── lifecycle.go
│   │   ├── lifecycle_test.go
│   │   ├── merge.go
│   │   ├── merge_evidence.go
│   │   ├── merge_operation.go
│   │   ├── merge_test.go
│   │   ├── publication.go
│   │   ├── publication_correction.go
│   │   ├── publication_intent.go
│   │   ├── publication_operation.go
│   │   ├── publication_refusal.go
│   │   ├── publication_test.go
│   │   ├── readiness.go
│   │   ├── readiness_test.go
│   │   ├── result.go
│   │   ├── review_native.go
│   │   ├── review_role_test.go
│   │   ├── run.go
│   │   ├── run_test.go
│   │   ├── sponsorship.go
│   │   ├── types.go
│   │   ├── views.go
│   │   └── views_test.go
│   ├── filelock/
│   │   ├── filelock.go
│   │   └── filelock_test.go
│   ├── forgejo/
│   │   ├── publish/
│   │   │   ├── candidate_validation.go
│   │   │   ├── credentials.go
│   │   │   ├── credentials_test.go
│   │   │   ├── git.go
│   │   │   ├── operation.go
│   │   │   ├── operation_git_test.go
│   │   │   ├── operation_observation.go
│   │   │   ├── operation_push.go
│   │   │   ├── operation_receipts.go
│   │   │   ├── operation_receipts_test.go
│   │   │   ├── operation_test.go
│   │   │   ├── publish.go
│   │   │   ├── publish_test.go
│   │   │   ├── review_role_test.go
│   │   │   └── source.go
│   │   ├── background.go
│   │   ├── background_admission.go
│   │   ├── background_test.go
│   │   ├── background_transport.go
│   │   ├── checks.go
│   │   ├── checks_test.go
│   │   ├── client.go
│   │   ├── client_test.go
│   │   ├── errors.go
│   │   ├── errors_test.go
│   │   ├── merge.go
│   │   ├── merge_completion.go
│   │   ├── merge_observation.go
│   │   ├── merge_test.go
│   │   ├── observe.go
│   │   ├── observe_test.go
│   │   ├── own_keys.go
│   │   ├── ownership.go
│   │   ├── publish.go
│   │   ├── publish_test.go
│   │   ├── repositories.go
│   │   ├── review.go
│   │   ├── review_test.go
│   │   ├── snapshot.go
│   │   ├── snapshot_issue.go
│   │   ├── snapshot_pull.go
│   │   ├── snapshot_request.go
│   │   ├── snapshot_test.go
│   │   ├── snapshot_transport.go
│   │   └── snapshot_transport_test.go
│   ├── host/
│   │   ├── access_keys.go
│   │   ├── client.go
│   │   ├── factory_candidate.go
│   │   ├── factory_candidate_test.go
│   │   ├── factory_client.go
│   │   ├── factory_export_test.go
│   │   ├── identity.go
│   │   ├── lifecycle.go
│   │   ├── os.go
│   │   ├── prepare.go
│   │   ├── prepare_test.go
│   │   ├── profiles.go
│   │   ├── tailnet.go
│   │   ├── tailnet_test.go
│   │   ├── terminal.go
│   │   ├── terminal_boundary_native_test.go
│   │   ├── terminal_client.go
│   │   └── terminal_native_test.go
│   ├── identity/
│   │   ├── client/
│   │   │   ├── broker_compat_test.go
│   │   │   ├── client.go
│   │   │   └── client_test.go
│   │   ├── enrollment.go
│   │   ├── event.go
│   │   ├── launch.go
│   │   ├── launch_test.go
│   │   ├── selection.go
│   │   ├── selection_test.go
│   │   ├── terminal.go
│   │   ├── transport.go
│   │   ├── types.go
│   │   └── types_test.go
│   ├── project/
│   │   ├── factory.go
│   │   ├── factory_candidate.go
│   │   ├── factory_candidate_test.go
│   │   ├── factory_export.go
│   │   ├── factory_export_test.go
│   │   ├── factory_harness.go
│   │   ├── factory_harness_test.go
│   │   ├── factory_output.go
│   │   ├── factory_output_test.go
│   │   ├── factory_test.go
│   │   ├── grants.go
│   │   ├── grants_test.go
│   │   ├── preparation.go
│   │   ├── preparation_test.go
│   │   ├── profile.go
│   │   ├── project.go
│   │   ├── project_test.go
│   │   ├── takeover.go
│   │   ├── takeover_test.go
│   │   └── types.go
│   ├── store/
│   │   ├── corruption.go
│   │   ├── ephemeral.go
│   │   ├── factory.go
│   │   ├── factory_assignments.go
│   │   ├── factory_checks.go
│   │   ├── factory_checks_test.go
│   │   ├── factory_dispatch_packet.go
│   │   ├── factory_dispatch_queue.go
│   │   ├── factory_dispatch_queue_test.go
│   │   ├── factory_dispatch_test.go
│   │   ├── factory_grants.go
│   │   ├── factory_grants_test.go
│   │   ├── factory_inventory.go
│   │   ├── factory_inventory_test.go
│   │   ├── factory_lifecycle.go
│   │   ├── factory_lifecycle_test.go
│   │   ├── factory_merges.go
│   │   ├── factory_merges_test.go
│   │   ├── factory_publication_intent_test.go
│   │   ├── factory_publications.go
│   │   ├── factory_publications_test.go
│   │   ├── factory_reservations.go
│   │   ├── factory_retry_packet.go
│   │   ├── factory_retry_packet_test.go
│   │   ├── factory_review_role_test.go
│   │   ├── factory_test.go
│   │   ├── factory_views.go
│   │   ├── factory_views_test.go
│   │   ├── grants.go
│   │   ├── grants_test.go
│   │   ├── identity_fixture.go
│   │   ├── identity_test.go
│   │   ├── issue_acceptances.go
│   │   ├── issue_acceptances_test.go
│   │   ├── issue_controls.go
│   │   ├── issue_controls_test.go
│   │   ├── members.go
│   │   ├── observe.go
│   │   ├── observe_test.go
│   │   ├── postgres_fixture_test.go
│   │   ├── preparation.go
│   │   ├── preparation_test.go
│   │   ├── project_grants.go
│   │   ├── project_grants_test.go
│   │   ├── project_profile_test.go
│   │   ├── schema.go
│   │   ├── schema_test.go
│   │   ├── staged_seed.go
│   │   ├── store.go
│   │   └── store_test.go
│   ├── strictjson/
│   │   ├── decode.go
│   │   └── decode_test.go
│   ├── tailnet/
│   │   ├── control_types.go
│   │   ├── control_validation.go
│   │   ├── project_runtime.go
│   │   ├── tailnet.go
│   │   └── tailnet_test.go
│   └── web/
│       ├── api/
│       │   ├── access_keys.go
│       │   ├── api.go
│       │   ├── dispatch_inputs.go
│       │   ├── dispatch_inputs_test.go
│       │   ├── environment_authority.go
│       │   ├── environment_os.go
│       │   ├── environments_api.go
│       │   ├── environments_create.go
│       │   ├── environments_join.go
│       │   ├── environments_join_test.go
│       │   ├── environments_preparation.go
│       │   ├── extension.go
│       │   ├── extension_native.go
│       │   ├── extension_terminal.go
│       │   ├── extension_terminal_authority.go
│       │   ├── extension_terminal_stream.go
│       │   ├── extension_test.go
│       │   ├── factory_assignments.go
│       │   ├── factory_assignments_test.go
│       │   ├── factory_intake.go
│       │   ├── factory_intake_test.go
│       │   ├── factory_issue_view.go
│       │   ├── factory_lifecycle.go
│       │   ├── factory_output.go
│       │   ├── factory_output_test.go
│       │   ├── factory_policy.go
│       │   ├── factory_readiness.go
│       │   ├── factory_settings.go
│       │   ├── factory_sponsorship.go
│       │   ├── factory_status.go
│       │   ├── factory_views.go
│       │   ├── factory_views_test.go
│       │   ├── identity.go
│       │   ├── identity_grants.go
│       │   ├── identity_launch.go
│       │   ├── issue_acceptance_evidence.go
│       │   ├── issue_acceptances.go
│       │   ├── issue_acceptances_test.go
│       │   ├── lifecycle.go
│       │   ├── operator.go
│       │   ├── preparation_decisions.go
│       │   ├── project_profiles.go
│       │   ├── provisioning.go
│       │   ├── repositories.go
│       │   ├── spaces.go
│       │   ├── spaces_authority.go
│       │   ├── spaces_inspection.go
│       │   ├── spaces_inventory.go
│       │   ├── tailnet.go
│       │   └── terminal_registry.go
│       ├── auth/
│       │   ├── auth.go
│       │   ├── development_key.go
│       │   ├── errors.go
│       │   ├── errors_test.go
│       │   ├── extension.go
│       │   ├── extension_service.go
│       │   ├── extension_test.go
│       │   ├── forgejo_keys.go
│       │   ├── http.go
│       │   ├── postgres_fixture_test.go
│       │   ├── service.go
│       │   └── session.go
│       ├── testdata/
│       │   └── avatar-browser.ts
│       ├── avatars.go
│       ├── avatars_browser_test.go
│       ├── avatars_test.go
│       ├── browser_join_test.go
│       ├── environment_authority_test.go
│       ├── environment_os_test.go
│       ├── environment_read_publication_test.go
│       ├── environments_api_test.go
│       ├── execution_access_test.go
│       ├── extension.go
│       ├── extension_terminal_test.go
│       ├── extension_test.go
│       ├── factory_checks_view_test.go
│       ├── factory_intake_route_test.go
│       ├── factory_lifecycle_test.go
│       ├── factory_output_fixture_test.go
│       ├── factory_output_stream_test.go
│       ├── factory_settings_test.go
│       ├── factory_views_test.go
│       ├── forgejo_keys_test.go
│       ├── identity_native_test.go
│       ├── issue_acceptances_test.go
│       ├── join_boundary_test.go
│       ├── lifecycle_access_keys_test.go
│       ├── mutation_admission_test.go
│       ├── postgres_fixture_test.go
│       ├── preparation_test.go
│       ├── project_profiles_test.go
│       ├── provisioning_lifetime_test.go
│       ├── repositories_test.go
│       ├── repository_access_test.go
│       ├── repository_settings_test.go
│       ├── retired_frontend_test.go
│       ├── server.go
│       ├── server_test.go
│       ├── spaces_inventory_test.go
│       ├── spaces_test.go
│       ├── tailnet_test.go
│       ├── terminal_test.go
│       └── test_helpers_test.go
├── lib/
│   ├── unix-http/
│   │   ├── src/
│   │   │   └── lib.rs
│   │   └── Cargo.toml
│   ├── host/
│   │   ├── src/
│   │   │   ├── account/
│   │   │   │   ├── access_keys_tests.rs
│   │   │   │   ├── mod.rs
│   │   │   │   └── tests.rs
│   │   │   ├── daemon/
│   │   │   │   ├── admission.rs
│   │   │   │   ├── backend.rs
│   │   │   │   ├── broker.rs
│   │   │   │   ├── config.rs
│   │   │   │   ├── http.rs
│   │   │   │   ├── mod.rs
│   │   │   │   ├── response.rs
│   │   │   │   ├── routes.rs
│   │   │   │   └── websocket.rs
│   │   │   ├── domain/
│   │   │   │   ├── account.rs
│   │   │   │   ├── mod.rs
│   │   │   │   ├── os.rs
│   │   │   │   ├── profile.rs
│   │   │   │   └── tests.rs
│   │   │   ├── factory/
│   │   │   │   ├── tests/
│   │   │   │   │   ├── artifacts.rs
│   │   │   │   │   ├── candidate.rs
│   │   │   │   │   ├── common.rs
│   │   │   │   │   ├── confirmation.rs
│   │   │   │   │   ├── finish.rs
│   │   │   │   │   ├── launch.rs
│   │   │   │   │   ├── mocks.rs
│   │   │   │   │   ├── mod.rs
│   │   │   │   │   ├── receipt.rs
│   │   │   │   │   ├── stop.rs
│   │   │   │   │   └── wire.rs
│   │   │   │   ├── artifacts.rs
│   │   │   │   ├── candidate.rs
│   │   │   │   ├── confirmation.rs
│   │   │   │   ├── deadline.rs
│   │   │   │   ├── finish.rs
│   │   │   │   ├── identity.rs
│   │   │   │   ├── inspect.rs
│   │   │   │   ├── launch.rs
│   │   │   │   ├── mod.rs
│   │   │   │   ├── receipt.rs
│   │   │   │   ├── requests.rs
│   │   │   │   ├── run.rs
│   │   │   │   ├── state.rs
│   │   │   │   └── stop.rs
│   │   │   ├── json/
│   │   │   │   ├── mod.rs
│   │   │   │   ├── number.rs
│   │   │   │   └── strict_tests.rs
│   │   │   ├── muse/
│   │   │   │   ├── tests/
│   │   │   │   │   ├── caller.rs
│   │   │   │   │   ├── common.rs
│   │   │   │   │   ├── execution.rs
│   │   │   │   │   ├── mod.rs
│   │   │   │   │   ├── program.rs
│   │   │   │   │   ├── socket.rs
│   │   │   │   │   └── wire.rs
│   │   │   │   ├── arguments.rs
│   │   │   │   ├── caller.rs
│   │   │   │   ├── cleanup.rs
│   │   │   │   ├── config.rs
│   │   │   │   ├── execution.rs
│   │   │   │   ├── launch.rs
│   │   │   │   ├── mod.rs
│   │   │   │   ├── nested.rs
│   │   │   │   ├── program.rs
│   │   │   │   ├── socket.rs
│   │   │   │   └── wire.rs
│   │   │   ├── preparation/
│   │   │   │   ├── candidate.rs
│   │   │   │   ├── decisions.rs
│   │   │   │   ├── mod.rs
│   │   │   │   ├── setup.rs
│   │   │   │   ├── state.rs
│   │   │   │   ├── validation_tests.rs
│   │   │   │   └── wire_tests.rs
│   │   │   ├── prepare/
│   │   │   │   ├── candidate.rs
│   │   │   │   ├── candidate_tests.rs
│   │   │   │   ├── execution_tests.rs
│   │   │   │   ├── helper.rs
│   │   │   │   ├── mod.rs
│   │   │   │   ├── paths.rs
│   │   │   │   ├── source.rs
│   │   │   │   ├── state.rs
│   │   │   │   ├── tests.rs
│   │   │   │   └── tools.rs
│   │   │   ├── project/
│   │   │   │   ├── confirmation.rs
│   │   │   │   ├── confirmation_tests.rs
│   │   │   │   ├── connection.rs
│   │   │   │   ├── create.rs
│   │   │   │   ├── executor.rs
│   │   │   │   ├── inspect.rs
│   │   │   │   ├── lifecycle.rs
│   │   │   │   ├── lifecycle_tests.rs
│   │   │   │   ├── mod.rs
│   │   │   │   ├── operations.rs
│   │   │   │   ├── os.rs
│   │   │   │   ├── profile.rs
│   │   │   │   └── tests.rs
│   │   │   ├── ssh/
│   │   │   │   ├── material.rs
│   │   │   │   ├── mod.rs
│   │   │   │   └── tests.rs
│   │   │   ├── tailnet/
│   │   │   │   ├── companion/
│   │   │   │   │   ├── enroll.rs
│   │   │   │   │   ├── execute.rs
│   │   │   │   │   ├── identity.rs
│   │   │   │   │   ├── identity_tests.rs
│   │   │   │   │   ├── lifecycle_tests.rs
│   │   │   │   │   ├── mod.rs
│   │   │   │   │   ├── start.rs
│   │   │   │   │   ├── stop.rs
│   │   │   │   │   ├── tests.rs
│   │   │   │   │   ├── view.rs
│   │   │   │   │   └── view_tests.rs
│   │   │   │   ├── control/
│   │   │   │   │   ├── enrollment.rs
│   │   │   │   │   ├── mod.rs
│   │   │   │   │   ├── native.rs
│   │   │   │   │   ├── policy.rs
│   │   │   │   │   ├── project.rs
│   │   │   │   │   ├── provider.rs
│   │   │   │   │   ├── tests.rs
│   │   │   │   │   └── wire.rs
│   │   │   │   ├── forgejo.rs
│   │   │   │   ├── domain/
│   │   │   │   │   ├── address_tests.rs
│   │   │   │   │   ├── addresses.rs
│   │   │   │   │   ├── mod.rs
│   │   │   │   │   ├── native.rs
│   │   │   │   │   ├── node_tests.rs
│   │   │   │   │   ├── status.rs
│   │   │   │   │   ├── status_tests.rs
│   │   │   │   │   └── time.rs
│   │   │   │   ├── files/
│   │   │   │   │   ├── keys.rs
│   │   │   │   │   ├── mod.rs
│   │   │   │   │   ├── resolver.rs
│   │   │   │   │   ├── run.rs
│   │   │   │   │   └── tests.rs
│   │   │   │   ├── runtime/
│   │   │   │   │   ├── mod.rs
│   │   │   │   │   ├── process.rs
│   │   │   │   │   ├── project.rs
│   │   │   │   │   ├── project_tests.rs
│   │   │   │   │   ├── tests.rs
│   │   │   │   │   └── wire.rs
│   │   │   │   └── mod.rs
│   │   │   ├── terminal/
│   │   │   │   ├── codex/
│   │   │   │   │   ├── tests/
│   │   │   │   │   │   ├── artifacts.rs
│   │   │   │   │   │   ├── common.rs
│   │   │   │   │   │   ├── lifecycle.rs
│   │   │   │   │   │   ├── mod.rs
│   │   │   │   │   │   ├── reserve.rs
│   │   │   │   │   │   └── wire.rs
│   │   │   │   │   ├── artifacts.rs
│   │   │   │   │   ├── commands.rs
│   │   │   │   │   ├── mod.rs
│   │   │   │   │   ├── paths.rs
│   │   │   │   │   ├── reserve.rs
│   │   │   │   │   ├── start.rs
│   │   │   │   │   └── stop.rs
│   │   │   │   ├── factory/
│   │   │   │   │   ├── muse/
│   │   │   │   │   │   ├── tests/
│   │   │   │   │   │   │   ├── commands.rs
│   │   │   │   │   │   │   ├── common.rs
│   │   │   │   │   │   │   ├── lifecycle.rs
│   │   │   │   │   │   │   ├── mod.rs
│   │   │   │   │   │   │   ├── paths.rs
│   │   │   │   │   │   │   ├── reserve.rs
│   │   │   │   │   │   │   └── start.rs
│   │   │   │   │   │   ├── commands.rs
│   │   │   │   │   │   ├── lifecycle.rs
│   │   │   │   │   │   ├── mod.rs
│   │   │   │   │   │   ├── paths.rs
│   │   │   │   │   │   ├── reserve.rs
│   │   │   │   │   │   └── start.rs
│   │   │   │   │   ├── tests/
│   │   │   │   │   │   ├── artifacts.rs
│   │   │   │   │   │   ├── mod.rs
│   │   │   │   │   │   └── run.rs
│   │   │   │   │   ├── artifacts.rs
│   │   │   │   │   ├── binding.rs
│   │   │   │   │   ├── lifecycle.rs
│   │   │   │   │   ├── mod.rs
│   │   │   │   │   ├── native.rs
│   │   │   │   │   ├── output.rs
│   │   │   │   │   └── run.rs
│   │   │   │   ├── tests/
│   │   │   │   │   ├── common.rs
│   │   │   │   │   ├── identity.rs
│   │   │   │   │   ├── identity_wire.rs
│   │   │   │   │   ├── launch.rs
│   │   │   │   │   ├── mod.rs
│   │   │   │   │   ├── protocol.rs
│   │   │   │   │   └── target.rs
│   │   │   │   ├── frame.rs
│   │   │   │   ├── identity.rs
│   │   │   │   ├── identity_protocol.rs
│   │   │   │   ├── identity_wire.rs
│   │   │   │   ├── launch.rs
│   │   │   │   ├── lease.rs
│   │   │   │   ├── mod.rs
│   │   │   │   ├── native.rs
│   │   │   │   ├── protocol.rs
│   │   │   │   ├── request.rs
│   │   │   │   └── target.rs
│   │   │   ├── lib.rs
│   │   │   └── net.rs
│   │   ├── tests/
│   │   │   ├── common/
│   │   │   │   ├── backend.rs
│   │   │   │   └── mod.rs
│   │   │   ├── data/
│   │   │   │   └── iconfig/
│   │   │   │       ├── bad-bridge.json
│   │   │   │       ├── bad-network.json
│   │   │   │       ├── bad-release.json
│   │   │   │       ├── bad-subnet.json
│   │   │   │       ├── bad-type-bool.json
│   │   │   │       ├── bad-type-string.json
│   │   │   │       ├── conflict-image.json
│   │   │   │       ├── conflict-tailnet.json
│   │   │   │       ├── dash-image.json
│   │   │   │       ├── duplicates.json
│   │   │   │       ├── empty-image.json
│   │   │   │       ├── empty.json
│   │   │   │       ├── identity-bad-sha.json
│   │   │   │       ├── identity-bad-version.json
│   │   │   │       ├── identity-relative-harness.json
│   │   │   │       ├── identity-skipped.json
│   │   │   │       ├── muse-bad-base.json
│   │   │   │       ├── muse-bad-digest.json
│   │   │   │       ├── muse-no-version.json
│   │   │   │       ├── muse-relative-identity.json
│   │   │   │       ├── muse-relative-socket.json
│   │   │   │       ├── muse-skipped.json
│   │   │   │       ├── null.json
│   │   │   │       ├── order-identity-before-tailnet.json
│   │   │   │       ├── order-muse-before-subnet.json
│   │   │   │       ├── order-subnet-before-network.json
│   │   │   │       ├── order-tailnet-before-network.json
│   │   │   │       ├── overlay-base.json
│   │   │   │       ├── overlay-nomgmt.json
│   │   │   │       ├── release.json
│   │   │   │       ├── tailnet-bad-ref.json
│   │   │   │       ├── tailnet-no-mgmt.json
│   │   │   │       ├── trailing.json
│   │   │   │       ├── unknown-field.json
│   │   │   │       ├── valid-full.json
│   │   │   │       └── valid-minimal.json
│   │   │   ├── identity_transport/
│   │   │   │   ├── common.rs
│   │   │   │   ├── main.rs
│   │   │   │   ├── requests.rs
│   │   │   │   └── responses.rs
│   │   │   ├── project_operations/
│   │   │   │   ├── access_keys.rs
│   │   │   │   ├── accounts.rs
│   │   │   │   ├── candidate.rs
│   │   │   │   ├── common.rs
│   │   │   │   ├── hold.rs
│   │   │   │   ├── main.rs
│   │   │   │   ├── preparation.rs
│   │   │   │   └── requests.rs
│   │   │   ├── daemon.rs
│   │   │   └── terminal_transport.rs
│   │   └── Cargo.toml
│   ├── release-inputs/
│   │   ├── src/
│   │   │   ├── reader/
│   │   │   │   ├── forgejo.rs
│   │   │   │   ├── muse.rs
│   │   │   │   ├── settings.rs
│   │   │   │   ├── signature.rs
│   │   │   │   ├── stream.rs
│   │   │   │   └── url.rs
│   │   │   ├── lib.rs
│   │   │   ├── reader.rs
│   │   │   └── trust_key.rs
│   │   └── Cargo.toml
│   ├── soda-release-build/
│   │   ├── src/
│   │   │   ├── coreos/
│   │   │   │   └── process.rs
│   │   │   ├── files/
│   │   │   │   └── tests.rs
│   │   │   ├── json_go/
│   │   │   │   └── tests.rs
│   │   │   ├── oci/
│   │   │   │   ├── content.rs
│   │   │   │   ├── manifest.rs
│   │   │   │   └── tests.rs
│   │   │   ├── production/
│   │   │   │   └── tests.rs
│   │   │   ├── confined_files.rs
│   │   │   ├── coreos.rs
│   │   │   ├── coreos_iso.rs
│   │   │   ├── coreos_registry.rs
│   │   │   ├── coreos_stream.rs
│   │   │   ├── elf.rs
│   │   │   ├── files.rs
│   │   │   ├── forgejo.rs
│   │   │   ├── http.rs
│   │   │   ├── json_emit.rs
│   │   │   ├── json_input.rs
│   │   │   ├── lib.rs
│   │   │   ├── live_inputs.rs
│   │   │   ├── oci.rs
│   │   │   ├── oci_layout.rs
│   │   │   ├── production.rs
│   │   │   ├── production_assets.rs
│   │   │   ├── production_compile.rs
│   │   │   ├── production_images.rs
│   │   │   ├── production_inputs.rs
│   │   │   ├── tailnet_inputs.rs
│   │   │   └── test_support.rs
│   │   ├── tests/
│   │   │   ├── data/
│   │   │   │   ├── go-layout/
│   │   │   │   │   ├── blobs/
│   │   │   │   │   │   └── sha256/
│   │   │   │   │   │       ├── 098b60ba449c4b81d38cca87e08b16ff83522b9edb36bdb36025d9d370a99295
│   │   │   │   │   │       ├── 9265b4ccf05645aeded07db50e8b29fd99e83b6b4994a4ae9e952aca54c3f691
│   │   │   │   │   │       └── 973a9bd7fe238b1604434f944e1ad4bff0629795321210dc11fcecf853ad3dce
│   │   │   │   │   ├── index.json
│   │   │   │   │   └── oci-layout
│   │   │   │   └── go-fixture.oci
│   │   │   ├── oracle/
│   │   │   │   ├── inputs.rs
│   │   │   │   ├── oci.rs
│   │   │   │   └── production.rs
│   │   │   ├── oracle.rs
│   │   │   └── oracle_vectors.rs
│   │   └── Cargo.toml
│   ├── soda-release-deliver/
│   │   ├── src/
│   │   │   ├── buildx/
│   │   │   │   ├── filesystem.rs
│   │   │   │   ├── mod.rs
│   │   │   │   └── tests.rs
│   │   │   ├── fetch/
│   │   │   │   ├── mod.rs
│   │   │   │   ├── state.rs
│   │   │   │   ├── tests.rs
│   │   │   │   └── verification.rs
│   │   │   ├── json_serde.rs
│   │   │   ├── model/
│   │   │   │   ├── candidate.rs
│   │   │   │   ├── channel.rs
│   │   │   │   ├── mod.rs
│   │   │   │   ├── release.rs
│   │   │   │   ├── tests.rs
│   │   │   │   └── trust.rs
│   │   │   ├── native/
│   │   │   │   ├── mod.rs
│   │   │   │   ├── policy.rs
│   │   │   │   └── sign.rs
│   │   │   ├── oci/
│   │   │   │   ├── archive.rs
│   │   │   │   ├── layers.rs
│   │   │   │   ├── mod.rs
│   │   │   │   └── schema.rs
│   │   │   ├── payload/
│   │   │   │   └── tests.rs
│   │   │   ├── publish/
│   │   │   │   ├── channel.rs
│   │   │   │   ├── ledger.rs
│   │   │   │   └── mod.rs
│   │   │   ├── admission.rs
│   │   │   ├── check.rs
│   │   │   ├── content.rs
│   │   │   ├── document.rs
│   │   │   ├── finalize.rs
│   │   │   ├── import.rs
│   │   │   ├── lib.rs
│   │   │   ├── payload.rs
│   │   │   └── prepare.rs
│   │   ├── tests/
│   │   │   ├── goldens/
│   │   │   │   └── deliver.json
│   │   │   └── oracle/
│   │   │       ├── artifacts.rs
│   │   │       ├── fetch_state.rs
│   │   │       └── main.rs
│   │   ├── Cargo.toml
│   │   └── tools.json
│   ├── soda-release-image/
│   │   ├── src/
│   │   │   ├── build_context/
│   │   │   │   └── tests.rs
│   │   │   ├── build_runner/
│   │   │   │   └── tests.rs
│   │   │   ├── media/
│   │   │   │   └── tests.rs
│   │   │   ├── model/
│   │   │   │   ├── candidate.rs
│   │   │   │   ├── images.rs
│   │   │   │   ├── live_inputs.rs
│   │   │   │   ├── payload.rs
│   │   │   │   ├── tests.rs
│   │   │   │   ├── trust.rs
│   │   │   │   └── url.rs
│   │   │   ├── prepare/
│   │   │   │   └── tests.rs
│   │   │   ├── build.rs
│   │   │   ├── build_candidate.rs
│   │   │   ├── build_compile.rs
│   │   │   ├── build_context.rs
│   │   │   ├── build_media.rs
│   │   │   ├── build_runner.rs
│   │   │   ├── build_source.rs
│   │   │   ├── complete.rs
│   │   │   ├── compression.rs
│   │   │   ├── error.rs
│   │   │   ├── events.rs
│   │   │   ├── extension.rs
│   │   │   ├── files.rs
│   │   │   ├── foreign.rs
│   │   │   ├── forgejo.rs
│   │   │   ├── host.rs
│   │   │   ├── ignition.rs
│   │   │   ├── inspect.rs
│   │   │   ├── jsonio.rs
│   │   │   ├── layout.rs
│   │   │   ├── lib.rs
│   │   │   ├── media.rs
│   │   │   ├── media_assembler.rs
│   │   │   ├── media_authentication.rs
│   │   │   ├── media_container.rs
│   │   │   ├── media_installer.rs
│   │   │   ├── model.rs
│   │   │   ├── ordered_json.rs
│   │   │   ├── packages.rs
│   │   │   ├── payload_stage.rs
│   │   │   ├── prepare.rs
│   │   │   ├── quadlet.rs
│   │   │   ├── recall.rs
│   │   │   ├── record.rs
│   │   │   ├── request.rs
│   │   │   ├── rootfs.rs
│   │   │   └── sys.rs
│   │   ├── tests/
│   │   │   ├── oracle/
│   │   │   │   ├── host.rs
│   │   │   │   ├── media.rs
│   │   │   │   └── staging.rs
│   │   │   └── oracle.rs
│   │   └── Cargo.toml
│   └── soda-release-tools/
│       ├── src/
│       │   ├── artifacts/
│       │   │   └── tests.rs
│       │   ├── bin/
│       │   │   ├── soda-artifacts.rs
│       │   │   ├── soda-build.rs
│       │   │   ├── soda-candidate-check.rs
│       │   │   └── soda-candidate.rs
│       │   ├── build_cli/
│       │   │   └── tests.rs
│       │   ├── candidate/
│       │   │   ├── mod.rs
│       │   │   ├── options.rs
│       │   │   └── tests.rs
│       │   ├── candidate_display/
│       │   │   ├── events.rs
│       │   │   ├── mod.rs
│       │   │   └── tests.rs
│       │   ├── candidate_fixture/
│       │   │   └── tests.rs
│       │   ├── candidate_prompts/
│       │   │   ├── defaults.rs
│       │   │   ├── mod.rs
│       │   │   └── tests.rs
│       │   ├── progress/
│       │   │   └── tests.rs
│       │   ├── worker/
│       │   │   ├── config.rs
│       │   │   ├── execution.rs
│       │   │   ├── mod.rs
│       │   │   ├── runtime.rs
│       │   │   └── tests.rs
│       │   ├── artifacts.rs
│       │   ├── build_cli.rs
│       │   ├── build_spec.rs
│       │   ├── candidate_check.rs
│       │   ├── candidate_controller.rs
│       │   ├── candidate_fixture.rs
│       │   ├── candidate_hints.rs
│       │   ├── digest.rs
│       │   ├── exitcode.rs
│       │   ├── lib.rs
│       │   └── progress.rs
│       ├── tests/
│       │   └── cli/
│       │       ├── main.rs
│       │       ├── soda_artifacts.rs
│       │       ├── soda_build.rs
│       │       ├── soda_candidate.rs
│       │       └── soda_candidate_check.rs
│       ├── Cargo.toml
│       └── build.rs
├── scripts/
│   ├── fixtures/
│   │   ├── portcontracts/
│   │   │   ├── cli_surface.json
│   │   │   ├── dashboard_config_vectors.json
│   │   │   ├── host_config_bytes.json
│   │   │   ├── operator_wire.json
│   │   │   ├── strictjson_vectors.json
│   │   │   └── systemd_wiring.json
│   │   ├── spaces-review-client.ts
│   │   ├── spaces-review.css
│   │   ├── spaces-review.html
│   │   └── spaces-scenarios.ts
│   ├── build-forgejo-preview.ts
│   ├── build-forgejo.test.ts
│   ├── build-forgejo.ts
│   ├── build-soda-extension.test.ts
│   ├── build-soda-extension.ts
│   ├── check-complexity.sh
│   ├── check-errcheck.sh
│   ├── check-forgejo-branding.ts
│   ├── check-gofumpt.sh
│   ├── check-lit.ts
│   ├── check-native.sh
│   ├── check-no-npm.sh
│   ├── check-no-python.sh
│   ├── check-oxfmt.sh
│   ├── check-oxlint.sh
│   ├── check-source.sh
│   ├── check-sql-locality.sh
│   ├── check-staticcheck.sh
│   ├── check-ts-complexity.sh
│   ├── console_welcome_test.go
│   ├── forgejo_account_details_test.go
│   ├── forgejo_account_settings_test.go
│   ├── forgejo_admin_details_test.go
│   ├── forgejo_admin_monitoring_test.go
│   ├── forgejo_admin_org_test.go
│   ├── forgejo_auth_test.go
│   ├── forgejo_cargo_test.go
│   ├── forgejo_code_search_test.go
│   ├── forgejo_components_test.go
│   ├── forgejo_federated_auth_test.go
│   ├── forgejo_form_components_test.go
│   ├── forgejo_form_layout_test.go
│   ├── forgejo_home_redesign_test.go
│   ├── forgejo_insights_test.go
│   ├── forgejo_migrate_test.go
│   ├── forgejo_native_pages_test.go
│   ├── forgejo_notification_preview_test.go
│   ├── forgejo_onboarding_test.go
│   ├── forgejo_org_details_test.go
│   ├── forgejo_org_home_test.go
│   ├── forgejo_org_projects_test.go
│   ├── forgejo_owner_code_test.go
│   ├── forgejo_packages_test.go
│   ├── forgejo_presentation_test.go
│   ├── forgejo_profiles_test.go
│   ├── forgejo_project_board_test.go
│   ├── forgejo_repository_code_test.go
│   ├── forgejo_repository_content_test.go
│   ├── forgejo_repository_general_settings_test.go
│   ├── forgejo_repository_issues_test.go
│   ├── forgejo_repository_settings_collections_test.go
│   ├── forgejo_repository_settings_details_test.go
│   ├── forgejo_repository_settings_navigation_test.go
│   ├── forgejo_repository_test.go
│   ├── forgejo_setup_test.go
│   ├── forgejo_shared_projects_test.go
│   ├── forgejo_soda_settings_test.go
│   ├── forgejo_status_test.go
│   ├── forgejo_template_fixture_test.go
│   ├── forgejo_theme_components_test.go
│   ├── pg_backup_test.go
│   ├── pg_runtime_test.go
│   ├── preview-spaces.ts
│   ├── render-forgejo-branding.sh
│   ├── render_forgejo_branding_test.go
│   ├── render_forgejo_native_test.go
│   ├── screenshot.ts
│   ├── sodaspaces_templates_test.go
│   ├── system_formats_test.go
│   ├── terminal_branding_test.go
│   └── wire_contracts_test.go
├── system/
│   ├── containers/
│   │   ├── dashboard/
│   │   │   └── Containerfile
│   │   ├── extension/
│   │   │   ├── Containerfile
│   │   │   ├── extension.json
│   │   │   └── run
│   │   ├── forgejo/
│   │   │   └── Containerfile
│   │   └── tailnet/
│   │       └── Containerfile
│   ├── host/
│   │   ├── config/
│   │   │   ├── 90-soda-routing.conf
│   │   │   ├── cockpit.conf
│   │   │   ├── cockpit.pam
│   │   │   ├── cockpit.socket.conf
│   │   │   ├── console-welcome.sh
│   │   │   ├── forgejo.env
│   │   │   ├── host.example.json
│   │   │   ├── proxy.Caddyfile
│   │   │   ├── soda.sysusers
│   │   │   └── soda.tmpfiles
│   │   ├── image/
│   │   │   ├── packages.tmpfiles
│   │   │   ├── retained-images.conf
│   │   │   └── soda-image-import.service
│   │   ├── installer/
│   │   │   └── load-console.sh
│   │   ├── provisioning/
│   │   │   ├── base.json
│   │   │   └── candidate.json
│   │   ├── selinux/
│   │   │   └── soda-build-worker.te
│   │   ├── services/
│   │   │   ├── README.md
│   │   │   ├── forgejo.container
│   │   │   ├── soda-console.service
│   │   │   ├── soda-dashboard.container
│   │   │   ├── soda-extension-install.service
│   │   │   ├── soda-forgejo-migrate.service
│   │   │   ├── soda-host.service
│   │   │   ├── soda-host.socket
│   │   │   ├── soda-identity-runtime.socket
│   │   │   ├── soda-identity.service
│   │   │   ├── soda-identity.socket
│   │   │   ├── soda-pg-provision.service
│   │   │   ├── soda-postgres-backup.service
│   │   │   ├── soda-postgres-backup.timer
│   │   │   ├── soda-postgres-init.service
│   │   │   ├── soda-postgres.container
│   │   │   ├── soda-project@.service
│   │   │   ├── soda-proxy.container
│   │   │   ├── soda-tailnet@.service
│   │   │   └── soda.network
│   │   ├── trust/
│   │   │   └── release-trust.json
│   │   └── Containerfile
│   ├── licenses/
│   │   ├── avatar-dependencies.txt
│   │   ├── forgejo-LICENSE
│   │   ├── lit-LICENSE
│   │   └── tailscale-LICENSE
│   └── project/
│       ├── licenses/
│       │   └── tea-LICENSE
│       ├── rootfs/
│       │   ├── etc/
│       │   │   ├── containers/
│       │   │   │   ├── containers.conf
│       │   │   │   └── storage.conf
│       │   │   ├── mise/
│       │   │   │   └── config.toml
│       │   │   ├── profile.d/
│       │   │   │   ├── soda-mise.sh
│       │   │   │   └── soda-podman.sh
│       │   │   ├── ssh/
│       │   │   │   └── sshd_config.d/
│       │   │   │       └── 10-soda.conf
│       │   │   ├── sudoers.d/
│       │   │   │   └── soda-project
│       │   │   ├── systemd/
│       │   │   │   └── system/
│       │   │   │       ├── soda-podman.service
│       │   │   │       ├── soda-podman.socket
│       │   │   │       └── soda-project-init.service
│       │   │   └── yum.repos.d/
│       │   │       └── gh-cli.repo
│       │   └── usr/
│       │       └── libexec/
│       │           └── soda/
│       │               └── project-init
│       ├── Containerfile
│       └── muse-release.json
├── tests/
│   ├── build/
│   │   ├── avatar_integration_test.go
│   │   ├── candidate_gate_test.go
│   │   ├── complexity_scope_test.go
│   │   ├── forgejo_domain_test.go
│   │   ├── forgejo_payload_test.go
│   │   ├── helpers.go
│   │   ├── muse_exec_test.go
│   │   ├── native_support_test.go
│   │   ├── operator_probe_test.go
│   │   ├── project_account_test.go
│   │   ├── project_factory_roles_accounts_test.go
│   │   ├── project_factory_roles_fixture_test.go
│   │   ├── project_factory_roles_inputs_test.go
│   │   ├── project_factory_roles_lifecycle_test.go
│   │   ├── project_factory_roles_output_test.go
│   │   ├── project_factory_roles_readiness_test.go
│   │   ├── project_factory_roles_test.go
│   │   ├── project_foundation_test.go
│   │   ├── project_runtime_test.go
│   │   ├── proxy_image_test.go
│   │   ├── sodaspaces_test.go
│   │   ├── source_checks_test.go
│   │   ├── tailnet_image_test.go
│   │   ├── terminal_assets_test.go
│   │   ├── u08_state_test.go
│   │   └── workload_probe_test.go
│   ├── fixtures/
│   │   └── workload/
│   │       ├── public/
│   │       │   └── index.html
│   │       ├── Containerfile
│   │       └── compose.yaml
│   ├── forgejo/
│   │   ├── fixtures/
│   │   │   ├── component-browser.ts
│   │   │   └── lit-smoke.ts
│   │   ├── presentation/
│   │   │   ├── dashboard-sidebar-browser.test.ts
│   │   │   ├── form-browser.test.ts
│   │   │   ├── form-native-contracts.json
│   │   │   ├── form-presentation-deltas.json
│   │   │   ├── form-source.test.ts
│   │   │   ├── gallery.test.ts
│   │   │   ├── gallery.tmpl
│   │   │   ├── home-background-browser.test.ts
│   │   │   ├── inventory.json
│   │   │   ├── inventory.test.ts
│   │   │   ├── locales.test.ts
│   │   │   ├── login-station-browser.test.ts
│   │   │   ├── migration-browser.test.ts
│   │   │   ├── notification-layout-browser.test.ts
│   │   │   ├── profile-browser.test.ts
│   │   │   ├── refinement-browser.test.ts
│   │   │   ├── refinement-gallery.tmpl
│   │   │   ├── repository-actionbar-browser.test.ts
│   │   │   ├── repository-settings-browser.test.ts
│   │   │   ├── repository-settings-gallery.tmpl
│   │   │   ├── repository-settings-native-contracts.json
│   │   │   ├── repository-settings-source.test.ts
│   │   │   ├── settings-browser.test.ts
│   │   │   ├── settings-contracts.ts
│   │   │   ├── settings-native-contracts.json
│   │   │   ├── settings-source.test.ts
│   │   │   └── workspace-tokens.test.ts
│   │   ├── branding.test.ts
│   │   ├── cockpit-branding.test.ts
│   │   ├── component-boundaries.test.ts
│   │   ├── component-layout-boundaries.test.ts
│   │   ├── component-toolbar-boundaries.test.ts
│   │   ├── explore-overflow.test.ts
│   │   ├── lit-build.test.ts
│   │   ├── lit-runtime.test.ts
│   │   ├── login-theme.test.ts
│   │   ├── milestone-page-boundaries.json
│   │   ├── milestone-source.test.ts
│   │   ├── milestones-layout.test.ts
│   │   ├── notification-preview.test.ts
│   │   ├── repository-actions.test.ts
│   │   ├── repository-container.test.ts
│   │   ├── repository-switcher.test.ts
│   │   └── work-item-lists.test.ts
│   ├── frontend/
│   │   ├── fixtures/
│   │   │   ├── drawer-fixture.ts
│   │   │   ├── persistent-panel-fixture.ts
│   │   │   ├── project-controls-driver.ts
│   │   │   ├── terminal-driver.ts
│   │   │   ├── terminal-fixture.ts
│   │   │   ├── workspace-driver.ts
│   │   │   ├── workspace-fixture.ts
│   │   │   ├── workspace-measurement-probe.ts
│   │   │   └── workspace-model.ts
│   │   ├── drawer-controls.test.ts
│   │   ├── factory-view.test.ts
│   │   ├── identity.test.ts
│   │   ├── journey-input.test.ts
│   │   ├── native-browser.test.ts
│   │   ├── native-paths.test.ts
│   │   ├── native-project-controls.test.ts
│   │   ├── persistent-panel.test.ts
│   │   ├── project-access-controls.test.ts
│   │   ├── project-environment-controls.test.ts
│   │   ├── repository-settings.test.ts
│   │   ├── sodaspaces-http.test.ts
│   │   ├── spaces-api.test.ts
│   │   ├── spaces-attention.test.ts
│   │   ├── spaces-layout.test.ts
│   │   ├── spaces-preview.test.ts
│   │   ├── tailnet.test.ts
│   │   ├── terminal-end.test.ts
│   │   ├── terminal-retirement.test.ts
│   │   ├── terminal-stream.test.ts
│   │   ├── terminal.test.ts
│   │   ├── workspace-attention.test.ts
│   │   ├── workspace-first-use.test.ts
│   │   ├── workspace-journey.test.ts
│   │   ├── workspace-layout.test.ts
│   │   ├── workspace-navigation.test.ts
│   │   ├── workspace-persistence.test.ts
│   │   ├── workspace-recovery.test.ts
│   │   ├── workspace-responsive.test.ts
│   │   ├── workspace-setup.test.ts
│   │   └── workspace.test.ts
│   └── installed/
│       ├── cockpit-types.ts
│       ├── forgejo-advertisement.sh
│       ├── host.sh
│       ├── native-browser.ts
│       ├── operator.sh
│       ├── operator.ts
│       ├── project-foundation.sh
│       ├── project-os.sh
│       ├── service-ordering.sh
│       ├── shared-tools.sh
│       ├── sodaspaces-cli.ts
│       ├── sodaspaces-controls.ts
│       ├── sodaspaces-first-use-journey.ts
│       ├── sodaspaces-http.ts
│       ├── sodaspaces-input.ts
│       ├── sodaspaces-journey-evidence.ts
│       ├── sodaspaces-matrix-input.ts
│       ├── sodaspaces-matrix-native.ts
│       ├── sodaspaces-workspace-journey.ts
│       └── workloads.sh
├── tools/
│   ├── acceptance/
│   │   ├── src/
│   │   │   ├── bin/
│   │   │   │   ├── soda-acceptance-remote.rs
│   │   │   │   └── soda-host-probes.rs
│   │   │   ├── command/
│   │   │   │   ├── execute.rs
│   │   │   │   ├── mod.rs
│   │   │   │   ├── ssh.rs
│   │   │   │   └── tests.rs
│   │   │   ├── coreos/
│   │   │   │   └── tests.rs
│   │   │   ├── driver/
│   │   │   │   ├── actions.rs
│   │   │   │   ├── finalization.rs
│   │   │   │   ├── inputs.rs
│   │   │   │   ├── mod.rs
│   │   │   │   ├── options.rs
│   │   │   │   └── tests.rs
│   │   │   ├── evidence/
│   │   │   │   ├── mod.rs
│   │   │   │   ├── redaction.rs
│   │   │   │   ├── store.rs
│   │   │   │   └── tests.rs
│   │   │   ├── files/
│   │   │   │   ├── inputs.rs
│   │   │   │   ├── mod.rs
│   │   │   │   ├── owned_directory.rs
│   │   │   │   ├── temporary.rs
│   │   │   │   └── tests.rs
│   │   │   ├── host_probes/
│   │   │   │   ├── content.rs
│   │   │   │   ├── deployments.rs
│   │   │   │   ├── forgejo.rs
│   │   │   │   ├── listeners.rs
│   │   │   │   ├── mod.rs
│   │   │   │   ├── tailnet.rs
│   │   │   │   └── tests.rs
│   │   │   ├── jsonio/
│   │   │   │   └── tests.rs
│   │   │   ├── native_phase/
│   │   │   │   └── tests.rs
│   │   │   ├── process/
│   │   │   │   ├── launch.rs
│   │   │   │   ├── mod.rs
│   │   │   │   ├── owned_process.rs
│   │   │   │   ├── phase.rs
│   │   │   │   └── tests.rs
│   │   │   ├── project_state/
│   │   │   │   ├── command.rs
│   │   │   │   ├── files.rs
│   │   │   │   ├── mod.rs
│   │   │   │   ├── snapshot.rs
│   │   │   │   ├── tests.rs
│   │   │   │   └── workloads.rs
│   │   │   ├── report/
│   │   │   │   ├── handoff.rs
│   │   │   │   ├── mod.rs
│   │   │   │   ├── observation.rs
│   │   │   │   └── tests.rs
│   │   │   ├── timestamps/
│   │   │   │   └── tests.rs
│   │   │   ├── trust/
│   │   │   │   ├── host_key.rs
│   │   │   │   ├── inline_data.rs
│   │   │   │   ├── mod.rs
│   │   │   │   └── tests.rs
│   │   │   ├── vm/
│   │   │   │   ├── base.rs
│   │   │   │   ├── config.rs
│   │   │   │   ├── launch.rs
│   │   │   │   ├── lifecycle.rs
│   │   │   │   ├── mod.rs
│   │   │   │   └── tests.rs
│   │   │   ├── cockpit.rs
│   │   │   ├── coreos.rs
│   │   │   ├── error.rs
│   │   │   ├── jsonio.rs
│   │   │   ├── lib.rs
│   │   │   ├── main.rs
│   │   │   ├── native_phase.rs
│   │   │   ├── personal_git.rs
│   │   │   ├── probe.rs
│   │   │   ├── provisioning.rs
│   │   │   ├── qmp.rs
│   │   │   ├── remote.rs
│   │   │   ├── structured.rs
│   │   │   └── timestamps.rs
│   │   ├── Cargo.toml
│   │   └── build.rs
│   ├── candidate-setup/
│   │   ├── src/
│   │   │   ├── config.rs
│   │   │   ├── controller.rs
│   │   │   ├── fixture_authority.rs
│   │   │   ├── main.rs
│   │   │   ├── preflight.rs
│   │   │   ├── process.rs
│   │   │   ├── selinux.rs
│   │   │   ├── storage.rs
│   │   │   ├── tests.rs
│   │   │   ├── worker_caches.rs
│   │   │   └── worker_tools.rs
│   │   ├── tests/
│   │   │   ├── support/
│   │   │   │   └── mod.rs
│   │   │   └── cli.rs
│   │   ├── Cargo.toml
│   │   └── README.md
│   ├── lab-credentials/
│   │   ├── src/
│   │   │   ├── fixture_authority.rs
│   │   │   ├── inventory.rs
│   │   │   ├── main.rs
│   │   │   ├── process.rs
│   │   │   ├── runbooks.rs
│   │   │   └── tests.rs
│   │   ├── tests/
│   │   │   └── cli.rs
│   │   ├── Cargo.toml
│   │   └── README.md
│   ├── lit-check/
│   │   ├── fixtures/
│   │   │   ├── aria.ts
│   │   │   ├── boolean.ts
│   │   │   ├── customProperty.ts
│   │   │   ├── directive.ts
│   │   │   ├── event.ts
│   │   │   ├── nullable.ts
│   │   │   ├── positive.ts
│   │   │   ├── property.ts
│   │   │   ├── unclosed.ts
│   │   │   ├── unknownEvent.ts
│   │   │   └── unknownProperty.ts
│   │   ├── README.md
│   │   ├── check.ts
│   │   ├── package.json
│   │   └── tsconfig.json
│   ├── png-equal/
│   │   ├── main.go
│   │   └── main_test.go
│   ├── postgres-fixture/
│   │   ├── src/
│   │   │   └── main.rs
│   │   └── Cargo.toml
│   ├── release-assets/
│   │   ├── src/
│   │   │   ├── bin/
│   │   │   │   ├── soda-fetch-muse.rs
│   │   │   │   ├── soda-fetch-tea.rs
│   │   │   │   ├── soda-fetch-terminal.rs
│   │   │   │   ├── soda-forgejo-locales.rs
│   │   │   │   ├── soda-render-provisioning.rs
│   │   │   │   ├── soda-render-terminal-logo.rs
│   │   │   │   └── soda-stage.rs
│   │   │   ├── fetch/
│   │   │   │   ├── muse/
│   │   │   │   │   └── tests.rs
│   │   │   │   ├── tea/
│   │   │   │   │   └── tests.rs
│   │   │   │   ├── mod.rs
│   │   │   │   ├── muse.rs
│   │   │   │   ├── tea.rs
│   │   │   │   ├── terminal.rs
│   │   │   │   └── test_server.rs
│   │   │   ├── locales/
│   │   │   │   ├── merge/
│   │   │   │   │   └── tests.rs
│   │   │   │   ├── merge.rs
│   │   │   │   └── mod.rs
│   │   │   ├── render/
│   │   │   │   ├── provisioning/
│   │   │   │   │   ├── document.rs
│   │   │   │   │   ├── mod.rs
│   │   │   │   │   ├── private_files.rs
│   │   │   │   │   ├── render.rs
│   │   │   │   │   └── tests.rs
│   │   │   │   ├── stage/
│   │   │   │   │   ├── branding.rs
│   │   │   │   │   ├── files.rs
│   │   │   │   │   ├── mod.rs
│   │   │   │   │   ├── payload.rs
│   │   │   │   │   └── tests.rs
│   │   │   │   ├── terminal_logo/
│   │   │   │   │   ├── geometry.rs
│   │   │   │   │   ├── mod.rs
│   │   │   │   │   ├── svg.rs
│   │   │   │   │   └── tests.rs
│   │   │   │   └── mod.rs
│   │   │   └── lib.rs
│   │   ├── tests/
│   │   │   ├── fixtures/
│   │   │   │   ├── prov-root/
│   │   │   │   │   ├── assets/
│   │   │   │   │   │   └── branding/
│   │   │   │   │   │       └── source/
│   │   │   │   │   │           └── soda-symbol.svg
│   │   │   │   │   ├── frontend/
│   │   │   │   │   │   └── forgejo/
│   │   │   │   │   │       └── payload.json
│   │   │   │   │   └── system/
│   │   │   │   │       └── host/
│   │   │   │   │           └── provisioning/
│   │   │   │   │               └── base.json
│   │   │   │   ├── ext-host.bu
│   │   │   │   ├── ext-hostkey.bu
│   │   │   │   ├── ext-product.bu
│   │   │   │   └── min-host.bu
│   │   │   ├── locales_support/
│   │   │   │   └── mod.rs
│   │   │   ├── render_support/
│   │   │   │   └── mod.rs
│   │   │   ├── fetch_cli.rs
│   │   │   ├── locales_cli.rs
│   │   │   ├── locales_locked_input.rs
│   │   │   ├── locales_native_merge.rs
│   │   │   ├── render_provisioning.rs
│   │   │   ├── render_staging.rs
│   │   │   └── render_terminal_logo.rs
│   │   ├── Cargo.toml
│   │   └── terminal-assets.lock.json
│   ├── soda-avatars/
│   │   ├── main.go
│   │   └── main_test.go
│   ├── soda-installed-probes/
│   │   ├── cockpit_account_test.go
│   │   ├── installed_probes_test.go
│   │   ├── main.go
│   │   ├── main_test.go
│   │   └── project_state_test.go
│   ├── soda-rootfs-server/
│   │   ├── main.go
│   │   ├── main_test.go
│   │   └── soda-rootfs-server.service
│   └── test-vm/
│       ├── src/
│       │   ├── main.rs
│       │   ├── process.rs
│       │   ├── start.rs
│       │   ├── state.rs
│       │   ├── tests.rs
│       │   └── transport.rs
│       ├── tests/
│       │   ├── support/
│       │   │   └── mod.rs
│       │   ├── cli.rs
│       │   ├── start.rs
│       │   ├── status.rs
│       │   └── transport.rs
│       ├── Cargo.toml
│       └── README.md
├── .containerignore
├── .gitignore
├── .oxfmtrc.json
├── .oxlintrc.json
├── AGENTS.md
├── Cargo.lock
├── Cargo.toml
├── LICENSE
├── NOTICE
├── README.md
├── bun.lock
├── bunfig.toml
├── go.mod
├── go.sum
├── package.json
├── rust-toolchain.toml
├── tsconfig.base.json
├── tsconfig.browser.json
├── tsconfig.json
└── tsconfig.tests.json
```

Ignored roots remain `.artifacts/` for build/evidence outputs, `.local/` for local inputs, `target/` for Cargo outputs, and `node_modules/` for dependencies. Their generated contents and private fixtures are outside the tracked-source inventory and are not enumerated here.

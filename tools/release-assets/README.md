# Soda release assets

Fetch public build inputs and prepare release assets, Forgejo presentation and
private provisioning documents. These commands are for release builders and
operators preparing a host. The native build pipeline normally invokes them.

## Choose a command

Run Cargo examples from the repository root. Each binary accepts `--help`:

```sh
cargo run -p soda-release-assets --bin soda-stage -- --help
```

| Binary | Inputs and result |
| --- | --- |
| `soda-fetch-muse` | `--arch x86_64 --out ABSOLUTE_FILE`, optionally `--manifest FILE`; download a size/checksum-pinned native Muse executable. Default manifest: `system/project/muse-release.json`. |
| `soda-fetch-tea` | `--arch x86_64 [--out DIR]`; fetch Tea and its license into `bin/tea` and `licenses/tea/LICENSE`. Default output: `.artifacts/native/x86_64/project-tools`. |
| `soda-fetch-terminal` | `--out DIR`; fetch the exact browser terminal files pinned in `terminal-assets.lock.json`. |
| `soda-forgejo-locales` | `(--native FILE \| --lock FILE) [--additions FILE] --out FILE`; merge a complete native Forgejo INI catalog with Soda additions into a new file. |
| `soda-stage` | `--arch x86_64 --host-context DIR --forgejo-context DIR`; copy existing native outputs, configuration and branding into prepared build contexts. |
| `soda-render-provisioning` | Public operator key and private password-hash/optional host-key files; write a private Butane document with `--out ABSOLUTE_FILE`. |
| `soda-render-terminal-logo` | Read the canonical emblem and regenerate terminal marks, or use `--check` to verify existing marks. |

Fetching needs network access. Staging requires native x86_64 Linux, existing
`.artifacts/native/x86_64` build outputs, a real host context with `rootfs/`, and
a Forgejo context whose `forgejo/` destination does not yet exist. Staging
prints the host rootfs path on success; it does not build or install the result.

## Fetch and verify assets

```sh
cargo run -p soda-release-assets --bin soda-fetch-muse -- \
  --arch x86_64 --out "$PWD/.artifacts/native/x86_64/project-tools/bin/muse-native"
cargo run -p soda-release-assets --bin soda-fetch-terminal -- \
  --out "$PWD/.artifacts/terminal-assets"
cargo run -p soda-release-assets --bin soda-render-terminal-logo -- --check
```

Fetchers verify their selected upstream inputs and reuse matching staged bytes
where supported. Run the logo renderer without `--check` only when you intend
to rewrite the committed files in `assets/branding/terminal/`.

## Prepare private provisioning

Prepare an existing private output parent with mode `0700`, the operator's
public SSH key and a root password hash file with mode `0600`. Then:

```sh
cargo run -p soda-release-assets --bin soda-render-provisioning -- \
  --operator-key-file /home/soda-builder/private/operator.pub \
  --root-password-hash-file /home/soda-builder/private/root.hash \
  --appliance-hostname soda-host \
  --out /home/soda-builder/private/new-host.bu
```

The command exclusively creates a `0600` output. `--bootstrap extensions` is
the default; `minimal` is for fixture provisioning. `--hostname soda-native-*`
selects a fixture name and cannot be combined with `--appliance-hostname`.
`--ssh-host-key-file` accepts a private per-instance Ed25519 host key. Keep
provisioning output private; Butane-to-Ignition conversion is a separate step.

See [native support](../../docs/development/native-support.md) for provisioning
and staging contracts, [release workflow](../../docs/development/release.md)
for the complete build, and [Forgejo locales](../../frontend/forgejo/locales/README.md)
for catalog inputs.

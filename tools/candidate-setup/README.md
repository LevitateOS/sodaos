# soda-candidate-setup

Prepare a native development builder for isolated Soda candidate builds. This
operator tool installs the candidate wrapper and admitted build controller,
prepares worker tools/caches and storage, installs the worker SELinux policy.

## Before running

Use an authorized x86_64 builder with working `sudo`, systemd, the existing
`soda-build-worker` user, Go, Bun, Podman, Skopeo and `flock`. Run from a clean,
committed Soda checkout with a clean canonical Forgejo fork checkout available.
The repository pins the build toolchain; setup may download tools and warm
caches. An active candidate worker prevents setup from proceeding.

This command changes the builder's installed files, policy and worker state.
The [native support guide](../../docs/development/native-support.md)
owns candidate isolation rules.

## Prepare the builder

From the repository root:

```sh
SODA_FORGEJO_SOURCE=/home/soda-builder/forgejo-source \
SODA_FOUNTAIN_RELEASE_URL=https://github.com/veighnsche/forgejo-ext/releases/download/v15.0.10-fountain.3/release.json \
  cargo run -p soda-candidate-setup
```

The command takes no CLI options; even `--help` is ignored and would run setup.
Configure it through the environment:

| Variable | Use |
| --- | --- |
| `SODA_FORGEJO_SOURCE` | Required absolute canonical Fountain checkout supplying the pinned SDK. Unrelated changes remain untouched; build admission checks SDK bytes against the selected release. |
| `SODA_FOUNTAIN_RELEASE_URL` | Required immutable HTTPS metadata for the separately released Fountain image and corresponding sources. The build controller checks its source against `fountain-source.json`. Run `soda-artifacts check-fountain-pin` from the checkout first: it reports the matching release URL or explains the gap. |
| `SODA_CANDIDATE_ROOT` | Worker storage root; defaults to `/home/soda-candidate`. |
| `SODA_CANDIDATE_HOME` | Worker home; defaults to `home` under the storage root. |
| `SODA_CANDIDATE_RUN` | Worker runtime; defaults to `run` under the storage root. |
| `SODA_CANDIDATE_SCRATCH` | Worker scratch; defaults to `scratch` under the storage root. |

Keep storage overrides on the workspace disk. They change worker storage, not
all installed paths. The setup lease uses `TMPDIR`; select a directory on the
workspace disk for it and other temporary bytes.

## Result

Successful setup prints a development build invocation using the generated
worker configuration. Candidate output is placed below
`.artifacts/releases/isolated/`. Follow the printed command with a fresh output
directory when ready to build; setup itself does not produce a candidate.

For the build commands, see [release tools](../../lib/soda-release-tools/README.md)
and [release workflow](../../docs/development/release.md). Installation and
fixture provisioning remain separate operations in the
[installation guide](../../docs/guides/installation.md).

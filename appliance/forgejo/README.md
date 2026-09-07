# Forgejo source preparation — U02/H02

**Implemented: verified source preparation only.** The user authorized the next
implementation batch after `2ff9e44`. This directory locks **v16.0.3 as a development
candidate**, not the installed/release baseline. No native extension is present;
`patches: []` deliberately advertises nothing. The service and aggregate native
builder still use the existing stock 15.0.7 image. H02 is not complete.

## Baseline review

Public metadata inspected on **2026-09-07**:

| Candidate | Published support | Implication |
| --- | --- | --- |
| 16.0.3 stable | Until 29 October 2026 | Reuse its human Actions jobs/logs/artifacts/cancel APIs; adopting it requires following subsequent supported stable releases, not freezing this patch level |
| 15.0.7 LTS, currently installed | Until 15 July 2027 | Longer support, but the needed newer Actions interfaces would require maintained backports in addition to the remaining headless gaps |

Sources: [release schedule](https://forgejo.org/releases/),
[release policy](https://forgejo.org/docs/latest/developer/release/),
[upgrade guidance](https://forgejo.org/docs/latest/admin/upgrade/) and the existing
[H01 action register](../../docs/forgejo-api-coverage.md). Stable gets bug/security
fixes; LTS receives critical/security fixes. Neither is automatically a safe
upgrade or a complete headless backend. Major-version changes require review of
all intervening release notes and migration effects; native database downgrade
refusal means that reverting an image alone is not rollback.

**Recommendation:** use 16.0.3 for the first source-backed development slice to
avoid recreating its useful APIs. Before adopting a deployment baseline, confirm
a human maintainer/reviewer responsible for the supported-release cadence. The
SodaOS repository owns build/patch integration (U02), native feature semantics
(the relevant U owner), consumer compatibility (U03/U17) and security/rebase/
retirement review (U01/U17). No person or upstream team has been assigned a new
maintenance commitment by this change. No security-support promise is made for a
Soda patch set that does not exist yet.

The source lock binds commit, compressed archive SHA-256 and ordered patch names/
digests. The commit archive was independently downloaded again from Codeberg and
matched the H01 archive digest. The archive's Git PAX comment also matches the
locked commit. Hashes and that comment establish expected source identity, **not**
a release signature, installed identity or security qualification.

## Current concrete entrypoint

From the repository root, using the existing pinned Go toolchain:

```sh
go run -mod=readonly ./tools/soda-forgejo-source \
  --lock appliance/forgejo/source.lock.json \
  --out .artifacts/NEW-OWNED-ATTEMPT
```

The parent must already exist and be real/not group- or world-writable; the output
must not exist. `--archive /absolute/existing/archive.tar.gz` uses an existing
regular archive instead of downloading. Without it, the only URL is the locked
commit's HTTPS Codeberg archive; redirects/non-200 responses fail, with no retries,
credentials, cookies or browser/provider-selected origin. This is build-source
retrieval, not a provider account/repository mutation.

`internal/forgejobuild/` owns preparation; the build-only tool is its caller.
It uses only the Go standard library and native `git apply` for an actual patch
series. No extra Go module, Python production helper, source clone or new
application service is needed. `go test ./...` includes its focused tests through
the existing source-test entrypoint. The command is not installed in the appliance
and is not a replacement for `scripts/build-native.sh`.

Output, retained even on failure:

- `source.lock.json`: exact checked input snapshot;
- `upstream.tar.gz`: private snapshot of the archive bytes checked/extracted;
- `source/`: extracted, then patched source including upstream licenses/assets;
- `patches/`: verified applied patch snapshots;
- `prepared.json`: written only after success, binding the lock and resulting
  source-tree contents/types/modes/link targets. This is an integrity receipt, not
  build or workflow acceptance. Source edited after preparation needs preparation
  again in a fresh attempt; downstream compilation must recheck the tree binding.

Limits are **source-preparation safety limits**, not blame/diff API budgets:
64 KiB lock, 128 MiB compressed archive, 256 MiB expanded tar including headers/
padding, 20,000 source entries, 32 patches/8 MiB each/32 MiB total, two-minute
public HTTP timeout and 30 seconds per native patch check/application. The actual
selected archive is 11,987,389 compressed bytes with about 47.6 MB of file content.
No limit was represented as measured native Git/API performance.

Preparation refuses unsupported lock fields/version/digests, nonregular inputs,
occupied/symlink-parent output, bad checksum/commit metadata, wrong archive roots,
duplicate/traversal/.git/special/set-ID entries, unsafe/dangling/cyclic links and
invalid gzip completion. Relative source symlinks are retained. Patch filenames
are contiguous `0001-name.patch`, `0002-name.patch`, etc.; hashes are verified
before running anything. Native Git performs check/apply in the fresh source,
without inherited repository/index/global configuration or `--unsafe-paths`.
Partial failures have no completion receipt and are not automatically removed or
retried. This tool is not a sandbox for malicious same-user workspace mutation.

## Remaining H02 work — not a build-ready image

Before connecting the aggregate build, finish these as one coherent source-backed
candidate, rather than changing the service reference first:

1. Review the concrete read/native-auth contracts and their native fixes; add
   cohesive patches, native tests/schema, provenance and retirement conditions.
2. Implement the native Containerfile/build caller, preserving the selected
   upstream `Dockerfile`/Makefile behavior: frontend assets, Go generation,
   SQLite/bindata/timezone tags, CGO/static build, environment-to-ini, entrypoint,
   s6/OpenSSH/Git, UID/data paths and protocols. The current upstream recipe uses
   floating xx/Go/Alpine image references; resolve/review their exact native inputs
   in this lock before claiming a pinned image build. No fabricated digests.
3. Preserve corresponding source and licenses for GPL-3.0-or-later distribution;
   Swagger's MIT license is not Forgejo's distribution license. Review transitive
   Go/npm/native dependencies and notices; the source archive alone does not
   complete distribution compliance.
4. Wire the source preparation/tree verification/native tests/build into the
   existing `forgejo.iid`, `images/forgejo.oci`, staging/seal/installer consumers.
   Coordinate nativebuild identity/allowlists and source/notices payloads with U02/
   U03/P04. Do not create a permanent stock/patched selector, second build platform
   or metadata-only compatibility claim. Caddy/project images are not changed.
5. Run actual native builds/contract tests and record exact outputs independently
   per architecture. Installed migration/provider/Git tests need their own scope.

The first real preparation exposed Git's global PAX header, which the initial
extractor rejected. The failure and output were retained; handling now validates
that metadata against the locked commit. Real local-archive preparation then
passed. Focused/aggregate checks and any additional evidence are recorded in the
[handoff](../../docs/implementation-status.md), not inferred from this README.

# Forgejo source preparation — U02/H02

**Current decision:** use Forgejo's official template overrides and native
rendering; see the [leading decision record](../../docs/dashboard-implementation-plan.md#selected-direction--official-template-overrides-partial-u01-invalidation).
The mandatory source-built/patched Forgejo direction below is superseded. Retain
this preparer, provenance and licensing research, but do not treat extending it
into a native image build as the next required task. U01's architecture acceptance
is withdrawn; no installed version, image or template was changed. Template
overrides do not inherently require rebuilding the Forgejo executable.

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

**Technical disposition after the U01 delta review:** retain 16.0.3 as the single
implementation input and follow the supported stable line, rather than backporting
its APIs to 15.0.7 or implementing two backends. This selects the engineering
candidate, not a live upgrade. **Forgejo upstream owns its security rules, native
semantics and upstream fixes.** Soda owns build/patch integration (U02), faithful
native-interface adaptation (feature owners), consumer compatibility (U03/U17)
and review/security of its own code and carried patches (U01/U17). Incorporating
upstream fixes and maintaining an unmerged extension remain Soda engineering
responsibilities; they do not require the user to become the owner of Forgejo's
policies. Record actual patch/release review in that existing work, not a new
policy-selection gate or a promise that upstream maintains Soda's unmerged code.

The source lock binds commit, compressed archive SHA-256 and ordered patch names/
digests. The commit archive was independently downloaded again from Codeberg and
matched the H01 archive digest. The archive's Git PAX comment also matches the
locked commit. Hashes and that comment establish expected source identity, **not**
a release signature, installed identity or security qualification.

## U01 release/migration gap closed

The formerly truncated 16.0.3 release note is now complete: 9,799 bytes, Git blob
`ba653de36399683ba5dc07658aea53d6e17fd9d5`. Retrieved through Codeberg's contents
API, checked decoded length and Git blob hash; preserved the earlier 8,192-byte
failure. Also read the complete 16.0.0–16.0.2 notes (153,389 / 2,153 / 6,340 bytes).
These are release documents from the upstream `forgejo` branch, **not additional
code from the locked tag**. Original responses, blob/SHA-256 provenance and readable
comment-stripped copies are retained in `.artifacts/research/u01-d5b5065/`.

Concrete upgrade constraints from that review, not a second API inventory:

- v16 removes Docker's wildcard trusted-proxy default. Preserve the current narrow
  trusted proxy design; test client-address/secure-cookie behavior through Caddy.
  Do not restore `*` to make login work.
- Migration/mirroring host rules now cover additional engines/LFS and reject HTTP
  mirror redirects. CGNAT is private in 16.0.2. U09/U12 must expose genuine native
  denial for private/Tailnet remotes, not bypass it with Soda credentials/networking.
- Centralized Git hooks are backward-compatible, but hook path/init behavior and
  ordinary Git pushes require populated-upgrade tests. Upstream's optional old-hook
  cleanup is **not selected**; no `find -delete` or hook regeneration on retained
  repositories is authorized by U01.
- PR `url` now means API URL, not browser navigation. OAuth/LFS access, independent
  session middleware state and cancellation fixes matter to U03/U04/U06/U09 proof.
  Native token restrictions and Actions warning rendering remain security gates.
- Compared the migration source directories, not a live database: v16 adds 13 Go
  files (11 migration implementations and two tests), covering authorized
  integrations, OIDC subject configuration, authentication-source linkage, review
  line counts, Actions priorities/warnings, team invitations, package indexes and
  granular watches. Existing migration-engine logic differs in its xorm import;
  a stable migration engine does not imply an unchanged schema. Only Forgejo runs
  these migrations. Preserve its complete data/config/signing/encryption inputs;
  restore a consistent pre-upgrade set only before later writes make it lossy.
- Release notes mention newer account/Actions interfaces. A targeted contradiction
  check confirmed v16's `is_2fa_enabled` People filter, now recorded in AD22 of the
  single register. This does not supply the missing MFA-reset/detail workflows.

**Disposition:** no new source-level reason to recreate the v16 APIs on the LTS
line was found. Implementation proceeds from the existing lock. Soda's actual patch/release
review and migration/build/conformance checks remain U02/U03/
U17/U20 and require their applicable target scope. Do not reopen the completed
release-note retrieval unless upstream corrects it or the selected version changes.

## U01 build/license dispositions

Review used the real manifests, selected upstream Makefile/README/license and
existing installed development-package metadata; no dependency upgrade/install or
new license inventory was added to tracked source.

- Forgejo's README grants GPL-3.0-or-later. Preserve upstream per-file licenses and
  copyright notices in patches; distributing the combined modified Forgejo must
  meet its GPL terms. Swagger's MIT interoperability grant does not relicense it.
- The native npm lock has 1,220 package entries. Its three absent license fields
  (`khroma` 2.1.0, `reserved` 0.1.2, `svg-tags` 1.0.0) were resolved from their exact
  lock-addressed, integrity-verified tarballs: all contain MIT grants. Do not edit
  the upstream lock to duplicate this conclusion. Retained tarballs/license texts
  and derived metadata are under the same ignored U01 evidence directory.
- MPL components, LGPL sharp/libvips variants, Python/BSD/Apache/MIT dependencies
  and CC-attribution data require their actual notices/source obligations, not a
  blanket “all MIT” label. Optional foreign-architecture packages are not evidence
  of shipped bytes. `dev` is not a reliable exclusion rule for browser assets.
  No AGPL expression occurs in this npm metadata; that is not a whole-product
  license verdict. v16 explicitly removed an AGPL EXIF dependency; do not restore
  its predecessor implementation accidentally.
- Dashboard's 17 and Cockpit's 19 direct installed package metadata entries were
  inspected against their manifests. They declare MIT except TypeScript's Apache
  license. PatternFly CSS lacks a top-level license file in that installed package;
  the closing review below supplies its exact upstream notice/font findings.
  U02 must deliver the matching notices, not infer font/asset licensing from the
  JavaScript package name. Retain Cockpit's existing
  LGPL text, HTMX and Tea licenses and canonical branding attribution.
- Upstream `Makefile`'s `go-licenses` target prefixes collection with `-` and hides
  stderr; `generate-go-licenses.go` excludes NOTICE/README files. **Do not accept
  that target alone as a complete notice check.** U02 must fail on unresolved
  collection errors and match licenses/notices to the actual tagged/native Go,
  frontend and runtime closure. The checked-in `assets/go-licenses.json` contains
  260 nonempty notice records, but only name/path/text—not dependency version or
  build-tag identity. Reuse those texts where they match, rather than claiming they
  prove the final closure. Full transitive Go/native/font license clearance remains
  open; metadata inspection is not that clearance.
- **Apache-2.0 for original SodaOS code is now implemented** in the root
  `LICENSE`, `NOTICE` and README licensing scope. The license text is a verbatim
  copy retrieved from the Apache Software Foundation. Inherited/third-party code
  and artwork are explicitly outside that new grant; Forgejo retains its terms.
  This is not complete distribution compliance or publication authorization.

Selected delivery contract for U02: the existing native stage/bundle must include
corresponding Forgejo source, ordered patches, build scripts/locks and required
notices tied to the exact image/source identity. Include the actual shipped
third-party source/notices where required, not merely a list of SPDX names or a
link to an unmodified upstream tag. Keep operator configuration, credentials,
repositories, databases and private evidence out of that source payload. Extend
existing staging/seal allowlists and provide Soda-owned notice/source access;
no separate release platform or publication job. Resolve native build-image pins
and assemble these outputs in U02, not by repeating U01 source preparation.

### U01 closing review — implementation-ready build and license contract

The following closes the **planning/readiness review**, not the U02 image build
or artifact-level license clearance. Reused the source lock/preparer and release
review; did not resolve/install dependencies or build/retrieve a new image.

Additional exact-input findings after `8727233`:

- Retrieved PatternFly 6.6.1's actual `LICENSE.txt` at its published npm `gitHead`
  `26b709bfeb14c3643a6b999a2619b6bd65641ffa`: MIT, copyright 2019 Red Hat, Inc.
  Compared all **33 installed font files** with Git blob IDs in that exact source:
  they match. They include Red Hat Display/Text/Mono and Font Awesome Solid.
- Retrieved Red Hat Font's actual OFL/author texts as verified Git blobs, including
  the **Reserved Font Name Red Hat**, and Font Awesome 5.0.13's license statement:
  font files OFL-1.1, SVG/JS icons CC-BY-4.0, other code MIT. Preserve those
  distinctions and attribution; CSS package MIT metadata cannot cover all fonts.
  Exact shipped font/notice pairing still needs the U02 bundle check, not an
  assumption that an arbitrary current OFL file matches every historical font.
- Reviewed top-level license texts for all **13 modules named in Soda's go.mod**
  from the existing exact-version caches. They use MIT/BSD grants and the YAML
  module's per-file MIT/Apache split plus NOTICE. The initial default-cache scan
  missed two modules; their exact files are in the existing go-runner cache. This
  is not the full dependency graph or native Forgejo binary closure.
- `cockpit/build/licenses.ts` explicitly exempts `@patternfly/*` from missing-text
  refusal; dashboard imports that collector. **Reject this exemption for U02
  distribution**: collect the exact missing CSS notice and emitted font/icon
  notices rather than accepting the package name. Neither this nor upstream's
  error-ignoring Go collector supplies complete compliance today.
- The predecessor has no root LICENSE/COPYING grant in the inspected checkout.
  Preserve file-level/vendor terms and provenance; do not relabel inherited
  support/Cockpit code or canonical artwork Apache merely because it is present
  here. U02's actual distributed-file review must establish the applicable grant
  for every inherited item; unresolved rights stop distribution, not create a
  license by inference. No predecessor file or license was modified.

Evidence is retained in `.artifacts/research/u01-8727233/`: retrieval provenance,
license texts, font Git-blob bindings and exact-module notice snapshots. Initial
`LICENSE` retrieval returned 404 (`LICENSE.txt` was correct); broad Git-tree reads
exceeded the 512 KiB and then 4 MiB bounds, so the successful review used only
font directories. The 4 MiB prefix is retained; the first oversized body was not
saved by the initial helper. No failed request is represented as a successful
integrity check.

**Selected U02 delivery decisions:**

| Boundary | Concrete implementation / refusal contract |
| --- | --- |
| Native build | Keep the selected upstream Dockerfile/Makefile behavior: native Go/frontend generation, CGO, `sqlite sqlite_unlock_notify bindata timetzdata`, static executable, environment-to-ini and the complete s6/OpenSSH/Git runtime. Preserve `/data`, git UID/GID 1000 and entrypoint behavior. Require actual build-platform = target-platform on each native architecture; upstream cross helpers are not emulated acceptance. |
| Input identities | Extend the existing source-lock/parser contract for the actual xx/Go/Alpine build-image/platform digests. Resolve them in U02, not invent them in U01. Keep native npm/go locks and the real upstream build tools; record apk/compiler/runtime resolver output. Digest pinning alone does not freeze package repositories or prove bit-for-bit reproducibility. |
| Existing caller | Replace only Forgejo's pull in `scripts/build-native.sh` with the reviewed `scripts/build-forgejo.sh` caller. Retain `forgejo.iid`, `images/forgejo.oci`, native-info/stage/seal/install consumption and untouched Caddy/project inputs. The build-only Go preparer remains the sole source owner, outside `cmd/`; no second builder/release platform. |
| Verification before consumption | Verify prepared tree/lock/patch binding immediately before build, then actual source/patch/platform/image identity and mandatory binary/assets/runtime content. Reject edited prepared trees, stale/missing outputs, wrong architecture, partial patches and mismatched image/notice/source payloads before staging/sealing. |
| License compatibility | Apache for original Soda does not change GPL Forgejo or inherited terms. Preserve MIT/BSD notices, Apache NOTICE, MPL covered-source requirements, LGPL source/relinking obligations and OFL/CC attribution according to the actual shipped material. Review executable linkage and embedded assets, not just OCI labels or `dev` flags. Missing/unresolved rights or source obligations stop distribution. |
| Source/notices payload | Supply the corresponding modified Forgejo source, ordered patches, scripts/locks and required dependency/runtime source/notices beside the exact image through the existing bundle. Match Go build tags/architecture and emitted frontend/assets to their actual versions/licenses; don't accept unbound `go-licenses.json` or swallowed collection errors. Add bounded Soda notice/source access and preserve third-party rights on removal/update. |
| Acceptance checks | U02 authors/runs missing-notice/source, identity-mismatch, wrong-platform and private-input exclusion tests; U03 covers compatibility/migrations; feature owners cover each native patch; U17 exercises the first update/rebase/retirement case. Exact native builds/installed migration/Git/browser proof stay with those milestones and their target scopes. |

**Review disposition:** proceed with this one source-backed build candidate.
No known source-level license incompatibility forces a replacement stack/baseline;
actual complete per-artifact closure and inherited-rights verification remain
mandatory U02 deliverables, not a U01 legal-clearance PASS. Soda maintains/reviews
its own packaging, adapters and carried patches under these existing owners;
Forgejo owns upstream policy/fixes. No new named human is invented or assigned an
upstream maintenance promise.

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

1. Implement the reviewed read/native-auth extraction contracts and required
   native fixes with cohesive patches, native tests/schema, provenance and
   retirement conditions. Review actual patch diffs before accepting them; do not
   repeat U01's source/architecture inventory.
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

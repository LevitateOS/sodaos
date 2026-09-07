# Public website documentation

This is the source handbook for the **release-day SodaOS product**. It teaches
installation, everyday development and operation of the finished product. It is
not a repository progress report. Do not insert candidate revisions, test-VM
instructions, milestone status, missing-build disclaimers or preview routes into
published pages. Implementation and publication gaps belong in
[the internal editorial checklist](../public-docs-review.md) and
[implementation status](../implementation-status.md).

The [architecture](../architecture.md), [leading dashboard plan](../dashboard-implementation-plan.md)
and [scope boundary](../deferred.md) govern the new product. The website's
release-day deployment presentation retains ISO, QCOW2 and Scaleway paths,
x86-64 and AArch64, and personally owned hardware as well as team servers.
WSL2 on x86-64 Windows is explicitly future support, not a launch download.
Do not restore predecessor host developer accounts, Cockpit Projects, managed
checkouts, private toolchain selectors, destructive project workflows or its
separately owned bootc Updates platform.

## Structure and routes

The structure follows the predecessor handbook, with newly authored instructions
for SodaOS. Existing page slugs are retained where their topics remain useful;
retaining a URL does not retain its former product behavior.

- Direct section folders use `NN-Title`, for example `40-Develop`. Numeric
  prefixes determine order; hyphens in the title become spaces in navigation.
- Direct page children use `NN-slug.md`; orders are unique within a section,
  and slugs are unique across the entire handbook. No nested page folders.
- Every page starts with exactly one H1, immediately followed by a descriptive
  paragraph. These become its title and summary; the website renders them
  separately from the remaining body.
- The `index` page maps to `/docs`; other pages map to `/docs/SLUG` regardless
  of section. H2/H3 headings supply the page table of contents and link anchors.
- Link to actual relative Markdown paths, with valid heading fragments when
  needed. Published pages cannot link outside this handbook. Link to upstream
  HTTPS documentation when the task leaves Soda's responsibility.
- Do not use raw HTML, front matter, a source manifest, absolute website paths,
  unsafe URI schemes or symlinks. This README is required but not published.

## Website ingestion

The canonical website checkout is `soda-os-website`. Its
`scripts/docs-sync.mjs` discovers the source tree, validates structure and links,
renders Markdown and emits a deterministic versioned snapshot under
`src/app/docs/generated/`. It accepts this repository's SSH/HTTPS Git origin,
requires clean committed `docs/public` bytes, and records the actual commit and
SHA-256 hashes. Do not fabricate a source revision or edit generated HTML.

After committing source, run in the website checkout:

```sh
node scripts/docs-sync.mjs --source ../sodaos
node scripts/docs-sync.mjs --check --source ../sodaos
node scripts/docs-sync.mjs --check
```

Commit the generated snapshot in that repository. The website's content adapter
uses its manifest for routes, navigation, adjacent pages and source links; Vite
imports the generated HTML and referenced assets. Deployment needs neither this
checkout nor a runtime GitHub fetch. Local integrity alone does not prove source
freshness or that public GitHub links exist before the source commit is pushed.

## Screenshots and review

Use only reviewed captures of the real release interface. Keep lowercase PNG,
JPEG or WebP files under `assets/dashboard/`, `assets/cockpit/` or
`assets/forgejo/`. Put a relative Markdown image after the description, beside
its instruction, with meaningful alt text. Remote images, SVG, data URLs,
symlinks and paths outside `assets/` are rejected. The synchronizer validates
signatures and hashes and copies only referenced images into Vite's asset pipeline.

Follow the unpublished [capture brief](../screenshot-capture.md) for consent,
secret removal and visual review. Missing captures stay there, not as broken
links or generated interface illustrations. Check narrow/wide and light/dark
website rendering after sync; syntax validation is not visual or product proof.

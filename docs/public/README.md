# Public website documentation

Source handbook for the **release-day Soda OS product**. It teaches installation,
everyday development and operation. It is not a repository progress report.

Do not insert candidate revisions, test-VM instructions, milestone status,
missing-build disclaimers or preview routes into published pages.

Product contracts that govern handbook content:

- [Architecture](../architecture/overview.md)
- [Product overview](../product/overview.md)
- [Scope](../product/scope.md)

## Structure and routes

- Section folders use `NN-Title`, for example `40-Develop`.
- Page children use `NN-slug.md`; orders are unique within a section; slugs are
  unique across the handbook. No nested page folders.
- Every page starts with exactly one H1, immediately followed by a descriptive
  paragraph (title and summary for the website).
- The `index` page maps to `/docs`; other pages map to `/docs/SLUG`.
- Link to actual relative Markdown paths inside this handbook only.
- Do not use raw HTML, front matter, a source manifest, absolute website paths,
  unsafe URI schemes or symlinks. This README is required but not published.

## Website ingestion

The canonical website checkout is `soda-os-website`. Its `scripts/docs-sync.mjs`
discovers this tree under `docs/public`, validates structure and links, and emits a
deterministic snapshot. Keep the `docs/public` path stable unless that sync contract
changes.

```sh
node scripts/docs-sync.mjs --source ../sodaos
node scripts/docs-sync.mjs --check --source ../sodaos
```

## Screenshots

Use reviewed captures of the real release interface under `assets/dashboard/`,
`assets/cockpit/` or `assets/forgejo/`. Follow [screenshot capture](../design/screenshot-capture.md).

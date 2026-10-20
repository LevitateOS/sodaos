# Public handbook source

This directory is the source for the release-day SodaOS handbook. Published pages teach the product's intended user journeys in present tense; this README is for maintainers and is not published.

## Sections and ownership

The numbered folders order the handbook by reader task:

| Section | Purpose |
| --- | --- |
| `10-Start-here` | Route readers and explain product concepts and authority. |
| `20-Install` | Choose, verify and install the appliance or Docker distribution. |
| `30-Set-up` | Configure the operator, access, people and provider connections. |
| `40-Factory` | Plan and authorize work, follow issue execution, review delivery and verify outcomes. |
| `50-Spaces-and-Projects` | Join and prepare persistent repository environments and sessions. |
| `60-Develop` | Collaborate and develop with native Forgejo, Git, tools and project services. |
| `70-Operate` | Administer, maintain, protect data and recover SodaOS and its applications. |

Each published procedure has one owner. Link to that owner instead of copying its steps into another page. Product contracts and architecture in `docs/product/` and `docs/architecture/` define intended behavior; existing interface details belong to references. The handbook describes the user outcome without turning internal schemas, source paths or implementation status into user steps.

## Page and link rules

- Use direct Markdown leaves named `NN-slug.md` inside one numbered section folder. Do not add nested chapter folders, front matter, manifests or raw HTML.
- Every page begins with exactly one H1, a blank line, then a descriptive paragraph.
- Section order comes from the folder prefix; page order comes from the filename prefix. Keep page slugs unique across the handbook.
- Published article links use relative Markdown paths to other pages inside `docs/public/`. Those target pages may be authored alongside the current change.
- Keep captures out unless they come from a reviewed, real interface and follow the [screenshot guidance](../design/screenshot-capture.md). Never use placeholder or fabricated screenshots.
- Do not publish task IDs, candidate revisions, milestone status, missing-build disclaimers, internal planning records or temporary routes.

## Sources for maintainers

The current section tree defines the published filenames. The [page responsibilities](../development/public-handbook/page-contracts.md) and [source audit](../development/public-handbook/source-audit.md) retain the allocation and inspected evidence while the website cutover is outstanding; they are not additional product contracts. Use the canonical product, architecture or design owner when changing behavior, and keep unresolved design alternatives out of user instructions.

The website source model consumes this directory from the SodaOS repository. Keep `docs/public/` stable and validate the complete set of handbook links and headings when integrating a content change.

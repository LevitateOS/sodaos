# Public handbook source

This directory is the source for the release-day SodaOS handbook. Published pages teach the product's intended user journeys in present tense; this README is for maintainers and is not published.

## Sections and ownership

The six numbered folders put everyday use first, followed by configuration,
operations and installation. Existing team members can find their work without
reading a host setup sequence; a new operator can follow the installation links
directly from the handbook entrance.

| Section | Purpose |
| --- | --- |
| `10-Start-here` | Understand SodaOS, find work in Spaces, make a first connection and optionally connect a provider. |
| `20-Factory` | Follow the Feed, plan work, understand execution, review delivery, verify results and resolve interruptions. |
| `30-Projects-and-development` | Understand Projects, open a terminal, contribute with Git, use shared services and develop Canlang applications. |
| `40-Configure` | Set access and Factory policy, prepare Projects and CI, and configure private networking or optional Tailscale. |
| `50-Operate` | Diagnose symptoms, administer services and apps, protect data, perform maintenance and recover safely. |
| `60-Install` | Choose and verify a distribution, prepare a VM or install on hardware, use Docker and establish the operator. |

The [complete filetree](../development/public-handbook/proposed-tree.md) and
[page contracts](../development/public-handbook/page-contracts.md) follow this
order. Numeric prefixes and section membership control navigation; each
semantic page slug and its heading anchors remain stable when a page moves.

Each published procedure has one owner. Link to that owner instead of copying its steps into another page. Product contracts and architecture in `docs/product/` and `docs/architecture/` define intended behavior; existing interface details belong to references. The handbook describes the user outcome without turning internal schemas, source paths or implementation status into user steps.

## Page and link rules

- Use direct Markdown leaves named `NN-slug.md` inside one numbered section folder. Do not add nested chapter folders, front matter, manifests or raw HTML.
- Every page begins with exactly one specific, descriptive H1, a blank line,
  then a standalone introductory paragraph of **140–220 characters**, including
  spaces. Explain what the reader will learn or accomplish; the website also
  uses this paragraph for descriptions and adjacent-page previews.
- Move detailed rules from the introduction into the relevant body sections.
  Preserve their meaning instead of dropping them to meet the summary length.
- Use meaningful H2 sections covering the subjects readers need to find.
  Place H3 subsections only beneath a relevant H2. Keep headings specific and
  preserve existing anchors where possible; update callers when an anchor changes.
- Section order comes from the folder prefix; page order comes from the filename prefix. Keep page slugs unique across the handbook.
- Published article links use relative Markdown paths to other pages inside `docs/public/`. Those target pages may be authored alongside the current change.
- Keep captures out unless they come from a reviewed, real interface and follow the [screenshot guidance](../design/screenshot-capture.md). Never use placeholder or fabricated screenshots.
- Do not publish task IDs, candidate revisions, milestone status, missing-build disclaimers, internal planning records or temporary routes.
- Mark a missing documentation fact or procedure with the `doc-pending` contract
  below. This identifies information to add later without changing the intended
  product behavior into an implementation-status report.
- Do not add a separate excerpt field, a manually maintained table of contents,
  or previous/next navigation. The website derives these from the Markdown
  structure and handbook order.

## Article patterns

Choose a task guide, concept explanation or reference pattern according to the
reader's goal. Adapt the headings, omit inapplicable sections and replace every
bracketed prompt. These are templates for authors, not text to publish unchanged.

### Task guide or tutorial

```md
# [Accomplish a specific task]

[Outcome-focused summary of 140–220 characters.]

## Before you begin

[Required access, tools, existing work and linked prerequisites.]

## [First meaningful stage]

[Numbered actions with exact commands or interface names.]

## [Next meaningful stage]

[Actions and relevant examples.]

## Verify the result

[Observable success criteria and how to check them.]

## Troubleshooting

[Relevant symptoms, diagnostics and recovery.]

## Continue

[The logical next task and why to follow it.]
```

### Concept or explanation

```md
# [Concept name]

[Purpose-focused summary of 140–220 characters.]

## [Core concept]

[Definition and responsibilities.]

## [How it works]

[Behavior, relationships and lifecycle.]

## Example

[A concrete example and what it demonstrates.]

## Constraints and related concepts

[Important boundaries and useful links.]
```

### Reference

```md
# [Feature, command or component]

[Scope-focused summary of 140–220 characters.]

## Syntax

[Canonical forms and placeholders.]

## Options and defaults

[Types, accepted values, defaults and required fields.]

## Behavior and constraints

[Permissions, interactions, errors and edge cases.]

## Examples

[Valid examples with their expected interpretation or result.]

## Related reference

[Directly relevant definitions and task guides.]
```

## Readability and completion

Keep each paragraph focused on one idea, usually in two to four sentences. Use
numbered lists for ordered actions, bullets for unordered information and tables
for repeated properties that readers need to compare. Never add empty or generic
sections just to fill a template.

Explain command context, placeholders and consequences before execution. Label
fenced code blocks with the correct language and explain the expected result
after each example. Preserve exact defaults, permissions, exceptions and failure
behavior; never invent behavior or commands to fill a missing section.

## Information to add later

Use a fenced `doc-pending` JSON object when readers need a fact, measurement or
supported procedure that the handbook cannot yet supply. Place it in the owning
section, before the step that depends on the missing information. Keep known
requirements, permissions and behavior in the surrounding article. Other pages
link to that section rather than repeat the notice.

```doc-pending
{
  "id": "measured-host-sizing",
  "title": "Measured hardware requirements",
  "message": "CPU, RAM, disk and update/backup reserve recommendations for basic and small-team workloads will be added after those workloads are measured.",
  "guidance": "Use the workload and storage factors below to plan what needs capacity. Per-application limits do not establish a minimum host size."
}
```

The object uses strict JSON: double quotes, no comments or trailing commas, and
no fields beyond these:

| Field | Required | Meaning |
| --- | --- | --- |
| `id` | Yes | Stable topic identifier, unique within this handbook, matching `[a-z][a-z0-9]*(?:-[a-z0-9]+)*`. Source-only; connects the notice to the existing todo list. |
| `title` | Yes | Short, specific name of the information to be added. |
| `message` | Yes | One or two sentences identifying what is missing and what information or confirmation will allow it to be added. |
| `guidance` | No | A useful, already-supported action or boundary while the information is absent. Omit when there is no meaningful guidance; never supply a guessed workaround. |

All values are nonempty plain-text strings when present. Keep Markdown links
and code in the surrounding article. Do not put HTML, formatting markup,
credentials, private paths, task IDs, owner notes or invented completion dates
in reader-facing fields. There is no status, priority, assignee or due-date
field; the existing handbook todo list owns that information.

### Rendering contract for the website

Render the block as a compact, static note in the article, with the fixed label
**Details to follow**, the topic title, the message and optional guidance.
Use a semantic aside/note with an accessible name from the topic title. Render
the title as note text, not an article heading. Escape all field values as text.
The website owns spacing, borders, icons and colors; the wording must identify
the notice without relying on color or an icon.

Do not show the JSON, `id`, a code-copy button, spinner, progress bar, dismissal
control or a promised date. Do not add the note to the heading outline,
navigation, page description or adjacent-page preview. The existing H1,
opening paragraph and H2/H3 hierarchy remain the source of those features.
The notice stays visible beside the relevant explanation, including on small
screens and in print.

Recognize `doc-pending` before generic code-fence rendering. Reject malformed
JSON, unknown fields, wrong types, empty values, invalid IDs and duplicate IDs
within the imported handbook, with the source page identified. An importer
without support for this contract must report that limitation instead of
publishing raw JSON or silently dropping the notice. Website support must land
before synchronizing pages containing these blocks; this source contract does
not implement the renderer.

### Resolve the information gap

Use notices for missing sizing measurements, identifier discovery, input
acquisition or executable procedures. Do not add one solely because a product
implementation task or a live validation run remains open. Missing screenshots
already have `doc-visual` placeholders. Keep the intended product's selected
behavior in present tense and avoid generic “coming soon” copy.

Map each `id` to its owning page and existing task in the
[handbook todo list](../development/public-handbook/cutover.md#pending-information-notices).
When the input arrives, incorporate it into the ordinary article, check affected
examples and links, then remove the block and update that same task. A resolved
notice does not remain as a “done” card. Remove it only when the missing
information is actually present, not merely because an owner has been assigned.

## Review the completed page

Before handing off a page, check that its opening works as a standalone preview,
the headings explain its coverage, each subsection has a meaningful parent, and
prerequisites, examples and outcomes are clear. Confirm that technical rules
survived restructuring, links and existing anchors resolve, and the repository's
documentation checks pass. Apply this contract in the source handbook before
regenerating the website.

Validate any `doc-pending` block against the contract above and confirm it
describes a specific documentation gap with a task in the existing queue.

## Sources for maintainers

The current section tree defines the published filenames. The [page responsibilities](../development/public-handbook/page-contracts.md) and [source audit](../development/public-handbook/source-audit.md) retain the allocation and inspected evidence while the website cutover is outstanding; they are not additional product contracts. Use the canonical product, architecture or design owner when changing behavior, and keep unresolved design alternatives out of user instructions.

The [handbook remaining tasks](../development/public-handbook/cutover.md) is the
single current documentation queue, with completed work, page allocation and
outstanding captures distinguished. The earlier coverage audit retains evidence;
its original findings are not a second open task list.

The website source model consumes this directory from the SodaOS repository. Keep `docs/public/` stable and validate the complete set of handbook links and headings when integrating a content change.

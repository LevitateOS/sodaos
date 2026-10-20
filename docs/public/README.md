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

Before handing off a page, check that its opening works as a standalone preview,
the headings explain its coverage, each subsection has a meaningful parent, and
prerequisites, examples and outcomes are clear. Confirm that technical rules
survived restructuring, links and existing anchors resolve, and the repository's
documentation checks pass. Apply this contract in the source handbook before
regenerating the website.

## Sources for maintainers

The current section tree defines the published filenames. The [page responsibilities](../development/public-handbook/page-contracts.md) and [source audit](../development/public-handbook/source-audit.md) retain the allocation and inspected evidence while the website cutover is outstanding; they are not additional product contracts. Use the canonical product, architecture or design owner when changing behavior, and keep unresolved design alternatives out of user instructions.

The website source model consumes this directory from the SodaOS repository. Keep `docs/public/` stable and validate the complete set of handbook links and headings when integrating a content change.

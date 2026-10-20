# Knowledge and decisions

Factory knowledge and decision records preserve sources, scope and revisions so workers can assess reuse, while Library Admin maintains items within granted authority.

## Search and assess what you find

Knowledge is useful when its claim is tied to evidence and describes where it applies. Check the source, repository or Project scope, version, limitations and current validity before relying on it. A past failure pattern can guide investigation without proving the same defect exists now. A method that worked with another fixture or version may not establish the current result.

Decision records show who had authority to choose, what scope the choice covers and the evidence or rationale recorded at the time. A worker may record a recommendation or a newly discovered fact, but cannot create product policy or replace an authorized human decision. Contradictory claims remain distinguishable until the evidence or actual decision owner resolves their meaning.

Search results are limited to information the current reader and work scope may access. An empty or incomplete result does not prove that no relevant knowledge exists. Library visibility does not expand repository, Project or source permissions.

When reusing an entry, keep its exact revision with the work it informed.
Changed source or contradictory evidence may make it stale for a new task
without changing what earlier work relied on. Required accepted decisions and
current task facts still apply; a library observation cannot replace the
[accepted issue inputs](40-issue-work-and-dependencies.md#from-issue-to-executable-plan).

1. In the repository Feed, open **Knowledge & decisions**.
2. Enter terms in **Search the library**. Use **Show** to choose Knowledge,
   Decisions or both, then select **Search**.
3. Open an item and inspect its scope, current revision, source references and
   limitations. Follow **Replaced by** or **Derived from** relationships where
   present.
4. Expand **Evidence** and **Recent revisions** to see what supports the record
   and who changed it. Confirm that it still applies to the task's source and
   accepted requirements before reusing it.

## Example: preserve a lesson and a decision

A worker investigating the catalog may establish this narrow lesson:
“At the inspected revision, an empty filtered result does not mean that the
catalog contains no records.” Its knowledge entry identifies the repository,
source revision and check evidence that established the distinction. Another
worker can use it to avoid misleading empty-state copy, while checking that the
behavior still holds.

A different record captures the authorized choice: “Clear search removes the
search text and preserves the other filters.” That decision identifies its
actual owner, accepted scope, rationale and alternatives. A worker's preferred
wording does not create this product decision.

The [research example](40-issue-work-and-dependencies.md#example-answer-a-research-question)
shows why an observed result must keep its tested revision, fixture and limits.
The [standalone tip](40-issue-work-and-dependencies.md#example-deliver-a-standalone-catalog-tip)
uses the accepted behavior to guide its wording. Accepting that text does not
give a writer authority to change the underlying decision; retain the decision
reference so later revisions can check whether the same facts still apply.

Worker learning and recorded decisions appear with their attributable source
and related Feed activity. Recording is separate from search indexing, so a new
item may appear in search later than its confirmed save. Reuse its original
reference instead of submitting the same lesson again.

If the product later changes Clear search to remove every filter, the authorized
replacement decision supersedes the earlier one. The old record remains useful
for understanding previous work; it cannot justify the new behavior. Library
Admin maintains those relationships without supplying the decision authority.

## Maintain the library

A scoped Library Admin can edit, remove, restore, consolidate, split, relate or supersede knowledge and decision items within its granted collection. Maintenance preserves provenance, applicability, visibility and prior revisions. A duplicate suggestion does not by itself justify merging claims; an apparent contradiction may reflect different versions or scopes.

```doc-visual
{
  "placeholder": "Inspect a scoped Project knowledge item with its evidence references and attributed revision history.",
  "caption": "The Project library detail for a fictional public-safe knowledge item, with its scope, owning role, evidence references, and expanded revision history. Markers identify provenance, current revision, and prior attributed changes.",
  "alt": "Project library item showing its scope, evidence references, current revision and attributed history",
  "production": {
    "method": "screenshot",
    "instructions": [
      "In the approved Soda documentation instance, open the repository Feed and select the actual Knowledge & decisions control to reach Project library. Open an existing fictional public-safe knowledge item from the sample collection; do not create, edit, remove, restore, or otherwise curate an item for the screenshot.",
      "Use an authorized documentation account with access to the item. Choose a record that genuinely displays its Project scope, owning role, current revision, evidence references, and at least one prior revision with its recorded reason and actor. Expand the existing evidence and recent-revisions details without changing them.",
      "Use only approved public-safe sample knowledge and evidence references. Do not expose private URLs, personal details, credentials, tokens, device codes, cookies, or profile paths. If the approved instance lacks a scoped item with real evidence and revision history, defer the capture rather than inventing provenance or history.",
      "Record the actual Soda and Fountain revisions and capture date in the asset notes. Capture the real screen at 1280 by 900 in the light theme, then check the labels at a mobile-width preview."
    ],
    "edits": [
      "Crop to the Project library item details and expanded evidence/revision sections, retaining the Project library heading, scope, owner and all labels that identify the history.",
      "Add at most three numbered rectangles to the scope/owner provenance, current item revision/evidence, and attributed prior revision history. Keep text readable, do not cover actor or reason labels, and do not use color alone.",
      "Explain each marker in the caption, identify the sample item as fictional, and verify that the caption and alt text describe only information visible in the actual capture."
    ]
  }
}
```

Removing an item from ordinary discovery does not erase its historical
references or revoke an operative decision. Restoring an item requires checking
its current source, validity and replacements. Superseding a decision requires
the actual authorized replacement; Library Admin preserves its history without
changing what was decided.

Historical references do not bypass current source access. Revoked material
must not remain available through a cached excerpt or consolidated summary.
Wider publication requires separate authority.

If another person edits an item during maintenance, the stale change must be reassessed against the current revision. A saved operation can be confirmed from its receipt if the response is lost. Feed and search indexing may lag behind a confirmed change; that delay does not justify repeating the mutation. Research that produces prerequisite evidence belongs to [Issue work and dependencies](40-issue-work-and-dependencies.md).

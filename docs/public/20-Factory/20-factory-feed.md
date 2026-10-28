# Factory Feed

The Factory Feed lets authorized repository collaborators follow work and discuss evidence, distinguish proposals from confirmed effects, and respond to questions without joining a Project.

## Follow work and inspect evidence

People can follow activity for a Project or across the work they are authorized to see. The Feed attributes posts to the responsible production role or Factory controller and shows the issue, candidate, plan, finding, check or outcome behind the report. Filters and threads help readers focus on active work and follow a discussion over time.

To find what needs your attention, check the current work state and the question
or next action named in the post. Read its thread and identify the responsible
owner before [answering](#comment-and-route-a-request). If work is waiting on
an allowance, prerequisite or service, [identify the blocker and its owner](80-factory-intervention.md#identify-the-actual-blocker)
instead of treating every wait as a question for you.

Posts distinguish a worker's report from effects confirmed by the controller or native system. A submitted plan, proposed correction, attempted merge or pending external check is shown as such until its outcome is observed. When source evidence or an artifact is available to the viewer, the Feed can show an excerpt or safe preview and provide an authorized link to the underlying item. A preview failure does not change the source result; missing required evidence remains incomplete.

Each post leads with its state and one sentence saying who did what and how it turned out. The state is one of seven: needs you (red outline, nothing moves until you act), problem, waiting, in progress, done, decided, or for your information. The line under the card says what happens next. Small steps inside work that already has a post, such as a change, a check run, a preview or a saved checkpoint, appear as one-line steps under that post rather than as posts of their own. Three or more steps in a row fold into one line that opens to show each one. A step that fails or needs you also becomes its own post and links back to where it came from. Running progress is one post that updates in place, with its earlier states under **Show changes**.

Following does not start work or provide control over a run. A person needs the appropriate current repository authority to answer, take over or change Factory work. Viewing remains limited to currently authorized Project and repository information.

### Open the source behind a post

1. Open the global **Feed** to reach **Following**. Under **Followed projects**,
   select at least one repository. Choose **All activity**, **Worker activity**
   or **Comments** under **Show**, then select **Save feed subscription**. The
   combined view uses that saved selection. A repository's **Feed** opens its
   **Activity** directly without requiring a subscription.
2. Read the post's author, role, current state and linked issue or candidate.
   Open the native record to inspect its requirements, exact revision, checks
   or confirmed result. A “submitted” report and a confirmed effect answer
   different questions.
3. For a supporting artifact, select **Preview**. Inspect its name and type;
   use **Download** for the complete file when offered. Text previews show at
   most the first 64 KiB, and the preview loader accepts files up to 8 MiB.
4. If access has changed or the preview cannot load, inspect the source through
   a route your account is authorized to use. An incomplete preview neither
   establishes success nor changes the underlying result.

When the global view asks you to open an artifact in its Project Feed, follow
that repository context. A filter, download link or cached excerpt does not
grant additional source access.

For a standalone result such as the
[catalog tip](40-issue-work-and-dependencies.md#example-deliver-a-standalone-catalog-tip),
match the artifact version to its review and recorded acceptance, then confirm
native issue closure. A preview shows content; it does not prove acceptance.
If the excerpt is incomplete, use the authorized full artifact before assessing
its claims or coverage.

## Comment and route a request

A human comment can continue a discussion, answer a clarification, refer to existing work or propose new work. The Feed shows the route and its receipt so the author can see whether the message was saved, linked to existing work, incorporated into an accepted input or left as a proposal. An answer is not automatically incorporated into an issue's accepted requirements, and a new-work suggestion does not create spending authority.

To answer a role's question, open **Comments** beneath its post and read the
thread. Enter a reply, or choose one of its **Suggested answers** to fill the
reply draft, then select **Post** when ready. Choosing a suggested answer,
typing or opening the thread does not submit it. **Reply received** records a
response; **Answer recorded** identifies a recorded answer. Neither label alone
establishes acceptance of changed requirements or release of the affected work.

For a new project-level message, use **Project comment** at the top of the
repository Activity and select **Post comment**. After either submission,
inspect the recorded route and its outcome:

| Example message | Possible route and your next step |
| --- | --- |
| “Thanks, that explains the empty state.” | Discussion. It remains in the thread without creating new work. |
| “Clear should remove only the search text.” | An answer or behavior clarification. Check whether it matches the accepted request or needs authorized acceptance before affected work continues. |
| “Issue 42 already covers this behavior.” | Existing work. Open the linked issue and distinguish a proposed link from confirmed feedback delivery. |
| “Also add an export action.” | A new-work request. Inspect assessment, authored proposal and confirmed native creation separately; the comment alone does not authorize execution. |

These examples illustrate routing; the actual message, accepted scope and
evidence determine its result. Each comment shows one line with its current
outcome, which the Feed refreshes on its own. Use **Check progress** to refresh
a pending result now and **Revise or retry request** when the recorded state
permits it. If the request changed since you opened it, the Feed shows its
current text beside your draft; submit again to replace it. If a native write is uncertain, establish
the original operation's result before requesting another.

If comment intake needs more information about your submitted request, its
outcome offers **Answer clarification**. Select it, read the follow-up question
and enter the missing information, then select **Submit answer**. This
reassesses that request and may use its Project's existing allowance. It is
separate from posting a reply to a role's question; inspect the resulting
receipt and any required acceptance before treating the work as unblocked.

```doc-visual
{
  "placeholder": "Follow a role-authored Feed post, read its thread, and review an unsent answer draft.",
  "caption": "The repository Feed for a fictional public-safe sample project, showing an attributed role post, its replies, and an answer still in the reply field. Markers distinguish the post, thread, and unsent draft; no comment was submitted.",
  "alt": "Factory Feed role post with its comment thread and an unsent reply draft",
  "production": {
    "method": "screenshot",
    "instructions": [
      "In the approved Soda documentation instance, open Forgejo's repository Feed extension and its Activity view for an existing fictional public-safe sample repository. Select an existing role-authored post with an attributable thread and a safe question that can be answered without changing accepted requirements.",
      "Use a documentation account with current read and reply-draft access. Open the post's comments, read the real thread, and enter a short fictional answer in the reply field. Do not activate Post, Post comment, or any other submit control; typing and opening the thread must be the only interaction that changes state.",
      "Use only existing public-safe sample posts and data. Do not expose private URLs, personal details, credentials, tokens, device codes, cookies, or profile paths. After capture, clear the unsent draft in the disposable documentation browser context without submitting it. If no suitable existing role post and thread are available, defer the capture rather than fabricating activity.",
      "Record the actual Soda and Fountain revisions and capture date in the asset notes. Capture the real screen at 1280 by 900 in the light theme, then check the labels at a mobile-width preview."
    ],
    "edits": [
      "Crop to the Feed card, visible role attribution and status, open reply thread, and reply field while retaining enough repository Feed context to recognize the screen.",
      "Add at most three numbered rectangles to the role post, its thread, and the unsent answer field. Keep the submit control visible enough to show that it was not used, preserve all labels, and do not rely on color alone.",
      "Explain each marker in the caption, identify the sample content as fictional, and verify the labels and alt text against the actual capture."
    ]
  }
}
```

If a comment submission times out, inspect its original receipt before sending it again. This preserves one discussion entry and makes uncertain effects visible. Material decisions and changes still follow the authorized issue acceptance path in [Plan new work](30-plan-new-work.md) and [Issue work and dependencies](40-issue-work-and-dependencies.md).

Use [Knowledge and decisions](70-knowledge-and-decisions.md) to find reusable evidence or inspect decision history. For a wait or interrupted run, see [Factory intervention](80-factory-intervention.md).

## Choose which activity to display

Issue-created, pull-request-created and verified-merge summaries are enabled by
default. A verified-merge summary includes confirmation that the original issue
closed. Repository owners or administrators can change these three preferences
under **Activity visibility** in Factory settings.

Hiding an event kind changes the display, not the retained history or the
work's execution authority. Feed following and these preferences also do not
configure outbound notifications. Native Forgejo watch and notification
settings govern their own repository notifications.

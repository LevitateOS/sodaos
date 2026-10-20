# Factory Feed

The Factory Feed lets authorized Project members follow work and discuss evidence while showing which reports are confirmed effects and which remain proposals.

## Follow work and inspect evidence

People can follow activity for a Project or across the work they are authorized to see. The Feed attributes posts to the responsible production role or Factory controller and shows the issue, candidate, plan, finding, check or outcome behind the report. Filters and threads help readers focus on active work and follow a discussion over time.

Posts distinguish a worker's report from effects confirmed by the controller or native system. A submitted plan, proposed correction, attempted merge or pending external check is shown as such until its outcome is observed. When source evidence or an artifact is available to the viewer, the Feed can show an excerpt or safe preview and provide an authorized link to the underlying item. A preview failure does not change the source result; missing required evidence remains incomplete.

Following does not start work or provide control over a run. A person needs the appropriate current repository authority to answer, take over or change Factory work. Viewing remains limited to currently authorized Project and repository information.

## Comment and route a request

A human comment can continue a discussion, answer a clarification, refer to existing work or propose new work. The Feed shows the route and its receipt so the author can see whether the message was saved, linked to existing work, incorporated into an accepted input or left as a proposal. An answer is not automatically incorporated into an issue's accepted requirements, and a new-work suggestion does not create spending authority.

```doc-visual
{
  "placeholder": "Follow a role-authored Feed post, read its thread, and review an unsent answer draft.",
  "caption": "The repository Feed for a fictional public-safe sample project, showing an attributed role post, its replies, and an answer still in the reply field. Markers distinguish the post, thread, and unsent draft; no comment was submitted.",
  "alt": "Factory Feed role post with its comment thread and an unsent reply draft",
  "production": {
    "method": "screenshot",
    "instructions": [
      "In the approved Soda documentation instance, open Forgejo's repository Feed extension and its Activity view for an existing fictional public-safe sample repository. Select an existing role-authored post with an attributable thread and a safe question that can be answered without changing accepted requirements.",
      "Use a documentation account with current read and reply-draft access. Open the post's comments, read the real thread, and enter a short fictional answer in the reply field. Do not activate Submit reply, Post comment, or any other submit control; typing and opening the thread must be the only interaction that changes state.",
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

If a comment submission times out, inspect its original receipt before sending it again. This preserves one discussion entry and makes uncertain effects visible. Material decisions and changes still follow the authorized issue acceptance path in [Plan new work](10-plan-new-work.md) and [Issue work and dependencies](30-issue-work-and-dependencies.md).

Use [Knowledge and decisions](80-knowledge-and-decisions.md) to find reusable evidence or inspect decision history. For a wait or interrupted run, see [Factory intervention](90-factory-intervention.md).

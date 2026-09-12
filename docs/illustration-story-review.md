# Illustration review and story direction

Review date: 2026-09-10. Starting source: `512afb0`. The user asked for a complete
asset review and explicitly selected redesign and integration of repetitive scenes.
This review supersedes the earlier aesthetic acceptance in the page checklist;
historical screenshots still describe their original image bytes.

## Findings

All **78 tracked image assets under `assets/`** were inspected: 34 papercraft PNGs
and 44 logos, icon exports, backgrounds and a retained logo concept. The 34 scenes
have distinct SHA-256 hashes, but that does not establish visual distinction.
The repeated standing bot, presentation gesture, small prop, elevated camera and
oval blue mat make much of the set interchangeable. The old settings prompt
explicitly imposed that recipe. Consistent identity does not require identical staging.

Home, Login, Dashboard, Organizations and New organization already depict a place
or relationship. Their scenes remain. Explore, Pull requests and Milestones have
useful concepts but drift in face construction, ears, hands, antennas or badges.
People is a posed accessory lineup. The other active scenes need stronger actions.

The palette and transparent cutouts coordinate with the native cream/navy page
surfaces. The presentation problem is mostly staging and scale. Shared header art
is 160px wide and shrinks to about 80–90px in the observed 390px mobile layout.
Fine diagrams, props and room furnishings cannot carry the meaning at that size.
The 1536×1024 source PNGs total 67.24 MiB before this work; ten unused illustrations
are still in the public payload despite having no current template caller.

The repeated logo exports are intentional identity/size/theme variants. The exact
16px and 32px duplicate pairs between desktop icons and web favicons are expected.
Keep canonical SVGs, wordmarks, icon derivatives, Cockpit backgrounds and their
attribution intact. The animated-wave example is a separate demo, not a page scene.
User avatars, repository content and provider marks retain their native purposes;
they are not replacements for, or variants of, the papercraft mascot.

## Character and art direction

Use [Dashboard](https://github.com/LevitateOS/sodaos/blob/cf6beb61e8bda6c6aeeae714ef50d84a2f03a477/assets/branding/forgejo/dashboard-papercraft.png) as the fixed
character/material reference. Match its rounded cream helmet and fine seam, inset
navy face, cyan oval eyes and small thin smile, cobalt ears with cyan centers,
short cream body, navy joints/mittens/soles. Keep one design across multiple bots.
Use gaze, body weight, head tilt and gesture to express a moment. Antennas, chest
badges, alternate face construction and glossy toy materials are not identity variation.

Every scene needs a page-specific action with an understandable before and after.
Vary the camera, number of characters, environment, silhouette and scale. A bot can
inhabit a scene, work beneath a structure, travel through it or interact with another
bot. Scene-specific paper structures provide grounding; the oval mat is optional.
Keep cream/navy/cobalt/mint and warm studio light. Reserve amber for a meaningful
small focal point. Avoid decorative confetti and objects added only to fill space.

A scene should remain distinguishable when its filename is hidden and it is shown
beside neighboring pages at the actual display width. At small sizes, prioritize
one action and a few broad shapes. A coherent series can share metaphors, such as
books for repositories/history/knowledge, while varying the action and silhouette.
Do not bake real state into decoration: no implied successful build, granted access,
finished issue or deleted repository. Native headings and controls remain authoritative.

## Complete scene decisions

Names below refer to `assets/branding/forgejo/NAME-papercraft.png`. These are the
selected direction; the implementation/evidence section records what was actually
generated, accepted, integrated and observed.

| Asset | Decision | Page story / distinction |
| --- | --- | --- |
| `admin-new-account` | Redesign | Operator prepares an empty workstation by pulling out a chair; distinguish from self-service arrival. |
| `dashboard` | Keep | A personal workbench ready to resume work; canonical bot reference. |
| `explore` | Correct identity | Small bot discovers oversized repository folders; retain the prop-dominant composition. |
| `fork` | Redesign | A seated bot draws an independent paper offshoot from an intact shared book; a broad branching silhouette. |
| `home` | Keep | Two developers use their own laptops with a shared machine; the strongest product-specific scene. Setup shares this welcome intentionally. |
| `issues` | Redesign | Two bots investigate a jammed mechanism: one crouches to inspect, the other steadies its lid. |
| `login` | Keep | Seated bot welcomes the user beside an open computer; relaxed return to the workshop. |
| `migrate` | Redesign | Bot pushes preserved archive volumes up a ramp into their new home; a clear direction of travel. |
| `milestones` | Correct identity and interaction | Preserve the stepped ascent; make progress shared by a bot reaching toward its teammate. |
| `new-org` | Keep | Two bots assemble a common structure; creation through cooperation. |
| `new-project` | Redesign | Overhead pair unroll a planning sheet and place the first task; the plan is still mostly empty. |
| `new-repo` | Redesign | Bot braces inside an unfolding book and raises its cover; making room for a new idea. |
| `new-team` | Redesign | Three bots lean into a hands-together huddle; belonging through an actual shared gesture. |
| `not-found` | Redesign | Bot peers around a map from a curling path; curiosity and an available return, without asserting deletion. |
| `notifications` | Redesign | A relaxed seated bot catches one arriving message; a curved delivery path. |
| `organization-packages` | Redesign | Two bots retrieve/return reusable components from opposite sides of a shared cabinet. |
| `orgs` | Keep | Inhabited shared studio; preserve its place-based silhouette, without adding detail. |
| `personal-packages` | Retain unused source | Current personal package view uses native profile/content layout and no illustration. Do not reintroduce a header merely to use this file. |
| `pulls` | Correct identity | Two collaborators discuss proposed changes; preserve the review/conversation concept. |
| `settings-account` | Retain unused source | Current account form has no art caller. Mailbox concept is too close to Notifications for renewed use. |
| `settings-appearance` | Retain unused source | Current theme form has no art caller. A future illustrated surface would need a changing environment, not swatches held up. |
| `settings-applications` | Retain unused source | Current integrations form has no art caller. A future scene should show an actual connection being made. |
| `settings-keys` | Retain unused source | Current credential form has no art caller. Keep real key controls primary. |
| `settings-organizations` | Retain unused source | Current memberships form has no art caller. Member cards repeat Team/Admin account and should not return. |
| `settings-packages` | Retain unused source | Current package controls have no art caller. Keep real retention/deletion controls primary. |
| `settings-profile` | Retain unused source | Current profile form shows the user's actual avatar. Do not compete with it using a mascot portrait. |
| `settings-security` | Retain unused source | Current security form has no art caller. A decorative shield cannot imply actual protection status. |
| `settings-webhooks` | Retain unused source | Current webhook form has no art caller. Its connection story is clearer than other retired settings props, but still unneeded here. |
| `signup` | Redesign | Bot steps through an open workshop doorway and invites the viewer in; arrival rather than operator preparation. |
| `subscriptions` | Redesign | Bots settle into an ongoing bookmarked conversation; distinct from arriving mail or watching repositories. |
| `users` | Redesign | Three bots share a paper-plane launch, with standing, seated and reaching poses; an informal social moment distinct from the team huddle. |
| `watching` | Redesign | Seated bot observes repository structures through a fixed telescope; a gaze across negative space. |
| `wiki-welcome` | Redesign | Bot reclines in a hammock made from an open book, suspended by bookmarks; knowledge as a place to spend time. |
| `workflows` | Redesign | Bot lies beneath a partly assembled conveyor to fit its missing wheel; preparing automation, with an empty output. |

## Evidence and implementation

Before images and SHA-256 inventory are retained under
`.artifacts/image-audit-20260910/`, including light/dark contact sheets, the separate
brand sheet and copies of all 34 original illustrations. The initial browser launch
failed inside the shell sandbox; the reviewed headless Chrome invocation succeeded.
No image-generation API fallback or new image-processing dependency was installed.

The documented `scripts/screenshot.ts` helper captured the existing authorized
`localhost:3300` fixture using its dedicated profile: twelve desktop pages, five
mobile pages and three dark-theme pages. These are observations of the running
local preview, not a claim that every template or current source revision is served.
Profile and Appearance have no header illustration in both source and observed UI.
No accounts, repositories, organizations, projects or preferences were changed by
these read-only captures. Admin/setup/registration-disabled and unavailable organization
routes need separate native coverage; an isolated asset gallery cannot supply it.

Integrated **16 narrative redesigns and three identity/interaction corrections**.
All other 59 tracked images, including the five retained active scenes, are
byte-identical to the starting source. All 34 illustration hashes remain distinct. Exact scene
prompts, reference choices, selected output filenames and rejected variants are in
[story-art-prompts.md](https://github.com/LevitateOS/sodaos/blob/cf6beb61e8bda6c6aeeae714ef50d84a2f03a477/assets/branding/forgejo/story-art-prompts.md); older prompt
records now point to it. The first Wiki replacement was rejected for repeating New
repository's triangular book silhouette; the accepted book hammock has a low U shape.

All 19 accepted PNGs are 1536×1024 RGBA, with transparent corners and alpha 0–254.
Their cream/navy compositing and 160px/88px silhouettes were visually inspected in
`final-candidate-review/`. Painted checkerboards and a failed Pull requests alpha
correction were rejected. The complete 24-scene series was then compared in
`complete-active-series-final/`; its HTML provides before/after, theme and size
toggles. That gallery reuses the recorded per-image alpha checks. Earlier oversized
embedded-gallery and local-file canvas failures are retained as failed attempts.

Removed the ten unused illustrations from the canonical payload, saving **18.97 MiB**
in new payloads while retaining every source PNG. A fresh isolated preview build
passed and contains exactly 24 scene files, each matching its canonical source.
The existing preview mount received only the 19 accepted image replacements, with
its previous bytes backed up in `preview-before/`. All 19 served HTTP bodies match
source SHA-256. No CSS, templates, mounts or services changed.

Final native captures used the existing fixture, stock Forgejo 15.0.7 and
`scripts/screenshot.ts`, with `--scroll-top` and browser-only theme selection:

| Evidence directory | Viewport/theme | Inspected routes, in numbered PNG order |
| --- | --- | --- |
| `native-fresh-desktop/` | 1440×1000, light | Issues, Pull requests, Milestones, Notifications, Subscriptions, Watching, New repository, Migrate, Explore repositories, People, Fork (`/repo/fork/23`), personal New project (`/soda-screenshot/-/projects/new`), Wiki (`/vince/activity-playground/wiki`), intentional 404 |
| `native-fresh-mobile/` | 390×844, light | Issues, New repository, People, Subscriptions, personal New project, Wiki, intentional 404 |
| `native-fresh-dark/` | 1440×1000, dark | Issues, New repository, People, Wiki |

These **25 final captures across 14 pages** show the accepted art beside real
native headings, controls and content, with clean edges and no illustration overlap.
The initial strict `--verify` attempt rejected pre-existing `components.css` byte
drift in the local preview. Final captures therefore describe its currently served
styles, not verification of the full current source presentation. Earlier captures
also exposed cached old images: the preview returns a six-hour asset cache lifetime.
Only the dedicated fixture browser's HTTP cache was refreshed; cookies and saved
login were preserved. Other already-open browsers may need a hard refresh.

Admin account creation, New team, Organization packages, enabled Signup and
no-workflows Actions have source-placement and asset-gallery review, but no matching
accessible native fixture state in this pass. Those five native rendering checks
remain open. Retained scenes' other conditional callers likewise gain no new runtime
claim. No forms, account preferences, repositories or fixture configuration changed.

Required `bun run typecheck` passed, including the analyzer and its fixture contracts.
Its first attempt found missing installed analyzer dependencies; the existing frozen
lock was installed with scripts disabled, with no manifest/lock changes. The three
canonical-payload tests passed again after integration. The preview build, hash/
preservation checks and visual evidence are local checks, not installed-appliance
acceptance or a deployment. No full native build, provider call or rollout occurred.

# Native/Soda page-and-state inventory (NFX-I02)

Isolated subjects only: synthetic fixture accounts/projects, the task-owned
native host socket and private test databases. No human session, Project or
terminal content is a fixture. Secrets and terminal content are masked in
every receipt; only IDs, states and masked markers are recorded.

| Page / state | Subject | Owner | Evidence driver |
| --- | --- | --- | --- |
| Native session generation (`/-/native/session`) | Fixture user `user2` | Fountain native | `soda_extension_terminal_live_test.go` |
| Workspace panel, new terminal (`create`) | Synthetic project `p…3`, login `soda-tester` | Soda terminal (D) | Fountain live test; `tests/frontend/terminal.test.ts` |
| Workspace panel, attach existing (`attach`) | Same synthetic project | Soda terminal (D) | Fountain live test; `terminal-stream/retirement` tests |
| Terminal ready/attached polling | Reserved session ID (hex) | Soda terminal (D) | Fountain live test state loop |
| Logout revocation closes socket | Fixture user session | Fountain native authority | Fountain live test `waitClosed` |
| Membership removal closes socket | Fixture membership row | Soda Store/API | Fountain live test |
| End terminal, then absent | Reserved session ID | Soda terminal (D) | Fountain live test; `terminal-end.test.ts` |
| Inert Reserve boundary (no host socket) | No native subject | — | Skipped, never passing preservation evidence |
| Browser fixture mount (synthetic API/renderer) | Fake socket/renderer in Chromium | Soda fixture (I) | `tests/frontend/fixtures/terminal-{fixture,driver}.ts` |

Rules: skipped or inert paths never count as preservation; a preservation
claim requires an actual renderer and socket object plus observed output.
Installed page/process coverage belongs to NFX-I08 against this inventory.

The following matrix extends those terminal subjects to the native and Soda
page families. A required state is a coverage target, not a claim that the
current installation or fixture contains it. I08 records the actual requested
and resulting path, HTTP status, visible state, viewport/theme, browser and
resource errors, geometry, masked screenshot, and unavailable reason. Existing
synthetic subjects are reused; capture does not create repositories, issues,
requests, Projects, terminals or provider executions to fill a row.

| Page family / normal native routes | Required states | Owner / actual subject | Evidence driver and limits |
| --- | --- | --- | --- |
| Sign-in, sign-up and password recovery (`/user/login`, `/user/sign_up`, `/user/forgot_password`) | Guest form, available authentication/captcha control, validation/refusal if already observed, authenticated redirect | Fountain document/authentication; isolated operator | I08 installed capture; login is the only credential submission. Auth widgets remain native document flows. |
| Native denial and missing page | Guest protected-route redirect; native missing synthetic repository; forbidden/unavailable response if actually returned | Fountain routing/authentication; fixture namespace | I08 captures real status/error pages, including redirects. It does not forge 401/403 or change permissions. |
| Home/dashboard and feeds (`/`, `/issues`, `/pulls`, `/notifications`) | Available/empty lists, real sort/filter/navigation controls, native content return after Workspace, mobile navigation | Fountain; synthetic operator | I08 installed capture and natural links. Notification mutations are excluded; non-fixture list rows are masked. |
| Explore/profile/organizations (`/explore/*`, `/soda-operator`, `/org/create`) | Available/empty results, profile navigation, empty create form, native search/filter state if linked | Fountain; admitted synthetic identities only | I08 bounded same-origin discovery. Unadmitted identities/organizations are masked and their detail pages are not visited. |
| Repository code and clone (`/{fixture-owner}/{fixture-repo}`, `/src/branch/*`) | Native repository header, clone control, file/tree, branch/tag selector closed/open, empty/missing file if returned | Fountain repository widgets; existing synthetic repositories | I08 installed capture; native widget source/component checks supplement it. No clone/download or repository write. |
| Branches/tags and commit history (`/branches`, `/tags`, `/commits/branch/*`, `/commit/*`) | Available/empty lists, branch/tag choice, history, commit/diff view, initialized code/notes/status controls | Fountain native repository widgets; fixture revisions | I08 discovered existing read links only; it does not manufacture branches, revisions or statuses. |
| Repository activity and charts (`/activity`, `/activity/contributors`, `/activity/code-frequency`, `/activity/recent-commits`) | Available chart/list, empty/loading/unavailable state if rendered, return/re-entry | Fountain native chart/Vue entry owners; fixture repository | I08 installed capture; focused native entry teardown tests cover pending imports and retirement separately. |
| Issues, labels and milestones (`/issues`, `/issues/new`, `/labels`, `/milestones`, existing detail links) | Available/empty lists, issue detail, labels/milestones, new editor form, existing rendered Markdown/reactions/attachments | Fountain; existing synthetic issue subjects | I08 native pages. No issue/comment/reaction/attachment submission or server Markdown-preview POST. |
| Pull requests and comparison (`/pulls`, existing detail/compare links) | Available/empty list, detail/diff/status, review/merge controls visible and inert, existing original navigation | Fountain; existing synthetic pull subjects | I08 read pages only. No review, Merge, Close or branch deletion. Missing pull subjects remain unavailable. |
| Releases and packages (`/releases`, `/releases/new`, package links) | Available/empty list, existing detail, empty release form, initialized editor/selector | Fountain; fixture repository/package subjects | I08 capture; no upload, publish, tag creation, download or package deletion. |
| Wiki and repository search (`/wiki`, existing wiki/search links) | Existing/empty/start page, native search controls, linked revision/page | Fountain; existing fixture wiki | I08 read pages. No wiki creation/edit or source-content mutation. |
| Projects and boards (`/projects`, existing board links) | Available/empty project list, existing board/cards, project controls without submission | Fountain; existing fixture board | I08 read pages. No board/card/project creation, drag/drop or update. |
| Actions (`/actions`, existing run/job links) | Available/empty workflow list, existing run/job/log/status, dispatch controls visible and inert | Fountain Actions; existing synthetic run | I08 read pages/polling. No dispatch, rerun, cancel, runner registration or provider execution; log bodies are masked. |
| Repository settings (`/settings`, collaboration/branches/tags/hooks/keys/LFS/units and available leaf links) | Native navigation, existing settings forms and choices, expanded non-mutating controls, disabled/refused controls if observed | Fountain; admitted fixture repository | I08 capture; no Save/Delete/Add/Transfer/Archive or integration call. Secrets, keys and hook/token values are masked. |
| Personal settings (`/user/settings/*`) | Profile/account/appearance/security/keys/applications/organizations/repositories/packages/hooks/blocked users; available tabs/forms and ordinary native links | Fountain plus Soda personal-settings leaf; synthetic operator | I08 capture; native saved account preferences are unchanged. No key/token/credential/security enrollment or account mutation. |
| Operator administration (`/admin`, users/orgs/repos/packages/hooks/auths/emails/runners/config) | Available list/settings pages and native permission/refusal, navigation and controls | Fountain administrator; synthetic operator | I08 read pages. All administration data rows and sensitive configuration values are masked; no non-fixture user detail or administrative submission. |
| Tailnet (`/tailnet`) | Actual connected/disconnected/unavailable state, operator-only admission/refusal, native branded controls | Soda Tailnet leaf and Fountain admission; isolated operator | I08 actual installed state only; peers/authentication URLs and addresses are masked. No login/connect/policy/provider mutation. |
| Global/repository Workspace (`/workspace`, `/{fixture-owner}/{fixture-repo}/workspace`) | Available/empty/recovery state, Projects list, selected fixture Project, settings entry, one native shell and current allocation | Soda shared Workspace owner mounted by Fountain; existing fixture Projects | I08 current native host capture, I04 composed navigation and I06 mandatory terminal identities. No iframe wrapper or forced consumer height. |
| Workspace new entry (`/workspace/new`) | Actual Create project entry button, empty intent form, department dialog, planning limits, keyboard/focus and short viewport | Soda intent/setup; synthetic operator | I08 presentation-only controls. The dedicated `.soda-create-another-project` entry is distinct from Configure's provisioning button. No Explore/Generate/Save/Create submission. |
| Saved request/review (`/workspace/requests/{admitted-id}`) | Existing draft/accepted/closed request, stored walkthrough/recommendations, moved-head/edit notice or actual unavailable response | Soda intent/review and native request alias; explicitly admitted existing synthetic request IDs | I08 optional existing-ID input; G07 captured-byte replay is separate. No new exploration, allowance write or missing-state synthesis. |
| Project settings within Workspace | Overview/Access/Network/Factory tabs, available/unavailable read, existing join/profile/budget/authority messages | Soda project controls and retained Go API/Store owners; fixture Project | I08 real visible tab controls. No Join/Start/Stop/Apply/key save/network change/factory intervention. |
| Native → Workspace → native and reverse | Both original normal-link sequences, retained single navbar/document/panel owner, correct URL/query/history and initialized arriving controls | Fountain native transition plus Soda host mount handle | I04/I06 actual identity checks; I08 screenshots record the installed link result and geometry. A PNG alone cannot prove object/process continuity. |
| Compact, short and light/dark views | Existing page allocation at desktop and 390px viewport, visible controls, measured overflow; browser-only light/dark | Native CSS host allocation plus Soda content | I08 uses actual installed CSS and existing `setCaptureTheme`; F05/F06 actual Chrome zoom checks remain separate from viewport screenshots. |
| Logout/session return | Actual native logout/sign-in page; revoked session state if returned | Fountain native authority; task-owned operator session | I08 final logout permits only its native session endpoint; I05/I06 stream/binding proof remains separately attributed. |

Capture uses only the existing `soda-operator/soda-workspace` and
`soda-tester/q08-journey` repository fixtures and synthetic operator account.
Additional saved-request IDs must be explicitly supplied as existing fixture
subjects. The prepared I08 driver is ignored task evidence at
`.artifacts/codex-i08-20261013/capture-pages.ts`; it is run only after matching
delivery is ready. Its manifest/index retain unavailable states and failures;
captured pages are observations, not aggregate acceptance or provisioning proof.

Mask passwords, email, private/access/registration tokens, authentication URLs,
SSH/private/public key material, terminal and Actions log bodies, Tailnet peers
and addresses, non-fixture repository/account/organization rows, and sensitive
administration configuration before saving images. Keep credentials out of
arguments, logs and metadata; use the existing restricted fixture password file.
Record route-only URLs and scrub browser error text. Browser profile, NSS trust
copy, caches and capture output are private repository artifact directories under
`/home`. Preserve native installed certificate validation: use the existing
browser-home trust through `launchNativeBrowser`, never certificate-error flags.

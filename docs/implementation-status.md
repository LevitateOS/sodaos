# Current handoff

## Integrated E2E execution started — isolated fixture only

The user authorized proceeding with E2E testing after the explicit build/native
journey, Stop/Start and temporary SSH-key rotation proposal. Execution is bounded
to this x86_64 builder and `soda-native-spaces-658f2af`; preserve its original project,
accounts, keys, later writes and evidence. Fresh paired backup precedes fixture
service changes. No `soda-test`, provider, routing, project deletion or host reboot
is included. New work/evidence: `.artifacts/e2e-806d0d9/`.

Clean `806d0d9` passed full native build, `check-native.sh x86_64` and verified bundle
export from its own detached worktree. Read-only pinned-SSH preflight confirmed the
original CID/image, one project/two memberships/two keys, schema5, running services
and query-free native logging. No fixture mutation or browser proof yet.

Preflight reproduced a real lifecycle compatibility blocker: systemd
`259.8-1.fc44` supplies the global `service.d/10-timeout-abort.conf`, setting only
`TimeoutStopFailureMode=abort`. The helper wrongly required no drop-ins whatsoever.
Source now admits exactly that stock vendor path (or no drop-ins), while refusing
all other/combined overrides and retaining fixed unit/CID validation. Host policy
was not changed. Focused host/web tests passed; the corrected candidate still needs
its own clean build/check and integrated native execution. Initial bundle/evidence
remain preserved, not relabeled as the corrected build or E2E success.

## Mounted Sodaspaces management and terminal in the established UI

The native repository button/right dialog now mounts the complete control component
through a single `sodaspaces.js` shell. Its historical duplicate API/action caller is
removed. Connect/logout, create, saved-key add/remove, join, own-key review/Apply,
Start/Stop, SSH/native Copy, Refresh and explicit terminal Open/Disconnect remain
separate actions. Existing native forms/navigation/notification hooks are preserved.
The drawer uses shared typography/tonal 44px buttons, responsive width and scoped
terminal styles. It retires on close/stale context; reopening requires an explicit
full-page reload rather than evading terminal or uncertain-operation guards. Refresh
cannot remount a used terminal. Mutations recheck session identity before dispatch;
JSON reads are MIME-checked and streaming-bounded to 64 KiB. No backend authority,
provider rule, schema, lifecycle or key semantics changed in this UI integration.

The merge's partial-payload packaging hold is resolved **in source**: staging and
the compiled verifier share `internal/nativebuild/forgejo-payload.json` (357 exact
entries, including 229 templates, presentation assets/fonts/notices and generated
locale/terminal inputs). The installer refuses unsafe ancestors and occupied
customization destinations, and chowns only admitted entries, not mutable data trees.
Native build now verifies the locked complete upstream 15.0.7 English catalog before
merging the Soda namespace; GPL/font/Soda/renderer notices remain paired with payloads.
This is not a new native candidate build, exported bundle or installed frontend.

Local checks passed: full Go suite; focused web/host/store/nativebuild races; 65 Node
tests (two opt-in/export-dependent checks skipped); 60 Python build tests (one opt-in
Caddy check skipped); shell/installed-probe syntax, documentation and whitespace.
A separate opt-in sandboxed Chromium layout test passed all 16 combinations of
1440/900/390/320 widths, light/dark and running/stopped, with real source styles and
hash-verified xterm, native-form preservation, keyboard escape and reload behavior.
Its APIs/WebSocket are synthetic: **not real Forgejo/OAuth/helper terminal proof**.
Logs and screenshots: `.artifacts/ui-integration-c6df099/`. Initial failures are
retained: presentation hook snapshots required explicit integration review, a closure
test wrongly required stock `custom/extra_tabs`, and the actual-stage packaging suite
refused to run without `SODA_STAGE` (zero tests; not a passing stage check).

The existing installed journey was adapted to reload/retired-context semantics and
current status text but was not executed. Combined native OAuth/proxy/helper/browser
proof, full candidate build/check/stage and native Start/Stop persistence/new-key
success/old-key refusal remain pending. No dependency install, service reload,
VM/project/account/key/provider mutation, retained rollout or push occurred. The
previous terminal-only fixture approval does not cover lifecycle/key mutations;
obtain exact scope first. Destroy and operator runner relocation remain separate.

## Merge of Forgejo presentation and Sodaspaces work

Merged remote `f838b80` with local `a652450`, preserving both histories, native
presentation/notification/avatar changes and the local environment/terminal/security
work. Shared header/footer hooks retain both features. Avatar requests remain public
and credential-stripped; other `/-/soda/*` traffic retains same-origin API/OAuth
credentials. No separate Soda browser origin or old schema/deployment state was restored.
Both real upstream dependency/checksum sets and packaged license notices are retained.

At this merge, the hooks referenced shared presentation templates/assets beyond the
then-current bounded staging allowlist. Staging refused that incomplete payload.
The source inventory above supersedes that hold; native delivery validation is still needed.
This merge does not deploy the preview, mount the new management module or change any
retained fixture/service/account/key. Imported preview evidence refers to its original
workspace and is not newly executed evidence here.

Checks: full Go suite, 121 local Node tests and Python build fixtures (58 passed,
one opt-in Caddy check skipped) passed. Focused Go races and documentation checks
passed; the resolution diff against `origin/main` is whitespace-clean. Existing remote
whitespace in native-parity templates/notices/assets is retained, not rewritten as
part of conflict resolution. Logs are under `.artifacts/merge-f838b80/`. An initial combined Forgejo
browser-test invocation failed at import because root Playwright is unavailable;
those browser-dependent tests remain unverified, not counted as passing. Original
merge/test failures are preserved. No dependency install, native stage/build,
browser fixture, deployment or provider action was performed.

## Selected tonal buttons — uniform 44px sizing

Local presentation `2026-09-08.17` implements the user's selected C — Tonal,
44px design, superseding revision .16's mixed sizes. Primary actions use tinted
surfaces and blue text; neutral actions are open. Shared buttons use 6px corners,
600-weight 14px labels and 14px horizontal padding. Native mini/tiny/small/compact
classes, icon actions, count labels and adjoining single-value controls all align
at 44px. Competing form, repository, home and administration button declarations
were removed. Native semantic colors, loading/disabled behavior and joined edges
remain. Single-value principal-form fields align with actions; multiline and
multiple-selection controls retain growing areas. The narrow header now fits
44px targets at 320px without shrinking the canonical logo.

Go Forgejo source checks passed. The full browser suite passed 39 checks with two
opt-in native session tests skipped; the dedicated native settings suite passed
11, including menu navigation, avatar dialog/focus and no-JavaScript behavior.
The final focused component suite passed 12 checks, including the additional
native-size and adjoining-input assertion. A read-only native audit recorded
1,298 control measurements across 13 routes, light/dark and 1440/900/390/320px
(104 page states), with no size failures, horizontal overflow or browser errors.

Final verified real-route captures are under `.artifacts/tonal44/release-desktop/`
and `release-mobile/`: Appearance, repository code, issue creation, Keys and
Explore. Representative final desktop/mobile captures were visually reviewed.
The production-derived gallery remains `.artifacts/forgejo-presentation/`;
`.artifacts/button-options/` is the separate design-choice comparison, not native
route evidence. Check logs and dimensional evidence are in `.artifacts/tonal44/`.
Admin/provider/conditional and populated credential states, long translations
and submission journeys remain unverified natively. No account data or preferences
were submitted. Activation used local template reloads only; no container restart
or appliance rollout occurred.

## Pronouns removed from Soda presentation

Personal profile editing, the pronoun privacy checkbox, administrator user editing
and public profile display no longer expose pronouns. The public-profile partial
is an attributed exact-native override with only its pronoun suffix removed.
The gallery and coverage inventory (229 overrides/helpers) reflect this change.
Hidden personal/admin form fields retain existing native values so unrelated
saves do not implicitly clear data. Forgejo's database, API and locale catalogs
remain unchanged; this is presentation removal, not an upstream feature fork.

Local revision `2026-09-08.15` is active via template reload. Go Forgejo checks,
five source/inventory checks and eleven settings browser checks passed. Verified
dark desktop settings/public-profile and light mobile settings captures under
`.artifacts/profile-without-pronouns/` were visually reviewed. Native profile
mutations and administrator routes were not exercised; administrator parity and
hidden-value contracts were checked in source. No saved data was changed.

## Settings menu link activation fix

A native pointer-click reproduction showed focusout closing the settings menu
before its destination link received focus, cancelling navigation. The handler
now checks `relatedTarget` instead of the transient `document.activeElement`.
Local presentation `2026-09-08.14` is active via template reload. Eleven settings
browser checks passed, now including twelve real link navigations across all
three menu groups at desktop/mobile widths with HTTP status, URL and page-heading
assertions. The Go Forgejo suite and four source/inventory checks also passed.
No account data was submitted or changed.

## Clickable avatar and modal dialog

The actual profile portrait now opens the native avatar form in a browser modal
dialog. A translucent dark pencil overlay appears on hover/focus and remains
visible on coarse pointers. The same form is moved, never copied; native upload,
source selection and deletion hooks remain intact. Escape, explicit close and
backdrop clicks restore focus; Tab wraps within the dialog. Without JavaScript,
unsupported dialogs or with server errors, the expanded inline editor remains
available alongside authoritative alerts. The local preview is active at
`2026-09-08.13` via template reload only.

The Go Forgejo suite, four source/inventory checks and eleven settings browser
checks passed, covering seven widths, keyboard/overlay/modal behavior and the
actual native no-JavaScript fallback. Verified dark 1440px and light 320px captures
in `.artifacts/avatar-modal/release-{desktop,mobile}/` were visually reviewed.
Earlier captures in that directory tree expose a corrected native dialog-style
conflict and are not final evidence. No upload/delete POST was submitted; native
server-error and lookup-enabled submission journeys remain unverified.

## Avatar source simplification

The avatar editor now starts directly with file selection when Gravatar is disabled;
it submits the native `source=local` field without showing a lone radio. When
lookup is available, both source radios and the saved selection remain native.
Active local presentation is `2026-09-08.12`, via template reload only. The Go
Forgejo suite passed, including rendered checks for both capability states and
both saved source selections; four source/inventory checks passed. Verified dark
1440px and light 320px captures under `.artifacts/avatar-source/` were visually
reviewed. No account settings, upload or deletion were submitted; lookup-enabled
runtime coverage remains unavailable on this fixture.

## Avatar editor refinement

The profile avatar disclosure now aligns its source and upload fields without
native indentation, uses a single native file-selector boundary, compact help
text and a borderless explicit delete action. Upload constraints, source choices,
form handlers and delete hooks are unchanged. This is active only in the existing
local preview at presentation `2026-09-08.11`; templates were reloaded without a
container restart. The Go Forgejo checks, four source/inventory checks and ten
settings browser checks passed. Verified dark desktop and light 320px captures
were visually reviewed under `.artifacts/avatar-refinement/final-{desktop,mobile}/`;
an earlier 390px capture was also reviewed. No upload or deletion was submitted.
The earlier comprehensive review package below predates this focused refinement.

## Personal settings structural overhaul — local review candidate

The approved personal-settings overhaul is implemented through Forgejo 15.0.7
configuration, template overrides, shared styles and a small presentation-only
script. The existing local preview is active at presentation `2026-09-08.10`.
There is no personal-settings sidebar or artwork hero: one actual identity row
and grouped destination menus compose around distinct profile, preference,
security, inventory and focused-editor layouts. Mobile uses one disclosure below
900px. Native routes, permissions, handlers and save boundaries remain upstream.

Profile retains one identity/address/privacy save and a separate avatar form.
Account places email management before its disclosed password editor and final
deletion warning. Appearance retains four saves. Security retains native factor
state and links to Account for passwords. Keys, Applications, resources and
conditional operational/child pages retain their native structures and gates.
Shared OAuth, runner, webhook and cleanup adapters use explicit personal-caller
inputs and preserve native root context; the organization Applications flag is
not used as a personal-only presentation gate. The test-only inventory covers
228 overrides/helpers, including 37 personal-settings files, their native callers,
compositions and required states.

The complete English locale was generated from the exact embedded native 15.0.7
INI plus Soda additions. Duplicate keys/namespaces are rejected; native INI bytes
and JSON catalogs retain upstream ownership. The complete file was copied into
the existing preview volume and activated with the single user-authorized restart
of `sodaos-local-forgejo`, retaining its image, configuration and data. Subsequent
template changes used native reloads. No appliance deployment occurred.

Actual checks and review evidence:

- `GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off SODA_FORGEJO_GALLERY=1 go test
  -mod=readonly ./scripts -run TestForgejo -count=1` passed. Tests retain native
  control/capability contracts, mandatory enrollment gates, caller-specific title
  handling, selected cleanup values and native OAuth/runner root context.
- The complete existing Forgejo browser suite passed 37 checks, with its separately
  invoked personal-session journey skipped in that batch. The dedicated settings
  run passed all 10 checks, including navigation at 1440/1000/900/899/768/390/320px,
  Escape/outside dismissal, focus return, password and avatar disclosures, key
  panel focus, token-select dimensions, no-JavaScript fallback and fixture errors.
- Read-only native inspection covered 14 routes at six widths: 84 combinations
  without document overflow or page errors. Forgejo 15.0.7 uses Go's native
  `http.NewCrossOriginProtection` in `routers/web/web.go`; an initial audit's
  hidden-CSRF-input assumption was corrected against exact source. That middleware
  was not changed. No POST security/submission journey is claimed.
- The review package contains 94 verified native captures in light/dark at
  1440×1000 and 390×844, including full-page content and open editors. Requested
  URLs, status, landmarks, presentation revision, stylesheet hashes and browser
  errors were checked by `scripts/screenshot.mjs`. The helper now supports
  `--full-page` and forces a fresh document for fragment-only capture requests.
  Failed intermediate capture attempts are excluded from the review manifest.
- `.artifacts/personal-settings/review.html` groups native evidence by family and
  links the actual check logs, route observations and verification sidecars.
  `.artifacts/forgejo-presentation/gallery-{light,dark}.html` is the separate
  production-derived component gallery. Earlier generated concepts remain in
  `.artifacts/settings-design-concepts/`; they are not native evidence.

Native landing pages and accessible child editors were visually reviewed, including
below-the-fold controls. This is not full visual or functional acceptance.
Remaining prerequisites include enrolled/mandatory TOTP and WebAuthn, recovery and
key verification challenges, populated credentials/OAuth grants/applications,
providers, organization memberships/adoption permissions, configured webhooks and
delivery history, populated cleanup previews, Actions/runners/secrets/variables,
and enabled Storage/Quota. Successful/error POST journeys and native fallback for
untranslated Soda additions were not manufactured. Nonpersonal shared callers
have source and regression checks, not newly authorized owner/admin sessions.
No credentials, saved preferences, resources or permissions were created or changed.

The earlier [UX investigation](forgejo-settings-ux-proposal.md) and its evidence in
`.artifacts/settings-ux-investigation/` remain the baseline for comparison, rather
than the current implementation contract. See `appliance/forgejo/README.md` and its
locale guide for the maintained presentation and local activation contracts.

## Ordinary repository container spacing

Local revision `2026-09-08.5` removes the extra 24px top padding from project
lists and wiki revisions. The ordinary repository body container now explicitly
owns zero padding; the shared repository header supplies the navigation gap.
Native fluid/explicitly padded canvases keep their separate gutter contract.
The earlier redesign had consolidated width without removing these family-local
insets; its coverage did not establish equivalent container spacing.

A read-only native regression compares Code, Projects, Issues and Releases at
1440px and 390px, asserting equal padding, width, left alignment and gap below
repository navigation. It passed, as did focused Forgejo Go checks. Wiki revision
padding removal is source-reviewed; no wiki fixture was created for verification.
Templates/assets refreshed only in the existing local preview.


## Repository file toolbar correction

Local revision `2026-09-08.4` aligns the branch picker, compare/find/add controls,
clone protocol buttons, URL field, copy and menu controls to the same 44px row.
The shared toolbar stylesheet owns their geometry and joined edges; the previous
40px clone-input minimum is removed. Dropdown contents remain native. A bounded
flex override lets the URL shrink instead of overflowing narrow viewports.

Focused Forgejo Go checks, both inventory checks and all 10 component-boundary
checks passed. The new toolbar case includes the owner-only Add file button,
light/dark and 1440/390/320px layouts. Native non-owner desktop/mobile captures of
`/alice/activity-workbench` passed route/revision/style verification and were
visually inspected: `.artifacts/screenshots/capture-vUErxz/` and `capture-MkUotY/`.
Owner-only Add file is fixture-tested, not claimed as authenticated owner evidence.
The earlier complete review package remains a record of revision `.3`; this is a
scoped local correction, with no form submissions or appliance deployment.


## Presentation redesign local candidate

Candidate `2026-09-08.3` is active only in the existing local Forgejo preview.
The inventory accounts for all 206 overrides/helpers and their local/embedded
15.0.7 callers across eight compositions. 125 templates select new explicit
presentation roles; the remaining helpers/native structures use existing shared
components or specialized family composition. Shared controls, typography,
spacing, open sections and editor containers replace duplicated page rules.
Native routes, inputs, permissions, CSRF and script hooks remain upstream-owned.

Actual checks: focused `go test ./scripts -run TestForgejo` passed using the local
toolchain and offline dependency settings; all 30 opt-in Node checks passed,
including component states, native Explore overflow/navigation, notification
lifecycle, milestone layout, gallery responsiveness and both sides of inventory
coverage. The gallery uses production intro/empty partials and the real registry,
with minimal native markup fixtures. It passed light/dark at 1440, 900, 390 and
320px. Native template source was read through the running container's embedded
resource export; no Forgejo source fork or rebuild was introduced.

The local review package is `.artifacts/forgejo-presentation/review.html`, with
light/dark galleries, a manifest and 36 verified native viewport captures across
nine accessible routes at 1440×1000 and 390×844. Each final PNG has a sidecar with
URL/status, landmark, revision, registry and stylesheet hashes, viewport/theme
and browser errors. The signed-out account-settings redirect was rejected with
no accepted image. Visual inspection caught and fixed primary-anchor text losing
contrast; a focused regression check now covers it. Earlier `review-*` and
`login-*` capture folders are superseded by the `final-*` candidate images.

This is **not full visual acceptance**. Owner-only repository editors/settings,
administrator pages, organization fixtures, setup/provider/authentication states,
populated packages and specialized canvas/permission/interaction states remain
unverified on native pages. No resources or permissions were created to fill
those gaps, no forms were submitted, no saved preferences were changed, and no
appliance rollout occurred. See `docs/forgejo-presentation-review.md` for the
review entry points and precise evidence limits.

## Shared repository form presentation

Issue/PR composers and milestone, project, release and wiki forms now opt into
the existing shared form controls. Repository creation/editor pages share their
container width, open heading treatment, explanatory copy and divider spacing
in `components-forms.css`. Removed competing wiki/release/project layout rules
and file-editor header/commit-choice cards rather than layering another theme.
Native templates, form actions, field names, editor internals and gates are unchanged.

Focused Forgejo checks passed after updating the stylesheet-ownership assertion
for release forms; the initial stale assertion failure was resolved by moving
ownership, not changing the native-body checks. New-issue desktop capture was
inspected (`.artifacts/screenshots/capture-cfXgkj/`); browser measurements verified
1440px/390px widths without horizontal overflow and one issue form. The separate
manual screenshot profile was signed out: milestone/release captures showed login
and wiki/project showed 404 (`capture-xXavbV`), not successful form evidence.
Remaining permission-restricted forms and dark rendering need native visual review.
Local templates reloaded for CSS versions only; no submissions or deployment.

## Reduce decorative cards and dividers

Shared toolbar, list, empty-state and form-section frames are removed. Native
settings navigation and attached form sections use open backgrounds; profile
privacy controls no longer have an extra enclosing box. Milestone cards and
descriptions, repository-sidebar rules and the landing README frame are removed.
Blank settings/milestone dividers retain a 24px margin; other removed section
borders retain their existing padding and gaps. Control, table-row, alert and
dialog boundaries remain. Changes are scoped to existing presentation owners.

Focused Forgejo checks and whitespace checks passed. Local templates were reloaded
for stylesheet versions. Desktop captures of repository/profile/explore and mobile
profile/explore/milestone detail were inspected (`capture-wlMSyp`, `capture-BGWs4M`
under `.artifacts/screenshots/`); inspection prompted removal of a remaining native
README segment frame and profile legend rule. Restricted admin/organization forms
and dark variants were not newly visually checked. No backend or appliance changes.

## Repository metadata sidebar

The repository code landing override now places existing description, website,
topics and their native editor, counts/size and conditional language statistics
in a right sidebar. Metadata is rendered once with its existing permission gates;
file, directory and blame views keep their previous layout. Below 1000px the
metadata stacks above the code. The main toolbar no longer has an enclosing card.
No release/contributor data fetch or backend change was added.

All focused Forgejo source checks passed, including upstream body recovery after
removing the explicit layout changes. Reloaded templates only in the existing
local preview. Inspected desktop/mobile captures (`capture-br6UQ5`,
`capture-mLOcnf` under `.artifacts/screenshots/`); browser measurements confirmed
1440px and 390px document widths, one sidebar/topics/summary instance, and no
sidebar on the README file view. The screenshot fixture lacks topic-admin rights;
interactive topic editing and populated language statistics were not exercised.
No appliance deployment or native acceptance.

## Soda robot avatars — source implementation

Original `soda-robot-v1` artwork now has 44 modular SVG variants and an eight-by-four
background/accent palette in one embedded DiceBear JSON definition. The pinned Go
renderer serves public GET/HEAD `/-/soda/avatars/v1/{hash}` with bounded inputs,
deterministic ETags and no session, identity/database lookup or outbound fetch.
The development-only catalog uses the same renderer and is not appliance payload.

Caddy source routes only `/-/soda/avatars/*` on the Forgejo origin to the existing
backend, dropping Cookie/Authorization for those requests. First activation derives
the supported provider URL from `forgejo_url`; native database-backed avatar
settings and explicit offline mode remain operator-owned. Uploaded photos/files
are preserved. This does not implement the broader Sodaspaces origin/session work.
New bundles include and require the exact avatar dependency notices. See
[behavior, configuration and restoration](avatars.md).

Local checks on 2026-09-08: full `go test -mod=readonly ./...` passed with Go 1.26.7
on macOS arm64; avatar/web race suites passed. The backend binary built locally
with the same toolchain and `go mod verify` passed. All 34 Python build fixtures passed,
including real Caddy 2.10.2 routing against test-owned loopback upstreams, mocked
first activation and actual notice collection. Caddy was downloaded into ignored
tooling and verified against its published SHA-512 checksum, not installed.
Some unchanged Go results were cached. Initial broader runs failed on macOS's
symlinked temp path and BSD `cp`; using a real workspace TMPDIR and the already
installed GNU coreutils resolved them without changing runner/provisioning code.
The first Caddy checksum comparison mistakenly used SHA-256; the correct SHA-512
comparison passed before the binary was executed. Earlier failed records remain.

Inspected all parts, all 32 palette pairs and the generated 100-robot grids.
Chrome checked all 100 images at 24/32/64/128px across light/dark and square/circular
modes, with no missing images or external resource requests. Six production HTTP
images also matched reference pixels under the restrictive response CSP, with no
browser errors or external requests. The first pixel comparison used different
screen positions; the corrected check uses the same position to avoid SVG
antialiasing differences. Final preview and
captures: `.artifacts/avatars/preview-606454540/`; check logs:
`.artifacts/avatars/checks/`. These are artwork/local-test evidence, not Forgejo
screenshots or native installed acceptance.

No live Forgejo settings, avatar uploads/deletions, retained fixture data, services,
appliance routing or installed artifacts were changed by this avatar work. Native
profile/list/discussion and upload/delete smoke checks, local proxy/backend rehearsal
and appliance rollout remain pending their target-specific authorization. The
direct-port local Forgejo preview cannot exercise this route through a template
reload alone.

## Tighter template spacing

Reduced larger margins, padding and layout gaps across 42 Forgejo presentation
stylesheets. Shared panels use an 18px desktop inset (16px narrow), list rows use
16px vertical padding, and page intros use a 192px minimum with 224px artwork.
Typography, control minimum heights and native workflow markup remain unchanged.
Changed stylesheet URLs are versioned in the header hook.

Focused Forgejo source checks passed with local Go and dependency resolution
disabled (`go test -mod=readonly ./scripts -run TestForgejo -count=1`); whitespace
checks passed. Inspected local candidate-CSS captures of repository exploration,
repository milestones and profile settings at 1440×1000 and 390×844. Captures:
`.artifacts/screenshots/capture-rkXrOj/` and `capture-LQW7zh/`; desktop baseline:
`capture-PA9rin/`. Settings mobile autofocus scrolls to the form. Repository
milestone cards already had flush content in the baseline; that native styling
is unchanged. Other page variants and dark mode were not newly visually checked.
No service reload, deployment or native acceptance was performed.

## Page illustration goal

The accumulated Forgejo-focused scripts suite passes (`go test ./scripts -run TestForgejo -count=1`) after correcting three stale parity normalizers for intentional artwork suppression. All 34 literal illustration references resolve to local assets. Native restricted-page rendering remains outstanding; neither check establishes that visual evidence.

Repository Actions no-workflows state now selects a distinct transparent workflow-tile illustration; native permission-specific guidance is preserved. Populated/filtered run lists, dispatch and log viewer remain undecorated. Focused body-parity/shared-presentation tests passed. The attempted native Actions capture returned 404; artwork rendering remains unverified.

Public registration now has a distinct transparent welcome-folder illustration, gated to enabled standalone registration. Exact prompt and original output are retained. Focused authentication/shared-presentation tests passed. Native guest desktop/mobile screenshots confirm disabled registration excludes the image; enabled registration rendering remains unverified because the local instance disables registration.

All current administrator layout callers are source-assessed. Admin artwork is now opt-in: account creation selects its distinct illustration; operational/configuration pages receive none. Redundant suppression flags were removed. Native admin verification remains pending.

Administrator account creation now selects a distinct transparent identity-card illustration through an explicit presentation input. The generated asset and exact prompt are retained; focused administrator parity/presentation tests passed. Native admin capture remains pending.

Wiki welcome now uses a distinct transparent reference-book illustration, preserving native text and the writer/mirror action gate. Repository content source-parity/gate tests passed; native read-only welcome was captured at desktop/mobile widths. Repository project wrappers and shared callers are assessed in the checklist.

Personal/organization project creation now uses a distinct planning-board illustration; editing excludes it. Personal creation was visually checked at desktop and mobile widths, and focused context/presentation tests passed. Organization and edit-state captures remain pending; checklist records list/board no-image decisions and the remaining repository callers.

Team creation now has a distinct transparent member-card illustration, gated to the creation state. Editing permissions and invitation acceptance retain focused native content. Native markup parity/parse and shared presentation tests passed; organization screenshots remain pending an accessible existing organization. The checklist advances to organization projects.

The [per-page checklist](page-illustration-checklist.md) inventories 206 current
overrides/helpers and tracks shared-template page variants separately. Migration,
fork, 404, discussion subscriptions and watched repositories now have distinct
illustrations. Native forms, state, permissions and meaningful status text remain
upstream-owned. Migration progress and 413 retain concise native diagnostics
without additional decorative art. Public contributor profile tabs were also
assessed without extra art: native identity, authored content and activity visuals
take precedence, supported by desktop/mobile captures. The personal package
registry now has a compact wrapping illustration, with focused package tests and
native desktop/mobile empty-state captures checked. Organization package registry
has a distinct shared-shelf illustration and passing focused package tests;
native verification is pending because the local public organization inventory
is empty. Package version lists and the common detail shell were assessed without
added art to prioritize release selection, installation content and metadata;
these are source decisions, not populated-page runtime evidence. Package settings and cleanup forms were assessed; six upstream settings callers
were added explicitly to the inventory, with personal landing/add-rule desktop
captures inspected. Personal registry settings now has its own maintenance illustration, scoped by an
explicit landing-template artwork input; desktop/mobile captures and add-cleanup
isolation were checked, and focused package/shared-boundary tests passed.
Organization registry settings was assessed without a second decorative header;
its native identity and direct cleanup/Cargo controls take precedence. Other
organization settings routes are now explicitly queued. Personal webhooks now have a connection illustration on the list page only;
desktop/mobile and new-form isolation captures were inspected, with focused
webhook/shared-boundary tests passing. Personal organization memberships now has a distinct card scene, with native
desktop/mobile empty-state captures checked and stock membership content verified
unchanged after the layout call. Personal repository settings was assessed without artwork to prioritize its
repository/directory inventory and permission-dependent confirmations; native
empty-state desktop capture was inspected. Personal and organization blocked-user pages were assessed without decorative
art; personal empty-state capture was inspected. Conditional personal Actions
and storage routes are explicitly queued. Existing profile artwork was retained and verified on desktop/mobile. The
screenshot helper now has an optional `--scroll-top` flag, exercised to inspect
headers after native form autofocus; default capture behavior is unchanged.
Account artwork was retained and verified on desktop/mobile, with native forms
left untouched. Appearance swatch artwork was also retained and verified on desktop/mobile,
without changing saved preferences. Security landing artwork was verified on desktop/mobile. Enrollment/re-enrollment
now explicitly suppress decorative art to prioritize the QR/passcode flow; native
enrollment capture remains unperformed. Focused shared-presentation tests passed.
Keys artwork was retained and verified on desktop/mobile; native SSH, GPG and
principal subpanels were source-reviewed without added decoration. Applications landing artwork was retained and verified on desktop/mobile;
OAuth editing and token creation now suppress inherited decoration; token
creation final desktop/mobile rendering was checked after restoring its existing presentation classes and focused shared-presentation tests
passed. Native OAuth edit verification remains pending. Shared OAuth list/create/grant
sections were traced to personal, organization and admin callers; they need no
independent artwork, while remaining page owners stay explicitly queued; individual decisions and unexercised variants remain in the checklist.

Each new PNG was visually inspected and verified as transparent RGBA. Focused
onboarding, status, presentation-boundary and notification-preview tests passed
for their respective changes. Local templates were reloaded and actual fixture
light-theme pages were inspected at 1440/390px; 404 covered general/repository
contexts, and subscriptions/watching covered shared-template isolation. A status
stylesheet version bump corrected observed cached sizing. Exact prompts, rejected
outputs, capture paths and per-page limits are in the checklist and linked records.
No fixture/account state or provider resources were changed. Dark appearance,
POST journeys and production deployment are not implied by these captures.

## Source versus installed state

| Area | Current state |
| --- | --- |
| Selected frontend | Stock Forgejo native pages plus delivered **Sodaspaces** repository button/right environment drawer (no new tab) |
| Soda UI source | Explicit stable-ID create/key/join/own-connection controls passed bounded native x86_64 build/export/browser/Copy/SSH at `bdbce8e`. Copied private-v3 migration/paired rollback and separately approved retained cutover passed, with native browser and own-access observations. Both standalone frontends remain removed |
| Minimum management controls | Start/Stop, own saved-key removal and reviewed native key apply/revoke, plus independent complete drawer content, are source implemented. Native lifecycle/SSH rotation and integrated delivery still pending |
| Browser terminal | Native helper passed bounded x86_64 PTY/teardown/SSH-preservation proof. Protected browser transport and independent drawer terminal component/locked renderer packaging are source implemented and locally tested. Template mounting, genuine combined browser proof and deployment remain pending; installed helper unchanged |
| Retained backend | `cmd/soda-dashboard`, Go API/OAuth, schema-v5 SQLite with unchanged grant encryption, real create/join/access integration and restricted helper/project OS |
| Retained operator frontend | Separate Cockpit React/PatternFly Tailnet/Runners, backing native logic/dependencies/tests |
| Installed affected components | Built `bdbce8e` dashboard/strict-config runners CLI, native hooks and namespaced proxy/config on `soda-test`; schema v5 and stock Forgejo 15.0.7. Unchanged helper/default project image/other native components retain prior `8b823db` provenance; old frontends are no longer served |
| Acceptance | Historical bounded **U08** native x86_64 first-product proof accepted (`a12b741`); Sodaspaces steps 5–6 have passed bounded execution and approved cutover. U01 architecture acceptance was withdrawn; no final-product/aarch64 acceptance |

Source removal is **not retained-appliance deployment**. `ee8091a` passed bounded
read-only delivery/browser proof; `bdbce8e` subsequently passed native create/key/join/
Copy/SSH on the separate fresh fixture. Separately approved `soda-test` cutover then
delivered those affected payloads with recorded configuration and schema v5. Root returns to configured Forgejo
home; OAuth can return to a freshly resolved repository under that origin using
single-use stored context and the acting grant, never a caller-supplied URL.
Schema v5 adds internal login cancellation contexts after v4's repository/expected-
user IDs. The historical return-path column remains unused. Both copied and live
v3 → v5 migration preserved original rows/ciphertext before login. Old pending OAuth
must restart; subsequent normal expiry/login/logout changes session/grant rows.
New consent requests read user/repository/organization scopes, not administrator
expansion; actual grants, not requested scope names, govern authority. Redirecting is not native-session
transfer or cross-origin authorization.

Keep [API](dashboard-api.md), [credential migration](dashboard-credentials.md),
[architecture](architecture.md) and [current work](sodaspaces-plan.md) authoritative.
Acting grants/current native ownership from `fed66cb` remain in retained callers;
no setup-token, stale-creator or copied-permission fallback was restored.

## Expanded component audit merged into canonical main

Merged exact branch candidate `919bb97cdc4d3c4db84bda6cfd57e02c870dfd3b`
(`codex/forgejo-expanded-component-audit`) with canonical `85f29a5` using a
non-fast-forward merge. The sole textual conflict was this handoff: both audit and
newer notification implementation/activation evidence were retained. The header
merged automatically and retains the notification stylesheet alongside the new
feature owners. Notification source/tests, Explore overflow and milestone grid
fixes remain byte-for-byte unchanged from pre-merge main. Both parent histories
are preserved; no rebase, cherry-pick or history rewrite.

Merged-tree checks on this development machine:
- `GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off go test -json -count=1 -mod=readonly ./scripts -run TestForgejo`:
  **97 top-level tests passed**, no failures/skips, plus their subtests. There are
  98 matching source declarations; `TestForgejoBrandingMatchesSVGMaster` requires
  the separate `branding` build tag/native renderer and was not selected. The
  audit-only tree has 96 declarations, so its earlier reported count is not used
  as the merged runtime count. The two newer notification tests are included.
- `SODA_FORGEJO_LAYOUT_ORIGIN=http://localhost:3300 node --test tests/forgejo/*.test.mjs`:
  **19 reported tests passed**, no failures/skips, including the eight boundary
  subtests, guest theme, Explore, milestone and newer notification fixture suite.
- `node --check scripts/screenshot.mjs`, CSS registry correspondence (46 files,
  each registered exactly once), and staged/working-tree whitespace checks passed.
  Logs retained under `.artifacts/component-merge-checks/`.

No template reload, authenticated journey, new capture, fixture/provider mutation,
dependency installation, service restart or deployment was performed for this merge.
The preview's cached templates still predate the audit: the former `admin-org.css`
and `workflow-details.css` references may outlive their removed source files until
an authorized reload. Shared source-mounted CSS changes are not a complete native
activation. Boundary/milestone fixtures load the merged registry; native-page tests
still use the existing server templates, and notification tests simulate signed-in
markup/responses. The audit's new native form/profile markers and other server
changes remain source-tested, not newly rendered. Earlier notification activation
is evidence for its pre-merge candidate only. Administrator/owner-only workflows,
populated boards, package cleanup, provider/POST journeys and production staging
retain the audit's documented limits.

## Notification bell quick-view local activation

User authorized activation and testing after `20ad60f`. Reloaded templates only in
`sodaos-local-forgejo` (`forgejo manager reload-templates` returned `Reloaded`).
Real signed-in requests now exercise the native `ctx.Context.FormBool` rendering
branch successfully; no product-code correction was needed for activation.

- Populated native inbox: three real unread entries rendered at 1440, 768, 390 and
  320px. Both bell anchors, panel containment and Escape/focus return passed. The
  native unread count remained **3 before / 3 after**. No entries were opened.
- On `/notifications`, the popup coexisted with exactly one native notification
  div/table. An ordinary `div-only` refresh still returned the full native fragment;
  “View all notifications” navigated to the normal page. Recorded notification
  requests were GET-only and no page errors occurred.
- Existing non-admin screenshot fixture profile exercised the real empty inbox in
  dark mode; zero rows, truthful empty copy, dismissal and **0 before / 0 after**
  unread count passed. Populated light/mobile and empty dark screenshots were read.
- The 320px signed-in navbar already extends 3px beyond the viewport before the
  popup opens (`navbar-left/right` and appearance link); the popup fits and does not
  increase document width. This unrelated navbar issue remains, not a popup pass
  disguised as a whole-page no-overflow claim.
- Focused Go Forgejo tests and all three browser suites (notification fixture,
  Explore overflow, milestone layout) passed again after reload. `git diff --check`
  passed. Evidence/scripts/screenshots are retained privately under
  `.artifacts/local-forgejo/notification-activation-20260908/`.

An isolated browser used the retained local fixture credential privately for normal
login; the existing screenshot profile was reused for empty-state checks. Exploratory
checks initially used an explicit submit-type selector absent from the native login
button and assumed five entries where the real account has three; test assumptions
were corrected, not fixture data. No new fixture, notification-status mutation,
account preference change, service restart or appliance deployment occurred. Actual
pinned rows, live badge changes from new events, account switching/session expiry
and native read-on-navigation remain unexercised; those existing synthetic/source
checks are not relabeled native evidence. Production staging/delivery remains pending.

## Notification bell quick-view source implementation (pre-activation evidence)

The requested [plan](notification-preview-plan.md) is implemented in source using
the existing native notification fragment and bundled HTMX. A signed-in-only footer
hook enhances both stock bell anchors; a presentation query flag selects a compact
list of the five native unread-plus-pinned entries. Dedicated markup avoids full-page
notification IDs/hooks. Native destinations, auth, queries, badge updates and read
behavior remain Forgejo-owned. There is no Soda backend, upstream patch, new library,
status POST or second poller. Missing JavaScript/HTMX/popover support retains native
bell navigation. The panel handles loading/errors/retry, cancellation, response
identity, focus/Escape/outside dismissal and native HTMX redirects. A 10-second
request timeout bounds the loading state. Retry/View all/Pinned/loading/empty/error
copy is custom English pending localization; existing notification/close keys are reused.

Performed locally:
- Offline readonly focused Go `./scripts -run TestForgejo`: passed, including compact
  and ordinary fragment branches, faithful embedded template-context method lookup,
  zero/one/five rows, pinned/content escaping, native subpath links and signed-in hooks.
- `SODA_FORGEJO_LAYOUT_ORIGIN=http://localhost:3300 node --test tests/forgejo/notification-preview.test.mjs`:
  passed against the real native HTMX bundle with browser-only signed-in markup and
  notification-response fixtures. Both bells, long five-row content, 320/390/768/1440px,
  dark/light scheme requests, loading/empty/errors/retry, stale responses, Enter/Escape,
  modified clicks, focus/outside dismissal, GET-only preview traffic, HX-Redirect and
  no-JS/no-HTMX/no-popover native fallback were checked. No authenticated fixture data
  or existing private credentials were accessed. Initial tests exposed a null detail
  on custom abort events; events now supply the native expected element detail.
- Existing Explore overflow, milestone layout and seven guest-theme tests passed.
  `git diff --check` passed. No dependency installation or full native build occurred.

Native authenticated rendering, actual unread/pinned data, badge/full-page coexistence,
real account switching and user visual review remain unverified. No template reload,
service restart, notification mutation or appliance deployment occurred. Existing local
mounts expose asset source, but changed/new templates require an explicitly authorized
reload to activate. Production template/asset staging remains separately pending.

## Notification bell quick-view investigation (historical, before implementation)

The requested [implementation plan](notification-preview-plan.md) is now authored:
prove compact native rendering, enhance both bells using bundled HTMX, validate
interaction/native coexistence, then deliver under separate target/action scope.
Unread plus pinned is the proposed native-matching default; no UI code or runtime
work was performed while writing the plan. Documentation whitespace checks passed.

The [integration guide](forgejo-frontend-integration.md#notification-bell-quick-view-investigation)
records source findings for stock 15.0.7 (`d4de9eb2a87c26b402fdd0259e079957f8cd2b4b`).
JSON notification APIs do not accept the ordinary web session, but the existing
`/notifications?div-only=true` HTML-fragment route does; Forgejo already ships HTMX.
The recommended candidate is a supported compact template branch loaded with native
HTMX, not a Soda Go/API proxy. Native unread results include pinned entries, so
strictly-unread-only semantics need a decision rather than silently filtering a page.
Anonymous local GET checks confirmed API 401, native fragment login redirect and
HTMX-aware 204/HX-Redirect. No existing credentials, authenticated requests, state
mutations, UI implementation, builds/tests, service reloads or deployment occurred.
`git diff --check` passed. Authenticated rendering and candidate interaction checks
remain unperformed; investigation is not implementation acceptance.

## Expanded component audit

Audited the `82379b4` expansion across all 201 template overrides/helpers and
38 linked CSS files. The [audit](forgejo-components-audit.md) and
[composition contract](../appliance/forgejo/README.md#presentation-component-contract)
record the resulting boundaries. Shared controls now preserve native focus/error
states; button-local theme variables give native primary actions one color owner.
Settings table padding is separate from card padding. Ordinary repository/org
width rules exclude fluid canvases; the bounded pull-files canvas is centered.
Repository-context status pages constrain their grid instead of expanding native
navigation beyond mobile width.

Repository, administrator, organization, projects, packages/code search, shared
runner/configuration/quota/webhook/moderation adapters now have explicit owners;
competing old rules and the mixed admin/workflow aggregators are removed. Native
profile-card callers opt in through a component marker. Two nested principal
settings forms opt in explicitly, preserving compact row/dialog/search forms.
The migrating wrapper derives guest state from native `.IsSigned`. The unused
`finalize_openid` override is removed. There are still 201 template files (one
removed, one native OAuth-list override added), four Soda partials and 45 CSS
files; all CSS files are registered exactly once. No Lit dependency was added.

Executed locally for this audit:

- Offline readonly `GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off go test -count=1
  -mod=readonly ./scripts -run TestForgejo` passed, covering 96 top-level tests
  plus their native composition/hash/security cases. Earlier runs exposed stale
  CSS-owner/cache-version assertions; those were corrected to the new contracts.
- The combined guest-theme, Explore overflow, milestone layout and component
  boundary JavaScript run passed all 18 reported tests. The new boundary suite
  covers eight cases, including light/dark focus/errors and native button states,
  table padding, fluid widths, narrow error pages, compact-form isolation, profile
  reuse and horizontally reachable package table columns. The error-border,
  table-padding, fluid-width and status-grid tests reproduced pre-fix failures.
  The boundary suite passed again after centering the bounded pull-files canvas.
- Read-only stock 15.0.7 embedded/source inspection retained native hooks and
  confirmed the unreachable OpenID template and anonymous migration route. Exact
  upstream commit: `d4de9eb2a87c26b402fdd0259e079957f8cd2b4b`.
- Captured and inspected 40 candidate CSS screenshots using
  `scripts/screenshot.mjs --local-css`, at 390px and 1654px, plus ten initial
  mobile baseline captures. Candidate captures cover account settings, public
  repository/list/release/wiki/project/activity/profile/package views, migration
  selection, pull files, branches/commits, error pages and guest sign-in/disabled
  registration/recovery notices. Evidence is retained under ignored
  `.artifacts/screenshots/audit-candidate-*`, `audit-final-*` and
  `audit-pull-files-centered/`. The latter recapture verifies the final wide canvas.
- Screenshot helper syntax, CSS registry/file correspondence and
  `git diff --check` passed. Existing development dependencies were reused.

The preview binds `/Users/vince/Projects/sodaos`, not this audit worktree. The
capture option substitutes only candidate Soda CSS in the isolated browser;
native server templates/scripts remain unchanged. New form/profile class markers,
the migration guest flag and removed unused override have source/caller-test
evidence only and await applying/reloading the templates. Missing quota/Actions
routes produced native errors, and global code search redirected to Explore;
those captures are not evidence of those workflows. The non-admin fixture cannot
exercise administrator/owner-only settings. Populated project boards, package
cleanup, provider authentication, native POST/error responses and appliance
staging/deployment were not newly exercised. No fixtures, account preferences,
credentials, service lifecycle or provider resources were changed.

## Expanded native Forgejo branding

Broad supported stock 15.0.7 header/layout overrides now cover repository pages
and settings, account settings, administrator and organization views, and nine
secondary authentication wrappers. Three coordinated task teams added 183 template
override files from the `a81bfae` baseline, including 141 during the final authoring
sprint; the source now contains 201 overrides/helpers. The added detailed families
include code/edit/diff/history, issues/pulls/milestones, releases/wiki/projects,
Actions/runners/webhooks, storage and access lists, profiles/packages/imports,
administrator monitoring, account/security, organization/team, federated auth,
and setup pages. These are template-file counts, including shared partials, not
independently exercised workflows. The shared native leaf forms/lists/scripts,
permissions and handlers remain upstream-owned. The page-marker adapter reaches
whole native pages from their shared header/helper; it is not a widget root.
Common settings sidebars/cards have one CSS owner. Principal native forms use a
positive structural adapter; nested dialog/table/row-action and settings search
forms keep native sizing. Guest theme routes share one presentation gate.
Six new settings illustrations from the separate art task are mapped by native
page flags, preserving its asset/provenance commits and earlier artwork.

Local source tests passed with the existing offline Go toolchain and readonly
modules (`go test -count=1 -mod=readonly ./scripts -run TestForgejo`). These check
native template composition, permission seams, exact-stock recovery hashes,
escaping, theme placement, artwork selection and CSS boundaries. Stock preview
`reload-templates` succeeded. Initial browser checks covered all 11 account
sidebar pages at desktop/390px, all 16 administrator sidebar pages at 390px,
and 12 repository sections at desktop/390px. One narrow native stacktrace overflow
and clipped-popup risks were found and corrected. After combining all three teams,
the full focused Go suite and all seven guest-theme JavaScript tests passed.
Final browser checks exercised 27 distinct pages at 390px and 1654px, plus all ten
native migration-provider forms at 390px, with no page-level horizontal overflow
after correcting the direct system-notices table. The native file editor mounted
CodeMirror and retained its commit form; a desktop release page was visually
checked. Browser viewport overrides were reset. The owner code-search URL
redirected to the profile under the existing configuration, so that route remains
source-tested only. Registration/recovery remain truthfully disabled. Setup,
MFA, activation, consent, POSTs, populated packages/organization teams/project
boards, Actions dispatch and fullscreen logs were not executed or fabricated.
Guest light/dark navigation was checked earlier and returned to light; no native
account preference, fixture, provider, native stage or deployed VM was changed.
All task commits were collected onto `main`; production staging and the Sodaspaces
drawer remain separate unfinished integration work.

`scripts/screenshot.mjs` is a small local capture helper: manual `--login` in a
dedicated private profile, then viewport PNGs for supplied URLs. Fixture changes
stay manual. It uses installed Chrome and existing Playwright; this development
checkout links the preinstalled desktop package through ignored `node_modules/`.
Verified two real guest preview captures at 390×844, interactive login-window
open/close, and persistent test-cookie reuse across browser runs with a local
temporary HTTP server (1440×1000 output). No real Forgejo login was submitted and
no Forgejo fixtures, native installation or deployment were changed. Usage is in
[screenshot capture](screenshot-capture.md#quick-local-page-screenshots).

## Explore tab overflow correction

The Explore tab wrapper now has a bounded 420px width (capped by the existing
100% maximum), with the native overflow-menu filling it rather than sizing to
visible children. Tabs align to the start so Forgejo's trailing overflow-button
reservation is not consumed by centering. This removes the ResizeObserver feedback
loop that repeatedly moved Organizations into/out of the popup. Native navigation,
visibility, overflow logic and keyboard handling remain unchanged; no custom JS.
The shared toolbar stylesheet cache version is bumped.

Read-only local Chrome checks on all three real anonymous Explore routes passed
at 1440, 768, 700, 390 and 320px, then back at 1440px. The new opt-in
`tests/forgejo/explore-overflow.test.mjs` waits for native initialization/fonts and
checks no child reparenting over 700ms after settling, no page overflow, all three
wide-screen links, narrow-screen popup visibility/destination and Escape dismissal.
With pre-fix CSS intercepted in the isolated browser, the same observation found
14 child mutations in 700ms; fixed pages had zero. An initial test attempted Escape
before the popup's deferred focus; focusing the menu item first corrected that test
race. Run with `SODA_FORGEJO_LAYOUT_ORIGIN=http://localhost:3300 node --test tests/forgejo/explore-overflow.test.mjs`.
The milestone layout regression and offline readonly focused Go Forgejo suite also
passed; `git diff --check` passed. No login, data writes, service/template reload,
new dependency or deployment occurred. Local live-mounted CSS can be hard-refreshed;
the header cache-version change awaits the next template reload. Authenticated-only
extra tabs and translated labels were not newly exercised.

## Shared Forgejo presentation components

The local stock 15.0.7 preview uses small Go template partials for page intros,
empty content and the guest theme button. The component audit separates page
shell/tokens, intro, toolbar/native controls, forms, list/pagination, empty feedback
and guest theme/shell ownership. Page files retain only their specific metadata
and layout. Explore now delegates navigation, visibility and overflow to native
`explore/navbar`; context-switcher CSS has an explicit wrapper. All list callers
use the same wrapper contract. Search selectors cannot reach nested dialog buttons;
filled actions use the selected theme's action colors. Login no longer decorates
Forgejo's loader pseudo-element. Guest theme listeners load only on anonymous
routes with a guest toggle.

[Composition contract](../appliance/forgejo/README.md#presentation-component-contract)
and [full audit](forgejo-components-audit.md). `.soda-page` is a full-page shell,
not a root for the future drawer. Home/login keep distinct content layouts; the
native dashboard Vue widget, milestone cards and notification row actions retain
bounded page adapters. The original extraction is recorded in `ce4129c`.

Native forms, permission gates, translations, asset prefixes and notification
replacement hooks remain upstream-owned. Exact embedded 15.0.7 source comparison
found no unexplained behavior divergence in issues, milestones, notifications,
subscriptions or organization creation. The documented historical research mirror
is absent on this machine; the audit used `forgejo embedded view` from the existing
stock preview. No Lit dependency or frontend build was added. The authenticated
drawer and production staging remain pending.

Executed for this audit on the development machine:

- `GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off go test -count=1 -mod=readonly ./scripts -run 'TestForgejo'`
  passed with Go 1.27.1 darwin/arm64. Tests render real Soda partials/callers with
  native seams stubbed, including creation permission branches, guest route gates,
  singleton placement, escaping, subpaths and Explore delegation. This is not a
  locked native toolchain or a full Go/native build check.
- `node --test tests/forgejo/login-theme.test.mjs`: all seven tests passed, now
  including actual head timing before the toggle exists and a no-toggle case.
- Reloaded templates only in existing `sodaos-local-forgejo`. All twelve signed-in
  route variants rendered at 1654px and 390px without horizontal overflow; shared
  intro artwork loaded and search inputs measured 44px. Native Explore overflow
  opened its Organizations menu item and navigated successfully. The syntax dialog
  opened/closed normally; its nested Cancel button matches zero toolbar rules even
  before Forgejo reparents the dialog. The personal context menu stayed within 390px.
- Notification bulk action uses dark action blue with white text; list wrappers and
  twenty visible native rows were verified without submitting actions. Expanded
  native repository initialization/advanced controls fit at 390px; organization
  fieldsets and required input remained intact.
- Guest home/login and all three directories rendered at measured 480px in both
  light/dark modes without overflow. Each page had one toggle (40px; login 44px),
  persisted the choice across navigation and kept native `data-theme`. Login's
  idle submit has no authored `::after`. Unrelated password recovery loaded no guest
  script or theme attribute. Both browser viewport overrides were reset afterward.
- All sixteen linked component/page CSS responses matched source bytes. Changed
  guide file links and `git diff --check` passed.

No fixture records, native account preferences, providers, dependencies, appliance
stage or deployed VM were changed. The guest local choice was restored. Native
notification POST/live replacement, creation POST/server-error and transient
loading/disabled journeys were not newly exercised; source contracts were reviewed.
Signed-in light appearance and organization/team context menus were not exercised
(the local account has no organization context). Template overrides still require
exact-version review and browser checks on upgrades. This is bounded local preview
validation, not production or final-product acceptance. Custom introductory copy
remains English pending the existing i18n work.

## Local New Organization preview

Official `org/create.tmpl` now uses the Soda shell and existing organization
workshop artwork. Native fields, defaults, visibility values, permission checkbox,
error flags and POST action are retained. Desktop cards match the repository form
with 24px padding/gaps and aligned labels. Local reload and Chrome checks covered
desktop, 390px layout without overflow, visibility selection and required name/
40-character constraint. The form was reset afterward; no organization was created.
Submission, server errors and light appearance were not newly exercised. Custom
intro remains English; no appliance deployment.

## Local New Repository preview

Official `repo/create.tmpl` now has the Soda shell, existing repository-folder
artwork and scoped form cards/styles. All native create-helper/basic/template/
initialization/advanced partials, permission gates and form action remain intact.
Local template reload and Chrome inspection verified expanded initialization and
advanced controls, required name/length constraints, asset loading and a 390px
layout without horizontal overflow. No repository was created; submission,
server-side error paths, template selection and light appearance were not newly
exercised. Custom intro is English. No appliance deployment.

## Local Notifications preview

Official stock 15.0.7 notification partial and subscriptions wrapper now use the
Soda shell, dedicated generated inbox artwork, separate consistent toolbar and
rounded list/empty state. Native notification IDs, sequence hooks, forms, data
attributes, status conditions and pagination remain unchanged. Existing fixtures
supply 40 unread notifications; no new data was seeded. Local Chrome verified
populated/read/empty views, mark-read then unread restoration, subscriptions shell,
and a 390px layout without horizontal overflow. Two fixture status checks were
restored to their original states. PNG alpha and template reload were verified;
`git diff --check` passed. Light appearance, pin/bulk actions and watching filters
were not newly exercised. Custom intro remains English; no appliance deployment.

## Local Milestones layout correction

Fixed the dashboard sidebar consuming the row and squeezing milestone cards off
screen. Stock 15.0.7's `.flex-container { display: flex !important }` defeated the
page's grid; the scoped milestone grid now explicitly overrides it. The stylesheet
cache version is bumped. Native filters, milestone data and templates are unchanged.

The new opt-in `tests/forgejo/milestones-layout.test.mjs` reproduced horizontal
overflow before the fix and passed afterward at 1440, 1024, 768, 700 and 390px.
It uses existing local preview public stock CSS, all authored custom styles in load
order and representative milestone markup in isolated headless Chrome—not an
authenticated native-page journey. Run with
`SODA_FORGEJO_LAYOUT_ORIGIN=http://localhost:3300 node --test tests/forgejo/milestones-layout.test.mjs`.
Offline readonly `go test -count=1 -mod=readonly ./scripts -run TestForgejo` and
`git diff --check` also passed. No service reload/restart, fixture mutation or
appliance deployment was performed. CSS is live-mounted in the local preview;
cached pages may need a hard refresh, and the header version takes effect on the
next separately performed template reload.

## Local Milestones preview

The official dashboard milestones override now uses Soda's separate toolbar,
repository filter panel and progress cards, reusing the checklist illustration.
Native milestone data, filtering, rendered content, dates and pagination remain
upstream-owned. Local reload and browser checks covered the populated 8% fixture,
closed empty state, keyword no-match and 390px layout without horizontal overflow.
Deadline/overdue, tracked time, org context, light appearance and pagination were
not newly exercised. No new fixtures or deployment; custom intro is English.

## Local global Issues preview

The official 15.0.7 dashboard Issues template now has Soda styling and new
checklist artwork. Native query/filter/count/context and shared issue-list logic
remain; Pull requests now shares the same layout with its own heading and icons. Local reload succeeded.
Chrome exercised six populated issues, type switching, closed/no-match empty
states, oldest sorting and 390px layout without horizontal overflow. Pull requests
retains its native list partial. Light appearance, org context and pagination were
not newly exercised. No appliance deployment; custom intro copy remains English.

The Pull requests list now shares the Soda list layout and dedicated collaboration
artwork. Local Chrome checks covered reviewed-by filtering, open/closed/merged
fixture rows and review summaries, no-match search and 390px layout without
horizontal overflow. Native review filters, query state and permissions are
unchanged. No new fixtures or deployment; light mode/pagination not newly checked.

## Local signed-in dashboard preview

The personal home feed now has an official Soda dashboard template override,
new generated workbench artwork, responsive feed/sidebar layout and branded
native empty guide. Native account/org navigation, alerts, heatmap, activity
partial/pagination and Vue repository/organization controls remain composed
upstream partials. Shared shell styles match the explorer; Forgejo account theme
state remains authoritative. Custom dashboard copy is English for now.

The local stock 15.0.7 preview reloaded successfully. Chrome verified empty feed,
repository/organization tab switching, light and auto/dark themes, appearance
navigation and 390px layout without horizontal overflow. Account theme was
restored to `forgejo-auto`. Existing Alice authenticated HTTP rendering returned
200 with populated activity and native repo-list markup; its populated layout was
not visually checked. Organization/team contexts, heatmap and feed pagination
were not newly exercised. No new fixture data or appliance deployment.

The dashboard context switcher now uses a compact Soda trigger and rounded
menu with palette colors, monospace caption, active state and keyboard focus.
Its stock markup and context links are unchanged. Local Chrome inspection
verified the open menu, ArrowDown expansion and personal-context navigation;
organization switching was not newly exercised. CSS-only change and local
template reload; no deployment.

## Forgejo illustration consistency

Reviewed all nine page illustrations together and regenerated the repository
explorer, Users and Issues assets using Home/Dashboard as style references.
The replacements align robot proportions, paper materials and scene balance;
canonical logos remain unchanged. PNG alpha and the cream contact sheet were
inspected, then all three pages were visually checked in local Chrome dark mode
after template reload. Asset query versions refresh cached images. No appliance
deployment or backend changes. Prompts and decisions are recorded in
`assets/branding/forgejo/art-consistency-review.md`.

## Local repository explorer preview

The stock Forgejo 15.0.7 local Docker preview now uses a Soda repository-explorer
wrapper and scoped CSS with the approved folder artwork. Native search, list,
filters, sorting, pagination and main navigation remain upstream partials. Guests
share the login/home theme preference; signed-in pages use the same Soda palette
with light/dark selection inherited from the native account color-scheme.
Local browser checks exercised matching and empty searches, alphabetical sorting,
not-archived filtering, light/dark switching and a 480 CSS-pixel narrow layout
without horizontal overflow. The preview now contains 21 public repositories with sample descriptions/topics;
browser checks confirmed 20 rows on page one and one on page two. Language
variants and a fresh signed-in journey were not exercised. This is local
preview evidence only; appliance staging/deployment remains unchanged.

The Users and Organizations directories now share the Soda shell with distinct
headings and separate newly generated papercraft artwork for individual contributors
and a shared organization workshop. Browser inspection confirmed the page-specific
asset references after local template reload. Native user-list privacy/email
conditions, search, sorting and pagination remain upstream-owned. Local browser
checks covered people search, organization sorting/empty state, light/dark
appearance and 480 CSS-pixel layouts without horizontal overflow. No sample
organizations were added, so populated organization rows remain unexercised.

Signed-in explorer parity now includes the Soda logo/palette, themed native account
menus and a shortcut to native appearance settings. Local Chrome checks using the
existing Vince session covered light and auto/dark themes, profile/admin links,
390 CSS-pixel mobile navigation without horizontal overflow, search and page two.
Vince's original `forgejo-auto` preference was restored after verification. Native
navbar permission conditions are unchanged; private-repository authorization and
additional theme families were not newly tested. No appliance deployment occurred.

Branded empty states now cover the three explorer directories through a shared
presentation partial, used only when native result collections are empty. Native
populated lists and visibility/creation authorization are unchanged. Local browser
checks verified repository reset, people search clearing, organization empty
states for guests and the signed-in administrator, dark desktop and light narrow
layout. Empty state copy remains English; no translations or deployments occurred.

## Accepted native evidence

U08 covers the named infra client → isolated `soda-test` journey, **not** a fresh
appliance install, whole-host upgrade, new UI, release or aarch64 proof:

- `f233a4a`: actual existing-container stop/start and guest reboot preservation.
- `c96c108`: exact-image fresh project and different-UID/default-user/PTY/nested
  exec, SQL and Bob's engine denial. The create-time SYS_PTRACE correction was
  exercised; this does not authorize a privileged parent or unrestricted host socket.
- `8b823db`: full native build/seal/aggregate check; backed-up copied populated-v3
  startup rehearsal and affected-component rollout. Go, Cockpit 60 tests, then-
  dashboard 21 tests, 30 build-fixture and nine staging tests passed at that scope.
  Subsequent corrected operator-probe coverage brought build fixtures to 31.
- After rollout: independent operator/Alice/Bob OAuth/navigation/logout, current
  connection authorization/public keys, root Cockpit/PAM and existing-nobody denial,
  direct own-key SSH/PTY/SCP/SFTP, cross-project/sudo boundaries, retained HTTP/SQL,
  separate personal Git agents/remote refs and shared executable identity checks.
  Four environments and declared state survived; probe files were explicit additions.

Earlier lifecycle evidence was reused **only for unchanged mechanisms**, not claimed
as a new `8b823db` reboot. An image-layer-ID equality probe failed; retained content/
mode/owner/link/capability comparisons found only Tea binary content changed while
runtime configuration/dependency content was unchanged. Native Tea version checked.
An existing-output browser observation failed before a new exclusive observation
path was used. Keep these failures, not just successful retries.

The corrected operator script **fails** on missing
`/etc/profile.d/soda-console-welcome.sh`; the earlier apparent success is invalid.
Tailscale was `NeedsLogin`; zero runners/listeners/capacity were observed. These are
not enrollment or provider-job proof. Service/package observations do not certify
interactive console, visual branding or complete operator journeys.

### Evidence locations

These are retained references, not commands to rerun or evidence revalidated by
this documentation cleanup. Private directories/files remain restricted.

| Location | Retained purpose |
| --- | --- |
| `.artifacts/logs/u08-closure-*` | Build/check exit records, `rollout-8b823db`, host/byte binding, corrected operator failure, browser/Cockpit/PAM, developer/exec/workload/client reads and four-project/image comparisons |
| `.artifacts/logs/u08-completion-*`, `.artifacts/logs/u08-ptrace-*` | Earlier lifecycle, runtime diagnostics, fresh-project exec/Git/workload evidence and failures |
| `.artifacts/test-vm/u08-8417a90/` | Original user/project inputs, bindings and observations |
| `.artifacts/test-vm/u08-completion-952f3b3/`, `.artifacts/test-vm/u08-completion-c96c108/` | Completion inputs, before/after snapshots, connection/Git/client evidence |
| `.artifacts/test-vm/u08-closure-8b823db/` | Merged-candidate browser observations |
| Guest `/var/lib/soda/u08-completion-8b823db/` | Private consistent DB/config/key/helper/runner/unit/prior-image backups and rehearsal |
| `.artifacts/retained-native-u08-c96c108/` | Earlier artifacts moved intact before the merged build |
| `.artifacts/retained-worktree-builds/` | Retained builds; `retention.json` maps old worktree/log paths |
| `.artifacts/retired-dashboard-88fc21f/` | Ignored React outputs/dependency cache moved out of source, not erased |

Preserve project IDs `p4a1ba7562c740b1419169fb5`, `pa7cfcfd898ce306d6b23836b`,
`ped30b9d6932974b14feb2278`, `p7b41edaf83f10a6fd7e579bf`, all identities, keys,
roots, dirty checkouts, workloads/volumes, private inputs and backups. Old backups
predate later writes and are not lossless rollback. The live VM overlay depends on
its retained base. See [local access and retention](local-testing.md).

## Latest source checks

| Revision | Actually executed; not installed acceptance |
| --- | --- |
| `752079e` React removal | Full Go suite; web/Forgejo/nativebuild races; 31 Python build fixtures; Cockpit types and 60 tests; shell/document/whitespace checks. Logs `.artifacts/research/react-removal-88fc21f/` |
| `9f3baa7` HTMX removal | Full Go suite; web/store/Forgejo races; 31 Python build fixtures; shell/document/whitespace/caller checks. Initial OAuth schema/scope test failures and final passes retained in `.artifacts/research/htmx-removal-752079e/`. Cockpit not retested in that slice |

Go checks used cached 1.26.7, readonly modules and disabled resolution; some results
were cached. No dependencies, images, native stage or VM state were changed in
these removals. HTTP join/failure coverage was adapted to the retained JSON API,
not discarded with the HTML forms. Retired browser scripts remain in Git at their
matching revision and must not be run against the new API-only source.

## Minimal UI source inspection

After `c65aa37`, inspected retained Forgejo 15.0.7 header/footer hooks, native
`<dialog>` styles/browser usage, loading CSS, clipboard delegation and selected
Fomantic components. Recorded the [bounded candidate and control states](forgejo-frontend-integration.md#minimal-button-drawer-and-loading-candidate):
footer-hook markup plus moving our own button into the existing repository action
row, scoped right-aligned dialog CSS, content-only native spinner and native copy
controls. No full header override, new tab or library is needed for this candidate.
The row selector/initialization, browser layout/accessibility and authenticated
connection remained untested at that revision. Its Soda Origin/CSRF checks and
two-origin proxy prevented simply wiring native-page fetches; the namespace was
then only a candidate. See the routing foundation below for subsequent source work.

Only source inspection/documentation/link/whitespace checks ran in this slice.
No production source, payload, dependency, browser/native test or installed state
changed. This is not a working drawer or a claim that the whole integration is
four static files. The existing-account terminal remains a separate follow-up.

## Implementation planning

After `64aad1f`, expanded the existing [Sodaspaces plan](sodaspaces-plan.md), not a
parallel roadmap: same-origin API/OAuth contract, repository-scoped reads, small
read-only hook/drawer delivery, explicit access actions, native-browser validation
and separately approved rehearsal/cutover. The candidate uses Go/JSON plus vanilla
JavaScript and native `<dialog>`, with no added HTMX or frontend build. Fixed
`/-/soda/` routing, single-origin configuration, scoped cookies, repository return
context and stable-ID API changes were **planning only** at that revision. The
routing foundation below records the first implemented subset.

Inspected actual Go/config/setup/staging callers and retained Forgejo 15.0.7 routing
and OAuth application handlers. Native callback editing need not rotate the secret;
the upstream API PATCH does. Recorded that distinction and the template allowlist
gap in the integration guide. Source hashes and documentation checks are retained
in `.artifacts/research/sodaspaces-plan-64aad1f/`. Only source inspection and
Markdown/link/whitespace checks ran; no product tests, build, browser, dependency,
provider, native or private-state actions occurred. The terminal stays separate.

## Test ownership clarification

Following `81bacca`, removed broad upstream-regression requirements from the
Sodaspaces plan and native-validation guide. Tests for removed standalone frontends
and duplicate forge adapters were already deleted in `752079e`/`9f3baa7`; no further
upstream-only test files were identified in the current tracked inventory. Retained
Forgejo client/credential/branding tests exercise Soda-owned code, as do native
project Git/access and Cockpit checks. Keep those and narrowly targeted integration
smoke checks; do not recreate upstream business-logic/conformance suites.

This change is documentation-only: source/test inventory inspection and Markdown
link/whitespace checks, no product test execution or installed-state changes.

## Sodaspaces routing foundation

First implementation slice after `e59f99e`: Go mounts API/login/callback routes
under `/-/soda/`; Caddy's source recipe forwards only that prefix unchanged and
leaves other paths with Forgejo. `forgejo_url` is the sole browser origin;
`public_url` / `--public-url` / `SODA_ORIGIN` are retired. Setup, activation, console
output, strict-loader tests, connection/operator probes and staging assertions
follow that contract. This configuration is incompatible with the installed old
loader/config pairing; no retained configuration or OAuth application was changed.

New host-only Secure/HttpOnly/SameSite=Lax cookies use unique names and the Soda
path. Legacy/native cookies are ignored; duplicate/empty/oversized Soda cookies
fail closed. Callback/session rotation and mutations retain PKCE/state, encrypted
grants, exact-origin/CSRF and refresh/logout checks. Go rejects unprefixed API/auth
aliases and encoded/unclean mounted paths without redirects; creation Location
headers include the prefix. Direct backend root/health remain. In that slice,
schema v3 and keys were unchanged and OAuth still returned only to Forgejo home.

Passed full Go suite, web/config/store/Forgejo races, 31 Python build fixtures,
Python/JavaScript/shell syntax and documentation/whitespace checks. Go used cached 1.26.7,
readonly modules and disabled dependency resolution; some results were cached.
Logs: `.artifacts/research/sodaspaces-routing-e59f99e/`. No native image/stage build,
staged-payload execution, Caddy/browser execution, Cockpit retest, deployment,
restart, provider action or retained-state change. The new staging assertion is
authored, not an executed staged-payload result.

No UI/mutation controls were added in `6deaf9a`. Actor/return handling followed in
the next slice below; the drawer and repository-scoped reads remain pending.

## Actor guards and OAuth context

After `6deaf9a`, protected APIs require `X-Soda-Expected-User-ID`, except optional
bootstrap on `GET /api/session`. Malformed/missing context is 400; mismatch with the
Soda session is 403 before handlers. Session/CSRF/provider/operation authorization
remains separate. The header cannot authenticate a native browser session or select
another actor. The retained connection probe now declares its checked fixture actor.

Login accepts bounded optional repository/expected-user IDs. Schema v4 appends two
default-zero fields to the existing OAuth table; atomic consume returns them with
the verifier. Callback checks the fresh provider subject before changing profiles/
sessions/grants, then uses actual consent and acting-grant repository-by-ID lookup
to reconstruct the native return path plus `#sodaspaces`, or home on unavailable
context. Caller callback IDs/URLs and provider URLs are ignored. Query limits,
duplicate/encoding rejection and no-referrer headers protect the authentication
boundary. Existing encrypted grants and project records are not rewritten.

Passed focused/full Go suites, web/store/Forgejo/config races, 31 Python build
fixtures, JavaScript syntax and documentation/whitespace checks. Go used cached
1.26.7, readonly modules and disabled resolution; some results were cached. The
migration test uses a genuine v3-schema local fixture: missing/wrong keys do not
migrate it; a correct key preserves product rows and encrypted bytes. This is not
copied-private-state rehearsal. Logs/source hashes:
`.artifacts/research/sodaspaces-context-6deaf9a/`. No Caddy/browser/native stage or
image execution, Cockpit retest, deployment, provider mutation or retained-data change.

**Milestone 1 still needs real native-page/proxy/browser proof.** The pending caller
must capture native page identity, compare page/session/fresh-provider IDs and reload
stale native context on resume/BFCache restoration before exposing actions. Native-
only login/logout while Soda's session is unchanged is not detectable by this header;
no atomic cross-system logout is claimed. See the [API caller boundary](dashboard-api.md#native-page-and-stale-tab-boundary).
At that revision the drawer, repository-scoped reads and mutation controls were
not implemented. Subsequent backend repairs are recorded below.

## Security review and fix plan

Additional review at `a9fef51` confirmed two pre-existing gaps, **not fixes or new
regressions in that commit**: an in-flight OAuth callback can create a live Soda
session/grant after Soda logout succeeds; and the trusted-team catalog/new-join
handlers do not enforce repository visibility. The latter is the already-planned
repository-scoping boundary, now explicitly required before UI work enables joins.

Review checks actually run: uncached web/store/config/Forgejo Go race suites passed;
two review-only negative assertions failed, reproducing the gaps through real Soda
handlers/temporary SQLite with fake provider/helper responses. Retained source,
overlay, logs and exit records: `.artifacts/research/sodaspaces-security-a9fef51/`.
No tracked source, installed state, provider or native operation changed in review.
The existing logout-winning persistence evidence covers grant refresh, not callbacks.

The user then requested a fix plan. The existing [step-1 callback/logout plan](sodaspaces-plan.md#oauth-callback-and-logout-fix)
selects a bounded persisted login context and atomic cancellation/finalization;
the [step-2 repository plan](sodaspaces-plan.md#repository-authorization-fix) moves new-join
authorization ahead of drawer wiring while preserving legitimate existing-member
access. At that planning revision both were **unimplemented**. No schema change,
session invalidation or Linux revocation occurred then. That change edited documentation only;
relative-link/anchor and whitespace checks ran, not additional product tests or builds.
Native browser/proxy proof, migration rehearsal and rollout remain separately scoped.

### Callback/logout repair in source

Implemented the first fix: one persisted login context and pending-state hash bind
OAuth claims and rotating Soda sessions. Callback finalization atomically checks
cancellation/expiry/supersession before profile/session/grant writes. Logout carries
its authenticated context across concurrent rotation; either commit ordering leaves
no usable session/grant after successful logout. Superseded callbacks write no
cookies; a delayed successful cookie for a deleted session remains unusable. No new
browser cookie, native identity authority or global revocation was introduced.

Schema v5 backfills independent contexts for existing sessions without rewriting
their token/identity/expiry/grant bytes. Pre-v5 pending OAuth must restart. Local
real-v3/v4 fixtures cover migration preservation and wrong/missing-key refusal.
Uncached web/store/config/Forgejo race suites passed, including deterministic HTTP
logout/callback orderings and store cancellation/supersession/rollback/restart tests.
Logs: `.artifacts/research/security-fixes-1b3e355/auth-race.log`; initial focused pass
also retained. Repository authorization remains next. No deployment, native/browser,
provider or retained-data changes; only local source checks ran.

### Repository authorization repair in source

Replaced the catalog with required `repository_id` lookup through the acting grant:
fresh subject, actual read user/repository consent, current visibility and the unique
Soda association. Response includes current repository context and advisory owner
creation availability, never all projects. Direct-ID detail/member reads authorize
ordinary nonmembers before metadata/native inspection; existing members' own degraded
reads and explicit Soda operator inspection remain. Full member-list elevation still
requires current human/org ownership or the configured operator, not visibility alone.

New joins independently repeat current identity/repository checks against the stored
repository ID before readiness disclosure or fixed account calls. No operator/admin
bypass or setup-token fallback. New accounts use the fresh provider login; existing
joins retain the original login with no reinstallation or provider dependency.
Membership remains contingent on helper success. Stable-ID creation is still pending;
this changes neither existing Linux access nor native Git permissions.

Full uncached Go suite passed. Focused race coverage includes denial/no-grant/consent,
subject mismatch, malformed/oversized/timeout responses, direct-ID disclosure, rename/
transfer, access lost between read/join, native failure, concurrent joins, unsaved
results and own connection during provider failure. The initial repository suite
failure exposed the obsolete catalog assertion; the corrected test now requires a
400 JSON response without repository context. Failure and final logs are retained at
`.artifacts/research/security-fixes-1b3e355/`. A further pool-replacement regression
reproduced lost SQLite FK cascades after connection recycling. Foreign-key and
busy-timeout pragmas now apply to every connection through an escaped file URI;
logout removes session/grant rows even after replacement. The failing reproduction
and subsequent full Go/race passes are retained. All 31 Python build fixtures also
passed. No native/browser/proxy execution or retained-project/provider/deployment changes. Both fixes are source-implemented;
real browser proof and v5 preserved-state rehearsal remain required before rollout.

## Next read-only milestone plan

After `ddb2d4f`, expanded [step 3 of the existing plan](sodaspaces-plan.md#3-deliver-the-read-only-button-and-drawer)
for native template IDs, explicit Soda authentication, repository-scoped reads and
one read-only dialog. The two security fixes stay implemented; this does not redo
them or add a roadmap. Selected stale-tab handling clears data and requires an
explicit native-page reload, preserving unsaved native form edits. Hook/assets and
source tests come first, bounded packaging/conflict fixtures next, then an opt-in
native browser journey. No new endpoint/schema/frontend build is planned.

Verified stock 15.0.7 identity fields/footer ordering and existing custom-asset
cache configuration in source. Recorded the installer target-file conflict gap and
narrow template allowlist/mode/ownership work; neither fix is implemented here.
The real OAuth/Caddy return, cookies, stale tabs and accessible native rendering
remain completion checks before mutation controls, with separately approved fixtures
and no implied retained-target migration, restart or delivery.

Changed documentation only. Source inspection and Markdown link/anchor/whitespace
checks ran; no product tests, builds, dependency resolution, browser/proxy execution,
provider or retained-state actions. Planning source hashes and check logs:
`.artifacts/research/read-only-plan-ddb2d4f/`.
The drawer remains absent; only historical bounded U08 is accepted.

## Read-only caller source

After `05217f7`, added two original custom hooks and scoped CSS/vanilla JavaScript
under `appliance/forgejo/`. Only our button moves into the native action row; browser
`<dialog>` owns the overlay. String IDs bind session/provider/repository reads;
explicit contextual OAuth and Soda-only logout remain distinct. Reads are bounded,
time-limited and generation-guarded. Hidden/blurred/restored pages clear data and
require explicit reload, without discarding native form edits automatically. No
create/key/join/connection/lifecycle/terminal controls or backend/schema changes.

Soda template tests and Node/jsdom tests exercise actual markup/script with fake
provider responses/dialog methods, not native rendering or account provisioning.
Focused Go script tests and DOM checks passed locally using cached tools; logs are
in `.artifacts/research/read-only-05217f7/`. The aggregate source-check entrypoint
now invokes the DOM test; its actual-stage requirement is unchanged. Packaging,
opt-in native journey and real browser/proxy proof still follow. No dependencies,
images, stage, services, provider credentials or retained state changed.

## Read-only packaging source

The production stage now copies the four hook/assets with readable modes despite
a private builder umask. The bundle requires them and admits only the two custom
templates/ancestors, not an arbitrary template tree; source LICENSE/NOTICE are
included alongside retained notices. First-install uses an actual readonly
preflight function to refuse occupied hook/asset targets before writes and adjusts
ownership only for the exact new template paths. It remains a first installer,
not a retained-target upgrade or customization merger.

Full uncached Go tests and all 34 Python build fixtures passed locally, including
real stage logic with synthetic build inputs and readonly installer logic against
temporary filesystems. Logs: `.artifacts/research/read-only-05217f7/`. These fixtures
are not an actual native stage/build/install. The added actual-stage assertions
remain unexecuted; no service, native target, credentials or project data changed.

## Native probe source and final local checks

The opt-in [read-only browser journey](native-validation.md#read-only-sodaspaces-browser-probe)
is now authored, not executed against a provider/browser/proxy. It uses actual
native password/consent forms, two existing users and a public repository; only
explicit authentication writes are permitted. It checks served asset bytes and
conditional revalidation, proxy aliases/encoding, scoped cookies, actor/CSRF denial,
OAuth returns, native-only switching, Soda-only logout, stale tabs, native form
coexistence, keyboard/layout/themes and actual BFCache restoration. An unobserved
BFCache restoration returns incomplete scope, not a synthetic pass. Source inspection
confirmed stock version compatibility suffixes and Playwright's default BFCache
exclusion. Sandbox/TLS protections remain enabled, and failed profiles are retained.

CLI preflight tests cover missing permission, sanitized malformed private input and
refusal to finalize into an occupied run. These use synthetic files and git/transport
doubles, never a browser, provider or real credential. Final delivery review also
extended the installed-byte verifier to the exact new template files, with changed
hook bytes/mode regressions. The read-only caller displays a fresh provider rename
while preserving the original own project login.

Full uncached Go suite and web/store/Forgejo/config/nativebuild race suites passed;
43 DOM tests and all 36 Python build fixtures passed. Node syntax, shell syntax and
documentation/whitespace checks passed. Logs and source hashes are retained in
`.artifacts/research/read-only-05217f7/`. Go used cached 1.26.7, readonly modules and
disabled resolution; Node used the existing pinned runtime/Cockpit jsdom dependency.
No dependency installation, real native stage/image build, staged-payload suite,
Cockpit retest, browser launch, Caddy/provider execution, service/VM action, private
state migration or project change occurred. The new source test invocations of the
installed probe stop at preflight. Real OAuth/proxy/browser and staged/installed
verification remain held for an explicitly approved fixture/target and exact artifacts.
Step-4 mutation controls remain absent; no new native/product acceptance is claimed.

## Isolated local Sodaspaces browser execution

The user explicitly approved a new isolated local Forgejo/Caddy fixture on the
existing development machine, with synthetic users/repository and private TLS.
This authorizes this fixture's initialization and authentication/browser checks,
not installation on the builder or changes to `soda-test`, retained environments,
host trust/network policy, provider resources outside the fixture or cutover.

Evidence/state: `.artifacts/local-sodaspaces-31e73bf/`, retained privately. Built
`31e73bf`'s Go backend with the cached pinned toolchain and readonly/offline modules.
Stock cached Forgejo 15.0.7 and Caddy 2.10.2 run with that backend in a new rootless
shared network/user namespace; only `127.0.0.1:31443` is published. This is an
integration fixture, not installed appliance topology or a full native stage.
NSS tools were downloaded/extracted locally, not installed; only a fresh private
browser home's trust database received the new fixture CA. Native CLI/official
APIs created two synthetic users, one public repository and one confidential OAuth
client. No Soda sessions, grants, environments or memberships were seeded; no host
helper is connected. Native passwords/token output went directly from captured
process memory into restricted secret files, not logs. Runtime service logs are
discarded to avoid recording OAuth URLs or other credentials.

Initial container failures are retained: missing fixture SELinux volume labels,
and stock Caddy's file capability refusing execution with an empty capability
bounding set. Only new fixture paths were relabelled. Caddy retains just
`NET_BIND_SERVICE` in the rootless namespace, with no-new-privileges; no host
capability/security policy was changed. Failed containers were not removed.

**Complete scoped journey passed at probe `dd793a2`**, against unchanged `31e73bf`
backend/UI bytes. `browser-x/sodaspaces-run/result.json` records real BFCache
restoration, both users' real OAuth returns, native-only/Soda-only transitions,
identity mismatch/no premature environment reads, protected logout actor/CSRF
denials, cookie scope, native unsaved-form preservation, keyboard/blur/explicit
reload, Escape/backdrop/focus return and 360/1280-pixel light/dark layout. Raw
proxy path/encoding denials and exact served asset hashes/conditional revalidation
also passed. Only **absent** environment views were native (three observations);
existing/running/stopped/incomplete views remain source/DOM fixtures in this run.
No responses or sessions were faked. This is bounded browser integration evidence,
not milestone/release acceptance or installed CoreOS/aarch64 validation.

`artifact-binding.json` binds the running backend's `/proc/1/exe` hash to its local
Go/VCS build and the four mounted/served UI hashes, with exact cached image IDs.
Read-only inspection of Soda's own fresh schema-v5 database found zero projects,
memberships and development keys. It did not inspect Forgejo's database. Services,
all failed containers, private profiles, inputs and logs remain retained. No real
stage/image build, installed checks, retained-state migration or cutover occurred.

The failed attempts remain under `browser-*`/`logs/`, with their exact revisions.
They exposed probe assumptions rather than requiring product/upstream changes:

- Empty native data regions have no visible box; wait on dialog/ARIA readiness.
- Playwright routes omit redirects; CDP Fetch guards every exercised-page hop.
- Playwright forces pages focused/visible. Stock Chromium now attaches with public
  `connectOverCDP({noDefaults:true})` through a private Unix socket/pipe, retaining
  sandbox/TLS and real focus/visibility/BFCache rather than synthesizing events.
- Native chrome/body focus transitions invalidate Soda; focus return is asynchronous.
- Forgejo's stock logout broadcasts navigate session tabs home and its link action
  posts `/-/fetch-redirect`. Only the exact navigation-only `redirect=/` form is
  allowed. Workers, upstream navigation and native beforeunload stay unmodified.
- BFCache history restoration needs a commit wait, not a fresh load-event wait.

Final local checks passed: full uncached Go suite, 46 Node tests (43 DOM plus three
probe/transport checks), 36 Python build fixtures, Node/bash syntax. Logs are under
`logs/final-*`. Go used cached 1.26.7 with readonly/offline modules. No Cockpit
retest, aggregate native-stage check or additional dependency install was implied.
Transport cleanup/refusal hardening also passed the full native journey at
`2d6a2b3`: `browser-y/sodaspaces-run/result.json` and its exit record. The final
`artifact-binding-final.json` ties that probe to unchanged product bytes; prepared
browser/tool identity and probe source hashes are retained separately. Both full
passes observed real BFCache. Documentation checks covered 55 Markdown files,
240 local links and 33 anchors with no errors; whitespace checks passed.

## Read-only native stage and exported delivery closure

The user gave standing implementation/testing approval for this planned work.
Executed the production `scripts/build-native.sh x86_64` and
`scripts/check-native.sh x86_64` in a fresh clean detached `ee8091a` worktree at
`.artifacts/worktrees/stage-ee8091a/`. Both exited 0. This built all native commands,
Cockpit, Tea, project/dashboard images and the four OCI archives, fetched locked
runner inputs, staged and sealed the actual payload. No placeholder stage was used.

The aggregate verified that exact stage before/after checks: Go suite, 46 Node
tests, Cockpit TypeScript and 60 tests, 36 build fixtures and **11 actual-stage
checks** passed. Separate uncached web/store/Forgejo/config/nativebuild races passed.
The exported bundle at `.artifacts/stage-validation-ee8091a/export/x86_64/` verified
before and after browser use. A mistaken first verifier invocation at the bundle
root failed with exit 127; its log remains. The correct `tools/soda-artifacts`
invocations passed. No source/product correction or dependency-baseline edit was
needed. Actual mutable package resolution is recorded by the build; the new project
image is not granted the older installed runtime's lifecycle/SSH/workload acceptance.

A **new** retained local delivery fixture, `.artifacts/delivery-ee8091a/`, used the
export's readonly templates/assets/branding and Caddy recipe, stock Forgejo/Caddy
images matching exported OCI config IDs, and the **built dashboard image at its
configured UID 2000**, read-only root with no capabilities. Rootless shared network/
user namespaces and only `127.0.0.1:32443` published; no builder appliance install.
Fixture-only configuration, new native users/repository/OAuth client and TLS were
initialized through native CLI/official APIs. No helper is connected. The previous
local fixture and retained `soda-test` installation/projects were untouched.

The exact `ee8091a` product-owned browser journey exited 0 against that delivered
payload: both users' real OAuth/consent/identity/cookie/logout flows, real native
blur/stale-tab and BFCache restoration, native unsaved form preservation, keyboard/
backdrop/focus and responsive light/dark rendering passed. Native `soda-auto` default
and its served stylesheet were separately verified against the export. Three native
environment observations were **absent**, not running/stopped/incomplete proof.
Executed backend `/proc/1/exe` matches the exported binary; runtime image IDs match
exported OCI configs. Read-only Soda database inspection found fresh schema v5 and
zero projects, memberships and development keys; no Forgejo database inspection.

Evidence: `.artifacts/stage-validation-ee8091a/` (build/check/race/export logs and
manifest) and `.artifacts/delivery-ee8091a/` (private native result, artifact/image/UID/
port/theme bindings, inputs and profiles). Both fixtures' services, all worktrees,
outputs and failures remain retained. **Step 3's bounded x86_64 read-only exit is
satisfied; step 4 is next.** This is not first-install/activation, retained-state
migration/cutover, existing-project access, full product/release or aarch64 acceptance.

## Access-action plan revision

Documentation-only review after `afda2d9` reconciles the leading plan and API guide
with the completed read-only build/export/browser evidence and standing testing
approval. Step 4 now explicitly separates action-time identity checks, pending writes,
late/stale results, confirmed versus uncertain outcomes and safe read-only observation
without replay. Existing-member idempotency/degraded access and current-owner/new-join
server authority remain intact. Step 5 extends the existing guarded native journey
with bounded access requests on an isolated helper-backed target; neither retained
browser fixture supplies provisioning or SSH proof. No new roadmap, backend contract,
helper protocol, recovery subsystem or live cutover is implemented by this revision.

Checks for this revision: documentation inspection passed (55 Markdown files,
242 local relative links, 34 Markdown anchors, zero errors); diff-whitespace checks
passed. No product tests, build, native execution, fixture mutation or data cleanup.

## Explicit access actions source implementation

Implemented step 4 after `f940338`, without a new frontend stack, helper protocol,
permission inventory or recovery subsystem:

- Create accepts only canonical decimal-string `repository_id`. The existing
  `visibleRepository` path checks actual user/repository consent, fresh subject and
  stable-ID visibility; the acting human must be the current owner. Reservation
  precedes native provisioning, uniqueness survives concurrent requests and create
  never joins. The unused owner/name Forgejo adapter is removed. The API's existing
  bounded decoder now uses `strictjson` to reject duplicate fields/invalid UTF-8 too.
- The existing four hook/assets add separate Create, development-key summary/public-
  key save, Add me and own SSH connection controls. Public-only input is checked
  before transmission and by the retained Go SSH parser. Save never joins or updates
  existing Linux keys. Existing-member original logins and degraded API access remain.
- Each explicit action rechecks page/session/provider consistency, sends one protected
  POST, disables duplicate dispatch/logout while pending and safely rereads afterwards.
  Close/blur/BFCache cannot cancel or replay native work. Close/backdrop invalidate
  synchronously before the native queued close event; late writes cannot populate a
  reopened/stale drawer. Uncertain create/join stays blocked in that document with
  operator-inspection guidance, not a claim that a missing row proves no native effect.
- Own connection rendering validates association, original login, current running
  address and public host-key/fingerprint fields; stopped/unavailable/stale views
  clear the command and Copy target. Copy delegates through stock 15.0.7's inspected
  `clipboard.js`, not a replacement handler. Address display is not routing proof.

Performed under standing local testing approval: full uncached Go suite; uncached
web/store/Forgejo/config/nativebuild races; **89 Node tests** (86 drawer DOM cases and
three retained probe/transport cases); **36 Python build fixtures**. All passed.
Source checks include owner transfer/admin non-bypass, large/invalid IDs, provider/
consent/actor denials, concurrent reservations and native/persistence failures,
separate actions, private/options/multiple-key refusal, pending/stale/queued-close
results, no replay or false membership and unavailable connection/Copy behavior.
Earlier focused passes and final logs are retained in
`.artifacts/research/access-f940338/`. Documentation/whitespace checks accompany the
change; no new dependencies or baseline versions changed.

This is handler/store/DOM and packaging-fixture evidence, **not** a new real image/
stage/export, native browser/Copy interaction, helper account/key installation, SSH,
project-runtime or aarch64 pass. No Cockpit retest, fixture service restart, helper
connection, retained database/config/OAuth migration, VM/project change or cutover
occurred. Both browser fixtures, `soda-test`, all four retained environments, earlier
worktrees/artifacts and private credentials/evidence remain untouched. Step 4's local
exit is satisfied; step 5 must bind updated delivered UI/backend/helper/project bytes
to real account/key/SSH results before native access is claimed.

## Phase-5 execution started

The user requested phases 5 and 6 after `658f2af`. A fresh native x86_64 KVM fixture
`soda-native-spaces-658f2af` is retained under `.artifacts/access-vm-658f2af/`, with a
new overlay over the preserved CoreOS base, localhost SSH 22230/browser 33443,
private generated host/client keys/password/Ignition inputs and the existing operator
public key. Native extension layering completed; one fixture-only activation reboot
was requested. The first SSH observation incorrectly assumed Python existed before
activating the layered deployment; exit 127/broken-pipe evidence is retained. The
retained `soda-test` hostname, architecture and application/helper service status were
read only; no retained service, configuration, callback, database or project changed.
An initial socket check used the wrong path; the actual `/run/soda/host.sock` is
root:soda 0660 and its socket unit is active.

The existing native browser journey now has an explicit, single-use actor/path/body-
bound access mode; the retained SSH/PTY/SCP/SFTP probe takes declared Sodaspaces
connections instead of historical fixed U08 fixture names. Stock 15.0.7's native
Copy tooltip appends to the document body by default; Soda's Copy button now uses
its supported `data-tooltip-appendto="parent"` attribute so feedback stays inside
the modal top layer. This is source-backed preparation, not a completed native Copy
or access pass. Phase 5 will use a separate fixture-local client network namespace;
no builder routing change or laptop-route proof is implied. Phase-6 preserved-state
rehearsal and live cutover have not begun. Local source checks passed: 89 Node tests,
37 Python build fixtures and focused Go scripts/nativebuild tests, plus documentation
links/anchors and whitespace. They do not establish a native access result.

## Phase-5 bounded native access proof

Candidate **`bdbce8e736b26dfaf81c37b1917386a536ef1fac`** passed production native
x86_64 build/check/export from its clean detached worktree, including full Go,
89 Node tests, Cockpit TypeScript/60 tests, 37 Python fixtures and 11 actual-stage
checks. Separate uncached web/store/Forgejo/config/nativebuild races passed. The
transferred verifier checksum and bundle inventory were checked before first
installation on the new CoreOS fixture; `verify-installed` subsequently passed.

The new fixture completed native layering/activation reboot, first installation,
native Forgejo setup, production `soda-setup` and `soda-activate` with private TLS.
The old initialization recipe wrongly expected a redirect: stock 15.0.7's
`InstallDone` returns HTTP 200 after committing installation. The failure is retained;
native CLI inspection confirmed only Alice existed, then a separately guarded
continuation created Bob/OAuth/repository without replaying installation or replacing
state. Public host-key inspection first omitted the production `soda-` container
prefix; no container was changed. A diagnostic image comparison initially failed on
Podman's omitted `sha256:` prefix; exact digest comparisons then passed. All failed
observations remain, not relabelled as native defects.

The exact candidate's real sandboxed/TLS-trusted browser access journey passed:
real BFCache/account/cookie/logout/stale/native-form checks, nonowner create denial,
one owner-created environment **`p4a530c394bcd53e563d3076d`**, separate Alice/Bob
public-key saves and real helper-backed joins, own connections, native Copy success
feedback and actual clipboard paste for both users. No Soda sessions/grants or
membership rows were seeded and no native response was substituted. Observed states
were absent and running; stopped/incomplete/uncertain-result branches retain their
source/DOM evidence, not invented native observations.

The separately named `soda-phase5-client` used a normal bridge network namespace
inside the fixture, read-only root, no capabilities and no-new-privileges. Both users
passed direct **10.90.0.2** SSH, interactive PTY, bidirectional SCP/SFTP, project-root
UID-map separation and expected owner/nonowner sudo behavior; Alice's key was denied
for Bob's login with a real public-key denial, not transport failure. Host trust came
from independently read public project host-key bytes, compared with browser output;
private client keys were never uploaded to Soda. All probe directories remain.
This proves that fixture-local client path, **not builder/laptop/Tailnet routing**.

`verify-installed`, runtime image IDs, the backend's actual `/proc/1/exe` and the new
project's image/labels match the export. The native schema-v5 Soda DB has one project,
two memberships and two development keys, with integrity checked. Source under
`internal/host/`, `cmd/soda-host/`, `project-os/`, `internal/runners/` and
`cmd/soda-runners/` is unchanged from installed `8b823db`; compiled provenance and
mutable image package resolution remain distinct from source equality.

Evidence: `.artifacts/stage-validation-bdbce8e/`, retained worktree
`.artifacts/worktrees/stage-bdbce8e/`, and `.artifacts/access-vm-658f2af/` (browser-a
result, exact artifact binding and copied client results). The VM/overlay/base,
project/accounts/keys, services, exited client/conversion containers, private inputs,
profiles and failures remain retained. Phase 5's bounded x86_64 access exit passed;
this is not whole-product/operator/workload/lifecycle/aarch64 acceptance.

A follow-up audit found five OAuth-state query lines in the fresh fixture's default
Forgejo router journal; no code/token parameter lines were observed by that bounded
audit. No raw journal or parameter values were emitted. Supported native logging
configuration now disables the query-bearing router logger and retains console
method/escaped-path/status access records. Native effective settings and a repeated
real read-only/OAuth/BFCache journey (`browser-c`) passed, with query-free OAuth access
records and zero audited credential/state query lines afterward. General service/error
logging remains native. Earlier journals and failed recipe diagnostics are preserved.
The default is also authored in `appliance/config/forgejo.env`, with a focused template
check; this is an explicit configuration follow-up to the built `bdbce8e` images,
not a claim that a later source revision was rebuilt or installed wholesale.

## Phase-6 preserved-state rehearsal

Evidence is retained in `.artifacts/phase6-658f2af/`. At rehearsal, `soda-test` was schema v3
with 3 profiles, 2 keys, 4 projects, 7 memberships, 10 sessions and 9 encrypted grants;
all four project roots were running and exact Soda hooks/assets absent. Existing
browser tunnels and trusted Forgejo TLS work. Only its original Soda service was
stopped for a SQLite backup plus matching private config/key/artifact capture and
resumed unchanged. The resume recipe initially checked the wrong mutable `:dev` tag;
inspection established the actual image-pinned Quadlet, which was resumed and verified
against the original image/config and added to the backup. No live schema, callback,
configuration, helper, project, default project image or runner-service change occurred.

The consistent private set is `/var/lib/soda-sodaspaces-bdbce8e/backup` on `soda-test`
and `.artifacts/phase6-658f2af/backup/` on the builder. Copies on the fresh fixture
(`/var/lib/soda-phase6-rehearsal/`) ran the actual prior and `bdbce8e` images with
network=none, no helper mount, UID2000, no capabilities, read-only root and explicit
copy-only writable data. The first negative recipe expected an unsanitized key error;
production correctly emitted its sanitized startup-stage message. That attempt is
retained; separate reviewed copies then passed all eight cases:

- legacy config and missing/wrong key refusal, leaving copied v3 bytes/rows unchanged;
- successful v3 → v5 migration preserving every original column/row and encrypted
  grant/key-check byte, with integrity/FKs and all migrated session contexts checked;
- healthy empty v5, future-version refusal, prior-image refusal of migrated v5;
- healthy paired prior-image/config/key/v3-copy rollback, preserving all original data.

This is actual copied private-state/native-image evidence, not a live rollback or
permission to restore an old backup after later writes. Only exact run-owned rehearsal
containers were stopped; all copies, failed/exited containers and original roots remain.

Official acting-owner API reads confirmed retained OAuth application 4, its unchanged
client and sole prior `https://localhost:24443/oauth/callback`; the planned callback is
`https://localhost:24444/-/soda/oauth/callback`. No application PATCH or Forgejo DB access
occurred. Retained `/u08-alice-8417/shared-alice` is private and Issues-enabled; it must
not be made public to fit a probe. The declared read-only private-repository probe
variant requires native anonymous 404/no Soda reads and then the normal authenticated
journey, never environment/key writes. That variant passed as probe revision
`0992f20ab295c1199ba58ccf34ac377012d67b9c` against the fresh fixture's built `bdbce8e`
images plus tested query-free logging configuration (`browser-d`). A new synthetic
private repository with native Alice ownership/Bob read access was used; no existing
repository visibility changed and no new Soda environment was created. Its three
absent observations and real BFCache are not retained-target running-view evidence.
Official reads confirm retained Alice ownership/Bob write access already exists.

Final retained observations match every preflight field, all four container/image/
running-state identities and every original Soda table row against the backup at that
observation. Key fingerprint and original-login shapes fit the drawer contract. The
fresh fixture still passes installed/runtime artifact binding with exactly one project,
two memberships and two development keys. No later backup should be assumed current.

Follow-up local checks passed: uncached full Go tests, **91 Node tests**, **37 Python
build fixtures**, JS syntax, whitespace and documentation links/anchors. Evidence is
`.artifacts/research/phase6-bdbce8e/`. No new compiled backend/helper/UI payload or
whole native bundle was built after `bdbce8e`; the follow-up changes are logging
configuration, probe/tests and documentation.

**The user subsequently approved live cutover.** The
[affected-component procedure](installation.md#retained-sodaspaces-cutover) covers
fresh-at-cutover backups, owner-native callback editing, image-pinned backend,
strict-config `soda-runners` CLI, proxy/namespace/hooks and query-free native logging.
Unchanged helper, project roots/default image and runner services are not upgrade
targets. See the executed cutover below.

## Approved retained cutover

The user explicitly approved the documented affected-component cutover after the
rehearsal. Evidence is under `.artifacts/cutover-c007eb6/`; the fresh private backup
on `soda-test` is `/var/lib/soda-cutover-c007eb6/backup`. The old rehearsal backup was
not reused as current state. The actual image-pinned unit, prior image, consistent
SQLite data, credentials/key/config, proxy, native configuration and file metadata
were preserved. An initial SCP transfer could not preserve two relative bundle
symlinks; that partial tree remains untouched. A separately named tar transfer passed
the production bundle verifier before deployment.

Built `bdbce8e` dashboard image/binary and strict-config runners CLI, exact four native
hooks/assets, namespaced proxy/config and reviewed query-free logging were delivered.
The actual owner changed only app 4's callback through native Applications settings;
client identity/name/confidential setting and credential-file bytes were preserved.
No API PATCH, secret generation or Forgejo DB access occurred. Only affected services
were restarted; no helper/project/default-image/runner-service or routing change.

Live schema v3 → v5 passed integrity/FK checks and preserved every original column/row
and grant/key-check ciphertext **before browser login**, with migrated contexts checked.
The running backend executable/image match the export. Native runner `list` succeeded.
The retained host lacked the selected static-cache setting (native default six hours);
asset validation caught it. `STATIC_CACHE_TIME=0` was then applied, Forgejo restarted
and actual conditional reads returned 304. Earlier input/cache/startup-preflight
failures remain, not overwritten. A restricted CA copy was used without changing the
original CA file or global trust.

The real private-repository browser journey passed at probe `c007eb6` (`browser-d`),
with three running views and genuine BFCache plus normal authentication/identity/
logout/form guards. No environments, keys or memberships were written by the probe.
The follow-up probe `44819462d52de86fe8d40e3b278ec26ce48b942a` also passed on the
delivered target (`browser-e`), recording both users' usable displayed own commands/
fingerprints, three running views and genuine BFCache. Those displayed values match
original membership logins and independent operator public-host-key observations.
Native `ssh-keygen` independently matched fingerprints for all four roots.

All **seven existing memberships across four projects** then passed direct own-key
SSH identity and PTY checks from `linux-infra.dimensionlab.net`, using the unchanged
`tun8417` route via `169.254.84.2`. No keys/accounts were created or updated, no project
files were written by the probe commands, and no project lifecycle/routing action was
performed. This is infra reachability, not laptop/Tailnet proof. Native private Git
HTTP advertisement at the unchanged Forgejo origin passed using the real native Alice
credential, without retaining its body or exposing the credential.

Final checks preserved all original profile/key/project/membership/key-check rows,
four container/image/running-state identities, helper bytes, native Forgejo/proxy units
and credential/TLS bytes. Affected files and running backend match the export. The old
Soda listener is absent; a preserved old SSH forward may still bind locally but no
longer serves Soda. Native query-free request/OAuth logging was observed with zero
audited credential/state query lines. Normal authentication/expiry/logout left two
sessions/grants; the pre-login check had preserved all ten sessions/nine grant rows.
Fresh paired backup is also retained at `.artifacts/cutover-c007eb6/backup/`.

The scoped phase-6 exit passed. **92 Node tests, 38 Python build fixtures**, JS/Bash
syntax, documentation and whitespace checks passed. The VM web-tunnel wrapper now
advertises/forwards only native 24444 on future invocation, with a fake-SSH regression;
no running tunnel was changed. No new Go compilation/native build
was needed or performed in this cutover turn. This is an affected-component deployment
of `bdbce8e` artifacts plus recorded configuration, not a wholesale install of the
later probe/document revision. Full product/provider/aarch64 acceptance, console and
laptop routing remain separate. All old/fresh fixtures and failures are retained.

## Browser terminal plan

The next concrete item is planned in the existing
[Sodaspaces plan](sodaspaces-plan.md#next-item-existing-account-browser-terminal), not
another roadmap. Candidate: local terminal renderer, same-origin authenticated
WebSocket and one fixed Unix-helper operation into an existing project account.
Native PTY/account/owned-process teardown proof comes before UI wiring; Podman client
exit is not assumed to terminate container exec. No private-key collection, automatic
join/start, host shell, project-image replacement or durable terminal sessions.

Inspected current API/session/helper/provisioning owners, upstream Podman v5.8.2
exec source, terminal package metadata/types and Go WebSocket/PTY documentation.
Research is retained in `.artifacts/research/terminal-plan-6206578/`; the initial
Podman manual URL failed, then the correct `.md.in` source was retrieved. No native
command, product test, build, dependency installation or deployed-state change ran.
Only planning documentation and link/whitespace checks changed; the terminal remains
unimplemented. Native helper changes and retained rollout need their applicable scope.

## Native terminal boundary source

This records the initial `10321ce` source checkpoint; approved native execution is
recorded in the following section.

Implemented the first source slice of the [terminal plan](sodaspaces-plan.md#next-item-existing-account-browser-terminal):
fixed embedded project-local Python PTY launcher, bounded private Unix WebSocket
operation/client, immutable-container-ID/namespace checks, marker/account validation,
credential dropping, framing/backpressure, heartbeat expiry and owned-shell teardown.
Streaming does not hold the mutation lock or inherit the buffered RPC timeout. Helper
shutdown now cancels and waits for pending/hijacked streams. No public API, drawer
terminal, native service change or project-image modification was made.

Coder/websocket v1.8.15 was genuinely resolved, without other dependency upgrades;
its ISC license text is included in the already-bundled root NOTICE. Local full Go,
focused host/command races and **49 Python build tests** passed. The new process tests
exercise actual local PTYs, resize/Ctrl-C, EOF/final output, silent-peer expiry and
slow-consumer cleanup with an unprivileged clean test shell; credential dropping is
unit-tested, not native project proof. No host accounts were created. The opt-in
`TestInstalledTerminalBoundary` is authored but skipped without private native input;
it uses a temporary root-private helper, not an installed service replacement, and
records only its bounded account/TTY/explicit-close scope. Lost-helper, unrelated
SSH/workload preservation and full browser evidence remain separate required checks.

Evidence: `.artifacts/research/terminal-native-163ccf9/`. At this initial checkpoint no
VM/installed/helper service, retained project, key, membership, routing or provider
state changed. UI wiring was held until the actual native gate; the separately
approved proof below now closes that bounded gate, not browser delivery. Retained
`soda-test` rollout still needs separate approval and a current backup.

## Approved native terminal fixture proof

The user approved temporary root-private helper/PTY proof on the existing isolated
`soda-native-spaces-658f2af`, without installed service replacement/restart, project
lifecycle/account/key or routing changes. Preflight independently confirmed both
native accounts, marker IDs, groups/homes, the existing container/image and unchanged
installed helper/services/product rows. No terminal had launched at this observation.

Actual Podman is **5.8.4**, not the builder's 5.8.2. Native preflight caught two source
assumptions: Go templates require `.ID` inside `json`, and auto-created namespaces
are reported as `private`. Inspected exact upstream source and observed UID/GID maps
`0:1000000:262144`; the candidate now checks private mode plus actual shifted mappings,
not just a create-time mode name. No runtime configuration/capability was changed to
fit the check. Original failed preflight and reviewed inputs remain under
`.artifacts/terminal-vm-10321ce/` and guest `/var/lib/soda-terminal-10321ce/`.

**Bounded native gate passed.** Final native test binary from `fae1696`, with verified
builder/guest SHA256 equality, passed on existing Rocky project
`p4a530c394bcd53e563d3076d`: both original accounts' real/effective/saved UID/GID,
supplementary groups, HOME/cwd, real PTY, resize/Ctrl-C, current sudo boundary and
shared mise/project-Podman profile settings. A mismatched marker/actor was refused.
Explicit close and independent observations proved actual owned login termination.
EOF, a real silent 60-second lease and SIGKILL of only the test-owned temporary helper
also ended the login, foreground job and project-local launcher. Final observation
times were approximately 0.19s, 60.18s and 0.20s respectively—not merely socket-close
or host-CLI exit observations.

Both users' ordinary own-key SSH processes survived throughout (14 paired observations
in final `run-e`), over the existing management SSH direct-TCP forwarding path to
project `10.90.0.2`; no laptop/direct-builder-route claim. The earlier bridge client
was already stopped and was left untouched. Original preflight and `run-a` parser
failures remain; the latter reached a native PTY but did not recognize Bash CSI/CR
output. The probe parser, not the shell/profile, was corrected. Successful earlier
`run-b`–`run-d` observations and all new private inputs/binaries/results remain.

Final verification confirmed unchanged container/image/running identity, installed
helper bytes and affected native service PIDs/start times, product rows and native
marker/accounts/groups, SSH host key and DB integrity. No candidate helper/launcher
remained. No installed service replacement/restart, account/key/lifecycle/provider or
routing change occurred; ordinary shell/sudo bookkeeping was permitted, not claimed
absent. `soda-test` was not contacted. Full browser authorization/transport, logout
races, drawer/renderer/packaging and genuine browser proof are **still unimplemented**;
proceed with plan step 2, not a new architecture or installed rollout.

Final local regressions passed: full uncached Go suite, host/command race tests,
49 Python build tests, documentation links and whitespace checks. The default local
suite skips the explicitly opted-in native probe. Only the native package test
binaries were built/transferred; no whole-appliance build/stage/export, renderer
dependency installation or deployment ran during this proof.

## Protected browser terminal and independent drawer component

Implemented `internal/web/terminal.go`: exact-origin/query/subprotocol/fetch guards,
pre-upgrade session/own-membership checks, bounded first-message actor/repository/CSRF,
fresh acting-provider consent/visibility, original membership login, one pending/live
slot per login-context/project and bounded frame/queue/write/lifetime handling. Native
dispatch and lease renewal serialize with local logout/rotation; pending authorization
is cancelled too. Session reads now expose their existing minimum expiry internally
(no migration). Shutdown closes hijacked streams before DB shutdown. No browser
heartbeat, automatic join/start/reconnect or copied provider authority was added.

The self-contained `sodaspaces-terminal.js`/CSS component supplies explicit Open and
Disconnect, local lazy xterm/fit, bounded Unicode input/output, suppressed OSC clipboard/
link/title actions and full-page stale/reload behavior. It neither discovers nor edits
Forgejo markup. **The other agent owns all template overrides/layout**; the small
[mount/dispose contract](terminal-integration.md) is the only integration surface.
Existing hook templates, navigation and original drawer implementation were untouched.
Host template mounting remains intentionally unwired, not a second standalone UI.

Actual npm archive integrity and per-file hashes pin xterm 6.0.0/fit 0.11.0. Native
build/stage/bundle/first-install source now carries seven new component/distribution/
MIT-notice files, verifies upstream hashes and refuses occupied exact destinations.
This adds no bundler/CDN/runtime download or arbitrary-template adoption. Local
fetches wrote only ignored `.artifacts/browser-terminal/vendor`.

Local full Go, focused web/host/store/command/nativebuild races, **104 Node tests**
(including standalone DOM/renderer doubles) and **51 Python build tests** passed.
New server cases exercise malformed first auth, actor/association/CSRF/provider and
pre-upgrade denials with zero helper calls, original login, duplicate refusal, pending
and active logout/shutdown, OAuth rotation, logout during fresh authority, and bad
controls/browser-heartbeat refusal. DOM cases cover inert mounting, Unicode, scoped
keyboard handling, stale/late events, disposal and bounded renderer backlog. Packaging
tests exercise the real stage/preflight in synthetic temporary trees; they are not a
new real native-stage result. Evidence: `.artifacts/browser-terminal/`.

No VM/installed service, retained project, account/key/provider or routing action ran
in this source turn. No template override was changed, whole-appliance bundle built,
Chromium/native OAuth journey executed or deployment performed. Real combined
browser/proxy/helper lifecycle proof and native candidate delivery remain required;
prior native-only proof is not public endpoint acceptance. Continue the documented
mounting coordination and integrated proof, not a retained rollout or template fork.

## Minimum management controls — source implementation

Added restricted `/lifecycle` and `/access-keys` helper operations, protected public
routes and own saved-key deletion. Lifecycle validates the existing isolated container
and selected project unit before `systemctl enable/disable --now`, then verifies the
same container and actual running/boot-enabled state. **Start enables host-boot start;
Stop disables it and interrupts all project sessions/workloads.** No recreate, direct
Podman stop competing with systemd, desired-state DB copy or automatic repair. Current
project administration requires fresh authority; the configured Soda operator is a
separate permitted authority. Stop invalidates that project's browser terminals.

Own-key preview/apply always requires existing membership and fresh user/repository
consent, with no operator bypass or caller-selected login. Saved removal is own-user
SQL only and explicitly does not change existing access. Applying the reviewed set
checks both saved fingerprints and the exact native file revision; last-key removal
requires explicit confirmation. The fixed embedded Python reuses the existing marker/
account validator, locks the root-owned dedicated key directory, refuses unsafe paths/
files/noncanonical data and atomically replaces only that account's managed file.
It changes no groups, accounts, homes, unrelated files or authenticated sessions. No
project image/file installation or key propagation to other projects was added.
Canonical root edits in the dedicated managed file are visible in the complete preview;
explicit Apply confirms their inclusion/removal, not hidden drift repair. Noncanonical
annotations/options refuse, and edits after preview fail revision checking. Native
failure/partial results remain uncertain, not claimed as rollback or successful revoke.

`mountSodaspaces` in the new independent `sodaspaces-drawer.js`/CSS provides the complete
minimum content: OAuth connect/local logout, explicit create/join, own public keys and
removal/review/apply, confirmed Start/Stop, actual SSH details/Copy/Refresh and the
existing terminal component. It is inert until `.refresh()` and owns only its supplied
mount. **All Forgejo templates/layout remain with the other agent**; none were edited.
Use the [mount contract](terminal-integration.md), not both old/new callers in one drawer.
The historical caller stays unchanged for existing integration/evidence. Packaging
source includes the two new assets and occupied-destination refusal.

No schema migration or new dependency. Local tests exercise Go authorization/operation
boundaries, real atomic file writes in owned temporary directories with mocked root
metadata, and DOM/API/renderer doubles. These are not native lifecycle/key-possession
proof. Native Stop/Start, new/removed-key SSH, merged template/browser/proxy/helper,
whole-candidate native build/stage and installed delivery remain outstanding. No VM,
installed service, retained project/account/key/provider or route action was executed
in this source pass. Evidence and exact local check results are under
`.artifacts/management-72ce126/`: full Go tests and focused web/host/store/nativebuild
races passed; 114 Node tests and 56 Python build tests passed; document links,
installation-shell syntax and whitespace checks passed. The assembled embedded Python
was also exercised locally as an unprivileged refusal (no host account/file changes).
These checks did not run installed/native tests, SSH key possession or a full appliance
build/stage. Destruction remains an explicit unimplemented decision.

## Login design font assets

Downloaded the website's exact Fontsource 5.3.0 Latin WOFF2 selection into
`assets/branding/fonts/`: Fraunces variable 100–900, Barlow 400/600 and IBM Plex
Mono 400/500, all normal style. Added relative-URL font-face CSS, original family
OFL licenses and package/file provenance. Existing `assets/branding/theme/palette.css`
remains the shared color source, unchanged. These are source assets only; no
Forgejo template, running preview, native staging or deployment was changed.

Verified published archive SHA-512 integrity, font signatures, local CSS paths and
file SHA-256 values. No build, font-rendering/browser test or native validation ran.

## Local branded login preview

With user authorization, added `appliance/forgejo/templates/user/auth/signin.tmpl`
and the custom header CSS hook, plus `assets/branding/forgejo/login.css` and the
approved original papercraft PNG. The login shell uses the website's local fonts,
canonical logo and unchanged shared palette. Native `signin_inner`, head/footer
and scripts remain upstream-owned. This is the light login design; responsive CSS
hides the illustration below 900px. The Sodaspaces drawer remains unimplemented.

Recreated only `sodaos-local-forgejo` on Docker Desktop to bind source directories
read-only, retaining `sodaos-local-forgejo_data` and port 3300. An initial mount failed
because nested mountpoint directories were absent beneath a read-only parent;
created those empty local mountpoints and startup succeeded. The existing other
preview and appliance VM were untouched. No appliance stage/install changes.

Checks: login HTML and all sampled CSS/font/palette/logo/image URLs returned 200;
Alice's native form sign-in succeeded; wrong-password submission rendered the
native error inside the new shell. Image alpha data was verified. An exploratory
foreign-Origin rejection assertion failed (HTTP 200, also with cross-site fetch
metadata), so these probes do not establish CSRF protection; no native middleware
was changed. Browser automation was blocked by the user's password-manager panel;
the user inspected the preview and reported it looked good. Automated mobile,
keyboard, provider/passkey, account-link and CAPTCHA browser checks remain unrun.
Source whitespace checks passed. No full build, test suite or native acceptance.

## Login viewport correction

Removed Forgejo's inherited 80px wrapper bottom padding and first-section margin
on the login page. The flex layout now reserves the footer's actual height instead
of assuming a fixed footer size; artwork height and compact spacing adapt to shorter
viewports. Content may still scroll when genuinely taller than the available space.
Bumped the login CSS URL and reloaded templates only in the local preview.

Browser measurements confirmed document height and footer bottom equal viewport
height at 1654×970, 1366×768 and 390×844; mobile width was also exactly 390px.
Restored the browser viewport afterward. Initial measurements used cached CSS;
the versioned stylesheet loaded the correction. Whitespace checks passed.

## Login theme toggle

Added a single borderless sun/moon button at the top right, shared-palette dark
colors and the canonical dark logo. The guest preference follows system appearance
until explicitly selected, persists in origin/subpath-scoped localStorage, syncs
across tabs and tolerates blocked storage. A head script initializes appearance;
Forgejo's native theme attribute and authenticated account preference are unchanged.
No authentication/provider or appliance deployment changes.

Six Node state tests passed (system changes, explicit choice, toggle/persistence,
blocked storage, storage events and invalid/subpath values). Browser checks confirmed
system dark initial appearance, switching to light, correct next-action labels,
persistence after reload and no desktop vertical overflow in dark mode. The user's
existing “Soda dashboard” wording edit was preserved separately from this commit.

## Public homepage and texture removal

Removed the experimental paper texture asset and CSS references, restoring the
smooth login button. Added the native `home.tmpl` override and scoped `home.css`
for the public homepage: Soda welcome copy, approved papercraft artwork, sign-in
and repository exploration links, shared guest theme toggle and native footer.
The authenticated dashboard is unchanged. No account/authentication handlers,
provider configuration, appliance staging or deployed VM were changed.

Browser checks covered light/dark desktop appearance, shared theme on navigation
to login, native repository-explore and login destinations, and 390px mobile layout
with no horizontal overflow. Desktop homepage height matched the 970px viewport.
Verified login's computed background contains only its gradient, no texture.
Native public HTML/assets served successfully; whitespace checks passed. No full
build or native validation ran. Source is live-mounted only in the local preview.

## Remaining work and permission boundary

The remaining-work plan now has an explicit [minimum user-controls contract](sodaspaces-plan.md#minimum-end-to-end-user-controls),
not just an engineering task list. At the planning checkpoint saved-key handling was
add/list plus join-time installation; the source implementation above now adds removal
and later explicit project apply/revoke, with native validation still outstanding. The user-requested revision selects those bounded own-account actions
alongside Start/Stop, with real SSH verification; automatic synchronization, global
session revocation/offboarding and destructive execution remain outside that scope.
The deferral guide was narrowed accordingly. Template/layout ownership is unchanged.
That earlier planning pass ran documentation/link/whitespace checks only; it changed
no source behavior, dependencies, native target state or execution permissions. The
subsequent source pass and its local tests are recorded above.

- Preserve the implemented security, native-page context and explicit-action
  regressions plus steps 5–6's bounded native delivery/access evidence. Keep read-only
  guard mode separate from explicitly bounded writes; future maintenance needs its
  own exact scope/current backup, not replay of the recorded cutover.
- Finish native template mounting and genuine browser/proxy/helper proof for the
  already implemented terminal source; do not restart its completed native-boundary
  work. The template/layout agent owns the overrides and uses the component contract.
- Follow the single [ordered remaining-work list](sodaspaces-plan.md#remaining-work--ordered):
  terminal integration, minimum access/lifecycle controls (including explicit own-key
  apply/revoke and Start/Stop), the Destroy scope decision, runner settings, operator/
  client gaps, whole-candidate validation and approved delivery. Start/Stop and explicit
  own-key updates now have helper/API/independent content implementations; native
  lifecycle/key-possession validation and template mounting remain, and deletion is
  still deferred. Planning these is not authorization for lifecycle or
  destructive execution.
- Move Soda's local runner capacity/service configuration into operator-only settings
  in the unified native SodaOS/Forgejo interface, as subsequently selected by the
  user. Inspect official administrator extension points and reuse backing logic/tests;
  retain the Cockpit Runners page until a working replacement and coordinated removal.
  Tailnet stays in Cockpit; provider authority and the Soda operator boundary remain
  unchanged. This decision is documentation-only so far, not implementation/deployment.
- Finish full fresh/populated product and independent native aarch64 acceptance;
  scoped x86_64 delivery/browser/SSH results are not final-product acceptance.
- Complete console delivery/interactive proof, Tailnet and both providers' real
  runner journeys, intended-client routes, native branding and package/tool closure.
- Close [support-tool validation gaps](native-support.md#remaining-validation) and
  [actual-artifact licensing/source obligations](licensing.md). Optional media and
  incomplete outside helper ports are not product gates.

Standing implementation/testing approval now covers this planned work; local native
build/stage/export and isolated delivery testing proceeded under it. Preserve all
retained roots, credentials and evidence. It is not an instruction to erase data,
change unrelated provider/host-network resources or silently cut over `soda-test`.
Recorded routes and agents are not promises of liveness.

## Documentation history

The old M/U/P roadmaps, dashboard inventory, 179-group forge audit and detailed
native audit are removed from active documentation, not from Git. Their complete
text and the chronological 2,076-line handoff remain at `9f3baa7`, for example:
`git show 9f3baa7:docs/implementation-status.md`. The native audit's remaining checks
are condensed into the support guide, not declared resolved. Original source/
license findings and all private evidence survive; old milestone labels in tool
arguments/evidence remain valid identifiers, not active roadmap assignments.

This cleanup changes Markdown/links only. Performed documentation link/anchor,
retired-reference and diff-whitespace checks; no build, product test, dependency
resolution, generated provisioning, service/provider/network action or data cleanup.
Checks covered 55 Markdown files, 218 local relative links and 17 Markdown anchors
with no errors; six retired documents have no active references. Logs:
`.artifacts/research/docs-cleanup-9f3baa7/`.

Local PR fixture follow-up (2026-09-08): added seven user-requested PRs through
native APIs within alice/activity-workbench. Browser confirmed 8 open/2 closed,
review summaries and conflict indicator; API confirmed a native draft and
non-mergeable conflicting PR. Added a new generated collaboration image selected
only for Pull requests. No deployment or non-fixture repository changes.

Milestone artwork/fixture follow-up (2026-09-08): dedicated generated steps/flag
illustration now replaces the reused checklist image. Prompt and provenance:
`assets/branding/forgejo/milestones-art-prompt.md`. User-authorized native API writes
added ten milestones and thirty linked issues within the existing three local
fixture repositories. Browser confirmed 9 open / 2 closed milestones, 0/25/33/50/75/100%
progress examples, overdue/upcoming/no-deadline states and empty milestone content.
The ignored one-shot execution record is `.artifacts/local-forgejo/seed-milestone-fixtures.py`;
do not blindly rerun it. No non-fixture repository writes or deployment.

<!-- Illustration queue: personal Actions lists, four runner subpages and both owner storage overviews source-assessed without art; organization general and deletion pages source-assessed without extra art; labels and hooks also source-assessed without extra art; organization applications and Actions source-assessed without art; organization home and members source-assessed without art; team list/member/repository pages source-assessed without art; creation and invitations are next. See the per-page checklist. -->

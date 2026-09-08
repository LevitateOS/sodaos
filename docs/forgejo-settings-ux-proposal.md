# Personal settings UX investigation and overhaul proposal

Investigated 2026-09-08 against source `2cc73b6`, Forgejo 15.0.7 and local
presentation revision `2026-09-08.5`. This is a proposal, not an implemented
redesign or visual acceptance. It covers the entire personal settings area and
its child workflows; repository/organization administration and Sodaspaces remain
separate responsibilities.

## Recommendation

The user's clarified goal is a **visibly different layout and structure**, while
retaining Forgejo's native capabilities and limits. A cleaner version of the old
sidebar-and-sections arrangement does not meet that goal.

Remove the settings sidebar. Use compact identity context and grouped destination
menus across the top, followed by a full-width page-specific composition. Profile
becomes an identity editor; Security uses status-led rows; preferences use
setting-and-control columns; resource pages become open inventories. Focused
child editors have a parent link and one clear task. Preserve the existing
palette and fonts, but redesign content order, proportions and interaction flow.

Forgejo constrains data availability, handler ownership, form submissions,
permissions and interactive hooks. It does **not** require retaining the existing
navigation position, surrounding div hierarchy, attached segments, heading
placement or page silhouette. Rewrite that presentation markup where needed;
preserve the native interactive structures inside it.

The previous pass changed surfaces and tokens but did not establish a consistent
task structure. Some native forms and nested callers still bypass the shared
presentation roles. This overhaul must address both problems together.

## What was actually investigated

- All 11 enabled personal-settings destinations were visited using the existing
  authorized fixture: Profile, Account, Appearance, Security, Applications, Keys,
  Packages, Webhooks, Organizations, Repositories and Blocked users.
- Three child editors were visited: new access token, new package cleanup rule
  and new Forgejo webhook. Requested URLs returned 200 without redirects.
- Native templates, shared partials, navbar feature gates and route definitions
  were read. Exact 15.0.7 Account and Security handlers were also read to check
  whether password management can move between pages.
- Native viewport captures were checked visually: 11 dark desktop landing pages,
  six dark mobile landing pages, four light desktop landing pages, and three
  dark child editors at desktop and mobile. All 27 captures use
  `scripts/screenshot.mjs --verify`, including route, landmark, revision, registry,
  stylesheet-byte and browser-error checks.
- DOM inspection covered the 11 landing pages at 1440 and 390px; the six core
  pages also at 320 and 768px. No horizontal document overflow was observed in
  those checks. Keyboard activation opened the native SSH add panel; Cancel
  closed the empty panel. No settings were submitted or saved.

Evidence is under `.artifacts/settings-ux-investigation/`: `native-dom.json`,
`interactions.json`, capture directories and `index.html`. The interactive
structure sketch is a planning artifact, not a production-derived gallery or
native-page evidence. Captures show the existing UI; they do not show this proposal.

## Findings that should drive the design

| Observed problem | Evidence and consequence | Required change |
| --- | --- | --- |
| Navigation occupies most of the mobile viewport | Core page content starts at about y=637 on 390px screens; at 768px it still starts around y=619. The first screen largely explains how to navigate. | One compact mobile destination menu; remove the introductory artwork block. |
| Repeated headings obscure hierarchy | Profile has “Profile”, “Public profile”, then “About you”. Webhooks says “Settings” twice. Child editors can retain a parent title instead of naming the task. | One meaningful h1; h2 for real sections; a parent link on child editors. |
| Form styling depends on incidental markup | Password inputs and submit are 38px; adjacent email controls are 44px. Expanded SSH title input is also 38px. | Explicit roles on principal forms, including nested native callers. Native `ignore-dirty` is a behavior hook, not a visual category. |
| Forms spread across an unnecessarily broad column | The desktop content column is 856px; password, URL and short text fields stretch across it. Profile’s mixed grid leaves an orphaned pronoun field. | Keep the 1120px shell, but constrain ordinary form content to roughly 720px. Use deliberate short-field pairs only. |
| Identity editing starts with account renaming | Username and its rename warning lead the profile; avatar heading is at y=1266, below privacy. | Put the actual avatar and public identity first. Keep account URL/rename consequences visible in their own part of the same profile form. |
| Different jobs compete for attention | Applications places API tokens, authorized third-party apps and owned OAuth clients in one stack, with client registration always open. | Three named sections with distinct actions; disclose the registration form on request and on validation errors. |
| Important state is buried in explanation | Security leads with paragraphs; the actual unenrolled status follows them. Keys includes an important SSH-disabled notice below general SSH guidance. | State first, brief explanation next, action beside or below it. Preserve all policy and recovery warnings. |
| Shared child pages escape the contract | Cleanup creation retains a filled header; token creation has a bordered access group, empty outlined selector region and native permission controls. | Treat each child editor and shared native partial as a migration target, not as covered merely by its layout wrapper. |
| Empty pages look unfinished | Webhooks, repositories and organizations mostly contain a repeated title and one sentence; blocked users has a thin enclosing segment. | One useful empty statement and a permitted next action, without a miniature card or repeated illustration. |
| The common introduction is English-only | `layout_head.tmpl` hard-codes the same English description and eyebrow on localized native pages. | Remove redundant text and keep native translated labels. New copy needs actual locale delivery. |

These are observed design deficiencies, not findings from usability testing with
representative users. Completion times and user preference remain hypotheses.

## Three possible directions

| Direction | Benefit | Cost | Decision |
| --- | --- | --- | --- |
| Grouped top destination menus with page-specific bodies | Removes the settings sidebar and gives each task the full page width; creates a different visual hierarchy. | Most desktop navigation takes an extra click; requires clear active location and accessible menus. | **Selected direction after the user's clarification.** |
| Grouped desktop rail, compact mobile menu | Keeps all desktop destinations visible and improves density. | Preserves too much of the previous layout and form-stack structure. | Superseded; insufficient structural change. |
| Category landing pages or a settings dashboard | Could emphasize different kinds of tasks. | Adds an intermediate destination; aggregate status would require unavailable shared data or new handler work. | Not selected. |

A settings dashboard made from status cards would add a stop before the task and
need data that the shared layout does not receive. A single giant settings page
would combine unrelated handler contexts and save operations. Neither is the
recommended direction.

## Proposed navigation and shell

Keep the existing URLs and familiar native page names. The following group names
are working copy, subject to localization:

| Group | Existing destinations |
| --- | --- |
| Personal | Profile, Account, Appearance, Blocked users |
| Access and integrations | Security, SSH / GPG keys, Applications, Webhooks, Actions → Runners / Secrets / Variables |
| Resources | Organizations, Repositories, Packages, Storage overview |

Retain `EnableActions`, `EnablePackages`, `DisableWebhooks`, `EnableQuota` and
`HideNavbarLinks` exactly. Mandatory factor enrollment must still suppress
ordinary navigation. Do not add global counters or security badges to these
menus: the different page handlers do not provide a shared status inventory.

Desktop uses the 1120px shell with **no persistent settings sidebar**. Start with
a compact identity/context strip using the actual avatar and signed-in identity.
It is not an illustrated hero and does not repeat the page title. Follow it with
three grouped destination menus across the top. Mark the active group and the
current destination. Native disclosure elements and existing links provide the
foundation; use a small enhancement only for responsive open state, dismissal
and focus behavior. Opening navigation must never submit or discard a form.

The page body uses four deliberate compositions, rather than inheriting one
vertical stack for every page:

| Composition | Structure | Pages |
| --- | --- | --- |
| Identity editor | Compact portrait/source area alongside public-identity editing, then username and a distinct privacy band. | Profile |
| Setting rows | Section title, purpose and actual state in a roughly 240px introduction column; controls and the real save action in the adjacent column. Stack on mobile. | Account, Appearance, operational configuration |
| Status and inventory | Name/state/action on the first row, supporting metadata beneath, with native details or local editors revealed when needed. Use the available page width. | Security, keys, Applications, memberships, repositories, webhooks, package rules, Actions, quota |
| Focused editor | Parent link, task-specific h1 and a roughly 720px editing column. Remove secondary identity detail and the ordinary settings navigation when native policy already requires focus. | Token/OAuth/webhook/cleanup editing, factor enrollment |

These are compositions of real elements, not new generic component types or a
schema renderer. A column for section explanations is not a navigation sidebar.

Below the shared 900px breakpoint, show a single disclosure labelled with the
current settings destination. Opening it reveals the same grouped native links.
Use one navigation DOM, with an expanded usable fallback when JavaScript is
unavailable. Do not require opening a second category disclosure just to reach
the mobile links. No modal, scrolling tab strip or second global hamburger.
The actual page title and first task should normally appear within the first
300px at 390×844, excluding mandatory alerts and unusually long translations.

Child pages name the task: “New access token”, “Add cleanup rule”, “Add webhook”.
Provide a native parent link and keep the parent navigation item selected. Avoid
a parent h1 followed immediately by a duplicate task heading.

## Page-by-page overhaul

### Profile: identity first

Show the current avatar near the top with an explicit change action. The native
avatar helper already renders `.SignedUser` in the global navbar; reuse that
helper, not a new image service. On desktop place the portrait/source area beside
the public-identity editor; on mobile place it above. This is page content, not a
replacement sidebar. Reveal the existing upload/source form locally.
Avatar upload still has its own submit, native file constraints and separate
delete action. A file selection must never imply it has already been saved.
Keep one form DOM and an expanded no-JavaScript fallback. Expand the editor on
relevant native error returns; if the error cannot be attributed reliably, keep
it expanded whenever that error could concern the avatar. Errors and submitted
state must not disappear behind the disclosure.

Then show public identity: full name, pronouns, biography, website and location.
Use full-width biography and deliberate short pairs where space allows. Put
username/account URL and its rename, cooldown or managed-account explanation
after these ordinary edits. Keep the same native field and disabled conditions.
Remove unconditional username autofocus as a deliberate interaction change:
otherwise native focus would scroll past the reordered identity content. Check
initial navigation, keyboard entry and validation-error focus explicitly.

Privacy follows with visibility and email/activity/pronoun choices as readable
rows. Keep the native warning that profile visibility also affects access to
non-private repositories; do not present it as merely hiding a profile card.
Public identity, username and privacy remain **one native profile save**. Avatar
remains separate. Show a link to the current public profile, not a fabricated
preview or profile-completeness score.
Use sibling forms and an explicit grid composition: never nest the avatar form
inside the profile form or duplicate fields to achieve the layout. The main
profile form may span the identity and lower privacy areas while retaining one
native submission boundary.

### Account: distinguish email, password and closure

Lead with email addresses: address, Primary and Activated/Requires activation state, then
applicable actions. Keep activation, notification preference and primary-email
operations distinct. The add-email form follows the list. Provider-managed and
disabled states explain the actual restriction using existing native text.
Compose the section as an introduction/state column beside the address inventory
and editor, rather than a full-width heading above another attached segment.

Place password management next, with a readable width and the same standard
control appearance as other forms. Preserve current-password conditions,
autofill behavior, password-manager semantics and `ignore-dirty`. Keep a clear
link from Security to this section so password management is easy to find.
Its editor can expand on demand within the setting row. It must expand on native
errors or when following the password section link, with a usable fallback and
no duplicate form. Essential policy text must remain visible before expansion.

Account deletion is a clearly named final section with space before it. Its
warning and native confirmation are a meaningful boundary and should remain.
Retain explicit destructive text and all conditional consequences. Do not style
deletion as another routine blue submit.

### Security: actual status before setup instructions

Present authenticator-app enrollment and registered security keys as distinct
sections. Lead with enrolled/not enrolled or the actual credential list, followed
by the relevant setup/manage action. Keep recovery instructions beside the
enrolled factor and preserve warnings before any removal or regeneration.
Use aligned status/action rows instead of successive large headings and prose
blocks. Local expansion reveals setup or credential detail; mandatory notices,
errors and security consequences remain visible. Do not collapse native errors
or turn the WebAuthn ceremony into a replacement custom form.

Show linked providers and OpenIDs only where native configuration permits them.
Keep active/inactive provider distinctions and native linking flows. Do not
invent a security score, session inventory, device history or recovery-health
indicator. Call security keys what the selected native workflow supports; do not
relabel every WebAuthn credential as a passwordless passkey.

Enrollment and re-enrollment are focused child tasks: native QR/secret/passcode,
clear sequence and feedback, without art or competing navigation. Mandatory
enrollment retains the native restriction and warning. No recovery secrets belong
in screenshots or a shared review package.

**Password location constraint:** the Account handler owns password validation
and redirects back to Account. Security loads a different context. Copying that
form into Security would leave errors and success returning to another page.
Keep the real form on Account and provide a section link; a clean route move is
not a template-only redesign.

### Applications: separate three kinds of access

Show access tokens, then, only when native `EnableOAuth2` permits, applications
authorized to access your account and OAuth applications you own. Each visible
section gets a short purpose statement and
its own primary action. Show the owned-app registration form when requested,
retaining its open state on errors. Do not turn the whole page into tabs that hide
warnings, validation errors or a newly generated secret.
Give these inventories the full task canvas. Use an in-page text outline where
the populated page warrants it; do not rebuild three enclosing cards or retain
an always-open registration form simply because it was previously in a segment.

Token rows foreground name, repository reach and concise permissions, with native
created/last-used metadata. Keep detailed scopes in the existing disclosure.
Regenerate and revoke/delete remain explicit actions with native confirmation.

The token editor follows name → repository reach → permissions → generate. Keep
native defaults, category labels and allowed combinations; do not invent “safe”
permission presets. Remove purely decorative fieldset/empty-wrapper outlines
without changing the repository selector’s conditional behavior. Preserve its
GET search/pagination/selection and final POST semantics. OAuth client editing
must preserve built-in app restrictions and one-time secret display.

### SSH / GPG keys: distinguish access and signing

Use purposeful SSH, optional principal and GPG sections. Put the instance’s
SSH-disabled/signing-only notice before generic access guidance when applicable.
Each existing key is a compact row with name, relevant verification/external
management state, fingerprint/key ID and native usage/expiry metadata.

Keep add panels close to their list and initially collapsed except on errors.
The keyboard-opened SSH panel currently leaves focus on its trigger and renders
a 38px title field. Standardize its form role and make entry/cancel focus behavior
deliberate while preserving native panel hooks. Keep verification commands,
challenge/signature fields and external-key restrictions intact. These are
Forgejo Git/signing keys, not Soda’s project-development access keys.

### Appearance: four clear save boundaries

Theme, language, repository hints and hidden comment types remain four native
forms. Use the same labels/help/action rhythm and retain a specific submit for
each. A global Save or autosaving toggles would misrepresent the native behavior.
Render the short settings as horizontal rows: explanation at left, selector or
checkbox and its submit at right. The comment checklist occupies the wider
control column. On mobile each row becomes an explanation followed immediately
by its controls and action, without an enclosing card.

Use the actual theme selector; illustrated theme previews are optional future
work requiring real theme rendering. Present the 14 comment groups in two
readable columns on desktop and one on mobile. Keep the meaning explicit:
checked types are hidden, not shown. Avoid flipping stored boolean semantics to
make a switch look more familiar.

### Remaining destinations and child workflows

| Destination | Proposed composition | Native workflows to retain |
| --- | --- | --- |
| Webhooks | Name the page Webhooks; compact endpoint/provider/status rows; provider selector within the editor; open event groups. | Configured provider list, new/edit, secrets, event conditions, delivery history, replay and delete. Last delivery state is not a general health score. |
| Packages | Cleanup-rule inventory first; Cargo and Chef as clearly named operational sections. | Rule add/edit/remove/preview, native guidance on keep/remove precedence, Cargo initialize/rebuild, Chef keypair-generation and replacement warnings. |
| Organizations | Membership rows with identity and native permitted actions. | Creation gate, membership pagination, named leave confirmation. No organization-management controls invented here. |
| Repositories | Owned-repository inventory with type/size/visibility metadata; explicit separate unadopted-directory area when permitted. | Adoption/deletion gates, origin links, native modal POSTs. This page is not repository collaboration settings. |
| Blocked users | Plain list with identity/date and explicit Unblock action; useful empty sentence. | Native direct unblock POST and server-result feedback; no confirmation modal is currently present. |
| Actions: runners | Compact status inventory and task history; focused new/edit/setup tasks. | Native runner state/labels, registration/setup token handling, one-time output and delete. Token-reset GET is state-changing and is not safe navigation to exercise for coverage. |
| Actions: secrets/variables | Name-first rows and focused add/edit forms; distinguish masked secrets from visible variables. | Native modal population, POST targets, blank secret edit values, delete. Never prefill or expose a stored secret. |
| Storage overview | Readable usage/limit bars and expandable subject breakdown. | Native quota groups, exceeded state and subject totals. No invented cleanup button. |

Also keep the related forced-password-change and recovery flows reachable and
consistent with the existing authentication boundary. They are focused native
journeys, not extra sections to embed into the ordinary settings shell.

## Shared interaction and presentation contract

- One h1, h2 section headings, 16px body, existing shared label/metadata styles.
  Retain Fraunces, Barlow and Plex Mono in their established roles.
- Standard controls 44px; compact list utilities 36px; icon actions 40px and at
  least 44px for coarse pointers; radius 8px. Joined controls have square inner
  corners. Checkboxes/radios need a usable label target, not a giant glyph.
- Use 4/8/12/16/24/32px spacing. Open sections generally separate by 32px; field
  groups by 16px. Removing a border must not collapse that separation.
- Save actions stay inside their actual forms. Use meaningful text, not icons,
  for submit, regenerate, revoke, delete and ambiguous operations. Familiar
  secondary utilities may use localized icon labels and hover/focus help.
- Keep native alerts immediately below the compact page heading. Do not invent
  “Saved” states from clicks or optimistically claim success. Preserve error
  text, field state, submitted-value handling and the native dirty-form behavior.
- Review unconditional autofocus on ordinary landing pages, including Account:
  it must not skip the reordered opening task. Preserve deliberate focus in
  focused editors/enrollment and validation returns, and leave autocomplete and
  password-manager protections intact. Record these as explicit interaction deltas.
- Do not use a global sticky save bar. Any future sticky element must keep
  focused controls visible, including when the mobile keyboard is open; see
  [W3C focus guidance](https://www.w3.org/WAI/WCAG22/Understanding/focus-not-obscured-minimum.html).
- Keep primary task content open. Boundaries remain appropriate for alerts,
  confirmation dialogs, code/challenge material and independently selectable
  items. Do not remove every line in native data tables or meaningful warnings.

## Implementation ownership and order

1. **Shared shell plus Profile.** Replace the generic settings intro, add the
   targeted native-navbar override with grouped top navigation, remove the
   settings sidebar, implement the mobile destination disclosure, fix heading
   semantics and complete the identity/privacy/avatar composition.
2. **Account, Security and Keys.** Establish one form appearance including native
   nested panels and behavior-hook exceptions. Complete state/action hierarchy
   and preserve each security flow’s conditions.
3. **Applications and Appearance.** Complete access/permission editors, independent
   save feedback and conditional form disclosure.
4. **Resources and integrations.** Apply the same contract to all inventories,
   provider forms, cleanup editors, Actions and quota callers.
5. **Full regression and evidence review.** Review the entire account-settings
   matrix, not only the first viewport of Profile. Internal checks continue
   throughout; these are implementation slices, not repeated design approvals.

`components.css`, `components-forms.css` and `components-settings.css` own shared
appearance. `account-settings.css` owns personal-settings composition;
`account-details.css` owns only necessary page structures. Remove duplicate
mobile-nav rules and conflicting appearance rules during migration. Avoid adding
another route-selector layer. Native partials need explicit form/action roles
when their nested structure escapes the shared contract.

Use targeted templates and small repeated presentation partials, never a
generic settings schema or new frontend framework. Preserve actions, methods,
field names, IDs, gates, CSRF, modal/data-panel hooks, WebAuthn, native dropdowns,
token selectors, pagination and provider dispatch. Keep the test-only inventory
current when adding a navbar/native-partial override. Update source parity checks
with deliberate presentation deltas while preserving assertions for every native
workflow; do not simply reset hashes to waive changed behavior.

New group names and explanatory copy require the existing
[localization delivery contract](../appliance/forgejo/i18n/README.md). Its empty
Soda catalog cannot be mounted directly: the selected version replaces native
catalogs rather than merging additions. Until proper generation/delivery is
implemented, reuse existing translated keys and avoid new English-only UI copy.
Locale activation is separate from an ordinary template reload.

This remains within Forgejo’s documented custom templates/assets mechanism, with
version-specific compatibility review. It does not require a downstream
executable or replacement settings backend. See
[Forgejo’s customization documentation](https://forgejo.org/docs/v15.0/contributor/customization/).

## Acceptance and remaining evidence

The implementation should demonstrate that a person can find a setting, understand
its current state, make the intended change and identify which operation saved.
It must also pass the user's structural goal: the default desktop composition
must not retain a settings sidebar or the old repeated heading/attached-section
stack. Profile, Security, preferences and inventories should be visibly distinct
task compositions, built from consistent shared presentation rules. Removing
borders and shrinking margins alone is not completion.
Proposed usability tasks are: update a public bio and avatar; add an email and
send/follow its native activation flow;
find password and factor settings; add a signing key; generate a scoped token;
change interface preferences; create a webhook; inspect a cleanup rule. Test task
completion and error recovery, not just matching dimensions.

Verify light/dark at 1440×1000 and 390×844, with 320px and intermediate widths;
long translations; keyboard navigation and focus; empty/populated lists; selected,
disabled, loading and validation states; native dialogs/panels/dropdowns; token
selection and provider-specific editors. Real captures must satisfy the existing
route/revision checks. Component fixtures remain separate evidence.

Current live evidence is mainly an empty, non-admin account with TOTP unenrolled
and SSH service disabled. Enabled/mandatory factors, actual WebAuthn ceremonies,
providers/OpenID, populated keys/tokens/grants/OAuth clients, one-time generated
access-token/OAuth-client secret output,
multiple/unverified emails, memberships, owned/unadopted repositories, populated
cleanup/history, Actions and quota branches remain unverified in native UI.
All successful and rejected mutation/validation-return journeys remain untested.
No resources, permissions or preferences were changed to manufacture coverage.
Do not claim complete visual or functional acceptance from this investigation.

## Source entry points

- `appliance/forgejo/templates/user/settings/layout_head.tmpl` and account leaves.
- Native `templates/user/settings/navbar.tmpl`, `keys_*.tmpl`, `security/*.tmpl`,
  shared runner/secret/variable/quota, package-cleanup and webhook partials in the
  retained `.artifacts/forgejo-presentation/upstream/templates/` export.
- Exact upstream [Account handler](https://codeberg.org/forgejo/forgejo/src/tag/v15.0.7/routers/web/user/setting/account.go)
  and [Security handler](https://codeberg.org/forgejo/forgejo/src/tag/v15.0.7/routers/web/user/setting/security/security.go),
  retained in the investigation directory.
- Exact upstream [native routes](https://codeberg.org/forgejo/forgejo/src/tag/v15.0.7/routers/web/web.go),
  retained in `.artifacts/research/forgejo-15.0.7-notifications/web.go`.


Later presentation decision (2026-09-08): pronoun editing, privacy controls and
public display are removed. Earlier pronoun references above describe the
investigation baseline, not the current UI contract.

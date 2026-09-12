# Native Soda pages — historical review

This records revision-bound findings, review recommendations and their resolution,
not an active completion checklist. “Changes required”, “remaining” and numbered
exits below describe the reviewed revisions; they do not reopen resolved findings
or prescribe the same run sequence for later changes.

Use the [current handoff](implementation-status.md) for installed state/evidence,
the [native page contracts](forgejo-soda-pages-plan.md) for required behavior and
the [combined plan](native-pages-runners-plan.md) for remaining coordination.
The original findings and check results below are retained as historical evidence.

## Resolution of R1–R4 — `18c6b07`

All four findings below have been corrected and locally verified. Entry lifetime
now guards OAuth navigation; real BFCache restores usable full-page owners after
original-actor validation without auto-login; uncertain/dispatched project writes
remain blocked; Go template callers follow the common epoch; navigation says
Runners. The restoration work also fixed repository mounting discarding the entry's
Lit marker. The common graph epoch is `2026-09-10.native-pages-6`.

The existing native fixture passed all 15 consumers and its real authentication/
cache/history/logout parent, including actual persisted Back for both affected
pages. Exact terminal/retention/Return/End and uncertain-write assertions use the
existing synthetic operation peers, not native project processes. Focused Go
template, TypeScript/Lit, Forgejo/frontend, fresh web/store race, Cockpit and
payload/staging/orchestrator checks passed. See the leading
[implementation handoff](implementation-status.md) and
`.artifacts/pages-review-fixes/` for complete receipts, retained failed attempts
and the local preview reload. The wider coverage gaps below remain open; no
step-5 exit or step-6 delivery is claimed.

## Original review (historical baseline)

Review baseline: **`97a2d5e`**, with production source from `d838262` and native
navigation/shell retirement from `99c2c31`. The checkout was clean. This reviews
steps 1–5 and the step-6 delivery boundary in [the plan](forgejo-soda-pages-plan.md),
not the separate runner backend/provider parity implementation.

**Baseline verdict: changes required.** The native-host architecture and most security/
operation boundaries are implemented, but the remaining work is not only visual
review or Linux validation. Two lifecycle defects and a mandatory-test regression
were reproduced locally. Do not describe steps 1–4's earlier passing evidence as
proof that the current combined candidate satisfies every integration exit.

No production fix is included in this review. Temporary reproduction additions
used the existing browser fixtures, were saved as patches in ignored evidence,
and were removed from the working source afterward. The two installed-driver files
handed to Runners in `97a2d5e` were not edited.

## Findings, in priority order

### R1 — high: retired entry can still initiate OAuth on a late 401

Owners: step 2 connection / step 3 entry.

- `frontend/spaces/soda-connection.ts:138–158` awaits session discovery and then
  calls `location.assign` itself. It has no entry-lifetime/generation input.
- `assets/branding/forgejo/soda-native-page.ts:11–26` increments the generation
  on departure/restoration, but checks it only **after** `connectPage` returns.
  At that point the navigation side effect has already happened.
- A delayed unauthenticated session response can therefore navigate a retired
  entry to OAuth, even after that entry has displayed explicit Retry feedback.
  This violates entry-only connection and can begin another login-context attempt
  instead of leaving the user in control. A repeated OAuth attempt is not a
  harmless probe; the existing callback lifecycle can retire terminal attachment.

**Reproduced:** augmented the existing `connection.test.ts` fixture to hold its
session 401, dispatch pagehide/persisted pageshow, wait for Retry, then release the
response. The no-new-login assertion observed **one** OAuth navigation. This is a
synthetic late-response/lifecycle reproduction, not a real pending-request BFCache
or native process test. The existing test holds the later *module import*, so it
misses this earlier boundary.

**Fix boundary:** make navigation conditional on the current entry lifetime, not
just rendering. Pass cancellation/currentness into the connection operation or
return an intent for the still-current entry to activate. Check again immediately
before navigation; transport abort alone is not the guard. Add cases for late 401,
removed mount and a superseding explicit retry. Do not add another OAuth owner or
replay a user operation.

### R2 — medium: real BFCache restores permanently stale full-page controls

Owners: steps 3–5 native page restoration.

- `frontend/spaces/sodaspaces-workspace.ts:226–234, 715–717, 1365–1385` permanently
  invalidates on pagehide/BFCache and refuses refresh while stale.
- `frontend/spaces/sodaspaces-project.ts:163–170, 319–327, 361–363` does the same
  for repository settings.
- `assets/branding/forgejo/soda-native-page.ts:16–18,45` does not revalidate/remount
  an already mounted component on persisted pageshow. `mounted` remains true.

**Reproduced on real stock Forgejo:** using `TestNativeConnectionFixture` and its
existing non-intercepted two-tab history setup, opened each full page, navigated
through native Issues and used real Back. Both documents reported
`pageshow.persisted=true`; both retained the permanent “Page … changed. Reload”
state. A full reload is required to use their controls again. Runners resumes;
this defect is specific to the full-page workspace/repository host integration.

This does **not** establish that terminal processes were killed or IDs lost. No
native terminal was opened. It establishes a missing usable restoration path in
the requested native navigation journey. The existing history assertion checks
Runners; page/drawer/page tests exercise fresh documents, not this restoration.

**Fix boundary:** on a genuine history restore, recheck the original actor and
recreate the retired UI owner through its existing mount/disposal interfaces.
Restore only saved exact terminal locators under existing authority. Preserve
finite retain/Return/End, uncertainty and explicit-retry suppression; never create
or replay a mutation. Disposal must also release the page-level ResizeObserver
and viewport listeners. Do not make a permanently stale component writable by
simply clearing its stale flag.

### R3 — medium: the cache-version change breaks mandatory Go template tests

Owners: steps 4–5 caller/payload regression checks.

- `scripts/forgejo_native_pages_test.go:82` still expects the entry URL ending
  `soda-native-page.js?v=1`.
- `scripts/sodaspaces_templates_test.go:57,73,125` still expects unversioned
  `sodaspaces.js` and `sodaspaces.css` URLs.
- Production templates now use `?v=2026-09-10.native-pages-5`.

**Reproduced on macOS without a Linux builder:**

```sh
go test -count=1 ./scripts \
  -run 'Test(NativeSodaPage|SodaspacesTemplate|SodaOperatorNavigation)'
```

`TestNativeSodaPageHost`, `TestSodaspacesTemplates` and
`TestSodaspacesTemplateEscapesContext` fail. These source assertions would also
fail on Linux. The earlier handoff's Linux/macOS-only explanation of aggregate
failure was incomplete: the failed aggregate log also contained these integration
regressions.

**Fix boundary:** migrate these existing assertions to the common reviewed epoch
and preserve path/prefix, escaping, no-duplicate-mount and native-context checks.
Do not remove them or weaken the asset-version contract. Run the scripts package
explicitly as well as web/store tests; another full native build is not necessary
to repair or verify these assertions.

### R4 — low: the normal Runners navigation is still labelled “SodaOS settings”

Owner: step 4 normal navigation.

`assets/branding/forgejo/soda-settings-link.ts:9` creates the operator link with
text `SodaOS settings`. `tests/forgejo/settings-link.test.ts:27` preserves that old
label. The plan explicitly selects **Runners** for this local-capacity destination.
The URL and configured-operator gate are correct; this is a discoverability/plan
mismatch, not an authorization bypass.

**Fix boundary:** update the link and its current callers/tests together. Keep
Forgejo Actions administration separate and keep Cockpit Runners.

## Step-by-step assessment

| Step | Established implementation/evidence | Remaining issue or limit |
| --- | --- | --- |
| 1 — native host | Uses stock dashboard handler with native head/footer; bounded view/canonical repository-ID selection; ordinary dashboard retained; one main landmark; no invented native authentication. Existing local browser proof covers actual HTML, native login, profile and notification menus. | Activation/password-change/2FA gates are source-backed upstream preservation, not independently exercised fixture states. Native CSP ownership is preserved; equality of policy headers is not proof of an additional Soda CSP. Version assertions currently regress under R3. |
| 2 — connection/logout | Fixed transaction-bound native returns, scope/actor validation, matching-session reuse, schema-v9 OAuth-cookie association, cancellation under the existing terminal lock, cross-tab retirement, native capture-phase logout and both partial outcomes have focused evidence. | R1 breaks entry-lifetime containment. Logout remains deliberately non-atomic; native-only/offline/missed-event paths do not guarantee immediate Soda revocation. No new schema/authentication architecture is indicated by this review. |
| 3 — page bodies | All three native hosts use their existing Lit controls and protected APIs. Repository display uses protected canonical context; current-session checks survive provider I/O. No new terminal/native runner owner was added. | R2 prevents usable full-page BFCache restoration. Complete populated themes/mobile/scroll/focus visual review is still missing. |
| 4 — navigation/retirement | Fixed native links/bookmark bridges; old Go shells/renderer/boot callers removed; collection/operation authority remains in the APIs. Existing page/drawer/page journey preserves exact synthetic session selection, retain/Return and named Ends without replay. | R4 label remains. Shared installed-driver adaptation is now runner-owned under the bounded handoff; it is not already completed native acceptance. R3 is a later regression to mandatory callers. |
| 5 — combined candidate | Common epoch covers entries and transitive module graph; duplicate/removed-mount cases, actual cache hits, real runner BFCache and additional synthetic authority cases exist. Focused source/browser/race/payload checks pass as recorded below. | R1–R3 block completion. The old/new-cache upgrade, populated visual review, combined history/draft/actor/terminal matrix and supported-Linux aggregate remain incomplete. |
| 6 — delivery | Exact-candidate, paired artifacts/schema, preserved-state rehearsal, explicit target/action authorization and single maintenance executor remain required. | Not executed for this native-pages candidate. No build/export, installation, provider registration/job, project lifecycle or Cockpit-retirement acceptance is inferred here. |

## Coverage gaps that are not additional proven production defects

- **Cached-client acceptance is narrower than an upgrade test.**
  `tests/forgejo/native-connection.test.ts:106–131` warms predecessor URL identities
  with **candidate bytes**, after current pages have already loaded. This verifies
  cache hits and current URL/graph identity, not a client holding the previous
  release through its first new navigation. Retain that useful test, then exercise
  a genuine old-payload/new-payload boundary with the same fixture. Already-open
  modules cannot be called upgraded merely because a new document succeeds.
  **Step-5 follow-up:** the same fixture now has a manifest-bound b8af68c asset case.
  A fresh context begins on native plaintext (no candidate scripts), verifies genuine
  old page/API/Lit module hashes and zero-age responder contact, then checks changed
  bare-URL bytes, first candidate entry/transitive graph and Back/Forward. This case,
  all 16 consumers and the full native parent passed locally; the default six-hour
  case remains explicitly labelled candidate-byte caching. This closes the missing
  genuine **asset-byte** boundary. A subsequent bound-binary phase now executes
  the actual b8af68c backend/page against the existing native fixture, preserves the
  open document through its fresh database's v6→v9 transition, uses the old Refresh
  control and observes genuine pagehide retirement without API mutation replay.
  Current-handler race and all mandatory consumers passed. Actual Back reloaded
  into the current owner; predecessor BFCache restoration is not claimed. This is
  not installed cache/CSP equivalence or native terminal continuity.
- **Distinguish stock preview from installed cache configuration.**
  `appliance/config/forgejo.env:12–13` selects native
  `FORGEJO__server__STATIC_CACHE_TIME=0`; it is not a Caddy cache override. The
  local stock preview/test file server uses six hours. Check the actual final
  payload/configuration rather than replacing installed revalidation assertions
  with the preview behavior. Locked vendor bytes and canonical payload paths
  remain intact.
- **Authority matrix has mixed evidence levels.** Real fixture admission proves
  operator-without-site-admin; Go/synthetic tests cover admin-without-operator,
  owner/member, denied/private/missing repository, unavailable provider and actor
  changes. This is useful server/client coverage, not a complete real-native-user
  matrix or new provider permission-change proof.
- **Visual evidence is failure-state only.** The retained `screenshot.ts` captures
  reviewed three error hosts at wide/light and narrow/dark. They cannot establish
  populated runner forms, repository controls or active workspace scrolling/focus.
  Continue with the existing authenticated fixture and capture tool, not another
  login harness. No production backend installation is needed for local review.
  **Step-5 follow-up:** existing consumers now use `scripts/screenshot.ts` for a
  bounded reviewed light/dark 390/768/1440 matrix: populated runner rows, scrolled
  confirmation/registration, repository metadata/Join and workspace navigation.
  The final run produced 36 private fixture captures and passed all consumers and
  the native parent. Initial transition/framing issues and the wrong-tab capture
  failure remain recorded. Synthetic operations and the empty synthetic terminal
  screen do not prove installed provisioning, terminal text/editor usability or
  complete Access/error/profile-menu/physical-keyboard acceptance.
- **Documentation remains inconsistent.** For example, the historical detailed
  Spaces section in `sodaspaces-plan.md` still describes the Go-shell owner;
  `AGENTS.md` retains source-vs-historical schema-v5/v6 phrasing even though this
  integration is v9. Top-level historical disclaimers help but are not completion
  of the current-guide consistency pass. Keep dated delivery evidence untouched.

## Checks performed for this review

Evidence: `.artifacts/native-pages-review-97a2d5e/`.

- Rebuilt emitted assets and passed `bun run typecheck` (strict TypeScript/Lit).
- Existing Forgejo suite: **37 pass, 21 gated skips**, after removing the temporary
  late-entry regression probe (`forgejo.log`).
- Web/store Go race command passed (cached results for unchanged source,
  `go-race.log`). Canonical payload **4 pass**; staging fixtures **7 pass** using
  a non-symlink evidence-local temporary directory. These are not a sealed export.
- Focused Go template command failed as described in R3 (`templates.log`).
- The temporary late-session case failed as described in R1 (`late-entry.log`;
  exact probe in `late-entry.patch`). Its other existing logout test passed.
- The real native page fixture's **15 existing consumer cases passed**, then the
  additional real-history assertions failed as described in R2. Native fixture,
  OAuth app, DB and browser state are retained at `.artifacts/pages-se6siP/native/`;
  `bfcache-pages.log` identifies it and `bfcache-pages.patch` preserves the probe.
- No installed driver/provider phase, retained-project mutation, appliance
  deployment, host trust change or Cockpit edit was performed. No new screenshot
  was generated in this review; visual conclusions above cite the earlier receipt.

## Follow-up order and handoff

1. Repair R1's navigation side-effect lifetime and add its permanent regressions.
2. Repair R2 using the existing full-page mount/disposal/identity contracts; extend
   real history coverage to usable state and exact terminal locators.
3. Update the missed Go template assertions and Runners label; rerun the relevant
   current source/browser suites on this Mac.
4. Complete populated visual, cached-upgrade and combined authority/draft/history
   acceptance, then the supported-Linux aggregate and documentation pass.
5. Identify the final source candidate and remaining shared-file handoff before
   separately authorized step-6 native build/export/delivery work.

All corrective implementation above can proceed without an x86_64 Linux builder.
That environment is a final validation/delivery dependency, not a reason to defer
these source fixes. Preserve the runner agent's exclusive ownership of
`tests/installed/sodaspaces.ts` and `tests/frontend/sodaspaces-probe.test.ts`;
coordinate any new need in those files instead of editing them concurrently.

# Migration source chooser redesign

2026-09-10, starting at `eed31b7`. The user requested a full redesign of the
`/repo/migrate` source chooser, with the design completed before implementation.

## Design

The original page repeated ten large cards with a second inset panel and nearly
invisible provider marks. The replacement separates the introduction from the
choice: a larger migration illustration and title at left, and a compact source
chooser at right. A distinct Git URL entry is followed by hosting services, without
ranking providers by popularity. The illustration, wordmark, fonts and palette are
existing canonical assets.

A separate HTML/CSS prototype was rendered and inspected in desktop, mobile and
dark layouts before production files changed. It remains under
`.artifacts/migrate-redesign-20260910/design.html`, with `design-desktop.png`,
`design-mobile.png` and `design-dark.png`. The prototype is design evidence, not a
native screenshot or a second application. Production uses the original native shell.

The three-column service grid becomes two columns at 1100px. Below 760px, the
introduction sits above the chooser with compact artwork; the extra next-step
explanation is omitted to prioritize choices. Cards retain full translated
descriptions, visible focus and hover feedback, and reduced-motion support.
Provider marks use an unpadded 32px frame (36px for Git); the wide Codebase wordmark
retains its aspect ratio. This avoids the original combination of native icon
padding and a shrunken fixed box. Page-owned classes replace the native chooser's
layout classes; provider forms and migration progress keep their existing CSS.

## Ownership

`repo/migrate/migrate.tmpl` still renders Forgejo's `.Services`, native helper and
translated provider descriptions. Git is featured only when present. Other services
retain their supplied order, titles and icon selection; an empty list renders no
fabricated options. Every link preserves `AppSubUrl`, service ID, organization and
mirror query context. The chooser adds no JavaScript, handler, authentication,
provider inventory or migration behavior. The provider forms remain stock fallbacks.

`onboarding.css` owns the redesign; the header increments its asset version and
presentation revision. The presentation inventory records the reviewed structural
change and removal of this page's shared-intro call. Other callers are unchanged.

## Validation and local preview

Passed checks:

- Full `go test -mod=readonly ./scripts`, including actual chooser rendering with
  Git in different positions, unavailable Git, Git only, no sources, escaped titles
  and query context that must not inject another parameter. The first full-suite
  attempt was blocked by filesystem access to Go's cache; the retry passed with
  normal cache access. Focused onboarding checks had already passed.
- Required `bun run typecheck`, including the analyzer and positive/negative fixtures.
- Both presentation-inventory tests, including native caller tracing.
- Isolated canonical preview builds, plus source/served `onboarding.css` equality.
- Read-only local browser test: seven widths (1440, 1100, 900, 761, 760, 390, 320)
  in light/dark, ten distinct source links, logo dimensions/padding, content
  containment, focus order and hover feedback. All ten links reached their native
  provider forms with the matching hidden service ID and visible clone-URL field.

The opt-in browser test is
`SODA_FORGEJO_MIGRATION_REVIEW=1 bun test --timeout 120000 tests/forgejo/presentation/migration-browser.test.ts`.
It uses only the existing dedicated local screenshot profile and bypasses its HTTP
cache for the run. No form is submitted and no credentials are entered.

The preview received only the new `onboarding.css`, backed up beforehand. Forgejo's
supported `manager reload-templates` loaded the mounted template revision; no
container restart or mount/configuration change occurred. Template reload also
exposed the already-mounted Spaces navigation from earlier source work; this change
does not author or validate that feature. Earlier general preview CSS drift remains;
the browser test verifies this page's changed CSS and template revision specifically,
not the entire current frontend asset set.

Final native captures use `scripts/screenshot.ts`, the saved fixture profile,
`--scroll-top`, browser-only theme selection and `--full-page`. Evidence under
`.artifacts/migrate-redesign-20260910/` includes `native-final-desktop/` (chooser,
Git form, GitHub form), `native-final-mobile/` (390px chooser), and
`native-final-dark/` (1440px chooser). Desktop/mobile/dark screenshots are actual
stock 15.0.7 pages, distinct from the design prototype. Logs and original bytes
are retained alongside them.

No repository was imported or created, provider credentials entered, account
preference submitted or fixture configuration changed. Other availability states
have actual-template tests, not mutated-fixture runtime evidence. This is local
presentation/navigation validation, not migration execution or appliance delivery.

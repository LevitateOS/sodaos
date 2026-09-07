# Sodaspaces: current work

Use **Forgejo's native frontend**, with one repository **Sodaspaces button** opening
a right-side shared-environment drawer—**no new tab or standalone page**. Both standalone Soda frontends and duplicate
forge adapters are removed. The button/drawer and authenticated connection to the
retained Go backend are **not implemented**. See the [handoff](implementation-status.md).

## Ownership

- Forgejo owns native pages, scripts/styles, identity, permissions, collaboration,
  Git keys and administration. Preserve its configured workflows instead of
  reimplementing them or measuring progress by public API coverage.
- Soda owns additional preferences, development-access public keys, environment
  associations/memberships and real native provisioning through its bounded API.
- Project environments remain persistent and shared, with personal Linux accounts
  inside each project. Sodaspaces is a UI name, not a container/database rename,
  disposable per-user environment, browser IDE or lifecycle expansion.
- Stock Cockpit and its separate Tailnet/Runners frontend, native logic and tests
  remain operator-only. Providers own CI scheduling/registration/results.

## Next implementation

1. **Prove the supported integration boundary.** Use the selected Forgejo version's
   actual hooks/configuration. Resolve stable repository context and an authenticated
   native-page → Soda request path, including origins, cookie scope, identity
   mismatch, CSRF, expiry/logout and return navigation. A hook or OAuth grant alone
   does not solve embedding or native-session transfer. Stop and explain a concrete
   unsupported requirement; do not fork Forgejo, borrow cookies or build an HTML proxy.
2. **Deliver the smallest read-only button/drawer.** Show real existing, incomplete or
   absent environment state under current actor authority. Use native markup/styles
   and focused JavaScript; preserve native DOM/scripts, keyboard/focus behavior and
   navigation. Opening it must not create, join, start or repair anything. Stage only
   required hooks/assets through existing build/install callers, with notices and
   explicit public configuration; do not shadow unchanged upstream templates.
3. **Connect explicit actions.** Wire create/join, own development keys and connection
   details to the existing Go/helper operations. Preserve incomplete reservations,
   stable memberships and Linux data; report actual failure without mutation replay.
   As a separate follow-up, add the requested terminal into the user's **existing** project-local
   account/home, with bounded PTY/transport and session lifetime. No implicit join,
   startup, host shell or private-key upload.
4. **Prove and deploy separately.** Author focused authorization/failure/browser tests
   and exercise a real native-page journey under explicit target/action permission.
   Rehearse a matching candidate against fresh and copied populated state before
   separately approved cutover. Preserve native Forgejo workflows, enrolled factors,
   grants, accounts, keys and projects. Complete [native validation](native-validation.md)
   on x86_64 and independently on aarch64, including retained operator services.

## Boundaries and completion

[Architecture](architecture.md), [API](dashboard-api.md),
[credentials/migrations](dashboard-credentials.md), [integration details](forgejo-frontend-integration.md)
and [deferred scope](deferred.md) govern implementation. No new component library,
release/update platform, private-resource branching or generalized recovery is selected.

Production `internal/host/` and `project-os/` belong to the product, not an outside
harness. [Native support tools](native-support.md) may invoke owned checks and supply
transport/artifact evidence, not duplicate scenarios, redefine API/config/schema
contracts or create a second readiness gate. Optional media/helper ports do not
block work using existing authorized tools. Coordinate shared build/stage/install
changes with the actual production caller and retain fail-closed payload checks.

Only the historical bounded **U08** native proof is accepted. Removing the old
M/U/P roadmaps and 179-group audit does not accept the new UI or final product, nor
cancel required security, persistence, native compatibility or operator checks.
Those labels remain in old evidence and some tool arguments; they are not a new
numbered work programme. Historical plans/audit are available in Git at `9f3baa7`.

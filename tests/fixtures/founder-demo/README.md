# Northstar reference application

`Northstar.can` is authored reference input for the founder journey. It is not
Factory output, a build receipt, an admitted native image or a released company
application. The fixture gives the runtime and browser work a concrete workflow
to validate while the complete Factory-to-application path is assembled.

## Blank Factory project: protected check only

The development-only protected check prepares a new Factory repository without
an application implementation. Copy **only these eight files** into the new
repository, naming the shell scripts `setup.sh` and `check.sh`:

- `project-setup.sh` and `project-check.sh`: native preparation and check entrypoints.
- `northstar-check.json`: accepted environment and policy configuration.
- `northstar-check.mjs`: configuration validation and the full scenario sequence.
- `northstar-check-source.mjs`: source compilation, exact ancestor extraction and bundle production.
- `northstar-check-runtime.mjs`: runtime lifecycle and authenticated HTTP transport.
- `northstar-check-actors.mjs`: synthetic accounts, real login and business operations.
- `northstar-check-policy.mjs`: manager and Finance outcomes, history and retained-record assertions.

Do not copy
`Northstar.can`, the reference runner, a compiled artifact, an earlier bundle,
accounts or data. The Factory must create the actual application source and
nonempty operation examples from the accepted business brief.

For native Project preparation, select all eight root filenames as source paths
and approved setup files. Required tools are `can`, `can-dev`, `can-platform`,
`can-soda`, `node` and `git`. Review the exact scripts/configuration and accept
them through the existing Project settings workflow; the Project administrator
supplies the full admitted Canlang tools under `/opt/canlang` and verifies both
role preparations. No source checkout, dependency download, browser add-on,
engine access, shared database or production credential is needed by this check.
The complete eight-file import closure is protected and each file is below
the native context reader's 8 KiB per-file limit as well as the setup-file
limit. Planning reads the complete approved source set in bounded batches;
the module split does not omit or weaken any validation. Keep the smaller
public business contract in the new repository's own README; this fixture
guide is not an additional protected setup file.

Blank-project preparation checks the actual installed commands and private
writable storage. Once `Northstar.can` exists, ordinary preparation also
compiles that actual candidate and runs its nonempty operation examples through
the protected runner's `--source-only` mode. A CI publisher must require that
source-check success, exact candidate and approved check revision; blank-project
readiness alone is insufficient CI evidence. Neither preparation result asserts
whole-application outcomes or emits application bundles. Native
delivery verification supplies `SODA_VERIFICATION_RUN_ID`; that execution must
find the generated source and pass every application assertion. Missing source
can never pass delivery. The same protected script therefore works before an
app exists without a native lifecycle change. The adopted `project.check`
manifest binds accepted outcomes to that script; no future accepted-intent ID
is invented in its initial source.

The initial accepted direction must include this small **public interface
contract**, so these are disclosed requirements rather than hidden assertions:

- Entry source `Northstar.can`; app `NorthstarEquipment`; team-scoped model
  `EquipmentRequest` with `title`, `reason`, exact EUR `amount`, server-owned
  `requester`, saved boolean `needs_finance`, `stage`, `decision_note`, and
  approval/delivery actor fields. User-facing labels remain human-readable.
- Roles `manager`, `operations` and, for the Finance change, `finance`.
  Members read their own requests; the authorized team roles read the requests
  they process. An unrelated employee's list/page cannot disclose another's
  request. Managers cannot approve their own requests.
- Public operations `request_equipment(title,reason,amount)`,
  `approve(request)`, `decline(request,note)` and `deliver(request)`, qualified by
  the app name. Record arguments use the canonical `{id,version}` reference.
  The MCP model read is `NorthstarEquipment.EquipmentRequest.read`.
- Stable stored stages `waiting_manager`, `waiting_finance`, `fulfillment`,
  `delivered`, `declined`. Initially each new request captures
  `needs_finance=false`; manager approval leads to fulfillment, Operations
  delivers, and decline preserves its note. Nonpositive/non-EUR costs and
  missing title/reason must be rejected without a mutation. The home page
  renders current-authorized request/approval/delivery history.
- After the actual Finance request, only new amounts strictly above 100000 EUR
  minor units capture `needs_finance=true`. Add
  `approve_finance_route(request)`, `approve_finance(request)` and
  `decline_finance(request,note)`. Manager approval then leads to
  `waiting_finance`, Finance approval to `fulfillment`. Exactly EUR1000 stays
  manager-only. Keep the original model/storage identities and saved routes;
  the pending pre-change EUR1500 request retains its old route.

The protected JSON initially selects `policy: "manager"` and no baseline.
It fixes separate company and preview HTTPS origins at
`equipment.northstar.soda.test` and `preview.northstar.soda.test`, plus independent
Identity/State resource IDs. An operator must supply the intended canonical
origins before approval; the example names are the agreed local rehearsal
configuration, not guessed deployed addresses. The runner derives the existing
Canlang descriptor from the actual compiled artifact and installed public
Contracts version, preserving all declared capabilities. It invokes:

```sh
can check --catalog=/opt/canlang/lib/node_modules/@canlang/cloudflare/dist/native-catalog.json Northstar.can
can compile --format=json --native-scenario-receipts \
  --catalog=/opt/canlang/lib/node_modules/@canlang/cloudflare/dist/native-catalog.json \
  Northstar.can
can test --artifact ACTUAL_CAPTURED_ARTIFACT
can-soda build --artifact ACTUAL_CAPTURED_ARTIFACT --descriptor ACTUAL_DESCRIPTOR \
  --environment ACTUAL_COMPANY_OR_PREVIEW_CONFIG --out FRESH_BUNDLE
```

The integration then uses public `can-soda bootstrap`, `member` and `serve`
commands with fresh private mode-0600 credential files. It obtains real
presession/login cookies, team selection, page CSRF tokens and self-issued MCP
grants. The actual owner invokes `system.team.role`; no role or business record
is inserted directly into a database. Fixed nonzero loopback ports preserve
each bundle's HTTPS origin: the existing native gateway dispatches requests
at that origin. The Node HTTP client checks this host ingress boundary and
Secure/HttpOnly/host-only/SameSite=Lax cookies; it does not establish TLS
termination, browser behavior, Forge embedding or appliance deployment.

Once generated v1 has actually passed and its Git commit is known, prepare
Finance verification by changing the protected JSON to `policy: "finance"`
and `baseline_source_commit: "ACTUAL_GENERATED_V1_40_CHARACTER_SHA"`. Review and
reaccept that changed protected input through the same preparation owner before
adopting the new verification definitions. The baseline must be an available
ancestor of the candidate; the check extracts and recompiles that exact source.
It never rewrites the candidate or substitutes the reference source. It creates
a delivered EUR420 request and a pending EUR1500 request in fresh disposable
v1 state, stops v1, starts actual v2 on the same state, compares complete saved
records, and exercises the EUR1000/EUR1001 boundary and Finance denials.

Company and preview use independent fresh accounts/data. The preview starts
empty, rejects company credentials, runs its own synthetic business workflow,
and cannot change company records. Both runtimes must retire before the exact
checked transport bytes are written to `.soda/application/bundle.json` and
`.soda/application/preview/bundle.json`. Only the native owner can retain those
outputs and issue the actual check receipt. Nothing in this fixture deploys,
seeds or authorizes a company application. The separate installed browser
rehearsal must still prove the user-facing demo.

Run the real local integration from the SodaOS repository:

```sh
bun tests/fixtures/founder-demo/verify-reference.ts
```

It requires SodaOS's installed Bun/Playwright dependencies and Chromium,
OpenSSL, and a built sibling `../canlang` checkout. Build that checkout through
its owning package workflow before running; do not emit over another active
writer. The harness consumes the compiler at `compiler/target/debug/can`, the
Values compiler catalog, and the built Contracts, Interfaces, UI, State and
Cloudflare native runtime packages. It does not invoke a model or Factory run.

For a different existing checkout or compiler:

```sh
CANLANG_ROOT=/home/me/Projects/canlang \
CANLANG_COMPILER=/home/me/Projects/canlang/compiler/target/debug/can \
FOUNDER_REFERENCE_DIR=/home/me/Projects/founder-demo-runs \
bun tests/fixtures/founder-demo/verify-reference.ts
```

The output directory must be beneath `/home`. Every execution creates a fresh
private child, compiles v1 and v2 from source, generates a temporary local TLS
certificate, and uses a separate fresh application data directory. The default
output is `../founder-demo-runs/<timestamp>`. Existing company/application data
is never selected. The process closes Chromium and its native runtime on exit;
the output remains for inspection. Its account passwords exist only in memory,
so the retained directory is evidence rather than a reusable presenter login.

The browser mounts the actual `SodaApplications` component in a small HTTPS
shell. Its launcher inventory is a fixture; the application itself is the real
Canlang native workerd runtime at a separate same-site HTTPS origin. Real
application login, Identity accounts and owner-authorized `system.team.role`
grants establish the five people. The harness creates its initial owner and
ordinary accounts through Canlang's existing private test provisioning owner;
it does not qualify the public first-owner setup ceremony. Record inspection
uses self-issued MCP grants, and guarded mutations use the actual authenticated
HTTP operation routes.

The assertions cover:

- €420 entered in the generated form becomes exactly 42000 EUR minor units.
- A real manager approves it; Operations delivers it. The requester cannot
  approve, and an unrelated employee cannot read another person's requests.
- Feed/App switches and a full shell reload retain the record and login.
- v2 restarts against the same retained company data. The delivered €420 and
  pending €1500 records retain their exact IDs, versions, values and old route.
- New €1000 requests stay manager-only. New €1001 requests require Finance;
  bypassing Finance, early delivery and manager self-approval are rejected.
- The actual Operations history disclosures render the original request,
  approval and delivery after the update.

Successful output ends with `v2-retained-data-and-threshold-guards` and writes
`result.json` and `app-own-tab.png`. The v1/v2 sources, compiler artifacts, canonical `bundle.json` files and
retained data are in the same private directory. A compiler/package mismatch,
authentication failure, missing row or incorrect rule exits unsuccessfully.

A successful fresh-source run is recorded at
`/home/vince/Projects/founder-demo-runs/1792997395565/result.json`, with its
`app-own-tab.png` and exact canonical v1/v2 `bundle.json` bytes alongside it.
Those are the bundles consumed by that actual runtime check, including its
explicit local HTTPS origin. The form presents readable labels and defaults
the canonical currency selector to EUR; employees enter major currency units.
Use the harness above to compile and verify the
current fixture anew. This result does not qualify native OCI installation,
Factory artifact provenance, an isolated verified preview or the complete
installed founder journey. Those remain separate integration checks in the
founder checklist.

The separate [local production-container result](/home/vince/Projects/founder-demo-native-check/runs/1792998172044/result.json)
also runs the native fixed packaging and image-byte readback with these real
bundles, the production Containerfile, browser first-owner setup, production
member CLI, role grants and retained v1/v2 data. Its `v1/native-image.json` and
`v2/native-image.json` identify the distinct exact images. The existing released
tools archive concretely refused this bundle's release preflight, so this check
assembled the current production-only runtime closure with the owning packager;
it is explicitly local reference input rather than a released tools archive.

The [full native producer result](/home/vince/Projects/founder-demo-native-check/full-factory-application-result.json)
additionally exercises the whole `Factory.application_build` method: an approved
fixture check retains the actual 32,251,446-byte company and 32,251,950-byte
preview bundles, the real helper exports them, the native decoder admits the
bounded base64 envelopes, and fixed packaging reads back the exact image bytes.
Replaying the company build keeps its image without another helper export.
The integration commits the authored source and input hashes; its approved
check copies the external retained fixture bytes. It does not invoke generation.
Its [retained launcher](/home/vince/Projects/founder-demo-native-check/run-full-factory-proof.sh)
uses a private mount namespace so the native owner's fixed storage path is
isolated from host applications.

The [preview isolation result](/home/vince/Projects/founder-demo-native-check/isolation-runs/1792998511456/result.json)
checks an independently built preview origin, separate Identity/State resource
IDs, an initially empty actual app, refused company credentials and a new
preview-only request. Every retained company data path and byte hashes equally
before and after. The [full preview lifecycle result](/home/vince/Projects/founder-demo-native-check/preview-runs/1792998363026/result.json)
also checks the money, roles, update and history assertions above.

The retained local [container harness](/home/vince/Projects/founder-demo-native-check/verify-native.ts)
can run against its existing isolated store and runtime image:

```sh
NODE_PATH="$PWD/node_modules" \
FOUNDER_REFERENCE_DIR=/home/vince/Projects/founder-demo-native-check/runs \
bun /home/vince/Projects/founder-demo-native-check/verify-native.ts
```

Its companion `verify-native-preview.ts` and `verify-native-isolation.ts` use
the same local fixture. They create fresh private credentials/data and retain
private mode-0600 credential files for the production CLI. The task-owned HTTPS
gateway and host networking adapter exercise the actual production runtime;
they do not qualify appliance systemd, Caddy ingress, Factory F10 generation or
an installed founder rehearsal. The outer shell is deliberately small and is
not evidence of final Forge visual polish.

For the disposable VM candidate, a separate [canonical full local tools archive](/home/vince/Projects/founder-demo-tools-20261026/artifacts/canlang-release.json)
was assembled from a private current-source snapshot through the existing
`pack-tools.mjs` workflow. All 26 package-build tasks, current producer checks,
catalog emission, distribution manifest, 13 workspace artifacts and native
notice checks passed. The [extracted-archive check](/home/vince/Projects/founder-demo-tools-20261026/archive-probe/result.json)
then compiled the actual Northstar source with the packaged compiler/catalog
and bootstrapped its real retained bundle with the packaged production runtime.
This is a full local candidate input, not an upstream publication or VM result.

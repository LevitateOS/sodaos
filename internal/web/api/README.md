# api

Serve Soda's product HTTP and WebSocket operations through verified native
Forgejo extension authority. This internal Go package is a backend integration
library, consumed by [`web`](../README.md); it has no standalone application.

## Integrate

`api.New(cfg, db, forgejoClient, hostClient, authService)` constructs an `API`.
Supply admitted configuration, an open store and the existing service clients.
The facade then attaches the identity broker client and factory coordinator.
Use `API.ExtensionHandler()` on the dedicated private extension listener.

`api.Extension(socket)` constructs the extension-side HTTP bridge and returns a
handler plus a close function for its idle transport. The socket must be the
private Soda listener. The bridge admits a fixed route set, checks native
authority, bounds ordinary messages, and strips cookies and authority headers
from responses. WebSocket streams retain their separate framing contract.

`api.Frontend(forgeSocket, factorySocket)` composes two such bridges for the
required Fountain package. Factory-owned routes go directly to Factory; the
remaining admitted routes go to Forge. The close function closes both idle
transports. An unavailable product returns a transport error without sending
its requests to the other product. The optional Tailnet package uses the Forge
bridge alone.
Human issue/PR lock submission and scoped command lookup/cancel now pass through
this bridge to Factory’s admitted Product/Application. The lock submit paths
carry a bounded escaped-reason envelope; cancel keeps the ordinary body limit.
Overflow is rejected with 413 before forwarding. See the
[shared lock HTTP contract](../../factory/httpapi/README.md).
Direct issue/PR close/reopen and retained-command lookup/cancel now use the same
explicit route admission and Factory Product/Application composition. State
commands keep the ordinary 64 KiB envelope, and overflow is rejected before
forwarding. Exact boolean input, safe outcome projection, native/outer reply-loss
recovery and unfinished completion follow the shared HTTP contract. Current
native session is rechecked after delivery; cached receipts cannot bypass it.
PR inline-comment submit and scoped command lookup/cancel now use the same
Product/Application boundary. Only the exact submit collection admits a 512 KiB
escaped Markdown/anchor envelope; cancellation retains 64 KiB. Full required
anchors and boolean mode are validated by the shared HTTP owner. Thread GET
responses receive a bounded 2 MiB allowance so admitted full-length comments and
canonical anchor metadata can be read back; command responses keep the ordinary
limit. Native/provider evidence remains private, and session loss after primary
withholds the result without discarding the recorded effect.

## Available operations

The handlers provide:

- Repository selection, creation profiles, shared project creation and Join.
- Project membership, access keys, OS state, lifecycle and preparation controls.
- Spaces inventory, managed terminals and native WebSocket attachment.
- Provider enrollment, connection grants, leases and authorized identity launch.
- Factory policy/grants, acceptance, lifecycle actions, recorded state and
  read-only run output.
- Operator Tailnet settings and authorized project Tailnet operations.
- Application references and stored inventory, operator registration/install/image
  replacement, and explicit administrator Inspect/Start/confirmed Stop.

The exact methods, routes and request contracts belong to
[HTTP API](../../../docs/reference/api.md). Browser clients use Forgejo's native
extension API base and current mount context. The private `/api/...` paths are
service routes rather than independent public browser endpoints.

## Effects and constraints

All seven Factory product route adapters now bind `httpapi.Product`, the same
router used by `runtime.NativeApplication`. Fountain supplies one canonical
store/host/actor, audit principal, display author and the route's existing current
session check. Each adapter constructs only the service ports needed for that
request before shared routing, preserving lazy catalogue/preference reads.
Provider session/origin admission, native identity/navigation metadata, webhook
decoding, output upgrades and worker channels remain host-owned.

Post authority capture delegates source/author/revision checks, retained-proof
reuse and journal persistence to `service.IssueIntakeCapture`. This adapter supplies
current Fountain visibility, author-bound native issuance and proof encoding.
Private proof bytes never enter browser responses or semantic mutation requests.
Standalone connectors supply their own trusted issuance port to the same service.

Worker capability preparation delegates to `service.WorkerAdmission` through
typed runtime observations. The shared `factory/httpapi.Workers` boundary owns
the fixed worker route, Bearer/HMAC authentication and admitted scope/body checks;
this package supplies appliance admission and runtime/service ports to the shared
`factory/mcpserver.NewHandler`. That package owns all role-filtered SDK routing
and retains its shared rate limiter across authenticated requests;
the service/store retain current capability, process, assignment and stop gates.
Native composition can supply a native observer without appliance fields.

`factory/mcpwire` owns worker tool definitions, descriptions, bound envelopes,
typed result/current-context schemas and strict canonical payload contracts.
The shared SDK server consumes those definitions for Factory and System One discovery;
per-call admission, source reads, provider dispatch and settlement retain their
existing owners. Shared schema construction alone authorizes no operation.

System One tools delegate question validation, single/batch dispatch and
conversation recording to `service.WorkerConsultations`. The shared server retains
capability/rate admission, the full envelope bound and MCP response framing;
production supplies its existing provider adapter and store journal. The actual
answer/status is retained after disconnect, and recording failures never become
a successful tool response or an implicit retry.

`service.WorkerContext` owns admitted assignment/lineage loading, recorded bundle
selection, worker metadata projection and bounded run evidence. The shared MCP server
retains per-call admission/rate-slot lifetime and adds its route. Current-input,
checkpoint and reassessment adapters use the same sources and loaded facts.
Missing/foreign lineage stays incomplete; unresolved run evidence returns an
error without an empty successful collection.

Required-current reads and Test Pilot evidence joining use
`service.WorkerInputs.Read` for both context and mutation admission. The shared
wire renderer owns tool/result guidance, compact metadata and finite snapshot
pages. The shared server retains protocol framing, optional run-observation metadata
and hold/stop presentation. Valid verification never clears unrelated missing
allowance or other required inputs.

Recorded worker result fields, terminal mapping/catalogue checks and settlement/
checkpoint readback delegate to `service.WorkerResults`. The same retained task
receipt port supplies output binding and context. The shared server authenticates,
checks wire/required inputs and records submissions through the canonical store.
A valid shape or filled binding neither grants work nor proves native execution.

Builder/Fixer task commands delegate to `service.WorkerTasks` using typed
observation and task-owner ports. The handler retains MCP schema/discovery and
transport admission. Discovery uses the domain's captured-tool rule for both
runtime variants; missing native appliance pins cannot become dereferences or
invented identity. Task execution/replay remains with the runtime owner.

Planning check catalogues and Test Pilot evidence delegate to
`service.WorkerVerification`, with the existing controller's protected readers
supplied as ports. The service owns current scope admission and readback;
the HTTP layer retains bearer/MCP transport. Required context propagates admission
and malformed-evidence errors, while missing/failed checks retain explicit
unavailable guidance. No read executes checks or adopts verification results.

`factory/httpapi.Records` owns artifact downloads and human library browsing.
`apiFactoryRecordsHTTP` binds canonical actor/host and supplies current native
admission and visibility with only the selected service's source port. The old
defining handlers were removed. Artifact reads use `service.Artifacts` for
canonical current admission and verification of complete retained bytes.
Shared HTTP retains attachment,
sandbox/nosniff/no-store and range handling. Missing/invisible artifacts stay 404;
unverified content and unknown access/store failures are unavailable, with no
artifact bytes returned.

Human project-library search and item/history reads use `service.Library` with
current canonical session/repository admission. Missing items and confirmed
invisibility return 404; unknown access/store failures remain unavailable.
Browser repository scope is a string without changing retained library JSON or
the worker MCP response encoding.

Complete feed/post-detail assembly belongs to `service.Feed.Page`. Its current
canonical repository/session port is shared with catalogue admission. Only known
native invisibility becomes a hidden row; changed sessions and unknown access
remain errors. `factory/httpapi.Feed` owns catalogue/feed/subscription and post
submit/detail/replies/retry routing,
parsing and response/error projection. This package supplies canonical actor,
current native admission and repository/post ports through one route adapter.
Catalogue and preference requests do not construct unrelated post collaborators.
Post mutations receive the same store, bound display author and existing native
proof/context capture ports. Shared post transport never drives background work
or fabricates native authority.

`factory/httpapi.Requests` owns department/request catalogue, revision, history,
saved review, overview and post product-review transport. `apiFactoryRequestsHTTP`
supplies canonical identity, current native admission, administration and source
ports without duplicating these handlers. Product review still opens only its
exact draft. Planning/generation/exploration/status/walkthrough-review transport
also lives in `httpapi.Requests`; native code supplies available planning and
current chooser ports only for their command routes. Status needs no provider
construction and retains read-only recorded advice. Private request scope-history
transport also belongs to that shared handler.
Canonical command/private-not-found error rendering now lives in `httpapi` and
all native callers use it directly; provider authority errors stay host-specific.

`factory/httpapi.Work` owns recorded issue/journey, assignment and priority
comparison transport. `apiFactoryWorkHTTP` supplies canonical identity, current
native visibility/status/control/navigation ports and sanitized native errors.
Constructing or reading these projections never starts work.

`factory/httpapi.Repositories` owns status/access, activity, policy, runtime
grants/capacity, sponsorship and repair-allowance transport.
`apiFactoryRepositoryHTTP` selects current native service ports for the route,
binds canonical actor and separate command principal, and supplies fresh session,
visibility and sanitized error presentation. The native constructors retain
administration, operator and custody admission; shared services retain revision
CAS and command journals. Native defining handlers have been removed.

`factory/httpapi.Runs` owns run-status and repository/run intervention transport,
including canonical intervention error responses. `apiFactoryRunHTTP` supplies
current native ports, bound actor/principal and sanitized native error presenters.
Services retain execution authority, run scope, recovery-destination admission
and the coordinator's actual uncertain/pending outcomes. Native defining handlers
and their private lifecycle-error mapper have been removed.

`factory/httpapi.Acceptances` owns acceptance preview/admit/withdraw transport.
`apiFactoryAcceptanceHTTP` supplies bound canonical identity and current native
visibility/execution ports; the services and coordinator retain exact evidence,
replay, predecessor and withdrawal journals. Shared `AcceptanceDecisionError`
also renders issue-work read failures. Verification-allowance transport belongs
to `httpapi.Requests`, supplied by `apiFactoryRequestsHTTP` with current native
administration and a separate host-derived command principal. Old defining
handlers were removed without aliases.

`factory/httpapi.Output.Serve` owns the read-only output handshake, cursor/frame
pump, heartbeat, input rejection and connection/input-reader shutdown over the
shared bound attachment. This package retains native origin/channel admission,
upgrade, peer registration/draining, runtime read ports and exact native
repository/session-generation handshake validation. Host wire labels never
replace canonical attachment authority. Old protocol/pump definitions were
removed without aliases. The separate native command now consumes private Rust
bootstrap inputs and owns authenticated Unix readiness without importing this
package. Its native application also composes shared product groups over the
same `service.Application`, with current account binding supplied by connector
ports. Actual native account admission and reconciliation remain open.

`factoryRepositoryAuthority` translates fresh native session/repository facts
to `service.RepositoryAuthority`. It resolves the canonical actor back to the
current native account and rechecks the session generation after permission
reads. Read evidence never admits configuration; request/policy writes use an
explicit current-administration check. Evidence has a bounded envelope for this
single fresh check, with no session/permission cache. The shared service checks
resource, actor, source, connection/runtime scope, expiry and cancellation;
desktop adapters must provide their own current account/connection authority.

`FactoryApplication` supplies an application view over the same canonical store
and coordinator, with this host's current judgment and captured issue-effect
ports. `service.NewApplication` owns coordinator/background assembly and binds
continuation after provider ports are supplied; the API object does not drive
background posts or contribution assessments. Issue workflows and collaborator
readback use that same shared application assembly.
Judgment connection readiness remains live, so connection/withdrawal takes effect
without restarting. Request handlers only read collaborator availability.

Factory acceptance, dispatch and completion observation adapters belong to
[`forgejo/observation`](../../forgejo/observation/). Factory composition supplies
them directly to the controller; the HTTP package retains request admission and
response projection rather than defining native snapshot or call-budget behavior.

Requests can persist database records and invoke host, broker or native Forgejo
operations. Protected calls verify the live actor, contribution, repository
authority and session generation; mutations also check same-origin context.
Operator actions require the configured stable Soda operator ID. Displayed IDs
and browser-supplied labels do not establish authority.

Application administration uses a separate stable Forgejo administrator grant;
app-local users are not Soda administrators. GET inventory/detail is passive.
Explicit native actions recheck the live session, current reference revision and
container generation. Stop and replacement require confirmation. The current
replacement path keeps the exact artifact/bundle, produces a stopped container,
and retains prior Specs. Unknown outcomes clear readiness before native work;
GET never retries them. Data must already be provisioned by Canlang's private
bootstrap boundary. App and Forgejo hostnames must differ even when their ports
differ. No Canlang credentials or provider configuration enter this API/store.

The API maintains bounded terminal peer state. Call `CloseTerminals()` while
draining the server to cancel and join those peers. Native lifecycle and
credential retirement remain owned by the host and identity broker.

See [Projects](../../../docs/product/projects.md),
[Terminal](../../../docs/reference/terminal.md),
[Credentials](../../../docs/reference/credentials.md) and
[Factory](../../../docs/reference/factory.md) for the behavior these handlers
expose. [Trust](../../../docs/architecture/trust.md) owns their authority limits.

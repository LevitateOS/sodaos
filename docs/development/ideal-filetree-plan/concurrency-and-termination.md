# Concurrency, cancellation and termination

This source audit identifies four new ownership/admission defects and one
conditional release-cleanup risk. It preserves the completed deadline, capture,
PostgreSQL, HTTP and terminal-pump repairs. The [existing tasks](implementation-tasks.md)
and [lanes](implementation-lanes.md) remain the execution plan; the correction
packets below refine those owners rather than launch a new schedule.

Source: `94c40095f8202deca720cece4582e1d67d5f5544`, tree
`d9f4ecfc67b07a1dacc152e08a37c8a084d2ba18`, inspected 2026-10-07. Application
source remains `0b0734398f0350d75cc6fdb4dc8129251d8ab308`; the 12 pre-existing
dirty guidance files and 39 dependency inputs retain their recorded bytes.
This extends [workflow tracing](workflow-traces.md),
[observation reliability](observation-reliability.md) and
[library caller challenges](library-integrations/adapter-challenges.md).
Product/native authority remains in the owning guides and domain slices.

## Budgets and evidence

One operation deadline must cover its lock/admission wait, connection,
negotiation, partial writes, reads, buffered events and response matching wherever
that owner promises one deadline. Checking each syscall separately or resetting
a timer on progress does not establish that contract. Cleanup can have a separate
explicit allowance: expiry requires failure, custody and an honest cleanup result,
not instantaneous termination. Request cancellation does not undo an external
mutation, and killing a client does not establish that its remote operation ended.

Tracked discovery found 895 program/service files with 7,348 lifecycle-related
selectors, including tests and distributions. These are discovery seeds, not
895 fresh body reviews. Fresh connected review covers the defining owners and
named caller chains below. Selected upstream sources include Tokio 1.53.2,
Hyper 1.12.0, tokio-postgres 0.7.18, tungstenite 0.30.0, coder/websocket 1.8.15
and the pinned Go 1.26.7 standard library. Prior L08/L09 native/fixture receipts
are reused only for matching unchanged source and questions; they were not rerun.
No tests, builds, database/provider/VM operations or native qualification ran.

## Ownership and deadline ledger

| Actual owner and caller chain | Operation, cancellation and final ownership |
| --- | --- |
| Acceptance Phase → command/VM process and evidence finalizer | Child phases have the earlier finite parent/child deadline, including a bounded child of an unbounded parent. The process group has a reaper; leader observation pins PID/PGID until group cleanup. Stop has 10s TERM + 5s KILL allowance. Pumps join and both capture writers close before buffers become parseable evidence. Incomplete cleanup transfers custody and returns failure; it cannot certify success |
| Acceptance VM → QMP connect/negotiation/send/event/response → readiness/powerdown | One finite Phase reaches nonblocking connect, partial I/O and buffered-message matching; poll slices are at most 10ms, messages at most 4 MiB. Readiness child is 30s, powerdown child 2m. A failed exchange does not become a reusable successful session. VM close separately owns process stop, pumps and writers; **OBS-D01** remains the SSH readiness exception |
| CoreOS native curl → process capture → metadata parser | Native curl has a 60s cap, HTTPS redirect policy and caller body cap; the enclosing Phase also applies. Wait/stop, pump joins, both writer closes and final-flush errors precede metadata/buffer use. Unterminated metadata reaches its parser. Completed L01 stays completed; curl behavior was not newly exercised |
| `lib/unix-http::request` → Factory CLI, Identity HostClient, host BrokerClient and Tailnet local request | Per-call current-thread Tokio runtime and Hyper driver have one deadline established before runtime construction. Connect, handshake, send, headers and body share its work deadline; a small reserve covers driver abort/join. Body is capped before growth. Failure does not retry POST or retain the per-call connection. Four synchronous callers retain their own status, peer and operation policy |
| Identity Store/Tx → tokio-postgres client + owned driver | Store owns one persistent runtime and one mutex-protected optional connection. The 30s deadline starts **before** mutex wait; a transaction retains the guard and original deadline through BEGIN/statements/COMMIT or ROLLBACK. The 250ms reserve includes cancel/drain and discard/driver join. Timeout removes the session; uncertain COMMIT is surfaced without retry. Completed server SQL errors alone do not require discarding a still-open synchronized connection |
| Identity HTTP → blocking controller/provider callbacks → main joins | One runtime/server owner admits at most 64 connections and 8 blocking callbacks. Header 5s, body 30s/512 KiB and Store/provider budgets are staged; they are not one end-to-end deadline. Shutdown closes callback admission, cancels idle transports and joins admitted synchronous work; main joins both server threads. Provider child/mutex lifetime remains its own duty |
| Host mux → upgraded WebSocket → native terminal attach/output reader | Mux admits 128 connections/16 callbacks and retains separate callback/upgrade joins. Read-ahead transfers to one tungstenite pump. First frame 5s, output queue eight frames, stalled write 5s, close flush 250ms; expiry/shutdown cancels and joins the output reader. NativeAttach close has 3s child grace then kill/reap. H01-F3's independent reader repair is present; post-kill blocking wait still cannot certify a native stop bound |
| Go dashboard/API → terminal and factory-output peers → host WebSocket | Registry closes admission, cancels peers and waits for registered handlers; max peers 128. Handshake/heartbeat/frame writes use 5s child contexts, terminal-state operations 30s. Each attach owns its socket and native connection; control queue capacity is one. AfterFunc cancellation closes transports. Child input/control goroutines have cancellation rather than separate join receipts; no independent material leak was established. `http.Server.Shutdown` does not own hijacked sockets; keep this registry join |
| Go dashboard → Coordinator flock + SQL pool + admitted handlers | Coordinator Start is synchronous reconciliation, not a background loop. Ten-second context bounds nonblocking flock acquisition; Close releases it. SQL pool has 16 open/4 idle connections; transactions own their row locks until commit/rollback. **CON-G01**: coordinator/database lifetime does not currently encompass admitted HTTP mutations during shutdown |
| Go observer/publisher/reviewer/merge/check → one ServiceBackground | Calls share one revoking admission bootstrap and pinned installation. Dial/HTTP/body use caller context, bodies cap+one, every new connection checks native peer UID; only an authenticated 401 permits the one safe rebind/retry. **CON-G02**: bootstrap holds the ordinary state mutex through network I/O, so another caller's lock wait ignores its cancellation/deadline |
| Host project executor; Go acceptance Process; guest PTY/relay | Native executor deadline reaches nonblocking stdin/stdout/stderr and concurrent pumps; WNOWAIT prevents PID reuse before process-group cleanup. Go Process owns the group/reaper and 2s WaitDelay; Wait cancellation needs the actual caller's Stop. Guest relay polls at 100ms, attach readiness 5s, heartbeat 30s; TERM 2s then KILL/reap 2s reports uncertainty. S05-F1's writefds correction remains present |
| Muse host listener → per-connection worker → podman child/control supervisor | Listener owns accepted descriptors; request record/FD admission is bounded at 64 KiB with a 5s pre-request poll. **CON-M01**: detached supervisor signals a numeric PID after the request owner may reap it. **CON-M02**: no aggregate worker bound or completed-handle reclamation. Shutdown stops acceptance then joins workers; the main 35s retirement allowance does not cancel every active shell |
| Installer enrollment receiver/broker → admitted unit/cgroup → cleanup | Armed window at most 30s, broker lifetime 5m; single accepted connection, peer/unit/cgroup admission and bounded key reads. Cleanup gets a detached 10s allowance, stops socket/exact unit, and requires inactive/cgroup-empty checks before release. No fresh counterexample established; native attribution/teardown was not exercised |
| Browser terminal/factory attachment owners → request/socket/timer/xterm | Generation checks reject late callbacks after detach/rebind; AbortController owns requests, timers clear, sockets close and observers disconnect. Opening budget is 45s; explicit terminal End request 30s. Terminal outbound bufferedAmount guard is 64 KiB, rendered pending output 256 KiB. Dispose releases renderer/listeners; factory detach may retain rendered output. Hiding a view is presentation, detaching does not terminate native work, and an uncertain End is not replayed |
| Release Runner/container/isolated unit → pipes/runtime directory | Runner drains both pipes concurrently; metadata operation cap 120s, capture cap 16 MiB and post-exit drain grace 2s. D03-F1 remains repaired. Direct-child kill/reap does not certify container retirement. Worker cancellation attempts exact-unit stop within 30s and records failures; **OBS-W01**: outer runtime cleanup still runs when that stop is unconfirmed. OBS-R01's diagnostic byte/fragment bound is separate |

Detailed defining sources include the [Unix client](../../../lib/unix-http/src/lib.rs),
[Store](../../../cmd/soda-identity/src/store.rs),
[query cancellation](../../../cmd/soda-identity/src/pg_query.rs),
[Identity server](../../../cmd/soda-identity/src/http.rs),
[host mux](../../../lib/host/src/gmux_server.rs),
[WebSocket pump](../../../lib/host/src/dbackend.rs),
[native attach](../../../lib/host/src/terminal/native.rs),
[Go terminal registry](../../../internal/web/api/terminal_registry.go),
[project executor](../../../lib/host/src/project/executor.rs),
[Go process owner](../../../internal/acceptance/process.go),
[guest relay](../../../cmd/soda-project-terminal/src/pty_relay.rs),
[browser terminal](../../../frontend/spaces/sodaspaces-terminal.ts) and
[factory watch](../../../frontend/spaces/sodaspaces-factory.ts).

## New findings and correction packets

### CON-G01 — shutdown releases coordination before mutations finish

[beginShutdown](../../../cmd/soda-dashboard/main.go) calls CloseTerminals and
CloseCoordinator before closing any HTTP listener. The
[Coordinator](../../../internal/factory/control/coordinator.go) releases its
exclusive ledger flock while [operator](../../../internal/factory/control/operator.go)
and public API requests can still enter. An already admitted mutation can also
outlive the subsequent shared 20s Shutdown allowance: selected Go Shutdown closes
listeners and waits, but does not interrupt active handlers when its context expires.
The helper only logs that error. A second process can acquire coordination and
start reconciliation while the old process still performs host/broker/store work.
Row locks/CAS protect particular records; they do not preserve that owner lifetime.

Simply reordering Shutdown is insufficient. Stop and Reconcile can take 3m/10m;
[Project stop](../../../internal/web/api/lifecycle.go) deliberately detaches its
bounded admitted sequence from request cancellation. Server.Close also does not
join handlers. Fatal listener/startup returns must enter the same custody path.
Close all admission while retaining coordination, then cancel/drain/join admitted
work to its safe recorded/fenced outcome before releasing the flock/database.
Define the timeout path in these owners; preserve terminal peer cleanup and
uncertain-operation semantics. No general task framework is selected.

### CON-G02 — another bootstrap defeats the caller's deadline

[ServiceBackground bootstrap](../../../internal/forgejo/background_admission.go)
holds `b.mu` from admission selection through dial, HTTP body and publication.
[Other callers](../../../internal/forgejo/background.go) use ordinary Lock.
A shorter-deadline or cancelled caller cannot wake from that lock wait while a
longer bootstrap is stalled. This is a deadline/cancellation overrun, not evidence
of token corruption or duplicate operation effects.

Keep exactly one revoking bootstrap and atomic publication of admission/pinned
installation. Make waiting context-aware and keep ordinary state locks short;
check cancellation before proceeding. The existing SDK capability gate
**SIMP-SDK-1** is separate and does not block this current-owner repair.

### CON-M01 — supervisor can signal after child identity is released

[Muse shell_inner](../../../lib/host/src/muse/launch.rs) detaches a control thread
with only the child's numeric PID and a duplicated connection descriptor. The
request thread independently waits/reaps the child. A delayed control EOF/error
then causes unconditional SIGKILL; after reap that number can name a different
process. The duplicate can keep the supervisor waiting after serve_one returns.
No PID-reuse reproduction was performed.

Bind cancellation to child identity/lifetime using the existing Linux process
primitives or one serialized signal/reap owner. Join the supervisor and close its
duplicate descriptors before teardown is complete. Preserve disconnect cancellation,
credential return and uncertain cleanup; a newly detached reaper is not a proof.

### CON-M02 — connection churn has no aggregate custody bound

The Muse listener exposes a local mode-0666 socket under a mode-0755 runtime
directory. `MuseLaunch::serve` starts one thread per accepted connection before
request admission and stores every JoinHandle until shutdown. Silent clients
occupy workers during the 5s request wait; connection churn grows retained handle
bookkeeping throughout uptime. Backlog bounds pending accepts, not admitted
threads/handles. Completed threads have exited; no exhaustion threshold is claimed.

Bound admitted workers and reclaim completed handles during uptime in the same
listener owner. Preserve authorized clients, same-FD/ancillary/peer checks and
request cap/timeout. Shutdown must still cancel or finish and join admitted work.

### OBS-W01 — runtime release proceeds after an unconfirmed unit stop

[run_worker](../../../lib/soda-release-tools/src/worker/execution.rs) reports
exact-unit stop failure/timeout, then returns failure/cancellation.
[run_build_worker](../../../lib/soda-release-tools/src/worker/runtime.rs) still
calls release_attempt_runtime, which attempts recursive removal of the claimed
tree bound as WORKER_RUNTIME/XDG_RUNTIME_DIR. The unit's actual terminal state is
unknown. This source-established ordering creates a conditional custody risk;
it does not prove an active unit, successful removal, data loss or bind-mount effects.

Release the attempt runtime only when dispatch never occurred or exact-unit
termination is confirmed. Preserve it and the cleanup error on an unconfirmed
stop. Establish the bounded native observation needed for that confirmation in
the existing worker owner; no unit adoption/recovery service is selected.

| Packet / single accountable owner | Scope and prerequisites in existing plan | Acceptance checks for later implementation |
| --- | --- | --- |
| **CON-G01 / B**, dashboard H01/B03 lifetime | Dashboard main/server admission, coordinator and admitted operation completion. Specify safe completion/cancellation for existing detached mutations and every run return. Completed structural work stays complete | All listeners stop admission before ownership release; hold an admitted mutation across graceful-drain expiry and show second Start cannot overlap it; database remains owned until its completion; shutdown errors propagate; hijacked terminal peers still cancel/join |
| **CON-G02 / B**, shared Forgejo owner/B06 | Bootstrap/state locking and real shared consumers; preserve single admission, per-dial peer/pin policy and safe 401 behavior. No SDK upgrade prerequisite | Stalled bootstrap with shorter-deadline and explicitly cancelled waiters returns each at its own context boundary; one bootstrap/rebind publishes one atomic valid pair; no competing revocation or uncertain POST replay |
| **CON-M01 / A**, host Muse/H01/A07/L12 | Control supervisor, child and descriptor custody. Select identity-safe signal/reap ordering before repetitive plumbing | Disconnect still cancels the owned child; delayed supervisor return cannot signal after child identity release; all thread/FD exits are joined/closed; teardown failures preserve credential/lease uncertainty |
| **CON-M02 / A**, host Muse/H01/A07 | Listener admission and completed-worker bookkeeping. Choose finite aggregate worker budget while preserving valid local request/FD admission | Silent peers cannot exceed the chosen admitted budget; churn reclaims completed handles during uptime; cap/timeout/credentials stay enforced; shutdown joins every admitted worker |
| **OBS-W01 / C**, release worker/D03/L12 | Worker execution/result and runtime release. Establish exact-unit terminal-state confirmation; distinguish never-dispatched work | Stop failure/timeout retains the runtime and both primary/cleanup errors; confirmed unit completion permits removal; successful cancellation cleans only after confirmation; no string-based success inference |

These rows specify consequential work for Luna medium implementation/review;
settled repetitive plumbing can use Luna low afterward. Coordinate CON-M01 and
CON-M02 through one writer of launch.rs. The coordinator owns tracked docs,
explicit-path commits and any later manifest/lock or resource-heavy checks.

## Preserved repairs, open profiles and completion

L01 Phase/QMP/CoreOS capture, L08 transaction/cancel/discard, L09 HTTP/upgrade/pump,
L12 process custody, D03-F1 concurrent drain and S05-F1 PTY readiness remain present.
The historical [H01 review](reviews/H01.md) predates that adoption and those repairs.
F1's retired callback decoder transfers complete framing to Hyper under L09,
including its historical chunked-body check; F2 executor and F3 shared-reader
repairs are present. These are not current open defects at the inspected paths.
This does not advance H01's whole-slice pin or establish installed retirement.
**OBS-D01**, **OBS-G01** and **OBS-R01** remain
open under their [existing correction owners](observation-reliability.md#prerequisites-and-acceptance-for-later-authorized-repairs).
B03.C body admission and L16.G secret collection remain independent concerns.

Some waits need a product/native profile before a shorter bound is selected.
Host terminal setup may wait until the admitted domain/session expiry (up to 12h)
before its next shutdown check; the server retains and joins this work. Codex's
small writes precede its response timer. Muse provider close can wait on a child
mutex if the pinned CLI closes stdout while staying alive; that lifecycle is
unverified. Compose/native remote commands and asynchronous setup/check jobs have
command/job-completion profiles rather than demonstrated request deadlines.
Go startup/schema initialization uses Background contexts; public/private body
budgets are not implied by ReadHeaderTimeout. Define required stop/admission
horizons in those owners before imposing caps or detaching tasks. Native Git
descendant cleanup, post-kill reap, Podman/systemd and SSH remote effects remain
qualification limits; absence of fresh execution is not itself a new defect.

Three Luna medium primaries and independent medium cross-challenges covered
transport/Store, native/guest/provider/install and evidence/release; the coordinator
traced Go/dashboard/browser owners. Challenges accepted the four source defects,
narrowed OBS-W01, confirmed repaired terminal ownership and rejected a speculative
standalone WebSocket goroutine defect. Source coverage and workflow tracing are
complete for the recorded owners/questions; independent challenge and exact
existing-owner correction scopes are recorded. Native behavior and the held
product horizons remain unresolved only for their dependent changes. The full
desired tree and historical 80-slice review are not regenerated or advanced.

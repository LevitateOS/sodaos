# File, descriptor and process custody

Four current source gaps need narrow corrections: Identity settings are read
before their existing cap; the Project snapshot checks a pathname separately
from the file it hashes; enrollment close can hide the primary uncertainty;
and Butane conversion claims output removal without checking it. A dormant Go
source-bundle method has a separate retention/deletion decision. Completed
same-descriptor, confinement, process and evidence repairs remain preserved.

Source: `1d8c4e111dfe0fd9c64aed0726b768b244f27ccd`, tree
`e40a495ce9d778f03b5624089ced1d02903593ad`, inspected 2026-10-07. Application
source remains `0b0734398f0350d75cc6fdb4dc8129251d8ab308`; the 12 pre-existing
dirty guidance files and 39 dependency inputs retain their recorded bytes.
This extends [workflow tracing](workflow-traces.md), [authority/state review](authority-and-state.md),
[resource bounds](resource-bounds.md) and [concurrency custody](concurrency-and-termination.md).
The [existing tasks](implementation-tasks.md) and [lanes](implementation-lanes.md)
remain the execution plan; these packets refine their defining owners.

Discovery found 313 custody-selector files and 1,141 hits across application,
tools, scripts and service roots. These are search seeds, not 313 fresh body
reviews. Three Luna medium primaries, independent cross-challenges and coordinator
tracing inspected the connected owners below; unchanged earlier evidence is reused
only for its matching source, contract and question. No tests, builds, database,
provider, VM, network or native qualification ran. No observed race, failed unlink,
OS wait/clone failure or exhaustion event is claimed.

## Custody ledger

A size check must apply to bytes consumed, and permission/type/identity checks
must describe the descriptor used. Decide leaf and ancestor symlink policy
separately. A confined child open does not prove the original configured root
cannot be replaced. One open file is not an atomic content or whole-tree snapshot.
Borrowed descriptors stay with their owner; transferred/cloned descriptors need
an explicit close/join owner. EOF, child exit, pipe drain, native termination and
confirmed cleanup are different observations. Preserve the primary failure and
cleanup uncertainty without turning a known external mutation into retry authority.

| Actual workflow / defining owner | Current ownership, checks and error path |
| --- | --- |
| Acceptance private input → hash/record/evidence | `private_file` opens no-follow/nonblocking/CLOEXEC, checks opened regular type/private mode and reads cap+one. OwnedDir traversal opens no-follow components relative to held directory FDs; hash_at compares pre-stat and opened identity before streaming. These completed L12 duties remain; standalone hash_file still has a path stat/open seam under its local caller profile |
| Go config / Secret; Rust setup/Identity secret | Go config reads cap+one (64 KiB) on one FD before single-object/EOF decoding. Secret/setup intentionally follow a leaf symlink, check the opened target and apply 64 KiB cap+one; group sharing is permitted by their secret policy. Stronger grant-key Lstat policy followed by Secret reopening is a separate conditional binding seam. Identity settings startup is **ID-CFG-01** below |
| Project root snapshot → mutable account files → hash/metadata JSON | `Entry::snapshot(contents=true)` lstat checks link/type/512 MiB, then opens by pathname and hashes to EOF. Returned metadata can describe a different file; the fixed 1 MiB hash scratch does not grow with content. **ACC-SNAPSHOT-FILE-1** governs stream consumption and coherent observation, not heap growth or an atomic filesystem snapshot |
| Terminal/Muse configuration and private state → native helper | Terminal private records use held directory FDs and no-follow traversal; record reads use cap+one before decode. Muse config-copy leaf symlinks are deliberately followed with admission on the opened target. Preserve actual policies rather than imposing universal no-symlink rules. Original FD, child-inherited copies and supervisor duplicates remain distinct; **CON-M01/M02** own Muse signal/reap and worker custody |
| Acceptance process / VM / CoreOS → pumps / redacting writers → finalizer | Owned reaper observes leader exit without releasing PID/PGID, sends SIGKILL to the remaining process group and reaps waitable children, recording errors; this does not certify every non-child descendant stopped. Pumps join, both writers close and final-flush errors precede capture parsing. Unterminated metadata reaches its parser. Expiry/failed cleanup returns failure or transfers custody; L01/L12 remain complete. **OBS-D01/G01** retain their separate observation/Go probe duties |
| Installer enroll-key → session/unit/key publication → close | Root/Linux, installed FCOS, SELinux and interactive admission precede the actual path. Key directory/temporary publication retain no-follow component confinement, lock, identity, permissions, sync and uncertainty. Close stops exact owned units with detached allowance and requires terminal/cgroup evidence before removal. **ENROLL-CLEANUP-RESULT-1** concerns combining operation and close results, not changing those native checks |
| Importer payload / OCI; installer media/OCI → validation | Import payload reads 4 MiB cap+one and compares opened regular dev/inode to path admission. Import OCI checks components and opened identity, streams exact size/digest and retains only bounded JSON. Its fixed installed root is admitted by path; installer OCI child traversal uses openat/no-follow. Do not infer root-FD anchoring or gzip verification from raw-blob hashing |
| Release Root / delivery builder root → child file/hash → output | Confined roots own directory descriptors, and child opens/hash_at bind opened identities; initial selected roots have local build profiles. Archive/sys hash_file paths retain stat/open seams under candidate/attempt-produced input profiles. Shared OCI scanner drains decoded EOF while gzip decoder lives and checks raw descriptor hash; historical D05 EOF defect stays closed |
| Release Runner / worker → process result → runtime removal | Concurrent pipe drain, post-exit allowance, capture errors and child wait remain in their current owners. Direct-child exit does not certify container/unit termination. **OBS-W01** still gates runtime removal after unconfirmed exact-unit stop; **OBS-R01** governs diagnostic retention independently |
| Butane CLI → private source/output FD clones → native child/finalizer | Exclusive output create is mode 0600 beneath admitted private parent; child receives FD clones. **CUST-C-BUTANE-CLEANUP-1** owns post-create early exits and false removal reporting. Path-private pre-stat/open and try_wait error branches are conditional seams; the native timeout's kill/wait attempts are not a confirmed descendant-cleanup receipt |
| Go publisher → private Git workspace → registered push/lookup | Candidate bytes admitted at 4 MiB before write, MkdirTemp workspace/private root, scrubbed Git config, object/credential scan limits. Child Wait follows scan; scan failure cancels. askpass refers to external token path rather than copying token bytes. RemoveAll errors are ignored and ValidatedRepo.Close returns no result; retained scratch is a local cleanup/profile question, separate from observed remote operation identity/outcome |
| Dashboard/Tailnet → file lock, sockets, policy files | Coordinator owns opened flock descriptor until Close; filelock waits context-aware on nonblocking flock. Operator listener binds after exclusive coordinator start; extension listener refuses live/non-socket occupancy under its sticky IPC parent. **CON-G01** owns premature coordination release. Tailnet policy uses anchored roots/owned components and same-directory publication; selected root/ancestor policy stays with that owner |
| Setup / PostgreSQL backup/restore / served build file | Setup validates/caps opened credential target; published config owns its uncertain remote revoke separately. Backup uses private temporary stage, synced dumps and no-overwrite publication; failed-stage removal ignores errors while preserving primary failure. **O05-F1** owns retaining the just-published run. Restore closes stdin before wait, requires delivery plus child success and surfaces exact-stage cleanup failure. Rootfs server allows exact leaf names, opens O_NOFOLLOW and checks regular type on that FD; ancestor trust remains its configured-root profile |

Defining sources include [private inputs](../../../tools/acceptance/src/files/inputs.rs),
[directory owner](../../../tools/acceptance/src/files/owned_directory.rs),
[Go configuration](../../../internal/config/config.go),
[Project snapshot](../../../tools/acceptance/src/project_state/files.rs),
[process reaper](../../../tools/acceptance/src/process/owned_process.rs),
[terminal files](../../../cmd/soda-project-terminal/src/fs.rs),
[enrollment session](../../../cmd/soda-install/src/enroll/session.rs),
[installer OCI](../../../cmd/soda-install/src/oci/layout.rs),
[import payload](../../../cmd/soda-image-import/src/payload.rs),
[release confined files](../../../lib/soda-release-build/src/confined_files.rs),
[Butane artifacts](../../../lib/soda-release-tools/src/artifacts.rs),
[Git workspace](../../../internal/forgejo/publish/publish.go),
[backup](../../../cmd/soda-pg-maintenance/src/bin/soda-pg-backup.rs)
and [restore](../../../cmd/soda-pg-maintenance/src/bin/soda-pg-restore.rs).

## Four correction packets

### ID-CFG-01 — existing settings limit follows whole-file allocation

[Identity startup load](../../../cmd/soda-identity/src/main.rs) uses fs::read,
then strict decode with the existing 1 MiB MAX_DOCUMENT. Startup errors already
propagate before provider/Store construction; the refusal comes after allocation.
Open once and read at most cap+one before decode. Keep strict schema/error behavior
and existing path policy; no new product maximum is needed. No unprivileged settings
writer or exploited allocation was established.

### ACC-SNAPSHOT-FILE-1 — content admission does not bind the hashed file

The actual [Go lifecycle caller](../../../internal/acceptance/lifecycle_state.go)
executes the root-run remote Project snapshot. Account homes/checkouts are
user-owned and mutable; source therefore establishes a concurrent writer profile.
Those accounts also have wheel/sudo: this is observer fidelity and bound enforcement,
not a privilege-escalation claim. A leaf swap can follow a symlink/different object
after lstat, including a blocking FIFO/device; growth can exceed the 512 MiB stream
limit. Output contains hashes/metadata, not raw file bytes. Snapshot errors already
propagate to the remote nonzero exit; static symlink tests do not prove race admission.

Admit a no-follow/nonblocking regular file through one FD; derive metadata and hash
from that admitted object and enforce cap+one while streaming. If retaining a
prelookup identity promise, compare the opened identity too. Preserve declared
ancestor policy; do not promise immutable content or a whole-tree atomic snapshot.

### ENROLL-CLEANUP-RESULT-1 — close replaces the operation's uncertainty

`arm_enrollment` saves publication/result-wait's Result, executes `session.close()?`,
then returns the saved result. A close error replaces any primary error, including
EnrollUncertain. The CLI renders only the returned error, losing the explicit
warning that the authorized-key import may already have completed. Close's own
preserve/inspect advice does not include that operation fact. Both failures are
source branches; no native enrollment or double failure was exercised.

Combine primary and cleanup results at this owner. Preserve safe primary warning
and control-flow identity where consumed, plus cleanup uncertainty. Successful
operation with failed close remains an error; failed operation with successful
close retains its original error. No automatic retry/recovery is selected.

### CUST-C-BUTANE-CLEANUP-1 — output custody and removal claims diverge

On conversion error, `finish_butane_conversion` ignores remove_file's result yet
returns text claiming partial output was removed. The live convert-butane CLI
returns failure, so this is false cleanup reporting rather than overall successful
conversion. Also, after exclusive create, dest.try_clone can return before the
finalizer owns cleanup; the created pathname remains on that OS failure branch.

Own output disposal from successful create through all later failures. Keep
conversion diagnostics sanitized, report confirmed removal versus failed/unconfirmed
cleanup truthfully and preserve both facts. Preserve private/exclusive creation;
do not overwrite another output or add a recovery service. No unlink/clone failure
or native Butane run was observed.

| Packet / single accountable owner | Scope and prerequisites in existing plan | Acceptance checks for later implementation |
| --- | --- | --- |
| **ID-CFG-01 / A**, Identity startup A05/L12 | load's one-open bounded read before unchanged strict decoder; existing 1 MiB profile is settled | Exact cap reaches decode; cap+one/growth refuses before oversized retention/decode; read errors fail startup; no second pathname open introduced |
| **ACC-SNAPSHOT-FILE-1 / C**, H06 acceptance observation/L12 | Content-hashed Entry branch and actual remote caller; mutable account profile established in source. Preserve snapshot schema/failure taxonomy | Leaf symlink/FIFO/nonregular refused without blocking open; returned metadata matches admitted file; cap+one streaming refusal on growth; ordinary admitted files hash correctly; no raw bytes exported or atomic-tree claim |
| **ENROLL-CLEANUP-RESULT-1 / C**, installer O07/L12 | arm_enrollment result/close join and current safe Error rendering. Check consumed sentinel semantics before choosing representation | Failed operation plus failed close exposes both; import-may-have-completed warning survives; success/failed close is failure; failed operation/successful close preserves primary; native stop uncertainty still retains custody |
| **CUST-C-BUTANE-CLEANUP-1 / C**, release artifact CLI/L12 | Output lifetime beginning at exclusive create, clone/finalizer failure paths and sanitized cleanup result; no product decision required | Clone/run failure attempts owned cleanup; forced unlink failure cannot claim removed; confirmed removal reports accurately; primary failure remains visible; normal success/existing-output refusal/private modes remain |

Use Luna low for ID-CFG-01 and the settled enrollment result join after the error
representation check; Luna medium for snapshot and Butane custody and independent
review. The coordinator stages explicit paths and owns shared guidance/resource-heavy
checks. This audit specifies corrections without authorizing their implementation.

## Retention decisions and conditional seams

**CUST-G-SOURCE-1 / B** is a method-level bound gap in
[Config.Source](../../../internal/forgejo/publish/source.go): native source.bundle
is fully read before its documented 4 MiB check. Search found no current production
caller of Source. Assess that exact method's retention/deletion with L18 first;
sourceRepository remains live through branch/native observation and must not be
deleted as a group. If Source is retained with a real caller, apply one-open cap+one
and visible overflow refusal. Acceptance is either deletion with all actual references
resolved or bounded retained read with its caller identified. No active publication
allocation/exhaustion claim or new source API is justified.

The following remain named custody duties rather than established hostile-path
defects: Go/Rust GrantKey's stronger Lstat policy before broader Secret admission;
publisher token metadata before askpass reopening; Codex executable hashing before
pathname execution; local release/archive/Butane stat-then-open; importer installed
root checks; snapshot workload credential reads without a settled size profile;
and Muse maintenance config sizing. Establish an actual mutable actor or input
contract at A/B/C's respective owner before selecting stronger path/size policy.
Leaf no-follow cannot substitute for ancestor confinement or same-file checks.

For **C**'s run_butane, try_wait errors return after spawn without an explicit
kill/reap guard; timeout attempts ignore kill/wait failures. Actual OS wait failure
or surviving child was not established. Retain this conditional native cleanup
question at the helper, preserving custody/error observations if that branch is
repaired; no generic process framework is selected. **B**'s publication Git direct-child
CommandContext similarly does not prove descendant retirement. Private Git scratch
and failed backup-stage removal suppress errors; settle required local cleanup
diagnostics/retention separately from known remote results and original failures.

Existing **CON-M01/M02**, **CON-G01**, **RES-I-PROBE-OUTPUT-1**, **OBS-W01/G01/D01/R01**,
**AUTH-I-ACQ-1/CLOSE-1**, **O02-F1/O05-F1** and evidence-finalization findings keep
their established scopes and acceptance. Completed L01/L12 and D05 EOF work is
not reopened. Recorded source coverage, caller tracing, independent challenge
and correction allocation are complete for this chapter's scope. Conditional
producer/native questions remain visible; historical full-slice validity and
parked restructuring checkpoints are not advanced.

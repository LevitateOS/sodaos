# RC6 qualification record — 25 October 2026

This is the retained account of the installed testing behind
`v0.1.3-rc.20261025.3`, including failures and the next safe action. It exists so
another agent can find and resume this work. It does not change the acceptance
contract in [Testing](testing.md), the [release workflow](release.md), or the
[remaining-outcome catalog](factory-remaining-tasks.md).

**Status: incomplete.** Native installation, the private application journey and
native cold recovery passed. Docker installation and the browser terminal journey
passed, but its full Factory, recreation, cold-recovery and qualification chain
has not passed. Promotion and both `latest` pointers remain held. ARM and upgrade
qualification are outside this selected fresh-install-only candidate
(`upgrade_from: []`).

Times below are UTC unless an offset is shown. Results describe actual executions,
not merely prepared commands. Paths beginning with `$RUN` are local to the
retained workstation:

```sh
RUN=/home/vince/Projects/.sodaos-rc-20261023
SOURCE=/home/vince/Projects/sodaos-rc-final-20261025-rc6
PREPARED="$SOURCE/.artifacts/releases/prepared/rc20261025-final-r1"
```

## Source and public bytes

| Input or result | Exact identity |
| --- | --- |
| Published Soda source | `f1fc05068464674a476258d4bdbcc6a49fb139ec` |
| Equivalent product fix on the working checkout's different parent | `d7d54a5d`; accepts check-only Factory tasks and permits retry of uncertain terminal retirement |
| Qualified Fountain input | `a70dbdd46322342d8b91b451f54fee798898dd06` |
| Public immutable prerelease | [v0.1.3-rc.20261025.3](https://github.com/LevitateOS/sodaos/releases/tag/v0.1.3-rc.20261025.3), release ID `423443972`, published at `14:11:36Z` |
| Main CI | [40199684738](https://github.com/LevitateOS/sodaos/actions/runs/40199684738), passed on the exact published source |
| Deep CI | [40199684821](https://github.com/LevitateOS/sodaos/actions/runs/40199684821), passed on the exact published source |
| Candidate SHA-256 | `319dc8305a6548cdb8b0a6f60b1df4d3d6557bcbfd32200c12e4755168e83812` |
| Payload SHA-256 | `76e37362e602c0130389d56191ec7c9b6b86b226fc88af927f8bc77e63ab4816` |
| Release manifest SHA-256 | `aea6dbc77131ca50a8471c92a25fed2b36afd68c9806ec007ab68d0a9dff1d5c` |
| ISO SHA-256 | `994b5c239977621707c29279e6901c47af43618c88f22c7c3d8fe50ea9aebf64` |
| Docker manifest | `sha256:3b523309f88f65a91fc13e9f3656dd7b90829ff7e05d4783e0b74cdc714d54d2` |
| Docker config | `sha256:f6ec0b202151491c9ee719e7d9809a5398ad8d950bd86054fff7408aa3a11bc5` |

The production build, independent check and preparation completed once for these
selected bytes. Publication uploaded 19 assets and five images. Anonymous
verification downloaded all 19 assets (1,672,476,112 bytes) and checked their
identities. Its successful directory is
`$RUN/public-rc6-downloads-20261025-net`; the similarly named directory without
`-net` retains a failed sandbox DNS attempt and must not be substituted.

The download result SHA-256 is
`8cdc1d3e14568656d7bac68d8e6b849a50885804bc695d951f9986e8dc76ec33`.
The public release remains **UNQUALIFIED**: successful publication and downloads
do not establish the installed Factory result.

The working checkout also retains newer logo and Fountain commits. They are not
silently included in this frozen release. Do not reset that checkout to recreate
the release; use the selected source and its recorded inputs.

## Executed journeys

| Journey | Result and scope | Retained evidence |
| --- | --- | --- |
| 27 focused model-free checks | Passed; do not replay merely because later collector work failed | `$RUN/r56-appliance-app-fixture/handoff.public.json` contains `checks[]`, `checkCount: 27` and the native-proof binding; component evidence, not the installed Factory result |
| Fresh native disk installation | Passed on a newly created 96 GiB disk; installed source, payload and six shipping images read back | `$RUN/rc6-public-vm-r3/owner-handoff.public.json` |
| Native application | Passed: actual owner registration/login, private credential handoff, record write/read, normal stop, native reboot and same owner/records | Protected native observations directory below |
| Native qualification API | Passed, caller handle `59202` exited 0 | `$RUN/r56-public-consumers/qualification-api/appliance-qualification-r1-result.public.json`; `$PREPARED/appliance-qualification.json` |
| Native cold recovery | Passed from separate offline disk and UEFI copies; distinct boot, same protected state/images and owner records; clone then stopped and network retired | `$RUN/r56-cold-restore/appliance/journey-result.public.json` |
| Docker public installation | Passed: actual HTTPS installer, anonymous image pull, ordinary owner setup and trusted HTTPS | `$RUN/r56-public-factory-fixture/docker/unpaid-setup-handoff.public.json`; original installer evidence retained by the collector |
| First Docker collector | Failed before Factory/model work, at `read_reserved_terminal`; installation passed | Original Docker authority below; caller `21934` exited 1 |
| Docker existing-terminal continuation | Failed/incomplete: attach, shell, same-process reload and HOME witness passed; investigation and implementation finished and a PR was published, but review/merge did not finish | New continuation authority below; caller `18965` exited 1 normally, with no finalized seal |
| Docker recreation / cold recovery / qualification | Not yet passed | Prepared cold lane and qualification caller remain held |

### Native target and proof

The successful installed VM is `sodaos-public-rc6-r3-20261025`, UUID
`e0fe36b4-fedd-4492-80ac-3f87ad8c60d3`, recorded address `192.168.2.163`.
Its guest hostname is `sodaos-rc6-native-r3`; the hostname and libvirt domain are
different identifiers. The owner handoff SHA-256 is
`410705c6e02edc2694776790d525bc719e5a72b86764d1cf7cd3fda5e7f865a2`.

The original sealed native observations are:

```text
/home/soda-rc-public-appliance-collector-authority-20261025-rc6-r3/outputs/actual-installed-app-journey-rc6-r3/observations
```

The selected application collector SHA-256 is
`dc796a3f4d788be5465fb850fea4a94a236ab47e2db17cac97176e8eba09e422`;
caller `35187` exited 0. The cold proof SHA-256 is
`7ee99e200c92c8cca14d07fd0f2729610b9d30df8843d195fcc75d07f734b0b1`.
The cold clone UUID is `b2ed3377-6ba9-483a-81de-6f64662b2524`.
Both original and clone were powered off after the completed native journey.

Native installation evidence includes
`$RUN/rc6-public-vm-r3/installation/console-install.json`, SHA-256
`f8264808a7bc891410f90acbdfb704a8422ea9e74ee04867e79b94bb1ef2486b`.
Actual operator commands included `/usr/libexec/soda/soda-install enroll-key`
and `/usr/libexec/soda/soda-install configure`. The temporary public-key import
window closed; the initial bootstrap token was revoked and its local file
removed. Only the public CA was copied for client trust.

The local native cold sequence was `bind.ts appliance`, then
`workflow.ts appliance capture`, isolated network start,
`workflow.ts appliance clone`, native and browser readback,
`workflow.ts appliance final-off`, and network retirement. These are the retained
scripts in `$RUN/r56-cold-restore/`, not commands shipped in the product.
The resulting appliance qualification receipt SHA-256 is
`4589be37b4b1327edd592d9cb0fd6ed4fb65f4363b16a1bece869e1d27e4a2ce`.

### Docker target, account and limits

The Docker VM is `sodaos-public-docker-rc6-20261025`, UUID
`e2959f9f-1271-449c-b123-38e89dc17937`, recorded address `192.168.2.161`.
Use the UUID and current native readback to establish identity; an address alone
does not identify a fixture.

Its real fixture is repository 1, Project `p67c460cb0621cd06f556915b`, issue 1
(“Reject an empty greeting name”), initially at source
`5c451bc948168bf71ff4cc80464634cebd276e3e`. Coder and reviewer standing setup
both passed. Five nonadmin native actors, protected provider configuration and
hooks were configured before account enrollment.

The user's confirmed Codex enrollment completed. The current recorded connection
is `225c002d1e7eafefdb348f0df945bfd0`. It enrolled at generation 1 and was Ready
at generation 3 in the final `16:52:44Z` readback. Do not generate another device
code merely because a collector failed: first inspect the existing connection
through the supported API. Earlier codes/enrollments expired on older attempts;
they are not current credentials.

The approved test has 15 sponsorship minutes, 15 rolling model minutes and one
active run, without automatic refill, retry or extra inquiries. The first failed
collector consumed no model work. The continuation used the same original test;
investigation and implementation completed, and seven sponsored minutes had been
accounted at the `16:43:26Z` snapshot. This is a dated observation, not permission
to assume that balance remains available.

## Failures and what they mean

These attempts remain failed. Their outputs must not be relabeled as successful
release evidence.

| Attempt | Actual failure / correction | Preserved boundary |
| --- | --- | --- |
| Early native collector | Hardcoded older VM domain/disk caused argument validation failure before guest mutation | Corrected only the fixture binding in a separate collector |
| Next native App attempt | Immediate read raced the UI's automatic inspect; the generated owner's private handoff had not yet been saved | Old stopped App/VM preserved; no database repair, password substitution or claim of owner recovery |
| Successful native r3 | Synchronize parent idle plus actual stopped state; save the owner's protected handoff before the fragile assertion | Fresh install and full owner journey, rather than reclassifying the preceding attempt |
| First native qualification invocation | Used the parent output directory, which lacked the seal | Failed on the initial read before writes; preserved journal, corrected to `observations`, new invocation passed |
| Docker collector `21934` | Failed at `read_reserved_terminal`; request existed but response receipt did not | Exact underlying response/read/save exception was not retained; do not claim a proven product API defect |
| Docker terminal diagnosis | One diagnostic used an undefined API base | Preserved the diagnostic error; corrected read-only native/UI probes both read the exact terminal successfully |
| Docker continuation | Collector publication identifier mismatch found while waiting for review | See the source diagnosis below; no honest full Factory pass yet |

The first Docker attempt left exactly one managed terminal,
`a425bf8ac8f907bef5de02b29972a122`, ready and detached, bound to the same member,
Project, container and source. Read-only reconciliation proved that state and
proved zero Factory runs/executions/usage before continuation. The new collector
copied eight prior evidence files inside its protected inventory, blocked new
reservation/Create requests and measured actual attach/reload/HOME behavior.
It reports `creates: 0`, `resumedExistingTerminal: true`, and no invented HTTP 201
receipt. The original attempt and binary remain unchanged.

The continuation collector SHA-256 is
`3a890050c7f9a41e6d3dcc85a08dda42e330c5f9bcb8153800f537fe1a22faa1`.
Its sole caller `18965` exited 1. At `16:50:25Z` its systemd unit was retired
(not found/inactive, no main or control PID) and the local execution controls were
false. Its attempt failure is retained at
`outputs/docker-rc6-continuation/attempt/failure.json` inside the new authority.
It reports an incomplete Factory phase; persistence did not start and no final
seal exists. The last selected snapshot at `16:47:11Z` showed both runs settled,
reconciled and credential-returned, no unsettled work/held reservations/leases/
nonterminal executions, seven sponsored minutes used, and 339.470224 seconds of
rolling usage. The original attempt allowance closed at 604 seconds against its
10-minute limit. Remaining sponsorship does not reopen that closed allowance.

### Current Factory observer diagnosis

The private collector's `factory-native.ts` selects `p.assignment` as publication
`id`. Production `factory.Publication` has separate `ID` and `AssignmentID`
fields; the canonical ID is `p.data->>'id'`. Candidate preparation admission and
completion records refer to that canonical publication ID. Consequently the
collector's preparation join cannot match, and its final completion validation
also cannot succeed with that snapshot representation.

The private `factory-ci.ts` has the same mismatch in its join from candidate
preparation `publication_id` to `factory_publications.assignment`. Both consumers
must use the actual canonical publication ID and preserve assignment identity
separately where the contract needs it. Fixing only the visible CI loop is
insufficient.

Production source starts are:

- `internal/factory/publication.go`: distinct publication/assignment identities.
- `internal/store/factory_publications.go`: persisted publication data.
- `internal/factory/control/publication_children.go`: candidate preparation before
  waiting on exact CI; preparation admission binds `PublicationID = p.ID`.

At `16:52:44Z`, the supported `/prepare-inspect` readback for the same candidate
preparation `fe65b2f359b562a1ce3ccd95c` returned HTTP 404. All eight custody
counters were zero, no native units were active, and the connection was Ready.
This establishes that native preparation was absent after attempt closure; it
does not establish whether it had never started or had subsequently retired.
The readback is
`$RUN/r56-docker-supervision/review-preparation-read-only-r1/result.public.json`,
SHA-256 `2a72ff5dbd5accf8c2f691bb682c6e9d04c1fdb04109647d7bc209f0fef04928`.
An initial probe refused on stale generation/role assumptions; correcting those
diagnostic assumptions made the read-only probe work.

Do not infer a product defect solely from the empty phase or 404. Do not create replacement
setup, patch database state, publish fabricated CI, or rerun completed coding to
work around an observer failure. An admitted running collector cannot be patched
in place and then presented as unchanged evidence.

A minimal source correction is retained at
`$RUN/r56-docker-collector-publication-identity/`: `factory-native.ts`,
`factory-phase.ts`, and `factory-ci.ts`. It preserves separate publication and
assignment identities and checks both in joins. Strict TypeScript checking passed;
it has **not** been built or run against a live candidate. The frozen continuation
source, binary and failed evidence remain unchanged.

## Where another agent should resume

| Location under `$RUN` | Purpose |
| --- | --- |
| `r56-rc6-inputs/` | Frozen release inputs, publication review and held promotion review |
| `r56-public-consumers/qualification-api/` | Small caller of unchanged canonical F1 qualification APIs; per-target controls, attempts and results |
| `r56-public-factory-fixture/` | Actual owner/repository/setup/account/funding metadata and bounded-test authorization |
| `r56-docker-collector-continuation/` | Selected private continuation source, binary and composition receipt |
| `r56-production-collector-admission/docker-rc6-continuation/` | Fresh native baseline and root-controlled collector admission |
| `r56-docker-supervision/` | Selected safe metadata and read-only terminal reconciliation |
| `r56-cold-restore/` | Native completed recovery and Docker prepared recovery tools |
| `recovery-net-rc6-docker/` | Separate prepared Docker cold network; do not reuse the retired native one-shot network |
| `sizing/` | Existing resource sampler and concrete remaining RC6 workload instructions |
| `rc2-public-vm/CONTINUATION.md` | Operational handoff history; newest entry first; older pending approvals may be superseded |

Original failed Docker authority:
`/home/soda-rc-public-docker-collector-authority-20261025-rc6`.
Continuation authority:
`/home/soda-rc-public-docker-collector-authority-20261025-rc6-continue`, output
`outputs/docker-rc6-continuation`.

For canonical qualification, the evidence argument is the protected
**`observations` directory containing `release-evidence.json`**, not its parent.
The selected release controller is
`/home/soda-rc-final-authority-20261025-rc6/controller/soda-release`, SHA-256
`3799c92745fa39d1c37744a45b502daf0f2b59a2602ad74e69bf39d2fae2827d`.
The build executable is a different program and is not interchangeable.

## Repeating the work and its limits

The committed [release workflow](release.md#guided-release-operator-flow) owns the
commands for a new attempt: select/preflight, build/prepare once, stage, actual
public installation, observation collection, qualify, and separately authorized
promotion. [Acceptance tools](../../tools/acceptance/README.md#collect-an-observation)
and the strict schemas in
`lib/soda-release-deliver/src/publish/qualification.rs` own observation inputs.
Use a fresh output and new disposable target when intentionally repeating a
mutating journey. Reuse unchanged successful checks at their actual scope.

This report and the committed tool contracts are portable. The retained native
and Docker collector compositions, workstation admission wrappers, VM disks and
protected evidence are **local artifacts, not a turnkey clean-clone test suite**.
Their source, exact binary identities and invocation records are retained at the
locations above. A next agent on this workstation can inspect and resume them;
another machine must provide its own admitted target, runtime and access. Do not
copy a stale fixture address or a control flag and assume it grants authority.

Never commit owner passwords, service/provider tokens, account directories, raw
model responses, or protected App data. Factory supervision uses selected state,
identity, custody and accounting fields; do not dump raw run bodies, summaries,
stop reasons or provider logs. Account approval belongs to the actual account
owner. Existing transport/cleanup authorization and the finite test limits remain
distinct from permission to publish or promote.

## Remaining work

1. Diagnose the same candidate preparation and correct the observer's publication
   identity handling. The original attempt is closed; do not silently reopen it
   or rerun coding. Choose any further live trial with an explicit finite scope,
   preserving this failed attempt and its accounting. No automatic model retry,
   replacement inquiry or refill is approved by this report.
2. Complete real candidate CI, review, conditional merge and accounting. Only
   actual successful Factory evidence can permit persistence and sealing.
3. Complete Docker same-image recreation, canonical qualification and separate
   offline cold recovery. Cold tools are prepared but have not run successfully
   for this Docker target. Bind their inputs to the actual final container/head
   and sealed proof; do not use initial values after recreation.
4. Measure the frozen release capacity profiles. Earlier T5/development sizing
   does not prove RC6 app + terminal + Factory together or two concurrent issues.
   The existing sampler is reusable; additional live model work requires its own
   explicit finite scope and must not refill this test.
5. Reconcile the actual results into the existing R55/P32 closure work. Promotion
   and `latest` remain held until the owner releases those explicit holds.

## Storage and retained work

Build scratch, consumed archives, retired caches and disposable fixtures were
pruned under the owner's cleanup authorization. The last `/home` space check
recorded approximately 233 GiB available. Allocations of reflinked/shared files must
not be added as though each were independent physical space.

Source/Git, release artifacts, diagnostic logs, signature/provenance metadata,
credentials and retained or unique VM/application state were preserved. Canlang
coordination was queued, but no reply was received; its owned active trees were
left untouched. Do not describe that as completed two-way cleanup coordination.

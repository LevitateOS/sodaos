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

The corrected Docker trial exposed a native finalization bug, now fixed and
verified locally. Its failed observation stays failed; the unchanged published
RC6 does not contain that fix. The corrected source is now on public main at
`9842e527c89f120a26bcdb942b0e718eb86aa879`; its exact-source main and deep CI
passed. Its local production build, independent candidate check and release
preparation passed; publication and installed qualification remain separate gates.

Times below are UTC unless an offset is shown. Results describe actual executions,
not merely prepared commands. Paths beginning with `$RUN` are local to the
retained workstation:

```sh
RUN=/home/vince/Projects/.sodaos-rc-20261023
SOURCE=/home/vince/Projects/sodaos-rc-final-20261025-rc6
PREPARED="$SOURCE/.artifacts/releases/prepared/rc20261025-final-r1"
```

## RC7 continuation

The owner approved the two-commit source push and CI. The canonical push completed
once, first passing qualification-tag CI and then fast-forwarding main. No release,
promotion or `latest` update was included. All three exact-source runs passed:
[qualification CI](https://github.com/LevitateOS/sodaos/actions/runs/40219519473),
[main CI](https://github.com/LevitateOS/sodaos/actions/runs/40219752691) and
[deep CI](https://github.com/LevitateOS/sodaos/actions/runs/40219752650).
The receipt is `$RUN/r57-source-push/source-push-result.public.json`.

The clean source is `/home/vince/Projects/sodaos-rc-final-20261025-rc7`, revision
`9842e527c89f120a26bcdb942b0e718eb86aa879`, with qualified Fountain `a70dbdd4`
unchanged. Local build instructions and results belong to
`$RUN/r57-private-builder-rc7/README.md`; the proposed immutable unqualified tag
is `v0.1.3-rc.20261025.4`. Build completion, publication and installed qualification
are separate results; none can be inferred from successful source CI.

The production build ran once from `18:17:59Z` to `18:41:11Z` and exited 0
(caller `65810`). The worker was subsequently confirmed retired, with no remaining
controller process. Its receipt is
`$RUN/r57-private-builder-rc7/production-build-result.public.json`; the build log
SHA-256 is `f724e2a32a08938118e52b432d8fe8d722e1bf69a4f01128a75e503bbf234b34`.
Independent candidate check `8946` exited 0 at `18:42:58Z`, recorded in
`independent-candidate-check-result.public.json` in the same directory. Neither
step retried automatically or performed public writes.

Canonical release preparation `79731` exited 0 at `18:46:12Z` and produced
`/home/vince/Projects/sodaos-rc-final-20261025-rc7/.artifacts/releases/prepared/rc20261025-final/release.json`.
Its receipt is `release-prepare-result.public.json` in the same builder directory;
its log SHA-256 is `2facd6cdc7f858f77d4403ccdd3345db905d3f16ffa2c18b6517e495e432117d`.
Public writes remain held until the concrete asset/image inventory is reviewed
and authorized. Build, check and prepare execution flags are retired.

The prepared batch contains 19 public files totaling 1,671,801,982 bytes and five
image descriptors. Exact sizes, digests, completed receipts and the held scope
belong to `$RUN/r57-rc7-inputs/asset-image-inventory.public.json` and
`$RUN/r57-rc7-inputs/PUBLICATION-REVIEW.md`. Selected new identities are:

| RC7 prepared input | SHA-256 |
| --- | --- |
| Candidate | `4f2a9a890f0997c9ac3c807c7b2d9aece7aad1b25a584b9bb4a6eb9920074399` |
| Payload | `d2addbba51e766e2851d80ccb3d26bce1eba9f8b59b20af38593a89a0d51cd83` |
| Release manifest | `0c435f47ee82da7a3c0beccc4735473b4c2a588f89fae2bccc7e3a5aa43fa7bd` |
| ISO | `def608acaa5576adbd332677929f4f2b00e771d2232070cdb39bf42212a0c723` |
| Docker manifest | `960935bc0baa43a4479199b7e225c04671c3cf090b38ca6a2c82390382ae4887` |

These are prepared local bytes, not a published release or an installed pass.

Native and Docker collector compositions compiled once against that source.
Their sources and build receipts are in `$RUN/r57-appliance-collector` and
`$RUN/r57-docker-collector`; the canonical qualification caller is in
`$RUN/r57-public-consumers/qualification-api`. Fresh blank test VMs are prepared
and shut off under `$RUN/rc7-public-vm` and `$RUN/rc7-public-docker-vm`.
The unpaid Factory fixture, context helper and separate cold-recovery preparations
are in `$RUN/r57-public-factory-fixture`, `$RUN/r57-context-status-helper` and
`$RUN/r57-cold-restore`. Unknown installed identities, credentials, final image
identities and protected journey seals remain unbound. No new live model trial
or sign-in has occurred; its finite scope requires actual authorization.

The RC6 results below retain their original source and scope. A local RC7 build
or source fix does not turn the failed RC6 observations into successful ones.

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
| Corrected Docker trial | Failed/incomplete: completed implementation report and PR, but native finalization failed on a legitimate tree-only verification task; review correctly refused | Corrected authority below; caller `28534` exited 1, no seal or persistence; journal predicates retained |
| Native final-head fix integration | Passed on the two-file source fix: actual production final-request function, real Git/journal/locking, unchanged verification and stale/unfinished refusal cases | `$RUN/r56-native-final-head-integration/integration-result.public.json` |
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

A further selected readback at `16:55:04Z` found the same admission marked
`stopped: true`, `retirement: confirmed`, while its separately stored observation
still had an empty phase and `ready: false`. Thus the 404 accompanies confirmed
admission retirement. The observer's old `state.stopped` field alone missed this
distinction. Evidence is
`$RUN/r56-docker-supervision/review-preparation-admission-read-only/result.public.json`,
SHA-256 `ef13c3bbd08c093444716c0a0ce07df2bf002730dd4ac09d8ed2e1163612da0a`.
Production Store authorization already selects by owner assignment and separately
checks canonical publication ID; it does not share the private observer's join
bug. No product patch is justified there by these findings.

Do not infer a product defect solely from the empty phase or 404. Do not create replacement
setup, patch database state, publish fabricated CI, or rerun completed coding to
work around an observer failure. An admitted running collector cannot be patched
in place and then presented as unchanged evidence.

A minimal source correction is retained at
`$RUN/r56-docker-collector-publication-identity/`: `factory-native.ts`,
`factory-phase.ts`, and `factory-ci.ts`. It preserves separate publication and
assignment identities and checks both in joins. Strict TypeScript checking passed.
The corrected **initial-mode** composition then built successfully at
`$RUN/r56-docker-collector-initial-corrected/public/soda-qualification-collector-docker`,
SHA-256 `ad1f7717bbac537295db0199c1f05f293613be3054f77e0ff11ef32709c0e4c1`.
Its composition receipt is
`$RUN/r56-docker-collector-initial-corrected/composed-build.public.json`, SHA-256
`d92ab100aef06dc779243ba6805edce52d514a200dfa04c50202dc3f6d59ef18`.
Only the three identity modules and recorded composition inputs changed from the
original initial collector; Project, witness, persistence and context modules
remain byte-identical. Its separately approved trial ended incomplete, as recorded
below; a new successful Factory result has not been established. The frozen
earlier sources, binaries and failed evidence remain unchanged.

Normal Retry after this closed root creates a new investigation/coding attempt;
it does not resume only the pending review or deterministically reuse PR 2.
The special same-root context retry applies to an unstarted context-pending run,
which is a different case. Source starts are `lifecycle_retry.go`,
`dispatch_attempt.go` and `internal/store/factory_explicit_retry.go`.
Remaining sponsorship alone cannot make the original closed attempt eligible.

The corrected snapshot was also exercised read-only against the retained real
PostgreSQL state. Canonical publication ID, separate assignment ID, their
inequality, and the candidate preparation's publication/owner/source joins all
matched. There were zero completions and merges, so those joins remain untested
on this attempt. An additional diagnostic that combined owner assignment and
implementation run into one identity returned false; that stronger predicate is
not the canonical contract. The collector proves owner/root lineage separately
from the implementation run. This diagnostic did not trigger a rebuild or waive
any actual required check.

## Corrected trial result: native finalization failure

All unpaid setup completed for repository 2,
`rc-public-docker-rc6-factory-corrected-20261025`, on the same Docker VM. Its
Project is `pb91edd2cd64a1979876333c3`, source
`f5d266757c4e5530ba2f7b8a30e1400628c9d4b9`, issue 1/native issue 3.
Both ordinary role preparations are ready with setup/check exit 0, the genuine
issue is accepted, and dispatch is paused. One preparation POST returned
`preparation_unconfirmed`; supported GET reconciliation proved success without
repeating that POST. Existing service actors, tokens, hooks and account were
reused. No new sign-in, funding or model work occurred during this setup.

The handoff is
`$RUN/r56-public-factory-fixture-corrected/docker/unpaid-setup-handoff.public.json`,
SHA-256 `a9fa5c3b9d735dc7bd392ea12fae9519a03388e9fc9814dc1c7fca963fb54af3`.
The actual two Projects and all retired history remain; no data was erased to
satisfy a stale one-Project or zero-history test assumption.

Context deployment and admission preparation completed under
`$RUN/r56-production-collector-admission/docker-rc6-corrected/`.
Its `preparation.public.json` SHA-256 is
`f79b325f515ccd13f4b7c0e6b8c8978a801d1f057150bce65e3a1aa57d6db623`.
The public context helper binary was unchanged; the exact preceding scope and
metadata were preserved before installing the new Project-bound scope. At that
model-free handoff, all execution controls were false and all active custody
counters were zero.

A new live test received explicit owner approval because the preceding test
excluded automatic retries. The approved scope is
`$RUN/r56-public-factory-fixture-corrected/setup-authorization.prepared.json`:
one new Project member grant, 15 sponsored minutes, shared rolling limit raised
from 15 to 25 minutes, max one active run, no automatic retry/refill/extra inquiry.
The previous seven spent minutes, old sponsorship balance, closed attempt, PR and
data remain. A 15-minute rolling limit does not have the full planned 15 minutes
of remaining headroom after the earlier usage; it must not be reset to conceal
that fact.

The commands executed **after that actual approval** were:

```sh
node "$RUN/r56-public-factory-fixture-corrected/run-phase.cjs" funding
node "$RUN/r56-public-factory-fixture-corrected/run-phase.cjs" emit
```

The exact reply, “Approve one bounded trial”, is recorded in
`$RUN/r56-public-factory-fixture-corrected/bounded-test-user-approval.public.json`.
Normal funding (`35947`) and emit (`58530`) both exited 0. The runner obtained the
real member grant through the owner API and read back its exact binding; no grant
ID was invented or copied from the old Project. The resulting
`docker/consumer-config.json` SHA-256 is
`4612773a066ed43bef2f00e83bcf757a5af95b2fd8acb11e9bfcaecc47f1ce75`.
Funding/emit controls returned to false after execution.

Fresh funded baseline `64460` passed at `17:20:35Z`, retaining both Projects and
the earlier usage. Admission `92925` passed. A preceding admission refusal
`50193` caught a local domain-label guard typo before authority or credential
writes; only that fixture guard was corrected. No model ran in the refused
admission.

The sole corrected collector caller is `28534`, supervised by `context_helper`.
Its authority is
`/home/soda-rc-public-docker-collector-authority-20261025-rc6-corrected`, output
`outputs/docker-rc6-corrected`. Runtime inventory SHA-256 is
`27fd1f7d2fa12dc786a17c0a0693d911cd866a76f35771e611e64fe89d041126`.
At the `17:30:40Z` saved observation, installation and browser Project/HOME witness
had passed, investigation had succeeded and retired with two sponsored minutes,
and implementation was active. No publication or failure was recorded yet.
An active caller must not be duplicated or concurrently polled by another agent.
The caller subsequently exited 1 at `17:35:06Z`; its execution controls and native
unit were retired. No persistence, seal, canonical Docker qualification or cold
recovery followed. Its local `collector-result.public.json` SHA-256 is
`0475fefce893b73e53e74bc38e3751476044034fccdef4a28c6f8991c9e60c99`.
No automatic retry, refill or extra model inquiry ran.

### Native finalization bug confirmed against this run

The implementation run was `50385e36d9ec3f1f646800bc8d8a5c15`. Its assignment had
a succeeded, reported, completed result for the exact published candidate, but
its reconciled native run outcome was `failed`. Review's successful-native-run
guard therefore refused before exporting the candidate or launching reviewer
preparation. Supported `/prepare-inspect` returned 404 at `17:33:16Z`, while the
attempt was still eligible. This removes the earlier ambiguity about a 404
observed only after retirement.

The exact native receipt showed started and delivered execution, confirmed
retirement, returned credentials and matching binding, with **no numeric exit
code**. This does not establish a nonzero CLI exit or a provider limit. Only
selected booleans/enums were read; no output, reason, summary, stop reason or
provider log was inspected. The existing metadata-only protocol counters contain
sanitized event names/types/statuses, not event ordering or thread identity;
their completed-turn counts cannot reconstruct the exact last event.

The decisive selected journal read at `17:47:59Z` found:

| Task/checkpoint | Actual evidence |
| --- | --- |
| `reject_empty` | Committed with valid tree and new commit |
| `independent_review` | Committed with a valid tree and no new commit; candidate equals expected head; 3/3 checks passed |
| Final candidate checkpoint | Committed at the unchanged candidate; 3/3 checks passed |
| Journal identity | All events match native invocation, manifest digest and known task identities |

Source `lib/host/src/factory/checkpoint/mod.rs` deliberately records only
`tree=HASH` when a successful task leaves the head unchanged. However,
`lib/host/src/factory/native_tasks.rs::final_request` requires a valid `commit=`
on **every** committed task, returning an error on the legitimate unchanged
verification task. The existing checkpoint `integration_head` reducer already
preserves the head for tree-only events and validates any present commit plus
invocation/manifest identity. Native finalization must use that same reducer.
The failed-run review guard must remain intact; stored outcomes must not be
rewritten to make this failed attempt pass.

The selected proof is
`$RUN/r56-production-collector-admission/docker-rc6-corrected/task-journal-metadata-read-r3.result.public.json`,
SHA-256 `11bb6ed55f97232f6d3a30716281007497fee684c0fb5b4c78e3eb8f610e7e15`.
The diagnostic initially asserted an invocation field on the manifest itself;
the canonical manifest contract does not require that field. Only that extra
diagnostic gate was removed. Journal invocation and manifest bindings were
actually checked and matched. The failed diagnostic attempts remain retained.

The two-file product fix is committed on the working checkout as
`2ecbf15870b3b360b2318550b9b229ef023a3c56`, and transplanted onto the frozen F1 base
as `f3789a1842c76251f6d4a120d891fdbd7ee5c5b7` in
`/home/vince/Projects/sodaos-rc-final-20261025-rc7`. It shares the existing
invocation/manifest-validating head reducer with native finalization, preserving
the two checkpoint callers' original starting heads. All ledger, task, child,
final-checkpoint and failed-run review guards remain. The published F1 release
is unchanged; its native App/recovery results retain their original scope, and
this RC6 Factory trial remains failed.

The focused integration executed the production `NativeTasks::final_request`
with real Git, private journal persistence and file locking. Changed commit →
unchanged verification returned the changed head. Stale invocation, stale
manifest, malformed commit and unfinished child cases all refused. It used
actual production Factory/terminal/broker types but made zero terminal, broker,
model or guest calls. This is component integration proof, not a new installed
provider qualification.

Retained reproduction root is `$RUN/r56-native-final-head-integration/`:
`host/Cargo.toml`, `host/integration/main.rs`, the private appended entry in
`host/src/factory/native_tasks.rs`, and `source-binding.public.json` retain the
exact compiled source and pins. Result `integration-result.public.json` SHA-256:
`03ba9d5ef5316b01809b61ac5096ed6780234807915db6ce9857a49d164426b0`.
Preserved executable `public/final-head-integration` SHA-256:
`7ac746e5d409798178f82332f0137b57d09555d96debc85c8e8a07071dd0cb95`.

Reproduce in a **fresh** fixture directory from the normal host environment:

```sh
CARGO_TARGET_DIR="$RUN/host-target" \
TMPDIR="$RUN/r56-native-final-head-integration" \
cargo run --locked --offline --release \
  --manifest-path "$RUN/r56-native-final-head-integration/host/Cargo.toml" \
  --bin final-head-integration -- \
  "$RUN/r56-native-final-head-integration/runs/FRESH_NAME"
```

The passing fixture is retained as `runs/r2`; do not overwrite it. `runs/r1`
preserves an earlier filesystem-sandbox ancestor-ownership refusal before
finalization. Running in the normal host environment resolved that test-environment
restriction. Rust formatting and whitespace checks passed; no new unit tests or
unrelated passing suites were added/replayed.

### Partial frozen resource measurement

The existing sampler recorded 138 valid five-second samples, zero collection
failures, over 686 seconds (`17:25:13Z`–`17:36:38Z`). Peak observed whole-guest
memory use was 1,599,172 KiB (about 1.53 GiB), minimum available memory
6,513,652 KiB, and peak sampled five-second CPU use 1.084 cores. The window had
at most one active Factory/issue-Project database row, zero Apps and two retained
Projects. Active rows alone do not prove native execution; this does not qualify
the combined App/terminal/Factory or two-issue capacity profiles.

Docker lacks `/var/home`, causing the first unchanged sampler to fail its `df`
call. A small local fix selects `/var/home` when present and `/home` otherwise;
the original and failed window are retained. The corrected sampler was stopped
normally before guest handoff and has no active process. Result:
`$RUN/sizing/rc6-docker-corrected-trial/window-r1/window-result.public.json`,
SHA-256 `c0d0e28f8a308af0ee92ff0e76dbab6c5ab6170a0d5935a8a11d9e89e988636d`.

## Where another agent should resume

| Location under `$RUN` | Purpose |
| --- | --- |
| `r56-rc6-inputs/` | Frozen release inputs, publication review and held promotion review |
| `r56-public-consumers/qualification-api/` | Small caller of unchanged canonical F1 qualification APIs; per-target controls, attempts and results |
| `r56-public-factory-fixture/` | Actual owner/repository/setup/account/funding metadata and bounded-test authorization |
| `r56-docker-collector-continuation/` | Selected private continuation source, binary and composition receipt |
| `r56-docker-collector-initial-corrected/` | Built initial-mode collector with canonical publication/assignment joins; no new live result yet |
| `r56-public-factory-fixture-corrected/` | Separate unpaid fixture preparation; existing repository/PR/accounting retained |
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

1. Obtain authorization for the next candidate's concrete unqualified publication
   batch after its completed build, check and preparation. The shared
   native head reducer fix and qualification report are now on main with passing
   exact-source CI. Local focused integration has passed. Observer publication identity was
   corrected; the new live trial exposed the separate tree-only finalization
   bug. Both attempts are closed and remain failed. Do not silently reopen them,
   rewrite outcomes or rerun coding. Release publication and any further live
   model trial need their actual task authorization; this report grants neither.
2. Qualify the corrected candidate through real CI, review, conditional merge and
   accounting. Only actual successful Factory evidence can permit persistence
   and sealing. A local source fix cannot relabel the unchanged public F1 bytes
   as fixed or reuse their failed observation as a pass.
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
pruned under the owner's cleanup authorization. After RC7 build, check and prepare
retired, only its completed assembler `work/media/work` directory was removed.
Three provenance JSONs (61,406 bytes) were preserved under
`$RUN/r57-private-builder-rc7-scratch/preserved-9842-cosa`. Actual filesystem
available space increased by 27,045,171,200 bytes (25.19 GiB), leaving about
206.31 GiB free. The receipt is
`$RUN/r57-private-builder-rc7-scratch/cleanup-result.public.json`, SHA-256
`e7bcc8dd16d333d504dcab802d0c419aa6ea71fa3c15a13273ce80d748e2174b`.
Allocations of reflinked/shared files must not be added as though each were
independent physical space.

Source/Git, release artifacts, diagnostic logs, signature/provenance metadata,
credentials and retained or unique VM/application state were preserved. Canlang
coordination was queued, but no reply was received; its owned active trees were
left untouched. Do not describe that as completed two-way cleanup coordination.

# RC6 and RC7 qualification record — 25 October 2026

This is the retained account of the installed testing behind
`v0.1.3-rc.20261025.3` and `.4`, including failures and the next safe action. It exists so
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
passed. Its local production build, independent candidate check, release
preparation and unqualified publication passed. RC7 native installation, App
qualification and cold recovery also passed. Its Docker Factory trial failed at
a reviewer context decision; the reviewer never received model credentials.
RC7 and RC6 outcomes remain separate.

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
`$RUN/r57-private-builder-rc7/README.md`; the published immutable unqualified tag
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
The owner separately approved the concrete public batch. Build, check, prepare
and publication execution flags are retired.

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

The canonical publisher ran once (`36206`) and exited 0 at `18:58:51Z`.
[Immutable UNQUALIFIED RC7](https://github.com/LevitateOS/sodaos/releases/tag/v0.1.3-rc.20261025.4)
is release `423597969`, published at `18:58:47Z`, with all 19 assets. The canonical
stage also checked the anonymous component/Docker image identities. The actual
approval and result are `$RUN/r57-rc7-inputs/publication-user-approval.public.json`
and `publication-stage-result.public.json`. No source push, promotion or `latest`
update was part of this publication.

Anonymous download `67965` exited 0 after verifying all 19 files against the
reviewed inventory, GitHub API digests, manifest and checksums. The retained
directory is `$RUN/public-rc7-downloads-20261025-net`. Before releasing either
installation, `gh release verify-asset` authenticated the actual ISO, rootfs,
Docker installer, checksum manifest and release manifest (`84846`, all five
exited 0). Exact commands and outcomes are in
`$RUN/r57-public-consumers/asset-auth/asset-auth-result.public.json`.
Public byte verification is separate from the installed results below.

Native installation on `192.168.2.164` (UUID
`3427ec8f-57d3-4ba7-830e-1ee5e3e63b13`) read back source `9842e527`, payload
`d2addbba` and all six shipping images. Ordinary-owner setup, trusted HTTPS,
key-only SSH and bootstrap-token revocation passed. The owner handoff is
`$RUN/rc7-public-vm/owner-handoff.public.json`.
Protected App collector `90197` exited 0: actual registration, record write,
native reboot, same-owner readback and final stopped state passed. Its original
protected observations are
`/home/soda-rc-public-appliance-collector-authority-20261025-rc7/outputs/actual-installed-app-journey-rc7/observations`.
Canonical appliance qualification `63749` then exited 0 at `19:28:46Z`, recorded
in `$RUN/r57-public-consumers/qualification-api/appliance-qualification-result.public.json`.
Native cold recovery then passed all seven callers: offline disk/UEFI capture,
isolated clone boot, native readback, actual same-owner App record readback,
final power-off and network retirement. The clone is
`aaca91bc-c216-4e0f-bd7f-a2edae94a724`; the original disk and UEFI were unchanged.
Both domains were off at completion, and the isolated network was retired.
Result: `$RUN/r57-cold-restore/appliance/journey-result.public.json`, SHA-256
`ae3c68a6d16c9d3cb894f82aabba977a12210319b7c233abc0f6e05852abd403`.
The original was subsequently booted normally for the separate unpaid capacity
fixture below; this does not alter the completed cold-recovery observation.

Docker's supported public streamed installation `21745` exited 0 on
`192.168.2.165` (UUID `4be9b314-8284-43f5-9acd-b1a9b28dc14d`), with source,
payload, image/config and fresh-volume readback. Actual outer container:
`75adde7d65c548a6c59ca37ac5877b69d4c766b995bea0469eb5455ed56a7cdc`.
Ordinary-owner setup, trusted HTTPS and bootstrap-token revocation passed;
handoff `$RUN/rc7-public-docker-vm/owner-handoff.public.json`. Its initial unpaid
Factory fixture completed before the separately approved trial below.

Native and Docker collector compositions compiled once against that source.
Their sources and build receipts are in `$RUN/r57-appliance-collector` and
`$RUN/r57-docker-collector`; the canonical qualification caller is in
`$RUN/r57-public-consumers/qualification-api`. Fresh blank test VMs were prepared
under `$RUN/rc7-public-vm` and `$RUN/rc7-public-docker-vm`; their actual installation
lanes were released only after the public byte/authentication checks above.
The unpaid Factory fixture, context helper and separate cold-recovery preparations
are in `$RUN/r57-public-factory-fixture`, `$RUN/r57-context-status-helper` and
`$RUN/r57-cold-restore`. Installed identities and native proof were bound only at
the actual handoffs above. Docker's failed trial has no final Factory proof;
its canonical qualification and cold-recovery controls remain held.

### RC7 Docker trial and correction

The owner approved one trial with 15 sponsorship minutes, 15 rolling model
minutes, one active run and 900 seconds, without automatic retry, refill or extra
inquiry. Confirmed Codex enrollment produced connection
`7caf661ee18cece2756739030a30fdea`. The existing sign-in is retained; collector
failure does not require a new device code. Enrollment and funding receipts
belong to `$RUN/r57-public-factory-fixture/`.

The original fixture is repository 1, Project `pdf46bf52e38d8e2e0bb06c1f`,
issue 1, initially at `0988255afa4492162546938cc38d229913cddceb`.
Protected collector `77809` exited 1 at `19:54:22Z`. Installation, actual browser
terminal attach/reload/HOME witness, investigation, implementation and candidate
publication passed. Implementation's native result succeeded and retired; this
attempt did not reproduce RC6's failed native finalization. Independent review,
completion, merge, persistence and the final observation seal did not pass.

The preserved authority is
`/home/soda-rc-public-docker-collector-authority-20261025-rc7`, output
`outputs/docker-rc7`. The collector binary SHA-256 is
`766050d754750ae2752282cf479fc4c6f337554bd177ef0ae57b247b8f9884a2`.
The result is
`$RUN/r57-production-collector-admission/docker-rc7/collector-result.public.json`,
SHA-256 `4d5767f56b4fd08d9e9247efe5acdb195a4e27d9863a910fcdcb2af8fc05ad69`.
Its selected failure stage was `receipt_held-policy`; that label is not a raw
provider error or evidence of a CLI exit failure.

Reviewer `ac64ed9de3bd7c064a0030033c98449b` reached a normal context-attention
decision with its candidate preparation ready. The private caller handled
`use_current_context` only for implementation and tracked one resolved run
globally. Its read-only helper was also bound only to the original coder
preparation. It therefore did not handle this review decision. The review had no
lease, delegated credentials, native binding or model execution. The caller
stopped after its context wait; the review subsequently expired and settled as
cancelled. A historical typed receipt showed confidence 0.69 and lineage
probability 0.79, below the canonical 0.80 bounds: attention was the correct
production decision, not grounds to bypass the context guard.

At `20:03:41Z`, all eight custody counters were zero, policy paused and dispatch
closed. Sponsorship usage was **10 of 15 minutes**, including the normally
settled staged review, rather than the seven minutes observed before cancellation.
Rolling usage was **472,550,938 of 900,000,000 microseconds**. These retained
charges must not be reset or hidden. The selected readback is
`$RUN/r57-context-review-diagnosis/diagnostic-r3-result.public.json`; the
confidence-only readback is `confidence-result.public.json` in that directory.
Neither exported raw run bodies, model text or provider logs.

The expired review cannot be resumed through `use_current_context`. Normal Retry
would open a fresh investigation; it is not review-only recovery. Exact supported
actions and source gates are recorded in `$RUN/r57-review-recovery/READY.md`.
A separate caller correction compiled with implementation and publication-bound
reviewer projections, exact canonical lineage and per-run context resolution.
It reads the current canonical connection generation; it does not pin the initial
enrollment generation. The normal owner action still requires canonical
`can_use_current`, complete current context and all three exact selection digests.
It uses the recorded judgment without another model inquiry. This action can
resolve attention while eligible; the expired original review remains ineligible.

An affected TypeScript check passed. A local integration using the retained real
implementation command exercised the actual permit arm/admit bridge with its new
per-run identity, exact request and one-use limit; changed path, body and label
were refused. This caught and corrected an old exact-label rule before any paid
successor ran. Results belong to `$RUN/r57-context-role-helper/` and
`$RUN/r57-docker-collector-context-aware/correction-ready.public.json`.
Eligible reviewer execution remains an unperformed live check, rather than a
claim inferred from compilation. Production source `9842e527` and the failed
authority remain unchanged.

Fresh unpaid setup passed on the same appliance for repository 2,
`rc-public-docker-rc7-factory-corrected-20261025`, Project
`p683bc93a2341173ca414e578`, source
`ab676d4cd92b85be10d2f06da61ef5bb2c38f20b`, issue 1/native issue 3.
Both original preparation IDs are ready with setup/check exit 0; supported
same-ID continuation completed the initially unconfirmed preparation. The issue's
successful UI POST was reconciled without creating another issue. The existing
Ready sign-in, seven service tokens and TypeSafe configuration were reused;
repository 1, its candidate/PR and every charge remain intact.
Handoff: `$RUN/r57-public-factory-fixture-corrected/docker/unpaid-setup-handoff.public.json`,
SHA-256 `9106b572b41a87d853a2c4a4643dcea09222443be1dff154389443771037c801`.

The separate unpaid reader passed at `20:22:46Z`: all eight custody counters
zero, two Projects, zero Apps, no new-repository history and all three old runs/
10 used minutes preserved. Only the new read-only role helper and its bound
scope were then deployed, without a service restart. No model eligibility was
invented from this idle check. Receipt:
`$RUN/r57-production-collector-admission/docker-rc7-context-aware/unpaid-ready.public.json`.

The proposed successor is one 15-minute sponsorship, rolling limit explicitly
raised from 15 to 25 minutes while retaining previous usage, one active run and
900 seconds. It includes ordinary saved-current actions for both roles only
when canonically eligible, and excludes automatic retry, refill and extra inquiry.
Its actual scope is
`$RUN/r57-public-factory-fixture-corrected/setup-authorization.prepared.json`.
The frozen scope SHA-256 is
`6c703c528e1adb38232ec42dace08f5dc908841311a1b6ce0d990785e62d1fb3`.
No new grant, funding, consumer emission or paid run has occurred. The prepared
consumer has `grant: null`; it is not evidence of actual funding. Another paid
trial requires a new finite approval because the prior approval explicitly
excluded automatic retries. After that real approval, the held runner's exact
commands are `node "$RUN/r57-public-factory-fixture-corrected/run-phase.cjs" funding`
then the same command with `emit`. Both require the actual scoped approval
receipt; an unapproved prepared scope is insufficient. Fresh funded baseline,
runtime inventory and admission must bind the emitted consumer before launch.

The owner subsequently approved this exact corrected trial, then requested a
review and normalization of the entire procedure before further execution.
**The trial is held and has not started.** No funding, actual consumer emission
or new model call followed that approval. Its unchanged finite authorization
remains valid; do not ask for the same approval again when the owner resumes it.
The [reusable procedure and tooling priorities](release.md#normalize-the-qualification-procedure)
now distinguish product fixes, private driver fixes and installed qualification.

The subsequent whole-process review found another deterministic driver mismatch:
the corrected admission accepts two retained Projects, while
`r57-docker-collector-context-aware/docker-persistence.ts::stopOwnedProject`
expects only the unique browser/Factory Project (one in this fixture). Its native
snapshot also requires all Projects stopped. Thus the compiled successor is
**not ready for a complete paid journey** until persistence and owned-resource
closure agree with the actual retained inventory. This was found before funding;
no further trial ran and no state was cleared to satisfy the assertion.
The owner selected compute, model usage and storage as the primary redesign
objective. The expanded [process proposal](release.md#normalize-the-qualification-procedure)
replaces the earlier outline with alternatives, a shared scenario, proof reuse,
whole-route readiness, resource lifecycle and staged implementation.

The RC7 App fixture was rebuilt against runtime config
`4a03219464e996f8bb35771a091d98978f169172f6305e9c171348e5cf5ef438`.
Its model-free local native run passed 27 checks; the container retired and no
guest action occurred in that run. Image/archive/manifest/proof identities are in
`$RUN/r57-appliance-app-fixture/handoff.public.json`. It is separate from RC6's
fixture; its installed App journey is reported above.

The native App-only resource window (`19:20:21Z`–`19:26:38Z`) retained 71 valid
samples and four reconnect failures around the canonical reboot. Peak sampled
whole-guest memory used was 1,584,980 KiB (about 1.51 GiB), minimum available
6,527,844 KiB, with at most one App and zero Projects/Factory runs. CPU accounting
spans a reboot and has that explicit limitation. This is partial App evidence,
not combined or two-issue capacity qualification; the sampler is retired.
Result: `$RUN/sizing/rc7-appliance/resources/window-result.public.json`, SHA-256
`f8fbbb34c8bdd97ebcf6871609e67bdbd627ab2e97db8e7cf125b806ace6e089`.

Docker's separate window (`19:46:04Z`–`19:54:39Z`) retained 104 valid samples,
zero failed samples and a retired sampler. Peak sampled whole-guest memory used
was 1,528,460 KiB (about 1.46 GiB), minimum available 6,584,356 KiB and maximum
busy CPU about 1.155 cores. It had one Project and zero Apps, with at most one
active Factory ledger run. Ledger rows alone do not prove simultaneous native
execution. Evidence: `$RUN/sizing/rc7-docker/resources/`.
These separate App-only and Docker Factory windows cannot be composed into a
combined workload or two-issue capacity pass. Both guests have four vCPUs and
8 GiB RAM; their allocation is not a measured supported minimum.

The native unpaid capacity fixture is complete on the original `.164` appliance:
two distinct Projects, ready coder/reviewer preparation pairs and genuine accepted
issues, alongside retained App `aba31fc21ab6e4473a27cc277` and its existing record.
The App is running and a real member terminal was verified. Policies remain
paused and there are zero connections, runs, sponsorships, budgets and model
minutes. Handoff:
`$RUN/r57-native-capacity-fixture/unpaid-setup-handoff.public.json`, SHA-256
`45021fba2fc2aa07e5f7e8d4bce90f7be33398b49d58479464b623f4ab0464db`.
Two distinct Ready connections through supported enrollment and finite model
authorization are still needed before measurement; they need not belong to two
different provider accounts. The proposed scope is at most five minutes per genuine issue,
ten additional model minutes total, one run per connection and two concurrently,
without automatic retry/refill. No native account or paid action has run.
The capacity-driver drafts are not a validated end-to-end runner. They remain
held during the owner's procedure review; completing setup or compiling helper
variants does not establish that runner's readiness.

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
| `r57-rc7-inputs/` | Published RC7 inventory, exact approval and publication result |
| `r57-public-consumers/qualification-api/` | RC7 canonical native qualification result; Docker held pending an actual seal |
| `r57-production-collector-admission/docker-rc7/` | Closed, failed RC7 Docker trial; original baseline/admission/result |
| `r57-context-review-diagnosis/` | Selected read-only expired-review lineage, accounting and historical confidence facts |
| `r57-review-recovery/READY.md` | Exact supported recovery actions and why this review cannot resume |
| `r57-cold-restore/appliance/` | Completed RC7 native offline recovery and final guest handoff |
| `r57-native-capacity-fixture/` | Actual unpaid two-Project/App/terminal setup; held account/model scope |
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

1. Finish the private RC7 reviewer-aware context caller, then bind it to the new
   unpaid Docker fixture. Preserve both roles' canonical eligibility and exact
   lineage/digest checks. Obtain a new finite approval before another paid trial,
   reusing the existing Ready connection and preserving all old usage. Its
   expired review cannot be reopened. Native App qualification and cold recovery
   already passed; do not replay them because the Docker caller failed.
2. Complete a genuine Docker Factory journey through real CI, independent review,
   conditional merge and accounting. Only actual successful evidence permits
   persistence and sealing. RC7 implementation succeeded, but no completion,
   merge or final seal exists. Keep all failed RC6/RC7 attempts failed.
3. Complete Docker same-image recreation, canonical qualification and separate
   offline cold recovery. Cold tools are prepared but have not run successfully
   for this Docker target. Bind their inputs to the actual final container/head
   and sealed proof; do not use initial values after recreation. The current
   Docker qualification fixture has zero Apps. Adding a model-free App and
   proving its owner/record changes requires a fresh post-App native capture
   binding before cold recovery; its full protected-state hash cannot equal the
   earlier App-free baseline. Retain that earlier seal at its actual historical
   scope rather than presenting it as App-data recovery proof.
4. Measure the frozen release capacity profiles using the completed unpaid native
   fixture and an actual same-interval workload. Earlier T5/development sizing
   and separate RC7 resource windows do not prove App + terminal + Factory
   together or two concurrent issues. The existing sampler is reusable;
   two Ready connections and explicit finite scope are still needed.
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
credentials and retained or unique VM/application state were preserved. The
Canlang agent subsequently replied at
`/home/vince/Projects/ssd-prune-canlang-reply.txt`: it removed nine confirmed
inactive Cargo debug trees and reported a net filesystem gain of 18,904,621,056
bytes (17.61 GiB), preserving its active work and retained outputs. The receipt
and current ownership boundary are recorded in
`/home/vince/Projects/ssd-prune-coordination-20261025.json`. A subsequent `/home`
check during VM population showed about 179 GiB free; these changing snapshots
must not be added as independent reclaim totals.

After the Canlang agent cleared the shared `go-runner` build cache, a privileged
current-use scan found no references and no unknown scan results. Only
`/home/vince/.cache/go-runner/build` was removed, reclaiming an actual
8,283,168,768 bytes (7.71 GiB). Source, other caches and active work were preserved.
Receipt: `$RUN/r57-go-runner-cache-cleanup.public.json`. No cleanup caller remains
active. Space readings taken during later VM population describe that moment,
not additional independent cleanup gains.

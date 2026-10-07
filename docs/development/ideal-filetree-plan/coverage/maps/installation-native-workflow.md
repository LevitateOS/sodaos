# Installation native workflow

[Responsibility map index](README.md) · [Coverage scope](../README.md).

Current responsibility accounting at `519b76bd` (2026-10-07).
One slice owns each named duty; disjoint complete symbols may share an owner.
Compound fields/clauses may share a physical line with distinct selectors.
Disposition concerns the duty, not source validity or installed qualification.

<a id="coverage-764f0f7586f6"></a>
<a id="rustsoda-installsrcdisksrs-1"></a>
<a id="coverage-292c2ab05f57"></a>

## [cmd/soda-install/src/disks.rs](../../../../../cmd/soda-install/src/disks.rs)

Current cohesive body inspected; single defining duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–551; whole file: native coreos live-media disk inventory parsing and candidate-disk selection, including removable/live-media filtering | [D11](../../slices/release-and-installation.md#d11-host-installation-and-payload-application) | retained | Native CoreOS live-media disk inventory parsing and candidate-disk selection, including removable/live-media filtering. — cmd/soda-install/src/disks.rs; consumed by explicit `soda-install disk` flow |

<a id="coverage-5ede9e441281"></a>
<a id="coverage-9ba41df7bbcc"></a>

## [cmd/soda-install/src/run.rs](../../../../../cmd/soda-install/src/run.rs)

Current dispatch, media identity parser, and direct install/enrollment consumers inspected

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–137, 239–299; module imports and media identity data/decoder; installer action and media identity source assertions | [D04](../../slices/release-and-installation.md#d04-authenticated-installation-media) | retained | Defines and decodes authenticated live-media identity fields and validates architecture, release, revision and included payload hashes.; Pins media identity parsing and admission behavior, including duplicate and unknown field handling. — cmd/soda-install/src/run.rs:1-137; media identity written by release media builder; cmd/soda-install/src/run.rs:239-298; in-file media identity matrix |
| 138–162; native architecture and enrollment action dispatch | [O07](../../slices/operator-administration.md#o07-operator-ssh-enrollment) | retained | Dispatches enrollment serve/receive actions after native host admission and without requiring the install console. — cmd/soda-install/src/run.rs:139-162; calls enroll::serve_enrollment/receive_enrollment |
| 163–238, 300–318; installer single-instance lock and disk/configure/enroll-key action flow; root/action refusal source assertion | [D11](../../slices/release-and-installation.md#d11-host-installation-and-payload-application) | retained | Enforces one installer attempt, requires root/CoreOS and a controlling terminal, then dispatches disk installation, host configuration, or key enrollment.; Pins root/action refusal behavior for the installer dispatcher. — cmd/soda-install/src/run.rs:164-237; called by cmd/soda-install main; disk/configure use setup and execute modules; cmd/soda-install/src/run.rs:300-318; in-file refusal test |

## Removed predecessor navigation

These anchors identify earlier mapped source. Their paths no longer define this current map;
the current inventory and normalized predecessor ledger preserve replacement/retirement evidence.
No historical source body is counted as current coverage.

<a id="coverage-b2c08d1a2d71"></a>
<a id="rustsoda-installsrccandidaters-1"></a>

Former source `rust/soda-install/src/candidate.rs`; consult its pinned earlier Git source and the current coverage disposition.

<a id="coverage-394745d4d1c9"></a>
<a id="rustsoda-installsrcconsolers-1"></a>

Former source `rust/soda-install/src/console.rs`; consult its pinned earlier Git source and the current coverage disposition.

<a id="coverage-dd156bf91d35"></a>
<a id="rustsoda-installsrcdeliverrs-1"></a>

Former source `rust/soda-install/src/deliver.rs`; consult its pinned earlier Git source and the current coverage disposition.

<a id="coverage-e81117a99fd6"></a>
<a id="rustsoda-installsrcexecuters-1"></a>

Former source `rust/soda-install/src/execute.rs`; consult its pinned earlier Git source and the current coverage disposition.

<a id="coverage-5961e5da78a9"></a>
<a id="rustsoda-installsrcinputsrs-1"></a>

Former source `rust/soda-install/src/inputs.rs`; consult its pinned earlier Git source and the current coverage disposition.

<a id="coverage-0e3ccd741c53"></a>
<a id="rustsoda-installsrcocirs-1"></a>

Former source `rust/soda-install/src/oci.rs`; consult its pinned earlier Git source and the current coverage disposition.

<a id="coverage-42c03aba3ef0"></a>
<a id="rustsoda-installsrcsetuprs-1"></a>

Former source `rust/soda-install/src/setup.rs`; consult its pinned earlier Git source and the current coverage disposition.

<a id="coverage-cbece29a32ae"></a>
<a id="rustsoda-installsrcwizardrs-1"></a>

Former source `rust/soda-install/src/wizard.rs`; consult its pinned earlier Git source and the current coverage disposition.

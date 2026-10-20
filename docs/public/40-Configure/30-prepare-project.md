# Prepare a Project for Factory work

Factory work reuses the repository's existing Project. Preparation accepts exact setup source, supplies shared prerequisites, and verifies coder and reviewer access without granting them administrator privileges.

## Accept setup source and requirements

First, a repository maintainer with current native code-write authority selects the setup files and requirements at an exact source revision. Inspect the selected source and complete file set in Project settings, then explicitly accept them.

Inputs cover setup entry points, referenced configuration, required tools and versions, writable build paths, services, and readiness checks. Issue text and agent suggestions cannot approve setup. If no source files are selected, acceptance applies only to that source commit.

## Provide shared prerequisites

Next, a Project administrator reviews privileged effects and the shared environment: installed packages and runtimes, root-owned configuration, service definitions, mounts, ports, persistent data, and assigned Factory resources. Provide missing prerequisites through normal Project administration. Repository setup does not run as root or wheel and receives no provider or Factory publication grants.

## Establish role readiness

After acceptance and administrator preparation, choose **Start role preparation**. This action creates separate coder and reviewer preparation identities; inspect each role's readiness result. Each gets its own checkout and private setup state. Readiness confirms actual tools and versions, writable build and dependency paths, assigned service access, and a repository-specific smoke check. A coder's successful setup does not prove that a fresh reviewer checkout is ready.

Factory work can proceed when required role checks succeed for the accepted inputs and current Project state. The result applies to those inputs and that Project state; a running Project alone does not establish readiness.

```mermaid
flowchart TD
    accTitle: Separate Project preparation responsibilities
    accDescr: A maintainer accepts exact setup source, an administrator provides shared prerequisites, and coder and reviewer readiness are checked separately. Both required role checks must pass for current inputs and Project state.
    A["Maintainer accepts setup source"] --> B["Admin provides shared prerequisites"]
    B --> C["Check coder readiness"]
    B --> D["Check reviewer readiness"]
    C --> E{"Both roles ready?"}
    D --> E
    E -->|"Yes"| F["Prepared for Factory work"]
    E -->|"No"| G["Resolve failed check"]
```

Both roles need their own successful evidence; one role's result cannot stand in for the other. A changed accepted input or Project state requires the affected checks to be established again.

```doc-visual
{
  "placeholder": "Factory preparation pairs accepted source and administrator review with separate recorded coder and reviewer results.",
  "caption": "handbook-demo Project preparation across authentic views: 1 accepted source inputs, 2 the recorded operational approval, and 3 separate coder and reviewer evidence with their actual statuses. The article explains the administrator checks behind that approval.",
  "alt": "Factory preparation showing accepted source, a recorded operational approval, and separate coder and reviewer evidence",
  "production": {
    "method": "screenshot",
    "instructions": [
      "Use an approved documentation instance and the real handbook-demo Project; sign in as its repository environment owner or authorized operator for read-only capture.",
      "Navigate from the repository through its visible Settings page to Soda Project settings, then select the Factory tab.",
      "Use only an already-recorded preparation state with exact-source acceptance, a recorded operational approval, and distinct coder and reviewer evidence rows. The approval ID identifies a record; it does not display or prove every prerequisite check described in this article.",
      "Capture genuine scroll positions from the same recorded state if one 1280 by 900 view cannot keep accepted source, recorded operational approval and both role rows legible; use only results the application actually recorded.",
      "If the existing instance has no such records, stop and report that limitation; do not submit forms, approve, start role preparation, execute setup commands, or invoke a provider to stage the image. Do not reconstruct the installed-services/resources inputs that disappear after approval.",
      "Record the actual SodaOS release, Fountain revision, and capture date in the repository asset notes; use the light theme, capture at 1280 by 900, and check label readability at a narrow viewport."
    ],
    "edits": [
      "If combining the two scroll positions, place authentic crops from the same recorded preparation state in reading order and make the split explicit in the caption; do not synthesize or imply a result that is absent.",
      "Add at most three numbered markers: 1 for accepted source, 2 for the recorded operational approval, and 3 for separate coder/reviewer rows; explain each marker in the caption. Do not relabel the approval ID as visible evidence of individual prerequisite checks.",
      "Preserve real labels and recorded ready/not-ready values, mask instance-specific opaque IDs and credentials, and do not use color alone to distinguish roles or statuses."
    ]
  }
}
```

## Example: prepare a small Canlang repository

Use a repository containing the [Todo application](../30-Projects-and-development/70-canlang-development.md#start-with-an-application-and-examples) as `app.can`. This example needs the installed matched Canlang tools, a writable build directory in each role's own checkout, and no shared database or production credentials.

The repository's `setup.sh` creates only that private build directory:

```sh
#!/bin/sh
set -eu
mkdir -p build
```

Its `check.sh` checks and compiles the selected source without starting an application or writing shared data:

```sh
#!/bin/sh
set -eu
can check app.can
can compile --format=json app.can > build/app.json
```

Review and commit these files through the native repository workflow before accepting them. In Project settings, open Factory preparation:

1. In **Source commit**, select the exact approved repository revision, or leave it blank to resolve the approved base. In **Source paths, one per line**, select `setup.sh`, `check.sh` and `app.can`. In **Approved setup files, one per line**, include `setup.sh` and `check.sh`. In **Required tools, one per line**, include can. If a script reads any other configuration, select that file too.
2. Choose **Preview selected source**. Read the returned commit, complete paths, digests and actual file contents. Confirm there are no downloads, privileged effects, shared writes or external services in this example. Choose **Record exact-input acceptance** only for that reviewed selection.
3. A Project administrator checks that the installed Canlang tools are available to both role accounts and that each assigned checkout can create its own build directory. Record those actual observations through the operational approval controls. If tools are missing, ask the host administrator for [supported same-root maintenance](../50-Operate/60-project-maintenance.md); do not add sudo to the scripts. An approval record identifies this review, not proof of successful role setup.
4. Choose **Start role preparation** once to start the separate coder and reviewer preparations. Inspect each result against the same accepted source and operational approval. Both setup and check must succeed in each role's assigned checkout, producing its own `build/app.json`.

| Role result | Interpretation and next action |
| --- | --- |
| Coder ready; reviewer ready | Both can perform this repository's setup and compilation on the accepted inputs. Other Factory grants still apply. |
| Coder ready; reviewer cannot write build | Correct the reviewer's assigned checkout access, then repeat its preparation. Do not reuse the coder's writable directory or count the coder result twice. |
| Either role cannot execute can | Administrator supplies or refreshes the shared tools, then re-establishes affected readiness. |

If `app.can` or either script changes, its selected digest changes and affected readiness is invalidated. Review and accept the changed input before preparing again. Before changing shared tools, hold affected Factory work, settle active runs and credentials, maintain the retained root, then establish fresh role results. Human terminals, provider enrollment and a running Project do not substitute for this evidence.

## Prepare a repository that needs PostgreSQL

For a repository that uses a development database, follow the [Project service example](../30-Projects-and-development/60-project-services.md#example-a-postgresql-development-service). The administrator prepares the service on `127.0.0.1:55432`, retains human data in `handbook_member`, and assigns `handbook_coder` and `handbook_reviewer` as separate disposable databases with separate native credentials.

Include the repository's actual database check script and its referenced configuration in the accepted source selection. Record the required PostgreSQL client, endpoint and assigned database names as preparation requirements. Keep passwords in each role's private credential delivery, outside accepted files and results. The administrator reviews and applies the Compose and database-administration changes; role setup receives neither those credentials nor engine access.

Use the example's [Factory data-boundary check](../30-Projects-and-development/60-project-services.md#verify-the-assigned-factory-data-boundary) as the database portion of the repository smoke check, alongside the application's own checks. Run it separately as coder and reviewer. Each result must show the assigned user/database, a successful disposable write/read and denied connection privileges to the human and other role's databases. A running container or successful administrator query cannot replace those observations.

Keep test resets within the explicitly assigned database after its prior work has stopped. Re-establish role access and health after a service restart, and reaccept any changed authoritative service or setup input. The [restart example](../30-Projects-and-development/60-project-services.md#restart-and-check-the-same-data) also checks that the human's committed note survives.

## Inspect and recover preparation failures

If a prerequisite is missing, preparation reports what the administrator must provide and waits; it does not elevate an agent or replace the Project. If setup fails or times out, inspect the recorded step and partial effects before retrying. Retry a known safe step only after its prior processes have stopped. Do not blindly repeat shared migrations or delete human data. An uncertain request retains its original preparation identity for inspection and retry.

## Reaccept changes and hold for maintenance

Changing a selected setup file, effective referenced input, accepted requirement, or required human authority invalidates affected readiness. Reaccept the changed exact source and repeat relevant checks.

Before shared maintenance, hold affected Factory work, stop its runs, and settle credential return. Retain the same Project root, make the change, and record fresh readiness. A hold does not authorize rebuilding the environment or resetting persistent data. See [Project maintenance](../50-Operate/60-project-maintenance.md) for retained-root refresh.

Preparation does not enable or fund the Factory. Repository policy, operator capacity, provider sponsorship, and usage limits belong to [Factory authorization](20-factory-authorization.md). Human Join and terminal access are covered in [Projects and workspaces](../30-Projects-and-development/10-projects-and-workspaces.md) and [Workspace terminals](../30-Projects-and-development/20-workspace-terminals.md).

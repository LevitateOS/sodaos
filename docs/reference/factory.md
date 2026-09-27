# Bounded factory operator interface

`cmd/soda-factory` admits explicit work and executes the fixed factory lifecycle.
Product responsibilities belong to [architecture](../architecture/overview.md);
authority belongs to [trust](../architecture/trust.md). Persistent
[Projects](../product/projects.md) use a separate runtime and lifecycle.

## Configuration and admission

Run the command as an unprivileged Linux operator with rootless Podman, user
namespaces, cgroup v2 and SELinux. Keep configuration, credential files and state
private. Create the configured root, `workspaces` and `publications` directories
with mode `0700` before starting. Configuration and token files use mode `0600`.

The configuration selects one private repository by numeric ID, the authorizing
human's token, two separate non-administrator bot accounts and their token files,
a Forgejo Actions workflow, and `workspace.Config`. Bot tokens can be restricted
to the target repository with `write:repository` and `write:issue`. Forgejo 15
excludes `read:user` from repository-restricted tokens; the operator pins bot
logins and Soda resolves their native profiles using the human token.

The configuration also names `identity_socket` (the private runtime socket),
`connection_id` and `project_id`. Run factory and broker as the same Unix operator
in the same rootless Podman context. The authorizing human remains the work actor;
connection sponsorship does not give the worker Forgejo write authority.
Worker Git setup belongs to
[Forgejo account custody](credentials.md#forgejo-account-custody).

The workspace configuration pins worker and proxy image IDs, the complete Codex
package, actual executable SHA256 (`harness_sha256`) and version, the model,
allowed TLS hostnames and CPU, memory, process and writable tmpfs limits. Soda
checks executable bytes and `--version` before leasing credentials. The workspace
root belongs to the controller's `workspaces` directory. Source and candidate
bundles are limited to 4 MiB.
The proxy has a quarter-CPU, 128 MiB memory and 32-process limit. Capacity checks
leave CPU, memory and disk headroom for human use.

The [Identity Broker](credentials.md#identity-broker) owns subscription credentials.
Factory waits for its selected connection within the existing execution deadline;
it cannot create a second refresh writer. The lease and exact OCI binding are
recorded before credentials enter the worker. Authentication travels through stdin
into bounded tmpfs, never a host credential-directory mount. Before publication,
Soda freezes the worker, captures updated CLI state and terminates the whole
container before returning that state to broker custody. Interrupted returns are
reconciled by exact lease ID; uncertain streams require reauthentication. Old
`credential_home` configuration and local credential stream locks are removed.

SQLite state and logs use fresh per-run paths; execution is ephemeral and ignores
user configuration. The writable budget covers checkout, scratch, authentication,
temporary files and shared memory; automatic writable mounts are disabled.

Build the operator binary with the repository's pinned Go toolchain:

```sh
go build -o .artifacts/soda-factory ./cmd/soda-factory
.artifacts/soda-factory --config /home/soda-tester/factory/config.json admit 12 issue-12-first-attempt
.artifacts/soda-factory --config /home/soda-tester/factory/config.json run ATTEMPT_ID
.artifacts/soda-factory --config /home/soda-tester/factory/config.json status ATTEMPT_ID
.artifacts/soda-factory --config /home/soda-tester/factory/config.json cancel ATTEMPT_ID
.artifacts/soda-factory --config /home/soda-tester/factory/config.json recover
.artifacts/soda-factory --config /home/soda-tester/factory/config.json report ATTEMPT_ID
```

Admission records the issue objective, current default-branch revision, authorizing
human and configuration digest. Repeating the same delivery returns the existing
attempt. Labels and agent comments do not admit work or reset budgets. A new human
attempt requires a new delivery ID and completion of earlier cleanup.

## Execution and publication

Each stage consumes its execution before resources are allocated. Podman's native
container timeout enforces the recorded deadline if the controller disappears.
Implementation, repair and review receive separate fresh checkouts and principals.
The selected workflow must evaluate each candidate PR commit once; use explicit
`pull_request` `opened` and `synchronize` triggers rather than review/comment
triggers. Soda observes stock Forgejo Actions and rejects multiple evaluations or
results for a different commit. It does not create another CI queue.

Only the publisher holds Forgejo write credentials. It imports a bounded bundle
into a fresh bare repository, disables hooks and inherited Git configuration,
checks ancestry, rejects changes under `.forgejo/` or `.github/workflows/`, and
rejects known before/after worker credential literals in decoded Git objects
reachable beyond the admitted base, including deleted historical files and commit
messages. Each publication scans at most 10,000 objects, 4 MiB per decoded object
and 32 MiB total decoded content; exceeding a limit denies publication. This
literal check does not prevent intentional encoding or exfiltration by code that
can read the injected credential. After those checks, the publisher
pushes only `soda/factory/ATTEMPT_ID` with an exact expected-revision lease.
Review can submit findings for its assigned PR and commit; it has no candidate
publication operation. Human merging remains a Forgejo operation.

Cancellation withdraws authority under the publication lock, waits for the running
controller to stop, and reconciles surviving recorded resources. Status,
cancellation and recovery can operate from local state without Forgejo or provider
availability. Recovery ends interrupted work in `needs-human`; it does not resume
an old agent conversation or silently create another attempt. An online `run` of
an already terminal attempt reports its existing outcome without executing an agent.
`report` retries Forgejo notification explicitly after offline cleanup. Reserved
`soda:` outcome labels describe the latest admission on an issue and each PR
individually; reporting an older attempt cannot overwrite the current issue state.
Human labels are preserved. Bounded structured results are retained privately
without raw transcripts; known enrolled credential strings are rejected before
retention and publication.

## Development checks

The opt-in native checks use a private task-owned fixture; they are development
integration evidence rather than installed appliance qualification. The workspace
lifecycle check uses synthetic credential state; subscription execution goes
through the controller, which owns serialized credential persistence:

```sh
SODA_FACTORY_NATIVE_CONFIG=/absolute/private/workspace-config.json \
  go test ./internal/host/workspace -run TestNativeLifecycle -count=1
SODA_FACTORY_CONTROLLER_CONFIG=/absolute/private/controller-config.json \
  go test ./internal/factory/control -run TestNativeWithdrawAndRecover -count=1
SODA_FACTORY_PUBLICATION_CONFIG=/absolute/private/publication-config.json \
  go test ./internal/host/publish -run TestNativePublication -count=1
```

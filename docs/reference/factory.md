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

The workspace configuration pins worker and proxy image IDs, the complete Codex
package and version, the supported model, a dedicated credential home, allowed
TLS hostnames, and worker CPU, memory, process and writable tmpfs limits. Its root
must be the controller root's `workspaces` directory. The proxy has a quarter-CPU,
128 MiB memory and 32-process limit. Capacity checks leave CPU, memory and disk
headroom for human use. Source and candidate bundles are limited to 4 MiB.

The dedicated Codex credential home contains CLI-maintained account authentication,
not implementation conversations. SQLite state and logs use fresh per-run paths;
execution uses `--ephemeral` and ignores user configuration. Enrollment must also
redirect SQLite state outside the credential home. Soda serializes this credential
stream and refuses another execution while a claimed workspace remains unclean.
Supported private subscription automation and renewal conditions are owned by the
[Codex account-auth workflow](https://learn.chatgpt.com/docs/auth/ci-cd-auth).

Build the operator binary with the repository's pinned Go toolchain:

```sh
go build -o .artifacts/soda-factory ./cmd/soda-factory
.artifacts/soda-factory --config /home/soda-tester/factory/config.json admit 12 issue-12-first-attempt
.artifacts/soda-factory --config /home/soda-tester/factory/config.json run ATTEMPT_ID
.artifacts/soda-factory --config /home/soda-tester/factory/config.json status ATTEMPT_ID
.artifacts/soda-factory --config /home/soda-tester/factory/config.json cancel ATTEMPT_ID
.artifacts/soda-factory --config /home/soda-tester/factory/config.json recover
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
pushes only `soda/factory/ATTEMPT_ID` with an exact expected-revision lease.
Review can submit findings for its assigned PR and commit; it has no candidate
publication operation. Human merging remains a Forgejo operation.

Cancellation withdraws authority under the publication lock, waits for the running
controller to stop, and reconciles surviving recorded resources. Status,
cancellation and recovery can operate from local state without Forgejo or provider
availability. Recovery ends interrupted work in `needs-human`; it does not resume
an old agent conversation or silently create another attempt. An online `run` of
an already terminal attempt reports its existing outcome without executing an agent.

## Development checks

The opt-in native checks use a private task-owned fixture; they are development
integration evidence rather than installed appliance qualification:

```sh
SODA_FACTORY_NATIVE_CONFIG=/absolute/private/workspace-config.json \
  go test ./internal/host/workspace -run TestNativeLifecycle -count=1
# Add SODA_FACTORY_NATIVE_AGENT=1 only when exercising the enrolled subscription.
SODA_FACTORY_PUBLICATION_CONFIG=/absolute/private/publication-config.json \
  go test ./internal/host/publish -run TestNativePublication -count=1
```

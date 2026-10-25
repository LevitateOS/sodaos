# soda-installed-probes

Run scoped native checks from a selected client against an installed appliance
or project fixture. Release developers and operators use this support tool to
observe trusted HTTPS, developer access, personal Git and nested workloads.
The implementation is in [internal/acceptance](../../internal/acceptance/README.md).

## Choose a probe

| Command | Inputs and effects |
| --- | --- |
| `service-https ORIGIN CA_FILE` | Make a trusted HTTPS request and report status without capturing the body or following redirects. |
| `developer-access ROOT_DIR` | Read a prepared access request, exercise SSH/PTY/SCP/SFTP and authority boundaries, and write retained access results. |
| `personal-git prepare FIXTURE_DIR` | Prepare private keys and retained personal Git fixture state. |
| `personal-git exercise FIXTURE_DIR` | Exercise the fixture's native Git workflow and record results. |
| `personal-git unlock FIXTURE_DIR` | Perform the fixture's explicit key-unlock phase. |
| `workload-access FIXTURE_DIR` | Exercise configured workload network/database access and write results. |
| `workload-exec FIXTURE_DIR` | Exercise owner workload execution and member denial through SSH. |
| `factory-context-status SCOPE_FILE` | Inside the selected guest, read protected Factory scope and return canonical context-selection metadata for one stdin run/assignment. No action, model call or database migration. |

The fixture probes require their existing private request/configuration files,
trusted SSH inputs and reachable target. Personal Git and workload probes also
require `SODA_NATIVE_VALIDATE=soda-test` and their expected fixture identities.
Read the selected probe's source and prepared fixture before invoking it;
these commands can write local state and execute remote Git or workload actions.

`factory-context-status` is a guest-side read-only observation. Its scope must be
an absolute `/home` file, root-owned mode `0600`, in a root-owned `0700` directory
with no symlink or writable ancestors. The bounded stdin object contains only
`run_id` and `assignment_id`. Scope binds the owner, operation, exact opening
policy command ID, repository/issue/Project, accepted input, initial source,
standing coder/reviewer preparations, connection, installed source/payload and
independent tool source. It uses canonical Store/control methods with read-only
database transactions; raw run bodies, output and credentials are never returned.
Implementation and review retain their exact parent, publication owner, candidate
preparation and recorded context-choice bindings.

Build with the selected full clean driver revision as the linker value
`github.com/levitateos/sodaos/internal/acceptance.FactoryContextToolRevision`.
An unpinned build refuses. The installed source/payload may differ from that tool
revision. Collector runtime preparation pins the binary and scope bytes separately;
this command does not deploy itself or grant current-context authority. Normal
owner actions still pass canonical eligibility/admission through Factory control.

## Observe configured HTTPS

Run from the repository root with the installed Go toolchain, an actual HTTPS
origin and an absolute trusted PEM CA file:

```sh
go run ./tools/soda-installed-probes service-https \
  https://soda.example /home/soda-tester/private/soda-ca.pem
```

The CA must be a regular file, not a symlink, and cannot be writable by group
or others. The probe uses only that CA, bypasses proxies and has a 15-second
HTTP timeout. It accepts HTTP status 200 through 399 after successful TLS
validation and prints the observed status. This result does not establish
browser login or authentication.

The binary exits `0` for a successful probe, `1` for failure or incomplete
work, and `2` for a usage rejection represented by `UsageError`. `go run`
reports a child failure as `exit status N`. There is no `--help` action;
invalid dispatcher arguments print the accepted subcommand names.

See [native support](../../docs/development/native-support.md#installed-substrate-and-retained-integrations)
for installed observations and [testing](../../docs/development/testing.md)
for acceptance scope and evidence.

# Installed support probes

This package runs outside support checks from an authorized client against
selected Soda fixtures. It owns installed probe mechanics and bounded native
process groups, rather than appliance runtime behavior.

## Operator entry point

Use the [`soda-installed-probes` tool](../../tools/soda-installed-probes/README.md)
for these existing commands:

| Command | Observation and effects |
| --- | --- |
| `service-https ORIGIN CA_FILE` | Configured-origin TLS and HTTP status using an independently trusted CA. |
| `developer-access ROOT_DIR` | Declared member SSH/PTY/SCP/SFTP, sudo and cross-user key-denial checks; retains new probe state. |
| `personal-git prepare\|exercise\|unlock FIXTURE_DIR` | Prepare encrypted project-local keys/agents, unlock retained keys or exercise personal Git. |
| `workload-access FIXTURE_DIR` | Real member/client HTTP and committed PostgreSQL probe writes. |
| `workload-exec FIXTURE_DIR` | Existing workload exec, UID and ordinary-member denial checks. |

For the TLS observation alone:

```sh
/path/to/soda-installed-probes service-https \
  https://soda.example.test /home/operator/trusted-ca.pem
```

Replace the origin and CA with the selected target's verified inputs. The CA
must be an absolute regular file without group/other write permission. The TLS
probe does not follow redirects, authenticate, capture bodies or use a proxy.

## Go consumers and scope

Go callers in this module use `RunServiceHTTPS`, `RunDeveloperAccess`,
`RunPersonalGit`, `RunWorkloadAccess` or `RunWorkloadExec` with argument slices
and an output writer. [`StartCommand`](process.go) owns a new native process
group; its `Process` exposes `Wait`, `Done` and bounded `Stop`. A timed-out
`Wait` does not itself stop the process, so callers retain cleanup responsibility.

Fixture probes require their private inputs, pinned SSH identity and native
tools. Personal Git and both workload probes specifically require
`SODA_NATIVE_VALIDATE=soda-test`; they are tied to the retained fixture contract.
Several probes write client/project files or test database data, so obtain the
action and target grants described by the
[native support guide](../../docs/development/native-support.md) before running.
Failures retain partial state; they do not authorize blind retries or cleanup.

See [testing](../../docs/development/testing.md) and
[local fixture access](../../docs/guides/local-testing.md) for scope and inputs.
Each result proves its stated observation rather than whole-product acceptance.

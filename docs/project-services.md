# Native project workloads — bounded installed evidence

The candidate remains nested Podman, not a host-socket backend. Rocky contains
Podman and podman-compose 1.6.0. The project-local engine uses
`/run/soda-podman/podman.sock`; project-native root/wheel members administer it with
ordinary native commands. Other members consume service endpoints, not its
rootful API. Giving that API to ordinary members would grant project-root power.
The [Project OS baseline](project-os.md) consolidates shared state, native authority,
startup/terminal distinctions and required additions to existing writable roots.

## Profile and catalog extension

The selected Fedora and KDE profiles must preserve this native workload ownership,
shared persistent service data and ordinary endpoint access. A desktop browser uses
the project's real service endpoints; opening it does not copy a database or install
a second workload engine. Revalidate changed package/runtime behavior per profile;
the Rocky evidence below is not automatic Fedora/KDE acceptance.

The [marketplace candidate](services-and-ai-plan.md#1-services-marketplace) is a
single appliance-wide, operator-managed catalog, including Adminer as an appliance
utility. Its data, app authentication and private ingress are separate from project
roots and this nested engine. If project-local catalog installation is later chosen,
revise that scope around this engine and existing project authority before adding it;
do not implement both candidates now. Neither surface grants ordinary members the
rootful project engine or host Podman socket. AI job isolation remains the runner's
separately verified boundary, not borrowed project credentials.

## Service and permission corrections

Installed testing found two concrete defects: `CONTAINER_HOST=` selected remote
mode even when empty in Podman 5.8.2, and a directly created API socket was mode
0600 despite the service umask. Current source unsets that variable and uses
native systemd socket activation with root:wheel mode 0660. Project initialization
creates the socket directory root:wheel 0750; the image enables the socket unit.
The service uses the activation FD, not a second listener. It keeps its native
root:root process identity; only the socket is root:wheel. Using wheel as the
engine's primary GID caused process-namespace access failures during exec.
Candidate `c96c108` now packages these corrections and passed fresh-project
creation with no manual integration patch. Earlier project roots keep their
recorded versions/explicit corrections; this is not an automatic image update.

## Native profile and installed evidence

The outer project retains its automatic user namespace, private cgroup/network
namespaces, `/dev/fuse`, disabled SELinux labeling and the host's default seccomp
policy. No privileged parent, host PID/network namespace, appliance engine socket
or arbitrary host mount was introduced. Storage uses fuse-overlayfs and inner
workload cgroups remain disabled.

The first real Compose image pull/build succeeded, but its default nested bridge
failed at startup with `netavark: Netlink error: Operation not permitted`. Native
inspection confirmed the project owns its network namespace but lacks NET_ADMIN.
Candidate `952f3b3` adds that bounded capability alongside SYS_ADMIN/MKNOD and was
built, installed and used for a fresh fixture. Podman 5.8.4 cannot update that
capability in place. Existing projects were not
replaced, exported or recreated to apply it. The fresh fixture then exposed read-only network sysctls. Current initialization
binds only `/proc/sys/net` from a private temporary proc mount, retaining read-only
kernel/fs/vm controls. After this project-local correction, ordinary bridge HTTP
and PostgreSQL operations passed from both users and infra. Corrected cold project
startup and the approved VM reboot now have full declared-state preservation
comparisons. The socket must not depend on init before sockets.target/basic.target;
the activated service retains that dependency. Existing workloads were explicitly
started after lifecycle operations, not recreated or automatically resurrected.

Different-UID exec exposed another concrete permission requirement. The approved
`c96c108` creation profile adds **SYS_PTRACE in the project's own user namespace**,
alongside SYS_ADMIN/MKNOD/NET_ADMIN. This lets the project-root engine enter
PostgreSQL's UID-999 process namespace; it does not grant appliance ptrace or a
host PID view. Fresh native testing passed default-root, explicit PostgreSQL-user,
SQL and PTY exec, while Bob remains denied the root:wheel engine socket. All three
older roots were preserved without capability retrofits. This profile remains
for trusted teams, not a claim of hostile-tenant isolation or complete cgroup /
SELinux confinement. Its capability follow-up did not repeat a VM reboot.

In the earlier retained project, two diagnostic workloads run through the same nested engine with
native `--network host`: here **host means the project's network namespace**, not
the appliance's. HTTP source bind-mount changes and committed PostgreSQL data were
verified by Alice, Bob and the routed infra client. This establishes useful nested
runtime evidence, not evidence for the newer default bridge profile or complete
isolation. It is not an alternate appliance backend; newer bridge and lifecycle
results are recorded separately above and in the handoff.

## Ordinary example and secrets

`tests/fixtures/workload/` contains the image build, source mount, database volume
and intended HTTP/PostgreSQL ports. Its Compose file remains the bridge-network
candidate; it was not silently switched to the diagnostic network mode.

Before running it, the project administrator creates the native `soda-example-db`
secret from a restricted disposable password file:

```sh
podman secret create soda-example-db /absolute/private/password-file
podman compose up -d --build
```

PostgreSQL reads `/run/secrets/soda-example-db` through `POSTGRES_PASSWORD_FILE`.
The external secret uses UID/GID 999 and mode 0400 for the selected postgres image.
The installed podman-compose source confirms external-secret mode/ownership
support; its file-secret path does not implement those fields. This avoids
passing the password through Compose's generated Podman argv. Do not print
expanded secret configuration, use real production credentials in fixtures or
commit password files. Normal image/container/volume/secret storage remains owned
by native Podman inside `/var/lib/containers/storage` and its native state paths.
No Soda artifact/secret database was added.

The installed workload test bounds HTTP/PostgreSQL readiness after detached
startup. `workloads.sh check` performs only readiness/SQL observations against
already started workloads; it does not replay up/build after an uncertain result.

See the [handoff](implementation-status.md#accepted-native-evidence)
for exact resources, revisions, failures and retained state. Do not remove failed
containers/volumes or run `down -v` as a repair shortcut.

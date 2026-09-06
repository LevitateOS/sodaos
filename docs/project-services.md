# Native project workloads — unvalidated candidate

The implemented candidate is nested Podman, not a second host-socket backend. Native Podman is installed in the Rocky userspace with podman-compose 1.6.0 (its source metadata requires Python >=3.9). The project-local systemd service owns the shared workload engine at `/run/soda-podman/podman.sock`.

The **project owner/wheel** can administer that engine with ordinary commands, such as `podman compose up -d --build`. Other project members consume its shared service endpoints; granting them access to a rootful project engine would also grant project-root authority, so the socket is not world/group-project writable. This is the normal project-admin boundary, not a per-user engine or private-resource selector.

Because the engine is inside the development environment, its bind-mount paths and build contexts refer to the same project filesystem. Workloads store native image/container/volume data under `/var/lib/containers/storage` in the retained project rootfs. Services use ordinary published project-IP ports. Developers manage changes and cleanup themselves.

## Host launch assumptions requiring native evidence

The host creates the persistent development container with a delegated automatic user namespace, private cgroup namespace, `/dev/fuse`, SYS_ADMIN/MKNOD in that namespace and disabled SELinux labeling for the nested mount setup. It does **not** use `--privileged`, a host PID/network namespace, a host engine socket or arbitrary host mounts. The project service uses fuse-overlayfs and disables inner workload cgroups. Retain the host's default seccomp policy rather than disabling it speculatively.

The capability/storage candidate is informed by Red Hat's upstream article [Podman inside a container](https://www.redhat.com/en/blog/podman-inside-container), not proved on this selected CoreOS/Rocky combination. The combination of user-namespace mapping, systemd, cgroups, fuse and seccomp is a high-priority native validation risk. If it fails, change the concrete mechanism based on observed evidence; do not silently weaken to privileged host access or claim a scoped host fallback has already been implemented.

The ordinary example is `tests/fixtures/workload/`. It exercises a real image build, a source bind mount, a persistent database volume, and HTTP/PostgreSQL TCP ports. Its password is an operator-supplied disposable test value, not a production credential or new repository schema. Native workload checks are authored but have not run.

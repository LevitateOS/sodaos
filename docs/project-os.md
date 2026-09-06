# Project userspace implementation candidate

Rocky Linux 9.6 (tag existence inspected in Quay's registry API), mise 2026.9.1 (upstream release asset names inspected), systemd and OpenSSH. Build recipes download only when explicitly built later; the recipe verifies mise against its release checksums. [Tea and GitHub CLI](project-clis.md) are project tools too, with separate native personal authentication. The project image now consumes matching-native Tea output and uses the repository-root build context through `scripts/build-native.sh`.

A project is created once and retained. Account databases, SSH host keys, `/home`, shared files, system packages and service data live in its persistent writable container root filesystem. Normal systemd startup must use `podman start` on that existing container, never `run --replace`, `--rm`, `rm` or a recreating Quadlet. Deletion/replacement/update is outside current scope. Losing host container storage is not covered by this persistence claim.

The image contains a stateless native account setup command, not a resident guest agent. Accounts get access to `/srv/project/shared` through `~/shared`; their home repository checkouts are normal Git clones. SSH keys are root-owned under `/etc/ssh/authorized_keys`. Project owners receive project-local wheel/sudo authority, not host privileges. Colliding/unassociated native accounts fail rather than being reassigned.

Host networking, user namespaces, cgroups and nested containers are implementation candidates to be exercised later. Image recipes and authored installed checks have not run. Native SSH/password-lock behavior, service initialization and retention across stop/start/reboot require actual evidence.

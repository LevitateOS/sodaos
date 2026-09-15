# soda-candidate

Interactive wrapper around the admitted `soda-build` controller for producing
Soda release candidates and installation media. It asks, it shows progress,
it never signs.

Contract and command effects live in the owning guide:
[Native support](../../docs/development/native-support.md). This file is only
the entry point.

## Run

Start from the repository root: the controller binds its source to the
working directory. Anything else is refused before privilege is touched.

```sh
cd ~/Projects/sodaos
bash scripts/setup-soda-candidate.sh
sudo soda-candidate
```

The script builds both tools from committed source, installs the wrapper
where sudo resolves it (`/usr/sbin`, since `secure_path` excludes
`/usr/local/bin`), admits the controller, creates the worker directories,
and writes the restricted worker config plus a fixture-only media authority.
Fixture scope only: it never creates qualification or signing configs.
Manual equivalent of each step is printed by the script as it goes.

One overview screen shows every choice (mode, output with freshness status,
controller, worker config, fixture URL or protected configs). Type a field
number to edit it, `go` to start, `quit` to abort. Nothing runs before an
explicit start. Flags only pre-seed answers:

```sh
soda-candidate --mode media --rootfs-base-url http://FIXTURE:PORT
soda-candidate --non-interactive --mode candidate \
  --controller /ADMITTED/soda-build \
  --worker-config /RESTRICTED/worker.json \
  --out /OUTPUT_PARENT/UNIQUE
```

## Modes

- `candidate`: development build only, no media or signing.
- `media`: development installer ISO from a fixture rootfs URL. Never
  release-qualified, never distribution-ready.
- `production`: protected qualification, optional final signing. Publication
  stays separately grant-scoped.

## The rootfs URL, in one paragraph

The ISO is only the boot menu. The rootfs is the actual operating system
disk image (gigabytes), which the installer downloads during installation.
The URL is the pickup address where the installer is told to fetch it. For
local builds the setup script creates `/var/lib/soda-rootfs` and the TUI
prefills `http://127.0.0.1:8080`; after the build, copy the produced
`*-rootfs.img` into that folder and serve it. There is nothing to look up.

## Never

Admit workers, sign payloads, publish images, or accept production keys as
fixture input. Exit 2 means what the controller means: qualification passed
but final signing is not connected, so there is no qualified release.

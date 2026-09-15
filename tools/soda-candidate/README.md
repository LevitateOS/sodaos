# soda-candidate

Interactive wrapper around the admitted `soda-build` controller for producing
Soda release candidates and installation media. It asks, it shows progress,
it never signs.

Contract and command effects live in the owning guide:
[Native support](../../docs/development/native-support.md). This file is only
the entry point.

## Run

```sh
GOTOOLCHAIN=local go build -o /usr/local/bin/soda-candidate ./tools/soda-candidate
soda-candidate
```

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

## Never

Admit workers, sign payloads, publish images, or accept production keys as
fixture input. Exit 2 means what the controller means: qualification passed
but final signing is not connected, so there is no qualified release.

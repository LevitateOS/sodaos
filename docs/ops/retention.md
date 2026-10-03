# Attempt retention (D2)

The 153G under `.artifacts/releases/isolated/` is three retained media
attempts (iso-27, iso-28, iso-29 at 51G each), one per live install-test
guest. Retention is deliberate, not leakage: each directory binds the
guest to its exact inputs and receipts.

## What is retained per attempt, and why

- `artifacts/candidate.json`, `artifacts/media/media.json`: the exact
  ISO/rootfs hash, size, and URL bindings the guest installed from.
- `evidence/build.json`: the worker result receipt for the attempt.
- `logs/build.log`, `logs/timing.log`, `logs/media-events.jsonl`: the
  only record of how the attempt behaved (all 0600, soda-build-worker).
- `work/`: host/Forgejo build contexts. Largest on disk; needed only
  for post-mortem inspection of a live attempt.
- `inputs/`: source snapshots the attempt consumed.

## Retention triggers

- Guest alive (soda-iso27/28/29 running): retain everything. The QCOW2,
  installer ISO, rootfs image, and attempt directory form one unit.
- Guest retired by the owner: the attempt directory becomes a GC
  candidate under `docs/ops/gc-policy.md` (inventory -> grant -> prune).
- Superseded attempts with no guest (today: iso-26, 213M): candidates
  now, pending the standing owner-grant rule. Not pruned by this lane.

## Served images

`/home/libvirt/images` holds the live QCOW2s (0600 qemu:qemu),
installer ISOs, and three hash-named `*-rootfs.img` files (0644).
Installer-to-rootfs pairing by mtime: iso27/d086ae.., iso28/643730..,
iso29/ab4531... . Keep all three rootfs files while any guest may
reinstall from them; the D3 replacement serves exactly this set.

No retention state was changed by this lane: investigate, document,
never touch retained evidence.

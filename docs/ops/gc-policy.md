# Worker storage GC policy (D1)

Measured 2026-10-03 (read-only `du`; no pruning performed by this lane):

- `.artifacts/releases/isolated/iso-27`, `iso-28`, `iso-29`: 51G each
  (153G total). Each is the retained build attempt behind a live libvirt
  guest (soda-iso27/28/29). Never prune while its guest exists.
- `.artifacts/releases/isolated/iso-26`: 213M. Superseded attempt; its
  guest is gone, but pruning still needs an owner grant (see below).
- `.artifacts` total: 170G. `/home` is 44% used (464G free): no
  capacity emergency, so GC is hygiene, not rescue.
- `/home/libvirt/images`: 35G on disk (sparse), 3x 42.9GB apparent live
  QCOW2 + installer ISOs + three `*-rootfs.img` files. Installer rootfs
  files are served over HTTP from the rootfs-only `/home/soda-rootfs` by
  `soda-rootfs-server.service` (see D3); never prune live guest disks
  or the rootfs files their installers reference.
- `/var/lib/soda-candidate-home`: ~2.1G readable (Go/Bun/Playwright
  worker caches, partly soda-build-worker-owned). Warm caches are what
  let the isolated worker build at all; pruning them only costs rebuild
  time and network.

## Rules

1. Retained evidence for a live guest is never a GC candidate. That is
   the guest's QCOW2, its installer ISO, its rootfs image, and its
   `iso-NN` attempt directory (candidate/media receipts, evidence, logs).
2. Superseded attempts (guest retired or attempt abandoned) become
   candidates only after the owner confirms the guest is gone.
3. Caches (`go-build`, `go-mod`, Bun, Playwright browsers) are never
   pruned to "save space" without a measured capacity reason; they are
   small (~2G) and expensive to rewarm.
4. Every prune is inventory -> explicit owner grant -> prune -> verify.
   This lane performs inventory only. The grant names exact paths.

## Prune procedure (owner runs, after grant)

```sh
# 1. Re-inventory (read-only):
sudo du -sh .artifacts/releases/isolated/iso-* /home/libvirt/images
virsh list --all   # confirm the guest behind each candidate is gone
# 2. Prune exactly the granted paths, e.g.:
sudo rm -rf .artifacts/releases/isolated/iso-26
# 3. Verify: re-run the inventory and confirm only granted paths changed.
```

Removing a superseded attempt directory is permanent: receipts in it
(`candidate.json`, `media.json`, `build.json`) are the only copies of
that attempt's bindings. Archive before deleting if the attempt matters.

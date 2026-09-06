#!/bin/bash
# Manage the isolated x86_64 test VM prepared under .artifacts/test-vm.
# Never installs on the builder, deletes a disk, or changes host networking.
set -euo pipefail
cd "$(dirname "$0")/.."
vm="$PWD/.artifacts/test-vm"
action=${1:-status}
shift || true
qemu=${QEMU:-/usr/libexec/qemu-kvm}
ssh_args=(-p 22220 -i "$vm/operator" -o IdentitiesOnly=yes
  -o StrictHostKeyChecking=yes -o "UserKnownHostsFile=$vm/known_hosts")

case "$action" in
  start)
    [[ $(uname -s) == Linux && $(uname -m) == x86_64 && -r /dev/kvm && -w /dev/kvm ]] || {
      echo 'Native x86_64 Linux with KVM access required' >&2; exit 1;
    }
    for file in disk.qcow2 soda.ign operator known_hosts; do
      [[ -f "$vm/$file" ]] || { echo "Missing $vm/$file; see docs/local-testing.md" >&2; exit 1; }
    done
    umask 077
    exec 9>"$vm/start.lock"
    flock -n 9 || { echo 'Another VM start is in progress' >&2; exit 1; }
    if [[ -f "$vm/qemu.pid" ]] && kill -0 "$(<"$vm/qemu.pid")" 2>/dev/null; then
      echo 'Test VM is already running'; exit 0
    fi
    "$qemu" -name soda-test -machine q35,accel=kvm -cpu host -smp 4 -m 8192 \
      -drive "if=virtio,format=qcow2,file=$vm/disk.qcow2" \
      -fw_cfg "name=opt/com.coreos/config,file=$vm/soda.ign" \
      -nic user,model=virtio-net-pci,hostfwd=tcp:127.0.0.1:22220-:22 \
      -display none -serial "file:$vm/console.log" -monitor none \
      -daemonize -pidfile "$vm/qemu.pid" 9>&-
    echo 'Started soda-test (4 vCPU, 8 GiB RAM). SSH: scripts/test-vm.sh ssh'
    ;;
  status)
    if [[ -f "$vm/qemu.pid" ]] && kill -0 "$(<"$vm/qemu.pid")" 2>/dev/null; then
      printf 'soda-test is running (PID %s); SSH at 127.0.0.1:22220\n' "$(<"$vm/qemu.pid")"
    else
      echo 'soda-test is not running'; exit 1
    fi
    ;;
  ssh)
    exec ssh "${ssh_args[@]}" root@127.0.0.1 "$@"
    ;;
  tunnel)
    echo 'Keep this running: Forgejo http://localhost:23000; Cockpit https://localhost:29090'
    exec ssh "${ssh_args[@]}" -o ExitOnForwardFailure=yes -NT \
      -L 127.0.0.1:23000:127.0.0.1:3000 \
      -L 127.0.0.1:29090:127.0.0.1:9090 root@127.0.0.1
    ;;
  web-tunnel)
    echo 'Keep this running: Soda https://localhost:24443; Forgejo https://localhost:24444'
    exec ssh "${ssh_args[@]}" -o ExitOnForwardFailure=yes -NT \
      -L 127.0.0.1:24443:127.0.0.1:24443 \
      -L 127.0.0.1:24444:127.0.0.1:24444 root@127.0.0.1
    ;;
  console)
    exec tail -n 80 -f "$vm/console.log"
    ;;
  *)
    echo 'usage: scripts/test-vm.sh [start|status|ssh [COMMAND...]|tunnel|web-tunnel|console]' >&2
    exit 2
    ;;
esac

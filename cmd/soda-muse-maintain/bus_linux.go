//go:build linux

package main

import "context"

// --wait/--pipe needs the native system bus, not just systemd's private socket.
// Installing this demonstrated prerequisite belongs only to explicit maintenance.
const busScript = `set -eu
if ! rpm -q dbus-broker >/dev/null; then dnf -y install dbus-broker; fi
systemctl start dbus.socket
systemd-run --quiet --wait --pipe --collect /usr/bin/true
`

func ensureSystemBus(ctx context.Context, target observation) error {
	if err := confirmProject(ctx, target); err != nil {
		return err
	}
	_, err := podman(ctx, nil, "exec", "--user", "0:0", target.ID, "/bin/sh", "-ceu", busScript)
	return err
}

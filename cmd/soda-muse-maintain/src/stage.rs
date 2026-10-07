use std::os::unix::io::RawFd;
use std::time::Instant;

use super::command::{podman, podman_streamed};
use super::filesystem::Tool;
use super::project::{confirm_project, Observation};

// --wait/--pipe needs the native system bus, not just systemd's private socket.
// Installing this demonstrated prerequisite belongs only to explicit maintenance.
const BUS_SCRIPT: &str = r#"set -eu
if ! rpm -q dbus-broker >/dev/null; then dnf -y install dbus-broker; fi
systemctl start dbus.socket
systemd-run --quiet --wait --pipe --collect /usr/bin/true
"#;

// Destinations are fixed by the installed project interface, never repository input.
const DESTINATIONS: [&str; 3] = [
    "/usr/local/bin/muse",
    "/usr/local/bin/soda-identity-compose",
    "/usr/local/libexec/soda/muse",
];

pub(crate) const INSTALL_SCRIPT: &str = r#"
set -eu
safe_parent() {
 path=$(dirname "$1")
 while [ "$path" != / ]; do
  test ! -L "$path"
  if test -e "$path"; then test -d "$path"; fi
  path=$(dirname "$path")
 done
}
for target in "$@"; do
 safe_parent "$target"
 test ! -L "$target"
 if test -e "$target"; then test -f "$target"; fi
 done
for target in "$@"; do mkdir -p "$(dirname "$target")"; done
stage=$(mktemp -d "$(dirname "$3")/.soda-muse-maintain.XXXXXXXX")
trap 'rm -rf -- "$stage"' EXIT
tar --extract --file=- --directory="$stage" --no-same-owner
chmod 0755 "$stage/muse" "$stage/soda-identity-compose" "$stage/muse-native"
mv -T -- "$stage/muse" "$1"
mv -T -- "$stage/soda-identity-compose" "$2"
mv -T -- "$stage/muse-native" "$3"
"#;

pub(crate) fn ensure_system_bus(target: &Observation, deadline: Instant) -> Result<(), String> {
    confirm_project(target, deadline)?;
    podman(
        &[
            "exec", "--user", "0:0", &target.id, "/bin/sh", "-ceu", BUS_SCRIPT,
        ],
        deadline,
    )?;
    Ok(())
}

pub(crate) fn stage_tools(
    target: &Observation,
    sources: &[Tool],
    deadline: Instant,
) -> Result<(), String> {
    confirm_project(target, deadline)?;
    // The feeder thread borrows no Tool state: the fds stay open in the
    // caller while only plain ints cross the thread boundary.
    let feeds: Vec<(String, RawFd, u64)> = sources
        .iter()
        .map(|t| (t.name.clone(), t.fd, t.size))
        .collect();
    let args: Vec<&str> = vec![
        "exec",
        "--user",
        "0:0",
        "-i",
        &target.id,
        "/bin/sh",
        "-ceu",
        INSTALL_SCRIPT,
        "soda-muse-maintain",
        DESTINATIONS[0],
        DESTINATIONS[1],
        DESTINATIONS[2],
    ];
    podman_streamed(feeds, &args, deadline)?;
    Ok(())
}

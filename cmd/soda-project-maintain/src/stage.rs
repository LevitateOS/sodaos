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
// The fourth entry is the admitted-identity receipt the in-project launch
// path enforces: it must agree with the staged binary, and an explicit
// stale marker refuses new Muse work without a human maintenance step.
const DESTINATIONS: [&str; 4] = [
    "/usr/local/bin/muse",
    "/usr/local/bin/soda-identity-compose",
    "/usr/local/libexec/soda/muse",
    "/usr/local/libexec/soda/muse-selection.json",
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
stage=$(mktemp -d "$(dirname "$3")/.soda-project-maintain.XXXXXXXX")
trap 'rm -rf -- "$stage"' EXIT
tar --extract --file=- --directory="$stage" --no-same-owner
chmod 0755 "$stage/muse" "$stage/soda-identity-compose" "$stage/muse-native"
mv -T -- "$stage/muse" "$1"
mv -T -- "$stage/soda-identity-compose" "$2"
mv -T -- "$stage/muse-native" "$3"
if [ -e "$stage/muse-selection.json" ]; then
  chmod 0644 "$stage/muse-selection.json"
  mv -T -- "$stage/muse-selection.json" "${4-}"
fi
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
    // Pinned staging feeds three tools, store staging four (plus receipt).
    // Anything else is a caller bug: refuse rather than misalign names.
    if sources.len() != 3 && sources.len() != 4 {
        return Err(String::from("public tool source count changed"));
    }
    // The feeder thread borrows no Tool state: the fds stay open in the
    // caller while only plain ints cross the thread boundary.
    let feeds: Vec<(String, RawFd, u64)> = sources
        .iter()
        .map(|t| (t.name.clone(), t.fd, t.size))
        .collect();
    let mut args: Vec<&str> = vec![
        "exec",
        "--user",
        "0:0",
        "-i",
        &target.id,
        "/bin/sh",
        "-ceu",
        INSTALL_SCRIPT,
        "soda-project-maintain",
    ];
    for destination in &DESTINATIONS[..sources.len()] {
        args.push(destination);
    }
    podman_streamed(feeds, &args, deadline)?;
    Ok(())
}

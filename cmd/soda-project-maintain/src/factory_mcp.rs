//! Restore the fixed Factory IPC directory in retained Project namespaces.
//! The directory contains only the authenticated socket; run secrets are staged
//! separately by the host before model start.

use std::os::unix::fs::MetadataExt;
use std::time::{Duration, Instant};

const SOURCE: &str = "/var/lib/soda/dashboard/factory-ipc";
const DESTINATION: &str = "run/soda/factory-ipc";
const PREPARE: &str = "set -eu; test ! -L /run; test -d /run; test ! -L /run/soda; if test -e /run/soda; then test -d /run/soda; fi; mkdir -p /run/soda; test ! -L /run/soda/factory-ipc; if test -e /run/soda/factory-ipc; then test -d /run/soda/factory-ipc; fi; mkdir -p /run/soda/factory-ipc";

pub(crate) fn run(argv: &[String]) -> Result<(), String> {
    let options = crate::options::parse(argv)?;
    let deadline = Instant::now() + Duration::from_secs(30);
    let source =
        std::fs::symlink_metadata(SOURCE).map_err(|_| "Factory IPC directory unavailable")?;
    if !source.is_dir() || source.mode() & 0o022 != 0 || !matches!(source.uid(), 0 | 2000) {
        return Err("Factory IPC directory is not service-owned".to_string());
    }
    let target = crate::project::wait_project(&options.project, deadline)?;
    crate::project::confirm_project(&target, deadline)?;
    let destination = format!("/proc/{}/root/{DESTINATION}", target.pid);
    if std::fs::symlink_metadata(&destination).is_ok_and(|existing| {
        existing.is_dir() && existing.dev() == source.dev() && existing.ino() == source.ino()
    }) {
        return Ok(());
    }
    crate::command::podman(
        &[
            "exec", "--user", "0:0", &target.id, "/bin/sh", "-ceu", PREPARE,
        ],
        deadline,
    )?;
    crate::interface::attach_socket_directory(&target, SOURCE, DESTINATION, deadline)
}

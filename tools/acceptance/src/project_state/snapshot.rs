use std::collections::BTreeSet;
use std::path::Path;

use soda_json::JsonValue;

use super::files::{has_git_part, list_files, walk_sorted};
use super::workloads;
use super::{
    arg_list, check_snapshot_gate, command, glob_prefix, obj, s, set, snapshot_ssh_files, sub_mut,
    Entry, SnapshotFailure,
};

/// Collect the full snapshot document, like the retired probe's `main`.
pub fn run_snapshot() -> Result<JsonValue, SnapshotFailure> {
    check_snapshot_gate(unsafe { libc::geteuid() }, Path::new("/run/.containerenv"))?;
    let mut data = obj();
    set(&mut data, "files", obj());
    set(&mut data, "people", obj());
    set(&mut data, "git", obj());
    set(&mut data, "workloads", JsonValue::Array(Vec::new()));
    set(&mut data, "volumes", JsonValue::Array(Vec::new()));

    let mut paths = vec![
        "/etc/passwd".to_string(),
        "/etc/group".to_string(),
        "/etc/ssh/sshd_config.d/10-soda.conf".to_string(),
        "/etc/ssh/ssh_host_ed25519_key.pub".to_string(),
        "/etc/mise/config.toml".to_string(),
        "/etc/containers/containers.conf".to_string(),
        "/etc/containers/storage.conf".to_string(),
        "/etc/systemd/system/soda-podman.service".to_string(),
        "/usr/libexec/soda/project-init".to_string(),
        "/usr/libexec/soda/project-account".to_string(),
        "/etc/sudoers.d/soda-project".to_string(),
    ];
    if Path::new("/etc/systemd/system/soda-podman.socket").exists() {
        paths.push("/etc/systemd/system/soda-podman.socket".to_string());
    }
    for base in ["/etc/ssh/authorized_keys", "/var/lib/soda/accounts"] {
        if !Path::new(base).is_dir() {
            return Err(SnapshotFailure::assertion(""));
        }
        for file in list_files(Path::new(base))? {
            paths.push(file.to_string_lossy().into_owned());
        }
    }

    for login in ["u08-alice-8417", "u08-bob-8417"] {
        let home = Path::new("/home").join(login);
        if !home.exists() {
            continue;
        }
        let mut person = obj();
        set(
            &mut person,
            "identity",
            s(command(&arg_list(&["id", login]), &[])?),
        );
        set(&mut person, "home", Entry::snapshot(&home, true)?.json());
        set(
            &mut person,
            "shared",
            Entry::snapshot(&home.join("shared"), true)?.json(),
        );
        for (name, value) in snapshot_ssh_files(&home)? {
            set(&mut person, &name, value);
        }
        set(sub_mut(&mut data, "people"), login, person);

        let checkout = home.join("u08-personal-checkout");
        if checkout.exists() {
            // Only the known run-owned repository; export hashes, never content.
            let files = walk_sorted(&checkout)?;
            if files.len() > 10000 {
                return Err(SnapshotFailure::runtime("Checkout snapshot exceeded bound"));
            }
            for file in files {
                if has_git_part(&checkout, &file) {
                    continue;
                }
                let file_type = std::fs::symlink_metadata(&file)
                    .map_err(SnapshotFailure::io)?
                    .file_type();
                if file_type.is_file() || file_type.is_symlink() {
                    paths.push(file.to_string_lossy().into_owned());
                }
            }
            let checkout_arg = checkout.to_string_lossy().into_owned();
            let mut git = obj();
            set(
                &mut git,
                "head",
                s(command(
                    &arg_list(&[
                        "runuser",
                        "-u",
                        login,
                        "--",
                        "git",
                        "-C",
                        &checkout_arg,
                        "rev-parse",
                        "HEAD",
                    ]),
                    &[],
                )?),
            );
            set(
                &mut git,
                "status",
                s(command(
                    &arg_list(&[
                        "runuser",
                        "-u",
                        login,
                        "--",
                        "git",
                        "-C",
                        &checkout_arg,
                        "status",
                        "--porcelain=v1",
                    ]),
                    &[],
                )?),
            );
            set(
                &mut git,
                "refs",
                s(command(
                    &arg_list(&[
                        "runuser",
                        "-u",
                        login,
                        "--",
                        "git",
                        "-C",
                        &checkout_arg,
                        "show-ref",
                    ]),
                    &[],
                )?),
            );
            set(sub_mut(&mut data, "git"), login, git);
        }

        for private in [
            ".ssh/u08-personal-git/identity",
            ".config/soda-u08-workload/database-password",
        ] {
            let target = home.join(private);
            if target.exists() {
                set(
                    sub_mut(&mut data, "files"),
                    &target.to_string_lossy(),
                    Entry::snapshot(&target, false)?.json(),
                );
            }
        }
        for probe in glob_prefix(&home, "u08-access-")? {
            let file_type = std::fs::symlink_metadata(&probe)
                .map_err(SnapshotFailure::io)?
                .file_type();
            if !file_type.is_dir() || file_type.is_symlink() {
                return Err(SnapshotFailure::assertion(""));
            }
            for file in list_files(&probe)? {
                paths.push(file.to_string_lossy().into_owned());
            }
        }
    }

    let shared = Path::new("/srv/project/shared");
    paths.push(shared.to_string_lossy().into_owned());
    for probe in glob_prefix(shared, "u08-shared-")? {
        paths.push(probe.to_string_lossy().into_owned());
        paths.push(probe.join("members").to_string_lossy().into_owned());
    }
    let node = Path::new("/opt/mise/installs/node/24.20.0/bin/node");
    if node.exists() {
        paths.push(node.to_string_lossy().into_owned());
    }
    let unique: BTreeSet<String> = paths.into_iter().collect();
    for path in &unique {
        set(
            sub_mut(&mut data, "files"),
            path,
            Entry::snapshot(Path::new(path), true)?.json(),
        );
    }

    // Fixed project-local API. A second project without initialized workload
    // storage still gets complete account/rootfs coverage, not fabricated lists.
    let expect = std::env::var("SODA_EXPECT_WORKLOADS").unwrap_or_default();
    workloads::snapshot_workloads(&mut data, &expect)?;
    Ok(data)
}

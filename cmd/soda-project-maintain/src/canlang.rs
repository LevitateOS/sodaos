//! Explicit tool installation into the same retained Project root.
use crate::command::{podman, podman_streamed};
use crate::filesystem::Tool;
use crate::project::{confirm_project, wait_project};
use serde::Deserialize;
use std::fs::OpenOptions;
use std::io::Read;
use std::os::unix::fs::{MetadataExt, OpenOptionsExt};
use std::os::unix::io::AsRawFd;
use std::time::{Duration, Instant};

const DIRECTORY: &str = "/usr/share/soda/canlang-tools";
const SCRIPT: &str = include_str!("canlang.sh");

#[derive(Deserialize)]
#[serde(rename_all = "camelCase", deny_unknown_fields)]
struct Release {
    source_revision: String,
    archive_filename: String,
    sha256: String,
    versions: serde_json::Value,
}

fn trusted(name: &str, limit: u64) -> Result<std::fs::File, String> {
    let file = OpenOptions::new()
        .read(true)
        .custom_flags(libc::O_NOFOLLOW | libc::O_NONBLOCK | libc::O_CLOEXEC)
        .open(format!("{DIRECTORY}/{name}"))
        .map_err(|_| "Canlang maintenance payload unavailable")?;
    let meta = file
        .metadata()
        .map_err(|_| "Canlang maintenance payload unavailable")?;
    if !meta.is_file() || meta.uid() != 0 || meta.mode() & 0o022 != 0 || meta.len() > limit {
        return Err("Canlang maintenance requires an admitted root-owned payload".into());
    }
    Ok(file)
}

/// Load the admitted release manifest and check its pinned identity.
fn load_release() -> Result<Release, String> {
    let mut bytes = Vec::new();
    trusted("release.json", 4096)?
        .take(4097)
        .read_to_end(&mut bytes)
        .map_err(|_| "Canlang release unreadable")?;
    let release: Release =
        serde_json::from_slice(&bytes).map_err(|_| "invalid Canlang maintenance release")?;
    if !crate::release::is_hex_string(&release.sha256, 64)
        || !crate::release::is_hex_string(&release.source_revision, 40)
        || release.archive_filename != "canlang-tools-linux-x86_64.tar.gz"
        || release.versions["node"] != "v24.21.0"
        || release.versions["bun"] != "1.4.2"
        || release.versions["miniflare"] != "4.20260730.0"
        || release.versions["workerd"] != "1.20260730.1"
    {
        return Err("invalid Canlang maintenance identity".into());
    }
    Ok(release)
}

/// Open the admitted archive and verify its checksum.
fn open_tool(release: &Release) -> Result<Tool, String> {
    let archive = trusted(&release.archive_filename, 1_000_000_000)?;
    // The existing checksum reader uses pread and retains the streaming offset.
    let tool = Tool {
        name: release.archive_filename.clone(),
        fd: unsafe { libc::dup(archive.as_raw_fd()) },
        size: archive
            .metadata()
            .map_err(|_| "Canlang archive unavailable")?
            .len(),
    };
    if tool.fd < 0 {
        return Err("Canlang archive unavailable".into());
    }
    crate::filesystem::verify_native(&tool, &release.sha256)?;
    Ok(tool)
}

/// Make the staging directory inside the project and check its spelling.
fn make_staging(target: &crate::project::Observation, deadline: Instant) -> Result<String, String> {
    let output = podman(
        &[
            "exec",
            "--user",
            "0:0",
            &target.id,
            "/bin/bash",
            "-ceu",
            "umask 077; mktemp -d /home/.soda-canlang-maintain.XXXXXXXX",
        ],
        deadline,
    )?;
    let staging = String::from_utf8(output)
        .map_err(|_| "invalid Canlang staging path")?
        .trim()
        .to_string();
    if !staging.starts_with("/home/.soda-canlang-maintain.")
        || !staging
            .bytes()
            .all(|b| b.is_ascii_alphanumeric() || b"/.-".contains(&b))
    {
        return Err("invalid Canlang staging path".into());
    }
    Ok(staging)
}

/// Stream the archive in, then run the maintenance script on the staging dir.
fn run_maintain(
    target: &crate::project::Observation,
    tool: &Tool,
    staging: &str,
    release: &Release,
    deadline: Instant,
) -> Result<(), String> {
    podman_streamed(
        vec![("archive.tar.gz".into(), tool.fd, tool.size)],
        &["cp", "-", &format!("{}:{staging}", target.id)],
        deadline,
    )?;
    confirm_project(target, deadline)?;
    podman(
        &[
            "exec",
            "--user",
            "0:0",
            &target.id,
            "/bin/bash",
            "-ceu",
            SCRIPT,
            "canlang-maintain",
            staging,
            &release.sha256,
        ],
        deadline,
    )?;
    Ok(())
}

pub(crate) fn run(args: &[String]) -> Result<(), String> {
    if args.len() != 2 || args[0] != "--project" || !soda_host::domain::valid_id(&args[1]) {
        return Err("canlang maintenance accepts --project ID".into());
    }
    let release = load_release()?;
    let tool = open_tool(&release)?;
    let deadline = Instant::now() + Duration::from_secs(120);
    let target = wait_project(&args[1], deadline)?;
    confirm_project(&target, deadline)?;
    let staging = make_staging(&target, deadline)?;
    run_maintain(&target, &tool, &staging, &release, deadline)?;
    Ok(())
}

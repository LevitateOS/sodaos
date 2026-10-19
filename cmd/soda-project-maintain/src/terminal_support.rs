//! Explicit, forward-only repair of an idle retained Project. Never a host updater.
use std::fs::{self, OpenOptions};
use std::io::{Read, Write};
use std::os::unix::fs::{MetadataExt, OpenOptionsExt};
use std::path::Path;
use std::time::{Duration, Instant};

use serde::{Deserialize, Serialize};
use soda_build_tools::project_support::{ProjectSupport, DESTINATION};
use soda_host::project::admission;

use crate::filesystem::{open_tool, verify_native, Tool};
use crate::project::Observation;
use crate::release::is_hex_string;
use crate::support_root::Root;

pub(crate) const SCRIPT: &str = include_str!("terminal_support.sh");
const PREVIOUS: &str = "/usr/libexec/soda/.project-terminal.previous-";

#[derive(Debug, Deserialize, Serialize, PartialEq)]
#[serde(deny_unknown_fields)]
struct Operation {
    project: String,
    container: String,
    release: String,
    old_sha256: String,
    target_sha256: String,
    previous: String,
}

pub(crate) fn trusted_read(path: &Path, limit: u64) -> Result<Vec<u8>, String> {
    let file = OpenOptions::new()
        .read(true)
        .custom_flags(libc::O_NOFOLLOW | libc::O_NONBLOCK | libc::O_CLOEXEC)
        .open(path)
        .map_err(|_| "maintenance input unavailable".to_string())?;
    let m = file
        .metadata()
        .map_err(|_| "maintenance input unavailable".to_string())?;
    if !m.is_file() || m.uid() != 0 || m.mode() & 0o022 != 0 || m.len() > limit {
        return Err("maintenance input is not an admitted root-owned file".into());
    }
    let mut out = Vec::new();
    file.take(limit + 1)
        .read_to_end(&mut out)
        .map_err(|_| "maintenance input unreadable".to_string())?;
    if out.len() as u64 > limit {
        return Err("maintenance input too large".into());
    }
    Ok(out)
}

struct Options {
    project: String,
    check: bool,
    development: bool,
}
fn options(args: &[String]) -> Result<Options, String> {
    let mut o = Options {
        project: String::new(),
        check: false,
        development: false,
    };
    let mut args = args.iter();
    while let Some(arg) = args.next() {
        match arg.as_str() {
            "--project" => o.project = args.next().ok_or("--project needs an ID")?.clone(),
            "--check" => o.check = true,
            "--development-installed-host" => o.development = true,
            _ => return Err(
                "terminal-support accepts --project ID, --check and --development-installed-host"
                    .into(),
            ),
        }
    }
    if !soda_host::domain::valid_id(&o.project) {
        return Err("exact Project ID required".into());
    }
    Ok(o)
}

fn target(development: bool) -> Result<(String, String, String), String> {
    let body = trusted_read(Path::new(crate::RELEASE_PATH), 4 << 20)?;
    let payload: serde_json::Value =
        serde_json::from_slice(&body).map_err(|_| "invalid installed release")?;
    // The established release reader retains the native/image/unsupported-upgrade gates.
    let admitted = crate::release::load_release_payload(crate::RELEASE_PATH)
        .map_err(|_| "installed release provenance unavailable")?;
    let release = payload
        .get("ID")
        .and_then(|v| v.as_str())
        .ok_or("release identity unavailable")?
        .to_string();
    if development {
        // Explicit development-only repair of the fixed installed host. No caller-selected bytes.
        let digest = soda_host::terminal::agent_program_hash(0)?;
        if soda_host::terminal::agent_program_path() != DESTINATION {
            return Err("installed-host repair refuses a helper environment override".into());
        }
        return Ok((release, DESTINATION.into(), digest));
    }
    let support: ProjectSupport = serde_json::from_value(
        payload
            .get("ProjectSupport")
            .ok_or("installed release has no maintenance payload; no upgrade is qualified")?
            .clone(),
    )
    .map_err(|_| "invalid Project support payload")?;
    support
        .validate(&admitted.project_os)
        .map_err(str::to_string)?;
    let host_hash = soda_host::terminal::agent_program_hash(0)?;
    if host_hash != support.sha256 {
        return Err("installed host differs from maintenance release".into());
    }
    Ok((release, support.payload_path, support.sha256))
}

pub(crate) fn guest(
    target: &Root,
    action: &str,
    expected: &str,
    deadline: Instant,
) -> Result<String, String> {
    let out = target.run(SCRIPT, &[action, expected], deadline)?;
    let text = String::from_utf8(out).map_err(|_| "invalid support observation")?;
    let hash = text.trim();
    if !is_hex_string(hash, 64) {
        return Err("invalid support digest observation".into());
    }
    Ok(hash.into())
}

pub(crate) fn retire_recovery_copies(
    target: &Root,
    expected: &str,
    old: &str,
    new: &str,
    deadline: Instant,
) -> Result<(), String> {
    let out = target.run(SCRIPT, &["retire", expected, old, new], deadline)?;
    if std::str::from_utf8(&out).map(str::trim).ok() != Some(expected) {
        return Err(
            "support recovery-copy retirement unconfirmed; admission remains closed".into(),
        );
    }
    Ok(())
}

fn reconcile(
    record: &Operation,
    target: &Observation,
    release: &str,
    digest: &str,
    actual: &str,
) -> Result<(), String> {
    if record.project != target.project
        || record.container != target.id
        || record.release != release
        || record.target_sha256 != digest
        || record.previous != format!("{PREVIOUS}{}", record.old_sha256)
        || !is_hex_string(&record.old_sha256, 64)
        || (actual != record.old_sha256 && actual != record.target_sha256)
    {
        return Err(
            "interrupted maintenance differs from actual Project/release; admission remains closed"
                .into(),
        );
    }
    Ok(())
}

fn save(path: &Path, record: &Operation) -> Result<(), String> {
    let mut file = OpenOptions::new()
        .write(true)
        .create_new(true)
        .mode(0o600)
        .custom_flags(libc::O_NOFOLLOW | libc::O_CLOEXEC)
        .open(path)
        .map_err(|_| "maintenance record already exists or is unavailable")?;
    file.write_all(&serde_json::to_vec(record).map_err(|_| "maintenance record encoding")?)
        .and_then(|_| file.sync_all())
        .map_err(|_| "maintenance record could not be persisted")?;
    fs::File::open(admission::DIRECTORY)
        .and_then(|f| f.sync_all())
        .map_err(|_| "maintenance directory sync failed".into())
}

fn admitted_source(path: &str, digest: &str) -> Result<Tool, String> {
    let (dir, name) = path.rsplit_once('/').ok_or("invalid support path")?;
    let source = open_tool(dir, name)?;
    if source.size == 0 || source.size > 32 << 20 {
        return Err("invalid support size".into());
    }
    verify_native(&source, digest)?;
    Ok(source)
}

fn load_interrupted(
    marker: &Path,
    observation: &Observation,
    release: &str,
    digest: &str,
    actual: &str,
) -> Result<Option<Operation>, String> {
    match fs::symlink_metadata(marker) {
        Ok(_) => {
            let record: Operation =
                serde_json::from_slice(&trusted_read(marker, 16384)?).map_err(|_| {
                    "interrupted maintenance record unreadable; admission remains closed"
                })?;
            reconcile(&record, observation, release, digest, actual)?;
            Ok(Some(record))
        }
        Err(e) if e.kind() == std::io::ErrorKind::NotFound => Ok(None),
        Err(_) => Err("maintenance state unavailable".into()),
    }
}

fn preflight(
    project: &str,
    target: Root,
    interrupted: bool,
    actual: &str,
    digest: &str,
    deadline: Instant,
) -> Result<(), String> {
    if actual != digest {
        guest(&target, "idle", digest, deadline).map_err(|_| {
            "active or uncertain managed terminal/factory work blocks maintenance; no files changed"
        })?;
    }
    println!(
        "Project {}: {}; {}",
        project,
        if actual == digest {
            "ready"
        } else {
            "support mismatch; explicit maintenance required"
        },
        if interrupted {
            "interrupted operation requires reconciliation"
        } else {
            "read-only preflight; runtime compatibility checked only during maintenance"
        }
    );
    target.close()
}

fn replace_support(
    target: &Root,
    record: &mut Option<Operation>,
    project: &str,
    release: String,
    digest: &str,
    actual: &str,
    source: &Tool,
    marker: &Path,
    deadline: Instant,
) -> Result<(), String> {
    // Prove idle before recording an operation; an ordinary busy refusal closes no admission.
    guest(target, "idle", digest, deadline).map_err(|_| {
        "active or uncertain managed terminal/factory work blocks maintenance; no files changed"
    })?;
    let record = record.get_or_insert(Operation {
        project: project.to_owned(),
        container: target.observation.id.clone(),
        release,
        old_sha256: actual.to_owned(),
        target_sha256: digest.to_owned(),
        previous: format!("{PREVIOUS}{actual}"),
    });
    if !marker.exists() {
        save(marker, record)?;
    }
    // Staging and replacement share the guest terminal/factory exclusion; interruption leaves the marker.
    target
        .install(SCRIPT, digest, &record.old_sha256, source, deadline)
        .map_err(|e| {
            format!(
                "{e}; rerun the same maintenance to reconcile; managed admission remains closed"
            )
        })?;
    Ok(())
}

fn finish(
    target: Root,
    record: &Option<Operation>,
    marker: &Path,
    project: &str,
    development: bool,
    digest: &str,
    deadline: Instant,
) -> Result<(), String> {
    if guest(&target, "observe", digest, deadline)? != digest {
        return Err("support replacement unconfirmed; admission remains closed".into());
    }
    if let Some(record) = &record {
        retire_recovery_copies(
            &target,
            digest,
            &record.old_sha256,
            &record.target_sha256,
            deadline,
        )?;
    }
    target.close()?;
    if marker.exists() {
        fs::remove_file(marker)
            .map_err(|_| "verified support, but admission record removal failed")?;
        fs::File::open(admission::DIRECTORY)
            .and_then(|f| f.sync_all())
            .map_err(|_| "admission directory sync failed")?;
    }
    println!(
        "Project {}: terminal support ready{}",
        project,
        if development {
            " (development installed-host repair; no release upgrade qualification)"
        } else {
            ""
        }
    );
    Ok(())
}

pub(crate) fn run(args: &[String]) -> Result<(), String> {
    let o = options(args)?;
    let (release, path, digest) = target(o.development)?;
    let source = admitted_source(&path, &digest)?;
    let _guard = admission::exclusive()?;
    if fs::symlink_metadata(Path::new(admission::DIRECTORY).join("operation.json")).is_ok() {
        return Err("release maintenance is in progress; use release recovery".into());
    }
    let deadline = Instant::now() + Duration::from_secs(90);
    let target = Root::open(&o.project, deadline)?;
    let actual = guest(&target, "observe", &digest, deadline)?;
    let marker = admission::marker_path(&o.project);
    let mut record = load_interrupted(&marker, &target.observation, &release, &digest, &actual)?;
    if o.check {
        return preflight(
            &o.project,
            target,
            record.is_some(),
            &actual,
            &digest,
            deadline,
        );
    }
    if actual != digest {
        replace_support(
            &target,
            &mut record,
            &o.project,
            release,
            &digest,
            &actual,
            &source,
            &marker,
            deadline,
        )?;
    }
    finish(
        target,
        &record,
        &marker,
        &o.project,
        o.development,
        &digest,
        deadline,
    )
}

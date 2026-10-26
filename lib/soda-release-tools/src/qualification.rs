//! Protected collector execution and exact-byte evidence finalization.
use crate::worker::{self, Worker, WorkerUnitCustody, WORKER_CONTROLLER};
use serde::{Deserialize, Serialize};
use soda_release_deliver::publish::qualification::{self as evidence, ProtectedEvidence};
use std::{
    collections::{BTreeMap, BTreeSet},
    fs,
    os::unix::fs::{DirBuilderExt, MetadataExt, OpenOptionsExt, PermissionsExt},
    path::{Path, PathBuf},
    time::Instant,
};
pub(crate) mod input;
pub(crate) mod retirement;
mod target;

const INPUT: &str = "/run/soda-qualification-candidate";
const OUTPUT: &str = "/run/soda-qualification-output";
const RUNTIME: &str = "/run/soda-qualification-runtime";
const RESOLVED_INPUT: &str = "/run/soda-qualification-journey.json";
#[derive(Clone, Serialize, Deserialize)]
#[serde(deny_unknown_fields)]
struct Runtime {
    directory: PathBuf,
    manifest: PathBuf,
    manifest_sha256: String,
}
#[derive(Clone, Serialize, Deserialize)]
#[serde(deny_unknown_fields)]
struct Credential {
    path: PathBuf,
    sha256: String,
}
#[derive(Clone, Serialize, Deserialize)]
#[serde(deny_unknown_fields)]
struct Authority {
    controller: String,
    controller_sha256: String,
    collector: String,
    collector_sha256: String,
    output_parent: PathBuf,
    candidate_sha256: String,
    payload_sha256: String,
    build_sha256: String,
    source_revision: String,
    purpose: String,
    target: String,
    timeout_seconds: u64,
    public_inputs: BTreeMap<String, soda_release_deliver::model::FileIdentity>,
    #[serde(default)]
    runtime: Option<Runtime>,
    #[serde(default)]
    credentials: BTreeMap<String, Credential>,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    resolved_input: Option<input::FileReference>,
}
#[derive(Clone, Serialize, Deserialize)]
#[serde(deny_unknown_fields)]
struct Request {
    authority: PathBuf,
    candidate: PathBuf,
    authority_sha256: String,
    unit: String,
    attempt: PathBuf,
    binding: ProtectedEvidence,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    resolved_input: Option<input::FileReference>,
}

#[derive(Serialize, Deserialize)]
#[serde(deny_unknown_fields)]
pub(crate) struct CollectionBinding {
    pub request: input::FileReference,
    pub unit: String,
}

pub(crate) fn collection_binding(out: &Path) -> Result<CollectionBinding, String> {
    let (request, _) = retained_request(out)?;
    Ok(CollectionBinding {
        request: input::FileReference {
            path: out.join("request.json"),
            sha256: hash(&out.join("request.json"))?,
        },
        unit: request.unit,
    })
}
#[derive(Serialize, Deserialize)]
#[serde(deny_unknown_fields)]
struct CollectorResult {
    candidate_sha256: String,
    payload_sha256: String,
    source_revision: String,
    purpose: String,
    target: String,
    complete: bool,
}
fn hash(path: &Path) -> Result<String, String> {
    crate::digest::hash_file(path.to_str().ok_or("UTF-8 release path required")?)
}
fn read<T: serde::de::DeserializeOwned>(path: &Path) -> Result<T, String> {
    soda_release_deliver::document::read_json(path.to_str().ok_or("UTF-8 release path required")?)
        .map_err(|e| e.to_string())
}
fn write_new<T: Serialize>(path: &Path, value: &T) -> Result<(), String> {
    input::write_new(
        path,
        &serde_json::to_vec_pretty(value).map_err(|e| e.to_string())?,
        0o600,
    )
}
fn runtime_files(
    root: &Path,
    path: &Path,
    files: &mut BTreeMap<String, soda_release_deliver::model::FileIdentity>,
    total: &mut u64,
) -> Result<(), String> {
    let meta = fs::symlink_metadata(path).map_err(|_| "qualification runtime unavailable")?;
    let mode = meta.mode() & 0o7777;
    if meta.uid() != 0 || meta.file_type().is_symlink() {
        return Err("root-owned symlink-free qualification runtime required".into());
    }
    if meta.is_dir() {
        if !matches!(mode, 0o555 | 0o755) {
            return Err("readable nonwritable qualification runtime directory required".into());
        }
        for entry in fs::read_dir(path).map_err(|_| "qualification runtime unavailable")? {
            runtime_files(
                root,
                &entry
                    .map_err(|_| "qualification runtime unavailable")?
                    .path(),
                files,
                total,
            )?;
        }
    } else {
        *total = total
            .checked_add(meta.len())
            .ok_or("qualification runtime too large")?;
        if !meta.is_file()
            || meta.nlink() != 1
            || !matches!(mode, 0o444 | 0o644 | 0o555 | 0o755)
            || meta.len() > 1 << 30
            || *total > 2 << 30
            || files.len() >= 4096
        {
            return Err("bounded unshared qualification runtime files required".into());
        }
        let name = path
            .strip_prefix(root)
            .ok()
            .and_then(Path::to_str)
            .ok_or("qualification runtime filename refused")?;
        files.insert(
            name.to_owned(),
            soda_release_deliver::model::FileIdentity {
                sha256: hash(path)?,
                bytes: meta.len(),
            },
        );
    }
    Ok(())
}
fn runtime_binding(runtime: &Runtime) -> Result<(), String> {
    #[derive(Deserialize)]
    #[serde(deny_unknown_fields)]
    struct Manifest {
        format: u32,
        files: BTreeMap<String, soda_release_deliver::model::FileIdentity>,
    }
    evidence::protected_path(&runtime.manifest, false)
        .map_err(|_| "protected runtime manifest required")?;
    worker::valid_worker_path(runtime.directory.to_str().ok_or("runtime path refused")?)?;
    if hash(&runtime.manifest)? != runtime.manifest_sha256
        || !runtime.directory.is_absolute()
        || fs::canonicalize(&runtime.directory).map_err(|_| "qualification runtime unavailable")?
            != runtime.directory
        || runtime.manifest.starts_with(&runtime.directory)
    {
        return Err("qualification runtime identity refused".into());
    }
    for ancestor in runtime.directory.ancestors().skip(1) {
        let meta = fs::symlink_metadata(ancestor)
            .map_err(|_| "qualification runtime ancestor unavailable")?;
        if !meta.is_dir()
            || meta.uid() != 0
            || meta.mode() & 0o022 != 0
            || meta.file_type().is_symlink()
        {
            return Err("trusted qualification runtime ancestors required".into());
        }
    }
    let manifest: Manifest = read(&runtime.manifest)?;
    let mut actual = BTreeMap::new();
    runtime_files(&runtime.directory, &runtime.directory, &mut actual, &mut 0)?;
    if manifest.format != 1 || actual.is_empty() || manifest.files != actual {
        return Err("qualification runtime differs from admitted inventory".into());
    }
    Ok(())
}
fn access_binding(path: &Path, a: &Authority) -> Result<(), String> {
    if let Some(runtime) = &a.runtime {
        runtime_binding(runtime)?;
    }
    if !worker::qualification_credential_names(a.credentials.keys().map(String::as_str))
        || (a.purpose == "production" && a.credentials.is_empty())
    {
        return Err("fixed qualification credentials and bounded App names required".into());
    }
    if a.purpose == "production" && a.runtime.is_none() {
        return Err("production qualification runtime required".into());
    }
    let parent = path
        .parent()
        .ok_or("authority parent required")?
        .join("credentials");
    for (name, credential) in &a.credentials {
        worker::valid_worker_path(credential.path.to_str().ok_or("credential path refused")?)?;
        evidence::protected_path(&credential.path, false)
            .map_err(|_| "protected qualification credential required")?;
        if credential.path != parent.join(name) || hash(&credential.path)? != credential.sha256 {
            return Err("qualification credential identity refused".into());
        }
    }
    Ok(())
}

/// Controller references are metadata only. Secret delivery always comes from
/// the authority's fixed private credential directory, never a scenario path.
fn selected_credentials(
    a: &Authority,
    reference: Option<&input::FileReference>,
    retained: bool,
) -> Result<(), String> {
    let mut expected: BTreeSet<String> = if a.credentials.is_empty() && a.purpose != "production" {
        BTreeSet::new()
    } else {
        worker::QUALIFICATION_CREDENTIALS
            .iter()
            .map(|name| (*name).to_owned())
            .collect()
    };
    if let Some(reference) = reference {
        if retained {
            let meta = fs::symlink_metadata(&reference.path)
                .map_err(|_| "retained resolved credential scope unavailable")?;
            if meta.uid() != 0 {
                return Err("root-owned retained credential scope required".into());
            }
            input::verify(reference)?;
        } else {
            evidence::protected_path(&reference.path, false)
                .map_err(|_| "root-protected resolved credential scope required")?;
            if hash(&reference.path)? != reference.sha256 {
                return Err("resolved credential scope changed".into());
            }
        }
        let resolved: serde_json::Value = read(&reference.path)?;
        if let Some(applications) = resolved.pointer("/scenario/applications") {
            let applications = applications
                .as_array()
                .ok_or("selected App credential scope must be an array")?;
            if applications.len() > 16 {
                return Err("selected App credential scope exceeds bound".into());
            }
            let mut ids = BTreeSet::new();
            for app in applications {
                let id = app
                    .get("application_id")
                    .and_then(serde_json::Value::as_str)
                    .ok_or("selected App credential ID required")?;
                if id.len() != 25
                    || !id.starts_with('a')
                    || !id.as_bytes()[1..]
                        .iter()
                        .all(|b| b.is_ascii_digit() || (b'a'..=b'f').contains(b))
                    || !ids.insert(id)
                {
                    return Err("unique selected App credential IDs required".into());
                }
                let owner_path = app
                    .get("owner_credentials_file")
                    .and_then(serde_json::Value::as_str)
                    .ok_or("selected App owner reference required")?;
                if !resolved
                    .pointer("/scenario/credentials/application_owner_files")
                    .and_then(serde_json::Value::as_array)
                    .is_some_and(|paths| paths.iter().any(|path| path.as_str() == Some(owner_path)))
                {
                    return Err("selected App owner reference is outside scenario custody".into());
                }
                expected.insert(format!("application-owner-{id}"));
            }
        }
    }
    if expected != a.credentials.keys().cloned().collect() {
        return Err("qualification credentials differ from exact selected Apps".into());
    }
    Ok(())
}
fn authority(path: &Path) -> Result<Authority, String> {
    if worker::euid() != 0 {
        return Err("trusted root release controller required".into());
    }
    evidence::protected_path(path, false).map_err(|e| e.to_string())?;
    let a: Authority = read(path)?;
    for (path, expected) in [
        (&a.controller, &a.controller_sha256),
        (&a.collector, &a.collector_sha256),
    ] {
        worker::trusted_executable(path)?;
        if hash(Path::new(path))? != *expected {
            return Err("admitted release executable changed".into());
        }
    }
    worker::worker_executable_matches(&a.controller)?;
    evidence::protected_path(&a.output_parent, true).map_err(|e| e.to_string())?;
    if !matches!(a.purpose.as_str(), "development" | "production")
        || a.target.is_empty()
        || a.target.len() > 512
        || a.target.contains(['\0', '\n', '\r'])
        || a.timeout_seconds == 0
        || a.timeout_seconds > 86400
    {
        return Err("bounded collector identity and duration required".into());
    }
    access_binding(path, &a)?;
    Ok(a)
}
fn secure_tree(path: &Path, private: bool) -> Result<(), String> {
    let meta = fs::symlink_metadata(path).map_err(|e| e.to_string())?;
    if meta.file_type().is_symlink() || (!meta.is_dir() && (!meta.is_file() || meta.nlink() != 1)) {
        return Err("unshared regular collector files required".into());
    }
    let c =
        std::ffi::CString::new(path.as_os_str().as_encoded_bytes()).map_err(|e| e.to_string())?;
    if unsafe { libc::chown(c.as_ptr(), 0, 0) } != 0 {
        return Err(std::io::Error::last_os_error().to_string());
    }
    fs::set_permissions(
        path,
        fs::Permissions::from_mode(if private {
            if meta.is_dir() {
                0o700
            } else {
                0o600
            }
        } else if meta.is_dir() || meta.mode() & 0o111 != 0 {
            0o755
        } else {
            0o644
        }),
    )
    .map_err(|e| e.to_string())?;
    if meta.is_dir() {
        for entry in fs::read_dir(path).map_err(|e| e.to_string())? {
            secure_tree(&entry.map_err(|e| e.to_string())?.path(), private)?;
        }
    }
    Ok(())
}
fn input_binding(a: &Authority, candidate: &Path) -> Result<(), String> {
    let (payload, c, _) = soda_release_deliver::admission::load_admitted_candidate(
        candidate.to_str().ok_or("candidate path required")?,
    )
    .map_err(|e| e.to_string())?;
    if payload.architecture != "x86_64"
        || payload.revision != a.source_revision
        || hash(&candidate.join("candidate.json"))? != a.candidate_sha256
        || hash(&candidate.join("payload.json"))? != a.payload_sha256
    {
        return Err("exact candidate/source/architecture binding required".into());
    }
    if c.forgejo_revision != payload.fountain.source_revision {
        return Err("candidate Fountain identity differs".into());
    }
    Ok(())
}

fn snapshot_inputs(a: &Authority, candidate: &Path, input: &Path) -> Result<(), String> {
    // Only the existing public artifact inventory crosses the worker boundary.
    // In particular, no recursive producer tree, authority, logs or private extras.
    let (payload, _, _) = soda_release_deliver::admission::load_admitted_candidate(
        candidate.to_str().ok_or("candidate path required")?,
    )
    .map_err(|e| e.to_string())?;
    let mut files: Vec<String> = [
        "candidate.json",
        "payload.json",
        "host.oci",
        "images/dashboard.oci",
        "images/forgejo.oci",
        "images/project-os.oci",
        "images/canlang-runtime.oci",
        "source.tar",
        "forgejo-source.tar",
        "app-inputs.json",
        "packages.txt",
        "image-config.json",
        "fountain-release.json",
        "forgejo-context/presentation.json",
        "packages/catalog.json",
        "tools.json",
        "tools/soda-acceptance",
        "tools/soda-acceptance-remote",
        "tools/soda-artifacts",
        "tools/soda-installed-probes",
        "tools/soda-installer",
    ]
    .into_iter()
    .map(str::to_owned)
    .collect();
    if candidate.join("docker").exists() {
        files.extend(
            [
                "docker/docker.json",
                "docker/image.oci",
                "docker/compose.yaml",
                "docker/packages.txt",
            ]
            .into_iter()
            .map(str::to_owned),
        );
    }
    for package in &payload.extension_catalog.packages {
        let url = url::Url::parse(&package.url).map_err(|_| "invalid public package URL")?;
        let name = url
            .path_segments()
            .and_then(|mut p| p.next_back())
            .filter(|n| {
                !n.is_empty()
                    && n.bytes()
                        .all(|b| b.is_ascii_alphanumeric() || b"._-".contains(&b))
            })
            .ok_or("public package basename required")?;
        let path = format!("packages/{name}");
        if hash(&candidate.join(&path))? != package.sha256
            || fs::metadata(candidate.join(&path))
                .map_err(|e| e.to_string())?
                .len()
                != package.size
        {
            return Err("public package differs from payload".into());
        }
        files.push(path);
    }
    if candidate.join("media/media.json").exists() {
        let media: soda_release_deliver::model::MediaBinding =
            read(&candidate.join("media/media.json"))?;
        files.push("media/media.json".into());
        for file in [&media.iso, &media.rootfs] {
            if file.path.contains('/')
                || file.path.contains('\\')
                || !(file.path == "installer.iso"
                    || file.path == format!("{}-rootfs.img", file.sha256))
            {
                return Err("public media basename required".into());
            }
            let name = format!("media/{}", file.path);
            if hash(&candidate.join(&name))? != file.sha256
                || fs::metadata(candidate.join(&name))
                    .map_err(|e| e.to_string())?
                    .len()
                    != u64::try_from(file.bytes).map_err(|_| "public media size required")?
            {
                return Err("public media hash differs".into());
            }
            files.push(name);
        }
    }
    files.sort();
    files.dedup();
    if files.iter().ne(a.public_inputs.keys()) {
        return Err("exact admitted public artifact inventory required".into());
    }
    fs::create_dir(input).map_err(|e| e.to_string())?;
    for name in files {
        let from = candidate.join(&name);
        let meta = fs::symlink_metadata(&from).map_err(|e| e.to_string())?;
        if !meta.is_file()
            || meta.file_type().is_symlink()
            || fs::canonicalize(&from).map_err(|e| e.to_string())? != from
        {
            return Err("symlink-free regular public artifact required".into());
        }
        let to = input.join(&name);
        fs::create_dir_all(to.parent().ok_or("artifact parent required")?)
            .map_err(|e| e.to_string())?;
        let status = std::process::Command::new("/usr/bin/cp")
            .args(["--reflink=auto", "--no-clobber", "--"])
            .arg(&from)
            .arg(&to)
            .status()
            .map_err(|e| e.to_string())?;
        if !status.success() {
            return Err("public candidate snapshot failed".into());
        }
        let expected = &a.public_inputs[&name];
        if fs::metadata(&to).map_err(|e| e.to_string())?.len() != expected.bytes
            || hash(&to)? != expected.sha256
        {
            return Err("public artifact differs from admitted collector input".into());
        }
    }
    secure_tree(input, false)
}
fn completed_input(a: &Authority, candidate: &Path) -> Result<(), String> {
    let parent = candidate.parent().ok_or("candidate parent required")?;
    let build = parent.join("evidence/build.json");
    let completion: soda_build_tools::reader::build::BuildCompletion =
        read(&parent.join("evidence/completed.json"))?;
    let result: soda_build_tools::reader::build::BuildResult = read(&build)?;
    if hash(&build)? != a.build_sha256
        || completion.build_sha256 != a.build_sha256
        || result.purpose != a.purpose
        || result.revision != a.source_revision
        || result.architecture != "x86_64"
        || result.candidate_sha256 != a.candidate_sha256
        || result.payload_sha256 != a.payload_sha256
        || result.checks.is_empty()
        || result.requested_target.is_empty()
        || result.completed_target.is_empty()
    {
        return Err("unchanged completed producer input required".into());
    }
    Ok(())
}
fn lock(out: &Path) -> Result<fs::File, String> {
    use std::os::fd::AsRawFd;
    evidence::protected_path(out, true).map_err(|e| e.to_string())?;
    let path = out.join("controller.lock");
    let file = fs::OpenOptions::new()
        .read(true)
        .write(true)
        .create(true)
        .mode(0o600)
        .open(&path)
        .map_err(|e| e.to_string())?;
    evidence::protected_path(&path, false).map_err(|e| e.to_string())?;
    if unsafe { libc::flock(file.as_raw_fd(), libc::LOCK_EX | libc::LOCK_NB) } != 0 {
        return Err("release evidence controller already active".into());
    }
    Ok(file)
}
fn retained_request(out: &Path) -> Result<(Request, Authority), String> {
    evidence::protected_path(&out.join("request.json"), false).map_err(|e| e.to_string())?;
    let request: Request = read(&out.join("request.json"))?;
    let a = authority(&request.authority)?;
    if hash(&request.authority)? != request.authority_sha256
        || request.binding.authority_sha256 != request.authority_sha256
        || request.binding.controller_sha256 != a.controller_sha256
        || request.binding.collector_sha256 != a.collector_sha256
        || request.binding.target != a.target
        || request.binding.purpose != a.purpose
        || request.binding.source_revision != a.source_revision
        || request.binding.candidate_sha256 != a.candidate_sha256
        || request.binding.payload_sha256 != a.payload_sha256
        || request.binding.build_sha256 != a.build_sha256
        || out.parent() != Some(a.output_parent.as_path())
        || request.binding.unit != request.unit
        || request.attempt != out.join("attempt")
    {
        return Err("retained release request binding refused".into());
    }
    input_binding(&a, &out.join("input"))?;
    resolved_binding(out, &a, &request)?;
    selected_credentials(&a, request.resolved_input.as_ref(), true)?;
    Ok((request, a))
}
fn resolved_binding(out: &Path, a: &Authority, request: &Request) -> Result<(), String> {
    match (&a.resolved_input, &request.resolved_input) {
        (None, None) => Ok(()),
        (source, Some(snapshot))
            if snapshot.path == out.join("resolved-input.json")
                && source
                    .as_ref()
                    .is_none_or(|source| snapshot.sha256 == source.sha256) =>
        {
            // The request and output parent are already root-protected. The
            // individual snapshot is readable through its read-only worker bind.
            let meta = fs::symlink_metadata(&snapshot.path).map_err(|e| e.to_string())?;
            if meta.uid() != 0 {
                return Err("root-owned resolved input snapshot required".into());
            }
            input::verify(snapshot)
        }
        _ => Err("retained resolved input binding refused".into()),
    }
}
fn result_binding(path: &Path, a: &Authority) -> Result<(), String> {
    let r: CollectorResult = read(&path.join("collector.json"))?;
    if !r.complete
        || r.candidate_sha256 != a.candidate_sha256
        || r.payload_sha256 != a.payload_sha256
        || r.source_revision != a.source_revision
        || r.purpose != a.purpose
        || r.target != a.target
    {
        return Err("collector output incomplete or mismatched".into());
    }
    Ok(())
}
fn private_work(attempt: &Path, uid: u32, gid: u32) -> Result<(), String> {
    for child in [".work", ".work/home", ".work/tmp", ".work/profile"] {
        let path = attempt.join(child);
        fs::DirBuilder::new()
            .mode(0o700)
            .create(&path)
            .map_err(|e| e.to_string())?;
        let path = std::ffi::CString::new(path.as_os_str().as_encoded_bytes())
            .map_err(|e| e.to_string())?;
        if unsafe { libc::chown(path.as_ptr(), uid, gid) } != 0 {
            return Err(std::io::Error::last_os_error().to_string());
        }
    }
    Ok(())
}
pub(crate) fn finalize(out: &Path) -> Result<(), String> {
    finalize_inner(out, true)?;
    retirement::automatic(out)
}
fn finalize_inner(out: &Path, announce: bool) -> Result<(), String> {
    let _lock = lock(out)?;
    let (request, a) = retained_request(out)?;
    let sealed = out.join("observations");
    if sealed.join("release-evidence.json").exists() {
        let actual = evidence::read_protected(&sealed).map_err(|e| e.to_string())?;
        let mut expected = request.binding;
        expected.exit_code = 0;
        expected.native_retirement = "confirmed".into();
        expected.files = actual.files.clone();
        if actual != expected {
            return Err("completed release evidence replay changed".into());
        }
        return Ok(());
    }
    evidence::protected_path(&out.join("executed.json"), false).map_err(|e| e.to_string())?;
    let executed: ProtectedEvidence = read(&out.join("executed.json"))?;
    let mut expected = request.binding;
    expected.exit_code = 0;
    expected.native_retirement = "confirmed".into();
    expected.files = executed.files.clone();
    if executed != expected
        || executed.files.is_empty()
        || evidence::protected_files(&sealed).map_err(|e| e.to_string())? != executed.files
    {
        return Err("successful retired collector receipt required".into());
    }
    result_binding(&sealed, &a)?;
    write_new(&sealed.join("release-evidence.json"), &executed)?;
    evidence::read_protected(&sealed).map_err(|e| e.to_string())?;
    if announce {
        println!(
            "Protected {} evidence finalized at {}; no publication.",
            a.purpose,
            sealed.display()
        );
    }
    Ok(())
}
pub(crate) fn collect(config: &Path, candidate: &Path, out: &Path) -> Result<(), String> {
    collect_inner(config, candidate, out, None)
}

/// A trusted journey driver supplies fresh resolved bytes and the collector
/// identity from its build receipt. Existing authority/credentials are reused;
/// no authority hash or per-phase flag needs to be edited for that snapshot.
pub(crate) fn collect_journey(
    config: &Path,
    candidate: &Path,
    out: &Path,
    resolved: &input::FileReference,
    collector_sha256: &str,
) -> Result<(), String> {
    evidence::protected_path(&resolved.path, false)
        .map_err(|_| "root-protected resolved journey input required")?;
    if hash(&resolved.path)? != resolved.sha256 {
        return Err("resolved journey input changed".into());
    }
    collect_inner(config, candidate, out, Some((resolved, collector_sha256)))
}

/// Reconcile only the immutable request of a previously admitted journey.
/// This path cannot allocate an output, dispatch a worker or stop a unit. A
/// terminal unit without a successful execution receipt stays incomplete.
pub(crate) fn recover_journey(
    config: &Path,
    candidate: &Path,
    out: &Path,
    resolved: &input::FileReference,
    collector_sha256: &str,
    finish: bool,
) -> Result<serde_json::Value, String> {
    let (request, a) = retained_request(out)?;
    if request.authority != config
        || request.candidate != candidate
        || a.collector_sha256 != collector_sha256
        || request
            .resolved_input
            .as_ref()
            .is_none_or(|snapshot| snapshot.sha256 != resolved.sha256)
    {
        return Err("retained journey collector binding changed".into());
    }
    let sealed = out.join("observations");
    if finish
        && (out.join("executed.json").exists() || sealed.join("release-evidence.json").exists())
    {
        // Existing finalization requires exact successful execution, native
        // retirement and protected observations; it never calls the collector.
        finalize_inner(out, false)?;
        retirement::automatic(out)?;
    }
    let completed = sealed.join("release-evidence.json").exists();
    let (state, native_retirement) = if completed {
        let actual = evidence::read_protected(&sealed).map_err(|e| e.to_string())?;
        let mut expected = request.binding;
        expected.exit_code = 0;
        expected.native_retirement = "confirmed".into();
        expected.files = actual.files.clone();
        if actual != expected {
            return Err("completed release evidence replay changed".into());
        }
        ("complete", "confirmed")
    } else {
        match worker::systemctl_unit_terminal(&request.unit) {
            Ok(false) => ("pending", "pending"),
            Ok(true) => ("retired_incomplete", "confirmed"),
            Err(_) => ("uncertain", "unknown"),
        }
    };
    Ok(serde_json::json!({
        "state": state,
        "native_retirement": native_retirement,
        "collection": collection_binding(out)?,
        "finalized": completed,
        "automatic_actions": false,
        "next_action": match state {
            "complete" => "Use the finalized protected evidence.",
            "pending" => "Observe this exact unit and retained attempt until native retirement is confirmed; do not launch another collector.",
            "retired_incomplete" => "Inspect the retained attempt and execution receipt. This run may only finalize an existing successful receipt; a new trial needs separate admission and fresh guest-custody checks.",
            _ => "Inspect this exact unit and retained request; unknown custody does not authorize another collector.",
        },
    }))
}

fn collect_inner(
    config: &Path,
    candidate: &Path,
    out: &Path,
    journey: Option<(&input::FileReference, &str)>,
) -> Result<(), String> {
    let a = authority(config)?;
    if let Some((resolved, collector_sha256)) = journey {
        if a.collector_sha256 != collector_sha256
            || !crate::digest::is_digest(collector_sha256)
            || a.resolved_input.as_ref().is_some_and(|source| {
                source.path != resolved.path || source.sha256 != resolved.sha256
            })
        {
            return Err("journey collector or fixed resolved input differs from authority".into());
        }
        let selected: serde_json::Value = read(&resolved.path)?;
        for (field, expected) in [
            (
                "/scenario/candidate/source_revision",
                a.source_revision.as_str(),
            ),
            (
                "/scenario/candidate/candidate_sha256",
                a.candidate_sha256.as_str(),
            ),
            (
                "/scenario/candidate/payload_sha256",
                a.payload_sha256.as_str(),
            ),
            ("/scenario/target/name", a.target.as_str()),
        ] {
            if selected.pointer(field).and_then(serde_json::Value::as_str) != Some(expected) {
                return Err("journey differs from collector candidate/target authority".into());
            }
        }
    }
    selected_credentials(
        &a,
        journey
            .map(|(resolved, _)| resolved)
            .or(a.resolved_input.as_ref()),
        false,
    )?;
    if out.parent() != Some(a.output_parent.as_path())
        || out.file_name().is_none()
        || out.starts_with(candidate)
        || candidate.starts_with(out)
        || fs::canonicalize(candidate).map_err(|e| e.to_string())? != candidate
    {
        return Err("fresh disjoint evidence output required".into());
    }
    // All entry points consuming a resolved journey share this target lock.
    // An output-directory lock alone permits two runs against the same VM.
    let target = journey
        .map(|(resolved, _)| resolved)
        .or(a.resolved_input.as_ref())
        .map(target::Target::acquire)
        .transpose()?;
    if let Some(target) = &target {
        target.check(out)?;
    }
    if out.exists() {
        let (r, _) = retained_request(out)?;
        if r.authority != config || r.candidate != candidate || hash(config)? != r.authority_sha256
        {
            return Err("different collector request already exists".into());
        }
        if journey.is_some_and(|(source, _)| {
            r.resolved_input
                .as_ref()
                .is_none_or(|snapshot| snapshot.sha256 != source.sha256)
        }) {
            return Err("different resolved journey already retained".into());
        }
        return finalize(out);
    }
    input_binding(&a, candidate)?;
    completed_input(&a, candidate)?;
    fs::DirBuilder::new()
        .mode(0o700)
        .create(out)
        .map_err(|e| e.to_string())?;
    let _lock = lock(out)?;
    let resolved_input = journey
        .map(|(source, _)| source)
        .or(a.resolved_input.as_ref())
        .map(|source| {
            evidence::protected_path(&source.path, false).map_err(|e| e.to_string())?;
            input::snapshot(source, &out.join("resolved-input.json"))
        })
        .transpose()?;
    let input = out.join("input");
    snapshot_inputs(&a, candidate, &input)?;
    input_binding(&a, &input)?;
    let (_, c, _) = soda_release_deliver::admission::load_admitted_candidate(
        input.to_str().ok_or("candidate path required")?,
    )
    .map_err(|e| e.to_string())?;
    soda_release_deliver::check::check_candidate(
        input.to_str().ok_or("candidate path required")?,
        "x86_64",
        &a.source_revision,
        &c.forgejo_revision,
    )
    .map_err(|e| e.to_string())?;
    let (uid, gid) = worker::lookup_user("soda-qualifier")?;
    if uid == 0 {
        return Err("separate qualifier identity required".into());
    }
    let attempt = out.join("attempt");
    fs::DirBuilder::new()
        .mode(0o700)
        .create(&attempt)
        .map_err(|e| e.to_string())?;
    let c = std::ffi::CString::new(attempt.as_os_str().as_encoded_bytes())
        .map_err(|e| e.to_string())?;
    if unsafe { libc::chown(c.as_ptr(), uid, gid) } != 0 {
        return Err(std::io::Error::last_os_error().to_string());
    }
    private_work(&attempt, uid, gid)?;
    retirement::record_origin(out, &attempt)?;
    let mut nonce = [0; 16];
    getrandom::fill(&mut nonce).map_err(|e| e.to_string())?;
    let name = format!(
        "soda-qualify-{}",
        nonce.iter().map(|v| format!("{v:02x}")).collect::<String>()
    );
    let binding = ProtectedEvidence {
        format: 1,
        purpose: a.purpose.clone(),
        source_revision: a.source_revision.clone(),
        architecture: "x86_64".into(),
        candidate_sha256: a.candidate_sha256.clone(),
        payload_sha256: a.payload_sha256.clone(),
        build_sha256: a.build_sha256.clone(),
        target: a.target.clone(),
        controller_sha256: a.controller_sha256.clone(),
        collector_sha256: a.collector_sha256.clone(),
        authority_sha256: hash(config)?,
        unit: format!("{name}.service"),
        exit_code: -1,
        native_retirement: "pending".into(),
        files: BTreeMap::new(),
    };
    let request = Request {
        authority: config.to_owned(),
        candidate: candidate.to_owned(),
        authority_sha256: hash(config)?,
        unit: binding.unit.clone(),
        attempt: attempt.clone(),
        binding,
        resolved_input,
    };
    write_new(&out.join("request.json"), &request)?;
    resolved_binding(out, &a, &request)?;
    let mut worker = Worker {
        name,
        user: "soda-qualifier".into(),
        executable: a.collector.clone(),
        controller: WORKER_CONTROLLER.into(),
        directory: OUTPUT.into(),
        read_only: vec![
            format!("{}:{WORKER_CONTROLLER}", a.collector),
            format!("{}:{INPUT}", input.display()),
        ],
        writable: vec![format!("{}:{OUTPUT}", attempt.display())],
        environment: vec![
            "PATH=/usr/sbin:/usr/bin:/sbin:/bin".into(),
            format!("HOME={OUTPUT}/.work/home"),
            format!("TMPDIR={OUTPUT}/.work/tmp"),
        ],
        arguments: vec![
            "--candidate".into(),
            INPUT.into(),
            "--out".into(),
            OUTPUT.into(),
            "--target".into(),
            a.target.clone(),
        ],
        credentials: a
            .credentials
            .iter()
            .map(|(name, value)| (name.clone(), value.path.to_string_lossy().into_owned()))
            .collect(),
    };
    if let Some(runtime) = &a.runtime {
        worker
            .read_only
            .push(format!("{}:{RUNTIME}", runtime.directory.display()));
        // Browser NSS trust does not cover Playwright's separate HTTP client
        // (route.fetch). Load the same admitted CA before the worker starts.
        worker
            .environment
            .push(format!("NODE_EXTRA_CA_CERTS={RUNTIME}/trust/appliance-ca.crt"));
    }
    if let Some(snapshot) = &request.resolved_input {
        worker
            .read_only
            .push(format!("{}:{RESOLVED_INPUT}", snapshot.path.display()));
    }
    if let Some(target) = &target {
        target.claim(out, &collection_binding(out)?)?;
    }
    crate::build_cli::watch_build_signals();
    let start = Instant::now();
    let mut custody = WorkerUnitCustody::NeverDispatched;
    let outcome = worker::run_worker(
        &worker,
        &mut std::io::sink(),
        &mut std::io::sink(),
        &|| crate::exitcode::has_interrupt() || start.elapsed().as_secs() >= a.timeout_seconds,
        &mut custody,
    );
    if custody == WorkerUnitCustody::ExactUnitTerminal {
        retirement::preserve(out, &attempt)?;
    }
    if let Err(e) = outcome {
        let text = match e {
            worker::WorkerError::Failed(s) | worker::WorkerError::Cancelled(s) => s,
        };
        return Err(format!(
            "{text}; collector evidence retained without finalization"
        ));
    }
    if custody != WorkerUnitCustody::ExactUnitTerminal || crate::exitcode::has_interrupt() {
        return Err("collector retirement or completion unconfirmed".into());
    }
    access_binding(config, &a)?;
    resolved_binding(out, &a, &request)?;
    input_binding(&a, &input)?;
    result_binding(&attempt, &a)?;
    secure_tree(&attempt, true)?;
    let sealed = out.join("observations");
    fs::rename(&attempt, &sealed).map_err(|e| e.to_string())?;
    let mut executed = request.binding;
    executed.exit_code = 0;
    executed.native_retirement = "confirmed".into();
    executed.files = evidence::protected_files(&sealed).map_err(|e| e.to_string())?;
    write_new(&out.join("executed.json"), &executed)?;
    drop(_lock);
    finalize(out)
}

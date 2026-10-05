//! Isolated-worker dispatch (Go `tools/soda-build` `worker_linux.go`):
//! config admission, worker argv construction, attempt runtime ownership,
//! and result validation. Sandbox execution and live-input resolution stay
//! behind the release-pipeline boundary in `build_cli`.

use std::os::unix::fs::{MetadataExt, PermissionsExt};
use std::path::Path;
use std::process::{Command, Stdio};
use std::time::{Duration, Instant};

use crate::build_spec::{ImageResult, Request};
use crate::digest::hash_file;
use crate::progress::BuildProgress;

pub const WORKER_SOURCE: &str = "/run/soda-build-source";
pub const WORKER_HOME: &str = "/var/lib/soda-build-worker";
pub const WORKER_RUNTIME: &str = "/run/soda-build-worker";
pub const WORKER_TOOLS: &str = "/run/soda-build-tools";
pub const WORKER_FORGEJO: &str = "/run/soda-build-forgejo-source";
pub const PINNED_GO_ROOT: &str = "/usr/local/lib/soda/pinned-go";

#[derive(Debug, Clone, Default)]
pub struct WorkerConfig {
    pub executable: String,
    pub source: String,
    pub forgejo_source: String,
    pub output_parent: String,
    pub storage_root: String,
    pub build_home: String,
    pub runtime: String,
    pub tools: String,
    pub media_authority_directory: String,
}

#[derive(Debug, Clone, Default)]
pub struct Worker {
    pub name: String,
    pub user: String,
    pub executable: String,
    pub directory: String,
    pub read_only: Vec<String>,
    pub writable: Vec<String>,
    pub environment: Vec<String>,
    pub arguments: Vec<String>,
}

pub fn euid() -> u32 {
    unsafe { libc::getuid() }
}

/// `user.Lookup` equivalent over `/etc/passwd` (pure-Go behavior).
pub fn lookup_user(name: &str) -> Result<(u32, u32), String> {
    let data = std::fs::read_to_string("/etc/passwd").map_err(|e| e.to_string())?;
    for line in data.lines() {
        if line.starts_with('#') {
            continue;
        }
        let mut fields = line.split(':');
        let (user, _, uid, gid) = (
            fields.next().unwrap_or_default(),
            fields.next(),
            fields.next().unwrap_or_default(),
            fields.next().unwrap_or_default(),
        );
        if user == name {
            let uid = uid.parse::<u32>().map_err(|e| e.to_string())?;
            let gid = gid.parse::<u32>().map_err(|e| e.to_string())?;
            return Ok((uid, gid));
        }
    }
    Err(format!("user: unknown user {name}"))
}

pub fn build_worker_identity() -> Result<(), String> {
    let (uid, _) = lookup_user("soda-build-worker")?;
    if uid == 0 || euid() != uid {
        return Err("build stage requires the isolated soda-build-worker identity".to_owned());
    }
    Ok(())
}

pub fn worker_runtime_ids() -> Result<(u32, u32), String> {
    lookup_user("soda-build-worker")
}

pub(crate) fn go_clean(path: &str) -> String {
    // `filepath.Clean` equivalent for absolute paths.
    let mut parts: Vec<&str> = Vec::new();
    for part in path.split('/') {
        match part {
            "" | "." => {}
            ".." => {
                parts.pop();
            }
            _ => parts.push(part),
        }
    }
    format!("/{}", parts.join("/"))
}

fn is_abs_clean(path: &str) -> bool {
    path.starts_with('/') && go_clean(path) == path
}

pub fn valid_worker_task_paths(c: &WorkerConfig, r: &Request) -> bool {
    if c.source != r.source || c.forgejo_source != r.forgejo_source {
        return false;
    }
    let parent = parent_dir(&r.out);
    if parent != c.output_parent {
        return false;
    }
    let prefix = format!("{}/.artifacts/releases/", r.source.trim_end_matches('/'));
    parent.starts_with(&prefix) && parent.len() > prefix.len()
}

fn parent_dir(path: &str) -> String {
    match path.rfind('/') {
        Some(0) => "/".to_owned(),
        Some(i) => path[..i].to_owned(),
        None => ".".to_owned(),
    }
}

pub fn worker_executable_matches(path: &str) -> Result<(), String> {
    let current = std::env::current_exe().map_err(|e| e.to_string())?;
    let a = hash_file(current.to_string_lossy().as_ref())?;
    let b = hash_file(path)?;
    if a != b {
        return Err("dispatcher differs from admitted worker executable".to_owned());
    }
    Ok(())
}

fn eval_strict(path: &str) -> Result<String, String> {
    let resolved = std::fs::canonicalize(path).map_err(|e| e.to_string())?;
    Ok(resolved.to_string_lossy().into_owned())
}

pub fn valid_worker_path(path: &str) -> Result<(), String> {
    if !is_abs_clean(path) || path.chars().any(|c| ":\n\r\t %".contains(c)) {
        return Err("explicit safe worker path required".to_owned());
    }
    match eval_strict(path) {
        Ok(resolved) if resolved == path => Ok(()),
        _ => Err("worker paths must exist without symlink traversal".to_owned()),
    }
}

/// `acceptance.TrustedExecutable`: every path component root-owned,
/// non-writable, non-symlink; the leaf a regular executable.
pub fn trusted_executable(path: &str) -> Result<(), String> {
    if !is_abs_clean(path) {
        return Err("absolute admitted executable required".to_owned());
    }
    let mut current = path.to_owned();
    loop {
        let st = std::fs::symlink_metadata(&current).map_err(|e| e.to_string())?;
        let mode = st.permissions().mode();
        if st.uid() != 0 || mode & 0o022 != 0 || st.file_type().is_symlink() {
            return Err(
                "worker executable and parents must be root-owned and not group/world writable"
                    .to_owned(),
            );
        }
        if current == path && (!st.file_type().is_file() || mode & 0o111 == 0) {
            return Err("admitted regular executable required".to_owned());
        }
        if current == "/" {
            break;
        }
        current = parent_dir(&current);
    }
    Ok(())
}

pub fn admit_loaded_worker_config(c: &WorkerConfig, r: &Request) -> Result<(), String> {
    if !valid_worker_task_paths(c, r) {
        return Err("worker source/output differs from approved task paths".to_owned());
    }
    trusted_executable(&c.executable)?;
    worker_executable_matches(&c.executable)
}

pub fn worker_config_paths(c: &WorkerConfig, r: &Request) -> Vec<String> {
    let mut paths = vec![
        c.source.clone(),
        c.forgejo_source.clone(),
        c.output_parent.clone(),
        c.storage_root.clone(),
        c.build_home.clone(),
        c.runtime.clone(),
        c.tools.clone(),
    ];
    if r.wants_media() {
        paths.push(c.media_authority_directory.clone());
    }
    paths
}

pub fn admit_storage_root(c: &WorkerConfig) -> Result<(), String> {
    if c.storage_root.is_empty() || !is_abs_clean(&c.storage_root) {
        return Err("worker storage root required; rerun the setup script".to_owned());
    }
    for (name, p) in [("build home", &c.build_home), ("runtime", &c.runtime)] {
        if p != &c.storage_root && !p.starts_with(&format!("{}/", c.storage_root)) {
            return Err(format!(
                "worker {name} {p:?} is outside the admitted storage root {:?}",
                c.storage_root
            ));
        }
    }
    Ok(())
}

/// `deliver.PrivateFile`: absolute, symlink-free, caller-owned private
/// regular file under a private parent, at most 1 MiB.
pub fn private_file(path: &str) -> Result<(), String> {
    if !path.starts_with('/') {
        return Err("refused".to_owned());
    }
    match eval_strict(path) {
        Ok(resolved) if resolved == path => {}
        _ => return Err("refused".to_owned()),
    }
    let st = std::fs::symlink_metadata(path).map_err(|_| "refused".to_owned())?;
    if !st.file_type().is_file()
        || st.permissions().mode() & 0o077 != 0
        || st.len() > 1 << 20
        || st.uid() != euid()
    {
        return Err("refused".to_owned());
    }
    let parent = std::fs::metadata(parent_dir(path)).map_err(|_| "refused".to_owned())?;
    if parent.permissions().mode() & 0o077 != 0 {
        return Err("refused".to_owned());
    }
    Ok(())
}

fn strict_string_field(
    entries: &[(String, soda_json::JsonValue)],
    key: &str,
) -> Result<String, String> {
    let mut found: Option<String> = None;
    for (k, v) in entries {
        if k == key {
            match v.as_str() {
                Some(s) => found = Some(s.to_owned()),
                None => return Err(format!("invalid {key}")),
            }
        }
    }
    found.ok_or_else(|| format!("missing {key}"))
}

fn decode_worker_config(data: &[u8]) -> Result<WorkerConfig, String> {
    const KNOWN: &[&str] = &[
        "Executable",
        "Source",
        "ForgejoSource",
        "OutputParent",
        "StorageRoot",
        "BuildHome",
        "Runtime",
        "Tools",
        "MediaAuthorityDirectory",
    ];
    let text = std::str::from_utf8(data).map_err(|e| e.to_string())?;
    let value =
        soda_json::JsonValue::parse(text).map_err(|_| "invalid worker configuration".to_owned())?;
    let soda_json::JsonValue::Object(entries) = &value else {
        return Err("invalid worker configuration".to_owned());
    };
    for (k, _) in entries {
        if !KNOWN.contains(&k.as_str()) {
            return Err(format!("unknown worker configuration field {k:?}"));
        }
    }
    Ok(WorkerConfig {
        executable: strict_string_field(entries, "Executable").unwrap_or_default(),
        source: strict_string_field(entries, "Source").unwrap_or_default(),
        forgejo_source: strict_string_field(entries, "ForgejoSource").unwrap_or_default(),
        output_parent: strict_string_field(entries, "OutputParent").unwrap_or_default(),
        storage_root: strict_string_field(entries, "StorageRoot").unwrap_or_default(),
        build_home: strict_string_field(entries, "BuildHome").unwrap_or_default(),
        runtime: strict_string_field(entries, "Runtime").unwrap_or_default(),
        tools: strict_string_field(entries, "Tools").unwrap_or_default(),
        media_authority_directory: strict_string_field(entries, "MediaAuthorityDirectory")
            .unwrap_or_default(),
    })
}

/// `deliver.ReadJSON` equivalent for the worker config: bounded regular
/// input decoded strictly.
pub fn read_worker_config(path: &str) -> Result<WorkerConfig, String> {
    let st = std::fs::symlink_metadata(path).map_err(|e| e.to_string())?;
    if !st.file_type().is_file() || st.len() > 1 << 20 {
        return Err("bounded regular JSON input required".to_owned());
    }
    let data = std::fs::read(path).map_err(|e| e.to_string())?;
    if data.len() > 1 << 20 {
        return Err("bounded regular JSON input required".to_owned());
    }
    decode_worker_config(&data)
}

pub fn load_worker_config(path: &str, r: &Request) -> Result<WorkerConfig, String> {
    if euid() != 0 {
        return Err(
            "run the admitted controller as root; source commands run only as soda-build-worker"
                .to_owned(),
        );
    }
    if private_file(path).is_err() {
        return Err("root-owned restricted worker configuration required".to_owned());
    }
    let c = read_worker_config(path)?;
    admit_storage_root(&c)?;
    admit_loaded_worker_config(&c, r)?;
    for p in worker_config_paths(&c, r) {
        valid_worker_path(&p)?;
    }
    Ok(c)
}

/// Claim one fresh attempt runtime directory under `parent` and hand it to
/// (`uid`, `gid`). Returns the directory; release with `release_attempt_runtime`.
pub fn claim_attempt_runtime(
    parent: &str,
    out: &str,
    uid: u32,
    gid: u32,
) -> Result<String, String> {
    let leaf = out.rsplit('/').next().unwrap_or_default();
    if leaf.is_empty() || leaf == "." || leaf == ".." {
        return Err("explicit safe output leaf required".to_owned());
    }
    match std::fs::metadata(parent) {
        Ok(st) if st.file_type().is_dir() => {}
        _ => {
            return Err("worker runtime directory missing; rerun the setup script".to_owned());
        }
    }
    for _ in 0..100 {
        let nonce: u32 = unsafe { libc::getpid() as u32 }
            ^ (std::time::SystemTime::now()
                .duration_since(std::time::UNIX_EPOCH)
                .map(|d| d.subsec_nanos())
                .unwrap_or(0))
            ^ rand_u32();
        let dir = format!("{parent}/soda-build-{leaf}-{nonce:08x}");
        match std::fs::create_dir(&dir) {
            Ok(()) => {
                if let Err(e) =
                    std::fs::set_permissions(&dir, std::fs::Permissions::from_mode(0o700))
                {
                    let _ = std::fs::remove_dir(&dir);
                    return Err(e.to_string());
                }
                let path = std::ffi::CString::new(dir.clone()).map_err(|e| e.to_string())?;
                if unsafe { libc::chown(path.as_ptr(), uid, gid) } != 0 {
                    let _ = std::fs::remove_dir_all(&dir);
                    return Err(format!(
                        "cannot hand attempt runtime to the worker: {}",
                        std::io::Error::last_os_error()
                    ));
                }
                return Ok(dir);
            }
            Err(e) if e.kind() == std::io::ErrorKind::AlreadyExists => continue,
            Err(e) => return Err(format!("cannot claim attempt runtime: {e}")),
        }
    }
    Err("cannot claim attempt runtime: no unique name".to_owned())
}

fn rand_u32() -> u32 {
    let mut bytes = [0u8; 4];
    if let Ok(f) = std::fs::File::open("/dev/urandom") {
        use std::io::Read;
        let mut f = f;
        let _ = f.read_exact(&mut bytes);
    }
    u32::from_ne_bytes(bytes)
}

/// Remove one claimed attempt directory, refusing anything that is not a
/// direct child directory of `parent`.
pub fn release_attempt_runtime(parent: &str, dir: &str) -> Result<(), String> {
    let rel = dir
        .strip_prefix(parent)
        .filter(|r| r.starts_with('/'))
        .map(|r| &r[1..]);
    let ok = matches!(rel, Some(r) if !r.is_empty() && r != "." && r != ".." && !r.contains('/') && !r.starts_with(".."));
    if !ok {
        return Err(format!(
            "refusing to release {dir:?} outside runtime {parent:?}"
        ));
    }
    let st = std::fs::symlink_metadata(dir)
        .map_err(|e| format!("cannot release attempt runtime: {e}"))?;
    if !st.file_type().is_dir() || st.file_type().is_symlink() {
        return Err(format!("refusing to release non-directory {dir:?}"));
    }
    std::fs::remove_dir_all(dir).map_err(|e| e.to_string())
}

fn join_under(root: &str, rel: &str) -> String {
    if rel.is_empty() {
        root.to_owned()
    } else {
        format!("{}/{}", root.trim_end_matches('/'), rel)
    }
}

fn rel_path(base: &str, path: &str) -> Result<String, String> {
    let base = base.trim_end_matches('/');
    if path == base {
        return Ok(String::new());
    }
    path.strip_prefix(&format!("{base}/"))
        .map(str::to_owned)
        .ok_or_else(|| format!("{path:?} is outside {base:?}"))
}

/// Worker name rule: `^soda-(build|qualify)-[a-z0-9-]{1,48}$`.
pub fn valid_worker_name(name: &str) -> bool {
    let rest = name
        .strip_prefix("soda-build-")
        .or_else(|| name.strip_prefix("soda-qualify-"));
    match rest {
        Some(tail) if !tail.is_empty() && tail.len() <= 48 => tail
            .bytes()
            .all(|b| b.is_ascii_lowercase() || b.is_ascii_digit() || b == b'-'),
        _ => false,
    }
}

/// Build the admitted worker description (pure; execution is the boundary).
pub fn build_worker(c: &WorkerConfig, r: &Request) -> Result<Worker, String> {
    r.validate_target()?;
    let rel = rel_path(&c.source, &r.out)?;
    let parent_rel = rel_path(&c.source, &c.output_parent)?;
    let name = format!(
        "soda-build-{}",
        r.out.rsplit('/').next().unwrap_or_default()
    );
    let start_ns = std::env::var("SODA_BUILD_START_NS").unwrap_or_default();
    let mut w = Worker {
        name,
        user: "soda-build-worker".to_owned(),
        executable: c.executable.clone(),
        directory: WORKER_SOURCE.to_owned(),
        read_only: vec![
            format!("{}:{WORKER_SOURCE}", c.source),
            format!("{}:{WORKER_FORGEJO}", c.forgejo_source),
            format!("{}:{WORKER_TOOLS}", c.tools),
        ],
        writable: vec![
            format!(
                "{}:{}",
                c.output_parent,
                join_under(WORKER_SOURCE, &parent_rel)
            ),
            format!("{}:{WORKER_HOME}", c.build_home),
            format!("{}:{WORKER_RUNTIME}", c.runtime),
        ],
        environment: vec![
            format!("HOME={WORKER_HOME}"),
            format!("PATH={PINNED_GO_ROOT}/bin:{WORKER_TOOLS}/bin:/usr/sbin:/usr/bin:/sbin:/bin"),
            format!("XDG_RUNTIME_DIR={WORKER_RUNTIME}"),
            "GOTOOLCHAIN=go1.26.7".to_owned(),
            format!("GOCACHE={WORKER_HOME}/go-build"),
            format!("GOMODCACHE={WORKER_HOME}/go-mod"),
            format!("PLAYWRIGHT_BROWSERS_PATH={WORKER_HOME}/browsers"),
            format!("SODA_BUILD_START_NS={start_ns}"),
        ],
        arguments: vec![
            "--worker-build".to_owned(),
            "--arch".to_owned(),
            r.arch.clone(),
            "--out".to_owned(),
            join_under(WORKER_SOURCE, &rel),
            "--repository-prefix".to_owned(),
            r.repository_prefix.clone(),
            "--forgejo-source".to_owned(),
            WORKER_FORGEJO.to_owned(),
            "--forgejo-revision".to_owned(),
            r.forgejo_revision.clone(),
        ],
    };
    if r.development {
        w.arguments.push("--development".to_owned());
        w.arguments.push("--target".to_owned());
        w.arguments.push(r.target.clone());
    }
    if !r.media_compression.is_empty() {
        w.arguments.push("--media-compression".to_owned());
        w.arguments.push(r.media_compression.clone());
    }
    if !r.live_inputs.is_empty() {
        w.arguments.push("--live-inputs".to_owned());
        w.arguments.push(r.live_inputs.clone());
    }
    if r.wants_media() {
        w.read_only.push(format!(
            "{}:/run/soda-media-authority",
            c.media_authority_directory
        ));
        w.arguments.push("--rootfs-base-url".to_owned());
        w.arguments.push(r.rootfs_base_url.clone());
        w.arguments.push("--media-authority".to_owned());
        w.arguments
            .push("/run/soda-media-authority/config.json".to_owned());
    }
    Ok(w)
}

fn worker_result_matches(
    r: &Request,
    result: &ImageResult,
    out: &str,
    media: &str,
    completed: &str,
) -> bool {
    result.revision == r.revision
        && result.architecture == r.arch
        && result.candidate == format!("{out}/artifacts/candidate.json")
        && result.media == media
        && result.purpose == r.purpose()
        && result.requested_target == r.requested_target()
        && result.completed_target == completed
        && result.media_compression == r.media_compression
}

pub fn validate_worker_result(r: &Request, result: &mut ImageResult) -> Result<(), String> {
    let rel = rel_path(&r.source, &r.out)?;
    let out = join_under(WORKER_SOURCE, &rel);
    let (media, completed) = if r.wants_media() {
        (format!("{out}/artifacts/media/media.json"), "media")
    } else {
        (String::new(), "candidate")
    };
    if !worker_result_matches(r, result, &out, &media, completed) {
        return Err("worker result does not match this run".to_owned());
    }
    let candidate = format!("{}/artifacts/candidate.json", r.out.trim_end_matches('/'));
    match hash_file(&candidate) {
        Ok(hash) if hash == result.candidate_sha256 => {}
        _ => return Err("worker candidate receipt differs".to_owned()),
    }
    result.candidate = candidate;
    if !media.is_empty() {
        result.media = format!("{}/artifacts/media/media.json", r.out.trim_end_matches('/'));
    }
    Ok(())
}

/// Decode `evidence/build.json` (the fields result validation reads).
pub fn decode_image_result(data: &[u8]) -> Result<ImageResult, String> {
    let text = std::str::from_utf8(data).map_err(|e| e.to_string())?;
    let value = soda_json::JsonValue::parse(text).map_err(|_| "invalid build result".to_owned())?;
    if !value.is_object() {
        return Err("invalid build result".to_owned());
    }
    let field = |key: &str| {
        value
            .get(key)
            .and_then(|v| v.as_str())
            .unwrap_or_default()
            .to_owned()
    };
    Ok(ImageResult {
        revision: field("Revision"),
        architecture: field("Architecture"),
        candidate: field("Candidate"),
        candidate_sha256: field("CandidateSHA256"),
        host_manifest: field("HostManifest"),
        payload_sha256: field("PayloadSHA256"),
        scope: field("Scope"),
        media: field("Media"),
        purpose: field("Purpose"),
        requested_target: field("RequestedTarget"),
        completed_target: field("CompletedTarget"),
        media_compression: field("MediaCompression"),
    })
}

pub fn read_image_result(path: &str) -> Result<ImageResult, String> {
    let st = std::fs::symlink_metadata(path).map_err(|e| e.to_string())?;
    if !st.file_type().is_file() || st.len() > 1 << 20 {
        return Err("bounded regular JSON input required".to_owned());
    }
    let data = std::fs::read(path).map_err(|e| e.to_string())?;
    decode_image_result(&data)
}

// Isolated-worker execution boundary (Go `internal/acceptance`
// `worker_linux.go` `arguments` + `Run`, `tools/soda-build`
// `runBuildWorker` + `resolveWorkerLiveInputs`). Identity and
// executable admission reuse `valid_worker_name` (which matches the Go
// `^soda-(build|qualify)-[a-z0-9-]{1,48}$` rule exactly) and
// `trusted_executable` (root-owned, non-writable, symlink-free chain
// with a regular executable leaf, as in Go).

fn valid_worker_identity(w: &Worker) -> Result<(), String> {
    if !valid_worker_name(&w.name) {
        return Err("exact task worker name required".to_owned());
    }
    if w.user != "soda-build-worker" && w.user != "soda-qualifier" {
        return Err("separate approved worker identity required".to_owned());
    }
    if !w.directory.starts_with('/')
        || w.directory
            .chars()
            .any(|c| c == '\n' || c == '\r' || c == ':')
    {
        return Err("absolute worker directory required".to_owned());
    }
    trusted_executable(&w.executable)
}

fn append_bind_paths(
    mut args: Vec<String>,
    property: &str,
    paths: &[String],
) -> Result<Vec<String>, String> {
    for pair in paths {
        let parts: Vec<&str> = pair.split(':').collect();
        let ok = parts.len() == 2
            && parts[0].starts_with('/')
            && parts[1].starts_with('/')
            && !pair
                .chars()
                .any(|c| c == '\n' || c == '\r' || c == '\t' || c == ' ' || c == '%');
        if !ok {
            return Err("explicit absolute worker bind pair required".to_owned());
        }
        args.push(format!("--property={property}={pair}"));
    }
    Ok(args)
}

fn allowed_worker_env_key(key: &str) -> bool {
    matches!(
        key,
        "HOME"
            | "PATH"
            | "XDG_RUNTIME_DIR"
            | "GOTOOLCHAIN"
            | "GOCACHE"
            | "GOMODCACHE"
            | "BUN_INSTALL_CACHE_DIR"
            | "PLAYWRIGHT_BROWSERS_PATH"
            | "SODA_BUILD_START_NS"
    )
}

fn append_worker_env(mut args: Vec<String>, environment: &[String]) -> Result<Vec<String>, String> {
    for env in environment {
        let (key, ok) = match env.split_once('=') {
            Some((key, _)) => (key, true),
            None => (env.as_str(), false),
        };
        if !allowed_worker_env_key(key) {
            return Err("worker environment key refused".to_owned());
        }
        if !ok || env.chars().any(|c| c == '\n' || c == '\r' || c == '\0') {
            return Err("invalid worker environment".to_owned());
        }
        args.push(format!("--setenv={env}"));
    }
    Ok(args)
}

/// `acceptance.Worker.arguments`: the exact `systemd-run` transient-unit
/// argv for one admitted worker dispatch.
pub fn worker_argv(w: &Worker) -> Result<Vec<String>, String> {
    valid_worker_identity(w)?;
    let mut args = vec![
        "--quiet".to_owned(),
        "--wait".to_owned(),
        "--pipe".to_owned(),
        "--collect".to_owned(),
        "--service-type=exec".to_owned(),
        format!("--unit={}", w.name),
        format!("--property=User={}", w.user),
        format!("--property=Group={}", w.user),
        format!("--property=WorkingDirectory={}", w.directory),
        "--property=ProtectHome=tmpfs".to_owned(),
        "--property=ProtectSystem=strict".to_owned(),
        "--property=PrivateTmp=yes".to_owned(),
        "--property=PrivateMounts=yes".to_owned(),
        "--property=Delegate=yes".to_owned(),
        "--property=CPUQuota=400%".to_owned(),
        "--property=MemoryMax=16G".to_owned(),
        "--property=CPUAffinity=0 1 2 3".to_owned(),
        "--property=KillMode=control-group".to_owned(),
        "--property=TimeoutStopSec=20s".to_owned(),
        "--property=UMask=0077".to_owned(),
        "--property=InaccessiblePaths=-/var/lib/soda-release -/root".to_owned(),
    ];
    args = append_bind_paths(args, "BindReadOnlyPaths", &w.read_only)?;
    args = append_bind_paths(args, "BindPaths", &w.writable)?;
    args = append_worker_env(args, &w.environment)?;
    args.push("--".to_owned());
    args.push(w.executable.clone());
    args.extend(w.arguments.iter().cloned());
    Ok(args)
}

fn drain_pipe<R: std::io::Read + Send + 'static>(
    pipe: Option<R>,
) -> Option<std::thread::JoinHandle<Vec<u8>>> {
    pipe.map(|mut pipe| {
        std::thread::spawn(move || {
            let mut buf = Vec::new();
            let _ = pipe.read_to_end(&mut buf);
            buf
        })
    })
}

fn join_drains(
    out: &mut dyn std::io::Write,
    err_out: &mut dyn std::io::Write,
    out_drain: Option<std::thread::JoinHandle<Vec<u8>>>,
    err_drain: Option<std::thread::JoinHandle<Vec<u8>>>,
) -> Result<(), String> {
    let stdout = out_drain
        .map(|t| t.join().unwrap_or_default())
        .unwrap_or_default();
    let stderr = err_drain
        .map(|t| t.join().unwrap_or_default())
        .unwrap_or_default();
    out.write_all(&stdout).map_err(|e| e.to_string())?;
    err_out.write_all(&stderr).map_err(|e| e.to_string())?;
    let _ = out.flush();
    let _ = err_out.flush();
    Ok(())
}

/// `acceptance.Worker.Run`: dispatch one admitted worker under its own
/// transient systemd unit, forwarding unit output to the given writers.
/// `cancelled` is polled while the unit runs (no async runtime in this
/// crate, so `try_wait` polling); on cancellation the exact unit is
/// stopped with a 30s timeout and the joined failure is reported.
/// Output is forwarded after the unit exits rather than streamed.
pub fn run_worker(
    w: &Worker,
    out: &mut dyn std::io::Write,
    err_out: &mut dyn std::io::Write,
    cancelled: &dyn Fn() -> bool,
) -> Result<(), String> {
    if euid() != 0 {
        return Err("trusted root controller required for worker dispatch".to_owned());
    }
    let args = worker_argv(w)?;
    // Refuse an existing unit instead of adopting or replacing its processes.
    let unit = format!("{}.service", w.name);
    let probed = Command::new("/usr/bin/systemctl")
        .args(["show", &unit, "--property=LoadState", "--value"])
        .output();
    match probed {
        Ok(output)
            if output.status.success()
                && String::from_utf8_lossy(&output.stdout).trim() == "not-found" => {}
        _ => {
            return Err("worker unit is already present or could not be checked".to_owned());
        }
    }
    // Give systemd anonymous pipes, not caller-owned log file descriptors.
    let mut child = Command::new("/usr/bin/systemd-run")
        .args(&args)
        .env_clear()
        .env("PATH", "/usr/sbin:/usr/bin:/sbin:/bin")
        .env("LANG", "C.UTF-8")
        .stdin(Stdio::null())
        .stdout(Stdio::piped())
        .stderr(Stdio::piped())
        .spawn()
        .map_err(|e| e.to_string())?;
    let out_drain = drain_pipe(child.stdout.take());
    let err_drain = drain_pipe(child.stderr.take());
    loop {
        if cancelled() {
            break;
        }
        match child.try_wait().map_err(|e| e.to_string())? {
            Some(status) => {
                join_drains(out, err_out, out_drain, err_drain)?;
                if status.success() {
                    return Ok(());
                }
                return Err(format!("worker {} failed: {status}", w.name));
            }
            None => std::thread::sleep(Duration::from_millis(10)),
        }
    }
    // Stopping only systemd-run does not stop its service. Own the exact unit
    // even when the controller's request has already been cancelled.
    let deadline = Instant::now() + Duration::from_secs(30);
    let stop_err: Option<String> = match Command::new("/usr/bin/systemctl")
        .args(["stop", &unit])
        .env_clear()
        .env("PATH", "/usr/sbin:/usr/bin:/sbin:/bin")
        .env("LANG", "C.UTF-8")
        .stdin(Stdio::null())
        .stdout(Stdio::null())
        .stderr(Stdio::null())
        .spawn()
    {
        Ok(mut stop) => loop {
            match stop.try_wait().map_err(|e| e.to_string())? {
                Some(status) if status.success() => break None,
                Some(status) => break Some(format!("systemctl stop {unit}: {status}")),
                None if Instant::now() >= deadline => {
                    let _ = stop.kill();
                    let _ = stop.wait();
                    break Some(format!("systemctl stop {unit}: timed out"));
                }
                None => std::thread::sleep(Duration::from_millis(10)),
            }
        },
        Err(e) => Some(e.to_string()),
    };
    let _ = child.kill();
    let child_err = child.wait().err().map(|e| e.to_string());
    join_drains(out, err_out, out_drain, err_drain).ok();
    let mut parts = vec!["cancelled".to_owned()];
    parts.extend(stop_err);
    parts.extend(child_err);
    Err(format!("worker {} failed: {}", w.name, parts.join("\n")))
}

/// Pure live-input file naming: `soda-live-inputs-<out-basename>.json`.
pub fn live_inputs_name(out: &str) -> String {
    format!(
        "soda-live-inputs-{}.json",
        out.trim_end_matches('/')
            .rsplit('/')
            .next()
            .unwrap_or_default()
    )
}

/// Pure live-input path computation: the controller-side file path and
/// the worker-visible `--live-inputs` value. No network.
pub fn live_inputs_paths(
    source: &str,
    output_parent: &str,
    out: &str,
) -> Result<(String, String), String> {
    let name = live_inputs_name(out);
    let parent_rel = rel_path(source, output_parent)?;
    let controller = format!("{}/{}", output_parent.trim_end_matches('/'), name);
    let worker_rel = if parent_rel.is_empty() {
        name
    } else {
        format!("{parent_rel}/{name}")
    };
    Ok((controller, join_under(WORKER_SOURCE, &worker_rel)))
}

// resolveWorkerLiveInputs resolves the live inputs on the controller,
// where outbound HTTPS is admitted, and records them beside the attempt
// output for the isolated worker. The worker still consumes only its own
// attempt's file and validates it like a live resolution; digest-pinned
// pulls verify content downstream. Live inputs stay controller-resolved
// even though the worker policy admits one narrow exception: P2 stages
// pinned, checksummed inputs over HTTPS from inside the worker (see
// scripts/selinux/soda-build-worker.te), so the worker is no longer
// fully denied outbound HTTPS. Keep this comment and the policy rule in
// sync; if P2 moves to the controller, remove the http_port_t grant.
pub fn resolve_live_inputs(c: &WorkerConfig, r: &mut Request) -> Result<(), String> {
    use soda_release_build::coreos_stream::{
        resolve_coreos, resolve_tailnet_inputs, write_live_inputs, LiveInputs,
    };
    let coreos = resolve_coreos().map_err(|e| e.to_string())?;
    let tailnet = resolve_tailnet_inputs(&r.arch).map_err(|e| e.to_string())?;
    let (controller, worker) = live_inputs_paths(&c.source, &c.output_parent, &r.out)?;
    write_live_inputs(Path::new(&controller), &LiveInputs { coreos, tailnet })
        .map_err(|e| e.to_string())?;
    r.live_inputs = worker;
    Ok(())
}

/// `runBuildWorker`: delegate the existing producer to the isolated
/// worker, never another recipe. The worker cannot see operator homes
/// or real release custody; only its selected source view, caches and
/// output parent are bound into that namespace.
pub fn run_build_worker(
    config: &WorkerConfig,
    request: &Request,
    progress: &mut BuildProgress,
) -> Result<ImageResult, String> {
    let boundary = if request.wants_media() {
        "P1-P8 / Isolated source-to-media worker"
    } else {
        "P1-P6 / Isolated development candidate worker"
    };
    progress.phase(boundary)?;
    let mut c = config.clone();
    let mut r = request.clone();
    resolve_live_inputs(&c, &mut r)?;
    let (uid, gid) = worker_runtime_ids()?;
    let attempt = claim_attempt_runtime(&c.runtime, &r.out, uid, gid)?;
    let parent = c.runtime.clone();
    c.runtime = attempt.clone();
    let outcome = run_build_worker_attempt(&c, &r, progress);
    if let Err(e) = release_attempt_runtime(&parent, &attempt) {
        eprintln!("warning: cannot release attempt runtime: {e}");
    }
    outcome
}

fn run_build_worker_attempt(
    c: &WorkerConfig,
    r: &Request,
    progress: &mut BuildProgress,
) -> Result<ImageResult, String> {
    let w = build_worker(c, r)?;
    let mut out = std::io::stderr();
    let mut err = std::io::stderr();
    run_worker(&w, &mut out, &mut err, &|| false)?;
    // Producer output is not qualification authority. P9 independently admits it.
    let mut result = read_image_result(&format!(
        "{}/evidence/build.json",
        r.out.trim_end_matches('/')
    ))?;
    validate_worker_result(r, &mut result)?;
    match (progress.end(None), progress.end_phase(None)) {
        (Ok(()), Ok(())) => Ok(result),
        (a, b) => {
            let mut parts = Vec::new();
            if let Err(e) = a {
                parts.push(e);
            }
            if let Err(e) = b {
                parts.push(e);
            }
            Err(parts.join("\n"))
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::path::PathBuf;
    use std::sync::atomic::{AtomicU64, Ordering};

    static COUNTER: AtomicU64 = AtomicU64::new(0);

    fn temp_dir(tag: &str) -> PathBuf {
        let id = COUNTER.fetch_add(1, Ordering::SeqCst);
        let path = std::env::temp_dir().join(format!(
            "soda-reltools-worker-{tag}-{}-{id}",
            std::process::id()
        ));
        let _ = std::fs::remove_dir_all(&path);
        std::fs::create_dir_all(&path).unwrap();
        path
    }

    fn test_config() -> WorkerConfig {
        WorkerConfig {
            executable: "/bin/true".to_owned(),
            source: "/source".to_owned(),
            forgejo_source: "/forgejo".to_owned(),
            output_parent: "/source/.artifacts/releases/isolated".to_owned(),
            tools: "/tools".to_owned(),
            media_authority_directory: "/authority".to_owned(),
            ..WorkerConfig::default()
        }
    }

    #[test]
    fn development_worker_inputs_and_completion() {
        let c = test_config();
        for target in ["candidate", "media", ""] {
            let mut r = Request {
                source: c.source.clone(),
                forgejo_source: c.forgejo_source.clone(),
                forgejo_revision: "a".repeat(40),
                out: format!("{}/test", c.output_parent),
                development: !target.is_empty(),
                target: target.to_owned(),
                ..Request::default()
            };
            if r.wants_media() {
                r.rootfs_base_url = "https://example.invalid".to_owned();
            }
            let w = build_worker(&c, &r).unwrap();
            assert_eq!(w.user, "soda-build-worker");
            assert!(w
                .read_only
                .contains(&format!("{}:{WORKER_FORGEJO}", c.forgejo_source)));
            assert!(w.arguments.contains(&WORKER_FORGEJO.to_owned()));
            assert!(w.arguments.contains(&r.forgejo_revision));
            if target == "candidate" {
                assert!(!w.read_only.join(" ").contains("authority"));
                assert!(!w.arguments.contains(&"--rootfs-base-url".to_owned()));
                assert!(!w.arguments.contains(&"--media-authority".to_owned()));
            } else {
                assert!(w
                    .read_only
                    .contains(&"/authority:/run/soda-media-authority".to_owned()));
                assert!(w.arguments.contains(&"--rootfs-base-url".to_owned()));
            }
            if target.is_empty() {
                assert!(!w.arguments.contains(&"--development".to_owned()));
            } else {
                assert!(w.arguments.contains(&"--development".to_owned()));
                assert!(w.arguments.contains(&target.to_owned()));
            }
        }
    }

    #[test]
    fn worker_forwards_controller_live_inputs() {
        let c = test_config();
        let mut r = Request {
            source: c.source.clone(),
            out: format!("{}/test", c.output_parent),
            development: true,
            target: "candidate".to_owned(),
            live_inputs:
                "/run/soda-build-source/.artifacts/releases/isolated/soda-live-inputs-test.json"
                    .to_owned(),
            ..Request::default()
        };
        let w = build_worker(&c, &r).unwrap();
        assert!(w.arguments.contains(&"--live-inputs".to_owned()));
        assert!(w.arguments.contains(&r.live_inputs));
        r.live_inputs.clear();
        let w = build_worker(&c, &r).unwrap();
        assert!(!w.arguments.contains(&"--live-inputs".to_owned()));
    }

    #[test]
    fn worker_env_has_no_bun_cache_and_pinned_go_first() {
        let c = test_config();
        let r = Request {
            source: c.source.clone(),
            out: format!("{}/test", c.output_parent),
            development: true,
            target: "media".to_owned(),
            rootfs_base_url: "https://example.invalid".to_owned(),
            ..Request::default()
        };
        let w = build_worker(&c, &r).unwrap();
        for env in &w.environment {
            assert!(!env.contains("BUN_INSTALL_CACHE_DIR"), "{env}");
        }
        assert!(w.environment.contains(&format!("HOME={WORKER_HOME}")));
        let path = w
            .environment
            .iter()
            .find(|e| e.starts_with("PATH="))
            .unwrap();
        assert!(
            path.starts_with(&format!("PATH={PINNED_GO_ROOT}/bin:")),
            "{path}"
        );
        assert!(!path.contains("soda-build-tools/go"), "{path}");
    }

    #[test]
    fn admit_storage_root_matrix() {
        let good = WorkerConfig {
            storage_root: "/home/soda-candidate".to_owned(),
            build_home: "/home/soda-candidate/home".to_owned(),
            runtime: "/home/soda-candidate/run".to_owned(),
            ..WorkerConfig::default()
        };
        assert!(admit_storage_root(&good).is_ok());
        for (name, bad) in [
            (
                "missing root",
                WorkerConfig {
                    storage_root: String::new(),
                    ..good.clone()
                },
            ),
            (
                "relative root",
                WorkerConfig {
                    storage_root: "home/soda-candidate".to_owned(),
                    ..good.clone()
                },
            ),
            (
                "home on root",
                WorkerConfig {
                    build_home: "/var/lib/soda-candidate-home".to_owned(),
                    ..good.clone()
                },
            ),
            (
                "runtime on root",
                WorkerConfig {
                    runtime: "/var/lib/soda-candidate-run".to_owned(),
                    ..good.clone()
                },
            ),
            (
                "sibling prefix",
                WorkerConfig {
                    build_home: "/home/soda-candidate-evil".to_owned(),
                    ..good.clone()
                },
            ),
        ] {
            assert!(admit_storage_root(&bad).is_err(), "{name}");
        }
    }

    #[test]
    fn fast_media_worker_selection() {
        let c = test_config();
        let r = Request {
            source: c.source.clone(),
            out: format!("{}/test", c.output_parent),
            development: true,
            target: "media".to_owned(),
            media_compression: "fast".to_owned(),
            rootfs_base_url: "https://example.invalid".to_owned(),
            ..Request::default()
        };
        let w = build_worker(&c, &r).unwrap();
        assert!(w.arguments.join(" ").contains("--media-compression fast"));
        let r = Request {
            development: false,
            target: String::new(),
            ..r
        };
        assert!(build_worker(&c, &r).is_err());
    }

    #[test]
    fn worker_result_binds_target_and_candidate() {
        let source = temp_dir("result");
        let out = source.join(".artifacts/releases/isolated/test");
        std::fs::create_dir_all(out.join("artifacts")).unwrap();
        std::fs::write(out.join("artifacts/candidate.json"), b"fixture").unwrap();
        let out = out.to_string_lossy().into_owned();
        let hash = hash_file(&format!("{out}/artifacts/candidate.json")).unwrap();
        let r = Request {
            source: source.to_string_lossy().into_owned(),
            out: out.clone(),
            revision: "a".repeat(40),
            arch: "x86_64".to_owned(),
            development: true,
            target: "candidate".to_owned(),
            ..Request::default()
        };
        let original = ImageResult {
            revision: r.revision.clone(),
            architecture: r.arch.clone(),
            candidate: format!(
                "{WORKER_SOURCE}/.artifacts/releases/isolated/test/artifacts/candidate.json"
            ),
            candidate_sha256: hash,
            purpose: "development".to_owned(),
            requested_target: "candidate".to_owned(),
            completed_target: "candidate".to_owned(),
            ..ImageResult::default()
        };
        for mode in [
            "valid",
            "media",
            "purpose",
            "requested",
            "completed",
            "hash",
            "compression",
        ] {
            let mut result = original.clone();
            match mode {
                "media" => result.media = "unexpected-media.json".to_owned(),
                "purpose" => result.purpose = "production".to_owned(),
                "requested" => result.requested_target = "release".to_owned(),
                "completed" => result.completed_target = "media".to_owned(),
                "compression" => result.media_compression = "fast".to_owned(),
                "hash" => result.candidate_sha256 = "b".repeat(64),
                _ => {}
            }
            let err = validate_worker_result(&r, &mut result);
            if mode == "valid" {
                assert!(err.is_ok(), "{err:?}");
                assert_eq!(result.candidate, format!("{out}/artifacts/candidate.json"));
                assert!(result.media.is_empty());
            } else {
                assert!(err.is_err(), "{mode}");
            }
        }
        let _ = std::fs::remove_dir_all(&source);
    }

    #[test]
    fn claim_attempt_runtime_isolates_attempts() {
        let parent = temp_dir("runtime");
        let parent = parent.to_string_lossy().into_owned();
        let uid = unsafe { libc::getuid() };
        let gid = unsafe { libc::getgid() };
        let first = claim_attempt_runtime(
            &parent,
            "/source/.artifacts/releases/isolated/manual-01",
            uid,
            gid,
        )
        .unwrap();
        let second = claim_attempt_runtime(
            &parent,
            "/source/.artifacts/releases/isolated/manual-01",
            uid,
            gid,
        )
        .unwrap();
        assert_ne!(first, second);
        for dir in [&first, &second] {
            assert_eq!(dir.rfind('/').map(|i| &dir[..i]), Some(parent.as_str()));
            let st = std::fs::metadata(dir).unwrap();
            assert!(st.file_type().is_dir());
            assert_eq!(st.permissions().mode() & 0o777, 0o700);
        }
        std::fs::write(format!("{first}/a.lock"), b"a").unwrap();
        std::fs::write(format!("{second}/b.lock"), b"b").unwrap();
        release_attempt_runtime(&parent, &first).unwrap();
        assert!(std::fs::metadata(&first).is_err());
        assert!(std::fs::metadata(format!("{second}/b.lock")).is_ok());
        release_attempt_runtime(&parent, &second).unwrap();
        assert!(std::fs::metadata(&second).is_err());
        assert!(std::fs::read_dir(&parent).unwrap().next().is_none());
        let _ = std::fs::remove_dir_all(&parent);
    }

    #[test]
    fn claim_attempt_runtime_refuses_bad_input() {
        let parent = temp_dir("runtime-bad");
        let parent = parent.to_string_lossy().into_owned();
        let uid = unsafe { libc::getuid() };
        let gid = unsafe { libc::getgid() };
        assert!(
            claim_attempt_runtime(&format!("{parent}/missing"), "/source/out/leaf", uid, gid)
                .is_err()
        );
        let plain = format!("{parent}/plain");
        std::fs::write(&plain, b"x").unwrap();
        assert!(claim_attempt_runtime(&plain, "/source/out/leaf", uid, gid).is_err());
        for out in ["", "/"] {
            assert!(
                claim_attempt_runtime(&parent, out, uid, gid).is_err(),
                "{out:?}"
            );
        }
        assert_eq!(std::fs::read_dir(&parent).unwrap().count(), 1);
        let _ = std::fs::remove_dir_all(&parent);
    }

    #[test]
    fn release_attempt_runtime_refuses_foreign_paths() {
        let parent = temp_dir("release");
        let parent = parent.to_string_lossy().into_owned();
        let uid = unsafe { libc::getuid() };
        let gid = unsafe { libc::getgid() };
        let owned = claim_attempt_runtime(&parent, "/source/out/leaf", uid, gid).unwrap();
        let outside = temp_dir("release-out");
        let nested = format!("{owned}/nested");
        std::fs::create_dir(&nested).unwrap();
        let link = format!("{parent}/link");
        std::os::unix::fs::symlink(&owned, &link).unwrap();
        let outside = outside.to_string_lossy().into_owned();
        let missing = format!("{parent}/missing");
        for dir in [&parent, &outside, &nested, &link, &missing] {
            assert!(release_attempt_runtime(&parent, dir).is_err(), "{dir}");
        }
        assert!(std::fs::metadata(&owned).is_ok());
        release_attempt_runtime(&parent, &owned).unwrap();
        let _ = std::fs::remove_dir_all(&parent);
        let _ = std::fs::remove_dir_all(&outside);
    }

    #[test]
    fn build_worker_binds_attempt_runtime() {
        let c = WorkerConfig {
            source: "/source".to_owned(),
            output_parent: "/source/.artifacts/releases/isolated".to_owned(),
            runtime: "/run/attempt-xyz".to_owned(),
            tools: "/tools".to_owned(),
            media_authority_directory: "/authority".to_owned(),
            ..WorkerConfig::default()
        };
        let r = Request {
            source: c.source.clone(),
            out: format!("{}/test", c.output_parent),
            development: true,
            target: "candidate".to_owned(),
            ..Request::default()
        };
        let w = build_worker(&c, &r).unwrap();
        assert!(w
            .writable
            .contains(&format!("{}:{WORKER_RUNTIME}", c.runtime)));
    }

    #[test]
    fn worker_name_rule() {
        assert!(valid_worker_name("soda-build-manual-01"));
        assert!(valid_worker_name("soda-qualify-x"));
        assert!(!valid_worker_name("soda-build-"));
        assert!(!valid_worker_name("soda-other-x"));
        assert!(!valid_worker_name("soda-build-UPPER"));
        assert!(!valid_worker_name(&format!(
            "soda-build-{}",
            "a".repeat(49)
        )));
    }

    #[test]
    fn decode_image_result_round_trip() {
        let document = r#"{"Revision":"r","Architecture":"x86_64","Candidate":"c","CandidateSHA256":"s","Scope":"scope","Purpose":"development","RequestedTarget":"candidate","CompletedTarget":"candidate"}"#;
        let result = decode_image_result(document.as_bytes()).unwrap();
        assert_eq!(result.revision, "r");
        assert_eq!(result.scope, "scope");
        assert!(result.media.is_empty());
        assert!(decode_image_result(b"[1,2]").is_err());
    }

    fn argv_executable() -> String {
        for candidate in ["/usr/bin/true", "/bin/true"] {
            if trusted_executable(candidate).is_ok() {
                return candidate.to_owned();
            }
        }
        panic!("no trusted test executable available");
    }

    fn argv_fixture() -> Worker {
        Worker {
            name: "soda-build-manual-01".to_owned(),
            user: "soda-build-worker".to_owned(),
            executable: argv_executable(),
            directory: "/run/soda-build-source".to_owned(),
            read_only: vec!["/source:/run/soda-build-source".to_owned()],
            writable: vec!["/out:/run/soda-build-source/.artifacts/releases/isolated".to_owned()],
            environment: vec!["HOME=/var/lib/soda-build-worker".to_owned()],
            arguments: vec!["--worker-build".to_owned()],
        }
    }

    #[test]
    fn worker_argv_golden() {
        let w = argv_fixture();
        let argv = worker_argv(&w).unwrap();
        let expected: Vec<String> = [
            "--quiet",
            "--wait",
            "--pipe",
            "--collect",
            "--service-type=exec",
            "--unit=soda-build-manual-01",
            "--property=User=soda-build-worker",
            "--property=Group=soda-build-worker",
            "--property=WorkingDirectory=/run/soda-build-source",
            "--property=ProtectHome=tmpfs",
            "--property=ProtectSystem=strict",
            "--property=PrivateTmp=yes",
            "--property=PrivateMounts=yes",
            "--property=Delegate=yes",
            "--property=CPUQuota=400%",
            "--property=MemoryMax=16G",
            "--property=CPUAffinity=0 1 2 3",
            "--property=KillMode=control-group",
            "--property=TimeoutStopSec=20s",
            "--property=UMask=0077",
            "--property=InaccessiblePaths=-/var/lib/soda-release -/root",
            "--property=BindReadOnlyPaths=/source:/run/soda-build-source",
            "--property=BindPaths=/out:/run/soda-build-source/.artifacts/releases/isolated",
            "--setenv=HOME=/var/lib/soda-build-worker",
            "--",
        ]
        .iter()
        .map(|s| s.to_string())
        .chain([w.executable.clone(), "--worker-build".to_owned()])
        .collect();
        assert_eq!(argv, expected);
    }

    #[test]
    fn worker_argv_refuses_bad_identity() {
        let w = argv_fixture();
        let mut bad = w.clone();
        bad.name = "soda-other-x".to_owned();
        assert_eq!(
            worker_argv(&bad).unwrap_err(),
            "exact task worker name required"
        );
        let mut bad = w.clone();
        bad.user = "root".to_owned();
        assert_eq!(
            worker_argv(&bad).unwrap_err(),
            "separate approved worker identity required"
        );
        let mut bad = w.clone();
        bad.directory = "relative".to_owned();
        assert_eq!(
            worker_argv(&bad).unwrap_err(),
            "absolute worker directory required"
        );
        let mut bad = w.clone();
        bad.directory = "/run/soda-build-source:extra".to_owned();
        assert_eq!(
            worker_argv(&bad).unwrap_err(),
            "absolute worker directory required"
        );
    }

    #[test]
    fn worker_argv_refuses_bad_bind_and_env() {
        let w = argv_fixture();
        for bad in [
            "relative:/guest",
            "/host:relative",
            "/host:/guest:extra",
            "/ho st:/guest",
            "/host:/gue\nst",
            "/host:/gue%st",
        ] {
            let mut bound = w.clone();
            bound.writable = vec![bad.to_owned()];
            assert_eq!(
                worker_argv(&bound).unwrap_err(),
                "explicit absolute worker bind pair required",
                "{bad:?}"
            );
        }
        let mut refused = w.clone();
        refused.environment = vec!["SODA_EVIL=1".to_owned()];
        assert_eq!(
            worker_argv(&refused).unwrap_err(),
            "worker environment key refused"
        );
        for bad in ["HOME", "HOME=/a\nb", "HOME=/a\rb", "PATH=/bin\0x"] {
            let mut invalid = w.clone();
            invalid.environment = vec![bad.to_owned()];
            assert_eq!(
                worker_argv(&invalid).unwrap_err(),
                "invalid worker environment",
                "{bad:?}"
            );
        }
    }

    #[test]
    fn live_inputs_paths_are_pure() {
        let (controller, worker) = live_inputs_paths(
            "/source",
            "/source/.artifacts/releases/isolated",
            "/source/.artifacts/releases/isolated/manual-01",
        )
        .unwrap();
        assert_eq!(
            controller,
            "/source/.artifacts/releases/isolated/soda-live-inputs-manual-01.json"
        );
        assert_eq!(
            worker,
            "/run/soda-build-source/.artifacts/releases/isolated/soda-live-inputs-manual-01.json"
        );
        assert_eq!(live_inputs_name("/a/b/c"), "soda-live-inputs-c.json");
        assert!(live_inputs_paths("/source", "/elsewhere", "/source/out").is_err());
    }

    struct EnvRestore {
        prior: Option<String>,
    }

    impl Drop for EnvRestore {
        fn drop(&mut self) {
            match &self.prior {
                Some(v) => std::env::set_var("SODA_BUILD_START_NS", v),
                None => std::env::remove_var("SODA_BUILD_START_NS"),
            }
        }
    }

    #[test]
    fn run_build_worker_fails_before_dispatch_without_progress() {
        let _restore = EnvRestore {
            prior: std::env::var("SODA_BUILD_START_NS").ok(),
        };
        std::env::remove_var("SODA_BUILD_START_NS");
        let mut progress = BuildProgress::new("test").unwrap();
        let _captured = progress.capture();
        progress.finish(None).unwrap();
        let config = WorkerConfig::default();
        // Both media branches fail at the phase boundary: no live-input
        // network, no root lookup, no systemd dispatch.
        for request in [
            Request {
                development: true,
                target: "candidate".to_owned(),
                ..Request::default()
            },
            Request::default(),
        ] {
            assert_eq!(
                run_build_worker(&config, &request, &mut progress).unwrap_err(),
                "invalid progress transition"
            );
        }
    }
}

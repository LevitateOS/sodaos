//! Worker runtime: attempt lifecycle, result validation, live inputs, and builds.
use std::os::unix::fs::PermissionsExt;
use std::path::Path;

use crate::build_spec::{ImageResult, Request};
use crate::digest::hash_file;
use crate::progress::BuildProgress;

use super::config::{
    worker_runtime_ids, Worker, WorkerConfig, PINNED_GO_ROOT, WORKER_FORGEJO, WORKER_HOME,
    WORKER_RUNTIME, WORKER_SOURCE, WORKER_TOOLS,
};
use super::execution::{run_worker, WorkerError};

/// Claim one fresh attempt runtime directory under `parent` and hand it to
/// (`uid`, `gid`). Returns the directory; release with `release_attempt_runtime`.
pub fn claim_attempt_runtime(
    parent: &str,
    out: &str,
    uid: u32,
    gid: u32,
) -> Result<String, String> {
    claim_attempt_runtime_with_random(parent, out, uid, gid, |bytes| {
        getrandom::fill(bytes).map_err(std::io::Error::other)
    })
}

pub(super) fn claim_attempt_runtime_with_random(
    parent: &str,
    out: &str,
    uid: u32,
    gid: u32,
    mut fill: impl FnMut(&mut [u8]) -> std::io::Result<()>,
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
            ^ rand_u32(&mut fill)?;
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

fn rand_u32(fill: &mut impl FnMut(&mut [u8]) -> std::io::Result<()>) -> Result<u32, String> {
    let mut bytes = [0u8; 4];
    fill(&mut bytes).map_err(|_| "worker runtime randomness unavailable".to_owned())?;
    Ok(u32::from_ne_bytes(bytes))
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
// system/host/selinux/soda-build-worker.te), so the worker is no longer
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
) -> Result<ImageResult, WorkerError> {
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

/// D01-F4: the worker attempt observes the controller interruption source
/// instead of a constant-false predicate, so a recorded INT/TERM cancels
/// the owned worker through run_worker's existing branch.
pub(crate) fn worker_cancelled() -> bool {
    crate::exitcode::has_interrupt()
}

fn run_build_worker_attempt(
    c: &WorkerConfig,
    r: &Request,
    progress: &mut BuildProgress,
) -> Result<ImageResult, WorkerError> {
    let w = build_worker(c, r)?;
    let mut out = std::io::stderr();
    let mut err = std::io::stderr();
    run_worker(&w, &mut out, &mut err, &worker_cancelled)?;
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
            Err(WorkerError::Failed(parts.join("\n")))
        }
    }
}

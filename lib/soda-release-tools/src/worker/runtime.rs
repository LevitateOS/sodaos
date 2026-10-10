//! Worker runtime: attempt lifecycle, result validation, live inputs, and builds.
use std::os::unix::fs::DirBuilderExt;
use std::path::Path;

use serde::de::{MapAccess, Visitor};
use serde::{Deserialize, Deserializer};
use serde_json::value::RawValue;

use crate::build_spec::{ImageResult, Request};
use crate::digest::hash_file;
use crate::progress::BuildProgress;

use super::config::{
    worker_runtime_ids, Worker, WorkerConfig, PINNED_GO_ROOT, WORKER_FORGEJO, WORKER_HOME,
    WORKER_RUNTIME, WORKER_SOURCE, WORKER_TOOLS,
};
use super::execution::{run_worker, WorkerError, WorkerUnitCustody};

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
        let nonce = rand_nonce(&mut fill)?;
        let dir = format!("{parent}/soda-build-{leaf}-{nonce}");
        let mut builder = std::fs::DirBuilder::new();
        builder.mode(0o700);
        match builder.create(&dir) {
            Ok(()) => {
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

fn rand_nonce(fill: &mut impl FnMut(&mut [u8]) -> std::io::Result<()>) -> Result<String, String> {
    let mut bytes = [0u8; 16];
    fill(&mut bytes).map_err(|_| "worker runtime randomness unavailable".to_owned())?;
    Ok(bytes.iter().map(|byte| format!("{byte:02x}")).collect())
}

pub(super) fn attempt_worker_name(attempt: &str) -> Result<String, String> {
    let leaf = Path::new(attempt)
        .file_name()
        .and_then(std::ffi::OsStr::to_str)
        .ok_or_else(|| "claimed worker runtime identity required".to_owned())?;
    if !leaf.starts_with("soda-build-") {
        return Err("claimed worker runtime identity required".to_owned());
    }
    let nonce = leaf
        .rsplit_once('-')
        .map(|(_, nonce)| nonce)
        .filter(|nonce| {
            nonce.len() == 32
                && nonce
                    .bytes()
                    .all(|byte| byte.is_ascii_digit() || (b'a'..=b'f').contains(&byte))
        })
        .ok_or_else(|| "claimed worker runtime identity required".to_owned())?;
    Ok(format!("soda-build-{nonce}"))
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
            format!("PATH={PINNED_GO_ROOT}/bin:{WORKER_TOOLS}/rust/bin:{WORKER_TOOLS}/bin:/usr/sbin:/usr/bin:/sbin:/bin"),
            format!("XDG_RUNTIME_DIR={WORKER_RUNTIME}"),
            "GOTOOLCHAIN=go1.26.7".to_owned(),
            format!("GOCACHE={WORKER_HOME}/go-build"),
            format!("GOMODCACHE={WORKER_HOME}/go-mod"),
            format!("CARGO_HOME={WORKER_HOME}/cargo"),
            format!("CARGO_TARGET_DIR={WORKER_HOME}/cargo-target"),
            "CARGO_NET_OFFLINE=true".to_owned(),
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
    let root: Box<RawValue> =
        serde_json::from_str(text).map_err(|_| "invalid build result".to_owned())?;
    if root.get().as_bytes()[0] != b'{' {
        return Err("invalid build result".to_owned());
    }
    let slots: ImageResultSlots =
        serde_json::from_str(root.get()).map_err(|_| "invalid build result".to_owned())?;
    Ok(ImageResult {
        revision: raw_string(slots.revision),
        architecture: raw_string(slots.architecture),
        candidate: raw_string(slots.candidate),
        candidate_sha256: raw_string(slots.candidate_sha256),
        host_manifest: raw_string(slots.host_manifest),
        payload_sha256: raw_string(slots.payload_sha256),
        scope: raw_string(slots.scope),
        media: raw_string(slots.media),
        purpose: raw_string(slots.purpose),
        requested_target: raw_string(slots.requested_target),
        completed_target: raw_string(slots.completed_target),
        media_compression: raw_string(slots.media_compression),
    })
}

fn raw_string(value: Option<Box<RawValue>>) -> String {
    value
        .and_then(|raw| serde_json::from_str::<String>(raw.get()).ok())
        .unwrap_or_default()
}

#[derive(Default)]
struct ImageResultSlots {
    revision: Option<Box<RawValue>>,
    architecture: Option<Box<RawValue>>,
    candidate: Option<Box<RawValue>>,
    candidate_sha256: Option<Box<RawValue>>,
    host_manifest: Option<Box<RawValue>>,
    payload_sha256: Option<Box<RawValue>>,
    scope: Option<Box<RawValue>>,
    media: Option<Box<RawValue>>,
    purpose: Option<Box<RawValue>>,
    requested_target: Option<Box<RawValue>>,
    completed_target: Option<Box<RawValue>>,
    media_compression: Option<Box<RawValue>>,
}

impl<'de> Deserialize<'de> for ImageResultSlots {
    fn deserialize<D>(deserializer: D) -> Result<Self, D::Error>
    where
        D: Deserializer<'de>,
    {
        struct ResultVisitor;
        impl<'de> Visitor<'de> for ResultVisitor {
            type Value = ImageResultSlots;

            fn expecting(&self, formatter: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
                formatter.write_str("a build result object")
            }

            fn visit_map<M>(self, mut map: M) -> Result<Self::Value, M::Error>
            where
                M: MapAccess<'de>,
            {
                let mut slots = ImageResultSlots::default();
                while let Some(key) = map.next_key::<String>()? {
                    let target = match key.as_str() {
                        "Revision" => &mut slots.revision,
                        "Architecture" => &mut slots.architecture,
                        "Candidate" => &mut slots.candidate,
                        "CandidateSHA256" => &mut slots.candidate_sha256,
                        "HostManifest" => &mut slots.host_manifest,
                        "PayloadSHA256" => &mut slots.payload_sha256,
                        "Scope" => &mut slots.scope,
                        "Media" => &mut slots.media,
                        "Purpose" => &mut slots.purpose,
                        "RequestedTarget" => &mut slots.requested_target,
                        "CompletedTarget" => &mut slots.completed_target,
                        "MediaCompression" => &mut slots.media_compression,
                        _ => {
                            let _: Box<RawValue> = map.next_value()?;
                            continue;
                        }
                    };
                    *target = Some(map.next_value()?);
                }
                Ok(slots)
            }
        }
        deserializer.deserialize_map(ResultVisitor)
    }
}

pub fn read_image_result(path: &str) -> Result<ImageResult, String> {
    use std::io::Read;
    use std::os::unix::fs::OpenOptionsExt;

    let mut options = std::fs::OpenOptions::new();
    options
        .read(true)
        .custom_flags(libc::O_CLOEXEC | libc::O_NOFOLLOW | libc::O_NONBLOCK);
    let file = options.open(path).map_err(|error| {
        if error.raw_os_error() == Some(libc::ELOOP) {
            "bounded regular JSON input required".to_owned()
        } else {
            error.to_string()
        }
    })?;
    let st = file.metadata().map_err(|e| e.to_string())?;
    if !st.file_type().is_file() || st.len() > 1 << 20 {
        return Err("bounded regular JSON input required".to_owned());
    }
    let mut data = Vec::new();
    file.take((1 << 20) + 1)
        .read_to_end(&mut data)
        .map_err(|e| e.to_string())?;
    if data.len() > 1 << 20 {
        return Err("bounded regular JSON input required".to_owned());
    }
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
    use soda_build_tools::reader::stream::LiveInputs;
    use soda_release_build::coreos_stream::{resolve_coreos, resolve_tailnet_inputs};
    use soda_release_build::live_inputs::write_live_inputs;
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
    let mut custody = WorkerUnitCustody::NeverDispatched;
    let outcome = run_build_worker_attempt(&c, &r, progress, &mut custody);
    finish_attempt(outcome, custody, &parent, &attempt)
}

pub(super) fn finish_attempt<T>(
    outcome: Result<T, WorkerError>,
    custody: WorkerUnitCustody,
    parent: &str,
    attempt: &str,
) -> Result<T, WorkerError> {
    match custody {
        WorkerUnitCustody::NeverDispatched | WorkerUnitCustody::ExactUnitTerminal => {
            combine_attempt_cleanup(outcome, release_attempt_runtime(parent, attempt))
        }
        WorkerUnitCustody::ExactUnitUnconfirmed => retain_attempt_runtime(outcome, attempt),
    }
}

fn retain_attempt_runtime<T>(
    outcome: Result<T, WorkerError>,
    attempt: &str,
) -> Result<T, WorkerError> {
    let retained =
        format!("worker runtime retained at {attempt}: exact unit termination is unconfirmed");
    match outcome {
        Ok(_) => Err(WorkerError::Failed(retained)),
        Err(WorkerError::Failed(primary)) => {
            Err(WorkerError::Failed(format!("{primary}\n{retained}")))
        }
        Err(WorkerError::Cancelled(primary)) => {
            Err(WorkerError::Cancelled(format!("{primary}\n{retained}")))
        }
    }
}

pub(super) fn combine_attempt_cleanup<T>(
    outcome: Result<T, WorkerError>,
    cleanup: Result<(), String>,
) -> Result<T, WorkerError> {
    match (outcome, cleanup) {
        (Ok(value), Ok(())) => Ok(value),
        (Ok(_), Err(cleanup)) => Err(WorkerError::Failed(format!(
            "worker attempt completed but runtime cleanup failed: {cleanup}"
        ))),
        (Err(error), Ok(())) => Err(error),
        (Err(WorkerError::Failed(primary)), Err(cleanup)) => Err(WorkerError::Failed(format!(
            "{primary}\nworker runtime cleanup also failed: {cleanup}"
        ))),
        (Err(WorkerError::Cancelled(primary)), Err(cleanup)) => Err(WorkerError::Cancelled(
            format!("{primary}\nworker runtime cleanup also failed: {cleanup}"),
        )),
    }
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
    custody: &mut WorkerUnitCustody,
) -> Result<ImageResult, WorkerError> {
    let mut w = build_worker(c, r)?;
    w.name = attempt_worker_name(&c.runtime)?;
    let mut out = std::io::stderr();
    let mut err = std::io::stderr();
    run_worker(&w, &mut out, &mut err, &worker_cancelled, custody)?;
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

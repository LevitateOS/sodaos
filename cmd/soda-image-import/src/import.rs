use std::process::{Command, Stdio};
use std::sync::atomic::Ordering;
use std::time::{Duration, Instant};

use super::context::ImportCtx;
use super::{ImageBinding, Payload, Role};

// ---------- acquisition (release-design Runtime acquisition) ----------

/// Selectable image sets. Boot acquires `Required`; the two trusted callers
/// (Tailnet prepare, maintenance) explicitly select their set. No image
/// URL/digest/registry override exists.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub(super) enum ImageSet {
    Required,
    Tailnet,
}

impl ImageSet {
    pub(super) fn parse(value: &str) -> Option<ImageSet> {
        match value {
            "required" => Some(ImageSet::Required),
            "tailnet" => Some(ImageSet::Tailnet),
            _ => None,
        }
    }
}

pub(super) fn selected<'a>(
    payload: &'a Payload,
    set: ImageSet,
) -> Vec<(&'static str, &'a ImageBinding)> {
    match set {
        ImageSet::Required => Role::iter()
            .map(|role| (role.name(), role.binding(&payload.images)))
            .collect(),
        ImageSet::Tailnet => vec![(
            "tailscale",
            &payload.optional_extensions.tailnet.images.tailscale,
        )],
    }
}

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub(super) enum PodmanOutcome {
    Code(i32),
    /// Spawn failure, signal death, or killed on cancel/timeout: never an
    /// exit-code match, exactly like a non-`ExitError` Go failure.
    Failed,
}

/// The caller-supplied Podman runner: argv in, exit plus captured stdout.
/// Real runners bound capture; fakes assert the exact acquisition sequence.
pub(super) type Runner<'a> = dyn FnMut(&str, &[String]) -> (PodmanOutcome, Vec<u8>) + 'a;

/// Compare one `podman image inspect` document with the sealed binding: the
/// stored config identity must match and the platform must be `linux/amd64`.
/// An already-present matching config is sufficient for offline use.
fn inspect_matches(output: &[u8], binding: &ImageBinding) -> bool {
    let entries: Vec<serde_json::Value> = match serde_json::from_slice(output) {
        Ok(entries) => entries,
        Err(_) => return false,
    };
    entries.iter().any(|entry| {
        entry.get("Id").and_then(|id| id.as_str()) == Some(binding.config.as_str())
            && entry.get("Os").and_then(|os| os.as_str()) == Some("linux")
            && entry.get("Architecture").and_then(|arch| arch.as_str()) == Some("amd64")
    })
}

fn inspect_verified(
    binding: &ImageBinding,
    podman: &str,
    ctx: &ImportCtx,
    run: &mut Runner<'_>,
) -> Result<(), String> {
    ctx.check()?;
    let inspect = vec![
        "--remote=false".to_string(),
        "image".to_string(),
        "inspect".to_string(),
        binding.config.clone(),
    ];
    match run(podman, &inspect) {
        (PodmanOutcome::Code(0), output) if inspect_matches(&output, binding) => Ok(()),
        (PodmanOutcome::Code(_), _) => Err("image observation failed".to_string()),
        (PodmanOutcome::Failed, _) => Err("image observation failed".to_string()),
    }
}

/// Verify the selected identities in the rootful Podman store, pulling only
/// missing configs by their exact sealed reference. Every selected identity
/// is verified before success; no service starts on a missing image.
pub(super) fn ensure_images(
    payload: &Payload,
    set: ImageSet,
    podman: &str,
    ctx: &ImportCtx,
    run: &mut Runner<'_>,
) -> Result<(), String> {
    payload.validate().map_err(|e| e.to_string())?;
    for (name, binding) in selected(payload, set) {
        ctx.check()?;
        let exists = vec![
            "--remote=false".to_string(),
            "image".to_string(),
            "exists".to_string(),
            binding.config.clone(),
        ];
        match run(podman, &exists).0 {
            PodmanOutcome::Code(0) => {}
            PodmanOutcome::Code(1) => {
                let pull = vec![
                    "--remote=false".to_string(),
                    "pull".to_string(),
                    "--retry=0".to_string(),
                    binding.reference.clone(),
                ];
                match run(podman, &pull).0 {
                    PodmanOutcome::Code(0) => {}
                    _ => return Err(format!("{name} image import unconfirmed")),
                }
                match run(podman, &exists).0 {
                    PodmanOutcome::Code(0) => {}
                    _ => return Err(format!("{name} imported image unavailable")),
                }
            }
            _ => return Err(format!("{name} image observation failed")),
        }
        inspect_verified(binding, podman, ctx, run)
            .map_err(|_| format!("{name} OCI identity mismatch"))?;
    }
    Ok(())
}

pub(super) fn run_podman(
    podman: &str,
    args: &[String],
    ctx: &ImportCtx,
) -> (PodmanOutcome, Vec<u8>) {
    let mut child = match Command::new(podman)
        .args(args)
        .stdin(Stdio::null())
        .stdout(Stdio::null())
        .stderr(Stdio::null())
        .spawn()
    {
        Ok(child) => child,
        Err(_) => return (PodmanOutcome::Failed, Vec::new()),
    };
    loop {
        if ctx.cancelled.load(Ordering::SeqCst) || Instant::now() >= ctx.deadline {
            let _ = child.kill();
            let _ = child.wait();
            return (PodmanOutcome::Failed, Vec::new());
        }
        match child.try_wait() {
            Ok(Some(status)) => {
                return (
                    match status.code() {
                        Some(code) => PodmanOutcome::Code(code),
                        None => PodmanOutcome::Failed,
                    },
                    Vec::new(),
                );
            }
            Ok(None) => std::thread::sleep(Duration::from_millis(10)),
            Err(_) => {
                let _ = child.kill();
                return (PodmanOutcome::Failed, Vec::new());
            }
        }
    }
}

/// Maximum `podman image inspect` document accepted for store verification.
/// Legitimate inspect documents fit well under this bound; anything larger
/// is rejected as an untrusted oversize capture with no bytes returned.
const INSPECT_OUTPUT_LIMIT: usize = 1 << 18;

/// Bounded `podman image inspect` capture for store verification. A reader
/// thread drains the stdout pipe while the parent polls child exit,
/// cancellation and the importer deadline, so a legitimate inspect document
/// larger than the OS pipe capacity can never block the Podman writer while
/// the parent waits. The bound is enforced during the read (no unbounded
/// buffering); oversize output kills and reaps the child and reports failure.
/// Assumes the child is the sole pipe writer, exactly like the sequential
/// drain this replaces.
pub(super) fn run_podman_output(
    podman: &str,
    args: &[String],
    ctx: &ImportCtx,
) -> (PodmanOutcome, Vec<u8>) {
    use std::io::Read;
    let mut child = match Command::new(podman)
        .args(args)
        .stdin(Stdio::null())
        .stdout(Stdio::piped())
        .stderr(Stdio::null())
        .spawn()
    {
        Ok(child) => child,
        Err(_) => return (PodmanOutcome::Failed, Vec::new()),
    };
    let oversize = std::sync::Arc::new(std::sync::atomic::AtomicBool::new(false));
    let oversize_reader = std::sync::Arc::clone(&oversize);
    let piped = child.stdout.take();
    let reader = std::thread::spawn(move || {
        let Some(mut out) = piped else {
            return Vec::new();
        };
        let mut output = Vec::new();
        let mut buf = [0u8; 8192];
        loop {
            match out.read(&mut buf) {
                Ok(0) => break,
                Ok(n) => {
                    if output.len().saturating_add(n) > INSPECT_OUTPUT_LIMIT {
                        oversize_reader.store(true, Ordering::SeqCst);
                        break;
                    }
                    output.extend_from_slice(&buf[..n]);
                }
                // A read error truncates capture; the caller validates bytes.
                Err(_) => break,
            }
        }
        output
    });
    loop {
        if ctx.cancelled.load(Ordering::SeqCst)
            || Instant::now() >= ctx.deadline
            || oversize.load(Ordering::SeqCst)
        {
            let _ = child.kill();
            let _ = child.wait();
            let _ = reader.join();
            return (PodmanOutcome::Failed, Vec::new());
        }
        match child.try_wait() {
            Ok(Some(status)) => {
                let _ = child.wait();
                // Fail closed when the reader panicked: no trusted bytes.
                let output = reader.join().unwrap_or_default();
                if oversize.load(Ordering::SeqCst) || output.len() > INSPECT_OUTPUT_LIMIT {
                    return (PodmanOutcome::Failed, Vec::new());
                }
                return (
                    match status.code() {
                        Some(code) => PodmanOutcome::Code(code),
                        None => PodmanOutcome::Failed,
                    },
                    output,
                );
            }
            Ok(None) => std::thread::sleep(Duration::from_millis(10)),
            Err(_) => {
                let _ = child.kill();
                let _ = child.wait();
                let _ = reader.join();
                return (PodmanOutcome::Failed, Vec::new());
            }
        }
    }
}

/// Boot acquisition: the six required roles, no Tailnet intent read.
pub(super) fn native_import(
    payload: &Payload,
    podman: &str,
    ctx: &ImportCtx,
) -> Result<(), String> {
    ensure_images(
        payload,
        ImageSet::Required,
        podman,
        ctx,
        &mut |cmd, args| {
            if args.get(1).is_some_and(|op| op == "inspect") {
                run_podman_output(cmd, args, ctx)
            } else {
                run_podman(cmd, args, ctx)
            }
        },
    )
}

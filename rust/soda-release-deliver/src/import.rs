//! `import.go`: verified image import into ordinary Podman.

use crate::content::verify_content;
use crate::payload::{Image, Payload, IMAGES_PATH, NAMES};
use crate::Error;

/// Podman invocation outcome: success, or a process exit code.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum RunOutcome {
    Success,
    ExitCode(i32),
    Failed,
}

/// Command runner for image engine invocations.
pub trait EngineRunner {
    fn run(&self, program: &str, args: &[&str]) -> RunOutcome;
}

fn import_missing_image(
    name: &str,
    image: &Image,
    images: &str,
    run: &dyn EngineRunner,
) -> Result<(), Error> {
    match run.run(
        "/usr/bin/podman",
        &["--remote=false", "image", "exists", &image.config],
    ) {
        RunOutcome::Success => return Ok(()),
        RunOutcome::ExitCode(1) => {}
        _ => return Err(Error::msg(format!("{name} image observation failed"))),
    }
    match run.run(
        "/usr/bin/podman",
        &[
            "--remote=false",
            "pull",
            "--retry=0",
            &format!("oci:{images}:{}", image.config),
        ],
    ) {
        RunOutcome::Success => {}
        _ => return Err(Error::msg(format!("{name} image import unconfirmed"))),
    }
    match run.run(
        "/usr/bin/podman",
        &["--remote=false", "image", "exists", &image.config],
    ) {
        RunOutcome::Success => Ok(()),
        _ => Err(Error::msg(format!("{name} imported image unavailable"))),
    }
}

/// `ImportImages`: verify content and import missing exact images.
pub fn import_images(
    p: &Payload,
    images: &str,
    run: &dyn EngineRunner,
) -> Result<(), Error> {
    verify_content(p, images)?;
    for name in NAMES {
        let image = p.images.get(name).cloned().unwrap_or_default();
        import_missing_image(name, &image, images, run)?;
    }
    Ok(())
}

struct NativeEngine;

impl EngineRunner for NativeEngine {
    fn run(&self, program: &str, args: &[&str]) -> RunOutcome {
        match std::process::Command::new(program).args(args).status() {
            Ok(status) if status.success() => RunOutcome::Success,
            Ok(status) => match status.code() {
                Some(code) => RunOutcome::ExitCode(code),
                None => RunOutcome::Failed,
            },
            Err(_) => RunOutcome::Failed,
        }
    }
}

/// `NativeImport`: import the installed payload's images.
pub fn native_import(p: &Payload) -> Result<(), Error> {
    import_images(p, IMAGES_PATH, &NativeEngine)
}

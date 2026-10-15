//! Production build pipeline (`production.go`): the single image layout's
//! concrete steps. Containerfiles, locks, and the Rust stage renderer own
//! content; this code never signs, publishes, or installs anything.

use crate::files::{is_revision, oci_architecture};
use crate::production_inputs::ResolvedInput;
use crate::{io_error, Error};
use soda_release_deliver::buildx::Image;
use std::collections::HashMap;
use std::path::{Path, PathBuf};

#[cfg(test)]
mod tests;

/// Command execution and capture hooks, like Go's `BuildExec`/`BuildCapture`.
pub type BuildExec = Box<dyn Fn(&str, &str, &[String]) -> Result<(), Error> + Send + Sync>;
pub type BuildCapture = Box<dyn Fn(&str, &str, &[String]) -> Result<String, Error> + Send + Sync>;

/// Production attempt: explicit native inputs plus execution hooks.
#[derive(Default)]
pub struct Production {
    pub source: String,
    pub forgejo_source: String,
    pub forgejo_revision: String,
    pub native: String,
    pub out: String,
    pub arch: String,
    pub revision: String,
    /// Controller-resolved live inputs file for this attempt. The worker
    /// never fetches: floating toolchain versions come from here.
    pub live_inputs: String,
    pub execute: Option<BuildExec>,
    pub capture: Option<BuildCapture>,
    pub(crate) inputs: Vec<ResolvedInput>,
}

impl Production {
    pub fn call_execute(&self, dir: &str, name: &str, args: &[String]) -> Result<(), Error> {
        match &self.execute {
            Some(execute) => execute(dir, name, args),
            None => Err(Error::msg("explicit native production inputs required")),
        }
    }

    pub fn call_capture(&self, dir: &str, name: &str, args: &[String]) -> Result<String, Error> {
        match &self.capture {
            Some(capture) => capture(dir, name, args),
            None => Err(Error::msg("explicit native production inputs required")),
        }
    }

    pub(crate) fn validate(&self) -> Result<(), Error> {
        oci_architecture(&self.arch)?;
        let native_want = PathBuf::from(&self.source)
            .join(".artifacts/native")
            .join(&self.arch);
        if !is_revision(&self.revision)
            || !Path::new(&self.source).is_absolute()
            || Path::new(&self.native) != native_want
            || !Path::new(&self.out).is_absolute()
            || self.execute.is_none()
            || self.capture.is_none()
        {
            return Err(Error::msg("explicit native production inputs required"));
        }
        Ok(())
    }

    pub fn images(&self, forgejo_context: &str) -> Result<HashMap<String, ProducedImage>, Error> {
        self.validate()?;
        if forgejo_context.is_empty() {
            return Err(Error::msg("explicit Forgejo context required"));
        }
        let archives = PathBuf::from(&self.out).join("images");
        std::fs::create_dir(&archives).map_err(|e| io_error("mkdir", &archives, e))?;
        crate::files::chmod(&archives, 0o755)?;
        // Exactly the five frozen inputs: Rocky base, admitted Fountain
        // image, upstream Caddy, upstream postgres, pinned official
        // Tailscale. Six roles leave production: four built archives plus
        // two upstream identity binds.
        if self.inputs.len() != 5 {
            return Err(Error::msg("image inputs must be frozen before production"));
        }
        let inputs = self.inputs.clone();
        self.export_images(forgejo_context, &archives, &inputs)
    }

    // The admitted Fountain release supplies the Forgejo image and its
    // source proof: the producer no longer checks out, compiles, or stages
    // a fork binary. The retired forgejo_source/forgejo_revision request
    // fields stay reserved for that selected-release wiring.

    /// Fountain compilation removed. Retained only so the existing
    /// pipeline wiring keeps compiling until its own cutover drops the
    /// call: current staging never invokes it, and it always refuses.
    pub fn stage_fork_binary(&self, _context: &str) -> Result<(), Error> {
        Err(Error::msg(
            "Fountain fork compilation removed; admission consumes the released Fountain image",
        ))
    }
}

/// Verified image plus its archive digest.
#[derive(Debug, Clone, Default)]
pub struct ProducedImage {
    pub image: Image,
    pub archive_sha256: String,
}

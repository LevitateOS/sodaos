//! Seams to the foreign `release/build`, `release/deliver`, and progress
//! surface the pipeline orchestrates but does not own: container builds,
//! OCI inspection, media signing, live CoreOS resolution, and phase
//! reporting. The Go pipeline calls these as struct methods and free
//! functions; here they are two traits so tests script them exactly like
//! the Go tests script `BuildExec`/`BuildCapture` closures.
//!
//! Method order follows the pipeline phases (P1 admission through P8
//! media), not alphabetical order.

use serde::Serialize;
use std::collections::BTreeMap;
use std::collections::HashMap;

use crate::error::Error;
use crate::media::MediaLock;
use crate::model;
use soda_build_tools::reader::stream::{LiveInputs, ResolvedCoreOS};

/// Ordered typed record written into the signed auxiliary-input inventory.
#[derive(Serialize)]
pub struct PackagingInputs {
    #[serde(rename = "Tools")]
    pub tools: MediaLock,
    #[serde(rename = "Files")]
    pub files: BTreeMap<String, String>,
}

/// Heavy foreign operations behind the thin `build.Production` step calls.
/// Data accessors mirror the struct fields the pipeline reads.
pub trait Production {
    fn source(&self) -> &str;
    fn forgejo_source(&self) -> &str;
    fn forgejo_revision(&self) -> &str;
    fn native(&self) -> &str;
    fn out(&self) -> &str;
    fn arch(&self) -> &str;
    fn revision(&self) -> &str;
    fn live_inputs(&self) -> &str;

    /// `p.Execute`: run a build command, streaming to the build log.
    fn execute(&self, dir: &str, name: &str, args: &[String]) -> Result<(), Error>;
    /// `p.Capture`: run a build command, returning trimmed stdout.
    fn capture(&self, dir: &str, name: &str, args: &[String]) -> Result<String, Error>;
    /// `p.Next`: emit a step label.
    fn next(&self, label: &str) -> Result<(), Error>;

    /// `p.ResolveInputs`: freeze image inputs on the production.
    fn resolve_inputs(&mut self) -> Result<(), Error>;
    /// `p.Dependencies`: verify and install frozen dependencies.
    fn dependencies(&self) -> Result<(), Error>;
    /// `p.Compile`: build one Go command.
    fn compile(&self, name: &str, pkg: &str, dest: &str) -> Result<(), Error>;
    /// `p.CompileRust`: build one Rust workspace binary.
    fn compile_rust(&self, crate_name: &str, bin: &str, dest: &str) -> Result<(), Error>;
    /// `p.StageForkBinary`: stage the compiled Forgejo fork binary.
    fn stage_fork_binary(&self, context: &str) -> Result<(), Error>;
    /// `p.Assets`: stage frontend/terminal assets.
    fn assets(&self, host_context: &str, forgejo_context: &str) -> Result<(), Error>;
    /// `p.Images`: build and export the application images.
    fn images(&self, forgejo_context: &str)
        -> Result<HashMap<String, model::ProducedImage>, Error>;

    /// `build.InspectOCI`: inspect an exported OCI archive.
    fn inspect_oci(&self, archive: &str, arch: &str, revision: &str)
        -> Result<model::Image, Error>;
    /// `deliver.VerifyContent`: bind the shared OCI layout to the payload.
    fn verify_content(
        &self,
        payload: &model::Payload,
        dir: &str,
    ) -> Result<(HashMap<String, String>, u64), Error>;
    /// `build.ResolveCoreOS`: resolve the current stable CoreOS build.
    fn resolve_core_os(&self) -> Result<ResolvedCoreOS, Error>;
    /// `build.ReadLiveInputs`: admit controller-resolved live inputs.
    fn read_live_inputs(&self, path: &str) -> Result<LiveInputs, Error>;

    /// `deliver.CheckNative`: verify the native signing toolchain.
    fn check_native(&self, trust_home: &str) -> Result<(), Error>;
    /// `deliver.Sign`: sign one media input.
    fn sign_media(
        &self,
        trust: &model::Trust,
        permit: &model::Permit,
        transport: &str,
        input: &str,
        out: &str,
        keys: &model::SecretFiles,
        trust_home: &str,
    ) -> Result<(), Error>;
    /// `deliver.VerifyCopy`: verify one signed input copy.
    fn verify_copy(
        &self,
        trust: &model::Trust,
        reference: &str,
        source: &str,
        out: &str,
        trust_home: &str,
    ) -> Result<(), Error>;
    /// `deliver.WriteDocument`: seal the auxiliary-input inventory document.
    fn write_document(&self, path: &str, value: &PackagingInputs) -> Result<String, Error>;
}

/// Phase/step reporting (`build.BuildProgress`).
pub trait Progress {
    fn phase(&mut self, label: &str) -> Result<(), Error>;
    fn next(&mut self, label: &str) -> Result<(), Error>;
    fn end_phase(&mut self) -> Result<(), Error>;
    fn end(&mut self) -> Result<(), Error>;
    fn create_log(&mut self, path: &str) -> Result<(), Error>;
    fn note_reason(&mut self, reason: &str);
}

//! Release-pipeline composition root: wires the real `release/build` and
//! `release/deliver` implementations behind the `release/image` pipeline
//! traits, following the Go owner (`internal/release/image/build.go` and
//! `assemble.go`).
//!
//! Bridging notes (all deliberate; see the report for integrator follow-ups):
//!
//! - The image [`Runner`] is `Rc`-based (`!Send`), while the build
//!   [`BuildProduction`] hooks require `Send + Sync`. The factory stashes the
//!   runner in a thread-local and the hooks borrow it back; the whole
//!   pipeline runs synchronously on the worker thread, so no thread boundary
//!   is ever crossed and no `unsafe` is needed. A call from any other thread
//!   fails closed with `production runner unavailable on this thread`.
//! - Build step labels are not wired as production callbacks. The pipeline
//!   reports active phase labels directly through [`ImageProgress`].
//! - [`run_worker_stage`] drops the image `Result.Checks` vector: the worker
//!   [`build_spec::ImageResult`] shape has no such field.
//! - Interrupts seed the image [`Cancel`] by peeking at (then restoring) the
//!   global interrupt flag, so the outer CLI still owns exit-code reporting.

use soda_build_tools::reader::build::BuildResult;

use std::collections::HashMap;
use std::path::Path;

use soda_build_tools::reader::stream::{LiveInputs, ResolvedCoreOS};
use soda_release_build::coreos_stream;
use soda_release_build::production::{
    ProducedImage as BuildProducedImage, Production as BuildProduction,
};
use soda_release_build::Error as BuildError;
use soda_release_deliver::content;
use soda_release_deliver::payload::Payload as DeliverPayload;
use soda_release_deliver::Error as DeliverError;
use soda_release_image::build::ProductionInputs;
use soda_release_image::build_runner::{Cancel, Runner};
use soda_release_image::error::Error as ImageError;
use soda_release_image::foreign::{Production as ImageProduction, Progress as ImageProgress};
use soda_release_image::{model, request};

use crate::build_spec;
use crate::exitcode;
use crate::progress::BuildProgress;

/// Map the admitted worker request onto the image pipeline request, including
/// the selected distribution and immutable Docker version.
pub fn image_request(r: &build_spec::Request) -> request::Request {
    request::Request {
        source: r.source.clone(),
        out: r.out.clone(),
        arch: r.arch.clone(),
        repository_prefix: r.repository_prefix.clone(),
        revision: r.revision.clone(),
        rootfs_base_url: r.rootfs_base_url.clone(),
        live_inputs: r.live_inputs.clone(),
        forgejo_source: r.forgejo_source.clone(),
        forgejo_revision: r.forgejo_revision.clone(),
        development: r.development,
        target: r.target.clone(),
        media_compression: r.media_compression.clone(),
        upgrade_from: r.upgrade_from.clone(),
        distribution: r.distribution.clone(),
        release_tag: r.release_tag.clone(),
        package_base_url: r.package_base_url.clone(),
    }
}

/// Map the pipeline result onto the worker result. The image `Checks` vector
/// has no worker counterpart and is dropped.
fn image_result(r: &BuildResult) -> build_spec::ImageResult {
    build_spec::ImageResult {
        revision: r.revision.clone(),
        architecture: r.architecture.clone(),
        candidate: r.candidate.clone(),
        candidate_sha256: r.candidate_sha256.clone(),
        host_manifest: r.host_manifest.clone(),
        payload_sha256: r.payload_sha256.clone(),
        scope: r.scope.clone(),
        media: r.media.clone(),
        purpose: r.purpose.clone(),
        requested_target: r.requested_target.clone(),
        completed_target: r.completed_target.clone(),
        media_compression: r.media_compression.clone(),
        distribution: r.distribution.clone(),
        release_tag: r.release_tag.clone(),
        docker: r.docker.clone(),
    }
}

fn build_err(e: BuildError) -> ImageError {
    ImageError::msg(e.message())
}

fn deliver_err(e: DeliverError) -> ImageError {
    ImageError::msg(e.0)
}

// ---------------------------------------------------------------------------
// Model conversions between the independent image and build/deliver domains.
// ---------------------------------------------------------------------------

fn produced_of(produced: &BuildProducedImage) -> model::ProducedImage {
    model::ProducedImage {
        base_name: produced.image.base_name.clone(),
        base_digest: produced.image.base_digest.clone(),
        manifest: produced.image.manifest.clone(),
        config: produced.image.config.clone(),
        archive_sha256: produced.archive_sha256.clone(),
    }
}

fn deliver_payload_of(payload: &model::Payload) -> DeliverPayload {
    // Both crates share the canonical Payload v4 definition; the producer
    // record is the deliver document with no conversion.
    payload.clone()
}

// ---------------------------------------------------------------------------
// RealProduction: the image `Production` trait over the real build owner.
// ---------------------------------------------------------------------------

/// Real image production: every build step delegates 1:1 to the owned
/// [`BuildProduction`], every media step to the deliver free functions. All
/// command flow runs through the inner production's hooks (wired to the image
/// runner by [`make_production`]), so there is exactly one execution path.
pub struct RealProduction {
    inner: BuildProduction,
}

impl RealProduction {
    pub fn new(inner: BuildProduction) -> RealProduction {
        RealProduction { inner }
    }
}

impl ImageProduction for RealProduction {
    fn source(&self) -> &str {
        &self.inner.source
    }

    fn forgejo_source(&self) -> &str {
        &self.inner.forgejo_source
    }

    fn forgejo_revision(&self) -> &str {
        &self.inner.forgejo_revision
    }

    fn native(&self) -> &str {
        &self.inner.native
    }

    fn out(&self) -> &str {
        &self.inner.out
    }

    fn arch(&self) -> &str {
        &self.inner.arch
    }

    fn revision(&self) -> &str {
        &self.inner.revision
    }

    fn live_inputs(&self) -> &str {
        &self.inner.live_inputs
    }

    fn execute(&self, dir: &str, name: &str, args: &[String]) -> Result<(), ImageError> {
        self.inner.call_execute(dir, name, args).map_err(build_err)
    }

    fn capture(&self, dir: &str, name: &str, args: &[String]) -> Result<String, ImageError> {
        self.inner.call_capture(dir, name, args).map_err(build_err)
    }

    fn frozen_fountain_release(&self) -> Option<model::FountainRelease> {
        self.inner.fountain.clone()
    }
    fn frozen_upstream_requested(&self, role: &str) -> String {
        self.inner
            .frozen_input(role)
            .map(|input| input.requested.clone())
            .unwrap_or_default()
    }
    fn frozen_tailscale_binding(&self) -> Option<model::PayloadImage> {
        let input = self.inner.frozen_input("tailscale")?;
        Some(model::PayloadImage {
            reference: input.reference.clone(),
            manifest: input.reference.rsplit_once('@')?.1.into(),
            config: input.config.clone(),
            platform: "linux/amd64".into(),
            provenance: model::ImageProvenance::Upstream {
                requested: input.requested.clone(),
            },
        })
    }
    fn source_timestamp(&self) -> Option<u64> {
        self.inner.source_timestamp
    }

    fn resolve_inputs(&mut self) -> Result<(), ImageError> {
        self.inner.resolve_inputs().map_err(build_err)
    }

    fn prepare_docker(&mut self) -> Result<(), ImageError> {
        self.inner.prepare_docker().map_err(build_err)
    }
    fn build_docker(&self, r: &request::Request) -> Result<(), ImageError> {
        use soda_release_deliver::model::{BuildPurpose, Distribution};
        self.inner
            .build_docker(
                &r.release_tag,
                if r.distribution() == "all" {
                    Distribution::All
                } else {
                    Distribution::Docker
                },
                if r.development {
                    BuildPurpose::Development
                } else {
                    BuildPurpose::Production
                },
            )
            .map_err(build_err)
    }
    fn dependencies(&self) -> Result<(), ImageError> {
        self.inner.dependencies().map_err(build_err)
    }

    fn compile(&self, _name: &str, pkg: &str, dest: &str) -> Result<(), ImageError> {
        self.inner.compile(pkg, dest).map_err(build_err)
    }

    fn compile_rust(&self, crate_name: &str, bin: &str, dest: &str) -> Result<(), ImageError> {
        self.inner
            .compile_rust(crate_name, bin, dest)
            .map_err(build_err)
    }

    fn assets(&self, host_context: &str, forgejo_context: &str) -> Result<(), ImageError> {
        self.inner
            .assets(host_context, forgejo_context)
            .map_err(build_err)
    }

    fn images(
        &self,
        forgejo_context: &str,
    ) -> Result<HashMap<String, model::ProducedImage>, ImageError> {
        self.inner
            .images(forgejo_context)
            .map(|images| {
                images
                    .into_iter()
                    .map(|(name, produced)| (name, produced_of(&produced)))
                    .collect()
            })
            .map_err(build_err)
    }

    fn inspect_oci(
        &self,
        archive: &str,
        arch: &str,
        revision: &str,
    ) -> Result<model::Image, ImageError> {
        soda_release_deliver::oci::inspect_oci(archive, arch, revision).map_err(deliver_err)
    }

    fn verify_content(
        &self,
        payload: &model::Payload,
        dir: &str,
    ) -> Result<(HashMap<String, String>, u64), ImageError> {
        content::verify_content(&deliver_payload_of(payload), dir)
            .map(|(files, size)| (files.into_iter().collect(), size))
            .map_err(deliver_err)
    }

    fn resolve_core_os(&self) -> Result<ResolvedCoreOS, ImageError> {
        coreos_stream::resolve_coreos().map_err(build_err)
    }

    fn read_live_inputs(&self, path: &str) -> Result<LiveInputs, ImageError> {
        soda_release_build::live_inputs::read_live_inputs(Path::new(path)).map_err(build_err)
    }
}

// ---------------------------------------------------------------------------
// Progress adapter: the image `Progress` trait over the worker `BuildProgress`.
// The orphan rule is satisfied because `BuildProgress` is local to this crate.
// ---------------------------------------------------------------------------

impl ImageProgress for BuildProgress {
    fn phase(&mut self, label: &str) -> Result<(), ImageError> {
        self.phase(label).map_err(ImageError::msg)
    }

    fn next(&mut self, label: &str) -> Result<(), ImageError> {
        self.next(label).map_err(ImageError::msg)
    }

    fn end_phase(&mut self) -> Result<(), ImageError> {
        self.end_phase(None).map_err(ImageError::msg)
    }

    fn end(&mut self) -> Result<(), ImageError> {
        self.end(None).map_err(ImageError::msg)
    }

    fn create_log(&mut self, path: &str) -> Result<(), ImageError> {
        self.create_log(path).map_err(ImageError::msg)
    }

    fn note_reason(&mut self, reason: &str) {
        self.note_reason(reason);
    }
}

// ---------------------------------------------------------------------------
// Factory + worker stage entry point.
// ---------------------------------------------------------------------------

thread_local! {
    /// Current-thread image runner. The build hooks must be `Send + Sync`
    /// but the image runner is `Rc`-based, so the factory stashes it here
    /// and the hooks borrow it back. Everything runs synchronously on the
    /// worker thread; any other thread fails closed.
    static CURRENT_RUNNER: std::cell::RefCell<Option<Runner>> =
        const { std::cell::RefCell::new(None) };
}

fn with_current_runner<T>(
    run: impl FnOnce(&Runner) -> Result<T, ImageError>,
) -> Result<T, BuildError> {
    let result = CURRENT_RUNNER.with(|cell| match cell.borrow().as_ref() {
        Some(runner) => run(runner),
        None => Err(ImageError::msg(
            "production runner unavailable on this thread",
        )),
    });
    result.map_err(|e| BuildError::msg(e.0))
}

/// Build the real production for one pipeline run: the inner build
/// production carries the pipeline-computed inputs, with its command hooks
/// wired to the image runner (recall ring + build log, like the Go
/// `runBuildCommand` closures). The `Next` hook stays `None`: the factory
/// carries no progress handle, so inner step labels are dropped (the
/// pipeline's own phase labels are unaffected).
pub fn make_production(runner: Runner, inputs: ProductionInputs) -> Box<dyn ImageProduction> {
    CURRENT_RUNNER.with(|cell| {
        *cell.borrow_mut() = Some(runner);
    });
    let ProductionInputs {
        source,
        forgejo_source,
        forgejo_revision,
        native,
        out,
        arch,
        revision,
        live_inputs,
    } = inputs;
    let mut inner = BuildProduction::default();
    inner.source = source;
    inner.forgejo_source = forgejo_source;
    inner.forgejo_revision = forgejo_revision;
    inner.native = native;
    inner.out = out;
    inner.arch = arch;
    inner.revision = revision;
    inner.live_inputs = live_inputs;
    inner.execute = Some(Box::new(|dir, name, args| {
        with_current_runner(|runner| runner.execute(dir, name, args))
    }));
    inner.capture = Some(Box::new(|dir, name, args| {
        with_current_runner(|runner| runner.capture(dir, name, args))
    }));
    Box::new(RealProduction::new(inner))
}

/// Seed an image [`Cancel`] from the pending interrupt state. The global
/// flag is peeked at and restored: the outer CLI still owns interrupt
/// reporting and exit-code mapping after the worker stage returns.
fn seeded_cancel() -> Cancel {
    let cancel = Cancel::new();
    if let Some(interrupted) = exitcode::take_interrupt() {
        cancel.cancel();
        exitcode::note_interrupt(interrupted.exit_code());
    }
    cancel
}

struct CancellationWatcher {
    stop: std::sync::mpsc::Sender<()>,
    thread: Option<std::thread::JoinHandle<()>>,
}

impl CancellationWatcher {
    fn start(cancel: Cancel) -> CancellationWatcher {
        let (stop, stopped) = std::sync::mpsc::channel();
        let thread = std::thread::spawn(move || loop {
            if exitcode::has_interrupt() {
                cancel.cancel();
            }
            match stopped.recv_timeout(std::time::Duration::from_millis(10)) {
                Ok(()) | Err(std::sync::mpsc::RecvTimeoutError::Disconnected) => return,
                Err(std::sync::mpsc::RecvTimeoutError::Timeout) => {}
            }
        });
        CancellationWatcher {
            stop,
            thread: Some(thread),
        }
    }
}

impl Drop for CancellationWatcher {
    fn drop(&mut self) {
        let _ = self.stop.send(());
        if let Some(thread) = self.thread.take() {
            let _ = thread.join();
        }
    }
}

/// Run the worker-stage image build: map the worker request, seed
/// cancellation from the interrupt state, and run the real pipeline. The
/// `build_cli` wiring calls this; error text is the pipeline message.
pub fn run_worker_stage(
    request: &build_spec::Request,
    progress: &mut BuildProgress,
) -> Result<build_spec::ImageResult, String> {
    let mapped = image_request(request);
    let cancel = seeded_cancel();
    let _watcher = CancellationWatcher::start(cancel.clone());
    let result = soda_release_image::build::build(&cancel, &mapped, progress, &make_production)
        .map_err(|e| e.0)?;
    Ok(image_result(&result))
}

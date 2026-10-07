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
//! - Go binds `Production.Next` to `progress.Next`, emitting every step label
//!   as a progress section. The factory signature carries no progress handle,
//!   so the inner `Next` hook stays `None` and step labels are dropped; the
//!   pipeline's own phase labels still flow through the [`ImageProgress`]
//!   adapter below.
//! - [`run_worker_stage`] drops the image `Result.Checks` vector: the worker
//!   [`build_spec::ImageResult`] shape has no such field.
//! - Interrupts seed the image [`Cancel`] by peeking at (then restoring) the
//!   global interrupt flag, so the outer CLI still owns exit-code reporting.

use std::collections::HashMap;
use std::path::Path;

use soda_release_build::coreos::CoreOSImage as BuildCoreOSImage;
use soda_release_build::coreos_stream::{
    self, LiveInputs as BuildLiveInputs, ResolvedCoreOS as BuildResolvedCoreOS,
    TailnetInputs as BuildTailnetInputs,
};
use soda_release_build::oci::{self, Image as BuildImage};
use soda_release_build::production::{
    ProducedImage as BuildProducedImage, Production as BuildProduction,
};
use soda_release_build::Error as BuildError;
use soda_release_deliver::model::{Permit as DeliverPermit, Trust as DeliverTrust};
use soda_release_deliver::native::SecretFiles as DeliverSecrets;
use soda_release_deliver::payload::{Image as DeliverImage, Payload as DeliverPayload};
use soda_release_deliver::Error as DeliverError;
use soda_release_deliver::{content, document, native};
use soda_release_image::build::ProductionInputs;
use soda_release_image::build_runner::{Cancel, Runner};
use soda_release_image::error::Error as ImageError;
use soda_release_image::foreign::PackagingInputs;
use soda_release_image::foreign::{Production as ImageProduction, Progress as ImageProgress};
use soda_release_image::{model, request};

use crate::build_spec;
use crate::exitcode;
use crate::progress::BuildProgress;

/// Map the worker request onto the image pipeline request. All 13 fields
/// share names and shapes with the Go owner; the mapping is total.
pub fn image_request(r: &build_spec::Request) -> request::Request {
    request::Request {
        source: r.source.clone(),
        out: r.out.clone(),
        arch: r.arch.clone(),
        repository_prefix: r.repository_prefix.clone(),
        revision: r.revision.clone(),
        rootfs_base_url: r.rootfs_base_url.clone(),
        media_authority: r.media_authority.clone(),
        live_inputs: r.live_inputs.clone(),
        forgejo_source: r.forgejo_source.clone(),
        forgejo_revision: r.forgejo_revision.clone(),
        development: r.development,
        target: r.target.clone(),
        media_compression: r.media_compression.clone(),
    }
}

/// Map the pipeline result onto the worker result. The image `Checks` vector
/// has no worker counterpart and is dropped.
fn image_result(r: &request::Result) -> build_spec::ImageResult {
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
    }
}

fn build_err(e: BuildError) -> ImageError {
    ImageError::msg(e.message())
}

fn deliver_err(e: DeliverError) -> ImageError {
    ImageError::msg(e.0)
}

// ---------------------------------------------------------------------------
// Model conversions: the image `model` mirrors and the build/deliver owners
// derive from the same Go structs, so every conversion is field-wise. The
// image mirrors use sorted `Vec` pairs where the owners use maps; conversions
// sort keys to keep the mapping deterministic.
// ---------------------------------------------------------------------------

fn image_of(image: &BuildImage) -> model::Image {
    model::Image {
        manifest: image.manifest.clone(),
        config: image.config.clone(),
        architecture: image.architecture.clone(),
        revision: image.revision.clone(),
        source: image.source.clone(),
        base_name: image.base_name.clone(),
        base_digest: image.base_digest.clone(),
    }
}

fn produced_of(produced: &BuildProducedImage) -> model::ProducedImage {
    model::ProducedImage {
        manifest: produced.image.manifest.clone(),
        config: produced.image.config.clone(),
        archive_sha256: produced.archive_sha256.clone(),
    }
}

fn coreos_image_of(image: &BuildCoreOSImage) -> model::CoreOSImage {
    model::CoreOSImage {
        url: image.url.clone(),
        signature_url: image.signature_url.clone(),
        sha256: image.sha256.clone(),
        uncompressed_sha256: image.uncompressed_sha256.clone(),
    }
}

fn sorted_pairs(map: &HashMap<String, String>) -> Vec<(String, String)> {
    let mut out: Vec<(String, String)> = map.iter().map(|(k, v)| (k.clone(), v.clone())).collect();
    out.sort_by(|a, b| a.0.cmp(&b.0));
    out
}

fn sorted_images(map: &HashMap<String, BuildCoreOSImage>) -> Vec<(String, model::CoreOSImage)> {
    let mut out: Vec<(String, model::CoreOSImage)> = map
        .iter()
        .map(|(k, v)| (k.clone(), coreos_image_of(v)))
        .collect();
    out.sort_by(|a, b| a.0.cmp(&b.0));
    out
}

fn resolved_coreos_of(resolved: &BuildResolvedCoreOS) -> model::ResolvedCoreOS {
    model::ResolvedCoreOS {
        release: resolved.release.clone(),
        metadata_url: resolved.metadata_url.clone(),
        container: sorted_pairs(&resolved.container),
        iso: sorted_images(&resolved.iso),
        qemu: sorted_images(&resolved.qemu),
    }
}

fn tailnet_of(tailnet: &BuildTailnetInputs) -> model::TailnetInputs {
    model::TailnetInputs {
        version: tailnet.version.clone(),
        sha256: tailnet.sha256.clone(),
        base: tailnet.base.clone(),
    }
}

fn live_inputs_of(inputs: &BuildLiveInputs) -> model::LiveInputs {
    model::LiveInputs {
        core_os: resolved_coreos_of(&inputs.coreos),
        tailnet: tailnet_of(&inputs.tailnet),
    }
}

fn deliver_payload_of(payload: &model::Payload) -> DeliverPayload {
    DeliverPayload {
        format: payload.format,
        id: payload.id.clone(),
        revision: payload.revision.clone(),
        architecture: payload.architecture.clone(),
        core_os: payload.core_os.clone(),
        base: payload.base.clone(),
        repository_prefix: payload.repository_prefix.clone(),
        schema: payload.schema,
        presentation_sha256: payload.presentation_sha256.clone(),
        host_packages_sha256: payload.host_packages_sha256.clone(),
        images: payload
            .images
            .iter()
            .map(|(name, image)| {
                (
                    name.clone(),
                    DeliverImage {
                        reference: image.reference.clone(),
                        config: image.config.clone(),
                        manifest: image.manifest.clone(),
                        archive_sha256: image.archive_sha256.clone(),
                    },
                )
            })
            .collect(),
        // Go leaves `UpgradeFrom` unset (`nil`) for candidates; only a
        // non-empty list becomes `Some`.
        upgrade_from: if payload.upgrade_from.is_empty() {
            None
        } else {
            Some(payload.upgrade_from.clone())
        },
    }
}

fn deliver_trust_of(trust: &model::Trust) -> DeliverTrust {
    DeliverTrust {
        format: trust.format,
        prefix: trust.prefix.clone(),
        epoch: trust.epoch,
        keys: trust.keys.iter().cloned().collect(),
        not_before: trust.not_before,
        max_age_seconds: trust.max_age_seconds,
        clock_skew_seconds: trust.clock_skew_seconds,
        minimum_sequence: trust.minimum_sequence.iter().cloned().collect(),
    }
}

fn deliver_permit_of(permit: &model::Permit) -> DeliverPermit {
    DeliverPermit {
        format: permit.format,
        repository: permit.repository.clone(),
        digest: permit.digest.clone(),
        previous: permit.previous.clone(),
        expires: permit.expires,
    }
}

fn deliver_secrets_of(secrets: &model::SecretFiles) -> DeliverSecrets {
    DeliverSecrets {
        key: secrets.key.clone(),
        passphrase: secrets.passphrase.clone(),
    }
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

    fn next(&self, label: &str) -> Result<(), ImageError> {
        self.inner.step(label).map_err(build_err)
    }

    fn resolve_inputs(&mut self) -> Result<(), ImageError> {
        self.inner.resolve_inputs().map_err(build_err)
    }

    fn dependencies(&self) -> Result<(), ImageError> {
        self.inner.dependencies().map_err(build_err)
    }

    fn compile(&self, name: &str, pkg: &str, dest: &str) -> Result<(), ImageError> {
        self.inner.compile(name, pkg, dest).map_err(build_err)
    }

    fn compile_rust(&self, crate_name: &str, bin: &str, dest: &str) -> Result<(), ImageError> {
        self.inner
            .compile_rust(crate_name, bin, dest)
            .map_err(build_err)
    }

    fn stage_fork_binary(&self, context: &str) -> Result<(), ImageError> {
        self.inner.stage_fork_binary(context).map_err(build_err)
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
        oci::inspect_oci(Path::new(archive), arch, revision)
            .map(|image| image_of(&image))
            .map_err(build_err)
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

    fn resolve_core_os(&self) -> Result<model::ResolvedCoreOS, ImageError> {
        coreos_stream::resolve_coreos()
            .map(|resolved| resolved_coreos_of(&resolved))
            .map_err(build_err)
    }

    fn read_live_inputs(&self, path: &str) -> Result<model::LiveInputs, ImageError> {
        coreos_stream::read_live_inputs(Path::new(path))
            .map(|inputs| live_inputs_of(&inputs))
            .map_err(build_err)
    }

    fn check_native(&self, trust_home: &str) -> Result<(), ImageError> {
        // Go runs these against `deliver.Native{Home: <trust home>}` with the
        // ambient context; the Rust owner takes no context.
        let runner = native::Native {
            home: trust_home.to_string(),
        };
        native::check_native(&runner).map_err(deliver_err)
    }

    fn sign_media(
        &self,
        trust: &model::Trust,
        permit: &model::Permit,
        transport: &str,
        input: &str,
        out: &str,
        keys: &model::SecretFiles,
        trust_home: &str,
    ) -> Result<(), ImageError> {
        let runner = native::Native {
            home: trust_home.to_string(),
        };
        native::sign(
            &runner,
            &deliver_trust_of(trust),
            &deliver_permit_of(permit),
            transport,
            input,
            out,
            &deliver_secrets_of(keys),
        )
        .map_err(deliver_err)
    }

    fn verify_copy(
        &self,
        trust: &model::Trust,
        reference: &str,
        source: &str,
        out: &str,
        trust_home: &str,
    ) -> Result<(), ImageError> {
        let runner = native::Native {
            home: trust_home.to_string(),
        };
        native::verify_copy(&runner, &deliver_trust_of(trust), reference, source, out)
            .map_err(deliver_err)
    }

    fn write_document(&self, path: &str, value: &PackagingInputs) -> Result<String, ImageError> {
        document::write_document(path, value).map_err(deliver_err)
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
    inner.next = None;
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

/// Run the worker-stage image build: map the worker request, seed
/// cancellation from the interrupt state, and run the real pipeline. The
/// `build_cli` wiring calls this; error text is the pipeline message.
pub fn run_worker_stage(
    request: &build_spec::Request,
    progress: &mut BuildProgress,
) -> Result<build_spec::ImageResult, String> {
    let mapped = image_request(request);
    let cancel = seeded_cancel();
    let result = soda_release_image::build::build(&cancel, &mapped, progress, &make_production)
        .map_err(|e| e.0)?;
    Ok(image_result(&result))
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::rc::Rc;
    use std::sync::{Arc, Mutex};
    use std::time::Duration;

    fn spec_request() -> build_spec::Request {
        build_spec::Request {
            source: "/src".to_string(),
            out: "/out".to_string(),
            arch: "x86_64".to_string(),
            repository_prefix: "ghcr.io/example/sodaos".to_string(),
            revision: "a".repeat(40),
            rootfs_base_url: "https://example.invalid/rootfs".to_string(),
            media_authority: "/authority".to_string(),
            live_inputs: "/live.json".to_string(),
            forgejo_source: "/forgejo".to_string(),
            forgejo_revision: "b".repeat(40),
            development: true,
            target: "media".to_string(),
            media_compression: "fast".to_string(),
        }
    }

    #[test]
    fn request_mapping_round_trip() {
        let spec = spec_request();
        let mapped = image_request(&spec);
        assert_eq!(mapped.source, spec.source);
        assert_eq!(mapped.out, spec.out);
        assert_eq!(mapped.arch, spec.arch);
        assert_eq!(mapped.repository_prefix, spec.repository_prefix);
        assert_eq!(mapped.revision, spec.revision);
        assert_eq!(mapped.rootfs_base_url, spec.rootfs_base_url);
        assert_eq!(mapped.media_authority, spec.media_authority);
        assert_eq!(mapped.live_inputs, spec.live_inputs);
        assert_eq!(mapped.forgejo_source, spec.forgejo_source);
        assert_eq!(mapped.forgejo_revision, spec.forgejo_revision);
        assert_eq!(mapped.development, spec.development);
        assert_eq!(mapped.target, spec.target);
        assert_eq!(mapped.media_compression, spec.media_compression);
    }

    #[test]
    fn result_mapping_drops_checks() {
        let result = request::Result {
            revision: "rev".to_string(),
            architecture: "amd64".to_string(),
            candidate: "cand".to_string(),
            candidate_sha256: "csha".to_string(),
            host_manifest: "hm".to_string(),
            payload_sha256: "psha".to_string(),
            scope: "scope".to_string(),
            media: "media".to_string(),
            purpose: "production".to_string(),
            requested_target: "release".to_string(),
            completed_target: "release".to_string(),
            checks: vec!["c1".to_string(), "c2".to_string()],
            media_compression: "".to_string(),
        };
        let mapped = image_result(&result);
        assert_eq!(mapped.revision, "rev");
        assert_eq!(mapped.architecture, "amd64");
        assert_eq!(mapped.candidate, "cand");
        assert_eq!(mapped.candidate_sha256, "csha");
        assert_eq!(mapped.host_manifest, "hm");
        assert_eq!(mapped.payload_sha256, "psha");
        assert_eq!(mapped.scope, "scope");
        assert_eq!(mapped.media, "media");
        assert_eq!(mapped.purpose, "production");
        assert_eq!(mapped.requested_target, "release");
        assert_eq!(mapped.completed_target, "release");
        assert_eq!(mapped.media_compression, "");
    }

    /// Build a captured progress. Other tests in this binary mutate
    /// `SODA_BUILD_START_NS`, so pin a valid origin and retry through races.
    fn test_progress(title: &str) -> (BuildProgress, Arc<Mutex<Vec<u8>>>) {
        for _ in 0..100 {
            unsafe {
                std::env::set_var("SODA_BUILD_START_NS", "1");
            }
            match BuildProgress::new_with_clock(
                title,
                Box::new(|| Duration::from_nanos(1_000_000_000)),
            ) {
                Ok(mut progress) => {
                    let buf = progress.capture();
                    return (progress, buf);
                }
                Err(_) => std::thread::yield_now(),
            }
        }
        panic!("could not construct BuildProgress (environment race)");
    }

    #[test]
    fn progress_adapter_call_through() {
        let (mut progress, buf) = test_progress("worker");
        ImageProgress::phase(&mut progress, "P1 / Admit").unwrap();
        ImageProgress::next(&mut progress, "Compile it").unwrap();
        progress.note_reason("kept for failures");
        ImageProgress::end(&mut progress).unwrap();
        ImageProgress::end_phase(&mut progress).unwrap();
        let log_path = std::env::temp_dir().join(format!(
            "soda-pipeline-{}.log",
            std::time::SystemTime::now()
                .duration_since(std::time::UNIX_EPOCH)
                .unwrap()
                .as_nanos()
        ));
        let log_path = log_path.to_string_lossy().into_owned();
        ImageProgress::create_log(&mut progress, &log_path).unwrap();
        let text = String::from_utf8(buf.lock().unwrap().clone()).unwrap();
        assert!(text.contains("START    P1 / Admit\n"), "{text}");
        assert!(text.contains("START    Compile it\n"), "{text}");
        assert!(text.contains("DONE     Compile it | section "), "{text}");
        assert!(text.contains("DONE     P1 / Admit | phase "), "{text}");
        assert!(text.contains(&format!("LOG      {log_path}\n")), "{text}");
        let _ = std::fs::remove_file(&log_path);
    }

    /// Fixture inner production with recording hooks, following the
    /// `soda-release-build` production test construction pattern.
    fn fixture_inner() -> (BuildProduction, Arc<Mutex<Vec<String>>>) {
        let calls: Arc<Mutex<Vec<String>>> = Arc::new(Mutex::new(Vec::new()));
        let mut inner = BuildProduction::default();
        inner.source = "/src".to_string();
        inner.forgejo_source = "/forgejo".to_string();
        inner.forgejo_revision = "f".repeat(40);
        inner.native = "/native".to_string();
        inner.out = "/out".to_string();
        inner.arch = "x86_64".to_string();
        inner.revision = "a".repeat(40);
        inner.live_inputs = "/live.json".to_string();
        let exec_calls = Arc::clone(&calls);
        inner.execute = Some(Box::new(move |dir, name, args| {
            exec_calls
                .lock()
                .unwrap()
                .push(format!("EXEC {dir} {name} {}", args.join(" ")));
            Ok(())
        }));
        let capture_calls = Arc::clone(&calls);
        inner.capture = Some(Box::new(move |dir, name, args| {
            capture_calls
                .lock()
                .unwrap()
                .push(format!("CAP {dir} {name} {}", args.join(" ")));
            Ok("captured".to_string())
        }));
        let next_calls = Arc::clone(&calls);
        inner.next = Some(Box::new(move |label| {
            next_calls.lock().unwrap().push(format!("NEXT {label}"));
            Ok(())
        }));
        (inner, calls)
    }

    #[test]
    fn accessors_passthrough() {
        let (inner, _) = fixture_inner();
        let production = RealProduction::new(inner);
        assert_eq!(ImageProduction::source(&production), "/src");
        assert_eq!(ImageProduction::forgejo_source(&production), "/forgejo");
        assert_eq!(
            ImageProduction::forgejo_revision(&production),
            "f".repeat(40)
        );
        assert_eq!(ImageProduction::native(&production), "/native");
        assert_eq!(ImageProduction::out(&production), "/out");
        assert_eq!(ImageProduction::arch(&production), "x86_64");
        assert_eq!(ImageProduction::revision(&production), "a".repeat(40));
        assert_eq!(ImageProduction::live_inputs(&production), "/live.json");
    }

    #[test]
    fn hooks_call_through() {
        let (inner, calls) = fixture_inner();
        let production = RealProduction::new(inner);
        ImageProduction::next(&production, "Compile it").unwrap();
        ImageProduction::execute(&production, "/d", "podman", &["pull".to_string()]).unwrap();
        let out =
            ImageProduction::capture(&production, "/d", "podman", &["id".to_string()]).unwrap();
        assert_eq!(out, "captured");
        let calls = calls.lock().unwrap().join("\n");
        assert!(calls.contains("NEXT Compile it"), "{calls}");
        assert!(calls.contains("EXEC /d podman pull"), "{calls}");
        assert!(calls.contains("CAP /d podman id"), "{calls}");
    }

    #[test]
    fn hook_errors_map() {
        let (mut inner, _) = fixture_inner();
        inner.execute = Some(Box::new(|_, _, _| Err(BuildError::msg("boom"))));
        inner.capture = Some(Box::new(|_, _, _| Err(BuildError::msg("snap"))));
        inner.next = Some(Box::new(|_| Err(BuildError::msg("crackle"))));
        let production = RealProduction::new(inner);
        assert_eq!(
            ImageProduction::execute(&production, "/d", "x", &[])
                .unwrap_err()
                .0,
            "boom"
        );
        assert_eq!(
            ImageProduction::capture(&production, "/d", "x", &[])
                .unwrap_err()
                .0,
            "snap"
        );
        assert_eq!(
            ImageProduction::next(&production, "step").unwrap_err().0,
            "crackle"
        );
        let bare = RealProduction::new(BuildProduction::default());
        assert_eq!(
            ImageProduction::capture(&bare, "/d", "x", &[])
                .unwrap_err()
                .0,
            "explicit native production inputs required"
        );
        assert_eq!(
            ImageProduction::execute(&bare, "/d", "x", &[])
                .unwrap_err()
                .0,
            "explicit native production inputs required"
        );
        // No `Next` hook wired (as in `make_production`): steps are dropped.
        assert!(ImageProduction::next(&bare, "step").is_ok());
    }

    #[test]
    fn build_model_conversions() {
        let image = BuildImage {
            manifest: "m".to_string(),
            config: "c".to_string(),
            architecture: "amd64".to_string(),
            revision: "r".to_string(),
            source: "s".to_string(),
            base_name: "b".to_string(),
            base_digest: "d".to_string(),
        };
        let converted = image_of(&image);
        assert_eq!(converted.manifest, "m");
        assert_eq!(converted.config, "c");
        assert_eq!(converted.architecture, "amd64");
        assert_eq!(converted.revision, "r");
        assert_eq!(converted.source, "s");
        assert_eq!(converted.base_name, "b");
        assert_eq!(converted.base_digest, "d");
        let produced = produced_of(&BuildProducedImage {
            image: image.clone(),
            archive_sha256: "h".to_string(),
        });
        assert_eq!(produced.manifest, "m");
        assert_eq!(produced.config, "c");
        assert_eq!(produced.archive_sha256, "h");

        let mut container = HashMap::new();
        container.insert("x86_64".to_string(), "pinned".to_string());
        let mut iso = HashMap::new();
        iso.insert(
            "x86_64".to_string(),
            BuildCoreOSImage {
                url: "https://example.invalid/f.iso".to_string(),
                signature_url: "https://example.invalid/f.iso.sig".to_string(),
                sha256: "s".to_string(),
                uncompressed_sha256: "u".to_string(),
            },
        );
        let resolved = resolved_coreos_of(&BuildResolvedCoreOS {
            release: "1.2.3.4".to_string(),
            metadata_url: "https://example.invalid/m".to_string(),
            container,
            iso,
            qemu: HashMap::new(),
        });
        assert_eq!(resolved.release, "1.2.3.4");
        assert_eq!(
            resolved.container,
            vec![("x86_64".to_string(), "pinned".to_string())]
        );
        assert_eq!(resolved.iso.len(), 1);
        assert_eq!(resolved.iso[0].1.url, "https://example.invalid/f.iso");
        assert!(resolved.qemu.is_empty());
        let live = live_inputs_of(&BuildLiveInputs {
            coreos: BuildResolvedCoreOS {
                release: "1.2.3.4".to_string(),
                ..BuildResolvedCoreOS::default()
            },
            tailnet: BuildTailnetInputs {
                version: "1.2.3".to_string(),
                sha256: "s".to_string(),
                base: "docker.io/tailscale/alpine-base:1.2".to_string(),
            },
        });
        assert_eq!(live.core_os.release, "1.2.3.4");
        assert_eq!(live.tailnet.version, "1.2.3");
        assert_eq!(live.tailnet.base, "docker.io/tailscale/alpine-base:1.2");
    }

    #[test]
    fn deliver_model_conversions() {
        let payload = model::Payload {
            format: 3,
            id: "id".to_string(),
            revision: "r".to_string(),
            architecture: "x86_64".to_string(),
            core_os: "1.2.3.4".to_string(),
            base: "base".to_string(),
            repository_prefix: "ghcr.io/e/s".to_string(),
            schema: 1,
            presentation_sha256: "p".to_string(),
            host_packages_sha256: "h".to_string(),
            images: vec![(
                "proxy".to_string(),
                model::PayloadImage {
                    reference: "ref".to_string(),
                    config: "cfg".to_string(),
                    manifest: "man".to_string(),
                    archive_sha256: "arc".to_string(),
                },
            )],
            upgrade_from: Vec::new(),
        };
        let converted = deliver_payload_of(&payload);
        assert_eq!(converted.format, 3);
        assert_eq!(converted.images.len(), 1);
        assert_eq!(converted.images["proxy"].reference, "ref");
        assert_eq!(converted.images["proxy"].archive_sha256, "arc");
        assert_eq!(converted.upgrade_from, None);
        let payload = model::Payload {
            upgrade_from: vec!["old".to_string()],
            ..payload
        };
        assert_eq!(
            deliver_payload_of(&payload).upgrade_from,
            Some(vec!["old".to_string()])
        );

        let trust = model::Trust {
            format: 1,
            prefix: "ghcr.io/e/s".to_string(),
            epoch: 7,
            keys: vec![("stable".to_string(), vec!["k".to_string()])],
            not_before: 1,
            max_age_seconds: 60,
            clock_skew_seconds: 0,
            minimum_sequence: vec![("stable".to_string(), 3)],
        };
        let converted = deliver_trust_of(&trust);
        assert_eq!(converted.epoch, 7);
        assert_eq!(converted.keys["stable"], vec!["k".to_string()]);
        assert_eq!(converted.minimum_sequence["stable"], 3);
        let permit = model::Permit {
            format: 1,
            repository: "repo".to_string(),
            digest: "digest".to_string(),
            previous: "prev".to_string(),
            expires: 9,
        };
        let converted = deliver_permit_of(&permit);
        assert_eq!(converted.repository, "repo");
        assert_eq!(converted.expires, 9);
        let secrets = model::SecretFiles {
            key: "/k".to_string(),
            passphrase: "/p".to_string(),
        };
        let converted = deliver_secrets_of(&secrets);
        assert_eq!(converted.key, "/k");
        assert_eq!(converted.passphrase, "/p");
    }

    #[test]
    fn deliver_refusals_map() {
        // All three fail in admission, before any filesystem or subprocess use.
        let production = RealProduction::new(BuildProduction::default());
        assert_eq!(
            ImageProduction::verify_content(&production, &model::Payload::default(), "/tmp/x")
                .unwrap_err()
                .0,
            "invalid appliance payload identity"
        );
        assert_eq!(
            ImageProduction::sign_media(
                &production,
                &model::Trust::default(),
                &model::Permit::default(),
                "oci",
                "/in",
                "/out",
                &model::SecretFiles::default(),
                "/home",
            )
            .unwrap_err()
            .0,
            "release authority or completeness refused"
        );
        assert_eq!(
            ImageProduction::verify_copy(
                &production,
                &model::Trust::default(),
                "not-a-reference",
                "dir:/x",
                "/tmp/soda-pipeline-verify",
                "/home",
            )
            .unwrap_err()
            .0,
            "release authority or completeness refused"
        );
    }

    #[test]
    fn factory_wires_inputs_and_runner_hooks() {
        let runner = Runner::new(Rc::new(Cancel::new()));
        let inputs = ProductionInputs {
            source: "/snap".to_string(),
            forgejo_source: "/forge".to_string(),
            forgejo_revision: "f".to_string(),
            native: "/nat".to_string(),
            out: "/art".to_string(),
            arch: "x86_64".to_string(),
            revision: "r".to_string(),
            live_inputs: "/live".to_string(),
        };
        let production = make_production(runner, inputs);
        assert_eq!(production.source(), "/snap");
        assert_eq!(production.forgejo_source(), "/forge");
        assert_eq!(production.forgejo_revision(), "f");
        assert_eq!(production.native(), "/nat");
        assert_eq!(production.out(), "/art");
        assert_eq!(production.arch(), "x86_64");
        assert_eq!(production.revision(), "r");
        assert_eq!(production.live_inputs(), "/live");
        // End-to-end hook bridge: harmless real commands through the
        // thread-local runner, proving the factory wiring executes.
        production.execute("/tmp", "true", &[]).unwrap();
        let out = production
            .capture("/tmp", "echo", &["hello".to_string()])
            .unwrap();
        assert_eq!(out, "hello");
        assert!(production.next("dropped step").is_ok());
    }

    #[test]
    fn cancel_seeds_from_interrupt_and_restores() {
        let _guard = crate::exitcode::interrupt_test_lock();
        let _ = exitcode::take_interrupt();
        exitcode::note_interrupt(130);
        let cancel = seeded_cancel();
        assert!(cancel.is_cancelled());
        assert_eq!(exitcode::take_interrupt().map(|i| i.exit_code()), Some(130));
        assert!(exitcode::take_interrupt().is_none());
        assert!(!seeded_cancel().is_cancelled());
    }
}

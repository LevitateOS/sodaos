//! `build.go`: single candidate execution owner + tool environment.
//!
//! Qualification and protected delivery consume its unchanged bytes; the
//! result is not a qualified release.

use std::collections::HashMap;
use std::fs;
use std::rc::Rc;
use std::sync::atomic::AtomicBool;

use crate::build_compile::compile_shipping_tools;
use crate::build_media;
use crate::build_runner::{Cancel, LogCloser, Runner};
use crate::build_source::admit_build_inputs;
use crate::compression;
use crate::error::Error;
use crate::files;
use crate::foreign::{Production, Progress};
use crate::host;
use crate::jsonio;
use crate::media;
use crate::model;
use crate::payload_stage;
use crate::prepare;
use crate::recall;
use crate::record;
use crate::request;
use crate::sys;

pub use crate::build_candidate::build_host_candidate;
pub use crate::build_context::{
    freeze_base_image_config, link_prepared_assets, prepare_build_host_context,
};

/// Pinned Go toolchain (`go1.26.7`): the controller shells to it for Soda
/// command compilation, and `GOTOOLCHAIN` forces the same version for every
/// child `go` invocation.
pub const PINNED_GO_VERSION: &str = "go1.26.7";

/// Computed production inputs (`build.Production` fields set by `runBuild`).
#[derive(Debug, Clone, Default)]
pub struct ProductionInputs {
    pub source: String,
    pub forgejo_source: String,
    pub forgejo_revision: String,
    pub native: String,
    pub out: String,
    pub arch: String,
    pub revision: String,
    pub live_inputs: String,
}

/// Build is the single candidate execution owner. The caller supplies the
/// foreign production implementation; the runner wires its commands into
/// the recall ring and build log.
pub fn build(
    cancel: &Cancel,
    request: &request::Request,
    progress: &mut dyn Progress,
    make_production: &dyn Fn(Runner, ProductionInputs) -> Box<dyn Production>,
) -> Result<request::Result, Error> {
    let runner = Runner::new(Rc::new(Cancel {
        flag: AtomicBool::new(cancel.is_cancelled()),
    }));
    // Poll the caller's flag through the run via shared ownership.
    let result = run_build(request, progress, &runner, make_production, &|path| {
        runner.open_log(path, request.wants_media())
    });
    if let Err(err) = &result {
        progress.note_reason(&recall::failure_reason(&runner.log.borrow(), err));
    }
    let _ = cancel;
    result
}

pub fn init_build_directories(out: &str) -> Result<(), Error> {
    for dir in ["work", "artifacts", "evidence", "release", "logs"] {
        sys::create_dir(&sys::join(&[out, dir]), 0o700)?;
    }
    Ok(())
}

pub fn extract_build_snapshot(
    source: &str,
    out: &str,
    revision: &str,
    runner: &Runner,
) -> Result<String, Error> {
    let snapshot = sys::join(&[out, "work/source"]);
    sys::create_dir(&snapshot, 0o700)?;
    let archive = sys::join(&[out, "artifacts/source.tar"]);
    runner.execute(
        source,
        "git",
        &[
            "archive".to_string(),
            "--format=tar".to_string(),
            "--output".to_string(),
            archive.clone(),
            revision.to_string(),
        ],
    )?;
    runner.execute(
        &snapshot,
        "tar",
        &[
            "--extract".to_string(),
            "--file".to_string(),
            archive,
            "--no-same-owner".to_string(),
        ],
    )?;
    Ok(snapshot)
}

pub fn setup_build_workspace(
    request: &request::Request,
    revision: &str,
    progress: &mut dyn Progress,
    open_log: &dyn Fn(&str) -> Result<LogCloser, Error>,
    runner: &Runner,
) -> Result<(String, LogCloser), Error> {
    init_build_directories(&request.out)?;
    progress.create_log(&sys::join(&[&request.out, "logs/timing.log"]))?;
    let close_log = open_log(&sys::join(&[&request.out, "logs/build.log"]))?;
    let snapshot = match extract_build_snapshot(&request.source, &request.out, revision, runner) {
        Ok(snapshot) => snapshot,
        Err(e) => {
            let _ = close_log.close();
            return Err(e);
        }
    };
    // NOTE: forgejo extraction needs a Production; the caller passes one via
    // run_build. Here we only snapshot; see run_build for the join.
    Ok((snapshot, close_log))
}

/// prepareBuildProduction shares one Production: ResolveInputs records the
/// frozen image inputs on it, and later phases read them back from the same
/// value.
pub fn prepare_build_production(
    production: &mut dyn Production,
    request: &request::Request,
    snapshot: &str,
    revision: &str,
    next: &mut dyn FnMut(&str) -> Result<(), Error>,
) -> Result<
    (
        String,
        prepare::Base,
        build_media::MediaTools,
        media::MediaLock,
    ),
    Error,
> {
    let (context_dir, base) = prepare_build_host_context(
        snapshot,
        &request.out,
        &request.arch,
        revision,
        &request.repository_prefix,
        &request.media_compression,
        &request.live_inputs,
        production,
    )?;
    next("P2 / Verify and install frozen dependencies")?;
    production.dependencies()?;
    let (tooling, assembler) = build_media::prepare_build_media(production, request)?;
    Ok((context_dir, base, tooling, assembler))
}

pub fn execute_build_production(
    production: &dyn Production,
    request: &request::Request,
    snapshot: &str,
    context_dir: &str,
    artifacts: &str,
    revision: &str,
    base: &prepare::Base,
    media_tooling: &build_media::MediaTools,
    assembler: &mut media::MediaLock,
    phase: &mut dyn FnMut(&str) -> Result<(), Error>,
) -> Result<request::Result, Error> {
    phase("P3 / Compile shipping programs")?;
    compile_shipping_tools(
        production,
        snapshot,
        context_dir,
        artifacts,
        revision,
        &request.arch,
    )?;
    build_host_candidate(
        snapshot,
        context_dir,
        artifacts,
        &request.arch,
        revision,
        &request.repository_prefix,
        base,
        production,
        phase,
    )?;
    build_media::finish_build_media(production, request, media_tooling, assembler, phase)?;
    record::record_build_result(production, request)
}

pub fn finalize_build(
    progress: &mut dyn Progress,
    result: request::Result,
    err: Result<(), Error>,
) -> Result<request::Result, Error> {
    match err {
        Ok(()) => {
            let end = progress.end();
            let end_phase = progress.end_phase();
            match (end, end_phase) {
                (Ok(()), Ok(())) => Ok(result),
                (Err(e), _) | (_, Err(e)) => Err(e),
            }
        }
        Err(e) => Err(e),
    }
}

pub fn run_build(
    request: &request::Request,
    progress: &mut dyn Progress,
    runner: &Runner,
    make_production: &dyn Fn(Runner, ProductionInputs) -> Box<dyn Production>,
    open_log: &dyn Fn(&str) -> Result<LogCloser, Error>,
) -> Result<request::Result, Error> {
    let revision = admit_build_inputs(request, runner)?;
    progress.phase("P1 / Admit and freeze inputs")?;
    let (snapshot, close_log) =
        setup_build_workspace(request, &revision, progress, open_log, runner)?;
    let result = run_build_inner(
        request,
        progress,
        runner,
        make_production,
        &snapshot,
        &revision,
    );
    // Go defers errors.Join(err, closeLog()).
    match (result, close_log.close()) {
        (Ok(result), Ok(())) => Ok(result),
        (Err(e), Ok(())) | (Ok(_), Err(e)) => Err(e),
        (Err(first), Err(second)) => Err(Error(format!("{}\n{}", first.0, second.0))),
    }
}

fn run_build_inner(
    request: &request::Request,
    progress: &mut dyn Progress,
    runner: &Runner,
    make_production: &dyn Fn(Runner, ProductionInputs) -> Box<dyn Production>,
    snapshot: &str,
    revision: &str,
) -> Result<request::Result, Error> {
    let artifacts = sys::join(&[&request.out, "artifacts"]);
    // Forgejo extraction runs against the source runner (same commands the
    // Go Production.Execute would run); the production below replays phases.
    {
        struct RunnerProduction {
            runner: Runner,
            inputs: ProductionInputs,
        }
        impl Production for RunnerProduction {
            fn source(&self) -> &str {
                &self.inputs.source
            }
            fn forgejo_source(&self) -> &str {
                &self.inputs.forgejo_source
            }
            fn forgejo_revision(&self) -> &str {
                &self.inputs.forgejo_revision
            }
            fn native(&self) -> &str {
                &self.inputs.native
            }
            fn out(&self) -> &str {
                &self.inputs.out
            }
            fn arch(&self) -> &str {
                &self.inputs.arch
            }
            fn revision(&self) -> &str {
                &self.inputs.revision
            }
            fn live_inputs(&self) -> &str {
                &self.inputs.live_inputs
            }
            fn execute(&self, dir: &str, name: &str, args: &[String]) -> Result<(), Error> {
                self.runner.execute(dir, name, args)
            }
            fn capture(&self, dir: &str, name: &str, args: &[String]) -> Result<String, Error> {
                self.runner.capture(dir, name, args)
            }
            fn next(&self, _label: &str) -> Result<(), Error> {
                Ok(())
            }
            fn resolve_inputs(&mut self) -> Result<(), Error> {
                Err(Error::msg("foreign production required"))
            }
            fn dependencies(&self) -> Result<(), Error> {
                Err(Error::msg("foreign production required"))
            }
            fn compile(&self, _: &str, _: &str, _: &str) -> Result<(), Error> {
                Err(Error::msg("foreign production required"))
            }
            fn compile_rust(&self, _: &str, _: &str, _: &str) -> Result<(), Error> {
                Err(Error::msg("foreign production required"))
            }
            fn stage_fork_binary(&self, _: &str) -> Result<(), Error> {
                Err(Error::msg("foreign production required"))
            }
            fn assets(&self, _: &str, _: &str) -> Result<(), Error> {
                Err(Error::msg("foreign production required"))
            }
            fn images(&self, _: &str) -> Result<HashMap<String, model::ProducedImage>, Error> {
                Err(Error::msg("foreign production required"))
            }
            fn inspect_oci(&self, _: &str, _: &str, _: &str) -> Result<model::Image, Error> {
                Err(Error::msg("foreign production required"))
            }
            fn verify_content(
                &self,
                _: &model::Payload,
                _: &str,
            ) -> Result<(HashMap<String, String>, u64), Error> {
                Err(Error::msg("foreign production required"))
            }
            fn resolve_core_os(&self) -> Result<model::ResolvedCoreOS, Error> {
                Err(Error::msg("foreign production required"))
            }
            fn read_live_inputs(&self, _: &str) -> Result<model::LiveInputs, Error> {
                Err(Error::msg("foreign production required"))
            }
            fn check_native(&self, _: &str) -> Result<(), Error> {
                Err(Error::msg("foreign production required"))
            }
            fn sign_media(
                &self,
                _: &model::Trust,
                _: &model::Permit,
                _: &str,
                _: &str,
                _: &str,
                _: &model::SecretFiles,
                _: &str,
            ) -> Result<(), Error> {
                Err(Error::msg("foreign production required"))
            }
            fn verify_copy(
                &self,
                _: &model::Trust,
                _: &str,
                _: &str,
                _: &str,
                _: &str,
            ) -> Result<(), Error> {
                Err(Error::msg("foreign production required"))
            }
            fn write_document(&self, _: &str, _: &soda_json::JsonValue) -> Result<String, Error> {
                Err(Error::msg("foreign production required"))
            }
        }
        let bootstrap = RunnerProduction {
            runner: runner.clone(),
            inputs: ProductionInputs::default(),
        };
        crate::forgejo::extract_forgejo_snapshot(&bootstrap, request)?;
    }
    let inputs = ProductionInputs {
        source: snapshot.to_string(),
        forgejo_source: sys::join(&[&request.out, "work/forgejo-ext"]),
        forgejo_revision: request.forgejo_revision.clone(),
        native: sys::join(&[snapshot, ".artifacts/native", &request.arch]),
        out: artifacts,
        arch: request.arch.clone(),
        revision: revision.to_string(),
        live_inputs: request.live_inputs.clone(),
    };
    let mut production = make_production(runner.clone(), inputs);
    // Borrow progress phases through a shared handle: the Go Production.Next
    // is progress.Next. Phase callbacks below call progress directly.
    let (context_dir, base, media_tooling, mut assembler) = {
        // phase = progress.Phase
        let phase_result: Result<
            (
                String,
                prepare::Base,
                build_media::MediaTools,
                media::MediaLock,
            ),
            Error,
        > = (|| {
            let context_dir: String;
            let base: prepare::Base;
            (context_dir, base) = prepare_build_host_context(
                snapshot,
                &request.out,
                &request.arch,
                revision,
                &request.repository_prefix,
                &request.media_compression,
                &request.live_inputs,
                &mut *production,
            )?;
            progress.phase("P2 / Verify and install frozen dependencies")?;
            production.dependencies()?;
            let (tooling, lock) = build_media::prepare_build_media(&*production, request)?;
            Ok((context_dir, base, tooling, lock))
        })();
        match phase_result {
            Ok(parts) => parts,
            Err(e) => return finalize_build(progress, request::Result::default(), Err(e)),
        }
    };
    let snapshot = snapshot.to_string();
    let artifacts = sys::join(&[&request.out, "artifacts"]);
    // Execute phases P3+ with progress-driven phase labels.
    let outcome: Result<request::Result, Error> = (|| {
        progress.phase("P3 / Compile shipping programs")?;
        compile_shipping_tools(
            &*production,
            &snapshot,
            &context_dir,
            &artifacts,
            revision,
            &request.arch,
        )?;
        build_host_candidate(
            &snapshot,
            &context_dir,
            &artifacts,
            &request.arch,
            revision,
            &request.repository_prefix,
            &base,
            &*production,
            &mut |label| progress.phase(label),
        )?;
        build_media::finish_build_media(
            &*production,
            request,
            &media_tooling,
            &mut assembler,
            &mut |label| progress.phase(label),
        )?;
        record::record_build_result(&*production, request)
    })();
    match outcome {
        Ok(result) => finalize_build(progress, result, Ok(())),
        Err(e) => finalize_build(progress, request::Result::default(), Err(e)),
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn oracle_link_prepared_assets_runs_no_commands() {
        // Oracle: Go TestLinkPreparedAssetsRunsNoCommands.
        struct Stub {
            source: String,
            native: String,
        }
        impl Production for Stub {
            fn source(&self) -> &str {
                &self.source
            }
            fn forgejo_source(&self) -> &str {
                ""
            }
            fn forgejo_revision(&self) -> &str {
                ""
            }
            fn native(&self) -> &str {
                &self.native
            }
            fn out(&self) -> &str {
                ""
            }
            fn arch(&self) -> &str {
                "x86_64"
            }
            fn revision(&self) -> &str {
                ""
            }
            fn live_inputs(&self) -> &str {
                ""
            }
            fn execute(&self, _: &str, _: &str, _: &[String]) -> Result<(), Error> {
                panic!("command run")
            }
            fn capture(&self, _: &str, _: &str, _: &[String]) -> Result<String, Error> {
                panic!("command run")
            }
            fn next(&self, _: &str) -> Result<(), Error> {
                Ok(())
            }
            fn resolve_inputs(&mut self) -> Result<(), Error> {
                Ok(())
            }
            fn dependencies(&self) -> Result<(), Error> {
                Ok(())
            }
            fn compile(&self, _: &str, _: &str, _: &str) -> Result<(), Error> {
                Ok(())
            }
            fn compile_rust(&self, _: &str, _: &str, _: &str) -> Result<(), Error> {
                Ok(())
            }
            fn stage_fork_binary(&self, _: &str) -> Result<(), Error> {
                Ok(())
            }
            fn assets(&self, _: &str, _: &str) -> Result<(), Error> {
                Ok(())
            }
            fn images(&self, _: &str) -> Result<HashMap<String, model::ProducedImage>, Error> {
                Ok(Default::default())
            }
            fn inspect_oci(&self, _: &str, _: &str, _: &str) -> Result<model::Image, Error> {
                Ok(Default::default())
            }
            fn verify_content(
                &self,
                _: &model::Payload,
                _: &str,
            ) -> Result<(HashMap<String, String>, u64), Error> {
                Ok(Default::default())
            }
            fn resolve_core_os(&self) -> Result<model::ResolvedCoreOS, Error> {
                Ok(Default::default())
            }
            fn read_live_inputs(&self, _: &str) -> Result<model::LiveInputs, Error> {
                Ok(Default::default())
            }
            fn check_native(&self, _: &str) -> Result<(), Error> {
                Ok(())
            }
            fn sign_media(
                &self,
                _: &model::Trust,
                _: &model::Permit,
                _: &str,
                _: &str,
                _: &str,
                _: &model::SecretFiles,
                _: &str,
            ) -> Result<(), Error> {
                Ok(())
            }
            fn verify_copy(
                &self,
                _: &model::Trust,
                _: &str,
                _: &str,
                _: &str,
                _: &str,
            ) -> Result<(), Error> {
                Ok(())
            }
            fn write_document(&self, _: &str, _: &soda_json::JsonValue) -> Result<String, Error> {
                Ok(String::new())
            }
        }
        let dir = std::env::temp_dir().join(format!("sri-lpa-{}", std::process::id()));
        let _ = fs::remove_dir_all(&dir);
        let stub = Stub {
            source: dir.join("src").to_str().unwrap().to_string(),
            native: dir.join("native").to_str().unwrap().to_string(),
        };
        link_prepared_assets(&stub).unwrap();
        assert!(fs::symlink_metadata(dir.join("src/.artifacts/forgejo-js"))
            .unwrap()
            .file_type()
            .is_symlink());
        let _ = fs::remove_dir_all(&dir);
    }
}

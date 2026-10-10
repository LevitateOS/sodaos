//! `build.go`: single candidate execution owner + tool environment.
//!
//! Qualification and protected delivery consume its unchanged bytes; the
//! result is not a qualified release.

use std::rc::Rc;

use crate::build_compile::compile_shipping_tools;
use crate::build_media;
use crate::build_runner::{Cancel, LogCloser, Runner};
use crate::build_source::admit_build_inputs;
use crate::error::Error;
use crate::foreign::{Production, Progress};
use crate::media;
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
    let runner = Runner::new(Rc::new(cancel.clone()));
    let result = run_build(request, progress, &runner, make_production, &|path| {
        runner.open_log(path, request.wants_media())
    });
    if let Err(err) = &result {
        progress.note_reason(&recall::failure_reason(&runner.log.borrow(), err));
    }
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
            "-c".to_string(),
            "tar.umask=0022".to_string(),
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
            "--same-permissions".to_string(),
        ],
    )?;
    Ok(snapshot)
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::fs;
    use std::os::unix::fs::PermissionsExt;
    use std::process::Command;

    #[test]
    fn extract_build_snapshot_restores_archived_modes_under_restrictive_umask() {
        const CHILD: &str = "SODA_SNAPSHOT_MODE_TEST_CHILD";
        if std::env::var_os(CHILD).is_none() {
            let status = Command::new("sh")
                .arg("-c")
                .arg("umask 077; exec \"$@\"")
                .arg("sh")
                .arg(std::env::current_exe().unwrap())
                .arg("extract_build_snapshot_restores_archived_modes_under_restrictive_umask")
                .arg("--nocapture")
                .env(CHILD, "1")
                .status()
                .unwrap();
            assert!(status.success(), "isolated restrictive-umask test failed");
            return;
        }

        let attempt = std::time::SystemTime::now()
            .duration_since(std::time::UNIX_EPOCH)
            .unwrap()
            .as_nanos();
        let root = std::env::temp_dir().join(format!(
            "soda-build-snapshot-mode-{}-{attempt}",
            std::process::id()
        ));
        fs::create_dir(&root).unwrap();
        let source = root.join("source");
        let out = root.join("out");
        let archived_usr = source.join("system/project/rootfs/usr");
        fs::create_dir_all(&archived_usr).unwrap();
        fs::create_dir_all(out.join("work")).unwrap();
        fs::create_dir_all(out.join("artifacts")).unwrap();
        let helper = archived_usr.join("libexec/soda/helper");
        fs::create_dir_all(helper.parent().unwrap()).unwrap();
        fs::write(&helper, "#!/bin/sh\nexit 0\n").unwrap();
        let data = archived_usr.join("share/example.conf");
        fs::create_dir_all(data.parent().unwrap()).unwrap();
        fs::write(&data, "example=true\n").unwrap();
        for dir in [
            &source,
            &source.join("system"),
            &source.join("system/project"),
            &source.join("system/project/rootfs"),
            &archived_usr,
            &archived_usr.join("libexec"),
            &archived_usr.join("libexec/soda"),
            &archived_usr.join("share"),
        ] {
            fs::set_permissions(dir, fs::Permissions::from_mode(0o755)).unwrap();
        }
        fs::set_permissions(&helper, fs::Permissions::from_mode(0o755)).unwrap();
        fs::set_permissions(&data, fs::Permissions::from_mode(0o644)).unwrap();

        for args in [
            vec!["init", "-q"],
            vec![
                "-C",
                source.to_str().unwrap(),
                "config",
                "user.email",
                "fixture@example.invalid",
            ],
            vec![
                "-C",
                source.to_str().unwrap(),
                "config",
                "user.name",
                "fixture",
            ],
            vec!["-C", source.to_str().unwrap(), "add", "system"],
            vec!["-C", source.to_str().unwrap(), "commit", "-qm", "fixture"],
        ] {
            let status = Command::new("git")
                .args(args)
                .current_dir(&source)
                .status()
                .unwrap();
            assert!(status.success(), "git fixture command failed");
        }
        let revision = Command::new("git")
            .args(["-C", source.to_str().unwrap(), "rev-parse", "HEAD"])
            .output()
            .unwrap();
        assert!(revision.status.success());
        let revision = String::from_utf8(revision.stdout)
            .unwrap()
            .trim()
            .to_string();

        let runner = Runner::new(Rc::new(Cancel::new()));
        let snapshot = extract_build_snapshot(
            source.to_str().unwrap(),
            out.to_str().unwrap(),
            &revision,
            &runner,
        )
        .unwrap();
        let mode =
            |path: &std::path::Path| fs::metadata(path).unwrap().permissions().mode() & 0o777;
        assert_eq!(mode(std::path::Path::new(&snapshot)), 0o700);
        assert_eq!(
            mode(&std::path::Path::new(&snapshot).join("system/project/rootfs/usr")),
            0o755
        );
        assert_eq!(
            mode(
                &std::path::Path::new(&snapshot)
                    .join("system/project/rootfs/usr/libexec/soda/helper")
            ),
            0o755
        );
        assert_eq!(
            mode(
                &std::path::Path::new(&snapshot)
                    .join("system/project/rootfs/usr/share/example.conf")
            ),
            0o644
        );
        fs::remove_dir_all(root).unwrap();
    }
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
    let inventory = compile_shipping_tools(
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
        &inventory,
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
    // This helper runs two source commands before production is constructed.
    // Reuse the same runner so cancellation and log ownership stay shared.
    crate::forgejo::extract_forgejo_snapshot(runner, request)?;
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
    // The production factory has no progress hook. Phase callbacks below
    // report active build steps directly through the progress owner.
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
        let inventory = compile_shipping_tools(
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
            &inventory,
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

//! Package only the retained bundle of a completed native verification run.
use super::{
    Factory, FactoryBroker, FactoryError, FactoryTerminal, PrivateState, FACTORY_COMPLETED,
};
use crate::{
    application::{BuildArtifact, BuildRequest, ReleaseBinding, Runtime},
    preparation,
    project::Executor,
    sha256,
};
use serde::Deserialize;
use std::os::unix::fs::{DirBuilderExt, MetadataExt};
use std::path::Path;
use std::time::Instant;

#[derive(Deserialize)]
#[serde(deny_unknown_fields)]
struct ExportedBundle {
    sha256: String,
    bundle: String,
}

impl<E: Executor, T: FactoryTerminal, B: FactoryBroker> Factory<E, T, B> {
    pub fn application_build(
        &self,
        request: &BuildRequest,
        release: ReleaseBinding,
        deadline: Instant,
    ) -> Result<BuildArtifact, FactoryError> {
        request.validate().map_err(FactoryError::msg)?;
        release.validate().map_err(FactoryError::msg)?;
        let _lock = self.lock_run(&request.project, &request.run, deadline)?;
        let (receipt, exists) = self.load_receipt(&request.project, &request.run)?;
        if !exists {
            return Err(FactoryError::NotFound);
        }
        if receipt.phase != FACTORY_COMPLETED {
            return Err(FactoryError::msg(
                "application packaging requires a completed native run",
            ));
        }
        let checks = self
            .saved_verification_checks(&receipt)?
            .ok_or(FactoryError::NotFound)?;
        checks
            .validate_for(&request.checks())
            .map_err(FactoryError::msg)?;
        let digest = if request.environment == "preview" {
            &checks.application_preview_bundle_sha256
        } else {
            &checks.application_bundle_sha256
        };
        if checks.phase != "passed" || !preparation::valid_digest(digest) {
            return Err(FactoryError::msg(
                "verified run has no retained application bundle",
            ));
        }
        let runtime = Runtime {
            exec: &self.exec,
            release: Some(release),
        };
        let state = PrivateState::open(&self.state_dir).map_err(FactoryError::msg)?;
        let name = format!(
            "application-build-{}-{}-{}.json",
            request.project, request.run, request.environment
        );
        if let Some(saved) = state
            .read::<BuildArtifact>(&name)
            .map_err(FactoryError::msg)?
        {
            if saved.source_commit != checks.source_commit || &saved.bundle_file_sha256 != digest {
                return Err(FactoryError::Stale);
            }
            runtime
                .check_build_artifact(&saved, request, deadline)
                .map_err(FactoryError::msg)?;
            return Ok(saved);
        }
        let manifest = self.load_verification_manifest(&receipt.run)?;
        let binding = receipt.binding.as_ref().ok_or(FactoryError::Stale)?;
        let input=serde_json::to_vec(&serde_json::json!({"op":"verify_bundle","environment":request.environment,"id":manifest.preparation,"run_id":request.run,"invocation_id":request.incarnation,"manifest_digest":request.manifest_digest,"check_digest":manifest.check_digest,"source_commit":manifest.source_commit,"deadline":manifest.deadline})).map_err(|_|FactoryError::msg("application export encoding failed"))?;
        let raw = self
            .exec
            .run_bounded(
                &input,
                "podman",
                &[
                    "--remote=false",
                    "exec",
                    "--interactive",
                    &binding.project,
                    preparation::FACTORY_HELPER,
                ],
                deadline,
                45 * 1024 * 1024,
                4096,
            )
            .map_err(FactoryError::msg)?;
        // This is the explicitly bounded helper output, not a host request.
        // Canonical bundles exceed the request decoder's 1 MiB ceiling. The
        // derived DTO still rejects unknown and duplicate fields and trailing
        // JSON, while run_bounded above caps its base64 envelope at 45 MiB.
        let exported: ExportedBundle = serde_json::from_slice(&raw)
            .map_err(|_| FactoryError::msg("application bundle export is invalid"))?;
        let bytes = crate::ssh::b64_decode(&exported.bundle).map_err(FactoryError::msg)?;
        if bytes.is_empty()
            || bytes.len() > 32 * 1024 * 1024
            || &exported.sha256 != digest
            || sha256::hex_lower(&sha256::digest(&bytes)) != exported.sha256
        {
            return Err(FactoryError::msg(
                "application bundle export differs from native receipt",
            ));
        }
        let root = Path::new("/home/soda-application-builds");
        match std::fs::DirBuilder::new().mode(0o700).create(root) {
            Ok(()) => {}
            Err(e) if e.kind() == std::io::ErrorKind::AlreadyExists => {}
            Err(_) => return Err(FactoryError::msg("application build storage unavailable")),
        }
        let root = crate::application::home_path(root).map_err(FactoryError::msg)?;
        let meta = std::fs::symlink_metadata(&root)
            .map_err(|_| FactoryError::msg("application build storage unavailable"))?;
        if !meta.is_dir() || meta.uid() != unsafe { libc::geteuid() } || meta.mode() & 0o077 != 0 {
            return Err(FactoryError::msg(
                "application build storage is not private",
            ));
        }
        let artifact = runtime
            .package_application(
                request,
                &checks.source_commit,
                &bytes,
                &root.join(format!("{}-{}", request.run, request.environment)),
                deadline,
            )
            .map_err(FactoryError::msg)?;
        state.write(&name, &artifact).map_err(FactoryError::msg)?;
        Ok(artifact)
    }
}

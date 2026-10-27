//! Fixed packaging of a retained verified Canlang bundle. No project-provided
//! Containerfile or executable participates, and no application is started.
use super::{
    wire::{hex, Spec},
    Runtime,
};
use crate::{factory::FactoryVerificationChecks, preparation, project::Executor, sha256};
use serde::{Deserialize, Serialize};
use std::fs::{self, OpenOptions};
use std::io::{Read, Write};
use std::os::unix::fs::{DirBuilderExt, MetadataExt, OpenOptionsExt};
use std::path::Path;
use std::time::Instant;

#[derive(Clone, Debug, PartialEq, Eq, Serialize, Deserialize)]
#[serde(deny_unknown_fields)]
pub struct BuildRequest {
    pub project: String,
    pub run: String,
    pub incarnation: String,
    pub manifest_digest: String,
    pub environment: String,
}

impl BuildRequest {
    pub fn checks(&self) -> FactoryVerificationChecks {
        FactoryVerificationChecks {
            project: self.project.clone(),
            run: self.run.clone(),
            incarnation: self.incarnation.clone(),
            manifest_digest: self.manifest_digest.clone(),
        }
    }
    pub fn validate(&self) -> Result<(), String> {
        if !matches!(self.environment.as_str(), "company" | "preview") {
            return Err("choose company or preview application build".into());
        }
        self.checks().validate()
    }
}

#[derive(Clone, Debug, PartialEq, Eq, Serialize, Deserialize)]
#[serde(deny_unknown_fields)]
pub struct BuildArtifact {
    pub project: String,
    pub run: String,
    pub incarnation: String,
    pub manifest_digest: String,
    pub source_commit: String,
    pub environment: String,
    pub name: String,
    pub artifact_sha256: String,
    pub bundle_sha256: String,
    pub bundle_file_sha256: String,
    pub storage_sha256: String,
    pub tools_sha256: String,
    pub runtime_image: String,
    pub image: String,
    pub origin: String,
}

impl BuildArtifact {
    pub fn validate_for(&self, request: &BuildRequest) -> Result<(), String> {
        request.validate()?;
        if self.environment != request.environment
            || self.project != request.project
            || self.run != request.run
            || self.incarnation != request.incarnation
            || self.manifest_digest != request.manifest_digest
            || !preparation::valid_commit(&self.source_commit)
            || !hex(&self.bundle_file_sha256, 64)
            || !hex(&self.storage_sha256, 64)
        {
            return Err("application build differs from native verification receipt".into());
        }
        self.references().validate()
    }
    pub(super) fn references(&self) -> Spec {
        Spec {
            id: format!("a{}", "0".repeat(24)),
            revision: 1,
            name: self.name.clone(),
            artifact_sha256: self.artifact_sha256.clone(),
            bundle_sha256: self.bundle_sha256.clone(),
            tools_sha256: self.tools_sha256.clone(),
            runtime_image: self.runtime_image.clone(),
            image: self.image.clone(),
            data: format!("a{}", "1".repeat(24)),
            owner_team: "00000000-0000-0000-0000-000000000000".into(),
            origin: self.origin.clone(),
            port: 20000,
        }
    }
}

fn exact_file(path: &Path, bytes: &[u8]) -> Result<(), String> {
    let file = OpenOptions::new()
        .write(true)
        .create_new(true)
        .mode(0o600)
        .custom_flags(libc::O_NOFOLLOW)
        .open(path);
    match file {
        Ok(mut file) => file
            .write_all(bytes)
            .and_then(|_| file.sync_all())
            .map_err(|_| "application build staging failed".into()),
        Err(error) if error.kind() == std::io::ErrorKind::AlreadyExists => {
            let file = OpenOptions::new()
                .read(true)
                .custom_flags(libc::O_NOFOLLOW)
                .open(path)
                .map_err(|_| "application build staging changed")?;
            let meta = file
                .metadata()
                .map_err(|_| "application build staging changed")?;
            if !meta.is_file()
                || meta.uid() != unsafe { libc::geteuid() }
                || meta.mode() & 0o077 != 0
                || meta.len() != bytes.len() as u64
            {
                return Err("application build staging changed".into());
            }
            let mut previous = Vec::new();
            file.take(bytes.len() as u64 + 1)
                .read_to_end(&mut previous)
                .map_err(|_| "application build staging changed")?;
            if previous != bytes {
                return Err("application build staging changed".into());
            }
            Ok(())
        }
        Err(_) => Err("application build staging failed".into()),
    }
}

impl<E: Executor> Runtime<E> {
    pub fn package_application(
        &self,
        request: &BuildRequest,
        source: &str,
        bundle: &[u8],
        directory: &Path,
        deadline: Instant,
    ) -> Result<BuildArtifact, String> {
        request.validate()?;
        let selected = self
            .release
            .as_ref()
            .ok_or("selected Canlang runtime unavailable")?;
        selected.validate()?;
        if !preparation::valid_commit(source)
            || bundle.is_empty()
            || bundle.len() > 32 * 1024 * 1024
        {
            return Err("application bundle is outside packaging bounds".into());
        }
        let value: serde_json::Value =
            serde_json::from_slice(bundle).map_err(|_| "application bundle metadata is invalid")?;
        let field = |path: &str| -> Result<String, String> {
            value
                .pointer(path)
                .and_then(serde_json::Value::as_str)
                .filter(|v| v.len() <= 256)
                .map(str::to_string)
                .ok_or_else(|| "application bundle metadata is missing".into())
        };
        let mut artifact = BuildArtifact {
            project: request.project.clone(),
            run: request.run.clone(),
            incarnation: request.incarnation.clone(),
            manifest_digest: request.manifest_digest.clone(),
            source_commit: source.into(),
            environment: request.environment.clone(),
            name: field("/descriptor/identity/appName")?,
            artifact_sha256: field("/descriptor/identity/artifactDigest")?,
            bundle_sha256: field("/bundle/mixedSha256")?,
            bundle_file_sha256: sha256::hex_lower(&sha256::digest(bundle)),
            storage_sha256: field("/storageFingerprint")?,
            tools_sha256: selected.tools_sha256.clone(),
            runtime_image: selected.image.clone(),
            image: selected.image.clone(),
            origin: field("/environment/vars/CAN_AUTH_ORIGIN")?,
        };
        artifact.validate_for(request)?;
        match fs::DirBuilder::new().mode(0o700).create(directory) {
            Ok(()) => {}
            Err(e) if e.kind() == std::io::ErrorKind::AlreadyExists => {}
            Err(_) => return Err("application build directory unavailable".into()),
        }
        let meta = fs::symlink_metadata(directory)
            .map_err(|_| "application build directory unavailable")?;
        if !meta.is_dir()
            || meta.uid() != unsafe { libc::geteuid() }
            || meta.mode() & 0o077 != 0
            || fs::canonicalize(directory).map_err(|_| "application build directory unavailable")?
                != directory
        {
            return Err("application build directory is not private".into());
        }
        exact_file(&directory.join("bundle.json"), bundle)?;
        let labels = [
            ("io.canlang.app", &artifact.name),
            ("io.canlang.artifact", &artifact.artifact_sha256),
            ("io.canlang.bundle", &artifact.bundle_sha256),
            ("io.canlang.storage", &artifact.storage_sha256),
            ("io.canlang.tools", &artifact.tools_sha256),
            ("io.soda.canlang.runtime", &artifact.runtime_image),
            ("io.canlang.origin", &artifact.origin),
        ];
        let mut recipe = format!("FROM {}\n", selected.image);
        for (key, value) in labels {
            recipe.push_str(&format!(
                "LABEL {key}={}\n",
                serde_json::to_string(value).map_err(|_| "application label encoding failed")?
            ));
        }
        recipe.push_str("COPY --chmod=0444 bundle.json /opt/application/bundle.json\n");
        exact_file(&directory.join("Containerfile"), recipe.as_bytes())?;
        let tag = format!(
            "localhost/soda-factory-application:{}-{}",
            request.run, request.environment
        );
        let inspect = || {
            self.command(
                "podman",
                &[
                    "image".into(),
                    "inspect".into(),
                    "--format={{.ID}}".into(),
                    tag.clone(),
                ],
                deadline,
            )
        };
        let image = match inspect() {
            Ok(bytes) => bytes,
            Err(_) => {
                self.command(
                    "podman",
                    &[
                        "build".into(),
                        "--network=none".into(),
                        "--pull=never".into(),
                        "--tag".into(),
                        tag.clone(),
                        "--file".into(),
                        directory
                            .join("Containerfile")
                            .to_string_lossy()
                            .into_owned(),
                        directory.to_string_lossy().into_owned(),
                    ],
                    deadline,
                )?;
                inspect()?
            }
        };
        artifact.image = String::from_utf8(image)
            .map_err(|_| "application image identity unavailable")?
            .trim()
            .into();
        if !artifact.image.starts_with("sha256:") {
            artifact.image = format!("sha256:{}", artifact.image);
        }
        artifact.validate_for(request)?;
        if self.bundle_file_sha256(&artifact.references(), deadline)? != artifact.bundle_file_sha256
        {
            return Err("packaged application bytes differ from native check".into());
        }
        Ok(artifact)
    }
    pub(crate) fn check_build_artifact(
        &self,
        artifact: &BuildArtifact,
        request: &BuildRequest,
        deadline: Instant,
    ) -> Result<(), String> {
        artifact.validate_for(request)?;
        let selected = self
            .release
            .as_ref()
            .ok_or("selected Canlang runtime unavailable")?;
        if artifact.runtime_image != selected.image
            || artifact.tools_sha256 != selected.tools_sha256
            || self.bundle_file_sha256(&artifact.references(), deadline)?
                != artifact.bundle_file_sha256
        {
            return Err("native application build identity changed".into());
        }
        Ok(())
    }
}

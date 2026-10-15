//! Production application images: the Podman build and export recipe,
//! the role sequence, and the lexical relative paths those recipes use.

use crate::files::{is_digest, oci_architecture};
use crate::production::{ProducedImage, Production};
use crate::production_inputs::ResolvedInput;
use crate::{io_error, Error};
use soda_build_tools::reader::settings::{recipe_base, unit_image};
use std::collections::HashMap;
use std::path::{Path, PathBuf};

/// Image-pipeline hook shapes shared by the export stages.
type PullFn<'a> = &'a dyn Fn(&str, &str) -> Result<(String, String), Error>;
type BuildFn<'a> = &'a dyn Fn(&str, &str, &str, &str, &[String]) -> Result<String, Error>;
type ExportFn<'a> = &'a mut dyn FnMut(&str, &str, &str) -> Result<(), Error>;

/// Lexical relative path from `base` to `target`, Go `filepath.Rel`
/// flavored for the containment check (both absolute, slash-separated).
fn lexical_rel(base: &Path, target: &Path) -> String {
    let base_clean = crate::path_clean(&base.to_string_lossy());
    let target_clean = crate::path_clean(&target.to_string_lossy());
    if base_clean == target_clean {
        return ".".to_string();
    }
    if base_clean == "/" {
        return target_clean.trim_start_matches('/').to_string();
    }
    if let Some(rest) = target_clean.strip_prefix(&format!("{base_clean}/")) {
        return rest.to_string();
    }
    // Outside: Go would produce a "../.." path; the caller only needs the
    // ".." prefix signal, so synthesize it cheaply.
    format!("..{target_clean}")
}

impl Production {
    pub(crate) fn build_image(
        &self,
        name: &str,
        dir: &str,
        file: &str,
        pinned: &str,
        args: &[String],
    ) -> Result<String, Error> {
        let platform = oci_architecture(&self.arch).unwrap_or("amd64").to_string();
        let iid = PathBuf::from(&self.out).join(format!("{name}.iid"));
        let base_digest = pinned.split_once('@').map(|(_, d)| d).unwrap_or(pinned);
        let mut cmd = vec![
            "--remote=false".to_string(),
            "build".to_string(),
            "--pull=never".to_string(),
            "--rm=true".to_string(),
            "--force-rm=false".to_string(),
            format!("--platform=linux/{platform}"),
            format!("--build-arg=BASE_IMAGE={pinned}"),
            format!(
                "--label=org.opencontainers.image.revision={}",
                self.revision
            ),
            "--label=org.opencontainers.image.source=https://github.com/LevitateOS/sodaos"
                .to_string(),
            format!("--label=org.opencontainers.image.base.name={pinned}"),
            format!("--label=org.opencontainers.image.base.digest={base_digest}"),
            "--iidfile".to_string(),
            iid.to_string_lossy().into_owned(),
            "--file".to_string(),
            file.to_string(),
        ];
        cmd.extend(args.iter().cloned());
        cmd.push(".".to_string());
        self.call_execute(dir, "podman", &cmd)?;
        let body = std::fs::read(&iid).map_err(|e| io_error("open", &iid, e))?;
        let id = String::from_utf8_lossy(&body).trim().to_string();
        let hex = id.strip_prefix("sha256:").unwrap_or("");
        if !id.starts_with("sha256:") || !is_digest(hex) {
            return Err(Error::msg("invalid built image ID"));
        }
        Ok(id)
    }

    pub(crate) fn export_image_archive(
        &self,
        archives: &Path,
        name: &str,
        id: &str,
        revision: &str,
    ) -> Result<ProducedImage, Error> {
        let file = archives.join(format!("{name}.oci"));
        // Newly built layers leave in registry-compatible compression before
        // the final manifest digest is sealed: Skopeo's containers-storage
        // to oci-archive transport at gzip level 1, reusing already
        // compressed upstream layers. The sealed archive digest, not the
        // pre-export store manifest, is the runtime and public digest.
        self.call_execute(
            &self.source,
            "skopeo",
            &[
                "copy".to_string(),
                "--format".to_string(),
                "oci".to_string(),
                "--dest-compress-format".to_string(),
                "gzip".to_string(),
                "--dest-compress-level".to_string(),
                "1".to_string(),
                format!("containers-storage:{id}"),
                format!("oci-archive:{}", file.to_string_lossy()),
            ],
        )?;
        let image =
            soda_release_deliver::oci::inspect_oci(&file.to_string_lossy(), &self.arch, revision)
                .map_err(|error| Error::msg(error.to_string()))?;
        if image.config != id {
            return Err(Error::msg("app archive/config mismatch"));
        }
        let hash = crate::files::hash_file(&file)?;
        Ok(ProducedImage {
            image,
            archive_sha256: hash,
        })
    }

    fn resolve_rocky_base(&self, pull: PullFn) -> Result<(String, String), Error> {
        let rocky = recipe_base(&PathBuf::from(&self.source).join("system/project/Containerfile"))
            .map_err(|e| Error::msg(e.to_string()))?;
        let dashboard = recipe_base(
            &PathBuf::from(&self.source).join("system/containers/dashboard/Containerfile"),
        )
        .map_err(|e| Error::msg(e.to_string()))?;
        let canlang = recipe_base(
            &PathBuf::from(&self.source).join("system/containers/canlang/Containerfile"),
        )
        .map_err(|e| Error::msg(e.to_string()))?;
        if rocky != dashboard || rocky != canlang {
            return Err(Error::msg("Rocky component base owners disagree"));
        }
        let (_, pinned) = pull(&rocky, "base")?;
        let native_rel = lexical_rel(Path::new(&self.source), Path::new(&self.native));
        if native_rel.starts_with("..") {
            return Err(Error::msg("native assets must be inside source context"));
        }
        Ok((pinned, native_rel))
    }

    fn export_app_images(
        &self,
        native_rel: &str,
        rocky: &str,
        build: BuildFn,
        export: ExportFn,
    ) -> Result<(), Error> {
        for name in ["dashboard", "project-os", "canlang-runtime"] {
            let file = match name {
                "project-os" => "system/project/Containerfile",
                "canlang-runtime" => "system/containers/canlang/Containerfile",
                _ => "system/containers/dashboard/Containerfile",
            };
            let id = build(
                name,
                &self.source,
                file,
                rocky,
                &[format!(
                    "--build-arg=ARTIFACT_DIR={}",
                    native_rel.replace('\\', "/")
                )],
            )?;
            export(name, &id, &self.revision)?;
        }
        Ok(())
    }

    fn export_forgejo_image(
        &self,
        forgejo_context: &str,
        pull: PullFn,
        build: BuildFn,
        export: ExportFn,
    ) -> Result<(), Error> {
        let forgejo =
            unit_image(&PathBuf::from(&self.source).join("system/host/services/forgejo.container"))
                .map_err(|e| Error::msg(e.to_string()))?;
        let (_, pinned) = pull(&forgejo, "forgejo-base")?;
        let id = build("forgejo", forgejo_context, "Containerfile", &pinned, &[])?;
        export("forgejo", &id, &self.revision)
    }

    /// Upstream application images are bound, not built and not exported:
    /// the frozen pull already resolved the exact platform manifest and
    /// config, the runtime verifies that binding on every acquisition, and
    /// no duplicate application archive is produced for them.
    fn bind_upstream_image(
        &self,
        name: &str,
        unit: &str,
        pull: PullFn,
    ) -> Result<ProducedImage, Error> {
        let reference = unit_image(
            &PathBuf::from(&self.source)
                .join("system/host/services")
                .join(unit),
        )
        .map_err(|e| Error::msg(e.to_string()))?;
        let (config, pinned) = pull(&reference, name)?;
        let manifest = pinned
            .split('@')
            .nth(1)
            .filter(|digest| digest.starts_with("sha256:"))
            .unwrap_or("")
            .to_string();
        if manifest.is_empty() {
            return Err(Error::msg("upstream binding is not digest-pinned"));
        }
        Ok(ProducedImage {
            image: soda_release_deliver::buildx::Image {
                manifest,
                config,
                ..Default::default()
            },
            archive_sha256: String::new(),
        })
    }

    pub(crate) fn export_images(
        &self,
        forgejo_context: &str,
        archives: &Path,
        inputs: &[ResolvedInput],
    ) -> Result<HashMap<String, ProducedImage>, Error> {
        let mut result: HashMap<String, ProducedImage> = HashMap::new();
        let pull = |reference: &str, iid_name: &str| -> Result<(String, String), Error> {
            self.pull_frozen_image(inputs, reference, iid_name)
        };
        let build = |name: &str,
                     dir: &str,
                     file: &str,
                     pinned: &str,
                     args: &[String]|
         -> Result<String, Error> {
            self.build_image(name, dir, file, pinned, args)
        };
        // The export closure borrows the result map; scope each step so a
        // later step can read the accumulated image identities.
        macro_rules! exporting {
            ($body:expr) => {{
                let mut export = |name: &str, id: &str, revision: &str| -> Result<(), Error> {
                    let produced = self.export_image_archive(archives, name, id, revision)?;
                    result.insert(name.to_string(), produced);
                    Ok(())
                };
                $body(&mut export)
            }};
        }
        let (rocky, native_rel) = self.resolve_rocky_base(&pull)?;
        exporting!(|export: ExportFn| {
            self.export_app_images(&native_rel, &rocky, &build, export)?;
            self.export_forgejo_image(forgejo_context, &pull, &build, export)
        })?;
        // Upstream roles join the same identity map without a local archive:
        // their bindings carry no archive checksum.
        for (name, unit) in [
            ("proxy", "soda-proxy.container"),
            ("postgres", "soda-postgres.container"),
        ] {
            let bound = self.bind_upstream_image(name, unit, &pull)?;
            result.insert(name.to_string(), bound);
        }
        Ok(result)
    }
}

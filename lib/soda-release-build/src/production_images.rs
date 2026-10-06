//! Production application images: the Podman build and export recipe,
//! the role sequence, and the lexical relative paths those recipes use.

use crate::files::{is_digest, oci_architecture};
use crate::oci::inspect_oci;
use crate::production::{ProducedImage, Production};
use crate::production_inputs::ResolvedInput;
use crate::{io_error, Error};
use soda_build_tools::reader::settings::{recipe_base, unit_image};
use std::collections::HashMap;
use std::path::{Path, PathBuf};

/// Image-pipeline hook shapes shared by the export stages.
type PullFn<'a> = &'a dyn Fn(&str, &str, &str) -> Result<(String, String), Error>;
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
    fn build_image(
        &self,
        name: &str,
        dir: &str,
        file: &str,
        pinned: &str,
        args: &[String],
    ) -> Result<String, Error> {
        let platform = oci_architecture(&self.arch).unwrap_or("amd64").to_string();
        self.step(&format!("Build image: {name}"))?;
        let iid = PathBuf::from(&self.out).join(format!("{name}.iid"));
        let base_digest = pinned.split_once('@').map(|(_, d)| d).unwrap_or(pinned);
        let mut cmd = vec![
            "--remote=false".to_string(),
            "build".to_string(),
            "--pull=never".to_string(),
            "--rm=false".to_string(),
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

    fn export_image_archive(
        &self,
        archives: &Path,
        name: &str,
        id: &str,
        revision: &str,
    ) -> Result<ProducedImage, Error> {
        self.step(&format!("Export and verify image: {name}"))?;
        let file = archives.join(format!("{name}.oci"));
        self.call_execute(
            &self.source,
            "podman",
            &[
                "--remote=false".to_string(),
                "save".to_string(),
                "--format=oci-archive".to_string(),
                "--output".to_string(),
                file.to_string_lossy().into_owned(),
                id.to_string(),
            ],
        )?;
        let image = inspect_oci(&file, &self.arch, revision)?;
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
        if rocky != dashboard {
            return Err(Error::msg("dashboard and Project OS base owners disagree"));
        }
        let (_, pinned) = pull("Rocky base", &rocky, "base")?;
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
        for name in ["dashboard", "project-os"] {
            let file = if name == "project-os" {
                "system/project/Containerfile"
            } else {
                "system/containers/dashboard/Containerfile"
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
        let (_, pinned) = pull("Forgejo", &forgejo, "forgejo-base")?;
        let id = build("forgejo", forgejo_context, "Containerfile", &pinned, &[])?;
        export("forgejo", &id, &self.revision)
    }

    fn export_proxy_image(&self, pull: PullFn, export: ExportFn) -> Result<(), Error> {
        let proxy = unit_image(
            &PathBuf::from(&self.source).join("system/host/services/soda-proxy.container"),
        )
        .map_err(|e| Error::msg(e.to_string()))?;
        let (id, _) = pull("Proxy", &proxy, "proxy")?;
        export("proxy", &id, "")
    }

    fn export_tailnet_image(
        &self,
        platform: &str,
        pull: PullFn,
        build: BuildFn,
        export: ExportFn,
    ) -> Result<(), Error> {
        let tail = self.live_tailnet_inputs()?;
        let (_, pinned) = pull("Tailnet base", &tail.base, "tailnet-base")?;
        let id = build(
            "tailnet",
            &self.source,
            "system/containers/tailnet/Containerfile",
            &pinned,
            &[
                format!("--build-arg=TAILSCALE_VERSION={}", tail.version),
                format!("--build-arg=TARGETARCH={platform}"),
                format!("--build-arg=ARCHIVE_SHA256={}", tail.sha256),
            ],
        )?;
        export("tailnet", &id, &self.revision)
    }

    fn export_extension_image(
        &self,
        forgejo_config: &str,
        build: BuildFn,
        export: ExportFn,
    ) -> Result<(), Error> {
        let id = build(
            "extension",
            &PathBuf::from(&self.out)
                .join("extension-context")
                .to_string_lossy(),
            "Containerfile",
            forgejo_config,
            &[],
        )?;
        export("extension", &id, &self.revision)
    }

    pub(crate) fn export_images(
        &self,
        forgejo_context: &str,
        archives: &Path,
        inputs: &[ResolvedInput],
    ) -> Result<HashMap<String, ProducedImage>, Error> {
        let platform = oci_architecture(&self.arch).unwrap_or("amd64").to_string();
        let mut result: HashMap<String, ProducedImage> = HashMap::new();
        let pull =
            |label: &str, reference: &str, iid_name: &str| -> Result<(String, String), Error> {
                self.pull_frozen_image(inputs, label, reference, iid_name)
            };
        let build = |name: &str,
                     dir: &str,
                     file: &str,
                     pinned: &str,
                     args: &[String]|
         -> Result<String, Error> {
            self.build_image(name, dir, file, pinned, args)
        };
        // The export closure borrows the result map; scope it so the
        // extension step can read the accumulated forgejo config.
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
        let forgejo_config = result
            .get("forgejo")
            .map(|i| i.image.config.clone())
            .unwrap_or_default();
        exporting!(|export: ExportFn| {
            self.export_extension_image(&forgejo_config, &build, export)
        })?;
        exporting!(|export: ExportFn| {
            self.export_proxy_image(&pull, export)?;
            self.export_tailnet_image(&platform, &pull, &build, export)
        })?;
        Ok(result)
    }
}

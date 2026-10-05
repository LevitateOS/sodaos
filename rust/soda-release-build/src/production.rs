//! Production build pipeline (`production.go`): the single image layout's
//! concrete steps. Containerfiles, locks, and the Rust stage renderer own
//! content; this code never signs, publishes, or installs anything.

use crate::coreos_stream::{read_live_inputs, TailnetInputs};
use crate::files::{is_digest, is_revision, oci_architecture, write_new};
use crate::json_go::{marshal_indent, Emit};
use crate::oci::{inspect_oci, Image};
use crate::{io_error, Error};
use soda_build_tools::reader::settings::{recipe_base, unit_image};
use soda_json::JsonValue;
use std::collections::HashMap;
use std::path::{Path, PathBuf};

/// Command execution and capture hooks, like Go's `BuildExec`/`BuildCapture`.
pub type BuildExec = Box<dyn Fn(&str, &str, &[String]) -> Result<(), Error> + Send + Sync>;
pub type BuildCapture = Box<dyn Fn(&str, &str, &[String]) -> Result<String, Error> + Send + Sync>;
pub type NextFn = Box<dyn Fn(&str) -> Result<(), Error> + Send + Sync>;

/// Image-pipeline hook shapes shared by the export stages.
type PullFn<'a> = &'a dyn Fn(&str, &str, &str) -> Result<(String, String), Error>;
type BuildFn<'a> = &'a dyn Fn(&str, &str, &str, &str, &[String]) -> Result<String, Error>;
type ExportFn<'a> = &'a mut dyn FnMut(&str, &str, &str) -> Result<(), Error>;

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
    pub next: Option<NextFn>,
    inputs: Vec<ResolvedInput>,
}

impl Production {
    pub fn step(&self, label: &str) -> Result<(), Error> {
        match &self.next {
            Some(next) => next(label),
            None => Ok(()),
        }
    }

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

    fn validate(&self) -> Result<(), Error> {
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

    /// The sole Rust command recipe for ported runtime programs. Builds
    /// resolve dependencies from the network; the release binary is copied
    /// to `dest`.
    pub fn compile_rust(&self, crate_name: &str, bin: &str, dest: &str) -> Result<(), Error> {
        self.validate()?;
        if crate_name.is_empty() || bin.is_empty() || crate_name.contains('/') || bin.contains('/')
        {
            return Err(Error::msg("explicit Rust crate and binary required"));
        }
        self.step(&format!("Compile {bin}"))?;
        let manifest = PathBuf::from(&self.source).join("Cargo.toml");
        self.call_execute(
            &self.source,
            "cargo",
            &[
                "build".to_string(),
                "--release".to_string(),
                "--locked".to_string(),
                "--manifest-path".to_string(),
                manifest.to_string_lossy().into_owned(),
                "-p".to_string(),
                crate_name.to_string(),
            ],
        )?;
        let raw = std::fs::read(PathBuf::from(&self.source).join("target/release").join(bin))
            .map_err(|e| io_error("open", Path::new(bin), e))?;
        let dest_path = PathBuf::from(dest);
        std::fs::write(&dest_path, &raw).map_err(|e| io_error("open", &dest_path, e))?;
        crate::files::chmod(&dest_path, 0o755)?;
        crate::elf::inspect_elf(&dest_path, &self.arch)
    }

    /// The sole Go command recipe for runtime programs and support tools.
    /// Binaries use archive-safe VCS mode; the single image layout owns
    /// every install path.
    pub fn compile(&self, name: &str, pkg: &str, dest: &str) -> Result<(), Error> {
        self.validate()?;
        self.step(&format!("Compile {name}"))?;
        self.call_execute(
            &self.source,
            "go",
            &[
                "build".to_string(),
                "-mod=readonly".to_string(),
                "-trimpath".to_string(),
                "-buildvcs=false".to_string(),
                "-o".to_string(),
                dest.to_string(),
                pkg.to_string(),
            ],
        )?;
        let dest_path = PathBuf::from(dest);
        crate::files::chmod(&dest_path, 0o755)?;
        crate::elf::inspect_elf(&dest_path, &self.arch)
    }

    /// Runs exactly once before image production.
    pub fn assets(&self, host_context: &str, forgejo_context: &str) -> Result<(), Error> {
        self.validate()?;
        if !Path::new(host_context).is_absolute() || !Path::new(forgejo_context).is_absolute() {
            return Err(Error::msg("explicit asset destinations required"));
        }
        let stage: Vec<String> = [
            "cargo",
            "run",
            "--release",
            "--locked",
            "-p",
            "soda-stage-render",
            "--bin",
            "soda-stage",
            "--",
            "--arch",
            &self.arch,
            "--host-context",
            host_context,
            "--forgejo-context",
            forgejo_context,
        ]
        .iter()
        .map(ToString::to_string)
        .collect();
        self.asset_steps(&stage)
    }

    fn require_pinned_bun(&self) -> Result<(), Error> {
        // The workspace has unrelated fields; read the pin from its owning manifest.
        let data = std::fs::read(PathBuf::from(&self.source).join("package.json"))
            .map_err(|e| io_error("open", Path::new("package.json"), e))?;
        let text = String::from_utf8_lossy(&data);
        let value =
            JsonValue::parse(&text).map_err(|_| Error::msg("invalid workspace manifest"))?;
        let fields = crate::json_go::Fields::of(&value)
            .ok_or_else(|| Error::msg("invalid workspace manifest"))?;
        let pinned = fields
            .string("packageManager")
            .map_err(|_| Error::msg("invalid workspace manifest"))?;
        let bun = self.call_capture(&self.source, "bun", &["--version".to_string()])?;
        if format!("bun@{bun}") != pinned {
            return Err(Error::msg("workspace-pinned Bun required"));
        }
        Ok(())
    }

    /// Runs before any production compilation.
    pub fn dependencies(&self) -> Result<(), Error> {
        self.validate()?;
        self.step("Check frontend toolchain")?;
        self.require_pinned_bun()?;
        self.step("Verify Go dependencies")?;
        self.call_execute(
            &self.source,
            "go",
            &["mod".to_string(), "verify".to_string()],
        )?;
        self.step("Install frontend dependencies")?;
        self.call_execute(
            &self.source,
            "bun",
            &["install".to_string(), "--frozen-lockfile".to_string()],
        )
    }

    fn asset_steps(&self, stage: &[String]) -> Result<(), Error> {
        let bindir = PathBuf::from(&self.native).join("project-tools/bin");
        std::fs::create_dir_all(&bindir).map_err(|e| io_error("mkdir", &bindir, e))?;
        crate::files::chmod(&bindir, 0o755)?;
        self.compile_rust(
            "soda-muse",
            "soda-muse",
            &bindir.join("muse").to_string_lossy(),
        )?;
        self.compile_rust(
            "soda-identity-compose",
            "soda-identity-compose",
            &bindir.join("soda-identity-compose").to_string_lossy(),
        )?;
        self.compile_rust(
            "soda-project-terminal",
            "project-terminal",
            &bindir.join("project-terminal").to_string_lossy(),
        )?;
        self.compile_rust(
            "soda-project-account",
            "project-account",
            &bindir.join("project-account").to_string_lossy(),
        )?;
        self.compile_rust(
            "soda-project-factory-roles",
            "project-factory-roles",
            &bindir.join("project-factory-roles").to_string_lossy(),
        )?;
        let native = self.native.clone();
        let steps: Vec<(&str, Vec<String>)> = vec![
            (
                "Build frontend assets",
                vec![
                    "bun".to_string(),
                    "scripts/build-forgejo.ts".to_string(),
                    "--out".to_string(),
                    format!("{native}/forgejo-js"),
                ],
            ),
            (
                "Fetch terminal assets",
                vec![
                    "cargo".to_string(),
                    "run".to_string(),
                    "--release".to_string(),
                    "--locked".to_string(),
                    "-p".to_string(),
                    "soda-asset-fetchers".to_string(),
                    "--bin".to_string(),
                    "soda-fetch-terminal".to_string(),
                    "--".to_string(),
                    "--out".to_string(),
                    format!("{native}/terminal-assets"),
                ],
            ),
            (
                "Build Soda extension browser assets",
                vec![
                    "bun".to_string(),
                    "scripts/build-soda-extension.ts".to_string(),
                    "--out".to_string(),
                    format!("{native}/soda-extension-assets"),
                    "--terminal-assets".to_string(),
                    format!("{native}/terminal-assets"),
                ],
            ),
            (
                "Prepare Forgejo translations",
                vec![
                    "cargo".to_string(),
                    "run".to_string(),
                    "--release".to_string(),
                    "--locked".to_string(),
                    "-p".to_string(),
                    "soda-forgejo-locales".to_string(),
                    "--bin".to_string(),
                    "soda-forgejo-locales".to_string(),
                    "--".to_string(),
                    "--lock".to_string(),
                    "appliance/forgejo/locale.lock.json".to_string(),
                    "--out".to_string(),
                    format!("{native}/forgejo-locales/locale_en-US.ini"),
                ],
            ),
            (
                "Fetch upstream Muse binary",
                vec![
                    "cargo".to_string(),
                    "run".to_string(),
                    "--release".to_string(),
                    "--locked".to_string(),
                    "-p".to_string(),
                    "soda-asset-fetchers".to_string(),
                    "--bin".to_string(),
                    "soda-fetch-muse".to_string(),
                    "--".to_string(),
                    "--arch".to_string(),
                    self.arch.clone(),
                    "--out".to_string(),
                    format!("{native}/project-tools/bin/muse-native"),
                ],
            ),
            (
                "Fetch upstream Tea binary",
                vec![
                    "cargo".to_string(),
                    "run".to_string(),
                    "--release".to_string(),
                    "--locked".to_string(),
                    "-p".to_string(),
                    "soda-asset-fetchers".to_string(),
                    "--bin".to_string(),
                    "soda-fetch-tea".to_string(),
                    "--".to_string(),
                    "--arch".to_string(),
                    self.arch.clone(),
                    "--out".to_string(),
                    format!("{native}/project-tools"),
                ],
            ),
            ("Stage appliance files", stage.to_vec()),
        ];
        for (label, args) in &steps {
            self.step(label)?;
            self.call_execute(&self.source, &args[0], &args[1..])?;
        }
        Ok(())
    }

    /// Produces each selected app archive once.
    fn pull_frozen_image(
        &self,
        inputs: &[ResolvedInput],
        label: &str,
        reference: &str,
        iid_name: &str,
    ) -> Result<(String, String), Error> {
        self.step(&format!("Select frozen {label}"))?;
        for input in inputs {
            if input.requested != reference {
                continue;
            }
            let hex = input.config.strip_prefix("sha256:").unwrap_or("");
            if !is_digest(hex) || !input.reference.contains("@sha256:") {
                return Err(Error::msg("invalid frozen image input"));
            }
            if !iid_name.is_empty() {
                write_new(
                    &PathBuf::from(&self.out).join(format!("{iid_name}.iid")),
                    format!("{}\n", input.config).as_bytes(),
                    0o600,
                )?;
            }
            return Ok((input.config.clone(), input.reference.clone()));
        }
        Err(Error::msg("image source was not frozen before production"))
    }

    pub fn images(&self, forgejo_context: &str) -> Result<HashMap<String, ProducedImage>, Error> {
        self.validate()?;
        if forgejo_context.is_empty() {
            return Err(Error::msg("explicit Forgejo context required"));
        }
        let archives = PathBuf::from(&self.out).join("images");
        std::fs::create_dir(&archives).map_err(|e| io_error("mkdir", &archives, e))?;
        crate::files::chmod(&archives, 0o755)?;
        if self.inputs.len() != 4 {
            return Err(Error::msg("image inputs must be frozen before production"));
        }
        let inputs = self.inputs.clone();
        self.export_images(forgejo_context, &archives, &inputs)
    }

    fn admit_resolved_input_record(&self, inputs: &[ResolvedInput]) -> Result<(), Error> {
        let body = marshal_indent(&Emit::List(
            inputs.iter().map(ResolvedInput::emit).collect(),
        )) + "\n";
        // This private, fresh production attempt owns this growing record.
        let path = PathBuf::from(&self.out).join("app-inputs.json");
        std::fs::write(&path, body.as_bytes()).map_err(|e| io_error("open", &path, e))?;
        crate::files::chmod(&path, 0o600)
    }

    fn pull_resolved_input(
        &self,
        label: &str,
        reference: &str,
        iid_name: &str,
        platform: &str,
        inputs: &mut Vec<ResolvedInput>,
    ) -> Result<(String, String), Error> {
        self.step(&format!("Pull and resolve {label}"))?;
        let id = self.call_capture(
            &self.source,
            "podman",
            &[
                "--remote=false".to_string(),
                "pull".to_string(),
                "--quiet".to_string(),
                format!("--platform=linux/{platform}"),
                reference.to_string(),
            ],
        )?;
        let hex = id.strip_prefix("sha256:").unwrap_or(&id);
        if !is_digest(hex) {
            return Err(Error::msg("invalid pulled image ID"));
        }
        let id = format!("sha256:{hex}");
        let digest = self.call_capture(
            &self.source,
            "podman",
            &[
                "--remote=false".to_string(),
                "image".to_string(),
                "inspect".to_string(),
                "--format".to_string(),
                "{{.Digest}}".to_string(),
                id.clone(),
            ],
        )?;
        let digest_hex = digest.strip_prefix("sha256:").unwrap_or("");
        if !digest.starts_with("sha256:") || !is_digest(digest_hex) {
            return Err(Error::msg("invalid registry digest"));
        }
        let pinned = format!("{}@{digest}", parse_image_repo(reference));
        if !iid_name.is_empty() {
            write_new(
                &PathBuf::from(&self.out).join(format!("{iid_name}.iid")),
                format!("{id}\n").as_bytes(),
                0o600,
            )?;
        }
        inputs.push(ResolvedInput {
            requested: reference.to_string(),
            reference: pinned.clone(),
            config: id.clone(),
        });
        self.admit_resolved_input_record(inputs)?;
        Ok((id, pinned))
    }

    fn recipe_image_refs(&self) -> Result<(String, String, String), Error> {
        let rocky = recipe_base(&PathBuf::from(&self.source).join("project-os/Containerfile"))
            .map_err(|e| Error::msg(e.to_string()))?;
        let dashboard =
            recipe_base(&PathBuf::from(&self.source).join("appliance/dashboard.Containerfile"))
                .map_err(|e| Error::msg(e.to_string()))?;
        if rocky != dashboard {
            return Err(Error::msg("dashboard and Project OS base owners disagree"));
        }
        let forgejo =
            unit_image(&PathBuf::from(&self.source).join("appliance/services/forgejo.container"))
                .map_err(|e| Error::msg(e.to_string()))?;
        let proxy = unit_image(
            &PathBuf::from(&self.source).join("appliance/services/soda-proxy.container"),
        )
        .map_err(|e| Error::msg(e.to_string()))?;
        Ok((rocky, forgejo, proxy))
    }

    /// Admits the controller-resolved floating Tailnet toolchain for this
    /// attempt. The file is validated on read; the pull digests join the
    /// record downstream.
    fn live_tailnet_inputs(&self) -> Result<TailnetInputs, Error> {
        if self.live_inputs.is_empty() {
            return Err(Error::msg("live inputs file required for floating Tailnet"));
        }
        Ok(read_live_inputs(Path::new(&self.live_inputs))?.tailnet)
    }

    /// Records the actual upstream manifests once, before shipping work.
    pub fn resolve_inputs(&mut self) -> Result<(), Error> {
        self.validate()?;
        match std::fs::symlink_metadata(PathBuf::from(&self.out).join("app-inputs.json")) {
            Err(e) if e.kind() == std::io::ErrorKind::NotFound => {}
            _ => return Err(Error::msg("occupied image input record")),
        }
        let platform = oci_architecture(&self.arch).unwrap_or("amd64").to_string();
        let (rocky, forgejo, proxy) = self.recipe_image_refs()?;
        let tailnet = self.live_tailnet_inputs()?;
        let tail_base = tailnet.base.clone();
        let mut inputs = Vec::new();
        for reference in [&rocky, &forgejo, &proxy, &tail_base] {
            self.pull_resolved_input(reference, reference, "", &platform, &mut inputs)?;
        }
        self.inputs = inputs;
        Ok(())
    }

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
        let rocky = recipe_base(&PathBuf::from(&self.source).join("project-os/Containerfile"))
            .map_err(|e| Error::msg(e.to_string()))?;
        let dashboard =
            recipe_base(&PathBuf::from(&self.source).join("appliance/dashboard.Containerfile"))
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
                "project-os/Containerfile"
            } else {
                "appliance/dashboard.Containerfile"
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
            unit_image(&PathBuf::from(&self.source).join("appliance/services/forgejo.container"))
                .map_err(|e| Error::msg(e.to_string()))?;
        let (_, pinned) = pull("Forgejo", &forgejo, "forgejo-base")?;
        let id = build("forgejo", forgejo_context, "Containerfile", &pinned, &[])?;
        export("forgejo", &id, &self.revision)
    }

    fn export_proxy_image(&self, pull: PullFn, export: ExportFn) -> Result<(), Error> {
        let proxy = unit_image(
            &PathBuf::from(&self.source).join("appliance/services/soda-proxy.container"),
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
            "appliance/tailnet.Containerfile",
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

    fn export_images(
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

    /// Compiles the exact archived fork under musl.
    pub fn build_forgejo_binary(&self) -> Result<String, Error> {
        crate::forgejo::build_forgejo_binary(self)
    }

    /// Builds the fork binary into this build's image context.
    pub fn stage_fork_binary(&self, context: &str) -> Result<(), Error> {
        crate::forgejo::stage_fork_binary(self, context)
    }
}

/// Verified image plus its archive digest.
#[derive(Debug, Clone, Default)]
pub struct ProducedImage {
    pub image: Image,
    pub archive_sha256: String,
}

/// One frozen upstream input: requested ref, pinned ref, config ID.
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct ResolvedInput {
    pub requested: String,
    pub reference: String,
    pub config: String,
}

impl ResolvedInput {
    pub fn emit(&self) -> Emit {
        Emit::Object(vec![
            ("Requested".to_string(), Emit::Str(self.requested.clone())),
            ("Reference".to_string(), Emit::Str(self.reference.clone())),
            ("Config".to_string(), Emit::Str(self.config.clone())),
        ])
    }
}

fn parse_image_repo(reference: &str) -> String {
    let repo = reference.split('@').next().unwrap_or(reference);
    match repo.rfind(':') {
        Some(colon) if colon > repo.rfind('/').unwrap_or(0) => repo[..colon].to_string(),
        _ => repo.to_string(),
    }
}

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

#[cfg(test)]
mod tests {
    use super::*;
    use crate::coreos_stream::tests::fixture_live_inputs;
    use crate::coreos_stream::write_live_inputs;
    use crate::oci::tests::{fixture_oci_bytes, FIXTURE_REVISION};
    use std::os::unix::fs::PermissionsExt;
    use std::sync::{Arc, Mutex};

    #[allow(clippy::field_reassign_with_default)]
    fn production_fixture() -> (Production, Arc<Mutex<Vec<String>>>, PathBuf) {
        let root = std::env::temp_dir().join(format!(
            "soda-prod-{}-{}",
            std::process::id(),
            std::time::SystemTime::now()
                .duration_since(std::time::UNIX_EPOCH)
                .unwrap()
                .as_nanos()
        ));
        let out = root.join(".artifacts/native/x86_64");
        std::fs::create_dir_all(&out).unwrap();
        let files = [
            (
                "package.json",
                "{\"packageManager\":\"bun@1.4.2\",\"unrelated\":true}",
            ),
            (
                "project-os/Containerfile",
                "ARG BASE_IMAGE=docker.io/rockylinux/rockylinux:10.2\n",
            ),
            (
                "appliance/dashboard.Containerfile",
                "ARG BASE_IMAGE=docker.io/rockylinux/rockylinux:10.2\n",
            ),
            (
                "appliance/services/forgejo.container",
                "[Container]\nImage=codeberg.org/forgejo/forgejo:15.0.9\n",
            ),
            (
                "appliance/services/soda-proxy.container",
                "[Container]\nImage=docker.io/library/caddy:2\n",
            ),
        ];
        for (name, body) in files {
            let path = root.join(name);
            std::fs::create_dir_all(path.parent().unwrap()).unwrap();
            std::fs::write(&path, body).unwrap();
        }
        let seed = fixture_oci_bytes("amd64");
        let seed_path = root.join("seed.oci");
        std::fs::write(&seed_path, &seed).unwrap();
        let image = inspect_oci(&seed_path, "x86_64", FIXTURE_REVISION).unwrap();
        let live_path = root.join("live-inputs.json");
        write_live_inputs(&live_path, &fixture_live_inputs()).unwrap();
        let calls: Arc<Mutex<Vec<String>>> = Arc::new(Mutex::new(Vec::new()));
        let calls_capture = calls.clone();
        let config = image.config.clone();
        let mut prod = Production::default();
        prod.source = root.to_string_lossy().into_owned();
        prod.native = out.to_string_lossy().into_owned();
        prod.out = out.to_string_lossy().into_owned();
        prod.arch = "x86_64".to_string();
        prod.revision = FIXTURE_REVISION.to_string();
        prod.live_inputs = live_path.to_string_lossy().into_owned();
        let calls_next = calls.clone();
        prod.next = Some(Box::new(move |label| {
            calls_next.lock().unwrap().push(format!("STEP {label}"));
            Ok(())
        }));
        prod.capture = Some(Box::new(move |_dir, name, args| {
            calls_capture
                .lock()
                .unwrap()
                .push(format!("{name} {}", args.join(" ")));
            if name == "bun" {
                return Ok("1.4.2".to_string());
            }
            assert_eq!(name, "podman");
            assert_eq!(args[0], "--remote=false");
            if args[1] == "pull" {
                return Ok(config.clone());
            }
            if args[1] == "image" {
                return Ok(format!("sha256:{}", "b".repeat(64)));
            }
            panic!("unexpected observation: {args:?}");
        }));
        let calls_exec = calls.clone();
        let seed_bytes = seed.clone();
        let image_config = image.config.clone();
        prod.execute = Some(Box::new(move |dir, name, args| {
            calls_exec
                .lock()
                .unwrap()
                .push(format!("{name} {}", args.join(" ")));
            if name == "go" && args.first().map(String::as_str) == Some("build") {
                for (i, arg) in args.iter().enumerate() {
                    if arg == "-o" {
                        let dest = PathBuf::from(&args[i + 1]);
                        std::fs::create_dir_all(dest.parent().unwrap()).unwrap();
                        std::fs::write(&dest, crate::elf::tests::fixture_elf()).unwrap();
                        return Ok(());
                    }
                }
            }
            if name == "cargo" && args.first().map(String::as_str) == Some("build") {
                for (i, arg) in args.iter().enumerate() {
                    if arg == "-p" && i + 1 < args.len() {
                        let mut bin = args[i + 1].clone();
                        if bin == "soda-project-terminal" {
                            bin = "project-terminal".to_string();
                        }
                        if bin == "soda-project-account" {
                            bin = "project-account".to_string();
                        }
                        if bin == "soda-project-factory-roles" {
                            bin = "project-factory-roles".to_string();
                        }
                        let dest = PathBuf::from(dir).join("target/release").join(bin);
                        std::fs::create_dir_all(dest.parent().unwrap()).unwrap();
                        std::fs::write(&dest, crate::elf::tests::fixture_elf()).unwrap();
                        return Ok(());
                    }
                }
            }
            if name != "podman" {
                return Ok(());
            }
            assert_eq!(args[0], "--remote=false");
            for (i, arg) in args.iter().enumerate() {
                if arg == "--iidfile" {
                    return write_new(Path::new(&args[i + 1]), image_config.as_bytes(), 0o600);
                }
                if arg == "--output" {
                    return write_new(Path::new(&args[i + 1]), &seed_bytes, 0o600);
                }
            }
            panic!("unexpected execution: {args:?}");
        }));
        (prod, calls, root)
    }

    #[test]
    fn oracle_production_sequence() {
        // Oracle: TestProductionUsesOneAssetAndImageSequence.
        let (mut prod, calls, _root) = production_fixture();
        let host = PathBuf::from(&prod.out).join("context");
        let forgejo = PathBuf::from(&prod.out).join("forgejo-context");
        prod.dependencies().unwrap();
        prod.resolve_inputs().unwrap();
        prod.assets(&host.to_string_lossy(), &forgejo.to_string_lossy())
            .unwrap();
        for tool in [
            "muse",
            "soda-identity-compose",
            "project-terminal",
            "project-account",
        ] {
            let path = PathBuf::from(&prod.native)
                .join("project-tools/bin")
                .join(tool);
            let mode = std::fs::metadata(&path).unwrap().permissions().mode() & 0o777;
            assert_eq!(mode, 0o755, "{tool}");
        }
        let images = prod.images(&forgejo.to_string_lossy()).unwrap();
        assert_eq!(images.len(), 6);
        let text = calls.lock().unwrap().join("\n");
        for needle in [
            "bun install --frozen-lockfile",
            "bun scripts/build-forgejo.ts",
            "cargo run --release --locked -p soda-asset-fetchers --bin soda-fetch-terminal -- --out ",
            "cargo run --release --locked -p soda-asset-fetchers --bin soda-fetch-muse -- --arch x86_64 --out ",
            "cargo run --release --locked -p soda-asset-fetchers --bin soda-fetch-tea -- --arch x86_64 --out ",
            "cargo run --release --locked -p soda-forgejo-locales --bin soda-forgejo-locales -- --lock appliance/forgejo/locale.lock.json --out ",
            "cargo run --release --locked -p soda-stage-render --bin soda-stage -- --arch x86_64 --host-context ",
            "STEP Build image: dashboard\n",
            "STEP Build image: project-os\n",
            "STEP Build image: tailnet\n",
            "STEP Build image: forgejo\n",
            "bun scripts/build-soda-extension.ts --out ",
        ] {
            assert_eq!(text.matches(needle).count(), 1, "needle: {needle}\n{text}");
        }
        assert!(!text.contains("python3 scripts/fetch-"));
        assert!(!text.contains("tools/soda-fetch-muse"));
        assert!(text.contains(&format!(
            "--host-context {} --forgejo-context {}",
            host.display(),
            forgejo.display()
        )));
        assert_eq!(text.matches(" save --format=oci-archive").count(), 6);
        assert!(
            !text.contains(" push ") && !text.contains(" --rm ") && !text.contains("--replace")
        );
        assert!(images.contains_key("proxy"));
        assert!(!images.contains_key("caddy"));
        for (name, produced) in &images {
            assert!(is_digest(&produced.archive_sha256));
            assert!(!produced.image.manifest.is_empty() && !produced.image.config.is_empty());
            assert!(PathBuf::from(&prod.out)
                .join("images")
                .join(format!("{name}.oci"))
                .exists());
        }
        let before = calls.lock().unwrap().len();
        assert!(prod.images(&forgejo.to_string_lossy()).is_err());
        assert_eq!(calls.lock().unwrap().len(), before);
    }

    #[test]
    fn oracle_production_failure_stops() {
        // Oracle: TestProductionFailureStopsBeforeLaterImages.
        let (mut prod, calls, _root) = production_fixture();
        let execute = prod.execute.take();
        prod.execute = Some(Box::new(move |dir, name, args| {
            if name == "podman" && args.join(" ").contains("dashboard.Containerfile") {
                return Err(Error::msg("fixture build refused"));
            }
            execute.as_ref().unwrap()(dir, name, args)
        }));
        prod.resolve_inputs().unwrap();
        let err = prod.images("forgejo-context").unwrap_err();
        assert_eq!(err.message(), "fixture build refused");
        let text = calls.lock().unwrap().join("\n");
        assert!(!text.contains("STEP Build image: project-os"));
        assert!(!text.contains("STEP Export and verify"));
        assert!(PathBuf::from(&prod.out).join("app-inputs.json").exists());
    }

    #[test]
    fn oracle_production_refusals() {
        // Oracle: TestProductionRefusesWrongToolchainInputsAndLayout.
        for mode in ["bun", "base", "tailnet", "forgejo", "platform", "revision"] {
            let (mut prod, _calls, _root) = production_fixture();
            match mode {
                "bun" => {
                    prod.capture = Some(Box::new(|_, _, _| Ok("different".to_string())));
                    assert!(prod.dependencies().is_err(), "{mode}");
                    continue;
                }
                "base" => {
                    std::fs::write(
                        PathBuf::from(&prod.source).join("appliance/dashboard.Containerfile"),
                        b"ARG BASE_IMAGE=unrelated\n",
                    )
                    .unwrap();
                }
                "tailnet" => {
                    let raw = fixture_live_inputs();
                    let mut text = raw.marshal();
                    text =
                        text.replacen("\"Version\": \"1.98.2\"", "\"Version\": \"yesterday\"", 1);
                    std::fs::write(PathBuf::from(&prod.source).join("live-inputs.json"), text)
                        .unwrap();
                }
                "forgejo" => {
                    assert!(prod.images("").is_err(), "{mode}");
                    continue;
                }
                "platform" => prod.arch = "other".to_string(),
                "revision" => prod.revision = "dirty".to_string(),
                _ => unreachable!(),
            }
            assert!(prod.images("forgejo-context").is_err(), "{mode}");
        }
    }

    #[test]
    fn oracle_asset_destinations_refuse_early() {
        // Oracle: TestProductionAssetDestinationsRefuseBeforeCommands.
        for destinations in [
            ("/host-context", ""),
            ("", "/forgejo-context"),
            ("relative-host", "/forgejo-context"),
            ("/host-context", "relative-forgejo"),
        ] {
            let (prod, calls, _root) = production_fixture();
            assert!(prod.assets(destinations.0, destinations.1).is_err());
            assert!(calls.lock().unwrap().is_empty());
        }
    }

    #[test]
    fn oracle_compile_recipes() {
        // Oracle: TestProductionCompileKeepsLayoutAndVerifiesELF +
        // TestProductionCompileRustKeepsLayoutAndVerifiesELF.
        let (prod, calls, _root) = production_fixture();
        let dest = PathBuf::from(&prod.out).join("program");
        prod.compile("soda-host", "./cmd/soda-host", &dest.to_string_lossy())
            .unwrap();
        let text = calls.lock().unwrap().join("\n");
        assert!(text.contains("-buildvcs=false") && !text.contains("-tags="));
        assert_eq!(text.matches("go build").count(), 1);
        let dest = PathBuf::from(&prod.out).join("soda-identity-compose");
        prod.compile_rust(
            "soda-identity-compose",
            "soda-identity-compose",
            &dest.to_string_lossy(),
        )
        .unwrap();
        let text = calls.lock().unwrap().join("\n");
        assert!(
            text.contains("cargo build --release --locked")
                && text.contains("-p soda-identity-compose")
        );
        assert_eq!(text.matches("cargo build").count(), 1);
        for (bad_crate, bad_bin) in [
            ("", "bin"),
            ("crate", ""),
            ("a/b", "bin"),
            ("crate", "a/bin"),
        ] {
            assert!(prod
                .compile_rust(bad_crate, bad_bin, &dest.to_string_lossy())
                .is_err());
        }
    }

    #[test]
    fn oracle_image_repo_parsing() {
        assert_eq!(
            parse_image_repo("docker.io/library/caddy:2"),
            "docker.io/library/caddy"
        );
        assert_eq!(parse_image_repo("quay.io/r@sha256:aaaa"), "quay.io/r");
        assert_eq!(parse_image_repo("localhost:5000/r:tag"), "localhost:5000/r");
        assert_eq!(parse_image_repo("localhost:5000/r"), "localhost:5000/r");
    }
}

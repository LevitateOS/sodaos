//! Production compile recipes: Rust/Go command builds, ELF and output
//! checks, the pinned Bun gate, and dependency admission.

use crate::production::Production;
use crate::{io_error, Error};
use soda_json::JsonValue;
use std::path::{Path, PathBuf};

impl Production {
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
}

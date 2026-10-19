//! Production compile recipes: Rust/Go command builds, ELF and output
//! checks and dependency setup.

use crate::production::Production;
use crate::{io_error, Error};
use std::env;
use std::path::{Path, PathBuf};

impl Production {
    /// The sole Rust command recipe for ported runtime programs. Dependencies
    /// are staged before worker admission; the release binary is copied to `dest`.
    pub fn compile_rust(&self, crate_name: &str, bin: &str, dest: &str) -> Result<(), Error> {
        self.validate()?;
        if crate_name.is_empty() || bin.is_empty() || crate_name.contains('/') || bin.contains('/')
        {
            return Err(Error::msg("explicit Rust crate and binary required"));
        }
        let manifest = PathBuf::from(&self.source).join("Cargo.toml");
        self.call_execute(
            &self.source,
            "cargo",
            &[
                "build".to_string(),
                "--release".to_string(),
                "--locked".to_string(),
                "--offline".to_string(),
                "--config".to_string(),
                "profile.release.strip=\"symbols\"".to_string(),
                "--manifest-path".to_string(),
                manifest.to_string_lossy().into_owned(),
                "-p".to_string(),
                crate_name.to_string(),
                "--bin".to_string(),
                bin.to_string(),
            ],
        )?;
        let configured_target = env::var_os("CARGO_TARGET_DIR")
            .map(PathBuf::from)
            .unwrap_or_else(|| PathBuf::from("target"));
        let target = if configured_target.is_absolute() {
            configured_target
        } else {
            PathBuf::from(&self.source).join(configured_target)
        };
        let raw = std::fs::read(target.join("release").join(bin))
            .map_err(|e| io_error("open", Path::new(bin), e))?;
        let dest_path = PathBuf::from(dest);
        std::fs::write(&dest_path, &raw).map_err(|e| io_error("open", &dest_path, e))?;
        crate::files::chmod(&dest_path, 0o755)?;
        crate::elf::inspect_elf(&dest_path, &self.arch)
    }

    /// The sole Go command recipe for runtime programs and support tools.
    /// Binaries use archive-safe VCS mode; the single image layout owns
    /// every install path.
    pub fn compile(&self, pkg: &str, dest: &str) -> Result<(), Error> {
        self.validate()?;
        self.call_execute(
            &self.source,
            "go",
            &[
                "build".to_string(),
                "-mod=readonly".to_string(),
                "-trimpath".to_string(),
                "-buildvcs=false".to_string(),
                "-ldflags=-s -w".to_string(),
                "-o".to_string(),
                dest.to_string(),
                pkg.to_string(),
            ],
        )?;
        let dest_path = PathBuf::from(dest);
        crate::files::chmod(&dest_path, 0o755)?;
        crate::elf::inspect_elf(&dest_path, &self.arch)
    }

    /// Runs before any production compilation.
    pub fn dependencies(&self) -> Result<(), Error> {
        self.validate()?;
        self.call_execute(
            &self.source,
            "go",
            &["mod".to_string(), "verify".to_string()],
        )?;
        self.call_execute(
            &self.source,
            "bun",
            &["install".to_string(), "--frozen-lockfile".to_string()],
        )
    }
}

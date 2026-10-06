//! Production asset staging: the one-time frontend, terminal, locales,
//! and upstream-tools sequence plus project helper compilation.

use crate::production::Production;
use crate::{io_error, Error};
use std::path::{Path, PathBuf};

impl Production {
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
            "soda-release-assets",
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
            "soda-project-terminal",
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
                    "soda-release-assets".to_string(),
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
                    "soda-release-assets".to_string(),
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
                    "soda-release-assets".to_string(),
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
                    "soda-release-assets".to_string(),
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
}

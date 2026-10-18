//! Production resolved inputs: the frozen record, pull and digest
//! admission, recipe references, and resolve_inputs.
//!
//! The frozen set is the Rocky base, the admitted Fountain image named by
//! the Fountain unit, the two upstream application images (Caddy/postgres
//! units), and the one pinned official Tailscale image. There is no floating
//! Tailnet toolchain resolution and no custom Tailscale image build.

use crate::files::{is_digest, oci_architecture, write_new};
use crate::production::Production;
use crate::{io_error, Error};
use serde::Serialize;
use soda_build_tools::reader::settings::{recipe_base, unit_image};
use std::path::PathBuf;

/// The one pinned official Tailscale image: the only Tailnet OCI identity,
/// admitted by immutable digest. No RPM, daemon, or custom image in the
/// default install; G-T proves the runtime behavior separately.
pub const TAILSCALE_OFFICIAL_IMAGE: &str = "docker.io/tailscale/tailscale@sha256:99691662a2e82fbfd2bf55db0fd52bacd2f4664ce6fd1deca3746a024144a2b7";
pub const FOUNTAIN_IDENTITY: &str = r#"{"os":{{json .Os}},"architecture":{{json .Architecture}},"revision":{{json (index .Labels "org.opencontainers.image.revision")}},"version":{{json (index .Labels "org.opencontainers.image.version")}}}"#;

/// One frozen upstream input: requested ref, pinned ref, config ID.
#[derive(Debug, Clone, Default, PartialEq, Eq, Serialize)]
pub struct ResolvedInput {
    #[serde(rename = "Requested")]
    pub requested: String,
    #[serde(rename = "Reference")]
    pub reference: String,
    #[serde(rename = "Config")]
    pub config: String,
}

pub(crate) fn parse_image_repo(reference: &str) -> String {
    let repo = reference.split('@').next().unwrap_or(reference);
    match repo.rfind(':') {
        Some(colon) if colon > repo.rfind('/').unwrap_or(0) => repo[..colon].to_string(),
        _ => repo.to_string(),
    }
}

impl Production {
    /// Produces each selected app archive once.
    pub(crate) fn pull_frozen_image(
        &self,
        inputs: &[ResolvedInput],
        reference: &str,
        iid_name: &str,
    ) -> Result<(String, String), Error> {
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

    fn admit_resolved_input_record(&self, inputs: &[ResolvedInput]) -> Result<(), Error> {
        let body = serde_json::to_string_pretty(inputs)
            .expect("serializing resolved app inputs cannot fail")
            + "\n";
        // This private, fresh production attempt owns this growing record.
        let path = PathBuf::from(&self.out).join("app-inputs.json");
        std::fs::write(&path, body.as_bytes()).map_err(|e| io_error("open", &path, e))?;
        crate::files::chmod(&path, 0o600)
    }

    pub(crate) fn pull_resolved_input(
        &self,
        reference: &str,
        iid_name: &str,
        platform: &str,
        inputs: &mut Vec<ResolvedInput>,
    ) -> Result<(String, String), Error> {
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

    fn recipe_image_refs(&self) -> Result<(String, String, String, String), Error> {
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
        let forgejo = self
            .fountain
            .as_ref()
            .ok_or_else(|| Error::msg("admitted Fountain release required"))?
            .image
            .clone();
        let proxy = unit_image(
            &PathBuf::from(&self.source).join("system/host/services/soda-proxy.container"),
        )
        .map_err(|e| Error::msg(e.to_string()))?;
        let postgres = unit_image(
            &PathBuf::from(&self.source).join("system/host/services/soda-postgres.container"),
        )
        .map_err(|e| Error::msg(e.to_string()))?;
        Ok((rocky, forgejo, proxy, postgres))
    }

    fn admit_fountain_dispatch(&mut self) -> Result<(), Error> {
        self.validate()?;
        match std::fs::symlink_metadata(PathBuf::from(&self.out).join("app-inputs.json")) {
            Err(e) if e.kind() == std::io::ErrorKind::NotFound => {}
            _ => return Err(Error::msg("occupied image input record")),
        }
        let live = crate::live_inputs::read_live_inputs(&PathBuf::from(&self.live_inputs))?;
        if live.fountain.source_revision != self.forgejo_revision {
            return Err(Error::msg("Fountain source differs from frozen dispatch"));
        }
        self.fountain = Some(live.fountain);
        let timestamp = self.call_capture(
            &self.source,
            "git",
            &[
                "show".into(),
                "--no-patch".into(),
                "--format=%ct".into(),
                self.revision.clone(),
            ],
        )?;
        self.source_timestamp = Some(
            timestamp
                .trim()
                .parse::<u64>()
                .ok()
                .filter(|value| *value > 0)
                .ok_or_else(|| Error::msg("source timestamp unavailable"))?,
        );
        let proof = serde_json::to_vec_pretty(self.fountain.as_ref().unwrap())
            .map_err(|_| Error::msg("invalid Fountain proof"))?;
        write_new(
            &PathBuf::from(&self.out).join("fountain-release.json"),
            &proof,
            0o644,
        )?;
        Ok(())
    }

    fn check_fountain_platform_manifest(
        &self,
        config: String,
        pinned: String,
        fountain: &soda_build_tools::payload::FountainRelease,
    ) -> Result<(), Error> {
        if config != fountain.image_config || pinned != fountain.image {
            return Err(Error::msg(
                "Fountain platform manifest/config differs from release metadata",
            ));
        }
        #[derive(serde::Deserialize)]
        #[serde(deny_unknown_fields)]
        struct Identity {
            os: String,
            architecture: String,
            revision: String,
            version: String,
        }
        let body = self.call_capture(
            &self.source,
            "podman",
            &[
                "--remote=false".into(),
                "image".into(),
                "inspect".into(),
                "--format".into(),
                FOUNTAIN_IDENTITY.into(),
                config,
            ],
        )?;
        let identity: Identity = serde_json::from_str(&body)
            .map_err(|_| Error::msg("Fountain image identity unavailable"))?;
        if identity.os != "linux"
            || identity.architecture != "amd64"
            || identity.revision != fountain.source_revision
            || identity.version != fountain.version
        {
            return Err(Error::msg(
                "Fountain image labels/platform differ from release metadata",
            ));
        }
        Ok(())
    }

    /// Records the actual upstream manifests once, before shipping work.
    /// Every pull is platform-scoped, so the recorded digest is the exact
    /// Linux child manifest the runtime verifies, never a mutable tag or a
    /// multi-platform index.
    pub fn resolve_inputs(&mut self) -> Result<(), Error> {
        self.admit_fountain_dispatch()?;
        let platform = oci_architecture(&self.arch).unwrap_or("amd64").to_string();
        let (rocky, forgejo, proxy, postgres) = self.recipe_image_refs()?;
        let mut inputs = Vec::new();
        for reference in [
            &rocky,
            &forgejo,
            &proxy,
            &postgres,
            &TAILSCALE_OFFICIAL_IMAGE.to_string(),
        ] {
            let (config, pinned) =
                self.pull_resolved_input(reference, "", &platform, &mut inputs)?;
            if reference == &forgejo {
                let fountain = self.fountain.as_ref().unwrap();
                self.check_fountain_platform_manifest(config, pinned, fountain)?;
            }
        }
        self.inputs = inputs;
        Ok(())
    }
}

impl Production {
    pub fn frozen_input(&self, role: &str) -> Option<&ResolvedInput> {
        let reference = match role {
            "tailscale" => TAILSCALE_OFFICIAL_IMAGE.to_string(),
            "proxy" | "postgres" => {
                let unit = if role == "proxy" {
                    "soda-proxy.container"
                } else {
                    "soda-postgres.container"
                };
                unit_image(
                    &PathBuf::from(&self.source)
                        .join("system/host/services")
                        .join(unit),
                )
                .ok()?
            }
            _ => return None,
        };
        self.inputs
            .iter()
            .find(|input| input.requested == reference)
    }
}

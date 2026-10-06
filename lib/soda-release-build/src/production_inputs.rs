//! Production resolved inputs: the frozen record, pull and digest
//! admission, recipe references, live Tailnet inputs, and resolve_inputs.

use crate::coreos_stream::{read_live_inputs, TailnetInputs};
use crate::files::{is_digest, oci_architecture, write_new};
use crate::json_emit::{marshal_indent, Emit};
use crate::production::Production;
use crate::{io_error, Error};
use soda_build_tools::reader::settings::{recipe_base, unit_image};
use std::path::{Path, PathBuf};

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
        let rocky = recipe_base(&PathBuf::from(&self.source).join("system/project/Containerfile"))
            .map_err(|e| Error::msg(e.to_string()))?;
        let dashboard = recipe_base(
            &PathBuf::from(&self.source).join("system/containers/dashboard/Containerfile"),
        )
        .map_err(|e| Error::msg(e.to_string()))?;
        if rocky != dashboard {
            return Err(Error::msg("dashboard and Project OS base owners disagree"));
        }
        let forgejo =
            unit_image(&PathBuf::from(&self.source).join("system/host/services/forgejo.container"))
                .map_err(|e| Error::msg(e.to_string()))?;
        let proxy = unit_image(
            &PathBuf::from(&self.source).join("system/host/services/soda-proxy.container"),
        )
        .map_err(|e| Error::msg(e.to_string()))?;
        Ok((rocky, forgejo, proxy))
    }

    /// Admits the controller-resolved floating Tailnet toolchain for this
    /// attempt. The file is validated on read; the pull digests join the
    /// record downstream.
    pub(crate) fn live_tailnet_inputs(&self) -> Result<TailnetInputs, Error> {
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
}

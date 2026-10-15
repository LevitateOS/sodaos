//! Docker packaging consumes the verified candidate from this same attempt.
//! Podman owns local building/storage; no Docker daemon or registry writer is
//! granted to the isolated worker.

use std::path::PathBuf;

use soda_build_tools::reader::settings::recipe_base;
use soda_release_deliver::docker::{self, DockerRequest};
use soda_release_deliver::model::{BuildPurpose, Distribution};

use crate::files::{hash_file, write_new};
use crate::production::Production;
use crate::production_inputs::parse_image_repo;
use crate::{io_error, Error};

impl Production {
    /// Extend the common frozen inputs before dependency installation/compilation.
    /// The five R02 production inputs (Rocky base, admitted Fountain image,
    /// upstream Caddy/postgres, pinned official Tailscale) precede the
    /// Fedora Docker base pull.
    pub fn prepare_docker(&mut self) -> Result<(), Error> {
        self.validate()?;
        if self.inputs.len() != 5 {
            return Err(Error::msg(
                "Docker base requires the frozen common candidate inputs",
            ));
        }
        let recipe = PathBuf::from(&self.source).join("system/docker/Dockerfile");
        let requested = recipe_base(&recipe)?;
        if parse_image_repo(&requested) != "registry.fedoraproject.org/fedora" {
            return Err(Error::msg(
                "Docker recipe requires the Fedora registry base",
            ));
        }
        let mut inputs = self.inputs.clone();
        self.pull_resolved_input(&requested, "", "amd64", &mut inputs)?;
        self.inputs = inputs;
        Ok(())
    }

    /// Export one OCI archive and verify every candidate-owned byte before
    /// issuing its metadata. The host config ID selects the local candidate;
    /// --pull=never prevents either stage from resolving a mutable tag.
    pub fn build_docker(
        &self,
        release_tag: &str,
        distribution: Distribution,
        purpose: BuildPurpose,
    ) -> Result<(), Error> {
        self.validate()?;
        if ![Distribution::Docker, Distribution::All].contains(&distribution)
            || !soda_build_tools::reader::settings::release_tag(release_tag)
            || release_tag.eq_ignore_ascii_case("latest")
        {
            return Err(Error::msg(
                "explicit Docker distribution and version required",
            ));
        }
        let recipe = PathBuf::from(&self.source).join("system/docker/Dockerfile");
        let requested = recipe_base(&recipe)?;
        let (_, base) = self.pull_frozen_image(&self.inputs, &requested, "")?;
        let request = DockerRequest {
            release_tag: release_tag.to_owned(),
            distribution,
            purpose,
            fedora_base: base.clone(),
        };
        let out = PathBuf::from(&self.out);
        let candidate_path = out.join("candidate.json");
        let (_, candidate, _) = soda_release_deliver::admission::load_admitted_candidate(&self.out)
            .map_err(|e| Error::msg(e.0))?;
        if candidate.host.revision != self.revision || candidate.architecture != self.arch {
            return Err(Error::msg(
                "Docker candidate differs from this production attempt",
            ));
        }
        let payload_hash = hash_file(&out.join("payload.json"))?;
        if payload_hash != candidate.payload_sha256 {
            return Err(Error::msg("Docker candidate payload differs"));
        }
        let candidate_hash = hash_file(&candidate_path)?;
        let recipe_hash = hash_file(&recipe)?;
        let context = PathBuf::from(&self.source).join("system/docker");
        let args = vec![
            format!("--build-arg=SODA_BASE_IMAGE={}", candidate.host.config),
            format!("--label=org.opencontainers.image.version={release_tag}"),
            format!(
                "--label=io.soda.distribution={}",
                if distribution == Distribution::All {
                    "all"
                } else {
                    "docker"
                }
            ),
            format!(
                "--label=io.soda.build-purpose={}",
                if purpose == BuildPurpose::Production {
                    "production"
                } else {
                    "development"
                }
            ),
            format!("--label=io.soda.candidate-sha256={candidate_hash}"),
            format!("--label=io.soda.payload-sha256={payload_hash}"),
            format!("--label=io.soda.docker.recipe-sha256={recipe_hash}"),
        ];
        let id = self.build_image(
            "docker",
            &context.to_string_lossy(),
            "Dockerfile",
            &base,
            &args,
        )?;
        let dir = out.join("docker");
        std::fs::create_dir(&dir).map_err(|e| io_error("mkdir", &dir, e))?;
        crate::files::chmod(&dir, 0o755)?;
        self.export_image_archive(&dir, "image", &id, &self.revision)?;
        let packages = self.call_capture(
            &self.source,
            "podman",
            &[
                "--remote=false".into(),
                "run".into(),
                "--pull=never".into(),
                "--cidfile".into(),
                out.join("docker-packages.cid")
                    .to_string_lossy()
                    .into_owned(),
                "--network=none".into(),
                "--read-only".into(),
                "--cap-drop=all".into(),
                "--security-opt=no-new-privileges".into(),
                "--entrypoint=/usr/bin/cat".into(),
                id,
                "/usr/share/soda/docker-runtime/packages.txt".into(),
            ],
        )?;
        soda_build_tools::reader::rpm::valid_rpm_inventory(
            &packages.lines().map(str::to_owned).collect::<Vec<_>>(),
        )?;
        write_new(
            &dir.join("packages.txt"),
            format!("{packages}\n").as_bytes(),
            0o644,
        )?;
        let compose = context.join("compose.yaml");
        let body = std::fs::read(&compose).map_err(|e| io_error("open", &compose, e))?;
        write_new(&dir.join("compose.yaml"), &body, 0o644)?;
        docker::record(&self.out, &request).map_err(|e| Error::msg(e.0))?;
        Ok(())
    }
}

// Launcher verification and required-tool resolution/recording.
use std::time::Instant;

use super::paths::{preparation_paths, single_line, valid_resolved_tool_path};
use super::state::LauncherEvidence;
use crate::json::{self};
use crate::preparation::{self, Preparation, ResolvedTool};
use crate::project::{Executor, Runtime};

impl<E: Executor> Runtime<E> {
    /// `verifyLauncherEnvironment`: observe the role; refusals are data.
    pub(crate) fn verify_launcher_environment(
        &self,
        prep: &Preparation,
        deadline: Instant,
    ) -> Result<LauncherEvidence, String> {
        let mut evidence = LauncherEvidence::default();
        let uid = single_line(
            &self.must_role_exec(prep, &["/usr/bin/id", "-u"], deadline),
            64,
        );
        let Some(uid) = uid else {
            evidence.refusal = "role uid is not observable".to_string();
            return Ok(evidence);
        };
        evidence.uid = uid;
        let login = single_line(
            &self.must_role_exec(prep, &["/usr/bin/id", "-un"], deadline),
            64,
        );
        match login {
            Some(login) if login == prep.role => evidence.login = login,
            _ => {
                evidence.refusal = "launcher is not the assigned role".to_string();
                return Ok(evidence);
            }
        }
        let groups = single_line(
            &self.must_role_exec(prep, &["/usr/bin/id", "-Gn"], deadline),
            256,
        );
        match groups {
            Some(groups) if groups == prep.role => evidence.groups = groups,
            _ => {
                evidence.refusal = "role holds unexpected groups".to_string();
                return Ok(evidence);
            }
        }
        let (_, snapshot, _, home) = preparation_paths(&prep.role, &prep.id);
        let setup_entry = format!("{snapshot}/{}", preparation::FACTORY_SETUP_ENTRY);
        if self
            .role_exec(prep, &["/usr/bin/test", "-r", &setup_entry], deadline)
            .is_err()
        {
            evidence.refusal = "approved setup is not readable".to_string();
            return Ok(evidence);
        }
        if self
            .role_exec(prep, &["/usr/bin/test", "-w", &setup_entry], deadline)
            .is_ok()
        {
            evidence.refusal = "approved setup is writable by the role".to_string();
            return Ok(evidence);
        }
        if self
            .role_exec(
                prep,
                &["/usr/bin/test", "-w", "/srv/project/shared"],
                deadline,
            )
            .is_ok()
        {
            evidence.refusal = "shared project data is writable by the role".to_string();
            return Ok(evidence);
        }
        for socket in ["/run/podman/podman.sock", "/run/docker.sock"] {
            if self
                .role_exec(prep, &["/usr/bin/test", "-e", socket], deadline)
                .is_ok()
            {
                evidence.refusal = "engine socket is visible to the role".to_string();
                return Ok(evidence);
            }
        }
        if self
            .role_exec(prep, &["/usr/bin/test", "-x", "/usr/bin/sudo"], deadline)
            .is_ok()
            && self
                .role_exec(prep, &["/usr/bin/sudo", "-n", "true"], deadline)
                .is_ok()
        {
            evidence.refusal = "role holds unexpected privilege".to_string();
            return Ok(evidence);
        }
        if self
            .role_exec(prep, &["/usr/bin/test", "-w", &home], deadline)
            .is_err()
        {
            evidence.refusal = "private role home is not writable".to_string();
            return Ok(evidence);
        }
        Ok(evidence)
    }

    /// `resolvePreparationTools`: resolve each required tool or report the
    /// first missing one. Never errors: absence is data.
    pub(crate) fn resolve_preparation_tools(
        &self,
        prep: &Preparation,
        deadline: Instant,
    ) -> Result<(Vec<ResolvedTool>, String), String> {
        let mut tools = Vec::new();
        for name in &prep.tools {
            let target = format!("soda-{}", prep.project);
            let resolved = match self.podman(
                &[],
                &[
                    "exec",
                    &target,
                    "/bin/bash",
                    "-c",
                    "command -v \"$0\"",
                    name,
                ],
                deadline,
            ) {
                Ok(out) => single_line(&out, 256),
                Err(_) => None,
            };
            let Some(resolved) = resolved else {
                return Ok((tools, name.clone()));
            };
            if !valid_resolved_tool_path(&resolved) {
                return Ok((tools, name.clone()));
            }
            let observed =
                match self.podman(&[], &["exec", &target, &resolved, "--version"], deadline) {
                    Ok(version) => {
                        let first = version.split(|b| *b == b'\n').next().unwrap_or(&[]);
                        single_line(first, 256)
                    }
                    Err(_) => None,
                };
            let Some(observed) = observed else {
                return Ok((tools, name.clone()));
            };
            tools.push(ResolvedTool {
                name: name.clone(),
                path: resolved,
                version: observed,
            });
        }
        Ok((tools, String::new()))
    }

    /// `recordPreparationTools`: persist tools, missing marker and evidence.
    pub(crate) fn record_preparation_tools(
        &self,
        prep: &Preparation,
        tools: &[ResolvedTool],
        missing: &str,
        verified: &LauncherEvidence,
        deadline: Instant,
    ) -> Result<(), String> {
        // Sorted map keys: id, missing, op, tools, verified; verified sorts
        // groups, login, refusal?, uid.
        let mut body = String::from("{\"id\":");
        body.push_str(&json::quote(&prep.id));
        body.push_str(",\"missing\":");
        body.push_str(&json::quote(missing));
        body.push_str(",\"op\":\"record\",\"tools\":[");
        for (i, tool) in tools.iter().enumerate() {
            if i > 0 {
                body.push(',');
            }
            tool.encode_into(&mut body);
        }
        body.push_str("],\"verified\":{\"groups\":");
        body.push_str(&json::quote(&verified.groups));
        body.push_str(",\"login\":");
        body.push_str(&json::quote(&verified.login));
        if !verified.refusal.is_empty() {
            body.push_str(",\"refusal\":");
            body.push_str(&json::quote(&verified.refusal));
        }
        body.push_str(",\"uid\":");
        body.push_str(&json::quote(&verified.uid));
        body.push_str("}}");
        self.factory_helper(&prep.project, &body, deadline)?;
        Ok(())
    }
}

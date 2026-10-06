// Role execution and preparation source checkout/head confirmation.
use std::time::Instant;

use super::paths::preparation_paths;
use crate::preparation::Preparation;
use crate::project::{Executor, Runtime};

impl<E: Executor> Runtime<E> {
    /// `roleExec`: run a fixed argv as the preparation role.
    pub(crate) fn role_exec(
        &self,
        prep: &Preparation,
        args: &[&str],
        deadline: Instant,
    ) -> Result<Vec<u8>, String> {
        let target = format!("soda-{}", prep.project);
        let mut full: Vec<&str> = Vec::with_capacity(args.len() + 4);
        full.extend_from_slice(&["exec", "--user", &prep.role, &target]);
        full.extend_from_slice(args);
        self.podman(&[], &full, deadline)
    }

    /// `mustRoleExec`: empty output on failure; callers refuse.
    pub(crate) fn must_role_exec(
        &self,
        prep: &Preparation,
        args: &[&str],
        deadline: Instant,
    ) -> Vec<u8> {
        self.role_exec(prep, args, deadline).unwrap_or_default()
    }

    /// `clonePreparationSource`: fresh clone or confirmed existing checkout,
    /// then the private role home.
    pub(crate) fn clone_preparation_source(
        &self,
        prep: &Preparation,
        deadline: Instant,
    ) -> Result<(), String> {
        let (checkout, _, bundle, home) = preparation_paths(&prep.role, &prep.id);
        let git_dir = format!("{checkout}/.git");
        if self
            .role_exec(prep, &["/usr/bin/test", "-d", &git_dir], deadline)
            .is_ok()
        {
            self.confirm_preparation_head(prep, &checkout, deadline)?;
        } else {
            let out = self
                .role_exec(prep, &["/usr/bin/ls", "-A", &checkout], deadline)
                .map_err(|_| "preparation checkout is not inspectable".to_string())?;
            if !single_line_trimmed(&out).is_empty() {
                return Err(
                    "preparation checkout holds unknown partial effects; use a new identity"
                        .to_string(),
                );
            }
            let home_env = format!("HOME={home}");
            self.role_exec(
                prep,
                &[
                    "/usr/bin/env",
                    "-i",
                    "PATH=/usr/bin:/bin",
                    &home_env,
                    "GIT_CONFIG_NOSYSTEM=1",
                    "GIT_CONFIG_GLOBAL=/dev/null",
                    "GIT_NO_REPLACE_OBJECTS=1",
                    "GIT_TERMINAL_PROMPT=0",
                    "/usr/bin/git",
                    "clone",
                    "--template=",
                    "--config",
                    "core.hooksPath=/dev/null",
                    &bundle,
                    &checkout,
                ],
                deadline,
            )
            .map_err(|_| "preparation source clone unconfirmed".to_string())?;
            self.confirm_preparation_head(prep, &checkout, deadline)?;
        }
        self.role_exec(
            prep,
            &["/usr/bin/mkdir", "-m", "700", "-p", &home],
            deadline,
        )
        .map_err(|_| "private role home unconfirmed".to_string())?;
        Ok(())
    }

    /// `confirmPreparationHead`: the checkout is the approved commit.
    fn confirm_preparation_head(
        &self,
        prep: &Preparation,
        checkout: &str,
        deadline: Instant,
    ) -> Result<(), String> {
        let out = self
            .role_exec(
                prep,
                &["/usr/bin/git", "-C", checkout, "rev-parse", "HEAD"],
                deadline,
            )
            .map_err(|_| "preparation source identity unconfirmed".to_string())?;
        if out.len() > 1024 {
            return Err("preparation source identity unconfirmed".to_string());
        }
        if single_line_trimmed(&out) != prep.source_commit {
            return Err("preparation checkout is not the approved commit".to_string());
        }
        Ok(())
    }
}

/// Unicode-blank check mirroring `len(bytes.TrimSpace(out)) != 0`.
fn single_line_trimmed(out: &[u8]) -> String {
    match std::str::from_utf8(out) {
        Ok(s) => s.trim().to_string(),
        Err(_) => String::from(" "),
    }
}

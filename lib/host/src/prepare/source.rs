// Role execution and preparation source checkout/head confirmation.
use std::collections::BTreeSet;
use std::time::Instant;

use super::paths::preparation_paths;
use crate::preparation::{
    self, ContextFile, Preparation, PreparationContext, PrepareContextRead, MAX_CONTEXT_FILES,
    MAX_CONTEXT_FILE_BYTES, MAX_CONTEXT_PATH_BYTES, MAX_CONTEXT_TOTAL_BYTES,
};
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

    /// Read bounded prompt material from exact Git objects in one ready
    /// preparation checkout. The returned bytes are transient and never
    /// enter the durable preparation receipt.
    pub fn read_preparation_context(
        &self,
        input: &PrepareContextRead,
        deadline: Instant,
    ) -> Result<PreparationContext, String> {
        input.validate()?;
        let deadline = Self::cap_by_wire_deadline(deadline, &input.not_after)?;
        let container = self.prepare_container(&input.project, true, deadline)?;
        let state = self
            .inspect_preparation_state(&input.project, &input.id, &container, deadline, None)
            .map_err(|error| error.to_string())?;
        if !state.ready
            || state.stopped
            || state.phase != preparation::PREPARE_READY
            || !preparation::valid_factory_role(&state.role)
            || state.source_commit != input.source_commit
        {
            return Err("preparation context source is not the recorded ready checkout".into());
        }
        if (state.role == preparation::ROLE_CODER && input.approved_base != state.source_commit)
            || (state.role == preparation::ROLE_REVIEWER && input.candidate != state.source_commit)
        {
            return Err("preparation context commits do not match the recorded role source".into());
        }

        let (checkout, _, _, home) = preparation_paths(&state.role, &state.id);
        let head = self.context_git(
            &container,
            &state.role,
            &checkout,
            &home,
            &["rev-parse", "--verify", "HEAD^{commit}"],
            1024,
            deadline,
        )?;
        if std::str::from_utf8(&head).ok().map(str::trim) != Some(input.candidate.as_str()) {
            return Err("preparation context candidate differs from checkout HEAD".into());
        }
        self.context_git(
            &container,
            &state.role,
            &checkout,
            &home,
            &[
                "merge-base",
                "--is-ancestor",
                &input.approved_base,
                &input.candidate,
            ],
            1,
            deadline,
        )?;
        self.context_git(
            &container,
            &state.role,
            &checkout,
            &home,
            &[
                "merge-base",
                "--is-ancestor",
                &input.diff_base,
                &input.candidate,
            ],
            1,
            deadline,
        )?;

        let changed = self.context_git(
            &container,
            &state.role,
            &checkout,
            &home,
            &[
                "diff",
                "--no-ext-diff",
                "--no-textconv",
                "--no-renames",
                "--name-only",
                "-z",
                &input.diff_base,
                &input.candidate,
                "--",
            ],
            MAX_CONTEXT_FILES * MAX_CONTEXT_PATH_BYTES + 1,
            deadline,
        )?;
        if changed.len() > MAX_CONTEXT_FILES * MAX_CONTEXT_PATH_BYTES {
            return Err("candidate context path set exceeds its bound".into());
        }
        let mut selected = BTreeSet::new();
        selected.insert("AGENTS.md".to_string());
        selected.extend(input.paths.iter().cloned());
        for raw_path in changed
            .split(|byte| *byte == 0)
            .filter(|path| !path.is_empty())
        {
            let path = std::str::from_utf8(raw_path)
                .map_err(|_| "candidate context path is not UTF-8 text")?;
            if !preparation::valid_context_path(path) {
                return Err("candidate changes a forbidden context path".into());
            }
            selected.insert(path.to_string());
        }
        let source_paths: Vec<String> = selected.iter().cloned().collect();
        for path in &source_paths {
            add_context_instruction_paths(&mut selected, path);
        }
        if selected.len() > MAX_CONTEXT_FILES {
            return Err("preparation context selects too many files".into());
        }
        let selected: Vec<String> = selected.into_iter().collect();
        for path in &selected {
            if !preparation::valid_context_path(path) {
                return Err("candidate changes a forbidden context path".into());
            }
        }

        let files = self.approved_preparation_files(
            &container,
            &state.id,
            &state.role,
            &state.setup_digest,
            None,
            &state.source_commit,
            deadline,
        )?;
        let setup = files
            .get(preparation::FACTORY_SETUP_ENTRY)
            .cloned()
            .ok_or_else(|| "approved setup entrypoint is missing".to_string())?;
        let check = files
            .get(preparation::FACTORY_CHECK_ENTRY)
            .cloned()
            .ok_or_else(|| "approved check entrypoint is missing".to_string())?;
        if std::str::from_utf8(&setup).is_err() || std::str::from_utf8(&check).is_err() {
            return Err("approved setup context is not UTF-8 text".into());
        }
        let mut total = setup.len().saturating_add(check.len());
        if total > MAX_CONTEXT_TOTAL_BYTES {
            return Err("preparation context exceeds its aggregate bound".into());
        }

        let mut tree_args = vec![
            "ls-tree".to_string(),
            "-r".to_string(),
            "-z".to_string(),
            "--full-tree".to_string(),
            input.approved_base.clone(),
            "--".to_string(),
        ];
        tree_args.extend(selected.iter().map(|path| format!(":(literal){path}")));
        let tree_refs: Vec<&str> = tree_args.iter().map(String::as_str).collect();
        let tree = self.context_git(
            &container,
            &state.role,
            &checkout,
            &home,
            &tree_refs,
            MAX_CONTEXT_FILES * (MAX_CONTEXT_PATH_BYTES + 96) + 1,
            deadline,
        )?;
        let mut existing = BTreeSet::new();
        for record in tree
            .split(|byte| *byte == 0)
            .filter(|record| !record.is_empty())
        {
            let Some(separator) = record.iter().position(|byte| *byte == b'\t') else {
                return Err("approved-base context tree is invalid".into());
            };
            let metadata = std::str::from_utf8(&record[..separator])
                .map_err(|_| "approved-base context tree is invalid")?;
            let path = std::str::from_utf8(&record[separator + 1..])
                .map_err(|_| "approved-base context path is not UTF-8 text")?;
            if selected
                .binary_search_by(|candidate| candidate.as_str().cmp(path))
                .is_err()
            {
                continue;
            }
            let mut fields = metadata.split_ascii_whitespace();
            let mode = fields.next().unwrap_or("");
            let kind = fields.next().unwrap_or("");
            if kind != "blob" || (mode != "100644" && mode != "100755") {
                return Err("approved-base context entry is not a regular file".into());
            }
            existing.insert(path.to_string());
        }
        let mut context_files = Vec::with_capacity(existing.len());
        for path in &selected {
            if !existing.contains(path) {
                continue;
            }
            let remaining = MAX_CONTEXT_TOTAL_BYTES.saturating_sub(total);
            let cap = MAX_CONTEXT_FILE_BYTES.min(remaining);
            let expression = format!("{}:{path}", input.approved_base);
            let content = self.context_git(
                &container,
                &state.role,
                &checkout,
                &home,
                &["cat-file", "blob", expression.as_str()],
                cap.saturating_add(1),
                deadline,
            )?;
            if content.len() > cap {
                return Err("approved-base context file exceeds its bound".into());
            }
            if content.contains(&0) || std::str::from_utf8(&content).is_err() {
                return Err("approved-base context file is not UTF-8 text".into());
            }
            total += content.len();
            context_files.push(ContextFile {
                path: path.clone(),
                content,
            });
        }

        let diff_args = vec![
            "diff".to_string(),
            "--no-ext-diff".to_string(),
            "--no-textconv".to_string(),
            "--no-renames".to_string(),
            input.diff_base.clone(),
            input.candidate.clone(),
        ];
        let diff_refs: Vec<&str> = diff_args.iter().map(String::as_str).collect();
        let remaining = MAX_CONTEXT_TOTAL_BYTES.saturating_sub(total);
        let diff = self.context_git(
            &container,
            &state.role,
            &checkout,
            &home,
            &diff_refs,
            remaining.saturating_add(1),
            deadline,
        )?;
        if diff.len() > remaining {
            return Err("candidate context diff exceeds its aggregate bound".into());
        }
        if diff
            .split(|byte| *byte == b'\n')
            .any(|line| line.starts_with(b"Binary files "))
        {
            return Err("candidate context diff contains a binary file".into());
        }
        if diff.contains(&0) || std::str::from_utf8(&diff).is_err() {
            return Err("candidate context diff is not UTF-8 text".into());
        }

        Ok(PreparationContext {
            project: input.project.clone(),
            id: input.id.clone(),
            role: state.role,
            source_commit: state.source_commit,
            approved_base: input.approved_base.clone(),
            diff_base: input.diff_base.clone(),
            candidate: input.candidate.clone(),
            setup,
            check,
            files: context_files,
            diff,
        })
    }

    fn context_git(
        &self,
        container: &str,
        role: &str,
        checkout: &str,
        home: &str,
        git_args: &[&str],
        stdout_limit: usize,
        deadline: Instant,
    ) -> Result<Vec<u8>, String> {
        let mut args = vec![
            "exec".to_string(),
            "--user".to_string(),
            role.to_string(),
            container.to_string(),
            "/usr/bin/env".to_string(),
            "-i".to_string(),
            "PATH=/usr/bin:/bin".to_string(),
            "LC_ALL=C".to_string(),
            format!("HOME={home}"),
            "GIT_CONFIG_NOSYSTEM=1".to_string(),
            "GIT_CONFIG_GLOBAL=/dev/null".to_string(),
            "GIT_NO_REPLACE_OBJECTS=1".to_string(),
            "GIT_TERMINAL_PROMPT=0".to_string(),
            "GIT_OPTIONAL_LOCKS=0".to_string(),
            "/usr/bin/git".to_string(),
            "-c".to_string(),
            "core.hooksPath=/dev/null".to_string(),
            "-c".to_string(),
            "core.fsmonitor=false".to_string(),
            "-C".to_string(),
            checkout.to_string(),
        ];
        args.extend(git_args.iter().map(|arg| (*arg).to_string()));
        let refs: Vec<&str> = args.iter().map(String::as_str).collect();
        self.exec
            .run_bounded(&[], "/usr/bin/podman", &refs, deadline, stdout_limit, 1024)
    }
}

fn add_context_instruction_paths(selected: &mut BTreeSet<String>, path: &str) {
    selected.insert("AGENTS.md".to_string());
    let parts: Vec<&str> = path.split('/').collect();
    let mut prefix = String::new();
    for part in parts.iter().take(parts.len().saturating_sub(1)) {
        if !prefix.is_empty() {
            prefix.push('/');
        }
        prefix.push_str(part);
        selected.insert(format!("{prefix}/AGENTS.md"));
    }
}

/// Unicode-blank check mirroring `len(bytes.TrimSpace(out)) != 0`.
fn single_line_trimmed(out: &[u8]) -> String {
    match std::str::from_utf8(out) {
        Ok(s) => s.trim().to_string(),
        Err(_) => String::from(" "),
    }
}

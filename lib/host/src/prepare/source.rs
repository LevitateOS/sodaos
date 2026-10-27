// Role execution and preparation source checkout/head confirmation.
use std::collections::BTreeSet;
use std::time::Instant;

use super::paths::preparation_paths;
use crate::preparation::{
    self, ContextFile, Preparation, PreparationContext, PrepareContextRead, PrepareState,
    MAX_CONTEXT_FILES, MAX_CONTEXT_FILE_BYTES, MAX_CONTEXT_PATH_BYTES, MAX_CONTEXT_TOTAL_BYTES,
};
use crate::project::{BoundedError, Executor, Runtime};

/// Typed failure for one bounded preparation-context read.
///
/// Explicit oversize/bounds refusals stay distinct from every other
/// failure at the producing site, so the route layer can signal them
/// distinctly (HTTP 413) without matching error text. Anything else —
/// identity, ancestry, path/UTF-8 shape, exec — fails closed elsewhere.
#[derive(Debug, Clone, PartialEq, Eq)]
pub enum ContextReadError {
    /// An explicit bounds refusal: the selection cannot fit the
    /// file-count, per-file or aggregate context bounds. Carries the
    /// fixed diagnostic; the wire carries only the typed signal.
    Bounds(&'static str),
    /// Every other failure. Carries the exact executor diagnostic for
    /// the operator journal; the wire stays a generic 500.
    Failed(String),
}

impl From<String> for ContextReadError {
    fn from(message: String) -> Self {
        ContextReadError::Failed(message)
    }
}

impl From<&str> for ContextReadError {
    fn from(message: &str) -> Self {
        ContextReadError::Failed(message.to_string())
    }
}

/// Admitted context source: the recorded ready checkout plus its
/// derived paths. Every later phase reads from this exact checkout.
struct AdmittedPreparationContext {
    container: String,
    state: PrepareState,
    checkout: String,
    home: String,
    deadline: Instant,
}

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
                &[
                    "/usr/bin/git",
                    "-c",
                    &format!("safe.directory={checkout}"),
                    "-C",
                    checkout,
                    "rev-parse",
                    "HEAD",
                ],
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

    /// Admit the context source: the container must hold the recorded ready
    /// checkout for the requested role and source commit.
    fn admit_preparation_context(
        &self,
        input: &PrepareContextRead,
        deadline: Instant,
    ) -> Result<AdmittedPreparationContext, String> {
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
        Ok(AdmittedPreparationContext {
            container,
            state,
            checkout,
            home,
            deadline,
        })
    }

    /// Confirm the checkout HEAD and both ancestry edges before reading.
    fn confirm_context_commits(
        &self,
        admitted: &AdmittedPreparationContext,
        input: &PrepareContextRead,
    ) -> Result<(), String> {
        let head = self.context_git(
            &admitted.container,
            &admitted.state.role,
            &admitted.checkout,
            &admitted.home,
            &["rev-parse", "--verify", "HEAD^{commit}"],
            1024,
            admitted.deadline,
        )?;
        if std::str::from_utf8(&head).ok().map(str::trim) != Some(input.candidate.as_str()) {
            return Err("preparation context candidate differs from checkout HEAD".into());
        }
        self.context_git(
            &admitted.container,
            &admitted.state.role,
            &admitted.checkout,
            &admitted.home,
            &[
                "merge-base",
                "--is-ancestor",
                &input.approved_base,
                &input.candidate,
            ],
            1,
            admitted.deadline,
        )?;
        self.context_git(
            &admitted.container,
            &admitted.state.role,
            &admitted.checkout,
            &admitted.home,
            &[
                "merge-base",
                "--is-ancestor",
                &input.diff_base,
                &input.candidate,
            ],
            1,
            admitted.deadline,
        )?;
        Ok(())
    }

    /// Select the bounded context path set: requested paths plus every
    /// changed path, expanded with instruction paths and re-validated.
    fn select_context_paths(
        &self,
        admitted: &AdmittedPreparationContext,
        input: &PrepareContextRead,
    ) -> Result<Vec<String>, ContextReadError> {
        let changed = self.context_git_bounded(
            &admitted.container,
            &admitted.state.role,
            &admitted.checkout,
            &admitted.home,
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
            admitted.deadline,
            "candidate context path set exceeds its bound",
        )?;
        if changed.len() > MAX_CONTEXT_FILES * MAX_CONTEXT_PATH_BYTES {
            return Err(ContextReadError::Bounds(
                "candidate context path set exceeds its bound",
            ));
        }
        let mut selected = BTreeSet::new();
        selected.insert("AGENTS.md".to_string());
        selected.extend(input.paths.iter().cloned());
        let mut changed_paths = BTreeSet::new();
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
            changed_paths.insert(path.to_string());
        }
        let source_paths: Vec<String> = selected.iter().cloned().collect();
        for path in &source_paths {
            add_context_instruction_paths(&mut selected, path);
        }
        let selected: Vec<String> = selected.into_iter().collect();
        for path in &selected {
            if !preparation::valid_context_path(path) {
                return Err("candidate changes a forbidden context path".into());
            }
        }
        // Issue prose may resemble a repository path. Only existing approved
        // files consume the file budget, while every actual changed path
        // remains selected, including additions absent from the approved base.
        let tree = self.list_approved_tree(admitted, input, &selected)?;
        let existing = Self::parse_approved_tree(&tree, &selected)?;
        let selected: Vec<String> = selected
            .into_iter()
            .filter(|path| existing.contains(path) || changed_paths.contains(path))
            .collect();
        if selected.len() > MAX_CONTEXT_FILES {
            return Err(ContextReadError::Bounds(
                "preparation context selects too many files",
            ));
        }
        Ok(selected)
    }

    /// Load the approved setup and check entrypoints and seed the
    /// aggregate byte budget.
    fn load_approved_setup(
        &self,
        admitted: &AdmittedPreparationContext,
    ) -> Result<(Vec<u8>, Vec<u8>, usize), ContextReadError> {
        let files = self.approved_preparation_files(
            &admitted.container,
            &admitted.state.id,
            &admitted.state.role,
            &admitted.state.setup_digest,
            None,
            &admitted.state.source_commit,
            admitted.deadline,
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
        let total = setup.len().saturating_add(check.len());
        if total > MAX_CONTEXT_TOTAL_BYTES {
            return Err(ContextReadError::Bounds(
                "preparation context exceeds its aggregate bound",
            ));
        }
        Ok((setup, check, total))
    }

    /// List the selected paths that exist as regular files at the
    /// approved base.
    fn list_approved_tree(
        &self,
        admitted: &AdmittedPreparationContext,
        input: &PrepareContextRead,
        selected: &[String],
    ) -> Result<Vec<u8>, String> {
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
        self.context_git(
            &admitted.container,
            &admitted.state.role,
            &admitted.checkout,
            &admitted.home,
            &tree_refs,
            MAX_CONTEXT_FILES * (MAX_CONTEXT_PATH_BYTES + 96) + 1,
            admitted.deadline,
        )
    }

    /// Keep only selected regular-file blobs from the approved-base tree.
    fn parse_approved_tree(tree: &[u8], selected: &[String]) -> Result<BTreeSet<String>, String> {
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
        Ok(existing)
    }

    /// Read every selected file that exists at the approved base into the
    /// aggregate byte budget. Each blob reads through one bounded call
    /// (cap+1 detects oversize without truncation); the Go caller chunks
    /// large selections across reads so each response stays bounded while
    /// the full file bytes return for digest verification.
    fn collect_approved_files(
        &self,
        admitted: &AdmittedPreparationContext,
        input: &PrepareContextRead,
        selected: &[String],
        total: usize,
    ) -> Result<(Vec<ContextFile>, usize), ContextReadError> {
        let tree = self.list_approved_tree(admitted, input, selected)?;
        let existing = Self::parse_approved_tree(&tree, selected)?;
        let mut total = total;
        let mut context_files = Vec::with_capacity(existing.len());
        for path in selected {
            if !existing.contains(path) {
                continue;
            }
            let remaining = MAX_CONTEXT_TOTAL_BYTES.saturating_sub(total);
            let cap = MAX_CONTEXT_FILE_BYTES.min(remaining);
            let expression = format!("{}:{path}", input.approved_base);
            let content = self.context_git_bounded(
                &admitted.container,
                &admitted.state.role,
                &admitted.checkout,
                &admitted.home,
                &["cat-file", "blob", expression.as_str()],
                cap.saturating_add(1),
                admitted.deadline,
                "approved-base context file exceeds its bound",
            )?;
            if content.len() > cap {
                return Err(ContextReadError::Bounds(
                    "approved-base context file exceeds its bound",
                ));
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
        Ok((context_files, total))
    }

    /// Read the bounded candidate diff against the remaining budget.
    fn load_candidate_diff(
        &self,
        admitted: &AdmittedPreparationContext,
        input: &PrepareContextRead,
        total: usize,
    ) -> Result<Vec<u8>, ContextReadError> {
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
        let diff = self.context_git_bounded(
            &admitted.container,
            &admitted.state.role,
            &admitted.checkout,
            &admitted.home,
            &diff_refs,
            remaining.saturating_add(1),
            admitted.deadline,
            "candidate context diff exceeds its aggregate bound",
        )?;
        if diff.len() > remaining {
            return Err(ContextReadError::Bounds(
                "candidate context diff exceeds its aggregate bound",
            ));
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
        Ok(diff)
    }

    /// Read bounded prompt material from exact Git objects in one ready
    /// preparation checkout. The returned bytes are transient and never
    /// enter the durable preparation receipt.
    pub fn read_preparation_context(
        &self,
        input: &PrepareContextRead,
        deadline: Instant,
    ) -> Result<PreparationContext, ContextReadError> {
        let admitted = self.admit_preparation_context(input, deadline)?;
        self.confirm_context_commits(&admitted, input)?;
        let selected = self.select_context_paths(&admitted, input)?;
        let (setup, check, total) = self.load_approved_setup(&admitted)?;
        let (context_files, total) =
            self.collect_approved_files(&admitted, input, &selected, total)?;
        let diff = self.load_candidate_diff(&admitted, input, total)?;
        Ok(PreparationContext {
            project: input.project.clone(),
            id: input.id.clone(),
            role: admitted.state.role,
            source_commit: admitted.state.source_commit,
            approved_base: input.approved_base.clone(),
            diff_base: input.diff_base.clone(),
            candidate: input.candidate.clone(),
            setup,
            check,
            files: context_files,
            diff,
        })
    }

    pub(super) fn context_git(
        &self,
        container: &str,
        role: &str,
        checkout: &str,
        home: &str,
        git_args: &[&str],
        stdout_limit: usize,
        deadline: Instant,
    ) -> Result<Vec<u8>, String> {
        let args = context_git_argv(container, role, checkout, home, git_args);
        let refs: Vec<&str> = args.iter().map(String::as_str).collect();
        self.exec
            .run_bounded(&[], "/usr/bin/podman", &refs, deadline, stdout_limit, 1024)
    }

    /// Typed bounded Git read for the producing-site bound checks: a
    /// genuine stdout-ceiling overflow from the real executor surfaces
    /// as `ContextReadError::Bounds` (the route's typed 413) carrying the
    /// site's bound message. Every other failure — stderr ceiling,
    /// timeout/cancellation, cleanup, wait, exit status — keeps the exact
    /// executor diagnostic and fails closed as `Failed`. No error text is
    /// matched.
    pub(super) fn context_git_bounded(
        &self,
        container: &str,
        role: &str,
        checkout: &str,
        home: &str,
        git_args: &[&str],
        stdout_limit: usize,
        deadline: Instant,
        overflow: &'static str,
    ) -> Result<Vec<u8>, ContextReadError> {
        let args = context_git_argv(container, role, checkout, home, git_args);
        let refs: Vec<&str> = args.iter().map(String::as_str).collect();
        self.exec
            .run_bounded_typed(&[], "/usr/bin/podman", &refs, deadline, stdout_limit, 1024)
            .map_err(|error| match error {
                BoundedError::StdoutLimit => ContextReadError::Bounds(overflow),
                BoundedError::Failed(message) => ContextReadError::Failed(message),
            })
    }
}

/// Shared `podman exec` + `git` argv for the bounded preparation reads.
/// One builder so the String and typed paths issue byte-identical
/// commands; only the error channel differs.
fn context_git_argv(
    container: &str,
    role: &str,
    checkout: &str,
    home: &str,
    git_args: &[&str],
) -> Vec<String> {
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
        format!("safe.directory={checkout}"),
        "-c".to_string(),
        "core.hooksPath=/dev/null".to_string(),
        "-c".to_string(),
        "core.fsmonitor=false".to_string(),
        "-C".to_string(),
        checkout.to_string(),
    ];
    args.extend(git_args.iter().map(|arg| (*arg).to_string()));
    args
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

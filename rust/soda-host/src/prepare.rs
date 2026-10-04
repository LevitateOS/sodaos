//! Factory preparation operations: the host side of the fixed
//! `project-factory-roles` helper exchange. Ports `prepare.go` and the
//! candidate-preparation half of `factory_candidate.go`
//! (`PrepareCandidate`; run-receipt inspection stays for the factory port).
//! Helper payloads are built byte-for-byte like the Go map marshals.

use std::collections::HashMap;
use std::time::Instant;

use crate::domain;
use crate::json::{self, Kind, Spec};
use crate::preparation::{
    self, FactoryCandidate, HoldState, Preparation, Prepare, PrepareHold, PrepareInspect,
    PrepareState, PrepareStop, ResolvedTool,
};
use crate::project::{Executor, Runtime};

const FACTORY_INSPECT_LIMIT: usize = 262144;

/// `path.IsAbs`: a leading slash.
fn path_is_abs(p: &str) -> bool {
    p.starts_with('/')
}

/// `path.Clean`: lexical cleanup, byte-for-byte with the Go algorithm.
pub fn path_clean(path: &str) -> String {
    let b = path.as_bytes();
    if b.is_empty() {
        return ".".to_string();
    }
    let rooted = b[0] == b'/';
    let n = b.len();
    let mut out: Vec<u8> = Vec::with_capacity(n);
    let mut r = 0usize;
    let mut dotdot = 0usize;
    if rooted {
        out.push(b'/');
        r = 1;
        dotdot = 1;
    }
    while r < n {
        if b[r] == b'/' || (b[r] == b'.' && (r + 1 == n || b[r + 1] == b'/')) {
            r += 1;
        } else if b[r] == b'.' && r + 1 < n && b[r + 1] == b'.' && (r + 2 == n || b[r + 2] == b'/')
        {
            r += 2;
            if out.len() > dotdot {
                // Back over the trailing element to its slash (Go's w-- loop
                // inspects the first excluded byte, not the last kept one).
                let mut w = out.len() - 1;
                while w > dotdot && out[w] != b'/' {
                    w -= 1;
                }
                out.truncate(w);
            } else if !rooted {
                if !out.is_empty() {
                    out.push(b'/');
                }
                out.push(b'.');
                out.push(b'.');
                dotdot = out.len();
            }
        } else {
            if (rooted && out.len() != 1) || (!rooted && !out.is_empty()) {
                out.push(b'/');
            }
            while r < n && b[r] != b'/' {
                out.push(b[r]);
                r += 1;
            }
        }
    }
    if out.is_empty() {
        return ".".to_string();
    }
    String::from_utf8(out).unwrap_or_else(|_| ".".to_string())
}

/// `path.Join`: leading empty elements skipped, the rest slash-joined
/// then cleaned; empty when every element is empty.
pub fn path_join(parts: &[&str]) -> String {
    let Some(first) = parts.iter().position(|p| !p.is_empty()) else {
        return String::new();
    };
    path_clean(&parts[first..].join("/"))
}

/// `prepareIDMap`: every mapping keeps host root out of the container and
/// some mapping shifts container root onto a usable range.
pub fn prepare_id_map(values: &[String]) -> bool {
    if values.is_empty() {
        return false;
    }
    let mut shifted = false;
    for value in values {
        let parts: Vec<&str> = value.split(':').collect();
        if parts.len() != 3 {
            return false;
        }
        let container: u32 = match parts[0].parse::<u32>() {
            Ok(v) if v.to_string() == parts[0] => v,
            _ => return false,
        };
        let base: u32 = match parts[1].parse::<u32>() {
            Ok(v) if v.to_string() == parts[1] && v > 0 => v,
            _ => return false,
        };
        let size: u32 = match parts[2].parse::<u32>() {
            Ok(v) if v.to_string() == parts[2] => v,
            _ => return false,
        };
        if container == 0 && size >= 65536 && u64::from(base) + u64::from(size) <= 4294967295 {
            shifted = true;
        }
    }
    shifted
}

/// `preparationPaths`: fixed role/checkout/snapshot/bundle/home layout.
pub fn preparation_paths(role: &str, id: &str) -> (String, String, String, String) {
    let checkout = format!("/home/{role}/checkouts/{id}");
    let snapshot = format!("{}/{id}/snapshot", preparation::FACTORY_PREPARATIONS_DIR);
    let bundle = format!("{snapshot}/source.bundle");
    let home = format!("{checkout}/.soda-home");
    (checkout, snapshot, bundle, home)
}

/// `singleLine`: one trimmed line, bounded, with no inner newline.
pub fn single_line(out: &[u8], limit: usize) -> Option<String> {
    if out.is_empty() || out.len() > limit {
        return None;
    }
    let line = std::str::from_utf8(out).ok()?.trim().to_string();
    if line.is_empty() || line.contains('\n') {
        return None;
    }
    Some(line)
}

/// `validResolvedToolPath`: absolute, clean, under a fixed tool prefix.
pub fn valid_resolved_tool_path(value: &str) -> bool {
    if value.is_empty() || value.len() > 256 || !path_is_abs(value) || path_clean(value) != value {
        return false;
    }
    value.starts_with("/usr/bin/") || value.starts_with("/usr/local/bin/")
}

const APPROVE_RESPONSE_SPECS: &[Spec] = &[
    Spec {
        name: "approved",
        kind: Kind::Str,
    },
    Spec {
        name: "repeated",
        kind: Kind::Bool,
    },
    Spec {
        name: "checkout",
        kind: Kind::Str,
    },
    Spec {
        name: "credential_file",
        kind: Kind::Str,
    },
];

const STOP_RESPONSE_SPECS: &[Spec] = &[
    Spec {
        name: "stopped",
        kind: Kind::Str,
    },
    Spec {
        name: "retirement",
        kind: Kind::Str,
    },
    Spec {
        name: "known",
        kind: Kind::Bool,
    },
];

const HOLD_RESPONSE_SPECS: &[Spec] = &[Spec {
    name: "hold",
    kind: Kind::Object {
        go_type: "project.HoldState",
        struct_name: "HoldState",
        specs: preparation::HOLD_STATE_SPECS,
    },
}];

const INSPECTION_HOLD_SPECS: &[Spec] = &[
    Spec {
        name: "active",
        kind: Kind::Bool,
    },
    Spec {
        name: "revision",
        kind: Kind::I64,
    },
];

const INSPECTION_VERIFIED_SPECS: &[Spec] = &[
    Spec {
        name: "uid",
        kind: Kind::Str,
    },
    Spec {
        name: "login",
        kind: Kind::Str,
    },
    Spec {
        name: "groups",
        kind: Kind::Str,
    },
    Spec {
        name: "refusal",
        kind: Kind::Str,
    },
];

const HELPER_INSPECTION_SPECS: &[Spec] = &[
    Spec {
        name: "hold",
        kind: Kind::Object {
            go_type: "struct",
            struct_name: "struct",
            specs: INSPECTION_HOLD_SPECS,
        },
    },
    Spec {
        name: "known",
        kind: Kind::Bool,
    },
    Spec {
        name: "phase",
        kind: Kind::Str,
    },
    Spec {
        name: "role",
        kind: Kind::Str,
    },
    Spec {
        name: "setup_digest",
        kind: Kind::Str,
    },
    Spec {
        name: "source_commit",
        kind: Kind::Str,
    },
    Spec {
        name: "tools",
        kind: Kind::StructList {
            go_type: "[]project.ResolvedTool",
            struct_name: "ResolvedTool",
            specs: preparation::RESOLVED_TOOL_SPECS,
        },
    },
    Spec {
        name: "verified",
        kind: Kind::Object {
            go_type: "struct",
            struct_name: "struct",
            specs: INSPECTION_VERIFIED_SPECS,
        },
    },
    Spec {
        name: "missing",
        kind: Kind::Str,
    },
    Spec {
        name: "stopped",
        kind: Kind::Bool,
    },
    Spec {
        name: "ready",
        kind: Kind::Bool,
    },
    Spec {
        name: "setup_exit",
        kind: Kind::OptInt,
    },
    Spec {
        name: "check_exit",
        kind: Kind::OptInt,
    },
    Spec {
        name: "setup_log",
        kind: Kind::Str,
    },
    Spec {
        name: "check_log",
        kind: Kind::Str,
    },
];

const SAVED_REQUEST_SPECS: &[Spec] = &[
    Spec {
        name: "id",
        kind: Kind::Str,
    },
    Spec {
        name: "role",
        kind: Kind::Str,
    },
    Spec {
        name: "setup_digest",
        kind: Kind::Str,
    },
    Spec {
        name: "source_commit",
        kind: Kind::Str,
    },
    Spec {
        name: "credential",
        kind: Kind::Str,
    },
];

/// Launcher evidence: observed role identity plus any refusal reason.
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct LauncherEvidence {
    pub uid: String,
    pub login: String,
    pub groups: String,
    pub refusal: String,
}

fn quote_bytes_base64(out: &mut String, bytes: &[u8]) {
    out.push_str(&json::quote(&crate::ssh::b64_encode(bytes)));
}

/// Sorted `files` object: names sorted, values padded base64.
fn encode_files_object(out: &mut String, files: &HashMap<String, Vec<u8>>) {
    let mut names: Vec<&String> = files.keys().collect();
    names.sort();
    out.push('{');
    for (i, name) in names.iter().enumerate() {
        if i > 0 {
            out.push(',');
        }
        out.push_str(&json::quote(name));
        out.push(':');
        quote_bytes_base64(out, &files[*name]);
    }
    out.push('}');
}

impl<E: Executor> Runtime<E> {
    /// `factoryHelper`: bounded JSON exchange with the in-container helper.
    /// Failures pass through raw; oversized responses are refused.
    fn factory_helper(&self, id: &str, body: &str, deadline: Instant) -> Result<Vec<u8>, String> {
        let target = format!("soda-{id}");
        let args = [
            "exec",
            "--interactive",
            &target,
            preparation::FACTORY_HELPER,
        ];
        let out = self.podman(body.as_bytes(), &args, deadline)?;
        if out.len() > FACTORY_INSPECT_LIMIT {
            return Err("factory helper response exceeds the bounded size".to_string());
        }
        Ok(out)
    }

    /// `prepareContainer`: bind the exact isolated container identity.
    pub fn prepare_container(
        &self,
        id: &str,
        require_running: bool,
        deadline: Instant,
    ) -> Result<String, String> {
        if !domain::valid_id(id) {
            return Err("invalid project".to_string());
        }
        let target = format!("soda-{id}");
        let data = self
            .podman(
                &[],
                &[
                    "--remote=false",
                    "inspect",
                    "--format",
                    crate::project::PROJECT_INSPECT_FORMAT,
                    &target,
                ],
                deadline,
            )
            .map_err(|_| "preparation target unavailable".to_string())?;
        if data.len() > 4096 {
            return Err("preparation target unavailable".to_string());
        }
        let v = json::decode_strict(&data).map_err(|_| "invalid preparation target".to_string())?;
        let m = json::bind_root(
            &v,
            "projectInspection",
            crate::project::INSPECTION_SPECS,
            false,
        )
        .map_err(|_| "invalid preparation target".to_string())?;
        let cid = m.take_string("id");
        let running = m.take_bool("running");
        let project = m.take_string("project");
        let owner = m.take_string("owner");
        let privileged = m.take_bool("privileged");
        let userns = m.take_string("userns");
        let mappings = m.take_map("mappings");
        let uid_map = mappings.take_str_list("UidMap");
        let gid_map = mappings.take_str_list("GidMap");
        let owner_num = json::parse_go_int64(&owner).unwrap_or(0);
        if owner_num <= 0
            || (require_running && !running)
            || !domain::valid_container_id(&cid)
            || project != id
            || privileged
            || userns != "private"
            || !prepare_id_map(&uid_map)
            || !prepare_id_map(&gid_map)
        {
            return Err("preparation target not ready or isolated".to_string());
        }
        Ok(cid)
    }

    /// `inspectPreparationState`: authoritative observed state for one identity.
    pub fn inspect_preparation_state(
        &self,
        project: &str,
        id: &str,
        container: &str,
        deadline: Instant,
    ) -> Result<PrepareState, String> {
        let mut body = String::from("{\"id\":");
        body.push_str(&json::quote(id));
        body.push_str(",\"op\":\"inspect\"}");
        let raw = self.factory_helper(project, &body, deadline)?;
        let mut state = map_preparation_state(container, &raw)?;
        state.id = id.to_string();
        state.project = project.to_string();
        Ok(state)
    }

    /// `Prepare`: approve, clone, verify, resolve tools, record, start.
    pub fn prepare(&self, input: &Prepare, deadline: Instant) -> Result<PrepareState, String> {
        input.validate()?;
        let prep = &input.preparation;
        let container = self.prepare_container(&prep.project, true, deadline)?;
        self.factory_helper(&prep.project, "{\"op\":\"ensure\"}", deadline)?;
        if let Err(err) = self.approve_preparation(prep, &input.setup, deadline) {
            if let Ok(stopped) =
                self.inspect_preparation_state(&prep.project, &prep.id, &container, deadline)
            {
                if stopped.stopped {
                    return Ok(stopped);
                }
            }
            return Err(err);
        }
        let state =
            self.inspect_preparation_state(&prep.project, &prep.id, &container, deadline)?;
        if state.stopped || state.ready || state.phase == preparation::PREPARE_FAILED {
            return Ok(state);
        }
        self.clone_preparation_source(prep, deadline)?;
        let verified = self.verify_launcher_environment(prep, deadline)?;
        let (tools, missing) = self.resolve_preparation_tools(prep, deadline)?;
        self.record_preparation_tools(prep, &tools, &missing, &verified, deadline)?;
        if !missing.is_empty() || !verified.refusal.is_empty() {
            return self.inspect_preparation_state(&prep.project, &prep.id, &container, deadline);
        }
        let mut body = String::from("{\"id\":");
        body.push_str(&json::quote(&prep.id));
        body.push_str(",\"op\":\"start\"}");
        if let Err(err) = self.factory_helper(&prep.project, &body, deadline) {
            // Go returns the re-observed state alongside the error; every
            // caller drops the state on error, so only the error crosses.
            let _ = self.inspect_preparation_state(&prep.project, &prep.id, &container, deadline);
            return Err(err);
        }
        self.inspect_preparation_state(&prep.project, &prep.id, &container, deadline)
    }

    /// `approvePreparation`: record approved inputs under the identity.
    fn approve_preparation(
        &self,
        prep: &Preparation,
        setup: &preparation::ApprovedSetup,
        deadline: Instant,
    ) -> Result<(), String> {
        let (checkout, _, _, _) = preparation_paths(&prep.role, &prep.id);
        // Sorted map keys: bundle, credential, files, id, op, role,
        // setup_digest, source_commit.
        let mut body = String::from("{\"bundle\":");
        quote_bytes_base64(&mut body, &setup.bundle);
        body.push_str(",\"credential\":");
        body.push_str(&json::quote(&prep.credential));
        body.push_str(",\"files\":");
        encode_files_object(&mut body, &setup.files);
        body.push_str(",\"id\":");
        body.push_str(&json::quote(&prep.id));
        body.push_str(",\"op\":\"approve\",\"role\":");
        body.push_str(&json::quote(&prep.role));
        body.push_str(",\"setup_digest\":");
        body.push_str(&json::quote(&prep.setup_digest));
        body.push_str(",\"source_commit\":");
        body.push_str(&json::quote(&prep.source_commit));
        body.push('}');
        let raw = self.factory_helper(&prep.project, &body, deadline)?;
        const ERR: &str = "preparation approval unconfirmed";
        let v = json::decode_strict(&raw).map_err(|_| ERR.to_string())?;
        let m = json::bind_root(&v, "struct", APPROVE_RESPONSE_SPECS, false)
            .map_err(|_| ERR.to_string())?;
        if m.take_string("approved") != prep.id {
            return Err(ERR.to_string());
        }
        if m.take_bool("repeated") {
            return Ok(());
        }
        let want_credential = if prep.credential.is_empty() {
            String::new()
        } else {
            format!(
                "{}/{}/{}",
                preparation::FACTORY_CREDENTIALS_DIR,
                prep.role,
                prep.credential
            )
        };
        if m.take_string("checkout") != checkout
            || m.take_string("credential_file") != want_credential
        {
            return Err("preparation approval resolved unexpected paths".to_string());
        }
        Ok(())
    }

    /// `roleExec`: run a fixed argv as the preparation role.
    fn role_exec(
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
    fn must_role_exec(&self, prep: &Preparation, args: &[&str], deadline: Instant) -> Vec<u8> {
        self.role_exec(prep, args, deadline).unwrap_or_default()
    }

    /// `clonePreparationSource`: fresh clone or confirmed existing checkout,
    /// then the private role home.
    fn clone_preparation_source(
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

    /// `verifyLauncherEnvironment`: observe the role; refusals are data.
    fn verify_launcher_environment(
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
    fn resolve_preparation_tools(
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
    fn record_preparation_tools(
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

    /// `InspectPreparation`: authoritative state, never mutating.
    pub fn inspect_preparation(
        &self,
        input: &PrepareInspect,
        deadline: Instant,
    ) -> Result<PrepareState, String> {
        input.validate()?;
        let container = self.prepare_container(&input.project, false, deadline)?;
        self.inspect_preparation_state(&input.project, &input.id, &container, deadline)
    }

    /// `StopPreparation`: persist the stop tombstone, report the state.
    pub fn stop_preparation(
        &self,
        input: &PrepareStop,
        deadline: Instant,
    ) -> Result<PrepareState, String> {
        input.validate()?;
        let container = self.prepare_container(&input.project, false, deadline)?;
        let mut body = String::from("{\"id\":");
        body.push_str(&json::quote(&input.id));
        body.push_str(",\"op\":\"stop\"}");
        let raw = self.factory_helper(&input.project, &body, deadline)?;
        const ERR: &str = "preparation stop unconfirmed";
        let v = json::decode_strict(&raw).map_err(|_| ERR.to_string())?;
        let m = json::bind_root(&v, "struct", STOP_RESPONSE_SPECS, false)
            .map_err(|_| ERR.to_string())?;
        let retirement = m.take_string("retirement");
        if m.take_string("stopped") != input.id
            || (retirement != "confirmed" && retirement != "uncertain")
        {
            return Err(ERR.to_string());
        }
        let mut state =
            self.inspect_preparation_state(&input.project, &input.id, &container, deadline)?;
        state.retirement = retirement;
        Ok(state)
    }

    /// `HoldPreparation`: enforce the maintenance hold marker natively.
    pub fn hold_preparation(
        &self,
        input: &PrepareHold,
        deadline: Instant,
    ) -> Result<HoldState, String> {
        input.validate()?;
        self.prepare_container(&input.project, true, deadline)?;
        let op = if input.hold { "hold" } else { "release" };
        let body = format!("{{\"op\":\"{op}\",\"revision\":{}}}", input.revision);
        let raw = self.factory_helper(&input.project, &body, deadline)?;
        let v =
            json::decode_strict(&raw).map_err(|_| "maintenance hold unconfirmed".to_string())?;
        let m = json::bind_root(&v, "struct", HOLD_RESPONSE_SPECS, false)
            .map_err(|_| "maintenance hold unconfirmed".to_string())?;
        let hold = HoldState::from_map(&m.take_map("hold"));
        if hold.active != input.hold {
            return Err("maintenance hold outcome not confirmed".to_string());
        }
        Ok(hold)
    }

    /// `PrepareCandidate`: fresh reviewer preparation reusing a ready
    /// preparation's protected approved setup.
    pub fn prepare_candidate(
        &self,
        input: &FactoryCandidate,
        deadline: Instant,
    ) -> Result<PrepareState, String> {
        input.validate()?;
        let container = self.prepare_container(&input.preparation.project, true, deadline)?;
        let source = self.inspect_preparation_state(
            &input.preparation.project,
            &input.source_preparation,
            &container,
            deadline,
        )?;
        if !source.ready
            || source.stopped
            || source.role != input.preparation.role
            || source.setup_digest != input.preparation.setup_digest
        {
            return Err(
                "candidate source preparation is not ready for this role and setup".to_string(),
            );
        }
        let files =
            self.candidate_approved_files(&container, input, &source.source_commit, deadline)?;
        self.prepare(
            &Prepare {
                preparation: input.preparation.clone(),
                setup: preparation::ApprovedSetup {
                    files,
                    bundle: input.bundle.clone(),
                },
            },
            deadline,
        )
    }

    /// `candidateProtectedFile`: one root-owned read-only regular file.
    fn candidate_protected_file(
        &self,
        container: &str,
        filename: &str,
        limit: usize,
        deadline: Instant,
    ) -> Result<Vec<u8>, String> {
        let meta = self
            .podman(
                &[],
                &[
                    "exec",
                    container,
                    "/usr/bin/env",
                    "LC_ALL=C",
                    "/usr/bin/stat",
                    "-c",
                    "%u:%g:%a:%h:%F",
                    "--",
                    filename,
                ],
                deadline,
            )
            .map_err(|_| "candidate approved file is not protected".to_string())?;
        if meta != b"0:0:644:1:regular file\n" {
            return Err("candidate approved file is not protected".to_string());
        }
        let cap = format!("{}", limit + 1);
        let raw = self
            .podman(
                &[],
                &[
                    "exec",
                    container,
                    "/usr/bin/head",
                    "-c",
                    &cap,
                    "--",
                    filename,
                ],
                deadline,
            )
            .map_err(|_| "candidate approved file is unavailable or exceeds bounds".to_string())?;
        if raw.is_empty() || raw.len() > limit {
            return Err("candidate approved file is unavailable or exceeds bounds".to_string());
        }
        Ok(raw)
    }

    /// `candidateApprovedFiles`: re-read the protected approved snapshot.
    fn candidate_approved_files(
        &self,
        container: &str,
        input: &FactoryCandidate,
        source_commit: &str,
        deadline: Instant,
    ) -> Result<HashMap<String, Vec<u8>>, String> {
        let dir = path_join(&[
            preparation::FACTORY_PREPARATIONS_DIR,
            &input.source_preparation,
        ]);
        let snapshot = path_join(&[&dir, "snapshot"]);
        let meta = self
            .podman(
                &[],
                &[
                    "exec",
                    container,
                    "/usr/bin/stat",
                    "-c",
                    "%u:%g:%a",
                    "--",
                    preparation::FACTORY_DIR,
                    preparation::FACTORY_PREPARATIONS_DIR,
                    &dir,
                    &snapshot,
                ],
                deadline,
            )
            .map_err(|_| "candidate approved snapshot is not protected".to_string())?;
        if meta != b"0:0:755\n0:0:755\n0:0:755\n0:0:755\n" {
            return Err("candidate approved snapshot is not protected".to_string());
        }
        let request_path = path_join(&[&dir, "request.json"]);
        let raw = self.candidate_protected_file(container, &request_path, 4096, deadline)?;
        const ERR: &str = "candidate approved snapshot differs from its recorded inputs";
        let v = json::decode_strict(&raw).map_err(|_| ERR.to_string())?;
        let m = json::bind_root(&v, "struct", SAVED_REQUEST_SPECS, false)
            .map_err(|_| ERR.to_string())?;
        if m.take_string("id") != input.source_preparation
            || m.take_string("role") != input.preparation.role
            || m.take_string("setup_digest") != input.preparation.setup_digest
            || m.take_string("source_commit") != source_commit
            || m.take_string("credential") != input.preparation.credential
        {
            return Err(ERR.to_string());
        }
        let listing = self
            .podman(
                &[],
                &["exec", container, "/usr/bin/ls", "-1A", "--", &snapshot],
                deadline,
            )
            .map_err(|_| "candidate approved snapshot is unavailable".to_string())?;
        if listing.len() > 1024 {
            return Err("candidate approved snapshot is unavailable".to_string());
        }
        let mut files: HashMap<String, Vec<u8>> = HashMap::new();
        let text = String::from_utf8_lossy(&listing);
        for name in text.trim().split('\n') {
            if name == "source.bundle" {
                continue;
            }
            if !preparation::valid_approved_name(name)
                || files.len() >= preparation::MAX_APPROVED_FILES
            {
                return Err("candidate approved snapshot has an invalid file set".to_string());
            }
            let file_path = path_join(&[&snapshot, name]);
            let contents = self.candidate_protected_file(
                container,
                &file_path,
                preparation::MAX_APPROVED_FILE_SIZE,
                deadline,
            )?;
            files.insert(name.to_string(), contents);
        }
        if preparation::setup_digest_of(&files) != input.preparation.setup_digest {
            return Err("candidate approved snapshot differs from its digest".to_string());
        }
        Ok(files)
    }
}

/// `mapPreparationState`: validated observation plus container identity.
/// ID and project are filled in by the caller.
pub fn map_preparation_state(container: &str, raw: &[u8]) -> Result<PrepareState, String> {
    const ERR: &str = "invalid preparation observation";
    let v = json::decode_strict(raw).map_err(|_| ERR.to_string())?;
    let m = json::bind_root(&v, "helperInspection", HELPER_INSPECTION_SPECS, false)
        .map_err(|_| ERR.to_string())?;
    let known = m.take_bool("known");
    let phase = m.take_string("phase");
    let role = m.take_string("role");
    let setup_digest = m.take_string("setup_digest");
    let source_commit = m.take_string("source_commit");
    if !known
        || !preparation::valid_prepare_phase(&phase)
        || !preparation::valid_factory_role(&role)
        || !preparation::valid_digest(&setup_digest)
        || !preparation::valid_commit(&source_commit)
    {
        return Err(ERR.to_string());
    }
    let setup_log = m.take_string("setup_log");
    let check_log = m.take_string("check_log");
    if setup_log.len() > 66560 || check_log.len() > 66560 {
        return Err("preparation observation exceeds the bounded size".to_string());
    }
    let tools = m
        .take_struct_list("tools")
        .iter()
        .map(ResolvedTool::from_map)
        .collect();
    let mut state = PrepareState {
        role,
        phase,
        container: container.to_string(),
        source_commit,
        setup_digest,
        tools,
        missing: m.take_string("missing"),
        setup_exit: m.take_opt_i64("setup_exit"),
        check_exit: m.take_opt_i64("check_exit"),
        ready: m.take_bool("ready"),
        stopped: m.take_bool("stopped"),
        ..Default::default()
    };
    if !setup_log.is_empty() || !check_log.is_empty() {
        state.output = format!("--- setup ---\n{setup_log}\n--- check ---\n{check_log}");
    }
    if state.ready != (state.phase == preparation::PREPARE_READY) {
        return Err("preparation observation is inconsistent".to_string());
    }
    Ok(state)
}

/// Unicode-blank check mirroring `len(bytes.TrimSpace(out)) != 0`.
fn single_line_trimmed(out: &[u8]) -> String {
    match std::str::from_utf8(out) {
        Ok(s) => s.trim().to_string(),
        Err(_) => String::from(" "),
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::project::Config;
    use std::cell::RefCell;
    use std::collections::VecDeque;

    const DIGEST: &str = "32c794ef2201b76b757bfba2c23bba06dcc5a8c6121f6fabc99115ef12043ced";
    const COMMIT: &str = "0123456789abcdef0123456789abcdef01234567";
    const EFFECTS: &str = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa";

    type MockCall = (Vec<u8>, String, Vec<String>);

    struct Mock {
        calls: RefCell<Vec<MockCall>>,
        script: RefCell<VecDeque<Result<Vec<u8>, String>>>,
    }

    impl Mock {
        fn new(responses: Vec<Result<Vec<u8>, String>>) -> Self {
            Mock {
                calls: RefCell::new(Vec::new()),
                script: RefCell::new(responses.into()),
            }
        }
    }

    impl Executor for Mock {
        fn run(
            &self,
            stdin: &[u8],
            cmd: &str,
            args: &[&str],
            _deadline: Instant,
        ) -> Result<Vec<u8>, String> {
            self.calls.borrow_mut().push((
                stdin.to_vec(),
                cmd.to_string(),
                args.iter().map(|s| s.to_string()).collect(),
            ));
            self.script
                .borrow_mut()
                .pop_front()
                .unwrap_or(Err("no scripted response".to_string()))
        }
    }

    fn deadline() -> Instant {
        Instant::now() + std::time::Duration::from_secs(5)
    }

    fn test_config() -> Config {
        Config {
            muse_socket: String::new(),
            image: "img".to_string(),
            network: "sodanet".to_string(),
            subnet: "10.0.0.0/24".to_string(),
            bridge: "sodabr".to_string(),
        }
    }

    fn pid() -> String {
        format!("p{}", "a".repeat(24))
    }

    fn fid() -> String {
        format!("f{}", "b".repeat(24))
    }

    fn container_payload(id: &str) -> Vec<u8> {
        let cid = "f".repeat(64);
        format!(
            "{{\"id\":{cid:?},\"running\":true,\"project\":{id:?},\"owner\":\"42\",\"privileged\":false,\"userns\":\"private\",\"mappings\":{{\"UidMap\":[\"0:1000000:262144\"],\"GidMap\":[\"0:1000000:262144\"]}}}}"
        )
        .into_bytes()
    }

    fn observation(phase: &str, role: &str, ready: bool, stopped: bool) -> Vec<u8> {
        format!(
            "{{\"hold\":{{\"active\":false,\"revision\":0}},\"known\":true,\
             \"phase\":{phase:?},\"role\":{role:?},\"setup_digest\":{DIGEST:?},\
             \"source_commit\":{COMMIT:?},\"tools\":[],\
             \"verified\":{{\"uid\":\"1001\",\"login\":{role:?},\"groups\":{role:?},\
             \"refusal\":\"\"}},\"missing\":\"\",\"stopped\":{stopped},\"ready\":{ready},\
             \"setup_exit\":null,\"check_exit\":null,\"setup_log\":\"\",\"check_log\":\"\"}}"
        )
        .into_bytes()
    }

    fn preparation() -> Preparation {
        Preparation {
            id: fid(),
            project: pid(),
            role: "soda-coder".to_string(),
            revision: 0,
            requirements: preparation::RequirementAcceptance {
                id: format!("d{:024x}", 1),
                revision: 0,
                approver: 1,
                source_commit: COMMIT.to_string(),
                digest: EFFECTS.to_string(),
            },
            approval: preparation::AdminApproval {
                id: format!("d{:024x}", 2),
                revision: 0,
                approver: 1,
                effects_digest: EFFECTS.to_string(),
            },
            source_commit: COMMIT.to_string(),
            setup_digest: DIGEST.to_string(),
            tools: vec![],
            credential: String::new(),
        }
    }

    fn setup() -> preparation::ApprovedSetup {
        preparation::ApprovedSetup {
            files: [
                ("setup.sh".to_string(), b"#!/bin/sh\n".to_vec()),
                ("check.sh".to_string(), b"#!/bin/sh\n".to_vec()),
            ]
            .into_iter()
            .collect(),
            bundle: b"BUNDLE".to_vec(),
        }
    }

    #[test]
    fn path_clean_matches_go_vectors() {
        for (input, want) in [
            ("", "."),
            ("/", "/"),
            ("//", "/"),
            ("/a/", "/a"),
            ("a/", "a"),
            ("/a//b", "/a/b"),
            ("/a/./b", "/a/b"),
            ("a/./b", "a/b"),
            ("/a/../b", "/b"),
            ("a/../../b", "../b"),
            ("/../", "/"),
            ("../a", "../a"),
            ("/..", "/"),
            ("..", ".."),
            (".", "."),
            ("/a/b/..", "/a"),
            ("a/b/../..", "."),
            ("/usr/bin/../bin/git", "/usr/bin/git"),
            ("/a/.../b", "/a/.../b"),
            ("a/.../b", "a/.../b"),
        ] {
            assert_eq!(path_clean(input), want, "{input:?}");
        }
        assert_eq!(path_join(&["a", "b", "c"]), "a/b/c");
        assert_eq!(path_join(&["/a", "b/../c"]), "/a/c");
        assert_eq!(path_join(&[]), "");
        assert_eq!(path_join(&[""]), "");
        assert_eq!(path_join(&["", "a", "b"]), "a/b");
        assert_eq!(path_join(&["a", "", "b"]), "a/b");
        assert_eq!(path_join(&["", ""]), "");
        assert!(path_is_abs("/x"));
        assert!(!path_is_abs("x"));
    }

    #[test]
    fn idmap_requires_shifted_usable_range() {
        assert!(prepare_id_map(&["0:1000000:262144".to_string()]));
        assert!(prepare_id_map(&[
            "1:1:1".to_string(),
            "0:1000000:262144".to_string()
        ]));
        assert!(!prepare_id_map(&[]));
        assert!(!prepare_id_map(&["1:1000000:262144".to_string()]));
        assert!(!prepare_id_map(&["0:1000000:100".to_string()]));
        assert!(!prepare_id_map(&["0:0:262144".to_string()]));
        assert!(!prepare_id_map(&["0:01:262144".to_string()]));
        assert!(!prepare_id_map(&["0:+1:262144".to_string()]));
        assert!(!prepare_id_map(&["0:1".to_string()]));
        assert!(!prepare_id_map(&["0:1:2:3".to_string()]));
        // base + size must fit uint32: 4294705151 + 262144 = max exactly.
        assert!(prepare_id_map(&["0:4294705151:262144".to_string()]));
        assert!(!prepare_id_map(&["0:4294705152:262144".to_string()]));
        assert!(!prepare_id_map(&["0:4294967295:262144".to_string()]));
        // Every entry must parse.
        assert!(!prepare_id_map(&[
            "0:1000000:262144".to_string(),
            "bogus".to_string()
        ]));
    }

    #[test]
    fn single_line_rules() {
        assert_eq!(single_line(b"abc\n", 64).as_deref(), Some("abc"));
        assert_eq!(single_line(b"  abc  ", 64).as_deref(), Some("abc"));
        assert_eq!(single_line(b"a\rb", 64).as_deref(), Some("a\rb"));
        assert_eq!(single_line(b"", 64), None);
        assert_eq!(single_line(b"   ", 64), None);
        assert_eq!(single_line(b"a\nb", 64), None);
        assert_eq!(single_line(b"abc", 2), None);
        assert_eq!(single_line(b"\xff\xfe", 64), None);
    }

    #[test]
    fn tool_paths_must_be_clean_and_pinned() {
        assert!(valid_resolved_tool_path("/usr/bin/git"));
        assert!(valid_resolved_tool_path("/usr/local/bin/tool"));
        assert!(!valid_resolved_tool_path(""));
        assert!(!valid_resolved_tool_path("git"));
        assert!(!valid_resolved_tool_path("/usr/bin/../bin/git"));
        assert!(!valid_resolved_tool_path("/bin/git"));
        assert!(!valid_resolved_tool_path("/usr/bin/"));
        assert!(!valid_resolved_tool_path(&format!(
            "/usr/bin/{}",
            "a".repeat(250)
        )));
    }

    #[test]
    fn preparation_paths_layout() {
        let (checkout, snapshot, bundle, home) = preparation_paths("soda-coder", "fid");
        assert_eq!(checkout, "/home/soda-coder/checkouts/fid");
        assert_eq!(snapshot, "/var/lib/soda/factory/preparations/fid/snapshot");
        assert_eq!(
            bundle,
            "/var/lib/soda/factory/preparations/fid/snapshot/source.bundle"
        );
        assert_eq!(home, "/home/soda-coder/checkouts/fid/.soda-home");
    }

    #[test]
    fn map_state_validates_and_assembles() {
        let raw = observation("ready", "soda-coder", true, false);
        let state = map_preparation_state("cid", &raw).unwrap();
        assert_eq!(state.container, "cid");
        assert!(state.ready);
        assert!(state.id.is_empty()); // caller fills identity
                                      // Ready must agree with the phase.
        let raw = observation("running", "soda-coder", true, false);
        assert_eq!(
            map_preparation_state("cid", &raw).unwrap_err(),
            "preparation observation is inconsistent"
        );
        let raw = observation("ready", "soda-coder", false, false);
        assert_eq!(
            map_preparation_state("cid", &raw).unwrap_err(),
            "preparation observation is inconsistent"
        );
        // Unknown phase / role / digest / commit.
        let mut bad = String::from_utf8(observation("nope", "soda-coder", false, false)).unwrap();
        assert_eq!(
            map_preparation_state("cid", bad.as_bytes()).unwrap_err(),
            "invalid preparation observation"
        );
        bad = bad.replace("\"soda-coder\"", "\"coder\"");
        assert_eq!(
            map_preparation_state("cid", bad.as_bytes()).unwrap_err(),
            "invalid preparation observation"
        );
        // Oversized logs.
        let big = "x".repeat(66561);
        let raw = format!(
            "{{\"known\":true,\"phase\":\"running\",\"role\":\"soda-coder\",\
             \"setup_digest\":{DIGEST:?},\"source_commit\":{COMMIT:?},\
             \"setup_log\":{big:?}}}"
        );
        assert_eq!(
            map_preparation_state("cid", raw.as_bytes()).unwrap_err(),
            "preparation observation exceeds the bounded size"
        );
        // Log assembly, tools, exits.
        let raw = format!(
            "{{\"known\":true,\"phase\":\"running\",\"role\":\"soda-coder\",\
             \"setup_digest\":{DIGEST:?},\"source_commit\":{COMMIT:?},\
             \"tools\":[{{\"name\":\"git\",\"path\":\"/usr/bin/git\",\"version\":\"v\"}}],\
             \"missing\":\"go\",\"setup_exit\":0,\"check_exit\":3,\
             \"setup_log\":\"s\",\"check_log\":\"c\"}}"
        );
        let state = map_preparation_state("cid", raw.as_bytes()).unwrap();
        assert_eq!(state.output, "--- setup ---\ns\n--- check ---\nc");
        assert_eq!(state.tools.len(), 1);
        assert_eq!(state.setup_exit, Some(0));
        assert_eq!(state.check_exit, Some(3));
        assert_eq!(state.missing, "go");
    }

    #[test]
    fn prepare_container_binds_isolated_target() {
        let id = pid();
        let mock = Mock::new(vec![Ok(container_payload(&id))]);
        let rt = Runtime {
            exec: &mock,
            config: test_config(),
        };
        let cid = rt.prepare_container(&id, true, deadline()).unwrap();
        assert_eq!(cid, "f".repeat(64));
        let calls = mock.calls.borrow();
        assert_eq!(calls[0].1, "/usr/bin/podman");
        assert_eq!(
            calls[0].2[..4],
            [
                "--remote=false".to_string(),
                "inspect".to_string(),
                "--format".to_string(),
                crate::project::PROJECT_INSPECT_FORMAT.to_string(),
            ]
        );
        assert_eq!(calls[0].2[4], format!("soda-{id}"));
        // Bad project id never execs.
        assert_eq!(
            rt.prepare_container("x", true, deadline()).unwrap_err(),
            "invalid project"
        );
        // Podman failure and oversized output.
        let mock = Mock::new(vec![Err("boom".to_string())]);
        let rt = Runtime {
            exec: &mock,
            config: test_config(),
        };
        assert_eq!(
            rt.prepare_container(&id, true, deadline()).unwrap_err(),
            "preparation target unavailable"
        );
        let mock = Mock::new(vec![Ok(vec![b'x'; 4097])]);
        let rt = Runtime {
            exec: &mock,
            config: test_config(),
        };
        assert_eq!(
            rt.prepare_container(&id, true, deadline()).unwrap_err(),
            "preparation target unavailable"
        );
        // Unparseable and not-ready targets.
        let mock = Mock::new(vec![Ok(b"{}".to_vec())]);
        let rt = Runtime {
            exec: &mock,
            config: test_config(),
        };
        assert_eq!(
            rt.prepare_container(&id, true, deadline()).unwrap_err(),
            "preparation target not ready or isolated"
        );
    }

    fn approve_response(fid: &str) -> Vec<u8> {
        format!(
            "{{\"approved\":{fid:?},\"repeated\":false,\
             \"checkout\":\"/home/soda-coder/checkouts/{fid}\",\"credential_file\":\"\"}}"
        )
        .into_bytes()
    }

    #[test]
    fn prepare_runs_the_full_chain() {
        let id = pid();
        let fid = fid();
        let mock = Mock::new(vec![
            Ok(container_payload(&id)), // prepareContainer
            Ok(b"{}".to_vec()),         // ensure
            Ok(approve_response(&fid)), // approve
            Ok(observation("approved", "soda-coder", false, false)), // inspect
            Ok(b"".to_vec()),           // test -d (exists)
            Ok(format!("{COMMIT}\n").into_bytes()), // rev-parse
            Ok(b"".to_vec()),           // mkdir
            Ok(b"1001\n".to_vec()),     // id -u
            Ok(b"soda-coder\n".to_vec()), // id -un
            Ok(b"soda-coder\n".to_vec()), // id -Gn
            Ok(b"".to_vec()),           // test -r setup
            Err("exit status 1".to_string()), // test -w setup
            Err("exit status 1".to_string()), // test -w shared
            Err("exit status 1".to_string()), // test -e podman.sock
            Err("exit status 1".to_string()), // test -e docker.sock
            Err("exit status 1".to_string()), // test -x sudo
            Ok(b"".to_vec()),           // test -w home
            Ok(b"{}".to_vec()),         // record
            Ok(b"{}".to_vec()),         // start
            Ok(observation("running", "soda-coder", false, false)), // final inspect
        ]);
        let rt = Runtime {
            exec: &mock,
            config: test_config(),
        };
        let input = Prepare {
            preparation: preparation(),
            setup: setup(),
        };
        let state = rt.prepare(&input, deadline()).unwrap();
        assert_eq!(state.phase, "running");
        assert_eq!(state.id, fid);
        assert_eq!(state.project, id);
        let calls = mock.calls.borrow();
        assert_eq!(calls.len(), 20);
        // Approve payload: sorted keys, padded base64 files.
        assert_eq!(
            String::from_utf8(calls[2].0.clone()).unwrap(),
            format!(
                "{{\"bundle\":\"QlVORExF\",\"credential\":\"\",\
                 \"files\":{{\"check.sh\":\"IyEvYmluL3NoCg==\",\
                 \"setup.sh\":\"IyEvYmluL3NoCg==\"}},\"id\":{fid:?},\
                 \"op\":\"approve\",\"role\":\"soda-coder\",\"setup_digest\":{DIGEST:?},\
                 \"source_commit\":{COMMIT:?}}}"
            )
        );
        assert_eq!(
            calls[2].2,
            vec![
                "exec".to_string(),
                "--interactive".to_string(),
                format!("soda-{id}"),
                "/usr/libexec/soda/project-factory-roles".to_string(),
            ]
        );
        // Record payload carries the observed evidence.
        assert_eq!(
            String::from_utf8(calls[17].0.clone()).unwrap(),
            format!(
                "{{\"id\":{fid:?},\"missing\":\"\",\"op\":\"record\",\"tools\":[],\
                 \"verified\":{{\"groups\":\"soda-coder\",\"login\":\"soda-coder\",\
                 \"uid\":\"1001\"}}}}"
            )
        );
        // Start payload.
        assert_eq!(
            String::from_utf8(calls[18].0.clone()).unwrap(),
            format!("{{\"id\":{fid:?},\"op\":\"start\"}}")
        );
    }

    #[test]
    fn prepare_short_circuits_and_recovers() {
        let id = pid();
        let fid = fid();
        // Ready state returns right after the first inspect.
        let mock = Mock::new(vec![
            Ok(container_payload(&id)),
            Ok(b"{}".to_vec()),
            Ok(approve_response(&fid)),
            Ok(observation("ready", "soda-coder", true, false)),
        ]);
        let rt = Runtime {
            exec: &mock,
            config: test_config(),
        };
        let input = Prepare {
            preparation: preparation(),
            setup: setup(),
        };
        let state = rt.prepare(&input, deadline()).unwrap();
        assert!(state.ready);
        assert_eq!(mock.calls.borrow().len(), 4);
        // Approve failure with a stopped tombstone returns the tombstone.
        let mock = Mock::new(vec![
            Ok(container_payload(&id)),
            Ok(b"{}".to_vec()),
            Err("helper blew up".to_string()),
            Ok(observation("stopped", "soda-coder", false, true)),
        ]);
        let rt = Runtime {
            exec: &mock,
            config: test_config(),
        };
        let state = rt.prepare(&input, deadline()).unwrap();
        assert!(state.stopped);
        // Approve failure without a tombstone surfaces the error.
        let mock = Mock::new(vec![
            Ok(container_payload(&id)),
            Ok(b"{}".to_vec()),
            Err("helper blew up".to_string()),
            Ok(observation("approved", "soda-coder", false, false)),
        ]);
        let rt = Runtime {
            exec: &mock,
            config: test_config(),
        };
        assert_eq!(
            rt.prepare(&input, deadline()).unwrap_err(),
            "helper blew up"
        );
        // Validation never execs.
        let mut bad = preparation();
        bad.id = "x".to_string();
        let input = Prepare {
            preparation: bad,
            setup: setup(),
        };
        let mock = Mock::new(vec![]);
        let rt = Runtime {
            exec: &mock,
            config: test_config(),
        };
        assert_eq!(
            rt.prepare(&input, deadline()).unwrap_err(),
            "invalid preparation identity"
        );
        assert!(mock.calls.borrow().is_empty());
    }

    #[test]
    fn prepare_skips_start_when_blocked() {
        let id = pid();
        let fid = fid();
        // Missing tool: record, then re-observe, never start.
        let mut prep = preparation();
        prep.tools = vec!["git".to_string(), "go".to_string()];
        let mock = Mock::new(vec![
            Ok(container_payload(&id)),
            Ok(b"{}".to_vec()),
            Ok(approve_response(&fid)),
            Ok(observation("approved", "soda-coder", false, false)),
            Ok(b"".to_vec()),
            Ok(format!("{COMMIT}\n").into_bytes()),
            Ok(b"".to_vec()),
            Ok(b"1001\n".to_vec()),
            Ok(b"soda-coder\n".to_vec()),
            Ok(b"soda-coder\n".to_vec()),
            Ok(b"".to_vec()),
            Err("exit status 1".to_string()),
            Err("exit status 1".to_string()),
            Err("exit status 1".to_string()),
            Err("exit status 1".to_string()),
            Err("exit status 1".to_string()),
            Ok(b"".to_vec()),
            Ok(b"/usr/bin/git\n".to_vec()),   // command -v git
            Ok(b"git version 2\n".to_vec()),  // git --version
            Err("exit status 1".to_string()), // command -v go missing
            Ok(b"{}".to_vec()),               // record
            Ok(observation("approved", "soda-coder", false, false)),
        ]);
        let rt = Runtime {
            exec: &mock,
            config: test_config(),
        };
        let input = Prepare {
            preparation: prep,
            setup: setup(),
        };
        let state = rt.prepare(&input, deadline()).unwrap();
        assert_eq!(state.phase, "approved");
        let calls = mock.calls.borrow();
        assert_eq!(calls.len(), 22);
        assert!(!calls
            .iter()
            .any(|c| String::from_utf8_lossy(&c.0).contains("\"op\":\"start\"")));
        // The missing marker names go.
        assert!(String::from_utf8_lossy(&calls[20].0).contains("\"missing\":\"go\""));
        assert!(
            String::from_utf8_lossy(&calls[20].0)
                .contains("\"tools\":[{\"name\":\"git\",\"path\":\"/usr/bin/git\",\"version\":\"git version 2\"}]")
        );
    }

    #[test]
    fn inspect_stop_hold_paths() {
        let id = pid();
        let fid = fid();
        // Inspect.
        let mock = Mock::new(vec![
            Ok(container_payload(&id)),
            Ok(observation("running", "soda-coder", false, false)),
        ]);
        let rt = Runtime {
            exec: &mock,
            config: test_config(),
        };
        let state = rt
            .inspect_preparation(
                &PrepareInspect {
                    project: id.clone(),
                    id: fid.clone(),
                },
                deadline(),
            )
            .unwrap();
        assert_eq!(state.phase, "running");
        assert_eq!(state.id, fid);
        // Stop.
        let mock = Mock::new(vec![
            Ok(container_payload(&id)),
            Ok(
                format!("{{\"stopped\":{fid:?},\"retirement\":\"confirmed\",\"known\":true}}")
                    .into_bytes(),
            ),
            Ok(observation("stopped", "soda-coder", false, true)),
        ]);
        let rt = Runtime {
            exec: &mock,
            config: test_config(),
        };
        let state = rt
            .stop_preparation(
                &PrepareStop {
                    project: id.clone(),
                    id: fid.clone(),
                },
                deadline(),
            )
            .unwrap();
        assert!(state.stopped);
        assert_eq!(state.retirement, "confirmed");
        assert_eq!(
            String::from_utf8(mock.calls.borrow()[1].0.clone()).unwrap(),
            format!("{{\"id\":{fid:?},\"op\":\"stop\"}}")
        );
        // Stop with a bad tombstone.
        let mock = Mock::new(vec![
            Ok(container_payload(&id)),
            Ok(
                format!("{{\"stopped\":{fid:?},\"retirement\":\"maybe\",\"known\":true}}")
                    .into_bytes(),
            ),
        ]);
        let rt = Runtime {
            exec: &mock,
            config: test_config(),
        };
        assert_eq!(
            rt.stop_preparation(
                &PrepareStop {
                    project: id.clone(),
                    id: fid.clone()
                },
                deadline()
            )
            .unwrap_err(),
            "preparation stop unconfirmed"
        );
        // Hold and release.
        for (hold, op) in [(true, "hold"), (false, "release")] {
            let mock = Mock::new(vec![
                Ok(container_payload(&id)),
                Ok(format!("{{\"hold\":{{\"active\":{hold},\"revision\":7}}}}").into_bytes()),
            ]);
            let rt = Runtime {
                exec: &mock,
                config: test_config(),
            };
            let state = rt
                .hold_preparation(
                    &PrepareHold {
                        project: id.clone(),
                        hold,
                        revision: 7,
                    },
                    deadline(),
                )
                .unwrap();
            assert_eq!(state.active, hold);
            assert_eq!(state.revision, 7);
            assert_eq!(
                String::from_utf8(mock.calls.borrow()[1].0.clone()).unwrap(),
                format!("{{\"op\":\"{op}\",\"revision\":7}}")
            );
        }
        // Hold outcome mismatch.
        let mock = Mock::new(vec![
            Ok(container_payload(&id)),
            Ok(b"{\"hold\":{\"active\":false,\"revision\":7}}".to_vec()),
        ]);
        let rt = Runtime {
            exec: &mock,
            config: test_config(),
        };
        assert_eq!(
            rt.hold_preparation(
                &PrepareHold {
                    project: id.clone(),
                    hold: true,
                    revision: 7
                },
                deadline()
            )
            .unwrap_err(),
            "maintenance hold outcome not confirmed"
        );
    }

    #[test]
    fn candidate_reuses_ready_source() {
        let id = pid();
        let source = format!("f{}", "c".repeat(24));
        let mut prep = preparation();
        prep.role = "soda-reviewer".to_string();
        let fid = prep.id.clone();
        let request = format!(
            "{{\"id\":{source:?},\"role\":\"soda-reviewer\",\"setup_digest\":{DIGEST:?},\
             \"source_commit\":{COMMIT:?},\"credential\":\"\"}}"
        );
        let mock = Mock::new(vec![
            Ok(container_payload(&id)),                             // prepareContainer
            Ok(observation("ready", "soda-reviewer", true, false)), // source inspect
            Ok(b"0:0:755\n0:0:755\n0:0:755\n0:0:755\n".to_vec()),   // dir stat
            Ok(b"0:0:644:1:regular file\n".to_vec()),               // request stat
            Ok(request.into_bytes()),                               // request head
            Ok(b"check.sh\nsetup.sh\n".to_vec()),                   // ls
            Ok(b"0:0:644:1:regular file\n".to_vec()),               // check.sh stat
            Ok(b"#!/bin/sh\n".to_vec()),                            // check.sh head
            Ok(b"0:0:644:1:regular file\n".to_vec()),               // setup.sh stat
            Ok(b"#!/bin/sh\n".to_vec()),                            // setup.sh head
            // ... then the full prepare chain for the fresh identity:
            Ok(container_payload(&id)), // prepareContainer
            Ok(b"{}".to_vec()),         // ensure
            Ok(String::from_utf8(approve_response(&fid))
                .unwrap()
                .replace("soda-coder", "soda-reviewer")
                .into_bytes()),
            Ok(observation("approved", "soda-reviewer", false, false)),
            Ok(b"".to_vec()), // test -d
            Ok(format!("{COMMIT}\n").into_bytes()),
            Ok(b"".to_vec()), // mkdir
            Ok(b"1002\n".to_vec()),
            Ok(b"soda-reviewer\n".to_vec()),
            Ok(b"soda-reviewer\n".to_vec()),
            Ok(b"".to_vec()),
            Err("exit status 1".to_string()),
            Err("exit status 1".to_string()),
            Err("exit status 1".to_string()),
            Err("exit status 1".to_string()),
            Err("exit status 1".to_string()),
            Ok(b"".to_vec()),
            Ok(b"{}".to_vec()), // record
            Ok(b"{}".to_vec()), // start
            Ok(observation("running", "soda-reviewer", false, false)),
        ]);
        let rt = Runtime {
            exec: &mock,
            config: test_config(),
        };
        let input = FactoryCandidate {
            preparation: prep,
            source_preparation: source.clone(),
            bundle: b"BUNDLE".to_vec(),
        };
        let state = rt.prepare_candidate(&input, deadline()).unwrap();
        assert_eq!(state.phase, "running");
        assert_eq!(state.id, fid);
        let calls = mock.calls.borrow();
        assert_eq!(calls.len(), 30);
        // The snapshot stat covers all four directories.
        assert_eq!(
            calls[2].2[..7],
            [
                "exec".to_string(),
                "f".repeat(64),
                "/usr/bin/stat".to_string(),
                "-c".to_string(),
                "%u:%g:%a".to_string(),
                "--".to_string(),
                "/var/lib/soda/factory".to_string(),
            ]
        );
        // Source not ready short-circuits before any snapshot reads.
        let mock = Mock::new(vec![
            Ok(container_payload(&id)),
            Ok(observation("running", "soda-reviewer", false, false)),
        ]);
        let rt = Runtime {
            exec: &mock,
            config: test_config(),
        };
        let mut prep = preparation();
        prep.role = "soda-reviewer".to_string();
        let input = FactoryCandidate {
            preparation: prep,
            source_preparation: source.clone(),
            bundle: b"BUNDLE".to_vec(),
        };
        assert_eq!(
            rt.prepare_candidate(&input, deadline()).unwrap_err(),
            "candidate source preparation is not ready for this role and setup"
        );
        assert_eq!(mock.calls.borrow().len(), 2);
    }

    #[test]
    fn candidate_rejects_bad_snapshots() {
        let id = pid();
        let source = format!("f{}", "c".repeat(24));
        let mut prep = preparation();
        prep.role = "soda-reviewer".to_string();
        let input = || FactoryCandidate {
            preparation: prep.clone(),
            source_preparation: source.clone(),
            bundle: b"BUNDLE".to_vec(),
        };
        // Unprotected directories.
        let mock = Mock::new(vec![
            Ok(container_payload(&id)),
            Ok(observation("ready", "soda-reviewer", true, false)),
            Ok(b"0:0:755\n0:0:755\n0:0:755\n0:0:700\n".to_vec()),
        ]);
        let rt = Runtime {
            exec: &mock,
            config: test_config(),
        };
        assert_eq!(
            rt.prepare_candidate(&input(), deadline()).unwrap_err(),
            "candidate approved snapshot is not protected"
        );
        // Request that differs from the recorded inputs.
        let request = format!(
            "{{\"id\":{source:?},\"role\":\"soda-reviewer\",\"setup_digest\":{DIGEST:?},\
             \"source_commit\":{COMMIT:?},\"credential\":\"other\"}}"
        );
        let mock = Mock::new(vec![
            Ok(container_payload(&id)),
            Ok(observation("ready", "soda-reviewer", true, false)),
            Ok(b"0:0:755\n0:0:755\n0:0:755\n0:0:755\n".to_vec()),
            Ok(b"0:0:644:1:regular file\n".to_vec()),
            Ok(request.into_bytes()),
        ]);
        let rt = Runtime {
            exec: &mock,
            config: test_config(),
        };
        assert_eq!(
            rt.prepare_candidate(&input(), deadline()).unwrap_err(),
            "candidate approved snapshot differs from its recorded inputs"
        );
        // Invalid file set via an illegal name.
        let request = format!(
            "{{\"id\":{source:?},\"role\":\"soda-reviewer\",\"setup_digest\":{DIGEST:?},\
             \"source_commit\":{COMMIT:?},\"credential\":\"\"}}"
        );
        let mock = Mock::new(vec![
            Ok(container_payload(&id)),
            Ok(observation("ready", "soda-reviewer", true, false)),
            Ok(b"0:0:755\n0:0:755\n0:0:755\n0:0:755\n".to_vec()),
            Ok(b"0:0:644:1:regular file\n".to_vec()),
            Ok(request.into_bytes()),
            Ok(b"setup.sh\n../evil\n".to_vec()),
            Ok(b"0:0:644:1:regular file\n".to_vec()),
            Ok(b"#!/bin/sh\n".to_vec()),
        ]);
        let rt = Runtime {
            exec: &mock,
            config: test_config(),
        };
        assert_eq!(
            rt.prepare_candidate(&input(), deadline()).unwrap_err(),
            "candidate approved snapshot has an invalid file set"
        );
    }
}

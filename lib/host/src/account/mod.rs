//! Project account operations.
//!
//! Ports the account tail of `create.go` (`Account`, `canonicalizeAccountKeys`,
//! `confirmAccount`) and `access_keys.go` (`AccessKeys`, `canonicalKeys`,
//! `decodeAccessKeyState`). In-container key operations run through the fixed
//! `project-terminal` agent binary (`keys` action), like the Go `AgentExec`.

use std::time::Instant;

use crate::domain::{self, AccessKeyState, AccessKeys, Account};
use crate::json::{self, Kind, Spec};
use crate::project::{Executor, Runtime};
use crate::ssh;

/// Fixed in-container terminal agent, mirroring Go `terminal.AgentProgram`.
pub const AGENT_PROGRAM: &str = "/usr/libexec/soda/project-terminal";

/// `keyRevision = ^[0-9a-f]{64}$`.
pub fn valid_key_revision(s: &str) -> bool {
    s.len() == 64
        && s.bytes()
            .all(|b| b.is_ascii_hexdigit() && (b.is_ascii_digit() || b.is_ascii_lowercase()))
}

/// `canonicalizeAccountKeys`: parse every key, drop comments, keep the
/// `MarshalAuthorizedKey` rendering (with its trailing newline) for the
/// container-side `project-account` request.
pub fn canonicalize_account_keys(values: &[String]) -> Result<Vec<String>, String> {
    let mut keys = Vec::with_capacity(values.len());
    for value in values {
        let (key, has_options) = ssh::parse_authorized_key(value.as_bytes())
            .map_err(|_| "invalid public key".to_string())?;
        if has_options {
            return Err("invalid public key".to_string());
        }
        keys.push(ssh::marshal_authorized_key(&key.key_type, &key.blob));
    }
    Ok(keys)
}

/// `confirmAccount`: the helper must echo the login identity back; every
/// shape mismatch collapses into one confirmation error.
const CONFIRM_SPECS: &[Spec] = &[
    Spec {
        name: "login",
        kind: Kind::Str,
    },
    Spec {
        name: "identity",
        kind: Kind::I64,
    },
];

fn confirm_account(out: &[u8], login: &str, identity: i64) -> Result<(), String> {
    const ERR: &str = "native account identity was not confirmed";
    if out.len() > 4096 {
        return Err(ERR.to_string());
    }
    let v = json::decode_strict(out).map_err(|_| ERR.to_string())?;
    let m = json::bind_root(&v, "struct", CONFIRM_SPECS, false).map_err(|_| ERR.to_string())?;
    let got_login = m.take_string("login");
    let got_identity = m.take_i64("identity");
    if got_login != login || got_identity != identity {
        return Err(ERR.to_string());
    }
    Ok(())
}

/// `canonicalKeys`: canonical `type base64` lines only, no options, no
/// comments, no duplicates, at most 32 keys and 48000 bytes.
pub fn canonical_keys(values: &[String]) -> Result<Vec<String>, String> {
    if values.len() > 32 {
        return Err("too many development keys".to_string());
    }
    let mut out = Vec::with_capacity(values.len());
    let mut seen = std::collections::HashSet::with_capacity(values.len());
    let mut size = 0usize;
    for value in values {
        let (key, has_options) = ssh::parse_authorized_key(value.as_bytes())
            .map_err(|_| "invalid development key".to_string())?;
        if has_options {
            return Err("invalid development key".to_string());
        }
        // `MarshalAuthorizedKey` ends with "\n"; `TrimSpace` strips it.
        let canonical = ssh::marshal_authorized_key(&key.key_type, &key.blob);
        let canonical = canonical.trim_end_matches('\n');
        if value.trim() != canonical || !seen.insert(canonical.to_string()) {
            return Err("noncanonical or duplicate development key".to_string());
        }
        size += canonical.len();
        if size > 48000 {
            return Err("development key set too large".to_string());
        }
        out.push(canonical.to_string());
    }
    Ok(out)
}

fn valid_access_keys_request(input: &AccessKeys) -> bool {
    if !domain::valid_login(&input.login) || input.login == "root" || input.identity <= 0 {
        return false;
    }
    if input.apply {
        return valid_key_revision(&input.revision);
    }
    input.revision.is_empty() && input.keys.is_empty()
}

/// `decodeAccessKeyState`: plain `encoding/json` semantics (unknown fields
/// ignored, last duplicate wins); only the revision shape, key presence and
/// key canonicality are enforced.
const KEY_STATE_SPECS: &[Spec] = &[
    Spec {
        name: "revision",
        kind: Kind::Str,
    },
    Spec {
        name: "keys",
        kind: Kind::StrList,
    },
];

pub fn decode_access_key_state(data: &[u8]) -> Result<AccessKeyState, String> {
    if data.len() > 65536 {
        return Err("native key operation not confirmed".to_string());
    }
    const ERR: &str = "invalid native key observation";
    let v = json::decode_tolerant(data).map_err(|_| ERR.to_string())?;
    // Tolerant binding: plain Unmarshal ignores unknown fields.
    let m = json::bind_root(&v, "AccessKeyState", KEY_STATE_SPECS, true)
        .map_err(|_| ERR.to_string())?;
    if !m.contains("keys") {
        return Err(ERR.to_string());
    }
    let revision = m.take_string("revision");
    let keys = m.take_str_list("keys");
    if !valid_key_revision(&revision) {
        return Err(ERR.to_string());
    }
    let keys = canonical_keys(&keys)?;
    Ok(AccessKeyState { revision, keys })
}

impl<E: Executor> Runtime<E> {
    /// `Account`: provision a project login via the in-container helper.
    pub fn account(&self, input: &Account, deadline: Instant) -> Result<(), String> {
        if !domain::valid_login(&input.login)
            || input.login == "root"
            || input.identity <= 0
            || input.keys.len() > 32
        {
            return Err("invalid project account".to_string());
        }
        let (env, owner) = self.inspect(&input.project, deadline)?;
        if !env.running {
            return Err("project is stopped".to_string());
        }
        let keys = canonicalize_account_keys(&input.keys)?;
        // Go map marshal emits keys sorted: admin, identity, keys, login.
        let mut body = String::from("{\"admin\":");
        body.push_str(if input.identity == owner {
            "true"
        } else {
            "false"
        });
        body.push_str(",\"identity\":");
        body.push_str(&input.identity.to_string());
        body.push_str(",\"keys\":[");
        for (i, key) in keys.iter().enumerate() {
            if i > 0 {
                body.push(',');
            }
            body.push_str(&json::quote(key));
        }
        body.push_str("],\"login\":");
        body.push_str(&json::quote(&input.login));
        body.push('}');
        let target = format!("soda-{}", input.project);
        // No `--remote=false` on this path, exactly like the Go call.
        let args = [
            "exec",
            "--interactive",
            &target,
            "/usr/libexec/soda/project-account",
        ];
        let out = self.podman(body.as_bytes(), &args, deadline)?;
        confirm_account(&out, &input.login, input.identity)
    }

    /// `AccessKeys`: preview or revision-checked key-set replacement.
    pub fn access_keys(
        &self,
        input: &AccessKeys,
        deadline: Instant,
    ) -> Result<AccessKeyState, String> {
        if !valid_access_keys_request(input) {
            return Err("invalid own-account key operation".to_string());
        }
        let keys = canonical_keys(&input.keys)?;
        let cid = self.project_container(&input.project, true, deadline)?;
        // `previewAccessKeys`: re-observe without applying; any observation
        // failure or revision drift collapses into one error.
        if input.apply {
            let preview = AccessKeys {
                project: input.project.clone(),
                login: input.login.clone(),
                identity: input.identity,
                revision: String::new(),
                keys: Vec::new(),
                apply: false,
            };
            let observed = self
                .access_keys(&preview, deadline)
                .map_err(|_| "native keys changed or are not managed canonical keys".to_string())?;
            if observed.revision != input.revision {
                return Err("native keys changed or are not managed canonical keys".to_string());
            }
        }
        // Go map marshal emits keys sorted: apply, identity, keys, login, revision.
        let mut body = String::from("{\"apply\":");
        body.push_str(if input.apply { "true" } else { "false" });
        body.push_str(",\"identity\":");
        body.push_str(&input.identity.to_string());
        body.push_str(",\"keys\":[");
        for (i, key) in keys.iter().enumerate() {
            if i > 0 {
                body.push(',');
            }
            body.push_str(&json::quote(key));
        }
        body.push_str("],\"login\":");
        body.push_str(&json::quote(&input.login));
        body.push_str(",\"revision\":");
        body.push_str(&json::quote(&input.revision));
        body.push('}');
        let args = [
            "--remote=false",
            "exec",
            "--interactive",
            &cid,
            AGENT_PROGRAM,
            "keys",
        ];
        let data = self
            .podman(body.as_bytes(), &args, deadline)
            .map_err(|_| "native key operation not confirmed".to_string())?;
        let out = decode_access_key_state(&data)?;
        if input.apply && out.keys.join("\n") != keys.join("\n") {
            return Err("native key result differs from requested set".to_string());
        }
        Ok(out)
    }
}

#[cfg(test)]
mod access_keys_tests;
#[cfg(test)]
mod tests;

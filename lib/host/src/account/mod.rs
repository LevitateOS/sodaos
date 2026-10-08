//! Project account operations.
//!
//! Ports the account tail of `create.go` (`Account`, `canonicalizeAccountKeys`,
//! `confirmAccount`) and `access_keys.go` (`AccessKeys`, `canonicalKeys`,
//! `decodeAccessKeyState`). In-container key operations run through the fixed
//! `project-terminal` agent binary (`keys` action), like the Go `AgentExec`.

use std::time::Instant;

use crate::domain::{self, AccessKeyState, AccessKeys, Account, ProjectAccessRequest};
use crate::json::{self, SignedInteger};
use crate::project::{Executor, Runtime};
use crate::ssh;
use serde::de::{self, MapAccess, SeqAccess, Visitor};
use serde::{Deserialize, Serialize};
use std::fmt;

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
#[derive(Default)]
struct AccountConfirmation {
    login: String,
    identity: i64,
}

impl<'de> Deserialize<'de> for AccountConfirmation {
    fn deserialize<D>(deserializer: D) -> Result<Self, D::Error>
    where
        D: serde::Deserializer<'de>,
    {
        struct ConfirmationVisitor;
        impl<'de> Visitor<'de> for ConfirmationVisitor {
            type Value = AccountConfirmation;
            fn expecting(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
                f.write_str("an account confirmation object")
            }
            fn visit_map<A>(self, mut map: A) -> Result<Self::Value, A::Error>
            where
                A: MapAccess<'de>,
            {
                let mut out = AccountConfirmation::default();
                while let Some(key) = map.next_key::<String>()? {
                    if key.eq_ignore_ascii_case("login") {
                        if let Some(v) = map.next_value::<Option<String>>()? {
                            out.login = v;
                        }
                    } else if key.eq_ignore_ascii_case("identity") {
                        if let Some(v) = map.next_value::<Option<SignedInteger>>()? {
                            out.identity = v.0;
                        }
                    } else {
                        return Err(de::Error::unknown_field(&key, &["login", "identity"]));
                    }
                }
                Ok(out)
            }
        }
        deserializer.deserialize_map(ConfirmationVisitor)
    }
}

fn confirm_account(out: &[u8], login: &str, identity: i64) -> Result<(), String> {
    const ERR: &str = "native account identity was not confirmed";
    if out.len() > 4096 {
        return Err(ERR.to_string());
    }
    let confirmation: AccountConfirmation =
        json::decode_strict_as(out).map_err(|_| ERR.to_string())?;
    if confirmation.login != login || confirmation.identity != identity {
        return Err(ERR.to_string());
    }
    Ok(())
}

#[derive(Deserialize)]
#[serde(deny_unknown_fields)]
struct ProjectPrivilegeConfirmation {
    login: String,
    identity: i64,
    administrator: bool,
}

#[derive(Serialize)]
struct GuestPrivilegeRequest<'a> {
    login: &'a str,
    identity: i64,
}

/// Observe native Project administrator status through the fixed read-only
/// guest helper after binding the running container by its current ID.
pub(crate) fn observe_project_access<E: Executor>(
    runtime: &Runtime<E>,
    input: &ProjectAccessRequest,
    deadline: Instant,
) -> Result<domain::ProjectAccessStatus, String> {
    if !domain::valid_id(&input.project)
        || !domain::valid_login(&input.login)
        || input.login == "root"
        || input.identity <= 0
    {
        return Err("invalid project privilege observation".to_string());
    }
    let container = runtime.project_container(&input.project, true, deadline)?;
    let stdin = serde_json::to_vec(&GuestPrivilegeRequest {
        login: &input.login,
        identity: input.identity,
    })
    .map_err(|_| "invalid project privilege observation".to_string())?;
    let args = [
        "exec",
        "--interactive",
        container.as_str(),
        "/usr/libexec/soda/project-account",
        "--status",
    ];
    let out = runtime
        .exec
        .run_bounded(&stdin, "/usr/bin/podman", &args, deadline, 4096, 4096)?;
    const ERR: &str = "native project privilege observation was not confirmed";
    if out.len() > 4096 {
        return Err(ERR.to_string());
    }
    let observed: ProjectPrivilegeConfirmation =
        json::decode_strict_as(&out).map_err(|_| ERR.to_string())?;
    if observed.login != input.login || observed.identity != input.identity {
        return Err(ERR.to_string());
    }
    Ok(domain::ProjectAccessStatus {
        project: input.project.clone(),
        login: input.login.clone(),
        identity: input.identity,
        administrator: observed.administrator,
    })
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
struct AccountKeyList(Vec<String>);

impl<'de> Deserialize<'de> for AccountKeyList {
    fn deserialize<D>(deserializer: D) -> Result<Self, D::Error>
    where
        D: serde::Deserializer<'de>,
    {
        struct ListVisitor;
        impl<'de> Visitor<'de> for ListVisitor {
            type Value = AccountKeyList;
            fn expecting(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
                f.write_str("an array of strings")
            }
            fn visit_unit<E>(self) -> Result<Self::Value, E>
            where
                E: de::Error,
            {
                Err(E::custom("keys must be an array"))
            }
            fn visit_seq<A>(self, mut seq: A) -> Result<Self::Value, A::Error>
            where
                A: SeqAccess<'de>,
            {
                let mut out = Vec::new();
                while let Some(value) = seq.next_element::<Option<String>>()? {
                    out.push(value.unwrap_or_default());
                }
                Ok(AccountKeyList(out))
            }
        }
        deserializer.deserialize_any(ListVisitor)
    }
}

#[derive(Default)]
struct AccessKeyStateWire {
    revision: String,
    keys: Option<Vec<String>>,
}

impl<'de> Deserialize<'de> for AccessKeyStateWire {
    fn deserialize<D>(deserializer: D) -> Result<Self, D::Error>
    where
        D: serde::Deserializer<'de>,
    {
        struct StateVisitor;
        impl<'de> Visitor<'de> for StateVisitor {
            type Value = AccessKeyStateWire;
            fn expecting(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
                f.write_str("an access key state object")
            }
            fn visit_map<A>(self, mut map: A) -> Result<Self::Value, A::Error>
            where
                A: MapAccess<'de>,
            {
                let mut out = AccessKeyStateWire::default();
                while let Some(key) = map.next_key::<String>()? {
                    if key.eq_ignore_ascii_case("revision") {
                        if let Some(v) = map.next_value::<Option<String>>()? {
                            out.revision = v;
                        }
                    } else if key.eq_ignore_ascii_case("keys") {
                        if let Some(value) = map.next_value::<Option<AccountKeyList>>()? {
                            out.keys = Some(value.0);
                        }
                    } else {
                        map.next_value::<serde::de::IgnoredAny>()?;
                    }
                }
                Ok(out)
            }
        }
        deserializer.deserialize_map(StateVisitor)
    }
}

pub fn decode_access_key_state(data: &[u8]) -> Result<AccessKeyState, String> {
    if data.len() > 65536 {
        return Err("native key operation not confirmed".to_string());
    }
    const ERR: &str = "invalid native key observation";
    let state: AccessKeyStateWire = json::decode_tolerant_as(data).map_err(|_| ERR.to_string())?;
    let Some(keys) = state.keys else {
        return Err(ERR.to_string());
    };
    let revision = state.revision;
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

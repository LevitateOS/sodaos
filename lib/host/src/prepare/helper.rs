// In-container helper exchange: bounded calls, admission, approval.
use std::collections::HashMap;
use std::time::Instant;

use super::paths::{preparation_paths, prepare_id_map};
use super::state::quote_bytes_base64;
use crate::domain;
use crate::json::{self, Kind, Spec};
use crate::preparation::{self, Preparation};
use crate::project::{Executor, Runtime};

const FACTORY_INSPECT_LIMIT: usize = 262144;

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

impl<E: Executor> Runtime<E> {
    /// `factoryHelper`: bounded JSON exchange with the in-container helper.
    /// Failures pass through raw; oversized responses are refused.
    pub(crate) fn factory_helper(
        &self,
        id: &str,
        body: &str,
        deadline: Instant,
    ) -> Result<Vec<u8>, String> {
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

    /// `approvePreparation`: record approved inputs under the identity.
    pub(crate) fn approve_preparation(
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
}

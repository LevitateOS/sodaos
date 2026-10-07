// In-container helper exchange: bounded calls, admission, approval.
use std::collections::HashMap;
use std::time::Instant;

use super::paths::{preparation_paths, prepare_id_map};
use super::state::quote_bytes_base64;
use crate::domain;
use crate::json;
use crate::preparation::{self, Preparation};
use crate::project::{Executor, Runtime};
use serde::de::{self, MapAccess, Visitor};
use serde::Deserialize;
use std::fmt;

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

#[derive(Default)]
struct ProjectMappings {
    uid_map: Vec<String>,
    gid_map: Vec<String>,
}

impl<'de> Deserialize<'de> for ProjectMappings {
    fn deserialize<D>(deserializer: D) -> Result<Self, D::Error>
    where
        D: serde::Deserializer<'de>,
    {
        struct MappingsVisitor;
        impl<'de> Visitor<'de> for MappingsVisitor {
            type Value = ProjectMappings;
            fn expecting(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
                f.write_str("an ID mappings object")
            }
            fn visit_map<A>(self, mut map: A) -> Result<Self::Value, A::Error>
            where
                A: MapAccess<'de>,
            {
                let mut out = ProjectMappings::default();
                while let Some(key) = map.next_key::<String>()? {
                    if key.eq_ignore_ascii_case("UidMap") {
                        if let Some(v) = map.next_value::<Option<Vec<String>>>()? {
                            out.uid_map = v;
                        }
                    } else if key.eq_ignore_ascii_case("GidMap") {
                        if let Some(v) = map.next_value::<Option<Vec<String>>>()? {
                            out.gid_map = v;
                        }
                    } else {
                        return Err(de::Error::unknown_field(&key, &["UidMap", "GidMap"]));
                    }
                }
                Ok(out)
            }
        }
        deserializer.deserialize_map(MappingsVisitor)
    }
}

#[derive(Default)]
struct ProjectInspection {
    id: String,
    running: bool,
    project: String,
    owner: String,
    privileged: bool,
    userns: String,
    mappings: ProjectMappings,
}

impl<'de> Deserialize<'de> for ProjectInspection {
    fn deserialize<D>(deserializer: D) -> Result<Self, D::Error>
    where
        D: serde::Deserializer<'de>,
    {
        struct InspectionVisitor;
        impl<'de> Visitor<'de> for InspectionVisitor {
            type Value = ProjectInspection;
            fn expecting(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
                f.write_str("a project inspection object")
            }
            fn visit_map<A>(self, mut map: A) -> Result<Self::Value, A::Error>
            where
                A: MapAccess<'de>,
            {
                let mut out = ProjectInspection::default();
                while let Some(key) = map.next_key::<String>()? {
                    if key.eq_ignore_ascii_case("id") {
                        if let Some(v) = map.next_value::<Option<String>>()? {
                            out.id = v;
                        }
                    } else if key.eq_ignore_ascii_case("running") {
                        if let Some(v) = map.next_value::<Option<bool>>()? {
                            out.running = v;
                        }
                    } else if key.eq_ignore_ascii_case("project") {
                        if let Some(v) = map.next_value::<Option<String>>()? {
                            out.project = v;
                        }
                    } else if key.eq_ignore_ascii_case("owner") {
                        if let Some(v) = map.next_value::<Option<String>>()? {
                            out.owner = v;
                        }
                    } else if key.eq_ignore_ascii_case("privileged") {
                        if let Some(v) = map.next_value::<Option<bool>>()? {
                            out.privileged = v;
                        }
                    } else if key.eq_ignore_ascii_case("userns") {
                        if let Some(v) = map.next_value::<Option<String>>()? {
                            out.userns = v;
                        }
                    } else if key.eq_ignore_ascii_case("mappings") {
                        if let Some(v) = map.next_value::<Option<ProjectMappings>>()? {
                            out.mappings = v;
                        }
                    } else {
                        return Err(de::Error::unknown_field(
                            &key,
                            &[
                                "id",
                                "running",
                                "project",
                                "owner",
                                "privileged",
                                "userns",
                                "mappings",
                            ],
                        ));
                    }
                }
                Ok(out)
            }
        }
        deserializer.deserialize_map(InspectionVisitor)
    }
}

#[derive(Default)]
struct ApprovalResponse {
    approved: String,
    repeated: bool,
    checkout: String,
    credential_file: String,
}

impl<'de> Deserialize<'de> for ApprovalResponse {
    fn deserialize<D>(deserializer: D) -> Result<Self, D::Error>
    where
        D: serde::Deserializer<'de>,
    {
        struct ApprovalVisitor;
        impl<'de> Visitor<'de> for ApprovalVisitor {
            type Value = ApprovalResponse;
            fn expecting(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
                f.write_str("a preparation approval response")
            }
            fn visit_map<A>(self, mut map: A) -> Result<Self::Value, A::Error>
            where
                A: MapAccess<'de>,
            {
                let mut out = ApprovalResponse::default();
                while let Some(key) = map.next_key::<String>()? {
                    if key.eq_ignore_ascii_case("approved") {
                        if let Some(v) = map.next_value::<Option<String>>()? {
                            out.approved = v;
                        }
                    } else if key.eq_ignore_ascii_case("repeated") {
                        if let Some(v) = map.next_value::<Option<bool>>()? {
                            out.repeated = v;
                        }
                    } else if key.eq_ignore_ascii_case("checkout") {
                        if let Some(v) = map.next_value::<Option<String>>()? {
                            out.checkout = v;
                        }
                    } else if key.eq_ignore_ascii_case("credential_file") {
                        if let Some(v) = map.next_value::<Option<String>>()? {
                            out.credential_file = v;
                        }
                    } else {
                        return Err(de::Error::unknown_field(
                            &key,
                            &["approved", "repeated", "checkout", "credential_file"],
                        ));
                    }
                }
                Ok(out)
            }
        }
        deserializer.deserialize_map(ApprovalVisitor)
    }
}

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
        let inspected: ProjectInspection =
            json::decode_strict_as(&data).map_err(|_| "invalid preparation target".to_string())?;
        let ProjectInspection {
            id: cid,
            running,
            project,
            owner,
            privileged,
            userns,
            mappings,
        } = inspected;
        let uid_map = mappings.uid_map;
        let gid_map = mappings.gid_map;
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
        let response: ApprovalResponse =
            json::decode_strict_as(&raw).map_err(|_| ERR.to_string())?;
        if response.approved != prep.id {
            return Err(ERR.to_string());
        }
        if response.repeated {
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
        if response.checkout != checkout || response.credential_file != want_credential {
            return Err("preparation approval resolved unexpected paths".to_string());
        }
        Ok(())
    }
}

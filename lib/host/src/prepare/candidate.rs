// Protected candidate snapshot reads and candidate preparation.
use std::collections::HashMap;
use std::time::Instant;

use super::paths::path_join;
use crate::json;
use crate::preparation::{self, FactoryCandidate, Prepare, PrepareState};
use crate::project::{Executor, Runtime};
use serde::de::{self, MapAccess, Visitor};
use serde::Deserialize;
use std::fmt;

#[derive(Default)]
struct SavedRequest {
    id: String,
    role: String,
    setup_digest: String,
    source_commit: String,
    credential: String,
}

impl<'de> Deserialize<'de> for SavedRequest {
    fn deserialize<D>(deserializer: D) -> Result<Self, D::Error>
    where
        D: serde::Deserializer<'de>,
    {
        struct SavedVisitor;
        impl<'de> Visitor<'de> for SavedVisitor {
            type Value = SavedRequest;
            fn expecting(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
                f.write_str("a saved preparation request")
            }
            fn visit_map<A>(self, mut map: A) -> Result<Self::Value, A::Error>
            where
                A: MapAccess<'de>,
            {
                let mut out = SavedRequest::default();
                while let Some(key) = map.next_key::<String>()? {
                    if key.eq_ignore_ascii_case("id") {
                        if let Some(v) = map.next_value::<Option<String>>()? {
                            out.id = v;
                        }
                    } else if key.eq_ignore_ascii_case("role") {
                        if let Some(v) = map.next_value::<Option<String>>()? {
                            out.role = v;
                        }
                    } else if key.eq_ignore_ascii_case("setup_digest") {
                        if let Some(v) = map.next_value::<Option<String>>()? {
                            out.setup_digest = v;
                        }
                    } else if key.eq_ignore_ascii_case("source_commit") {
                        if let Some(v) = map.next_value::<Option<String>>()? {
                            out.source_commit = v;
                        }
                    } else if key.eq_ignore_ascii_case("credential") {
                        if let Some(v) = map.next_value::<Option<String>>()? {
                            out.credential = v;
                        }
                    } else {
                        return Err(de::Error::unknown_field(
                            &key,
                            &["id", "role", "setup_digest", "source_commit", "credential"],
                        ));
                    }
                }
                Ok(out)
            }
        }
        deserializer.deserialize_map(SavedVisitor)
    }
}

impl<E: Executor> Runtime<E> {
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
        let request: SavedRequest = json::decode_strict_as(&raw).map_err(|_| ERR.to_string())?;
        if request.id != input.source_preparation
            || request.role != input.preparation.role
            || request.setup_digest != input.preparation.setup_digest
            || request.source_commit != source_commit
            || request.credential != input.preparation.credential
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

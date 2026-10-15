// Protected candidate snapshot reads and candidate preparation.
use std::collections::HashMap;
use std::time::{Duration, Instant};

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
    /// Fresh role preparation reusing a ready preparation's protected setup.
    /// A coder Retry can copy only its protected original source bundle.
    pub fn prepare_candidate(
        &self,
        input: &FactoryCandidate,
        deadline: Instant,
    ) -> Result<PrepareState, String> {
        if input.source_run.is_some() {
            return Err("candidate source run requires protected native export".into());
        }
        self.prepare_candidate_inner(input, None, deadline)
    }

    /// Only the daemon's settled native export enters this path; request
    /// validation still forbids caller-provided coder bundle bytes.
    pub(crate) fn prepare_candidate_from_run(
        &self,
        input: &FactoryCandidate,
        bundle: &[u8],
        deadline: Instant,
    ) -> Result<PrepareState, String> {
        if input.source_run.is_none()
            || bundle.is_empty()
            || bundle.len() > preparation::MAX_SOURCE_BUNDLE
        {
            return Err("candidate source run export exceeds preparation bounds".into());
        }
        if input.preparation.requirements.source_inputs.is_empty() {
            return Err("accepted effective source input selection is unavailable".into());
        }
        self.prepare_candidate_inner(input, Some(bundle), deadline)
    }

    fn prepare_candidate_inner(
        &self,
        input: &FactoryCandidate,
        source_bundle: Option<&[u8]>,
        deadline: Instant,
    ) -> Result<PrepareState, String> {
        input.validate()?;
        let deadline = Self::cap_by_wire_deadline(deadline, &input.deadline)?;
        let container = self.prepare_container(&input.preparation.project, true, deadline)?;
        let source = self
            .inspect_preparation_state(
                &input.preparation.project,
                &input.source_preparation,
                &container,
                deadline,
                None,
            )
            .map_err(|error| error.to_string())?;
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
        let bundle = if let Some(bundle) = source_bundle {
            bundle.to_vec()
        } else if input.preparation.role == preparation::ROLE_CODER {
            if input.preparation.source_commit != source.source_commit {
                return Err("coder retry must retain its exact approved source".into());
            }
            let (_, _, bundle, _) = super::paths::preparation_paths(&source.role, &source.id);
            self.candidate_protected_file(
                &container,
                &bundle,
                preparation::MAX_SOURCE_BUNDLE,
                deadline,
            )?
        } else {
            input.bundle.clone()
        };
        self.prepare_inner(
            &Prepare {
                preparation: input.preparation.clone(),
                setup: preparation::ApprovedSetup { files, bundle },
            },
            deadline,
            Some(&input.deadline),
            source_bundle.map(|_| source.source_commit.as_str()),
        )
    }

    /// Verify the human-declared complete effective repository inputs at both
    /// exact commits before setup. No script or manifest discovery can broaden
    /// this selection. Protected setup bytes are checked independently.
    pub(super) fn confirm_unchanged_setup_inputs(
        &self,
        preparation: &preparation::Preparation,
        source: &str,
        files: &HashMap<String, Vec<u8>>,
        container: &str,
        deadline: Instant,
    ) -> Result<(), String> {
        let (checkout, _, _, home) =
            super::paths::preparation_paths(&preparation.role, &preparation.id);
        self.context_git(
            container,
            &preparation.role,
            &checkout,
            &home,
            &[
                "merge-base",
                "--is-ancestor",
                source,
                &preparation.source_commit,
            ],
            1,
            deadline,
        )?;
        let mut args = vec![
            "diff".to_string(),
            "--no-ext-diff".to_string(),
            "--no-textconv".to_string(),
            "--no-renames".to_string(),
            "--name-only".to_string(),
            "-z".to_string(),
            source.to_string(),
            preparation.source_commit.clone(),
            "--".to_string(),
        ];
        let mut selected: Vec<_> = files.keys().collect();
        selected.sort();
        args.extend(selected.into_iter().map(|name| format!(":(literal){name}")));
        let refs: Vec<_> = args.iter().map(String::as_str).collect();
        let changed = self.context_git(
            container,
            &preparation.role,
            &checkout,
            &home,
            &refs,
            1024,
            deadline,
        )?;
        if !changed.is_empty() {
            return Err("candidate changed selected approved setup inputs".into());
        }
        let mut selected: Vec<_> = preparation.requirements.source_inputs.iter().collect();
        selected.sort_by(|left, right| left.0.cmp(right.0));
        let mut approved_modes = HashMap::new();
        for commit in [
            &preparation.requirements.source_commit,
            &preparation.source_commit,
        ] {
            for (path, expected) in &selected {
                let literal = format!(":(literal){path}");
                let entry = self.context_git(
                    container,
                    &preparation.role,
                    &checkout,
                    &home,
                    &["ls-tree", "-z", commit, "--", &literal],
                    1024,
                    deadline,
                )?;
                let entry = std::str::from_utf8(&entry)
                    .map_err(|_| "approved source input is not a regular file")?;
                let (metadata, name) = entry
                    .strip_suffix('\0')
                    .and_then(|value| value.split_once('\t'))
                    .ok_or("approved source input is not a regular file")?;
                if name != *path
                    || !(metadata.starts_with("100644 blob ")
                        || metadata.starts_with("100755 blob "))
                {
                    return Err("approved source input is not a regular file".into());
                }
                let mode = metadata.split_whitespace().next().unwrap_or_default();
                if commit == &preparation.requirements.source_commit {
                    approved_modes.insert((*path).clone(), mode.to_string());
                } else if approved_modes.get(*path).map(String::as_str) != Some(mode) {
                    return Err("candidate changed accepted source input mode".into());
                }
                let object = format!("{commit}:{path}");
                let bytes = self.context_git(
                    container,
                    &preparation.role,
                    &checkout,
                    &home,
                    &["cat-file", "blob", &object],
                    preparation::MAX_APPROVED_FILE_SIZE,
                    deadline,
                )?;
                if crate::sha256::hex_lower(&crate::sha256::digest(&bytes)) != **expected {
                    return Err("candidate changed accepted effective source inputs".into());
                }
            }
        }
        Ok(())
    }

    pub(crate) fn cap_by_wire_deadline(
        operation_deadline: Instant,
        wire_deadline: &str,
    ) -> Result<Instant, String> {
        let absolute_nanos = soda_wire_time::parse_nanos(wire_deadline)
            .ok_or_else(|| "candidate preparation deadline is invalid".to_string())?;
        let now_wall = match std::time::SystemTime::now().duration_since(std::time::UNIX_EPOCH) {
            Ok(duration) => duration.as_nanos() as i128,
            Err(error) => -(error.duration().as_nanos() as i128),
        };
        let remaining_nanos = absolute_nanos - now_wall;
        if remaining_nanos <= 0 {
            return Err("candidate preparation deadline has expired".to_string());
        }
        let now = Instant::now();
        let operation_remaining = operation_deadline
            .saturating_duration_since(now)
            .as_nanos()
            .min(i128::MAX as u128) as i128;
        if operation_remaining == 0 {
            return Err("candidate preparation deadline has expired".to_string());
        }
        let bounded_nanos = remaining_nanos
            .min(operation_remaining)
            .min(u64::MAX as i128);
        let bounded = Duration::from_nanos(bounded_nanos as u64);
        Ok(operation_deadline.min(now + bounded))
    }

    /// `candidateProtectedFile`: one root-owned read-only regular file.
    pub(super) fn candidate_protected_file(
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
        self.approved_preparation_files(
            container,
            &input.source_preparation,
            &input.preparation.role,
            &input.preparation.setup_digest,
            Some(&input.preparation.credential),
            source_commit,
            deadline,
        )
    }

    pub(super) fn approved_preparation_files(
        &self,
        container: &str,
        preparation_id: &str,
        role: &str,
        setup_digest: &str,
        expected_credential: Option<&str>,
        source_commit: &str,
        deadline: Instant,
    ) -> Result<HashMap<String, Vec<u8>>, String> {
        let dir = path_join(&[preparation::FACTORY_PREPARATIONS_DIR, preparation_id]);
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
        if request.id != preparation_id
            || request.role != role
            || request.setup_digest != setup_digest
            || request.source_commit != source_commit
            || expected_credential.is_some_and(|credential| request.credential != credential)
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
        if preparation::setup_digest_of(&files) != setup_digest {
            return Err("candidate approved snapshot differs from its digest".to_string());
        }
        Ok(files)
    }
}

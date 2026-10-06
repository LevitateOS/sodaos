//! `ensure` and `approve`: role provisioning plus approved-input intake
//! (snapshot, bundle verification, credential binding).

use crate::account;
use crate::b64;
use crate::emit::{self, obj, str_value};
use crate::error::{fail, Error};
use crate::fsx;
use crate::ops_inspect;
use crate::sha;
use crate::validate;
use soda_json::JsonValue;
use std::os::unix::fs::DirBuilderExt;
use std::os::unix::fs::MetadataExt;
use std::os::unix::fs::OpenOptionsExt;
use std::os::unix::io::AsRawFd;
use std::path::{Path, PathBuf};

pub const MAX_APPROVED_FILES: usize = 8;
pub const MAX_APPROVED_FILE_SIZE: usize = 32 * 1024;
pub const MAX_APPROVED_TOTAL: usize = 128 * 1024;
pub const MAX_SOURCE_BUNDLE: usize = 512 * 1024;

/// Validated `request.json` fields, in file order.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct ReqFields {
    pub id: String,
    pub role: String,
    pub setup_digest: String,
    pub source_commit: String,
    pub credential: String,
}

impl ReqFields {
    pub fn to_json(&self) -> JsonValue {
        obj(vec![
            ("id", str_value(&self.id)),
            ("role", str_value(&self.role)),
            ("setup_digest", str_value(&self.setup_digest)),
            ("source_commit", str_value(&self.source_commit)),
            ("credential", str_value(&self.credential)),
        ])
    }

    /// Strict re-validation of stored fields (the helper wrote them, so
    /// this only trips on root-corrupted state).
    pub fn from_stored(value: &JsonValue) -> Result<ReqFields, Error> {
        let entries =
            validate::as_object(value).ok_or_else(|| Error::fail("unsafe factory metadata"))?;
        if !validate::key_set(
            entries,
            &["id", "role", "setup_digest", "source_commit", "credential"],
        ) {
            return Err(Error::fail("unsafe factory metadata"));
        }
        let get = |key: &str| value.get(key).and_then(|v| v.as_str()).unwrap_or("");
        let fields = ReqFields {
            id: get("id").to_string(),
            role: get("role").to_string(),
            setup_digest: get("setup_digest").to_string(),
            source_commit: get("source_commit").to_string(),
            credential: get("credential").to_string(),
        };
        if !validate::is_id(&fields.id)
            || !validate::is_role(&fields.role)
            || !validate::is_digest(&fields.setup_digest)
            || !validate::is_commit(&fields.source_commit)
            || (!fields.credential.is_empty() && !validate::is_name(&fields.credential))
        {
            return Err(Error::fail("unsafe factory metadata"));
        }
        Ok(fields)
    }
}

pub fn do_ensure(ctx: &crate::Ctx, data: &JsonValue) -> Result<JsonValue, Error> {
    if !validate::as_object(data).is_some_and(|e| validate::key_set(e, &["op"])) {
        return fail("unsupported ensure request");
    }
    fsx::ensure_layout(ctx)?;
    let mut roles = Vec::new();
    for login in validate::ROLES {
        roles.push(str_value(&account::ensure_role(ctx, login)?.name));
    }
    Ok(obj(vec![("roles", JsonValue::Array(roles))]))
}

/// Last-wins collapse of a JSON object, like `json.loads` into a dict.
fn collapse(entries: &[(String, JsonValue)]) -> Vec<(&str, &JsonValue)> {
    let mut out: Vec<(&str, &JsonValue)> = Vec::new();
    for (key, value) in entries {
        if let Some(slot) = out.iter_mut().find(|(k, _)| *k == key) {
            slot.1 = value;
        } else {
            out.push((key, value));
        }
    }
    out
}

/// Bounded approved-file intake: both entry points required, per-file and
/// total caps enforced.
pub fn decode_files(raw: &JsonValue) -> Result<Vec<(String, Vec<u8>)>, Error> {
    let entries = match raw {
        JsonValue::Object(entries) => entries,
        _ => return fail("unsupported approved file set"),
    };
    let files_in = collapse(entries);
    if files_in.is_empty() || files_in.len() > MAX_APPROVED_FILES {
        return fail("unsupported approved file set");
    }
    if !files_in.iter().any(|(k, _)| *k == crate::SETUP_ENTRY)
        || !files_in.iter().any(|(k, _)| *k == crate::CHECK_ENTRY)
    {
        return fail("approved entrypoints are required");
    }
    let mut total = 0usize;
    let mut files = Vec::new();
    for (name, encoded) in files_in {
        validate::check_name_str(name)?;
        let text = match encoded.as_str() {
            Some(text) => text,
            None => return fail("unsupported approved file encoding"),
        };
        let contents =
            b64::decode(text).ok_or_else(|| Error::fail("unsupported approved file encoding"))?;
        if contents.is_empty() || contents.len() > MAX_APPROVED_FILE_SIZE {
            return fail("unsupported approved file size");
        }
        total += contents.len();
        files.push((name.to_string(), contents));
    }
    if total > MAX_APPROVED_TOTAL {
        return fail("approved inputs exceed the bounded size");
    }
    Ok(files)
}

/// Exact `git clone` recipe, kept pure so tests pin the argv.
pub fn git_clone_argv(git: &str, bundle: &Path, dest: &Path) -> Vec<String> {
    [
        git,
        "clone",
        "-q",
        "--no-checkout",
        &bundle.to_string_lossy(),
        &dest.to_string_lossy(),
    ]
    .iter()
    .map(|word| word.to_string())
    .collect()
}

/// Exact `git cat-file` recipe, kept pure so tests pin the argv.
pub fn git_catfile_argv(git: &str, repo: &Path, commit: &str) -> Vec<String> {
    [git, "-C", &repo.to_string_lossy(), "cat-file", "-e", commit]
        .iter()
        .map(|word| word.to_string())
        .collect()
}

/// Clone the bundle into a scratch dir and prove it carries the commit.
/// The scratch dir is always removed, like the `.py` `finally` arm.
pub fn verify_bundle(ctx: &crate::Ctx, snapshot: &Path, commit: &str) -> Result<(), Error> {
    let work = snapshot.join("verify-tmp");
    match std::fs::symlink_metadata(&work) {
        Ok(meta) => {
            if meta.file_type().is_symlink() {
                // `shutil.rmtree` raises on a symlink; same failure here.
                return Err(Error::io_msg("verify scratch is a link"));
            }
            if std::fs::remove_dir_all(&work).is_err() {
                return Err(Error::io_msg("verify scratch busy"));
            }
        }
        Err(err) if err.kind() == std::io::ErrorKind::NotFound => {}
        Err(err) => return Err(Error::classify(err)),
    }
    std::fs::DirBuilder::new()
        .mode(0o700)
        .create(&work)
        .map_err(Error::classify)?;
    let git = ctx.git().to_string_lossy().into_owned();
    let result: Result<(), Error> = (|| {
        account::run_check(&git_clone_argv(
            &git,
            &snapshot.join("source.bundle"),
            &work.join("repo"),
        ))?;
        account::run_check(&git_catfile_argv(&git, &work.join("repo"), commit))?;
        Ok(())
    })();
    let _ = std::fs::remove_dir_all(&work);
    result.map_err(|_| Error::fail("source bundle does not carry the approved commit"))
}

/// Role-private credential binding: directory owned by the helper, file
/// owned by the role with mode `0600`, opened `O_NOFOLLOW`.
pub fn check_credential_file(ctx: &crate::Ctx, role: &str, name: &str) -> Result<String, Error> {
    validate::check_name_str(name)?;
    let account = account::role_record(ctx, role)?
        .ok_or_else(|| Error::fail("assigned service credential is missing"))?;
    let parent = ctx.credentials.join(role);
    let meta = match std::fs::symlink_metadata(&parent) {
        Ok(meta) => meta,
        Err(err) if err.kind() == std::io::ErrorKind::NotFound => {
            return fail("assigned service credential is missing");
        }
        Err(err) => return Err(Error::classify(err)),
    };
    if !meta.is_dir() || meta.uid() != ctx.priv_uid() {
        return fail("unsafe service credential directory");
    }
    let path = parent.join(name);
    let file = match fsx::open_ro(&path, true) {
        Ok(file) => file,
        Err(Error::Missing) => return fail("assigned service credential is missing"),
        Err(err) => return Err(err),
    };
    let meta = file.metadata().map_err(Error::classify)?;
    if !meta.is_file() || meta.uid() != account.uid || meta.mode() & 0o777 != 0o600 {
        return fail("assigned service credential is not role-private");
    }
    Ok(path.to_string_lossy().into_owned())
}

pub struct ApprovedInputs {
    pub fields: ReqFields,
    pub files: Vec<(String, Vec<u8>)>,
    pub bundle: Vec<u8>,
}

/// Pure request validation: no effects before every refusal, so rejected
/// inputs touch neither accounts nor git.
pub fn approve_inputs(data: &JsonValue) -> Result<ApprovedInputs, Error> {
    if !validate::as_object(data).is_some_and(|e| {
        validate::key_set(
            e,
            &[
                "op",
                "id",
                "role",
                "setup_digest",
                "source_commit",
                "files",
                "bundle",
                "credential",
            ],
        )
    }) {
        return fail("unsupported approve request");
    }
    let pid = validate::check_id(data.get("id").unwrap_or(&JsonValue::Null))?.to_string();
    let role = validate::check_role(data.get("role").unwrap_or(&JsonValue::Null))?.to_string();
    let setup_digest =
        validate::check_digest(data.get("setup_digest").unwrap_or(&JsonValue::Null))?.to_string();
    let source_commit =
        validate::check_commit(data.get("source_commit").unwrap_or(&JsonValue::Null))?.to_string();
    let credential = match data.get("credential").and_then(|v| v.as_str()) {
        Some(text) => text,
        None => return fail("unsupported credential reference"),
    }
    .to_string();
    if !credential.is_empty() {
        validate::check_name_str(&credential)?;
    }
    let bundle_text = match data.get("bundle").and_then(|v| v.as_str()) {
        Some(text) => text,
        None => return fail("unsupported source bundle encoding"),
    };
    let bundle = b64::decode(bundle_text)
        .ok_or_else(|| Error::fail("unsupported source bundle encoding"))?;
    if bundle.is_empty() || bundle.len() > MAX_SOURCE_BUNDLE {
        return fail("unsupported source bundle size");
    }
    let files = decode_files(data.get("files").unwrap_or(&JsonValue::Null))?;
    if sha::approved_digest(&files) != setup_digest {
        return fail("approved inputs do not match their digest");
    }
    Ok(ApprovedInputs {
        fields: ReqFields {
            id: pid,
            role,
            setup_digest,
            source_commit,
            credential,
        },
        files,
        bundle,
    })
}

/// CODEX-P07-003b: open a directory by path without following a trailing
/// symlink and refusing non-directories. The returned descriptor pins the
/// opened directory's identity for fd-bound chmod/chown/removal.
fn open_dir_no_follow(path: &Path) -> Result<std::fs::File, Error> {
    std::fs::OpenOptions::new()
        .read(true)
        .custom_flags(libc::O_DIRECTORY | libc::O_NOFOLLOW)
        .open(path)
        .map_err(Error::classify)
}

/// fchmod/fchown bound to the retained descriptor: never follows a
/// swapped pathname.
fn fchmod(file: &std::fs::File, mode: u32) -> Result<(), Error> {
    if unsafe { libc::fchmod(file.as_raw_fd(), mode) } != 0 {
        return Err(Error::classify(std::io::Error::last_os_error()));
    }
    Ok(())
}

fn fchown(file: &std::fs::File, uid: u32, gid: u32) -> Result<(), Error> {
    if unsafe { libc::fchown(file.as_raw_fd(), uid, gid) } != 0 {
        return Err(Error::classify(std::io::Error::last_os_error()));
    }
    Ok(())
}

/// True only when the retained descriptor's directory holds no entries.
/// Any read failure reports uncertainty (`None`): the caller preserves.
fn dir_is_empty(file: &std::fs::File) -> Option<bool> {
    // Reopening through /proc pins the same directory even if the
    // pathname was swapped; read_dir never yields `.`/`..`.
    let entries = std::fs::read_dir(format!("/proc/self/fd/{}", file.as_raw_fd())).ok()?;
    for entry in entries {
        entry.ok()?;
        return Some(false);
    }
    Some(true)
}

/// Remove `name` from the retained parent when — and only when — the
/// retained child is verifiably empty. Anything else (content, unreadable
/// state, a failed removal) preserves whatever the pathname holds now;
/// only an empty directory can ever go.
fn remove_if_empty_owned(parent: &std::fs::File, child: &std::fs::File, name: &str) {
    if dir_is_empty(child) != Some(true) {
        return;
    }
    let cname = match std::ffi::CString::new(name) {
        Ok(cname) => cname,
        Err(_) => return,
    };
    unsafe {
        libc::unlinkat(parent.as_raw_fd(), cname.as_ptr(), libc::AT_REMOVEDIR);
    }
}

/// Root-owned read-only snapshot, verified bundle, role-owned checkout,
/// credential binding, then the request receipt. The clone always starts
/// from an empty directory; any failure removes invocation-created state.
/// Checkout creation is exclusive, so a preexisting path of any identity
/// is refused untouched. Permission and cleanup steps bind the created
/// directory's descriptor; replaced or uncertain state is preserved and
/// reported as failure.
pub fn write_snapshot(
    ctx: &crate::Ctx,
    directory: &Path,
    inputs: &ApprovedInputs,
    account: &account::Account,
) -> Result<(PathBuf, String), Error> {
    let snapshot = directory.join("snapshot");
    std::fs::DirBuilder::new()
        .mode(0o755)
        .create(&snapshot)
        .map_err(Error::classify)?;
    for (name, contents) in &inputs.files {
        fsx::write_new(&snapshot.join(name), contents, 0o644)?;
    }
    fsx::write_new(&snapshot.join("source.bundle"), &inputs.bundle, 0o644)?;
    verify_bundle(ctx, &snapshot, &inputs.fields.source_commit)?;
    let checkout = account.dir.join("checkouts").join(&inputs.fields.id);
    // CODEX-P07-003: exclusive creation is the provenance record. Any
    // preexisting path — present before or planted during verification,
    // of any identity — is refused untouched; only a directory this call
    // created is ever removed, by the tail scope below.
    match std::fs::create_dir(&checkout) {
        Ok(()) => {}
        Err(err) if err.kind() == std::io::ErrorKind::AlreadyExists => {
            return fail("checkout path already exists");
        }
        Err(err) => return Err(Error::classify(err)),
    }
    // CODEX-P07-003b: bind the created directory's identity before any
    // permission or removal step. The checkouts parent is role-owned, so
    // the name can be swapped under us; pathname chmod/chown/removal
    // would follow the swap. If stable ownership cannot be established,
    // the checkout is preserved and failure is returned.
    let parent_path = checkout
        .parent()
        .ok_or_else(|| Error::io_msg("checkout has no parent"))?;
    let parent = open_dir_no_follow(parent_path)?;
    let child = open_dir_no_follow(&checkout)?;
    let tail: Result<String, Error> = (|| {
        fchmod(&child, 0o755)?;
        fchown(&child, account.uid, account.gid)?;
        let mut credential_path = String::new();
        if !inputs.fields.credential.is_empty() {
            credential_path =
                check_credential_file(ctx, &inputs.fields.role, &inputs.fields.credential)?;
        }
        let receipt = emit::dumps_default(&inputs.fields.to_json());
        fsx::write_new(&directory.join("request.json"), receipt.as_bytes(), 0o644)?;
        Ok(credential_path)
    })();
    match tail {
        Ok(credential_path) => Ok((checkout, credential_path)),
        Err(err) => {
            remove_if_empty_owned(&parent, &child, &inputs.fields.id);
            Err(err)
        }
    }
}

pub fn do_approve(ctx: &crate::Ctx, data: &JsonValue) -> Result<JsonValue, Error> {
    let inputs = approve_inputs(data)?;
    fsx::ensure_layout(ctx)?;
    let directory = fsx::prep_dir(ctx, &inputs.fields.id)?;
    ops_inspect::refuse_barred(ctx, &directory)?;
    let saved = match fsx::read_json(ctx, &directory.join("request.json"), 4096) {
        Ok(value) => Some(value),
        Err(Error::Missing) => None,
        Err(err) => return Err(err),
    };
    if let Some(saved) = saved {
        if !emit::json_equal(&saved, &inputs.fields.to_json()) {
            return fail("preparation identity already carries different approved inputs");
        }
        return Ok(obj(vec![
            ("approved", str_value(&inputs.fields.id)),
            ("repeated", JsonValue::Bool(true)),
        ]));
    }
    let account = account::ensure_role(ctx, &inputs.fields.role)?;
    std::fs::DirBuilder::new()
        .mode(0o755)
        .create(&directory)
        .map_err(Error::classify)?;
    match write_snapshot(ctx, &directory, &inputs, &account) {
        Ok((checkout, credential_path)) => Ok(obj(vec![
            ("approved", str_value(&inputs.fields.id)),
            ("repeated", JsonValue::Bool(false)),
            ("checkout", str_value(&checkout.to_string_lossy())),
            ("credential_file", str_value(&credential_path)),
        ])),
        Err(err) => {
            // CODEX-P07-003: only the invocation-owned preparation
            // directory is removed here; checkout cleanup belongs to the
            // scope that created it, inside `write_snapshot`.
            let _ = std::fs::remove_dir_all(&directory);
            Err(err)
        }
    }
}

#[cfg(test)]
#[path = "inputs_tests.rs"]
mod tests;

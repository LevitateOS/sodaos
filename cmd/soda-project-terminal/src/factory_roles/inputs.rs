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
use std::os::unix::io::{AsRawFd, FromRawFd};
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

/// NUL-free checkout name for descriptor-relative calls. Validated ids
/// (`f` + 24 hex) can never contain NUL; anything else is refused rather
/// than truncated or reinterpreted.
fn checkout_cname(name: &str) -> Result<std::ffi::CString, Error> {
    std::ffi::CString::new(name).map_err(|_| Error::fail("unsafe checkout name"))
}

/// Exclusive publication of the checkout name under the pinned parent.
/// Any preexisting identity fails untouched with the established error.
fn mkdirat_exclusive(parent: &std::fs::File, name: &str, mode: u32) -> Result<(), Error> {
    let cname = checkout_cname(name)?;
    if unsafe { libc::mkdirat(parent.as_raw_fd(), cname.as_ptr(), mode) } != 0 {
        let err = std::io::Error::last_os_error();
        if err.kind() == std::io::ErrorKind::AlreadyExists {
            return fail("checkout path already exists");
        }
        return Err(Error::classify(err));
    }
    Ok(())
}

/// Bind the published name to a descriptor without following a trailing
/// symlink and refusing non-directories. Close-on-exec matches `std` so
/// git children never inherit the descriptor.
fn openat_dir_no_follow(parent: &std::fs::File, name: &str) -> Result<std::fs::File, Error> {
    let cname = checkout_cname(name)?;
    let fd = unsafe {
        libc::openat(
            parent.as_raw_fd(),
            cname.as_ptr(),
            libc::O_RDONLY | libc::O_DIRECTORY | libc::O_NOFOLLOW | libc::O_CLOEXEC,
        )
    };
    if fd < 0 {
        return Err(Error::classify(std::io::Error::last_os_error()));
    }
    Ok(unsafe { std::fs::File::from_raw_fd(fd) })
}

/// CODEX-P07-003b CORRECTION-05 admission gate: the bound descriptor must
/// be this call's created directory — a directory, owned by the privileged
/// creator, and empty. Exclusive mkdir proves the name was absent; only the
/// creator makes creator-owned entries, so a swapped-in role directory is
/// refused before any privileged mutation. A name↔descriptor comparison
/// cannot help here: after a create→open swap the two agree by
/// construction; creation ownership is the only proof.
fn checkout_created(child: &std::fs::File, priv_uid: u32) -> bool {
    let meta = match child.metadata() {
        Ok(meta) => meta,
        Err(_) => return false,
    };
    meta.is_dir() && meta.uid() == priv_uid && dir_is_empty(child) == Some(true)
}

/// Publication/cleanup gate: the published name must still resolve to the
/// bound descriptor (same device+inode, a directory — never a link) and the
/// directory must still be empty; nothing legitimate lands in it before
/// handoff. Any replacement, removal, plant, or unreadable state fails and
/// the caller preserves. Descriptor-relative calls keep the residual
/// check→use sliver to adjacent syscalls under the pinned parent; a
/// pathname lstat here would re-resolve the role-mutable parent chain.
fn checkout_intact(parent: &std::fs::File, child: &std::fs::File, name: &str) -> bool {
    let cname = match checkout_cname(name) {
        Ok(cname) => cname,
        Err(_) => return false,
    };
    let mut at_name: libc::stat = unsafe { std::mem::zeroed() };
    if unsafe {
        libc::fstatat(
            parent.as_raw_fd(),
            cname.as_ptr(),
            &mut at_name,
            libc::AT_SYMLINK_NOFOLLOW,
        )
    } != 0
    {
        return false;
    }
    if at_name.st_mode & libc::S_IFMT != libc::S_IFDIR {
        return false;
    }
    let mut at_child: libc::stat = unsafe { std::mem::zeroed() };
    if unsafe { libc::fstat(child.as_raw_fd(), &mut at_child) } != 0 {
        return false;
    }
    if at_name.st_dev != at_child.st_dev || at_name.st_ino != at_child.st_ino {
        return false;
    }
    dir_is_empty(child) == Some(true)
}

/// Remove the gate-verified empty name. Returns false only when the removal
/// itself voids the verified binding (gone or planted meanwhile): ownership
/// is then unestablishable and the caller reports the missing precondition.
/// Mechanical failures leave the binding intact, so the original error
/// stands and the preserved directory waits for operator inspection.
fn remove_verified_empty(parent: &std::fs::File, name: &str) -> bool {
    let cname = match checkout_cname(name) {
        Ok(cname) => cname,
        Err(_) => return false,
    };
    if unsafe { libc::unlinkat(parent.as_raw_fd(), cname.as_ptr(), libc::AT_REMOVEDIR) } == 0 {
        return true;
    }
    match std::io::Error::last_os_error().raw_os_error() {
        Some(code) if code == libc::ENOENT || code == libc::ENOTEMPTY => false,
        _ => true,
    }
}

/// Root-owned read-only snapshot, verified bundle, role-owned checkout,
/// credential binding, then the request receipt. The clone always starts
/// from an empty directory; any failure removes invocation-created state.
/// Checkout creation is exclusive under a pinned parent descriptor, so a
/// preexisting path of any identity is refused untouched. Permission,
/// publication, and cleanup steps each verify the created directory's
/// binding first; replaced or uncertain state is preserved and reported
/// as an unestablished-ownership failure.
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
    // of any identity — is refused untouched; removal below only ever
    // targets a name verified bound to this call's created directory.
    // CODEX-P07-003b CORRECTION-05: pin the parent descriptor first; the
    // exclusive mkdir, the bind, every gate, and the removal are all
    // descriptor-relative, so the role-mutable namespace cannot redirect
    // any step across calls. If stable ownership cannot be established
    // at any gate, the checkout is preserved and failure names the
    // missing precondition instead of the untrustworthy tail state.
    let parent_path = checkout
        .parent()
        .ok_or_else(|| Error::io_msg("checkout has no parent"))?;
    let parent = open_dir_no_follow(parent_path)?;
    mkdirat_exclusive(&parent, &inputs.fields.id, 0o755)?;
    let child = openat_dir_no_follow(&parent, &inputs.fields.id)?;
    if !checkout_created(&child, ctx.priv_uid()) {
        return fail("checkout ownership unestablished");
    }
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
        Ok(credential_path) => {
            if !checkout_intact(&parent, &child, &inputs.fields.id) {
                return fail("checkout ownership unestablished");
            }
            Ok((checkout, credential_path))
        }
        Err(err) => {
            if !checkout_intact(&parent, &child, &inputs.fields.id) {
                return fail("checkout ownership unestablished");
            }
            if remove_verified_empty(&parent, &inputs.fields.id) {
                Err(err)
            } else {
                fail("checkout ownership unestablished")
            }
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

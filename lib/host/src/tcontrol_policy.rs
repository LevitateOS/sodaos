//! Tailnet locked policy files: enrollment and project policy storage.
//!
//! PR26 port of `internal/tailnet/policy.go` (locked-file plumbing),
//! `policy_enrollment.go`, `policy_project.go`, and `RunBinding`
//! (`project_runtime.go`). One stable directory lock guards the policy
//! directory; production anchors at `/var/lib/soda-tailnet` (tests supply a
//! private directory, never a request path).
//!
//! File access is relative to an open directory fd (`openat` with
//! `O_NOFOLLOW`), mirroring the `os.Root` pinning in Go; all names below are
//! fixed or validated (`project-<p[0-9a-f]{24}>.json`), so no intermediate
//! path components can escape.

use std::ffi::CString;
use std::fs::File;
use std::os::unix::fs::{MetadataExt, PermissionsExt};
use std::os::unix::io::{AsRawFd, FromRawFd};
use std::path::{Path, PathBuf};
use std::time::{Duration, Instant};

use crate::json::{Kind, Spec};
use crate::tailnet_domain;
use crate::tcontrol_wire as wire;

const POLICY_VERSION: i64 = 2;
const PROJECT_VERSION: i64 = 1;
const POLICY_FILE_LIMIT: u64 = 65536;

/// A stored OAuth client credential. `Debug` redacts the secret.
#[derive(Clone, Default, PartialEq, Eq)]
pub struct Credential {
    pub client_id: String,
    pub secret: String,
}

impl std::fmt::Debug for Credential {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        f.debug_struct("Credential")
            .field("client_id", &self.client_id)
            .field("secret", &"[redacted]")
            .finish()
    }
}

impl Credential {
    pub fn zero_secret(&mut self) {
        wire::zero_string(&mut self.secret);
    }
}

/// Stored enrollment policy (`policy.json`). `Debug` redacts the credential.
#[derive(Clone, Default, PartialEq, Eq)]
pub struct EnrollmentPolicy {
    pub version: i64,
    pub revision: String,
    pub binding: String,
    pub tailnet: String,
    pub tags: Vec<String>,
    pub preauthorized: bool,
    pub admission: bool,
    pub default: bool,
    pub credential: Credential,
}

impl std::fmt::Debug for EnrollmentPolicy {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        f.debug_struct("EnrollmentPolicy")
            .field("version", &self.version)
            .field("revision", &self.revision)
            .field("binding", &self.binding)
            .field("tailnet", &self.tailnet)
            .field("tags", &self.tags)
            .field("preauthorized", &self.preauthorized)
            .field("admission", &self.admission)
            .field("default", &self.default)
            .field("credential", &self.credential)
            .finish()
    }
}

/// Stored project policy (`project-<id>.json`).
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct ProjectPolicyEntry {
    pub version: i64,
    pub revision: String,
    pub project: String,
    pub container: String,
    pub binding: String,
    pub enabled: bool,
}

// ---------- Policy file codecs ----------

const CREDENTIAL_SPECS: &[Spec] = &[
    Spec {
        name: "client_id",
        kind: Kind::Str,
    },
    Spec {
        name: "secret",
        kind: Kind::Str,
    },
];

const ENROLLMENT_POLICY_SPECS: &[Spec] = &[
    Spec {
        name: "version",
        kind: Kind::Int,
    },
    Spec {
        name: "revision",
        kind: Kind::Str,
    },
    Spec {
        name: "binding",
        kind: Kind::Str,
    },
    Spec {
        name: "tailnet",
        kind: Kind::Str,
    },
    Spec {
        name: "tags",
        kind: Kind::StrList,
    },
    Spec {
        name: "preauthorized",
        kind: Kind::Bool,
    },
    Spec {
        name: "admission",
        kind: Kind::Bool,
    },
    Spec {
        name: "default",
        kind: Kind::Bool,
    },
    Spec {
        name: "credential",
        kind: Kind::Object {
            go_type: "struct",
            struct_name: "credential",
            specs: CREDENTIAL_SPECS,
        },
    },
];

const PROJECT_POLICY_SPECS: &[Spec] = &[
    Spec {
        name: "version",
        kind: Kind::Int,
    },
    Spec {
        name: "revision",
        kind: Kind::Str,
    },
    Spec {
        name: "project",
        kind: Kind::Str,
    },
    Spec {
        name: "container",
        kind: Kind::Str,
    },
    Spec {
        name: "binding",
        kind: Kind::Str,
    },
    Spec {
        name: "enabled",
        kind: Kind::Bool,
    },
];

fn decode_policy_file(
    body: &[u8],
    specs: &[Spec],
    name: &'static str,
) -> Result<crate::json::BoundMap, String> {
    // Store files are small trusted-local JSON, but decode them with the
    // same strict grammar as the wire (Go uses strictjson here too).
    let v = crate::json::decode_strict(body).map_err(|_| wire::err_unavailable())?;
    crate::json::bind_root(&v, name, specs, false).map_err(|_| wire::err_unavailable())
}

fn decode_enrollment_policy(body: &[u8]) -> Result<EnrollmentPolicy, String> {
    let bound = decode_policy_file(body, ENROLLMENT_POLICY_SPECS, "policy")?;
    let cred = bound.take_map("credential");
    Ok(EnrollmentPolicy {
        version: bound.take_i64("version"),
        revision: bound.take_string("revision"),
        binding: bound.take_string("binding"),
        tailnet: bound.take_string("tailnet"),
        tags: bound.take_str_list("tags"),
        preauthorized: bound.take_bool("preauthorized"),
        admission: bound.take_bool("admission"),
        default: bound.take_bool("default"),
        credential: Credential {
            client_id: cred.take_string("client_id"),
            secret: cred.take_string("secret"),
        },
    })
}

fn encode_enrollment_policy(v: &EnrollmentPolicy) -> Vec<u8> {
    let q = crate::json::quote;
    let mut tags = String::from("[");
    for (i, t) in v.tags.iter().enumerate() {
        if i > 0 {
            tags.push(',');
        }
        tags.push_str(&q(t));
    }
    tags.push(']');
    format!(
        "{{\"version\":{},\"revision\":{},\"binding\":{},\"tailnet\":{},\"tags\":{},\"preauthorized\":{},\"admission\":{},\"default\":{},\"credential\":{{\"client_id\":{},\"secret\":{}}}}}",
        v.version,
        q(&v.revision),
        q(&v.binding),
        q(&v.tailnet),
        tags,
        v.preauthorized,
        v.admission,
        v.default,
        q(&v.credential.client_id),
        q(&v.credential.secret),
    )
    .into_bytes()
}

fn decode_project_entry(body: &[u8]) -> Result<ProjectPolicyEntry, String> {
    let bound = decode_policy_file(body, PROJECT_POLICY_SPECS, "project")?;
    Ok(ProjectPolicyEntry {
        version: bound.take_i64("version"),
        revision: bound.take_string("revision"),
        project: bound.take_string("project"),
        container: bound.take_string("container"),
        binding: bound.take_string("binding"),
        enabled: bound.take_bool("enabled"),
    })
}

fn encode_project_entry(v: &ProjectPolicyEntry) -> Vec<u8> {
    let q = crate::json::quote;
    format!(
        "{{\"version\":{},\"revision\":{},\"project\":{},\"container\":{},\"binding\":{},\"enabled\":{}}}",
        v.version,
        q(&v.revision),
        q(&v.project),
        q(&v.container),
        q(&v.binding),
        v.enabled,
    )
    .into_bytes()
}

// ---------- Random revisions ----------

/// Fresh 128-bit hex revision.
pub fn new_revision() -> Result<String, String> {
    new_revision_with(|out| getrandom::fill(out).map_err(std::io::Error::other))
}

fn new_revision_with(
    fill: impl FnOnce(&mut [u8]) -> std::io::Result<()>,
) -> Result<String, String> {
    let mut buf = [0u8; 16];
    fill(&mut buf).map_err(|_| wire::err_unavailable())?;
    let mut out = String::with_capacity(32);
    for b in buf {
        out.push_str(&format!("{b:02x}"));
    }
    Ok(out)
}

// ---------- Locked directory plumbing ----------

/// An exclusive advisory lock on the policy directory. Closing the file
/// releases the lock, like Go's deferred `Close` calls.
pub struct PolicyLock {
    dir: File,
}

impl PolicyLock {
    fn fd(&self) -> std::os::unix::io::RawFd {
        self.dir.as_raw_fd()
    }
}

fn dir_owned(meta: &std::fs::Metadata, uid: u32) -> bool {
    meta.is_dir() && meta.uid() == uid && meta.mode() & 0o077 == 0
}

fn file_owned(meta: &std::fs::Metadata, uid: u32) -> bool {
    meta.is_file()
        && meta.uid() == uid
        && meta.mode() & 0o077 == 0
        && meta.nlink() == 1
        && meta.size() <= POLICY_FILE_LIMIT
}

fn last_errno() -> i32 {
    std::io::Error::last_os_error()
        .raw_os_error()
        .unwrap_or(libc::EIO)
}

fn acquire_exclusive(dir: &File, deadline: Instant) -> Result<(), String> {
    // Mirror of `filelock.Acquire`: non-blocking attempts on a 25 ms tick so
    // an expired deadline never traps us in a blocking `flock` call. Every
    // failure (including expiry) is unavailable, like Go's `acquirePolicyLock`.
    loop {
        if Instant::now() >= deadline {
            return Err(wire::err_unavailable());
        }
        // SAFETY: `flock` on a live directory fd.
        let r = unsafe { libc::flock(dir.as_raw_fd(), libc::LOCK_EX | libc::LOCK_NB) };
        if r == 0 {
            return Ok(());
        }
        match last_errno() {
            e if e == libc::EWOULDBLOCK || e == libc::EINTR => {}
            _ => return Err(wire::err_unavailable()),
        }
        if Instant::now() >= deadline {
            return Err(wire::err_unavailable());
        }
        std::thread::sleep(Duration::from_millis(25));
    }
}

fn cstr(name: &str) -> CString {
    CString::new(name).expect("policy names are fixed or validated")
}

enum EnsureErr {
    NotFound,
    Failed(String),
}

/// Current effective uid, for tests that must own their scratch state.
pub fn current_uid() -> u32 {
    // SAFETY: `geteuid` takes no arguments and always succeeds.
    unsafe { libc::geteuid() }
}

/// Policy directory handle. `state_dir` is the `soda-tailnet` directory
/// itself (production: `/var/lib/soda-tailnet`).
pub struct PolicyStore {
    pub state_dir: PathBuf,
    pub uid: u32,
    pub runtime: bool,
    /// Test seam for the directory-fsync step of publication (mirrors Go's
    /// `syncDir` hook). Production leaves this `None`.
    pub sync_hook: Option<Box<dyn Fn() -> Result<(), String> + Send + Sync>>,
}

impl PolicyStore {
    pub fn new(state_dir: PathBuf, uid: u32, runtime: bool) -> Self {
        PolicyStore {
            state_dir,
            uid,
            runtime,
            sync_hook: None,
        }
    }

    fn create_dir(&self, deadline: Instant) -> Result<(), String> {
        if Instant::now() >= deadline {
            return Err(wire::err_unconfirmed());
        }
        match std::fs::create_dir(&self.state_dir) {
            Ok(()) => {}
            Err(e) if e.kind() == std::io::ErrorKind::AlreadyExists => {}
            Err(_) => return Err(wire::err_unavailable()),
        }
        // Fresh directories inherit the process umask; pin 0700 explicitly.
        let _ = std::fs::set_permissions(&self.state_dir, std::fs::Permissions::from_mode(0o700));
        // Fsync the parent so the new entry survives a crash, like Go.
        if let Some(parent) = self.state_dir.parent() {
            if parent != Path::new("") {
                if let Ok(dir) = File::open(parent) {
                    if dir.sync_all().is_err() {
                        return Err(wire::err_unconfirmed());
                    }
                } else {
                    return Err(wire::err_unconfirmed());
                }
            }
        }
        Ok(())
    }

    fn ensure_dir(&self, deadline: Instant, create: bool) -> Result<(), EnsureErr> {
        match std::fs::symlink_metadata(&self.state_dir) {
            Err(e) if e.kind() == std::io::ErrorKind::NotFound && create => {
                self.create_dir(deadline).map_err(EnsureErr::Failed)?;
            }
            Err(_) | Ok(_) => {}
        }
        match std::fs::symlink_metadata(&self.state_dir) {
            Err(e) if e.kind() == std::io::ErrorKind::NotFound => Err(EnsureErr::NotFound),
            Ok(meta) if dir_owned(&meta, self.uid) => Ok(()),
            _ => Err(EnsureErr::Failed(wire::err_unavailable())),
        }
    }

    /// Acquire the stable exclusive directory lock. `Ok(None)` is a missing
    /// directory without `create` (Go's `os.ErrNotExist`).
    pub fn lock(&self, deadline: Instant, create: bool) -> Result<Option<PolicyLock>, String> {
        if let Err(e) = self.ensure_dir(deadline, create) {
            match e {
                EnsureErr::NotFound => return Ok(None),
                EnsureErr::Failed(cause) => return Err(cause),
            }
        }
        let dir = File::open(&self.state_dir).map_err(|_| wire::err_unavailable())?;
        let meta = dir.metadata().map_err(|_| wire::err_unavailable())?;
        if !dir_owned(&meta, self.uid) {
            return Err(wire::err_unavailable());
        }
        acquire_exclusive(&dir, deadline)?;
        Ok(Some(PolicyLock { dir }))
    }

    /// Open a directory-relative policy file without following a final
    /// symlink. `Ok(None)` is a missing file (Go's `os.ErrNotExist`).
    fn open_owned(&self, lock: &PolicyLock, name: &str) -> Result<Option<File>, String> {
        let path = cstr(name);
        // SAFETY: `openat` on the live lock fd with a NUL-free name.
        let fd = unsafe {
            libc::openat(
                lock.fd(),
                path.as_ptr(),
                libc::O_RDONLY | libc::O_NOFOLLOW | libc::O_NONBLOCK | libc::O_CLOEXEC,
            )
        };
        if fd < 0 {
            return match last_errno() {
                e if e == libc::ENOENT => Ok(None),
                _ => Err(wire::err_unavailable()),
            };
        }
        // SAFETY: `fd` is a fresh owned descriptor.
        let file = unsafe { File::from_raw_fd(fd) };
        let meta = file.metadata().map_err(|_| wire::err_unavailable())?;
        if !file_owned(&meta, self.uid) {
            return Err(wire::err_unavailable());
        }
        Ok(Some(file))
    }

    /// Read a directory-relative policy file, capped at 64 KiB.
    fn read(&self, lock: &PolicyLock, name: &str) -> Result<Option<Vec<u8>>, String> {
        let file = match self.open_owned(lock, name)? {
            Some(f) => f,
            None => return Ok(None),
        };
        use std::io::Read;
        let mut buf = Vec::new();
        file.take(POLICY_FILE_LIMIT + 1)
            .read_to_end(&mut buf)
            .map_err(|_| wire::err_unavailable())?;
        if buf.len() as u64 > POLICY_FILE_LIMIT {
            return Err(wire::err_unavailable());
        }
        Ok(Some(buf))
    }

    fn sync_dir(&self, lock: &PolicyLock) -> Result<(), String> {
        if let Some(hook) = &self.sync_hook {
            return hook();
        }
        lock.dir.sync_all().map_err(|_| wire::err_unconfirmed())
    }

    /// Atomically publish a policy file: refuse unsafe occupants, write a
    /// unique temp file, fsync, rename, fsync the directory.
    fn publish(&self, lock: &PolicyLock, name: &str, body: &[u8]) -> Result<(), String> {
        // Refuse special/relinked occupants before publication, like Go
        // (`open_owned` enforces the ownership rules or errors).
        let _occupant: Option<File> = self.open_owned(lock, name)?;
        if body.len() as u64 > POLICY_FILE_LIMIT {
            return Err(wire::err_invalid());
        }
        let temp = cstr(&format!("pending-{}", new_revision()?));
        // SAFETY: `openat` on the live lock fd.
        let fd = unsafe {
            libc::openat(
                lock.fd(),
                temp.as_ptr(),
                libc::O_WRONLY | libc::O_CREAT | libc::O_EXCL | libc::O_CLOEXEC,
                0o600 as libc::mode_t,
            )
        };
        if fd < 0 {
            return Err(wire::err_unconfirmed());
        }
        // SAFETY: `fd` is a fresh owned descriptor.
        let mut file = unsafe { File::from_raw_fd(fd) };
        use std::io::Write;
        use std::os::unix::io::IntoRawFd;
        let write_ok = file.write_all(body).is_ok() && file.sync_all().is_ok();
        // SAFETY: reclaim the descriptor to observe the close status, like Go.
        let fd = file.into_raw_fd();
        let close_ok = unsafe { libc::close(fd) } == 0;
        if !write_ok || !close_ok {
            return Err(wire::err_unconfirmed());
        }
        let target = cstr(name);
        // SAFETY: `renameat` on the live lock fd.
        let r = unsafe { libc::renameat(lock.fd(), temp.as_ptr(), lock.fd(), target.as_ptr()) };
        if r != 0 {
            return Err(wire::err_unconfirmed());
        }
        self.sync_dir(lock)
    }

    /// Load the enrollment policy (missing file is the fresh zero policy).
    pub fn load(&self, lock: &PolicyLock) -> Result<EnrollmentPolicy, String> {
        let body = match self.read(lock, "policy.json")? {
            Some(b) => b,
            None => {
                return Ok(EnrollmentPolicy {
                    revision: "0".to_string(),
                    ..Default::default()
                })
            }
        };
        let v = decode_enrollment_policy(&body)?;
        if v.version != POLICY_VERSION
            || !wire::is_hex32(&v.revision)
            || !wire::is_hex32(&v.binding)
            || (v.default && !v.admission)
        {
            return Err(wire::err_unavailable());
        }
        let probe = wire::EnrollmentRequest {
            action: "save".to_string(),
            revision: v.revision.clone(),
            tailnet: v.tailnet.clone(),
            tags: Some(v.tags.clone()),
            preauthorized: Some(v.preauthorized),
            client_id: v.credential.client_id.clone(),
            client_secret: v.credential.secret.clone(),
            default: None,
        };
        if probe.validate().is_err() {
            return Err(wire::err_unavailable());
        }
        Ok(v)
    }

    fn view(&self, v: &EnrollmentPolicy) -> wire::EnrollmentView {
        wire::EnrollmentView {
            revision: v.revision.clone(),
            binding: v.binding.clone(),
            tailnet: v.tailnet.clone(),
            tags: v.tags.clone(),
            configured: v.revision != "0",
            admission: v.admission,
            default: v.default,
            preauthorized: v.preauthorized,
            credential_checked: v.revision != "0",
            enrollment_verified: false,
            runtime_supported: self.runtime,
        }
    }

    /// Read the enrollment view without mutating (missing directory or file
    /// is the fresh zero view).
    pub fn enrollment(&self, deadline: Instant) -> Result<wire::EnrollmentView, String> {
        let lock = match self.lock(deadline, false)? {
            Some(l) => l,
            None => {
                return Ok(self.view(&EnrollmentPolicy {
                    revision: "0".to_string(),
                    ..Default::default()
                }))
            }
        };
        let v = self.load(&lock)?;
        Ok(self.view(&v))
    }

    fn check_enrollment(
        &self,
        deadline: Instant,
        r: &wire::EnrollmentRequest,
        check: &dyn Fn(Instant, &wire::EnrollmentRequest) -> Result<(), String>,
    ) -> Result<wire::EnrollmentResult, String> {
        let before = self.enrollment(deadline)?;
        if before.revision != r.revision {
            return Err(wire::err_conflict());
        }
        check(deadline, r)?;
        let after = self.enrollment(deadline)?;
        if after.revision != before.revision {
            return Err(wire::err_conflict());
        }
        Ok(wire::EnrollmentResult {
            outcome: "confirmed".to_string(),
            saved: false,
            credential_checked: true,
            enrollment: after,
        })
    }

    fn apply_save_or_rotate(
        v: &EnrollmentPolicy,
        r: &wire::EnrollmentRequest,
        deadline: Instant,
        check: &dyn Fn(Instant, &wire::EnrollmentRequest) -> Result<(), String>,
    ) -> Result<EnrollmentPolicy, String> {
        if r.action == "rotate"
            && (v.revision == "0"
                || v.tailnet != r.tailnet
                || v.tags != r.tags.clone().unwrap_or_default()
                || v.preauthorized != r.preauthorized.unwrap_or(false))
        {
            return Err(wire::err_conflict());
        }
        check(deadline, r)?;
        if Instant::now() >= deadline {
            return Err(wire::err_unconfirmed());
        }
        let mut out = v.clone();
        if r.action == "save" {
            out.binding = new_revision()?;
            out.default = false;
            out.admission = true;
        }
        // Policy and credential are one restricted atomic publication.
        out.version = POLICY_VERSION;
        out.tailnet = r.tailnet.clone();
        out.tags = r.tags.clone().unwrap_or_default();
        out.preauthorized = r.preauthorized.unwrap_or(false);
        out.credential = Credential {
            client_id: r.client_id.clone(),
            secret: r.client_secret.clone(),
        };
        Ok(out)
    }

    fn apply_default(
        v: &EnrollmentPolicy,
        r: &wire::EnrollmentRequest,
    ) -> Result<EnrollmentPolicy, String> {
        if v.revision == "0" {
            return Err(wire::err_conflict());
        }
        let want = r.default.unwrap_or(false);
        if want && !v.admission {
            return Err(wire::err_conflict());
        }
        let mut out = v.clone();
        out.default = want;
        Ok(out)
    }

    fn apply_disable(v: &EnrollmentPolicy) -> Result<EnrollmentPolicy, String> {
        if v.revision == "0" {
            return Err(wire::err_conflict());
        }
        let mut out = v.clone();
        out.admission = false;
        out.default = false;
        Ok(out)
    }

    /// Apply an enrollment mutation under the directory lock. `check`
    /// verifies the credential against the provider (or a test stub).
    pub fn update(
        &self,
        deadline: Instant,
        r: &wire::EnrollmentRequest,
        check: &dyn Fn(Instant, &wire::EnrollmentRequest) -> Result<(), String>,
    ) -> Result<wire::EnrollmentResult, String> {
        r.validate().map_err(|_| wire::err_invalid())?;
        if r.action == "default" && r.default.unwrap_or(false) && !self.runtime {
            return Err(wire::err_unsupported());
        }
        // A pure credential check must not create directories, files or keys.
        if r.action == "check" {
            return self.check_enrollment(deadline, r, check);
        }
        let lock = match self.lock(deadline, r.action == "save")? {
            Some(l) => l,
            None => return Err(wire::err_conflict()),
        };
        let v = self.load(&lock)?;
        if v.revision != r.revision {
            return Err(wire::err_conflict());
        }
        let mut v = match r.action.as_str() {
            "save" | "rotate" => Self::apply_save_or_rotate(&v, r, deadline, check)?,
            "default" => Self::apply_default(&v, r)?,
            "disable" => Self::apply_disable(&v)?,
            _ => v,
        };
        if Instant::now() >= deadline {
            return Err(wire::err_unconfirmed());
        }
        v.revision = new_revision()?;
        let mut body = encode_enrollment_policy(&v);
        let published = self.publish(&lock, "policy.json", &body);
        // Zero the transient encoded copy; the file keeps the credential, as
        // in Go.
        for b in body.iter_mut() {
            *b = 0;
        }
        published?;
        let checked = v.revision != "0";
        Ok(wire::EnrollmentResult {
            outcome: "confirmed".to_string(),
            saved: true,
            credential_checked: checked,
            enrollment: self.view(&v),
        })
    }

    // ---------- Project policy ----------

    /// Load a project entry (missing file is the zero Off entry). The caller
    /// must have validated `project`; `cid` binds the original container.
    pub fn load_project(
        &self,
        lock: &PolicyLock,
        project: &str,
        cid: &str,
    ) -> Result<ProjectPolicyEntry, String> {
        let zero = ProjectPolicyEntry {
            revision: "0".to_string(),
            project: project.to_string(),
            container: cid.to_string(),
            ..Default::default()
        };
        let body = match self.read(lock, &format!("project-{project}.json"))? {
            Some(b) => b,
            None => return Ok(zero),
        };
        let loaded = decode_project_entry(&body)?;
        if loaded.version != PROJECT_VERSION
            || !wire::is_hex32(&loaded.revision)
            || (!loaded.binding.is_empty() && !wire::is_hex32(&loaded.binding))
            || (loaded.enabled && loaded.binding.is_empty())
        {
            return Err(wire::err_unavailable());
        }
        if loaded.project != project || loaded.container != cid {
            return Err(wire::err_conflict());
        }
        Ok(loaded)
    }

    fn validate_project_binding(
        &self,
        lock: &PolicyLock,
        v: &ProjectPolicyEntry,
        r: &tailnet_domain::ProjectRequest,
    ) -> Result<(), String> {
        let policy = self.load(lock)?;
        let mismatch = if r.action == "retry" {
            !v.enabled || v.binding != r.binding
        } else {
            !v.binding.is_empty() && v.binding != r.binding
        };
        if !policy.admission || policy.binding != r.binding || mismatch {
            return Err(wire::err_conflict());
        }
        Ok(())
    }

    fn mutate_project(
        &self,
        lock: &PolicyLock,
        mut v: ProjectPolicyEntry,
        r: &tailnet_domain::ProjectRequest,
        deadline: Instant,
    ) -> Result<ProjectPolicyEntry, String> {
        if v.revision != r.revision {
            return Err(wire::err_conflict());
        }
        if Instant::now() >= deadline {
            return Err(wire::err_unconfirmed());
        }
        if r.action == "enable" || r.action == "retry" {
            self.validate_project_binding(lock, &v, r)?;
            v.binding.clone_from(&r.binding);
        }
        v.enabled = r.action != "disable";
        v.version = PROJECT_VERSION;
        v.revision = new_revision()?;
        let body = encode_project_entry(&v);
        self.publish(lock, &format!("project-{}.json", r.project), &body)?;
        Ok(v)
    }

    fn build_project_view(
        &self,
        lock: Option<&PolicyLock>,
        v: &ProjectPolicyEntry,
        r: &tailnet_domain::ProjectRequest,
    ) -> Result<tailnet_domain::ProjectView, String> {
        let (state, outcome) = if self.runtime {
            let outcome = if r.action == "inspect" {
                "observed"
            } else {
                "runtime-unconfirmed"
            };
            ("unconfirmed", outcome)
        } else if r.action == "disable" {
            ("runtime-unsupported", "disconnect-unconfirmed")
        } else {
            ("runtime-unsupported", "observed")
        };
        let mut result = tailnet_domain::ProjectView {
            saved: r.action != "inspect",
            project: r.project.clone(),
            revision: v.revision.clone(),
            binding: v.binding.clone(),
            enabled: v.enabled,
            state: state.to_string(),
            outcome: outcome.to_string(),
            ..Default::default()
        };
        if self.runtime {
            if let Some(lock) = lock {
                let policy = self.load(lock)?;
                if policy.admission {
                    result.available_binding = policy.binding;
                    result.available_network = policy.tailnet;
                }
            }
        }
        Ok(result)
    }

    /// Apply a project inspect/enable/retry/disable operation.
    pub fn project(
        &self,
        deadline: Instant,
        r: &tailnet_domain::ProjectRequest,
        cid: &str,
    ) -> Result<tailnet_domain::ProjectView, String> {
        if wire::check_project_request(r).is_err() || !tailnet_domain::valid_container_id(cid) {
            return Err(wire::err_invalid());
        }
        if !self.runtime && (r.action == "enable" || r.action == "retry") {
            return Err(wire::err_unsupported());
        }
        let lock = self.lock(deadline, r.action != "inspect")?;
        let mut v = ProjectPolicyEntry {
            revision: "0".to_string(),
            project: r.project.clone(),
            container: cid.to_string(),
            ..Default::default()
        };
        if let Some(lock) = &lock {
            v = self.load_project(lock, &r.project, cid)?;
        }
        if r.action != "inspect" {
            let lock = lock.as_ref().expect("mutation holds the directory lock");
            v = self.mutate_project(lock, v, r, deadline)?;
        }
        self.build_project_view(lock.as_ref(), &v, r)
    }

    /// Native-only public-metadata projection for one run incarnation
    /// (mirror of `Control.RunBinding`). Missing policy is Off, never an
    /// error; existing nodes keep their saved binding when admission closes.
    pub fn run_binding(
        &self,
        deadline: Instant,
        target: &tailnet_domain::RunTarget,
    ) -> Result<tailnet_domain::RunBinding, String> {
        if !tailnet_domain::valid_project_id(&target.project)
            || !tailnet_domain::valid_container_id(&target.container)
            || !tailnet_domain::valid_container_id(&target.run)
        {
            return Err(wire::err_invalid());
        }
        let lock = match self.lock(deadline, false)? {
            Some(l) => l,
            None => return Ok(tailnet_domain::RunBinding::default()),
        };
        let project = self.load_project(&lock, &target.project, &target.container)?;
        if !project.enabled {
            return Ok(tailnet_domain::RunBinding {
                enabled: false,
                ..Default::default()
            });
        }
        let policy = self.load(&lock)?;
        if policy.binding != project.binding {
            return Err(wire::err_conflict());
        }
        Ok(tailnet_domain::RunBinding {
            enabled: true,
            admission: policy.admission,
            tailnet: policy.tailnet,
            tags: policy.tags,
        })
    }
}

#[cfg(test)]
mod entropy_tests {
    use super::*;

    #[test]
    fn revision_generation_propagates_partial_entropy_failure() {
        let result = new_revision_with(|out| {
            out[0] = 1;
            Err(std::io::Error::other("injected entropy failure"))
        });
        assert_eq!(result.unwrap_err(), wire::err_unavailable());
    }
}

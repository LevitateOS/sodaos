// Native Muse subscription enrollment:
// config validation and executable integrity,
// the `login` device-code session and the auth.json custody checks.

use super::resolution::{self, BoundBinary, ProviderKind};
use super::sha256;
use super::types::{self, Connection, Enrollment};
use super::{enrollment_tempdir, private_tmpfs, Error};
use serde::Deserialize;
use std::io::Read;
use std::os::unix::ffi::OsStrExt;
use std::os::unix::fs::PermissionsExt;
use std::path::Path;
use std::process::{Child, Command, Stdio};
use std::sync::atomic::{AtomicBool, Ordering};
use std::sync::{mpsc, Arc, Mutex};
use std::time::Duration;

const DEVICE_PREFIX: &[u8] = b"https://auth.meta.com/oauth/device/?code=";

#[derive(Debug, Clone, Deserialize)]
pub struct Config {
    pub binary: String,
    #[serde(default)]
    pub version: String,
    pub sha256: String,
    pub root: String,
    /// Resolve the executable from the shared provider store at each
    /// session creation instead of spawning the pinned `binary`. When set,
    /// `binary`/`sha256` may be empty; the bound bundle digest and version
    /// come from the current store selection.
    #[serde(default)]
    pub store: bool,
}

pub struct Provider {
    config: Config,
}

impl Provider {
    /// Bind the fresh shared selection for NEW lease work. Explicit mode
    /// carries no store bundle (`None`); store mode resolves and pins the
    /// current selection so cleanup retains it for the lease lifetime.
    pub fn bind_new(&self) -> Result<Option<BoundBinary>, Error> {
        if !self.config.store {
            return Ok(None);
        }
        Ok(Some(resolution::resolve_new(ProviderKind::Muse)?))
    }

    pub fn new(config: Config) -> Result<Provider, Error> {
        validate_config(&config)?;
        check_binary(&config)?;
        Ok(Provider { config })
    }

    /// Start a device-code enrollment. The owner is accepted for interface
    /// parity; like the Go provider, enrollment inherits no caller-selected
    /// identity.
    ///
    /// Store mode resolves the fresh shared selection on every call, so a
    /// refreshed bundle is picked up without a broker restart and no hash
    /// is cached across sessions.
    pub fn start(&self, _owner: i64) -> Result<Session, Error> {
        let bound = if self.config.store {
            Some(resolution::resolve_new(ProviderKind::Muse)?)
        } else {
            None
        };
        let binary = bound
            .as_ref()
            .map(|held| held.binary.to_string_lossy().into_owned())
            .unwrap_or_else(|| self.config.binary.clone());
        let dir = enrollment_tempdir(Path::new(&self.config.root))?;
        let root = dir.path().to_path_buf();
        let id = root
            .file_name()
            .map(|n| n.as_bytes().to_vec())
            .unwrap_or_default();
        let mut child = Command::new(&binary)
            .arg("login")
            .env_clear()
            .envs(
                environment(root.to_string_lossy().as_ref())
                    .iter()
                    .map(|entry| split_env(entry)),
            )
            .current_dir(&root)
            .stdout(Stdio::piped())
            // Never relay native diagnostics: they may carry credentials.
            .stderr(Stdio::null())
            .spawn()
            .map_err(|_| Error::failed("muse enrollment could not start"))?;
        let stdout = match child.stdout.take() {
            Some(stdout) => stdout,
            None => {
                let _ = child.kill();
                let _ = child.wait();
                return Err(Error::failed("muse enrollment could not start"));
            }
        };
        let (done_tx, done_rx) = mpsc::channel();
        let session = Session {
            inner: Arc::new(Inner {
                state: Mutex::new(Enrollment {
                    provider_id: "muse".to_string(),
                    id: String::from_utf8_lossy(&id).into_owned(),
                    verification_url: String::new(),
                    user_code: String::new(),
                    state: "pending".to_string(),
                    error: String::new(),
                    connection: None,
                }),
                child: Mutex::new(Some(child)),
                root: Mutex::new(Some(dir)),
                done: Mutex::new(done_rx),
                finished: AtomicBool::new(false),
                bound: Mutex::new(bound),
                fallback_version: self.config.version.clone(),
                fallback_digest: self.config.sha256.clone(),
            }),
        };
        {
            let inner = Arc::clone(&session.inner);
            std::thread::spawn(move || read_loop(inner, stdout, done_tx));
        }
        {
            let inner = Arc::clone(&session.inner);
            std::thread::spawn(move || {
                std::thread::sleep(Duration::from_secs(10 * 60));
                if !inner.finished.load(Ordering::SeqCst) {
                    let _ = Session { inner }.close();
                }
            });
        }
        Ok(session)
    }
}

fn split_env(entry: &str) -> (&str, &str) {
    entry.split_once('=').unwrap_or((entry, ""))
}

struct Inner {
    state: Mutex<Enrollment>,
    child: Mutex<Option<Child>>,
    root: Mutex<Option<tempfile::TempDir>>,
    done: Mutex<mpsc::Receiver<()>>,
    finished: AtomicBool,
    /// Store-bound executable for this session. Present only in store mode;
    /// the token pins the exact bundle against cleanup until released.
    bound: Mutex<Option<BoundBinary>>,
    /// Explicit-mode attribution when no store binding exists.
    fallback_version: String,
    fallback_digest: String,
}

impl Drop for Inner {
    fn drop(&mut self) {
        // Session and pump threads retain this owner until the child is reaped.
        // TempDir's fallback cleanup covers abandoned completed sessions.
        self.root.get_mut().unwrap().take();
        // Fail-safe: a dropped-but-unclosed session must not leave its
        // bundle pinned, but release failures are best-effort here.
        if let Ok(mut bound) = self.bound.try_lock() {
            if let Some(mut held) = bound.take() {
                resolution::release_bound(&mut held);
            }
        }
    }
}

#[derive(Clone)]
pub struct Session {
    inner: Arc<Inner>,
}

impl Session {
    pub fn snapshot(&self) -> Enrollment {
        self.inner.state.lock().unwrap().clone()
    }

    /// Version of the exact executable this session runs: the bound store
    /// selection in store mode, the configured pin otherwise. This replaces
    /// the old hardcoded `VERSION` assumption with per-session resolution.
    pub fn provider_version(&self) -> String {
        if let Some(bound) = self.inner.bound.lock().unwrap().as_ref() {
            return bound.version.clone();
        }
        self.inner.fallback_version.clone()
    }

    /// Digest of the exact executable this session runs: the admitted bundle
    /// digest in store mode (usable with `resolve_existing` for recovery),
    /// the configured file digest otherwise.
    pub fn provider_digest(&self) -> String {
        if let Some(bound) = self.inner.bound.lock().unwrap().as_ref() {
            return bound.digest.clone();
        }
        self.inner.fallback_digest.clone()
    }

    pub fn finish(&self) -> Result<(Connection, Vec<u8>), Error> {
        if self.snapshot().state != "completed" {
            return Err(Error::failed("muse enrollment is incomplete"));
        }
        let root = self.inner.root.lock().unwrap();
        let path = root
            .as_ref()
            .expect("session root retained")
            .path()
            .join("config/muse/auth.json");
        let data = credential_file(&path)?;
        Ok((
            Connection {
                provider_id: "muse".to_string(),
                ..Connection::default()
            },
            data,
        ))
    }

    pub fn close(&self) -> Result<(), Error> {
        {
            let mut state = self.inner.state.lock().unwrap();
            if state.state == "pending" {
                state.state = "canceled".to_string();
            }
        }
        if let Some(child) = self.inner.child.lock().unwrap().as_mut() {
            let _ = child.kill();
        }
        let done = self.inner.done.lock().unwrap();
        match done.recv_timeout(Duration::from_secs(5)) {
            Ok(()) | Err(mpsc::RecvTimeoutError::Disconnected) => {
                if let Some(root) = self.inner.root.lock().unwrap().take() {
                    root.close()?;
                }
                // The session no longer needs its exact bundle pinned:
                // release the store reference so cleanup may reclaim it
                // once unreferenced. On timeout below the child may still
                // run, so the pin is deliberately kept there.
                if let Some(mut held) = self.inner.bound.lock().unwrap().take() {
                    resolution::release_bound(&mut held);
                }
                Ok(())
            }
            Err(mpsc::RecvTimeoutError::Timeout) => {
                Err(Error::uncertain("subscription requires reconnection"))
            }
        }
    }
}

fn read_loop(inner: Arc<Inner>, mut out: std::process::ChildStdout, done_tx: mpsc::Sender<()>) {
    // Native prompts need not end with a newline. Keep only a bounded window
    // long enough to recognize the fixed device URL.
    let mut window: Vec<u8> = Vec::with_capacity(128);
    let mut byte = [0u8; 1];
    while out.read_exact(&mut byte).is_ok() {
        window.push(byte[0]);
        if window.len() > 128 {
            window.drain(..window.len() - 128);
        }
        if let Some((url, code)) = scan_device_url(&window) {
            let mut state = inner.state.lock().unwrap();
            state.verification_url = url;
            state.user_code = code;
        }
    }
    let status = inner.child.lock().unwrap().as_mut().map(|c| c.wait());
    complete(&inner, status.and_then(|s| s.ok()).map(|s| s.success()));
    inner.finished.store(true, Ordering::SeqCst);
    drop(done_tx);
}

fn complete(inner: &Inner, success: Option<bool>) {
    let mut state = inner.state.lock().unwrap();
    if state.state != "pending" {
        return;
    }
    if success == Some(true) {
        state.state = "completed".to_string();
        return;
    }
    state.state = "failed".to_string();
    state.error = "Provider enrollment failed".to_string();
}

/// Leftmost `https://auth.meta.com/oauth/device/?code=XXXX-XXXX` match over
/// the current window, mirroring the Go device URL pattern.
fn scan_device_url(window: &[u8]) -> Option<(String, String)> {
    if window.len() < DEVICE_PREFIX.len() + 9 {
        return None;
    }
    for start in 0..=window.len() - DEVICE_PREFIX.len() - 9 {
        if &window[start..start + DEVICE_PREFIX.len()] != DEVICE_PREFIX {
            continue;
        }
        let code = &window[start + DEVICE_PREFIX.len()..start + DEVICE_PREFIX.len() + 9];
        if code[4] == b'-'
            && code[..4]
                .iter()
                .all(|b| b.is_ascii_uppercase() || b.is_ascii_digit())
            && code[5..]
                .iter()
                .all(|b| b.is_ascii_uppercase() || b.is_ascii_digit())
        {
            let url = &window[start..start + DEVICE_PREFIX.len() + 9];
            return Some((
                String::from_utf8_lossy(url).into_owned(),
                String::from_utf8_lossy(code).into_owned(),
            ));
        }
    }
    None
}

// Enrollment inherits no user-selected key, provider endpoint or account state.
fn environment(root: &str) -> Vec<String> {
    vec![
        "PATH=/usr/local/bin:/usr/bin:/bin".to_string(),
        format!("HOME={root}"),
        format!("XDG_CONFIG_HOME={root}/config"),
        format!("XDG_DATA_HOME={root}/data"),
        format!("XDG_STATE_HOME={root}/state"),
        format!("XDG_CACHE_HOME={root}/cache"),
        format!("TMPDIR={root}"),
        "TBH_CREDENTIAL_BACKEND=file".to_string(),
        "NO_COLOR=1".to_string(),
    ]
}

fn validate_config(c: &Config) -> Result<(), Error> {
    if !Path::new(&c.root).is_absolute() {
        return Err(Error::denied("invalid Muse configuration"));
    }
    if c.store {
        // Store mode resolves the executable per session; a pinned binary
        // must not shadow the shared selection.
        if !c.binary.is_empty() || !c.sha256.is_empty() {
            return Err(Error::denied("invalid Muse configuration"));
        }
    } else if !Path::new(&c.binary).is_absolute() || c.sha256.len() != 64 {
        return Err(Error::denied("invalid Muse configuration"));
    }
    let info = std::fs::symlink_metadata(&c.root)
        .map_err(|_| Error::denied("muse enrollment root must be private"))?;
    if !info.file_type().is_dir() || info.permissions().mode() & 0o077 != 0 {
        return Err(Error::denied("muse enrollment root must be private"));
    }
    private_tmpfs(Path::new(&c.root))
}

fn check_binary(c: &Config) -> Result<(), Error> {
    if c.store {
        // The executable is resolved and admitted per session creation.
        return Ok(());
    }
    let mut f = std::fs::File::open(&c.binary)?;
    let sum = sha256::digest_reader(&mut f)?;
    if sha256::hex(&sum) != c.sha256 {
        return Err(Error::denied("muse digest mismatch"));
    }
    Ok(())
}

/// Accepts native device OAuth, including its CLI subscription key.
/// A manually inserted PAYG key is not a subscription enrollment.
pub fn credential_valid(data: &[u8]) -> bool {
    if !types::credential_valid(data) {
        return false;
    }
    #[derive(Deserialize)]
    struct Wire {
        schema_version: u32,
        providers: Providers,
    }
    #[derive(Deserialize)]
    struct Providers {
        meta: Meta,
    }
    #[derive(Deserialize)]
    struct Meta {
        #[serde(default)]
        access_token: String,
        #[serde(default)]
        api_key: String,
        #[serde(default)]
        api_base_url: String,
        #[serde(default)]
        mechanism: String,
        #[serde(default)]
        obtained_via: String,
    }
    // serde_json numbers reject `1.0` for u32 exactly like encoding/json
    // rejects it for int, and unknown fields are ignored on both sides.
    let Ok(wire) = serde_json::from_slice::<Wire>(data) else {
        return false;
    };
    let m = wire.providers.meta;
    wire.schema_version == 1
        && m.mechanism == "oauth"
        && m.obtained_via == "device_code"
        && m.api_base_url == "https://api.meta.ai/v1"
        && !m.access_token.is_empty()
        && !m.api_key.is_empty()
}

fn credential_file(path: &Path) -> Result<Vec<u8>, Error> {
    let info = std::fs::symlink_metadata(path)
        .map_err(|_| Error::denied("invalid Muse credential file"))?;
    if !info.file_type().is_file()
        || info.permissions().mode() & 0o777 != 0o600
        || info.len() > (256 << 10)
    {
        return Err(Error::denied("invalid Muse credential file"));
    }
    let mut f =
        std::fs::File::open(path).map_err(|_| Error::denied("invalid Muse credential file"))?;
    let mut data = Vec::new();
    f.by_ref()
        .take((256 << 10) + 1)
        .read_to_end(&mut data)
        .map_err(|_| Error::denied("invalid Muse subscription credential"))?;
    if !credential_valid(&data) {
        return Err(Error::denied("invalid Muse subscription credential"));
    }
    Ok(data)
}

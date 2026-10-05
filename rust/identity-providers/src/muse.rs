// Pinned native Muse subscription enrollment, mirroring
// internal/identity/muse: config validation, digest and version checks,
// the `login` device-code session and the auth.json custody checks.
use crate::sha256;
use crate::types::{self, Connection, Enrollment};
use crate::{enrollment_tempdir, private_tmpfs, run_capture, Error};
use serde::Deserialize;
use std::io::Read;
use std::os::unix::ffi::OsStrExt;
use std::os::unix::fs::PermissionsExt;
use std::path::{Path, PathBuf};
use std::process::{Child, Command, Stdio};
use std::sync::atomic::{AtomicBool, Ordering};
use std::sync::{mpsc, Arc, Mutex};
use std::time::Duration;

pub const VERSION: &str = "1.4.0-R4161.1";
const VERSION_LINE: &str = "Muse Code 1.4.0 (1.4.0-R4161.1)";
const DEVICE_PREFIX: &[u8] = b"https://auth.meta.com/oauth/device/?code=";

#[derive(Debug, Clone, Deserialize)]
pub struct Config {
    pub binary: String,
    pub version: String,
    pub sha256: String,
    pub root: String,
}

pub struct Provider {
    config: Config,
}

impl Provider {
    pub fn new(config: Config) -> Result<Provider, Error> {
        validate_config(&config)?;
        check_binary(&config)?;
        check_version(&config)?;
        Ok(Provider { config })
    }

    /// Start a device-code enrollment. The owner is accepted for interface
    /// parity; like the Go provider, enrollment inherits no caller-selected
    /// identity.
    pub fn start(&self, _owner: i64) -> Result<Session, Error> {
        let dir = enrollment_tempdir(Path::new(&self.config.root))?;
        let id = dir
            .file_name()
            .map(|n| n.as_bytes().to_vec())
            .unwrap_or_default();
        let mut child = Command::new(&self.config.binary)
            .arg("login")
            .env_clear()
            .envs(
                environment(dir.to_string_lossy().as_ref())
                    .iter()
                    .map(|entry| split_env(entry)),
            )
            .current_dir(&dir)
            .stdout(Stdio::piped())
            // Never relay native diagnostics: they may carry credentials.
            .stderr(Stdio::null())
            .spawn()
            .map_err(|_| {
                let _ = std::fs::remove_dir_all(&dir);
                Error::failed("muse enrollment could not start")
            })?;
        let stdout = child.stdout.take().ok_or_else(|| {
            let _ = std::fs::remove_dir_all(&dir);
            Error::failed("muse enrollment could not start")
        })?;
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
                root: dir,
                done: Mutex::new(done_rx),
                finished: AtomicBool::new(false),
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
    root: PathBuf,
    done: Mutex<mpsc::Receiver<()>>,
    finished: AtomicBool,
}

#[derive(Clone)]
pub struct Session {
    inner: Arc<Inner>,
}

impl Session {
    pub fn snapshot(&self) -> Enrollment {
        self.inner.state.lock().unwrap().clone()
    }

    pub fn finish(&self) -> Result<(Connection, Vec<u8>), Error> {
        if self.snapshot().state != "completed" {
            return Err(Error::failed("muse enrollment is incomplete"));
        }
        let data = credential_file(&self.inner.root.join("config/muse/auth.json"))?;
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
                std::fs::remove_dir_all(&self.inner.root)?;
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
    if !Path::new(&c.binary).is_absolute()
        || !Path::new(&c.root).is_absolute()
        || c.version != VERSION
        || c.sha256.len() != 64
    {
        return Err(Error::denied("invalid pinned Muse configuration"));
    }
    let info = std::fs::symlink_metadata(&c.root)
        .map_err(|_| Error::denied("muse enrollment root must be private"))?;
    if !info.file_type().is_dir() || info.permissions().mode() & 0o077 != 0 {
        return Err(Error::denied("muse enrollment root must be private"));
    }
    private_tmpfs(Path::new(&c.root))
}

fn check_binary(c: &Config) -> Result<(), Error> {
    let mut f = std::fs::File::open(&c.binary)?;
    let sum = sha256::digest_reader(&mut f)?;
    if sha256::hex(&sum) != c.sha256 {
        return Err(Error::denied("muse digest mismatch"));
    }
    Ok(())
}

fn check_version(c: &Config) -> Result<(), Error> {
    let out = run_capture(
        &c.binary,
        &["--version"],
        &environment(&c.root),
        None,
        Duration::from_secs(10),
    )
    .map_err(|_| Error::denied("muse version mismatch"))?;
    if out.trim() != VERSION_LINE {
        return Err(Error::denied("muse version mismatch"));
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

#[cfg(test)]
mod tests {
    use super::*;
    use crate::TestDir;
    use std::os::unix::fs::PermissionsExt;

    const SUBSCRIPTION: &str = r#"{"schema_version":1,"providers":{"meta":{"access_token":"synthetic-oauth","api_key":"synthetic-subscription","api_base_url":"https://api.meta.ai/v1","mechanism":"oauth","obtained_via":"device_code","account_name":"synthetic presentation"}}}"#;

    #[test]
    fn native_subscription_and_private_file() {
        assert!(credential_valid(SUBSCRIPTION.as_bytes()));
        for value in [
            r#"{"schema_version":1,"providers":{"meta":{"api_key":"synthetic-payg","mechanism":"api_key"}}}"#,
            r#"{"schema_version":1,"providers":{"meta":{"access_token":"synthetic","api_key":"synthetic","api_base_url":"https://other.invalid","mechanism":"oauth","obtained_via":"device_code"}}}"#,
        ] {
            assert!(!credential_valid(value.as_bytes()), "admitted {value}");
        }
        assert!(!credential_valid(b"not json"));
        assert!(!credential_valid(b""));

        let dir = TestDir::new();
        let path = dir.path().join("auth.json");
        std::fs::write(&path, SUBSCRIPTION).unwrap();
        std::fs::set_permissions(&path, std::fs::Permissions::from_mode(0o600)).unwrap();
        credential_file(&path).unwrap();
        std::fs::set_permissions(&path, std::fs::Permissions::from_mode(0o644)).unwrap();
        assert!(
            credential_file(&path).is_err(),
            "public credential accepted"
        );
        let link_dir = TestDir::new();
        let link = link_dir.path().join("auth.json");
        std::os::unix::fs::symlink(&path, &link).unwrap();
        assert!(credential_file(&link).is_err(), "symlink accepted");
    }

    #[test]
    fn enrollment_environment_and_device_prompt() {
        let env = environment("/private-enrollment");
        assert_eq!(
            env,
            vec![
                "PATH=/usr/local/bin:/usr/bin:/bin",
                "HOME=/private-enrollment",
                "XDG_CONFIG_HOME=/private-enrollment/config",
                "XDG_DATA_HOME=/private-enrollment/data",
                "XDG_STATE_HOME=/private-enrollment/state",
                "XDG_CACHE_HOME=/private-enrollment/cache",
                "TMPDIR=/private-enrollment",
                "TBH_CREDENTIAL_BACKEND=file",
                "NO_COLOR=1",
            ]
        );
        let found =
            scan_device_url(b"Open https://auth.meta.com/oauth/device/?code=ABCD-1234 to enroll");
        assert_eq!(
            found,
            Some((
                "https://auth.meta.com/oauth/device/?code=ABCD-1234".to_string(),
                "ABCD-1234".to_string()
            ))
        );
        // Lowercase codes, short codes and lookalike hosts are not prompts.
        assert_eq!(
            scan_device_url(b"https://auth.meta.com/oauth/device/?code=abcd-1234"),
            None
        );
        assert_eq!(
            scan_device_url(b"https://auth.meta.com/oauth/device/?code=ABC-1234"),
            None
        );
        assert_eq!(
            scan_device_url(b"https://auth.example.com/oauth/device/?code=ABCD-1234"),
            None
        );
    }

    #[test]
    fn config_validation_matches_go() {
        let dir = TestDir::new();
        std::fs::set_permissions(dir.path(), std::fs::Permissions::from_mode(0o700)).unwrap();
        let good = Config {
            binary: "/usr/local/bin/muse".to_string(),
            version: VERSION.to_string(),
            sha256: "a".repeat(64),
            root: dir.path().to_string_lossy().into_owned(),
        };
        // /tmp is not tmpfs on test hosts, so a fully valid config still
        // fails at the tmpfs gate; every other field must fail earlier
        // with the configuration error.
        for mut bad in [good.clone(), good.clone(), good.clone()] {
            bad.binary = "relative/muse".to_string();
            assert_eq!(
                validate_config(&bad).unwrap_err().to_string(),
                "invalid pinned Muse configuration"
            );
            bad = good.clone();
            bad.version = "other".to_string();
            assert_eq!(
                validate_config(&bad).unwrap_err().to_string(),
                "invalid pinned Muse configuration"
            );
            bad = good.clone();
            bad.sha256 = "short".to_string();
            assert_eq!(
                validate_config(&bad).unwrap_err().to_string(),
                "invalid pinned Muse configuration"
            );
        }
        let mut bad = good.clone();
        bad.root = "relative/root".to_string();
        assert_eq!(
            validate_config(&bad).unwrap_err().to_string(),
            "invalid pinned Muse configuration"
        );
        std::fs::set_permissions(dir.path(), std::fs::Permissions::from_mode(0o755)).unwrap();
        assert_eq!(
            validate_config(&good).unwrap_err().to_string(),
            "muse enrollment root must be private"
        );
    }

    #[test]
    fn binary_digest_checked_before_version() {
        let dir = TestDir::new();
        let binary = dir.path().join("muse-fixture");
        std::fs::write(&binary, b"fixture-bytes").unwrap();
        let sum = sha256::hex(&sha256::digest(b"fixture-bytes"));
        let config = Config {
            binary: binary.to_string_lossy().into_owned(),
            version: VERSION.to_string(),
            sha256: sum,
            root: dir.path().to_string_lossy().into_owned(),
        };
        check_binary(&config).unwrap();
        let bad = Config {
            sha256: "b".repeat(64),
            ..config
        };
        assert_eq!(
            check_binary(&bad).unwrap_err().to_string(),
            "muse digest mismatch"
        );
    }

    #[test]
    fn version_line_must_match_exactly() {
        let dir = TestDir::new();
        let binary = dir.path().join("muse-fixture");
        std::fs::write(&binary, format!("#!/bin/sh\necho '{VERSION_LINE}'\n")).unwrap();
        std::fs::set_permissions(&binary, std::fs::Permissions::from_mode(0o700)).unwrap();
        let config = Config {
            binary: binary.to_string_lossy().into_owned(),
            version: VERSION.to_string(),
            sha256: "c".repeat(64),
            root: dir.path().to_string_lossy().into_owned(),
        };
        check_version(&config).unwrap();
        std::fs::write(&binary, "#!/bin/sh\necho 'Muse Code 9.9.9 (other)'\n").unwrap();
        assert_eq!(
            check_version(&config).unwrap_err().to_string(),
            "muse version mismatch"
        );
    }

    #[test]
    fn native_enrollment_keeps_presentation_private() {
        let dir = TestDir::new();
        let binary = dir.path().join("muse-fixture");
        let script = format!(
            "#!/bin/sh\nset -eu\n[ \"$1\" = login ]\n[ \"$TBH_CREDENTIAL_BACKEND\" = file ]\n[ -z \"${{META_API_KEY:-}}\" ]\nprintf 'https://auth.meta.com/oauth/device/?code=ABCD-1234'\nmkdir -p \"$XDG_CONFIG_HOME/muse\"\numask 077\ncat > \"$XDG_CONFIG_HOME/muse/auth.json\" <<'JSON'\n{SUBSCRIPTION}\nJSON\nprintf 'synthetic secret diagnostic' >&2\n"
        );
        std::fs::write(&binary, script).unwrap();
        std::fs::set_permissions(&binary, std::fs::Permissions::from_mode(0o700)).unwrap();
        let provider = Provider {
            config: Config {
                binary: binary.to_string_lossy().into_owned(),
                version: VERSION.to_string(),
                sha256: String::new(),
                root: dir.path().to_string_lossy().into_owned(),
            },
        };
        let session = provider.start(1).unwrap();
        let deadline = std::time::Instant::now() + Duration::from_secs(5);
        loop {
            if session.inner.finished.load(Ordering::SeqCst) {
                break;
            }
            assert!(
                std::time::Instant::now() < deadline,
                "enrollment fixture did not stop"
            );
            std::thread::sleep(Duration::from_millis(5));
        }
        let snapshot = session.snapshot();
        assert_eq!(snapshot.state, "completed");
        assert_eq!(snapshot.user_code, "ABCD-1234");
        let (conn, data) = session.finish().unwrap();
        assert!(credential_valid(&data));
        let presentation = serde_json::to_string(&conn).unwrap();
        assert!(
            !presentation.contains("synthetic"),
            "credentials escaped: {presentation}"
        );
        let public = serde_json::to_string(&snapshot).unwrap();
        assert!(
            !public.contains("synthetic"),
            "diagnostics escaped: {public}"
        );
        let root = session.inner.root.clone();
        session.close().unwrap();
        assert!(!root.exists(), "credential root not removed");
    }
}

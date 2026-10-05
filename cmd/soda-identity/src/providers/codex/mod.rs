// Codex provider sessions and enrollment lifetime, folded from identity-providers (A06.M).
use super::types::{self, Connection, Enrollment};
use super::{enrollment_tempdir, private_tmpfs, Error};
use config::{check_binary, check_version, environment, validate_config};
use protocol::{read_loop, Message};

mod config;
mod protocol;

pub use config::Config;
use serde::Deserialize;
use std::collections::HashMap;
use std::io::Read;
use std::os::unix::fs::PermissionsExt;
use std::os::unix::io::FromRawFd;
use std::path::{Path, PathBuf};
use std::process::{Child, ChildStdin, Command, Stdio};
use std::sync::atomic::{AtomicBool, Ordering};
use std::sync::{mpsc, Arc, Mutex};
use std::time::Duration;
pub const VERSION: &str = "0.153.4";

pub struct Provider {
    config: Config,
}

impl Provider {
    pub fn new(config: Config) -> Result<Provider, Error> {
        validate_config(&config)?;
        let info =
            std::fs::symlink_metadata(&config.root).map_err(|e| Error::failed(e.to_string()))?;
        if !info.file_type().is_dir() || info.permissions().mode() & 0o077 != 0 {
            return Err(Error::denied("enrollment root must be private"));
        }
        private_tmpfs(Path::new(&config.root))?;
        check_binary(&config)?;
        let provider = Provider { config };
        check_version(&provider)?;
        Ok(provider)
    }

    /// Start a device-code enrollment. Cancellation of the surrounding
    /// request is honored through [`Session::cancel`]; expiry still
    /// reaps abandoned sessions after ten minutes.
    pub fn start(&self, _owner: i64) -> Result<Session, Error> {
        let session = self.create_session()?;
        if let Err(err) = session.initialize() {
            let _ = session.close();
            return Err(err);
        }
        if let Err(err) = session.start_device() {
            let _ = session.close();
            return Err(err);
        }
        let inner = Arc::clone(&session.inner);
        std::thread::spawn(move || {
            std::thread::sleep(Duration::from_secs(10 * 60));
            if !inner.finished.load(Ordering::SeqCst) {
                let _ = Session { inner }.close();
            }
        });
        Ok(session)
    }

    fn create_session(&self) -> Result<Session, Error> {
        let dir = enrollment_tempdir(Path::new(&self.config.root))?;
        let state_dir = dir.join("state");
        let log_dir = dir.join("logs");
        // Enrollment diagnostics and credentials remain confined to the tmpfs root.
        let stderr_path = dir.join("stderr");
        let log = std::fs::File::create(&stderr_path).map_err(|_| {
            let _ = std::fs::remove_dir_all(&dir);
            Error::failed("codex enrollment could not start")
        })?;
        let _ = std::fs::set_permissions(&stderr_path, std::fs::Permissions::from_mode(0o600));
        use std::os::unix::io::AsRawFd;
        let dup = unsafe { libc::dup(log.as_raw_fd()) };
        if dup < 0 {
            let _ = std::fs::remove_dir_all(&dir);
            return Err(Error::failed("codex enrollment could not start"));
        }
        let stderr = unsafe { Stdio::from_raw_fd(dup) };
        let mut child = Command::new(&self.config.binary)
            .arg("-c")
            .arg("cli_auth_credentials_store=\"file\"")
            .arg("-c")
            .arg(format!("sqlite_home=\"{}\"", state_dir.to_string_lossy()))
            .arg("-c")
            .arg(format!("log_dir=\"{}\"", log_dir.to_string_lossy()))
            .arg("app-server")
            .env_clear()
            .envs(
                environment(dir.to_string_lossy().as_ref())
                    .iter()
                    .map(|entry| split_env(entry)),
            )
            .current_dir(&dir)
            .stdin(Stdio::piped())
            .stdout(Stdio::piped())
            .stderr(stderr)
            .spawn()
            .map_err(|_| {
                let _ = std::fs::remove_dir_all(&dir);
                Error::failed("codex enrollment could not start")
            })?;
        let stdin = child.stdin.take().ok_or_else(|| {
            let _ = std::fs::remove_dir_all(&dir);
            Error::failed("codex enrollment could not start")
        })?;
        let stdout = child.stdout.take().ok_or_else(|| {
            let _ = std::fs::remove_dir_all(&dir);
            Error::failed("codex enrollment could not start")
        })?;
        let (done_tx, done_rx) = mpsc::channel();
        let session = Session {
            inner: Arc::new(Inner {
                state: Mutex::new(Enrollment {
                    provider_id: "codex".to_string(),
                    id: String::new(),
                    verification_url: String::new(),
                    user_code: String::new(),
                    state: "pending".to_string(),
                    error: String::new(),
                    connection: None,
                }),
                write: Mutex::new(Some(stdin)),
                child: Mutex::new(Some(child)),
                replies: Mutex::new(HashMap::new()),
                next: Mutex::new(0),
                root: dir,
                done: Mutex::new(done_rx),
                finished: AtomicBool::new(false),
                cancelled: AtomicBool::new(false),
                closed: AtomicBool::new(false),
            }),
        };
        let inner = Arc::clone(&session.inner);
        std::thread::spawn(move || {
            read_loop(&inner, stdout);
            if let Some(child) = inner.child.lock().unwrap().as_mut() {
                let _ = child.kill();
                let _ = child.wait();
            }
            drop(log);
            inner.finished.store(true, Ordering::SeqCst);
            drop(done_tx);
        });
        Ok(session)
    }
}

fn split_env(entry: &str) -> (&str, &str) {
    entry.split_once('=').unwrap_or((entry, ""))
}

struct Inner {
    state: Mutex<Enrollment>,
    write: Mutex<Option<ChildStdin>>,
    child: Mutex<Option<Child>>,
    replies: Mutex<HashMap<i64, mpsc::SyncSender<Message>>>,
    next: Mutex<i64>,
    root: PathBuf,
    done: Mutex<mpsc::Receiver<()>>,
    finished: AtomicBool,
    cancelled: AtomicBool,
    closed: AtomicBool,
}

#[derive(Clone)]
pub struct Session {
    inner: Arc<Inner>,
}

impl Session {
    pub fn snapshot(&self) -> Enrollment {
        self.inner.state.lock().unwrap().clone()
    }

    /// Mirror Go context cancellation for in-flight protocol calls.
    pub fn cancel(&self) {
        self.inner.cancelled.store(true, Ordering::SeqCst);
    }

    fn initialize(&self) -> Result<(), Error> {
        let params =
            serde_json::json!({"clientInfo": {"name": "soda_identity_broker", "version": "1"}});
        self.call("initialize", &params, Duration::from_secs(20))?;
        self.send(&serde_json::json!({"method": "initialized", "params": {}}))
    }

    fn start_device(&self) -> Result<(), Error> {
        let raw = self.call(
            "account/login/start",
            &serde_json::json!({"type": "chatgptDeviceCode"}),
            Duration::from_secs(20),
        )?;
        #[derive(Deserialize)]
        struct Wire {
            #[serde(rename = "type")]
            wire_type: String,
            #[serde(rename = "loginId")]
            login_id: String,
            #[serde(rename = "verificationUrl")]
            verification_url: String,
            #[serde(rename = "userCode")]
            user_code: String,
        }
        let wire: Wire = serde_json::from_value(raw)
            .map_err(|_| Error::failed("invalid device enrollment response"))?;
        if wire.wire_type != "chatgptDeviceCode"
            || wire.login_id.is_empty()
            || wire.verification_url != "https://auth.openai.com/codex/device"
            || wire.user_code.is_empty()
        {
            return Err(Error::failed("invalid device enrollment response"));
        }
        let mut state = self.inner.state.lock().unwrap();
        state.id = wire.login_id;
        state.verification_url = wire.verification_url;
        state.user_code = wire.user_code;
        Ok(())
    }

    pub fn finish(&self) -> Result<(Connection, Vec<u8>), Error> {
        if self.snapshot().state != "completed" {
            return Err(Error::failed("enrollment is incomplete"));
        }
        let conn = self.account()?;
        self.stop()?;
        let data = self.credential_file()?;
        Ok((conn, data))
    }

    fn account(&self) -> Result<Connection, Error> {
        let raw = self.call(
            "account/read",
            &serde_json::json!({"refreshToken": false}),
            Duration::from_secs(20),
        )?;
        #[derive(Deserialize)]
        struct Response {
            account: Option<Account>,
        }
        #[derive(Deserialize)]
        struct Account {
            #[serde(rename = "type")]
            account_type: String,
            email: Option<String>,
            #[serde(rename = "planType")]
            plan: String,
        }
        let response: Response = serde_json::from_value(raw)
            .map_err(|_| Error::failed("managed ChatGPT enrollment required"))?;
        let Some(account) = response.account else {
            return Err(Error::failed("managed ChatGPT enrollment required"));
        };
        if account.account_type != "chatgpt" {
            return Err(Error::failed("managed ChatGPT enrollment required"));
        }
        Ok(Connection {
            plan: account.plan,
            email: account.email.unwrap_or_default(),
            ..Connection::default()
        })
    }

    fn credential_file(&self) -> Result<Vec<u8>, Error> {
        let path = self.inner.root.join("auth.json");
        let info = std::fs::symlink_metadata(&path)
            .map_err(|_| Error::denied("invalid credential file"))?;
        if !info.file_type().is_file()
            || info.permissions().mode() & 0o077 != 0
            || info.len() > (256 << 10)
        {
            return Err(Error::denied("invalid credential file"));
        }
        let mut f =
            std::fs::File::open(&path).map_err(|_| Error::denied("invalid credential file"))?;
        let mut data = Vec::new();
        std::io::Read::by_ref(&mut f)
            .take((256 << 10) + 1)
            .read_to_end(&mut data)
            .map_err(|_| Error::denied("invalid credential state"))?;
        if !types::credential_valid(&data) {
            return Err(Error::denied("invalid credential state"));
        }
        Ok(data)
    }

    fn stop(&self) -> Result<(), Error> {
        if !self.inner.closed.swap(true, Ordering::SeqCst) {
            if let Some(child) = self.inner.child.lock().unwrap().as_mut() {
                let _ = child.kill();
            }
        }
        let done = self.inner.done.lock().unwrap();
        match done.recv_timeout(Duration::from_secs(5)) {
            Ok(()) | Err(mpsc::RecvTimeoutError::Disconnected) => Ok(()),
            Err(mpsc::RecvTimeoutError::Timeout) => {
                Err(Error::uncertain("subscription requires reconnection"))
            }
        }
    }

    pub fn close(&self) -> Result<(), Error> {
        let state = self.snapshot();
        if state.state == "pending" && !state.id.is_empty() {
            let _ = self.call(
                "account/login/cancel",
                &serde_json::json!({"loginId": state.id}),
                Duration::from_secs(2),
            );
        }
        self.stop()?;
        std::fs::remove_dir_all(&self.inner.root)?;
        Ok(())
    }
}

#[cfg(test)]
#[path = "tests.rs"]
mod tests;

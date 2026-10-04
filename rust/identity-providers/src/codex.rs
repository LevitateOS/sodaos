// Pinned Codex app-server managed enrollment, mirroring
// internal/identity/codex: config validation, the JSON-RPC enrollment
// protocol and the auth.json custody checks.
use crate::sha256;
use crate::types::{self, Connection, Enrollment};
use crate::{enrollment_tempdir, private_tmpfs, run_capture, Error};
use serde::Deserialize;
use std::collections::HashMap;
use std::io::{BufRead, BufReader, Read, Write};
use std::os::unix::fs::PermissionsExt;
use std::os::unix::io::FromRawFd;
use std::path::{Path, PathBuf};
use std::process::{Child, ChildStdin, ChildStdout, Command, Stdio};
use std::sync::atomic::{AtomicBool, Ordering};
use std::sync::{mpsc, Arc, Mutex};
use std::time::{Duration, Instant};

pub const VERSION: &str = "0.153.4";

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

#[derive(Debug, Clone, Deserialize)]
struct Message {
    #[serde(default)]
    id: i64,
    #[serde(default)]
    method: String,
    #[serde(default)]
    result: serde_json::Value,
    #[serde(default)]
    error: serde_json::Value,
    #[serde(default)]
    params: serde_json::Value,
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

    fn send(&self, msg: &serde_json::Value) -> Result<(), Error> {
        let mut guard = self.inner.write.lock().unwrap();
        let stdin = guard
            .as_mut()
            .ok_or_else(|| Error::failed("codex process stopped"))?;
        serde_json::to_writer(&mut *stdin, msg)?;
        stdin.write_all(b"\n")?;
        stdin.flush()?;
        Ok(())
    }

    fn call(
        &self,
        method: &str,
        params: &serde_json::Value,
        timeout: Duration,
    ) -> Result<serde_json::Value, Error> {
        let id = {
            let mut next = self.inner.next.lock().unwrap();
            *next += 1;
            *next
        };
        let (tx, rx) = mpsc::sync_channel(1);
        self.inner.replies.lock().unwrap().insert(id, tx);
        let request = serde_json::json!({"id": id, "method": method, "params": params});
        if let Err(err) = self.send(&request) {
            self.inner.replies.lock().unwrap().remove(&id);
            return Err(err);
        }
        let deadline = Instant::now() + timeout;
        loop {
            if self.inner.cancelled.load(Ordering::SeqCst) {
                self.inner.replies.lock().unwrap().remove(&id);
                return Err(Error::failed("context canceled"));
            }
            match rx.recv_timeout(Duration::from_millis(50)) {
                Ok(msg) => {
                    self.inner.replies.lock().unwrap().remove(&id);
                    if !msg.error.is_null() {
                        return Err(protocol_error(&msg.error));
                    }
                    return Ok(msg.result);
                }
                Err(mpsc::RecvTimeoutError::Timeout) => {
                    if self.inner.finished.load(Ordering::SeqCst) {
                        self.inner.replies.lock().unwrap().remove(&id);
                        return Err(Error::failed("codex process stopped"));
                    }
                    if Instant::now() >= deadline {
                        self.inner.replies.lock().unwrap().remove(&id);
                        return Err(Error::failed("codex protocol timed out"));
                    }
                }
                Err(mpsc::RecvTimeoutError::Disconnected) => {
                    self.inner.replies.lock().unwrap().remove(&id);
                    return Err(Error::failed("codex process stopped"));
                }
            }
        }
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

fn protocol_error(raw: &serde_json::Value) -> Error {
    #[derive(Deserialize)]
    struct Detail {
        #[serde(default)]
        message: String,
    }
    let detail: Detail = serde_json::from_value(raw.clone()).unwrap_or(Detail {
        message: String::new(),
    });
    let message = detail.message.to_lowercase();
    if message.contains("device")
        && (message.contains("not enabled") || message.contains("disabled"))
    {
        return Error::failed(
            "device-code login is disabled; enable it in ChatGPT security settings or workspace permissions",
        );
    }
    Error::failed("codex protocol request failed")
}

fn read_loop(inner: &Inner, out: ChildStdout) {
    let mut reader = BufReader::new(out);
    let mut line = Vec::with_capacity(4096);
    loop {
        line.clear();
        // Mirror bufio.Scanner with a 512 KiB token cap: overlong or
        // malformed lines end the enrollment read loop.
        let mut total = 0usize;
        let mut complete = false;
        while let Ok(chunk) = reader.fill_buf() {
            if chunk.is_empty() {
                break;
            }
            let mut consumed = 0usize;
            for &b in chunk {
                consumed += 1;
                total += 1;
                if total > (512 << 10) {
                    break;
                }
                if b == b'\n' {
                    complete = true;
                    break;
                }
                line.push(b);
            }
            reader.consume(consumed);
            if complete || total > (512 << 10) {
                break;
            }
        }
        if !complete {
            break;
        }
        if line.last() == Some(&b'\r') {
            line.pop();
        }
        let Ok(msg) = serde_json::from_slice::<Message>(&line) else {
            break;
        };
        if msg.id != 0 {
            if let Some(tx) = inner.replies.lock().unwrap().get(&msg.id) {
                // The protocol never repeats an id; a duplicate is dropped
                // instead of blocking the reader on a full channel.
                let _ = tx.try_send(msg);
            }
        } else {
            notify(inner, &msg);
        }
    }
    let mut state = inner.state.lock().unwrap();
    if state.state == "pending" {
        state.state = "failed".to_string();
        state.error = "Provider enrollment ended".to_string();
    }
}

fn notify(inner: &Inner, msg: &Message) {
    if msg.method != "account/login/completed" {
        return;
    }
    #[derive(Deserialize)]
    struct Event {
        #[serde(rename = "loginId")]
        login_id: String,
        success: bool,
    }
    let Ok(event) = serde_json::from_value::<Event>(msg.params.clone()) else {
        return;
    };
    let mut state = inner.state.lock().unwrap();
    if !state.id.is_empty() && event.login_id != state.id {
        return;
    }
    if event.success {
        state.state = "completed".to_string();
        return;
    }
    state.state = "failed".to_string();
    state.error = "Provider enrollment failed; retry or check device-code access".to_string();
}

fn environment(root: &str) -> Vec<String> {
    filter_env(std::env::vars(), root)
}

fn filter_env(vars: impl Iterator<Item = (String, String)>, root: &str) -> Vec<String> {
    let mut out = Vec::new();
    for (name, value) in vars {
        if name.starts_with("OPENAI")
            || name.starts_with("CODEX")
            || name.starts_with("CHATGPT")
            || name.starts_with("AWS")
            || name == "TMPDIR"
            || name == "XDG_CACHE_HOME"
        {
            continue;
        }
        out.push(format!("{name}={value}"));
    }
    out.push(format!("CODEX_HOME={root}"));
    out.push(format!("TMPDIR={root}"));
    out.push(format!("XDG_CACHE_HOME={root}/cache"));
    out
}

fn validate_config(c: &Config) -> Result<(), Error> {
    if !Path::new(&c.binary).is_absolute()
        || !Path::new(&c.root).is_absolute()
        || c.version != VERSION
        || c.sha256.len() != 64
    {
        return Err(Error::denied("invalid pinned Codex configuration"));
    }
    Ok(())
}

fn check_binary(c: &Config) -> Result<(), Error> {
    let data = std::fs::read(&c.binary)?;
    if sha256::hex(&sha256::digest(&data)) != c.sha256 {
        return Err(Error::denied("codex digest mismatch"));
    }
    Ok(())
}

fn check_version(p: &Provider) -> Result<(), Error> {
    let out = run_capture(
        &p.config.binary,
        &["--version"],
        &environment(&p.config.root),
        None,
        Duration::from_secs(10),
    )
    .map_err(|_| Error::denied("codex version mismatch"))?;
    if out.trim() != format!("codex-cli {}", p.config.version) {
        return Err(Error::denied("codex version mismatch"));
    }
    Ok(())
}

impl From<serde_json::Error> for Error {
    fn from(err: serde_json::Error) -> Error {
        Error::failed(err.to_string())
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::TestDir;

    #[test]
    fn environment_filters_provider_credentials() {
        let vars = vec![
            ("OPENAI_API_KEY".to_string(), "synthetic".to_string()),
            ("CODEX_HOME".to_string(), "synthetic".to_string()),
            ("CHATGPT_AUTH".to_string(), "synthetic".to_string()),
            ("AWS_SECRET".to_string(), "synthetic".to_string()),
            ("TMPDIR".to_string(), "synthetic".to_string()),
            ("XDG_CACHE_HOME".to_string(), "synthetic".to_string()),
            ("PATH".to_string(), "/usr/bin".to_string()),
        ];
        let env = filter_env(vars.into_iter(), "/enrollment-root");
        assert!(
            !env.iter().any(|e| e.contains("synthetic")),
            "leaked: {env:?}"
        );
        assert!(env.contains(&"CODEX_HOME=/enrollment-root".to_string()));
        assert!(env.contains(&"TMPDIR=/enrollment-root".to_string()));
        assert!(env.contains(&"XDG_CACHE_HOME=/enrollment-root/cache".to_string()));
        assert!(env.contains(&"PATH=/usr/bin".to_string()));
    }

    #[test]
    fn config_validation_matches_go() {
        let good = Config {
            binary: "/usr/local/bin/codex".to_string(),
            version: VERSION.to_string(),
            sha256: "a".repeat(64),
            root: "/enrollment-root".to_string(),
        };
        validate_config(&good).unwrap();
        for bad in [
            Config {
                binary: "relative".to_string(),
                ..good.clone()
            },
            Config {
                version: "other".to_string(),
                ..good.clone()
            },
            Config {
                sha256: "short".to_string(),
                ..good.clone()
            },
            Config {
                root: "relative".to_string(),
                ..good.clone()
            },
        ] {
            assert_eq!(
                validate_config(&bad).unwrap_err().to_string(),
                "invalid pinned Codex configuration"
            );
        }
    }

    fn protocol_fixture(completed: bool) -> String {
        // Minimal app-server fixture: initialize, device-code start with an
        // immediate completion event, account read and login cancel. Methods
        // match exactly so the `initialized` notification falls through.
        let completion = if completed {
            "printf '{\"tokens\":{\"refresh_token\":\"synthetic-enrollment\"}}' > \"$CODEX_HOME/auth.json\"; chmod 600 \"$CODEX_HOME/auth.json\"; echo '{\"method\":\"account/login/completed\",\"params\":{\"loginId\":\"synthetic-login\",\"success\":true,\"error\":null}}'".to_string()
        } else {
            ":".to_string()
        };
        format!(
            "#!/bin/sh\nset -eu\nid_of() {{ printf '%s' \"$1\" | sed 's/.*\"id\":\\([0-9][0-9]*\\).*/\\1/'; }}\nwhile IFS= read -r line; do\ncase \"$line\" in\n*\\\"method\\\":\\\"initialize\\\"*) echo \"{{\\\"id\\\":$(id_of \"$line\"),\\\"result\\\":{{}}}}\";;\n*\\\"method\\\":\\\"account/login/start\\\"*) echo \"{{\\\"id\\\":$(id_of \"$line\"),\\\"result\\\":{{\\\"type\\\":\\\"chatgptDeviceCode\\\",\\\"loginId\\\":\\\"synthetic-login\\\",\\\"verificationUrl\\\":\\\"https://auth.openai.com/codex/device\\\",\\\"userCode\\\":\\\"ABCD-1234\\\"}}}}\"; {completion};;\n*\\\"method\\\":\\\"account/read\\\"*) echo \"{{\\\"id\\\":$(id_of \"$line\"),\\\"result\\\":{{\\\"account\\\":{{\\\"type\\\":\\\"chatgpt\\\",\\\"email\\\":null,\\\"planType\\\":\\\"plus\\\"}},\\\"requiresOpenaiAuth\\\":true}}}}\";;\n*\\\"method\\\":\\\"account/login/cancel\\\"*) echo \"{{\\\"id\\\":$(id_of \"$line\"),\\\"result\\\":{{\\\"status\\\":\\\"canceled\\\"}}}}\";;\n*) :;;\nesac\ndone\n"
        )
    }

    fn fixture_provider(dir: &TestDir, completed: bool) -> Provider {
        let binary = dir.path().join("codex-fixture");
        std::fs::write(&binary, protocol_fixture(completed)).unwrap();
        std::fs::set_permissions(&binary, std::fs::Permissions::from_mode(0o700)).unwrap();
        Provider {
            config: Config {
                binary: binary.to_string_lossy().into_owned(),
                version: VERSION.to_string(),
                sha256: String::new(),
                root: dir.path().to_string_lossy().into_owned(),
            },
        }
    }

    #[test]
    fn managed_enrollment_persists_only_after_process_stop() {
        let dir = TestDir::new();
        let provider = fixture_provider(&dir, true);
        let session = provider.start(1).unwrap();
        let deadline = Instant::now() + Duration::from_secs(5);
        while session.snapshot().state == "pending" {
            assert!(Instant::now() < deadline, "fixture enrollment timeout");
            std::thread::sleep(Duration::from_millis(1));
        }
        let (conn, data) = session.finish().unwrap();
        assert_eq!(conn.plan, "plus");
        assert_eq!(conn.email, "");
        assert_eq!(
            data,
            br#"{"tokens":{"refresh_token":"synthetic-enrollment"}}"#
        );
        assert!(
            session.inner.finished.load(Ordering::SeqCst),
            "credential retained before provider process ended"
        );
        let root = session.inner.root.clone();
        session.close().unwrap();
        assert!(!root.exists(), "credential tmpfs root not removed");
    }

    #[test]
    fn cancel_removes_unfinished_enrollment() {
        let dir = TestDir::new();
        let provider = fixture_provider(&dir, false);
        let session = provider.start(1).unwrap();
        assert_eq!(session.snapshot().state, "pending");
        let root = session.inner.root.clone();
        session.close().unwrap();
        assert!(!root.exists(), "canceled credentials retained");
    }

    #[test]
    fn device_disabled_maps_to_actionable_error() {
        let err =
            protocol_error(&serde_json::json!({"message": "Device code login is not enabled"}));
        assert!(err
            .to_string()
            .contains("enable it in ChatGPT security settings"));
        let err = protocol_error(&serde_json::json!({"message": "boom"}));
        assert_eq!(err.to_string(), "codex protocol request failed");
    }
}

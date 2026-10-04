// soda-identity owns the private subscription connection service.
use serde::Deserialize;
use soda_identity::control::{self, Controller};
use soda_identity::runtime::HostClient;
use soda_identity::store::Store;
use std::collections::HashMap;
use std::os::unix::fs::{DirBuilderExt, PermissionsExt};
use std::os::unix::net::UnixListener;
use std::path::Path;
use std::sync::atomic::{AtomicBool, AtomicUsize, Ordering};
use std::sync::Arc;
use std::time::{Duration, Instant};

const SETTINGS_FIELDS: &[&str] = &[
    "database_dsn_file",
    "key_file",
    "admin_socket",
    "runtime_socket",
    "host_socket",
    "codex",
    "muse",
];

const PROVIDER_FIELDS: &[&str] = &["binary", "version", "sha256", "root"];

#[derive(Debug, Clone, Default, Deserialize)]
struct ProviderSettings {
    #[serde(default)]
    binary: String,
    #[serde(default)]
    version: String,
    #[serde(default)]
    sha256: String,
    #[serde(default)]
    root: String,
}

#[derive(Debug, Clone, Deserialize)]
struct Settings {
    #[serde(default)]
    database_dsn_file: String,
    #[serde(default)]
    key_file: String,
    #[serde(default)]
    admin_socket: String,
    #[serde(default)]
    runtime_socket: String,
    #[serde(default)]
    host_socket: String,
    #[serde(default)]
    codex: ProviderSettings,
    #[serde(default)]
    muse: ProviderSettings,
}

static SHUTDOWN: AtomicBool = AtomicBool::new(false);

extern "C" fn handle_signal(_: libc::c_int) {
    SHUTDOWN.store(true, Ordering::SeqCst);
}

fn main() {
    if let Err(err) = run() {
        eprintln!("soda-identity: {err}");
        std::process::exit(1);
    }
}

fn run() -> Result<(), String> {
    let config_path = parse_args()?;
    let settings = load(&config_path)?;
    let (admin, runtime) = service_listeners(&settings)?;
    let broker = open_broker(&settings)?;
    let broker = Arc::new(broker);
    if let Err(err) = broker.reconcile() {
        eprintln!("identity execution termination remains unconfirmed: {err}");
    }
    serve(admin, runtime, broker);
    Ok(())
}

fn parse_args() -> Result<String, String> {
    let mut config = "/etc/soda/identity.json".to_string();
    let mut args = std::env::args().skip(1).peekable();
    while let Some(arg) = args.next() {
        if arg == "--config" || arg == "-config" {
            let Some(value) = args.next() else {
                return Err("flag needs an argument: -config".to_string());
            };
            config = value;
        } else if let Some(value) = arg.strip_prefix("--config=") {
            config = value.to_string();
        } else if let Some(value) = arg.strip_prefix("-config=") {
            config = value.to_string();
        } else if arg == "-h" || arg == "--help" || arg == "-help" {
            return Err("flag: help requested".to_string());
        } else if arg.starts_with('-') {
            return Err(format!("flag provided but not defined: {arg}"));
        } else {
            return Err("unexpected arguments".to_string());
        }
    }
    Ok(config)
}

fn load(path: &str) -> Result<Settings, String> {
    let raw = std::fs::read(path).map_err(|e| format!("read identity settings: {e}"))?;
    let settings: Settings = soda_identity::strict::decode(
        &raw,
        soda_identity::strict::MAX_DOCUMENT,
        SETTINGS_FIELDS,
        &[("codex", PROVIDER_FIELDS), ("muse", PROVIDER_FIELDS)],
    )
    .map_err(|e| format!("decode identity settings: {e}"))?;
    for path in [
        &settings.database_dsn_file,
        &settings.key_file,
        &settings.admin_socket,
        &settings.runtime_socket,
        &settings.host_socket,
    ] {
        if !Path::new(path).is_absolute() {
            return Err("explicit absolute service paths required".to_string());
        }
    }
    if settings.admin_socket == settings.runtime_socket {
        return Err("administration and runtime sockets must differ".to_string());
    }
    Ok(settings)
}

fn service_listeners(settings: &Settings) -> Result<(UnixListener, UnixListener), String> {
    if let Ok(pid) = std::env::var("LISTEN_PID") {
        if pid == std::process::id().to_string() {
            return activated_listeners(settings);
        }
    }
    Ok((
        listen(&settings.admin_socket, 0o660)?,
        listen(&settings.runtime_socket, 0o600)?,
    ))
}

fn listen(path: &str, mode: u32) -> Result<UnixListener, String> {
    // Refuse to replace an occupied path. Activation cleanup only removes
    // sockets it created.
    if std::fs::symlink_metadata(path).is_ok() {
        return Err("broker socket path is occupied".to_string());
    }
    let listener = UnixListener::bind(path).map_err(|e| e.to_string())?;
    std::fs::set_permissions(path, std::fs::Permissions::from_mode(mode))
        .map_err(|e| e.to_string())?;
    Ok(listener)
}

fn activated_listeners(settings: &Settings) -> Result<(UnixListener, UnixListener), String> {
    if std::env::var("LISTEN_FDS").unwrap_or_default() != "2" {
        return Err("two broker sockets required".to_string());
    }
    let names = std::env::var("LISTEN_FDNAMES").unwrap_or_default();
    let parts: Vec<&str> = names.split(':').collect();
    if parts.len() != 2 {
        return Err("named broker sockets required".to_string());
    }
    let mut admin: Option<UnixListener> = None;
    let mut runtime: Option<UnixListener> = None;
    for (fd, name) in [(3, parts[0]), (4, parts[1])] {
        // Systemd passes the bound sockets above stderr; take ownership.
        let listener = unsafe { UnixListener::from_raw_fd(fd) };
        match name {
            "admin" => admin = Some(listener),
            "runtime" => runtime = Some(listener),
            _ => return Err("named broker sockets required".to_string()),
        }
    }
    let (admin, runtime) = match (admin, runtime) {
        (Some(admin), Some(runtime)) => (admin, runtime),
        _ => return Err("named broker sockets required".to_string()),
    };
    for (listener, want) in [
        (&admin, &settings.admin_socket),
        (&runtime, &settings.runtime_socket),
    ] {
        let actual = listener
            .local_addr()
            .map_err(|e| e.to_string())?
            .as_pathname()
            .map(|p| p.to_string_lossy().into_owned())
            .unwrap_or_default();
        if actual != *want {
            return Err("broker socket configuration differs from systemd".to_string());
        }
    }
    Ok((admin, runtime))
}

use std::os::unix::io::FromRawFd;

fn open_store(settings: &Settings) -> Result<Store, String> {
    let key = grant_key(&settings.key_file)?;
    let dsn = secret(&settings.database_dsn_file)?;
    Store::open_encrypted(&dsn, &key).map_err(|e| e.to_string())
}

fn grant_key(path: &str) -> Result<Vec<u8>, String> {
    let info = std::fs::symlink_metadata(path)
        .map_err(|_| "grant key must be a restricted regular file (0600 or 0640)".to_string())?;
    if !info.file_type().is_file() || info.permissions().mode() & 0o027 != 0 {
        return Err("grant key must be a restricted regular file (0600 or 0640)".to_string());
    }
    let encoded = secret(path)?;
    let key = soda_identity::wire::base64_bytes::decode(&encoded)
        .map_err(|_| "grant key must encode exactly 32 bytes".to_string())?;
    if key.len() != 32 {
        return Err("grant key must encode exactly 32 bytes".to_string());
    }
    Ok(key)
}

fn secret(path: &str) -> Result<String, String> {
    let file = std::fs::File::open(path)
        .map_err(|_| "cannot open configured credential file".to_string())?;
    let info = file
        .metadata()
        .map_err(|_| "cannot open configured credential file".to_string())?;
    if !info.file_type().is_file() || info.permissions().mode() & 0o007 != 0 {
        return Err("credential must be a regular file inaccessible to other users".to_string());
    }
    use std::io::Read;
    let mut data = Vec::new();
    file.take((64 << 10) as u64 + 1)
        .read_to_end(&mut data)
        .map_err(|_| "cannot read credential file".to_string())?;
    if data.len() > 64 << 10 {
        return Err("cannot read credential file".to_string());
    }
    let text =
        String::from_utf8(data).map_err(|_| "credential is empty or malformed".to_string())?;
    let trimmed = text.trim().to_string();
    if trimmed.is_empty() || trimmed.contains(['\r', '\n', '\0']) {
        return Err("credential is empty or malformed".to_string());
    }
    Ok(trimmed)
}

fn open_broker(settings: &Settings) -> Result<Controller, String> {
    let mut providers: HashMap<String, Box<dyn control::Provider>> = HashMap::new();
    if !settings.codex.binary.is_empty() {
        mkdir_all_mode(&settings.codex.root, 0o700)?;
        let provider =
            identity_providers::codex::Provider::new(identity_providers::codex::Config {
                binary: settings.codex.binary.clone(),
                version: settings.codex.version.clone(),
                sha256: settings.codex.sha256.clone(),
                root: settings.codex.root.clone(),
            })
            .map_err(|e| e.to_string())?;
        providers.insert(
            "codex".to_string(),
            Box::new(control::CodexProvider(provider)),
        );
    }
    if !settings.muse.binary.is_empty() {
        mkdir_all_mode(&settings.muse.root, 0o700)?;
        let provider = identity_providers::muse::Provider::new(identity_providers::muse::Config {
            binary: settings.muse.binary.clone(),
            version: settings.muse.version.clone(),
            sha256: settings.muse.sha256.clone(),
            root: settings.muse.root.clone(),
        })
        .map_err(|e| e.to_string())?;
        providers.insert(
            "muse".to_string(),
            Box::new(control::MuseProvider(provider)),
        );
    }
    let store = open_store(settings)?;
    let runtime: Box<dyn control::Runtime> = Box::new(HostClient::new(&settings.host_socket));
    Controller::new(store, providers, runtime).map_err(|e| e.to_string())
}

fn mkdir_all_mode(path: &str, mode: u32) -> Result<(), String> {
    let path = Path::new(path);
    let mut stack = Vec::new();
    let mut current = path;
    loop {
        match std::fs::symlink_metadata(current) {
            Ok(info) => {
                if !info.file_type().is_dir() {
                    return Err(format!("not a directory: {}", current.display()));
                }
                break;
            }
            Err(e) if e.kind() == std::io::ErrorKind::NotFound => {
                stack.push(current);
                match current.parent() {
                    Some(parent) if !parent.as_os_str().is_empty() => current = parent,
                    _ => break,
                }
            }
            Err(e) => return Err(e.to_string()),
        }
    }
    for dir in stack.iter().rev() {
        std::fs::DirBuilder::new()
            .mode(mode)
            .create(dir)
            .map_err(|e| {
                if e.kind() == std::io::ErrorKind::AlreadyExists {
                    return "concurrent enrollment root creation".to_string();
                }
                e.to_string()
            })?;
    }
    Ok(())
}

fn serve(admin: UnixListener, runtime: UnixListener, broker: Arc<Controller>) {
    unsafe {
        libc::signal(
            libc::SIGTERM,
            handle_signal as *const () as libc::sighandler_t,
        );
        libc::signal(
            libc::SIGINT,
            handle_signal as *const () as libc::sighandler_t,
        );
    }
    let shutdown = Arc::new(AtomicBool::new(false));
    let inflight = Arc::new(AtomicUsize::new(0));
    let admin_server = soda_identity::http::Server::new(
        Arc::clone(&broker),
        false,
        Arc::clone(&shutdown),
        Arc::clone(&inflight),
    );
    let runtime_server = soda_identity::http::Server::new(
        Arc::clone(&broker),
        true,
        Arc::clone(&shutdown),
        Arc::clone(&inflight),
    );
    std::thread::scope(|scope| {
        scope.spawn(|| admin_server.serve(&admin));
        scope.spawn(|| runtime_server.serve(&runtime));
        // Reconcile every 5s; each sweep is bounded by the store timeouts.
        let mut last = Instant::now() - Duration::from_secs(5);
        loop {
            if SHUTDOWN.load(Ordering::SeqCst) {
                break;
            }
            if last.elapsed() >= Duration::from_secs(5) {
                last = Instant::now();
                if broker.sweep().is_err() {
                    eprintln!("identity execution termination remains unconfirmed");
                }
            }
            std::thread::sleep(Duration::from_millis(100));
        }
        shutdown.store(true, Ordering::SeqCst);
        // Drain in-flight requests up to the 10s shutdown budget.
        let deadline = Instant::now() + Duration::from_secs(10);
        while inflight.load(Ordering::SeqCst) > 0 && Instant::now() < deadline {
            std::thread::sleep(Duration::from_millis(20));
        }
    });
}

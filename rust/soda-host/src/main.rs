// Rust `soda-host` daemon binary (PR26): operator flags, systemd socket
// activation, the Unix-socket HTTP server, the muse launch listener, and
// graceful shutdown. Mirrors cmd/soda-host/main.go exactly: same flags,
// same error strings, same exit codes (0 clean, 78 tailnet preparation
// unconfirmed, 1 any other failure, 2 flag-parse errors).
//
// One documented deviation: Go's `CloseTerminals` force-closes hijacked
// terminal streams so shutdown drains fast; the mux owns stream slots as
// pure counters (no handles), so a shutdown with live terminals waits out
// the 5s drain instead. The outcome is identical (exit after ≤5s).
use std::os::unix::io::AsRawFd;
use std::sync::atomic::{AtomicBool, Ordering};
use std::sync::Arc;
use std::time::{Duration, Instant};

use soda_host::dbackend::{BackendConfig, DaemonBackend, MuseConfig};
use soda_host::gmux_routes::DaemonConfig;
use soda_host::gmux_server::{self, Server};
use soda_host::{iconfig, muse_serve, project, tailnet_domain};

const DEFAULT_CONFIG: &str = "/etc/soda/host.json";
const RELEASE_CONFIG: &str = "/usr/share/soda/release.json";
const FACTORY_STATE_DIR: &str = "/var/lib/soda/host/factory";
const MUSE_JOIN_TIMEOUT: Duration = Duration::from_secs(35);
const SHUTDOWN_DRAIN: Duration = Duration::from_secs(5);

static SHUTDOWN: AtomicBool = AtomicBool::new(false);

extern "C" fn handle_signal(_signo: libc::c_int) {
    SHUTDOWN.store(true, Ordering::SeqCst);
}

fn install_signal_handlers() {
    unsafe {
        let mut action: libc::sigaction = std::mem::zeroed();
        action.sa_sigaction = handle_signal as *const () as usize;
        action.sa_flags = libc::SA_RESTART;
        libc::sigemptyset(&mut action.sa_mask);
        libc::sigaction(libc::SIGTERM, &action, std::ptr::null_mut());
        libc::sigaction(libc::SIGINT, &action, std::ptr::null_mut());
    }
}

/// CLI failure with Go's exit-code contract: preparation failures exit 78
/// (Go's `errors.Is(err, errTailnetPreparation)` through the join),
/// everything else exits 1.
enum MainError {
    TailnetPreparation(String),
    Other(String),
}

impl MainError {
    fn message(&self) -> &str {
        match self {
            MainError::TailnetPreparation(m) | MainError::Other(m) => m,
        }
    }

    fn exit_code(&self) -> i32 {
        match self {
            MainError::TailnetPreparation(_) => 78,
            MainError::Other(_) => 1,
        }
    }
}

struct Flags {
    config: String,
    release: String,
    listen_path: String,
    tailnet_action: String,
    project: String,
    positionals: Vec<String>,
}

fn usage() -> String {
    "Usage: soda-host [--config PATH] [--release PATH] [--listen-path PATH] [--tailnet-action run|stop --project ID]\n\n\
     Installed privileged host daemon: serves the root:soda Unix operation\n\
     socket under systemd socket activation. --release selects the appliance\n\
     release file (default /usr/share/soda/release.json); an empty value\n\
     disables the release overlay for test fixtures. --listen-path binds\n\
     the Unix socket directly instead of taking activation (fixtures);\n\
     it still requires root.\n"
        .to_string()
}

/// Go `flag` subset: `--name value`, `--name=value`, `-name value`,
/// single `-h`. Unknown flags and missing values are parse errors.
fn parse_flags(args: &[String]) -> Result<Flags, String> {
    let mut flags = Flags {
        config: DEFAULT_CONFIG.to_string(),
        release: RELEASE_CONFIG.to_string(),
        listen_path: String::new(),
        tailnet_action: String::new(),
        project: String::new(),
        positionals: Vec::new(),
    };
    let mut i = 0;
    while i < args.len() {
        let arg = &args[i];
        let (name, inline) = if let Some(rest) = arg.strip_prefix("--") {
            match rest.split_once('=') {
                Some((name, value)) => (name, Some(value.to_string())),
                None => (rest, None),
            }
        } else if let Some(rest) = arg.strip_prefix('-') {
            match rest.split_once('=') {
                Some((name, value)) => (name, Some(value.to_string())),
                None => (rest, None),
            }
        } else {
            flags.positionals.push(arg.clone());
            i += 1;
            continue;
        };
        if name == "h" || name == "help" {
            return Err("__help__".to_string());
        }
        let value = match inline {
            Some(value) => value,
            None => {
                i += 1;
                if i >= args.len() {
                    return Err(format!("flag needs an argument: -{name}"));
                }
                args[i].clone()
            }
        };
        match name {
            "config" => flags.config = value,
            "release" => flags.release = value,
            "listen-path" => flags.listen_path = value,
            "tailnet-action" => flags.tailnet_action = value,
            "project" => flags.project = value,
            _ => return Err(format!("flag provided but not defined: -{name}")),
        }
        i += 1;
    }
    Ok(flags)
}

fn is_root() -> bool {
    // SAFETY: `geteuid` is async-signal-safe and infallible.
    unsafe { libc::geteuid() == 0 }
}

// -- companion phase (`runTailnetAction`) --

fn run_tailnet_action(
    config_path: &str,
    release_path: &str,
    action: &str,
    project_id: &str,
    positionals: usize,
) -> Result<(), MainError> {
    if !is_root()
        || !tailnet_domain::valid_project_id(project_id)
        || (action != "run" && action != "stop")
        || positionals != 0
    {
        // Go `tailnet.ErrInvalid`.
        return Err(MainError::Other("invalid Tailnet request".to_string()));
    }
    let config = iconfig::load_config(config_path, release_path)
        .map_err(|_| MainError::Other("tailnet unavailable".to_string()))?;
    if !config.tailnet_management || config.tailnet_image.is_empty() {
        return Ok(());
    }
    let backend = open_backend(&config);
    let phase = Instant::now() + Duration::from_secs(45);
    if action == "stop" {
        let bounded = phase.min(Instant::now() + Duration::from_secs(30));
        return backend
            .companion_stop(project_id, bounded)
            .map_err(MainError::Other);
    }
    // Go cancels the phase timeout once the run starts, then waits on the
    // signal context. The wait below observes a distant horizon; signal
    // handlers terminate the process promptly at the OS level.
    install_signal_handlers();
    match backend.companion_start(project_id, phase) {
        Ok(cid) => backend
            .companion_wait(&cid, Instant::now() + Duration::from_secs(365 * 24 * 3600))
            .map_err(MainError::Other),
        Err(e) => Err(MainError::TailnetPreparation(format!(
            "tailnet preparation unconfirmed; observe and explicitly retry: {e}"
        ))),
    }
}

// -- serve path (`serveHostSocket`) --

fn open_backend(config: &iconfig::Config) -> DaemonBackend {
    DaemonBackend::open(
        BackendConfig {
            project: project::Config {
                muse_socket: config.muse_socket.clone(),
                image: config.image.clone(),
                network: config.network.clone(),
                subnet: config.subnet.clone(),
                bridge: config.bridge.clone(),
            },
            codex_harness: config.codex_harness.clone(),
            codex_harness_sha256: config.codex_harness_sha256.clone(),
            codex_harness_version: config.codex_harness_version.clone(),
            broker_socket: config.identity_socket.clone(),
            tailnet_image: config.tailnet_image.clone(),
            muse: if config.muse_sha256.is_empty() {
                None
            } else {
                Some(MuseConfig {
                    version: config.muse_version.clone(),
                    sha256: config.muse_sha256.clone(),
                })
            },
        },
        FACTORY_STATE_DIR,
    )
}

fn serve_host_socket(config: iconfig::Config, listen_path: &str) -> Result<(), MainError> {
    if !is_root() {
        return Err(MainError::Other(
            "requires root and the soda-host systemd Unix socket".to_string(),
        ));
    }
    let listener = if listen_path.is_empty() {
        if std::env::var("LISTEN_PID").unwrap_or_default() != std::process::id().to_string()
            || std::env::var("LISTEN_FDS").unwrap_or_default() != "1"
        {
            return Err(MainError::Other(
                "requires root and the soda-host systemd Unix socket".to_string(),
            ));
        }
        gmux_server::systemd_listener().map_err(|e| MainError::Other(e.to_string()))?
    } else {
        gmux_server::bind_listener(listen_path).map_err(|e| MainError::Other(e.to_string()))?
    };

    let backend = Arc::new(open_backend(&config));
    let server = Arc::new(Server::new(
        Arc::clone(&backend),
        DaemonConfig {
            image: config.image.clone(),
            tailnet_management: config.tailnet_management,
            terminal_available: true,
            identity_available: true,
        },
    ));

    // Muse launch listener (Go `startRuntimeLaunchers`): empty socket
    // skips the launcher; open failures abort before serving.
    let muse_listener = if config.muse_socket.is_empty() {
        None
    } else {
        match muse_serve::open_muse_listener(&config.muse_socket) {
            Ok(listener) => Some(listener),
            Err(e) => return Err(MainError::Other(e)),
        }
    };

    install_signal_handlers();

    // Muse serve thread joins the 35s retirement window on shutdown.
    let muse_shutdown = Arc::new(AtomicBool::new(false));
    let muse_handle = muse_listener.map(|listener| {
        let launch = Arc::new(soda_host::muse::MuseLaunch::new(
            soda_host::muse::MuseRuntime::new(
                project::Native,
                soda_host::dbackend::HooksSeam::new(&config.identity_socket),
                config.muse_version.clone(),
                config.muse_sha256.clone(),
            ),
        ));
        let flag = Arc::clone(&muse_shutdown);
        let fd = listener.as_raw_fd();
        // The listener outlives the serve loop: it is dropped after the
        // join below while `serve` borrows only the fd.
        std::thread::spawn(move || {
            let result = launch.serve(fd, flag.as_ref());
            drop(listener);
            result
        })
    });

    // Serve until a signal arrives.
    let serve_server = Arc::clone(&server);
    let serve_handle = std::thread::spawn(move || {
        // `serve` polls its own shutdown flag; the listener is borrowed
        // for the loop and closed on return.
        serve_server.serve(&listener);
        drop(listener);
    });
    while !SHUTDOWN.load(Ordering::SeqCst) {
        if serve_handle.is_finished() {
            break;
        }
        std::thread::sleep(Duration::from_millis(50));
    }

    // Shutdown order mirrors Go: retire launch executions (35s), then
    // stop accepting and drain (5s), then report muse failures first.
    muse_shutdown.store(true, Ordering::SeqCst);
    let mut launch_failure: Option<String> = None;
    if let Some(handle) = muse_handle {
        let deadline = Instant::now() + MUSE_JOIN_TIMEOUT;
        loop {
            if handle.is_finished() {
                match handle.join() {
                    Ok(Ok(())) => {}
                    Ok(Err(e)) => launch_failure = Some(e),
                    Err(_) => {
                        launch_failure = Some("launch execution retirement panicked".to_string());
                    }
                }
                break;
            }
            if Instant::now() >= deadline {
                launch_failure =
                    Some("launch execution retirement remains unconfirmed".to_string());
                break;
            }
            std::thread::sleep(Duration::from_millis(50));
        }
    }
    server.shutdown();
    let drain = Instant::now() + SHUTDOWN_DRAIN;
    while server.inflight() > 0 && Instant::now() < drain {
        std::thread::sleep(Duration::from_millis(50));
    }
    let _ = serve_handle.join();

    if let Some(e) = launch_failure {
        return Err(MainError::Other(e));
    }
    Ok(())
}

fn run() -> Result<(), MainError> {
    let args: Vec<String> = std::env::args().skip(1).collect();
    let flags = match parse_flags(&args) {
        Ok(flags) => flags,
        Err(marker) if marker == "__help__" => {
            eprint!("{}", usage());
            std::process::exit(0);
        }
        Err(e) => {
            eprintln!("{e}");
            eprint!("{}", usage());
            std::process::exit(2);
        }
    };
    if !flags.tailnet_action.is_empty() {
        return run_tailnet_action(
            &flags.config,
            &flags.release,
            &flags.tailnet_action,
            &flags.project,
            flags.positionals.len(),
        );
    }
    if !flags.project.is_empty() || !flags.positionals.is_empty() {
        return Err(MainError::Other("invalid Tailnet request".to_string()));
    }
    let config = iconfig::load_config(&flags.config, &flags.release)
        .map_err(|e| MainError::Other(format!("invalid host configuration: {e}")))?;
    serve_host_socket(config, &flags.listen_path)
}

fn main() {
    if let Err(e) = run() {
        eprintln!("{}", e.message());
        std::process::exit(e.exit_code());
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    fn args(words: &[&str]) -> Vec<String> {
        words.iter().map(|w| w.to_string()).collect()
    }

    #[test]
    fn flag_forms() {
        let flags = parse_flags(&args(&["--config", "/x", "--project=p1"])).unwrap();
        assert_eq!(flags.config, "/x");
        assert_eq!(flags.project, "p1");
        assert_eq!(flags.release, RELEASE_CONFIG);
        let flags = parse_flags(&args(&["-config=/y"])).unwrap();
        assert_eq!(flags.config, "/y");
        let flags = parse_flags(&args(&["--release", ""])).unwrap();
        assert_eq!(flags.release, "");
        let flags = parse_flags(&args(&["--release=/r.json"])).unwrap();
        assert_eq!(flags.release, "/r.json");
        let flags = parse_flags(&args(&["--listen-path", "/s/st15.sock"])).unwrap();
        assert_eq!(flags.listen_path, "/s/st15.sock");
        let flags = parse_flags(&args(&[])).unwrap();
        assert!(flags.listen_path.is_empty());
        assert!(parse_flags(&args(&["--bogus"])).is_err());
        assert!(parse_flags(&args(&["--config"])).is_err());
        assert!(parse_flags(&args(&["--release"])).is_err());
        assert!(parse_flags(&args(&["--listen-path"])).is_err());
        assert!(parse_flags(&args(&["-h"])).is_err());
    }

    #[test]
    fn bind_listener_keeps_root_gate() {
        // SAFETY: `geteuid` is async-signal-safe and infallible.
        if unsafe { libc::geteuid() } == 0 {
            return;
        }
        let err = gmux_server::bind_listener("/nonexistent-dir-xyz/st15.sock").unwrap_err();
        assert_eq!(err.kind(), std::io::ErrorKind::PermissionDenied);
    }

    #[test]
    fn tailnet_action_args_rejected_without_root_or_shape() {
        // Shape failures reject even as root; non-root rejects everything.
        // Either way the result is an error, never a launch.
        let result = run_tailnet_action("/nonexistent.json", "", "bogus", "bad", 0);
        assert!(result.is_err());
        let result = run_tailnet_action("/nonexistent.json", "", "run", "bad id", 0);
        assert!(result.is_err());
    }

    #[test]
    fn exit_code_contract_and_systemd_retry() {
        // Go `exitStatus` parity (cmd/soda-host/main_test.go, retired at
        // cutover): preparation failures exit 78 even with staged
        // diagnostics appended; any other failure exits 1; clean is 0.
        let staged = MainError::TailnetPreparation(
            "tailnet preparation unconfirmed; observe and explicitly retry: synthetic backend refused"
                .to_string(),
        );
        assert_eq!(staged.exit_code(), 78);
        assert_eq!(MainError::Other("daemon exited".to_string()).exit_code(), 1);
        // The companion unit must not restart the explicit-retry signal.
        let unit = std::fs::read_to_string("../../appliance/services/soda-tailnet@.service")
            .expect("companion unit file");
        assert!(unit.contains("RestartPreventExitStatus=78\n"), "{unit}");
        assert!(unit.contains("Restart=on-failure\n"), "{unit}");
    }
}

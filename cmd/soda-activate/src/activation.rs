use std::fs::{self, OpenOptions};
use std::io::{self, Read, Write};
use std::net::IpAddr;
use std::os::unix::fs::{DirBuilderExt, MetadataExt, OpenOptionsExt, PermissionsExt};
use std::path::Path;

use crate::cli::CliArgs;
use crate::forgejo_env::rewrite_forgejo_env;
use crate::origin::{activate_rejects_ip, check_browser_origin, origin_host_port};
use crate::system::{runtime, usage, ActivateError, Paths, Sys};

const MAX_DASHBOARD_BYTES: usize = 64 * 1024;

pub(crate) fn activate(
    args: &CliArgs,
    paths: &Paths,
    sys: &mut dyn Sys,
    stdout: &mut dyn Write,
) -> Result<(), ActivateError> {
    if sys.euid() != 0 {
        return Err(usage("native host operator/root required"));
    }
    if args.local_tls {
        if args.certificate.is_some() || args.private_key.is_some() {
            return Err(usage(
                "local TLS and supplied certificates are separate choices",
            ));
        }
    } else if args.certificate.is_none() || args.private_key.is_none() {
        return Err(usage(
            "select --local-tls or supply both --certificate and --private-key",
        ));
    }
    let address: IpAddr = args
        .bind_ip
        .parse()
        .map_err(|e| runtime(format!("invalid bind IP {:?}: {e}", args.bind_ip)))?;
    if activate_rejects_ip(&address) {
        return Err(usage(
            "select an explicit private address, not loopback or a public administration exposure",
        ));
    }
    let root = &paths.root;
    if root.join("activated").exists() {
        return Err(usage("already activated; use explicit native configuration maintenance instead of blind reactivation"));
    }
    let dashboard_path = root.join("dashboard.json");
    let dashboard_file = fs::File::open(&dashboard_path)
        .map_err(|e| runtime(format!("cannot read dashboard.json: {e}")))?;
    let mut raw = Vec::with_capacity(MAX_DASHBOARD_BYTES + 1);
    dashboard_file
        .take((MAX_DASHBOARD_BYTES + 1) as u64)
        .read_to_end(&mut raw)
        .map_err(|e| runtime(format!("cannot read dashboard.json: {e}")))?;
    if raw.len() > MAX_DASHBOARD_BYTES {
        return Err(runtime("dashboard.json exceeds size limit"));
    }
    let value: serde_json::Value =
        serde_json::from_slice(&raw).map_err(|_| runtime("dashboard.json is not valid JSON"))?;
    let cfg = value
        .as_object()
        .ok_or_else(|| runtime("dashboard configuration must be a JSON object"))?;
    if cfg.contains_key("public_url") {
        return Err(usage(
            "legacy separate-origin configuration; use rehearsed configuration maintenance",
        ));
    }
    let operator_id = match cfg.get("operator_id").and_then(serde_json::Value::as_i64) {
        Some(id) if id > 0 => id,
        _ => return Err(usage("dashboard operator identity is invalid")),
    };
    let forgejo_url = cfg
        .get("forgejo_url")
        .and_then(serde_json::Value::as_str)
        .ok_or_else(|| runtime("dashboard forgejo_url must be a string"))?;
    check_browser_origin(forgejo_url)?;
    if args.local_tls {
        let origin = origin_host_port(forgejo_url)?;
        if origin.hostname != address.to_string()
            || (origin.port_present && origin.port != Some(443))
        {
            return Err(usage(
                "local TLS uses the exact selected private IP on HTTPS port 443",
            ));
        }
    }
    let internal = cfg
        .get("forgejo_internal_url")
        .and_then(serde_json::Value::as_str)
        .unwrap_or_default();
    let listen = cfg
        .get("listen")
        .and_then(serde_json::Value::as_str)
        .unwrap_or_default();
    if internal != "http://127.0.0.1:3000" || listen != "127.0.0.1:8080" {
        return Err(usage(
            "bundled service recipes require the documented internal listeners",
        ));
    }
    let grant_key = cfg
        .get("grant_key_file")
        .and_then(serde_json::Value::as_str)
        .unwrap_or_default();
    let host_socket = cfg
        .get("host_socket")
        .and_then(serde_json::Value::as_str)
        .unwrap_or_default();
    let identity_socket = cfg
        .get("identity_socket")
        .and_then(serde_json::Value::as_str)
        .filter(|value| !value.is_empty())
        .unwrap_or("/run/soda/identity/admin.sock");
    if grant_key != "/etc/soda/grant-key"
        || host_socket != "/run/soda/host.sock"
        || identity_socket != "/run/soda/identity/admin.sock"
    {
        return Err(usage(
            "bundled isolated service mounts require the standard private file and socket paths",
        ));
    }
    let (soda_uid, soda_gid) = sys.lookup_user("soda").map_err(ActivateError::Runtime)?;
    if soda_uid != 2000 || soda_gid != 2000 {
        return Err(usage("unexpected native soda service identity"));
    }
    // Mutations below run only after every validation above passed.
    fs::set_permissions(root, fs::Permissions::from_mode(0o750))
        .map_err(|e| runtime(e.to_string()))?;
    sys.chown(root, 0, soda_gid)
        .map_err(|e| runtime(e.to_string()))?;
    for name in ["dashboard.json", "grant-key"] {
        let path = root.join(name);
        if path.parent() != Some(root.as_path()) || is_symlink(&path) {
            return Err(usage(
                "credentials must be regular operator-created files under /etc/soda",
            ));
        }
        sys.chown(&path, 0, soda_gid)
            .map_err(|e| runtime(e.to_string()))?;
        fs::set_permissions(&path, fs::Permissions::from_mode(0o640))
            .map_err(|e| runtime(e.to_string()))?;
    }
    // The extension package needs only the stable operator ID for its navigation
    // guard. Keep it in the package's private data directory, outside the image.
    let extension_root = paths.var_lib.join("forgejo/gitea/extensions");
    for path in [
        &paths.var_lib,
        &paths.var_lib.join("forgejo"),
        &paths.var_lib.join("forgejo/gitea"),
        &extension_root,
    ] {
        if is_symlink(path) {
            return Err(usage(
                "Forgejo extension data path must not contain symlinks",
            ));
        }
    }
    let extension_data = extension_root.join(".data");
    fs::create_dir(&extension_root).map_err(|e| runtime(e.to_string()))?;
    sys.chown(&extension_root, 1000, 1000)
        .map_err(|e| runtime(e.to_string()))?;
    fs::set_permissions(&extension_root, fs::Permissions::from_mode(0o700))
        .map_err(|e| runtime(e.to_string()))?;
    if is_symlink(&extension_data) {
        return Err(usage(
            "Forgejo extension data path must not contain symlinks",
        ));
    }
    fs::create_dir(&extension_data).map_err(|e| runtime(e.to_string()))?;
    sys.chown(&extension_data, 1000, 1000)
        .map_err(|e| runtime(e.to_string()))?;
    fs::set_permissions(&extension_data, fs::Permissions::from_mode(0o700))
        .map_err(|e| runtime(e.to_string()))?;
    let operator_data = extension_data.join("soda");
    if is_symlink(&operator_data) {
        return Err(usage("Soda extension data path must not contain symlinks"));
    }
    fs::create_dir(&operator_data).map_err(|e| runtime(e.to_string()))?;
    sys.chown(&operator_data, 1000, 1000)
        .map_err(|e| runtime(e.to_string()))?;
    fs::set_permissions(&operator_data, fs::Permissions::from_mode(0o700))
        .map_err(|e| runtime(e.to_string()))?;
    let operator_file = operator_data.join("operator-id");
    if operator_file.exists() || is_symlink(&operator_file) {
        let st = fs::symlink_metadata(&operator_file).map_err(|e| runtime(e.to_string()))?;
        if !st.file_type().is_file() || st.nlink() != 1 {
            return Err(usage(
                "Soda extension operator identity must be a private regular file",
            ));
        }
    }
    fs::write(&operator_file, format!("{operator_id}\n")).map_err(|e| runtime(e.to_string()))?;
    sys.chown(&operator_file, 1000, 1000)
        .map_err(|e| runtime(e.to_string()))?;
    fs::set_permissions(&operator_file, fs::Permissions::from_mode(0o600))
        .map_err(|e| runtime(e.to_string()))?;
    let tls = root.join("tls");
    match fs::DirBuilder::new().mode(0o700).create(&tls) {
        Ok(()) => {}
        Err(e) if e.kind() == io::ErrorKind::AlreadyExists => {}
        Err(e) => return Err(runtime(e.to_string())),
    }
    if !args.local_tls {
        for (source, dest) in [
            (&args.certificate, "cert.pem"),
            (&args.private_key, "key.pem"),
        ] {
            let data =
                fs::read(source.as_deref().unwrap_or("")).map_err(|e| runtime(e.to_string()))?;
            let dest_path = tls.join(dest);
            fs::write(&dest_path, data).map_err(|e| runtime(e.to_string()))?;
            fs::set_permissions(&dest_path, fs::Permissions::from_mode(0o600))
                .map_err(|e| runtime(e.to_string()))?;
        }
    }
    let tls_input = if args.local_tls {
        "internal".to_string()
    } else {
        "/etc/soda/tls/cert.pem /etc/soda/tls/key.pem".to_string()
    };
    fs::write(
        root.join("proxy.env"),
        format!("FORGEJO_ORIGIN={forgejo_url}\nSODA_BIND={address}\nSODA_TLS={tls_input}\n"),
    )
    .map_err(|e| runtime(e.to_string()))?;
    fs::set_permissions(root.join("proxy.env"), fs::Permissions::from_mode(0o600))
        .map_err(|e| runtime(e.to_string()))?;
    rewrite_forgejo_env(&root.join("forgejo.env"), forgejo_url, &address.to_string())?;
    let network = paths.containers_systemd.join("forgejo.container.d");
    fs::create_dir_all(&network).map_err(|e| runtime(e.to_string()))?;
    let bind = match address {
        IpAddr::V6(_) => format!("[{address}]"),
        IpAddr::V4(_) => address.to_string(),
    };
    fs::write(network.join("10-private-network.conf"), format!("[Container]\nPublishPort=\nPublishPort=127.0.0.1:3000:3000\nPublishPort={bind}:2222:22\n"))
        .map_err(|e| runtime(e.to_string()))?;
    OpenOptions::new()
        .create(true)
        .write(true)
        .truncate(false)
        .mode(0o600)
        .open(root.join("activated"))
        .map_err(|e| runtime(e.to_string()))?;
    run_checked(sys, &["systemctl", "daemon-reload"])?;
    // No enable: Quadlet-generated units refuse it. Boot-time binding lives in
    // soda-console.service; activation only (re)starts the configured listeners.
    run_checked(sys, &["systemctl", "restart", "forgejo.service"])?;
    run_checked(
        sys,
        &[
            "systemctl",
            "start",
            "soda-dashboard.service",
            "soda-proxy.service",
        ],
    )?;
    let mut pending = vec![
        "forgejo.service",
        "soda-dashboard.service",
        "soda-proxy.service",
    ];
    let deadline = sys.elapsed() + std::time::Duration::from_secs(60);
    while !pending.is_empty() && sys.elapsed() < deadline {
        pending.retain(|unit| {
            sys.run(&["systemctl", "is-active", "--quiet", unit])
                .unwrap_or(1)
                != 0
        });
        if !pending.is_empty() {
            sys.sleep(5);
        }
    }
    if !pending.is_empty() {
        return Err(usage(format!(
            "activation started but not active: {}; inspect journalctl -u {}",
            pending.join(", "),
            pending[0]
        )));
    }
    let _ = writeln!(stdout, "Activation requested. Services report active; perform the later login/SSH validation before trusting browser setup.");
    if args.local_tls {
        let _ = writeln!(stdout, "Caddy manages local HTTPS certificates. Trust its public root certificate on each intended client before opening Soda; no domain or public certificate service is required.");
    }
    Ok(())
}

fn is_symlink(path: &Path) -> bool {
    fs::symlink_metadata(path)
        .map(|st| st.file_type().is_symlink())
        .unwrap_or(false)
}

fn run_checked(sys: &mut dyn Sys, argv: &[&str]) -> Result<(), ActivateError> {
    match sys.run(argv) {
        Ok(0) => Ok(()),
        Ok(code) => Err(runtime(format!(
            "Command {argv:?} returned non-zero exit status {code}."
        ))),
        Err(e) => Err(runtime(e.to_string())),
    }
}

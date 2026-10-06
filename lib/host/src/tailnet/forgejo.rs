//! Private Rust port of the `soda-forgejo-tailnet` refresh helper
//! (`cmd/soda-forgejo-tailnet/main.go` + `internal/forgejo/tailnet.go`):
//! root gate, 90s deadline, native Tailnet endpoint, `podman inspect`
//! listener admission, SSH_DOMAIN-only env rewrite, conditional restart.
//! Included by the thin `soda-forgejo-tailnet` bin; not part of the
//! `soda_host` library API, so only `soda_host::` paths are used here.

use std::os::unix::fs::OpenOptionsExt;
use std::os::unix::io::{IntoRawFd, RawFd};
use std::sync::atomic::{AtomicU64, Ordering};
use std::time::{Duration, Instant};

use soda_host::json::{decode_tolerant, Value};
use soda_host::project::{Executor, NativeStatusOnly};
use soda_host::tcontrol_native;

const DEADLINE_SECS: u64 = 90;
const PODMAN: &str = "/usr/bin/podman";
const SYSTEMCTL: &str = "/usr/bin/systemctl";
const TAILSCALE_CLI: &str = tcontrol_native::DEFAULT_CLI;
const FORGEJO_CONTAINER: &str = "soda-forgejo";
const FORGEJO_ENV: &str = "/etc/soda/forgejo.env";
const SSH_DOMAIN_KEY: &str = "FORGEJO__server__SSH_DOMAIN=";

/// Full refresh flow; mirrors Go `run`.
pub fn run(exec: &dyn Executor) -> Result<(), String> {
    if unsafe { libc::geteuid() } != 0 {
        return Err(String::from("host operator required"));
    }
    let deadline = Instant::now() + Duration::from_secs(DEADLINE_SECS);
    let endpoint = tcontrol_native::cli_endpoint(exec, TAILSCALE_CLI, deadline)?;
    // Do not advertise a Tailnet address while the native port only binds a LAN IP.
    // Raw inspection may contain credentials; never print it or include it in errors.
    let live = inspect_forgejo(exec, deadline)?;
    let (domain, running) = published_state(&live, &endpoint.ipv4)?;
    let changed = update_ssh_domain(FORGEJO_ENV, &endpoint.identity)?;
    // Restart diagnostics stay status-only like Go's Run: the shared
    // runner with the status-only error policy, never subprocess stderr.
    let status_exec = NativeStatusOnly;
    restart_forgejo_if_needed(
        &status_exec,
        changed,
        &domain,
        &endpoint.identity,
        running,
        deadline,
    )?;
    println!("Forgejo SSH address refreshed; configured browser/OAuth origins preserved.");
    Ok(())
}

fn inspect_forgejo(exec: &dyn Executor, deadline: Instant) -> Result<Vec<u8>, String> {
    exec.run(&[], PODMAN, &["inspect", FORGEJO_CONTAINER], deadline)
        .map_err(|_| {
            String::from("cannot inspect native Forgejo; inspect its operator service journal")
        })
}

/// Last matching object field (exact or ASCII case-insensitive). Go binds
/// each incoming key in order with exact-or-fold matching, so the last
/// matching key wins even across exact/fold forms; with distinct field
/// names this reverse lookup is equivalent. Null counts as absent.
fn field<'a>(fields: &'a [(String, Value)], name: &str) -> Option<&'a Value> {
    fields
        .iter()
        .rev()
        .find(|(k, _)| k == name || k.eq_ignore_ascii_case(name))
        .map(|(_, v)| v)
        .filter(|v| !v.is_null())
}

fn published_state(data: &[u8], ip: &str) -> Result<(String, bool), String> {
    let invalid = || String::from("cannot read native Forgejo network state");
    let value = decode_tolerant(data).map_err(|_| invalid())?;
    let items = value.as_array().ok_or_else(|| invalid())?;
    if items.len() != 1 {
        return Err(invalid());
    }
    let root = items[0].as_object().ok_or_else(|| invalid())?;
    // Go unmarshals the whole struct before the listener check, so shape
    // errors anywhere refuse as unreadable even when unbound.
    let bindings = port_bindings(root).ok_or_else(|| invalid())?;
    let (domain, running) = ssh_domain_state(root).ok_or_else(|| invalid())?;
    if !ssh_bound_to_ip(&bindings, ip) {
        return Err(format!(
            "forgejo Git SSH is not bound to Tailnet IP {ip}:2222; configure the intended private native listener before refreshing its advertised address"
        ));
    }
    Ok((domain, running))
}

/// (HostIp, HostPort) pairs for `22/tcp`. Every map value is validated
/// like Go's whole-struct unmarshal, so a malformed `80/tcp` refuses even
/// beside a valid `22/tcp`; the last exact `22/tcp` key wins like Go's map
/// binding, absent values stay empty.
fn port_bindings(root: &[(String, Value)]) -> Option<Vec<(String, String)>> {
    let host_config = match field(root, "HostConfig") {
        None => return Some(Vec::new()),
        Some(v) => v.as_object()?,
    };
    let map = match field(host_config, "PortBindings") {
        None => return Some(Vec::new()),
        Some(v) => v.as_object()?,
    };
    let mut out = Vec::new();
    for (key, value) in map {
        let pairs = binding_entries(value)?;
        if key == "22/tcp" {
            out = pairs;
        }
    }
    Some(out)
}

fn binding_entries(value: &Value) -> Option<Vec<(String, String)>> {
    let entries = match value {
        Value::Null => return Some(Vec::new()),
        v => v.as_array()?,
    };
    let mut out = Vec::new();
    for entry in entries {
        let fields = entry.as_object()?;
        let host_ip = match field(fields, "HostIp") {
            None => String::new(),
            Some(Value::Str(s)) => s.clone(),
            Some(_) => return None,
        };
        let host_port = match field(fields, "HostPort") {
            None => String::new(),
            Some(Value::Str(s)) => s.clone(),
            Some(_) => return None,
        };
        out.push((host_ip, host_port));
    }
    Some(out)
}

fn ssh_bound_to_ip(bindings: &[(String, String)], ip: &str) -> bool {
    bindings.iter().any(|(host_ip, host_port)| {
        host_port == "2222" && (host_ip == ip || host_ip == "0.0.0.0" || host_ip.is_empty())
    })
}

fn ssh_domain_state(root: &[(String, Value)]) -> Option<(String, bool)> {
    let mut domain = String::new();
    if let Some(config) = field(root, "Config") {
        let fields = config.as_object()?;
        if let Some(env_value) = field(fields, "Env") {
            // Every item is typed like Go's unmarshal, but the first key
            // match still wins even when later items follow.
            let mut found = false;
            for value in env_value.as_array()? {
                let line = match value {
                    Value::Null => continue,
                    Value::Str(s) => s,
                    _ => return None,
                };
                if !found {
                    if let Some(stripped) = line.strip_prefix(SSH_DOMAIN_KEY) {
                        domain = stripped.to_string();
                        found = true;
                    }
                }
            }
        }
    }
    let running = match field(root, "State") {
        None => false,
        Some(v) => {
            let state = v.as_object()?;
            match field(state, "Running") {
                None => false,
                Some(Value::Bool(b)) => *b,
                Some(_) => return None,
            }
        }
    };
    Some((domain, running))
}

fn valid_endpoint_name(endpoint: &str) -> bool {
    // ^[A-Za-z0-9][A-Za-z0-9.:-]*$ without a regex dependency.
    let mut chars = endpoint.chars();
    match chars.next() {
        Some(c) if c.is_ascii_alphanumeric() => {}
        _ => return false,
    }
    chars.all(|c| c.is_ascii_alphanumeric() || matches!(c, '.' | ':' | '-'))
}

fn update_ssh_domain(path: &str, endpoint: &str) -> Result<bool, String> {
    if !valid_endpoint_name(endpoint) {
        return Err(String::from("invalid native Tailnet endpoint"));
    }
    let before = std::fs::read(path).map_err(|e| format!("open {path}: {e}"))?;
    let updated = rewrite_ssh_domain(&before, endpoint);
    if updated == before {
        return Ok(false);
    }
    write_forgejo_env(path, &updated)?;
    Ok(true)
}

/// Byte-level port of Go `rewriteSSHDomain`, so non-UTF8 files round-trip
/// exactly like the Go string rewrite: first key line replaced, later key
/// lines dropped, key appended when missing, single trailing newline.
fn rewrite_ssh_domain(content: &[u8], endpoint: &str) -> Vec<u8> {
    let key = SSH_DOMAIN_KEY.as_bytes();
    let value = [key, endpoint.as_bytes()].concat();
    let mut trimmed = content;
    while trimmed.last() == Some(&b'\n') {
        trimmed = &trimmed[..trimmed.len() - 1];
    }
    let mut found = false;
    let mut out: Vec<&[u8]> = Vec::new();
    for line in trimmed.split(|b| *b == b'\n') {
        if line.starts_with(key) {
            if !found {
                out.push(&value);
                found = true;
            }
            continue;
        }
        out.push(line);
    }
    if !found {
        out.push(&value);
    }
    let mut joined = Vec::new();
    for (i, line) in out.iter().enumerate() {
        if i > 0 {
            joined.push(b'\n');
        }
        joined.extend_from_slice(line);
    }
    joined.push(b'\n');
    joined
}

/// 0600 private temp in the target directory, chmod/write/close checks,
/// atomic rename; the temp is removed on every failure path.
fn write_forgejo_env(path: &str, updated: &[u8]) -> Result<(), String> {
    static TEMP_SEQ: AtomicU64 = AtomicU64::new(0);
    let dir = match std::path::Path::new(path).parent() {
        Some(p) if !p.as_os_str().is_empty() => p.to_path_buf(),
        _ => std::path::PathBuf::from("."),
    };
    let pid = std::process::id();
    let mut temp_path = std::path::PathBuf::new();
    let mut file = None;
    let mut last_err = String::new();
    for _ in 0..100 {
        let id = TEMP_SEQ.fetch_add(1, Ordering::Relaxed);
        temp_path = dir.join(format!(".forgejo-env-{pid}-{id}"));
        match std::fs::OpenOptions::new()
            .write(true)
            .create_new(true)
            .mode(0o600)
            .open(&temp_path)
        {
            Ok(f) => {
                file = Some(f);
                break;
            }
            Err(e) if e.kind() == std::io::ErrorKind::AlreadyExists => continue,
            Err(e) => {
                last_err = format!("open {}: {e}", temp_path.display());
                break;
            }
        }
    }
    let file = match file {
        Some(f) => f,
        None if last_err.is_empty() => {
            return Err(format!(
                "create temp file in {}: too many collisions",
                dir.display()
            ));
        }
        None => return Err(last_err),
    };
    // Owned-descriptor staging with a checked close, mirroring Go's
    // explicit close check that File's Drop cannot report. The fd closes
    // exactly once on every path below; rename runs only after a
    // successful chmod, write, and close.
    let fd = file.into_raw_fd();
    let staged = stage_forgejo_env(fd, &temp_path, updated);
    let closed = close_owned(fd).map_err(|e| format!("close {}: {e}", temp_path.display()));
    if let Err(e) = staged {
        let _ = std::fs::remove_file(&temp_path);
        return Err(e);
    }
    if let Err(e) = closed {
        let _ = std::fs::remove_file(&temp_path);
        return Err(e);
    }
    if let Err(e) = std::fs::rename(&temp_path, path) {
        let _ = std::fs::remove_file(&temp_path);
        return Err(format!("rename {} {path}: {e}", temp_path.display()));
    }
    Ok(())
}

/// chmod + full write on an owned staged descriptor; the caller owns the
/// checked close. First error wins, like Go's staged checks.
fn stage_forgejo_env(fd: RawFd, temp: &std::path::Path, updated: &[u8]) -> Result<(), String> {
    if unsafe { libc::fchmod(fd, 0o600) } != 0 {
        return Err(format!(
            "chmod {}: {}",
            temp.display(),
            std::io::Error::last_os_error()
        ));
    }
    let mut written = 0;
    while written < updated.len() {
        let n = unsafe {
            libc::write(
                fd,
                updated[written..].as_ptr() as *const libc::c_void,
                updated.len() - written,
            )
        };
        if n < 0 {
            let e = std::io::Error::last_os_error();
            if e.kind() == std::io::ErrorKind::Interrupted {
                continue;
            }
            return Err(format!("write {}: {e}", temp.display()));
        }
        written += n as usize;
    }
    Ok(())
}

/// Checked close for an owned descriptor.
fn close_owned(fd: RawFd) -> Result<(), std::io::Error> {
    if unsafe { libc::close(fd) } == 0 {
        Ok(())
    } else {
        Err(std::io::Error::last_os_error())
    }
}

fn restart_forgejo_if_needed(
    exec: &dyn Executor,
    changed: bool,
    domain: &str,
    identity: &str,
    running: bool,
    deadline: Instant,
) -> Result<(), String> {
    if changed || domain != identity || !running {
        exec.run(&[], SYSTEMCTL, &["restart", "forgejo.service"], deadline)
            .map_err(|e| format!("native Forgejo configuration saved, restart failed: {e}"))?;
    }
    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::cell::RefCell;

    struct Mock {
        calls: RefCell<Vec<(String, Vec<String>)>>,
        inspect: Result<Vec<u8>, String>,
        restart_err: Option<String>,
    }

    impl Mock {
        fn new(inspect: Result<Vec<u8>, String>) -> Mock {
            Mock {
                calls: RefCell::new(Vec::new()),
                inspect,
                restart_err: None,
            }
        }

        fn argv(&self) -> Vec<(String, Vec<String>)> {
            self.calls.borrow().clone()
        }
    }

    impl Executor for Mock {
        fn run(
            &self,
            _stdin: &[u8],
            cmd: &str,
            args: &[&str],
            _deadline: Instant,
        ) -> Result<Vec<u8>, String> {
            self.calls.borrow_mut().push((
                cmd.to_string(),
                args.iter().map(|s| s.to_string()).collect(),
            ));
            match cmd {
                PODMAN => self.inspect.clone(),
                SYSTEMCTL => match &self.restart_err {
                    Some(e) => Err(e.clone()),
                    None => Ok(Vec::new()),
                },
                other => Err(format!("unexpected command {other}")),
            }
        }
    }

    static TEST_SEQ: AtomicU64 = AtomicU64::new(0);

    struct TestEnv {
        dir: std::path::PathBuf,
    }

    impl TestEnv {
        fn fresh(tag: &str) -> TestEnv {
            let id = TEST_SEQ.fetch_add(1, Ordering::SeqCst);
            let dir = std::env::temp_dir()
                .join(format!("forgejo-tailnet-{}-{id}-{tag}", std::process::id()));
            let _ = std::fs::remove_dir_all(&dir);
            std::fs::create_dir_all(&dir).unwrap();
            TestEnv { dir }
        }

        fn path(&self, name: &str) -> String {
            self.dir.join(name).to_string_lossy().into_owned()
        }
    }

    impl Drop for TestEnv {
        fn drop(&mut self) {
            let _ = std::fs::remove_dir_all(&self.dir);
        }
    }

    const PUBLISHED: &str = r#"[{"Config":{"Env":["SECRET=never-log","FORGEJO__server__SSH_DOMAIN=old"]},"State":{"Running":true},"HostConfig":{"PortBindings":{"22/tcp":[{"HostIp":"100.64.0.1","HostPort":"2222"}]}}}]"#;

    #[test]
    fn refresh_requires_actual_tailnet_listener() {
        let (domain, running) = published_state(PUBLISHED.as_bytes(), "100.64.0.1").unwrap();
        assert_eq!(domain, "old");
        assert!(running);
        let moved = PUBLISHED.replacen("100.64.0.1", "192.168.1.10", 1);
        let err = published_state(moved.as_bytes(), "100.64.0.1").unwrap_err();
        assert!(
            err.starts_with("forgejo Git SSH is not bound to Tailnet IP 100.64.0.1:2222"),
            "missing private listener guard: {err}"
        );
        assert!(!err.contains("never-log"), "credential leakage: {err}");
    }

    #[test]
    fn published_state_rejects_malformed() {
        let bad = [
            "",
            "{oops",
            "[]",
            "[{},{}]",
            "[5]",
            r#"[{"HostConfig": 5}]"#,
            r#"[{"HostConfig": {"PortBindings": []}}]"#,
            r#"[{"HostConfig": {"PortBindings": {"22/tcp": {}}}}]"#,
            r#"[{"HostConfig": {"PortBindings": {"22/tcp": [5]}}}]"#,
            r#"[{"HostConfig": {"PortBindings": {"22/tcp": [{"HostPort": 2222}]}}}]"#,
            r#"[{"Config": {"Env": [1]}}]"#,
            r#"[{"State": {"Running": "yes"}}]"#,
        ];
        for input in bad {
            assert_eq!(
                published_state(input.as_bytes(), "100.64.0.1").unwrap_err(),
                "cannot read native Forgejo network state",
                "input {input:?}"
            );
        }
    }

    #[test]
    fn inspection_validates_every_port_binding() {
        let ssh = r#""22/tcp":[{"HostIp":"100.64.0.1","HostPort":"2222"}]"#;
        // A malformed sibling refuses even beside a valid 22/tcp, like Go's
        // whole-struct unmarshal.
        for extra in [
            r#""80/tcp":{}"#,
            r#""80/tcp":[5]"#,
            r#""80/tcp":[{"HostIp":1,"HostPort":"80"}]"#,
            r#""80/tcp":[{"HostIp":"0.0.0.0","HostPort":80}]"#,
            r#""80/tcp":[{"HostIp":["0.0.0.0"],"HostPort":"80"}]"#,
        ] {
            let input = PUBLISHED.replacen(ssh, &format!("{ssh},{extra}"), 1);
            assert_eq!(
                published_state(input.as_bytes(), "100.64.0.1").unwrap_err(),
                "cannot read native Forgejo network state",
                "extra {extra:?}"
            );
        }
        // Well-typed and null siblings pass through to admission.
        for extra in [
            r#""80/tcp":[{"HostIp":"0.0.0.0","HostPort":"80"}]"#,
            r#""80/tcp":[]"#,
            r#""80/tcp":null"#,
        ] {
            let input = PUBLISHED.replacen(ssh, &format!("{ssh},{extra}"), 1);
            let (domain, running) = published_state(input.as_bytes(), "100.64.0.1").unwrap();
            assert_eq!((domain.as_str(), running), ("old", true));
        }
    }

    #[test]
    fn inspection_validates_every_env_item() {
        let bound = |env: &str| {
            let body = "[{\"Config\": {\"Env\": ".to_string()
                + env
                + "}, \"State\": {\"Running\": true}, \"HostConfig\": {\"PortBindings\": {\"22/tcp\": [{\"HostIp\": \"100.64.0.1\", \"HostPort\": \"2222\"}]}}}]";
            published_state(body.as_bytes(), "100.64.0.1")
        };
        // A later mistyped item refuses even after a valid SSH_DOMAIN.
        assert_eq!(
            bound(r#"["FORGEJO__server__SSH_DOMAIN=first",7]"#).unwrap_err(),
            "cannot read native Forgejo network state"
        );
        assert_eq!(
            bound(r#"["FORGEJO__server__SSH_DOMAIN=first",{}]"#).unwrap_err(),
            "cannot read native Forgejo network state"
        );
        // Null items decode to empty like Go and never match the key.
        assert_eq!(
            bound(r#"["A=1",null,"FORGEJO__server__SSH_DOMAIN=late"]"#).unwrap(),
            ("late".to_string(), true)
        );
        // The first key match wins even with an empty value.
        assert_eq!(
            bound(r#"["FORGEJO__server__SSH_DOMAIN=","FORGEJO__server__SSH_DOMAIN=late"]"#)
                .unwrap(),
            (String::new(), true)
        );
    }

    #[test]
    fn duplicate_and_case_merge() {
        // Duplicate 22/tcp keys: last wins like Go's map binding.
        let unbound = r#""22/tcp":[{"HostIp":"192.168.1.10","HostPort":"2222"}]"#;
        let bound = r#""22/tcp":[{"HostIp":"100.64.0.1","HostPort":"2222"}]"#;
        for (first, second, admitted) in [(unbound, bound, true), (bound, unbound, false)] {
            let input = format!(
                r#"[{{"Config":{{"Env":[]}},"State":{{"Running":true}},"HostConfig":{{"PortBindings":{{{first},{second}}}}}}}]"#
            );
            if admitted {
                assert_eq!(
                    published_state(input.as_bytes(), "100.64.0.1").unwrap(),
                    (String::new(), true)
                );
            } else {
                assert!(published_state(input.as_bytes(), "100.64.0.1")
                    .unwrap_err()
                    .starts_with("forgejo Git SSH is not bound to Tailnet IP"));
            }
        }
        // Duplicate entry fields: last wins, exact or folded.
        let input = r#"[{"Config":{"Env":[]},"State":{"Running":true},"HostConfig":{"PortBindings":{"22/tcp":[{"HostIp":"192.168.1.10","hostip":"100.64.0.1","HostPort":"2222"}]}}}]"#;
        assert_eq!(
            published_state(input.as_bytes(), "100.64.0.1").unwrap(),
            (String::new(), true)
        );
        // Case-variant field names bind like Go's fold match.
        let input = r#"[{"config":{"env":["FORGEJO__server__SSH_DOMAIN=fold"]},"state":{"running":true},"hostconfig":{"portbindings":{"22/tcp":[{"hostip":"100.64.0.1","hostport":"2222"}]}}}]"#;
        assert_eq!(
            published_state(input.as_bytes(), "100.64.0.1").unwrap(),
            ("fold".to_string(), true)
        );
        // Map keys stay exact: a folded port key is not 22/tcp.
        let input = r#"[{"Config":{"Env":[]},"State":{"Running":true},"HostConfig":{"PortBindings":{"22/TCP":[{"HostIp":"100.64.0.1","HostPort":"2222"}]}}}]"#;
        assert!(published_state(input.as_bytes(), "100.64.0.1")
            .unwrap_err()
            .starts_with("forgejo Git SSH is not bound to Tailnet IP"));
    }

    #[test]
    fn bound_to_ip_matrix() {
        let ip = "100.64.0.1";
        let yes: &[&[(String, String)]] = &[
            &[("100.64.0.1".to_string(), "2222".to_string())],
            &[("0.0.0.0".to_string(), "2222".to_string())],
            &[(String::new(), "2222".to_string())],
            &[
                ("192.168.1.10".to_string(), "2222".to_string()),
                ("100.64.0.1".to_string(), "2222".to_string()),
            ],
        ];
        for bindings in yes {
            assert!(ssh_bound_to_ip(bindings, ip), "bindings {bindings:?}");
        }
        let no: &[&[(String, String)]] = &[
            &[],
            &[("100.64.0.1".to_string(), "2223".to_string())],
            &[("192.168.1.10".to_string(), "2222".to_string())],
            &[("100.64.0.1".to_string(), String::new())],
        ];
        for bindings in no {
            assert!(!ssh_bound_to_ip(bindings, ip), "bindings {bindings:?}");
        }
    }

    #[test]
    fn ssh_domain_env_cases() {
        let root = |env: &str, running: &str| {
            let body =
                format!(r#"[{{"Config": {{"Env": {env}}}, "State": {{"Running": {running}}}}}]"#);
            published_state(body.as_bytes(), "100.64.0.1").unwrap_err()
        };
        // Unbound listener refuses before env parsing; assert via bound input.
        let bound = |env: &str, running: &str| {
            let body = "[{\"Config\": {\"Env\": ".to_string()
                + env
                + "}, \"State\": {\"Running\": "
                + running
                + "}, \"HostConfig\": {\"PortBindings\": {\"22/tcp\": [{\"HostIp\": \"100.64.0.1\", \"HostPort\": \"2222\"}]}}}]";
            published_state(body.as_bytes(), "100.64.0.1").unwrap()
        };
        assert_eq!(
            bound(
                r#"["A=1","FORGEJO__server__SSH_DOMAIN=first","FORGEJO__server__SSH_DOMAIN=second"]"#,
                "true"
            ),
            ("first".to_string(), true)
        );
        assert_eq!(bound("[]", "false"), (String::new(), false));
        assert_eq!(
            root("[]", "false"),
            "forgejo Git SSH is not bound to Tailnet IP 100.64.0.1:2222; configure the intended private native listener before refreshing its advertised address"
        );
    }

    #[test]
    fn tailnet_preserves_browser_origin() {
        let env = TestEnv::fresh("origin");
        let path = env.path("forgejo.env");
        let original = "FORGEJO__server__ROOT_URL=https://forgejo.example/\n";
        std::fs::write(&path, original).unwrap();
        assert!(update_ssh_domain(&path, "100.100.0.1").unwrap());
        let body = std::fs::read_to_string(&path).unwrap();
        assert!(body.contains(original), "browser origin changed");
        assert!(
            body.contains("FORGEJO__server__SSH_DOMAIN=100.100.0.1\n"),
            "endpoint missing: {body:?}"
        );
        assert!(
            !update_ssh_domain(&path, "100.100.0.1").unwrap(),
            "unchanged native endpoint should not restart service"
        );
    }

    #[test]
    fn rewrite_matrix() {
        let key = "FORGEJO__server__SSH_DOMAIN=";
        // Missing key appends; absent trailing newline is added.
        assert_eq!(
            rewrite_ssh_domain(b"a=1", "e"),
            format!("a=1\n{key}e\n").into_bytes()
        );
        // Extra trailing newlines collapse to one.
        assert_eq!(
            rewrite_ssh_domain(b"a=1\n\n\n", "e"),
            format!("a=1\n{key}e\n").into_bytes()
        );
        // First key line replaced, later key lines dropped.
        assert_eq!(
            rewrite_ssh_domain(format!("{key}old\nb=2\n{key}older\n").as_bytes(), "e"),
            format!("{key}e\nb=2\n").into_bytes()
        );
        // Empty file keeps Go's leading blank line.
        assert_eq!(
            rewrite_ssh_domain(b"", "e"),
            format!("\n{key}e\n").into_bytes()
        );
        // Unrelated lines byte-round-trip, non-UTF8 included.
        let raw = b"\xff\xfe=a\nb=2\n";
        let out = rewrite_ssh_domain(raw, "e");
        assert!(out.starts_with(b"\xff\xfe=a\nb=2\n"));
        assert!(out.ends_with(format!("{key}e\n").as_bytes()));
    }

    #[test]
    fn endpoint_name_matrix() {
        for valid in [
            "100.100.0.1",
            "a",
            "A0.:-x",
            "host.example.com",
            "fe80::1",
            "0",
        ] {
            assert!(valid_endpoint_name(valid), "valid {valid:?}");
        }
        for invalid in ["", "-x", ".x", ":x", "a b", "a/b", "a_b", "é", "a\nb"] {
            assert!(!valid_endpoint_name(invalid), "invalid {invalid:?}");
        }
    }

    #[test]
    fn invalid_endpoint_refused() {
        let env = TestEnv::fresh("endpoint");
        let path = env.path("forgejo.env");
        std::fs::write(&path, b"a=1\n").unwrap();
        assert_eq!(
            update_ssh_domain(&path, "../x").unwrap_err(),
            "invalid native Tailnet endpoint"
        );
        assert_eq!(
            update_ssh_domain(&env.path("missing.env"), "100.64.0.1").is_err(),
            true
        );
    }

    #[test]
    fn written_env_is_private() {
        use std::os::unix::fs::MetadataExt;
        let env = TestEnv::fresh("mode");
        let path = env.path("forgejo.env");
        std::fs::write(&path, b"a=1\n").unwrap();
        assert!(update_ssh_domain(&path, "100.64.0.1").unwrap());
        assert_eq!(std::fs::metadata(&path).unwrap().mode() & 0o777, 0o600);
        assert_eq!(
            std::fs::read(&path).unwrap(),
            b"a=1\nFORGEJO__server__SSH_DOMAIN=100.64.0.1\n"
        );
        let leftovers: Vec<_> = std::fs::read_dir(&env.dir)
            .unwrap()
            .filter_map(|e| e.ok())
            .map(|e| e.file_name().to_string_lossy().into_owned())
            .filter(|n| n.starts_with(".forgejo-env-"))
            .collect();
        assert!(leftovers.is_empty(), "temp leftovers: {leftovers:?}");
    }

    #[test]
    fn checked_close_reports_errors() {
        // A second close of the same owned descriptor deterministically
        // fails, proving close errors surface instead of dropping.
        let file = std::fs::File::open("/dev/null").unwrap();
        let fd = file.into_raw_fd();
        close_owned(fd).unwrap();
        assert!(close_owned(fd).is_err());
    }

    #[test]
    fn restart_conditions_and_argv() {
        let deadline = Instant::now() + Duration::from_secs(90);
        let expected = vec![(
            SYSTEMCTL.to_string(),
            vec!["restart".to_string(), "forgejo.service".to_string()],
        )];
        // Changed config restarts.
        let mock = Mock::new(Ok(Vec::new()));
        restart_forgejo_if_needed(&mock, true, "id", "id", true, deadline).unwrap();
        assert_eq!(mock.argv(), expected);
        // Drifted domain restarts.
        let mock = Mock::new(Ok(Vec::new()));
        restart_forgejo_if_needed(&mock, false, "old", "new", true, deadline).unwrap();
        assert_eq!(mock.argv(), expected);
        // Stopped container restarts.
        let mock = Mock::new(Ok(Vec::new()));
        restart_forgejo_if_needed(&mock, false, "id", "id", false, deadline).unwrap();
        assert_eq!(mock.argv(), expected);
        // Steady state runs nothing.
        let mock = Mock::new(Ok(Vec::new()));
        restart_forgejo_if_needed(&mock, false, "id", "id", true, deadline).unwrap();
        assert!(mock.argv().is_empty());
        // Failure keeps the exact saved-configuration prefix.
        let mut mock = Mock::new(Ok(Vec::new()));
        mock.restart_err = Some(String::from("exit status 1"));
        assert_eq!(
            restart_forgejo_if_needed(&mock, true, "id", "id", true, deadline).unwrap_err(),
            "native Forgejo configuration saved, restart failed: exit status 1"
        );
    }

    #[test]
    fn status_only_reports_real_status() {
        let exec = NativeStatusOnly;
        let deadline = Instant::now() + Duration::from_secs(30);
        // Real exit code, status-only like Go's Run.
        assert_eq!(
            exec.run(&[], "/bin/sh", &["-c", "exit 3"], deadline)
                .unwrap_err(),
            "exit status 3"
        );
        // Subprocess stderr never surfaces, even on failure.
        assert_eq!(
            exec.run(
                &[],
                "/bin/sh",
                &["-c", "echo SECRET=never-log >&2; exit 3"],
                deadline
            )
            .unwrap_err(),
            "exit status 3"
        );
        assert!(exec.run(&[], "/bin/true", &[], deadline).is_ok());
    }

    #[test]
    fn status_only_reports_real_signal() {
        let exec = NativeStatusOnly;
        let deadline = Instant::now() + Duration::from_secs(30);
        assert_eq!(
            exec.run(&[], "/bin/sh", &["-c", "kill -TERM $$"], deadline)
                .unwrap_err(),
            "signal: terminated"
        );
    }

    #[test]
    fn status_only_argv_passthrough() {
        use std::os::unix::fs::PermissionsExt;
        let env = TestEnv::fresh("argv");
        let script = env.path("record.sh");
        let record = env.path("argv.bin");
        std::fs::write(
            &script,
            format!("#!/bin/sh\nprintf \"%s\\0\" \"$@\" > \"{record}\"\n"),
        )
        .unwrap();
        std::fs::set_permissions(&script, std::fs::Permissions::from_mode(0o755)).unwrap();
        let exec = NativeStatusOnly;
        let deadline = Instant::now() + Duration::from_secs(30);
        exec.run(&[], &script, &["restart", "forgejo.service"], deadline)
            .unwrap();
        assert_eq!(
            std::fs::read(&record).unwrap(),
            b"restart\0forgejo.service\0"
        );
    }

    #[test]
    fn status_only_spawn_failure() {
        let exec = NativeStatusOnly;
        let deadline = Instant::now() + Duration::from_secs(30);
        let err = exec
            .run(&[], "/nonexistent-forgejo-helper", &[], deadline)
            .unwrap_err();
        assert!(
            err.starts_with("fork/exec /nonexistent-forgejo-helper: "),
            "unexpected spawn diagnostic: {err}"
        );
    }

    #[test]
    fn status_only_deadline_kills() {
        let exec = NativeStatusOnly;
        // An already-expired deadline kills deterministically: sleep cannot
        // exit before the first poll observes the expiry.
        let deadline = Instant::now();
        assert_eq!(
            exec.run(&[], "/bin/sleep", &["60"], deadline).unwrap_err(),
            "signal: killed"
        );
    }

    #[test]
    fn inspect_failure_sanitized() {
        let deadline = Instant::now() + Duration::from_secs(90);
        let mock = Mock::new(Err(String::from("SECRET=never-log: exit 1")));
        let err = inspect_forgejo(&mock, deadline).unwrap_err();
        assert_eq!(
            err,
            "cannot inspect native Forgejo; inspect its operator service journal"
        );
        assert!(!err.contains("never-log"));
        assert_eq!(
            mock.argv(),
            vec![(
                PODMAN.to_string(),
                vec!["inspect".to_string(), "soda-forgejo".to_string()]
            )]
        );
        let mock = Mock::new(Ok(b"live".to_vec()));
        assert_eq!(inspect_forgejo(&mock, deadline).unwrap(), b"live".to_vec());
    }

    #[test]
    fn root_gate_refused() {
        if unsafe { libc::geteuid() } == 0 {
            eprintln!("SKIP root-gate case: running as root");
            return;
        }
        let mock = Mock::new(Ok(Vec::new()));
        assert_eq!(run(&mock).unwrap_err(), "host operator required");
        assert!(mock.argv().is_empty(), "gate must precede effects");
    }
}

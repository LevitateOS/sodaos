use super::*;
use crate::account::Account;

fn sample() -> Account {
    Account {
        pw_name: "op".to_string(),
        pw_uid: 1001,
        pw_gid: 1001,
        pw_dir: "/home/op".to_string(),
        pw_shell: "/bin/bash".to_string(),
    }
}

fn show_fixture(active: &str, load: &str) -> Vec<u8> {
    let id = "a".repeat(32);
    let unit = unit_name(&id);
    format!(
        "LoadState={load}\nActiveState={active}\nDescription=Soda terminal {id}\nFragmentPath=/run/systemd/transient/{unit}\nDropInPaths=\nType=exec\nUser=op\nKillMode=control-group\nRestart=no\nSendSIGKILL=yes\nTimeoutStopUSec=3s\nStandardInput=null\nStandardOutput=null\nStandardError=null\n"
    )
    .into_bytes()
}

#[test]
fn argv_constructors_exact() {
    let id = "a".repeat(32);
    let unit = unit_name(&id);
    assert_eq!(unit, format!("soda-terminal-{id}.service"));
    assert_eq!(
        systemctl_show_argv(&unit),
        vec![
            "/usr/bin/systemctl".to_string(),
            "show".to_string(),
            unit.clone(),
            format!("--property={}", service_properties()),
        ]
    );
    assert_eq!(
        systemctl_stop_argv(&unit),
        vec![
            "/usr/bin/systemctl".to_string(),
            "stop".to_string(),
            unit.clone()
        ]
    );
    assert_eq!(
        invocation_show_argv(&unit),
        vec![
            "/usr/bin/systemctl".to_string(),
            "show".to_string(),
            "--value".to_string(),
            "--property=InvocationID".to_string(),
            unit,
        ]
    );
    assert_eq!(
        stat_fs_argv(7),
        vec![
            "/usr/bin/stat".to_string(),
            "-f".to_string(),
            "-c".to_string(),
            "%T".to_string(),
            "/proc/self/fd/7".to_string(),
        ]
    );
    assert_eq!(
        infocmp_argv("screen-256color"),
        vec![
            "/usr/bin/infocmp".to_string(),
            "screen-256color".to_string()
        ]
    );
    assert_eq!(
        tmux_argv("/run/s/x", &["new-session".to_string(), "-d".to_string()]),
        vec![
            "/usr/bin/tmux".to_string(),
            "-N".to_string(),
            "-S".to_string(),
            "/run/s/x".to_string(),
            "new-session".to_string(),
            "-d".to_string(),
        ]
    );
}

#[test]
fn service_fields_matrix() {
    let id = "a".repeat(32);
    let fields =
        parse_service_fields(&show_fixture("active", "loaded"), service_properties()).unwrap();
    assert_eq!(fields.get("ActiveState").unwrap(), "active");
    verify_loaded_unit(&fields, &id, &sample()).unwrap();
    // Duplicate keys: last wins.
    let mut dup = show_fixture("inactive", "loaded");
    dup.extend_from_slice(b"ActiveState=active\n");
    let fields = parse_service_fields(&dup, service_properties()).unwrap();
    assert_eq!(fields.get("ActiveState").unwrap(), "active");
    // Oversize, non-ASCII, malformed line, missing key.
    assert!(parse_service_fields(&vec![b'x'; 4097], service_properties()).is_err());
    assert!(parse_service_fields(b"LoadState=\xff\n", service_properties()).is_err());
    assert!(parse_service_fields(b"no-equals-here\n", service_properties()).is_err());
    assert!(parse_service_fields(b"LoadState=loaded\n", service_properties()).is_err());
    assert!(parse_service_fields(b"", service_properties()).is_err());
    // Blank middle line raises like `dict(''.split('=', 1))`.
    let mut blank = show_fixture("active", "loaded");
    blank.extend_from_slice(b"\n");
    assert!(parse_service_fields(&blank, service_properties()).is_err());
    // Supervision drift refused.
    let mut drift = show_fixture("active", "loaded");
    let text = String::from_utf8(drift.clone())
        .unwrap()
        .replace("Restart=no", "Restart=always");
    drift = text.into_bytes();
    let fields = parse_service_fields(&drift, service_properties()).unwrap();
    assert!(verify_loaded_unit(&fields, &id, &sample()).is_err());
    // Foreign unit refused.
    let foreign = show_fixture("active", "loaded");
    assert!(verify_loaded_unit(
        &parse_service_fields(&foreign, service_properties()).unwrap(),
        &"b".repeat(32),
        &sample()
    )
    .is_err());
}

#[test]
fn cgroup_events_matrix() {
    assert!(parse_cgroup_populated(b"populated 0\nfrozen 0\n").unwrap());
    assert!(!parse_cgroup_populated(b"populated 1\n").unwrap());
    assert!(!parse_cgroup_populated(b"").unwrap());
    assert!(parse_cgroup_populated(b"populated 0\n\n").is_err()); // blank middle
    assert!(parse_cgroup_populated(b"populated\n").is_err());
    assert!(parse_cgroup_populated(b"a b c\n").is_err());
    assert!(parse_cgroup_populated(b"\xff\n").is_err());
}

#[test]
fn socket_negative_paths() {
    let account = sample();
    // Absent path is retryable (prepare poll semantics).
    assert!(matches!(
        socket_identity_kinded("/definitely/not/here.sock", &account, 1),
        Err(SocketCheck::Retryable(_))
    ));
    // Present non-socket is fatal.
    let dir = std::env::temp_dir().join(format!("soda-pt-svc-{}-sock", std::process::id()));
    std::fs::create_dir_all(&dir).unwrap();
    let plain = dir.join("plain");
    std::fs::write(&plain, b"x").unwrap();
    assert!(matches!(
        socket_identity_kinded(&plain.to_string_lossy(), &account, 1),
        Err(SocketCheck::Fatal(_))
    ));
    // Unlistening socket path: lstat passes only for real sockets; a
    // refused connection is retryable — exercised via a bound-then-closed
    // listener below.
    let sock_path = dir.join("t.sock");
    let listener = std::os::unix::net::UnixListener::bind(&sock_path).unwrap();
    let path = sock_path.to_string_lossy().into_owned();
    drop(listener);
    // lstat: socket owned by test user, not the fixture uid → fatal.
    assert!(matches!(
        socket_identity_kinded(&path, &account, 1),
        Err(SocketCheck::Fatal(_))
    ));
    std::fs::remove_dir_all(&dir).unwrap();
    // Public wrapper flattens kinds to String.
    assert!(socket_identity("/definitely/not/here.sock", &account, 1).is_err());
}

#[test]
fn stat_fs_detects_tmpfs() {
    // /proc/self/fd/N for a held dir reports its filesystem; the scratch
    // dir is almost surely NOT sysfs/cgroup2fs — assert self-consistency
    // with the real `stat` output instead of a fixed fstype.
    let dir = std::env::temp_dir().join(format!("soda-pt-svc-{}-fs", std::process::id()));
    std::fs::create_dir_all(&dir).unwrap();
    let file = std::fs::File::open(&dir).unwrap();
    let got = run_stat_fs(file.as_raw_fd()).unwrap();
    // Path-based reference: same directory, hence same filesystem.
    let expect = std::process::Command::new("/usr/bin/stat")
        .args(["-f", "-c", "%T", dir.to_str().unwrap()])
        .output()
        .unwrap();
    assert_eq!(got, expect.stdout);
    assert!(fstatvfs_readonly(&file).is_ok());
    std::fs::remove_dir_all(&dir).unwrap();
}

#[test]
fn supervisor_smoke() {
    // Read-only probes against certainly-absent units; outcomes vary
    // by host (systemd vs container), so exercise without asserting.
    // `stop_service` never reaches `systemctl stop` for absent units.
    let id = "e".repeat(32);
    let account = sample();
    let _result = service_state(&id, None);
    let _result = cgroup_empty(&id);
    let _result = stop_service(&id, &account);
}

#[test]
fn tmux_control_fails_closed() {
    // Spawns the real argv (as the fixture uid it cannot become) and
    // reports failure without hanging: 2s timeout honored.
    let account = sample();
    let start = sys::monotonic();
    assert!(tmux_control(
        &account,
        "/definitely/not/here.sock",
        &["list-sessions".to_string()]
    )
    .is_err());
    assert!(sys::monotonic() - start < 10.0);
}

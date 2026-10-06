use super::{LaunchRequest, MuseCaller};

// ---------- argv builders ----------

/// `museCommand` argv (after the `/usr/bin/podman` argv0): the fixed
/// systemd-run boundary plus the pinned dispatcher. Golden-pinned.
pub fn muse_command_argv(
    caller: &MuseCaller,
    request: &LaunchRequest,
    unit: &str,
    path: &str,
) -> Vec<String> {
    let home = if request.home.is_empty() {
        caller.home.clone()
    } else {
        request.home.clone()
    };
    let mut config_path = path.to_string();
    if !caller.child.is_empty() {
        let prefix = format!("/run/soda-muse/nested/{}/", caller.registration);
        config_path = format!(
            "/run/soda-muse/credentials/{}",
            path.strip_prefix(&prefix).unwrap_or(path)
        );
    }
    let mut args = vec![
        "--remote=false".to_string(),
        "exec".to_string(),
        "--interactive".to_string(),
    ];
    if request.tty {
        args.push("--tty".to_string());
    }
    args.extend([
        caller.container.clone(),
        "/usr/bin/systemd-run".to_string(),
        "--quiet".to_string(),
        "--wait".to_string(),
        "--collect".to_string(),
        "--service-type=exec".to_string(),
        format!("--unit={unit}"),
        "--property=KillMode=control-group".to_string(),
        "--property=TimeoutStopSec=10".to_string(),
        "--property=RuntimeMaxSec=43200".to_string(),
        "--property=UMask=0077".to_string(),
        format!("--setenv=HOME={home}"),
        format!("--setenv=USER={}", caller.login),
        format!("--setenv=LOGNAME={}", caller.login),
        format!("--setenv=TERM={}", request.term),
        format!("--setenv=XDG_CONFIG_HOME={path}/config"),
        format!("--setenv=XDG_STATE_HOME={path}/state"),
        format!("--setenv=XDG_CACHE_HOME={path}/cache"),
        "--setenv=TBH_CREDENTIAL_BACKEND=file".to_string(),
        "--setenv=PATH=/usr/local/bin:/usr/bin:/bin".to_string(),
    ]);
    if caller.child.is_empty() {
        args.extend([
            format!("--uid={}", caller.login),
            format!("--gid={}", caller.gid),
            format!("--working-directory={}", request.cwd),
            format!("--property=ReadOnlyPaths={path}/auth.json"),
            format!("--property=BindReadOnlyPaths={path}/auth.json:{path}/config/muse/auth.json"),
        ]);
    }
    args.push(if request.tty {
        "--pty".to_string()
    } else {
        "--pipe".to_string()
    });
    args.push("--".to_string());
    if !caller.child.is_empty() {
        args.extend([
            "/usr/bin/nsenter".to_string(),
            format!("--target={}", caller.nested_pid),
            "--mount".to_string(),
            "--pid".to_string(),
            "--uts".to_string(),
            "--ipc".to_string(),
            "--net".to_string(),
            "--root".to_string(),
            format!("--setuid={}", caller.uid),
            format!("--setgid={}", caller.gid),
            "--".to_string(),
        ]);
    }
    args.extend([
        "/usr/local/bin/muse".to_string(),
        "--soda-exec".to_string(),
        config_path,
        request.cwd.clone(),
    ]);
    args.extend(request.args.iter().cloned());
    args
}

/// Guest `systemctl show ActiveState` argv tail (after container).
pub fn unit_active_argv(unit: &str) -> Vec<String> {
    vec![
        "/usr/bin/systemctl".to_string(),
        "show".to_string(),
        "--property=ActiveState".to_string(),
        "--value".to_string(),
        unit.to_string(),
    ]
}

/// Guest `systemctl show InvocationID` argv tail.
pub fn unit_invocation_argv(unit: &str) -> Vec<String> {
    vec![
        "/usr/bin/systemctl".to_string(),
        "show".to_string(),
        "--property=InvocationID".to_string(),
        "--value".to_string(),
        unit.to_string(),
    ]
}

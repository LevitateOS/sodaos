use super::native::{reserve_run_argv, shell_quote, systemd_escape};
use super::native_io::NativeChannels;
use super::run::FactoryRun;
use super::tcodex::FactoryCodexPaths;

pub(crate) fn reserve_argv(unit: &str, max_secs: i64, channels: &NativeChannels) -> Vec<String> {
    let mut args = reserve_run_argv(unit, max_secs);
    let position = args
        .iter()
        .position(|part| part == "--")
        .expect("fixed systemd boundary");
    args.splice(
        position..position,
        [
            format!("--property=StandardInput=file:{}", channels.input_path()),
            format!("--property=StandardOutput=file:{}", channels.output_path()),
            "--property=StandardError=journal".into(),
        ],
    );
    args
}

pub(crate) fn exec_argv(
    container: &str,
    run: &FactoryRun,
    p: &FactoryCodexPaths,
    guest: &str,
    max_secs: i64,
) -> Vec<String> {
    vec![
        "--remote=false".to_string(),
        "exec".to_string(),
        "--interactive".to_string(),
        // These fixed locations were created with the admitted role's private home.
        "--env".to_string(),
        format!("TMPDIR={}/tmp", p.home),
        "--env".to_string(),
        format!("XDG_CACHE_HOME={}/cache", p.home),
        "--env".to_string(),
        "GIT_CONFIG_COUNT=1".to_string(),
        "--env".to_string(),
        "GIT_CONFIG_KEY_0=safe.directory".to_string(),
        "--env".to_string(),
        format!("GIT_CONFIG_VALUE_0={}", p.checkout),
        "--user".to_string(),
        run.role.clone(),
        "--workdir".to_string(),
        p.checkout.clone(),
        "--env".to_string(),
        format!("HOME={}", p.home),
        "--env".to_string(),
        format!("CODEX_HOME={}", p.codex),
        "--env".to_string(),
        "TERM=dumb".to_string(),
        "--env".to_string(),
        format!("GIT_AUTHOR_NAME={}", run.role),
        "--env".to_string(),
        format!("GIT_AUTHOR_EMAIL={}@localhost", run.role),
        "--env".to_string(),
        format!("GIT_COMMITTER_NAME={}", run.role),
        "--env".to_string(),
        format!("GIT_COMMITTER_EMAIL={}@localhost", run.role),
        container.to_string(),
        "/usr/bin/setsid".to_string(),
        "--wait".to_string(),
        "/usr/bin/sh".to_string(),
        "-c".to_string(),
        systemd_escape(&supervisor(p, guest, max_secs)),
    ]
}

fn supervisor(p: &FactoryCodexPaths, guest: &str, max_secs: i64) -> String {
    let mut command=format!("{} app-server --listen stdio:// -c 'cli_auth_credentials_store=\"file\"' -c 'features.multi_agent=true'",shell_quote(guest));
    if let Some(profile) = &p.native_profile {
        let path = profile["guest_path"]
            .as_str()
            .expect("validated native profile path");
        command.push_str(&format!(
            " -c {}",
            shell_quote(&format!("model_catalog_json={}", crate::json::quote(path)))
        ));
    }
    [
        format!("RUNDIR={}",shell_quote(&p.run_dir)),
        "exec 2>>\"$RUNDIR/stdout.log\"".into(),
        "STAT=$(cat /proc/$$/stat); REST=${STAT##*)}; set -- $REST; echo \"$$ ${20}\" >\"$RUNDIR/supervisor.pid\"".into(),
        "fail() { echo \"$1\" >\"$RUNDIR/exit\"; exit \"$1\"; }".into(),
        // The unit RuntimeMaxSec independently enforces this same admitted
        // lifetime while approved checks run before releasing the marker.
        format!("i=0; while [ ! -f \"$RUNDIR/marker\" ]; do [ -f \"$RUNDIR/stop\" ] && fail 44; i=$((i+1)); [ \"$i\" -gt {max_secs} ] && fail 42; sleep 1; done"),
        "mv \"$RUNDIR/marker\" \"$RUNDIR/started\" || fail 43".into(),
        command,
        "CODE=$?; echo \"$CODE\" >\"$RUNDIR/exit\"; exit \"$CODE\"".into(),
    ].join("\n")+"\n"
}

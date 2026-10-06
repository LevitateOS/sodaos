use super::paths::FactoryMusePaths;
use crate::terminal::factory::native::{shell_quote, systemd_escape};
use crate::terminal::factory::run::FactoryRun;

/// `factorySupervisor` shape for Muse runs: marker-gated fixed
/// `muse exec` entrypoint. Stdout carries exactly the final answer
/// (headless contract: diagnostics go to stderr), so the answer lands
/// in last-message.txt and diagnostics in stdout.log. The CLI reads the
/// staged `auth.json` through the file backend (same env as enrollment);
/// any inherited `META_API_KEY` is unset so no container env can smuggle
/// a key past the staged file. Exit 45 is the missing-credential refusal
/// (42/43/44 keep the codex gate meanings).
pub fn factory_muse_supervisor(p: &FactoryMusePaths, guest: &str, model: &str) -> String {
    let mut command = format!(
        "{} exec --provider meta --reasoning-effort low --workspace {} --trust-workspace --no-session-log --disable-approval --disable-sandbox",
        shell_quote(guest),
        shell_quote(&p.checkout),
    );
    if !model.is_empty() {
        command.push_str(&format!(" --model {}", shell_quote(model)));
    }
    command.push_str(&format!(
        " --prompt-file {} >{} 2>>{}",
        shell_quote(&p.prompt),
        shell_quote(&p.output),
        shell_quote(&p.stdout),
    ));
    let steps = [
        format!("RUNDIR={}", shell_quote(&p.run_dir)),
        ": >\"$RUNDIR/stdout.log\"; : >\"$RUNDIR/last-message.txt\"".to_string(),
        "exec 2>>\"$RUNDIR/stdout.log\"".to_string(),
        "STAT=$(cat /proc/$$/stat); REST=${STAT##*)}; set -- $REST; echo \"$$ ${20}\" >\"$RUNDIR/supervisor.pid\"".to_string(),
        "fail() { echo \"$1\" >\"$RUNDIR/exit\"; exit \"$1\"; }".to_string(),
        "i=0; while [ ! -f \"$RUNDIR/marker\" ]; do [ -f \"$RUNDIR/stop\" ] && fail 44; i=$((i+1)); [ \"$i\" -gt 600 ] && fail 42; sleep 1; done".to_string(),
        "mv \"$RUNDIR/marker\" \"$RUNDIR/started\" || fail 43".to_string(),
        "unset META_API_KEY".to_string(),
        format!("[ -s {} ] || fail 45", shell_quote(&p.auth)),
        command,
        "CODE=$?; echo \"$CODE\" >\"$RUNDIR/exit\"; exit \"$CODE\"".to_string(),
    ];
    steps.join("\n") + "\n"
}

/// `factoryCodexSetup` directory-preparation shape for Muse runs.
pub fn muse_setup_script(p: &FactoryMusePaths, uid: i64, gid: i64) -> String {
    format!(
        "set -u\nmkdir -p -m 700 {} {}\nchown {uid}:{gid} {} {} {}\nchmod 700 {} {} {}\n",
        shell_quote(&p.home),
        shell_quote(&p.muse_config),
        shell_quote(&p.run_dir),
        shell_quote(&p.home),
        shell_quote(&p.muse_config),
        shell_quote(&p.run_dir),
        shell_quote(&p.home),
        shell_quote(&p.muse_config),
    )
}

/// `FactoryCodexStart` staging-gate shape for Muse runs: a non-empty
/// staged `auth.json` and prompt plus the started-or-pending marker.
pub fn muse_start_gate_script(p: &FactoryMusePaths) -> String {
    format!(
        "test -s {} && test -s {} && {{ test -f {} || test -f {}; }}\n",
        shell_quote(&p.auth),
        shell_quote(&p.prompt),
        shell_quote(&p.marker),
        shell_quote(&p.started),
    )
}

/// `FactoryCodexReserve` podman-supervisor argv shape for Muse runs (the
/// unit's exec payload). No `CODEX_HOME`: the staged `auth.json` carries
/// the credential through the CLI file backend (`XDG_CONFIG_HOME` pins
/// the lookup; the launcher is bypassed so no update check can run).
pub fn muse_exec_argv(
    container: &str,
    run: &FactoryRun,
    p: &FactoryMusePaths,
    guest: &str,
) -> Vec<String> {
    vec![
        "--remote=false".to_string(),
        "exec".to_string(),
        "--user".to_string(),
        run.role.clone(),
        "--workdir".to_string(),
        p.checkout.clone(),
        "--env".to_string(),
        format!("HOME={}", p.home),
        "--env".to_string(),
        format!("XDG_CONFIG_HOME={}/.config", p.home),
        "--env".to_string(),
        "TBH_CREDENTIAL_BACKEND=file".to_string(),
        "--env".to_string(),
        "MUSE_NO_AUTO_UPDATE=1".to_string(),
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
        systemd_escape(&factory_muse_supervisor(p, guest, &run.model)),
    ]
}

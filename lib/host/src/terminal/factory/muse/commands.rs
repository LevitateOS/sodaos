use super::paths::FactoryMusePaths;
use crate::terminal::factory::native::{shell_quote, systemd_escape};
use crate::terminal::factory::run::FactoryRun;

pub(crate) fn muse_controlled_argv(
    container: &str,
    run: &FactoryRun,
    p: &FactoryMusePaths,
    guest: &str,
    max_secs: i64,
) -> Vec<String> {
    let instructions = "Follow the admitted Factory phase and immutable manifest. Never modify Git metadata, acquire credentials or start independent CLI agents. For implementation, select a ready task with factory.start_task({payload:{task_id:...}}) before spawning. Spawn exactly one native child with the returned prompt verbatim as objective and subagent_type factory-task, whose admitted native definition fixes the returned model and effort; no worktree or nested children. Never use general-purpose, resume, reopen, follow up or replace a task child. Wait for each native result, read Factory context and submit_task_output with the host-observed exact output digest and native_child_ref; wait for its committed checkpoint before dependent or same-lane work. The parent must not edit task source. For correction, do the singleton task yourself, then call factory.release_output({payload:{task_id:...}}), never write source afterward, and submit the returned exact output digest with empty native_child_ref. Stay alive until all task checkpoints are committed. Native child result text is not verification evidence.";
    let configured = super::super::mcp_relay::instruction_argv(
        super::super::mcp_relay::PerRunInstructionRoute::MuseDeveloperPrompt,
        instructions,
    )
    .expect("fixed Muse instructions are valid");
    let command = format!(
        "{} serve --provider meta --model {} --contributor-mode {} --trust-workspace --disable-sandbox --disable-request-user-input {} {}",
        shell_quote(guest),
        shell_quote(&run.model),
        if run.contributor { "on" } else { "off" },
        shell_quote(&configured[0]),
        shell_quote(&configured[1]),
    );
    let script = [
        format!("RUNDIR={}", shell_quote(&p.run_dir)),
        ": >\"$RUNDIR/last-message.txt\"; exec 2>>\"$RUNDIR/stdout.log\"".into(),
        "STAT=$(cat /proc/$$/stat); REST=${STAT##*)}; set -- $REST; echo \"$$ ${20}\" >\"$RUNDIR/supervisor.pid\"".into(),
        "fail() { echo \"$1\" >\"$RUNDIR/exit\"; exit \"$1\"; }".into(),
        format!("i=0; while [ ! -f \"$RUNDIR/marker\" ]; do [ -f \"$RUNDIR/stop\" ] && fail 44; i=$((i+1)); [ \"$i\" -gt {max_secs} ] && fail 42; sleep 1; done"),
        "mv \"$RUNDIR/marker\" \"$RUNDIR/started\" || fail 43".into(),
        "unset META_API_KEY".into(),
        format!("[ -s {} ] || fail 45", shell_quote(&p.auth)),
        command,
        "CODE=$?; echo \"$CODE\" >\"$RUNDIR/exit\"; exit \"$CODE\"".into(),
    ].join("\n")+"\n";
    // No `CODEX_HOME`: the staged `auth.json` carries the credential
    // through the CLI file backend (`XDG_CONFIG_HOME` pins the lookup; the
    // launcher is bypassed so no update check can run).
    vec![
        "--remote=false".to_string(),
        "exec".to_string(),
        "--interactive".to_string(),
        "--env".to_string(),
        format!("TMPDIR={}/tmp", p.home),
        "--env".to_string(),
        format!("XDG_CACHE_HOME={}/cache", p.home),
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
        "GIT_CONFIG_COUNT=1".to_string(),
        "--env".to_string(),
        "GIT_CONFIG_KEY_0=safe.directory".to_string(),
        "--env".to_string(),
        format!("GIT_CONFIG_VALUE_0={}", p.checkout),
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
        systemd_escape(&script),
    ]
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

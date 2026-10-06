use super::{muse_command_argv, muse_host_environment, MuseExecution};

/// Spawn the prepared execution with owned stdio files.
pub(in crate::muse) fn spawn_execution(
    execution: &MuseExecution,
    stdio: [std::fs::File; 3],
) -> Result<std::process::Child, String> {
    let [stdin, stdout, stderr] = stdio;
    let argv = muse_command_argv(
        &execution.caller,
        &execution.request,
        &execution.unit,
        &execution.path,
    );
    std::process::Command::new("/usr/bin/podman")
        .args(&argv)
        .env_clear()
        .envs(muse_host_environment())
        .stdin(std::process::Stdio::from(stdin))
        .stdout(std::process::Stdio::from(stdout))
        .stderr(std::process::Stdio::from(stderr))
        .spawn()
        .map_err(|e| format!("/usr/bin/podman failed: {e}"))
}

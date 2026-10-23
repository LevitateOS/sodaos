use std::collections::HashMap;
use std::time::Instant;

use crate::project::Executor;
use crate::terminal::factory::native::shell_quote;

use super::git::GitContext;
use super::state::{operation_key, CheckpointOperation};
use super::wire::{CheckpointCheck, CheckpointCheckResult};

fn unit(operation: &CheckpointOperation, check: &CheckpointCheck) -> String {
    format!(
        "soda-factory-check-{}-{}.service",
        &operation_key(&operation.request, operation.final_candidate)[..24],
        check.id
    )
}

fn description(operation: &CheckpointOperation, check: &CheckpointCheck) -> String {
    format!(
        "soda Factory checkpoint:{}:{}:{}",
        operation_key(&operation.request, operation.final_candidate),
        check.id,
        operation.request.invocation
    )
}

pub(super) fn start<E: Executor>(
    exec: &E,
    context: &GitContext,
    operation: &CheckpointOperation,
    check: &CheckpointCheck,
    lifetime_seconds: u64,
    deadline: Instant,
) -> Result<(), String> {
    let cwd = if check.cwd == "." {
        format!("{}/export", context.work)
    } else {
        format!(
            "{}/export/{}",
            context.work,
            check.cwd.trim_end_matches('/')
        )
    };
    let mut args = vec![
        "/usr/bin/systemd-run".to_string(),
        "--no-block".into(),
        format!("--unit={}", unit(operation, check)),
        format!("--property=Description={}", description(operation, check)),
        format!("--property=User={}", context.uid),
        format!("--property=Group={}", context.gid),
        format!("--property=WorkingDirectory={cwd}"),
        "--property=KillMode=control-group".into(),
        "--property=TimeoutStopSec=5".into(),
        "--property=RemainAfterExit=yes".into(),
        "--property=StandardOutput=journal".into(),
        "--property=StandardError=journal".into(),
        format!(
            "--property=RuntimeMaxSec={}",
            lifetime_seconds.min(check.timeout_seconds).max(1)
        ),
        "--".into(),
        "/usr/bin/env".into(),
        "-i".into(),
        "PATH=/usr/local/bin:/usr/bin:/bin".into(),
        format!("HOME={}", context.home),
        format!("TMPDIR={}/export", context.work),
        "LC_ALL=C".into(),
        "GIT_CONFIG_NOSYSTEM=1".into(),
        "GIT_CONFIG_GLOBAL=/dev/null".into(),
        "GIT_NO_REPLACE_OBJECTS=1".into(),
        "GIT_TERMINAL_PROMPT=0".into(),
    ];
    args.extend(check.argv.iter().cloned());
    let command = args
        .iter()
        .map(|arg| shell_quote(arg))
        .collect::<Vec<_>>()
        .join(" ");
    let command = format!(
        "[ \"$(/usr/bin/realpath -e {})\" = {} ]\n[ -d {} ]\n{}",
        shell_quote(&cwd),
        shell_quote(&cwd),
        shell_quote(&cwd),
        command
    );
    context.exec(exec, &command, deadline).map(|_| ())
}

fn properties<E: Executor>(
    exec: &E,
    context: &GitContext,
    unit: &str,
    deadline: Instant,
) -> Result<HashMap<String, String>, String> {
    let bytes = context.exec(exec, &format!("/usr/bin/systemctl show --property=LoadState,Description,InvocationID,ActiveState,SubState,Result,ExecMainCode,ExecMainStatus,MainPID,ControlPID {}", shell_quote(unit)), deadline)?;
    if bytes.len() > 16 * 1024 {
        return Err("checkpoint process observation exceeds its bound".into());
    }
    let text = std::str::from_utf8(&bytes).map_err(|_| "invalid checkpoint process observation")?;
    Ok(text
        .lines()
        .filter_map(|line| {
            line.split_once('=')
                .map(|(k, v)| (k.to_string(), v.to_string()))
        })
        .collect())
}

pub(super) enum Observation {
    Pending(String),
    Complete(String, CheckpointCheckResult),
}

pub(super) fn observe<E: Executor>(
    exec: &E,
    context: &GitContext,
    operation: &CheckpointOperation,
    check: &CheckpointCheck,
    deadline: Instant,
) -> Result<Observation, String> {
    let name = unit(operation, check);
    let props = properties(exec, context, &name, deadline)?;
    if props.get("LoadState").map(String::as_str) != Some("loaded")
        || props.get("Description") != Some(&description(operation, check))
    {
        return Err("checkpoint process outcome is unknown".into());
    }
    let invocation = props.get("InvocationID").cloned().unwrap_or_default();
    if invocation.len() != 32
        || !invocation.bytes().all(|b| b.is_ascii_hexdigit())
        || !operation.check_invocation.is_empty() && operation.check_invocation != invocation
    {
        return Err("checkpoint process invocation changed".into());
    }
    let active = props.get("ActiveState").map(String::as_str).unwrap_or("");
    let substate = props.get("SubState").map(String::as_str).unwrap_or("");
    if matches!(active, "activating" | "reloading" | "deactivating")
        || active == "active" && substate != "exited"
    {
        return Ok(Observation::Pending(invocation));
    }
    let passed = props.get("Result").map(String::as_str) == Some("success")
        && props.get("ExecMainCode").map(String::as_str) == Some("1")
        && props.get("ExecMainStatus").map(String::as_str) == Some("0");
    // Retire every descendant before checking the export. A successful leader
    // may have forked helpers; their continued execution is not completion.
    stop(exec, context, operation, check, deadline)?;
    let logs = context.exec(
        exec,
        &format!(
            "/usr/bin/journalctl --no-pager --output=cat --lines=128 {}",
            shell_quote(&format!("_SYSTEMD_INVOCATION_ID={invocation}"))
        ),
        deadline,
    )?;
    let digest = crate::sha256::hex_lower(&crate::sha256::digest(&logs));
    Ok(Observation::Complete(
        invocation,
        CheckpointCheckResult {
            id: check.id.clone(),
            passed,
            log_digest: digest,
        },
    ))
}

pub(super) fn stop<E: Executor>(
    exec: &E,
    context: &GitContext,
    operation: &CheckpointOperation,
    check: &CheckpointCheck,
    deadline: Instant,
) -> Result<(), String> {
    let name = unit(operation, check);
    let props = properties(exec, context, &name, deadline)?;
    if props.get("LoadState").map(String::as_str) == Some("not-found") {
        return if operation.check_invocation.is_empty() {
            Ok(())
        } else {
            Err("checkpoint process disappeared before retirement".into())
        };
    }
    if props.get("Description") != Some(&description(operation, check))
        || !operation.check_invocation.is_empty()
            && props.get("InvocationID") != Some(&operation.check_invocation)
    {
        return Err("checkpoint process retirement binding changed".into());
    }
    context.exec(
        exec,
        &format!("/usr/bin/systemctl stop {}", shell_quote(&name)),
        deadline,
    )?;
    let stopped = properties(exec, context, &name, deadline)?;
    if !matches!(
        stopped.get("ActiveState").map(String::as_str),
        Some("inactive" | "failed")
    ) || stopped.get("MainPID").map(String::as_str) != Some("0")
        || stopped.get("ControlPID").map(String::as_str) != Some("0")
    {
        return Err("checkpoint process retirement is unconfirmed".into());
    }
    Ok(())
}

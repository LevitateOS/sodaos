use crate::terminal;

// ---------- Muse argument policy ----------

fn muse_auth_override(arg: &str) -> bool {
    matches!(
        arg.split('=').next(),
        Some("--provider") | Some("--base-url")
    )
}

fn muse_value_flag(flag: &str) -> bool {
    matches!(
        flag,
        "--model"
            | "--reasoning-effort"
            | "--agents"
            | "--preset"
            | "--image"
            | "--workspace"
            | "--worktree-base"
            | "--worktree-existing"
            | "--approval-mode"
            | "--permission-profile"
            | "--approval-judge"
            | "--sandbox-network"
            | "--echo-delay-ms"
    )
}

fn muse_positional(args: &[String]) -> &str {
    let mut skip = false;
    for arg in args {
        if skip {
            skip = false;
            continue;
        }
        if arg.starts_with('-') {
            skip = muse_value_flag(arg);
            continue;
        }
        return arg;
    }
    ""
}

fn muse_provider_arguments(args: &[String]) -> Vec<String> {
    if args.is_empty() {
        return vec!["--provider".to_string(), "meta".to_string()];
    }
    match args[0].as_str() {
        "exec" | "resume" | "serve" => {
            let mut out = vec![
                args[0].clone(),
                "--provider".to_string(),
                "meta".to_string(),
            ];
            out.extend(args[1..].iter().cloned());
            out
        }
        "config" | "export" | "trace" | "skills" | "sandbox" | "schema" | "session-message"
        | "mcp" | "init" => args.to_vec(),
        _ => {
            let mut out = vec!["--provider".to_string(), "meta".to_string()];
            out.extend(args.iter().cloned());
            out
        }
    }
}

/// `identity.MuseArguments`: ordinary invocation bytes under the
/// subscription provider; auth subcommands and overrides denied.
pub fn muse_arguments(args: &[String]) -> Result<Vec<String>, String> {
    for arg in args {
        if muse_auth_override(arg) {
            return Err(terminal::err_denied());
        }
    }
    match muse_positional(args) {
        "auth" | "login" | "logout" => Err(terminal::err_denied()),
        _ => Ok(muse_provider_arguments(args)),
    }
}

use std::time::Duration;

use crate::error::Error;
use crate::report;

pub(super) const HELP_REQUESTED: &str = "support command help requested";
use clap::{Arg, ArgAction, Command};

/// Parse the admitted positive humantime grammar into a bounded duration.
pub(super) fn parse_duration(text: &str) -> Result<Duration, ()> {
    let unsigned = text.strip_prefix('+').unwrap_or(text);
    let normalized = unsigned.replace('μ', "µ");
    let duration = humantime::parse_duration(&normalized).map_err(|_| ())?;
    if duration.is_zero() {
        return Err(());
    }
    Ok(duration)
}

pub(super) struct RunOptions {
    pub(super) action: String,
    pub(super) owner: String,
    pub(super) revision: String,
    pub(super) arch: String,
    pub(super) target: String,
    pub(super) evidence: String,
    pub(super) remote_file: String,
    pub(super) request: String,
    pub(super) config: String,
    pub(super) hold: bool,
    pub(super) restart: bool,
    pub(super) timeout: Duration,
    pub(super) secret_files: Vec<String>,
    pub(super) artifact_files: Vec<String>,
    pub(super) cmd_args: Vec<String>,
}

/// `^[A-Za-z0-9][A-Za-z0-9_.:-]{0,127}$` non-secret targets.
fn valid_target(target: &str) -> bool {
    let mut chars = target.chars();
    match chars.next() {
        Some(c) if c.is_ascii_alphanumeric() => {}
        _ => return false,
    }
    let rest = chars.as_str();
    rest.len() <= 127
        && rest
            .chars()
            .all(|c| c.is_ascii_alphanumeric() || matches!(c, '_' | '.' | ':' | '-'))
}

fn validate_common_options(opts: &RunOptions) -> Result<(), Error> {
    if !report::revision(&opts.revision)
        || !valid_target(&opts.target)
        || opts.timeout.is_zero()
        || opts.timeout > Duration::from_secs(24 * 3600)
    {
        return Err(Error::msg(
            "revision, non-secret target and bounded timeout required",
        ));
    }
    report::oci_architecture(&opts.arch)?;
    if !report::valid_owner(&opts.owner) {
        return Err(Error::msg(
            "explicit owner required; P07/P08 are not independent tasks",
        ));
    }
    Ok(())
}

fn validate_action_options(opts: &RunOptions) -> Result<(), Error> {
    if opts.action == "exec" && opts.cmd_args.is_empty() {
        return Err(Error::msg("an existing owned check command is required"));
    }
    if opts.action != "exec" && !opts.cmd_args.is_empty() {
        return Err(Error::msg("unexpected check command"));
    }
    if opts.action != "vm" && (opts.hold || opts.restart) {
        return Err(Error::msg(
            "hold/restart are explicit fresh-VM actions only",
        ));
    }
    match opts.action.as_str() {
        "native" => {
            if opts.owner != "P02" || opts.request.is_empty() || opts.remote_file.is_empty() {
                return Err(Error::msg("native requires P02, remote and request"));
            }
        }
        "probe-ssh" => {
            if opts.owner != "P11" || opts.remote_file.is_empty() {
                return Err(Error::msg(
                    "probe-ssh requires P11 and a pinned Git endpoint",
                ));
            }
        }
        "vm" if (opts.owner != "P03" || opts.config.is_empty() || !opts.remote_file.is_empty()) => {
            return Err(Error::msg("vm requires P03 and config"));
        }
        _ => {}
    }
    Ok(())
}

fn value(name: &'static str) -> Arg {
    Arg::new(name)
        .long(name)
        .action(ArgAction::Set)
        .num_args(1)
        .allow_hyphen_values(true)
}

fn boolean(name: &'static str) -> Arg {
    Arg::new(name)
        .long(name)
        .action(ArgAction::Set)
        .num_args(0..=1)
        .require_equals(true)
        .default_missing_value("true")
        .value_parser([
            "true", "false", "1", "0", "t", "T", "TRUE", "True", "f", "F", "FALSE", "False",
        ])
}

fn run_command(action: &str) -> Command {
    let command = Command::new("soda-acceptance")
        .args_override_self(true)
        .arg(
            Arg::new("action")
                .required(true)
                .value_parser(["exec", "native", "vm", "probe-ssh"]),
        )
        .arg(value("owner"))
        .arg(value("revision"))
        .arg(value("arch"))
        .arg(value("target"))
        .arg(value("evidence"))
        .arg(value("remote"))
        .arg(value("request"))
        .arg(value("config"))
        .arg(boolean("hold"))
        .arg(boolean("restart"))
        .arg(value("timeout"))
        .arg(
            Arg::new("secret-file")
                .long("secret-file")
                .action(ArgAction::Append)
                .num_args(1)
                .allow_hyphen_values(true),
        )
        .arg(
            Arg::new("artifact-file")
                .long("artifact-file")
                .action(ArgAction::Append)
                .num_args(1)
                .allow_hyphen_values(true),
        );
    if action == "exec" {
        command
            .trailing_var_arg(true)
            .arg(Arg::new("cmd").num_args(1..))
    } else {
        command
    }
}

pub(super) fn help_text(action: &str) -> String {
    run_command(action).render_help().to_string()
}

pub(super) fn parse_run_options(args: &[String]) -> Result<RunOptions, Error> {
    let action = args
        .first()
        .cloned()
        .ok_or_else(|| Error::msg("support action required"))?;
    match action.as_str() {
        "exec" | "native" | "vm" | "probe-ssh" => {}
        _ => {
            return Err(Error::msg(
                "no product/media/release workflow is implemented by this support tool",
            ))
        }
    }
    let matches = run_command(&action)
        .try_get_matches_from(
            std::iter::once("soda-acceptance").chain(args.iter().map(String::as_str)),
        )
        .map_err(|error| {
            if error.kind() == clap::error::ErrorKind::DisplayHelp {
                Error::msg(HELP_REQUESTED)
            } else {
                Error::msg("invalid support command flags")
            }
        })?;
    let get = |name: &str| matches.get_one::<String>(name).cloned().unwrap_or_default();
    let timeout = matches
        .get_one::<String>("timeout")
        .map(|s| parse_duration(s).map_err(|_| Error::msg("invalid support command flags")))
        .transpose()?
        .unwrap_or(Duration::from_secs(30 * 60));
    let cmd_args = if action == "exec" {
        matches
            .get_many::<String>("cmd")
            .map(|values| values.cloned().collect())
            .unwrap_or_default()
    } else {
        Vec::new()
    };
    let opts = RunOptions {
        action,
        owner: get("owner"),
        revision: get("revision"),
        arch: get("arch"),
        target: get("target"),
        evidence: get("evidence"),
        remote_file: get("remote"),
        request: get("request"),
        config: get("config"),
        hold: matches
            .get_one::<String>("hold")
            .is_some_and(|v| matches!(v.as_str(), "true" | "1" | "t" | "T" | "TRUE" | "True")),
        restart: matches
            .get_one::<String>("restart")
            .is_some_and(|v| matches!(v.as_str(), "true" | "1" | "t" | "T" | "TRUE" | "True")),
        timeout,
        secret_files: matches
            .get_many::<String>("secret-file")
            .map(|v| v.cloned().collect())
            .unwrap_or_default(),
        artifact_files: matches
            .get_many::<String>("artifact-file")
            .map(|v| v.cloned().collect())
            .unwrap_or_default(),
        cmd_args,
    };
    validate_common_options(&opts)?;
    validate_action_options(&opts)?;
    Ok(opts)
}

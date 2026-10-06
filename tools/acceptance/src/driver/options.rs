use std::collections::HashMap;
use std::time::Duration;

use crate::error::Error;
use crate::report;

/// Parse a Go `time.ParseDuration` string into nanoseconds.
pub(super) fn parse_duration(text: &str) -> Result<i128, ()> {
    if text.is_empty() || text == "+" || text == "-" {
        return Err(());
    }
    if text == "0" {
        return Ok(0);
    }
    let (negative, mut rest) = match text.strip_prefix('-') {
        Some(rest) => (true, rest),
        None => (false, text.strip_prefix('+').unwrap_or(text)),
    };
    if rest.is_empty() {
        return Err(());
    }
    let mut total: i128 = 0;
    while !rest.is_empty() {
        let mut value: i128 = 0;
        while let Some(head) = rest.chars().next() {
            if !head.is_ascii_digit() {
                break;
            }
            value = value * 10 + (head as i128 - '0' as i128);
            rest = &rest[head.len_utf8()..];
        }
        if value > i64::MAX as i128 {
            return Err(());
        }
        let mut fraction: i128 = 0;
        let mut scale: i128 = 1;
        if let Some(after_dot) = rest.strip_prefix('.') {
            rest = after_dot;
            while let Some(head) = rest.chars().next() {
                if !head.is_ascii_digit() {
                    break;
                }
                fraction = fraction * 10 + (head as i128 - '0' as i128);
                scale *= 10;
                rest = &rest[head.len_utf8()..];
            }
        }
        let (multiplier, after_unit) = consume_unit(rest)?;
        rest = after_unit;
        total += value * multiplier + fraction * multiplier / scale;
    }
    if total > i64::MAX as i128 {
        return Err(());
    }
    Ok(if negative { -total } else { total })
}

/// Consume one duration unit suffix, longest match first.
fn consume_unit(text: &str) -> Result<(i128, &str), ()> {
    for unit in ["ns", "us", "µs", "μs", "ms"] {
        if let Some(rest) = text.strip_prefix(unit) {
            let multiplier = match unit {
                "ns" => 1,
                "us" | "µs" | "μs" => 1_000,
                _ => 1_000_000,
            };
            return Ok((multiplier, rest));
        }
    }
    if let Some(head) = text.chars().next() {
        let multiplier = match head {
            's' => 1_000_000_000,
            'm' => 60_000_000_000,
            'h' => 3_600_000_000_000,
            _ => return Err(()),
        };
        return Ok((multiplier, &text[head.len_utf8()..]));
    }
    Err(())
}

#[derive(Clone, Copy, PartialEq, Eq)]
pub(super) enum FlagKind {
    Str,
    Bool,
    Duration,
    Repeat,
}

pub(super) struct ParsedFlags {
    strings: HashMap<String, String>,
    pub(super) bools: HashMap<String, bool>,
    durations: HashMap<String, i128>,
    pub(super) repeats: HashMap<String, Vec<String>>,
    pub(super) positionals: Vec<String>,
}

/// Parse one Go `flag.FlagSet` (ContinueOnError, discarded output):
/// single/double dashes, `-flag value` or `-flag=value`, bools taking
/// no separate value, parsing stops at the first positional or `--`.
pub(super) fn parse_flags(args: &[String], specs: &[(&str, FlagKind)]) -> Result<ParsedFlags, ()> {
    let kinds: HashMap<&str, FlagKind> = specs.iter().copied().collect();
    let mut parsed = ParsedFlags {
        strings: HashMap::new(),
        bools: HashMap::new(),
        durations: HashMap::new(),
        repeats: HashMap::new(),
        positionals: Vec::new(),
    };
    let mut index = 0;
    while index < args.len() {
        let arg = &args[index];
        if arg.len() < 2 || !arg.starts_with('-') || arg == "-" {
            break;
        }
        if arg == "--" {
            index += 1;
            break;
        }
        let flag = if let Some(stripped) = arg.strip_prefix("--") {
            stripped
        } else {
            &arg[1..]
        };
        let (name, inline) = match flag.split_once('=') {
            Some((name, value)) => (name, Some(value)),
            None => (flag, None),
        };
        let kind = kinds.get(name).copied().ok_or(())?;
        if kind == FlagKind::Bool {
            let value = match inline {
                Some(text) => parse_bool(text)?,
                None => true,
            };
            parsed.bools.insert(name.to_string(), value);
            index += 1;
            continue;
        }
        let value = match inline {
            Some(text) => {
                index += 1;
                text.to_string()
            }
            None => {
                index += 1;
                args.get(index).cloned().ok_or(())?;
                index += 1;
                args[index - 1].clone()
            }
        };
        match kind {
            FlagKind::Str => {
                parsed.strings.insert(name.to_string(), value);
            }
            FlagKind::Duration => {
                parsed
                    .durations
                    .insert(name.to_string(), parse_duration(&value)?);
            }
            FlagKind::Repeat => {
                parsed
                    .repeats
                    .entry(name.to_string())
                    .or_default()
                    .push(value);
            }
            FlagKind::Bool => unreachable!(),
        }
    }
    parsed.positionals = args[index..].to_vec();
    Ok(parsed)
}

/// Go `strconv.ParseBool` vocabulary.
fn parse_bool(text: &str) -> Result<bool, ()> {
    match text {
        "1" | "t" | "T" | "TRUE" | "true" | "True" => Ok(true),
        "0" | "f" | "F" | "FALSE" | "false" | "False" => Ok(false),
        _ => Err(()),
    }
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

pub(super) fn flag_string(parsed: &ParsedFlags, name: &str) -> String {
    parsed.strings.get(name).cloned().unwrap_or_default()
}

pub(super) fn parse_run_options(args: &[String]) -> Result<RunOptions, Error> {
    let action = args[0].clone();
    match action.as_str() {
        "exec" | "native" | "vm" | "probe-ssh" => {}
        _ => {
            return Err(Error::msg(
                "no product/media/release workflow is implemented by this support tool",
            ))
        }
    }
    let parsed = parse_flags(
        &args[1..],
        &[
            ("owner", FlagKind::Str),
            ("revision", FlagKind::Str),
            ("arch", FlagKind::Str),
            ("target", FlagKind::Str),
            ("evidence", FlagKind::Str),
            ("remote", FlagKind::Str),
            ("request", FlagKind::Str),
            ("config", FlagKind::Str),
            ("hold", FlagKind::Bool),
            ("restart", FlagKind::Bool),
            ("timeout", FlagKind::Duration),
            ("secret-file", FlagKind::Repeat),
            ("artifact-file", FlagKind::Repeat),
        ],
    )
    .map_err(|_| Error::msg("invalid support command flags"))?;
    let timeout_nanos = parsed
        .durations
        .get("timeout")
        .copied()
        .unwrap_or(30 * 60 * 1_000_000_000);
    let opts = RunOptions {
        action,
        owner: flag_string(&parsed, "owner"),
        revision: flag_string(&parsed, "revision"),
        arch: flag_string(&parsed, "arch"),
        target: flag_string(&parsed, "target"),
        evidence: flag_string(&parsed, "evidence"),
        remote_file: flag_string(&parsed, "remote"),
        request: flag_string(&parsed, "request"),
        config: flag_string(&parsed, "config"),
        hold: parsed.bools.get("hold").copied().unwrap_or(false),
        restart: parsed.bools.get("restart").copied().unwrap_or(false),
        timeout: if timeout_nanos <= 0 {
            Duration::ZERO
        } else {
            Duration::from_nanos(timeout_nanos as u64)
        },
        secret_files: parsed
            .repeats
            .get("secret-file")
            .cloned()
            .unwrap_or_default(),
        artifact_files: parsed
            .repeats
            .get("artifact-file")
            .cloned()
            .unwrap_or_default(),
        cmd_args: parsed.positionals,
    };
    validate_common_options(&opts)?;
    validate_action_options(&opts)?;
    Ok(opts)
}

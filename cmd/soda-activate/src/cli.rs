#[derive(Debug, PartialEq)]
pub(crate) struct CliArgs {
    pub(crate) bind_ip: String,
    pub(crate) certificate: Option<String>,
    pub(crate) private_key: Option<String>,
    pub(crate) local_tls: bool,
}

/// Split a `--name[=value]` token. Bare `-`, non-dashes, `-h`,
/// single-dash flags and empty names are the caller's positionals.
fn long_option(arg: &str) -> Option<(&str, Option<&str>)> {
    if arg == "-" || !arg.starts_with('-') || arg == "-h" || !arg.starts_with("--") {
        return None;
    }
    let body = &arg[2..];
    let (name, inline) = match body.split_once('=') {
        Some((n, v)) => (n, Some(v)),
        None => (body, None),
    };
    if name.is_empty() {
        return None;
    }
    Some((name, inline))
}

/// Resolve an unambiguous long-option prefix. `None` means unrecognized
/// (the caller's positional); ambiguity errors argparse-style.
fn resolve_option(long_opts: &[&str], name: &str, arg: &str) -> Option<Result<String, String>> {
    let mut matches: Vec<&&str> = long_opts.iter().filter(|o| o.starts_with(name)).collect();
    if long_opts.contains(&name) {
        matches = long_opts.iter().filter(|o| **o == name).collect();
    }
    if matches.is_empty() {
        // argparse reports the whole token for unrecognized long options.
        return None;
    }
    if matches.len() > 1 {
        let options: Vec<String> = matches.iter().map(|o| format!("--{o}")).collect();
        return Some(Err(format!(
            "argument --{name}: ambiguous option: {arg} could match {}",
            options.join(", ")
        )));
    }
    Some(Ok(matches[0].to_string()))
}

/// Long flags take no inline value: `--opt=v` errors argparse-style.
fn reject_inline_value(display: &str, inline: Option<&str>) -> Result<(), String> {
    if let Some(v) = inline {
        return Err(format!(
            "argument {display}: ignored explicit argument '{v}'"
        ));
    }
    Ok(())
}

/// Take a string option value inline or from the next argument.
fn take_option_value(
    args: &[String],
    i: &mut usize,
    opt: &str,
    inline: Option<&str>,
) -> Result<String, String> {
    match inline {
        Some(v) => Ok(v.to_string()),
        None => {
            if *i + 1 >= args.len() {
                return Err(format!("argument --{opt}: expected one argument"));
            }
            *i += 1;
            Ok(args[*i].clone())
        }
    }
}

/// Reject trailing positionals, then require `--bind-ip`.
fn finish_args(
    bind_ip: Option<String>,
    certificate: Option<String>,
    private_key: Option<String>,
    local_tls: bool,
    positionals: Vec<String>,
) -> Result<Option<CliArgs>, String> {
    if !positionals.is_empty() {
        return Err(format!("unrecognized arguments: {}", positionals.join(" ")));
    }
    match bind_ip {
        Some(bind_ip) => Ok(Some(CliArgs {
            bind_ip,
            certificate,
            private_key,
            local_tls,
        })),
        None => Err("the following arguments are required: --bind-ip".to_string()),
    }
}

/// Parse argv argparse-style: exact long options, unambiguous prefixes,
/// `--opt=value`, `-h/--help`. Errors carry argparse's messages.
pub(crate) fn parse_args(prog: &str, args: &[String]) -> Result<Option<CliArgs>, String> {
    let mut bind_ip: Option<String> = None;
    let mut certificate: Option<String> = None;
    let mut private_key: Option<String> = None;
    let mut local_tls = false;
    let mut i = 0;
    let mut positionals: Vec<String> = Vec::new();
    let long_opts = ["bind-ip", "certificate", "private-key", "local-tls", "help"];
    while i < args.len() {
        let arg = &args[i];
        if arg == "--" {
            positionals.extend(args[i + 1..].iter().cloned());
            break;
        }
        if arg == "-h" {
            return Ok(None);
        }
        let Some((name, inline)) = long_option(arg) else {
            positionals.push(arg.clone());
            i += 1;
            continue;
        };
        let Some(resolved) = resolve_option(&long_opts, name, arg) else {
            positionals.push(arg.clone());
            i += 1;
            continue;
        };
        let opt = resolved?;
        if opt == "help" {
            reject_inline_value("-h/--help", inline)?;
            let _ = prog;
            return Ok(None);
        }
        if opt == "local-tls" {
            reject_inline_value("--local-tls", inline)?;
            local_tls = true;
            i += 1;
            continue;
        }
        let value = take_option_value(args, &mut i, &opt, inline)?;
        match opt.as_str() {
            "bind-ip" => bind_ip = Some(value),
            "certificate" => certificate = Some(value),
            "private-key" => private_key = Some(value),
            _ => {}
        }
        i += 1;
    }
    finish_args(bind_ip, certificate, private_key, local_tls, positionals)
}

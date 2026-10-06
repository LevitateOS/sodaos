#[derive(Debug, PartialEq)]
pub(crate) struct CliArgs {
    pub(crate) bind_ip: String,
    pub(crate) certificate: Option<String>,
    pub(crate) private_key: Option<String>,
    pub(crate) local_tls: bool,
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
        if arg == "-" || !arg.starts_with('-') {
            positionals.push(arg.clone());
            i += 1;
            continue;
        }
        if arg == "-h" {
            return Ok(None);
        }
        if !arg.starts_with("--") {
            positionals.push(arg.clone());
            i += 1;
            continue;
        }
        let body = &arg[2..];
        let (name, inline) = match body.split_once('=') {
            Some((n, v)) => (n, Some(v)),
            None => (body, None),
        };
        if name.is_empty() {
            positionals.push(arg.clone());
            i += 1;
            continue;
        }
        let mut matches: Vec<&&str> = long_opts.iter().filter(|o| o.starts_with(name)).collect();
        if long_opts.contains(&name) {
            matches = long_opts.iter().filter(|o| **o == name).collect();
        }
        if matches.is_empty() {
            // argparse reports the whole token for unrecognized long options.
            positionals.push(arg.clone());
            i += 1;
            continue;
        }
        if matches.len() > 1 {
            let options: Vec<String> = matches.iter().map(|o| format!("--{o}")).collect();
            return Err(format!(
                "argument --{name}: ambiguous option: {arg} could match {}",
                options.join(", ")
            ));
        }
        let opt = matches[0].to_string();
        if opt == "help" {
            if let Some(v) = inline {
                return Err(format!("argument -h/--help: ignored explicit argument '{v}'"));
            }
            let _ = prog;
            return Ok(None);
        }
        if opt == "local-tls" {
            if let Some(v) = inline {
                return Err(format!(
                    "argument --local-tls: ignored explicit argument '{v}'"
                ));
            }
            local_tls = true;
            i += 1;
            continue;
        }
        let value = match inline {
            Some(v) => v.to_string(),
            None => {
                if i + 1 >= args.len() {
                    return Err(format!("argument --{opt}: expected one argument"));
                }
                i += 1;
                args[i].clone()
            }
        };
        match opt.as_str() {
            "bind-ip" => bind_ip = Some(value),
            "certificate" => certificate = Some(value),
            "private-key" => private_key = Some(value),
            _ => {}
        }
        i += 1;
    }
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

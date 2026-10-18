const DESCRIPTION: &str = "Whole-domain stop and restart-inhibition controls for native recovery.\n\nHost-operator controls for the offline native-mutation recovery procedure: stop every native writer, inhibit restart, reconcile via Forgejo's own `admin native-operation` commands, lift inhibition, restart. There is no force-unlock verb: a fenced reservation stays held for intervention.\n\nVerbs: stop, inhibit, status, lift, start. Run as root.";

pub(crate) fn usage(prog: &str) -> String {
    format!("usage: {prog} [-h] {{stop,inhibit,status,lift,start}}")
}

pub(crate) fn help_text(prog: &str) -> String {
    format!(
        "{usage}\n\n{DESCRIPTION}\n\npositional arguments:\n  {{stop,inhibit,status,lift,start}}\n                        stop writers and verify; mask unit and create the offline marker; bounded host-side readout; remove marker and unmask; refuse while inhibited, else start the unit\n\noptions:\n  -h, --help            show this help message and exit\n",
        usage = usage(prog),
    )
}

fn probe_help_prefix(arg: &str) -> Option<Result<Option<String>, String>> {
    if arg.starts_with("--help=") {
        let explicit = arg["--help=".len()..].to_string();
        return Some(Err(format!(
            "argument -h/--help: ignored explicit argument '{explicit}'"
        )));
    }
    if arg.starts_with("--") && !arg.contains('=') {
        // Unambiguous long-option prefixes (help is the only one).
        let body = &arg[2..];
        if !body.is_empty() && "help".starts_with(body) {
            return Some(Ok(None));
        }
    }
    None
}

fn check_extras(extras: &[String]) -> Result<(), String> {
    if !extras.is_empty() {
        return Err(format!("unrecognized arguments: {}", extras.join(" ")));
    }
    Ok(())
}

fn resolve_verb(verb: Option<String>) -> Result<Option<String>, String> {
    match verb {
        None => Err("the following arguments are required: verb".to_string()),
        Some(v) => match v.as_str() {
            "stop" | "inhibit" | "status" | "lift" | "start" => Ok(Some(v)),
            _ => Err(format!(
                "argument verb: invalid choice: '{v}' (choose from stop, inhibit, status, lift, start)"
            )),
        },
    }
}

/// File a non-option token as the verb or a trailing extra.
fn take_positional(verb: &mut Option<String>, extras: &mut Vec<String>, arg: &str) {
    if verb.is_none() {
        *verb = Some(arg.to_string());
    } else {
        extras.push(arg.to_string());
    }
}

/// Consume one dash-led token: `--` flips end-of-options, `-h/--help`
/// and help prefixes exit early, anything else files an extra.
/// `None` means "advance and keep parsing".
fn take_option_token(
    prog: &str,
    arg: &str,
    end_of_opts: &mut bool,
    extras: &mut Vec<String>,
) -> Option<Result<Option<String>, String>> {
    if arg == "--" {
        *end_of_opts = true;
        return None;
    }
    if arg == "-h" || arg == "--help" {
        let _ = prog;
        return Some(Ok(None));
    }
    if let Some(early) = probe_help_prefix(arg) {
        return Some(early);
    }
    extras.push(arg.to_string());
    None
}

pub(crate) fn parse_args(prog: &str, args: &[String]) -> Result<Option<String>, String> {
    let mut verb: Option<String> = None;
    let mut i = 0;
    let mut end_of_opts = false;
    let mut extras: Vec<String> = Vec::new();
    while i < args.len() {
        let arg = &args[i];
        if end_of_opts || !arg.starts_with('-') || arg == "-" {
            take_positional(&mut verb, &mut extras, arg);
            i += 1;
            continue;
        }
        if let Some(early) = take_option_token(prog, arg, &mut end_of_opts, &mut extras) {
            return early;
        }
        i += 1;
    }
    check_extras(&extras)?;
    resolve_verb(verb)
}

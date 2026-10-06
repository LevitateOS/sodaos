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

pub(crate) fn parse_args(prog: &str, args: &[String]) -> Result<Option<String>, String> {
    let mut verb: Option<String> = None;
    let mut i = 0;
    let mut end_of_opts = false;
    let mut extras: Vec<String> = Vec::new();
    while i < args.len() {
        let arg = &args[i];
        if end_of_opts || !arg.starts_with('-') || arg == "-" {
            if verb.is_none() {
                verb = Some(arg.clone());
            } else {
                extras.push(arg.clone());
            }
            i += 1;
            continue;
        }
        if arg == "--" {
            end_of_opts = true;
            i += 1;
            continue;
        }
        if arg == "-h" || arg == "--help" {
            let _ = prog;
            return Ok(None);
        }
        if arg.starts_with("--help=") {
            let explicit = arg["--help=".len()..].to_string();
            return Err(format!("argument -h/--help: ignored explicit argument '{explicit}'"));
        }
        if arg.starts_with("--") && !arg.contains('=') {
            // Unambiguous long-option prefixes (help is the only one).
            let body = &arg[2..];
            if !body.is_empty() && "help".starts_with(body) {
                return Ok(None);
            }
        }
        extras.push(arg.clone());
        i += 1;
    }
    if !extras.is_empty() {
        return Err(format!("unrecognized arguments: {}", extras.join(" ")));
    }
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

#[derive(Debug)]
pub(crate) struct Options {
    pub(crate) config: String,
    pub(crate) project: String,
    pub(crate) tools: String,
    pub(crate) bind_only: bool,
}

pub(crate) fn default_tools() -> String {
    "/usr/share/soda/muse-tools".to_string()
}

pub(crate) fn parse(args: &[String]) -> Result<Options, String> {
    let mut config = String::from("/etc/soda/host.json");
    let mut project = String::new();
    let mut tools = default_tools();
    let mut bind_only = false;
    let mut i = 0;
    let mut positional = 0;
    while i < args.len() {
        let arg = &args[i];
        if arg == "--" {
            positional += args.len() - i - 1;
            break;
        }
        // Go stops flag parsing at the first non-flag argument.
        if !(arg.starts_with('-') && arg.len() > 1) {
            positional += args.len() - i;
            break;
        }
        // Go strips one dash, or two for a long flag; more is bad syntax.
        let raw = match arg.strip_prefix("--") {
            Some(s) => s,
            None => &arg[1..],
        };
        if raw.is_empty() || raw.starts_with('-') || raw.starts_with('=') {
            return Err(flag_error(format!("bad flag syntax: {arg}")));
        }
        let (key, inline) = match raw.find('=') {
            Some(p) => (&raw[..p], Some(&raw[p + 1..])),
            None => (raw, None),
        };
        match key {
            "config" | "project" | "tools" => {
                let value = match inline {
                    Some(v) => v.to_string(),
                    None => {
                        i += 1;
                        if i >= args.len() {
                            return Err(flag_error(format!("flag needs an argument: -{key}")));
                        }
                        args[i].clone()
                    }
                };
                match key {
                    "config" => config = value,
                    "project" => project = value,
                    _ => tools = value,
                }
            }
            "bind-only" => match inline {
                Some(v) => {
                    bind_only = parse_bool_flag(v).ok_or_else(|| {
                        flag_error(format!(
                            "invalid boolean value {} for -bind-only: parse error",
                            super::config_wire::go_quoted(v)
                        ))
                    })?;
                }
                None => bind_only = true,
            },
            "h" | "help" => {
                print_usage(&default_tools());
                return Err(String::from("flag: help requested"));
            }
            _ => {
                return Err(flag_error(format!("flag provided but not defined: -{key}")));
            }
        }
        i += 1;
    }
    let o = Options {
        config,
        project,
        tools,
        bind_only,
    };
    if positional != 0
        || !valid_project_id(&o.project)
        || !o.config.starts_with('/')
        || !o.tools.starts_with('/')
    {
        return Err(String::from(
            "explicit project and absolute maintenance paths required",
        ));
    }
    Ok(o)
}

fn print_usage(tools_default: &str) {
    eprint!("{}", usage_text(tools_default));
}

pub(crate) fn usage_text(tools_default: &str) -> String {
    format!(
        "Usage of soda-muse-maintain:\n  -bind-only\n    \trestore only the launch interface\n  -config string\n    \toperator host configuration (default \"/etc/soda/host.json\")\n  -project string\n    \texact project identity\n  -tools string\n    \tinstalled public tool directory (default \"{tools_default}\")\n"
    )
}

fn flag_error(msg: String) -> String {
    eprintln!("{msg}");
    print_usage(&default_tools());
    msg
}

fn parse_bool_flag(v: &str) -> Option<bool> {
    match v {
        "1" | "t" | "T" | "true" | "TRUE" | "True" => Some(true),
        "0" | "f" | "F" | "false" | "FALSE" | "False" => Some(false),
        _ => None,
    }
}

fn valid_project_id(id: &str) -> bool {
    // ^p[0-9a-f]{24}$: lowercase hex only.
    id.len() == 25 && id.starts_with('p') && id.bytes().skip(1).all(super::is_lower_hex)
}

#[cfg(test)]
#[path = "options_tests.rs"]
mod options_tests;

#[derive(Debug)]
pub(crate) struct Options {
    pub(crate) login: String,
    pub(crate) service: String,
    pub(crate) file: String,
    pub(crate) muse: bool,
}

pub(crate) fn parse_options(args: &[String]) -> Result<Options, String> {
    let mut login = String::new();
    let mut service = String::new();
    let mut file = String::from("compose.yml");
    let mut muse = false;
    let mut i = 0;
    let mut positional = 0;
    while i < args.len() {
        let arg = &args[i];
        if arg == "--" {
            positional += args.len() - i - 1;
            break;
        }
        if arg.starts_with('-') && arg.len() > 1 {
            let (name, inline) = match arg.find('=') {
                Some(p) => (&arg[..p], Some(&arg[p + 1..])),
                None => (arg.as_str(), None),
            };
            let key = name.trim_start_matches('-');
            match key {
                "login" | "service" | "file" => {
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
                        "login" => login = value,
                        "service" => service = value,
                        _ => file = value,
                    }
                }
                "muse" => {
                    match inline {
                        Some(v) => {
                            muse = parse_bool_flag(v).ok_or_else(|| {
                                flag_error(format!(
                                    "invalid boolean value {v:?} for -muse: parse error"
                                ))
                            })?;
                        }
                        None => {
                            // Support `-muse=false` style via next arg only when
                            // explicitly boolean; otherwise bare flag means true.
                            // Go's flag package does not consume the next arg
                            // for bools, so mirror that: bare presence is true.
                            muse = true;
                        }
                    }
                }
                "h" | "help" => {
                    print_usage();
                    return Err(String::from("flag: help requested"));
                }
                _ => {
                    return Err(flag_error(format!("flag provided but not defined: -{key}")));
                }
            }
        } else {
            positional += 1;
        }
        i += 1;
    }
    let o = Options {
        login,
        service,
        file,
        muse,
    };
    if !valid_options(&o, positional) {
        return Err(String::from(
            "project root, provisioned login, one service and Muse access required",
        ));
    }
    Ok(o)
}

fn print_usage() {
    eprintln!("Usage of soda-identity-compose:");
    eprintln!("  -file string");
    eprintln!("    \tCompose file (default \"compose.yml\")");
    eprintln!("  -login string");
    eprintln!("    \tauthorizing provisioned Soda account");
    eprintln!("  -muse");
    eprintln!("    \tallow Muse in the selected service");
    eprintln!("  -service string");
    eprintln!("    \tCompose service to opt in");
}

fn flag_error(msg: String) -> String {
    eprintln!("{msg}");
    print_usage();
    msg
}

pub(crate) fn parse_bool_flag(v: &str) -> Option<bool> {
    match v {
        "1" | "t" | "T" | "true" | "TRUE" | "True" => Some(true),
        "0" | "f" | "F" | "false" | "FALSE" | "False" => Some(false),
        _ => None,
    }
}

fn valid_options(o: &Options, remaining: usize) -> bool {
    remaining == 0
        && unsafe { libc::geteuid() } == 0
        && !o.login.is_empty()
        && super::go_base(&o.login) == o.login
        && !o.service.is_empty()
        && o.muse
}

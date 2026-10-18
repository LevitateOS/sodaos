#[derive(Debug)]
pub(crate) struct Options {
    pub(crate) login: String,
    pub(crate) service: String,
    pub(crate) file: String,
    pub(crate) muse: bool,
}

/// Take a string flag value from `-flag=value` or the following arg.
fn take_string_flag(
    args: &[String],
    i: &mut usize,
    key: &str,
    inline: Option<&str>,
) -> Result<String, String> {
    match inline {
        Some(v) => Ok(v.to_string()),
        None => {
            *i += 1;
            if *i >= args.len() {
                return Err(flag_error(format!("flag needs an argument: -{key}")));
            }
            Ok(args[*i].clone())
        }
    }
}

/// Bind the `-muse` boolean flag; bare presence means true.
/// Go's flag package does not consume the next arg for bools, so mirror
/// that instead of supporting `-muse=false` via the following arg.
fn bind_claude_flag(muse: &mut bool, inline: Option<&str>) -> Result<(), String> {
    match inline {
        Some(v) => {
            *muse = parse_bool_flag(v).ok_or_else(|| {
                flag_error(format!(
                    "invalid boolean value {v:?} for -muse: parse error"
                ))
            })?;
        }
        None => {
            *muse = true;
        }
    }
    Ok(())
}

/// Bind one `-flag[=value]` token into the partial slots.
fn bind_option_flag(
    args: &[String],
    i: &mut usize,
    login: &mut String,
    service: &mut String,
    file: &mut String,
    muse: &mut bool,
    arg: &str,
) -> Result<(), String> {
    let (name, inline) = match arg.find('=') {
        Some(p) => (&arg[..p], Some(&arg[p + 1..])),
        None => (arg, None),
    };
    let key = name.trim_start_matches('-');
    match key {
        "login" | "service" | "file" => {
            let value = take_string_flag(args, i, key, inline)?;
            match key {
                "login" => *login = value,
                "service" => *service = value,
                _ => *file = value,
            }
        }
        "muse" => bind_claude_flag(muse, inline)?,
        "h" | "help" => {
            print_usage();
            return Err(String::from("flag: help requested"));
        }
        _ => {
            return Err(flag_error(format!("flag provided but not defined: -{key}")));
        }
    }
    Ok(())
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
            bind_option_flag(
                args,
                &mut i,
                &mut login,
                &mut service,
                &mut file,
                &mut muse,
                arg,
            )?;
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
        && std::path::Path::new(&o.login).file_name() == Some(std::ffi::OsStr::new(&o.login))
        && !o.service.is_empty()
        && o.muse
}

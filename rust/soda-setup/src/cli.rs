use std::io::{self, Write};
use std::path::Path;

use crate::secrets::provision_postgres_secrets;
use crate::setup::setup;
use crate::system::{euid, POSTGRES_SECRET_DIR};

pub(crate) fn run() -> i32 {
    if euid() != 0 {
        eprintln!("run through the operator's native root access");
        return 1;
    }
    let args: Vec<String> = std::env::args().skip(1).collect();
    let opts = match parse_args(&args) {
        Ok(opts) => opts,
        Err(ParseOutcome::Help) => return 0,
        Err(ParseOutcome::Error) => return 2,
    };
    let mut out = io::stdout();
    if opts.provision_db_only {
        match provision_postgres_secrets(Path::new(POSTGRES_SECRET_DIR)) {
            Ok((_, dsn_path)) => {
                println!("Database credentials ready: {}", dsn_path.display());
                0
            }
            Err(err) => {
                eprintln!("{err}");
                1
            }
        }
    } else {
        match setup(
            &opts.forgejo_url,
            &opts.forgejo_internal_url,
            &opts.token_file,
            Path::new(&opts.out),
            Path::new(POSTGRES_SECRET_DIR),
            &mut out,
        ) {
            Ok(()) => 0,
            Err(err) => {
                let _ = out.flush();
                eprintln!("{err}");
                1
            }
        }
    }
}

pub(crate) struct SetupOpts {
    pub(crate) forgejo_url: String,
    pub(crate) forgejo_internal_url: String,
    token_file: String,
    out: String,
    pub(crate) provision_db_only: bool,
}

#[derive(Debug)]
pub(crate) enum ParseOutcome {
    Help,
    Error,
}

fn print_setup_usage() {
    eprint!(
        "Usage of soda-setup:\n  -forgejo-internal-url string\n    \tnative Forgejo origin\n  -forgejo-url string\n    \tForgejo HTTPS origin\n  -out string\n    \tnew dashboard configuration\n  -provision-db-only\n    \tgenerate database credentials without operator binding\n  -token-file string\n    \toperator Forgejo access token file\n"
    );
}

fn parse_bool_flag(
    name: &str,
    inline: Option<&str>,
    args: &[String],
    i: &mut usize,
) -> Result<bool, ParseOutcome> {
    match inline {
        Some(v) => parse_go_bool(name, v),
        None => {
            // A following token that is not another flag is the value; Go
            // treats a bare boolean flag as true.
            if *i + 1 < args.len() && !looks_like_flag(&args[*i + 1]) {
                *i += 1;
                parse_go_bool(name, &args[*i].clone())
            } else {
                Ok(true)
            }
        }
    }
}

fn parse_go_bool(name: &str, value: &str) -> Result<bool, ParseOutcome> {
    match value {
        "1" | "t" | "T" | "true" | "TRUE" | "True" => Ok(true),
        "0" | "f" | "F" | "false" | "FALSE" | "False" => Ok(false),
        _ => {
            let msg = format!("invalid boolean value {value:?} for -{name}: parse error");
            eprintln!("{msg}");
            print_setup_usage();
            Err(ParseOutcome::Error)
        }
    }
}

fn looks_like_flag(arg: &str) -> bool {
    arg.len() > 1 && arg.starts_with('-')
}

pub(crate) fn parse_args(args: &[String]) -> Result<SetupOpts, ParseOutcome> {
    let mut opts = SetupOpts {
        forgejo_url: String::new(),
        forgejo_internal_url: "http://127.0.0.1:3000".to_string(),
        token_file: String::new(),
        out: "/etc/soda/dashboard.json".to_string(),
        provision_db_only: false,
    };
    let mut i = 0;
    while i < args.len() {
        let arg = &args[i];
        if !arg.starts_with('-') || arg == "-" {
            // Go's flag package stops at the first positional argument and
            // setup never inspects Args, so positionals are ignored.
            break;
        }
        if arg == "--" {
            break;
        }
        let flag = arg.trim_start_matches('-');
        let (name, inline) = match flag.split_once('=') {
            Some((n, v)) => (n, Some(v)),
            None => (flag, None),
        };
        if name == "h" || name == "help" {
            print_setup_usage();
            return Err(ParseOutcome::Help);
        }
        match name {
            "provision-db-only" => {
                opts.provision_db_only = parse_bool_flag(name, inline, args, &mut i)?;
            }
            "forgejo-url" | "forgejo-internal-url" | "token-file" | "out" => {
                let value = match inline {
                    Some(v) => v.to_string(),
                    None => {
                        i += 1;
                        if i >= args.len() {
                            let msg = format!("flag needs an argument: -{name}");
                            eprintln!("{msg}");
                            print_setup_usage();
                            return Err(ParseOutcome::Error);
                        }
                        args[i].clone()
                    }
                };
                match name {
                    "forgejo-url" => opts.forgejo_url = value,
                    "forgejo-internal-url" => opts.forgejo_internal_url = value,
                    "token-file" => opts.token_file = value,
                    _ => opts.out = value,
                }
            }
            _ => {
                let msg = format!("flag provided but not defined: -{name}");
                eprintln!("{msg}");
                print_setup_usage();
                return Err(ParseOutcome::Error);
            }
        }
        i += 1;
    }
    Ok(opts)
}

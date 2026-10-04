// soda-forgejo-locales merges the exact native Forgejo INI bytes with Soda
// additions, without reserializing (ports scripts/forgejo-locales.py).
use std::ffi::OsString;
use std::path::PathBuf;

use soda_forgejo_locales::locales::{Error, Native, DEFAULT_ADDITIONS};
use soda_forgejo_locales::split_flag;

const PROG: &str = "soda-forgejo-locales";
const USAGE: &str =
    "usage: soda-forgejo-locales (--native FILE | --lock FILE) [--additions FILE] --out FILE";
const HELP: &str = "usage: soda-forgejo-locales (--native FILE | --lock FILE) [--additions FILE] --out FILE\n\nCombine exact native Forgejo INI bytes with Soda additions, without reserializing.\n\nNative JSON catalogs remain untouched. Custom INI catalogs replace native files;\nthis command therefore requires the complete extracted native catalog as input.\n\noptions:\n  --native FILE     complete extracted native catalog (max 1 MiB)\n  --lock FILE       fetch the exact locked native catalog at build time\n  --additions FILE  Soda additions (default: appliance/forgejo/i18n/en-US.ini)\n  --out FILE        merged catalog to exclusively create\n";

struct Args {
    native: Native,
    additions: PathBuf,
    out: PathBuf,
}

fn refusal(message: String) -> ! {
    eprintln!("{USAGE}\n{PROG}: error: {message}");
    std::process::exit(2);
}

fn fail(error: Error) -> ! {
    match error {
        Error::Usage(message) => refusal(message),
        Error::Runtime(message) => {
            eprintln!("{PROG}: error: {message}");
            std::process::exit(1);
        }
    }
}

fn take_value(name: &str, inline: Option<String>, args: &[OsString], i: &mut usize) -> String {
    if let Some(value) = inline {
        return value;
    }
    *i += 1;
    if *i >= args.len() {
        refusal(format!("argument --{name}: expected one argument"));
    }
    args[*i].to_string_lossy().into_owned()
}

fn parse(args: &[OsString]) -> Args {
    let mut native: Option<PathBuf> = None;
    let mut lock: Option<PathBuf> = None;
    let mut additions = PathBuf::from(DEFAULT_ADDITIONS);
    let mut out: Option<PathBuf> = None;
    let mut stray: Vec<String> = Vec::new();
    let mut i = 0;
    let mut only_positionals = false;
    while i < args.len() {
        let raw = args[i].to_string_lossy().into_owned();
        if !only_positionals && raw == "--" {
            only_positionals = true;
            i += 1;
            continue;
        }
        if !only_positionals && (raw == "-h" || raw == "-help") {
            print!("{HELP}");
            std::process::exit(0);
        }
        let token = if only_positionals {
            None
        } else {
            split_flag(&raw)
        };
        match token {
            Some((name, inline)) if name == "native" => {
                native = Some(PathBuf::from(take_value(&name, inline, args, &mut i)));
            }
            Some((name, inline)) if name == "lock" => {
                lock = Some(PathBuf::from(take_value(&name, inline, args, &mut i)));
            }
            Some((name, inline)) if name == "additions" => {
                additions = PathBuf::from(take_value(&name, inline, args, &mut i));
            }
            Some((name, inline)) if name == "out" => {
                out = Some(PathBuf::from(take_value(&name, inline, args, &mut i)));
            }
            Some((name, _)) if name == "help" => {
                print!("{HELP}");
                std::process::exit(0);
            }
            Some((name, _)) => {
                refusal(format!("unrecognized arguments: --{name}"));
            }
            None => {
                stray.push(raw);
            }
        }
        i += 1;
    }
    if !stray.is_empty() {
        refusal(format!("unrecognized arguments: {}", stray.join(" ")));
    }
    if native.is_some() && lock.is_some() {
        refusal("argument --lock: not allowed with argument --native".to_string());
    }
    let Some(out) = out else {
        refusal("the following arguments are required: --out".to_string());
    };
    if let Some(path) = native {
        Args {
            native: Native::File(path),
            additions,
            out,
        }
    } else if let Some(path) = lock {
        Args {
            native: Native::Lock(path),
            additions,
            out,
        }
    } else {
        refusal("one of the arguments --native --lock is required".to_string())
    }
}

fn main() {
    let argv: Vec<OsString> = std::env::args_os().skip(1).collect();
    let parsed = parse(&argv);
    if let Err(error) =
        soda_forgejo_locales::locales::run(&parsed.native, &parsed.additions, &parsed.out)
    {
        fail(error);
    }
}

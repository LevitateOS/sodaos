// soda-fetch-terminal fetches exact upstream terminal distributions, never
// runtime/CDN dependencies (ports scripts/fetch-terminal.py).
use std::path::PathBuf;

use soda_asset_fetchers::{flag_token, terminal};

const USAGE: &str = "usage: soda-fetch-terminal --out DIR";

fn parse(args: &[String]) -> Result<PathBuf, String> {
    let mut out: Option<PathBuf> = None;
    let mut i = 0;
    while i < args.len() {
        let arg = &args[i];
        if arg == "--" {
            return Err(USAGE.to_string());
        }
        let Some((name, inline)) = flag_token(arg) else {
            return Err(USAGE.to_string());
        };
        let value = match inline {
            Some(value) => value,
            None => {
                i += 1;
                args.get(i).cloned().ok_or_else(|| USAGE.to_string())?
            }
        };
        match name.as_str() {
            "out" => out = Some(PathBuf::from(value)),
            _ => return Err(USAGE.to_string()),
        }
        i += 1;
    }
    out.ok_or_else(|| USAGE.to_string())
}

fn main() {
    let args: Vec<String> = std::env::args().skip(1).collect();
    if args
        .iter()
        .any(|a| a == "--help" || a == "-help" || a == "-h")
    {
        println!("{USAGE}");
        return;
    }
    let out = match parse(&args) {
        Ok(out) => out,
        Err(usage) => {
            eprintln!("{usage}");
            std::process::exit(2);
        }
    };
    let lock = match terminal::default_lock() {
        Ok(lock) => lock,
        Err(e) => {
            eprintln!("{e}");
            std::process::exit(1);
        }
    };
    if let Err(e) = terminal::fetch(&lock, &out) {
        eprintln!("{e}");
        std::process::exit(1);
    }
}

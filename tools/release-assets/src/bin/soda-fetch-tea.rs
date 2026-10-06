// soda-fetch-tea stages an upstream Linux Tea binary and license for the
// project image (ports scripts/fetch-tea.py).
use std::path::PathBuf;

use soda_release_assets::fetch::{flag_token, tea};

const USAGE: &str = "usage: soda-fetch-tea --arch x86_64 [--out DIR]";

fn parse(args: &[String]) -> Result<(String, Option<PathBuf>), String> {
    let mut arch: Option<String> = None;
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
            "arch" => arch = Some(value),
            "out" => out = Some(PathBuf::from(value)),
            _ => return Err(USAGE.to_string()),
        }
        i += 1;
    }
    Ok((arch.ok_or_else(|| USAGE.to_string())?, out))
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
    let (arch, out) = match parse(&args) {
        Ok(parsed) => parsed,
        Err(usage) => {
            eprintln!("{usage}");
            std::process::exit(2);
        }
    };
    let out = match out {
        Some(out) => out,
        None => match tea::default_out(&arch) {
            Ok(out) => out,
            Err(e) => {
                eprintln!("{e}");
                std::process::exit(1);
            }
        },
    };
    match tea::fetch(&arch, &out, &tea::Endpoints::production()) {
        Ok(message) => println!("{message}"),
        Err(e) => {
            eprintln!("{e}");
            std::process::exit(1);
        }
    }
}

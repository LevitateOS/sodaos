// soda-fetch-muse stages the checksum-pinned native Muse executable for builds.
use std::path::Path;

use soda_asset_fetchers::{flag_token, muse};

const USAGE: &str = "usage: soda-fetch-muse [--arch ARCH] [--manifest PATH] [--out PATH]";

struct Options {
    arch: String,
    manifest: String,
    out: String,
}

fn parse(args: &[String]) -> Result<Options, String> {
    let mut options = Options {
        arch: String::new(),
        manifest: "project-os/muse-release.json".to_string(),
        out: String::new(),
    };
    let mut i = 0;
    while i < args.len() {
        let arg = &args[i];
        if arg == "--" {
            // Like the Go flag owner, trailing positionals are ignored.
            break;
        }
        let Some((name, inline)) = flag_token(arg) else {
            i += 1;
            continue;
        };
        let value = match inline {
            Some(value) => value,
            None => {
                i += 1;
                args.get(i).cloned().ok_or_else(|| USAGE.to_string())?
            }
        };
        match name.as_str() {
            "arch" => options.arch = value,
            "manifest" => options.manifest = value,
            "out" => options.out = value,
            _ => return Err(USAGE.to_string()),
        }
        i += 1;
    }
    Ok(options)
}

fn run() -> Result<(), String> {
    let args: Vec<String> = std::env::args().skip(1).collect();
    if args.iter().any(|a| a == "--help" || a == "-help" || a == "-h") {
        println!("{USAGE}");
        return Ok(());
    }
    let options = parse(&args)?;
    muse::fetch_muse(
        &options.manifest,
        &options.arch,
        Path::new(&options.out),
        muse::DOWNLOAD_BASE,
    )?;
    Ok(())
}

fn main() {
    if let Err(e) = run() {
        eprintln!("soda-fetch-muse: {e}");
        std::process::exit(1);
    }
}

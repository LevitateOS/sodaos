// soda-fetch-muse stages the checksum-pinned native Muse executable for builds.
//
// The default (no-flag) entrypoint admits bytes only after live freshness
// validation: current pins stage (cached reuse or verified download), stale
// pins are refused with exit 2 (record via --refresh first), and unknown
// upstream refuses any current-baseline claim. Ordinary builds therefore no
// longer depend on the operator remembering --check.
//
// --check compares the pinned manifest against the live official channel and
// reports current/stale/unknown without touching the file (exit 0 current,
// 2 stale-or-unknown, 1 operational error). --refresh resolves the live
// official release and records it in the manifest; it is the only mode that
// writes, and it leaves the file untouched when upstream is unusable or the
// pin is already current.
use std::path::Path;

use soda_release_assets::fetch::{flag_token, muse};

const USAGE: &str =
    "usage: soda-fetch-muse [--arch ARCH] [--manifest PATH] [--out PATH] [--check] [--refresh]";

struct Options {
    arch: String,
    manifest: String,
    out: String,
    check: bool,
    refresh: bool,
}

fn parse(args: &[String]) -> Result<Options, String> {
    let mut options = Options {
        arch: String::new(),
        manifest: "system/project/muse-release.json".to_string(),
        out: String::new(),
        check: false,
        refresh: false,
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
        match name.as_str() {
            "check" => {
                if inline.is_some() {
                    return Err(USAGE.to_string());
                }
                options.check = true;
            }
            "refresh" => {
                if inline.is_some() {
                    return Err(USAGE.to_string());
                }
                options.refresh = true;
            }
            "arch" | "manifest" | "out" => {
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
                    _ => options.out = value,
                }
            }
            _ => return Err(USAGE.to_string()),
        }
        i += 1;
    }
    if options.check && options.refresh {
        return Err(USAGE.to_string());
    }
    Ok(options)
}

fn run() -> (i32, Option<String>) {
    let args: Vec<String> = std::env::args().skip(1).collect();
    if args
        .iter()
        .any(|a| a == "--help" || a == "-help" || a == "-h")
    {
        println!("{USAGE}");
        return (0, None);
    }
    let options = match parse(&args) {
        Ok(options) => options,
        Err(e) => return (1, Some(e)),
    };
    if options.check {
        return match muse::check_muse(&options.manifest, &options.arch) {
            Ok(muse::Baseline::Current { version }) => (
                0,
                Some(format!("muse baseline is current: {version}")),
            ),
            Ok(muse::Baseline::Stale { detail }) => {
                (2, Some(format!("muse baseline is stale: {detail}")))
            }
            Ok(muse::Baseline::Unknown { reason }) => (
                2,
                Some(format!("muse baseline is unknown: {reason}")),
            ),
            Err(e) => (1, Some(e)),
        };
    }
    if options.refresh {
        return match muse::refresh_muse(&options.manifest, &options.arch) {
            Ok(version) => (
                0,
                Some(format!("muse baseline records current release: {version}")),
            ),
            Err(e) => {
                let code = if e.starts_with("refusing to record an unknown baseline") {
                    2
                } else {
                    1
                };
                (code, Some(e))
            }
        };
    }
    // Default entrypoint: live freshness gates admission, so ordinary builds
    // can no longer stage a stale pin or claim a baseline on unknown input.
    match muse::admit_current(
        &options.manifest,
        &options.arch,
        Path::new(&options.out),
        muse::DOWNLOAD_BASE,
    ) {
        Ok(admitted) => (
            0,
            Some(format!(
                "muse baseline admitted current: {} sha256:{} size:{}",
                admitted.version, admitted.sha256, admitted.size
            )),
        ),
        Err(muse::FetchRefusal::Stale(detail)) => {
            (2, Some(format!("muse baseline is stale: {detail}")))
        }
        Err(muse::FetchRefusal::Unknown(reason)) => (
            2,
            Some(format!("muse baseline is unknown: {reason}")),
        ),
        Err(muse::FetchRefusal::Failed(error)) => (1, Some(error)),
    }
}

fn main() {
    let (code, message) = run();
    match (code, message) {
        (0, None) => {}
        (0, Some(message)) => println!("soda-fetch-muse: {message}"),
        (_, Some(message)) => {
            eprintln!("soda-fetch-muse: {message}");
            std::process::exit(code);
        }
        (_, None) => std::process::exit(code),
    }
}

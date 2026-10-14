use soda_release_assets::fetch::canlang;
use std::path::PathBuf;

fn run() -> Result<(), String> {
    let mut manifest = PathBuf::from("system/project/canlang-release.json");
    let mut archive = None;
    let mut out = None;
    let mut args = std::env::args().skip(1);
    while let Some(flag) = args.next() {
        if flag == "--help" {
            println!("usage: soda-stage-canlang --archive PATH --out DIR [--manifest PATH]");
            return Ok(());
        }
        let value = args.next().ok_or("missing option value")?;
        match flag.as_str() {
            "--manifest" => manifest = PathBuf::from(value),
            "--archive" if archive.is_none() => archive = Some(PathBuf::from(value)),
            "--out" if out.is_none() => out = Some(PathBuf::from(value)),
            _ => return Err("unknown or duplicate option".into()),
        }
    }
    canlang::stage(
        &manifest,
        &archive.ok_or("--archive required")?,
        &out.ok_or("--out required")?,
    )
}

fn main() {
    if let Err(error) = run() {
        eprintln!("soda-stage-canlang: {error}");
        std::process::exit(1);
    }
}

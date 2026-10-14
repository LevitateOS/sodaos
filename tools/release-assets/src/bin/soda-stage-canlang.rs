use soda_release_assets::fetch::canlang;
use std::path::PathBuf;

fn run() -> Result<(), String> {
    let mut manifest = None;
    let mut arch = None;
    let mut archive = None;
    let mut out = None;
    let mut args = std::env::args().skip(1);
    while let Some(flag) = args.next() {
        if flag == "--help" {
            println!("usage: soda-stage-canlang --arch x86_64|aarch64 --archive PATH --out DIR [--manifest PATH]");
            return Ok(());
        }
        let value = args.next().ok_or("missing option value")?;
        match flag.as_str() {
            "--manifest" if manifest.is_none() => manifest = Some(PathBuf::from(value)),
            "--arch" if arch.is_none() => arch = Some(value),
            "--archive" if archive.is_none() => archive = Some(PathBuf::from(value)),
            "--out" if out.is_none() => out = Some(PathBuf::from(value)),
            _ => return Err("unknown or duplicate option".into()),
        }
    }
    let arch = arch.ok_or("--arch required")?;
    let selected =
        soda_build_tools::reader::canlang::release_path(&arch).map_err(|error| error.0)?;
    let manifest = manifest.unwrap_or_else(|| PathBuf::from(selected));
    canlang::stage(
        &manifest,
        &archive.ok_or("--archive required")?,
        &out.ok_or("--out required")?,
        &arch,
    )
}

fn main() {
    if let Err(error) = run() {
        eprintln!("soda-stage-canlang: {error}");
        std::process::exit(1);
    }
}

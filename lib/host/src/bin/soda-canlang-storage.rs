//! Explicit stopped backup and fresh-directory restore; no automatic schedule.
use std::path::PathBuf;

fn run() -> Result<(), String> {
    let mut args = std::env::args().skip(1);
    let operation = args.next().ok_or("choose backup or restore")?;
    if operation == "--help" {
        println!("soda-canlang-storage (backup|restore) --source ABSOLUTE_DIRECTORY --out NEW_ABSOLUTE_DIRECTORY");
        return Ok(());
    }
    let mut source = None;
    let mut out = None;
    while let Some(flag) = args.next() {
        let value = args.next().ok_or("missing snapshot option value")?;
        match flag.as_str() {
            "--source" if source.is_none() => source = Some(PathBuf::from(value)),
            "--out" if out.is_none() => out = Some(PathBuf::from(value)),
            _ => return Err("unknown or duplicate snapshot option".into()),
        }
    }
    let source = source.ok_or("--source is required")?;
    let out = out.ok_or("--out is required")?;
    match operation.as_str() {
        "backup" => soda_host::application::snapshot::backup(&source, &out),
        "restore" => soda_host::application::snapshot::restore(&source, &out),
        _ => Err("choose backup or restore".into()),
    }
}

fn main() {
    if let Err(error) = run() {
        eprintln!("soda-canlang-storage: {error}");
        std::process::exit(1);
    }
}

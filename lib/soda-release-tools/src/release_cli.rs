//! Thin operator entrypoint for GitHub staging and exact install/boot promotion.
use clap::{Arg, Command};
use soda_release_deliver::publish;
use std::path::Path;

fn command() -> Command {
    let stage = ["candidate", "media", "tag", "notes", "out"]
        .into_iter()
        .fold(
            Command::new("stage").about("Publish an explicitly unqualified immutable prerelease"),
            |c, name| c.arg(Arg::new(name).long(name).required(true).num_args(1)),
        );
    let promote = ["input", "qualification", "downloaded-iso"]
        .into_iter()
        .fold(
            Command::new("promote")
                .about("Promote unchanged bytes after native install/boot smoke checks"),
            |c, name| c.arg(Arg::new(name).long(name).required(true).num_args(1)),
        );
    Command::new("soda-release")
        .subcommand_required(true)
        .subcommand(stage)
        .subcommand(promote)
}
pub fn run(args: impl IntoIterator<Item = String>) -> Result<(), String> {
    let cli = match command().try_get_matches_from(args) {
        Ok(value) => value,
        Err(error) if error.kind() == clap::error::ErrorKind::DisplayHelp => {
            print!("{error}");
            return Ok(());
        }
        Err(error) => return Err(error.to_string()),
    };
    let (action, flags) = cli
        .subcommand()
        .ok_or_else(|| "release action required".to_owned())?;
    let get = |name: &str| {
        flags
            .get_one::<String>(name)
            .map(String::as_str)
            .ok_or_else(|| format!("{name} required"))
    };
    match action {
        "stage" => {
            let manifest = publish::prepare(
                Path::new(get("candidate")?),
                Path::new(get("media")?),
                get("tag")?,
                Path::new(get("notes")?),
                Path::new(get("out")?),
            )
            .map_err(|e| e.to_string())?;
            publish::stage(&publish::Gh, &manifest, Path::new(get("out")?))
                .map_err(|e| e.to_string())
        }
        "promote" => publish::promote(
            &publish::Gh,
            Path::new(get("input")?),
            Path::new(get("qualification")?),
            Path::new(get("downloaded-iso")?),
        )
        .map_err(|e| e.to_string()),
        _ => Err("unknown release action".to_owned()),
    }
}
pub fn main() {
    if let Err(error) = run(std::env::args()) {
        eprintln!("{error}");
        std::process::exit(1);
    }
}
#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn release_cli_requires_exact_inputs_and_handles_help() {
        assert!(run(["soda-release", "--help"].map(str::to_owned)).is_ok());
        assert!(run(["soda-release", "stage"].map(str::to_owned)).is_err());
        assert!(run(["soda-release", "promote", "--input", "missing"].map(str::to_owned)).is_err());
    }
}

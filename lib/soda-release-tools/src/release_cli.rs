//! Thin operator entrypoint for GitHub staging and exact install/boot promotion.
use clap::{Arg, ArgAction, Command};
use soda_release_deliver::publish;
use std::path::Path;

fn command() -> Command {
    let stage = ["candidate", "tag", "notes", "out"].into_iter().fold(
        Command::new("stage").about("Publish an explicitly unqualified immutable prerelease"),
        |c, name| c.arg(Arg::new(name).long(name).required(true).num_args(1)),
    );
    let promote = ["input", "downloaded-iso", "observed-revision"]
        .into_iter()
        .fold(
            Command::new("promote")
                .about("Promote unchanged bytes after native install/boot smoke checks"),
            |c, name| c.arg(Arg::new(name).long(name).required(true).num_args(1)),
        )
        .arg(
            Arg::new("smoke-passed")
                .long("smoke-passed")
                .help("Confirm this downloaded ISO installed, booted without media, and all candidate images were present")
                .action(ArgAction::SetTrue)
                .required(true),
        );
    let qualify_upgrade = ["candidate", "evidence"].into_iter().fold(
        Command::new("qualify-upgrade")
            .about("Stamp exact native transition, recovery and stopped-Project observations"),
        |c, name| c.arg(Arg::new(name).long(name).required(true).num_args(1)),
    );
    Command::new("soda-release")
        .subcommand_required(true)
        .subcommand(stage)
        .subcommand(promote)
        .subcommand(qualify_upgrade)
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
        "qualify-upgrade" => soda_release_deliver::upgrade::stamp(
            Path::new(get("candidate")?),
            Path::new(get("evidence")?),
        )
        .map(|_| ())
        .map_err(|e| e.to_string()),
        "stage" => {
            let manifest = publish::prepare(
                Path::new(get("candidate")?),
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
            Path::new(get("downloaded-iso")?),
            get("observed-revision")?,
            flags.get_flag("smoke-passed"),
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
        assert!(run(["soda-release", "qualify-upgrade"].map(str::to_owned)).is_err());
        assert!(run(["soda-release", "promote", "--input", "missing"].map(str::to_owned)).is_err());
        let promote = [
            "soda-release",
            "promote",
            "--input",
            "release",
            "--downloaded-iso",
            "download.iso",
            "--observed-revision",
            "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
        ];
        assert!(command().try_get_matches_from(promote).is_err());
        assert!(command()
            .try_get_matches_from(promote.into_iter().chain(["--smoke-passed"]))
            .is_ok());
    }
}

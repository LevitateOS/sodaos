//! Thin operator entrypoint for local preparation, GitHub staging and promotion.
use clap::{Arg, ArgAction, Command};
use soda_release_deliver::publish;
use std::path::Path;

pub(crate) fn command() -> Command {
    let release_inputs = |action, help| {
        ["candidate", "tag", "notes", "out"]
            .into_iter()
            .fold(Command::new(action).about(help), |c, name| {
                c.arg(Arg::new(name).long(name).required(true).num_args(1))
            })
    };
    let prepare = release_inputs(
        "prepare",
        "Prepare the complete release bundle locally without publication",
    )
    .arg(
        Arg::new("distribution")
            .long("distribution")
            .default_value("appliance")
            .value_parser(["appliance", "docker", "all"]),
    );
    let stage = Command::new("stage")
        .about("Stage every selected target as one unqualified immutable prerelease")
        .arg(Arg::new("input").long("input").required(true).num_args(1))
        .arg(Arg::new("auth-file").long("auth-file").num_args(1));
    let verify_images = Command::new("verify-images")
        .about("Finish public registry readback without repeating an upload")
        .arg(Arg::new("input").long("input").required(true).num_args(1));
    let promote = Command::new("promote")
        .about("Promote the staged prerelease and move the recommendation pointers")
        .arg(Arg::new("input").long("input").required(true).num_args(1))
        .arg(Arg::new("auth-file").long("auth-file").num_args(1));
    let status = Command::new("status")
        .about("Inspect prepared inputs, remote state, blockers and next action without mutation")
        .arg(Arg::new("input").long("input").required(true).num_args(1));
    let start = Command::new("start")
        .about("Guide a production build and local preparation; never publish or deploy")
        .args(
            ["distribution", "tag", "worker-config", "notes", "out"]
                .map(|name| Arg::new(name).long(name).num_args(1)),
        )
        .arg(
            Arg::new("non-interactive")
                .long("non-interactive")
                .action(ArgAction::SetTrue),
        )
        .arg(
            Arg::new("preflight")
                .long("preflight")
                .help("Check all release prerequisites without building or creating an attempt")
                .action(ArgAction::SetTrue),
        );
    Command::new("soda-release")
        .subcommand_required(true)
        .subcommand(prepare)
        .subcommand(stage)
        .subcommand(verify_images)
        .subcommand(promote)
        .subcommand(start)
        .subcommand(status)
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
    match action {
        "start" => crate::release_start::run(flags),
        "stage" | "promote" => run_stage_or_promote(action, flags),
        "status" => run_status(flags),
        "verify-images" => run_verify_images(flags),
        "prepare" => run_prepare(flags),
        _ => Err("unknown release action".to_owned()),
    }
}

fn required_input<'a>(flags: &'a clap::ArgMatches, name: &str) -> Result<&'a str, String> {
    flags
        .get_one::<String>(name)
        .map(String::as_str)
        .ok_or_else(|| format!("{name} required"))
}

fn run_stage_or_promote(action: &str, flags: &clap::ArgMatches) -> Result<(), String> {
    let out = Path::new(required_input(flags, "input")?);
    let auth = flags
        .get_one::<String>("auth-file")
        .map(|file| Path::new(file));
    let result = if action == "stage" {
        publish::stage(&publish::Gh, &publish::registry::Skopeo, out, auth)
    } else {
        publish::promote(&publish::Gh, &publish::registry::Skopeo, out, auth)
    };
    result.map_err(|error| error.to_string())
}

fn run_status(flags: &clap::ArgMatches) -> Result<(), String> {
    let result = publish::status(
        &publish::Gh,
        &publish::registry::Skopeo,
        Path::new(required_input(flags, "input")?),
    )
    .map_err(|error| error.to_string())?;
    println!(
        "{}",
        serde_json::to_string_pretty(&result).map_err(|error| error.to_string())?
    );
    Ok(())
}

fn run_verify_images(flags: &clap::ArgMatches) -> Result<(), String> {
    let input = Path::new(required_input(flags, "input")?);
    publish::verify_registry(&publish::Gh, &publish::registry::Skopeo, input)
        .map_err(|error| error.to_string())
}

fn run_prepare(flags: &clap::ArgMatches) -> Result<(), String> {
    let manifest = publish::prepare(
        Path::new(required_input(flags, "candidate")?),
        required_input(flags, "tag")?,
        crate::release_start::distribution(required_input(flags, "distribution")?)?,
        Path::new(required_input(flags, "notes")?),
        Path::new(required_input(flags, "out")?),
    )
    .map_err(|e| e.to_string())?;
    println!(
        "Prepared {} at {}; no remote publication.",
        manifest.tag,
        required_input(flags, "out")?
    );
    Ok(())
}
pub fn main() {
    if let Err(error) = run(std::env::args()) {
        eprintln!("{error}");
        std::process::exit(1);
    }
}

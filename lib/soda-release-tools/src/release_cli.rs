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
    let verify_docker = Command::new("verify-docker")
        .about("Finish public registry readback without repeating an upload")
        .arg(Arg::new("input").long("input").required(true).num_args(1));
    let promote = Command::new("promote")
        .about("Promote the unchanged qualified digest and eligible recommendation pointers")
        .arg(Arg::new("input").long("input").required(true).num_args(1))
        .arg(Arg::new("auth-file").long("auth-file").num_args(1));
    let status = Command::new("status")
        .about("Inspect prepared inputs, remote state, blockers and next action without mutation")
        .arg(Arg::new("input").long("input").required(true).num_args(1));
    let qualify = Command::new("qualify")
        .about("Validate exact installed evidence for every selected target")
        .arg(Arg::new("input").long("input").required(true).num_args(1))
        .args(
            [
                "evidence",
                "downloaded-iso",
                "observed-revision",
                "platform-evidence",
            ]
            .map(|name| Arg::new(name).long(name).num_args(1)),
        )
        .arg(
            Arg::new("smoke-passed")
                .long("smoke-passed")
                .action(ArgAction::SetTrue),
        );
    let start = Command::new("start")
        .about("Guide a production build and local preparation; never publish or deploy")
        .arg(
            Arg::new("arch")
                .long("arch")
                .num_args(1)
                .default_value(std::env::consts::ARCH),
        )
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
    let qualify_upgrade = ["candidate", "evidence"].into_iter().fold(
        Command::new("qualify-upgrade")
            .about("Stamp exact native transition, recovery and stopped-Project observations"),
        |c, name| c.arg(Arg::new(name).long(name).required(true).num_args(1)),
    );
    Command::new("soda-release")
        .subcommand_required(true)
        .subcommand(prepare)
        .subcommand(stage)
        .subcommand(verify_docker)
        .subcommand(promote)
        .subcommand(start)
        .subcommand(status)
        .subcommand(qualify)
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
        "start" => crate::release_start::run(flags),
        "stage" | "promote" => {
            let out = Path::new(get("input")?);
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
        "status" => {
            let result = publish::status(
                &publish::Gh,
                &publish::registry::Skopeo,
                Path::new(get("input")?),
            )
            .map_err(|error| error.to_string())?;
            println!(
                "{}",
                serde_json::to_string_pretty(&result).map_err(|error| error.to_string())?
            );
            Ok(())
        }
        "verify-docker" => {
            let input = Path::new(get("input")?);
            publish::verify_registry(&publish::Gh, &publish::registry::Skopeo, input)
                .map_err(|error| error.to_string())
        }
        "qualify" => {
            let input = Path::new(get("input")?);
            let manifest = publish::assets::load(input).map_err(|error| error.to_string())?;
            qualification_inputs(flags, manifest.distribution)?;
            if manifest.architecture == "aarch64"
                && manifest.has_appliance()
                && flags
                    .get_one::<String>("platform-evidence")
                    .is_none_or(String::is_empty)
            {
                return Err("--platform-evidence required for ARM64 UTM qualification before any qualification is recorded".into());
            }
            if manifest.docker.is_some() {
                publish::qualify_docker(
                    &publish::Gh,
                    &publish::registry::Skopeo,
                    &manifest,
                    input,
                    Path::new(get("evidence")?),
                )
                .map_err(|error| error.to_string())?;
            }
            if manifest.has_appliance() {
                publish::qualify_appliance(
                    &publish::Gh,
                    input,
                    Path::new(get("downloaded-iso")?),
                    get("observed-revision")?,
                    flags.get_flag("smoke-passed"),
                    flags.get_one::<String>("platform-evidence").map(Path::new),
                )
                .map_err(|error| error.to_string())?;
            }
            println!("Required qualification receipts validated; use status before promotion.");
            Ok(())
        }
        "qualify-upgrade" => soda_release_deliver::upgrade::stamp(
            Path::new(get("candidate")?),
            Path::new(get("evidence")?),
        )
        .map(|_| ())
        .map_err(|e| e.to_string()),
        "prepare" => {
            let manifest = publish::prepare(
                Path::new(get("candidate")?),
                get("tag")?,
                crate::release_start::distribution(get("distribution")?)?,
                Path::new(get("notes")?),
                Path::new(get("out")?),
            )
            .map_err(|e| e.to_string())?;
            println!(
                "Prepared {} at {}; no remote publication.",
                manifest.tag,
                get("out")?
            );
            Ok(())
        }
        _ => Err("unknown release action".to_owned()),
    }
}
fn qualification_inputs(
    flags: &clap::ArgMatches,
    distribution: soda_release_deliver::model::Distribution,
) -> Result<(), String> {
    use soda_release_deliver::model::Distribution;
    let mut required = Vec::new();
    if distribution != Distribution::Appliance {
        required.push("evidence");
    }
    if distribution != Distribution::Docker {
        required.extend(["downloaded-iso", "observed-revision"]);
    }
    for name in required {
        if flags
            .get_one::<String>(name)
            .is_none_or(|value| value.is_empty())
        {
            return Err(format!(
                "--{name} required for the selected distribution before qualification"
            ));
        }
    }
    if distribution != Distribution::Docker && !flags.get_flag("smoke-passed") {
        return Err("--smoke-passed required after the actual appliance installation smoke".into());
    }
    Ok(())
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
    fn commands_require_explicit_inputs() {
        assert!(run(["soda-release", "--help"].map(str::to_owned)).is_ok());
        for action in [
            "prepare",
            "stage",
            "promote",
            "qualify",
            "status",
            "verify-docker",
        ] {
            assert!(run(["soda-release", action].map(str::to_owned)).is_err());
        }
        for action in ["stage", "promote", "qualify", "status", "verify-docker"] {
            assert!(command()
                .try_get_matches_from(["soda-release", action, "--input", "prepared"])
                .is_ok());
        }
        assert!(command()
            .try_get_matches_from(["soda-release", "stage", "--candidate", "candidate"])
            .is_err());
        assert!(command()
            .try_get_matches_from(["soda-release", "stage-docker"])
            .is_err());
    }

    #[test]
    fn all_qualification_requires_both_targets_before_stamping_either() {
        use soda_release_deliver::model::Distribution;
        let parsed = command()
            .try_get_matches_from([
                "soda-release",
                "qualify",
                "--input",
                "prepared",
                "--evidence",
                "observations",
            ])
            .unwrap();
        let (_, flags) = parsed.subcommand().unwrap();
        assert!(qualification_inputs(flags, Distribution::Docker).is_ok());
        assert!(qualification_inputs(flags, Distribution::All)
            .unwrap_err()
            .contains("--downloaded-iso required"));
        let parsed = command()
            .try_get_matches_from([
                "soda-release",
                "qualify",
                "--input",
                "prepared",
                "--evidence",
                "observations",
                "--downloaded-iso",
                "download.iso",
                "--observed-revision",
                "revision",
                "--smoke-passed",
            ])
            .unwrap();
        let (_, flags) = parsed.subcommand().unwrap();
        assert!(qualification_inputs(flags, Distribution::All).is_ok());
    }
}

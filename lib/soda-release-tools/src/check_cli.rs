//! `soda-candidate-check` (Go `tools/soda-candidate-check` `main.go`):
//! verify one delivered candidate directory through the release
//! validation owner. It never builds, installs, publishes or changes
//! trust.

use clap::{Arg, ArgAction, Command};

const HELP_REQUESTED: &str = "candidate check help requested";

fn command() -> Command {
    Command::new("soda-candidate-check")
        .args_override_self(true)
        .arg(
            Arg::new("candidate")
                .long("candidate")
                .help("delivered candidate artifacts directory")
                .action(ArgAction::Set)
                .num_args(1)
                .allow_hyphen_values(true),
        )
        .arg(
            Arg::new("arch")
                .long("arch")
                .help("requested native architecture")
                .action(ArgAction::Set)
                .num_args(1)
                .allow_hyphen_values(true),
        )
        .arg(
            Arg::new("soda-revision")
                .long("soda-revision")
                .help("requested Soda source revision")
                .action(ArgAction::Set)
                .num_args(1)
                .allow_hyphen_values(true),
        )
        .arg(
            Arg::new("forgejo-revision")
                .long("forgejo-revision")
                .help("requested Fountain source revision")
                .action(ArgAction::Set)
                .num_args(1)
                .allow_hyphen_values(true),
        )
}

fn help_text() -> String {
    command().render_help().to_string()
}

#[derive(Debug, Clone, Default)]
pub struct CheckFlags {
    pub candidate: String,
    pub arch: String,
    pub soda_revision: String,
    pub forgejo_revision: String,
}

pub fn parse_check_flags(args: &[String]) -> Result<CheckFlags, String> {
    let matches = command()
        .try_get_matches_from(
            std::iter::once("soda-candidate-check").chain(args.iter().map(String::as_str)),
        )
        .map_err(|error| {
            if error.kind() == clap::error::ErrorKind::DisplayHelp {
                HELP_REQUESTED.to_owned()
            } else {
                "invalid candidate check flags".to_owned()
            }
        })?;
    let flags = CheckFlags {
        candidate: matches
            .get_one::<String>("candidate")
            .cloned()
            .unwrap_or_default(),
        arch: matches
            .get_one::<String>("arch")
            .cloned()
            .unwrap_or_default(),
        soda_revision: matches
            .get_one::<String>("soda-revision")
            .cloned()
            .unwrap_or_default(),
        forgejo_revision: matches
            .get_one::<String>("forgejo-revision")
            .cloned()
            .unwrap_or_default(),
    };
    if flags.candidate.is_empty()
        || flags.arch.is_empty()
        || flags.soda_revision.is_empty()
        || flags.forgejo_revision.is_empty()
    {
        return Err("candidate, arch, soda-revision and forgejo-revision are required".to_owned());
    }
    Ok(flags)
}

pub fn run(args: &[String]) -> Result<(), String> {
    let flags = match parse_check_flags(args) {
        Ok(flags) => flags,
        Err(e) if e == HELP_REQUESTED => {
            print!("{}", help_text());
            return Ok(());
        }
        Err(e) => return Err(e),
    };
    soda_release_deliver::check::check_candidate(
        &flags.candidate,
        &flags.arch,
        &flags.soda_revision,
        &flags.forgejo_revision,
    )
    .map_err(|e| e.to_string())
}

pub fn main() {
    // Signal disposition matches the Go owner (default terminate); no
    // handler is installed.
    let argv: Vec<String> = std::env::args().collect();
    let args = if argv.len() > 1 { &argv[1..] } else { &[] };
    if let Err(e) = run(args) {
        eprintln!("{e}");
        std::process::exit(1);
    }
}

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

#[cfg(test)]
mod tests {
    use super::*;

    fn parse_list(args: &[&str]) -> Result<CheckFlags, String> {
        parse_check_flags(&args.iter().map(|s| s.to_string()).collect::<Vec<_>>())
    }

    fn run_list(args: &[&str]) -> Result<(), String> {
        run(&args.iter().map(|s| s.to_string()).collect::<Vec<_>>())
    }

    #[test]
    fn flag_parse_matrix() {
        let full = [
            "--candidate",
            "/c",
            "--arch",
            "x86_64",
            "--soda-revision",
            "s",
            "--forgejo-revision",
            "f",
        ];
        let flags = parse_list(&full).unwrap();
        assert_eq!(flags.candidate, "/c");
        assert_eq!(flags.arch, "x86_64");
        assert_eq!(flags.soda_revision, "s");
        assert_eq!(flags.forgejo_revision, "f");
        // Equals form also parses.
        let flags = parse_list(&[
            "--candidate=/c",
            "--arch=x86_64",
            "--soda-revision=s",
            "--forgejo-revision=f",
        ])
        .unwrap();
        assert_eq!(flags.forgejo_revision, "f");
        // Each missing flag refuses with the required-flags message.
        for missing in 0..4 {
            let mut args: Vec<&str> = Vec::new();
            for (i, pair) in full.chunks(2).enumerate() {
                if i != missing {
                    args.push(pair[0]);
                    args.push(pair[1]);
                }
            }
            assert_eq!(
                parse_list(&args).unwrap_err(),
                "candidate, arch, soda-revision and forgejo-revision are required"
            );
        }
        assert_eq!(
            parse_list(&[]).unwrap_err(),
            "candidate, arch, soda-revision and forgejo-revision are required"
        );
        // Positionals refuse.
        let mut positional = full.to_vec();
        positional.push("extra");
        assert_eq!(
            parse_list(&positional).unwrap_err(),
            "invalid candidate check flags"
        );
        // Unknown flag refuses.
        let mut unknown = full.to_vec();
        unknown.push("--bogus");
        unknown.push("x");
        assert_eq!(
            parse_list(&unknown).unwrap_err(),
            "invalid candidate check flags"
        );
        assert_eq!(parse_list(&["-h"]).unwrap_err(), HELP_REQUESTED);
        assert_eq!(parse_list(&["--help"]).unwrap_err(), HELP_REQUESTED);
        assert!(help_text().contains("--forgejo-revision"));
        assert!(run_list(&["--help"]).is_ok());
    }

    #[test]
    fn nonexistent_candidate_dir_errors() {
        let err = run_list(&[
            "--candidate",
            "/nonexistent-candidate-xyz",
            "--arch",
            "x86_64",
            "--soda-revision",
            "s",
            "--forgejo-revision",
            "f",
        ])
        .unwrap_err();
        assert!(!err.is_empty(), "expected a check error, got empty");
    }
}

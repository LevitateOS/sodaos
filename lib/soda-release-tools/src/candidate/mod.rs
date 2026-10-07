//! `soda-candidate` wrapper core (Go `tools/soda-candidate` `main.go`):
//! options, flag parsing, answer validation, and checkout preflight.

use crate::candidate_controller::{file_built_rootfs, maybe_serve_fixture, start_controller_run};
use crate::candidate_display::is_terminal;
use crate::candidate_prompts::{prompter_overview, suggest_out};

pub mod options;

pub use options::*;

/// Exit-code-carrying controller failure.
#[derive(Debug, Clone)]
pub struct ExitError {
    pub code: i32,
}

impl std::fmt::Display for ExitError {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        if self.code == 2 {
            write!(
                f,
                "controller incomplete: no qualified release without final signing"
            )
        } else {
            write!(f, "controller exited {}", self.code)
        }
    }
}

pub fn run_candidate() -> Result<(), CandidateError> {
    let argv: Vec<String> = std::env::args().collect();
    let args = if argv.len() > 1 { &argv[1..] } else { &[] };
    if let Err(e) = parse_options(args) {
        if e == "candidate help requested" {
            print!("{}", usage());
            return Ok(());
        }
        if e == "invalid candidate command flags" {
            eprintln!("{e}\n{}", usage());
        }
        return Err(CandidateError::Message(e));
    }
    let stdin_tty = is_terminal(libc::STDIN_FILENO);
    let stderr_tty = is_terminal(libc::STDERR_FILENO);
    let mut prompt = |o: &mut Options| prompter_overview(o, suggest_out);
    let mut o = resolve_options(args, stdin_tty, stderr_tty, &mut prompt)
        .map_err(CandidateError::Message)?;
    validate_resolved(&o).map_err(CandidateError::Message)?;
    ready_run(&mut o).map_err(CandidateError::Message)?;
    let stop = maybe_serve_fixture(&o).map_err(CandidateError::Message)?;
    let controller = start_controller_run(&o).map_err(CandidateError::Message)?;
    let code = controller.wait().map_err(CandidateError::Message)?;
    stop();
    if code != 0 {
        return Err(CandidateError::Exit(ExitError { code }));
    }
    let mut stderr = std::io::stderr().lock();
    file_built_rootfs(&o, &mut stderr).map_err(CandidateError::Message)
}

#[derive(Debug)]
pub enum CandidateError {
    Message(String),
    Exit(ExitError),
}

impl std::fmt::Display for CandidateError {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        match self {
            CandidateError::Message(m) => write!(f, "{m}"),
            CandidateError::Exit(e) => write!(f, "{e}"),
        }
    }
}

pub fn main() {
    if let Err(err) = run_candidate() {
        eprintln!("soda-candidate: {err}");
        match err {
            CandidateError::Exit(e) => std::process::exit(e.code),
            CandidateError::Message(_) => std::process::exit(1),
        }
    }
}

#[cfg(test)]
pub mod tests;

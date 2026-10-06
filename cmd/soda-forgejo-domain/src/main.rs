//! Whole-domain stop and restart-inhibition controls for native recovery.
//!
//! Host-operator controls for the offline native-mutation recovery procedure:
//! stop every native writer, inhibit restart, reconcile via Forgejo's own
//! `admin native-operation` commands, lift inhibition, restart. There is no
//! force-unlock verb: a fenced reservation stays held for intervention.
//!
//! Verbs: stop, inhibit, status, lift, start. Run as root.

mod cli;
mod config;
mod domain;
mod system;
#[cfg(test)]
mod tests;

use std::io::{self, Write};

use crate::cli::{help_text, parse_args, usage};
use crate::domain::dispatch;
use crate::system::{Paths, RealSys};

#[link(name = "c")]
extern "C" {
    fn geteuid() -> u32;
}

fn main() {
    let argv: Vec<String> = std::env::args().collect();
    let prog = argv
        .first()
        .map(|a| a.rsplit('/').next().unwrap_or("soda-forgejo-domain"))
        .unwrap_or("soda-forgejo-domain");
    let prog = if prog.is_empty() {
        "soda-forgejo-domain"
    } else {
        prog
    };
    let verb = match parse_args(prog, &argv[1..]) {
        Ok(None) => {
            print!("{}", help_text(prog));
            let _ = io::stdout().flush();
            std::process::exit(0);
        }
        Ok(Some(verb)) => verb,
        Err(msg) => {
            eprintln!("{}\n{prog}: error: {msg}", usage(prog));
            std::process::exit(2);
        }
    };
    if unsafe { geteuid() } != 0 {
        eprintln!(
            "{}\n{prog}: error: native host operator/root required",
            usage(prog)
        );
        std::process::exit(2);
    }
    let paths = Paths::production();
    let mut sys = RealSys;
    let mut stdout = io::stdout();
    if let Err(msg) = dispatch(&verb, &paths, &mut sys, &mut stdout) {
        eprintln!("soda-forgejo-domain: {msg}");
        std::process::exit(1);
    }
}

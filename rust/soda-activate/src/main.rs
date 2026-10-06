//! Operator-only activation after native Forgejo setup, with explicit private TLS.

mod activation;
mod cli;
mod forgejo_env;
mod origin;
mod system;
#[cfg(test)]
mod tests;

use std::io::{self, Write};

use crate::activation::activate;
use crate::cli::parse_args;
use crate::system::{ActivateError, Paths, RealSys};

const USAGE: &str = "usage: soda-activate [-h] --bind-ip BIND_IP [--certificate CERTIFICATE] [--private-key PRIVATE_KEY] [--local-tls]";

fn help_text(prog: &str) -> String {
    format!(
        "usage: {prog} [-h] --bind-ip BIND_IP [--certificate CERTIFICATE] [--private-key PRIVATE_KEY] [--local-tls]\n\noptions:\n  -h, --help            show this help message and exit\n  --bind-ip BIND_IP     explicit private appliance address\n  --certificate CERTIFICATE\n                        PEM certificate covering the Forgejo/Sodaspaces browser origin\n  --private-key PRIVATE_KEY\n                        PEM private key; never committed\n  --local-tls           use Caddy local certificates for the selected private IP; client trust is explicit\n"
    )
}

fn main() {
    let argv: Vec<String> = std::env::args().collect();
    let prog = argv
        .first()
        .map(|a| a.rsplit('/').next().unwrap_or("soda-activate"))
        .unwrap_or("soda-activate");
    let prog = if prog.is_empty() {
        "soda-activate"
    } else {
        prog
    };
    let args = match parse_args(prog, &argv[1..]) {
        Ok(None) => {
            print!("{}", help_text(prog));
            let _ = io::stdout().flush();
            std::process::exit(0);
        }
        Ok(Some(args)) => args,
        Err(msg) => {
            eprintln!("{USAGE}\n{prog}: error: {msg}");
            std::process::exit(2);
        }
    };
    let paths = Paths::production();
    let mut sys = RealSys;
    let mut stdout = io::stdout();
    match activate(&args, &paths, &mut sys, &mut stdout) {
        Ok(()) => {}
        Err(ActivateError::Usage(msg)) => {
            eprintln!("{USAGE}\n{prog}: error: {msg}");
            std::process::exit(2);
        }
        Err(ActivateError::Runtime(msg)) => {
            eprintln!("{prog}: {msg}");
            std::process::exit(1);
        }
    }
}

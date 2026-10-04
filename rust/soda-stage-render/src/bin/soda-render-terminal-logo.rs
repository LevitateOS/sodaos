// soda-render-terminal-logo samples the canonical polygon emblem into a
// 32-column, 16-row ASCII mark (ports scripts/render-terminal-logo.py).
use std::ffi::OsString;

use soda_stage_render::{source_root, split_flag, terminal_logo};

const PROG: &str = "soda-render-terminal-logo";
const USAGE: &str = "usage: soda-render-terminal-logo [--check]";
const HELP: &str = "usage: soda-render-terminal-logo [--check]\n\nSample the canonical polygon emblem into a 32-column, 16-row ASCII mark.\n\noptions:\n  --check  verify the committed outputs instead of rewriting them\n";

fn refusal(message: String) -> ! {
    eprintln!("{USAGE}\n{PROG}: error: {message}");
    std::process::exit(2);
}

fn parse(args: &[OsString]) -> bool {
    let mut check = false;
    let mut stray: Vec<String> = Vec::new();
    let mut i = 0;
    let mut only_positionals = false;
    while i < args.len() {
        let raw = args[i].to_string_lossy().into_owned();
        if !only_positionals && raw == "--" {
            only_positionals = true;
            i += 1;
            continue;
        }
        let token = if only_positionals {
            None
        } else {
            split_flag(&raw)
        };
        match token {
            None => {
                stray.push(raw);
                i += 1;
            }
            Some((name, inline)) => {
                if name == "h" || name == "help" {
                    if let Some(explicit) = inline {
                        refusal(format!(
                            "argument --help: ignored explicit argument '{explicit}'"
                        ));
                    }
                    print!("{HELP}");
                    std::process::exit(0);
                }
                if name == "check" {
                    if let Some(explicit) = inline {
                        refusal(format!(
                            "argument --check: ignored explicit argument '{explicit}'"
                        ));
                    }
                    check = true;
                    i += 1;
                    continue;
                }
                stray.push(raw);
                i += 1;
            }
        }
    }
    if !stray.is_empty() {
        refusal(format!("unrecognized arguments: {}", stray.join(" ")));
    }
    check
}

fn main() {
    let args: Vec<OsString> = std::env::args_os().skip(1).collect();
    let check = parse(&args);
    let source = match source_root() {
        Ok(source) => source,
        Err(message) => {
            eprintln!("{message}");
            std::process::exit(1);
        }
    };
    if let Err(message) = terminal_logo::run(&source, check) {
        eprintln!("{message}");
        std::process::exit(1);
    }
}

// soda-stage stages existing native build outputs plus configuration; it
// does not build or install (ports scripts/stage.py).
use std::ffi::OsString;
use std::path::PathBuf;

use soda_stage_render::{source_root, split_flag, stage};

const PROG: &str = "soda-stage";
const USAGE: &str = "usage: soda-stage --arch x86_64 --host-context DIR --forgejo-context DIR";
const HELP: &str = "usage: soda-stage --arch x86_64 --host-context DIR --forgejo-context DIR\n\nStage existing native build outputs plus configuration; does not build or install.\n\noptions:\n  --arch x86_64        native architecture (only x86_64)\n  --host-context DIR     prepared host context holding rootfs/\n  --forgejo-context DIR  fresh Forgejo context for the presentation\n";

struct Args {
    arch: String,
    host_context: PathBuf,
    forgejo_context: PathBuf,
}

fn refusal(message: String) -> ! {
    eprintln!("{USAGE}\n{PROG}: error: {message}");
    std::process::exit(2);
}

fn parse(args: &[OsString]) -> Args {
    let mut arch: Option<String> = None;
    let mut host_context: Option<PathBuf> = None;
    let mut forgejo_context: Option<PathBuf> = None;
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
                // Unknown flags never consume a value: argparse cannot know
                // their arity, so the following token stays unrecognized too.
                if name != "arch" && name != "host-context" && name != "forgejo-context" {
                    stray.push(raw);
                    i += 1;
                    continue;
                }
                let value = match inline {
                    Some(value) => OsString::from(value),
                    None => {
                        i += 1;
                        match args.get(i) {
                            Some(value) => value.clone(),
                            None => refusal(format!("argument --{name}: expected one argument")),
                        }
                    }
                };
                match name.as_str() {
                    // Non-UTF-8 can never equal x86_64; it falls out below
                    // as an invalid choice, like argparse's failed test.
                    "arch" => arch = Some(value.to_string_lossy().into_owned()),
                    "host-context" => host_context = Some(PathBuf::from(value)),
                    "forgejo-context" => forgejo_context = Some(PathBuf::from(value)),
                    _ => {
                        stray.push(raw);
                    }
                }
                i += 1;
            }
        }
    }
    if !stray.is_empty() {
        refusal(format!("unrecognized arguments: {}", stray.join(" ")));
    }
    let mut missing = Vec::new();
    if arch.is_none() {
        missing.push("--arch");
    }
    if host_context.is_none() {
        missing.push("--host-context");
    }
    if forgejo_context.is_none() {
        missing.push("--forgejo-context");
    }
    if !missing.is_empty() {
        refusal(format!(
            "the following arguments are required: {}",
            missing.join(", ")
        ));
    }
    let arch = arch.unwrap();
    if arch != "x86_64" {
        refusal(format!(
            "argument --arch: invalid choice: '{arch}' (choose from x86_64)"
        ));
    }
    Args {
        arch,
        host_context: host_context.unwrap(),
        forgejo_context: forgejo_context.unwrap(),
    }
}

fn main() {
    let args: Vec<OsString> = std::env::args_os().skip(1).collect();
    let parsed = parse(&args);
    if let Err(message) = stage::check_platform(&parsed.arch) {
        refusal(message);
    }
    let (stage_dir, forgejo) =
        match stage::check_contexts(&parsed.host_context, &parsed.forgejo_context) {
            Ok(pair) => pair,
            Err(stage::StageError::Refusal(message)) => refusal(message),
            Err(stage::StageError::Failure(message)) => {
                eprintln!("{message}");
                std::process::exit(1);
            }
        };
    let source = match source_root() {
        Ok(source) => source,
        Err(message) => {
            eprintln!("{message}");
            std::process::exit(1);
        }
    };
    match stage::run(&source, &parsed.arch, &stage_dir, &forgejo) {
        Ok(()) => {}
        Err(stage::StageError::Refusal(message)) => refusal(message),
        Err(stage::StageError::Failure(message)) => {
            eprintln!("{message}");
            std::process::exit(1);
        }
    }
    #[cfg(unix)]
    {
        use std::io::Write;
        use std::os::unix::ffi::OsStrExt;
        let mut stdout = std::io::stdout().lock();
        let _ = stdout.write_all(stage_dir.as_os_str().as_bytes());
        let _ = stdout.write_all(b"\n");
    }
    #[cfg(not(unix))]
    {
        println!("{}", stage_dir.display());
    }
}

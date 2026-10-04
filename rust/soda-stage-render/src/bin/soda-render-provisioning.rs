// soda-render-provisioning merges the public Butane bootstrap with private
// per-instance operator inputs (ports scripts/render-provisioning.py).
use std::ffi::OsString;
use std::path::PathBuf;

use soda_stage_render::{provisioning, source_root, split_flag};

const PROG: &str = "soda-render-provisioning";
const USAGE: &str = "usage: soda-render-provisioning --operator-key-file PATH --root-password-hash-file PATH [--hostname NAME | --appliance-hostname NAME] [--bootstrap {minimal,extensions}] [--ssh-host-key-file PATH] --out PATH";
const HELP: &str = "usage: soda-render-provisioning --operator-key-file PATH --root-password-hash-file PATH [--hostname NAME | --appliance-hostname NAME] [--bootstrap {minimal,extensions}] [--ssh-host-key-file PATH] --out PATH\n\nMerge public Butane bootstrap with private per-instance operator inputs.\nNo conversion, installation, account enrollment or reboot is implicit.\n\noptions:\n  --operator-key-file PATH       public SSH key input\n  --root-password-hash-file PATH   crypt(3) hash input (mode 0600)\n  --hostname NAME                  fixture-only soda-native-* name\n  --appliance-hostname NAME        product hostname (default: unchanged CoreOS behavior)\n  --bootstrap {minimal,extensions} bootstrap profile (default: extensions)\n  --ssh-host-key-file PATH         per-instance Ed25519 host key (mode 0600)\n  --out PATH                       absolute output under a real private parent\n";

struct Args {
    operator_key: PathBuf,
    password_hash: PathBuf,
    hostname: Option<String>,
    appliance_hostname: Option<String>,
    bootstrap: String,
    host_key: Option<PathBuf>,
    out: PathBuf,
}

fn refusal(message: String) -> ! {
    eprintln!("{USAGE}\n{PROG}: error: {message}");
    std::process::exit(2);
}

fn parse(args: &[OsString]) -> Args {
    let mut operator_key: Option<PathBuf> = None;
    let mut password_hash: Option<PathBuf> = None;
    let mut hostname: Option<String> = None;
    let mut appliance_hostname: Option<String> = None;
    let mut bootstrap = "extensions".to_string();
    let mut host_key: Option<PathBuf> = None;
    let mut out: Option<PathBuf> = None;
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
                if !matches!(
                    name.as_str(),
                    "operator-key-file"
                        | "root-password-hash-file"
                        | "hostname"
                        | "appliance-hostname"
                        | "bootstrap"
                        | "ssh-host-key-file"
                        | "out"
                ) {
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
                    "operator-key-file" => operator_key = Some(PathBuf::from(value)),
                    "root-password-hash-file" => password_hash = Some(PathBuf::from(value)),
                    "hostname" => hostname = Some(value.to_string_lossy().into_owned()),
                    "appliance-hostname" => {
                        appliance_hostname = Some(value.to_string_lossy().into_owned())
                    }
                    "bootstrap" => bootstrap = value.to_string_lossy().into_owned(),
                    "ssh-host-key-file" => host_key = Some(PathBuf::from(value)),
                    "out" => out = Some(PathBuf::from(value)),
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
    if operator_key.is_none() {
        missing.push("--operator-key-file");
    }
    if password_hash.is_none() {
        missing.push("--root-password-hash-file");
    }
    if out.is_none() {
        missing.push("--out");
    }
    if !missing.is_empty() {
        refusal(format!(
            "the following arguments are required: {}",
            missing.join(", ")
        ));
    }
    if hostname.is_some() && appliance_hostname.is_some() {
        refusal("argument --appliance-hostname: not allowed with argument --hostname".to_string());
    }
    if bootstrap != "minimal" && bootstrap != "extensions" {
        refusal(format!(
            "argument --bootstrap: invalid choice: '{bootstrap}' (choose from minimal, extensions)"
        ));
    }
    Args {
        operator_key: operator_key.unwrap(),
        password_hash: password_hash.unwrap(),
        hostname,
        appliance_hostname,
        bootstrap,
        host_key,
        out: out.unwrap(),
    }
}

fn main() {
    let args: Vec<OsString> = std::env::args_os().skip(1).collect();
    let parsed = parse(&args);
    let source = match source_root() {
        Ok(source) => source,
        Err(message) => {
            eprintln!("{message}");
            std::process::exit(1);
        }
    };
    let inputs = provisioning::RenderInputs {
        operator_key: &parsed.operator_key,
        password_hash: &parsed.password_hash,
        out: &parsed.out,
        hostname: parsed.hostname.as_deref(),
        host_key: parsed.host_key.as_deref(),
        bootstrap: &parsed.bootstrap,
        product_hostname: parsed.appliance_hostname.as_deref(),
    };
    if let Err(error) = provisioning::render(&source, &inputs) {
        // No input values or subprocess diagnostics: either can contain secrets.
        eprintln!(
            "Private provisioning failed ({}); check paths/modes/key format.",
            error.kind.name()
        );
        std::process::exit(1);
    }
}

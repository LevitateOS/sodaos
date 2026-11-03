// soda-factory is the private operator interface to supervised factory runs:
// status reads recorded state, stop retires one run, and reconcile settles
// outstanding runs. It addresses the standalone Factory service's operator endpoint
// over a restricted Unix socket and keeps no database of its own.

use std::collections::BTreeMap;
use std::env;
use std::io::{self, Write};
use std::path::Path;
use std::time::Duration;

const USAGE: &str = "usage: soda-factory --socket PATH status [RUN] / stop RUN / reconcile";
const OPERATOR_PATH: &str = "/operator/factory";
const OPERATOR_HOST: &str = "soda-operator";
const RESPONSE_LIMIT: usize = 256 * 1024;
const CLIENT_TIMEOUT: Duration = Duration::from_secs(11 * 60);

fn main() {
    let args: Vec<String> = env::args().skip(1).collect();
    let mut out = io::stdout();
    if let Err(err) = run(&args, &mut out) {
        let _ = out.flush();
        eprintln!("soda-factory: {err}");
        std::process::exit(1);
    }
}

fn run(args: &[String], out: &mut dyn Write) -> Result<(), String> {
    let parsed = parse_args(args)?;
    dispatch(&parsed.socket, &parsed.positional, out)
}

struct ParsedArgs {
    socket: String,
    positional: Vec<String>,
}

fn print_usage() {
    eprint!(
        "Usage of soda-factory:\n  -socket string\n    \tprivate backend operator Unix socket\n"
    );
}

fn parse_args(args: &[String]) -> Result<ParsedArgs, String> {
    let mut socket = String::new();
    let mut positional: Vec<String> = Vec::new();
    let mut i = 0;
    let mut end_of_flags = false;
    while i < args.len() {
        let arg = &args[i];
        if end_of_flags || !arg.starts_with('-') || arg == "-" {
            positional.push(arg.clone());
            i += 1;
            continue;
        }
        if arg == "--" {
            end_of_flags = true;
            i += 1;
            continue;
        }
        let flag = arg.trim_start_matches('-');
        let (name, inline) = match flag.split_once('=') {
            Some((n, v)) => (n, Some(v)),
            None => (flag, None),
        };
        if name == "h" || name == "help" {
            print_usage();
            return Err("flag: help requested".to_string());
        }
        let value = match inline {
            Some(v) => v.to_string(),
            None => {
                i += 1;
                if i >= args.len() {
                    let msg = format!("flag needs an argument: -{name}");
                    eprintln!("{msg}");
                    print_usage();
                    return Err(msg);
                }
                args[i].clone()
            }
        };
        match name {
            "socket" => socket = value,
            _ => {
                let msg = format!("flag provided but not defined: -{name}");
                eprintln!("{msg}");
                print_usage();
                return Err(msg);
            }
        }
        i += 1;
    }
    if socket.is_empty() {
        return Err(USAGE.to_string());
    }
    Ok(ParsedArgs { socket, positional })
}

fn dispatch(socket: &str, args: &[String], out: &mut dyn Write) -> Result<(), String> {
    if args.is_empty() {
        return Err(USAGE.to_string());
    }
    let mut envelope: Vec<(&str, &str)> = Vec::new();
    match args[0].as_str() {
        "status" => {
            if args.len() > 2 || (args.len() == 2 && args[1].is_empty()) {
                return Err("status takes an optional run identity".to_string());
            }
            envelope.push(("type", "status"));
            if args.len() == 2 {
                envelope.push(("target", &args[1]));
            }
        }
        "stop" => {
            if args.len() != 2 {
                return Err("stop requires one recorded run".to_string());
            }
            envelope.push(("type", "stop"));
            envelope.push(("target", &args[1]));
        }
        "reconcile" => {
            if args.len() != 1 {
                return Err("reconcile takes no target".to_string());
            }
            envelope.push(("type", "reconcile"));
        }
        _ => return Err("unknown factory command".to_string()),
    }
    send(socket, &envelope, out)
}

fn send(socket: &str, envelope: &[(&str, &str)], out: &mut dyn Write) -> Result<(), String> {
    let object: BTreeMap<_, _> = envelope.iter().copied().collect();
    let body = serde_json::to_vec(&object).expect("string envelope serializes");
    let response = soda_unix_http::request(
        Path::new(socket),
        "POST",
        OPERATOR_PATH,
        OPERATOR_HOST,
        &body,
        CLIENT_TIMEOUT,
        soda_unix_http::Limits {
            header_bytes: 65536,
            body_bytes: RESPONSE_LIMIT,
        },
    )
    .map_err(|error| match error {
        soda_unix_http::Error::BodyTooLarge => "operator response exceeds limit",
        _ => "operator endpoint unavailable",
    })?;
    let status = response.status;
    let data = response.body;
    match status {
        200 => {
            out.write_all(&data)
                .map_err(|e| format!("write response: {e}"))?;
            if data.is_empty() || data[data.len() - 1] != b'\n' {
                out.write_all(b"\n")
                    .map_err(|e| format!("write response: {e}"))?;
            }
            Ok(())
        }
        404 => Err("factory run not found".to_string()),
        code => Err(format!("operator command failed (HTTP {code})")),
    }
}

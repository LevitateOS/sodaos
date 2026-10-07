// soda-factory is the private operator interface to supervised factory runs:
// status reads recorded state, stop retires one run, and reconcile settles
// outstanding runs. It addresses the dashboard backend's operator endpoint
// over a restricted Unix socket and keeps no database of its own.

use std::env;
use std::io::{self, Read, Write};
use std::os::unix::net::UnixStream;
use std::time::Duration;

const USAGE: &str =
    "usage: soda-factory --socket PATH [--command ID] status [RUN] / stop RUN / reconcile";
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
    dispatch(&parsed.socket, &parsed.command, &parsed.positional, out)
}

struct ParsedArgs {
    socket: String,
    command: String,
    positional: Vec<String>,
}

fn print_usage() {
    eprint!(
        "Usage of soda-factory:\n  -command string\n    \tclient-generated command identity for stop and reconcile\n  -socket string\n    \tprivate backend operator Unix socket\n"
    );
}

fn parse_args(args: &[String]) -> Result<ParsedArgs, String> {
    let mut socket = String::new();
    let mut command = String::new();
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
            "command" => command = value,
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
    Ok(ParsedArgs {
        socket,
        command,
        positional,
    })
}

fn dispatch(
    socket: &str,
    command: &str,
    args: &[String],
    out: &mut dyn Write,
) -> Result<(), String> {
    if args.is_empty() {
        return Err(USAGE.to_string());
    }
    let mut envelope: Vec<(&str, &str)> = Vec::new();
    match args[0].as_str() {
        "status" => {
            if args.len() > 2 || (args.len() == 2 && args[1].is_empty()) {
                return Err("status takes an optional run identity".to_string());
            }
            if !command.is_empty() {
                return Err("status carries no durable command".to_string());
            }
            envelope.push(("type", "status"));
            if args.len() == 2 {
                envelope.push(("target", &args[1]));
            }
        }
        "stop" => {
            if args.len() != 2 || command.is_empty() {
                return Err("stop requires --command ID and one recorded run".to_string());
            }
            envelope.push(("type", "stop"));
            envelope.push(("command_id", command));
            envelope.push(("target", &args[1]));
        }
        "reconcile" => {
            if args.len() != 1 || command.is_empty() {
                return Err("reconcile requires --command ID and no target".to_string());
            }
            envelope.push(("type", "reconcile"));
            envelope.push(("command_id", command));
        }
        _ => return Err("unknown factory command".to_string()),
    }
    send(socket, &envelope, out)
}

/// Encode the fixed string envelope with Go-compatible sorted keys and escapes.
fn encode_envelope(envelope: &[(&str, &str)]) -> Vec<u8> {
    use serde::Serialize;
    use serde_json::ser::{CharEscape, CompactFormatter, Formatter, Serializer};
    use std::collections::BTreeMap;
    use std::io;

    struct GoFormatter;
    impl Formatter for GoFormatter {
        fn write_string_fragment<W: ?Sized + io::Write>(
            &mut self,
            writer: &mut W,
            fragment: &str,
        ) -> io::Result<()> {
            let mut start = 0;
            for (index, ch) in fragment.char_indices() {
                let escaped = match ch {
                    '<' => Some("\\u003c"),
                    '>' => Some("\\u003e"),
                    '&' => Some("\\u0026"),
                    '\u{2028}' => Some("\\u2028"),
                    '\u{2029}' => Some("\\u2029"),
                    _ => None,
                };
                if let Some(escaped) = escaped {
                    writer.write_all(fragment[start..index].as_bytes())?;
                    writer.write_all(escaped.as_bytes())?;
                    start = index + ch.len_utf8();
                }
            }
            writer.write_all(fragment[start..].as_bytes())
        }

        fn write_char_escape<W: ?Sized + io::Write>(
            &mut self,
            writer: &mut W,
            escape: CharEscape,
        ) -> io::Result<()> {
            CompactFormatter.write_char_escape(writer, escape)
        }
    }

    let mut object = BTreeMap::new();
    for (key, value) in envelope {
        object.insert(*key, *value);
    }
    let mut body = Vec::new();
    let mut serializer = Serializer::with_formatter(&mut body, GoFormatter);
    object
        .serialize(&mut serializer)
        .expect("writing to Vec cannot fail");
    body.push(b'\n');
    body
}

fn send(socket: &str, envelope: &[(&str, &str)], out: &mut dyn Write) -> Result<(), String> {
    let body = encode_envelope(envelope);
    let mut stream = UnixStream::connect(socket).map_err(|_| "operator endpoint unavailable")?;
    stream
        .set_read_timeout(Some(CLIENT_TIMEOUT))
        .map_err(|_| "operator endpoint unavailable")?;
    stream
        .set_write_timeout(Some(CLIENT_TIMEOUT))
        .map_err(|_| "operator endpoint unavailable")?;
    let request = format!(
        "POST {} HTTP/1.1\r\nHost: {}\r\nContent-Type: application/json\r\nContent-Length: {}\r\nConnection: close\r\n\r\n",
        OPERATOR_PATH,
        OPERATOR_HOST,
        body.len()
    );
    stream
        .write_all(request.as_bytes())
        .map_err(|_| "operator endpoint unavailable")?;
    stream
        .write_all(&body)
        .map_err(|_| "operator endpoint unavailable")?;
    let (status, data) = read_response(&mut stream)?;
    match status {
        200 | 202 => {
            out.write_all(&data)
                .map_err(|e| format!("write response: {e}"))?;
            if data.is_empty() || data[data.len() - 1] != b'\n' {
                out.write_all(b"\n")
                    .map_err(|e| format!("write response: {e}"))?;
            }
            Ok(())
        }
        404 => Err("factory run not found".to_string()),
        409 => Err("command identity reused for different content".to_string()),
        code => Err(format!("operator command failed (HTTP {code})")),
    }
}

fn read_response(stream: &mut UnixStream) -> Result<(u16, Vec<u8>), String> {
    let mut raw: Vec<u8> = Vec::new();
    let mut byte = [0u8; 1];
    loop {
        match stream.read(&mut byte) {
            Ok(0) => return Err("operator endpoint unavailable".to_string()),
            Ok(_) => raw.push(byte[0]),
            Err(_) => return Err("operator endpoint unavailable".to_string()),
        }
        if raw.len() >= 4 && raw[raw.len() - 4..] == *b"\r\n\r\n" {
            break;
        }
    }
    let head = String::from_utf8(raw).map_err(|_| "operator endpoint unavailable")?;
    let mut lines = head.split("\r\n");
    let status_line = lines.next().ok_or("operator endpoint unavailable")?;
    let status: u16 = status_line
        .split_whitespace()
        .nth(1)
        .and_then(|code| code.parse().ok())
        .ok_or("operator endpoint unavailable")?;
    let mut content_length: Option<usize> = None;
    let mut chunked = false;
    for line in lines {
        if line.is_empty() {
            break;
        }
        let (name, value) = line
            .split_once(':')
            .ok_or("operator endpoint unavailable")?;
        if name.trim().eq_ignore_ascii_case("content-length") {
            content_length = value.trim().parse().ok();
        } else if name.trim().eq_ignore_ascii_case("transfer-encoding")
            && value.to_ascii_lowercase().contains("chunked")
        {
            chunked = true;
        }
    }
    let data = if chunked {
        read_chunked(stream)?
    } else if let Some(len) = content_length {
        read_exact_limited(stream, len)?
    } else {
        read_to_end_limited(stream)?
    };
    Ok((status, data))
}

fn read_exact_limited(stream: &mut UnixStream, len: usize) -> Result<Vec<u8>, String> {
    // Mirror Go's io.ReadAll(io.LimitReader(body, RESPONSE_LIMIT + 1)): never
    // allocate past the limit, refuse short bodies (Go surfaces those as a
    // body read error), and refuse anything over the limit.
    let want = len.min(RESPONSE_LIMIT + 1);
    let mut data = vec![0u8; want];
    stream
        .read_exact(&mut data)
        .map_err(|_| "operator response exceeds limit")?;
    if len > RESPONSE_LIMIT {
        return Err("operator response exceeds limit".to_string());
    }
    Ok(data)
}

fn read_to_end_limited(stream: &mut UnixStream) -> Result<Vec<u8>, String> {
    let mut data: Vec<u8> = Vec::new();
    let mut chunk = [0u8; 8192];
    loop {
        match stream.read(&mut chunk) {
            Ok(0) => break,
            Ok(n) => {
                data.extend_from_slice(&chunk[..n]);
                if data.len() > RESPONSE_LIMIT {
                    return Err("operator response exceeds limit".to_string());
                }
            }
            Err(_) => return Err("operator response exceeds limit".to_string()),
        }
    }
    Ok(data)
}

fn read_chunked(stream: &mut UnixStream) -> Result<Vec<u8>, String> {
    let mut data: Vec<u8> = Vec::new();
    loop {
        let line = read_line(stream)?;
        let size = usize::from_str_radix(line.split(';').next().unwrap_or("").trim(), 16)
            .map_err(|_| "operator response exceeds limit")?;
        if size == 0 {
            let _ = read_line(stream);
            break;
        }
        if data.len() + size > RESPONSE_LIMIT {
            return Err("operator response exceeds limit".to_string());
        }
        let mut chunk = vec![0u8; size];
        stream
            .read_exact(&mut chunk)
            .map_err(|_| "operator response exceeds limit")?;
        data.extend_from_slice(&chunk);
        let _ = read_line(stream);
    }
    Ok(data)
}

fn read_line(stream: &mut UnixStream) -> Result<String, String> {
    let mut line: Vec<u8> = Vec::new();
    let mut byte = [0u8; 1];
    loop {
        match stream.read(&mut byte) {
            Ok(0) => return Err("operator response exceeds limit".to_string()),
            Ok(_) => line.push(byte[0]),
            Err(_) => return Err("operator response exceeds limit".to_string()),
        }
        if line.len() >= 2 && line[line.len() - 2..] == *b"\r\n" {
            line.truncate(line.len() - 2);
            break;
        }
        if line.len() > 65536 {
            return Err("operator response exceeds limit".to_string());
        }
    }
    String::from_utf8(line).map_err(|_| "operator response exceeds limit".to_string())
}

#[cfg(test)]
mod operator_tests;

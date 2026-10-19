//! Stdio MCP transport for one already-admitted Factory worker.
//! Workflow and identity enforcement stay in the Go server. Credentials are
//! read from the root-staged run file; callers choose no endpoint or identity.

use serde::Deserialize;
use serde_json::Value;
use std::fs::OpenOptions;
use std::io::{BufRead, Read, Write};
use std::os::unix::fs::{MetadataExt, OpenOptionsExt};
use std::os::unix::net::UnixStream;
use std::path::{Path, PathBuf};
use std::time::Duration;

const SOCKET: &str = "/run/soda/factory-ipc/mcp.sock";
const MAX_REQUEST: usize = 96 * 1024;
const MAX_RESPONSE: usize = 256 * 1024;

#[derive(Deserialize)]
#[serde(deny_unknown_fields)]
struct Credential {
    capability: String,
    secret: String,
}

fn hex(value: &str, len: usize) -> bool {
    value.len() == len
        && value
            .bytes()
            .all(|b| b.is_ascii_digit() || (b'a'..=b'f').contains(&b))
}

fn credential_path(home: &str) -> Result<PathBuf, String> {
    let parts: Vec<_> = home.split('/').collect();
    if parts.len() != 9
        || parts[0] != ""
        || parts[1] != "home"
        || !matches!(parts[2], "soda-coder" | "soda-reviewer")
        || parts[3] != "checkouts"
        || !parts[4].starts_with('f')
        || !hex(&parts[4][1..], 24)
        || parts[5] != ".soda-home"
        || parts[6] != "runs"
        || !hex(parts[7], 32)
        || parts[8] != "home"
    {
        return Err("Factory MCP requires its staged run home".to_string());
    }
    Ok(Path::new(home).parent().unwrap().join("factory-mcp.json"))
}

fn read_credential(path: &Path) -> Result<Credential, String> {
    let file = OpenOptions::new()
        .read(true)
        .custom_flags(libc::O_NOFOLLOW | libc::O_NONBLOCK)
        .open(path)
        .map_err(|_| "Factory MCP credential is unavailable")?;
    let metadata = file
        .metadata()
        .map_err(|_| "Factory MCP credential cannot be inspected")?;
    if !metadata.is_file()
        || metadata.uid() != 0
        || metadata.nlink() != 1
        || metadata.mode() & 0o777 != 0o440
    {
        return Err("Factory MCP credential is not root-staged".to_string());
    }
    let mut data = Vec::new();
    file.take(4097)
        .read_to_end(&mut data)
        .map_err(|_| "Factory MCP credential cannot be read")?;
    if data.len() > 4096 {
        return Err("Factory MCP credential exceeds its bound".to_string());
    }
    let credential: Credential =
        serde_json::from_slice(&data).map_err(|_| "Factory MCP credential is invalid")?;
    if !hex(&credential.capability, 32) || !hex(&credential.secret, 64) {
        return Err("Factory MCP credential is invalid".to_string());
    }
    Ok(credential)
}

fn chunked_body(mut raw: &[u8]) -> Result<Vec<u8>, String> {
    let mut body = Vec::new();
    loop {
        let end = raw
            .windows(2)
            .position(|w| w == b"\r\n")
            .ok_or("Invalid MCP response framing")?;
        let size = std::str::from_utf8(&raw[..end])
            .map_err(|_| "Invalid MCP response framing")?
            .split(';')
            .next()
            .unwrap();
        let size = usize::from_str_radix(size, 16).map_err(|_| "Invalid MCP response framing")?;
        raw = &raw[end + 2..];
        if size == 0 {
            return Ok(body);
        }
        if size > MAX_RESPONSE
            || raw.len() < size + 2
            || &raw[size..size + 2] != b"\r\n"
            || body.len() + size > MAX_RESPONSE
        {
            return Err("Invalid MCP response framing".to_string());
        }
        body.extend_from_slice(&raw[..size]);
        raw = &raw[size + 2..];
    }
}

fn decode_response(raw: &[u8]) -> Result<Option<Value>, String> {
    let split = raw
        .windows(4)
        .position(|w| w == b"\r\n\r\n")
        .ok_or("Invalid MCP response framing")?;
    let headers = std::str::from_utf8(&raw[..split]).map_err(|_| "Invalid MCP response framing")?;
    let mut lines = headers.split("\r\n");
    let status = lines
        .next()
        .and_then(|line| line.split_whitespace().nth(1))
        .and_then(|status| status.parse::<u16>().ok())
        .ok_or("Invalid MCP response status")?;
    if status == 204 || status == 202 {
        return Ok(None);
    }
    if status != 200 {
        return Err(format!("Factory MCP request refused (HTTP {status})"));
    }
    let mut chunked = false;
    let mut length = None;
    for line in lines {
        if let Some((name, value)) = line.split_once(':') {
            if name.eq_ignore_ascii_case("transfer-encoding") {
                chunked = value.trim().eq_ignore_ascii_case("chunked");
            } else if name.eq_ignore_ascii_case("content-length") {
                length = Some(
                    value
                        .trim()
                        .parse::<usize>()
                        .map_err(|_| "Invalid MCP response length")?,
                );
            }
        }
    }
    let raw_body = &raw[split + 4..];
    let body = if chunked {
        chunked_body(raw_body)?
    } else {
        if length.is_some_and(|length| length != raw_body.len()) {
            return Err("Incomplete MCP response".to_string());
        }
        raw_body.to_vec()
    };
    serde_json::from_slice(&body)
        .map(Some)
        .map_err(|_| "Invalid MCP JSON response".to_string())
}

fn request(socket: &Path, credential: &Credential, body: &[u8]) -> Result<Option<Value>, String> {
    let mut stream =
        UnixStream::connect(socket).map_err(|_| "Factory MCP server is unavailable")?;
    let timeout = Some(Duration::from_secs(120));
    stream
        .set_read_timeout(timeout)
        .map_err(|_| "Factory MCP transport failed")?;
    stream
        .set_write_timeout(timeout)
        .map_err(|_| "Factory MCP transport failed")?;
    write!(stream, "POST /mcp HTTP/1.1\r\nHost: factory\r\nContent-Type: application/json\r\nAccept: application/json, text/event-stream\r\nMCP-Protocol-Version: 2025-03-26\r\nX-Soda-Factory-Capability: {}\r\nAuthorization: Bearer {}\r\nContent-Length: {}\r\nConnection: close\r\n\r\n",
        credential.capability, credential.secret, body.len()).map_err(|_| "Factory MCP write failed")?;
    stream
        .write_all(body)
        .map_err(|_| "Factory MCP write failed")?;
    let mut response = Vec::new();
    stream
        .take(MAX_RESPONSE as u64 + 1)
        .read_to_end(&mut response)
        .map_err(|_| "Factory MCP response was interrupted")?;
    if response.len() > MAX_RESPONSE {
        return Err("Factory MCP response exceeds its bound".to_string());
    }
    decode_response(&response)
}

fn relay(
    input: &mut impl BufRead,
    output: &mut impl Write,
    socket: &Path,
    credential: &Credential,
) -> Result<(), String> {
    loop {
        let mut line = Vec::new();
        let count = input
            .take(MAX_REQUEST as u64 + 1)
            .read_until(b'\n', &mut line)
            .map_err(|_| "Factory MCP input failed")?;
        if count == 0 {
            return Ok(());
        }
        if count > MAX_REQUEST {
            return Err("Factory MCP request exceeds its bound".to_string());
        }
        if line.iter().all(u8::is_ascii_whitespace) {
            continue;
        }
        let message: Value = serde_json::from_slice(&line).map_err(|_| "Invalid MCP input")?;
        if message.get("jsonrpc").and_then(Value::as_str) != Some("2.0")
            || message.get("method").and_then(Value::as_str).is_none()
        {
            return Err("Invalid MCP request".to_string());
        }
        let response = match request(socket, credential, &line) {
            Ok(response) => response,
            Err(error) => message.get("id").map(|id| {
                serde_json::json!({
                    "jsonrpc":"2.0","id":id,"error":{"code":-32603,"message":error}
                })
            }),
        };
        if let Some(response) = response {
            serde_json::to_writer(&mut *output, &response)
                .map_err(|_| "Factory MCP output failed")?;
            output
                .write_all(b"\n")
                .and_then(|_| output.flush())
                .map_err(|_| "Factory MCP output failed")?;
        }
    }
}

pub fn main() -> i32 {
    let result = (|| {
        let home = std::env::var("HOME").map_err(|_| "Factory MCP requires its staged run home")?;
        let credential = read_credential(&credential_path(&home)?)?;
        relay(
            &mut std::io::stdin().lock(),
            &mut std::io::stdout().lock(),
            Path::new(SOCKET),
            &credential,
        )
    })();
    match result {
        Ok(()) => 0,
        Err(error) => {
            eprintln!("{error}");
            1
        }
    }
}

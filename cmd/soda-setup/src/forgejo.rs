use std::io::{self, Read, Write};
use std::net::TcpStream;

use crate::system::{FORGEJO_RESPONSE_LIMIT, FORGEJO_TIMEOUT};
use serde::Deserialize;

#[derive(Debug)]
pub(crate) struct ForgejoUser {
    pub(crate) id: i64,
    pub(crate) admin: bool,
}

#[derive(Debug, Default)]
struct ForgejoUserResponse {
    id: i64,
    admin: bool,
}

impl<'de> Deserialize<'de> for ForgejoUserResponse {
    fn deserialize<D>(deserializer: D) -> Result<Self, D::Error>
    where
        D: serde::Deserializer<'de>,
    {
        use serde::de::{IgnoredAny, MapAccess, Visitor};
        use std::fmt;

        struct UserVisitor;

        impl<'de> Visitor<'de> for UserVisitor {
            type Value = ForgejoUserResponse;

            fn expecting(&self, formatter: &mut fmt::Formatter<'_>) -> fmt::Result {
                formatter.write_str("a Forgejo user object")
            }

            fn visit_map<A>(self, mut map: A) -> Result<Self::Value, A::Error>
            where
                A: MapAccess<'de>,
            {
                let mut user = ForgejoUserResponse::default();
                while let Some(key) = map.next_key::<String>()? {
                    match key.as_str() {
                        "id" => user.id = map.next_value()?,
                        "is_admin" => user.admin = map.next_value()?,
                        _ => {
                            map.next_value::<IgnoredAny>()?;
                        }
                    }
                }
                Ok(user)
            }
        }

        deserializer.deserialize_map(UserVisitor)
    }
}

pub(crate) fn forgejo_get_user(base: &str, token: &str) -> Result<ForgejoUser, String> {
    let (status, body) = forgejo_request(base, "GET", "/api/v1/user", token)?;
    if !(200..300).contains(&status) {
        return Err(format!("Forgejo rejected operation (HTTP {status})"));
    }
    decode_user(&body)
}

pub(crate) fn forgejo_revoke_token(base: &str, token: &str) -> Result<(), String> {
    let (status, _) = forgejo_request(base, "DELETE", "/api/v1/user/token", token)?;
    if !(200..300).contains(&status) {
        return Err(format!("Forgejo rejected operation (HTTP {status})"));
    }
    Ok(())
}

fn forgejo_request(
    base: &str,
    method: &str,
    path: &str,
    token: &str,
) -> Result<(u16, Vec<u8>), String> {
    let base = base.trim_end_matches('/');
    let host = base
        .strip_prefix("http://")
        .ok_or_else(|| "forgejo could not be reached".to_string())?;
    if host.is_empty() || host.contains('/') {
        return Err("forgejo could not be reached".to_string());
    }
    let (host_only, port) = match host.rsplit_once(':') {
        Some((h, p)) if !p.contains(']') => {
            let port: u16 = p
                .parse()
                .map_err(|_| "forgejo could not be reached".to_string())?;
            (h, port)
        }
        _ => (host, 80),
    };
    let addr: std::net::SocketAddr = match format!("{host_only}:{port}").parse() {
        Ok(addr) => addr,
        Err(_) => {
            dns_lookup(host_only, port).map_err(|_| "forgejo could not be reached".to_string())?
        }
    };
    let mut stream = TcpStream::connect_timeout(&addr, FORGEJO_TIMEOUT)
        .map_err(|_| "forgejo could not be reached".to_string())?;
    stream
        .set_read_timeout(Some(FORGEJO_TIMEOUT))
        .map_err(|_| "forgejo could not be reached".to_string())?;
    stream
        .set_write_timeout(Some(FORGEJO_TIMEOUT))
        .map_err(|_| "forgejo could not be reached".to_string())?;
    let request = format!(
        "{method} {path} HTTP/1.1\r\nHost: {host}\r\nAccept: application/json\r\nContent-Type: application/json\r\nAuthorization: token {token}\r\nConnection: close\r\n\r\n"
    );
    stream
        .write_all(request.as_bytes())
        .map_err(|_| "forgejo could not be reached".to_string())?;
    read_http_response(&mut stream)
}

fn dns_lookup(host: &str, port: u16) -> io::Result<std::net::SocketAddr> {
    use std::net::ToSocketAddrs;
    (host, port)
        .to_socket_addrs()?
        .next()
        .ok_or_else(|| io::Error::new(io::ErrorKind::NotFound, "no address"))
}

fn read_http_response(stream: &mut TcpStream) -> Result<(u16, Vec<u8>), String> {
    let mut raw: Vec<u8> = Vec::new();
    let mut byte = [0u8; 1];
    loop {
        match stream.read(&mut byte) {
            Ok(0) => return Err("forgejo could not be reached".to_string()),
            Ok(_) => raw.push(byte[0]),
            Err(_) => return Err("forgejo could not be reached".to_string()),
        }
        if raw.len() >= 4 && raw[raw.len() - 4..] == *b"\r\n\r\n" {
            break;
        }
    }
    let head = String::from_utf8(raw).map_err(|_| "forgejo could not be reached".to_string())?;
    let mut lines = head.split("\r\n");
    let status_line = lines
        .next()
        .ok_or_else(|| "forgejo could not be reached".to_string())?;
    let status: u16 = status_line
        .split_whitespace()
        .nth(1)
        .and_then(|code| code.parse().ok())
        .ok_or_else(|| "forgejo could not be reached".to_string())?;
    let mut content_length: Option<usize> = None;
    let mut chunked = false;
    for line in lines {
        if line.is_empty() {
            break;
        }
        let (name, value) = line
            .split_once(':')
            .ok_or_else(|| "forgejo could not be reached".to_string())?;
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
        let want = len.min(FORGEJO_RESPONSE_LIMIT + 1);
        let mut data = vec![0u8; want];
        stream
            .read_exact(&mut data)
            .map_err(|_| "forgejo could not be reached".to_string())?;
        if len > FORGEJO_RESPONSE_LIMIT {
            return Err("forgejo response exceeds the supported size".to_string());
        }
        data
    } else {
        read_to_end_limited(stream)?
    };
    Ok((status, data))
}

fn read_to_end_limited(stream: &mut TcpStream) -> Result<Vec<u8>, String> {
    let mut data: Vec<u8> = Vec::new();
    let mut chunk = [0u8; 8192];
    loop {
        match stream.read(&mut chunk) {
            Ok(0) => break,
            Ok(n) => {
                data.extend_from_slice(&chunk[..n]);
                if data.len() > FORGEJO_RESPONSE_LIMIT + 1 {
                    return Err("forgejo response exceeds the supported size".to_string());
                }
            }
            Err(_) => return Err("forgejo could not be reached".to_string()),
        }
    }
    if data.len() > FORGEJO_RESPONSE_LIMIT {
        return Err("forgejo response exceeds the supported size".to_string());
    }
    Ok(data)
}

fn read_chunked(stream: &mut TcpStream) -> Result<Vec<u8>, String> {
    let mut data: Vec<u8> = Vec::new();
    loop {
        let line = read_line(stream)?;
        let size = usize::from_str_radix(line.split(';').next().unwrap_or("").trim(), 16)
            .map_err(|_| "invalid Forgejo response".to_string())?;
        if size == 0 {
            let _ = read_line(stream);
            break;
        }
        if data.len() + size > FORGEJO_RESPONSE_LIMIT {
            return Err("forgejo response exceeds the supported size".to_string());
        }
        let mut chunk = vec![0u8; size];
        stream
            .read_exact(&mut chunk)
            .map_err(|_| "forgejo could not be reached".to_string())?;
        data.extend_from_slice(&chunk);
        let _ = read_line(stream);
    }
    Ok(data)
}

fn read_line(stream: &mut TcpStream) -> Result<String, String> {
    let mut line: Vec<u8> = Vec::new();
    let mut byte = [0u8; 1];
    loop {
        match stream.read(&mut byte) {
            Ok(0) => return Err("forgejo could not be reached".to_string()),
            Ok(_) => line.push(byte[0]),
            Err(_) => return Err("forgejo could not be reached".to_string()),
        }
        if line.len() >= 2 && line[line.len() - 2..] == *b"\r\n" {
            line.truncate(line.len() - 2);
            break;
        }
        if line.len() > 65536 {
            return Err("forgejo could not be reached".to_string());
        }
    }
    String::from_utf8(line).map_err(|_| "forgejo could not be reached".to_string())
}

// decode_user mirrors the forgejo client's decodeResponse into User: the body
// must be one JSON value within the size bound, "null" is invalid, and id /
// is_admin keep Go's strict scalar types (unknown fields allowed).
pub(crate) fn decode_user(body: &[u8]) -> Result<ForgejoUser, String> {
    if body.len() > FORGEJO_RESPONSE_LIMIT {
        return Err("forgejo response exceeds the supported size".to_string());
    }
    let text = std::str::from_utf8(body).map_err(|_| "invalid Forgejo response".to_string())?;
    if text.trim() == "null" {
        return Err("invalid Forgejo response".to_string());
    }
    let mut deserializer = serde_json::Deserializer::from_str(text);
    let user = ForgejoUserResponse::deserialize(&mut deserializer)
        .map_err(|_| "invalid Forgejo response".to_string())?;
    deserializer
        .end()
        .map_err(|_| "invalid Forgejo response".to_string())?;
    Ok(ForgejoUser {
        id: user.id,
        admin: user.admin,
    })
}

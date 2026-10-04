// soda-setup records the native operator identity and Soda service configuration.
use std::fs::{self, OpenOptions};
use std::io::{self, Read, Write};
use std::net::TcpStream;
use std::os::unix::ffi::OsStrExt;
use std::os::unix::fs::{DirBuilderExt, OpenOptionsExt, PermissionsExt};
use std::path::Path;
use std::time::Duration;

// postgresSecretDir holds the PostgreSQL credential files setup generates:
// super/forgejo/soda .passwd (0600) plus the soda connection URL (0640,
// group soda). The database units stay skipped until these exist, so live
// media shows a clean skip instead of a failed database.
const POSTGRES_SECRET_DIR: &str = "/etc/soda/postgres";
// Forgejo reads its database password over a root-owned read-only mount as
// the image user (USER_UID/USER_GID 1000 in forgejo.container); the source
// file must carry that ownership because the entrypoint drops supplementary
// groups, so group readability alone does not admit it.
const FORGEJO_DB_USER: u32 = 1000;
// The soda DSN is read by the dashboard (uid/gid 2000) and the identity
// broker (gid 2000). Group soda (2000, appliance/config/soda.sysusers)
// admits exactly those service identities.
const SODA_SERVICE_GROUP: u32 = 2000;
// postgresSocketDir is the host path of the PostgreSQL unix-socket
// directory shared with host-network and native clients (soda.tmpfiles);
// it doubles as the DSN host so no database TCP reaches the host.
const POSTGRES_SOCKET_DIR: &str = "/run/soda/postgres";
const FORGEJO_TIMEOUT: Duration = Duration::from_secs(30);
const FORGEJO_RESPONSE_LIMIT: usize = 2 * 1024 * 1024;

// Minimal libc surface (chown/geteuid/errno) declared directly so this crate
// stays dependency-free and offline-buildable; the full libc crate is not
// vendored in this tree.
#[link(name = "c")]
extern "C" {
    fn geteuid() -> u32;
    fn chown(path: *const std::os::raw::c_char, owner: u32, group: u32) -> std::os::raw::c_int;
    #[cfg(target_os = "linux")]
    fn __errno_location() -> *mut std::os::raw::c_int;
}

fn euid() -> u32 {
    unsafe { geteuid() }
}

fn chown_path(path: &Path, owner: u32, group: u32) -> io::Result<()> {
    use std::os::raw::c_char;
    let bytes = path.as_os_str().as_bytes();
    let mut nul = Vec::with_capacity(bytes.len() + 1);
    nul.extend_from_slice(bytes);
    nul.push(0);
    let ret = unsafe { chown(nul.as_ptr() as *const c_char, owner, group) };
    if ret == 0 {
        Ok(())
    } else {
        Err(io::Error::new(
            io::ErrorKind::PermissionDenied,
            format!("chown {}: {}", path.display(), errno_message()),
        ))
    }
}

fn errno_message() -> String {
    #[cfg(target_os = "linux")]
    {
        let code = unsafe { *__errno_location() };
        io::Error::from_raw_os_error(code).to_string()
    }
    #[cfg(not(target_os = "linux"))]
    {
        "operation failed".to_string()
    }
}

fn main() {
    std::process::exit(run());
}

fn run() -> i32 {
    if euid() != 0 {
        eprintln!("run through the operator's native root access");
        return 1;
    }
    let args: Vec<String> = std::env::args().skip(1).collect();
    let opts = match parse_args(&args) {
        Ok(opts) => opts,
        Err(ParseOutcome::Help) => return 0,
        Err(ParseOutcome::Error) => return 2,
    };
    let mut out = io::stdout();
    if opts.provision_db_only {
        match provision_postgres_secrets(Path::new(POSTGRES_SECRET_DIR)) {
            Ok((_, dsn_path)) => {
                println!("Database credentials ready: {}", dsn_path.display());
                0
            }
            Err(err) => {
                eprintln!("{err}");
                1
            }
        }
    } else {
        match setup(
            &opts.forgejo_url,
            &opts.forgejo_internal_url,
            &opts.token_file,
            Path::new(&opts.out),
            Path::new(POSTGRES_SECRET_DIR),
            &mut out,
        ) {
            Ok(()) => 0,
            Err(err) => {
                let _ = out.flush();
                eprintln!("{err}");
                1
            }
        }
    }
}

struct SetupOpts {
    forgejo_url: String,
    forgejo_internal_url: String,
    token_file: String,
    out: String,
    provision_db_only: bool,
}

#[derive(Debug)]
enum ParseOutcome {
    Help,
    Error,
}

fn print_setup_usage() {
    eprint!(
        "Usage of soda-setup:\n  -forgejo-internal-url string\n    \tnative Forgejo origin\n  -forgejo-url string\n    \tForgejo HTTPS origin\n  -out string\n    \tnew dashboard configuration\n  -provision-db-only\n    \tgenerate database credentials without operator binding\n  -token-file string\n    \toperator Forgejo access token file\n"
    );
}

fn parse_bool_flag(
    name: &str,
    inline: Option<&str>,
    args: &[String],
    i: &mut usize,
) -> Result<bool, ParseOutcome> {
    match inline {
        Some(v) => parse_go_bool(name, v),
        None => {
            // A following token that is not another flag is the value; Go
            // treats a bare boolean flag as true.
            if *i + 1 < args.len() && !looks_like_flag(&args[*i + 1]) {
                *i += 1;
                parse_go_bool(name, &args[*i].clone())
            } else {
                Ok(true)
            }
        }
    }
}

fn parse_go_bool(name: &str, value: &str) -> Result<bool, ParseOutcome> {
    match value {
        "1" | "t" | "T" | "true" | "TRUE" | "True" => Ok(true),
        "0" | "f" | "F" | "false" | "FALSE" | "False" => Ok(false),
        _ => {
            let msg = format!("invalid boolean value {value:?} for -{name}: parse error");
            eprintln!("{msg}");
            print_setup_usage();
            Err(ParseOutcome::Error)
        }
    }
}

fn looks_like_flag(arg: &str) -> bool {
    arg.len() > 1 && arg.starts_with('-')
}

fn parse_args(args: &[String]) -> Result<SetupOpts, ParseOutcome> {
    let mut opts = SetupOpts {
        forgejo_url: String::new(),
        forgejo_internal_url: "http://127.0.0.1:3000".to_string(),
        token_file: String::new(),
        out: "/etc/soda/dashboard.json".to_string(),
        provision_db_only: false,
    };
    let mut i = 0;
    while i < args.len() {
        let arg = &args[i];
        if !arg.starts_with('-') || arg == "-" {
            // Go's flag package stops at the first positional argument and
            // setup never inspects Args, so positionals are ignored.
            break;
        }
        if arg == "--" {
            break;
        }
        let flag = arg.trim_start_matches('-');
        let (name, inline) = match flag.split_once('=') {
            Some((n, v)) => (n, Some(v)),
            None => (flag, None),
        };
        if name == "h" || name == "help" {
            print_setup_usage();
            return Err(ParseOutcome::Help);
        }
        match name {
            "provision-db-only" => {
                opts.provision_db_only = parse_bool_flag(name, inline, args, &mut i)?;
            }
            "forgejo-url" | "forgejo-internal-url" | "token-file" | "out" => {
                let value = match inline {
                    Some(v) => v.to_string(),
                    None => {
                        i += 1;
                        if i >= args.len() {
                            let msg = format!("flag needs an argument: -{name}");
                            eprintln!("{msg}");
                            print_setup_usage();
                            return Err(ParseOutcome::Error);
                        }
                        args[i].clone()
                    }
                };
                match name {
                    "forgejo-url" => opts.forgejo_url = value,
                    "forgejo-internal-url" => opts.forgejo_internal_url = value,
                    "token-file" => opts.token_file = value,
                    _ => opts.out = value,
                }
            }
            _ => {
                let msg = format!("flag provided but not defined: -{name}");
                eprintln!("{msg}");
                print_setup_usage();
                return Err(ParseOutcome::Error);
            }
        }
        i += 1;
    }
    Ok(opts)
}

// base_url mirrors config.BaseURL: an HTTP(S) origin without credentials,
// path, query or fragment. It follows Go url.Parse admission for origins:
// exact lowercase scheme, non-empty host, no userinfo, path "" or "/".
fn base_url(value: &str) -> Result<(), String> {
    if value.bytes().any(|b| b < 0x20 || b == 0x7f || b == b' ') {
        return Err(
            "must be an HTTP(S) origin without credentials, path, query or fragment".to_string(),
        );
    }
    // Go's url.Parse lowercases the scheme before matching.
    let lower = value.to_ascii_lowercase();
    let rest = if lower.starts_with("http://") {
        &value["http://".len()..]
    } else if lower.starts_with("https://") {
        &value["https://".len()..]
    } else {
        return Err(
            "must be an HTTP(S) origin without credentials, path, query or fragment".to_string(),
        );
    };
    if rest.is_empty() {
        return Err(
            "must be an HTTP(S) origin without credentials, path, query or fragment".to_string(),
        );
    }
    let authority_end = rest.find(&['/', '?', '#'][..]).unwrap_or(rest.len());
    let (authority, remainder) = rest.split_at(authority_end);
    if authority.is_empty() || authority.contains('@') {
        return Err(
            "must be an HTTP(S) origin without credentials, path, query or fragment".to_string(),
        );
    }
    if !remainder.is_empty() && remainder != "/" {
        return Err(
            "must be an HTTP(S) origin without credentials, path, query or fragment".to_string(),
        );
    }
    if !valid_percent_escapes(value) {
        return Err(
            "must be an HTTP(S) origin without credentials, path, query or fragment".to_string(),
        );
    }
    Ok(())
}

fn valid_percent_escapes(value: &str) -> bool {
    let bytes = value.as_bytes();
    let mut i = 0;
    while i < bytes.len() {
        if bytes[i] == b'%' {
            if i + 2 >= bytes.len()
                || !bytes[i + 1].is_ascii_hexdigit()
                || !bytes[i + 2].is_ascii_hexdigit()
            {
                return false;
            }
            i += 3;
        } else {
            i += 1;
        }
    }
    true
}

// credential mirrors config.Secret: a regular file inaccessible to others,
// at most 64 KiB, with non-empty trimmed content free of CR/LF/NUL.
fn credential(path: &str) -> Result<String, String> {
    // os.Open follows symlinks and f.Stat describes the target; match that
    // with an open handle plus following metadata.
    let file =
        fs::File::open(path).map_err(|_| "cannot open configured credential file".to_string())?;
    let st = file
        .metadata()
        .map_err(|_| "credential must be a regular file inaccessible to other users".to_string())?;
    if !st.file_type().is_file() || st.permissions().mode() & 0o007 != 0 {
        return Err("credential must be a regular file inaccessible to other users".to_string());
    }
    let mut data: Vec<u8> = Vec::new();
    if file.take(65537).read_to_end(&mut data).is_err() || data.len() > 65536 {
        return Err("cannot read credential file".to_string());
    }
    let value = String::from_utf8_lossy(&data);
    let trimmed = value.trim().to_string();
    if trimmed.is_empty() || trimmed.contains(['\r', '\n', '\0']) {
        return Err("credential is empty or malformed".to_string());
    }
    Ok(trimmed)
}

#[derive(Debug)]
struct ForgejoUser {
    id: i64,
    admin: bool,
}

fn forgejo_get_user(base: &str, token: &str) -> Result<ForgejoUser, String> {
    let (status, body) = forgejo_request(base, "GET", "/api/v1/user", token)?;
    if !(200..300).contains(&status) {
        return Err(format!("Forgejo rejected operation (HTTP {status})"));
    }
    decode_user(&body)
}

fn forgejo_revoke_token(base: &str, token: &str) -> Result<(), String> {
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
fn decode_user(body: &[u8]) -> Result<ForgejoUser, String> {
    if body.len() > FORGEJO_RESPONSE_LIMIT {
        return Err("forgejo response exceeds the supported size".to_string());
    }
    let text = std::str::from_utf8(body).map_err(|_| "invalid Forgejo response".to_string())?;
    if text.trim() == "null" {
        return Err("invalid Forgejo response".to_string());
    }
    let value = JsonValue::parse(text)?;
    let obj = value
        .as_object()
        .ok_or_else(|| "invalid Forgejo response".to_string())?;
    let mut id: i64 = 0;
    let mut admin = false;
    for (key, val) in obj {
        match key.as_str() {
            "id" => {
                id = val
                    .as_i64()
                    .ok_or_else(|| "invalid Forgejo response".to_string())?;
            }
            "is_admin" => {
                admin = val
                    .as_bool()
                    .ok_or_else(|| "invalid Forgejo response".to_string())?;
            }
            _ => {}
        }
    }
    Ok(ForgejoUser { id, admin })
}

#[derive(Debug)]
#[allow(dead_code)]
enum JsonValue {
    Null,
    Bool(bool),
    Number(String),
    Str(String),
    Array(Vec<JsonValue>),
    Object(Vec<(String, JsonValue)>),
}

impl JsonValue {
    fn parse(text: &str) -> Result<JsonValue, String> {
        let mut parser = JsonParser {
            bytes: text.as_bytes(),
            pos: 0,
        };
        parser.skip_ws();
        let value = parser.parse_value()?;
        parser.skip_ws();
        if parser.pos != parser.bytes.len() {
            return Err("invalid Forgejo response".to_string());
        }
        Ok(value)
    }

    fn as_object(&self) -> Option<&[(String, JsonValue)]> {
        match self {
            JsonValue::Object(entries) => Some(entries),
            _ => None,
        }
    }

    fn as_bool(&self) -> Option<bool> {
        match self {
            JsonValue::Bool(b) => Some(*b),
            _ => None,
        }
    }

    fn as_i64(&self) -> Option<i64> {
        match self {
            JsonValue::Number(raw) => {
                let digits = raw.strip_prefix('-').unwrap_or(raw);
                digits
                    .bytes()
                    .all(|b| b.is_ascii_digit())
                    .then(|| raw.parse().ok())?
            }
            _ => None,
        }
    }
}

struct JsonParser<'a> {
    bytes: &'a [u8],
    pos: usize,
}

impl<'a> JsonParser<'a> {
    fn skip_ws(&mut self) {
        while self.pos < self.bytes.len()
            && matches!(self.bytes[self.pos], b' ' | b'\t' | b'\n' | b'\r')
        {
            self.pos += 1;
        }
    }

    fn parse_value(&mut self) -> Result<JsonValue, String> {
        let invalid = || "invalid Forgejo response".to_string();
        let byte = *self.bytes.get(self.pos).ok_or_else(invalid)?;
        match byte {
            b'{' => self.parse_object(),
            b'[' => self.parse_array(),
            b'"' => Ok(JsonValue::Str(self.parse_string()?)),
            b't' => self.parse_literal("true", JsonValue::Bool(true)),
            b'f' => self.parse_literal("false", JsonValue::Bool(false)),
            b'n' => self.parse_literal("null", JsonValue::Null),
            b'-' | b'0'..=b'9' => Ok(JsonValue::Number(self.parse_number()?)),
            _ => Err(invalid()),
        }
    }

    fn parse_literal(&mut self, word: &str, value: JsonValue) -> Result<JsonValue, String> {
        if self.bytes[self.pos..].starts_with(word.as_bytes()) {
            self.pos += word.len();
            Ok(value)
        } else {
            Err("invalid Forgejo response".to_string())
        }
    }

    fn parse_object(&mut self) -> Result<JsonValue, String> {
        let invalid = || "invalid Forgejo response".to_string();
        self.pos += 1;
        let mut entries = Vec::new();
        self.skip_ws();
        if self.bytes.get(self.pos) == Some(&b'}') {
            self.pos += 1;
            return Ok(JsonValue::Object(entries));
        }
        loop {
            self.skip_ws();
            if self.bytes.get(self.pos) != Some(&b'"') {
                return Err(invalid());
            }
            let key = self.parse_string()?;
            self.skip_ws();
            if self.bytes.get(self.pos) != Some(&b':') {
                return Err(invalid());
            }
            self.pos += 1;
            self.skip_ws();
            let value = self.parse_value()?;
            entries.push((key, value));
            self.skip_ws();
            match self.bytes.get(self.pos) {
                Some(b',') => {
                    self.pos += 1;
                }
                Some(b'}') => {
                    self.pos += 1;
                    return Ok(JsonValue::Object(entries));
                }
                _ => return Err(invalid()),
            }
        }
    }

    fn parse_array(&mut self) -> Result<JsonValue, String> {
        let invalid = || "invalid Forgejo response".to_string();
        self.pos += 1;
        let mut items = Vec::new();
        self.skip_ws();
        if self.bytes.get(self.pos) == Some(&b']') {
            self.pos += 1;
            return Ok(JsonValue::Array(items));
        }
        loop {
            self.skip_ws();
            items.push(self.parse_value()?);
            self.skip_ws();
            match self.bytes.get(self.pos) {
                Some(b',') => {
                    self.pos += 1;
                }
                Some(b']') => {
                    self.pos += 1;
                    return Ok(JsonValue::Array(items));
                }
                _ => return Err(invalid()),
            }
        }
    }

    fn parse_string(&mut self) -> Result<String, String> {
        let invalid = || "invalid Forgejo response".to_string();
        self.pos += 1;
        let mut out: Vec<u8> = Vec::new();
        loop {
            let byte = *self.bytes.get(self.pos).ok_or_else(invalid)?;
            self.pos += 1;
            match byte {
                b'"' => break,
                b'\\' => {
                    let esc = *self.bytes.get(self.pos).ok_or_else(invalid)?;
                    self.pos += 1;
                    match esc {
                        b'"' => out.push(b'"'),
                        b'\\' => out.push(b'\\'),
                        b'/' => out.push(b'/'),
                        b'b' => out.push(0x08),
                        b'f' => out.push(0x0c),
                        b'n' => out.push(b'\n'),
                        b'r' => out.push(b'\r'),
                        b't' => out.push(b'\t'),
                        b'u' => {
                            if self.pos + 4 > self.bytes.len() {
                                return Err(invalid());
                            }
                            let hex = std::str::from_utf8(&self.bytes[self.pos..self.pos + 4])
                                .map_err(|_| invalid())?;
                            let mut code = u32::from_str_radix(hex, 16).map_err(|_| invalid())?;
                            self.pos += 4;
                            if (0xd800..0xdc00).contains(&code)
                                && self.bytes.get(self.pos..self.pos + 2) == Some(b"\\u".as_slice())
                            {
                                let low_hex =
                                    std::str::from_utf8(&self.bytes[self.pos + 2..self.pos + 6])
                                        .map_err(|_| invalid())?;
                                let low =
                                    u32::from_str_radix(low_hex, 16).map_err(|_| invalid())?;
                                if (0xdc00..0xe000).contains(&low) {
                                    code = 0x10000 + ((code - 0xd800) << 10) + (low - 0xdc00);
                                    self.pos += 6;
                                }
                            }
                            let ch = char::from_u32(code).ok_or_else(invalid)?;
                            let mut buf = [0u8; 4];
                            out.extend_from_slice(ch.encode_utf8(&mut buf).as_bytes());
                        }
                        _ => return Err(invalid()),
                    }
                }
                0x00..=0x1f => return Err(invalid()),
                _ => out.push(byte),
            }
        }
        String::from_utf8(out).map_err(|_| invalid())
    }

    fn parse_number(&mut self) -> Result<String, String> {
        let invalid = || "invalid Forgejo response".to_string();
        let start = self.pos;
        if self.bytes.get(self.pos) == Some(&b'-') {
            self.pos += 1;
        }
        match self.bytes.get(self.pos) {
            Some(b'0') => {
                self.pos += 1;
            }
            Some(b'1'..=b'9') => {
                while matches!(self.bytes.get(self.pos), Some(b'0'..=b'9')) {
                    self.pos += 1;
                }
            }
            _ => return Err(invalid()),
        }
        if self.bytes.get(self.pos) == Some(&b'.') {
            self.pos += 1;
            if !matches!(self.bytes.get(self.pos), Some(b'0'..=b'9')) {
                return Err(invalid());
            }
            while matches!(self.bytes.get(self.pos), Some(b'0'..=b'9')) {
                self.pos += 1;
            }
        }
        if matches!(self.bytes.get(self.pos), Some(b'e' | b'E')) {
            self.pos += 1;
            if matches!(self.bytes.get(self.pos), Some(b'+' | b'-')) {
                self.pos += 1;
            }
            if !matches!(self.bytes.get(self.pos), Some(b'0'..=b'9')) {
                return Err(invalid());
            }
            while matches!(self.bytes.get(self.pos), Some(b'0'..=b'9')) {
                self.pos += 1;
            }
        }
        std::str::from_utf8(&self.bytes[start..self.pos])
            .map(|s| s.to_string())
            .map_err(|_| invalid())
    }
}

fn push_json_string(out: &mut String, value: &str) {
    out.push('"');
    for ch in value.chars() {
        match ch {
            '"' => out.push_str("\\\""),
            '\\' => out.push_str("\\\\"),
            '\n' => out.push_str("\\n"),
            '\r' => out.push_str("\\r"),
            '\t' => out.push_str("\\t"),
            '<' => out.push_str("\\u003c"),
            '>' => out.push_str("\\u003e"),
            '&' => out.push_str("\\u0026"),
            c if (c as u32) < 0x20 => {
                out.push_str(&format!("\\u{:04x}", c as u32));
            }
            c => out.push(c),
        }
    }
    out.push('"');
}

// encode_dashboard_config mirrors Go's json.Encoder over config.Config for a
// fresh setup record: all keys in struct order (including "" and null for
// unset optionals), Go string escaping, and a trailing newline.
#[allow(clippy::too_many_arguments)]
fn encode_dashboard_config(
    listen: &str,
    forgejo_url: &str,
    forgejo_internal_url: &str,
    database_dsn_file: &str,
    host_socket: &str,
    identity_socket: &str,
    grant_key_file: &str,
    operator_id: i64,
) -> Vec<u8> {
    let mut body = String::from("{");
    let mut first = true;
    let field = |body: &mut String, first: &mut bool, key: &str, value: &str| {
        if !*first {
            body.push(',');
        }
        *first = false;
        push_json_string(body, key);
        body.push(':');
        push_json_string(body, value);
    };
    field(&mut body, &mut first, "listen", listen);
    field(&mut body, &mut first, "forgejo_url", forgejo_url);
    field(
        &mut body,
        &mut first,
        "forgejo_internal_url",
        forgejo_internal_url,
    );
    field(
        &mut body,
        &mut first,
        "database_dsn_file",
        database_dsn_file,
    );
    field(&mut body, &mut first, "host_socket", host_socket);
    field(&mut body, &mut first, "identity_socket", identity_socket);
    field(&mut body, &mut first, "grant_key_file", grant_key_file);
    body.push(',');
    push_json_string(&mut body, "operator_id");
    body.push(':');
    body.push_str(&operator_id.to_string());
    body.push(',');
    push_json_string(&mut body, "forgejo_background_socket");
    body.push_str(":\"\",");
    push_json_string(&mut body, "forgejo_background_host_uid");
    body.push_str(":null");
    for key in [
        "forgejo_background_credential_file",
        "forgejo_review_credential_file",
        "forgejo_merge_credential_file",
        "factory_intake_secret_file",
        "factory_publication_root",
    ] {
        body.push(',');
        push_json_string(&mut body, key);
        body.push_str(":\"\"");
    }
    body.push_str("}\n");
    body.into_bytes()
}

fn read_random_32() -> Result<[u8; 32], String> {
    let mut file =
        fs::File::open("/dev/urandom").map_err(|e| format!("generate database password: {e}"))?;
    let mut raw = [0u8; 32];
    file.read_exact(&mut raw)
        .map_err(|e| format!("generate database password: {e}"))?;
    Ok(raw)
}

fn hex_encode(raw: &[u8]) -> String {
    const HEX: &[u8; 16] = b"0123456789abcdef";
    let mut out = String::with_capacity(raw.len() * 2);
    for byte in raw {
        out.push(HEX[(byte >> 4) as usize] as char);
        out.push(HEX[(byte & 0x0f) as usize] as char);
    }
    out
}

fn base64_encode(raw: &[u8]) -> String {
    const ALPHABET: &[u8; 64] = b"ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/";
    let mut out = String::with_capacity(raw.len().div_ceil(3) * 4);
    for chunk in raw.chunks(3) {
        let mut n: u32 = 0;
        for (i, byte) in chunk.iter().enumerate() {
            n |= (*byte as u32) << (16 - 8 * i);
        }
        out.push(ALPHABET[(n >> 18) as usize & 63] as char);
        out.push(ALPHABET[(n >> 12) as usize & 63] as char);
        if chunk.len() > 1 {
            out.push(ALPHABET[(n >> 6) as usize & 63] as char);
        } else {
            out.push('=');
        }
        if chunk.len() > 2 {
            out.push(ALPHABET[n as usize & 63] as char);
        } else {
            out.push('=');
        }
    }
    out
}

fn write_secret_file(path: &Path, value: &str) -> Result<(), String> {
    let mut file = OpenOptions::new()
        .write(true)
        .create_new(true)
        .mode(0o600)
        .open(path)
        .map_err(|e| format!("create {}: {e}", path.display()))?;
    if let Err(e) = file.write_all(format!("{value}\n").as_bytes()) {
        let _ = fs::remove_file(path);
        return Err(e.to_string());
    }
    drop(file);
    Ok(())
}

// reuse_postgres_secrets adopts a complete first-boot credential set so the
// operator flow never regenerates live database passwords. It returns the
// DSN path when all four files exist and agree, an error when they exist
// but disagree, and None when anything is missing (the caller then creates
// every file O_EXCL, so partial sets still fail instead of mixing).
fn reuse_postgres_secrets(dir: &Path) -> Result<Option<std::path::PathBuf>, String> {
    let mut reads = std::collections::HashMap::new();
    for name in ["super.passwd", "forgejo.passwd", "soda.passwd", "soda.dsn"] {
        match fs::read(dir.join(name)) {
            Ok(data) => {
                reads.insert(name, String::from_utf8_lossy(&data).trim().to_string());
            }
            Err(_) => return Ok(None),
        }
    }
    let empty = String::new();
    let soda_pw = reads.get("soda.passwd").unwrap_or(&empty);
    let dsn = reads.get("soda.dsn").unwrap_or(&empty);
    if soda_pw.is_empty() || !dsn.contains(soda_pw) {
        return Err(format!(
            "database secrets in {} exist but disagree; inspect them before setup",
            dir.display()
        ));
    }
    Ok(Some(dir.join("soda.dsn")))
}

// provision_postgres_secrets generates the database credential files: one
// hex password per role plus the soda connection URL. Files land O_EXCL so
// a pre-existing secret is never overwritten; on failure this run removes
// only the files it created, preserving anything already there. Ownership
// fixes apply at creation only; pre-existing files keep their metadata.
fn provision_postgres_secrets(
    dir: &Path,
) -> Result<(Vec<std::path::PathBuf>, std::path::PathBuf), String> {
    fs::DirBuilder::new()
        .recursive(true)
        .mode(0o700)
        .create(dir)
        .map_err(|e| e.to_string())?;
    if let Some(reuse) = reuse_postgres_secrets(dir)? {
        return Ok((Vec::new(), reuse));
    }
    let mut created: Vec<std::path::PathBuf> = Vec::new();
    let mut passwords = std::collections::HashMap::new();
    let mut failure: Option<String> = None;
    for role in ["super", "forgejo", "soda"] {
        let raw = match read_random_32() {
            Ok(raw) => raw,
            Err(err) => {
                failure = Some(err);
                break;
            }
        };
        let pw = hex_encode(&raw);
        let path = dir.join(format!("{role}.passwd"));
        if let Err(err) = write_secret_file(&path, &pw) {
            failure = Some(err);
            break;
        }
        if role == "forgejo" && euid() == 0 {
            if let Err(err) = chown_path(&path, FORGEJO_DB_USER, FORGEJO_DB_USER) {
                let _ = fs::remove_file(&path);
                failure = Some(err.to_string());
                break;
            }
        }
        created.push(path);
        passwords.insert(role, pw);
    }
    let mut dsn_path = std::path::PathBuf::new();
    if failure.is_none() {
        dsn_path = dir.join("soda.dsn");
        let dsn = format!(
            "postgres://soda:{}@/soda?host={POSTGRES_SOCKET_DIR}&sslmode=disable",
            passwords.get("soda").cloned().unwrap_or_default()
        );
        match write_secret_file(&dsn_path, &dsn) {
            Ok(()) => {
                created.push(dsn_path.clone());
                let chmod_err =
                    fs::set_permissions(&dsn_path, fs::Permissions::from_mode(0o640)).err();
                if let Some(err) = chmod_err {
                    failure = Some(err.to_string());
                } else if euid() == 0 {
                    if let Err(err) = chown_path(&dsn_path, 0, SODA_SERVICE_GROUP) {
                        failure = Some(err.to_string());
                    }
                }
            }
            Err(err) => {
                failure = Some(err);
            }
        }
    }
    if let Some(err) = failure {
        for path in &created {
            let _ = fs::remove_file(path);
        }
        return Err(err);
    }
    Ok((created, dsn_path))
}

fn admit_setup_paths(external: &str, internal: &str, out: &Path) -> Result<(), String> {
    base_url(external)?;
    base_url(internal)?;
    if !external.starts_with("https://") {
        return Err("browser origins must use HTTPS".to_string());
    }
    if !out.is_absolute() {
        return Err("out must be absolute".to_string());
    }
    match fs::symlink_metadata(out) {
        Err(e) if e.kind() == io::ErrorKind::NotFound => {}
        _ => {
            return Err(
                "configuration already exists or cannot be inspected; refusing overwrite"
                    .to_string(),
            );
        }
    }
    Ok(())
}

fn setup(
    external: &str,
    internal: &str,
    token_path: &str,
    out: &Path,
    pg_dir: &Path,
    stdout: &mut dyn Write,
) -> Result<(), String> {
    admit_setup_paths(external, internal, out)?;
    let token = credential(token_path)?;
    let user = forgejo_get_user(internal.trim_end_matches('/'), &token)?;
    if !user.admin {
        return Err("forgejo operator token is not a site administrator".to_string());
    }
    let dir = out
        .parent()
        .ok_or_else(|| "out must be absolute".to_string())?;
    fs::DirBuilder::new()
        .recursive(true)
        .mode(0o700)
        .create(dir)
        .map_err(|e| e.to_string())?;
    let key_path = dir.join("grant-key");
    let key_raw =
        read_random_32_inner().map_err(|e| format!("generate identity encryption key: {e}"))?;
    write_setup_secret(&key_path, &base64_encode(&key_raw))?;
    let (pg_created, dsn_path) = match provision_postgres_secrets(pg_dir) {
        Ok(result) => result,
        Err(err) => {
            let _ = fs::remove_file(&key_path);
            return Err(err);
        }
    };
    let config = encode_dashboard_config(
        "127.0.0.1:8080",
        external.trim_end_matches('/'),
        internal.trim_end_matches('/'),
        &dsn_path.to_string_lossy(),
        "/run/soda/host.sock",
        "",
        &key_path.to_string_lossy(),
        user.id,
    );
    if let Err(err) = write_setup_config(out, &config) {
        // This run created the key and database secrets through O_EXCL, so
        // no other setup owns them; remove them so a retry is not blocked
        // by our own partial state.
        let _ = fs::remove_file(&key_path);
        for path in &pg_created {
            let _ = fs::remove_file(path);
        }
        return Err(err);
    }
    // Revoke only after the configuration is durable: earlier failures keep
    // the bootstrap token so the operator can retry with it.
    if let Err(err) = forgejo_revoke_token(internal.trim_end_matches('/'), &token) {
        return Err(format!(
            "dashboard configuration written, but the bootstrap token is still live; revoke it in Forgejo Settings > Applications, then continue activation with the installed soda-activate command: {err}"
        ));
    }
    let _ = writeln!(
        stdout,
        "Dashboard configuration created and the bootstrap token revoked. Set native soda service ownership before enabling the dashboard. No host privilege was granted to a Forgejo user."
    );
    Ok(())
}

fn read_random_32_inner() -> io::Result<[u8; 32]> {
    let mut file = fs::File::open("/dev/urandom")?;
    let mut raw = [0u8; 32];
    file.read_exact(&mut raw)?;
    Ok(raw)
}

fn write_setup_secret(path: &Path, value: &str) -> Result<(), String> {
    let mut file = OpenOptions::new()
        .write(true)
        .create_new(true)
        .mode(0o600)
        .open(path)
        .map_err(|e| format!("create credential encryption key: {e}"))?;
    let data = format!("{value}\n");
    file.write_all(data.as_bytes()).map_err(|e| e.to_string())?;
    drop(file);
    Ok(())
}

fn write_setup_config(out: &Path, config: &[u8]) -> Result<(), String> {
    let mut file = OpenOptions::new()
        .write(true)
        .create_new(true)
        .mode(0o600)
        .open(out)
        .map_err(|e| e.to_string())?;
    file.write_all(config).map_err(|e| e.to_string())?;
    drop(file);
    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::net::TcpListener;
    use std::os::unix::fs::PermissionsExt;
    use std::sync::atomic::{AtomicU64, Ordering};
    use std::sync::{Arc, Mutex};
    use std::thread;

    static TEST_SEQ: AtomicU64 = AtomicU64::new(0);

    fn test_root() -> std::path::PathBuf {
        let seq = TEST_SEQ.fetch_add(1, Ordering::SeqCst);
        let dir =
            std::env::temp_dir().join(format!("soda-setup-test-{}-{seq}", std::process::id()));
        fs::create_dir_all(&dir).expect("test root");
        dir
    }

    struct Stub {
        url: String,
        calls: Arc<Mutex<Vec<String>>>,
    }

    fn stub_server(admin: bool, deny_current: bool, token: &str) -> Stub {
        let listener = TcpListener::bind("127.0.0.1:0").expect("bind stub");
        let url = format!("http://{}", listener.local_addr().expect("stub addr"));
        let calls: Arc<Mutex<Vec<String>>> = Arc::new(Mutex::new(Vec::new()));
        let calls_clone = Arc::clone(&calls);
        let token = token.to_string();
        let token_clone = token.clone();
        thread::spawn(move || {
            listener.set_nonblocking(false).ok();
            // Serve until the test ends; each connection carries one request.
            for stream in listener.incoming().take(6) {
                let mut stream = match stream {
                    Ok(s) => s,
                    Err(_) => break,
                };
                stream.set_read_timeout(Some(Duration::from_secs(10))).ok();
                let mut raw: Vec<u8> = Vec::new();
                let mut byte = [0u8; 1];
                let mut head = String::new();
                loop {
                    match stream.read(&mut byte) {
                        Ok(0) => break,
                        Ok(_) => raw.push(byte[0]),
                        Err(_) => break,
                    }
                    if raw.len() >= 4 && raw[raw.len() - 4..] == *b"\r\n\r\n" {
                        head = String::from_utf8_lossy(&raw).into_owned();
                        break;
                    }
                    if raw.len() > 65536 {
                        break;
                    }
                }
                let mut lines = head.split("\r\n");
                let request_line = lines.next().unwrap_or("").to_string();
                let mut authorized = false;
                for line in lines {
                    if let Some((name, value)) = line.split_once(':') {
                        if name.trim().eq_ignore_ascii_case("authorization") {
                            authorized = value.trim() == format!("token {token_clone}");
                        }
                    }
                }
                let mut parts = request_line.split_whitespace();
                let method = parts.next().unwrap_or("").to_string();
                let path = parts.next().unwrap_or("").to_string();
                calls_clone
                    .lock()
                    .expect("calls")
                    .push(format!("{method} {path}"));
                let (code, body) = if !authorized {
                    (401, "unauthorized".to_string())
                } else {
                    match format!("{method} {path}").as_str() {
                        "GET /api/v1/user" => {
                            if deny_current {
                                (403, token_clone.clone())
                            } else {
                                (200, format!("{{\"id\": 42, \"login\": \"soda-tester\", \"is_admin\": {admin}}}"))
                            }
                        }
                        "DELETE /api/v1/user/token" => (204, String::new()),
                        _ => (404, "unexpected endpoint".to_string()),
                    }
                };
                let reason = match code {
                    200 => "OK",
                    204 => "No Content",
                    401 => "Unauthorized",
                    403 => "Forbidden",
                    _ => "Not Found",
                };
                let response = format!(
                    "HTTP/1.1 {code} {reason}\r\nContent-Type: application/json\r\nContent-Length: {}\r\nConnection: close\r\n\r\n{body}",
                    body.len()
                );
                let _ = stream.write_all(response.as_bytes());
            }
        });
        Stub { url, calls }
    }

    fn write_token(dir: &Path) -> std::path::PathBuf {
        let path = dir.join("operator-input");
        fs::write(&path, "synthetic-bootstrap-token-not-for-retention\n").expect("token");
        fs::set_permissions(&path, fs::Permissions::from_mode(0o600)).expect("token mode");
        path
    }

    #[test]
    fn success_revokes_and_failure_keeps_retry_token() {
        for admin in [true, false] {
            let root = test_root();
            let dir = root.join("soda");
            fs::create_dir(&dir).expect("soda dir");
            let token_path = write_token(&root);
            let out = dir.join("dashboard.json");
            let stub = stub_server(admin, false, "synthetic-bootstrap-token-not-for-retention");
            let mut stdout: Vec<u8> = Vec::new();
            let err = setup(
                "https://forgejo.test/",
                &stub.url,
                &token_path.to_string_lossy(),
                &out,
                &root.join("postgres"),
                &mut stdout,
            );
            let calls = stub.calls.lock().expect("calls").clone();
            if admin {
                assert!(err.is_ok(), "setup failed: {err:?}");
                assert_eq!(calls, vec!["GET /api/v1/user", "DELETE /api/v1/user/token"]);
            } else {
                assert!(err.is_err(), "failing setup succeeded");
                assert_eq!(calls, vec!["GET /api/v1/user"]);
            }
        }
    }

    #[test]
    fn provisions_all_four_postgres_files() {
        let root = test_root();
        let dir = root.join("soda");
        fs::create_dir(&dir).expect("soda dir");
        let token_path = write_token(&root);
        let stub = stub_server(true, false, "synthetic-bootstrap-token-not-for-retention");
        let pg_dir = root.join("postgres");
        let mut stdout: Vec<u8> = Vec::new();
        setup(
            "https://forgejo.test/",
            &stub.url,
            &token_path.to_string_lossy(),
            &dir.join("dashboard.json"),
            &pg_dir,
            &mut stdout,
        )
        .expect("setup");
        let mut passwords = Vec::new();
        for role in ["super", "forgejo", "soda"] {
            let path = pg_dir.join(format!("{role}.passwd"));
            let mode = fs::metadata(&path).expect("passwd").permissions().mode() & 0o777;
            assert_eq!(mode, 0o600, "{role} mode");
            let pw = fs::read_to_string(&path)
                .expect("passwd")
                .trim()
                .to_string();
            assert_eq!(pw.len(), 64, "{role} length");
            assert!(
                pw.bytes()
                    .all(|b| b.is_ascii_hexdigit() && !b.is_ascii_uppercase()),
                "{role} hex"
            );
            passwords.push(pw);
        }
        assert_ne!(passwords[0], passwords[1]);
        assert_ne!(passwords[1], passwords[2]);
        assert_ne!(passwords[0], passwords[2]);
        let dsn_path = pg_dir.join("soda.dsn");
        let mode = fs::metadata(&dsn_path).expect("dsn").permissions().mode() & 0o777;
        assert_eq!(mode, 0o640);
        let dsn = fs::read_to_string(&dsn_path).expect("dsn");
        assert_eq!(
            dsn.trim(),
            format!(
                "postgres://soda:{}@/soda?host=/run/soda/postgres&sslmode=disable",
                passwords[2]
            )
        );
    }

    #[test]
    fn preserves_pre_existing_secrets_and_cleans_own_key() {
        let root = test_root();
        let dir = root.join("soda");
        fs::create_dir(&dir).expect("soda dir");
        let token_path = write_token(&root);
        let stub = stub_server(true, false, "synthetic-bootstrap-token-not-for-retention");
        let pg_dir = root.join("postgres");
        fs::create_dir_all(&pg_dir).expect("pg dir");
        fs::write(pg_dir.join("super.passwd"), "operator-owned-secret\n").expect("prior");
        let mut stdout: Vec<u8> = Vec::new();
        let err = setup(
            "https://forgejo.test/",
            &stub.url,
            &token_path.to_string_lossy(),
            &dir.join("dashboard.json"),
            &pg_dir,
            &mut stdout,
        );
        assert!(err.is_err(), "setup overwrote pre-existing secrets");
        assert_eq!(
            fs::read_to_string(pg_dir.join("super.passwd")).expect("prior"),
            "operator-owned-secret\n"
        );
        assert!(
            !dir.join("grant-key").exists(),
            "failed setup left its grant key"
        );
        assert!(
            !dir.join("dashboard.json").exists(),
            "failed setup published config"
        );
    }

    #[test]
    fn reuses_complete_pre_existing_secrets() {
        let root = test_root();
        let dir = root.join("soda");
        fs::create_dir(&dir).expect("soda dir");
        let token_path = write_token(&root);
        let stub = stub_server(true, false, "synthetic-bootstrap-token-not-for-retention");
        let pg_dir = root.join("postgres");
        fs::create_dir_all(&pg_dir).expect("pg dir");
        let soda_pw = "b".repeat(64);
        let seed = [
            ("super.passwd", "a".repeat(64) + "\n"),
            ("forgejo.passwd", "c".repeat(64) + "\n"),
            ("soda.passwd", soda_pw.clone() + "\n"),
            (
                "soda.dsn",
                format!(
                    "postgres://soda:{soda_pw}@/soda?host=/run/soda/postgres&sslmode=disable\n"
                ),
            ),
        ];
        for (name, value) in &seed {
            fs::write(pg_dir.join(name), value).expect("seed");
        }
        let mut stdout: Vec<u8> = Vec::new();
        setup(
            "https://forgejo.test/",
            &stub.url,
            &token_path.to_string_lossy(),
            &dir.join("dashboard.json"),
            &pg_dir,
            &mut stdout,
        )
        .expect("setup with complete secrets");
        for (name, value) in &seed {
            assert_eq!(
                &fs::read_to_string(pg_dir.join(name)).expect("seed"),
                value,
                "{name} regenerated"
            );
        }
        let config = fs::read_to_string(dir.join("dashboard.json")).expect("config");
        assert!(config.contains(&pg_dir.join("soda.dsn").to_string_lossy().into_owned()));
    }

    #[test]
    fn credential_boundary_matrix() {
        for name in [
            "new",
            "unrelated-file",
            "existing-config",
            "existing-key",
            "non-admin",
            "current-denied",
            "missing-token",
        ] {
            let root = test_root();
            let dir = root.join("soda");
            fs::create_dir(&dir).expect("soda dir");
            let token_path = root.join("operator-input");
            if name != "missing-token" {
                fs::write(&token_path, "synthetic-bootstrap-token-not-for-retention\n")
                    .expect("token");
                fs::set_permissions(&token_path, fs::Permissions::from_mode(0o600))
                    .expect("token mode");
            }
            let out = dir.join("dashboard.json");
            let retained = match name {
                "unrelated-file" => Some(dir.join("operator-notes")),
                "existing-config" => Some(out.clone()),
                "existing-key" => Some(dir.join("grant-key")),
                _ => None,
            };
            if let Some(path) = &retained {
                fs::write(path, "preserve original bytes\n").expect("retained");
                fs::set_permissions(path, fs::Permissions::from_mode(0o600))
                    .expect("retained mode");
            }
            let stub = stub_server(
                name != "non-admin",
                name == "current-denied",
                "synthetic-bootstrap-token-not-for-retention",
            );
            let mut stdout: Vec<u8> = Vec::new();
            let err = setup(
                "https://forgejo.test/",
                &stub.url,
                &token_path.to_string_lossy(),
                &out,
                &root.join("postgres"),
                &mut stdout,
            );
            let output = String::from_utf8_lossy(&stdout).into_owned();
            assert!(
                !output.contains("synthetic-bootstrap-token"),
                "{name}: token in stdout"
            );
            if let Err(err) = &err {
                assert!(
                    !err.contains("synthetic-bootstrap-token"),
                    "{name}: token in error"
                );
            }
            let success = name == "new" || name == "unrelated-file";
            assert_eq!(err.is_ok(), success, "{name}: success={}", err.is_ok());
            let calls = stub.calls.lock().expect("calls").clone();
            let mut want_calls: Vec<String> = Vec::new();
            if name != "existing-config" && name != "missing-token" {
                want_calls.push("GET /api/v1/user".to_string());
            }
            if success {
                want_calls.push("DELETE /api/v1/user/token".to_string());
            }
            assert_eq!(calls, want_calls, "{name}: provider calls");
            if let Some(path) = &retained {
                assert_eq!(
                    fs::read_to_string(path).expect("retained"),
                    "preserve original bytes\n",
                    "{name}: retained changed"
                );
                assert_eq!(
                    fs::metadata(path).expect("retained").permissions().mode() & 0o777,
                    0o600
                );
            }
            for entry in fs::read_dir(&dir).expect("dir") {
                let entry = entry.expect("entry");
                let data = fs::read_to_string(entry.path()).expect("entry data");
                assert!(
                    !data.contains("synthetic-bootstrap-token"),
                    "{name}: token retained"
                );
            }
            if !success {
                if name != "existing-config" {
                    assert!(!out.exists(), "{name}: failed setup published config");
                }
                continue;
            }
            // dashboard.json parses back with the operator identity and the
            // trimmed browser origin, and the grant key decodes to 32 bytes.
            let config = fs::read_to_string(&out).expect("config");
            assert!(config.contains("\"operator_id\":42"), "{config}");
            assert!(
                config.contains("\"forgejo_url\":\"https://forgejo.test\""),
                "{config}"
            );
            assert!(config.contains("\"identity_socket\":\"\""), "{config}");
            assert_eq!(
                fs::metadata(&out).expect("config").permissions().mode() & 0o777,
                0o600
            );
            let key_path = dir.join("grant-key");
            assert_eq!(
                fs::metadata(&key_path).expect("key").permissions().mode() & 0o777,
                0o600
            );
            let key_text = fs::read_to_string(&key_path)
                .expect("key")
                .trim()
                .to_string();
            assert_eq!(key_text.len(), 44, "grant key length");
            assert!(key_text.ends_with('='), "grant key padding");
        }
    }

    #[test]
    fn dashboard_config_bytes_match_go_encoder() {
        // Byte-exact reference captured from Go's json.Encoder over
        // config.Config for the same logical record (see PR07 notes).
        let golden = "{\"listen\":\"127.0.0.1:8080\",\"forgejo_url\":\"https://forgejo.test\",\"forgejo_internal_url\":\"http://127.0.0.1:3000\",\"database_dsn_file\":\"/etc/soda/postgres/soda.dsn\",\"host_socket\":\"/run/soda/host.sock\",\"identity_socket\":\"\",\"grant_key_file\":\"/etc/soda/grant-key\",\"operator_id\":42,\"forgejo_background_socket\":\"\",\"forgejo_background_host_uid\":null,\"forgejo_background_credential_file\":\"\",\"forgejo_review_credential_file\":\"\",\"forgejo_merge_credential_file\":\"\",\"factory_intake_secret_file\":\"\",\"factory_publication_root\":\"\"}\n";
        let encoded = encode_dashboard_config(
            "127.0.0.1:8080",
            "https://forgejo.test",
            "http://127.0.0.1:3000",
            "/etc/soda/postgres/soda.dsn",
            "/run/soda/host.sock",
            "",
            "/etc/soda/grant-key",
            42,
        );
        assert_eq!(String::from_utf8(encoded).expect("utf8"), golden);
    }

    #[test]
    fn admits_only_valid_setup_paths() {
        let root = test_root();
        let out = root.join("soda").join("dashboard.json");
        assert!(admit_setup_paths("https://forgejo.test", "http://127.0.0.1:3000", &out).is_ok());
        assert!(admit_setup_paths("http://forgejo.test", "http://127.0.0.1:3000", &out).is_err());
        assert!(
            admit_setup_paths("https://forgejo.test/x", "http://127.0.0.1:3000", &out).is_err()
        );
        assert!(
            admit_setup_paths("https://forgejo.test", "http://127.0.0.1:3000/x", &out).is_err()
        );
        assert!(
            admit_setup_paths("https://user@forgejo.test", "http://127.0.0.1:3000", &out).is_err()
        );
        assert!(admit_setup_paths(
            "https://forgejo.test",
            "http://127.0.0.1:3000",
            Path::new("relative.json")
        )
        .is_err());
        fs::create_dir_all(out.parent().expect("parent")).expect("parent");
        fs::write(&out, "{}\n").expect("existing");
        assert!(admit_setup_paths("https://forgejo.test", "http://127.0.0.1:3000", &out).is_err());
    }

    #[test]
    fn forgejo_user_decode_matches_client_rules() {
        let user = decode_user(
            b"{\"id\": 42, \"login\": \"soda-tester\", \"is_admin\": true, \"extra\": [1]}",
        )
        .expect("valid");
        assert_eq!(user.id, 42);
        assert!(user.admin);
        assert!(decode_user(b"null").is_err());
        assert!(decode_user(b"{\"id\": 1} garbage").is_err());
        assert!(decode_user(b"{\"id\": \"42\", \"is_admin\": true}").is_err());
        assert!(decode_user(b"{\"id\": 42.5, \"is_admin\": true}").is_err());
        assert!(decode_user(b"{\"id\": 42, \"is_admin\": \"yes\"}").is_err());
        assert!(decode_user(b"[1,2]").is_err());
        let big = vec![b'x'; FORGEJO_RESPONSE_LIMIT + 1];
        assert_eq!(
            decode_user(&big).expect_err("oversize"),
            "forgejo response exceeds the supported size"
        );
    }

    #[test]
    fn disagreeing_secrets_are_reported() {
        let root = test_root();
        let pg_dir = root.join("postgres");
        fs::create_dir_all(&pg_dir).expect("pg dir");
        fs::write(pg_dir.join("super.passwd"), "a\n").expect("seed");
        fs::write(pg_dir.join("forgejo.passwd"), "c\n").expect("seed");
        fs::write(pg_dir.join("soda.passwd"), "b\n").expect("seed");
        fs::write(pg_dir.join("soda.dsn"), "postgres://soda:other@/soda\n").expect("seed");
        let err = reuse_postgres_secrets(&pg_dir).expect_err("disagree");
        assert!(err.contains("exist but disagree"), "{err}");
        // Partial sets report None so the caller fails O_EXCL instead.
        fs::remove_file(pg_dir.join("soda.dsn")).expect("remove");
        assert!(reuse_postgres_secrets(&pg_dir).expect("partial").is_none());
    }

    #[test]
    fn provision_only_reports_dsn_path() {
        let root = test_root();
        let pg_dir = root.join("postgres");
        let (_, dsn) = provision_postgres_secrets(&pg_dir).expect("provision");
        assert_eq!(dsn, pg_dir.join("soda.dsn"));
        // Second run reuses without regenerating.
        let before = fs::read_to_string(&dsn).expect("dsn");
        let (created, again) = provision_postgres_secrets(&pg_dir).expect("reuse");
        assert!(created.is_empty());
        assert_eq!(again, dsn);
        assert_eq!(fs::read_to_string(&dsn).expect("dsn"), before);
    }

    #[test]
    fn non_root_run_reports_operator_access() {
        if euid() == 0 {
            return;
        }
        // run() checks euid before parsing flags; only the message is pinned.
        assert_ne!(euid(), 0);
    }

    #[test]
    fn flag_parsing_matches_go_setup() {
        let args: Vec<String> = [
            "--forgejo-url",
            "https://x.test",
            "--token-file",
            "/t",
            "--out",
            "/o",
        ]
        .iter()
        .map(|s| s.to_string())
        .collect();
        let opts = parse_args(&args).expect("parse");
        assert_eq!(opts.forgejo_url, "https://x.test");
        assert_eq!(opts.forgejo_internal_url, "http://127.0.0.1:3000");
        assert!(!opts.provision_db_only);
        let args: Vec<String> = ["--provision-db-only"]
            .iter()
            .map(|s| s.to_string())
            .collect();
        assert!(parse_args(&args).expect("bare bool").provision_db_only);
        let args: Vec<String> = ["--bogus"].iter().map(|s| s.to_string()).collect();
        assert!(parse_args(&args).is_err());
        let args: Vec<String> = ["--socket"].iter().map(|s| s.to_string()).collect();
        assert!(parse_args(&args).is_err());
    }

    #[test]
    fn json_string_escapes_match_go() {
        let mut out = String::new();
        push_json_string(&mut out, "a<b>&\"c\"\n");
        assert_eq!(out, "\"a\\u003cb\\u003e\\u0026\\\"c\\\"\\n\"");
    }

    #[test]
    fn base64_and_hex_vectors() {
        assert_eq!(hex_encode(b"\x00\xff\x10abc"), "00ff10616263");
        assert_eq!(base64_encode(b""), "");
        assert_eq!(base64_encode(b"f"), "Zg==");
        assert_eq!(base64_encode(b"fo"), "Zm8=");
        assert_eq!(base64_encode(b"foo"), "Zm9v");
        assert_eq!(base64_encode(b"foob"), "Zm9vYg==");
        assert_eq!(base64_encode(&[0u8; 32]).len(), 44);
    }
}

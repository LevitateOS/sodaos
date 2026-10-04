//! Soda OS host operator console welcome. Read-only, operator-only.
//! Optional config path supports inspection; the installed login hook uses
//! the default. Preserves the predecessor's main-table uplink lookup.
//!
//! Rust port of appliance/bin/soda-console-welcome, including its embedded
//! Python config/origin validation. Std-only (`cargo build --offline`).
//! Every banner line, subprocess argv shape, validation rule and exit code
//! matches the shell original; subprocesses are resolved via PATH so the
//! installed command doubles keep working.

use std::env;
use std::fs;
use std::process::{Command, Stdio};

fn main() {
    std::process::exit(run());
}

fn run() -> i32 {
    let argv: Vec<String> = env::args().collect();
    if argv.len() > 2 {
        let prog = &argv[0];
        eprintln!("usage: {prog} [DASHBOARD_CONFIG]");
        return 2;
    }
    // `[ "$(id -u)" = 0 ] || exit 0`: a real subprocess, so PATH doubles
    // (and a missing id) behave exactly as in the shell.
    if capture("id", &["-u"], &[]).trim_end_matches('\n') != "0" {
        return 0;
    }
    let config = argv
        .get(1)
        .map(String::as_str)
        .unwrap_or("/etc/soda/dashboard.json");

    println!("\nWelcome to Soda OS — host operator console");
    println!("The operating system for software factories.");
    println!(
        "Run your coding agents on your infrastructure, with controlled access and reviewable results."
    );
    println!("Hostname: {}", hostname());

    // Main-table lookup: an exit node may install routes in a separate
    // table. Connected Ethernet/Wi-Fi without a default route counts too.
    let mut devices: Vec<String> = Vec::new();
    for line in capture(
        "ip",
        &["-o", "-4", "route", "show", "default", "table", "main"],
        &[],
    )
    .lines()
    {
        let fields: Vec<&str> = line.split_whitespace().collect();
        for i in 0..fields.len().saturating_sub(1) {
            if fields[i] == "dev" {
                devices.push(fields[i + 1].to_string());
            }
        }
    }
    for line in capture(
        "nmcli",
        &["-t", "-f", "DEVICE,TYPE,STATE", "device", "status"],
        &[("LC_ALL", "C")],
    )
    .lines()
    {
        let fields: Vec<&str> = line.split(':').collect();
        if fields.len() >= 3
            && fields[2] == "connected"
            && (fields[1] == "ethernet" || fields[1] == "wifi")
        {
            devices.push(fields[0].to_string());
        }
    }
    // `awk '!seen[$0]++'`: exact-string dedupe, first occurrence wins.
    let mut seen = std::collections::HashSet::new();
    devices.retain(|d| seen.insert(d.clone()));

    let mut found = false;
    let mut tunnel_host = String::new();
    for device in &devices {
        // `case "$device" in lo|tailscale*) continue`.
        if device == "lo" || device.starts_with("tailscale") {
            continue;
        }
        let out = capture(
            "ip",
            &["-o", "-4", "addr", "show", "dev", device, "scope", "global"],
            &[],
        );
        for line in out.lines() {
            // `awk '{sub(/\/.*/, "", $4); print $4}'`, empties dropped by
            // the shell word split.
            let field = line.split_whitespace().nth(3).unwrap_or("");
            let address = field.split('/').next().unwrap_or("");
            if address.is_empty() {
                continue;
            }
            found = true;
            if tunnel_host.is_empty() {
                tunnel_host = address.to_string();
            }
            println!("Observed local IPv4 ({device}): {address}");
        }
    }
    if !found {
        println!("No local uplink IPv4 address is currently available.");
    }
    if !tunnel_host.is_empty() {
        println!("\nCockpit: https://{tunnel_host}:9090");
        println!(
            "Self-signed certificate; expect a browser warning. Root password login is enabled."
        );
    } else {
        println!(
            "\nCockpit: unavailable until a local uplink IPv4 address exists; configure networking first."
        );
    }

    render_config(config, &tunnel_host);

    // `command -v soda-tailnet`, run with inherited stdio; failures ignored.
    if have_command("soda-tailnet") {
        let _ = Command::new("soda-tailnet").status();
    } else {
        println!("\nTailnet status helper is unavailable; inspect native tailscaled.service.");
    }

    println!("\nProject developers use their project IPs, not host Linux accounts.");
    println!("People authorize factory work and merge the final verified commit in Forgejo.");
    println!(
        "Persistent human projects support development and intervention; run cleanup leaves them intact."
    );
    println!("Host Tailnet enrollment does not establish project-subnet routing.");
    println!("No firewall, listener, enrollment or service configuration was changed.");
    0
}

/// `$(...)`: stdout captured, stderr discarded, "" on any failure.
fn capture(program: &str, args: &[&str], extra_env: &[(&str, &str)]) -> String {
    let mut command = Command::new(program);
    command
        .args(args)
        .stdout(Stdio::piped())
        .stderr(Stdio::null());
    for (key, value) in extra_env {
        command.env(key, value);
    }
    command
        .output()
        .ok()
        .filter(|out| out.status.success())
        .map(|out| String::from_utf8_lossy(&out.stdout).into_owned())
        .unwrap_or_default()
}

/// `command -v`: an executable file on PATH.
fn have_command(program: &str) -> bool {
    env::var_os("PATH").is_some_and(|paths| {
        env::split_paths(&paths).any(|dir| {
            let candidate = dir.join(program);
            fs::metadata(&candidate).is_ok_and(|info| {
                use std::os::unix::fs::PermissionsExt;
                info.is_file() && info.permissions().mode() & 0o111 != 0
            })
        })
    })
}

fn hostname() -> String {
    // `$(...)` strips trailing newlines only, never other whitespace.
    let host = capture("hostnamectl", &["--static"], &[]);
    let host = host.trim_end_matches('\n');
    if !host.is_empty() {
        return host.to_string();
    }
    capture("uname", &["-n"], &[])
        .trim_end_matches('\n')
        .to_string()
}

/// The embedded Python block: read only public origins and the dashboard
/// port, never credentials or the whole document.
fn render_config(config: &str, tunnel_host: &str) {
    let host = if tunnel_host == "HOST" {
        ""
    } else {
        tunnel_host
    };
    let raw = fs::read(config).unwrap_or_default();
    let text = String::from_utf8_lossy(&raw);
    let parsed = parse_top_object(&text);
    let rendered = parsed.as_ref().and_then(|fields| {
        // `config.get('listen', default)`: absent takes the default, but a
        // present non-string is invalid, never defaulted.
        let listen = match fields.get("listen") {
            None => "127.0.0.1:8080".to_string(),
            Some(Some(value)) => value.clone(),
            Some(None) => return None,
        };
        let (listen_host, listen_port) = valid_listen(&listen)?;
        let url = match fields.get("forgejo_url") {
            Some(Some(url)) => url,
            _ => return None,
        };
        let display = valid_origin(url)?;
        Some((listen_host, listen_port, display))
    });
    match rendered {
        None => {
            println!(
                "\nBrowser origins are not configured or cannot be read; complete operator setup."
            );
            println!("\nForgejo installer is loopback-first: http://127.0.0.1:3000");
            if !host.is_empty() {
                println!("From your client, tunnel over SSH:");
                println!("  ssh -N -L 33000:127.0.0.1:3000 root@{host}");
                println!("Then open http://localhost:33000 in that client browser.");
            } else {
                println!(
                    "No local uplink address is available for an SSH tunnel; configure networking first."
                );
            }
        }
        Some((listen_host, listen_port, forgejo)) => {
            println!("\nDashboard is loopback-first: http://{listen_host}:{listen_port}");
            if !host.is_empty() {
                println!("From your client, tunnel over SSH:");
                println!("  ssh -N -L {listen_port}:{listen_host}:{listen_port} root@{host}");
                println!("Then open http://{listen_host}:{listen_port} in that client browser.");
            } else {
                println!(
                    "No local uplink address is available for an SSH tunnel; configure networking first."
                );
            }
            println!("\nConfigured browser origins (not a listener or reachability check):");
            println!("  Forgejo / Sodaspaces: {forgejo}");
        }
    }
}

/// Top-level JSON object as string-field map (last duplicate wins, like
/// Python's dict). None = parse failure or non-object. Values decode to
/// Some(string) for JSON strings, None for any other JSON value.
fn parse_top_object(text: &str) -> Option<std::collections::HashMap<String, Option<String>>> {
    let bytes = text.as_bytes();
    let mut parser = JsonParser {
        bytes,
        pos: 0,
        depth: 0,
    };
    parser.skip_ws();
    if parser.peek() != Some(b'{') {
        return None;
    }
    let fields = parser.parse_object_top()?;
    parser.skip_ws();
    if parser.pos != bytes.len() {
        return None; // json.loads rejects trailing data ("Extra data").
    }
    Some(fields)
}

struct JsonParser<'a> {
    bytes: &'a [u8],
    pos: usize,
    depth: u32,
}

impl<'a> JsonParser<'a> {
    fn peek(&self) -> Option<u8> {
        self.bytes.get(self.pos).copied()
    }

    fn skip_ws(&mut self) {
        while matches!(self.peek(), Some(b' ' | b'\t' | b'\n' | b'\r')) {
            self.pos += 1;
        }
    }

    fn literal(&mut self, word: &[u8]) -> bool {
        if self.bytes[self.pos..].starts_with(word) {
            self.pos += word.len();
            true
        } else {
            false
        }
    }

    fn parse_object_top(&mut self) -> Option<std::collections::HashMap<String, Option<String>>> {
        // pos at '{'.
        self.pos += 1;
        let mut fields = std::collections::HashMap::new();
        self.skip_ws();
        if self.peek() == Some(b'}') {
            self.pos += 1;
            return Some(fields);
        }
        loop {
            self.skip_ws();
            let key = self.parse_string()?;
            self.skip_ws();
            if self.peek() != Some(b':') {
                return None;
            }
            self.pos += 1;
            self.skip_ws();
            let value = self.parse_value_top()?;
            fields.insert(key, value);
            self.skip_ws();
            match self.peek() {
                Some(b',') => {
                    self.pos += 1;
                }
                Some(b'}') => {
                    self.pos += 1;
                    return Some(fields);
                }
                _ => return None,
            }
        }
    }

    /// Top-level member value: decoded for strings, validated-and-discarded
    /// otherwise (objects/arrays recurse for syntax validation, as json.loads
    /// parses the whole document).
    fn parse_value_top(&mut self) -> Option<Option<String>> {
        match self.peek()? {
            b'"' => Some(Some(self.parse_string()?)),
            _ => {
                self.parse_any()?;
                Some(None)
            }
        }
    }

    fn parse_any(&mut self) -> Option<()> {
        // json.loads has no depth limit (deep input raises RecursionError,
        // which the script does NOT catch); cap here and treat overflow as
        // a parse failure instead of a traceback.
        if self.depth > 1000 {
            return None;
        }
        self.depth += 1;
        let result = self.parse_any_inner();
        self.depth -= 1;
        result
    }

    fn parse_any_inner(&mut self) -> Option<()> {
        match self.peek()? {
            b'"' => {
                self.parse_string()?;
                Some(())
            }
            b'{' => {
                self.pos += 1;
                self.skip_ws();
                if self.peek() == Some(b'}') {
                    self.pos += 1;
                    return Some(());
                }
                loop {
                    self.skip_ws();
                    self.parse_string()?;
                    self.skip_ws();
                    if self.peek() != Some(b':') {
                        return None;
                    }
                    self.pos += 1;
                    self.skip_ws();
                    self.parse_any()?;
                    self.skip_ws();
                    match self.peek() {
                        Some(b',') => self.pos += 1,
                        Some(b'}') => {
                            self.pos += 1;
                            return Some(());
                        }
                        _ => return None,
                    }
                }
            }
            b'[' => {
                self.pos += 1;
                self.skip_ws();
                if self.peek() == Some(b']') {
                    self.pos += 1;
                    return Some(());
                }
                loop {
                    self.skip_ws();
                    self.parse_any()?;
                    self.skip_ws();
                    match self.peek() {
                        Some(b',') => self.pos += 1,
                        Some(b']') => {
                            self.pos += 1;
                            return Some(());
                        }
                        _ => return None,
                    }
                }
            }
            b't' => self.literal(b"true").then_some(()),
            b'f' => self.literal(b"false").then_some(()),
            b'n' => self.literal(b"null").then_some(()),
            // json.loads accepts NaN/Infinity/-Infinity; all land in the
            // except branch later as non-strings.
            b'N' => self.literal(b"NaN").then_some(()),
            b'I' => self.literal(b"Infinity").then_some(()),
            b'-' => {
                if self.bytes[self.pos..].starts_with(b"-Infinity") {
                    self.pos += "-Infinity".len();
                    Some(())
                } else {
                    self.parse_number()
                }
            }
            b'0'..=b'9' => self.parse_number(),
            _ => None,
        }
    }

    fn parse_number(&mut self) -> Option<()> {
        let start = self.pos;
        if self.peek() == Some(b'-') {
            self.pos += 1;
        }
        match self.peek() {
            Some(b'0') => self.pos += 1,
            Some(b'1'..=b'9') => {
                while matches!(self.peek(), Some(b'0'..=b'9')) {
                    self.pos += 1;
                }
            }
            _ => return None,
        }
        if self.peek() == Some(b'.') {
            self.pos += 1;
            if !matches!(self.peek(), Some(b'0'..=b'9')) {
                return None;
            }
            while matches!(self.peek(), Some(b'0'..=b'9')) {
                self.pos += 1;
            }
        }
        if matches!(self.peek(), Some(b'e' | b'E')) {
            self.pos += 1;
            if matches!(self.peek(), Some(b'+' | b'-')) {
                self.pos += 1;
            }
            if !matches!(self.peek(), Some(b'0'..=b'9')) {
                return None;
            }
            while matches!(self.peek(), Some(b'0'..=b'9')) {
                self.pos += 1;
            }
        }
        if self.pos == start {
            return None;
        }
        Some(())
    }

    /// JSON string with full escape decoding, including \uXXXX surrogate
    /// pairs. Lone surrogates fail (Python keeps them, then crashes printing
    /// a validating URL — a traceback, not a contract).
    fn parse_string(&mut self) -> Option<String> {
        if self.peek() != Some(b'"') {
            return None;
        }
        self.pos += 1;
        let mut out = String::new();
        loop {
            let byte = self.peek()?;
            match byte {
                b'"' => {
                    self.pos += 1;
                    return Some(out);
                }
                b'\\' => {
                    self.pos += 1;
                    match self.peek()? {
                        b'"' => {
                            out.push('"');
                            self.pos += 1;
                        }
                        b'\\' => {
                            out.push('\\');
                            self.pos += 1;
                        }
                        b'/' => {
                            out.push('/');
                            self.pos += 1;
                        }
                        b'b' => {
                            out.push('\u{8}');
                            self.pos += 1;
                        }
                        b'f' => {
                            out.push('\u{c}');
                            self.pos += 1;
                        }
                        b'n' => {
                            out.push('\n');
                            self.pos += 1;
                        }
                        b'r' => {
                            out.push('\r');
                            self.pos += 1;
                        }
                        b't' => {
                            out.push('\t');
                            self.pos += 1;
                        }
                        b'u' => {
                            self.pos += 1;
                            let high = self.parse_hex4()?;
                            if (0xD800..0xDC00).contains(&high) {
                                if self.bytes.get(self.pos..self.pos + 2) == Some(b"\\u") {
                                    self.pos += 2;
                                    let low = self.parse_hex4()?;
                                    if (0xDC00..0xE000).contains(&low) {
                                        let code =
                                            0x10000 + ((high - 0xD800) << 10) + (low - 0xDC00);
                                        out.push(char::from_u32(code)?);
                                    } else {
                                        return None;
                                    }
                                } else {
                                    return None;
                                }
                            } else if (0xDC00..0xE000).contains(&high) {
                                return None;
                            } else {
                                out.push(char::from_u32(high)?);
                            }
                        }
                        _ => return None,
                    }
                }
                0x00..=0x1F => return None, // json.loads rejects literal controls.
                _ => {
                    // Regular UTF-8 scalar (input is already valid UTF-8).
                    let rest = &self.bytes[self.pos..];
                    let text = std::str::from_utf8(rest).ok()?;
                    let ch = text.chars().next()?;
                    out.push(ch);
                    self.pos += ch.len_utf8();
                }
            }
        }
    }

    fn parse_hex4(&mut self) -> Option<u32> {
        if self.pos + 4 > self.bytes.len() {
            return None;
        }
        let mut value = 0u32;
        for i in 0..4 {
            value = value * 16 + hex_value(self.bytes[self.pos + i])?;
        }
        self.pos += 4;
        Some(value)
    }
}

fn hex_value(byte: u8) -> Option<u32> {
    match byte {
        b'0'..=b'9' => Some((byte - b'0') as u32),
        b'a'..=b'f' => Some((byte - b'a' + 10) as u32),
        b'A'..=b'F' => Some((byte - b'A' + 10) as u32),
        _ => None,
    }
}

/// `listen` of "host:port" with a loopback IP literal, exactly as the
/// Python validates it: one colon, numeric port 1..=65535, IPv4 127/8.
/// (IPv6 can never pass: any IPv6 form holds more than one colon.)
fn valid_listen(listen: &str) -> Option<(String, String)> {
    if listen.bytes().filter(|b| *b == b':').count() != 1 {
        return None;
    }
    let (host, port) = listen.rsplit_once(':')?;
    // Python's isdigit()+int() also accept non-ASCII decimal digits; only
    // ASCII is reachable in real configs, and only ASCII is matched here.
    if port.is_empty() || !port.bytes().all(|b| b.is_ascii_digit()) {
        return None;
    }
    let number: u32 = port.parse().ok()?;
    if !(1..=65535).contains(&number) {
        return None;
    }
    if !is_loopback_ipv4(host) {
        return None;
    }
    Some((host.to_string(), port.to_string()))
}

/// `ipaddress.ip_address(host).is_loopback` for the reachable shape: modern
/// Python requires 4 dot-decimal parts with no leading zeros.
fn is_loopback_ipv4(host: &str) -> bool {
    let parts: Vec<&str> = host.split('.').collect();
    if parts.len() != 4 {
        return false;
    }
    for (i, part) in parts.iter().enumerate() {
        if part.is_empty() || part.len() > 3 || !part.bytes().all(|b| b.is_ascii_digit()) {
            return false;
        }
        if part.len() > 1 && part.starts_with('0') {
            return false;
        }
        let octet: u32 = part.parse().unwrap_or(256);
        if octet > 255 {
            return false;
        }
        if i == 0 && octet != 127 {
            return false;
        }
    }
    true
}

/// The `forgejo_url` checks, replicating urlsplit semantics: https scheme
/// (case-insensitive, as urlsplit lowercases it), non-empty host, no
/// userinfo, empty query/fragment, path "" or "/". Returns the display form
/// (`value.rstrip('/')`).
fn valid_origin(value: &str) -> Option<String> {
    if value
        .chars()
        .any(|c| c.is_whitespace() || c < '\u{20}' || c == '\u{7f}')
    {
        return None;
    }
    let (scheme, rest) = value.split_once(':')?;
    if !scheme.eq_ignore_ascii_case("https") {
        return None;
    }
    let after = rest.strip_prefix("//")?;
    // Authority runs to the first '/', '?' or '#'.
    let end = after.find(['/', '?', '#']).unwrap_or(after.len());
    let (authority, remainder) = after.split_at(end);
    // Any '@' means username (possibly empty) is not None.
    if authority.contains('@') {
        return None;
    }
    let (host, port) = split_host_port(authority)?;
    if host.is_empty() {
        return None;
    }
    // `_ = url.port`: malformed or out-of-range ports raise.
    if let Some(port) = port {
        if !valid_url_port(port) {
            return None;
        }
    }
    // Query and fragment must be empty (a bare '?' or '#' still passes).
    let (before_frag, fragment) = match remainder.split_once('#') {
        Some((before, after)) => (before, after),
        None => (remainder, ""),
    };
    if !fragment.is_empty() {
        return None;
    }
    let (path, query) = match before_frag.split_once('?') {
        Some((before, after)) => (before, after),
        None => (before_frag, ""),
    };
    if !query.is_empty() || (!path.is_empty() && path != "/") {
        return None;
    }
    Some(value.trim_end_matches('/').to_string())
}

fn split_host_port(authority: &str) -> Option<(&str, Option<&str>)> {
    if let Some(bracketed) = authority.strip_prefix('[') {
        let (host, rest) = bracketed.split_once(']')?;
        match rest.strip_prefix(':') {
            Some(port) => Some((host, Some(port))),
            None if rest.is_empty() => Some((host, None)),
            _ => None,
        }
    } else {
        match authority.split_once(':') {
            Some((host, port)) => {
                if port.contains(':') {
                    return None;
                }
                Some((host, Some(port)))
            }
            None => Some((authority, None)),
        }
    }
}

/// `url.port`: ASCII digits only (Python rejects signs, spaces and
/// non-ASCII digits with "Port could not be cast"), 0..=65535. Empty port
/// (from "host:") is None upstream and never reaches here.
fn valid_url_port(port: &str) -> bool {
    if port.is_empty() || !port.bytes().all(|b| b.is_ascii_digit()) {
        return false;
    }
    port.parse::<u32>().is_ok_and(|number| number <= 65535)
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn top_object_extracts_strings_last_wins() {
        let fields = parse_top_object(r#"{"a": "1", "b": 2, "a": "3", "c": null}"#).unwrap();
        assert_eq!(fields.get("a"), Some(&Some("3".to_string())));
        assert_eq!(fields.get("b"), Some(&None));
        assert_eq!(fields.get("c"), Some(&None));
        assert!(parse_top_object(r#"{"unclosed""#).is_none());
        assert!(parse_top_object(r#"[1,2]"#).is_none());
        assert!(parse_top_object(r#"{"a": "1"} trailing"#).is_none());
        assert!(parse_top_object(r#"{"a": NaN, "b": Infinity, "c": -Infinity}"#).is_some());
    }

    #[test]
    fn string_escapes_decode_like_python() {
        let fields = parse_top_object(r#"{"a": "x\"y\\z\/\b\f\n\r\té☃"}"#).unwrap();
        assert_eq!(
            fields.get("a"),
            Some(&Some("x\"y\\z/\u{8}\u{c}\n\r\té☃".to_string()))
        );
        let fields = parse_top_object(r#"{"a": "A𝄞B"}"#).unwrap();
        assert_eq!(fields.get("a"), Some(&Some("A𝄞B".to_string())));
        assert!(parse_top_object("{\"a\": \"\\ud800\"}").is_none());
        assert!(parse_top_object("{\"a\": \"line\nbreak\"}").is_none());
    }

    #[test]
    fn listen_shapes() {
        assert_eq!(
            valid_listen("127.0.0.1:8080"),
            Some(("127.0.0.1".to_string(), "8080".to_string()))
        );
        assert!(valid_listen("127.1.2.3:1").is_some());
        for bad in [
            "192.168.1.10:8080",
            "127.0.0.1:notaport",
            "127.0.0.1:0",
            "127.0.0.1:65536",
            "127.0.0.1:8080:extra",
            "42",
            "localhost:8080",
            "127.0.0.01:8080",
            "::1:8080",
            "",
        ] {
            assert!(valid_listen(bad).is_none(), "{bad:?}");
        }
    }

    #[test]
    fn origin_shapes() {
        assert_eq!(
            valid_origin("https://forgejo.example.test"),
            Some("https://forgejo.example.test".to_string())
        );
        assert_eq!(
            valid_origin("https://forgejo.example.test/"),
            Some("https://forgejo.example.test".to_string())
        );
        assert!(valid_origin("HTTPS://forgejo.example.test").is_some());
        assert!(valid_origin("https://forgejo.example.test:8443").is_some());
        assert!(valid_origin("https://forgejo.example.test/?").is_some());
        for bad in [
            "https://name:private-value@soda.example.test",
            "https://@soda.example.test",
            "https://soda.example.test:bad-port",
            "https://soda.example.test:65536",
            "http://soda.example.test",
            "https://soda.example.test/path",
            "https://soda.example.test?q=1",
            "https://soda.example.test#frag",
            "https://soda.example.test/pa th",
            "https://",
            "not a url",
            "",
        ] {
            assert!(valid_origin(bad).is_none(), "{bad:?}");
        }
    }
}

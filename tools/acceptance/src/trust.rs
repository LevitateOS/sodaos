//! Fixture trust and inline data, mirroring `trust.go` with OpenSSH
//! delegation.
//!
//! The Go owner parses SSH keys in-process (`x/crypto/ssh`). Hand-rolling
//! SSH key formats and `known_hosts` matching (including hashed entries)
//! would risk subtle verification gaps, so this port delegates those exact
//! operations to the host's OpenSSH: `ssh-keygen -y` parses the embedded
//! private key, and `ssh-keygen -F` matches the pinned entry. Gzip stays
//! in-process via `flate2` so the byte bounds and error taxonomy match
//! exactly.

use soda_json::JsonValue;

use crate::command::Remote;
use crate::error::Error;
use crate::files;

/// Inline gzip bound: 1 MiB of decoded bytes.
const INLINE_GZIP_LIMIT: u64 = 1 << 20;

const B64_ALPHABET: &[u8; 64] = b"ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/";

fn b64_value(byte: u8) -> Option<u8> {
    B64_ALPHABET
        .iter()
        .position(|b| *b == byte)
        .map(|i| i as u8)
}

/// Strict standard-alphabet base64 decode, like Go's `StdEncoding`.
pub fn decode_base64(input: &str) -> Result<Vec<u8>, Error> {
    let bytes = input.as_bytes();
    if !bytes.len().is_multiple_of(4) {
        return Err(Error::msg("invalid base64"));
    }
    let mut out = Vec::with_capacity(bytes.len() / 4 * 3);
    let (chunks, _) = bytes.as_chunks::<4>();
    let total = chunks.len();
    for (index, chunk) in chunks.iter().enumerate() {
        let last = index + 1 == total;
        let mut values = [0u8; 4];
        let mut padding = 0;
        for (i, byte) in chunk.iter().enumerate() {
            if *byte == b'=' {
                if !last || i < 2 {
                    return Err(Error::msg("invalid base64"));
                }
                padding += 1;
                values[i] = 0;
            } else {
                if padding > 0 {
                    return Err(Error::msg("invalid base64"));
                }
                values[i] = b64_value(*byte).ok_or_else(|| Error::msg("invalid base64"))?;
            }
        }
        let triple = ((values[0] as u32) << 18)
            | ((values[1] as u32) << 12)
            | ((values[2] as u32) << 6)
            | values[3] as u32;
        out.push((triple >> 16) as u8);
        if padding < 2 {
            out.push((triple >> 8) as u8);
        }
        if padding == 0 {
            out.push(triple as u8);
        }
    }
    Ok(out)
}

/// Standard base64 encode, for tests only. Production never emits secrets.
#[cfg(test)]
pub fn encode_base64(input: &[u8]) -> String {
    let mut out = String::new();
    for chunk in input.chunks(3) {
        let mut block = [0u8; 3];
        block[..chunk.len()].copy_from_slice(chunk);
        let triple = ((block[0] as u32) << 16) | ((block[1] as u32) << 8) | block[2] as u32;
        out.push(B64_ALPHABET[(triple >> 18) as usize] as char);
        out.push(B64_ALPHABET[((triple >> 12) & 63) as usize] as char);
        if chunk.len() > 1 {
            out.push(B64_ALPHABET[((triple >> 6) & 63) as usize] as char);
        } else {
            out.push('=');
        }
        if chunk.len() > 2 {
            out.push(B64_ALPHABET[(triple & 63) as usize] as char);
        } else {
            out.push('=');
        }
    }
    out
}

/// Percent-decode bytes, like Go's `url.PathUnescape`: `+` stays literal,
/// malformed escapes fail, and non-UTF-8 results are kept as bytes.
fn percent_decode(input: &str) -> Result<Vec<u8>, Error> {
    let bytes = input.as_bytes();
    let mut out = Vec::with_capacity(bytes.len());
    let mut i = 0;
    while i < bytes.len() {
        if bytes[i] == b'%' {
            if i + 2 >= bytes.len() {
                return Err(Error::msg("invalid inline encoding"));
            }
            let hex = std::str::from_utf8(&bytes[i + 1..i + 3])
                .map_err(|_| Error::msg("invalid inline encoding"))?;
            let byte =
                u8::from_str_radix(hex, 16).map_err(|_| Error::msg("invalid inline encoding"))?;
            out.push(byte);
            i += 3;
        } else {
            out.push(bytes[i]);
            i += 1;
        }
    }
    Ok(out)
}

/// Decode an inline `data:` URI: unescape first, then base64-decode when
/// the mediatype ends with `;base64`. Mirrors `decodeDataURI`.
pub fn decode_data_uri(source: &str) -> Result<Vec<u8>, Error> {
    let (head, data) = source
        .split_once(',')
        .ok_or_else(|| Error::msg("inline data URI required"))?;
    if !head.starts_with("data:") {
        return Err(Error::msg("inline data URI required"));
    }
    let unescaped = percent_decode(data)?;
    if !head.ends_with(";base64") {
        return Ok(unescaped);
    }
    let text = String::from_utf8(unescaped).map_err(|_| Error::msg("invalid inline base64"))?;
    decode_base64(&text).map_err(|_| Error::msg("invalid inline base64"))
}

/// Bounded gzip decode with the Go owner's error taxonomy.
pub fn gunzip_bounded(data: &[u8]) -> Result<Vec<u8>, Error> {
    if data.len() < 2 || data[0] != 0x1f || data[1] != 0x8b {
        return Err(Error::msg("invalid compressed inline data"));
    }
    let mut decoder = flate2::read::GzDecoder::new(data);
    let mut decoded = Vec::new();
    let mut bounded = std::io::Read::take(&mut decoder, INLINE_GZIP_LIMIT + 1);
    match std::io::Read::read_to_end(&mut bounded, &mut decoded) {
        Ok(_) if decoded.len() as u64 <= INLINE_GZIP_LIMIT => Ok(decoded),
        _ => Err(Error::msg("invalid or oversized compressed inline data")),
    }
}

/// Decode one file's inline bytes: plain or bounded gzip. Mirrors
/// `inlineData`, including raw error passthrough.
pub fn inline_data(source: &str, compression: &str) -> Result<Vec<u8>, Error> {
    let data = decode_data_uri(source)?;
    if compression.is_empty() {
        return Ok(data);
    }
    if compression != "gzip" {
        return Err(Error::msg("unsupported inline compression"));
    }
    gunzip_bounded(&data)
}

/// Ignition file entry (lenient shape: unknown fields ignored, like Go's
/// plain `Unmarshal`).
pub struct IgnitionFile {
    /// Absolute target path.
    pub path: String,
    /// Inline source URI.
    pub source: String,
    /// Compression name, when present.
    pub compression: String,
}

/// Decode the storage files of an Ignition document, ignoring unknown
/// fields like the Go owner does. Present-but-mistyped shapes fail like
/// Go's `Unmarshal` type errors; callers map them to their own message.
pub fn decode_ignition_files(value: &JsonValue) -> Result<Vec<IgnitionFile>, Error> {
    let mut files = Vec::new();
    let storage = match value.get("storage") {
        None | Some(JsonValue::Null) => return Ok(files),
        Some(JsonValue::Object(_)) => value.get("storage"),
        Some(_) => return Err(Error::msg("invalid storage: object required")),
    };
    let items = match storage.and_then(|s| s.get("files")) {
        None | Some(JsonValue::Null) => return Ok(files),
        Some(JsonValue::Array(items)) => items,
        Some(_) => return Err(Error::msg("invalid files: array required")),
    };
    for item in items {
        let JsonValue::Object(_) = item else {
            return Err(Error::msg("invalid file: object required"));
        };
        let contents = match item.get("contents") {
            None | Some(JsonValue::Null) => None,
            Some(JsonValue::Object(_)) => item.get("contents"),
            Some(_) => return Err(Error::msg("invalid contents: object required")),
        };
        let field = |name: &str| -> Result<String, Error> {
            match contents.and_then(|c| c.get(name)) {
                None | Some(JsonValue::Null) => Ok(String::new()),
                Some(JsonValue::Str(s)) => Ok(s.clone()),
                Some(_) => Err(Error::msg(format!("invalid {name}: string required"))),
            }
        };
        files.push(IgnitionFile {
            path: crate::jsonio::opt_string(item, "path")?,
            source: field("source")?,
            compression: field("compression")?,
        });
    }
    Ok(files)
}

/// Lenient Ignition parse failure. Callers map it to their own input
/// message, like Go's per-caller `Unmarshal` checks.
#[derive(Debug)]
pub struct IgnitionError;

/// Parse a lenient Ignition document. Callers map failure to their own
/// input message, like Go's per-caller `Unmarshal` checks.
pub fn parse_ignition(data: &[u8]) -> Result<JsonValue, IgnitionError> {
    let text = std::str::from_utf8(data).map_err(|_| IgnitionError)?;
    JsonValue::parse(text).map_err(|_| IgnitionError)
}

fn is_fixture_trust_file(path: &str) -> bool {
    path == "/etc/hostname" || path == "/etc/ssh/ssh_host_ed25519_key"
}

/// Go's `strings.TrimSpace` is Unicode White Space, which is exactly
/// Rust's `str::trim`.
fn trim_space(text: &str) -> &str {
    text.trim()
}

fn run_ssh_keygen(args: &[&str]) -> Result<String, Error> {
    let output = std::process::Command::new("ssh-keygen")
        .args(args)
        .stdin(std::process::Stdio::null())
        .output()
        .map_err(Error::from)?;
    if !output.status.success() {
        return Err(Error::msg("ssh-keygen failed"));
    }
    String::from_utf8(output.stdout).map_err(|_| Error::msg("ssh-keygen failed"))
}

/// `known_hosts` lookup form for an explicit host and port.
fn known_hosts_query(host: &str, port: i64) -> String {
    if port == 22 {
        return host.to_string();
    }
    if host.starts_with('[') {
        return format!("{host}:{port}");
    }
    format!("[{host}]:{port}")
}

/// Verify the supplied trust root against the selected private bootstrap
/// before starting anything. Mirrors `VerifyFixtureTrust`: the fixture
/// hostname must match and the embedded host key must match the pinned
/// machine `known_hosts`. No keyscan, trust replacement, or product key
/// API is used.
pub fn verify_fixture_trust(path: &str, name: &str, remote: &Remote) -> Result<(), Error> {
    let data = files::private_file(path)?;
    let parsed = parse_ignition(&data).map_err(|_| Error::msg("invalid private Ignition"))?;
    let trust_parse = std::process::Command::new("ssh-keygen")
        .args(["-l", "-f", &remote.known_hosts])
        .stdin(std::process::Stdio::null())
        .stdout(std::process::Stdio::null())
        .stderr(std::process::Stdio::null())
        .status()
        .map_err(Error::from)?;
    if !trust_parse.success() {
        return Err(Error::msg("invalid pinned known_hosts"));
    }
    let files =
        decode_ignition_files(&parsed).map_err(|_| Error::msg("invalid private Ignition"))?;
    let mut matched = false;
    let mut hostname = false;
    for file in &files {
        if !is_fixture_trust_file(&file.path) {
            continue;
        }
        let body = inline_data(&file.source, &file.compression)?;
        if file.path == "/etc/hostname" {
            hostname = trim_space(&String::from_utf8_lossy(&body)) == name;
            continue;
        }
        verify_pinned_host_key(
            &body,
            &known_hosts_query(&remote.host, remote.port),
            &remote.known_hosts,
        )?;
        matched = true;
    }
    if !matched || !hostname {
        return Err(Error::msg(
            "matching fixture hostname and pinned Ed25519 host key required in Ignition",
        ));
    }
    Ok(())
}

fn verify_pinned_host_key(body: &[u8], query: &str, known_hosts: &str) -> Result<(), Error> {
    let scratch = files::TempDir::new("soda-trust")?;
    let priv_path = scratch.join("host_key");
    std::fs::write(&priv_path, body).map_err(Error::from)?;
    use std::os::unix::fs::PermissionsExt;
    std::fs::set_permissions(&priv_path, std::fs::Permissions::from_mode(0o600))
        .map_err(Error::from)?;
    let derived = run_ssh_keygen(&["-y", "-P", "", "-f", &priv_path.to_string_lossy()])
        .map_err(|_| Error::msg("invalid per-instance host key"))?;
    let mut parts = derived.split_whitespace();
    let (key_type, key_b64) = (parts.next().unwrap_or(""), parts.next().unwrap_or(""));
    let found = run_ssh_keygen(&["-F", query, "-f", known_hosts]).unwrap_or_default();
    for line in found.lines() {
        if line.starts_with('#') {
            continue;
        }
        let fields: Vec<&str> = line.split_whitespace().collect();
        if fields.len() >= 3 && fields[1] == key_type && fields[2] == key_b64 {
            return Ok(());
        }
    }
    Err(Error::msg(
        "ignition host key does not match pinned management trust",
    ))
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn base64_round_trip_and_rejections() {
        for raw in [
            b"f".as_slice(),
            b"fo",
            b"foo",
            b"foob",
            b"fooba",
            b"foobar",
            b"\x00\xff binary \x01",
        ] {
            assert_eq!(decode_base64(&encode_base64(raw)).unwrap(), raw);
        }
        assert_eq!(decode_base64("").unwrap(), b"");
        for bad in ["a", "abc", "ab=c", "a===", "ab!d", "abcd=", "===="] {
            assert!(decode_base64(bad).is_err(), "{bad}");
        }
        assert_eq!(decode_base64("Zm9v").unwrap(), b"foo");
    }

    #[test]
    fn data_uri_matrix() {
        assert_eq!(
            decode_data_uri("data:,soda-native-fixture%0A").unwrap(),
            b"soda-native-fixture\n"
        );
        assert_eq!(decode_data_uri("data:,a+b%41").unwrap(), b"a+bA");
        assert_eq!(decode_data_uri("data:;base64,Zm9v").unwrap(), b"foo");
        assert_eq!(
            decode_data_uri("data:text/plain;base64,Zm9v").unwrap(),
            b"foo"
        );
        assert_eq!(
            decode_data_uri("data:text/plain;charset=utf-8;base64,Zm9v").unwrap(),
            b"foo"
        );
        assert_eq!(decode_data_uri("data:,plain").unwrap(), b"plain");
        assert_eq!(
            decode_data_uri("plain").unwrap_err().to_string(),
            "inline data URI required"
        );
        assert_eq!(
            decode_data_uri("https://example.test/x")
                .unwrap_err()
                .to_string(),
            "inline data URI required"
        );
        assert_eq!(
            decode_data_uri("data:,a%zz").unwrap_err().to_string(),
            "invalid inline encoding"
        );
        assert_eq!(
            decode_data_uri("data:;base64,!!!").unwrap_err().to_string(),
            "invalid inline base64"
        );
        assert_eq!(decode_data_uri("data:,not-gzip").unwrap(), b"not-gzip");
    }

    #[test]
    fn inline_data_compression_gates() {
        assert!(inline_data("data:,not-gzip", "gzip").is_err());
        assert_eq!(
            inline_data("data:,plain", "unknown")
                .unwrap_err()
                .to_string(),
            "unsupported inline compression"
        );
        assert_eq!(inline_data("data:,plain", "").unwrap(), b"plain");
    }

    #[test]
    fn gzip_bounds_match_go() {
        let mut encoder = flate2::write::GzEncoder::new(Vec::new(), flate2::Compression::default());
        std::io::Write::write_all(&mut encoder, b"payload").unwrap();
        let compressed = encoder.finish().unwrap();
        assert_eq!(gunzip_bounded(&compressed).unwrap(), b"payload");
        assert_eq!(
            gunzip_bounded(b"not gzip").unwrap_err().to_string(),
            "invalid compressed inline data"
        );
        assert_eq!(
            gunzip_bounded(&[0x1f, 0x8b, 0x00]).unwrap_err().to_string(),
            "invalid or oversized compressed inline data"
        );
        let big = vec![b'x'; (INLINE_GZIP_LIMIT + 1) as usize];
        let mut encoder = flate2::write::GzEncoder::new(Vec::new(), flate2::Compression::default());
        std::io::Write::write_all(&mut encoder, &big).unwrap();
        let compressed = encoder.finish().unwrap();
        assert_eq!(
            gunzip_bounded(&compressed).unwrap_err().to_string(),
            "invalid or oversized compressed inline data"
        );
    }

    fn key_fixture(dir: &std::path::Path) -> (Vec<u8>, String) {
        let key = dir.join("id_ed25519");
        let status = std::process::Command::new("ssh-keygen")
            .args(["-t", "ed25519", "-f", key.to_str().unwrap(), "-N", "", "-q"])
            .stdin(std::process::Stdio::null())
            .status()
            .unwrap();
        assert!(status.success(), "ssh-keygen is required for trust tests");
        let private = std::fs::read(&key).unwrap();
        let public = std::fs::read_to_string(format!("{}.pub", key.display())).unwrap();
        (private, public.trim().to_string())
    }

    fn trust_fixture(dir: &std::path::Path) -> (String, Remote) {
        let (private, public) = key_fixture(dir);
        let hosts = dir.join("known_hosts");
        std::fs::write(&hosts, format!("[127.0.0.1]:22222 {public}\n")).unwrap();
        let input = dir.join("instance.ign");
        let body = format!(
            "{{\"ignition\":{{\"version\":\"3.5.0\"}},\"storage\":{{\"files\":[{{\"path\":\"/etc/hostname\",\"contents\":{{\"source\":\"data:,soda-native-fixture%0A\"}}}},{{\"path\":\"/etc/ssh/ssh_host_ed25519_key\",\"contents\":{{\"source\":\"data:;base64,{}\"}}}}]}}}}",
            encode_base64(&private)
        );
        std::fs::write(&input, body).unwrap();
        use std::os::unix::fs::PermissionsExt;
        std::fs::set_permissions(&input, std::fs::Permissions::from_mode(0o600)).unwrap();
        let remote = Remote {
            user: "root".to_string(),
            host: "127.0.0.1".to_string(),
            key: String::new(),
            known_hosts: hosts.to_string_lossy().into_owned(),
            port: 22222,
            timeout: std::time::Duration::ZERO,
        };
        (input.to_string_lossy().into_owned(), remote)
    }

    #[test]
    fn fixture_trust_happy_and_mismatch() {
        let dir = std::env::temp_dir().join(format!("soda-trust-test-{}", std::process::id()));
        let _ = std::fs::remove_dir_all(&dir);
        std::fs::create_dir_all(&dir).unwrap();
        let (input, remote) = trust_fixture(&dir);
        verify_fixture_trust(&input, "soda-native-fixture", &remote).unwrap();
        assert!(verify_fixture_trust(&input, "soda-native-other", &remote).is_err());
        let mut remote = remote;
        remote.port = 22223;
        assert!(verify_fixture_trust(&input, "soda-native-fixture", &remote).is_err());
        std::fs::remove_dir_all(&dir).unwrap();
    }

    #[test]
    fn fixture_trust_rejects_shape_violations() {
        let dir = std::env::temp_dir().join(format!("soda-trust-shape-{}", std::process::id()));
        let _ = std::fs::remove_dir_all(&dir);
        std::fs::create_dir_all(&dir).unwrap();
        let (input, remote) = trust_fixture(&dir);
        std::fs::write(&input, b"not json").unwrap();
        use std::os::unix::fs::PermissionsExt;
        std::fs::set_permissions(&input, std::fs::Permissions::from_mode(0o600)).unwrap();
        assert_eq!(
            verify_fixture_trust(&input, "soda-native-fixture", &remote)
                .unwrap_err()
                .to_string(),
            "invalid private Ignition"
        );
        std::fs::write(&input, b"{\"storage\":{\"files\":[]}}").unwrap();
        std::fs::set_permissions(&input, std::fs::Permissions::from_mode(0o600)).unwrap();
        assert_eq!(
            verify_fixture_trust(&input, "soda-native-fixture", &remote)
                .unwrap_err()
                .to_string(),
            "matching fixture hostname and pinned Ed25519 host key required in Ignition"
        );
        std::fs::remove_dir_all(&dir).unwrap();
    }
}

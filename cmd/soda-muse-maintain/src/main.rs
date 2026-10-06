// soda-muse-maintain installs public Muse tools and restores a live project interface.
use std::ffi::CString;
use std::fs;
use std::io::{self, Read};
use std::os::unix::fs::FileTypeExt;
use std::os::unix::fs::MetadataExt;
use std::os::unix::io::{FromRawFd, RawFd};
use std::process::{Command, Stdio};
use std::time::{Duration, Instant};

mod config;
mod options;

use config::{load_config, Config};
use options::{parse, Options};

const MUSE_VERSION: &str = "1.4.0-R4161.1";
const RELEASE_PATH: &str = "/usr/share/soda/release.json";

// --wait/--pipe needs the native system bus, not just systemd's private socket.
// Installing this demonstrated prerequisite belongs only to explicit maintenance.
const BUS_SCRIPT: &str = r#"set -eu
if ! rpm -q dbus-broker >/dev/null; then dnf -y install dbus-broker; fi
systemctl start dbus.socket
systemd-run --quiet --wait --pipe --collect /usr/bin/true
"#;

const INTERFACE_SCRIPT: &str = "set -eu; test ! -L /run; test -d /run; test ! -L /run/soda-muse-interface; if test -e /run/soda-muse-interface; then test -d /run/soda-muse-interface; fi; mkdir -p /run/soda-muse-interface";

// Destinations are fixed by the installed project interface, never repository input.
const DESTINATIONS: [&str; 3] = [
    "/usr/local/bin/muse",
    "/usr/local/bin/soda-identity-compose",
    "/usr/local/libexec/soda/muse",
];

const INSTALL_SCRIPT: &str = r#"
set -eu
safe_parent() {
 path=$(dirname "$1")
 while [ "$path" != / ]; do
  test ! -L "$path"
  if test -e "$path"; then test -d "$path"; fi
  path=$(dirname "$path")
 done
}
for target in "$@"; do
 safe_parent "$target"
 test ! -L "$target"
 if test -e "$target"; then test -f "$target"; fi
 done
for target in "$@"; do mkdir -p "$(dirname "$target")"; done
stage=$(mktemp -d "$(dirname "$3")/.soda-muse-maintain.XXXXXXXX")
trap 'rm -rf -- "$stage"' EXIT
tar --extract --file=- --directory="$stage" --no-same-owner
chmod 0755 "$stage/muse" "$stage/soda-identity-compose" "$stage/muse-native"
mv -T -- "$stage/muse" "$1"
mv -T -- "$stage/soda-identity-compose" "$2"
mv -T -- "$stage/muse-native" "$3"
"#;

const INSPECT_FORMAT: &str = r#"{"id":{{json .ID}},"project":{{json (index .Config.Labels "org.soda.project")}},"owner":{{json (index .Config.Labels "org.soda.owner")}},"pid":{{json .State.Pid}},"running":{{json .State.Running}}}"#;

fn main() {
    if let Err(e) = run() {
        eprintln!("soda-muse-maintain: {e}");
        std::process::exit(1);
    }
}

fn run() -> Result<(), String> {
    if unsafe { libc::geteuid() } != 0 {
        return Err(String::from("project maintenance requires root"));
    }
    let argv: Vec<String> = std::env::args_os()
        .skip(1)
        .map(|a| a.to_string_lossy().into_owned())
        .collect();
    let o = parse(&argv)?;
    let c = load_config(&o.config)?;
    if c.muse_sha256.is_empty() {
        if o.bind_only {
            return Ok(());
        }
        return Err(String::from("muse project runtime is disabled"));
    }
    maintain(&o, &c)
}

fn is_lower_hex(b: u8) -> bool {
    b.is_ascii_digit() || (b'a'..=b'f').contains(&b)
}

// go_quoted mirrors strconv.Quote: ASCII graphic bytes and U+0020 pass
// through, C0/DEL take short or \x escapes, and every other character
// Go's IsPrint rejects (controls, other separators, format,
// private-use) takes \u or \U escapes.
fn go_quoted(s: &str) -> String {
    let mut out = String::with_capacity(s.len() + 2);
    out.push('"');
    for c in s.chars() {
        match c {
            '\u{07}' => out.push_str("\\a"),
            '\u{08}' => out.push_str("\\b"),
            '\u{0c}' => out.push_str("\\f"),
            '\n' => out.push_str("\\n"),
            '\r' => out.push_str("\\r"),
            '\t' => out.push_str("\\t"),
            '\u{0b}' => out.push_str("\\v"),
            '"' => out.push_str("\\\""),
            '\\' => out.push_str("\\\\"),
            c if (c as u32) < 0x80 => {
                if c.is_ascii_graphic() || c == ' ' {
                    out.push(c);
                } else {
                    out.push_str(&format!("\\x{:02x}", c as u32));
                }
            }
            c if c.is_control() || (c.is_whitespace() && c != ' ') || is_go_nonprint(c) => {
                if (c as u32) <= 0xffff {
                    out.push_str(&format!("\\u{:04x}", c as u32));
                } else {
                    out.push_str(&format!("\\U{:08x}", c as u32));
                }
            }
            c => out.push(c),
        }
    }
    out.push('"');
    out
}

// is_go_nonprint covers the Unicode format (Cf) and private-use (Co)
// ranges, which strconv.Quote escapes but char::is_control misses.
fn is_go_nonprint(c: char) -> bool {
    matches!(c as u32,
        0x00AD | 0x061C | 0x06DD | 0x070F | 0x08E2 | 0x180E | 0xFEFF
        | 0x110BD | 0x110CD | 0xE0001
        | 0x0600..=0x0605 | 0x200B..=0x200F | 0x202A..=0x202E
        | 0x2060..=0x2064 | 0x2066..=0x206F | 0xFFF9..=0xFFFB
        | 0x13430..=0x13438 | 0x1BCA0..=0x1BCA3 | 0x1D173..=0x1D17A
        | 0xE0020..=0xE007F | 0xE0100..=0xE01EF
        | 0xE000..=0xF8FF | 0xF0000..=0xFFFFD | 0x100000..=0x10FFFD)
}

fn maintain(o: &Options, c: &Config) -> Result<(), String> {
    // Tool closes its own fd on drop, mirroring deferred closeTools.
    let sources = load_tools(&o.tools, &c.muse_sha256, &c.muse_version)?;
    let deadline = Instant::now() + Duration::from_secs(120);
    let target = wait_project(&o.project, deadline)?;
    if !o.bind_only {
        ensure_system_bus(&target, deadline)?;
        stage_tools(&target, &sources, deadline)?;
    }
    prepare_interface(&target, deadline)?;
    attach_interface(&target, &c.muse_socket, deadline)
}

// go_clean mirrors filepath.Clean lexical rules.
fn go_clean(path: &str) -> String {
    if path.is_empty() {
        return String::from(".");
    }
    let rooted = path.starts_with('/');
    let mut out: Vec<&str> = Vec::new();
    for part in path.split('/') {
        match part {
            "" | "." => {}
            ".." => {
                if let Some(last) = out.pop() {
                    if last == ".." {
                        out.push("..");
                        out.push("..");
                    }
                } else if !rooted {
                    out.push("..");
                }
            }
            _ => out.push(part),
        }
    }
    let mut clean = out.join("/");
    if rooted {
        clean.insert(0, '/');
    }
    if clean.is_empty() {
        clean.push('.');
    }
    clean
}

fn go_base(path: &str) -> &str {
    if path.is_empty() {
        return ".";
    }
    let stripped = path.trim_end_matches('/');
    if stripped.is_empty() {
        return "/";
    }
    match stripped.rfind('/') {
        Some(i) => &stripped[i + 1..],
        None => stripped,
    }
}

fn go_dir(path: &str) -> String {
    // filepath.Dir: Clean of everything through the last separator.
    let mut i = path.len();
    let b = path.as_bytes();
    while i > 0 && b[i - 1] != b'/' {
        i -= 1;
    }
    go_clean(&path[..i])
}

fn path_error(op: &str, path: &str, e: io::Error) -> String {
    format!("{op} {path}: {}", go_errno(e.raw_os_error().unwrap_or(0)))
}

// go_errno renders the stable Go syscall errno table, which never follows
// the process locale the way libc strerror does.
fn go_errno(no: i32) -> String {
    let text = match no {
        libc::EPERM => "operation not permitted",
        libc::ENOENT => "no such file or directory",
        libc::ESRCH => "no such process",
        libc::EINTR => "interrupted system call",
        libc::EIO => "input/output error",
        libc::ENXIO => "no such device or address",
        libc::E2BIG => "argument list too long",
        libc::ENOEXEC => "exec format error",
        libc::EBADF => "bad file descriptor",
        libc::ECHILD => "no child processes",
        libc::EAGAIN => "resource temporarily unavailable",
        libc::ENOMEM => "cannot allocate memory",
        libc::EACCES => "permission denied",
        libc::EFAULT => "bad address",
        libc::ENOTBLK => "block device required",
        libc::EBUSY => "device or resource busy",
        libc::EEXIST => "file exists",
        libc::EXDEV => "invalid cross-device link",
        libc::ENODEV => "no such device",
        libc::ENOTDIR => "not a directory",
        libc::EISDIR => "is a directory",
        libc::EINVAL => "invalid argument",
        libc::ENFILE => "too many open files in system",
        libc::EMFILE => "too many open files",
        libc::ENOTTY => "inappropriate ioctl for device",
        libc::ETXTBSY => "text file busy",
        libc::EFBIG => "file too large",
        libc::ENOSPC => "no space left on device",
        libc::ESPIPE => "illegal seek",
        libc::EROFS => "read-only file system",
        libc::EMLINK => "too many links",
        libc::EPIPE => "broken pipe",
        libc::EDOM => "numerical argument out of domain",
        libc::ERANGE => "numerical result out of range",
        libc::EDEADLK => "resource deadlock avoided",
        libc::ENAMETOOLONG => "file name too long",
        libc::ENOLCK => "no locks available",
        libc::ENOSYS => "function not implemented",
        libc::ENOTEMPTY => "directory not empty",
        libc::ELOOP => "too many levels of symbolic links",
        libc::ENOMSG => "no message of desired type",
        libc::EIDRM => "identifier removed",
        libc::ECHRNG => "channel number out of range",
        libc::EL2NSYNC => "level 2 not synchronized",
        libc::EL3HLT => "level 3 halted",
        libc::EL3RST => "level 3 reset",
        libc::ELNRNG => "link number out of range",
        libc::EUNATCH => "protocol driver not attached",
        libc::ENOCSI => "no CSI structure available",
        libc::EL2HLT => "level 2 halted",
        libc::EBADE => "invalid exchange",
        libc::EBADR => "invalid request descriptor",
        libc::EXFULL => "exchange full",
        libc::ENOANO => "no anode",
        libc::EBADRQC => "invalid request code",
        libc::EBADSLT => "invalid slot",
        libc::EBFONT => "bad font file format",
        libc::ENOSTR => "device not a stream",
        libc::ENODATA => "no data available",
        libc::ETIME => "timer expired",
        libc::ENOSR => "out of streams resources",
        libc::ENONET => "machine is not on the network",
        libc::ENOPKG => "package not installed",
        libc::EREMOTE => "object is remote",
        libc::ENOLINK => "link has been severed",
        libc::EADV => "advertise error",
        libc::ESRMNT => "srmount error",
        libc::ECOMM => "communication error on send",
        libc::EPROTO => "protocol error",
        libc::EMULTIHOP => "multihop attempted",
        libc::EDOTDOT => "RFS specific error",
        libc::EBADMSG => "bad message",
        libc::EOVERFLOW => "value too large for defined data type",
        libc::ENOTUNIQ => "name not unique on network",
        libc::EBADFD => "file descriptor in bad state",
        libc::EREMCHG => "remote address changed",
        libc::ELIBACC => "can not access a needed shared library",
        libc::ELIBBAD => "accessing a corrupted shared library",
        libc::ELIBSCN => ".lib section in a.out corrupted",
        libc::ELIBMAX => "attempting to link in too many shared libraries",
        libc::ELIBEXEC => "cannot exec a shared library directly",
        libc::EILSEQ => "invalid or incomplete multibyte or wide character",
        libc::ERESTART => "interrupted system call should be restarted",
        libc::ESTRPIPE => "streams pipe error",
        libc::EUSERS => "too many users",
        libc::ENOTSOCK => "socket operation on non-socket",
        libc::EDESTADDRREQ => "destination address required",
        libc::EMSGSIZE => "message too long",
        libc::EPROTOTYPE => "protocol wrong type for socket",
        libc::ENOPROTOOPT => "protocol not available",
        libc::EPROTONOSUPPORT => "protocol not supported",
        libc::ESOCKTNOSUPPORT => "socket type not supported",
        libc::EOPNOTSUPP => "operation not supported",
        libc::EPFNOSUPPORT => "protocol family not supported",
        libc::EAFNOSUPPORT => "address family not supported by protocol",
        libc::EADDRINUSE => "address already in use",
        libc::EADDRNOTAVAIL => "cannot assign requested address",
        libc::ENETDOWN => "network is down",
        libc::ENETUNREACH => "network is unreachable",
        libc::ENETRESET => "network dropped connection on reset",
        libc::ECONNABORTED => "software caused connection abort",
        libc::ECONNRESET => "connection reset by peer",
        libc::ENOBUFS => "no buffer space available",
        libc::EISCONN => "transport endpoint is already connected",
        libc::ENOTCONN => "transport endpoint is not connected",
        libc::ESHUTDOWN => "cannot send after transport endpoint shutdown",
        libc::ETOOMANYREFS => "too many references: cannot splice",
        libc::ETIMEDOUT => "connection timed out",
        libc::ECONNREFUSED => "connection refused",
        libc::EHOSTDOWN => "host is down",
        libc::EHOSTUNREACH => "no route to host",
        libc::EALREADY => "operation already in progress",
        libc::EINPROGRESS => "operation now in progress",
        libc::ESTALE => "stale file handle",
        libc::EUCLEAN => "structure needs cleaning",
        libc::ENOTNAM => "not a XENIX named type file",
        libc::ENAVAIL => "no XENIX semaphores available",
        libc::EISNAM => "is a named type file",
        libc::EREMOTEIO => "remote I/O error",
        libc::EDQUOT => "disk quota exceeded",
        libc::ENOMEDIUM => "no medium found",
        libc::EMEDIUMTYPE => "wrong medium type",
        libc::ECANCELED => "operation canceled",
        libc::ENOKEY => "required key not available",
        libc::EKEYEXPIRED => "key has expired",
        libc::EKEYREVOKED => "key has been revoked",
        libc::EKEYREJECTED => "key was rejected by service",
        libc::EOWNERDEAD => "owner died",
        libc::ENOTRECOVERABLE => "state not recoverable",
        libc::ERFKILL => "operation not possible due to RF-kill",
        libc::EHWPOISON => "memory page has hardware error",
        _ => return format!("errno {no}"),
    };
    String::from(text)
}

fn last_errno() -> i32 {
    io::Error::last_os_error().raw_os_error().unwrap_or(0)
}

// decode_host_config mirrors encoding/json Decode with DisallowUnknownFields
// into host.Config: exact-then-case-insensitive keys, last duplicate wins,
// null is a no-op, trailing data ignored, Go-compatible error strings.
fn decode_host_config(data: &[u8]) -> Result<Config, String> {
    let mut p = JsonParser::new(data);
    p.skip_ws();
    if p.eof() {
        return Err(String::from("EOF"));
    }
    match p.peek() {
        Some(b'{') => {
            p.bump();
        }
        Some(b'n') => {
            p.parse_literal()?;
            return Ok(Config::default());
        }
        Some(_) => {
            let kind = p.value_kind()?;
            return Err(format!(
                "json: cannot unmarshal {kind} into Go value of type host.Config"
            ));
        }
        None => return Err(String::from("EOF")),
    }
    let mut c = Config::default();
    // Go saves the first unknown-field/type error but keeps parsing: a
    // later syntax error overwrites it, later save-errors do not.
    let mut saved: Option<String> = None;
    let mut first = true;
    loop {
        p.skip_ws();
        if p.eof() {
            return Err(String::from("unexpected EOF"));
        }
        if first && p.peek() == Some(b'}') {
            p.bump();
            break;
        }
        if p.peek() != Some(b'"') {
            return Err(invalid_character(
                &p,
                "looking for beginning of object key string",
            ));
        }
        let key = p.parse_string()?;
        p.skip_ws();
        if p.eof() {
            return Err(String::from("unexpected EOF"));
        }
        if p.peek() != Some(b':') {
            return Err(invalid_character(&p, "after object key"));
        }
        p.bump();
        p.skip_ws();
        if p.eof() {
            return Err(String::from("unexpected EOF"));
        }
        if let Some(err) = decode_host_field(&mut p, &key, &mut c)? {
            if saved.is_none() {
                saved = Some(err);
            }
        }
        p.skip_ws();
        if p.eof() {
            return Err(String::from("unexpected EOF"));
        }
        match p.peek() {
            Some(b',') => {
                p.bump();
            }
            Some(b'}') => {
                p.bump();
                break;
            }
            _ => return Err(invalid_character(&p, "after object key:value pair")),
        }
        first = false;
    }
    if let Some(err) = saved {
        return Err(err);
    }
    Ok(c)
}

fn invalid_character(p: &JsonParser, context: &str) -> String {
    let c = p.peek_char();
    format!("invalid character '{c}' {context}")
}

fn decode_host_field(
    p: &mut JsonParser,
    key: &str,
    c: &mut Config,
) -> Result<Option<String>, String> {
    // Value syntax validates before field assignment: Go reports a broken
    // value even for unknown fields. The returned save-error (unknown field
    // or type mismatch) lets the caller keep parsing.
    enum Value {
        Str(String),
        Bool(bool),
        Null,
        Other(&'static str),
    }
    let value = match p.peek() {
        Some(b'"') => Value::Str(p.parse_string()?),
        Some(b't') | Some(b'f') | Some(b'n') => match p.parse_literal()? {
            "true" => Value::Bool(true),
            "false" => Value::Bool(false),
            _ => Value::Null,
        },
        Some(b'-') | Some(b'0'..=b'9') => {
            p.scan_number()?;
            Value::Other("number")
        }
        Some(b'[') => {
            p.skip_value()?;
            Value::Other("array")
        }
        Some(b'{') => {
            p.skip_value()?;
            Value::Other("object")
        }
        _ => return Err(invalid_character(p, "looking for beginning of value")),
    };
    let slot = match host_field_slot(key) {
        Some(s) => s,
        None => return Ok(Some(format!("json: unknown field {}", go_quoted(key)))),
    };
    if matches!(value, Value::Null) {
        return Ok(None);
    }
    let is_bool = slot == 7;
    match value {
        Value::Str(s) => {
            if is_bool {
                return Ok(Some(type_error("string", key, "bool")));
            }
            set_host_string(c, slot, s);
            Ok(None)
        }
        Value::Bool(b) => {
            if !is_bool {
                return Ok(Some(type_error("bool", key, "string")));
            }
            c.tailnet_management = b;
            Ok(None)
        }
        Value::Other(kind) => Ok(Some(type_error(
            kind,
            key,
            if is_bool { "bool" } else { "string" },
        ))),
        Value::Null => Ok(None),
    }
}

fn type_error(kind: &str, key: &str, ty: &str) -> String {
    format!("json: cannot unmarshal {kind} into Go struct field Config.{key} of type {ty}")
}

// host_field_slot resolves exact keys first, then one case-insensitive
// fallback, mirroring encoding/json field matching.
fn host_field_slot(key: &str) -> Option<usize> {
    const KEYS: [&str; 13] = [
        "muse_sha256",
        "muse_version",
        "muse_socket",
        "identity_socket",
        "codex_harness",
        "codex_harness_sha256",
        "codex_harness_version",
        "tailnet_management",
        "tailnet_image",
        "image",
        "network",
        "subnet",
        "bridge",
    ];
    if let Some(i) = KEYS.iter().position(|k| *k == key) {
        return Some(i);
    }
    let mut found = None;
    for (i, k) in KEYS.iter().enumerate() {
        if k.eq_ignore_ascii_case(key) {
            if found.is_some() {
                return None;
            }
            found = Some(i);
        }
    }
    found
}

fn set_host_string(c: &mut Config, slot: usize, value: String) {
    match slot {
        0 => c.muse_sha256 = value,
        1 => c.muse_version = value,
        2 => c.muse_socket = value,
        3 => c.identity_socket = value,
        4 => c.codex_harness = value,
        5 => c.codex_harness_sha256 = value,
        6 => c.codex_harness_version = value,
        8 => c.tailnet_image = value,
        9 => c.image = value,
        10 => c.network = value,
        11 => c.subnet = value,
        12 => c.bridge = value,
        _ => {}
    }
}

struct JsonParser<'a> {
    bytes: &'a [u8],
    pos: usize,
}

impl<'a> JsonParser<'a> {
    fn new(bytes: &'a [u8]) -> Self {
        JsonParser { bytes, pos: 0 }
    }

    fn eof(&self) -> bool {
        self.pos >= self.bytes.len()
    }

    fn peek(&self) -> Option<u8> {
        self.bytes.get(self.pos).copied()
    }

    fn peek_char(&self) -> String {
        match self.peek() {
            Some(b'\'') => String::from("\\'"),
            Some(b) if (b as char).is_ascii_graphic() || b == b' ' => format!("{}", b as char),
            Some(b'\n') => String::from("\\n"),
            Some(b'\r') => String::from("\\r"),
            Some(b'\t') => String::from("\\t"),
            Some(b) => format!("\\x{b:02x}"),
            None => String::from("EOF"),
        }
    }

    fn bump(&mut self) {
        self.pos += 1;
    }

    fn skip_ws(&mut self) {
        while matches!(self.peek(), Some(b' ' | b'\t' | b'\n' | b'\r')) {
            self.pos += 1;
        }
    }

    fn value_kind(&mut self) -> Result<&'static str, String> {
        match self.peek() {
            Some(b'"') => {
                self.parse_string()?;
                Ok("string")
            }
            Some(b't') | Some(b'f') | Some(b'n') => {
                let lit = self.parse_literal()?;
                Ok(if lit == "null" { "null" } else { "bool" })
            }
            Some(b'-') | Some(b'0'..=b'9') => {
                self.scan_number()?;
                Ok("number")
            }
            Some(b'[') => {
                self.skip_value()?;
                Ok("array")
            }
            Some(b'{') => {
                self.skip_value()?;
                Ok("object")
            }
            _ => Err(invalid_character(self, "looking for beginning of value")),
        }
    }

    // parse_string decodes with Go's leniency: lone surrogates become U+FFFD.
    fn parse_string(&mut self) -> Result<String, String> {
        if self.peek() != Some(b'"') {
            return Err(invalid_character(self, "looking for beginning of value"));
        }
        self.bump();
        let mut out = String::new();
        loop {
            if self.eof() {
                return Err(String::from("unexpected EOF"));
            }
            match self.bytes[self.pos] {
                b'"' => {
                    self.bump();
                    return Ok(out);
                }
                b'\\' => {
                    self.bump();
                    if self.eof() {
                        return Err(String::from("unexpected EOF"));
                    }
                    match self.bytes[self.pos] {
                        b'"' => out.push('"'),
                        b'\\' => out.push('\\'),
                        b'/' => out.push('/'),
                        b'b' => out.push('\u{0008}'),
                        b'f' => out.push('\u{000c}'),
                        b'n' => out.push('\n'),
                        b'r' => out.push('\r'),
                        b't' => out.push('\t'),
                        b'u' => {
                            // The escape consumes itself; continue below
                            // without the shared bump.
                            if self.pos + 4 >= self.bytes.len() {
                                return Err(String::from("unexpected EOF"));
                            }
                            let hex = &self.bytes[self.pos + 1..self.pos + 5];
                            if !hex.iter().all(|b| b.is_ascii_hexdigit()) {
                                let seq = String::from_utf8_lossy(
                                    &self.bytes[self.pos - 1..self.pos + 5],
                                )
                                .into_owned();
                                return Err(format!("invalid escape sequence `{seq}` in string"));
                            }
                            let cp =
                                u32::from_str_radix(std::str::from_utf8(hex).unwrap(), 16).unwrap();
                            self.pos += 5;
                            // Combine valid surrogate pairs; lone halves
                            // become U+FFFD exactly like encoding/json.
                            if (0xd800..0xdc00).contains(&cp) {
                                self.surrogate_tail(cp, &mut out)?;
                                continue;
                            } else if (0xd800..0xe000).contains(&cp) {
                                out.push('\u{FFFD}');
                            } else {
                                out.push(char::from_u32(cp).unwrap_or('\u{FFFD}'));
                            }
                            continue;
                        }
                        _ => {
                            let seq =
                                String::from_utf8_lossy(&self.bytes[self.pos - 1..self.pos + 1])
                                    .into_owned();
                            return Err(format!("invalid escape sequence `{seq}` in string"));
                        }
                    }
                    self.bump();
                }
                0x00..=0x1f => {
                    return Err(format!(
                        "invalid character '{}' in string",
                        self.peek_char()
                    ));
                }
                _ => {
                    // Invalid UTF-8 becomes U+FFFD like encoding/json.
                    let (c, n) = decode_first_char(&self.bytes[self.pos..]);
                    out.push(c);
                    self.pos += n;
                }
            }
        }
    }

    fn surrogate_tail(&mut self, high: u32, out: &mut String) -> Result<(), String> {
        if self.pos + 1 >= self.bytes.len()
            || self.bytes[self.pos] != b'\\'
            || self.bytes[self.pos + 1] != b'u'
        {
            out.push('\u{FFFD}');
            return Ok(());
        }
        if self.pos + 5 >= self.bytes.len() {
            return Err(String::from("unexpected EOF"));
        }
        let hex = &self.bytes[self.pos + 2..self.pos + 6];
        if !hex.iter().all(|b| b.is_ascii_hexdigit()) {
            let seq = String::from_utf8_lossy(&self.bytes[self.pos..self.pos + 6]).into_owned();
            return Err(format!("invalid escape sequence `{seq}` in string"));
        }
        let lo = u32::from_str_radix(std::str::from_utf8(hex).unwrap(), 16).unwrap();
        self.pos += 6;
        if (0xdc00..0xe000).contains(&lo) {
            let pair = 0x10000 + ((high - 0xd800) << 10) + (lo - 0xdc00);
            out.push(char::from_u32(pair).unwrap());
        } else {
            out.push('\u{FFFD}');
            out.push(char::from_u32(lo).unwrap_or('\u{FFFD}'));
        }
        Ok(())
    }

    fn parse_literal(&mut self) -> Result<&'static str, String> {
        for lit in ["true", "false", "null"] {
            if self.bytes[self.pos..].starts_with(lit.as_bytes()) {
                self.pos += lit.len();
                return Ok(lit);
            }
        }
        let full: &[u8] = match self.peek() {
            Some(b't') => b"true",
            Some(b'f') => b"false",
            Some(b'n') => b"null",
            _ => return Err(invalid_character(self, "looking for beginning of value")),
        };
        // Walk the literal to find the offending character.
        let mut i = 1;
        while i < full.len()
            && self.pos + i < self.bytes.len()
            && self.bytes[self.pos + i] == full[i]
        {
            i += 1;
        }
        self.pos += i;
        if self.eof() {
            return Err(String::from("unexpected EOF"));
        }
        let c = self.peek_char();
        let want = std::str::from_utf8(full).unwrap();
        let exp = full
            .get(i)
            .map(|b| format!("'{}'", *b as char))
            .unwrap_or_default();
        Err(format!(
            "invalid character '{c}' in literal {want} (expecting {exp})"
        ))
    }

    fn scan_number(&mut self) -> Result<(), String> {
        if self.peek() == Some(b'-') {
            self.bump();
            if self.eof() {
                return Err(String::from("unexpected EOF"));
            }
        }
        match self.peek() {
            Some(b'0') => {
                self.bump();
            }
            Some(b'1'..=b'9') => {
                while matches!(self.peek(), Some(b'0'..=b'9')) {
                    self.bump();
                }
            }
            _ => return Err(self.numeric_error()),
        }
        if self.peek() == Some(b'.') {
            self.bump();
            if !matches!(self.peek(), Some(b'0'..=b'9')) {
                return Err(self.numeric_error());
            }
            while matches!(self.peek(), Some(b'0'..=b'9')) {
                self.bump();
            }
        }
        if matches!(self.peek(), Some(b'e' | b'E')) {
            self.bump();
            if matches!(self.peek(), Some(b'+' | b'-')) {
                self.bump();
            }
            if !matches!(self.peek(), Some(b'0'..=b'9')) {
                return Err(self.numeric_error());
            }
            while matches!(self.peek(), Some(b'0'..=b'9')) {
                self.bump();
            }
        }
        Ok(())
    }

    fn numeric_error(&self) -> String {
        if self.eof() {
            return String::from("unexpected EOF");
        }
        format!(
            "invalid character '{}' in numeric literal",
            self.peek_char()
        )
    }

    fn skip_value(&mut self) -> Result<(), String> {
        match self.peek() {
            Some(b'"') => {
                self.parse_string()?;
                Ok(())
            }
            Some(b't') | Some(b'f') | Some(b'n') => {
                self.parse_literal()?;
                Ok(())
            }
            Some(b'-') | Some(b'0'..=b'9') => self.scan_number(),
            Some(b'{') | Some(b'[') => {
                let open = self.peek().unwrap();
                let close = if open == b'{' { b'}' } else { b']' };
                // Go rejects trailing commas and names the separator by
                // container: "after array element" or "after object
                // key:value pair".
                let after = if open == b'{' {
                    "after object key:value pair"
                } else {
                    "after array element"
                };
                self.bump();
                self.skip_ws();
                if self.eof() {
                    return Err(String::from("unexpected EOF"));
                }
                if self.peek() == Some(close) {
                    self.bump();
                    return Ok(());
                }
                loop {
                    if open == b'{' {
                        if self.peek() != Some(b'"') {
                            return Err(invalid_character(
                                self,
                                "looking for beginning of object key string",
                            ));
                        }
                        self.parse_string()?;
                        self.skip_ws();
                        if self.peek() != Some(b':') {
                            if self.eof() {
                                return Err(String::from("unexpected EOF"));
                            }
                            return Err(invalid_character(self, "after object key"));
                        }
                        self.bump();
                        self.skip_ws();
                    }
                    self.skip_value()?;
                    self.skip_ws();
                    if self.eof() {
                        return Err(String::from("unexpected EOF"));
                    }
                    if self.peek() == Some(b',') {
                        self.bump();
                        self.skip_ws();
                        if self.eof() {
                            return Err(String::from("unexpected EOF"));
                        }
                        // No close check here: a closer after a comma is a
                        // trailing comma, rejected by element parsing above.
                        continue;
                    }
                    if self.peek() == Some(close) {
                        self.bump();
                        return Ok(());
                    }
                    return Err(invalid_character(self, after));
                }
            }
            _ => {
                if self.eof() {
                    return Err(String::from("unexpected EOF"));
                }
                Err(invalid_character(self, "looking for beginning of value"))
            }
        }
    }
}

// decode_first_char decodes one UTF-8 unit, substituting U+FFFD for bytes
// encoding/json would also refuse to fail on.
fn decode_first_char(rest: &[u8]) -> (char, usize) {
    if rest.is_empty() {
        return ('\u{FFFD}', 0);
    }
    let b0 = rest[0];
    let width = if b0 < 0x80 {
        return (b0 as char, 1);
    } else if (0xc2..=0xdf).contains(&b0) {
        2
    } else if (0xe0..=0xef).contains(&b0) {
        3
    } else if (0xf0..=0xf4).contains(&b0) {
        4
    } else {
        return ('\u{FFFD}', 1);
    };
    if rest.len() < width || !rest[1..width].iter().all(|b| (0x80..=0xbf).contains(b)) {
        return ('\u{FFFD}', 1);
    }
    match std::str::from_utf8(&rest[..width]) {
        Ok(s) => (s.chars().next().unwrap(), width),
        Err(_) => ('\u{FFFD}', 1),
    }
}

fn apply_release_images(c: &mut Config, path: &str) -> Result<(), String> {
    if path.is_empty() {
        return Ok(());
    }
    let failed = String::from("immutable appliance image defaults unavailable");
    let payload = load_release_payload(path).map_err(|_| failed.clone())?;
    // RequireNative: x86_64 payload on a linux/amd64 binary only.
    if payload.architecture != "x86_64"
        || !cfg!(target_arch = "x86_64")
        || !cfg!(target_os = "linux")
    {
        return Err(failed);
    }
    let project = payload
        .image_config("project-os")
        .ok_or_else(|| failed.clone())?;
    let companion = payload
        .image_config("tailnet")
        .ok_or_else(|| failed.clone())?;
    if (!c.image.is_empty() && c.image != project)
        || (!c.tailnet_image.is_empty() && c.tailnet_image != companion)
    {
        return Err(String::from(
            "saved image selection conflicts with appliance release; explicit migration required",
        ));
    }
    c.image = project;
    if c.tailnet_management {
        c.tailnet_image = companion;
    }
    Ok(())
}

#[derive(Debug)]
struct ReleasePayload {
    architecture: String,
    images: Vec<(String, ReleaseImage)>,
}

#[derive(Debug)]
struct ReleaseImage {
    reference: String,
    config: String,
    manifest: String,
    archive_sha256: String,
}

impl ReleasePayload {
    fn image_config(&self, name: &str) -> Option<String> {
        self.images
            .iter()
            .find(|(n, _)| n == name)
            .map(|(_, image)| image.config.clone())
    }
}

// load_release_payload mirrors deliver.Load plus build.ReadJSON: confined
// regular file under 4 MiB, strict single object, full payload validation.
// Every failure collapses to the caller's unavailable message.
fn load_release_payload(path: &str) -> Result<ReleasePayload, ()> {
    let (dir, base) = match path.rfind('/') {
        Some(i) => (&path[..i], &path[i + 1..]),
        None => return Err(()),
    };
    let dir_path = if dir.is_empty() { "/" } else { dir };
    let lstat = fs::symlink_metadata(path).map_err(|_| ())?;
    if !lstat.is_file() || lstat.len() > 4 << 20 {
        return Err(());
    }
    let dir_fd = unsafe {
        let c = CString::new(dir_path).map_err(|_| ())?;
        libc::open(
            c.as_ptr(),
            libc::O_RDONLY | libc::O_CLOEXEC | libc::O_DIRECTORY,
        )
    };
    if dir_fd < 0 {
        return Err(());
    }
    struct Guard(RawFd);
    impl Drop for Guard {
        fn drop(&mut self) {
            unsafe { libc::close(self.0) };
        }
    }
    let _dir_guard = Guard(dir_fd);
    let base_c = CString::new(base).map_err(|_| ())?;
    let fd = unsafe {
        libc::openat(
            dir_fd,
            base_c.as_ptr(),
            libc::O_RDONLY | libc::O_NONBLOCK | libc::O_CLOEXEC,
            0,
        )
    };
    if fd < 0 {
        return Err(());
    }
    let _fd_guard = Guard(fd);
    let mut fst: libc::stat = unsafe { std::mem::zeroed() };
    if unsafe { libc::fstat(fd, &mut fst) } != 0 {
        return Err(());
    }
    if (fst.st_mode & libc::S_IFMT) != libc::S_IFREG
        || fst.st_dev as u64 != lstat.dev()
        || fst.st_ino as u64 != lstat.ino()
    {
        return Err(());
    }
    let mut body = Vec::new();
    let mut chunk = [0u8; 65536];
    loop {
        let n = unsafe { libc::read(fd, chunk.as_mut_ptr() as *mut libc::c_void, chunk.len()) };
        if n < 0 {
            return Err(());
        }
        if n == 0 {
            break;
        }
        body.extend_from_slice(&chunk[..n as usize]);
        if body.len() > (4 << 20) + 1 {
            return Err(());
        }
    }
    if body.len() > 4 << 20 {
        return Err(());
    }
    decode_release_payload(&body)
}

fn decode_release_payload(body: &[u8]) -> Result<ReleasePayload, ()> {
    let mut p = JsonParser::new(body);
    p.skip_ws();
    if p.peek() != Some(b'{') {
        return Err(());
    }
    p.bump();
    let mut format = -1i64;
    let mut id = String::new();
    let mut revision = String::new();
    let mut architecture = String::new();
    let mut coreos = String::new();
    let mut base = String::new();
    let mut repository_prefix = String::new();
    let mut schema = -1i64;
    let mut presentation = String::new();
    let mut host_packages = String::new();
    let mut images: Vec<(String, ReleaseImage)> = Vec::new();
    let mut upgrade_from: Vec<String> = Vec::new();
    let mut upgrade_seen = false;
    // encoding/json applies duplicates in order; the last value wins.
    // Trailing commas are rejected.
    let mut first = true;
    loop {
        p.skip_ws();
        if p.eof() {
            return Err(());
        }
        if first && p.peek() == Some(b'}') {
            p.bump();
            break;
        }
        if p.peek() != Some(b'"') {
            return Err(());
        }
        let key = p.parse_string().map_err(|_| ())?;
        p.skip_ws();
        if p.peek() != Some(b':') {
            return Err(());
        }
        p.bump();
        p.skip_ws();
        match payload_slot(&key) {
            Some(0) => format = p.parse_payload_int().map_err(|_| ())?,
            Some(1) => id = p.parse_payload_string().map_err(|_| ())?,
            Some(2) => revision = p.parse_payload_string().map_err(|_| ())?,
            Some(3) => architecture = p.parse_payload_string().map_err(|_| ())?,
            Some(4) => coreos = p.parse_payload_string().map_err(|_| ())?,
            Some(5) => base = p.parse_payload_string().map_err(|_| ())?,
            Some(6) => repository_prefix = p.parse_payload_string().map_err(|_| ())?,
            Some(7) => schema = p.parse_payload_int().map_err(|_| ())?,
            Some(8) => presentation = p.parse_payload_string().map_err(|_| ())?,
            Some(9) => host_packages = p.parse_payload_string().map_err(|_| ())?,
            Some(10) => images = p.parse_payload_images().map_err(|_| ())?,
            Some(11) => {
                upgrade_from = p.parse_payload_string_list().map_err(|_| ())?;
                upgrade_seen = true;
            }
            _ => return Err(()),
        }
        p.skip_ws();
        if p.eof() {
            return Err(());
        }
        match p.peek() {
            Some(b',') => {
                p.bump();
            }
            Some(b'}') => {
                p.bump();
                break;
            }
            _ => return Err(()),
        }
        first = false;
    }
    p.skip_ws();
    if !p.eof() {
        return Err(());
    }
    if !upgrade_seen {
        upgrade_from = Vec::new();
    }
    validate_release_payload(
        format,
        &id,
        &revision,
        &architecture,
        &coreos,
        &base,
        &repository_prefix,
        schema,
        &presentation,
        &host_packages,
        &images,
        &upgrade_from,
    )?;
    Ok(ReleasePayload {
        architecture,
        images,
    })
}

// payload_slot matches Payload's untagged Go field names, exact first with
// the same case-insensitive fallback encoding/json applies.
fn payload_slot(key: &str) -> Option<usize> {
    const KEYS: [&str; 12] = [
        "Format",
        "ID",
        "Revision",
        "Architecture",
        "CoreOS",
        "Base",
        "RepositoryPrefix",
        "Schema",
        "PresentationSHA256",
        "HostPackagesSHA256",
        "Images",
        "UpgradeFrom",
    ];
    if let Some(i) = KEYS.iter().position(|k| *k == key) {
        return Some(i);
    }
    let mut found = None;
    for (i, k) in KEYS.iter().enumerate() {
        if k.eq_ignore_ascii_case(key) {
            if found.is_some() {
                return None;
            }
            found = Some(i);
        }
    }
    found
}

#[allow(clippy::too_many_arguments)]
fn validate_release_payload(
    format: i64,
    id: &str,
    revision: &str,
    architecture: &str,
    coreos: &str,
    base: &str,
    repository_prefix: &str,
    schema: i64,
    presentation: &str,
    host_packages: &str,
    images: &[(String, ReleaseImage)],
    upgrade_from: &[String],
) -> Result<(), ()> {
    if format != 3
        || !is_hex_string(revision, 40)
        || id != format!("{coreos}.soda-{}", &revision[..12])
        || !valid_coreos_version(coreos)
        || architecture != "x86_64"
        || !valid_repository_prefix(repository_prefix)
        || schema < 1
        || !is_hex_string(presentation, 64)
        || !is_hex_string(host_packages, 64)
        || !upgrade_from.is_empty()
    {
        return Err(());
    }
    let (host, rest) = match base.split_once("/fedora/fedora-coreos@sha256:") {
        Some(v) => v,
        None => return Err(()),
    };
    if host.is_empty() || host.contains('/') || !is_hex_string(rest, 64) {
        return Err(());
    }
    const NAMES: [&str; 6] = [
        "dashboard",
        "forgejo",
        "extension",
        "proxy",
        "project-os",
        "tailnet",
    ];
    if images.len() != NAMES.len() {
        return Err(());
    }
    for name in NAMES {
        let image = &images.iter().find(|(n, _)| n == name).ok_or(())?.1;
        if !is_digest(&image.config)
            || !is_digest(&image.manifest)
            || !is_hex_string(&image.archive_sha256, 64)
            || image.reference != format!("{repository_prefix}-{name}@{}", image.manifest)
        {
            return Err(());
        }
    }
    let ext = &images.iter().find(|(n, _)| n == "extension").ok_or(())?.1;
    let forgejo = &images.iter().find(|(n, _)| n == "forgejo").ok_or(())?.1;
    if ext.config == forgejo.config {
        return Err(());
    }
    Ok(())
}

fn is_hex_string(s: &str, len: usize) -> bool {
    // Go's [0-9a-f] classes: lowercase hex only.
    s.len() == len && s.bytes().all(is_lower_hex)
}

fn is_digest(s: &str) -> bool {
    match s.strip_prefix("sha256:") {
        Some(hex) => is_hex_string(hex, 64),
        None => false,
    }
}

fn valid_coreos_version(s: &str) -> bool {
    let parts: Vec<&str> = s.split('.').collect();
    parts.len() == 4
        && parts
            .iter()
            .all(|p| !p.is_empty() && p.bytes().all(|b| b.is_ascii_digit()))
}

fn valid_repository_prefix(s: &str) -> bool {
    if s.len() >= 200 {
        return false;
    }
    let rest = match s.strip_prefix("ghcr.io/") {
        Some(r) => r,
        None => return false,
    };
    let (org, name) = match rest.split_once('/') {
        Some(v) => v,
        None => return false,
    };
    !org.is_empty()
        && !name.is_empty()
        && org
            .bytes()
            .next()
            .is_some_and(|b| b.is_ascii_lowercase() || b.is_ascii_digit())
        && org
            .bytes()
            .all(|b| b.is_ascii_lowercase() || b.is_ascii_digit() || b == b'-')
        && (name
            .bytes()
            .next()
            .is_some_and(|b| b.is_ascii_lowercase() || b.is_ascii_digit()))
        && name.bytes().all(|b| {
            b.is_ascii_lowercase() || b.is_ascii_digit() || b == b'.' || b == b'_' || b == b'-'
        })
        && !rest[org.len() + 1..].contains('/')
}

impl JsonParser<'_> {
    fn parse_payload_string(&mut self) -> Result<String, String> {
        if self.peek() == Some(b'n') && self.bytes[self.pos..].starts_with(b"null") {
            self.pos += 4;
            return Ok(String::new());
        }
        self.parse_string()
    }

    fn parse_payload_int(&mut self) -> Result<i64, String> {
        if self.peek() == Some(b'n') && self.bytes[self.pos..].starts_with(b"null") {
            self.pos += 4;
            return Ok(0);
        }
        let start = self.pos;
        if self.peek() == Some(b'-') {
            self.bump();
        }
        match self.peek() {
            Some(b'0') => {
                self.bump();
            }
            Some(b'1'..=b'9') => {
                while matches!(self.peek(), Some(b'0'..=b'9')) {
                    self.bump();
                }
            }
            _ => return Err(String::from("invalid number")),
        }
        if matches!(self.peek(), Some(b'.' | b'e' | b'E')) {
            return Err(String::from("invalid number"));
        }
        std::str::from_utf8(&self.bytes[start..self.pos])
            .map_err(|_| String::from("invalid number"))?
            .parse::<i64>()
            .map_err(|_| String::from("invalid number"))
    }

    fn parse_payload_images(&mut self) -> Result<Vec<(String, ReleaseImage)>, String> {
        if self.peek() != Some(b'{') {
            return Err(String::from("invalid images"));
        }
        self.bump();
        // Map entries apply in order; a repeated name replaces the
        // earlier binding exactly like encoding/json into a map.
        let mut images = Vec::new();
        let mut first = true;
        loop {
            self.skip_ws();
            if self.eof() {
                return Err(String::from("invalid images"));
            }
            if first && self.peek() == Some(b'}') {
                self.bump();
                return Ok(images);
            }
            if self.peek() != Some(b'"') {
                return Err(String::from("invalid images"));
            }
            let name = self.parse_string()?;
            self.skip_ws();
            if self.peek() != Some(b':') {
                return Err(String::from("invalid images"));
            }
            self.bump();
            self.skip_ws();
            let image = self.parse_payload_image()?;
            match images.iter_mut().find(|(n, _)| *n == name) {
                Some(slot) => slot.1 = image,
                None => images.push((name, image)),
            }
            self.skip_ws();
            if self.eof() {
                return Err(String::from("invalid images"));
            }
            match self.peek() {
                Some(b',') => {
                    self.bump();
                }
                Some(b'}') => {
                    self.bump();
                    return Ok(images);
                }
                _ => return Err(String::from("invalid images")),
            }
            first = false;
        }
    }

    fn parse_payload_image(&mut self) -> Result<ReleaseImage, String> {
        if self.peek() != Some(b'{') {
            return Err(String::from("invalid image"));
        }
        self.bump();
        let mut reference = String::new();
        let mut config = String::new();
        let mut manifest = String::new();
        let mut archive = String::new();
        // Repeated fields overwrite in order like encoding/json; trailing
        // commas are rejected.
        let mut first = true;
        loop {
            self.skip_ws();
            if self.eof() {
                return Err(String::from("invalid image"));
            }
            if first && self.peek() == Some(b'}') {
                self.bump();
                return Ok(ReleaseImage {
                    reference,
                    config,
                    manifest,
                    archive_sha256: archive,
                });
            }
            if self.peek() != Some(b'"') {
                return Err(String::from("invalid image"));
            }
            let key = self.parse_string()?;
            let slot = match image_slot(&key) {
                Some(s) => s,
                None => return Err(String::from("invalid image")),
            };
            self.skip_ws();
            if self.peek() != Some(b':') {
                return Err(String::from("invalid image"));
            }
            self.bump();
            self.skip_ws();
            let value = self.parse_payload_string()?;
            match slot {
                "Reference" => reference = value,
                "Config" => config = value,
                "Manifest" => manifest = value,
                _ => archive = value,
            }
            self.skip_ws();
            if self.eof() {
                return Err(String::from("invalid image"));
            }
            match self.peek() {
                Some(b',') => {
                    self.bump();
                }
                Some(b'}') => {
                    self.bump();
                    return Ok(ReleaseImage {
                        reference,
                        config,
                        manifest,
                        archive_sha256: archive,
                    });
                }
                _ => return Err(String::from("invalid image")),
            }
            first = false;
        }
    }

    fn parse_payload_string_list(&mut self) -> Result<Vec<String>, String> {
        if self.peek() == Some(b'n') && self.bytes[self.pos..].starts_with(b"null") {
            self.pos += 4;
            return Ok(Vec::new());
        }
        if self.peek() != Some(b'[') {
            return Err(String::from("invalid list"));
        }
        self.bump();
        let mut out = Vec::new();
        let mut first = true;
        loop {
            self.skip_ws();
            if self.eof() {
                return Err(String::from("invalid list"));
            }
            if first && self.peek() == Some(b']') {
                self.bump();
                return Ok(out);
            }
            if !first {
                if self.peek() != Some(b',') {
                    return Err(String::from("invalid list"));
                }
                self.bump();
                self.skip_ws();
            }
            out.push(self.parse_string()?);
            first = false;
        }
    }
}

fn image_slot(key: &str) -> Option<&'static str> {
    const KEYS: [&str; 4] = ["Reference", "Config", "Manifest", "ArchiveSHA256"];
    if let Some(k) = KEYS.iter().find(|k| **k == key).copied() {
        return Some(k);
    }
    let mut found = None;
    for k in KEYS {
        if k.eq_ignore_ascii_case(key) {
            if found.is_some() {
                return None;
            }
            found = Some(k);
        }
    }
    found
}

fn validate_runtime_config(c: &Config) -> Result<(), String> {
    // Order matches validateRuntimeConfig: muse, identity, subnet,
    // tailnet, then network names.
    validate_muse_runtime(c)?;
    validate_identity_runtime(c)?;
    parse_prefix(&c.subnet)?;
    if !valid_tailnet_config(c) {
        return Err(String::from(
            "invalid immutable Tailnet companion configuration",
        ));
    }
    if !valid_network_names(&c.image, &c.network, &c.bridge) {
        return Err(String::from("invalid native runtime configuration"));
    }
    Ok(())
}

fn valid_tailnet_config(c: &Config) -> bool {
    // An empty image is always fine; a set image needs management, a
    // sha256: prefix, and a valid digest reference.
    c.tailnet_image.is_empty() || (c.tailnet_management && valid_tailnet_image(&c.tailnet_image))
}

fn validate_muse_runtime(c: &Config) -> Result<(), String> {
    if c.muse_sha256.is_empty() {
        return Ok(());
    }
    if !c.muse_socket.starts_with('/')
        || !c.identity_socket.starts_with('/')
        || !is_hex_string(&c.muse_sha256, 64)
        || c.muse_version.is_empty()
        || go_base(&c.muse_socket) != "launch.sock"
    {
        return Err(String::from(
            "explicit muse socket, broker socket and release digest required",
        ));
    }
    Ok(())
}

fn validate_identity_runtime(c: &Config) -> Result<(), String> {
    if c.codex_harness.is_empty() {
        return Ok(());
    }
    if !c.identity_socket.starts_with('/')
        || !c.codex_harness.starts_with('/')
        || c.codex_harness_sha256.len() != 64
    {
        return Err(String::from(
            "explicit identity runtime socket and verified harness required",
        ));
    }
    if !c.codex_harness_version.is_empty() && !valid_harness_version(&c.codex_harness_version) {
        return Err(String::from("invalid staged harness version"));
    }
    Ok(())
}

fn valid_harness_version(s: &str) -> bool {
    // ^[A-Za-z0-9][A-Za-z0-9._-]{0,31}$
    let mut chars = s.bytes();
    match chars.next() {
        Some(b) if b.is_ascii_alphanumeric() => {}
        _ => return false,
    }
    let rest: Vec<u8> = chars.collect();
    rest.len() <= 31
        && rest
            .iter()
            .all(|b| b.is_ascii_alphanumeric() || matches!(b, b'.' | b'_' | b'-'))
}

fn valid_network_names(image: &str, network: &str, bridge: &str) -> bool {
    if image.is_empty() || image.starts_with('-') {
        return false;
    }
    valid_network_name(network) && valid_network_name(bridge)
}

fn valid_network_name(s: &str) -> bool {
    // ^[a-z][a-z0-9_-]{0,30}$
    let mut bytes = s.bytes();
    match bytes.next() {
        Some(b) if b.is_ascii_lowercase() => {}
        _ => return false,
    }
    let rest: Vec<u8> = bytes.collect();
    rest.len() <= 30
        && rest
            .iter()
            .all(|b| b.is_ascii_lowercase() || b.is_ascii_digit() || matches!(b, b'_' | b'-'))
}

fn valid_tailnet_image(s: &str) -> bool {
    // sha256: prefix plus ^[0-9a-f]{64}$ (ValidImageRef with prefix).
    match s.strip_prefix("sha256:") {
        Some(hex) => is_hex_string(hex, 64),
        None => false,
    }
}

// parse_prefix mirrors netip.ParsePrefix accept/reject behavior with
// Go-identical error strings for the observed input shapes.
fn parse_prefix(s: &str) -> Result<(), String> {
    let quoted = go_quoted(s);
    let fail = |detail: String| format!("netip.ParsePrefix({quoted}): {detail}");
    let slash = match s.rfind('/') {
        Some(i) => i,
        None => return Err(fail(String::from("no '/'"))),
    };
    let (ip, bits) = (&s[..slash], &s[slash + 1..]);
    let quoted_ip = go_quoted(ip);
    let (is_v6, zone) =
        parse_addr(ip).map_err(|detail| fail(format!("ParseAddr({quoted_ip}): {detail}")))?;
    if zone {
        return Err(fail(String::from(
            "IPv6 zones cannot be present in a prefix",
        )));
    }
    // strconv.Atoi accepts signs and leading zeroes, but prefixes do not:
    // multi-character bits must start with 1-9, then parse as digits.
    if bits.len() > 1 && (bits.as_bytes()[0] < b'1' || bits.as_bytes()[0] > b'9') {
        return Err(fail(format!("bad bits after slash: {}", go_quoted(bits))));
    }
    if bits.is_empty() || !bits.bytes().all(|b| b.is_ascii_digit()) {
        return Err(fail(format!("bad bits after slash: {}", go_quoted(bits))));
    }
    let max: i64 = if is_v6 { 128 } else { 32 };
    match bits.parse::<i64>() {
        // Unparseable magnitudes are bad bits; parsed excess is out of range.
        Err(_) => Err(fail(format!("bad bits after slash: {}", go_quoted(bits)))),
        Ok(n) if n <= max => Ok(()),
        Ok(_) => Err(fail(String::from("prefix length out of range"))),
    }
}

// parse_addr mirrors netip.ParseAddr dispatch on the first of '.', ':',
// '%', returning (is_v6, has_zone).
fn parse_addr(ip: &str) -> Result<(bool, bool), String> {
    for b in ip.bytes() {
        match b {
            b'.' => {
                parse_ipv4(ip)?;
                return Ok((false, false));
            }
            b':' => {
                let zone = parse_ipv6(ip)?;
                return Ok((true, zone));
            }
            b'%' => return Err(String::from("missing IPv6 address")),
            _ => {}
        }
    }
    Err(String::from("unable to parse IP"))
}

// parse_ipv4_fields is a line-for-line port of netip's scanner: errors
// fire left to right, and `at` is the unparsed remainder of the slice.
fn parse_ipv4_fields(s: &str) -> Result<(), String> {
    let b = s.as_bytes();
    let mut val: u32 = 0;
    let mut pos = 0;
    let mut dig_len = 0;
    for i in 0..b.len() {
        if b[i].is_ascii_digit() {
            if dig_len == 1 && val == 0 {
                return Err(String::from("IPv4 field has octet with leading zero"));
            }
            val = val * 10 + (b[i] - b'0') as u32;
            dig_len += 1;
            if val > 255 {
                return Err(String::from("IPv4 field has value >255"));
            }
        } else if b[i] == b'.' {
            if i == 0 || i == b.len() - 1 || b[i - 1] == b'.' {
                return Err(format!(
                    "IPv4 field must have at least one digit (at {})",
                    go_quoted(&s[i..])
                ));
            }
            if pos == 3 {
                return Err(String::from("IPv4 address too long"));
            }
            pos += 1;
            val = 0;
            dig_len = 0;
        } else {
            return Err(format!("unexpected character (at {})", go_quoted(&s[i..])));
        }
    }
    if pos < 3 {
        return Err(String::from("IPv4 address too short"));
    }
    Ok(())
}

fn parse_ipv4(ip: &str) -> Result<(), String> {
    parse_ipv4_fields(ip)
}

// parse_ipv6 ports netip's progressive loop, returning has_zone. `at`
// reports the unparsed remainder, never the whole input.
fn parse_ipv6(input: &str) -> Result<bool, String> {
    let mut s = input;
    let mut has_zone = false;
    if let Some(i) = s.find('%') {
        if s[i + 1..].is_empty() {
            return Err(String::from("zone must be a non-empty string"));
        }
        has_zone = true;
        s = &s[..i];
    }
    let mut ellipsis = false;
    if s.len() >= 2 && s.starts_with("::") {
        ellipsis = true;
        s = &s[2..];
        if s.is_empty() {
            return Ok(has_zone);
        }
    }
    let mut i = 0;
    while i < 16 {
        let sb = s.as_bytes();
        let mut off = 0;
        while off < sb.len() && sb[off].is_ascii_hexdigit() {
            off += 1;
        }
        if off > 4 {
            return Err(format!(
                "each group must have 4 or less digits (at {})",
                go_quoted(s)
            ));
        }
        if off == 0 {
            return Err(format!(
                "each colon-separated field must have at least one digit (at {})",
                go_quoted(s)
            ));
        }
        if off < sb.len() && sb[off] == b'.' {
            if !ellipsis && i != 12 {
                return Err(format!(
                    "embedded IPv4 address must replace the final 2 fields of the address (at {})",
                    go_quoted(s)
                ));
            }
            if i + 4 > 16 {
                return Err(format!(
                    "too many hex fields to fit an embedded IPv4 at the end of the address (at {})",
                    go_quoted(s)
                ));
            }
            parse_ipv4_fields(s)?;
            s = "";
            i += 4;
            break;
        }
        i += 2;
        s = &s[off..];
        if s.is_empty() {
            break;
        }
        if !s.starts_with(':') {
            return Err(format!(
                "unexpected character, want colon (at {})",
                go_quoted(s)
            ));
        } else if s.len() == 1 {
            return Err(format!(
                "colon must be followed by more characters (at {})",
                go_quoted(s)
            ));
        }
        s = &s[1..];
        if s.starts_with(':') {
            if ellipsis {
                return Err(format!("multiple :: in address (at {})", go_quoted(s)));
            }
            ellipsis = true;
            s = &s[1..];
            if s.is_empty() {
                break;
            }
        }
    }
    if !s.is_empty() {
        return Err(format!(
            "trailing garbage after address (at {})",
            go_quoted(s)
        ));
    }
    if i < 16 {
        if !ellipsis {
            return Err(String::from("address string too short"));
        }
    } else if ellipsis {
        return Err(String::from(
            "the :: must expand to at least one field of zeros",
        ));
    }
    Ok(has_zone)
}

#[derive(Debug)]
struct Tool {
    name: String,
    fd: RawFd,
    size: u64,
}

impl Drop for Tool {
    fn drop(&mut self) {
        if self.fd >= 0 {
            unsafe { libc::close(self.fd) };
        }
    }
}

fn load_tools(dir: &str, digest: &str, version: &str) -> Result<Vec<Tool>, String> {
    if version != MUSE_VERSION {
        return Err(String::from(
            "muse maintenance version differs from pinned release",
        ));
    }
    let mut tools = Vec::new();
    for name in ["muse", "soda-identity-compose", "muse-native"] {
        tools.push(open_tool(dir, name)?);
    }
    verify_native(&tools[2], digest)?;
    Ok(tools)
}

fn open_tool(dir: &str, name: &str) -> Result<Tool, String> {
    let path = format!("{dir}/{name}");
    let trusted = String::from("public tool source must be a regular root-owned executable");
    let lstat = fs::symlink_metadata(&path).map_err(|_| trusted.clone())?;
    if !trusted_tool(&lstat) {
        return Err(trusted);
    }
    // A NUL byte fails like Go's BytePtrFromString: raw EINVAL.
    let c = CString::new(path).map_err(|_| go_errno(libc::EINVAL))?;
    let fd = unsafe {
        libc::open(
            c.as_ptr(),
            libc::O_RDONLY | libc::O_NOFOLLOW | libc::O_CLOEXEC | libc::O_NONBLOCK,
        )
    };
    if fd < 0 {
        return Err(go_errno(last_errno()));
    }
    let mut fst: libc::stat = unsafe { std::mem::zeroed() };
    if unsafe { libc::fstat(fd, &mut fst) } != 0 {
        unsafe { libc::close(fd) };
        return Err(String::from("public tool unavailable"));
    }
    if (fst.st_mode & libc::S_IFMT) != libc::S_IFREG
        || (fst.st_mode & 0o7777) & 0o022 != 0
        || fst.st_uid != 0
        || (fst.st_mode & 0o7777) & 0o111 == 0
    {
        unsafe { libc::close(fd) };
        return Err(String::from("public tool source changed"));
    }
    Ok(Tool {
        name: name.to_string(),
        fd,
        size: fst.st_size as u64,
    })
}

fn trusted_tool(info: &fs::Metadata) -> bool {
    info.is_file() && info.uid() == 0 && info.mode() & 0o022 == 0 && info.mode() & 0o111 != 0
}

fn verify_native(tool: &Tool, digest: &str) -> Result<(), String> {
    // io.Copy hashes from the start to EOF; a short or grown file fails
    // the digest comparison, and read failures keep Go's PathError shape.
    // pread leaves the offset at zero like Go's trailing Seek.
    let mut hasher = Sha256::new();
    let mut offset: i64 = 0;
    let mut chunk = [0u8; 65536];
    loop {
        let n = unsafe {
            libc::pread(
                tool.fd,
                chunk.as_mut_ptr() as *mut libc::c_void,
                chunk.len(),
                offset,
            )
        };
        if n < 0 {
            return Err(format!("read {}: {}", tool.name, go_errno(last_errno())));
        }
        if n == 0 {
            break;
        }
        hasher.update(&chunk[..n as usize]);
        offset += n as i64;
    }
    let sum = hasher.finish();
    if hex_encode(&sum) != digest {
        return Err(String::from("muse native digest mismatch"));
    }
    Ok(())
}

fn hex_encode(bytes: &[u8]) -> String {
    const HEX: &[u8; 16] = b"0123456789abcdef";
    let mut out = String::with_capacity(bytes.len() * 2);
    for b in bytes {
        out.push(HEX[(b >> 4) as usize] as char);
        out.push(HEX[(b & 0xf) as usize] as char);
    }
    out
}

// Sha256 is a minimal streaming implementation so multi-hundred-megabyte
// native binaries verify without buffering the whole file.
struct Sha256 {
    h: [u32; 8],
    buf: [u8; 64],
    used: usize,
    len: u64,
}

impl Sha256 {
    fn new() -> Self {
        Sha256 {
            h: [
                0x6a09e667, 0xbb67ae85, 0x3c6ef372, 0xa54ff53a, 0x510e527f, 0x9b05688c, 0x1f83d9ab,
                0x5be0cd19,
            ],
            buf: [0; 64],
            used: 0,
            len: 0,
        }
    }

    fn update(&mut self, mut data: &[u8]) {
        self.len += data.len() as u64;
        if self.used > 0 {
            let take = (64 - self.used).min(data.len());
            self.buf[self.used..self.used + take].copy_from_slice(&data[..take]);
            self.used += take;
            data = &data[take..];
            if self.used == 64 {
                let block = self.buf;
                Self::block(&mut self.h, &block);
                self.used = 0;
            }
        }
        while data.len() >= 64 {
            let mut block = [0u8; 64];
            block.copy_from_slice(&data[..64]);
            Self::block(&mut self.h, &block);
            data = &data[64..];
        }
        if !data.is_empty() {
            self.buf[..data.len()].copy_from_slice(data);
            self.used = data.len();
        }
    }

    fn finish(mut self) -> [u8; 32] {
        let bit_len = self.len * 8;
        self.buf[self.used] = 0x80;
        self.used += 1;
        if self.used > 56 {
            self.buf[self.used..].fill(0);
            let block = self.buf;
            Self::block(&mut self.h, &block);
            self.buf = [0; 64];
            self.used = 0;
        }
        self.buf[self.used..56].fill(0);
        self.buf[56..64].copy_from_slice(&bit_len.to_be_bytes());
        let block = self.buf;
        Self::block(&mut self.h, &block);
        let mut out = [0u8; 32];
        for (i, word) in self.h.iter().enumerate() {
            out[i * 4..i * 4 + 4].copy_from_slice(&word.to_be_bytes());
        }
        out
    }

    fn block(h: &mut [u32; 8], block: &[u8; 64]) {
        const K: [u32; 64] = [
            0x428a2f98, 0x71374491, 0xb5c0fbcf, 0xe9b5dba5, 0x3956c25b, 0x59f111f1, 0x923f82a4,
            0xab1c5ed5, 0xd807aa98, 0x12835b01, 0x243185be, 0x550c7dc3, 0x72be5d74, 0x80deb1fe,
            0x9bdc06a7, 0xc19bf174, 0xe49b69c1, 0xefbe4786, 0x0fc19dc6, 0x240ca1cc, 0x2de92c6f,
            0x4a7484aa, 0x5cb0a9dc, 0x76f988da, 0x983e5152, 0xa831c66d, 0xb00327c8, 0xbf597fc7,
            0xc6e00bf3, 0xd5a79147, 0x06ca6351, 0x14292967, 0x27b70a85, 0x2e1b2138, 0x4d2c6dfc,
            0x53380d13, 0x650a7354, 0x766a0abb, 0x81c2c92e, 0x92722c85, 0xa2bfe8a1, 0xa81a664b,
            0xc24b8b70, 0xc76c51a3, 0xd192e819, 0xd6990624, 0xf40e3585, 0x106aa070, 0x19a4c116,
            0x1e376c08, 0x2748774c, 0x34b0bcb5, 0x391c0cb3, 0x4ed8aa4a, 0x5b9cca4f, 0x682e6ff3,
            0x748f82ee, 0x78a5636f, 0x84c87814, 0x8cc70208, 0x90befffa, 0xa4506ceb, 0xbef9a3f7,
            0xc67178f2,
        ];
        let mut w = [0u32; 64];
        for i in 0..16 {
            w[i] = u32::from_be_bytes([
                block[i * 4],
                block[i * 4 + 1],
                block[i * 4 + 2],
                block[i * 4 + 3],
            ]);
        }
        for i in 16..64 {
            let s0 = w[i - 15].rotate_right(7) ^ w[i - 15].rotate_right(18) ^ (w[i - 15] >> 3);
            let s1 = w[i - 2].rotate_right(17) ^ w[i - 2].rotate_right(19) ^ (w[i - 2] >> 10);
            w[i] = w[i - 16]
                .wrapping_add(s0)
                .wrapping_add(w[i - 7])
                .wrapping_add(s1);
        }
        let (mut a, mut b, mut c, mut d, mut e, mut f, mut g, mut hh) =
            (h[0], h[1], h[2], h[3], h[4], h[5], h[6], h[7]);
        for i in 0..64 {
            let s1 = e.rotate_right(6) ^ e.rotate_right(11) ^ e.rotate_right(25);
            let ch = (e & f) ^ ((!e) & g);
            let t1 = hh
                .wrapping_add(s1)
                .wrapping_add(ch)
                .wrapping_add(K[i])
                .wrapping_add(w[i]);
            let s0 = a.rotate_right(2) ^ a.rotate_right(13) ^ a.rotate_right(22);
            let maj = (a & b) ^ (a & c) ^ (b & c);
            let t2 = s0.wrapping_add(maj);
            hh = g;
            g = f;
            f = e;
            e = d.wrapping_add(t1);
            d = c;
            c = b;
            b = a;
            a = t1.wrapping_add(t2);
        }
        h[0] = h[0].wrapping_add(a);
        h[1] = h[1].wrapping_add(b);
        h[2] = h[2].wrapping_add(c);
        h[3] = h[3].wrapping_add(d);
        h[4] = h[4].wrapping_add(e);
        h[5] = h[5].wrapping_add(f);
        h[6] = h[6].wrapping_add(g);
        h[7] = h[7].wrapping_add(hh);
    }
}

#[derive(PartialEq, Debug)]
struct Observation {
    id: String,
    project: String,
    owner: String,
    pid: i64,
    running: bool,
}

fn podman(args: &[&str], deadline: Instant) -> Result<Vec<u8>, String> {
    let mut cmd = Command::new("podman");
    cmd.arg("--remote=false");
    for a in args {
        cmd.arg(a);
    }
    cmd.stdin(Stdio::null());
    cmd.stdout(Stdio::piped());
    cmd.stderr(Stdio::null());
    let child = cmd
        .spawn()
        .map_err(|_| String::from("project maintenance command failed"))?;
    wait_output(child, deadline)
}

// podman_streamed feeds stdin from a writer thread through a pipe while
// podman consumes it, mirroring Go's io.Pipe archive delivery. A podman
// failure wins over the archive result exactly like stagePublicTools.
fn podman_streamed(
    feed: impl FnOnce(fs::File) -> Result<(), String> + Send + 'static,
    args: &[&str],
    deadline: Instant,
) -> Result<(), String> {
    let mut fds = [0; 2];
    // O_CLOEXEC is load-bearing: without it the spawned child inherits
    // the stdin write end and its stdin reader never sees EOF.
    if unsafe { libc::pipe2(fds.as_mut_ptr(), libc::O_CLOEXEC) } != 0 {
        return Err(String::from("project maintenance command failed"));
    }
    let writer = unsafe { fs::File::from_raw_fd(fds[1]) };
    let reader = unsafe { fs::File::from_raw_fd(fds[0]) };
    let feeder = std::thread::spawn(move || feed(writer));
    let mut cmd = Command::new("podman");
    cmd.arg("--remote=false");
    for a in args {
        cmd.arg(a);
    }
    cmd.stdin(Stdio::from(reader));
    cmd.stdout(Stdio::piped());
    cmd.stderr(Stdio::null());
    let child = match cmd.spawn() {
        Ok(child) => child,
        Err(_) => {
            let _ = feeder.join();
            return Err(String::from("project maintenance command failed"));
        }
    };
    let output = wait_output(child, deadline);
    let fed = feeder
        .join()
        .map_err(|_| String::from("project maintenance command failed"))?;
    output?;
    fed
}

fn wait_output(mut child: std::process::Child, deadline: Instant) -> Result<Vec<u8>, String> {
    // Stdout drains on its own thread like exec.Output, so large output
    // never deadlocks against the exit wait. The child handle stays here
    // for the deadline kill.
    let failed = String::from("project maintenance command failed");
    let stdout = child.stdout.take();
    let reader = std::thread::spawn(move || {
        let mut out = Vec::new();
        match stdout {
            Some(mut pipe) => match pipe.read_to_end(&mut out) {
                Ok(_) => Some(out),
                Err(_) => None,
            },
            None => Some(out),
        }
    });
    loop {
        match child.try_wait() {
            Ok(Some(status)) => {
                let out = reader
                    .join()
                    .map_err(|_| failed.clone())?
                    .ok_or(failed.clone())?;
                if !status.success() {
                    return Err(failed);
                }
                return Ok(out);
            }
            Ok(None) => {
                if Instant::now() >= deadline {
                    let _ = child.kill();
                    let _ = child.wait();
                    let _ = reader.join();
                    return Err(failed);
                }
                std::thread::sleep(Duration::from_millis(10));
            }
            Err(_) => {
                let _ = reader.join();
                return Err(failed);
            }
        }
    }
}

fn inspect_project(key: &str, project: &str, deadline: Instant) -> Result<Observation, String> {
    let body = podman(&["inspect", "--format", INSPECT_FORMAT, key], deadline)?;
    if body.len() > 8192 {
        return Err(String::from("invalid project observation"));
    }
    let o = decode_observation(&body)?;
    validate_observation(&o, project)?;
    Ok(o)
}

fn validate_observation(o: &Observation, project: &str) -> Result<(), String> {
    if !is_hex_string(&o.id, 64) || o.project != project {
        return Err(String::from("project container identity differs"));
    }
    match o.owner.parse::<i64>() {
        Ok(n) if n > 0 => {}
        _ => return Err(String::from("project owner label is invalid")),
    }
    Ok(())
}

// decode_observation mirrors strictjson.Decode into project inspection:
// one object, no duplicate or unknown fields, Go value semantics.
// Pairs apply in sorted-key order like the normalized re-decode, so exact
// lowercase keys win over case-variant duplicates exactly as in Go.
fn decode_observation(body: &[u8]) -> Result<Observation, String> {
    let invalid = String::from("invalid project observation");
    if body.len() > 1 << 20 {
        return Err(invalid.clone());
    }
    if std::str::from_utf8(body).is_err() {
        return Err(invalid.clone());
    }
    enum Field {
        Str(String),
        Int(i64),
        Bool(bool),
        Null,
    }
    let mut p = JsonParser::new(body);
    p.skip_ws();
    if p.peek() != Some(b'{') {
        return Err(invalid.clone());
    }
    p.bump();
    let mut pairs: Vec<(String, Field)> = Vec::new();
    let mut first = true;
    loop {
        p.skip_ws();
        if p.eof() {
            return Err(invalid.clone());
        }
        if first && p.peek() == Some(b'}') {
            p.bump();
            break;
        }
        if p.peek() != Some(b'"') {
            return Err(invalid.clone());
        }
        let key = p.parse_string().map_err(|_| invalid.clone())?;
        if pairs.iter().any(|(k, _)| k == &key) {
            return Err(invalid.clone());
        }
        p.skip_ws();
        if p.peek() != Some(b':') {
            return Err(invalid.clone());
        }
        p.bump();
        p.skip_ws();
        if p.eof() {
            return Err(invalid.clone());
        }
        let field = match p.peek() {
            Some(b'"') => Field::Str(p.parse_string().map_err(|_| invalid.clone())?),
            Some(b't') | Some(b'f') | Some(b'n') => {
                // Literals must be exact; prefixes fail like encoding/json.
                if p.bytes[p.pos..].starts_with(b"true") {
                    p.pos += 4;
                    Field::Bool(true)
                } else if p.bytes[p.pos..].starts_with(b"false") {
                    p.pos += 5;
                    Field::Bool(false)
                } else if p.bytes[p.pos..].starts_with(b"null") {
                    p.pos += 4;
                    Field::Null
                } else {
                    return Err(invalid.clone());
                }
            }
            Some(b'-') | Some(b'0'..=b'9') => {
                Field::Int(p.parse_payload_int().map_err(|_| invalid.clone())?)
            }
            _ => return Err(invalid.clone()),
        };
        pairs.push((key, field));
        p.skip_ws();
        if p.eof() {
            return Err(invalid.clone());
        }
        match p.peek() {
            Some(b',') => {
                p.bump();
            }
            Some(b'}') => {
                p.bump();
                break;
            }
            _ => return Err(invalid.clone()),
        }
        first = false;
    }
    p.skip_ws();
    if !p.eof() {
        return Err(invalid.clone());
    }
    pairs.sort_by(|a, b| a.0.as_bytes().cmp(b.0.as_bytes()));
    let mut o = Observation {
        id: String::new(),
        project: String::new(),
        owner: String::new(),
        pid: 0,
        running: false,
    };
    for (key, field) in &pairs {
        let slot = match observation_slot(key) {
            Some(s) => s,
            None => return Err(invalid.clone()),
        };
        match (slot, field) {
            (0, Field::Str(v)) => o.id = v.clone(),
            (1, Field::Str(v)) => o.project = v.clone(),
            (2, Field::Str(v)) => o.owner = v.clone(),
            (3, Field::Int(v)) => o.pid = *v,
            (4, Field::Bool(v)) => o.running = *v,
            (_, Field::Null) => {}
            _ => return Err(invalid.clone()),
        }
    }
    Ok(o)
}

fn observation_slot(key: &str) -> Option<usize> {
    const KEYS: [&str; 5] = ["id", "project", "owner", "pid", "running"];
    if let Some(i) = KEYS.iter().position(|k| *k == key) {
        return Some(i);
    }
    let mut found = None;
    for (i, k) in KEYS.iter().enumerate() {
        if k.eq_ignore_ascii_case(key) {
            if found.is_some() {
                return None;
            }
            found = Some(i);
        }
    }
    found
}

fn wait_project(project: &str, deadline: Instant) -> Result<Observation, String> {
    let end = Instant::now() + Duration::from_secs(10);
    let end = end.min(deadline);
    let name = format!("soda-{project}");
    loop {
        let target = inspect_project(&name, project, end)?;
        if target.running && target.pid > 0 {
            return Ok(target);
        }
        if Instant::now() >= end {
            return Err(String::from(
                "project did not become running within maintenance deadline",
            ));
        }
        std::thread::sleep(Duration::from_millis(100));
    }
}

fn confirm_project(target: &Observation, deadline: Instant) -> Result<(), String> {
    let live = inspect_project(&target.id, &target.project, deadline)?;
    if live != *target || !live.running {
        return Err(String::from(
            "project incarnation changed during maintenance",
        ));
    }
    Ok(())
}

fn ensure_system_bus(target: &Observation, deadline: Instant) -> Result<(), String> {
    confirm_project(target, deadline)?;
    podman(
        &[
            "exec", "--user", "0:0", &target.id, "/bin/sh", "-ceu", BUS_SCRIPT,
        ],
        deadline,
    )?;
    Ok(())
}

fn stage_tools(target: &Observation, sources: &[Tool], deadline: Instant) -> Result<(), String> {
    confirm_project(target, deadline)?;
    // The feeder thread borrows no Tool state: the fds stay open in the
    // caller while only plain ints cross the thread boundary.
    let feeds: Vec<(String, RawFd, u64)> = sources
        .iter()
        .map(|t| (t.name.clone(), t.fd, t.size))
        .collect();
    let args: Vec<&str> = vec![
        "exec",
        "--user",
        "0:0",
        "-i",
        &target.id,
        "/bin/sh",
        "-ceu",
        INSTALL_SCRIPT,
        "soda-muse-maintain",
        DESTINATIONS[0],
        DESTINATIONS[1],
        DESTINATIONS[2],
    ];
    podman_streamed(
        move |writer| feed_archive(writer, &feeds, deadline),
        &args,
        deadline,
    )?;
    Ok(())
}

// feed_archive streams the tar byte sequence to podman's stdin. Writes
// poll non-blocking against the deadline so an abandoned pipe ends with
// the maintenance failure instead of hanging past it; a dead reader
// surfaces Go's closed-pipe error.
fn feed_archive(
    writer: fs::File,
    feeds: &[(String, RawFd, u64)],
    deadline: Instant,
) -> Result<(), String> {
    use std::os::unix::io::AsRawFd;
    let wfd = writer.as_raw_fd();
    let flags = unsafe { libc::fcntl(wfd, libc::F_GETFL) };
    if flags >= 0 {
        unsafe {
            libc::fcntl(wfd, libc::F_SETFL, flags | libc::O_NONBLOCK);
        }
    }
    let mut emit = |buf: &[u8]| -> Result<(), String> {
        let mut rest = buf;
        while !rest.is_empty() {
            let n = unsafe { libc::write(wfd, rest.as_ptr() as *const libc::c_void, rest.len()) };
            if n > 0 {
                rest = &rest[n as usize..];
                continue;
            }
            if n == 0 {
                continue;
            }
            let no = last_errno();
            if no == libc::EINTR {
                continue;
            }
            if no == libc::EAGAIN {
                if Instant::now() >= deadline {
                    return Err(String::from("project maintenance command failed"));
                }
                std::thread::sleep(Duration::from_millis(10));
                continue;
            }
            return Err(String::from("io: read/write on closed pipe"));
        }
        Ok(())
    };
    emit_archive(&mut emit, feeds)
}

// emit_archive ports archiveTools: one USTAR header per tool, CopyN of
// exactly size bytes, 512 padding, and the two zero trailer blocks.
// A short file ends CopyN with raw io.EOF; read failures keep the
// PathError shape with the tool's archive name.
fn emit_archive(
    emit: &mut dyn FnMut(&[u8]) -> Result<(), String>,
    feeds: &[(String, RawFd, u64)],
) -> Result<(), String> {
    for (name, fd, size) in feeds {
        emit(&tar_header(name, *size)?)?;
        let mut remaining = *size;
        let mut offset: i64 = 0;
        let mut chunk = [0u8; 65536];
        while remaining > 0 {
            let want = remaining.min(chunk.len() as u64) as usize;
            let n =
                unsafe { libc::pread(*fd, chunk.as_mut_ptr() as *mut libc::c_void, want, offset) };
            if n < 0 {
                return Err(format!("read {name}: {}", go_errno(last_errno())));
            }
            if n == 0 {
                return Err(String::from("EOF"));
            }
            emit(&chunk[..n as usize])?;
            offset += n as i64;
            remaining -= n as u64;
        }
        let pad = (512 - (size % 512)) % 512;
        if pad > 0 {
            let zeros = vec![0u8; pad as usize];
            emit(&zeros)?;
        }
    }
    emit(&[0u8; 1024])?;
    Ok(())
}

// tar_header writes the USTAR byte stream Go's archive/tar emits for
// these short regular names: fixed headers, 512-block data, two zero
// blocks. Names are fixed constants, so the length guard never fires.
fn tar_header(name: &str, size: u64) -> Result<[u8; 512], String> {
    let mut header = [0u8; 512];
    if name.len() > 100 || name.contains('\0') {
        return Err(String::from("public tool name exceeds archive limit"));
    }
    header[..name.len()].copy_from_slice(name.as_bytes());
    // Mode 0755, uid/gid 0, size octal, mtime 0, regular file, USTAR.
    header[100..108].copy_from_slice(format!("{:07o}\0", 0o755).as_bytes());
    header[108..116].copy_from_slice(b"0000000\0");
    header[116..124].copy_from_slice(b"0000000\0");
    header[124..136].copy_from_slice(format!("{:011o}\0", size).as_bytes());
    header[136..148].copy_from_slice(b"00000000000\0");
    header[156] = b'0';
    header[329..337].copy_from_slice(b"0000000\0");
    header[337..345].copy_from_slice(b"0000000\0");
    header[257..263].copy_from_slice(b"ustar\0");
    header[263..265].copy_from_slice(b"00");
    // Checksum over spaces, then six octal digits, NUL, space.
    header[148..156].copy_from_slice(b"        ");
    let sum: u32 = header.iter().map(|b| *b as u32).sum();
    header[148..156].copy_from_slice(format!("{sum:06o}\0 ").as_bytes());
    Ok(header)
}

fn prepare_interface(target: &Observation, deadline: Instant) -> Result<(), String> {
    confirm_project(target, deadline)?;
    podman(
        &[
            "exec",
            "--user",
            "0:0",
            &target.id,
            "/bin/sh",
            "-ceu",
            INTERFACE_SCRIPT,
        ],
        deadline,
    )?;
    Ok(())
}

struct FdGuard(RawFd);

impl Drop for FdGuard {
    fn drop(&mut self) {
        if self.0 >= 0 {
            unsafe { libc::close(self.0) };
        }
    }
}

fn attach_interface(
    target: &Observation,
    muse_socket: &str,
    deadline: Instant,
) -> Result<(), String> {
    // attachLaunchInterface with kind "muse".
    let source = public_socket_directory(muse_socket)?;
    let c = CString::new(source).map_err(|_| go_errno(libc::EINVAL))?;
    let source_fd = unsafe {
        libc::open(
            c.as_ptr(),
            libc::O_PATH | libc::O_DIRECTORY | libc::O_NOFOLLOW | libc::O_CLOEXEC,
            0,
        )
    };
    if source_fd < 0 {
        return Err(go_errno(last_errno()));
    }
    let _source_guard = FdGuard(source_fd);
    let tree = syscall_open_tree(source_fd)?;
    let _tree_guard = FdGuard(tree);
    restrict_interface_mount(tree)?;
    attach_project_mount(target, tree, deadline)
}

fn syscall_open_tree(source_fd: RawFd) -> Result<RawFd, String> {
    let empty = c"";
    let tree = unsafe {
        libc::syscall(
            libc::SYS_open_tree,
            source_fd as libc::c_long,
            empty.as_ptr(),
            (libc::OPEN_TREE_CLONE | libc::OPEN_TREE_CLOEXEC | libc::AT_EMPTY_PATH as u32)
                as libc::c_long,
        )
    };
    if tree < 0 {
        return Err(format!(
            "clone public interface mount: {}",
            go_errno(last_errno())
        ));
    }
    Ok(tree as RawFd)
}

fn restrict_interface_mount(tree: RawFd) -> Result<(), String> {
    let mut attr: libc::mount_attr = unsafe { std::mem::zeroed() };
    attr.attr_set = libc::MOUNT_ATTR_RDONLY
        | libc::MOUNT_ATTR_NOSUID
        | libc::MOUNT_ATTR_NODEV
        | libc::MOUNT_ATTR_NOEXEC;
    let empty = c"";
    let rc = unsafe {
        libc::syscall(
            libc::SYS_mount_setattr,
            tree as libc::c_long,
            empty.as_ptr(),
            libc::AT_EMPTY_PATH as libc::c_long,
            &mut attr as *mut libc::mount_attr,
            std::mem::size_of::<libc::mount_attr>() as libc::c_long,
        )
    };
    if rc != 0 {
        return Err(format!(
            "restrict public interface mount: {}",
            go_errno(last_errno())
        ));
    }
    Ok(())
}

fn attach_project_mount(
    target: &Observation,
    tree: RawFd,
    deadline: Instant,
) -> Result<(), String> {
    let root = format!("/proc/{}", target.pid);
    let dest = format!("{root}/root/run/soda-muse-interface");
    let cdest = CString::new(dest).map_err(|_| go_errno(libc::EINVAL))?;
    let target_fd = unsafe {
        libc::open(
            cdest.as_ptr(),
            libc::O_PATH | libc::O_DIRECTORY | libc::O_NOFOLLOW | libc::O_CLOEXEC,
            0,
        )
    };
    if target_fd < 0 {
        return Err(go_errno(last_errno()));
    }
    let _target_guard = FdGuard(target_fd);
    let ns = format!("{root}/ns/mnt");
    let cns = CString::new(ns).map_err(|_| go_errno(libc::EINVAL))?;
    let namespace = unsafe { libc::open(cns.as_ptr(), libc::O_RDONLY | libc::O_CLOEXEC, 0) };
    if namespace < 0 {
        return Err(go_errno(last_errno()));
    }
    let _ns_guard = FdGuard(namespace);
    // The incarnation check runs after both opens, just before entry.
    confirm_project(target, deadline)?;
    if unsafe { libc::unshare(libc::CLONE_FS) } != 0 {
        return Err(format!(
            "separate maintenance filesystem context: {}",
            go_errno(last_errno())
        ));
    }
    if unsafe { libc::setns(namespace, libc::CLONE_NEWNS) } != 0 {
        return Err(format!(
            "enter project mount namespace: {}",
            go_errno(last_errno())
        ));
    }
    let empty = c"";
    let rc = unsafe {
        libc::syscall(
            libc::SYS_move_mount,
            tree as libc::c_long,
            empty.as_ptr(),
            target_fd as libc::c_long,
            empty.as_ptr(),
            (libc::MOVE_MOUNT_F_EMPTY_PATH | libc::MOVE_MOUNT_T_EMPTY_PATH) as libc::c_long,
        )
    };
    if rc != 0 {
        return Err(format!(
            "attach public interface mount: {}",
            go_errno(last_errno())
        ));
    }
    Ok(())
}

fn public_socket_directory(socket: &str) -> Result<String, String> {
    if !socket.starts_with('/') || go_base(socket) != "launch.sock" {
        return Err(String::from("explicit public launch socket required"));
    }
    let root = go_dir(socket);
    let only = String::from("launch directory must contain only the public socket");
    let entries = fs::read_dir(&root).map_err(|_| only.clone())?;
    let mut names: Vec<String> = Vec::new();
    for entry in entries {
        let entry = entry.map_err(|_| only.clone())?;
        names.push(entry.file_name().to_string_lossy().into_owned());
    }
    names.sort();
    if names.len() != 1 || names[0] != "launch.sock" {
        return Err(only);
    }
    validate_public_socket(socket)?;
    validate_interface_directory(&root)?;
    Ok(root)
}

fn validate_public_socket(socket: &str) -> Result<(), String> {
    let info = fs::symlink_metadata(socket)
        .map_err(|_| String::from("public launch socket is unavailable"))?;
    if !info.file_type().is_socket() {
        return Err(String::from("public launch socket is unavailable"));
    }
    if info.uid() != 0 || info.mode() & 0o777 != 0o666 {
        return Err(String::from(
            "public launch socket must be root-owned and public",
        ));
    }
    Ok(())
}

fn validate_interface_directory(root: &str) -> Result<(), String> {
    let info = fs::symlink_metadata(root)
        .map_err(|_| String::from("public launch directory must be a protected directory"))?;
    if !info.is_dir() || info.mode() & 0o777 & 0o022 != 0 {
        return Err(String::from(
            "public launch directory must be a protected directory",
        ));
    }
    if info.uid() != 0 || info.mode() & 0o777 & 0o055 != 0o055 {
        return Err(String::from(
            "public launch directory must be root-owned and accessible",
        ));
    }
    Ok(())
}

#[cfg(test)]
mod tests {
    use super::config::{load_config, Config};
    use super::options::{default_tools, parse, usage_text};
    use super::*;
    use std::io::Write as _;
    use std::os::unix::fs::PermissionsExt;
    use std::sync::atomic::{AtomicU64, Ordering};

    static TEST_SEQ: AtomicU64 = AtomicU64::new(0);

    struct TestDir(std::path::PathBuf);

    impl TestDir {
        fn make(tag: &str) -> TestDir {
            let id = TEST_SEQ.fetch_add(1, Ordering::SeqCst);
            let dir = std::env::temp_dir().join(format!(
                "smm-test-{}-{}-{}",
                std::process::id(),
                id,
                tag
            ));
            let _ = fs::remove_dir_all(&dir);
            fs::create_dir_all(&dir).unwrap();
            TestDir(dir)
        }

        fn path(&self, name: &str) -> String {
            self.0.join(name).to_string_lossy().into_owned()
        }

        fn dir_str(&self) -> String {
            self.0.to_string_lossy().into_owned()
        }
    }

    impl Drop for TestDir {
        fn drop(&mut self) {
            let _ = fs::remove_dir_all(&self.0);
        }
    }

    fn is_root() -> bool {
        unsafe { libc::geteuid() == 0 }
    }

    #[test]
    fn sha256_known_answers() {
        let vectors: &[(&[u8], &str)] = &[
            (
                b"",
                "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
            ),
            (
                b"abc",
                "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad",
            ),
            // 55 bytes: padding fits in the final block.
            (
                b"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
                "9f4390f8d30c2dd92ec9f095b65e2b9ae9b0a925a5258e241c9f1e910f734318",
            ),
            // 56 bytes: padding spills into a second block.
            (
                b"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
                "b35439a4ac6f0948b6d6f9e3c6af0f5f590ce20f1bde7090ef7970686ec6738a",
            ),
            // 64 bytes: exactly one full block plus padding block.
            (
                b"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
                "ffe054fe7ae0cb6dc65c3af9b61d5209f439851db43d0ba5997337df154668eb",
            ),
        ];
        for (input, want) in vectors {
            let mut h = Sha256::new();
            // Feed one byte at a time to stress the streaming buffer.
            for b in input.iter() {
                h.update(std::slice::from_ref(b));
            }
            assert_eq!(&hex_encode(&h.finish()), want);
        }
    }

    #[test]
    fn go_quoted_vectors() {
        assert_eq!(go_quoted("plain"), "\"plain\"");
        assert_eq!(
            go_quoted("a\x01b\x7f\u{80}é\"\\"),
            "\"a\\x01b\\x7f\\u0080é\\\"\\\\\""
        );
        assert_eq!(go_quoted("\u{a0}"), "\"\\u00a0\"");
        assert_eq!(
            go_quoted("\u{200b}\u{2028}\u{e000}\u{f0000}"),
            "\"\\u200b\\u2028\\ue000\\U000f0000\""
        );
        assert_eq!(go_quoted("\n\r\t"), "\"\\n\\r\\t\"");
        assert_eq!(go_quoted("\u{7}\u{8}\u{c}\u{b}"), "\"\\a\\b\\f\\v\"");
        assert_eq!(go_quoted("space kept"), "\"space kept\"");
    }

    #[test]
    fn errno_table_spot_checks() {
        assert_eq!(go_errno(libc::ENOENT), "no such file or directory");
        assert_eq!(go_errno(libc::EACCES), "permission denied");
        assert_eq!(go_errno(libc::EPIPE), "broken pipe");
        assert_eq!(go_errno(libc::ENOSYS), "function not implemented");
        assert_eq!(go_errno(libc::ELOOP), "too many levels of symbolic links");
        assert_eq!(go_errno(libc::EINVAL), "invalid argument");
        assert_eq!(go_errno(libc::EISDIR), "is a directory");
        assert_eq!(go_errno(9999), "errno 9999");
    }

    #[test]
    fn config_read_error_shapes() {
        let dir = TestDir::make("cfgread");
        let missing = dir.path("nope.json");
        assert_eq!(
            load_config(&missing).unwrap_err(),
            format!("open {missing}: no such file or directory")
        );
        let sub = dir.path("sub");
        fs::create_dir(&sub).unwrap();
        assert_eq!(
            load_config(&sub).unwrap_err(),
            format!("read {sub}: is a directory")
        );
        // Decode runs before the release lookup.
        let bad = dir.path("bad.json");
        fs::write(&bad, b"{oops").unwrap();
        assert_eq!(
            load_config(&bad).unwrap_err(),
            "invalid character 'o' looking for beginning of object key string"
        );
    }

    #[test]
    fn go_path_helpers() {
        assert_eq!(
            go_clean("/usr/libexec/soda/../../share/soda/muse-tools"),
            "/usr/share/soda/muse-tools"
        );
        assert_eq!(go_clean("a//b/./c/"), "a/b/c");
        assert_eq!(go_clean(""), ".");
        assert_eq!(go_clean("/"), "/");
        assert_eq!(go_base("/x/launch.sock"), "launch.sock");
        assert_eq!(go_base("/x/launch.sock/"), "launch.sock");
        assert_eq!(go_base("/"), "/");
        assert_eq!(go_base(""), ".");
        assert_eq!(go_dir("/x/launch.sock"), "/x");
        assert_eq!(go_dir("/launch.sock"), "/");
        assert_eq!(go_dir("/x/launch.sock/"), "/x/launch.sock");
        assert_eq!(default_tools(), "/usr/share/soda/muse-tools");
    }

    fn args(v: &[&str]) -> Vec<String> {
        v.iter().map(|s| s.to_string()).collect()
    }

    const PROJECT: &str = "p0123456789abcdef01234567";

    #[test]
    fn flag_parsing_vectors() {
        let o = parse(&args(&["--project", PROJECT])).unwrap();
        assert_eq!(o.config, "/etc/soda/host.json");
        assert_eq!(o.project, PROJECT);
        assert_eq!(o.tools, "/usr/share/soda/muse-tools");
        assert!(!o.bind_only);
        let o = parse(&args(&[
            "--config=/x/host.json",
            "--tools",
            "/y/tools",
            "--bind-only",
            "--project",
            PROJECT,
        ]))
        .unwrap();
        assert_eq!(o.config, "/x/host.json");
        assert_eq!(o.tools, "/y/tools");
        assert!(o.bind_only);
        let o = parse(&args(&["-project", PROJECT, "-bind-only=false"])).unwrap();
        assert!(!o.bind_only);
        let o = parse(&args(&["--project", PROJECT, "--", "--bind-only"]));
        assert!(o.is_err());
        // First non-flag argument stops parsing.
        assert!(parse(&args(&["--project", PROJECT, "extra"])).is_err());
        assert!(parse(&args(&["extra", "--project", PROJECT])).is_err());
        // Usage errors print usage and name the flag.
        assert_eq!(
            parse(&args(&["--project"])).unwrap_err(),
            "flag needs an argument: -project"
        );
        assert_eq!(
            parse(&args(&["--nope", "x", "--project", PROJECT])).unwrap_err(),
            "flag provided but not defined: -nope"
        );
        assert_eq!(
            parse(&args(&["---project", PROJECT])).unwrap_err(),
            "bad flag syntax: ---project"
        );
        assert_eq!(parse(&args(&["-=x"])).unwrap_err(), "bad flag syntax: -=x");
        assert_eq!(
            parse(&args(&["--bind-only=maybe"])).unwrap_err(),
            "invalid boolean value \"maybe\" for -bind-only: parse error"
        );
        assert_eq!(parse(&args(&["-h"])).unwrap_err(), "flag: help requested");
        assert_eq!(
            usage_text("/usr/share/soda/muse-tools"),
            "Usage of soda-muse-maintain:\n  -bind-only\n    \trestore only the launch interface\n  -config string\n    \toperator host configuration (default \"/etc/soda/host.json\")\n  -project string\n    \texact project identity\n  -tools string\n    \tinstalled public tool directory (default \"/usr/share/soda/muse-tools\")\n"
        );
        // Validation errors.
        assert_eq!(
            parse(&args(&["--project", "../foreign"])).unwrap_err(),
            "explicit project and absolute maintenance paths required"
        );
        assert_eq!(
            parse(&args(&["--project", "P0123456789ABCDEF01234567"])).unwrap_err(),
            "explicit project and absolute maintenance paths required"
        );
        assert_eq!(
            parse(&args(&["--project", PROJECT, "--config", "relative.json"])).unwrap_err(),
            "explicit project and absolute maintenance paths required"
        );
        assert_eq!(
            parse(&args(&[])).unwrap_err(),
            "explicit project and absolute maintenance paths required"
        );
    }

    #[test]
    fn host_config_decode_vectors() {
        assert_eq!(decode_host_config(b"").unwrap_err(), "EOF");
        assert_eq!(decode_host_config(b"   ").unwrap_err(), "EOF");
        let c = decode_host_config(b"null").unwrap();
        assert_eq!(c.image, "");
        assert_eq!(
            decode_host_config(b"[1,2]").unwrap_err(),
            "json: cannot unmarshal array into Go value of type host.Config"
        );
        assert_eq!(
            decode_host_config(b"\"str\"").unwrap_err(),
            "json: cannot unmarshal string into Go value of type host.Config"
        );
        assert_eq!(decode_host_config(b"[1,2").unwrap_err(), "unexpected EOF");
        // Last duplicate wins; keys match case-insensitively.
        let c = decode_host_config(br#"{"image": "a", "image": "b", "NETWORK": "n"}"#).unwrap();
        assert_eq!(c.image, "b");
        assert_eq!(c.network, "n");
        let c = decode_host_config(br#"{"tailnet_management": true}"#).unwrap();
        assert!(c.tailnet_management);
        let c = decode_host_config(br#"{"image": null, "tailnet_management": null}"#).unwrap();
        assert_eq!(c.image, "");
        assert!(!c.tailnet_management);
        // Type errors name the key as written.
        assert_eq!(
            decode_host_config(br#"{"muse_sha256": 1}"#).unwrap_err(),
            "json: cannot unmarshal number into Go struct field Config.muse_sha256 of type string"
        );
        assert_eq!(
            decode_host_config(br#"{"MUSE_SHA256": 1}"#).unwrap_err(),
            "json: cannot unmarshal number into Go struct field Config.MUSE_SHA256 of type string"
        );
        assert_eq!(
            decode_host_config(br#"{"tailnet_management": "yes"}"#).unwrap_err(),
            "json: cannot unmarshal string into Go struct field Config.tailnet_management of type bool"
        );
        assert_eq!(
            decode_host_config(br#"{"muse_sha256": [1,2]}"#).unwrap_err(),
            "json: cannot unmarshal array into Go struct field Config.muse_sha256 of type string"
        );
        assert_eq!(
            decode_host_config(br#"{"muse_sha256": {"a":1}}"#).unwrap_err(),
            "json: cannot unmarshal object into Go struct field Config.muse_sha256 of type string"
        );
        // Unknown fields, including null-valued ones.
        assert_eq!(
            decode_host_config(br#"{"bogus": null}"#).unwrap_err(),
            "json: unknown field \"bogus\""
        );
        assert_eq!(
            decode_host_config(br#"{"quo\"te": 1}"#).unwrap_err(),
            "json: unknown field \"quo\\\"te\""
        );
        assert_eq!(
            decode_host_config(br#"{"muse_sha256": "x", "bogus": 1}"#).unwrap_err(),
            "json: unknown field \"bogus\""
        );
        // Broken values beat unknown fields; a later syntax error beats a
        // saved error; the first saved error beats later saved errors.
        assert_eq!(
            decode_host_config(br#"{"bogus": truX}"#).unwrap_err(),
            "invalid character 'X' in literal true (expecting 'e')"
        );
        assert_eq!(
            decode_host_config(br#"{"bogus": [1,2}"#).unwrap_err(),
            "invalid character '}' after array element"
        );
        assert_eq!(
            decode_host_config(br#"{"bogus": 1, "tailnet_management": truX}"#).unwrap_err(),
            "invalid character 'X' in literal true (expecting 'e')"
        );
        assert_eq!(
            decode_host_config(br#"{"muse_sha256": 1, "tailnet_management": truX}"#).unwrap_err(),
            "invalid character 'X' in literal true (expecting 'e')"
        );
        assert_eq!(
            decode_host_config(br#"{"tailnet_management": "x", "muse_sha256": 1}"#).unwrap_err(),
            "json: cannot unmarshal string into Go struct field Config.tailnet_management of type bool"
        );
        assert_eq!(
            decode_host_config(br#"{"muse_sha256": 1, "bogus": 2}"#).unwrap_err(),
            "json: cannot unmarshal number into Go struct field Config.muse_sha256 of type string"
        );
        assert_eq!(
            decode_host_config(br#"{"bogus": 2, "muse_sha256": 1}"#).unwrap_err(),
            "json: unknown field \"bogus\""
        );
        // Escapes and structural errors, byte-identical to encoding/json.
        assert_eq!(
            decode_host_config(br#"{"muse_sha256": "a\qb"}"#).unwrap_err(),
            "invalid escape sequence `\\q` in string"
        );
        assert_eq!(
            decode_host_config(b"{\"muse_sha256\": \"a\x01b\"}").unwrap_err(),
            "invalid character '\\x01' in string"
        );
        assert_eq!(
            decode_host_config(br#"{"muse_sha256": "\u00G1"}"#).unwrap_err(),
            "invalid escape sequence `\\u00G1` in string"
        );
        assert_eq!(
            decode_host_config(br#"{"muse_sha256": "\uD800\u00G1"}"#).unwrap_err(),
            "invalid escape sequence `\\u00G1` in string"
        );
        let c = decode_host_config("{\"image\": \"a�b\"}".as_bytes()).unwrap();
        assert_eq!(c.image, "a�b");
        let c = decode_host_config(br#"{"image": "a\uD800b"}"#).unwrap();
        assert_eq!(c.image, "a\u{FFFD}b");
        let c = decode_host_config(br#"{"image": "a\uD800\u0041"}"#).unwrap();
        assert_eq!(c.image, "a\u{FFFD}A");
        assert_eq!(
            decode_host_config(b"{': 1}").unwrap_err(),
            "invalid character '\\'' looking for beginning of object key string"
        );
        assert_eq!(
            decode_host_config(br#"{"a": "b",}"#).unwrap_err(),
            "invalid character '}' looking for beginning of object key string"
        );
        assert_eq!(
            decode_host_config(b"{\"a\": \"b\"").unwrap_err(),
            "unexpected EOF"
        );
        assert_eq!(decode_host_config(b"nul").unwrap_err(), "unexpected EOF");
    }

    #[test]
    fn prefix_parity_vectors() {
        let cases: &[(&str, Option<&str>)] = &[
            ("10.0.0.0/24", None),
            ("10.0.0.0/0", None),
            ("10.0.0.0/32", None),
            ("::/0", None),
            ("::/128", None),
            ("fe80::1/128", None),
            ("::ffff:1.2.3.4/128", None),
            ("1:2:3:4:5:6:1.2.3.4/128", None),
            (
                "1.2.3.4/33",
                Some("netip.ParsePrefix(\"1.2.3.4/33\"): prefix length out of range"),
            ),
            (
                "::gggg/1",
                Some(
                    "netip.ParsePrefix(\"::gggg/1\"): ParseAddr(\"::gggg\"): each colon-separated field must have at least one digit (at \"gggg\")",
                ),
            ),
            ("1.2.3.4", Some("netip.ParsePrefix(\"1.2.3.4\"): no '/'")),
            (
                "1.2.3/24",
                Some("netip.ParsePrefix(\"1.2.3/24\"): ParseAddr(\"1.2.3\"): IPv4 address too short"),
            ),
            (
                "1.2.3.4.5/24",
                Some(
                    "netip.ParsePrefix(\"1.2.3.4.5/24\"): ParseAddr(\"1.2.3.4.5\"): IPv4 address too long",
                ),
            ),
            (
                "fe80::1%eth0/64",
                Some(
                    "netip.ParsePrefix(\"fe80::1%eth0/64\"): IPv6 zones cannot be present in a prefix",
                ),
            ),
            (
                "1.2.3.4/ab",
                Some("netip.ParsePrefix(\"1.2.3.4/ab\"): bad bits after slash: \"ab\""),
            ),
            (
                "1.2.3.4/",
                Some("netip.ParsePrefix(\"1.2.3.4/\"): bad bits after slash: \"\""),
            ),
            (
                "/24",
                Some("netip.ParsePrefix(\"/24\"): ParseAddr(\"\"): unable to parse IP"),
            ),
            (
                "1.2.3.4/-1",
                Some("netip.ParsePrefix(\"1.2.3.4/-1\"): bad bits after slash: \"-1\""),
            ),
            (
                "::/129",
                Some("netip.ParsePrefix(\"::/129\"): prefix length out of range"),
            ),
            (
                "1::2::3/64",
                Some(
                    "netip.ParsePrefix(\"1::2::3/64\"): ParseAddr(\"1::2::3\"): multiple :: in address (at \":3\")",
                ),
            ),
            (
                "1.2.3.256/24",
                Some(
                    "netip.ParsePrefix(\"1.2.3.256/24\"): ParseAddr(\"1.2.3.256\"): IPv4 field has value >255",
                ),
            ),
            (
                "12345::/33",
                Some(
                    "netip.ParsePrefix(\"12345::/33\"): ParseAddr(\"12345::\"): each group must have 4 or less digits (at \"12345::\")",
                ),
            ),
            (
                "1.2.3.4/ 24",
                Some("netip.ParsePrefix(\"1.2.3.4/ 24\"): bad bits after slash: \" 24\""),
            ),
            (
                "1.2.3.4\x01/24",
                Some(
                    "netip.ParsePrefix(\"1.2.3.4\\x01/24\"): ParseAddr(\"1.2.3.4\\x01\"): unexpected character (at \"\\x01\")",
                ),
            ),
            (
                "1.2.3.4/\x034",
                Some("netip.ParsePrefix(\"1.2.3.4/\\x034\"): bad bits after slash: \"\\x034\""),
            ),
            (
                "01.2.3.4/24",
                Some(
                    "netip.ParsePrefix(\"01.2.3.4/24\"): ParseAddr(\"01.2.3.4\"): IPv4 field has octet with leading zero",
                ),
            ),
            // Left-to-right order: the value error fires before the length check.
            (
                "1.2.3.99999.5/24",
                Some(
                    "netip.ParsePrefix(\"1.2.3.99999.5/24\"): ParseAddr(\"1.2.3.99999.5\"): IPv4 field has value >255",
                ),
            ),
            (
                "1..2.3/24",
                Some(
                    "netip.ParsePrefix(\"1..2.3/24\"): ParseAddr(\"1..2.3\"): IPv4 field must have at least one digit (at \".2.3\")",
                ),
            ),
            (
                "1.2.3./24",
                Some(
                    "netip.ParsePrefix(\"1.2.3./24\"): ParseAddr(\"1.2.3.\"): IPv4 field must have at least one digit (at \".\")",
                ),
            ),
            (
                "12g4::/64",
                Some(
                    "netip.ParsePrefix(\"12g4::/64\"): ParseAddr(\"12g4::\"): unexpected character, want colon (at \"g4::\")",
                ),
            ),
            (
                "1.2.3.4::/64",
                Some(
                    "netip.ParsePrefix(\"1.2.3.4::/64\"): ParseAddr(\"1.2.3.4::\"): unexpected character (at \"::\")",
                ),
            ),
            (
                "1:2.3.4.5/24",
                Some(
                    "netip.ParsePrefix(\"1:2.3.4.5/24\"): ParseAddr(\"1:2.3.4.5\"): embedded IPv4 address must replace the final 2 fields of the address (at \"2.3.4.5\")",
                ),
            ),
            (
                "::1:2:3:4:5:6:7:1.2.3.4/64",
                Some(
                    "netip.ParsePrefix(\"::1:2:3:4:5:6:7:1.2.3.4/64\"): ParseAddr(\"::1:2:3:4:5:6:7:1.2.3.4\"): too many hex fields to fit an embedded IPv4 at the end of the address (at \"1.2.3.4\")",
                ),
            ),
            (
                "::ffff:1.2.3.999/64",
                Some(
                    "netip.ParsePrefix(\"::ffff:1.2.3.999/64\"): ParseAddr(\"::ffff:1.2.3.999\"): IPv4 field has value >255",
                ),
            ),
            (
                "fe80::1%/64",
                Some(
                    "netip.ParsePrefix(\"fe80::1%/64\"): ParseAddr(\"fe80::1%\"): zone must be a non-empty string",
                ),
            ),
            (
                "abcd%eth0/64",
                Some(
                    "netip.ParsePrefix(\"abcd%eth0/64\"): ParseAddr(\"abcd%eth0\"): missing IPv6 address",
                ),
            ),
            (
                "1.2.3.4%eth0/24",
                Some(
                    "netip.ParsePrefix(\"1.2.3.4%eth0/24\"): ParseAddr(\"1.2.3.4%eth0\"): unexpected character (at \"%eth0\")",
                ),
            ),
            (
                "1:2:3:4:5:6:7:8:9/64",
                Some(
                    "netip.ParsePrefix(\"1:2:3:4:5:6:7:8:9/64\"): ParseAddr(\"1:2:3:4:5:6:7:8:9\"): trailing garbage after address (at \"9\")",
                ),
            ),
            (
                "1:2:3/64",
                Some(
                    "netip.ParsePrefix(\"1:2:3/64\"): ParseAddr(\"1:2:3\"): address string too short",
                ),
            ),
            (
                "::1:2:3:4:5:6:7:8/64",
                Some(
                    "netip.ParsePrefix(\"::1:2:3:4:5:6:7:8/64\"): ParseAddr(\"::1:2:3:4:5:6:7:8\"): the :: must expand to at least one field of zeros",
                ),
            ),
            (
                "1.2.3.4/00",
                Some("netip.ParsePrefix(\"1.2.3.4/00\"): bad bits after slash: \"00\""),
            ),
            (
                "1.2.3.4/+5",
                Some("netip.ParsePrefix(\"1.2.3.4/+5\"): bad bits after slash: \"+5\""),
            ),
            (
                "1.2.3.4/99999999999999999999999",
                Some(
                    "netip.ParsePrefix(\"1.2.3.4/99999999999999999999999\"): bad bits after slash: \"99999999999999999999999\"",
                ),
            ),
        ];
        for (input, expected) in cases {
            match (parse_prefix(input), expected) {
                (Ok(()), None) => {}
                (Err(got), Some(want)) => assert_eq!(&got, want, "input {input:?}"),
                (Ok(()), Some(want)) => panic!("input {input:?}: accepted, want {want}"),
                (Err(got), None) => panic!("input {input:?}: rejected: {got}"),
            }
        }
    }

    fn hex_string(c: char, len: usize) -> String {
        std::iter::repeat_n(c, len).collect()
    }

    fn valid_release_json() -> (String, String, String) {
        let rev = hex_string('b', 40);
        let coreos = "41.20250101.3.0";
        let id = format!("{coreos}.soda-{}", &rev[..12]);
        let prefix = "ghcr.io/test/soda";
        let mut images = String::from("{");
        let mut cfgs = std::collections::HashMap::new();
        for (i, name) in [
            "dashboard",
            "forgejo",
            "extension",
            "proxy",
            "project-os",
            "tailnet",
        ]
        .iter()
        .enumerate()
        {
            let c = char::from_digit(i as u32 + 1, 16).unwrap();
            let m = char::from_digit(i as u32 + 7, 16).unwrap();
            let cfg = format!("sha256:{}", hex_string(c, 64));
            let man = format!("sha256:{}", hex_string(m, 64));
            let arch = hex_string('d', 64);
            let reference = format!("{prefix}-{name}@{man}");
            images.push_str(&format!(
                "\"{name}\":{{\"Reference\":\"{reference}\",\"Config\":\"{cfg}\",\"Manifest\":\"{man}\",\"ArchiveSHA256\":\"{arch}\"}},"
            ));
            cfgs.insert(*name, cfg);
        }
        images.pop();
        images.push('}');
        let base = format!(
            "quay.io/fedora/fedora-coreos@sha256:{}",
            hex_string('0', 64)
        );
        let json = format!(
            "{{\"Format\":3,\"ID\":\"{id}\",\"Revision\":\"{rev}\",\"Architecture\":\"x86_64\",\"CoreOS\":\"{coreos}\",\"Base\":\"{base}\",\"RepositoryPrefix\":\"{prefix}\",\"Schema\":1,\"PresentationSHA256\":\"{}\",\"HostPackagesSHA256\":\"{}\",\"Images\":{images},\"UpgradeFrom\":[]}}",
            hex_string('e', 64),
            hex_string('f', 64),
        );
        (json, cfgs["project-os"].clone(), cfgs["tailnet"].clone())
    }

    #[test]
    fn release_payload_accept_and_reject() {
        let dir = TestDir::make("release");
        let (valid, project_cfg, tailnet_cfg) = valid_release_json();
        let path = dir.path("release.json");
        fs::write(&path, &valid).unwrap();
        let payload = load_release_payload(&path).expect("valid payload refused");
        assert_eq!(payload.architecture, "x86_64");
        assert_eq!(
            payload.image_config("project-os").as_deref(),
            Some(project_cfg.as_str())
        );
        assert_eq!(
            payload.image_config("tailnet").as_deref(),
            Some(tailnet_cfg.as_str())
        );
        // Each mutation is rejected.
        let forgejo_cfg = format!("sha256:{}", hex_string('2', 64));
        let ext_cfg = format!("sha256:{}", hex_string('3', 64));
        let mutations: Vec<(&str, String)> = vec![
            ("format", valid.replacen("\"Format\":3", "\"Format\":2", 1)),
            (
                "revision",
                valid.replacen(&hex_string('b', 40), &hex_string('b', 39), 1),
            ),
            ("id", valid.replacen(".soda-bbbbbbbbbbbb", ".soda-cccccccccccc", 1)),
            ("coreos", valid.replacen("41.20250101.3.0", "41.1", 1)),
            ("arch", valid.replacen("\"x86_64\"", "\"aarch64\"", 1)),
            ("prefix", valid.replacen("ghcr.io/test/soda", "docker.io/test/soda", 1)),
            ("schema", valid.replacen("\"Schema\":1", "\"Schema\":0", 1)),
            (
                "presentation",
                valid.replacen(&hex_string('e', 64), &format!("g{}", hex_string('e', 63)), 1),
            ),
            ("reference", valid.replacen("-project-os@sha256:", "-project-os@sha257:", 1)),
            ("ext-forgejo", valid.replacen(&ext_cfg, &forgejo_cfg, 1)),
            ("missing-name", valid.replacen("\"tailnet\":{", "\"tailnet2\":{", 1)),
            (
                "seventh",
                valid.replacen("\"UpgradeFrom\"", "\"extra\":{\"Reference\":\"\",\"Config\":\"\",\"Manifest\":\"\",\"ArchiveSHA256\":\"\"},\"UpgradeFrom", 1),
            ),
            ("unknown", valid.replacen("\"Format\":3,", "\"Format\":3,\"Bogus\":1,", 1)),
            ("trailing", format!("{valid} {{}}")),
            ("trailing-comma", valid.replacen("\"UpgradeFrom\":[]}", "\"UpgradeFrom\":[],}", 1)),
            (
                "null-images",
                valid.replacen(
                    &valid[valid.find("\"Images\":").unwrap()..valid.find(",\"UpgradeFrom\"").unwrap()],
                    "\"Images\":null",
                    1,
                ),
            ),
        ];
        for (tag, bad) in &mutations {
            let p = dir.path(&format!("bad-{tag}.json"));
            fs::write(&p, bad).unwrap();
            assert!(
                load_release_payload(&p).is_err(),
                "mutation accepted: {tag}"
            );
        }
        // Duplicates apply last-wins; null UpgradeFrom stays empty.
        let dup = valid.replacen("\"Format\":3,", "\"Format\":2,\"Format\":3,", 1);
        let p = dir.path("dup.json");
        fs::write(&p, &dup).unwrap();
        assert!(load_release_payload(&p).is_ok(), "duplicate rejected");
        let dup_image = valid.replacen(
            "\"Images\":{\"dashboard\":{",
            "\"Images\":{\"dashboard\":{\"Reference\":\"\",\"Config\":\"\",\"Manifest\":\"\",\"ArchiveSHA256\":\"\"},\"dashboard\":{",
            1,
        );
        let p = dir.path("dup-image.json");
        fs::write(&p, &dup_image).unwrap();
        assert!(load_release_payload(&p).is_ok(), "duplicate image rejected");
        let null_upgrade = valid.replacen("\"UpgradeFrom\":[]", "\"UpgradeFrom\":null", 1);
        let p = dir.path("null-upgrade.json");
        fs::write(&p, &null_upgrade).unwrap();
        assert!(
            load_release_payload(&p).is_ok(),
            "null UpgradeFrom rejected"
        );
        // Shape rejections: symlink, missing, directory, oversize.
        let link = dir.path("link.json");
        std::os::unix::fs::symlink(&path, &link).unwrap();
        assert!(load_release_payload(&link).is_err(), "symlink accepted");
        assert!(load_release_payload(&dir.path("missing.json")).is_err());
        assert!(load_release_payload(&dir.dir_str()).is_err());
        let big = dir.path("big.json");
        fs::write(&big, vec![b' '; (4 << 20) + 1]).unwrap();
        assert!(load_release_payload(&big).is_err(), "oversize accepted");
    }

    #[test]
    fn release_images_apply_and_conflict() {
        let dir = TestDir::make("relapply");
        let (valid, project_cfg, tailnet_cfg) = valid_release_json();
        let path = dir.path("release.json");
        fs::write(&path, &valid).unwrap();
        let mut c = Config::default();
        apply_release_images(&mut c, &path).unwrap();
        assert_eq!(c.image, project_cfg);
        assert_eq!(c.tailnet_image, "");
        c.tailnet_management = true;
        apply_release_images(&mut c, &path).unwrap();
        assert_eq!(c.tailnet_image, tailnet_cfg);
        c.image = String::from("other");
        assert_eq!(
            apply_release_images(&mut c, &path).unwrap_err(),
            "saved image selection conflicts with appliance release; explicit migration required"
        );
        let mut c = Config::default();
        assert_eq!(
            apply_release_images(&mut c, &dir.path("missing.json")).unwrap_err(),
            "immutable appliance image defaults unavailable"
        );
    }

    #[test]
    fn runtime_config_validation_order() {
        // Muse errors precede identity, subnet, tailnet, and network errors.
        let c = Config {
            muse_sha256: hex_string('a', 64),
            subnet: String::from("bogus"),
            ..Default::default()
        };
        assert_eq!(
            validate_runtime_config(&c).unwrap_err(),
            "explicit muse socket, broker socket and release digest required"
        );
        // Subnet errors precede tailnet and network errors.
        let c = Config {
            subnet: String::from("bogus"),
            ..Default::default()
        };
        assert!(validate_runtime_config(&c)
            .unwrap_err()
            .starts_with("netip.ParsePrefix"));
        // Tailnet errors precede network errors.
        let c = Config {
            subnet: String::from("10.0.0.0/24"),
            tailnet_image: String::from("bare-hex"),
            ..Default::default()
        };
        assert_eq!(
            validate_runtime_config(&c).unwrap_err(),
            "invalid immutable Tailnet companion configuration"
        );
        // Network errors come last.
        let c = Config {
            subnet: String::from("10.0.0.0/24"),
            ..Default::default()
        };
        assert_eq!(
            validate_runtime_config(&c).unwrap_err(),
            "invalid native runtime configuration"
        );
        // A fully valid config passes.
        let c = Config {
            image: String::from("img"),
            network: String::from("net"),
            bridge: String::from("br"),
            subnet: String::from("10.0.0.0/24"),
            ..Default::default()
        };
        validate_runtime_config(&c).unwrap();
    }

    #[test]
    fn observation_decode_and_validate() {
        let id = hex_string('a', 64);
        let body = format!(
            "{{\"id\":\"{id}\",\"project\":\"{PROJECT}\",\"owner\":\"42\",\"pid\":123,\"running\":true}}"
        );
        let o = decode_observation(body.as_bytes()).unwrap();
        assert_eq!(o.pid, 123);
        assert!(o.running);
        validate_observation(&o, PROJECT).unwrap();
        // Identity failures.
        let mut foreign = Observation {
            id: id.clone(),
            project: String::from("pabcdef0123456789abcdef01"),
            owner: String::from("42"),
            pid: 1,
            running: true,
        };
        assert_eq!(
            validate_observation(&foreign, PROJECT).unwrap_err(),
            "project container identity differs"
        );
        foreign.project = PROJECT.to_string();
        foreign.id = hex_string('A', 64);
        assert_eq!(
            validate_observation(&foreign, PROJECT).unwrap_err(),
            "project container identity differs"
        );
        foreign.id = hex_string('a', 63);
        assert_eq!(
            validate_observation(&foreign, PROJECT).unwrap_err(),
            "project container identity differs"
        );
        // Owner failures and edge acceptances.
        foreign.id = id.clone();
        for owner in ["0", "-5", "abc", "", "42 ", "99999999999999999999999"] {
            foreign.owner = owner.to_string();
            assert_eq!(
                validate_observation(&foreign, PROJECT).unwrap_err(),
                "project owner label is invalid",
                "owner {owner:?}"
            );
        }
        foreign.owner = String::from("+42");
        validate_observation(&foreign, PROJECT).unwrap();
        foreign.owner = String::from("42");
        foreign.pid = -5;
        validate_observation(&foreign, PROJECT).unwrap();
        // Decode failures collapse to the observation message.
        for bad in [
            format!("{{\"id\":\"{id}\",\"project\":\"{PROJECT}\",\"owner\":\"42\",\"pid\":1.5,\"running\":true}}"),
            format!("{{\"id\":\"{id}\",\"project\":\"{PROJECT}\",\"owner\":\"42\",\"pid\":\"12\",\"running\":true}}"),
            format!("{{\"id\":\"{id}\",\"project\":\"{PROJECT}\",\"owner\":\"42\",\"pid\":1,\"running\":1}}"),
            format!("{{\"id\":\"{id}\",\"project\":\"{PROJECT}\",\"owner\":\"42\",\"pid\":1,\"running\":\"true\"}}"),
            format!("{{\"id\":\"{id}\",\"id\":\"{id}\",\"project\":\"{PROJECT}\",\"owner\":\"42\",\"pid\":1,\"running\":true}}"),
            format!("{{\"id\":\"{id}\",\"project\":\"{PROJECT}\",\"owner\":\"42\",\"pid\":1,\"running\":true,\"bogus\":1}}"),
            format!("{{\"id\":\"{id}\",\"project\":\"{PROJECT}\",\"owner\":\"42\",\"pid\":1,\"running\":true,}}"),
            format!("{{\"id\":\"{id}\",\"project\":\"{PROJECT}\",\"owner\":\"42\",\"pid\":1,\"running\":true}} {{}}"),
            String::from("[]"),
            String::from("null"),
            String::from("{oops"),
        ] {
            assert_eq!(
                decode_observation(bad.as_bytes()).unwrap_err(),
                "invalid project observation",
                "input {bad:?}"
            );
        }
        assert_eq!(
            decode_observation(&vec![b'{'; 2 << 20]).unwrap_err(),
            "invalid project observation"
        );
        assert_eq!(
            decode_observation(b"{\"id\":\"\xff\"}").unwrap_err(),
            "invalid project observation"
        );
        // Nulls decode to zero values, then fail validation.
        let nulls = decode_observation(
            format!("{{\"id\":null,\"project\":\"{PROJECT}\",\"owner\":null,\"pid\":null,\"running\":null}}")
                .as_bytes(),
        )
        .unwrap();
        assert_eq!(
            validate_observation(&nulls, PROJECT).unwrap_err(),
            "project container identity differs"
        );
        // Exact keys win over earlier case variants in sorted order.
        let other = hex_string('b', 64);
        let o = decode_observation(
            format!("{{\"ID\":\"{other}\",\"id\":\"{id}\",\"project\":\"{PROJECT}\",\"owner\":\"42\",\"pid\":1,\"running\":true}}")
                .as_bytes(),
        )
        .unwrap();
        assert_eq!(o.id, id);
        validate_observation(&o, PROJECT).unwrap();
    }

    #[test]
    fn tool_source_and_digest_admission() {
        let dir = TestDir::make("tools");
        let native = dir.path("muse-native");
        let body = b"synthetic native bytes";
        fs::write(&native, body).unwrap();
        fs::set_permissions(&native, fs::Permissions::from_mode(0o755)).unwrap();
        // Symlinks never admit, regardless of owner.
        let link = dir.path("muse");
        std::os::unix::fs::symlink(&native, &link).unwrap();
        assert_eq!(
            open_tool(&dir.dir_str(), "muse").unwrap_err(),
            "public tool source must be a regular root-owned executable"
        );
        // Group-writable and non-executable sources refuse alike.
        let weak = dir.path("weak");
        fs::write(&weak, body).unwrap();
        fs::set_permissions(&weak, fs::Permissions::from_mode(0o775)).unwrap();
        assert!(open_tool(&dir.dir_str(), "weak").is_err());
        fs::set_permissions(&weak, fs::Permissions::from_mode(0o644)).unwrap();
        assert!(open_tool(&dir.dir_str(), "weak").is_err());
        // Digest admission through an open tool fd.
        let c = CString::new(native.clone()).unwrap();
        let fd = unsafe { libc::open(c.as_ptr(), libc::O_RDONLY) };
        assert!(fd >= 0);
        let tool = Tool {
            name: String::from("muse-native"),
            fd,
            size: body.len() as u64,
        };
        assert_eq!(
            verify_native(&tool, &hex_string('0', 64)).unwrap_err(),
            "muse native digest mismatch"
        );
        let mut h = Sha256::new();
        h.update(body);
        verify_native(&tool, &hex_encode(&h.finish())).unwrap();
        // pread verification leaves the offset at zero.
        assert_eq!(unsafe { libc::lseek(fd, 0, libc::SEEK_CUR) }, 0);
        drop(tool);
        if is_root() {
            open_tool(&dir.dir_str(), "muse-native").expect("root-owned source refused");
        } else {
            assert!(open_tool(&dir.dir_str(), "muse-native").is_err());
        }
        // The version gate fires before any tool opens.
        assert_eq!(
            load_tools(
                "definitely-missing",
                &hex_string('0', 64),
                &format!("{MUSE_VERSION}-other")
            )
            .unwrap_err(),
            "muse maintenance version differs from pinned release"
        );
    }

    #[test]
    fn public_interface_admission() {
        use std::os::unix::net::UnixListener;
        let dir = TestDir::make("sock");
        let sock = dir.path("launch.sock");
        let listener = UnixListener::bind(&sock).unwrap();
        fs::set_permissions(&sock, fs::Permissions::from_mode(0o666)).unwrap();
        fs::set_permissions(dir.0.clone(), fs::Permissions::from_mode(0o755)).unwrap();
        if is_root() {
            assert_eq!(public_socket_directory(&sock).unwrap(), dir.dir_str());
            fs::set_permissions(dir.0.clone(), fs::Permissions::from_mode(0o775)).unwrap();
            assert_eq!(
                public_socket_directory(&sock).unwrap_err(),
                "public launch directory must be a protected directory"
            );
            fs::set_permissions(dir.0.clone(), fs::Permissions::from_mode(0o700)).unwrap();
            assert_eq!(
                public_socket_directory(&sock).unwrap_err(),
                "public launch directory must be root-owned and accessible"
            );
            fs::set_permissions(dir.0.clone(), fs::Permissions::from_mode(0o755)).unwrap();
        }
        assert_eq!(
            public_socket_directory("relative/launch.sock").unwrap_err(),
            "explicit public launch socket required"
        );
        assert_eq!(
            public_socket_directory("/tmp/other.sock").unwrap_err(),
            "explicit public launch socket required"
        );
        fs::write(dir.path("credential"), b"synthetic private data").unwrap();
        assert_eq!(
            public_socket_directory(&sock).unwrap_err(),
            "launch directory must contain only the public socket"
        );
        fs::remove_file(dir.path("credential")).unwrap();
        // Same message whether the uid or the mode check fires.
        fs::set_permissions(&sock, fs::Permissions::from_mode(0o600)).unwrap();
        assert_eq!(
            public_socket_directory(&sock).unwrap_err(),
            "public launch socket must be root-owned and public"
        );
        fs::set_permissions(&sock, fs::Permissions::from_mode(0o666)).unwrap();
        drop(listener);
        fs::remove_file(&sock).unwrap();
        fs::write(&sock, b"not a socket").unwrap();
        assert_eq!(
            public_socket_directory(&sock).unwrap_err(),
            "public launch socket is unavailable"
        );
    }

    #[test]
    fn tar_header_layout_matches_ustar() {
        let h = tar_header("muse", 14).unwrap();
        assert_eq!(&h[..4], b"muse");
        assert!(h[4..100].iter().all(|b| *b == 0));
        assert_eq!(&h[100..108], b"0000755\0");
        assert_eq!(&h[108..116], b"0000000\0");
        assert_eq!(&h[116..124], b"0000000\0");
        assert_eq!(&h[124..136], b"00000000016\0");
        assert_eq!(&h[136..148], b"00000000000\0");
        assert_eq!(h[156], b'0');
        assert_eq!(&h[257..263], b"ustar\0");
        assert_eq!(&h[263..265], b"00");
        let mut sum: u32 = 0;
        for (i, b) in h.iter().enumerate() {
            sum += if (148..156).contains(&i) {
                b' ' as u32
            } else {
                *b as u32
            };
        }
        assert_eq!(&h[148..156], format!("{sum:06o}\0 ").as_bytes());
        assert!(tar_header(&"n".repeat(101), 0).is_err());
    }

    fn synthetic_feeds(tools_dir: &TestDir) -> (Vec<(String, RawFd, u64)>, Vec<FdGuard>) {
        let mut feeds = Vec::new();
        let mut guards = Vec::new();
        for name in ["muse", "soda-identity-compose", "muse-native"] {
            let p = tools_dir.path(name);
            fs::write(&p, format!("synthetic {name}")).unwrap();
            let c = CString::new(p).unwrap();
            let fd = unsafe { libc::open(c.as_ptr(), libc::O_RDONLY) };
            assert!(fd >= 0);
            let len = format!("synthetic {name}").len() as u64;
            guards.push(FdGuard(fd));
            feeds.push((name.to_string(), fd, len));
        }
        (feeds, guards)
    }

    fn emit_synthetic() -> Vec<u8> {
        let tools_dir = TestDir::make("emit");
        let (feeds, guards) = synthetic_feeds(&tools_dir);
        let mut archive = Vec::new();
        emit_archive(
            &mut |b: &[u8]| {
                archive.extend_from_slice(b);
                Ok(())
            },
            &feeds,
        )
        .unwrap();
        drop(guards);
        archive
    }

    fn run_install_script(archive: &[u8], targets: &[String]) -> (bool, String) {
        let mut cmd = Command::new("/bin/sh");
        cmd.args(["-ceu", INSTALL_SCRIPT, "soda-muse-maintain"]);
        for t in targets {
            cmd.arg(t);
        }
        cmd.stdin(Stdio::piped())
            .stdout(Stdio::piped())
            .stderr(Stdio::piped());
        let mut child = cmd.spawn().unwrap();
        child.stdin.take().unwrap().write_all(archive).unwrap();
        let out = child.wait_with_output().unwrap();
        (
            out.status.success(),
            String::from_utf8_lossy(&out.stderr).into_owned(),
        )
    }

    #[test]
    fn tool_replacement_preserves_other_files_and_refuses_symlinks() {
        let dir = TestDir::make("stage");
        let targets: Vec<String> = ["bin/muse", "bin/soda-identity-compose", "libexec/soda/muse"]
            .iter()
            .map(|t| dir.path(t))
            .collect();
        for t in &targets {
            fs::create_dir_all(std::path::Path::new(t).parent().unwrap()).unwrap();
            fs::write(t, b"obsolete public tool").unwrap();
        }
        let retained = dir.path("account-state");
        fs::write(&retained, b"preserved account and project state").unwrap();
        fs::set_permissions(&retained, fs::Permissions::from_mode(0o600)).unwrap();
        let before = fs::symlink_metadata(&retained).unwrap();
        let archive = emit_synthetic();
        // 3 headers + 3 data blocks + 2 trailer blocks.
        assert_eq!(archive.len(), 3 * 512 + 3 * 512 + 1024);
        // The install script is byte-pinned where transcription once dropped
        // the space before the first loop's done.
        assert!(INSTALL_SCRIPT.contains("fi\n done\nfor target"));
        let (ok, stderr) = run_install_script(&archive, &targets);
        assert!(ok, "public replacement failed: {stderr}");
        let after = fs::symlink_metadata(&retained).unwrap();
        assert_eq!((before.dev(), before.ino()), (after.dev(), after.ino()));
        assert_eq!(
            fs::read(&retained).unwrap(),
            b"preserved account and project state"
        );
        for t in &targets {
            let body = fs::read(t).unwrap();
            assert!(body.starts_with(b"synthetic "), "tool not replaced: {t}");
        }
        fs::remove_file(&targets[0]).unwrap();
        std::os::unix::fs::symlink(&retained, &targets[0]).unwrap();
        let (ok, _) = run_install_script(&archive, &targets);
        assert!(!ok, "target symlink accepted");
        assert_eq!(
            fs::read(&retained).unwrap(),
            b"preserved account and project state"
        );
    }

    #[test]
    fn streamed_stage_delivery() {
        // A fake podman proves the feeder/child contract: the child must
        // not inherit the stdin write end, or its reader never sees EOF.
        // (A raw pipe without O_CLOEXEC deadlocked here until the deadline.)
        let dir = TestDir::make("streamed");
        let bindir = dir.path("bin");
        fs::create_dir(&bindir).unwrap();
        fs::write(
            format!("{bindir}/podman"),
            "#!/bin/sh\nprintf \"%s\\0\" \"$@\" > \"$E2E/argv.bin\"\ncat > \"$E2E/stdin.bin\"\nexit 0\n",
        )
        .unwrap();
        fs::set_permissions(
            format!("{bindir}/podman"),
            fs::Permissions::from_mode(0o755),
        )
        .unwrap();
        std::env::set_var(
            "PATH",
            format!("{bindir}:{}", std::env::var("PATH").unwrap()),
        );
        std::env::set_var("E2E", dir.dir_str());
        let tools = TestDir::make("streamed-tools");
        let (feeds, guards) = synthetic_feeds(&tools);
        let deadline = Instant::now() + Duration::from_secs(15);
        podman_streamed(
            move |w| feed_archive(w, &feeds, deadline),
            &["exec", "--user", "0:0", "-i", "some-id"],
            deadline,
        )
        .unwrap();
        drop(guards);
        assert_eq!(fs::read(dir.path("stdin.bin")).unwrap(), emit_synthetic());
        let argv = fs::read(dir.path("argv.bin")).unwrap();
        assert!(argv.starts_with(b"--remote=false\0exec\0--user\0"));
    }

    #[test]
    fn wait_output_reports_status() {
        let c = Command::new("/bin/sh")
            .args(["-c", "echo hello; exit 0"])
            .stdin(Stdio::null())
            .stdout(Stdio::piped())
            .stderr(Stdio::null())
            .spawn()
            .unwrap();
        let out = wait_output(c, Instant::now() + Duration::from_secs(5)).unwrap();
        assert_eq!(out, b"hello\n");
        let c = Command::new("/bin/sh")
            .args(["-c", "exit 3"])
            .stdin(Stdio::null())
            .stdout(Stdio::piped())
            .stderr(Stdio::null())
            .spawn()
            .unwrap();
        assert_eq!(
            wait_output(c, Instant::now() + Duration::from_secs(5)).unwrap_err(),
            "project maintenance command failed"
        );
    }

    #[test]
    fn short_tool_file_ends_copy_with_eof() {
        let dir = TestDir::make("short");
        let p = dir.path("tool");
        fs::write(&p, b"twelve bytes").unwrap();
        let c = CString::new(p).unwrap();
        let fd = unsafe { libc::open(c.as_ptr(), libc::O_RDONLY) };
        assert!(fd >= 0);
        let _guard = FdGuard(fd);
        // Claim more bytes than the file holds, like a shrink race.
        let feeds = vec![(String::from("tool"), fd, 1_000_000u64)];
        let mut out = Vec::new();
        assert_eq!(
            emit_archive(
                &mut |b: &[u8]| {
                    out.extend_from_slice(b);
                    Ok(())
                },
                &feeds,
            )
            .unwrap_err(),
            "EOF"
        );
    }
}

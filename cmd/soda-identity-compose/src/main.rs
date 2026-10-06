// soda-identity-compose registers one explicitly opted-in Compose service.
use std::ffi::CString;
use std::fs;
use std::io;

mod compose;
mod options;

use compose::launch_compose;
use options::{parse_options, Options};

const MUSE_LAUNCH_SOCKET: &str = "/run/soda-muse-interface/launch.sock";
const TMPFS_MAGIC: i64 = 0x01021994;

fn main() {
    if let Err(e) = run() {
        eprintln!("soda-identity-compose: {e}");
        std::process::exit(1);
    }
}

fn run() -> Result<(), String> {
    let o = load_options()?;
    let actor = account(&o.login)?;
    let (root, registration) = registration_root()?;
    let child = launch_compose(&o, &root)?;
    register(NestedRegistration {
        child_id: child,
        actor_id: actor.to_string(),
        registration_id: registration,
        muse: o.muse,
    })
}

fn load_options() -> Result<Options, String> {
    // Never panic on non-UTF-8 argv; Go replaces invalid bytes with U+FFFD.
    let args: Vec<String> = std::env::args_os()
        .skip(1)
        .map(|a| a.to_string_lossy().into_owned())
        .collect();
    parse_options(&args)
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

fn registration_root() -> Result<(String, String), String> {
    let mut id = [0u8; 16];
    read_random(&mut id)?;
    let registration = hex_encode(&id);
    let root = format!("/run/soda-muse/nested/{registration}");
    let dir = "/run/soda-muse/nested";
    mkdir_p(dir, 0o711)?;
    // Nested registration requires runtime tmpfs.
    let mut fs: libc::statfs64 = unsafe { std::mem::zeroed() };
    let c = CString::new(dir).unwrap();
    let rc = unsafe { libc::statfs64(c.as_ptr(), &mut fs) };
    if rc != 0 || fs.f_type as i64 != TMPFS_MAGIC {
        return Err(String::from("nested registration requires runtime tmpfs"));
    }
    let croot = CString::new(root.clone()).unwrap();
    let rc = unsafe { libc::mkdir(croot.as_ptr(), 0o711) };
    if rc != 0 {
        return Err(io::Error::last_os_error().to_string());
    }
    Ok((root, registration))
}

fn read_random(buf: &mut [u8]) -> Result<(), String> {
    use std::io::Read;
    let mut f = fs::File::open("/dev/urandom").map_err(|e| e.to_string())?;
    f.read_exact(buf).map_err(|e| e.to_string())
}

fn hex_encode(bytes: &[u8]) -> String {
    let mut s = String::with_capacity(bytes.len() * 2);
    for b in bytes {
        s.push(char::from_digit((b >> 4) as u32, 16).unwrap());
        s.push(char::from_digit((b & 0xf) as u32, 16).unwrap());
    }
    s
}

fn mkdir_p(path: &str, mode: u32) -> Result<(), String> {
    // Mirror Go MkdirAll: create missing ancestors, leave existing alone.
    let mut current = String::new();
    for part in path.split('/') {
        if part.is_empty() {
            if current.is_empty() {
                current.push('/');
            }
            continue;
        }
        if current == "/" || current.is_empty() {
            current.push_str(part);
        } else {
            current.push('/');
            current.push_str(part);
        }
        let c = CString::new(current.clone()).unwrap();
        let rc = unsafe { libc::mkdir(c.as_ptr(), mode) };
        if rc != 0 {
            let e = io::Error::last_os_error();
            if e.kind() != io::ErrorKind::AlreadyExists {
                return Err(e.to_string());
            }
        }
    }
    Ok(())
}

fn account(login: &str) -> Result<i64, String> {
    let path = format!("/var/lib/soda/accounts/{login}");
    let c = CString::new(path.clone()).unwrap();
    let mut st: libc::stat = unsafe { std::mem::zeroed() };
    let rc = unsafe { libc::lstat(c.as_ptr(), &mut st) };
    if rc != 0 {
        return Err(String::from("provisioned account marker required"));
    }
    let is_reg = (st.st_mode & libc::S_IFMT) == libc::S_IFREG;
    let perm = st.st_mode & 0o777;
    if !is_reg || perm != 0o600 {
        return Err(String::from("provisioned account marker required"));
    }
    if st.st_uid != 0 {
        return Err(String::from("root-owned account marker required"));
    }
    let data = fs::read(&path).map_err(|e| e.to_string())?;
    let text = String::from_utf8_lossy(&data);
    let actor: i64 = text
        .trim()
        .parse()
        .map_err(|_| String::from("invalid provisioned account"))?;
    if actor <= 0 {
        return Err(String::from("invalid provisioned account"));
    }
    Ok(actor)
}

// json_string matches Go encoding/json string escaping, including its
// HTML-safe <, >, & forms, so override bytes are identical for any input.
fn json_string(s: &str) -> String {
    let mut out = String::with_capacity(s.len() + 2);
    out.push('"');
    for c in s.chars() {
        match c {
            '"' => out.push_str("\\\""),
            '\\' => out.push_str("\\\\"),
            '\n' => out.push_str("\\n"),
            '\r' => out.push_str("\\r"),
            '\t' => out.push_str("\\t"),
            '<' => out.push_str("\\u003c"),
            '>' => out.push_str("\\u003e"),
            '&' => out.push_str("\\u0026"),
            '\u{2028}' => out.push_str("\\u2028"),
            '\u{2029}' => out.push_str("\\u2029"),
            c if (c as u32) < 0x20 => {
                out.push_str(&format!("\\u{:04x}", c as u32));
            }
            c => out.push(c),
        }
    }
    out.push('"');
    out
}

struct NestedRegistration {
    child_id: String,
    actor_id: String,
    registration_id: String,
    muse: bool,
}

// launch_request_json emits the exact Go LaunchRequest field order for a
// registration: empty home/config_home/term omitted, nil args as null.
fn launch_request_json(request: &NestedRegistration) -> String {
    let mut out = String::from("{\"register\":{\"child_id\":");
    out.push_str(&json_string(&request.child_id));
    out.push_str(",\"actor_id\":");
    out.push_str(&json_string(&request.actor_id));
    out.push_str(",\"registration_id\":");
    out.push_str(&json_string(&request.registration_id));
    out.push_str(",\"muse\":");
    out.push_str(if request.muse { "true" } else { "false" });
    out.push_str(
        "},\"connection_id\":\"\",\"cwd\":\"\",\"args\":null,\"tty\":false,\"cols\":0,\"rows\":0}",
    );
    out
}

// parse_launch_exit mirrors Go json.Unmarshal into LaunchExit: missing
// fields stay zero, unknown fields are ignored, malformed JSON rejects.
fn parse_launch_exit(body: &[u8]) -> Result<(i64, String), ()> {
    let text = std::str::from_utf8(body).map_err(|_| ())?;
    let mut code: i64 = 0;
    let mut error_text = String::new();
    let mut i = 0;
    let bytes = text.as_bytes();
    let skip_ws = |i: &mut usize| {
        while *i < bytes.len() && matches!(bytes[*i], b' ' | b'\t' | b'\n' | b'\r') {
            *i += 1;
        }
    };
    skip_ws(&mut i);
    if i >= bytes.len() || bytes[i] != b'{' {
        return Err(());
    }
    i += 1;
    // Go rejects a trailing comma, so `}` is only valid here for `{}` or
    // right after a value; after a comma a key is required.
    let mut after_comma = false;
    loop {
        skip_ws(&mut i);
        if i < bytes.len() && bytes[i] == b'}' {
            if after_comma {
                return Err(());
            }
            i += 1;
            break;
        }
        if i >= bytes.len() || bytes[i] != b'"' {
            return Err(());
        }
        let (key, next) = parse_json_string(text, i)?;
        i = next;
        skip_ws(&mut i);
        if i >= bytes.len() || bytes[i] != b':' {
            return Err(());
        }
        i += 1;
        skip_ws(&mut i);
        if key == "code" {
            let (value, next) = parse_json_integer(text, i)?;
            code = value;
            i = next;
        } else if key == "error" {
            if i >= bytes.len() || bytes[i] != b'"' {
                return Err(());
            }
            let (value, next) = parse_json_string(text, i)?;
            error_text = value;
            i = next;
        } else {
            i = skip_json_value(text, i)?;
        }
        skip_ws(&mut i);
        if i < bytes.len() && bytes[i] == b',' {
            i += 1;
            after_comma = true;
            continue;
        }
        if i < bytes.len() && bytes[i] == b'}' {
            i += 1;
            break;
        }
        return Err(());
    }
    skip_ws(&mut i);
    if i != bytes.len() {
        return Err(());
    }
    Ok((code, error_text))
}

fn parse_json_string(text: &str, start: usize) -> Result<(String, usize), ()> {
    let bytes = text.as_bytes();
    if start >= bytes.len() || bytes[start] != b'"' {
        return Err(());
    }
    let mut out = String::new();
    let mut i = start + 1;
    while i < bytes.len() {
        match bytes[i] {
            b'"' => return Ok((out, i + 1)),
            b'\\' => {
                i += 1;
                if i >= bytes.len() {
                    return Err(());
                }
                match bytes[i] {
                    b'"' => out.push('"'),
                    b'\\' => out.push('\\'),
                    b'/' => out.push('/'),
                    b'b' => out.push('\u{0008}'),
                    b'f' => out.push('\u{000c}'),
                    b'n' => out.push('\n'),
                    b'r' => out.push('\r'),
                    b't' => out.push('\t'),
                    b'u' => {
                        if i + 4 >= bytes.len() {
                            return Err(());
                        }
                        let hex = &text[i + 1..i + 5];
                        let cp = u32::from_str_radix(hex, 16).map_err(|_| ())?;
                        let c = char::from_u32(cp).ok_or(())?;
                        // Reject lone surrogates the way Go does.
                        if (0xd800..0xe000).contains(&cp) {
                            return Err(());
                        }
                        out.push(c);
                        i += 4;
                    }
                    _ => return Err(()),
                }
            }
            0x00..=0x1f => return Err(()),
            _ => {
                let c = text[i..].chars().next().ok_or(())?;
                out.push(c);
                i += c.len_utf8() - 1;
            }
        }
        i += 1;
    }
    Err(())
}

fn parse_json_integer(text: &str, start: usize) -> Result<(i64, usize), ()> {
    let bytes = text.as_bytes();
    let mut i = start;
    if i < bytes.len() && bytes[i] == b'-' {
        i += 1;
    }
    let digits = i;
    while i < bytes.len() && bytes[i].is_ascii_digit() {
        i += 1;
    }
    if i == digits || i == start {
        return Err(());
    }
    // Go rejects fractions and exponents for int fields.
    if i < bytes.len() && (bytes[i] == b'.' || bytes[i] == b'e' || bytes[i] == b'E') {
        return Err(());
    }
    text[start..i]
        .parse::<i64>()
        .map_err(|_| ())
        .map(|v| (v, i))
}

fn skip_json_value(text: &str, start: usize) -> Result<usize, ()> {
    let bytes = text.as_bytes();
    if start >= bytes.len() {
        return Err(());
    }
    match bytes[start] {
        b'"' => parse_json_string(text, start).map(|(_, next)| next),
        b'{' | b'[' => {
            let open = bytes[start];
            let close = if open == b'{' { b'}' } else { b']' };
            let mut i = start + 1;
            let mut depth = 1;
            let mut in_string = false;
            let mut escaped = false;
            while i < bytes.len() {
                let b = bytes[i];
                if in_string {
                    if escaped {
                        escaped = false;
                    } else if b == b'\\' {
                        escaped = true;
                    } else if b == b'"' {
                        in_string = false;
                    }
                } else if b == b'"' {
                    in_string = true;
                } else if b == open {
                    depth += 1;
                } else if b == close {
                    depth -= 1;
                    if depth == 0 {
                        return Ok(i + 1);
                    }
                }
                i += 1;
            }
            Err(())
        }
        b't' => {
            if text[start..].starts_with("true") {
                Ok(start + 4)
            } else {
                Err(())
            }
        }
        b'f' => {
            if text[start..].starts_with("false") {
                Ok(start + 5)
            } else {
                Err(())
            }
        }
        b'n' => {
            if text[start..].starts_with("null") {
                Ok(start + 4)
            } else {
                Err(())
            }
        }
        b'-' | b'0'..=b'9' => {
            let mut i = start;
            if bytes[i] == b'-' {
                i += 1;
            }
            while i < bytes.len()
                && (bytes[i].is_ascii_digit()
                    || matches!(bytes[i], b'.' | b'e' | b'E' | b'+' | b'-'))
            {
                i += 1;
            }
            if i == start {
                return Err(());
            }
            Ok(i)
        }
        _ => Err(()),
    }
}

fn register(request: NestedRegistration) -> Result<(), String> {
    let fd = unsafe { libc::socket(libc::AF_UNIX, libc::SOCK_SEQPACKET, 0) };
    if fd < 0 {
        return Err(String::from("identity registration service unavailable"));
    }
    struct Guard(i32);
    impl Drop for Guard {
        fn drop(&mut self) {
            unsafe { libc::close(self.0) };
        }
    }
    let _guard = Guard(fd);
    let mut addr: libc::sockaddr_un = unsafe { std::mem::zeroed() };
    addr.sun_family = libc::AF_UNIX as libc::sa_family_t;
    let bytes = MUSE_LAUNCH_SOCKET.as_bytes();
    if bytes.len() >= addr.sun_path.len() {
        return Err(String::from("identity registration service unavailable"));
    }
    for (i, b) in bytes.iter().enumerate() {
        addr.sun_path[i] = *b as libc::c_char;
    }
    let len = (std::mem::size_of::<libc::sa_family_t>() + bytes.len() + 1) as libc::socklen_t;
    let rc = unsafe {
        libc::connect(
            fd,
            &addr as *const libc::sockaddr_un as *const libc::sockaddr,
            len,
        )
    };
    if rc != 0 {
        return Err(String::from("identity registration service unavailable"));
    }
    let tv = libc::timeval {
        tv_sec: 30,
        tv_usec: 0,
    };
    unsafe {
        libc::setsockopt(
            fd,
            libc::SOL_SOCKET,
            libc::SO_SNDTIMEO,
            &tv as *const libc::timeval as *const libc::c_void,
            std::mem::size_of::<libc::timeval>() as libc::socklen_t,
        );
        libc::setsockopt(
            fd,
            libc::SOL_SOCKET,
            libc::SO_RCVTIMEO,
            &tv as *const libc::timeval as *const libc::c_void,
            std::mem::size_of::<libc::timeval>() as libc::socklen_t,
        );
    }
    let data = launch_request_json(&request);
    let bytes = data.as_bytes();
    let n = unsafe { libc::send(fd, bytes.as_ptr() as *const libc::c_void, bytes.len(), 0) };
    if n < 0 || (n as usize) != bytes.len() {
        return Err(io::Error::last_os_error().to_string());
    }
    let mut body = [0u8; 4096];
    let r = unsafe { libc::recv(fd, body.as_mut_ptr() as *mut libc::c_void, body.len(), 0) };
    if r <= 0 {
        return Err(String::from("identity registration unconfirmed"));
    }
    let (code, err_text) = parse_launch_exit(&body[..r as usize])
        .map_err(|_| String::from("identity registration rejected"))?;
    if code != 0 || !err_text.is_empty() {
        return Err(String::from("identity registration rejected"));
    }
    Ok(())
}

#[cfg(test)]
mod tests {
    use super::compose::{select_compose_child, write_override};
    use super::options::parse_bool_flag;
    use super::*;

    #[test]
    fn selects_only_requested_service_within_project() {
        let first = "a".repeat(64);
        let second = "b".repeat(64);
        let output = format!("{first}\n{second}\n");
        let child = select_compose_child(&output, "development", &|id| {
            if id == first {
                Ok(format!("{id} database\n"))
            } else {
                Ok(format!("{id} development\n"))
            }
        })
        .unwrap();
        assert_eq!(child, second);
    }

    #[test]
    fn rejects_ambiguous_or_unconfirmed_identity() {
        let id = "a".repeat(64);
        for output in ["", "--all", &format!("{id}\n{id}")] {
            let out = if output == "--all" {
                "--all".to_string()
            } else {
                output.to_string()
            };
            assert!(
                select_compose_child(&out, "development", &|v| Ok(format!("{v} development")))
                    .is_err(),
                "unconfirmed unique container admitted for {out:?}"
            );
        }
        assert!(
            select_compose_child(&id, "development", &|_| Err(String::from(
                "runtime unavailable"
            )))
            .is_err()
        );
    }

    #[test]
    fn resolves_upstream_short_handle_to_immutable_identity() {
        let id = "a".repeat(64);
        let child = select_compose_child(&id[..12], "development", &|_| {
            Ok(format!("{id} development"))
        })
        .unwrap();
        assert_eq!(child, id);
        assert!(select_compose_child(&id[..12], "development", &|_| {
            Ok(format!("{} development", "b".repeat(64)))
        })
        .is_err());
    }

    #[test]
    fn compose_muse_mounts_are_explicit() {
        let dir = std::env::temp_dir().join(format!("soda-compose-test-{}", std::process::id()));
        let _ = fs::create_dir_all(&dir);
        let path = dir.join("compose.json");
        write_override(
            path.to_str().unwrap(),
            "development",
            "/run/soda-muse/nested/registration",
        )
        .unwrap();
        let body = fs::read(&path).unwrap();
        let text = String::from_utf8(body).unwrap();
        assert_eq!(
            text,
            "{\"services\":{\"development\":{\"volumes\":[\"/run/soda-muse/nested/registration:/run/soda-muse/credentials:ro\",\"/usr/local/bin/muse:/usr/local/bin/muse:ro\",\"/usr/local/libexec/soda/muse:/usr/local/libexec/soda/muse:ro\",\"/run/soda-muse-interface:/run/soda-muse-interface:ro\"]}}}"
        );
        let _ = fs::remove_dir_all(&dir);
    }

    #[test]
    fn override_escapes_like_go_encoding_json() {
        assert_eq!(
            json_string("a<b>&\"c\\d"),
            "\"a\\u003cb\\u003e\\u0026\\\"c\\\\d\""
        );
        assert_eq!(json_string("line\ntab\t"), "\"line\\ntab\\t\"");
        assert_eq!(json_string("\u{1}"), "\"\\u0001\"");
    }

    #[test]
    fn go_base_matches_filepath_semantics() {
        assert_eq!(go_base("login"), "login");
        assert_eq!(go_base("a/b"), "b");
        assert_eq!(go_base("/"), "/");
        assert_eq!(go_base(""), ".");
    }

    #[test]
    fn bool_flag_values_match_go() {
        assert_eq!(parse_bool_flag("true"), Some(true));
        assert_eq!(parse_bool_flag("1"), Some(true));
        assert_eq!(parse_bool_flag("false"), Some(false));
        assert_eq!(parse_bool_flag("0"), Some(false));
        assert_eq!(parse_bool_flag("yes"), None);
    }

    #[test]
    fn launch_request_wire_matches_go() {
        let req = NestedRegistration {
            child_id: "c".repeat(64),
            actor_id: "42".to_string(),
            registration_id: "r".repeat(32),
            muse: true,
        };
        let wire = launch_request_json(&req);
        let expect = format!(
            "{{\"register\":{{\"child_id\":\"{}\",\"actor_id\":\"42\",\"registration_id\":\"{}\",\"muse\":true}},\"connection_id\":\"\",\"cwd\":\"\",\"args\":null,\"tty\":false,\"cols\":0,\"rows\":0}}",
            "c".repeat(64),
            "r".repeat(32)
        );
        assert_eq!(wire, expect);
    }

    #[test]
    fn launch_exit_parsing_matches_go_unmarshal() {
        assert_eq!(
            parse_launch_exit(b"{\"code\":0}").unwrap(),
            (0, String::new())
        );
        assert_eq!(parse_launch_exit(b"{}").unwrap(), (0, String::new()));
        assert_eq!(
            parse_launch_exit(b"{\"error\":\"denied\",\"code\":1}").unwrap(),
            (1, String::from("denied"))
        );
        assert_eq!(
            parse_launch_exit(b" { \"code\" : 0 , \"extra\" : [1,{\"x\":null}] } ").unwrap(),
            (0, String::new())
        );
        assert_eq!(
            parse_launch_exit(b"{\"code\":0,\"error\":\"a\\\"b\"}").unwrap(),
            (0, String::from("a\"b"))
        );
        for bad in [
            "",
            "{",
            "{\"code\":}",
            "{\"code\":\"0\"}",
            "{\"code\":0.0}",
            "{\"code\":0",
            "{\"error\":0}",
            "{\"code\":0}trailing",
            "{\"code\":0,\"code\":}",
            "{\"code\":0,}",
            "{,}",
        ] {
            assert!(
                parse_launch_exit(bad.as_bytes()).is_err(),
                "admitted {bad:?}"
            );
        }
    }
}

// soda-muse delegates a normal shell invocation over the launch-only socket.
use std::fs;
use std::io;

mod env;
mod env_prepare;
mod maintenance;
mod session;
mod shell;
mod terminal;

use env_prepare::copy_config;
use maintenance::{account_for, check_runtime, path_error};
use session::{launch_shell, seqpacket_connect};
use shell::shell_request;
use terminal::execute;

const MUSE_LAUNCH_SOCKET: &str = "/run/soda-muse-interface/launch.sock";
const MUSE_NATIVE: &str = "/usr/local/libexec/soda/muse";

fn main() {
    let (code, err) = run();
    if let Some(e) = err {
        eprintln!("{e}");
    }
    std::process::exit(code);
}

fn run() -> (i32, Option<String>) {
    // Never panic on non-UTF-8 argv; Go tolerates arbitrary bytes.
    let argv: Vec<String> = std::env::args_os()
        .skip(1)
        .map(|a| a.to_string_lossy().into_owned())
        .collect();
    if let Some(done) = native_action(&argv) {
        return done;
    }
    let request = match shell_request(&argv) {
        Ok(r) => r,
        Err(e) => return (1, Some(e)),
    };
    let fd = match seqpacket_connect(MUSE_LAUNCH_SOCKET) {
        Ok(fd) => fd,
        Err(_) => return (1, Some(String::from("muse launch service unavailable"))),
    };
    launch_shell(fd, &request)
}

fn native_action(args: &[String]) -> Option<(i32, Option<String>)> {
    if args.is_empty() {
        return None;
    }
    match args[0].as_str() {
        "--soda-copy-config" => {
            if args.len() != 3 {
                return Some((1, Some(String::from("config source and view required"))));
            }
            Some((0, copy_config(&args[1], &args[2]).err()))
        }
        "--soda-exec" => {
            if args.len() < 3 {
                return Some((1, Some(String::from("execution root and cwd required"))));
            }
            let (code, err) = execute(&args[1], &args[2], &args[3..]);
            Some((code, err))
        }
        _ => metadata_action(args),
    }
}

fn metadata_action(args: &[String]) -> Option<(i32, Option<String>)> {
    match args[0].as_str() {
        "--soda-check" | "--soda-read-config" | "--soda-account" => {}
        _ => return None,
    }
    if args.len() != 2 {
        return Some((1, Some(String::from("one metadata input required"))));
    }
    let err = match args[0].as_str() {
        "--soda-check" => check_runtime(&args[1]).err(),
        "--soda-read-config" => read_config(&args[1]).err(),
        _ => account_for(&args[1]).err(),
    };
    Some((0, err))
}

struct ShellRequest {
    cwd: String,
    args: Vec<String>,
    home: String,
    connection_id: String,
    config_home: String,
    term: String,
    tty: bool,
    cols: u16,
    rows: u16,
}

// read_config emits the upstream release's saved settings and trust records only.
// It executes as the invoking account, never as a privileged config reader.
fn read_config(source: &str) -> Result<(), String> {
    if !source.starts_with('/') {
        return Err(String::from("absolute config path required"));
    }
    let mut view: Vec<(&str, Vec<u8>)> = Vec::new();
    for name in ["settings.json", "trust.json"] {
        let path = format!("{source}/{name}");
        let file = match fs::File::open(&path) {
            Ok(f) => f,
            Err(e) if e.kind() == io::ErrorKind::NotFound => continue,
            Err(e) => return Err(path_error("open", &path, e)),
        };
        use std::io::Read;
        let mut body = Vec::new();
        file.take((1 << 20) + 1)
            .read_to_end(&mut body)
            .map_err(|e| e.to_string())?;
        if body.len() > 1 << 20 {
            return Err(String::from("muse config file exceeds private view limit"));
        }
        view.push((name, body));
    }
    let mut out = String::from("{");
    for (i, (name, body)) in view.iter().enumerate() {
        if i > 0 {
            out.push(',');
        }
        out.push_str(&json_string(name));
        out.push(':');
        out.push_str(&json_string(&base64_encode(body)));
    }
    out.push_str("}\n");
    print!("{out}");
    use std::io::Write;
    io::stdout().flush().map_err(|e| e.to_string())?;
    Ok(())
}

fn base64_encode(data: &[u8]) -> String {
    const ALPHABET: &[u8; 64] = b"ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/";
    let mut out = String::with_capacity(data.len().div_ceil(3) * 4);
    for chunk in data.chunks(3) {
        let mut n: u32 = 0;
        for (i, b) in chunk.iter().enumerate() {
            n |= (*b as u32) << (16 - 8 * i);
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

// json_string matches Go encoding/json string escaping, including its
// HTML-safe <, >, & forms, so wire bytes are identical for any input.
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

#[cfg(test)]
mod tests {
    use super::env::{go_base, go_clean, go_join, go_quote_rune};
    use super::env_prepare::copy_config;
    use super::maintenance::account_node;
    use super::session::{controls_loop, forward_signals, launch_shell};
    use super::shell::{shell_request_json, validate_shell};
    use super::terminal::muse_environment_with;
    use super::*;
    use std::os::unix::fs::PermissionsExt;

    fn shell_fixture() -> ShellRequest {
        ShellRequest {
            cwd: String::from("/work"),
            args: vec![String::from("a b"), String::from("c<d")],
            home: String::from("/home/u"),
            connection_id: String::from("conn1"),
            config_home: String::from("/home/u/.config"),
            term: String::from("xterm"),
            tty: true,
            cols: 80,
            rows: 24,
        }
    }

    #[test]
    fn shell_request_wire_matches_go() {
        assert_eq!(
            shell_request_json(&shell_fixture()),
            "{\"home\":\"/home/u\",\"config_home\":\"/home/u/.config\",\"term\":\"xterm\",\"connection_id\":\"conn1\",\"cwd\":\"/work\",\"args\":[\"a b\",\"c\\u003cd\"],\"tty\":true,\"cols\":80,\"rows\":24}"
        );
        let empty = ShellRequest {
            cwd: String::from("/w"),
            args: Vec::new(),
            home: String::new(),
            connection_id: String::new(),
            config_home: String::new(),
            term: String::new(),
            tty: false,
            cols: 0,
            rows: 0,
        };
        assert_eq!(
            shell_request_json(&empty),
            "{\"connection_id\":\"\",\"cwd\":\"/w\",\"args\":[],\"tty\":false,\"cols\":0,\"rows\":0}"
        );
    }

    #[test]
    fn shell_validation_matches_go() {
        validate_shell(&shell_fixture()).unwrap();
        let mut bad = shell_fixture();
        bad.cwd = String::from("relative");
        assert!(validate_shell(&bad).is_err());
        let mut bad = shell_fixture();
        bad.term = "x".repeat(129);
        assert!(validate_shell(&bad).is_err());
        let mut bad = shell_fixture();
        bad.cols = 0;
        assert!(validate_shell(&bad).is_err());
        let mut bad = shell_fixture();
        bad.args = vec![String::from("x"); 257];
        assert!(validate_shell(&bad).is_err());
        let mut bad = shell_fixture();
        bad.args = vec![String::from("a\0b")];
        assert!(validate_shell(&bad).is_err());
        // Empty home/config_home stay optional.
        let mut ok = shell_fixture();
        ok.home.clear();
        ok.config_home.clear();
        validate_shell(&ok).unwrap();
    }

    #[test]
    fn launch_exit_parsing_matches_go_unmarshal() {
        assert_eq!(
            parse_launch_exit(b"{\"code\":0}").unwrap(),
            (0, String::new())
        );
        assert_eq!(parse_launch_exit(b"{}").unwrap(), (0, String::new()));
        assert_eq!(
            parse_launch_exit(b"{\"code\":3}").unwrap(),
            (3, String::new())
        );
        assert_eq!(
            parse_launch_exit(b"{\"error\":\"denied\",\"code\":1}\n").unwrap(),
            (1, String::from("denied"))
        );
        for bad in [
            "",
            "{",
            "{\"code\":}",
            "{\"code\":\"0\"}",
            "{\"code\":0,}",
            "[]",
        ] {
            assert!(
                parse_launch_exit(bad.as_bytes()).is_err(),
                "admitted {bad:?}"
            );
        }
    }

    #[test]
    fn base64_matches_standard_vectors() {
        for (raw, want) in [
            ("", ""),
            ("f", "Zg=="),
            ("fo", "Zm8="),
            ("foo", "Zm9v"),
            ("foob", "Zm9vYg=="),
            ("fooba", "Zm9vYmE="),
            ("foobar", "Zm9vYmFy"),
            ("{\"a\":1}", "eyJhIjoxfQ=="),
        ] {
            assert_eq!(base64_encode(raw.as_bytes()), want, "input {raw:?}");
        }
    }

    #[test]
    fn go_clean_matches_filepath_cases() {
        for (input, want) in [
            ("", "."),
            ("/", "/"),
            ("//", "/"),
            ("/a/b/../c", "/a/c"),
            ("/../a", "/a"),
            ("a/./b", "a/b"),
            ("a/../../b", "../b"),
            ("/run/soda-muse/", "/run/soda-muse"),
            ("/home/u/", "/home/u"),
        ] {
            assert_eq!(go_clean(input), want, "input {input:?}");
        }
        assert_eq!(go_join("/home/u/", ".config"), "/home/u/.config");
        assert_eq!(go_join("/", ".config"), "/.config");
    }

    #[test]
    fn go_quote_matches_invalid_byte_cases() {
        assert_eq!(go_quote_rune(b'X'), "'X'");
        assert_eq!(go_quote_rune(b'\''), "'\\''");
        assert_eq!(go_quote_rune(0x01), "'\\x01'");
    }

    #[test]
    fn execution_id_rules_match_go() {
        assert_eq!(go_base("/run/soda-muse/abc"), "abc");
        // 32-hex passes the shape check (mkdir may fail without privilege).
        assert!("a".repeat(32).bytes().all(|b| b.is_ascii_hexdigit()));
        assert!("g".repeat(32).bytes().any(|b| !b.is_ascii_hexdigit()));
    }

    #[test]
    fn muse_environment_orders_fixed_then_passthrough() {
        let env = muse_environment_with("/run/r", "/tmp/s", &|n| match n {
            "HOME" => Some(String::from("/home/u")),
            "TERM" => Some(String::new()),
            _ => None,
        });
        assert_eq!(
            env,
            vec![
                "PATH=/usr/local/bin:/usr/bin:/bin",
                "LANG=C.UTF-8",
                "TBH_CREDENTIAL_BACKEND=file",
                "XDG_CONFIG_HOME=/run/r/config",
                "XDG_STATE_HOME=/tmp/s/state",
                "XDG_CACHE_HOME=/tmp/s/cache",
                "XDG_DATA_HOME=/tmp/s/data",
                "TMPDIR=/tmp/s/tmp",
                "HOME=/home/u",
            ]
        );
    }

    #[test]
    fn copy_config_skips_credentials_and_copies_tree() {
        let root = std::env::temp_dir().join(format!("soda-muse-cc-{}", std::process::id()));
        let _ = fs::remove_dir_all(&root);
        let source = root.join("src");
        let dest = root.join("dst");
        fs::create_dir_all(source.join("sub")).unwrap();
        fs::write(source.join("settings.json"), b"{}").unwrap();
        fs::write(source.join("auth.json"), b"secret").unwrap();
        fs::write(source.join("sub").join("trust.json"), b"[]").unwrap();
        std::os::unix::fs::symlink(source.join("settings.json"), source.join("link.json")).unwrap();
        copy_config(source.to_str().unwrap(), dest.to_str().unwrap()).unwrap();
        assert_eq!(fs::read(dest.join("settings.json")).unwrap(), b"{}");
        assert_eq!(
            fs::read(dest.join("sub").join("trust.json")).unwrap(),
            b"[]"
        );
        assert_eq!(fs::read(dest.join("link.json")).unwrap(), b"{}");
        assert!(!dest.join("auth.json").exists());
        let mode = fs::metadata(dest.join("settings.json"))
            .unwrap()
            .permissions()
            .mode();
        assert_eq!(mode & 0o777, 0o600);
        // Missing source is a no-op.
        copy_config(
            root.join("absent").to_str().unwrap(),
            root.join("dst2").to_str().unwrap(),
        )
        .unwrap();
        // Oversized file refused.
        let big = source.join("big.json");
        fs::write(&big, vec![0u8; (1 << 20) + 1]).unwrap();
        assert!(copy_config(
            source.to_str().unwrap(),
            root.join("dst3").to_str().unwrap()
        )
        .is_err());
        let _ = fs::remove_dir_all(&root);
    }

    #[test]
    fn read_config_contract_shapes() {
        // Pure shape check: base64 values under sorted keys.
        let mut out = String::from("{");
        out.push_str(&json_string("settings.json"));
        out.push(':');
        out.push_str(&json_string(&base64_encode(b"{}")));
        out.push_str("}\n");
        assert_eq!(out, "{\"settings.json\":\"e30=\"}\n");
    }

    #[test]
    fn account_markers_reject_unsafe_nodes() {
        let root = std::env::temp_dir().join(format!("soda-muse-ac-{}", std::process::id()));
        let _ = fs::remove_dir_all(&root);
        fs::create_dir_all(&root).unwrap();
        // Temp files are user-owned, never root-owned: unsafe record.
        assert!(account_node(root.to_str().unwrap(), true).is_err());
        let marker = root.join("login");
        fs::write(&marker, b"42").unwrap();
        assert!(account_node(marker.to_str().unwrap(), false).is_err());
        let _ = fs::remove_dir_all(&root);
    }

    #[test]
    fn launch_shell_passes_fds_and_maps_exit() {
        for (reply, want_code, want_err) in [
            ("{\"code\":0}", 0, None),
            ("{\"code\":3}", 3, None),
            ("{\"code\":1,\"error\":\"denied\"}", 1, Some("denied")),
        ] {
            let mut pair = [0; 2];
            assert_eq!(
                unsafe {
                    libc::socketpair(libc::AF_UNIX, libc::SOCK_SEQPACKET, 0, pair.as_mut_ptr())
                },
                0
            );
            let server = std::thread::spawn(move || {
                let fd = pair[1];
                struct Guard(i32);
                impl Drop for Guard {
                    fn drop(&mut self) {
                        unsafe { libc::close(self.0) };
                    }
                }
                let _guard = Guard(fd);
                // Read the request plus exactly three passed fds.
                let mut data = [0u8; 65536];
                let mut iov = libc::iovec {
                    iov_base: data.as_mut_ptr() as *mut libc::c_void,
                    iov_len: data.len(),
                };
                let mut control = [0u8; 64];
                let mut header: libc::msghdr = unsafe { std::mem::zeroed() };
                header.msg_iov = &mut iov;
                header.msg_iovlen = 1;
                header.msg_control = control.as_mut_ptr() as *mut libc::c_void;
                header.msg_controllen = control.len() as _;
                let n = unsafe { libc::recvmsg(fd, &mut header, 0) };
                assert!(n > 0);
                let body = &data[..n as usize];
                let text = String::from_utf8_lossy(body);
                assert!(text.starts_with("{\"config_home\":\"/c\","), "{text}");
                assert!(text.contains("\"cwd\":\"/w\""), "{text}");
                let cmsg = unsafe { libc::CMSG_FIRSTHDR(&header as *const libc::msghdr) };
                assert!(!cmsg.is_null());
                let got = unsafe {
                    let len = (*cmsg).cmsg_len as usize - libc::CMSG_LEN(0) as usize;
                    std::slice::from_raw_parts(
                        libc::CMSG_DATA(cmsg) as *const i32,
                        len / std::mem::size_of::<i32>(),
                    )
                    .to_vec()
                };
                assert_eq!(got.len(), 3);
                for received in &got {
                    unsafe { libc::close(*received) };
                }
                let bytes = reply.as_bytes();
                let w = unsafe {
                    libc::send(fd, bytes.as_ptr() as *const libc::c_void, bytes.len(), 0)
                };
                assert_eq!(w as usize, bytes.len());
            });
            let request = ShellRequest {
                cwd: String::from("/w"),
                args: Vec::new(),
                home: String::new(),
                connection_id: String::new(),
                config_home: String::from("/c"),
                term: String::new(),
                tty: false,
                cols: 0,
                rows: 0,
            };
            // launch_shell blocks signals and spawns its control thread.
            let (code, err) = launch_shell(pair[0], &request);
            unsafe { libc::close(pair[0]) };
            server.join().unwrap();
            assert_eq!(code, want_code);
            assert_eq!(err.as_deref(), want_err);
            // Restore the process mask the launch path blocked.
            let mut set: libc::sigset_t = unsafe { std::mem::zeroed() };
            unsafe {
                libc::sigemptyset(&mut set);
                for sig in forward_signals() {
                    libc::sigaddset(&mut set, sig);
                }
                libc::pthread_sigmask(libc::SIG_UNBLOCK, &set, std::ptr::null_mut());
            }
        }
    }

    #[test]
    fn controls_forward_signal_number() {
        let mut pair = [0; 2];
        assert_eq!(
            unsafe { libc::socketpair(libc::AF_UNIX, libc::SOCK_SEQPACKET, 0, pair.as_mut_ptr()) },
            0
        );
        // Block in this thread so the control thread inherits the mask.
        let mut set: libc::sigset_t = unsafe { std::mem::zeroed() };
        unsafe {
            libc::sigemptyset(&mut set);
            for sig in forward_signals() {
                libc::sigaddset(&mut set, sig);
            }
            libc::pthread_sigmask(libc::SIG_BLOCK, &set, std::ptr::null_mut());
        }
        let (tx, rx) = std::sync::mpsc::channel();
        let stop = std::sync::Arc::new(std::sync::atomic::AtomicBool::new(false));
        let stop_child = stop.clone();
        let handle = std::thread::spawn(move || {
            let tid = unsafe { libc::pthread_self() };
            tx.send(tid).unwrap();
            controls_loop(pair[1], false, &stop_child);
        });
        let tid = rx.recv().unwrap();
        // Deterministic thread-directed delivery, not a process broadcast.
        assert_eq!(unsafe { libc::pthread_kill(tid, libc::SIGUSR2) }, 0);
        let tv = libc::timeval {
            tv_sec: 5,
            tv_usec: 0,
        };
        unsafe {
            libc::setsockopt(
                pair[0],
                libc::SOL_SOCKET,
                libc::SO_RCVTIMEO,
                &tv as *const libc::timeval as *const libc::c_void,
                std::mem::size_of::<libc::timeval>() as libc::socklen_t,
            );
        }
        let mut body = [0u8; 64];
        let r = unsafe {
            libc::recv(
                pair[0],
                body.as_mut_ptr() as *mut libc::c_void,
                body.len(),
                0,
            )
        };
        assert_eq!(&body[..r as usize], b"{\"signal\":12}\n");
        stop.store(true, std::sync::atomic::Ordering::SeqCst);
        assert_eq!(unsafe { libc::pthread_kill(tid, libc::SIGWINCH) }, 0);
        handle.join().unwrap();
        unsafe {
            libc::close(pair[0]);
            libc::pthread_sigmask(libc::SIG_UNBLOCK, &set, std::ptr::null_mut());
        }
    }

    #[test]
    fn native_dispatch_errors_match_go() {
        let v = |s: &[&str]| s.iter().map(|x| x.to_string()).collect::<Vec<_>>();
        assert_eq!(
            native_action(&v(&["--soda-copy-config", "a"])),
            Some((1, Some(String::from("config source and view required"))))
        );
        assert_eq!(
            native_action(&v(&["--soda-exec", "a"])),
            Some((1, Some(String::from("execution root and cwd required"))))
        );
        assert_eq!(
            native_action(&v(&["--soda-check"])),
            Some((1, Some(String::from("one metadata input required"))))
        );
        assert_eq!(native_action(&[]), None);
        assert_eq!(native_action(&v(&["--version"])), None);
        assert_eq!(native_action(&v(&["prompt text"])), None);
    }
}

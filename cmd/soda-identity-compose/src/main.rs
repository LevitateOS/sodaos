// soda-identity-compose registers one explicitly opted-in Compose service.
#[cfg(test)]
use std::fs;

mod compose;
mod launch_wire;
mod options;
mod registration;

use compose::launch_compose;
use launch_wire::NestedRegistration;
use options::{parse_options, Options};
use registration::{account, register, registration_root};

const MUSE_LAUNCH_SOCKET: &str = "/run/soda-muse-interface/launch.sock";

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

#[cfg(test)]
mod tests {
    use super::compose::{select_compose_child, write_override};
    use super::launch_wire::{
        json_string, launch_request_json, parse_launch_exit, NestedRegistration,
    };
    use super::options::parse_bool_flag;
    use super::registration::go_base;
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

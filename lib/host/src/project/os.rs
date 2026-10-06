use std::collections::HashMap;

use super::Executor;
use crate::domain;
use std::time::Instant;

impl<E: Executor> super::Runtime<E> {
    /// `ObserveOS`: read-only OS release observation, never starting a
    /// stopped container. Unparseable or missing data reports unavailable.
    pub fn observe_os(&self, id: &str, deadline: Instant) -> Result<domain::OsObservation, String> {
        let (env, _) = self.inspect(id, deadline)?;
        let mut result = domain::OsObservation {
            environment: env.clone(),
            release: None,
            unavailable: true,
        };
        if !env.running {
            return Ok(result);
        }
        // Fast regular-file refusal (the Python helper opened O_NONBLOCK and
        // required S_ISREG); the read below still bounds the window.
        let target = format!("soda-{id}");
        if self
            .podman(
                &[],
                &["exec", &target, "/usr/bin/test", "-f", "/etc/os-release"],
                deadline,
            )
            .is_err()
        {
            return Ok(result);
        }
        let raw = match self.podman(
            &[],
            &[
                "exec",
                &target,
                "/usr/bin/head",
                "-c",
                "4097",
                "/etc/os-release",
            ],
            deadline,
        ) {
            Ok(raw) if raw.len() <= 4096 => raw,
            _ => return Ok(result),
        };
        let release = match parse_os_release(&raw) {
            Ok(r) => r,
            Err(_) => return Ok(result),
        };
        if !domain::valid_os_release(&release) {
            return Ok(result);
        }
        result.release = Some(release);
        result.unavailable = false;
        Ok(result)
    }
}

// ---------- os-release parsing (project_os.py rules) ----------

fn is_python_space(c: char) -> bool {
    matches!(
        c,
        '\u{0009}'..='\u{000D}'
            | ' '
            | '\u{0085}'
            | '\u{00A0}'
            | '\u{1680}'
            | '\u{2000}'..='\u{200A}'
            | '\u{2028}'
            | '\u{2029}'
            | '\u{202F}'
            | '\u{205F}'
            | '\u{3000}'
    )
}

fn split_lines(text: &str) -> Vec<&str> {
    // Python str.splitlines boundaries.
    let mut lines = Vec::new();
    let mut start = 0;
    let bytes = text.as_bytes();
    let mut i = 0;
    while i < bytes.len() {
        let c = text[i..].chars().next().unwrap();
        let len = c.len_utf8();
        let boundary = matches!(
            c,
            '\n' | '\r'
                | '\u{000B}'
                | '\u{000C}'
                | '\u{001C}'
                | '\u{001D}'
                | '\u{001E}'
                | '\u{0085}'
                | '\u{2028}'
                | '\u{2029}'
        );
        if boundary {
            lines.push(&text[start..i]);
            if c == '\r' && bytes.get(i + 1) == Some(&b'\n') {
                i += 1;
            }
            i += len;
            start = i;
        } else {
            i += len;
        }
    }
    lines.push(&text[start..]);
    // Python drops the final empty element after a trailing break.
    if text.is_empty() {
        return Vec::new();
    }
    if lines.last() == Some(&"") && start == text.len() && !text.is_empty() {
        lines.pop();
    }
    lines
}

/// POSIX shell word parsing for one os-release value: `shlex.split(value,
/// comments=False, posix=True)` semantics with `#` ordinary.
fn shlex_posix(value: &str) -> Result<Vec<String>, ()> {
    let mut words = Vec::new();
    let mut word = String::new();
    let mut started = false;
    let mut chars = value.chars().peekable();
    while let Some(c) = chars.next() {
        if matches!(c, ' ' | '\t' | '\r' | '\n') {
            if started {
                words.push(std::mem::take(&mut word));
                started = false;
            }
            continue;
        }
        started = true;
        match c {
            '\'' => {
                let mut closed = false;
                for c in chars.by_ref() {
                    if c == '\'' {
                        closed = true;
                        break;
                    }
                    word.push(c);
                }
                if !closed {
                    return Err(());
                }
            }
            '"' => {
                let mut closed = false;
                while let Some(c) = chars.next() {
                    if c == '"' {
                        closed = true;
                        break;
                    }
                    if c == '\\' {
                        match chars.next() {
                            Some(n @ ('$' | '`' | '"' | '\\')) => word.push(n),
                            Some('\n') => {}
                            Some(other) => {
                                word.push('\\');
                                word.push(other);
                            }
                            None => return Err(()),
                        }
                    } else {
                        word.push(c);
                    }
                }
                if !closed {
                    return Err(());
                }
            }
            '\\' => match chars.next() {
                Some(n) => word.push(n),
                None => return Err(()),
            },
            c => word.push(c),
        }
    }
    if started {
        words.push(word);
    }
    Ok(words)
}

fn valid_release_id(s: &str) -> bool {
    let b = s.as_bytes();
    !b.is_empty()
        && b.len() <= 64
        && (b[0].is_ascii_lowercase() || b[0].is_ascii_digit())
        && b[1..].iter().all(|c| {
            c.is_ascii_lowercase() || c.is_ascii_digit() || *c == b'.' || *c == b'_' || *c == b'-'
        })
}

fn valid_release_version(s: &str) -> bool {
    let b = s.as_bytes();
    !b.is_empty()
        && b.len() <= 64
        && b[0].is_ascii_alphanumeric()
        && b[1..]
            .iter()
            .all(|c| c.is_ascii_alphanumeric() || *c == b'.' || *c == b'_' || *c == b'-')
}

pub fn parse_os_release(raw: &[u8]) -> Result<domain::OsRelease, String> {
    let text = std::str::from_utf8(raw).map_err(|_| "invalid OS release text".to_string())?;
    let mut values: HashMap<&str, String> = HashMap::new();
    for line in split_lines(text) {
        if line.trim_matches(is_python_space).is_empty() {
            continue;
        }
        if line.trim_start_matches(is_python_space).starts_with('#') {
            continue;
        }
        let (key, sep, value) = match line.find('=') {
            Some(i) => (&line[..i], "=", &line[i + 1..]),
            None => (line, "", ""),
        };
        if key != "ID" && key != "VERSION_ID" && key != "PRETTY_NAME" {
            continue;
        }
        if sep.is_empty() || values.contains_key(key) {
            return Err("ambiguous OS release field".to_string());
        }
        let mut words = shlex_posix(value).map_err(|_| "invalid OS release field".to_string())?;
        if words.len() != 1 || words[0].is_empty() || words[0].len() > 256 {
            return Err("invalid OS release field".to_string());
        }
        if words[0].chars().any(|c| (c as u32) < 32 || c as u32 == 127) {
            return Err("invalid OS release text".to_string());
        }
        values.insert(key, words.remove(0));
    }
    let id = values.get("ID").map(String::as_str).unwrap_or("");
    let version = values.get("VERSION_ID").map(String::as_str).unwrap_or("");
    if !valid_release_id(id) {
        return Err("missing OS identity".to_string());
    }
    if !valid_release_version(version) {
        return Err("missing OS version".to_string());
    }
    let name = values
        .get("PRETTY_NAME")
        .cloned()
        .unwrap_or_else(|| id.to_string());
    Ok(domain::OsRelease {
        id: id.to_string(),
        version: version.to_string(),
        name,
    })
}

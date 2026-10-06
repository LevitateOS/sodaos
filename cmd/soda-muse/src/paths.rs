use std::ffi::CStr;
use std::io;

pub(crate) fn go_base(path: &str) -> &str {
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

// go_clean mirrors filepath.Clean lexical rules.
pub(crate) fn go_clean(path: &str) -> String {
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

pub(crate) fn go_join(first: &str, second: &str) -> String {
    go_clean(&format!("{first}/{second}"))
}

// errno_str renders the current errno the way Go formats syscall errors.
pub(crate) fn errno_str() -> String {
    go_strerror(Some(unsafe { *libc::__errno_location() }))
}

pub(crate) fn go_strerror(errno: Option<i32>) -> String {
    let no = errno.unwrap_or(0);
    let text = unsafe {
        let ptr = libc::strerror(no);
        if ptr.is_null() {
            return format!("errno {no}");
        }
        CStr::from_ptr(ptr).to_string_lossy().into_owned()
    };
    // Go spells errno text lowercase; glibc capitalizes the first letter.
    let mut chars = text.chars();
    match chars.next() {
        Some(c) => c.to_lowercase().collect::<String>() + chars.as_str(),
        None => text,
    }
}

pub(crate) fn path_error(op: &str, path: &str, e: io::Error) -> String {
    format!("{op} {path}: {}", go_strerror(e.raw_os_error()))
}

// go_quote_rune mirrors strconv.QuoteRune for the invalid-hex-byte error.
pub(crate) fn go_quote_rune(b: u8) -> String {
    match b {
        0x20..=0x7e if b != b'\'' && b != b'\\' => format!("'{}'", b as char),
        b'\'' => String::from("'\\''"),
        b'\\' => String::from("'\\\\'"),
        0x00..=0x7f => format!("'\\x{b:02x}'"),
        _ => format!("'\\u{b:04x}'"),
    }
}

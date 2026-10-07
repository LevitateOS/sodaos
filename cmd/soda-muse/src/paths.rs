use std::ffi::CStr;
use std::io;
use std::path::{Component, Path, PathBuf};

pub(crate) fn go_base(path: &str) -> &str {
    if path.is_empty() {
        return ".";
    }
    Path::new(path)
        .file_name()
        .and_then(|name| name.to_str())
        .unwrap_or(if Path::new(path).is_absolute() {
            "/"
        } else {
            "."
        })
}

pub(crate) fn go_join(first: &str, second: &str) -> String {
    let mut joined = PathBuf::from(first);
    joined.push(second);
    joined.to_string_lossy().into_owned()
}

pub(crate) fn is_clean_absolute_path(path: &str) -> bool {
    if !Path::new(path).is_absolute() {
        return false;
    }
    let mut rebuilt = PathBuf::new();
    for component in Path::new(path).components() {
        match component {
            Component::RootDir => rebuilt.push("/"),
            Component::Normal(part) => rebuilt.push(part),
            Component::CurDir | Component::ParentDir | Component::Prefix(_) => return false,
        }
    }
    rebuilt.to_str() == Some(path)
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

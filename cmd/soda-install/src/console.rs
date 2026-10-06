//! Operator console: prompting, line/password entry with terminal echo
//! control, and the DHCP/`nmtui` network step.

use std::fs::File;

pub struct Console {
    tty_path: String,
    tty: File,
}

/// Length of one Go `unicode.IsSpace` rune opening the byte string, or
/// zero. Only exact UTF-8 sequences match; invalid bytes stop the trim,
/// exactly like Go decoding `RuneError` (never space).
fn go_space_len(prefix: &[u8]) -> usize {
    if prefix.is_empty() {
        return 0;
    }
    match prefix[0] {
        b'\t' | b'\n' | 0x0b | 0x0c | b'\r' | b' ' => return 1,
        0xc2 if prefix.len() >= 2 && (prefix[1] == 0x85 || prefix[1] == 0xa0) => return 2,
        0xe1 if prefix.len() >= 3 && prefix[1] == 0x9a && prefix[2] == 0x80 => return 3, // U+1680
        0xe2 if prefix.len() >= 3
            && prefix[1] == 0x80
            && (prefix[2] <= 0x8a
                || prefix[2] == 0xa8
                || prefix[2] == 0xa9
                || prefix[2] == 0xaf) =>
        {
            return 3; // U+2000..U+200A, U+2028, U+2029, U+202F
        }
        0xe2 if prefix.len() >= 3 && prefix[1] == 0x81 && prefix[2] == 0x9f => return 3, // U+205F
        0xe3 if prefix.len() >= 3 && prefix[1] == 0x80 && prefix[2] == 0x80 => return 3, // U+3000
        _ => {}
    }
    0
}

fn trim_space_bytes(data: &[u8]) -> &[u8] {
    let mut start = 0;
    while start < data.len() {
        let len = go_space_len(&data[start..]);
        if len == 0 {
            break;
        }
        start += len;
    }
    let mut end = data.len();
    while end > start {
        // A trailing space run ends `end`; find its opening boundary by
        // scanning the longest candidate suffix first.
        let mut len = 0;
        for candidate in [3usize, 2, 1] {
            if end >= candidate
                && start <= end - candidate
                && go_space_len(&data[end - candidate..end]) == candidate
            {
                // The candidate must align with a rune boundary: it opens
                // either at `start` or right after a non-space byte run.
                // Mid-rune splits never match `go_space_len`, except a
                // 1-byte ASCII match inside a multibyte sequence, which
                // ASCII bytes cannot start.
                len = candidate;
                break;
            }
        }
        // ASCII spaces are single bytes, so a 1-byte match always aligns;
        // multibyte matches align because continuation bytes never match a
        // space opening.
        if len == 0 {
            break;
        }
        end -= len;
    }
    &data[start..end]
}

fn errno_name(errno: libc::c_int) -> String {
    unsafe {
        let text = libc::strerror(errno);
        if text.is_null() {
            return format!("errno {errno}");
        }
        String::from_utf8_lossy(std::ffi::CStr::from_ptr(text).to_bytes()).into_owned()
    }
}

fn errno_text(err: std::io::Error) -> String {
    match err.raw_os_error() {
        Some(errno) => {
            // Go spells errno text lowercase ("input/output error").
            let text = errno_name(errno);
            let mut chars = text.chars();
            match chars.next() {
                Some(first) => first.to_lowercase().collect::<String>() + chars.as_str(),
                None => text,
            }
        }
        None => err.to_string(),
    }
}

mod network;
mod terminal;

#[cfg(test)]
pub mod test_support;

#[cfg(test)]
mod tests;

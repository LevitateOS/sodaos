// Path, ID-map and output-line helpers for preparation operations.
use crate::preparation::FACTORY_PREPARATIONS_DIR;

/// `path.IsAbs`: a leading slash.
pub(crate) fn path_is_abs(p: &str) -> bool {
    p.starts_with('/')
}

/// `path.Clean`: lexical cleanup, byte-for-byte with the Go algorithm.
pub fn path_clean(path: &str) -> String {
    let b = path.as_bytes();
    if b.is_empty() {
        return ".".to_string();
    }
    let rooted = b[0] == b'/';
    let n = b.len();
    let mut out: Vec<u8> = Vec::with_capacity(n);
    let mut r = 0usize;
    let mut dotdot = 0usize;
    if rooted {
        out.push(b'/');
        r = 1;
        dotdot = 1;
    }
    while r < n {
        if b[r] == b'/' || (b[r] == b'.' && (r + 1 == n || b[r + 1] == b'/')) {
            r += 1;
        } else if b[r] == b'.' && r + 1 < n && b[r + 1] == b'.' && (r + 2 == n || b[r + 2] == b'/')
        {
            r += 2;
            if out.len() > dotdot {
                // Back over the trailing element to its slash (Go's w-- loop
                // inspects the first excluded byte, not the last kept one).
                let mut w = out.len() - 1;
                while w > dotdot && out[w] != b'/' {
                    w -= 1;
                }
                out.truncate(w);
            } else if !rooted {
                if !out.is_empty() {
                    out.push(b'/');
                }
                out.push(b'.');
                out.push(b'.');
                dotdot = out.len();
            }
        } else {
            if (rooted && out.len() != 1) || (!rooted && !out.is_empty()) {
                out.push(b'/');
            }
            while r < n && b[r] != b'/' {
                out.push(b[r]);
                r += 1;
            }
        }
    }
    if out.is_empty() {
        return ".".to_string();
    }
    String::from_utf8(out).unwrap_or_else(|_| ".".to_string())
}

/// `path.Join`: leading empty elements skipped, the rest slash-joined
/// then cleaned; empty when every element is empty.
pub fn path_join(parts: &[&str]) -> String {
    let Some(first) = parts.iter().position(|p| !p.is_empty()) else {
        return String::new();
    };
    path_clean(&parts[first..].join("/"))
}

/// `prepareIDMap`: every mapping keeps host root out of the container and
/// some mapping shifts container root onto a usable range.
pub fn prepare_id_map(values: &[String]) -> bool {
    if values.is_empty() {
        return false;
    }
    let mut shifted = false;
    for value in values {
        let parts: Vec<&str> = value.split(':').collect();
        if parts.len() != 3 {
            return false;
        }
        let container: u32 = match parts[0].parse::<u32>() {
            Ok(v) if v.to_string() == parts[0] => v,
            _ => return false,
        };
        let base: u32 = match parts[1].parse::<u32>() {
            Ok(v) if v.to_string() == parts[1] && v > 0 => v,
            _ => return false,
        };
        let size: u32 = match parts[2].parse::<u32>() {
            Ok(v) if v.to_string() == parts[2] => v,
            _ => return false,
        };
        if container == 0 && size >= 65536 && u64::from(base) + u64::from(size) <= 4294967295 {
            shifted = true;
        }
    }
    shifted
}

/// `preparationPaths`: fixed role/checkout/snapshot/bundle/home layout.
pub fn preparation_paths(role: &str, id: &str) -> (String, String, String, String) {
    let checkout = format!("/home/{role}/checkouts/{id}");
    let snapshot = format!("{}/{id}/snapshot", FACTORY_PREPARATIONS_DIR);
    let bundle = format!("{snapshot}/source.bundle");
    let home = format!("{checkout}/.soda-home");
    (checkout, snapshot, bundle, home)
}

/// `singleLine`: one trimmed line, bounded, with no inner newline.
pub fn single_line(out: &[u8], limit: usize) -> Option<String> {
    if out.is_empty() || out.len() > limit {
        return None;
    }
    let line = std::str::from_utf8(out).ok()?.trim().to_string();
    if line.is_empty() || line.contains('\n') {
        return None;
    }
    Some(line)
}

/// `validResolvedToolPath`: absolute, clean, under a fixed tool prefix.
pub fn valid_resolved_tool_path(value: &str) -> bool {
    if value.is_empty() || value.len() > 256 || !path_is_abs(value) || path_clean(value) != value {
        return false;
    }
    value.starts_with("/usr/bin/") || value.starts_with("/usr/local/bin/")
}

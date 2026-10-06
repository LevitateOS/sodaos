use std::fs;
use std::io::{self, Read};
use std::os::unix::fs::PermissionsExt;
use std::path::Path;

// base_url mirrors config.BaseURL: an HTTP(S) origin without credentials,
// path, query or fragment. It follows Go url.Parse admission for origins:
// exact lowercase scheme, non-empty host, no userinfo, path "" or "/".
fn base_url(value: &str) -> Result<(), String> {
    if value.bytes().any(|b| b < 0x20 || b == 0x7f || b == b' ') {
        return Err(
            "must be an HTTP(S) origin without credentials, path, query or fragment".to_string(),
        );
    }
    // Go's url.Parse lowercases the scheme before matching.
    let lower = value.to_ascii_lowercase();
    let rest = if lower.starts_with("http://") {
        &value["http://".len()..]
    } else if lower.starts_with("https://") {
        &value["https://".len()..]
    } else {
        return Err(
            "must be an HTTP(S) origin without credentials, path, query or fragment".to_string(),
        );
    };
    if rest.is_empty() {
        return Err(
            "must be an HTTP(S) origin without credentials, path, query or fragment".to_string(),
        );
    }
    let authority_end = rest.find(&['/', '?', '#'][..]).unwrap_or(rest.len());
    let (authority, remainder) = rest.split_at(authority_end);
    if authority.is_empty() || authority.contains('@') {
        return Err(
            "must be an HTTP(S) origin without credentials, path, query or fragment".to_string(),
        );
    }
    if !remainder.is_empty() && remainder != "/" {
        return Err(
            "must be an HTTP(S) origin without credentials, path, query or fragment".to_string(),
        );
    }
    if !valid_percent_escapes(value) {
        return Err(
            "must be an HTTP(S) origin without credentials, path, query or fragment".to_string(),
        );
    }
    Ok(())
}

fn valid_percent_escapes(value: &str) -> bool {
    let bytes = value.as_bytes();
    let mut i = 0;
    while i < bytes.len() {
        if bytes[i] == b'%' {
            if i + 2 >= bytes.len()
                || !bytes[i + 1].is_ascii_hexdigit()
                || !bytes[i + 2].is_ascii_hexdigit()
            {
                return false;
            }
            i += 3;
        } else {
            i += 1;
        }
    }
    true
}

// credential mirrors config.Secret: a regular file inaccessible to others,
// at most 64 KiB, with non-empty trimmed content free of CR/LF/NUL.
pub(crate) fn credential(path: &str) -> Result<String, String> {
    // os.Open follows symlinks and f.Stat describes the target; match that
    // with an open handle plus following metadata.
    let file =
        fs::File::open(path).map_err(|_| "cannot open configured credential file".to_string())?;
    let st = file
        .metadata()
        .map_err(|_| "credential must be a regular file inaccessible to other users".to_string())?;
    if !st.file_type().is_file() || st.permissions().mode() & 0o007 != 0 {
        return Err("credential must be a regular file inaccessible to other users".to_string());
    }
    let mut data: Vec<u8> = Vec::new();
    if file.take(65537).read_to_end(&mut data).is_err() || data.len() > 65536 {
        return Err("cannot read credential file".to_string());
    }
    let value = String::from_utf8_lossy(&data);
    let trimmed = value.trim().to_string();
    if trimmed.is_empty() || trimmed.contains(['\r', '\n', '\0']) {
        return Err("credential is empty or malformed".to_string());
    }
    Ok(trimmed)
}

pub(crate) fn admit_setup_paths(external: &str, internal: &str, out: &Path) -> Result<(), String> {
    base_url(external)?;
    base_url(internal)?;
    if !external.starts_with("https://") {
        return Err("browser origins must use HTTPS".to_string());
    }
    if !out.is_absolute() {
        return Err("out must be absolute".to_string());
    }
    match fs::symlink_metadata(out) {
        Err(e) if e.kind() == io::ErrorKind::NotFound => {}
        _ => {
            return Err(
                "configuration already exists or cannot be inspected; refusing overwrite"
                    .to_string(),
            );
        }
    }
    Ok(())
}

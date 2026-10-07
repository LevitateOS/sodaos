use std::fs;
use std::io::{self, Read};
use std::os::unix::fs::PermissionsExt;
use std::path::Path;
use url::Url;

// base_url admits the setup origin while retaining the entered literal.
fn base_url(value: &str) -> Result<(), String> {
    let reject =
        || "must be an HTTP(S) origin without credentials, path, query or fragment".to_string();
    if value.bytes().any(|b| b <= 0x20 || b == 0x7f || b == b'\\')
        || !valid_percent_escapes(value)
        || !(value.starts_with("http://") || value.starts_with("https://"))
    {
        return Err(reject());
    }
    let rest = value.split_once("://").map(|(_, rest)| rest).unwrap_or("");
    let authority = rest.split(['/', '?', '#']).next().unwrap_or("");
    if authority.is_empty()
        || authority.contains('@')
        || authority.starts_with('[')
            && authority
                .split_once(']')
                .is_some_and(|(inside, _)| inside.contains('%'))
    {
        return Err(reject());
    }
    let after_authority = rest.strip_prefix(authority).unwrap_or("");
    let raw_path = if after_authority.starts_with('/') {
        after_authority.split(['?', '#']).next().unwrap_or("")
    } else {
        ""
    };
    if !matches!(raw_path, "" | "/") {
        return Err(reject());
    }
    let parsed = Url::parse(value).map_err(|_| reject())?;
    if !matches!(parsed.scheme(), "http" | "https")
        || parsed.host_str().is_none_or(str::is_empty)
        || !matches!(parsed.path(), "" | "/")
        || parsed.query().is_some()
        || parsed.fragment().is_some()
    {
        return Err(reject());
    }
    // Keep explicit port zero syntactic and retain it in the input literal.
    // Empty explicit ports follow Url's parser behavior and are not defaulted.
    if let Some(port) = raw_port(authority) {
        if !port.is_empty()
            && (!port.bytes().all(|b| b.is_ascii_digit())
                || port.parse::<u32>().map_or(true, |p| p > 65535))
        {
            return Err(reject());
        }
    }
    Ok(())
}

fn raw_port(authority: &str) -> Option<&str> {
    let host_port = authority
        .rsplit_once('@')
        .map_or(authority, |(_, host)| host);
    if let Some(bracketed) = host_port.strip_prefix('[') {
        let (_, tail) = bracketed.split_once(']')?;
        tail.strip_prefix(':')
    } else {
        host_port.rsplit_once(':').map(|(_, port)| port)
    }
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

//! Forgejo locale catalog merge (`scripts/forgejo-locales.py`).
//!
//! The validation parser below replicates the script's
//! `configparser.ConfigParser` configuration (`strict=True`,
//! `delimiters=('=',)`, `comment_prefixes=('#', ';')`,
//! `empty_lines_in_values=False`, case-sensitive names) as observed by
//! probing the interpreter: lines split on `\n` only, full-line comments,
//! section headers matched greedily to the last `]` with trailing text
//! ignored, option names taken up to the first `=` and right-stripped,
//! indented continuation lines while a value is open. Only section and
//! option-name sets are retained; values never matter to the merge gate.

use std::collections::HashMap;
use std::io::Read;
use std::path::Path;
use std::time::Duration;

/// Locked native catalogs come only from the upstream raw host.
pub const SOURCE_PREFIX: &str = "https://codeberg.org/forgejo/forgejo/raw/tag/";
/// Same bound as the script, on bytes: never load an unbounded catalog.
pub const MAX_NATIVE: u64 = 1024 * 1024;
/// Whole-request timeout, like the script's `urlopen(..., timeout=30)`.
pub const FETCH_TIMEOUT: Duration = Duration::from_secs(30);
/// Default additions catalog, resolved against the working directory.
pub const DEFAULT_ADDITIONS: &str = "appliance/forgejo/i18n/en-US.ini";

/// CLI failure with the script's exit-code contract: usage and lock
/// refusals (`parser.error`) exit 2, everything else exits 1.
#[derive(Debug)]
pub enum Error {
    Usage(String),
    Runtime(String),
}

impl Error {
    pub fn usage(message: impl Into<String>) -> Error {
        Error::Usage(message.into())
    }

    pub fn runtime(message: impl Into<String>) -> Error {
        Error::Runtime(message.into())
    }

    pub fn message(&self) -> &str {
        match self {
            Error::Usage(message) | Error::Runtime(message) => message,
        }
    }

    pub fn exit_code(&self) -> i32 {
        match self {
            Error::Usage(_) => 2,
            Error::Runtime(_) => 1,
        }
    }
}

/// Exactly the codepoints Python's `str.strip()` removes (probed over the
/// interpreter: 0009-000d, 001c-001f, 0020, 0085, 00a0, 1680, 2000-200a,
/// 2028, 2029, 202f, 205f, 3000). This differs from both Rust
/// `char::is_whitespace` and Unicode White_Space.
pub fn python_space(c: char) -> bool {
    matches!(
        c,
        '\u{0009}'..='\u{000d}'
            | '\u{001c}'..='\u{001f}'
            | '\u{0020}'
            | '\u{0085}'
            | '\u{00a0}'
            | '\u{1680}'
            | '\u{2000}'..='\u{200a}'
            | '\u{2028}'
            | '\u{2029}'
            | '\u{202f}'
            | '\u{205f}'
            | '\u{3000}'
    )
}

fn strip(value: &str) -> &str {
    value.trim_matches(python_space)
}

fn rstrip(value: &str) -> &str {
    value.trim_end_matches(python_space)
}

fn indent_of(line: &str) -> usize {
    line.chars().take_while(|c| python_space(*c)).count()
}

/// Parsed section and option-name sets; values are never retained.
pub struct Ini {
    sections: Vec<String>,
    options: HashMap<String, Vec<String>>,
}

impl Ini {
    fn has_section(&self, name: &str) -> bool {
        name != "DEFAULT" && self.options.contains_key(name)
    }

    fn sections(&self) -> Vec<&str> {
        self.sections
            .iter()
            .filter(|name| name.as_str() != "DEFAULT")
            .map(String::as_str)
            .collect()
    }
}

/// Parse one INI document the way the script's `configparser` does. The
/// detail text is unpinned (the script raises tracebacks); acceptance and
/// rejection match the interpreter probe for probe.
pub fn parse_ini(text: &str) -> Result<Ini, String> {
    let mut sections: Vec<String> = Vec::new();
    let mut options: HashMap<String, Vec<String>> = HashMap::new();
    let mut current: Option<String> = None;
    let mut in_value = false;
    let mut indent_level = usize::MAX;
    for (index, line) in text.split('\n').enumerate() {
        let lineno = index + 1;
        if strip(line).starts_with('#') || strip(line).starts_with(';') {
            indent_level = usize::MAX;
            continue;
        }
        let value = strip(line);
        if value.is_empty() {
            indent_level = usize::MAX;
            continue;
        }
        let indent = indent_of(line);
        if current.is_some() && in_value && indent > indent_level {
            continue;
        }
        if let Some(header) = section_header(value) {
            if options.contains_key(&header) {
                return Err(format!("line {lineno}: section '{header}' already exists"));
            }
            sections.push(header.clone());
            options.insert(header.clone(), Vec::new());
            current = Some(header);
            in_value = false;
            indent_level = usize::MAX;
            continue;
        }
        let Some(name) = current.clone() else {
            return Err(format!("line {lineno}: file contains no section headers"));
        };
        let Some((raw, _)) = value.split_once('=') else {
            return Err(format!("line {lineno}: source contains parsing errors"));
        };
        let option = rstrip(raw).to_string();
        if option.is_empty() {
            return Err(format!("line {lineno}: source contains parsing errors"));
        }
        let names = options.get_mut(&name).expect("current section parsed");
        if names.contains(&option) {
            return Err(format!(
                "line {lineno}: option '{option}' in section '{name}' already exists"
            ));
        }
        names.push(option);
        in_value = true;
        indent_level = indent;
    }
    Ok(Ini { sections, options })
}

/// Greedy section match: up to the last `]` with trailing text ignored,
/// like the interpreter's `SECTCRE.match`. Returns `None` when the line
/// cannot be a header so it falls through to option parsing.
fn section_header(value: &str) -> Option<String> {
    if !value.starts_with('[') {
        return None;
    }
    let end = value.rfind(']')?;
    if end < 2 {
        return None;
    }
    Some(value[1..end].to_string())
}

/// Merge the complete native catalog with the Soda additions, byte for
/// byte like the script: `native.rstrip() + '\n\n' + additions.rstrip()`.
pub fn merge(native: &str, additions: &str) -> Result<String, String> {
    let base = parse_ini(native).map_err(|detail| format!("invalid native catalog: {detail}"))?;
    let extra =
        parse_ini(additions).map_err(|detail| format!("invalid additions catalog: {detail}"))?;
    if !base.has_section("common") || !base.has_section("settings") {
        return Err("Expected a complete native Forgejo English catalog".to_string());
    }
    if extra.sections() != ["soda"] {
        return Err("Additions must use only the Soda namespace".to_string());
    }
    if base.has_section("soda") {
        return Err("Native catalog already owns the Soda namespace".to_string());
    }
    Ok(format!("{}\n\n{}\n", rstrip(native), rstrip(additions)))
}

/// Read at most `MAX_NATIVE + 1` bytes so callers can tell "exactly at the
/// cap" from "over the cap", like the script's `read(limit + 1)` idiom.
fn read_capped(reader: &mut dyn Read) -> Result<Vec<u8>, String> {
    let mut body = Vec::new();
    reader
        .take(MAX_NATIVE.saturating_add(1))
        .read_to_end(&mut body)
        .map_err(|e| e.to_string())?;
    Ok(body)
}

/// Load and check the lock document: pinned source plus expected bytes.
fn load_lock(path: &Path) -> Result<(String, String), Error> {
    let text = std::fs::read_to_string(path)
        .map_err(|e| Error::runtime(format!("cannot read lock {}: {e}", path.display())))?;
    let lock =
        soda_json::JsonValue::parse(&text).map_err(|_| Error::runtime("invalid lock document"))?;
    let url = lock.get("url").and_then(|v| v.as_str()).map(str::to_string);
    let sha256 = lock
        .get("sha256")
        .and_then(|v| v.as_str())
        .map(str::to_string);
    let (Some(url), Some(sha256)) = (url, sha256) else {
        return Err(Error::runtime("invalid lock document"));
    };
    if !url.starts_with(SOURCE_PREFIX) {
        return Err(Error::usage("unexpected native catalog source"));
    }
    Ok((url, sha256))
}

/// Fetch the exact locked bytes: bounded read plus pinned SHA-256.
/// Proxy environment handling matches the script's `urlopen` default.
pub fn fetch_locked(url: &str, sha256: &str) -> Result<Vec<u8>, Error> {
    let response = ureq::get(url)
        .timeout(FETCH_TIMEOUT)
        .set("User-Agent", "soda-forgejo-locales/0.1.0")
        .call();
    let response = match response {
        Ok(ok) => ok,
        Err(ureq::Error::Status(status, _)) => {
            return Err(Error::runtime(format!(
                "fetch {url}: unexpected HTTP status {status}"
            )));
        }
        Err(ureq::Error::Transport(transport)) => {
            return Err(Error::runtime(format!("fetch {url}: {transport}")));
        }
    };
    if !(200..300).contains(&response.status()) {
        return Err(Error::runtime(format!(
            "fetch {url}: unexpected HTTP status {}",
            response.status()
        )));
    }
    let data = read_capped(&mut response.into_reader()).map_err(Error::runtime)?;
    if data.len() as u64 > MAX_NATIVE || crate::locales::sha256_hex(&data) != sha256 {
        return Err(Error::usage("native catalog differs from locked bytes"));
    }
    Ok(data)
}

fn read_native_file(path: &Path) -> Result<Vec<u8>, Error> {
    let data = std::fs::read(path).map_err(|e| {
        Error::runtime(format!(
            "cannot read native catalog {}: {e}",
            path.display()
        ))
    })?;
    if data.len() as u64 > MAX_NATIVE {
        return Err(Error::usage("native catalog exceeds the 1 MiB bound"));
    }
    Ok(data)
}

/// Native input selection, mirroring the script's argparse group.
pub enum Native {
    File(std::path::PathBuf),
    Lock(std::path::PathBuf),
}

/// Run the merge: resolve the native catalog, validate both inputs, and
/// exclusively create the merged output (parents included).
pub fn run(native: &Native, additions_path: &Path, out_path: &Path) -> Result<(), Error> {
    let data = match native {
        Native::File(path) => read_native_file(path)?,
        Native::Lock(path) => {
            let (url, sha256) = load_lock(path)?;
            fetch_locked(&url, &sha256)?
        }
    };
    let native_text = String::from_utf8(data)
        .map_err(|e| Error::runtime(format!("native catalog is not UTF-8: {e}")))?;
    let additions = std::fs::read_to_string(additions_path).map_err(|e| {
        Error::runtime(format!(
            "cannot read additions {}: {e}",
            additions_path.display()
        ))
    })?;
    let output = merge(&native_text, &additions).map_err(Error::runtime)?;
    if let Some(parent) = out_path.parent() {
        if !parent.as_os_str().is_empty() {
            std::fs::create_dir_all(parent)
                .map_err(|e| Error::runtime(format!("cannot create {}: {e}", parent.display())))?;
        }
    }
    std::fs::OpenOptions::new()
        .write(true)
        .create_new(true)
        .open(out_path)
        .map_err(|e| Error::runtime(format!("cannot create {}: {e}", out_path.display())))
        .and_then(|mut file| {
            use std::io::Write;
            file.write_all(output.as_bytes())
                .map_err(|e| Error::runtime(format!("cannot write {}: {e}", out_path.display())))
        })
}

#[cfg(test)]
mod tests;

// Codex executable integrity and enrollment configuration.
use super::super::sha256;
use super::super::Error;
use serde::Deserialize;
use std::path::Path;
#[derive(Debug, Clone, Deserialize)]
pub struct Config {
    pub binary: String,
    #[serde(default)]
    pub version: String,
    pub sha256: String,
    pub root: String,
    /// Resolve the executable from the shared provider store at each
    /// session creation instead of spawning the pinned `binary`. When set,
    /// legacy `binary`/`sha256` pins are ignored for new sessions; the bound
    /// bundle digest and version come from the current store selection.
    #[serde(default)]
    pub store: bool,
}

pub(super) fn environment(root: &str) -> Vec<String> {
    filter_env(std::env::vars(), root)
}

pub(super) fn filter_env(vars: impl Iterator<Item = (String, String)>, root: &str) -> Vec<String> {
    let mut out = Vec::new();
    for (name, value) in vars {
        if name.starts_with("OPENAI")
            || name.starts_with("CODEX")
            || name.starts_with("CHATGPT")
            || name.starts_with("AWS")
            || name == "TMPDIR"
            || name == "XDG_CACHE_HOME"
        {
            continue;
        }
        out.push(format!("{name}={value}"));
    }
    out.push(format!("CODEX_HOME={root}"));
    out.push(format!("TMPDIR={root}"));
    out.push(format!("XDG_CACHE_HOME={root}/cache"));
    out
}

pub(super) fn validate_config(c: &Config) -> Result<(), Error> {
    if !Path::new(&c.root).is_absolute() {
        return Err(Error::denied("invalid Codex configuration"));
    }
    if c.store {
        // Store mode resolves the executable per session from the shared
        // selection; legacy `binary`/`sha256` pins are ignored for new
        // sessions (historical attribution only) and never shadow freshness.
        return Ok(());
    }
    if !Path::new(&c.binary).is_absolute() || c.sha256.len() != 64 {
        return Err(Error::denied("invalid Codex configuration"));
    }
    Ok(())
}

pub(super) fn check_binary(c: &Config) -> Result<(), Error> {
    if c.store {
        // The executable is resolved and admitted per session creation.
        return Ok(());
    }
    let data = std::fs::read(&c.binary)?;
    if sha256::hex(&sha256::digest(&data)) != c.sha256 {
        return Err(Error::denied("codex digest mismatch"));
    }
    Ok(())
}

impl From<serde_json::Error> for Error {
    fn from(err: serde_json::Error) -> Error {
        Error::failed(err.to_string())
    }
}

// Codex pinned configuration and admission gates, folded from identity-providers (A06.M).
use super::super::sha256;
use super::super::{run_capture, Error};
use super::{Provider, VERSION};
use serde::Deserialize;
use std::path::Path;
use std::time::Duration;
#[derive(Debug, Clone, Deserialize)]
pub struct Config {
    pub binary: String,
    pub version: String,
    pub sha256: String,
    pub root: String,
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
    if !Path::new(&c.binary).is_absolute()
        || !Path::new(&c.root).is_absolute()
        || c.version != VERSION
        || c.sha256.len() != 64
    {
        return Err(Error::denied("invalid pinned Codex configuration"));
    }
    Ok(())
}

pub(super) fn check_binary(c: &Config) -> Result<(), Error> {
    let data = std::fs::read(&c.binary)?;
    if sha256::hex(&sha256::digest(&data)) != c.sha256 {
        return Err(Error::denied("codex digest mismatch"));
    }
    Ok(())
}

pub(super) fn check_version(p: &Provider) -> Result<(), Error> {
    let out = run_capture(
        &p.config.binary,
        &["--version"],
        &environment(&p.config.root),
        None,
        Duration::from_secs(10),
    )
    .map_err(|_| Error::denied("codex version mismatch"))?;
    if out.trim() != format!("codex-cli {}", p.config.version) {
        return Err(Error::denied("codex version mismatch"));
    }
    Ok(())
}

impl From<serde_json::Error> for Error {
    fn from(err: serde_json::Error) -> Error {
        Error::failed(err.to_string())
    }
}

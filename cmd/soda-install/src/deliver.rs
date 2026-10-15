//! Strict Payload v4 admission for the live installer. Application images
//! are acquired from sealed references by the installed host at startup.

use crate::buildx;
use crate::errors::Error;
use serde::Deserialize;

pub use soda_build_tools::payload::PayloadV4 as Payload;

pub const PATH: &str = "/usr/share/soda/release.json";

pub fn load(path: &str) -> Result<Payload, Error> {
    let data = buildx::read_json_bytes(path)?;
    let text = std::str::from_utf8(&data).map_err(|_| Error::msg("invalid payload document"))?;
    let mut deserializer = serde_json::Deserializer::from_str(text);
    let payload = Payload::deserialize(&mut deserializer)
        .map_err(|_| Error::msg("invalid payload document"))?;
    deserializer
        .end()
        .map_err(|_| Error::msg("invalid payload document"))?;
    payload.validate().map_err(|e| Error::msg(e.to_string()))?;
    Ok(payload)
}

#[cfg(test)]
mod tests;

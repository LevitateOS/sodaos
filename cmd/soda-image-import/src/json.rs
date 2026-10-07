use serde::de::DeserializeOwned;
use serde_json::value::RawValue;

pub(super) fn parse_json<T: DeserializeOwned>(data: &[u8]) -> Result<T, String> {
    let text = std::str::from_utf8(data).map_err(|_| "invalid UTF-8 in JSON".to_string())?;
    let mut de = serde_json::Deserializer::from_str(text);
    let value = T::deserialize(&mut de).map_err(|_| "invalid JSON".to_string())?;
    de.end().map_err(|_| "invalid JSON".to_string())?;
    Ok(value)
}

pub(super) fn json_valid(data: &[u8]) -> bool {
    parse_json::<serde::de::IgnoredAny>(data).is_ok()
}

pub(super) fn raw_string(raw: Option<&RawValue>, field: &str) -> Result<String, String> {
    match raw {
        None => Ok(String::new()),
        Some(raw) => serde_json::from_str::<String>(raw.get())
            .map_err(|_| format!("field {field} must be a string")),
    }
}

pub(super) fn raw_int(raw: Option<&RawValue>, field: &str) -> Result<i64, String> {
    let Some(raw) = raw else { return Ok(0) };
    let text = raw.get();
    if text.is_empty() || text.bytes().any(|b| matches!(b, b'.' | b'e' | b'E' | b'+')) {
        return Err(format!("field {field} must be an integer"));
    }
    text.parse::<i128>()
        .ok()
        .and_then(|n| i64::try_from(n).ok())
        .ok_or_else(|| format!("field {field} must be an integer"))
}

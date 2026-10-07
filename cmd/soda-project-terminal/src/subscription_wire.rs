use crate::state_json::StateValue;

use crate::broker::{HORIZON_SECS, REQUEST_LIMIT};
use crate::pyemit;

// ---------------------------------------------------------------------------
// Pure JSON helpers.
// ---------------------------------------------------------------------------

/// Last value wins for duplicate keys, like `.py` `json.loads` dicts.
fn last<'a>(entries: &'a [(String, StateValue)], key: &str) -> Option<&'a StateValue> {
    entries.iter().rev().find(|(k, _)| k == key).map(|(_, v)| v)
}

fn number_f64(raw: &str) -> Option<f64> {
    raw.parse::<f64>().ok().filter(|n| n.is_finite())
}

/// Numeric equivalence for JSON numbers: `i128` compare when both are plain
/// ints, else finite-float compare (so `100` equals `1e2`, like `.py`).
fn number_equal(a: &str, b: &str) -> bool {
    let left = StateValue::Number(a.to_string());
    let right = StateValue::Number(b.to_string());
    match (left.as_integer(), right.as_integer()) {
        (Some(x), Some(y)) => x == y,
        _ => match (number_f64(a), number_f64(b)) {
            (Some(x), Some(y)) => x == y,
            _ => a == b,
        },
    }
}

/// Order-insensitive deep equality mirroring `.py` `dict ==`:
/// objects compare by key set with last-wins values, arrays pairwise,
/// numbers numerically, `True == 1` / `False == 0`.
pub fn json_equal(a: &StateValue, b: &StateValue) -> bool {
    match (a, b) {
        (StateValue::Null, StateValue::Null) => true,
        (StateValue::Bool(x), StateValue::Bool(y)) => x == y,
        (StateValue::Bool(x), StateValue::Number(raw))
        | (StateValue::Number(raw), StateValue::Bool(x)) => {
            number_f64(raw) == Some(f64::from(u8::from(*x)))
        }
        (StateValue::Number(x), StateValue::Number(y)) => number_equal(x, y),
        (StateValue::Str(x), StateValue::Str(y)) => x == y,
        (StateValue::Array(x), StateValue::Array(y)) => {
            x.len() == y.len() && x.iter().zip(y.iter()).all(|(u, v)| json_equal(u, v))
        }
        (StateValue::Object(x), StateValue::Object(y)) => {
            let mut xkeys: Vec<&str> = x.iter().map(|(k, _)| k.as_str()).collect();
            let mut ykeys: Vec<&str> = y.iter().map(|(k, _)| k.as_str()).collect();
            xkeys.sort_unstable();
            xkeys.dedup();
            ykeys.sort_unstable();
            ykeys.dedup();
            if xkeys != ykeys {
                return false;
            }
            xkeys.iter().all(|k| match (last(x, k), last(y, k)) {
                (Some(u), Some(v)) => json_equal(u, v),
                _ => false,
            })
        }
        _ => false,
    }
}

/// The `.py` `int()` conversions for lease `actor_id`: JSON ints, floats
/// truncated toward zero, and (trimmed, optionally signed) plain numeric
/// strings.
pub fn json_int(value: &StateValue) -> Option<i64> {
    match value {
        StateValue::Number(raw) => {
            if let Some(n) = pyemit::as_int(value) {
                return Some(n);
            }
            // `.py` `int()`: floats truncate toward zero.
            let n = number_f64(raw)?.trunc();
            const LO: f64 = -9_223_372_036_854_775_808.0;
            const HI: f64 = 9_223_372_036_854_775_808.0;
            if n < LO || n >= HI {
                return None;
            }
            #[allow(clippy::cast_possible_truncation)]
            Some(n as i64)
        }
        // `.py` `int(True) == 1`, `int(False) == 0`.
        StateValue::Bool(b) => Some(i64::from(*b)),
        StateValue::Str(text) => {
            let trimmed = text.trim();
            let (negative, digits) = match trimmed.strip_prefix('-') {
                Some(rest) => (true, rest),
                None => (false, trimmed.strip_prefix('+').unwrap_or(trimmed)),
            };
            if digits.is_empty() || !digits.bytes().all(|b| b.is_ascii_digit()) {
                return None;
            }
            let n: i64 = digits.parse().ok()?;
            Some(if negative { -n } else { n })
        }
        _ => None,
    }
}

/// `lease['binding'] = native`: in-place replace, else append (dict order
/// preserved like the `.py` mutation).
pub fn lease_with_binding(lease: &StateValue, native: StateValue) -> Option<StateValue> {
    match lease {
        StateValue::Object(entries) => {
            let mut out = entries.clone();
            if let Some(slot) = out.iter_mut().find(|(k, _)| k == "binding") {
                slot.1 = native;
            } else {
                out.push(("binding".to_string(), native));
            }
            Some(StateValue::Object(out))
        }
        _ => None,
    }
}

/// `0 < deadline - now <= 12h`.
pub fn deadline_ok(deadline: i64, now: i64) -> bool {
    let rest = deadline.saturating_sub(now);
    rest > 0 && rest <= HORIZON_SECS
}

/// `{'lease': lease, 'credential': ...}` in `.py` key order.
pub fn result_object(lease: &StateValue, credential: &str) -> StateValue {
    StateValue::Object(vec![
        ("lease".to_string(), lease.clone()),
        (
            "credential".to_string(),
            StateValue::Str(credential.to_string()),
        ),
    ])
}

/// Empty-lease lookup result: `{'lease': {}, 'credential': ''}`.
pub fn empty_result() -> StateValue {
    result_object(&StateValue::Object(Vec::new()), "")
}

/// Pure request decode: size cap plus JSON parse (last-wins, no shape or
/// duplicate checks — `subscription_main` runs plain `json.loads`).
pub fn decode_request(body: &[u8]) -> Result<StateValue, String> {
    if body.len() > REQUEST_LIMIT {
        return Err("request size".to_string());
    }
    let text = std::str::from_utf8(body).map_err(|_| "request encoding".to_string())?;
    StateValue::parse(text).map_err(|_| "request json".to_string())
}

/// Native binding record in `.py` key order.
pub fn native_binding(
    identifier: &str,
    project: &StateValue,
    login: &str,
    generation: &StateValue,
) -> StateValue {
    StateValue::Object(vec![
        ("kind".to_string(), StateValue::Str("terminal".to_string())),
        ("id".to_string(), StateValue::Str(identifier.to_string())),
        ("project".to_string(), project.clone()),
        ("login".to_string(), StateValue::Str(login.to_string())),
        ("generation".to_string(), generation.clone()),
    ])
}

/// Stored subscription profile in `.py` key order.
pub fn profile_object(
    lease: &StateValue,
    binding: &StateValue,
    deadline: i64,
    scope: &str,
) -> StateValue {
    StateValue::Object(vec![
        ("lease".to_string(), lease.clone()),
        ("binding".to_string(), binding.clone()),
        (
            "deadline".to_string(),
            StateValue::Number(deadline.to_string()),
        ),
        ("scope".to_string(), StateValue::Str(scope.to_string())),
    ])
}

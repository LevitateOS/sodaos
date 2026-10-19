//! Plain `json.dumps(value)` (default `(', ', ': ')` separators,
//! `ensure_ascii=True`): every response line and every state file the helper
//! writes is emitted exactly so. Plus Python `==` over parsed values for the
//! idempotency comparisons (`saved != request`).

use crate::state_json::StateValue;

/// Build an object preserving insertion order (response key order matters).
pub fn obj(pairs: Vec<(&str, StateValue)>) -> StateValue {
    StateValue::Object(pairs.into_iter().map(|(k, v)| (k.to_string(), v)).collect())
}

pub fn str_value(text: &str) -> StateValue {
    StateValue::Str(text.to_string())
}

/// Emit one compact document with default separators, byte-identical to
/// CPython `json.dumps`.
pub fn dumps_default(value: &StateValue) -> String {
    crate::pyemit::dumps_default_serde(value)
}

/// Python `==` over two parsed values: objects compare order-insensitive
/// with last-wins duplicates, and numbers compare numerically (`1 == 1.0`,
/// `True == 1`, unbounded ints exact).
pub fn json_equal(left: &StateValue, right: &StateValue) -> bool {
    match (left, right) {
        (StateValue::Null, StateValue::Null) => true,
        (StateValue::Bool(a), StateValue::Bool(b)) => a == b,
        (StateValue::Str(a), StateValue::Str(b)) => a == b,
        (StateValue::Array(a), StateValue::Array(b)) => {
            a.len() == b.len() && a.iter().zip(b.iter()).all(|(x, y)| json_equal(x, y))
        }
        (StateValue::Object(a), StateValue::Object(b)) => {
            let keys_a = unique_keys(a);
            let keys_b = unique_keys(b);
            if keys_a != keys_b {
                return false;
            }
            keys_a.iter().all(|key| {
                let Some(x) = left.get(key) else {
                    return false;
                };
                let Some(y) = right.get(key) else {
                    return false;
                };
                json_equal(x, y)
            })
        }
        (StateValue::Number(a), StateValue::Number(b)) => numbers_equal(a, b),
        (StateValue::Bool(a), StateValue::Number(b)) => {
            numbers_equal(if *a { "1" } else { "0" }, b)
        }
        (StateValue::Number(a), StateValue::Bool(b)) => {
            numbers_equal(a, if *b { "1" } else { "0" })
        }
        _ => false,
    }
}

fn unique_keys(entries: &[(String, StateValue)]) -> Vec<String> {
    let mut keys: Vec<String> = Vec::new();
    for (key, _) in entries {
        if !keys.contains(key) {
            keys.push(key.clone());
        }
    }
    keys.sort();
    keys
}

fn is_int_literal(raw: &str) -> bool {
    let digits = raw.strip_prefix('-').unwrap_or(raw);
    !digits.is_empty() && digits.bytes().all(|b| b.is_ascii_digit())
}

/// Normalize an integer literal for exact comparison (`-0` is `0`;
/// leading zeros cannot occur: the parser rejects them).
fn norm_int(raw: &str) -> (bool, &str) {
    let raw = if raw == "-0" { "0" } else { raw };
    match raw.strip_prefix('-') {
        Some(digits) => (true, digits),
        None => (false, raw),
    }
}

fn numbers_equal(left: &str, right: &str) -> bool {
    if is_int_literal(left) && is_int_literal(right) {
        let (neg_a, a) = norm_int(left);
        let (neg_b, b) = norm_int(right);
        return neg_a == neg_b && a.len() == b.len() && a == b;
    }
    // At least one side is a float literal: compare as f64, like CPython
    // for values in float range. (Out-of-range literals such as `1e400`
    // parse to infinity on both sides here; CPython can distinguish them,
    // but the protocol only carries small ints and strings.)
    match (left.parse::<f64>(), right.parse::<f64>()) {
        (Ok(a), Ok(b)) => a == b,
        _ => left == right,
    }
}

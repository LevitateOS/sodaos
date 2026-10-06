use crate::json::{self, Value};

use super::unavailable;

/// Single-rune lowercasing like Go's `unicode.ToLower`: the first rune of the
/// full mapping, never an expansion.
pub(crate) fn lower_char(c: char) -> char {
    c.to_lowercase().next().unwrap_or(c)
}

/// One rune's field-name fold representative. The known-field side is always
/// an ASCII name, so only ASCII fold relations matter: ASCII case plus the
/// two non-ASCII runes Go links to ASCII letters, `ſ` (U+017F, with S/s) and
/// `K` (U+212A Kelvin, with K/k). Notably `İ` (U+0130) does NOT fold with
/// `i` in Go. All other non-ASCII runes compare by identity.
fn fold_char(c: char) -> char {
    match c {
        'ſ' => 's',
        '\u{212a}' => 'k',
        x if x.is_ascii_alphabetic() => x.to_ascii_lowercase(),
        x => x,
    }
}

/// Field-name case folding matching `strings.EqualFold` whenever one side is
/// an ASCII field name (the only use: JSON keys against known ASCII names).
pub(crate) fn fold_eq(a: &str, b: &str) -> bool {
    if a == b {
        return true;
    }
    debug_assert!(a.is_ascii() || b.is_ascii(), "fold needs an ASCII side");
    let mut ai = a.chars();
    let mut bi = b.chars();
    loop {
        match (ai.next(), bi.next()) {
            (None, None) => return true,
            (Some(x), Some(y)) => {
                if x != y && fold_char(x) != fold_char(y) {
                    return false;
                }
            }
            (None, Some(_)) | (Some(_), None) => return false,
        }
    }
}

/// Mirror of `nativeObject`: strict single-object decode with duplicate
/// rejection at every level (Go's `strictjson.Decode`), exact-name required
/// field presence with the `HaveNodeKey` omission rule, then `encoding/json`
/// binding performed by the caller's typed decoder. Returns the document-order
/// value so case-variant keys bind last-wins exactly like Go.
pub(crate) fn native_object(data: &[u8], required: &[&str]) -> Result<Value, String> {
    if json::decode_strict(data).is_err() {
        return unavailable();
    }
    // A strict success implies tolerant success (same grammar, fewer bans).
    let v = match json::decode_tolerant(data) {
        Ok(v) => v,
        Err(_) => return unavailable(),
    };
    let fields = match v.as_object() {
        Some(f) => f,
        None => return unavailable(),
    };
    for key in required {
        if !required_native_field(fields, key) {
            return unavailable();
        }
    }
    Ok(v)
}

/// Mirror of `requiredNativeField` + `haveNodeKeyOmitted`.
fn required_native_field(fields: &[(String, Value)], key: &str) -> bool {
    match fields.iter().find(|(k, _)| k == key) {
        None => {
            if key != "HaveNodeKey" {
                return false;
            }
            !fields.iter().any(|(k, _)| fold_eq(k, key))
        }
        Some((_, v)) => !v.is_null() || key == "AdvertiseRoutes",
    }
}

/// All values bound to one struct field: exact or fold-equal keys in document
/// order. `encoding/json` applies each in order: the last non-null value wins,
/// `null` is a no-op, and any mistyped value fails the whole decode.
fn bound_values<'a>(fields: &'a [(String, Value)], name: &str) -> Vec<&'a Value> {
    fields
        .iter()
        .filter(|(k, _)| fold_eq(k, name))
        .map(|(_, v)| v)
        .collect()
}

pub(crate) fn bind_string_into(
    fields: &[(String, Value)],
    name: &str,
    out: &mut String,
) -> Result<(), String> {
    for v in bound_values(fields, name) {
        match v {
            Value::Null => {}
            Value::Str(s) => *out = s.clone(),
            _ => return unavailable(),
        }
    }
    Ok(())
}

pub(crate) fn bind_bool_into(
    fields: &[(String, Value)],
    name: &str,
    out: &mut bool,
) -> Result<(), String> {
    for v in bound_values(fields, name) {
        match v {
            Value::Null => {}
            Value::Bool(b) => *out = *b,
            _ => return unavailable(),
        }
    }
    Ok(())
}

fn bind_list_into(
    fields: &[(String, Value)],
    name: &str,
    out: &mut Vec<String>,
) -> Result<(), String> {
    for v in bound_values(fields, name) {
        match v {
            Value::Null => {}
            Value::Array(items) => {
                let mut next = Vec::with_capacity(items.len());
                for item in items {
                    match item {
                        Value::Null => next.push(String::new()),
                        Value::Str(s) => next.push(s.clone()),
                        _ => return unavailable(),
                    }
                }
                *out = next;
            }
            _ => return unavailable(),
        }
    }
    Ok(())
}

#[derive(Debug, Clone, Default)]
pub(crate) struct SelfPeer {
    pub(crate) id: String,
    pub(crate) dns_name: String,
    pub(crate) ips: Vec<String>,
    pub(crate) tags: Vec<String>,
    pub(crate) online: bool,
    pub(crate) expired: bool,
}

/// Pointer-struct binding merges across case-variant keys into one struct.
fn bind_self(fields: &[(String, Value)]) -> Result<Option<SelfPeer>, String> {
    let mut out: Option<SelfPeer> = None;
    for v in bound_values(fields, "Self") {
        match v {
            Value::Null => {}
            Value::Object(inner) => {
                let p = out.get_or_insert_with(SelfPeer::default);
                bind_string_into(inner, "ID", &mut p.id)?;
                bind_string_into(inner, "DNSName", &mut p.dns_name)?;
                bind_list_into(inner, "TailscaleIPs", &mut p.ips)?;
                bind_list_into(inner, "Tags", &mut p.tags)?;
                bind_bool_into(inner, "Online", &mut p.online)?;
                bind_bool_into(inner, "Expired", &mut p.expired)?;
            }
            _ => return unavailable(),
        }
    }
    Ok(out)
}

fn bind_tailnet(fields: &[(String, Value)]) -> Result<Option<String>, String> {
    let mut out: Option<String> = None;
    for v in bound_values(fields, "CurrentTailnet") {
        match v {
            Value::Null => {}
            Value::Object(inner) => {
                let slot = out.get_or_insert_with(String::new);
                bind_string_into(inner, "Name", slot)?;
            }
            _ => return unavailable(),
        }
    }
    Ok(out)
}

#[derive(Debug, Clone, Default)]
pub(crate) struct NativeStatus {
    pub(crate) backend_state: String,
    pub(crate) have_node_key: bool,
    pub(crate) tailnet: Option<String>,
    pub(crate) peer: Option<SelfPeer>,
}

pub(crate) fn decode_native_status(v: &Value) -> Result<NativeStatus, String> {
    let fields = match v.as_object() {
        Some(f) => f,
        None => return unavailable(),
    };
    let mut s = NativeStatus::default();
    bind_string_into(fields, "BackendState", &mut s.backend_state)?;
    bind_bool_into(fields, "HaveNodeKey", &mut s.have_node_key)?;
    s.tailnet = bind_tailnet(fields)?;
    s.peer = bind_self(fields)?;
    Ok(s)
}

#[derive(Debug, Clone, Default)]
pub(crate) struct NativePrefs {
    pub(crate) want_running: bool,
    pub(crate) corp_dns: bool,
    pub(crate) route_all: bool,
    pub(crate) run_ssh: bool,
    pub(crate) exit_node_id: String,
    pub(crate) exit_node_ip: String,
    pub(crate) advertise_routes: Vec<String>,
}

pub(crate) fn decode_native_prefs(v: &Value) -> Result<NativePrefs, String> {
    let fields = match v.as_object() {
        Some(f) => f,
        None => return unavailable(),
    };
    let mut p = NativePrefs::default();
    bind_bool_into(fields, "WantRunning", &mut p.want_running)?;
    bind_bool_into(fields, "CorpDNS", &mut p.corp_dns)?;
    bind_bool_into(fields, "RouteAll", &mut p.route_all)?;
    bind_bool_into(fields, "RunSSH", &mut p.run_ssh)?;
    bind_string_into(fields, "ExitNodeID", &mut p.exit_node_id)?;
    bind_string_into(fields, "ExitNodeIP", &mut p.exit_node_ip)?;
    bind_list_into(fields, "AdvertiseRoutes", &mut p.advertise_routes)?;
    Ok(p)
}

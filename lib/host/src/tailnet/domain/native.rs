use crate::json;
use serde::de::{self, MapAccess, SeqAccess, Visitor};
use serde::{Deserialize, Deserializer};

/// One-rune lowercasing like Go's `unicode.ToLower`.
pub(crate) fn lower_char(c: char) -> char {
    c.to_lowercase().next().unwrap_or(c)
}

fn fold_char(c: char) -> char {
    match c {
        'ſ' => 's',
        '\u{212a}' => 'k',
        x if x.is_ascii_alphabetic() => x.to_ascii_lowercase(),
        x => x,
    }
}

pub(crate) fn fold_eq(a: &str, b: &str) -> bool {
    if a == b {
        return true;
    }
    let mut ai = a.chars();
    let mut bi = b.chars();
    loop {
        match (ai.next(), bi.next()) {
            (None, None) => return true,
            (Some(x), Some(y)) if x == y || fold_char(x) == fold_char(y) => {}
            (Some(_), Some(_)) | (None, Some(_)) | (Some(_), None) => return false,
        }
    }
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

#[derive(Default)]
struct StringList(Vec<String>);
impl<'de> Deserialize<'de> for StringList {
    fn deserialize<D>(d: D) -> Result<Self, D::Error>
    where
        D: Deserializer<'de>,
    {
        struct V;
        impl<'de> Visitor<'de> for V {
            type Value = StringList;
            fn expecting(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
                f.write_str("string list")
            }
            fn visit_unit<E>(self) -> Result<Self::Value, E>
            where
                E: de::Error,
            {
                Ok(StringList::default())
            }
            fn visit_seq<A>(self, mut seq: A) -> Result<Self::Value, A::Error>
            where
                A: SeqAccess<'de>,
            {
                let mut out = Vec::new();
                while let Some(v) = seq.next_element::<Option<String>>()? {
                    out.push(v.unwrap_or_default());
                }
                Ok(StringList(out))
            }
        }
        d.deserialize_any(V)
    }
}

#[derive(Default)]
struct PeerWire {
    id: Option<String>,
    dns_name: Option<String>,
    ips: Option<StringList>,
    tags: Option<StringList>,
    online: Option<bool>,
    expired: Option<bool>,
}
impl<'de> Deserialize<'de> for PeerWire {
    fn deserialize<D>(d: D) -> Result<Self, D::Error>
    where
        D: Deserializer<'de>,
    {
        struct V;
        impl<'de> Visitor<'de> for V {
            type Value = PeerWire;
            fn expecting(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
                f.write_str("native peer")
            }
            fn visit_map<A>(self, mut map: A) -> Result<Self::Value, A::Error>
            where
                A: MapAccess<'de>,
            {
                let mut out = PeerWire::default();
                while let Some(k) = map.next_key::<String>()? {
                    if fold_eq(&k, "ID") {
                        if let Some(v) = map.next_value::<Option<String>>()? {
                            out.id = Some(v);
                        }
                    } else if fold_eq(&k, "DNSName") {
                        if let Some(v) = map.next_value::<Option<String>>()? {
                            out.dns_name = Some(v);
                        }
                    } else if fold_eq(&k, "TailscaleIPs") {
                        if let Some(v) = map.next_value::<Option<StringList>>()? {
                            out.ips = Some(v);
                        }
                    } else if fold_eq(&k, "Tags") {
                        if let Some(v) = map.next_value::<Option<StringList>>()? {
                            out.tags = Some(v);
                        }
                    } else if fold_eq(&k, "Online") {
                        if let Some(v) = map.next_value::<Option<bool>>()? {
                            out.online = Some(v);
                        }
                    } else if fold_eq(&k, "Expired") {
                        if let Some(v) = map.next_value::<Option<bool>>()? {
                            out.expired = Some(v);
                        }
                    } else {
                        map.next_value::<de::IgnoredAny>()?;
                    }
                }
                Ok(out)
            }
        }
        d.deserialize_map(V)
    }
}

fn merge_peer(into: &mut SelfPeer, from: PeerWire) {
    if let Some(v) = from.id {
        into.id = v;
    }
    if let Some(v) = from.dns_name {
        into.dns_name = v;
    }
    if let Some(v) = from.ips {
        into.ips = v.0;
    }
    if let Some(v) = from.tags {
        into.tags = v.0;
    }
    if let Some(v) = from.online {
        into.online = v;
    }
    if let Some(v) = from.expired {
        into.expired = v;
    }
}

#[derive(Default)]
pub(crate) struct NativeStatus {
    pub(crate) backend_state: String,
    pub(crate) have_node_key: bool,
    pub(crate) tailnet: Option<String>,
    pub(crate) peer: Option<SelfPeer>,
}

#[derive(Default)]
struct StatusWire {
    value: NativeStatus,
    required_state: bool,
    key_seen: bool,
    exact_key_seen: bool,
    exact_key_null: bool,
    key_value_seen: bool,
}
impl<'de> Deserialize<'de> for StatusWire {
    fn deserialize<D>(d: D) -> Result<Self, D::Error>
    where
        D: Deserializer<'de>,
    {
        struct V;
        impl<'de> Visitor<'de> for V {
            type Value = StatusWire;
            fn expecting(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
                f.write_str("native status")
            }
            fn visit_map<A>(self, mut map: A) -> Result<Self::Value, A::Error>
            where
                A: MapAccess<'de>,
            {
                let mut out = StatusWire::default();
                while let Some(k) = map.next_key::<String>()? {
                    if fold_eq(&k, "BackendState") {
                        let value = map.next_value::<Option<String>>()?;
                        if k == "BackendState" {
                            out.required_state = value.is_some();
                        }
                        if let Some(v) = value {
                            out.value.backend_state = v;
                        }
                    } else if fold_eq(&k, "HaveNodeKey") {
                        let value = map.next_value::<Option<bool>>()?;
                        out.key_seen = true;
                        out.exact_key_seen |= k == "HaveNodeKey";
                        out.exact_key_null |= k == "HaveNodeKey" && value.is_none();
                        out.key_value_seen |= value.is_some();
                        if let Some(v) = value {
                            out.value.have_node_key = v;
                        }
                    } else if fold_eq(&k, "CurrentTailnet") {
                        let value = map.next_value::<Option<TailnetWire>>()?;
                        if let Some(v) = value {
                            if let Some(name) = v.name {
                                out.value.tailnet = Some(name);
                            } else {
                                out.value.tailnet.get_or_insert_with(String::new);
                            }
                        }
                    } else if fold_eq(&k, "Self") {
                        let value = map.next_value::<Option<PeerWire>>()?;
                        if let Some(v) = value {
                            let peer = out.value.peer.get_or_insert_with(SelfPeer::default);
                            merge_peer(peer, v);
                        }
                    } else {
                        map.next_value::<de::IgnoredAny>()?;
                    }
                }
                if !out.required_state
                    || (out.key_seen
                        && (!out.exact_key_seen || out.exact_key_null || !out.key_value_seen))
                {
                    return Err(de::Error::custom("missing required native status field"));
                }
                Ok(out)
            }
        }
        d.deserialize_map(V)
    }
}

#[derive(Default)]
struct TailnetWire {
    name: Option<String>,
}
impl<'de> Deserialize<'de> for TailnetWire {
    fn deserialize<D>(d: D) -> Result<Self, D::Error>
    where
        D: Deserializer<'de>,
    {
        struct V;
        impl<'de> Visitor<'de> for V {
            type Value = TailnetWire;
            fn expecting(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
                f.write_str("current tailnet")
            }
            fn visit_map<A>(self, mut map: A) -> Result<Self::Value, A::Error>
            where
                A: MapAccess<'de>,
            {
                let mut out = TailnetWire::default();
                while let Some(k) = map.next_key::<String>()? {
                    if fold_eq(&k, "Name") {
                        if let Some(v) = map.next_value::<Option<String>>()? {
                            out.name = Some(v);
                        }
                    } else {
                        map.next_value::<de::IgnoredAny>()?;
                    }
                }
                Ok(out)
            }
        }
        d.deserialize_map(V)
    }
}

pub(crate) fn decode_native_status(data: &[u8]) -> Result<NativeStatus, String> {
    let wire: StatusWire =
        json::decode_strict_as(data).map_err(|_| super::ERR_UNAVAILABLE.to_string())?;
    Ok(wire.value)
}

#[derive(Default)]
pub(crate) struct NativePrefs {
    pub(crate) want_running: bool,
    pub(crate) corp_dns: bool,
    pub(crate) route_all: bool,
    pub(crate) run_ssh: bool,
    pub(crate) exit_node_id: String,
    pub(crate) exit_node_ip: String,
    pub(crate) advertise_routes: Vec<String>,
}

#[derive(Default)]
struct PrefsWire {
    value: NativePrefs,
    required: [bool; 7],
}
impl<'de> Deserialize<'de> for PrefsWire {
    fn deserialize<D>(d: D) -> Result<Self, D::Error>
    where
        D: Deserializer<'de>,
    {
        struct V;
        impl<'de> Visitor<'de> for V {
            type Value = PrefsWire;
            fn expecting(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
                f.write_str("native preferences")
            }
            fn visit_map<A>(self, mut map: A) -> Result<Self::Value, A::Error>
            where
                A: MapAccess<'de>,
            {
                let mut out = PrefsWire::default();
                while let Some(k) = map.next_key::<String>()? {
                    let (index, exact) = [
                        "WantRunning",
                        "CorpDNS",
                        "RouteAll",
                        "RunSSH",
                        "ExitNodeID",
                        "ExitNodeIP",
                        "AdvertiseRoutes",
                    ]
                    .iter()
                    .enumerate()
                    .find(|(_, name)| fold_eq(&k, name))
                    .map(|(i, n)| (i, k == *n))
                    .unwrap_or((usize::MAX, false));
                    if index == usize::MAX {
                        map.next_value::<de::IgnoredAny>()?;
                        continue;
                    }
                    match index {
                        0 => {
                            let v = map.next_value::<Option<bool>>()?;
                            if exact {
                                out.required[0] = v.is_some();
                            }
                            if let Some(v) = v {
                                out.value.want_running = v;
                            }
                        }
                        1 => {
                            let v = map.next_value::<Option<bool>>()?;
                            if exact {
                                out.required[1] = v.is_some();
                            }
                            if let Some(v) = v {
                                out.value.corp_dns = v;
                            }
                        }
                        2 => {
                            let v = map.next_value::<Option<bool>>()?;
                            if exact {
                                out.required[2] = v.is_some();
                            }
                            if let Some(v) = v {
                                out.value.route_all = v;
                            }
                        }
                        3 => {
                            let v = map.next_value::<Option<bool>>()?;
                            if exact {
                                out.required[3] = v.is_some();
                            }
                            if let Some(v) = v {
                                out.value.run_ssh = v;
                            }
                        }
                        4 => {
                            let v = map.next_value::<Option<String>>()?;
                            if exact {
                                out.required[4] = v.is_some();
                            }
                            if let Some(v) = v {
                                out.value.exit_node_id = v;
                            }
                        }
                        5 => {
                            let v = map.next_value::<Option<String>>()?;
                            if exact {
                                out.required[5] = v.is_some();
                            }
                            if let Some(v) = v {
                                out.value.exit_node_ip = v;
                            }
                        }
                        _ => {
                            let v = map.next_value::<Option<StringList>>()?;
                            if exact {
                                out.required[6] = true;
                            }
                            if let Some(v) = v {
                                out.value.advertise_routes = v.0;
                            }
                        }
                    }
                }
                if !out.required.iter().all(|v| *v) {
                    return Err(de::Error::custom(
                        "missing required exact native preference field",
                    ));
                }
                Ok(out)
            }
        }
        d.deserialize_map(V)
    }
}

pub(crate) fn decode_native_prefs(data: &[u8]) -> Result<NativePrefs, String> {
    let wire: PrefsWire =
        json::decode_strict_as(data).map_err(|_| super::ERR_UNAVAILABLE.to_string())?;
    Ok(wire.value)
}

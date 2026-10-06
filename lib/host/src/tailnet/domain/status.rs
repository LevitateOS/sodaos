use super::{
    bind_bool_into, bind_string_into, decode_native_prefs, decode_native_status, native_object,
    resolve_project_peer, unavailable, NativeStatus, RunBinding, SelfPeer, ERR_CONFLICT,
    RESPONSE_LIMIT,
};

/// Mirror of `parseProjectStatus`: the decoded status plus a terminal outcome
/// (`""` means Running, continue to preferences and binding checks).
fn parse_project_status(data: &[u8]) -> Result<(NativeStatus, String), String> {
    if data.len() > RESPONSE_LIMIT {
        return unavailable();
    }
    let v = native_object(data, &["BackendState", "HaveNodeKey"])?;
    let s = decode_native_status(&v)?;
    let outcome = match s.backend_state.as_str() {
        "NeedsLogin" => "needs-login",
        "NeedsMachineAuth" => "approval-required",
        "Starting" | "NoState" | "Stopped" => "pending",
        "Running" => "",
        _ => "unconfirmed",
    };
    Ok((s, outcome.to_string()))
}

/// Mirror of `validateProjectPreferences`.
fn validate_project_preferences(preferences: &[u8]) -> Result<(), String> {
    if preferences.len() > RESPONSE_LIMIT {
        return unavailable();
    }
    let v = native_object(
        preferences,
        &[
            "WantRunning",
            "CorpDNS",
            "RouteAll",
            "RunSSH",
            "ExitNodeID",
            "ExitNodeIP",
            "AdvertiseRoutes",
        ],
    )?;
    let p = decode_native_prefs(&v)?;
    if !p.want_running || !p.corp_dns || p.route_all || p.run_ssh {
        return Err(ERR_CONFLICT.to_string());
    }
    if !p.exit_node_id.is_empty() || !p.exit_node_ip.is_empty() || !p.advertise_routes.is_empty() {
        return Err(ERR_CONFLICT.to_string());
    }
    Ok(())
}

/// Mirror of `matchProjectSelf` (binding tags compared as-is, like Go).
fn match_project_self(peer: Option<&SelfPeer>, tags: &[String]) -> bool {
    let p = match peer {
        Some(p) => p,
        None => return false,
    };
    if p.id.is_empty() || !p.online || p.expired {
        return false;
    }
    let mut cloned = p.tags.clone();
    cloned.sort();
    cloned == tags
}

/// Mirror of `matchProjectBinding`.
fn match_project_binding(s: &NativeStatus, binding: &RunBinding) -> bool {
    if !binding.enabled || !s.have_node_key {
        return false;
    }
    match &s.tailnet {
        Some(name) if name == &binding.tailnet => {}
        _ => return false,
    }
    match_project_self(s.peer.as_ref(), &binding.tags)
}

/// Mirror of `ProjectStatus`: `(outcome, addresses, dns_name)`.
pub fn project_status(
    data: &[u8],
    prefs: &[u8],
    binding: &RunBinding,
) -> Result<(String, Vec<String>, String), String> {
    let (s, outcome) = parse_project_status(data)?;
    if !outcome.is_empty() {
        return Ok((outcome, Vec::new(), String::new()));
    }
    validate_project_preferences(prefs)?;
    if !match_project_binding(&s, binding) {
        return Ok(("unconfirmed".to_string(), Vec::new(), String::new()));
    }
    let peer = match &s.peer {
        Some(p) => p,
        None => return Ok(("unconfirmed".to_string(), Vec::new(), String::new())),
    };
    let (addrs, dns_name) = resolve_project_peer(&peer.dns_name, &peer.ips)?;
    Ok(("connected".to_string(), addrs, dns_name))
}

/// Mirror of `ProjectHasNode`.
pub fn project_has_node(data: &[u8]) -> Result<bool, String> {
    if data.len() > RESPONSE_LIMIT {
        return unavailable();
    }
    let v = native_object(data, &["BackendState", "HaveNodeKey"])?;
    let fields = match v.as_object() {
        Some(f) => f,
        None => return unavailable(),
    };
    let mut backend = String::new();
    let mut key = false;
    bind_string_into(fields, "BackendState", &mut backend)?;
    bind_bool_into(fields, "HaveNodeKey", &mut key)?;
    match backend.as_str() {
        "NoState" | "NeedsLogin" | "NeedsMachineAuth" | "Stopped" | "Starting" => {}
        "Running" => {
            if !key {
                return unavailable();
            }
        }
        _ => return unavailable(),
    }
    Ok(key)
}

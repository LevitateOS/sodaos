use super::{go_arch, Lifecycle, LifecycleState};
use crate::domain;

// ---------- facade confirmations ----------
//
// Pure client-side logic from `internal/host/{lifecycle,profiles,os,
// access_keys}.go`: transport (`c.call`) arrives with the route layer,
// but the confirmation predicates are exact here.

/// `validAddress`: a usable endpoint address, never unspecified,
/// multicast or loopback. Go's `netip.ParseAddr` also accepts scoped
/// (`%zone`) addresses, which this daemon never reports; those are
/// rejected here.
pub fn valid_address(value: &str) -> bool {
    match value.parse::<std::net::IpAddr>() {
        Ok(ip) => !ip.is_unspecified() && !ip.is_multicast() && !ip.is_loopback(),
        Err(_) => false,
    }
}

fn lifecycle_action_confirmed(action: &str, out: &LifecycleState) -> bool {
    match action {
        "start" => out.environment.running && out.boot_enabled,
        "stop" => !out.environment.running && !out.boot_enabled,
        _ => true,
    }
}

/// `Client.Lifecycle` outcome confirmation.
pub fn confirm_lifecycle(input: &Lifecycle, out: &LifecycleState) -> Result<(), String> {
    if out.environment.id != input.project {
        return Err("native lifecycle outcome not confirmed".to_string());
    }
    if !out.environment.ip.is_empty() && !valid_address(&out.environment.ip) {
        return Err("native lifecycle outcome not confirmed".to_string());
    }
    if !lifecycle_action_confirmed(&input.action, out) {
        return Err("native lifecycle outcome not confirmed".to_string());
    }
    Ok(())
}

/// `Client.ResolveProfile` confirmation: the profile validates and the
/// image is native to this backend. Like Go, a foreign architecture
/// overwrites a validation failure.
pub fn confirm_resolve_profile(p: &domain::Profile) -> Result<(), String> {
    let mut err = p.validate().err();
    if p.architecture != go_arch() {
        err = Some("project image is not native to this backend".to_string());
    }
    err.map_or(Ok(()), Err)
}

fn valid_os_environment(env: &domain::Environment) -> bool {
    if !env.image.is_empty()
        && (!env.image.starts_with("sha256:") || !domain::valid_image_ref(&env.image))
    {
        return false;
    }
    if !env.ip.is_empty() && !valid_address(&env.ip) {
        return false;
    }
    env.profile
        .as_ref()
        .map(|p| p.validate().is_ok())
        .unwrap_or(true)
}

fn valid_os_observation(id: &str, out: &domain::OsObservation) -> bool {
    if out.environment.id != id || out.release.is_none() != out.unavailable {
        return false;
    }
    if !valid_os_environment(&out.environment) {
        return false;
    }
    match &out.release {
        Some(release) => out.environment.running && domain::valid_os_release(release),
        None => true,
    }
}

/// `Client.ObserveOS` confirmation.
pub fn confirm_observe_os(id: &str, out: &domain::OsObservation) -> Result<(), String> {
    if valid_os_observation(id, out) {
        Ok(())
    } else {
        Err("invalid native OS observation".to_string())
    }
}

/// `Client.AccessKeys` confirmation. Go additionally rejects a null key
/// list, which the decoded struct cannot distinguish from empty; that
/// single check is not mirrored.
pub fn confirm_access_keys(
    input: &domain::AccessKeys,
    out: &domain::AccessKeyState,
) -> Result<(), String> {
    let mut err = crate::account::canonical_keys(&out.keys).err();
    if !crate::account::valid_key_revision(&out.revision) {
        err = Some("invalid native key revision".to_string());
    }
    if input.apply && out.keys.join("\n") != input.keys.join("\n") {
        err = Some("native key result differs from request".to_string());
    }
    err.map_or(Ok(()), Err)
}

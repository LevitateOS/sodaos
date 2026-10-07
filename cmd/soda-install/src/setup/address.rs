use crate::errors::Error;
use crate::netip;

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct SetupAddress {
    pub interface: String,
    pub address: String,
}

fn skip_setup_interface(name: &str, flags: &[String]) -> bool {
    let mut up = false;
    for flag in flags {
        up = up || flag == "UP";
    }
    !up || name == "soda0" || name.starts_with("podman") || name.starts_with("veth")
}

fn append_setup_address(
    choices: &mut Vec<SetupAddress>,
    seen: &mut std::collections::HashSet<String>,
    excluded: &mut Vec<String>,
    name: &str,
    local: &str,
    scope: &str,
) {
    if private_setup_origin(local).is_ok() && scope == "global" && !seen.contains(local) {
        choices.push(SetupAddress {
            interface: name.to_string(),
            address: local.to_string(),
        });
        seen.insert(local.to_string());
        return;
    }
    if excluded.len() < 8 {
        let reason = if scope != "global" {
            format!("scope {scope}, not global")
        } else if seen.contains(local) {
            "duplicate address".to_string()
        } else {
            "not a private LAN or Tailscale address".to_string()
        };
        excluded.push(format!("{name} {local} ({reason})"));
    }
}

pub fn private_setup_origin(value: &str) -> Result<String, Error> {
    let cgnat = netip::parse_prefix("100.64.0.0/10").expect("CGNAT prefix is valid");
    let address = netip::parse_addr(value)
        .map_err(|_| Error::msg("select an assigned private LAN or Tailscale address"))?;
    if !address.zone().is_empty()
        || address.is4_in6()
        || (!address.is_private() && !cgnat.contains(&address))
    {
        return Err(Error::msg(
            "select an assigned private LAN or Tailscale address",
        ));
    }
    let mut host = String::from_utf8_lossy(&address.to_string_go()).into_owned();
    if address.is6() {
        host = format!("[{host}]");
    }
    Ok(format!("https://{host}"))
}

pub(super) fn setup_addresses(data: &[u8]) -> Result<Vec<SetupAddress>, Error> {
    let text = String::from_utf8_lossy(data);
    let value: serde_json::Value = serde_json::from_str(&text)
        .map_err(|_| Error::msg("cannot inspect private setup addresses"))?;
    let interfaces = match &value {
        serde_json::Value::Array(items) => items,
        _ => return Err(Error::msg("cannot inspect private setup addresses")),
    };
    let mut choices = Vec::new();
    let mut excluded = Vec::new();
    let mut seen = std::collections::HashSet::new();
    for item in interfaces {
        let network = item
            .as_object()
            .ok_or_else(|| Error::msg("cannot inspect private setup addresses"))?;
        let name = optional_string(network.get("ifname"))?;
        let flags: Vec<String> = match network.get("flags") {
            None | Some(serde_json::Value::Null) => Vec::new(),
            Some(serde_json::Value::Array(items)) => items
                .iter()
                .map(|item| {
                    item.as_str()
                        .map(str::to_owned)
                        .ok_or_else(|| Error::msg("cannot inspect private setup addresses"))
                })
                .collect::<Result<_, _>>()?,
            Some(_) => return Err(Error::msg("cannot inspect private setup addresses")),
        };
        if skip_setup_interface(&name, &flags) {
            continue;
        }
        let addr_info: &[serde_json::Value] = match network.get("addr_info") {
            None | Some(serde_json::Value::Null) => &[],
            Some(serde_json::Value::Array(items)) => items,
            Some(_) => return Err(Error::msg("cannot inspect private setup addresses")),
        };
        for entry in addr_info {
            let address = entry
                .as_object()
                .ok_or_else(|| Error::msg("cannot inspect private setup addresses"))?;
            let local = optional_string(address.get("local"))?;
            let scope = optional_string(address.get("scope"))?;
            append_setup_address(
                &mut choices,
                &mut seen,
                &mut excluded,
                &name,
                &local,
                &scope,
            );
        }
    }
    if choices.is_empty() {
        // Name what the filter rejected so the operator can compare with the
        // unfiltered live addresses shown during disk installation.
        if excluded.is_empty() {
            return Err(Error::msg(
                "no private setup address is available; configure networking first",
            ));
        }
        return Err(Error::msg(format!(
            "no private setup address is available; configure networking first (observed but unusable: {})",
            excluded.join("; ")
        )));
    }
    Ok(choices)
}

fn optional_string(value: Option<&serde_json::Value>) -> Result<String, Error> {
    match value {
        None | Some(serde_json::Value::Null) => Ok(String::new()),
        Some(serde_json::Value::String(value)) => Ok(value.clone()),
        Some(_) => Err(Error::msg("cannot inspect private setup addresses")),
    }
}

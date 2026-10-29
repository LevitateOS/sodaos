use super::LaunchProvider;
use crate::terminal;

// ---------- connection directory (internal/host/muse.go + selection.go) ----------

/// Minimal broker connection record the Muse directory needs.
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct MuseConnection {
    pub id: String,
    pub provider_id: String,
    pub state: String,
}

/// Daemon `authorizeNested`/muse-authorize closure over `Available`:
/// any ready Muse connection authorizes.
pub fn muse_connection_authorized(connections: &[MuseConnection]) -> Result<(), String> {
    connection_authorized(connections, LaunchProvider::Muse)
}

/// Any ready connection of the launched provider authorizes.
pub fn connection_authorized(
    connections: &[MuseConnection],
    provider: LaunchProvider,
) -> Result<(), String> {
    for connection in connections {
        if connection.provider_id == provider.id() && connection.state == "ready" {
            return Ok(());
        }
    }
    Err(terminal::err_denied())
}

/// `identity.SelectMuseConnection`: the single ready Muse connection,
/// or the explicit choice; several require `SODA_MUSE_CONNECTION`.
pub fn select_muse_connection(
    connections: &[MuseConnection],
    selected: &str,
) -> Result<String, String> {
    select_connection(connections, LaunchProvider::Muse, selected)
}

/// The single ready connection of the launched provider, or the explicit
/// choice; several require the provider's connection setting.
pub fn select_connection(
    connections: &[MuseConnection],
    provider: LaunchProvider,
    selected: &str,
) -> Result<String, String> {
    let mut matches = Vec::new();
    for connection in connections {
        if connection.provider_id != provider.id() || connection.state != "ready" {
            continue;
        }
        if !selected.is_empty() && connection.id != selected {
            continue;
        }
        matches.push(connection.id.clone());
    }
    if matches.len() == 1 {
        return Ok(matches.pop().unwrap());
    }
    if matches.len() > 1 {
        return Err(format!(
            "choose an authorized {} connection with {}",
            provider.id(),
            provider.connection_setting()
        ));
    }
    Err(terminal::err_denied())
}

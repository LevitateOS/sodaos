use super::MUSE_CONNECTION_SETTING;
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
    for connection in connections {
        if connection.provider_id == terminal::PROVIDER_MUSE && connection.state == "ready" {
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
    let mut matches = Vec::new();
    for connection in connections {
        if connection.provider_id != terminal::PROVIDER_MUSE || connection.state != "ready" {
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
            "choose an authorized {} connection with {MUSE_CONNECTION_SETTING}",
            terminal::PROVIDER_MUSE
        ));
    }
    Err(terminal::err_denied())
}

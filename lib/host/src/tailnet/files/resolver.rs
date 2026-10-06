use crate::tailnet_domain::{ReadFile, StatFile, ERR_CONFLICT, ERR_UNAVAILABLE, ERR_UNSUPPORTED};
use crate::tailnet_runtime::ProjectRun;

use super::same_file;

fn fresh_tailscale_conflict(devices: &str) -> bool {
    for line in devices.split('\n') {
        if let Some((name, _)) = line.split_once(':') {
            if name.trim() == "tailscale0" {
                return true;
            }
        }
    }
    false
}

fn validate_resolver_text(data: &[u8], fresh: bool) -> Result<(), String> {
    if data.len() > 16384 || data.is_empty() {
        return Err(ERR_UNSUPPORTED.to_string());
    }
    let text = String::from_utf8_lossy(data).to_lowercase();
    if text.contains("systemd-resolved")
        || text.contains("resolvconf")
        || (fresh && text.contains("tailscale"))
    {
        return Err(ERR_UNSUPPORTED.to_string());
    }
    Ok(())
}

fn resolver_path(container: &str) -> String {
    format!("/var/lib/containers/storage/overlay-containers/{container}/userdata/resolv.conf")
}

fn validate_resolver_inode(run: &ProjectRun, stat: &StatFile) -> Result<(), String> {
    let expected = resolver_path(&run.target.container);
    if run.resolver != expected {
        return Err(ERR_UNSUPPORTED.to_string());
    }
    let info = stat(&expected).map_err(|_| ERR_UNSUPPORTED.to_string())?;
    if !info.is_file() || info.len() > 16384 {
        return Err(ERR_UNSUPPORTED.to_string());
    }
    let actual = stat(&format!("/proc/{}/root/etc/resolv.conf", run.pid))
        .map_err(|_| ERR_UNSUPPORTED.to_string())?;
    if !same_file(&info, &actual) {
        return Err(ERR_UNSUPPORTED.to_string());
    }
    Ok(())
}

/// Validate the real shared resolver inode, not an arbitrary path obtained
/// from the project. Tailscale owns backup/restore in the retained companion
/// root. `stat` must follow links (like Go `os.Stat`).
pub fn validate_run_resolver(
    run: &ProjectRun,
    read: ReadFile,
    stat: StatFile,
    fresh: bool,
) -> Result<(), String> {
    validate_resolver_inode(run, &stat)?;
    let expected = resolver_path(&run.target.container);
    let data = read(&expected).map_err(|_| ERR_UNSUPPORTED.to_string())?;
    validate_resolver_text(&data, fresh)?;
    let devices =
        read(&format!("/proc/{}/net/dev", run.pid)).map_err(|_| ERR_UNAVAILABLE.to_string())?;
    if devices.len() > 65536 {
        return Err(ERR_UNAVAILABLE.to_string());
    }
    if fresh && fresh_tailscale_conflict(&String::from_utf8_lossy(&devices)) {
        return Err(ERR_CONFLICT.to_string());
    }
    Ok(())
}

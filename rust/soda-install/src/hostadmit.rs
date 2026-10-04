//! Host admission: the disk action requires the live ISO on Fedora
//! CoreOS with SELinux enforcing; installed-host actions require the
//! installed host.

use std::collections::BTreeMap;

use crate::command::Runner;
use crate::errors::Error;
use crate::fmtx::go_trim_space;
use crate::run;
use crate::signal::Ctx;

pub fn live_iso_from_cmdline(data: &[u8]) -> bool {
    for arg in String::from_utf8_lossy(data).split_whitespace() {
        if arg == "coreos.liveiso"
            || arg.starts_with("coreos.liveiso=")
            || arg.starts_with("coreos.live.rootfs_url=")
        {
            return true;
        }
    }
    false
}

fn admit_live_installer(
    ctx: &Ctx,
    run: &dyn Runner,
    values: &BTreeMap<String, String>,
) -> Result<(), Error> {
    let media = run::read_media_identity(&format!("{}/media.json", run::DATA_DIR))
        .map_err(|_| Error::msg("missing media identity"))?;
    // Selected CoreOS reports the Fedora major in VERSION_ID (44), and
    // the exact image release in IMAGE_VERSION (44.20260817.3.2).
    media.validate(
        values
            .get("IMAGE_VERSION")
            .map(|s| s.as_str())
            .unwrap_or(""),
        &run::architecture(),
    )?;
    let observed = run.run(ctx, "coreos-installer", &["--version".to_string()], None);
    let matches = match observed {
        Ok(data) => go_trim_space(&String::from_utf8_lossy(&data)) == media.installer_version,
        Err(_) => false,
    };
    if !matches {
        return Err(Error::msg("unreviewed CoreOS Installer version"));
    }
    Ok(())
}

pub fn selinux_enforcing() -> Result<(), Error> {
    let enforcing = std::fs::read("/sys/fs/selinux/enforce");
    let ok = match enforcing {
        Ok(data) => go_trim_space(&String::from_utf8_lossy(&data)) == "1",
        Err(_) => false,
    };
    if !ok {
        return Err(Error::msg("SELinux must remain enforcing"));
    }
    Ok(())
}

pub fn os_release(data: &[u8]) -> BTreeMap<String, String> {
    let mut values = BTreeMap::new();
    for line in String::from_utf8_lossy(data).split('\n') {
        if let Some((key, value)) = line.split_once('=') {
            values.insert(
                key.to_string(),
                value.trim_matches(|c| c == '"' || c == '\'').to_string(),
            );
        }
    }
    values
}

pub fn core_os_host(live: bool, run: &dyn Runner) -> Result<(), Error> {
    // Display branding in /etc must not become a stale copy of base identity.
    let data = std::fs::read("/usr/lib/os-release")
        .map_err(|e| crate::errors::path_error("open", "/usr/lib/os-release", e))?;
    let values = os_release(&data);
    if values.get("ID").map(|s| s.as_str()) != Some("fedora")
        || values.get("VARIANT_ID").map(|s| s.as_str()) != Some("coreos")
    {
        return Err(Error::msg("upstream Fedora CoreOS required"));
    }
    let cmdline = std::fs::read("/proc/cmdline")
        .map_err(|e| crate::errors::path_error("open", "/proc/cmdline", e))?;
    if live != live_iso_from_cmdline(&cmdline) {
        return Err(Error::msg(
            "disk action requires the live ISO; installed-host actions require the installed host",
        ));
    }
    if live {
        // Go uses a background context for this version inspection.
        let (ctx, _flag) = Ctx::test();
        admit_live_installer(&ctx, run, &values)?;
    }
    selinux_enforcing()
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn os_release_and_live_detection() {
        let values = os_release(
            b"ID=fedora\nVARIANT_ID=coreos\nVERSION_ID=44\nIMAGE_VERSION='44.20260817.3.2'\n",
        );
        assert_eq!(values["VERSION_ID"], "44");
        assert_eq!(values["IMAGE_VERSION"], "44.20260817.3.2");
        assert_eq!(values["VARIANT_ID"], "coreos");
        assert!(live_iso_from_cmdline(
            b"root=live:/dev/sdb1 coreos.liveiso quiet"
        ));
        assert!(live_iso_from_cmdline(b"coreos.liveiso=1"));
        assert!(live_iso_from_cmdline(b"coreos.live.rootfs_url=http://x/y"));
        assert!(!live_iso_from_cmdline(b"root=/dev/sda2 quiet"));
        assert!(!live_iso_from_cmdline(b""));
    }
}

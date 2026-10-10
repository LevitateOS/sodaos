use std::fs;

use crate::error::Error;
use crate::foreign::Production;
use crate::ignition;
use crate::media::{MediaMeta, NativeFn};
use crate::rootfs;
use crate::sys;

pub fn validate_installer_version(version: &str) -> Result<(), Error> {
    let Some(reported) = version.strip_prefix("coreos-installer ") else {
        return Err(Error::msg("unrecognized installer version output"));
    };
    if reported.trim().is_empty() || version.chars().any(|c| c.is_control()) {
        return Err(Error::msg("unrecognized installer version output"));
    }
    Ok(())
}

/// Read the installer version from the exact selected OCI image.
pub fn observe_installer_version(
    production: &dyn Production,
    root: &str,
    image: &str,
) -> Result<String, Error> {
    let version = production.capture(
        root,
        "podman",
        &[
            "--remote=false".to_string(),
            "run".to_string(),
            "--rm".to_string(),
            "--timeout=110".to_string(),
            "--pull=never".to_string(),
            "--network=none".to_string(),
            "--read-only".to_string(),
            "--cap-drop=all".to_string(),
            "--security-opt=label=disable".to_string(),
            "--entrypoint=/usr/bin/coreos-installer".to_string(),
            image.to_string(),
            "--version".to_string(),
        ],
    )?;
    let version = version.trim().to_string();
    validate_installer_version(&version)?;
    Ok(version)
}

pub fn setup_media_rootfs(
    artifacts: &str,
    build_dir: &str,
    rootfs_path: &str,
    rootfs_sha: &str,
) -> Result<(String, String), Error> {
    let media_dir = sys::join(&[artifacts, "media"]);
    sys::create_dir(&media_dir, 0o755)?;
    let rootfs_name = format!("{rootfs_sha}-rootfs.img");
    fs::hard_link(
        sys::join(&[build_dir, rootfs_path]),
        sys::join(&[&media_dir, &rootfs_name]),
    )
    .map_err(|e| Error::msg(e.to_string()))?;
    Ok((media_dir, rootfs_name))
}

pub fn verify_customized_iso(
    native: NativeFn<'_>,
    artifacts: &str,
    rootfs_url: &str,
) -> Result<(), Error> {
    let ignition = native(
        "/usr/bin/coreos-installer",
        &[
            "iso".to_string(),
            "ignition".to_string(),
            "show".to_string(),
            "/out/media/installer.iso".to_string(),
        ],
    )?;
    let expected = fs::read(sys::join(&[artifacts, "live.ign"]))?;
    // iso customize wraps the supplied fragment in a merge source. Require the
    // exact public fragment to survive native readback (checked below).
    ignition::verify_live_ignition(ignition.as_bytes(), &expected)?;
    let kargs = native(
        "/usr/bin/coreos-installer",
        &[
            "iso".to_string(),
            "kargs".to_string(),
            "show".to_string(),
            "/out/media/installer.iso".to_string(),
        ],
    )?;
    if !kargs.contains(&format!("coreos.live.rootfs_url={rootfs_url}"))
        || kargs.contains("coreos.liveiso")
    {
        return Err(Error::msg("minimal-media kernel arguments differ"));
    }
    Ok(())
}

pub fn customize_installer_iso(
    production: &dyn Production,
    root: &str,
    artifacts: &str,
    build_dir: &str,
    id: &str,
    rootfs_url: &str,
    live_iso_path: &str,
) -> Result<String, Error> {
    let native = |name: &str, args: &[String]| -> Result<String, Error> {
        let mut prefix = vec![
            "--remote=false".to_string(),
            "run".to_string(),
            "--rm".to_string(),
            "--pull=never".to_string(),
            "--network=none".to_string(),
            "--read-only".to_string(),
            "--cap-drop=all".to_string(),
            "--security-opt=label=disable".to_string(),
            format!("--volume={build_dir}:/build:ro"),
            format!("--volume={artifacts}:/out:rw"),
            format!("--entrypoint={name}"),
            id.to_string(),
        ];
        prefix.extend_from_slice(args);
        production.capture(root, "podman", &prefix)
    };
    let version = native("/usr/bin/coreos-installer", &["--version".to_string()])?;
    let version = version.trim().to_string();
    validate_installer_version(&version)?;
    native(
        "/usr/bin/coreos-installer",
        &[
            "iso".to_string(),
            "extract".to_string(),
            "minimal-iso".to_string(),
            format!("/build/{live_iso_path}"),
            "/out/media/minimal.iso".to_string(),
        ],
    )?;
    native(
        "/usr/bin/coreos-installer",
        &[
            "iso".to_string(),
            "customize".to_string(),
            "--live-ignition".to_string(),
            "/out/live.ign".to_string(),
            "--live-karg-append".to_string(),
            format!("coreos.live.rootfs_url={rootfs_url}"),
            "--output".to_string(),
            "/out/media/installer.iso".to_string(),
            "/out/media/minimal.iso".to_string(),
        ],
    )?;
    verify_customized_iso(&native, artifacts, rootfs_url)?;
    Ok(version)
}

pub fn verify_media_readback(
    production: &dyn Production,
    root: &str,
    artifacts: &str,
    build_dir: &str,
    id: &str,
    media_dir: &str,
    rootfs_name: &str,
) -> Result<(), Error> {
    let native = |name: &str, args: &[String]| -> Result<String, Error> {
        let mut prefix = vec![
            "--remote=false".to_string(),
            "run".to_string(),
            "--rm".to_string(),
            "--pull=never".to_string(),
            "--network=none".to_string(),
            "--read-only".to_string(),
            "--cap-drop=all".to_string(),
            "--security-opt=label=disable".to_string(),
            format!("--volume={build_dir}:/build:ro"),
            format!("--volume={artifacts}:/out:rw"),
            format!("--entrypoint={name}"),
            id.to_string(),
        ];
        prefix.extend_from_slice(args);
        production.capture(root, "podman", &prefix)
    };
    let readback = sys::join(&[media_dir, "readback"]);
    sys::create_dir(&readback, 0o755)?;
    native(
        "/usr/bin/coreos-installer",
        &[
            "iso".to_string(),
            "extract".to_string(),
            "pxe".to_string(),
            "--output-dir".to_string(),
            "/out/media/readback".to_string(),
            "/out/media/installer.iso".to_string(),
        ],
    )?;
    let mut initrds: Vec<String> = Vec::new();
    for entry in fs::read_dir(&readback)? {
        let entry = entry?;
        let name = entry.file_name().to_string_lossy().into_owned();
        if name.ends_with("-initrd.img") {
            initrds.push(name);
        }
    }
    if initrds.len() != 1 {
        return Err(Error::msg("missing ISO initrd readback"));
    }
    native(
        "/usr/bin/coreos-installer",
        &[
            "dev".to_string(),
            "extract".to_string(),
            "initrd".to_string(),
            "--directory".to_string(),
            "/out/media/readback".to_string(),
            format!("/out/media/readback/{}", initrds[0]),
            "etc/coreos-live-want-rootfs".to_string(),
        ],
    )?;
    let chunks = fs::read(sys::join(&[&readback, "etc/coreos-live-want-rootfs"]))?;
    rootfs::verify_rootfs_chunks(
        &sys::join(&[media_dir, rootfs_name]),
        &String::from_utf8_lossy(&chunks),
    )
}

pub fn prepare_and_verify_media(
    production: &dyn Production,
    root: &str,
    artifacts: &str,
    build_dir: &str,
    id: &str,
    media_dir: &str,
    meta: &MediaMeta,
    rootfs_url: &str,
    rootfs_name: &str,
) -> Result<String, Error> {
    let live_iso = meta.images.get("live-iso").cloned().unwrap_or_default();
    let version = customize_installer_iso(
        production,
        root,
        artifacts,
        build_dir,
        id,
        rootfs_url,
        &live_iso.path,
    )?;
    verify_media_readback(
        production,
        root,
        artifacts,
        build_dir,
        id,
        media_dir,
        rootfs_name,
    )?;
    Ok(version)
}

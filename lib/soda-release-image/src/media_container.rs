use std::collections::HashMap;
use std::fs;

use crate::error::Error;
use crate::foreign::Production;
use crate::jsonio;
use crate::media::builder_id;
use crate::model;
use crate::sys;

pub fn stop_packaging_container(root: &str) {
    let cid_path = sys::join(&[root, "packaging.cid"]);
    let Ok(cid_bytes) = fs::read(&cid_path) else {
        return;
    };
    let cid = String::from_utf8_lossy(&cid_bytes).trim().to_string();
    if !model::is_digest(&cid) {
        return;
    }
    let mut child = match std::process::Command::new("/usr/bin/podman")
        .args(["--remote=false", "stop", "--time=10", &cid])
        .env_clear()
        .envs(crate::build_runner::build_environment_pairs())
        .spawn()
    {
        Ok(child) => child,
        Err(_) => return,
    };
    let deadline = std::time::Instant::now() + std::time::Duration::from_secs(30);
    loop {
        match child.try_wait() {
            Ok(Some(_)) => break,
            Ok(None) if std::time::Instant::now() < deadline => {
                std::thread::sleep(std::time::Duration::from_millis(50));
            }
            _ => {
                let _ = child.kill();
                let _ = child.wait();
                break;
            }
        }
    }
}

pub fn build_media_container(
    production: &dyn Production,
    root: &str,
    artifacts: &str,
    payload_id: &str,
    work: &str,
    id: &str,
) -> Result<(), Error> {
    let mut base = vec![
        "--remote=false".to_string(),
        "run".to_string(),
        "--rm".to_string(),
        format!("--cidfile={}", sys::join(&[root, "packaging.cid"])),
        "--network=none".to_string(),
        "--privileged".to_string(),
        "--security-opt=label=disable".to_string(),
        "--device=/dev/kvm".to_string(),
        "--device=/dev/fuse".to_string(),
        "--cpus=4".to_string(),
        "--memory=16g".to_string(),
        "--env=COSA_SUPERMIN_MEMORY=12288".to_string(),
        "--env=RUNVM_NONET=1".to_string(),
        format!("--volume={work}:/srv:rw"),
        format!("--volume={}/config:/config:ro", root),
        format!("--volume={artifacts}:/inputs:ro"),
        format!(
            "--volume={}/assembler-root.oci:/assembler-root.oci:ro",
            root
        ),
        "--workdir=/srv".to_string(),
        "--entrypoint=/usr/bin/taskset".to_string(),
        id.to_string(),
        "-c".to_string(),
        "0-3".to_string(),
        "/bin/bash".to_string(),
        "-euc".to_string(),
    ];
    let script = "umask 0022; cosa init /config; cp /assembler-root.oci /srv/tmp/assembler-root.oci; cosa import --skip-prune oci-archive:/inputs/host.oci; cosa buildextend-live --build \"$1\"";
    base.push(script.to_string());
    base.push("assemble".to_string());
    base.push(payload_id.to_string());
    if let Err(e) = production.execute(root, "podman", &base) {
        // Podman owns the helper VM's namespace, outside the CLI process group.
        // Stop only this run's container even when the build context is cancelled.
        stop_packaging_container(root);
        return Err(e);
    }
    Ok(())
}

pub fn assemble_native_media(
    production: &dyn Production,
    root: &str,
    artifacts: &str,
    payload_id: &str,
) -> Result<(String, String), Error> {
    let work = sys::join(&[root, "work"]);
    sys::create_dir(&work, 0o755)?;
    let id = builder_id(root)?;
    build_media_container(production, root, artifacts, payload_id, &work, &id)?;
    Ok((
        sys::join(&[&work, "builds", payload_id, production.arch()]),
        id,
    ))
}

#[derive(Debug, Clone, Default)]
pub struct MediaMetaImage {
    pub path: String,
    pub sha256: String,
    pub size: i64,
}

#[derive(Debug, Clone, Default)]
pub struct MediaMeta {
    pub ostree_commit: String,
    pub images: HashMap<String, MediaMetaImage>,
}

#[derive(Default)]
struct MediaMetaWire {
    ostree_commit: String,
    images: crate::jsonio::OrderedMap<MediaMetaImage>,
}

crate::jsonio::case_record!(MediaMetaImage, ignore_unknown, {
    path: String => "Path",
    sha256: String => "SHA256",
    size: i64 => "Size",
});

crate::jsonio::case_record!(MediaMetaWire, ignore_unknown, {
    ostree_commit: String => "ostree-commit",
    images: crate::jsonio::OrderedMap<MediaMetaImage> => "Images",
});

#[cfg(test)]
mod tests {
    use super::MediaMeta;

    #[test]
    fn assembler_metadata_ignores_observed_root_and_image_extras() {
        let meta = MediaMeta::parse(
            r#"{
                "ostree-commit":"commit-id",
                "ostree-version":"44.20260913.3.2.soda-test",
                "images":{
                    "ostree":{
                        "path":"ostree-commit.tar",
                        "sha256":"abc",
                        "size":12,
                        "skip-compression":true
                    }
                }
            }"#,
        )
        .expect("Assembler metadata with its producer fields should parse");

        assert_eq!(meta.ostree_commit, "commit-id");
        let image = meta.images.get("ostree").unwrap();
        assert_eq!(image.path, "ostree-commit.tar");
        assert_eq!(image.sha256, "abc");
        assert_eq!(image.size, 12);
    }
}

impl MediaMeta {
    pub fn parse(text: &str) -> Result<MediaMeta, Error> {
        let wire: MediaMetaWire = jsonio::parse(text)?;
        Ok(MediaMeta {
            ostree_commit: wire.ostree_commit,
            images: wire.images.0.into_iter().collect(),
        })
    }
}

pub fn verify_meta_images(
    build_dir: &str,
    images: &HashMap<String, MediaMetaImage>,
) -> Result<(), Error> {
    for name in [
        "ostree",
        "oci-manifest",
        "live-iso",
        "live-rootfs",
        "live-initramfs",
    ] {
        let file = images.get(name);
        match file {
            Some(file) if sys::base_name(&file.path) == file.path => {
                match sys::hash_file(&sys::join(&[build_dir, &file.path])) {
                    Ok(hash) if hash == file.sha256 => {}
                    _ => return Err(Error::msg("native output checksum mismatch")),
                }
            }
            _ => return Err(Error::msg("missing native media output")),
        }
    }
    Ok(())
}

pub fn verify_build_meta(
    build_dir: &str,
    candidate: &model::Candidate,
) -> Result<MediaMeta, Error> {
    let data = fs::read(sys::join(&[build_dir, "meta.json"]))?;
    let text = String::from_utf8_lossy(&data).into_owned();
    // Plain (non-strict) decode, like the Go owner.
    let meta = MediaMeta::parse(&text)?;
    verify_meta_images(build_dir, &meta.images)?;
    let ostree = meta.images.get("ostree").cloned().unwrap_or_default();
    let oci_manifest = meta.images.get("oci-manifest").cloned().unwrap_or_default();
    if ostree.sha256 != candidate.host_archive_sha256
        || oci_manifest.sha256 != candidate.host.manifest.trim_start_matches("sha256:")
    {
        return Err(Error::msg("native import changed candidate"));
    }
    Ok(meta)
}

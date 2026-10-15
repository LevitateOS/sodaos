//! `complete.go`: candidate completion into the host context.

use std::fs;
use std::os::unix::fs::MetadataExt;

use crate::error::Error;
use crate::files;
use crate::jsonio;
use crate::model;
use crate::quadlet;
use crate::sys;

pub fn stage_presentation(forgejo_context: &str, context: &str) -> Result<String, Error> {
    let mut staged = files::public_files(&sys::join(&[forgejo_context, "forgejo"]))?;
    staged.sort_by(|a, b| a.0.cmp(&b.0));
    let header = staged
        .iter()
        .find(|(name, _)| name == "templates/custom/header.tmpl")
        .map(|(_, hash)| hash.clone())
        .unwrap_or_default();
    if staged.is_empty() || header.is_empty() {
        return Err(Error::msg("complete Forgejo presentation required"));
    }
    let mut data = serde_json::to_string_pretty(&jsonio::SortedPairs(&staged))
        .expect("serialization to String cannot fail");
    data.push('\n');
    files::owned_write(
        &sys::join(&[forgejo_context, "presentation.json"]),
        data.as_bytes(),
        0o644,
    )?;
    files::owned_write(
        &sys::join(&[
            context,
            "rootfs/usr/share/soda/host-image/presentation.json",
        ]),
        data.as_bytes(),
        0o644,
    )?;
    Ok(files::hash_bytes(data.as_bytes()))
}

pub fn validate_complete_payload(payload: &model::Payload) -> Result<(), Error> {
    payload.validate().map_err(|e| Error::msg(e.to_string()))?;
    if payload.format != 4 {
        return Err(Error::msg("new candidates require the thin v4 payload"));
    }
    Ok(())
}

/// Generate the service-owned Quadlets by exact config ID with Pull=never,
/// ordered after the required image acquisition (the generator pins the
/// image, drops any source pull policy, and adds the importer ordering).
/// The required Soda native package bootstraps from the trusted local
/// catalog, not from an installer image: the obsolete extension-install
/// path is removed.
pub fn write_complete_quadlets(
    source: &str,
    root: &str,
    payload: &model::Payload,
) -> Result<(), Error> {
    for (role, unit) in [
        (model::Role::Forgejo, "forgejo.container"),
        (model::Role::Dashboard, "soda-dashboard.container"),
        (model::Role::Proxy, "soda-proxy.container"),
        (model::Role::Postgres, "soda-postgres.container"),
    ] {
        let original = fs::read(sys::join(&[source, "system/host/services", unit]))?;
        let body = quadlet::local_quadlet(
            &String::from_utf8_lossy(&original),
            &role.binding(&payload.images).config,
        )?;
        files::owned_write(
            &sys::join(&[root, "usr/share/containers/systemd", unit]),
            body.as_bytes(),
            0o644,
        )?;
    }
    Ok(())
}

/// Stage the trusted extension catalog the required Soda package
/// bootstraps from. The bytes are the payload's own projection, verified
/// equal to the candidate evidence before staging.
pub fn stage_extension_catalog(root: &str, payload: &model::Payload) -> Result<(), Error> {
    let bytes = payload
        .projected_catalog_bytes()
        .map_err(|e| Error::msg(e.to_string()))?;
    let dir = sys::join(&[root, "usr/share/soda/extensions"]);
    fs::create_dir_all(&dir)?;
    files::owned_write(&sys::join(&[&dir, "catalog.json"]), &bytes, 0o644)
}

pub fn verify_public_branding_and_motd(root: &str) -> Result<(), Error> {
    for dir in [
        "etc/cockpit/branding",
        "usr/share/soda/fastfetch",
        "etc/fastfetch",
    ] {
        files::public_files(&sys::join(&[root, dir]))?;
    }
    let motd = fs::symlink_metadata(sys::join(&[root, "etc/motd"]));
    match motd {
        Ok(info) => {
            if info.mode() != 0o100644 {
                return Err(Error::msg("regular public MOTD required"));
            }
        }
        Err(_) => return Err(Error::msg("regular public MOTD required")),
    }
    Ok(())
}

pub fn write_factory_defaults(root: &str) -> Result<(), Error> {
    for name in ["forgejo.env", "proxy.Caddyfile"] {
        let data = fs::read(sys::join(&[root, "etc/soda", name]))?;
        files::owned_write(
            &sys::join(&[root, "usr/share/soda/defaults", name]),
            &data,
            0o644,
        )?;
    }
    // Machine setup must supply the explicit private subnet. Images are not saved
    // here: the vendor helper reads their immutable IDs from the release owner.
    let example =
        b"{\n  \"network\": \"soda-projects\",\n  \"bridge\": \"soda0\",\n  \"subnet\": \"\"\n}\n";
    files::owned_write(
        &sys::join(&[root, "usr/share/soda/defaults/host.example.json"]),
        example,
        0o644,
    )
}

pub fn configure_complete_systemd(source: &str, root: &str) -> Result<(), Error> {
    let mut pairs = [
        (
            "system/host/image/soda-image-import.service",
            "usr/lib/systemd/system/soda-image-import.service",
        ),
        (
            "system/host/image/retained-images.conf",
            "usr/lib/systemd/system/soda-host.service.d/10-images.conf",
        ),
    ];
    pairs.sort();
    for (from, to) in pairs {
        let data = fs::read(sys::join(&[source, from]))?;
        files::owned_write(&sys::join(&[root, to]), &data, 0o644)?;
    }
    let dropin = fs::read(sys::join(&[
        source,
        "system/host/image/retained-images.conf",
    ]))?;
    // Keep the project unit's exact-fragment/drop-in admission contract unchanged:
    // put this dependency directly in its generated vendor fragment, not a drop-in.
    let project = sys::join(&[root, "usr/lib/systemd/system/soda-project@.service"]);
    let body = fs::read(&project)?;
    let text = String::from_utf8_lossy(&body);
    let replaced = text.replacen(
        "[Unit]\n",
        &format!("{}\n", String::from_utf8_lossy(&dropin)),
        1,
    );
    files::owned_write(&project, replaced.as_bytes(), 0o644)
}

pub fn write_release_metadata_and_normalize(
    root: &str,
    payload: &model::Payload,
) -> Result<(), Error> {
    // The complete payload is the sole resolved-image/defaults owner. Leave only
    // a pointer in the earlier host-content build marker, not stale scope data.
    files::owned_write(
        &sys::join(&[root, "usr/share/soda/host-image/build.json"]),
        b"{\"Scope\":\"thin-local-payload\",\"ReleaseMetadata\":\"/usr/share/soda/release.json\"}\n",
        0o644,
    )?;
    let mut data =
        serde_json::to_string_pretty(payload).expect("serialization to String cannot fail");
    data.push('\n');
    files::owned_write(
        &sys::join(&[root, "usr/share/soda/release.json"]),
        data.as_bytes(),
        0o644,
    )?;
    // Normalize only fresh image context directories, not credentials or sources.
    sys::walk(root, |path, is_dir, _| {
        if is_dir {
            use std::os::unix::fs::PermissionsExt;
            fs::set_permissions(path, fs::Permissions::from_mode(0o755))?;
        }
        Ok(())
    })
}

/// Complete binds all callers to one thin v4 payload and ordinary Podman
/// storage. No application OCI body enters the host: the six required
/// images are acquired at boot from the sealed bindings, and the required
/// native package bootstraps from the staged trusted catalog.
pub fn complete(source: &str, context: &str, payload: &model::Payload) -> Result<(), Error> {
    validate_complete_payload(payload)?;
    let root = sys::join(&[context, "rootfs"]);
    write_complete_quadlets(source, &root, payload)?;
    verify_public_branding_and_motd(&root)?;
    write_factory_defaults(&root)?;
    configure_complete_systemd(source, &root)?;
    stage_extension_catalog(&root, payload)?;
    write_release_metadata_and_normalize(&root, payload)
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn oracle_complete_refuses_before_payload() {
        // Oracle: Go TestCompleteCandidateBindingsAndStateOwnership (admission half).
        let payload = model::Payload::default();
        assert_eq!(
            validate_complete_payload(&payload).unwrap_err().0,
            "invalid appliance payload identity"
        );
        assert!(complete("/src", "/ctx", &payload).is_err());
    }

    #[test]
    fn quadlets_read_new_layout() {
        // CORR-C-005: units live under system/host/services/. The obsolete
        // extension-install unit is gone: generated units are the four
        // service-owned Quadlets ordered after the image importer.
        let dir = std::env::temp_dir().join(format!("sri-cq-{}", std::process::id()));
        let _ = fs::remove_dir_all(&dir);
        let source = dir.join("src");
        let units = source.join("system/host/services");
        fs::create_dir_all(&units).unwrap();
        for name in [
            "forgejo.container",
            "soda-dashboard.container",
            "soda-proxy.container",
            "soda-postgres.container",
        ] {
            let body = if name == "forgejo.container" {
                include_str!("../../../system/host/services/forgejo.container")
            } else {
                "[Unit]\nDescription=x\n[Container]\nImage=localhost/x:dev\nPull=newer\nExec=a\n"
            };
            fs::write(units.join(name), body).unwrap();
        }
        let root = dir.join("root");
        let image_config =
            "sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef";
        let mut payload: model::Payload = serde_json::from_str(include_str!(
            "../../release-inputs/fixtures/payload-v4.json"
        ))
        .unwrap();
        model::Role::Forgejo.binding_mut(&mut payload.images).config = image_config.to_string();
        write_complete_quadlets(source.to_str().unwrap(), root.to_str().unwrap(), &payload)
            .unwrap();
        for name in [
            "forgejo.container",
            "soda-dashboard.container",
            "soda-proxy.container",
            "soda-postgres.container",
        ] {
            let generated =
                fs::read_to_string(root.join("usr/share/containers/systemd").join(name)).unwrap();
            // Exact config ID, never a source pull policy, always ordered
            // after the required image acquisition.
            assert!(generated.contains("Pull=never"), "{name}");
            assert!(!generated.contains("Pull=newer"), "{name}");
            assert!(
                generated.contains("Requires=soda-image-import.service"),
                "{name}"
            );
            assert!(
                generated.contains("After=soda-image-import.service"),
                "{name}"
            );
        }
        let forgejo =
            fs::read_to_string(root.join("usr/share/containers/systemd/forgejo.container"))
                .unwrap();
        assert!(forgejo.contains(&format!("Image={image_config}")));
        assert!(forgejo.contains("Environment=FORGEJO__server__APP_DATA_PATH=/data/gitea"));
        assert!(forgejo.contains("Volume=/var/lib/soda/forgejo:/data:Z"));
        let _ = fs::remove_dir_all(&dir);
    }

    #[test]
    fn catalog_stages_projected_bytes() {
        let dir = std::env::temp_dir().join(format!("sri-cat-{}", std::process::id()));
        let _ = fs::remove_dir_all(&dir);
        let root = dir.join("root");
        let payload: model::Payload = serde_json::from_str(include_str!(
            "../../release-inputs/fixtures/payload-v4.json"
        ))
        .unwrap();
        stage_extension_catalog(root.to_str().unwrap(), &payload).unwrap();
        let staged = fs::read(root.join("usr/share/soda/extensions/catalog.json")).unwrap();
        assert_eq!(staged, payload.projected_catalog_bytes().unwrap());
        let _ = fs::remove_dir_all(&dir);
    }
}

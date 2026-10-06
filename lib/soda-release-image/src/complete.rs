//! `complete.go`: candidate completion into the host context.

use std::fs;
use std::os::unix::fs::MetadataExt;

use crate::error::Error;
use crate::files;
use crate::foreign::Production;
use crate::jsonio;
use crate::layout;
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
    let value = soda_json::JsonValue::Object(
        staged
            .into_iter()
            .map(|(k, v)| (k, soda_json::JsonValue::Str(v)))
            .collect(),
    );
    let mut data = jsonio::to_indent(&value);
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
    payload.validate()?;
    if payload.format != 3 {
        return Err(Error::msg(
            "new candidates require the shared-layout v3 payload",
        ));
    }
    Ok(())
}

pub fn write_complete_quadlets(
    source: &str,
    root: &str,
    payload: &model::Payload,
) -> Result<(), Error> {
    for (name, unit) in [
        ("forgejo", "forgejo.container"),
        ("dashboard", "soda-dashboard.container"),
        ("proxy", "soda-proxy.container"),
    ] {
        let original = fs::read(sys::join(&[source, "appliance/services", unit]))?;
        let body = quadlet::local_quadlet(
            &String::from_utf8_lossy(&original),
            &payload.image(name).config,
        )?;
        files::owned_write(
            &sys::join(&[root, "usr/share/containers/systemd", unit]),
            body.as_bytes(),
            0o644,
        )?;
    }
    Ok(())
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
    let example = b"{\n  \"network\": \"soda-projects\",\n  \"bridge\": \"soda0\",\n  \"subnet\": \"\",\n  \"tailnet_management\": false\n}\n";
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

pub fn bind_extension_install_image(root: &str, payload: &model::Payload) -> Result<(), Error> {
    let path = sys::join(&[
        root,
        "usr/lib/systemd/system/soda-extension-install.service",
    ]);
    let unit = fs::read(&path)?;
    const PLACEHOLDER: &str = "localhost/soda-extension:dev";
    let text = String::from_utf8_lossy(&unit);
    if text.matches(PLACEHOLDER).count() != 1 {
        return Err(Error::msg(
            "soda extension installer image binding is ambiguous",
        ));
    }
    let body = text.replacen(PLACEHOLDER, &payload.image("extension").config, 1);
    files::owned_write(&path, body.as_bytes(), 0o644)
}

pub fn write_release_metadata_and_normalize(
    root: &str,
    payload: &model::Payload,
) -> Result<(), Error> {
    // The complete payload is the sole resolved-image/defaults owner. Leave only
    // a pointer in the earlier host-content build marker, not stale scope data.
    files::owned_write(
        &sys::join(&[root, "usr/share/soda/host-image/build.json"]),
        b"{\"Scope\":\"complete-local-payload\",\"ReleaseMetadata\":\"/usr/share/soda/release.json\"}\n",
        0o644,
    )?;
    let mut data = jsonio::to_indent(&payload.to_json());
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

/// Complete binds all callers to one v3 payload and ordinary Podman storage.
pub fn complete(
    source: &str,
    context: &str,
    archives: &str,
    payload: &model::Payload,
    production: Option<&dyn Production>,
) -> Result<(), Error> {
    validate_complete_payload(payload)?;
    let root = sys::join(&[context, "rootfs"]);
    write_complete_quadlets(source, &root, payload)?;
    verify_public_branding_and_motd(&root)?;
    write_factory_defaults(&root)?;
    configure_complete_systemd(source, &root)?;
    bind_extension_install_image(&root, payload)?;
    layout::stage_images(
        archives,
        &sys::join(&[&root, "usr/share/soda/images"]),
        payload,
        production,
    )?;
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
        assert!(complete("/src", "/ctx", "/arc", &payload, None).is_err());
    }

    #[test]
    fn oracle_extension_binding_refuses_ambiguity() {
        let dir = std::env::temp_dir().join(format!("sri-cb-{}", std::process::id()));
        let _ = fs::remove_dir_all(&dir);
        let unit = dir.join("usr/lib/systemd/system/soda-extension-install.service");
        fs::create_dir_all(unit.parent().unwrap()).unwrap();
        fs::write(
            &unit,
            b"Image=localhost/soda-extension:dev plus localhost/soda-extension:dev\n",
        )
        .unwrap();
        let payload = model::Payload::default();
        assert_eq!(
            bind_extension_install_image(dir.to_str().unwrap(), &payload)
                .unwrap_err()
                .0,
            "soda extension installer image binding is ambiguous"
        );
        let _ = fs::remove_dir_all(&dir);
    }
}

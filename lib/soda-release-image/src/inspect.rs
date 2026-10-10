//! `build_inspect.go`: sealed-context readback inspection.

use std::fs;

use crate::error::Error;
use crate::foreign::Production;
use crate::model;
use crate::sys;

/// Container run helper: `run(name, entrypoint, args...) -> trimmed stdout`.
pub type InspectRun<'a> = &'a dyn Fn(&str, &str, &[String]) -> Result<String, Error>;

pub fn inspect_complete_payload(
    _context: &str,
    out: &str,
    _id: &str,
    run: InspectRun<'_>,
) -> Result<model::Payload, Error> {
    let observed = run(
        "payload-inspect",
        "/usr/bin/cat",
        &[model::DELIVER_PATH.to_string()],
    )?;
    let expected = fs::read(sys::join(&[out, "payload.json"]))?;
    if observed != String::from_utf8_lossy(&expected).trim() {
        return Err(Error::msg("image payload metadata differs from candidate"));
    }
    let text = String::from_utf8_lossy(&expected).into_owned();
    // Plain (non-strict) decode, like the Go owner.
    let payload = model::Payload::parse(&text)?;
    payload.validate()?;
    Ok(payload)
}

pub fn inspect_complete_content(
    context: &str,
    _out: &str,
    payload: &model::Payload,
    run: InspectRun<'_>,
    production: &dyn Production,
) -> Result<(), Error> {
    let (files, _) = production.verify_content(
        payload,
        &sys::join(&[
            context,
            "rootfs",
            model::DELIVER_IMAGES_PATH.trim_start_matches('/'),
        ]),
    )?;
    let mut paths: Vec<String> = files.keys().cloned().collect();
    paths.sort();
    let installed: Vec<String> = paths
        .iter()
        .map(|name| format!("{}/{}", model::DELIVER_IMAGES_PATH, name))
        .collect();
    let sums = run("content-inspect", "/usr/bin/sha256sum", &installed)?;
    let lines: Vec<&str> = sums.split('\n').collect();
    if lines.len() != paths.len() {
        return Err(Error::msg("incomplete embedded content inventory"));
    }
    for (i, name) in paths.iter().enumerate() {
        if lines[i] != format!("{}  {}", files[name], installed[i]) {
            return Err(Error::msg("host content differs from payload"));
        }
    }
    Ok(())
}

pub fn inspect_complete_quadlets(
    out: &str,
    payload: &model::Payload,
    run: InspectRun<'_>,
) -> Result<(), Error> {
    let generated = run(
        "quadlet-inspect",
        "/usr/lib/systemd/system-generators/podman-system-generator",
        &["--dryrun".to_string()],
    )?;
    fs::write(
        sys::join(&[out, "generated-quadlets.txt"]),
        format!("{generated}\n").as_bytes(),
    )?;
    use std::os::unix::fs::PermissionsExt;
    fs::set_permissions(
        sys::join(&[out, "generated-quadlets.txt"]),
        fs::Permissions::from_mode(0o600),
    )?;
    if generated.contains("additionalimagestore")
        || !generated.contains("soda-image-import.service")
    {
        return Err(Error::msg(
            "native Quadlet generator lost ordinary Podman import ordering",
        ));
    }
    for name in ["forgejo", "dashboard", "proxy"] {
        if !generated.contains(&payload.image(name).config) {
            return Err(Error::msg("native Quadlet generator lost image binding"));
        }
    }
    Ok(())
}

pub fn inspect_complete_extension_unit(
    payload: &model::Payload,
    run: InspectRun<'_>,
) -> Result<(), Error> {
    let unit = run(
        "extension-unit-inspect",
        "/usr/bin/cat",
        &["/usr/lib/systemd/system/soda-extension-install.service".to_string()],
    )?;
    let extension_image = &payload.image("extension").config;
    if unit.contains("localhost/soda-extension:dev")
        || !unit.contains(&format!("--entrypoint=/bin/sh {extension_image} -ec '"))
    {
        return Err(Error::msg(
            "host image lost Soda extension installer binding",
        ));
    }
    Ok(())
}

pub fn inspect_complete_service_files(context: &str, run: InspectRun<'_>) -> Result<(), Error> {
    for path in [
        "/usr/share/containers/systemd/forgejo.container",
        "/usr/lib/systemd/system/soda-extension-install.service",
    ] {
        let want = sys::hash_file(&sys::join(&[
            context,
            "rootfs",
            path.trim_start_matches('/'),
        ]))?;
        let got = run(
            "service-hash-inspect",
            "/usr/bin/sha256sum",
            &[path.to_string()],
        )?;
        if got != format!("{want}  {path}") {
            return Err(Error::msg("host service differs from staged source"));
        }
    }
    Ok(())
}

pub fn inspect_complete_inventory(context: &str, run: InspectRun<'_>) -> Result<(), Error> {
    const PATH: &str = "/usr/share/soda/host-image/content.json";
    let want = fs::read(sys::join(&[
        context,
        "rootfs",
        PATH.trim_start_matches('/'),
    ]))?;
    let got = run("inventory-inspect", "/usr/bin/cat", &[PATH.to_string()])?;
    if format!("{got}\n") != String::from_utf8_lossy(&want) {
        return Err(Error::msg(
            "host content inventory differs from staged source",
        ));
    }
    Ok(())
}

pub fn inspect_complete(
    context: &str,
    out: &str,
    id: &str,
    production: &dyn Production,
) -> Result<(), Error> {
    let run = |name: &str, entry: &str, args: &[String]| -> Result<String, Error> {
        let mut command = vec![
            "--remote=false".to_string(),
            "run".to_string(),
            "--cidfile".to_string(),
            sys::join(&[out, &format!("{name}.cid")]),
            "--network=none".to_string(),
            "--read-only".to_string(),
            format!("--entrypoint={entry}"),
            id.to_string(),
        ];
        command.extend_from_slice(args);
        production.capture(context, "podman", &command)
    };
    let run_ref: InspectRun<'_> = &run;
    let payload = inspect_complete_payload(context, out, id, run_ref)?;
    inspect_complete_content(context, out, &payload, run_ref, production)?;
    let console_hash = sys::hash_file(&sys::join(&[out, "tools/soda-installer"]))?;
    let console = run_ref(
        "installer-inspect",
        "/usr/bin/sha256sum",
        &["/usr/libexec/soda/soda-install".to_string()],
    )?;
    if console != format!("{console_hash}  /usr/libexec/soda/soda-install") {
        return Err(Error::msg("image installer differs from prebuilt tool"));
    }
    inspect_complete_quadlets(out, &payload, run_ref)?;
    inspect_complete_extension_unit(&payload, run_ref)?;
    inspect_complete_service_files(context, run_ref)?;
    inspect_complete_inventory(context, run_ref)
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn extension_unit_binds_selected_image_from_actual_service_fixture() {
        const SERVICE: &str =
            include_str!("../../../system/host/services/soda-extension-install.service");
        const PLACEHOLDER: &str = "localhost/soda-extension:dev";
        let selected = "sha256:selected-extension";
        let unit = SERVICE.replace(PLACEHOLDER, selected);
        let run = |_: &str, _: &str, _: &[String]| Ok(unit.clone());
        let run_ref: InspectRun<'_> = &run;
        let mut payload = model::Payload::default();
        payload.images.push((
            "extension".to_string(),
            model::PayloadImage {
                config: selected.to_string(),
                ..Default::default()
            },
        ));

        assert!(inspect_complete_extension_unit(&payload, run_ref).is_ok());
        payload.images[0].1.config = "sha256:wrong-extension".to_string();
        assert_eq!(
            inspect_complete_extension_unit(&payload, run_ref)
                .unwrap_err()
                .0,
            "host image lost Soda extension installer binding"
        );
    }

    #[test]
    fn oracle_quadlet_generator_binding_checks() {
        // Oracle: Go inspectCompleteQuadlets error strings.
        let mut payload = model::Payload::default();
        payload.images.push((
            "forgejo".to_string(),
            model::PayloadImage {
                config: "sha256:forgejo".to_string(),
                ..Default::default()
            },
        ));
        let run = |_: &str, _: &str, _: &[String]| -> Result<String, Error> {
            Ok("additionalimagestore enabled".to_string())
        };
        let run_ref: InspectRun<'_> = &run;
        let dir = std::env::temp_dir().join(format!("sri-iq-{}", std::process::id()));
        let _ = fs::remove_dir_all(&dir);
        fs::create_dir_all(&dir).unwrap();
        assert_eq!(
            inspect_complete_quadlets(dir.to_str().unwrap(), &payload, run_ref)
                .unwrap_err()
                .0,
            "native Quadlet generator lost ordinary Podman import ordering"
        );
        let _ = fs::remove_dir_all(&dir);
    }
}

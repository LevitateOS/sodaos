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
    let payload = model::Payload::parse(text.as_bytes()).map_err(|e| Error::msg(e.to_string()))?;
    payload.validate().map_err(|e| Error::msg(e.to_string()))?;
    Ok(payload)
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
    for role in [
        model::Role::Forgejo,
        model::Role::Dashboard,
        model::Role::Proxy,
        model::Role::Postgres,
    ] {
        if !generated.contains(&role.binding(&payload.images).config) {
            return Err(Error::msg("native Quadlet generator lost image binding"));
        }
    }
    Ok(())
}

pub fn inspect_complete_service_files(context: &str, run: InspectRun<'_>) -> Result<(), Error> {
    // The required native package bootstraps from this trusted catalog;
    // the obsolete installer unit is gone.
    for path in [
        "/usr/share/containers/systemd/forgejo.container",
        "/usr/share/soda/extensions/catalog.json",
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
    inspect_complete_service_files(context, run_ref)?;
    inspect_complete_inventory(context, run_ref)
}

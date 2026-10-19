//! `build_host.go`: host image bake, observation, and verification.

use std::fs;

use crate::compression;
use crate::error::Error;
use crate::files;
use crate::foreign::Production;
use crate::inspect;
use crate::model;
use crate::packages;
use crate::prepare;
use crate::record;
use crate::sys;

pub fn read_host_image_id(iid: &str) -> Result<String, Error> {
    let data = fs::read(iid)?;
    let id = String::from_utf8_lossy(&data).trim().to_string();
    if !model::prefixed_digest(&id) {
        return Err(Error::msg("invalid built host image ID"));
    }
    Ok(id)
}

pub fn inspect_host_identity(
    context: &str,
    platform: &str,
    revision: &str,
    scope: &str,
    id: &str,
    production: &dyn Production,
) -> Result<(), Error> {
    let observed = production.capture(
        context,
        "podman",
        &[
            "--remote=false".to_string(),
            "image".to_string(),
            "inspect".to_string(),
            "--format".to_string(),
            "{{.Os}}/{{.Architecture}} {{index .Labels \"org.opencontainers.image.revision\"}} {{index .Labels \"io.soda.host-image.scope\"}}".to_string(),
            id.to_string(),
        ],
    )?;
    if observed != format!("linux/{platform} {revision} {scope}") {
        return Err(Error::msg("built host identity/platform mismatch"));
    }
    Ok(())
}

/// observeHostPackages reads the built host inventory as the
/// bill-of-materials. The install floats on bare names, so nothing precedes
/// the observation: the image's own inventory is validated for shape, and its
/// SHA256 becomes the payload fingerprint.
pub fn observe_host_packages(
    context: &str,
    out: &str,
    id: &str,
    production: &dyn Production,
) -> Result<Vec<u8>, Error> {
    // The thin host carries no application OCI bodies and no native
    // Tailnet: no shared image layout, no Tailscale RPM or repository, and
    // no Tailnet service enablement in either final filesystem.
    let script = "test \"$(stat -c %a /usr/libexec/soda/soda-host)\" = 755\ntest -L /usr/sbin\ntest \"$(readlink /usr/sbin)\" = bin\nfor name in grub2-install soda-setup soda-activate; do test -x /usr/sbin/$name; done\ntest -f /usr/lib/systemd/system/soda-project@.service\ntest -f /usr/share/containers/systemd/forgejo.container\nfor file in /etc/subuid /etc/subgid; do test \"$(cat \"$file\")\" = 'containers:1000000:268435456'; done\ntest ! -e /usr/local/libexec/soda/soda-host\ntest ! -e /etc/soda/host.json\ntest ! -e /etc/soda/dashboard.json\ntest ! -e /etc/zincati/config.d/90-soda-image.toml\ntest ! -e /usr/lib/bootc/bound-images.d/forgejo.container\ntest ! -e /usr/share/soda/images\ntest ! -e /etc/yum.repos.d/tailscale.repo\ntest ! -e /usr/lib/systemd/system/soda-extension-install.service\nrpm -q rpm-ostree zincati ignition cockpit-ostree >/dev/null\n! rpm -q tailscale >/dev/null 2>&1\ncat /usr/share/soda/host-image/packages.txt";
    let packages = production
        .capture(
            context,
            "podman",
            &[
                "--remote=false".to_string(),
                "run".to_string(),
                "--cidfile".to_string(),
                sys::join(&[out, "inspect.cid"]),
                "--network=none".to_string(),
                "--read-only".to_string(),
                "--cap-drop=all".to_string(),
                "--security-opt=no-new-privileges".to_string(),
                "--entrypoint=/bin/sh".to_string(),
                id.to_string(),
                "-ec".to_string(),
                script.to_string(),
            ],
        )
        .map_err(|e| {
            Error::msg(format!(
                "read-only host inspection failed; retain inspect.cid: {}",
                e.0
            ))
        })?;
    let lines: Vec<String> = packages.trim().split('\n').map(|s| s.to_string()).collect();
    packages::valid_rpm_inventory(&lines).map_err(|error| Error::msg(error.0))?;
    Ok(format!("{}\n", lines.join("\n")).into_bytes())
}

/// recordHostPackages files the observed inventory: one copy beside the
/// context for the record, one frozen evidence copy in the output.
pub fn record_host_packages(
    context: &str,
    out: &str,
    id: &str,
    production: &dyn Production,
) -> Result<String, Error> {
    let record = observe_host_packages(context, out, id, production)?;
    files::owned_write(&sys::join(&[context, "packages.recorded"]), &record, 0o644)?;
    sys::write_new(&sys::join(&[out, "packages.txt"]), &record, 0o600)?;
    Ok(files::hash_bytes(&record))
}

pub fn export_host_archive(
    context: &str,
    out: &str,
    arch: &str,
    revision: &str,
    prefix: &str,
    pinned: &str,
    id: &str,
    production: &dyn Production,
) -> Result<(), Error> {
    let archive = sys::join(&[out, "host.oci"]);
    let hex = id
        .strip_prefix("sha256:")
        .filter(|value| model::is_digest(value))
        .ok_or_else(|| Error::msg("invalid host export ID"))?;
    production.execute(
        context,
        "skopeo",
        &[
            "copy".into(),
            "--format".into(),
            "oci".into(),
            "--dest-compress-format".into(),
            "gzip".into(),
            "--dest-compress-level".into(),
            "1".into(),
            format!("containers-storage:@{hex}"),
            format!("oci-archive:{archive}"),
        ],
    )?;
    let host = production.inspect_oci(&archive, arch, revision)?;
    let pinned_digest = pinned.split('@').nth(1).unwrap_or("");
    if host.config != id || host.base_name != pinned || host.base_digest != pinned_digest {
        return Err(Error::msg("host export identity/base mismatch"));
    }
    let hash = sys::hash_file(&archive)?;
    record::record_candidate(
        out,
        production.native(),
        prefix,
        &host,
        &hash,
        production.forgejo_revision(),
        arch,
    )
}

/// buildHostImage bakes one host image from the context as staged. The first
/// pass stages no release metadata: its only job is package observation for
/// the bill-of-materials. The final pass builds the sealed context.
pub fn build_host_image(
    context: &str,
    out: &str,
    arch: &str,
    revision: &str,
    flatten: bool,
    base: &prepare::Base,
    production: &dyn Production,
) -> Result<String, Error> {
    let platform = model::oci_architecture(arch).unwrap_or("");
    let pinned = base.image(arch);
    let iid = sys::join(&[out, "host.iid"]);
    const SCOPE: &str = "thin-local-payload";
    let mut args = vec![
        "--remote=false".to_string(),
        "build".to_string(),
        "--pull=never".to_string(),
        "--unsetlabel=ostree.commit".to_string(),
        "--unsetlabel=ostree.final-diffid".to_string(),
        "--rm=true".to_string(),
        "--force-rm=false".to_string(),
        format!("--platform=linux/{platform}"),
        format!("--build-arg=BASE_IMAGE={pinned}"),
        format!("--build-arg=PAYLOAD_SCOPE={SCOPE}"),
        format!("--label=org.opencontainers.image.revision={revision}"),
        format!("--label=org.opencontainers.image.base.name={pinned}"),
        format!(
            "--label=org.opencontainers.image.base.digest={}",
            pinned.split('@').nth(1).unwrap_or("")
        ),
        format!(
            "--label=org.opencontainers.image.version={}.soda-{}",
            base.release,
            revision.get(..12).unwrap_or(revision)
        ),
        "--iidfile".to_string(),
        iid.clone(),
        "--file".to_string(),
        "Containerfile".to_string(),
        ".".to_string(),
    ];
    // Package observation needs a runnable image, not a flattened OCI export.
    // Flatten only the final sealed host consumed by native OSTree import.
    if flatten {
        args.insert(3, "--squash-all".to_string());
    }
    production.execute(context, "podman", &args)?;
    read_host_image_id(&iid)
}

/// verifyBuiltHost inspects the final sealed-context image: identity, a
/// package inventory that must match the sealed fingerprint, image config,
/// embedded payload/content/Quadlets, then OCI export.
pub fn verify_built_host(
    context: &str,
    out: &str,
    arch: &str,
    revision: &str,
    prefix: &str,
    id: &str,
    package_hash: &str,
    base: &prepare::Base,
    production: &dyn Production,
    phase: &mut dyn FnMut(&str) -> Result<(), Error>,
) -> Result<(), Error> {
    let platform = model::oci_architecture(arch).unwrap_or("");
    const SCOPE: &str = "thin-local-payload";
    phase("P6 / Verify native host identity and content")?;
    inspect_host_identity(context, platform, revision, SCOPE, id, production)?;
    let observed = observe_host_packages(context, out, id, production)?;
    if files::hash_bytes(&observed) != package_hash {
        return Err(Error::msg(
            "host inventory shifted during build; refusing mismatched fingerprint",
        ));
    }
    let image_config = production.capture(
        context,
        "podman",
        &[
            "--remote=false".to_string(),
            "run".to_string(),
            "--cidfile".to_string(),
            sys::join(&[out, "image-config-inspect.cid"]),
            "--network=none".to_string(),
            "--read-only".to_string(),
            "--cap-drop=all".to_string(),
            "--entrypoint=/bin/sh".to_string(),
            id.to_string(),
            "-euc".to_string(),
            crate::build_context::COREOS_IMAGE_CONFIG_SCRIPT.to_string(),
            "inspect-coreos-host".to_string(),
            base.release.clone(),
        ],
    )?;
    compression::record_image_config(context, out, &image_config)?;
    inspect::inspect_complete(context, out, id, production)?;
    let payload = model::load_payload(&sys::join(&[out, "payload.json"]))?;
    for installed in [
        soda_build_tools::project_support::DESTINATION,
        soda_build_tools::project_support::PAYLOAD_PATH,
    ] {
        crate::payload_stage::inspect_packaged_digest(
            context,
            id,
            &payload.project_support.sha256,
            installed,
            production,
        )?;
    }
    export_host_archive(
        context,
        out,
        arch,
        revision,
        prefix,
        &base.image(arch),
        id,
        production,
    )
}

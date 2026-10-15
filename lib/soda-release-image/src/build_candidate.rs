//! Single host-candidate execution: observe the floating package
//! inventory, seal the payload, bake and verify the final image.
//! Callback and progress callers share this one implementation.

use crate::error::Error;
use crate::foreign::Production;
use crate::host;
use crate::payload_stage;
use crate::prepare;
use crate::record;

/// buildHostCandidate observes the floating package inventory from an
/// observation build, seals the payload (release metadata included) into the
/// context, then bakes and verifies the final image. The sealed fingerprint
/// must reproduce from the final image; a shift fails the build.
pub fn build_host_candidate(
    snapshot: &str,
    context_dir: &str,
    artifacts: &str,
    arch: &str,
    revision: &str,
    prefix: &str,
    base: &prepare::Base,
    production: &dyn Production,
    inventory: &crate::sys::ShippingInventory,
    upgrade_from: &str,
    release_tag: &str,
    next: &mut dyn FnMut(&str) -> Result<(), Error>,
) -> Result<(), Error> {
    // The sealed Payload v4 inputs ride the admitted request and the
    // frozen production: an empty default fails staging closed with the
    // exact missing input named.
    let release = crate::native_package::ReleaseInputs {
        release_tag: release_tag.to_string(),
        fountain: production.frozen_fountain_release().unwrap_or_default(),
        proxy_requested: production.frozen_upstream_requested("proxy"),
        postgres_requested: production.frozen_upstream_requested("postgres"),
        tailscale: production.frozen_tailscale_binding().unwrap_or_default(),
        source_timestamp: production.source_timestamp().unwrap_or(0),
    };
    let mut payload = payload_stage::complete_candidate(
        snapshot,
        context_dir,
        artifacts,
        arch,
        revision,
        prefix,
        base,
        production,
        inventory.commands(),
        &release,
        next,
    )?;
    if !upgrade_from.is_empty() {
        payload.upgrade_from = vec![upgrade_from.to_owned()];
        soda_build_tools::project_support::validate_upgrade_sources(
            &payload.upgrade_from,
            &payload.id,
        )
        .map_err(Error::msg)?;
    }
    next("P5 / Build FCOS host candidate")?;
    let observation = host::build_host_image(
        context_dir,
        artifacts,
        arch,
        revision,
        false,
        base,
        production,
    )?;
    let package_hash =
        host::record_host_packages(context_dir, artifacts, &observation, production)?;
    payload.host_packages_sha256 = package_hash.clone();
    payload_stage::seal_candidate_payload(&payload, snapshot, context_dir, artifacts)?;
    record::write_content_inventory(context_dir, artifacts, production.native())?;
    prepare::inventory(context_dir)?;
    let id = host::build_host_image(
        context_dir,
        artifacts,
        arch,
        revision,
        true,
        base,
        production,
    )?;
    host::verify_built_host(
        context_dir,
        artifacts,
        arch,
        revision,
        prefix,
        &id,
        &package_hash,
        base,
        production,
        next,
    )
}

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
    next: &mut dyn FnMut(&str) -> Result<(), Error>,
) -> Result<(), Error> {
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
        next,
    )?;
    next("P5 / Build FCOS host candidate")?;
    let observation = host::build_host_image(
        context_dir,
        artifacts,
        arch,
        revision,
        prefix,
        base,
        production,
    )?;
    let package_hash =
        host::record_host_packages(context_dir, artifacts, &observation, production)?;
    payload.host_packages_sha256 = package_hash.clone();
    payload_stage::seal_candidate_payload(&payload, snapshot, context_dir, artifacts, production)?;
    record::write_content_inventory(context_dir, artifacts)?;
    prepare::inventory(context_dir)?;
    let id = host::build_host_image(
        context_dir,
        artifacts,
        arch,
        revision,
        prefix,
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

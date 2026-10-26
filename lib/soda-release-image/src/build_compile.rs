//! Shipping-tool compilation: Soda command discovery, Rust tool builds,
//! and the tools.json inventory record.
use std::fs;

use crate::error::Error;
use crate::foreign::Production;
use crate::sys;

const RUNTIME_COMMANDS: [(&str, &str); 10] = [
    ("soda-identity-compose", "soda-identity-compose"),
    ("soda-factory", "soda-factory"),
    ("soda-setup", "soda-setup"),
    ("soda-image-import", "soda-image-import"),
    ("soda-muse", "soda-muse"),
    ("soda-project-maintain", "soda-project-maintain"),
    ("soda-identity", "soda-identity"),
    ("soda-host", "soda-host"),
    ("soda-host", "soda-canlang-storage"),
    // The retired Go tailnet command is now a second soda-host binary.
    ("soda-host", "soda-forgejo-tailnet"),
];

pub fn compile_soda_commands(
    production: &dyn Production,
    snapshot: &str,
    context_dir: &str,
    inventory: &sys::ShippingInventory,
) -> Result<(), Error> {
    let names = inventory.commands().to_vec();
    // CORR-C-001: discovery follows actual owners. A cmd directory with
    // its own manifest is Rust-owned (post-cutover layout); anything else
    // stays on the Go recipe. Pre-cutover trees behave exactly as before.
    let mut rust_seen: Vec<(String, String)> = Vec::new();
    for name in &names {
        let dest = sys::join(&[context_dir, "rootfs/usr/libexec/soda", name]);
        let manifest = sys::join(&[snapshot, "cmd", name, "Cargo.toml"]);
        if sys::is_rust_command(&sys::join(&[snapshot, "cmd", name])) {
            let package = inventory.package_name_for_manifest(&manifest)?;
            inventory.require_bin_at(&manifest, name)?;
            rust_seen.push((package.to_string(), name.clone()));
            production.compile_rust(package, name, &dest)?;
        } else {
            production.compile(name, &format!("./cmd/{name}"), &dest)?;
        }
    }
    // Rust-ported commands no longer live under cmd/; the workspace owns them.
    // Names already produced from cmd/ above are skipped: each existing
    // binary is produced and staged exactly once.
    for (member, bin) in RUNTIME_COMMANDS {
        inventory.require_bin(member, bin)?;
        if !rust_seen
            .iter()
            .any(|seen| seen.0 == member && seen.1 == bin)
        {
            production.compile_rust(
                member,
                bin,
                &sys::join(&[context_dir, "rootfs/usr/libexec/soda", bin]),
            )?;
        }
    }
    Ok(())
}

pub fn record_tool_files(
    tools: &str,
    revision: &str,
    arch: &str,
    artifacts: &str,
) -> Result<(), Error> {
    let mut entries: Vec<String> = fs::read_dir(tools)?
        .filter_map(|entry| entry.ok())
        .map(|entry| entry.file_name().to_string_lossy().into_owned())
        .collect();
    entries.sort();
    let mut tool_files: Vec<(String, sys::File)> = Vec::new();
    for name in entries {
        let hash = sys::hash_file(&sys::join(&[tools, &name]))?;
        tool_files.push((
            name,
            sys::File {
                sha256: hash,
                mode: 0o755,
                ..sys::File::default()
            },
        ));
    }
    #[derive(serde::Serialize)]
    struct ToolsRecord<'a> {
        #[serde(rename = "Revision")]
        revision: &'a str,
        #[serde(rename = "Architecture")]
        architecture: &'a str,
        #[serde(rename = "Files")]
        files: crate::jsonio::SortedPairs<'a, sys::File>,
    }
    let record = ToolsRecord {
        revision,
        architecture: arch,
        files: crate::jsonio::SortedPairs(&tool_files),
    };
    let mut tool_data =
        serde_json::to_string_pretty(&record).expect("serialization to String cannot fail");
    tool_data.push('\n');
    sys::write_new(
        &sys::join(&[artifacts, "tools.json"]),
        tool_data.as_bytes(),
        0o600,
    )
}

/// rustTools maps ported Rust crates to the binaries they install in the
/// image. Each port PR extends this table and drops its appliance/bin
/// source; Prepare no longer stages these paths.
pub const RUST_TOOLS: [(&str, &str, &str); 11] = [
    (
        "soda-activate",
        "soda-activate",
        "rootfs/usr/bin/soda-activate",
    ),
    (
        "soda-forgejo-domain",
        "soda-forgejo-domain",
        "rootfs/usr/bin/soda-forgejo-domain",
    ),
    (
        "soda-forgejo-migrate",
        "soda-forgejo-migrate",
        "rootfs/usr/bin/soda-forgejo-migrate",
    ),
    (
        "soda-pg-maintenance",
        "soda-pg-backup",
        "rootfs/usr/bin/soda-pg-backup",
    ),
    (
        "soda-pg-maintenance",
        "soda-pg-restore",
        "rootfs/usr/bin/soda-pg-restore",
    ),
    (
        "soda-pg-maintenance",
        "soda-pg-init-roles",
        "rootfs/usr/bin/soda-pg-init-roles",
    ),
    (
        "soda-console-welcome",
        "soda-console-welcome",
        "rootfs/usr/libexec/soda/soda-console-welcome",
    ),
    (
        "soda-install",
        "soda-install",
        "rootfs/usr/libexec/soda/soda-install",
    ),
    (
        "soda-acceptance",
        "soda-host-probes",
        "rootfs/usr/libexec/soda/soda-host-probes",
    ),
    (
        "soda-project-terminal",
        "project-terminal",
        "rootfs/usr/libexec/soda/project-terminal",
    ),
    (
        "soda-provider-update",
        "soda-provider-update",
        "rootfs/usr/bin/soda-provider-update",
    ),
];

pub fn compile_rust_tools(
    production: &dyn Production,
    context_dir: &str,
    inventory: &sys::ShippingInventory,
) -> Result<(), Error> {
    for (member, bin, dest) in RUST_TOOLS {
        inventory.require_bin(member, bin)?;
        // Assets produces the shared helper once. Payload staging installs
        // those same bytes into the host and maintenance payload afterwards.
        if bin == "project-terminal" {
            continue;
        }
        let dest = sys::join(&[context_dir, dest]);
        fs::create_dir_all(sys::dir_name(&dest))?;
        production.compile_rust(member, bin, &dest)?;
    }
    Ok(())
}

fn admit_shipping_targets(
    production: &dyn Production,
    snapshot: &str,
) -> Result<sys::ShippingInventory, Error> {
    let manifest = sys::join(&[snapshot, "Cargo.toml"]);
    let metadata = production.capture(
        snapshot,
        "cargo",
        &[
            "metadata".to_string(),
            "--format-version".to_string(),
            "1".to_string(),
            "--no-deps".to_string(),
            "--locked".to_string(),
            "--offline".to_string(),
            "--manifest-path".to_string(),
            manifest,
        ],
    )?;
    let tool_members: Vec<&str> = RUST_TOOLS.iter().map(|(member, _, _)| *member).collect();
    let inventory = sys::parse_shipping_inventory(snapshot, &metadata)?
        .select_commands(snapshot, &tool_members)?;
    // Validate every selected target before the first compilation side effect.
    for name in inventory.commands() {
        let cmd_dir = sys::join(&[snapshot, "cmd", name]);
        if sys::is_rust_command(&cmd_dir) {
            inventory.require_bin_at(&sys::join(&[&cmd_dir, "Cargo.toml"]), name)?;
        }
    }
    for (member, bin) in RUNTIME_COMMANDS.into_iter().chain([
        ("soda-release-tools", "soda-artifacts"),
        ("soda-acceptance", "soda-acceptance"),
        ("soda-acceptance", "soda-acceptance-remote"),
    ]) {
        inventory.require_bin(member, bin)?;
    }
    for (member, bin, _) in RUST_TOOLS {
        inventory.require_bin(member, bin)?;
    }
    Ok(inventory)
}

pub fn compile_shipping_tools(
    production: &dyn Production,
    snapshot: &str,
    context_dir: &str,
    artifacts: &str,
    revision: &str,
    arch: &str,
) -> Result<sys::ShippingInventory, Error> {
    // Prepare no longer stages anything under usr/libexec/soda (the last
    // shell tool compiled out), so create it explicitly for the compilers.
    fs::create_dir_all(sys::join(&[context_dir, "rootfs/usr/libexec/soda"]))?;
    let inventory = admit_shipping_targets(production, snapshot)?;
    compile_soda_commands(production, snapshot, context_dir, &inventory)?;
    compile_rust_tools(production, context_dir, &inventory)?;
    let tools = sys::join(&[artifacts, "tools"]);
    sys::create_dir(&tools, 0o755)?;
    // D03-F2: soda-artifacts is Rust-ported; build the existing
    // release-tools binary into the unchanged tools output.
    production.compile_rust(
        "soda-release-tools",
        "soda-artifacts",
        &sys::join(&[&tools, "soda-artifacts"]),
    )?;
    // The acceptance driver is Rust-ported; the workspace owns it.
    production.compile_rust(
        "soda-acceptance",
        "soda-acceptance",
        &sys::join(&[&tools, "soda-acceptance"]),
    )?;
    // D03-F3: the remote companion ships alongside the driver, which
    // resolves it as a runtime sibling before remote invocation.
    production.compile_rust(
        "soda-acceptance",
        "soda-acceptance-remote",
        &sys::join(&[&tools, "soda-acceptance-remote"]),
    )?;
    // Personal Git sends this sibling payload from the outside Go probe driver.
    production.compile(
        "soda-installed-probes",
        "./tools/soda-installed-probes",
        &sys::join(&[&tools, "soda-installed-probes"]),
    )?;
    // The Rust console is compiled into the image above; link the tools copy
    // from it so media hashes the exact shipped bytes.
    fs::hard_link(
        sys::join(&[context_dir, "rootfs/usr/libexec/soda/soda-install"]),
        sys::join(&[&tools, "soda-installer"]),
    )
    .map_err(|e| Error::msg(e.to_string()))?;
    record_tool_files(&tools, revision, arch, artifacts)?;
    Ok(inventory)
}

//! Shipping-tool compilation: Soda command discovery, Rust tool builds,
//! and the tools.json inventory record.
use std::fs;

use crate::error::Error;
use crate::foreign::Production;
use crate::sys;

const RUNTIME_COMMANDS: [(&str, &str); 9] = [
    ("soda-identity-compose", "soda-identity-compose"),
    ("soda-factory", "soda-factory"),
    ("soda-setup", "soda-setup"),
    ("soda-image-import", "soda-image-import"),
    ("soda-muse", "soda-muse"),
    ("soda-muse-maintain", "soda-muse-maintain"),
    ("soda-identity", "soda-identity"),
    ("soda-host", "soda-host"),
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
pub const RUST_TOOLS: [(&str, &str, &str); 10] = [
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
];

pub fn compile_rust_tools(
    production: &dyn Production,
    context_dir: &str,
    inventory: &sys::ShippingInventory,
) -> Result<(), Error> {
    for (member, bin, dest) in RUST_TOOLS {
        inventory.require_bin(member, bin)?;
        let dest = sys::join(&[context_dir, dest]);
        fs::create_dir_all(sys::dir_name(&dest))?;
        production.compile_rust(member, bin, &dest)?;
    }
    Ok(())
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

#[cfg(test)]
mod tests {
    use super::*;

    use std::cell::RefCell;
    use std::collections::HashMap;

    use crate::model;
    use crate::sys::ShippingInventory;

    fn test_metadata(snapshot: &str) -> String {
        fs::create_dir_all(snapshot).unwrap();
        let mut packages: std::collections::BTreeMap<String, (String, Vec<String>)> =
            std::collections::BTreeMap::new();
        let mut add = |package: &str, bins: &[&str]| {
            packages
                .entry(package.to_string())
                .or_insert_with(|| (package.to_string(), Vec::new()))
                .1
                .extend(bins.iter().map(|bin| (*bin).to_string()));
        };
        for (member, bin) in [
            ("soda-identity-compose", "soda-identity-compose"),
            ("soda-factory", "soda-factory"),
            ("soda-setup", "soda-setup"),
            ("soda-image-import", "soda-image-import"),
            ("soda-muse", "soda-muse"),
            ("soda-muse-maintain", "soda-muse-maintain"),
            ("soda-identity", "soda-identity"),
            ("soda-host", "soda-host"),
            ("soda-host", "soda-forgejo-tailnet"),
            ("soda-release-tools", "soda-artifacts"),
            ("soda-acceptance", "soda-acceptance"),
            ("soda-acceptance", "soda-acceptance-remote"),
        ] {
            add(member, &[bin]);
        }
        for (member, bin, _) in RUST_TOOLS {
            add(member, &[bin]);
        }
        let cmd_root = sys::join(&[snapshot, "cmd"]);
        if let Ok(entries) = fs::read_dir(&cmd_root) {
            for entry in entries
                .flatten()
                .filter(|entry| entry.path().join("Cargo.toml").is_file())
            {
                let name = entry.file_name().to_string_lossy().into_owned();
                let (package, bins) = match name.as_str() {
                    "soda-project-terminal" => {
                        (name.clone(), vec!["project-terminal", "project-account"])
                    }
                    "soda-pg-maintenance" => (
                        name.clone(),
                        vec!["soda-pg-backup", "soda-pg-restore", "soda-pg-init-roles"],
                    ),
                    _ => (name.clone(), vec![name.as_str()]),
                };
                add(&package, &bins);
            }
        }
        drop(add);
        for (_, bins) in packages.values_mut() {
            bins.sort();
            bins.dedup();
        }
        let mut member_ids = Vec::new();
        let mut member_dirs = Vec::new();
        let mut metadata_packages = Vec::new();
        for (index, (name, (_, bins))) in packages.iter().enumerate() {
            let id = format!("opaque-member-{index}");
            member_ids.push(id.clone());
            let manifest = if std::path::Path::new(&sys::join(&[&cmd_root, name])).is_dir() {
                member_dirs.push(format!("cmd/{name}"));
                sys::join(&[&cmd_root, name, "Cargo.toml"])
            } else {
                member_dirs.push(format!("fixture/{name}"));
                sys::join(&[snapshot, &format!("fixture/{name}/Cargo.toml")])
            };
            fs::create_dir_all(sys::dir_name(&manifest)).unwrap();
            let mut cargo =
                format!("[package]\nname = {name:?}\nversion = \"0.1.0\"\nedition = \"2021\"\n");
            for bin in bins {
                cargo.push_str(&format!(
                    "[[bin]]\nname = {bin:?}\npath = \"src/{bin}.rs\"\n"
                ));
            }
            fs::write(&manifest, cargo).unwrap();
            for bin in bins {
                let source = sys::join(&[
                    sys::dir_name(&manifest).as_str(),
                    "src",
                    &format!("{bin}.rs"),
                ]);
                fs::create_dir_all(sys::dir_name(&source)).unwrap();
                fs::write(source, "fn main() {}\n").unwrap();
            }
            metadata_packages.push(serde_json::json!({
                "id": id, "name": name, "manifest_path": std::fs::canonicalize(manifest).unwrap(),
                "features": {"default": []},
                "targets": bins.iter().map(|bin| serde_json::json!({"name": bin, "kind": ["bin"], "required-features": []})).collect::<Vec<_>>()
            }));
        }
        let workspace = format!(
            "[workspace]\nmembers = [{}]\nresolver = \"2\"\n",
            member_dirs
                .iter()
                .map(|member| format!("{member:?}"))
                .collect::<Vec<_>>()
                .join(", ")
        );
        fs::write(sys::join(&[snapshot, "Cargo.toml"]), workspace).unwrap();
        let metadata = serde_json::json!({
            "workspace_root": std::fs::canonicalize(snapshot).unwrap(),
            "workspace_members": member_ids, "packages": metadata_packages
        })
        .to_string();
        metadata
    }

    fn test_inventory(snapshot: &str) -> ShippingInventory {
        let tool_members: Vec<&str> = RUST_TOOLS.iter().map(|(member, _, _)| *member).collect();
        sys::parse_shipping_inventory(snapshot, &test_metadata(snapshot))
            .unwrap()
            .select_commands(snapshot, &tool_members)
            .unwrap()
    }
    struct Recorder {
        go: RefCell<Vec<(String, String, String)>>,
        rust: RefCell<Vec<(String, String, String)>>,
    }
    impl Production for Recorder {
        fn source(&self) -> &str {
            ""
        }
        fn forgejo_source(&self) -> &str {
            ""
        }
        fn forgejo_revision(&self) -> &str {
            ""
        }
        fn native(&self) -> &str {
            ""
        }
        fn out(&self) -> &str {
            ""
        }
        fn arch(&self) -> &str {
            "x86_64"
        }
        fn revision(&self) -> &str {
            ""
        }
        fn live_inputs(&self) -> &str {
            ""
        }
        fn execute(&self, _: &str, _: &str, _: &[String]) -> Result<(), Error> {
            panic!("command run")
        }
        fn capture(&self, _: &str, _: &str, args: &[String]) -> Result<String, Error> {
            let manifest = args.last().expect("metadata manifest argument");
            let snapshot = std::path::Path::new(manifest)
                .parent()
                .unwrap()
                .to_string_lossy();
            Ok(test_metadata(&snapshot))
        }
        fn next(&self, _: &str) -> Result<(), Error> {
            Ok(())
        }
        fn resolve_inputs(&mut self) -> Result<(), Error> {
            Ok(())
        }
        fn dependencies(&self) -> Result<(), Error> {
            Ok(())
        }
        fn compile(&self, name: &str, pkg: &str, dest: &str) -> Result<(), Error> {
            self.go
                .borrow_mut()
                .push((name.into(), pkg.into(), dest.into()));
            fs::write(dest, b"stub-go").map_err(Error::from)?;
            Ok(())
        }
        fn compile_rust(&self, crate_name: &str, bin: &str, dest: &str) -> Result<(), Error> {
            self.rust
                .borrow_mut()
                .push((crate_name.into(), bin.into(), dest.into()));
            fs::write(dest, b"stub-rust").map_err(Error::from)?;
            Ok(())
        }
        fn stage_fork_binary(&self, _: &str) -> Result<(), Error> {
            Ok(())
        }
        fn assets(&self, _: &str, _: &str) -> Result<(), Error> {
            Ok(())
        }
        fn images(&self, _: &str) -> Result<HashMap<String, model::ProducedImage>, Error> {
            Ok(Default::default())
        }
        fn inspect_oci(&self, _: &str, _: &str, _: &str) -> Result<model::Image, Error> {
            Ok(Default::default())
        }
        fn verify_content(
            &self,
            _: &model::Payload,
            _: &str,
        ) -> Result<(HashMap<String, String>, u64), Error> {
            Ok(Default::default())
        }
        fn resolve_core_os(&self) -> Result<model::ResolvedCoreOS, Error> {
            Ok(Default::default())
        }
        fn read_live_inputs(&self, _: &str) -> Result<model::LiveInputs, Error> {
            Ok(Default::default())
        }
        fn check_native(&self, _: &str) -> Result<(), Error> {
            Ok(())
        }
        fn sign_media(
            &self,
            _: &model::Trust,
            _: &model::Permit,
            _: &str,
            _: &str,
            _: &str,
            _: &model::SecretFiles,
            _: &str,
        ) -> Result<(), Error> {
            Ok(())
        }
        fn verify_copy(
            &self,
            _: &model::Trust,
            _: &str,
            _: &str,
            _: &str,
            _: &str,
        ) -> Result<(), Error> {
            Ok(())
        }
        fn write_document(
            &self,
            _: &str,
            _: &crate::foreign::PackagingInputs,
        ) -> Result<String, Error> {
            Ok(String::new())
        }
    }

    #[test]
    fn shipping_tools_use_rust_recipes_and_ship_remote_companion() {
        // D03-F2/D03-F3: the actual production recipe must build
        // soda-artifacts from the Rust release-tools package (never the
        // deleted ./tools/soda-artifacts Go path) and must ship the
        // soda-acceptance-remote companion alongside the driver.
        let dir = std::env::temp_dir().join(format!("sri-ship-{}", std::process::id()));
        let _ = fs::remove_dir_all(&dir);
        let snapshot = dir.join("snap");
        fs::create_dir_all(snapshot.join("cmd/soda-fake")).unwrap();
        let context = dir.join("ctx").to_str().unwrap().to_string();
        let artifacts = dir.join("art").to_str().unwrap().to_string();
        fs::create_dir_all(&artifacts).unwrap();
        let recorder = Recorder {
            go: RefCell::new(Vec::new()),
            rust: RefCell::new(Vec::new()),
        };
        compile_shipping_tools(
            &recorder,
            snapshot.to_str().unwrap(),
            &context,
            &artifacts,
            &"r".repeat(40),
            "x86_64",
        )
        .unwrap();
        let tools = format!("{artifacts}/tools");
        // D03-F2: artifacts via Rust, never the deleted Go path.
        assert!(recorder.rust.borrow().iter().any(|(c, b, d)| {
            c == "soda-release-tools"
                && b == "soda-artifacts"
                && d == &format!("{tools}/soda-artifacts")
        }));
        assert!(!recorder
            .go
            .borrow()
            .iter()
            .any(|(_, p, _)| p == "./tools/soda-artifacts"));
        // D03-F3: remote companion recorded alongside the driver.
        assert!(recorder.rust.borrow().iter().any(|(c, b, d)| {
            c == "soda-acceptance"
                && b == "soda-acceptance-remote"
                && d == &format!("{tools}/soda-acceptance-remote")
        }));
        let inventory = fs::read(format!("{artifacts}/tools.json")).unwrap();
        let record: serde_json::Value = serde_json::from_slice(&inventory).unwrap();
        let files = record.get("Files").unwrap();
        assert!(files.get("soda-acceptance-remote").is_some());
        assert!(files.get("soda-artifacts").is_some());
        let _ = fs::remove_dir_all(&dir);
    }

    #[test]
    fn soda_commands_follow_actual_go_rust_owners() {
        // CORR-C-001: discovery follows actual owners. A cmd directory
        // with its own manifest is Rust-owned (post-cutover layout) and
        // takes the Rust recipe exactly once even when the fixed ported
        // list names it too; anything else stays on the Go recipe.
        let dir = std::env::temp_dir().join(format!("sri-mixed-{}", std::process::id()));
        let _ = fs::remove_dir_all(&dir);
        let snapshot = dir.join("snap");
        fs::create_dir_all(snapshot.join("cmd/soda-fakego")).unwrap();
        fs::create_dir_all(snapshot.join("cmd/soda-identity")).unwrap();
        fs::write(
            snapshot.join("cmd/soda-identity/Cargo.toml"),
            b"[package]\nname = \"soda-identity\"\n",
        )
        .unwrap();
        fs::create_dir_all(dir.join("ctx/rootfs/usr/libexec/soda")).unwrap();
        let context = dir.join("ctx").to_str().unwrap().to_string();
        let recorder = Recorder {
            go: RefCell::new(Vec::new()),
            rust: RefCell::new(Vec::new()),
        };
        let inventory = test_inventory(snapshot.to_str().unwrap());
        compile_soda_commands(&recorder, snapshot.to_str().unwrap(), &context, &inventory).unwrap();
        // Go-owned dir takes the Go recipe with its cmd path.
        assert!(recorder.go.borrow().iter().any(|(n, p, d)| {
            n == "soda-fakego"
                && p == "./cmd/soda-fakego"
                && d == &format!("{context}/rootfs/usr/libexec/soda/soda-fakego")
        }));
        // Rust-owned dir takes the Rust recipe exactly once despite the
        // fixed ported list also naming soda-identity.
        let identity_rust = recorder
            .rust
            .borrow()
            .iter()
            .filter(|(c, b, _)| c == "soda-identity" && b == "soda-identity")
            .count();
        assert_eq!(identity_rust, 1);
        assert!(!recorder
            .go
            .borrow()
            .iter()
            .any(|(_, p, _)| p == "./cmd/soda-identity"));
        let _ = fs::remove_dir_all(&dir);
    }

    #[test]
    fn n07t3_tailnet_compiles_from_host_package_to_libexec() {
        // N07-T3: package != bin selector. soda-forgejo-tailnet has no cmd
        // manifest of its own; it compiles from the soda-host package to the
        // same libexec destination the Go recipe used, exactly once, while
        // the existing identity pairs keep their (member, member) shape.
        let dir = std::env::temp_dir().join(format!("sri-n07t3a-{}", std::process::id()));
        let _ = fs::remove_dir_all(&dir);
        let snapshot = dir.join("snap");
        fs::create_dir_all(snapshot.join("cmd/soda-fakego")).unwrap();
        fs::create_dir_all(dir.join("ctx/rootfs/usr/libexec/soda")).unwrap();
        let context = dir.join("ctx").to_str().unwrap().to_string();
        let recorder = Recorder {
            go: RefCell::new(Vec::new()),
            rust: RefCell::new(Vec::new()),
        };
        let inventory = test_inventory(snapshot.to_str().unwrap());
        compile_soda_commands(&recorder, snapshot.to_str().unwrap(), &context, &inventory).unwrap();
        let tailnet = recorder
            .rust
            .borrow()
            .iter()
            .filter(|(c, b, d)| {
                c == "soda-host"
                    && b == "soda-forgejo-tailnet"
                    && d == &format!("{context}/rootfs/usr/libexec/soda/soda-forgejo-tailnet")
            })
            .count();
        assert_eq!(tailnet, 1);
        assert!(recorder.rust.borrow().iter().any(|(c, b, d)| {
            c == "soda-host"
                && b == "soda-host"
                && d == &format!("{context}/rootfs/usr/libexec/soda/soda-host")
        }));
        let _ = fs::remove_dir_all(&dir);
    }

    #[test]
    fn cmd1_compile_skips_terminal_identity_and_tools_ship_it_once() {
        // CORR-C-004-AMEND-1 (CODEX-A01-CMD-1): compile must never reference
        // the bogus (soda-project-terminal, soda-project-terminal) identity —
        // that binary does not exist post-fold. RUST_TOOLS retains the real
        // tuple and produces project-terminal exactly once.
        let dir = std::env::temp_dir().join(format!("sri-cmd1b-{}", std::process::id()));
        let _ = fs::remove_dir_all(&dir);
        let snapshot = dir.join("snap");
        fs::create_dir_all(snapshot.join("cmd/soda-fakego")).unwrap();
        let folded = snapshot.join("cmd/soda-project-terminal");
        fs::create_dir_all(&folded).unwrap();
        fs::write(
            folded.join("Cargo.toml"),
            b"[package]\nname = \"soda-project-terminal\"\n[[bin]]\nname = \"project-terminal\"\n[[bin]]\nname = \"project-account\"\n",
        )
        .unwrap();
        fs::create_dir_all(dir.join("ctx/rootfs/usr/libexec/soda")).unwrap();
        let context = dir.join("ctx").to_str().unwrap().to_string();
        let recorder = Recorder {
            go: RefCell::new(Vec::new()),
            rust: RefCell::new(Vec::new()),
        };
        let inventory = test_inventory(snapshot.to_str().unwrap());
        compile_soda_commands(&recorder, snapshot.to_str().unwrap(), &context, &inventory).unwrap();
        assert!(!recorder
            .rust
            .borrow()
            .iter()
            .any(|(c, b, _)| c == "soda-project-terminal" && b == "soda-project-terminal"));
        assert!(!recorder
            .go
            .borrow()
            .iter()
            .any(|(_, p, _)| p == "./cmd/soda-project-terminal"));
        // The retained RUST_TOOLS tuple ships project-terminal exactly once.
        let tuples = RUST_TOOLS
            .iter()
            .filter(|(m, b, d)| {
                *m == "soda-project-terminal"
                    && *b == "project-terminal"
                    && *d == "rootfs/usr/libexec/soda/project-terminal"
            })
            .count();
        assert_eq!(tuples, 1);
        let tools = Recorder {
            go: RefCell::new(Vec::new()),
            rust: RefCell::new(Vec::new()),
        };
        let inventory = test_inventory(snapshot.to_str().unwrap());
        compile_rust_tools(&tools, &context, &inventory).unwrap();
        let shipped = tools
            .rust
            .borrow()
            .iter()
            .filter(|(c, b, _)| c == "soda-project-terminal" && b == "project-terminal")
            .count();
        assert_eq!(shipped, 1);
        let _ = fs::remove_dir_all(&dir);
    }

    #[test]
    fn cr02_compile_skips_pg_identity_and_tools_ship_all_three() {
        // CODEX-CR02-001: compile must never reference the bogus
        // (soda-pg-maintenance, soda-pg-maintenance) identity — that binary
        // does not exist. RUST_TOOLS retains the three real tuples and ships
        // backup/restore/init-roles exactly once each.
        let dir = std::env::temp_dir().join(format!("sri-cr02b-{}", std::process::id()));
        let _ = fs::remove_dir_all(&dir);
        let snapshot = dir.join("snap");
        fs::create_dir_all(snapshot.join("cmd/soda-fakego")).unwrap();
        let pg = snapshot.join("cmd/soda-pg-maintenance");
        fs::create_dir_all(&pg).unwrap();
        fs::write(
            pg.join("Cargo.toml"),
            b"[package]\nname = \"soda-pg-maintenance\"\n[[bin]]\nname = \"soda-pg-backup\"\n[[bin]]\nname = \"soda-pg-restore\"\n[[bin]]\nname = \"soda-pg-init-roles\"\n",
        )
        .unwrap();
        fs::create_dir_all(dir.join("ctx/rootfs/usr/libexec/soda")).unwrap();
        let context = dir.join("ctx").to_str().unwrap().to_string();
        let recorder = Recorder {
            go: RefCell::new(Vec::new()),
            rust: RefCell::new(Vec::new()),
        };
        let inventory = test_inventory(snapshot.to_str().unwrap());
        compile_soda_commands(&recorder, snapshot.to_str().unwrap(), &context, &inventory).unwrap();
        assert!(!recorder
            .rust
            .borrow()
            .iter()
            .any(|(c, b, _)| c == "soda-pg-maintenance" && b == "soda-pg-maintenance"));
        assert!(!recorder
            .go
            .borrow()
            .iter()
            .any(|(_, p, _)| p == "./cmd/soda-pg-maintenance"));
        // The three retained RUST_TOOLS tuples keep their exact destinations.
        let mut tuples: Vec<(&str, &str)> = RUST_TOOLS
            .iter()
            .filter(|(m, _, _)| *m == "soda-pg-maintenance")
            .map(|(_, b, d)| (*b, *d))
            .collect();
        tuples.sort();
        assert_eq!(
            tuples,
            vec![
                ("soda-pg-backup", "rootfs/usr/bin/soda-pg-backup"),
                ("soda-pg-init-roles", "rootfs/usr/bin/soda-pg-init-roles"),
                ("soda-pg-restore", "rootfs/usr/bin/soda-pg-restore"),
            ]
        );
        let tools = Recorder {
            go: RefCell::new(Vec::new()),
            rust: RefCell::new(Vec::new()),
        };
        let inventory = test_inventory(snapshot.to_str().unwrap());
        compile_rust_tools(&tools, &context, &inventory).unwrap();
        for bin in ["soda-pg-backup", "soda-pg-restore", "soda-pg-init-roles"] {
            let shipped = tools
                .rust
                .borrow()
                .iter()
                .filter(|(c, b, _)| c == "soda-pg-maintenance" && b == &bin)
                .count();
            assert_eq!(shipped, 1);
        }
        let _ = fs::remove_dir_all(&dir);
    }

    #[test]
    fn toolsown_compile_preserves_member_destinations() {
        // Moved member crates (activate/domain/migrate/console) must not be
        // produced through cmd discovery; RUST_TOOLS ships each exactly once
        // to its established destination.
        let dir = std::env::temp_dir().join(format!("sri-toolsown-b-{}", std::process::id()));
        let _ = fs::remove_dir_all(&dir);
        let snapshot = dir.join("snap");
        fs::create_dir_all(snapshot.join("cmd/soda-fakego")).unwrap();
        for member in [
            "soda-activate",
            "soda-forgejo-domain",
            "soda-forgejo-migrate",
            "soda-console-welcome",
        ] {
            let cmd = snapshot.join("cmd").join(member);
            fs::create_dir_all(&cmd).unwrap();
            fs::write(
                cmd.join("Cargo.toml"),
                format!("[package]\nname = \"{member}\"\n[[bin]]\nname = \"{member}\"\n"),
            )
            .unwrap();
        }
        fs::create_dir_all(dir.join("ctx/rootfs/usr/libexec/soda")).unwrap();
        let context = dir.join("ctx").to_str().unwrap().to_string();
        let recorder = Recorder {
            go: RefCell::new(Vec::new()),
            rust: RefCell::new(Vec::new()),
        };
        let inventory = test_inventory(snapshot.to_str().unwrap());
        compile_soda_commands(&recorder, snapshot.to_str().unwrap(), &context, &inventory).unwrap();
        for member in [
            "soda-activate",
            "soda-forgejo-domain",
            "soda-forgejo-migrate",
            "soda-console-welcome",
        ] {
            assert!(!recorder.rust.borrow().iter().any(|(c, _, _)| c == member));
        }
        let tools = Recorder {
            go: RefCell::new(Vec::new()),
            rust: RefCell::new(Vec::new()),
        };
        let inventory = test_inventory(snapshot.to_str().unwrap());
        compile_rust_tools(&tools, &context, &inventory).unwrap();
        for (member, bin, dest) in [
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
                "soda-console-welcome",
                "soda-console-welcome",
                "rootfs/usr/libexec/soda/soda-console-welcome",
            ),
        ] {
            let shipped: Vec<_> = tools
                .rust
                .borrow()
                .iter()
                .filter(|(c, b, _)| c == &member && b == &bin)
                .map(|(_, _, d)| d.clone())
                .collect();
            assert_eq!(shipped, vec![format!("{context}/{dest}")]);
        }
        let _ = fs::remove_dir_all(&dir);
    }
}

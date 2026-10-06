//! Shipping-tool compilation: Soda command discovery, Rust tool builds,
//! and the tools.json inventory record.
use std::fs;

use crate::error::Error;
use crate::foreign::Production;
use crate::jsonio;
use crate::sys;

pub fn compile_soda_commands(
    production: &dyn Production,
    snapshot: &str,
    context_dir: &str,
) -> Result<(), Error> {
    let names = sys::soda_commands(snapshot)?;
    // CORR-C-001: discovery follows actual owners. A cmd directory with
    // its own manifest is Rust-owned (post-cutover layout); anything else
    // stays on the Go recipe. Pre-cutover trees behave exactly as before.
    let mut rust_seen: Vec<String> = Vec::new();
    for name in &names {
        let dest = sys::join(&[context_dir, "rootfs/usr/libexec/soda", name]);
        if sys::is_rust_command(&sys::join(&[snapshot, "cmd", name])) {
            rust_seen.push(name.clone());
            production.compile_rust(name, name, &dest)?;
        } else {
            production.compile(name, &format!("./cmd/{name}"), &dest)?;
        }
    }
    // Rust-ported commands no longer live under cmd/; the workspace owns them.
    // Names already produced from cmd/ above are skipped: each existing
    // binary is produced and staged exactly once.
    for name in [
        "soda-identity-compose",
        "soda-factory",
        "soda-setup",
        "soda-image-import",
        "soda-muse",
        "soda-muse-maintain",
        "soda-identity",
        "soda-host",
    ] {
        if !rust_seen.iter().any(|seen| seen.as_str() == name) {
            production.compile_rust(
                name,
                name,
                &sys::join(&[context_dir, "rootfs/usr/libexec/soda", name]),
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
    let record = soda_json::JsonValue::Object(vec![
        (
            "Revision".to_string(),
            soda_json::JsonValue::Str(revision.to_string()),
        ),
        (
            "Architecture".to_string(),
            soda_json::JsonValue::Str(arch.to_string()),
        ),
        (
            "Files".to_string(),
            soda_json::JsonValue::Object(
                tool_files
                    .into_iter()
                    .map(|(k, v)| (k, v.to_json()))
                    .collect(),
            ),
        ),
    ]);
    let mut tool_data = jsonio::to_indent(&record);
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

pub fn compile_rust_tools(production: &dyn Production, context_dir: &str) -> Result<(), Error> {
    for (member, bin, dest) in RUST_TOOLS {
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
) -> Result<(), Error> {
    // Prepare no longer stages anything under usr/libexec/soda (the last
    // shell tool compiled out), so create it explicitly for the compilers.
    fs::create_dir_all(sys::join(&[context_dir, "rootfs/usr/libexec/soda"]))?;
    compile_soda_commands(production, snapshot, context_dir)?;
    compile_rust_tools(production, context_dir)?;
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
    record_tool_files(&tools, revision, arch, artifacts)
}

#[cfg(test)]
mod tests {
    use super::*;

    use std::cell::RefCell;
    use std::collections::HashMap;

    use crate::model;
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
        fn capture(&self, _: &str, _: &str, _: &[String]) -> Result<String, Error> {
            panic!("command run")
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
        fn write_document(&self, _: &str, _: &soda_json::JsonValue) -> Result<String, Error> {
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
        let record = soda_json::JsonValue::parse(std::str::from_utf8(&inventory).unwrap()).unwrap();
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
        compile_soda_commands(&recorder, snapshot.to_str().unwrap(), &context).unwrap();
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
        compile_soda_commands(&recorder, snapshot.to_str().unwrap(), &context).unwrap();
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
        compile_rust_tools(&tools, &context).unwrap();
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
        compile_soda_commands(&recorder, snapshot.to_str().unwrap(), &context).unwrap();
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
        compile_rust_tools(&tools, &context).unwrap();
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
}

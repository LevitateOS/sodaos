use super::*;
use std::collections::HashMap;

#[test]
fn freezing_image_config_preserves_unknown_values_and_replaces_selected_fields() {
    let opaque = format!(
        "{}{{\"extreme\":1e400,\"wide\":18446744073709551616}}{}",
        "{\"deep\":".repeat(140),
        "}".repeat(140)
    );
    let input = format!(
        r#"{{"z":1e2,"container-imgref":"old","unknown":{{"number":1e2,"negative_zero":-0,"items":[1.0]}},"opaque":{opaque},"bootc-install-to-fs":true,"a":-0}}"#
    );
    let got = freeze_image_config(&input, "ghcr.io/example/sodaos", "").unwrap();
    assert!(got.ends_with('\n'));
    let key_positions = [
        "\"a\":",
        "\"bootc-install-to-fs\":",
        "\"container-imgref\":",
        "\"opaque\":",
        "\"unknown\":",
        "\"z\":",
    ]
    .map(|key| got.find(key).unwrap());
    assert!(key_positions.windows(2).all(|pair| pair[0] < pair[1]));
    let parsed: BTreeMap<String, Box<RawValue>> = serde_json::from_str(got.trim_end()).unwrap();
    assert_eq!(parsed.len(), 6);
    assert_eq!(parsed["a"].get(), "-0");
    assert_eq!(parsed["z"].get(), "1e2");
    assert_eq!(parsed["opaque"].get(), opaque.as_str());
    assert_eq!(
        parsed["unknown"].get(),
        r#"{"number":1e2,"negative_zero":-0,"items":[1.0]}"#
    );
    assert_eq!(
        serde_json::from_str::<String>(parsed["container-imgref"].get()).unwrap(),
        "ostree-image-signed:docker://ghcr.io/example/sodaos-host:candidate"
    );
    assert_eq!(parsed["bootc-install-to-fs"].get(), "false");
}

#[test]
fn oracle_link_prepared_assets_runs_no_commands() {
    // Oracle: Go TestLinkPreparedAssetsRunsNoCommands.
    struct Stub {
        source: String,
        native: String,
    }
    impl Production for Stub {
        fn source(&self) -> &str {
            &self.source
        }
        fn forgejo_source(&self) -> &str {
            ""
        }
        fn forgejo_revision(&self) -> &str {
            ""
        }
        fn native(&self) -> &str {
            &self.native
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
        fn compile(&self, _: &str, _: &str, _: &str) -> Result<(), Error> {
            Ok(())
        }
        fn compile_rust(&self, _: &str, _: &str, _: &str) -> Result<(), Error> {
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
        fn resolve_core_os(
            &self,
        ) -> Result<soda_build_tools::reader::stream::ResolvedCoreOS, Error> {
            Ok(Default::default())
        }
        fn read_live_inputs(
            &self,
            _: &str,
        ) -> Result<soda_build_tools::reader::stream::LiveInputs, Error> {
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
    let dir = std::env::temp_dir().join(format!("sri-lpa-{}", std::process::id()));
    let _ = fs::remove_dir_all(&dir);
    let stub = Stub {
        source: dir.join("src").to_str().unwrap().to_string(),
        native: dir.join("native").to_str().unwrap().to_string(),
    };
    link_prepared_assets(&stub).unwrap();
    assert!(fs::symlink_metadata(dir.join("src/.artifacts/forgejo-js"))
        .unwrap()
        .file_type()
        .is_symlink());
    let _ = fs::remove_dir_all(&dir);
}

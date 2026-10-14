use super::*;

#[test]
fn frozen_arm_coreos_base_requires_matching_target() {
    use soda_build_tools::reader::stream::CoreOSImage;
    let image = CoreOSImage {
        url: "https://builds.test/aarch64/live.iso".to_string(),
        signature_url: "https://builds.test/aarch64/live.iso.sig".to_string(),
        sha256: "a".repeat(64),
        uncompressed_sha256: "b".repeat(64),
    };
    let pinned = format!("quay.io/fedora/fedora-coreos@sha256:{}", "c".repeat(64));
    let coreos = ResolvedCoreOS {
        release: "44.20260901.1.0".to_string(),
        metadata_url: "https://builds.test/prod/streams/stable/builds/44.20260901.1.0/release.json"
            .to_string(),
        container: [("aarch64".to_string(), pinned.clone())].into(),
        iso: [("aarch64".to_string(), image.clone())].into(),
        qemu: [(
            "aarch64".to_string(),
            CoreOSImage {
                url: "https://builds.test/aarch64/image.qcow2.xz".to_string(),
                signature_url: "https://builds.test/aarch64/image.qcow2.xz.sig".to_string(),
                ..image
            },
        )]
        .into(),
    };
    assert_eq!(
        base_from_resolved("aarch64", &coreos)
            .unwrap()
            .image("aarch64"),
        pinned
    );
    assert_eq!(
        base_from_resolved("x86_64", &coreos).unwrap_err().0,
        "resolved CoreOS architecture differs from requested target"
    );
}

#[test]
fn host_context_delivers_avatar_dependency_notices() {
    let source = std::path::Path::new(env!("CARGO_MANIFEST_DIR")).join("../..");
    let output = std::env::temp_dir().join(format!(
        "soda-release-avatar-notices-{}",
        std::process::id()
    ));
    fs::create_dir(&output).expect("fresh output");
    PreparedWriter {
        source: source.to_string_lossy().into_owned(),
        out: output.to_string_lossy().into_owned(),
    }
    .stage_rootfs_files()
    .expect("actual host staging");
    let notice = output.join("rootfs/usr/share/licenses/soda/avatar-dependencies.txt");
    assert_eq!(
        fs::read(&notice).expect("installed dependency notices"),
        fs::read(source.join("system/licenses/avatar-dependencies.txt"))
            .expect("retained dependency notices")
    );
    assert_eq!(fs::metadata(&notice).unwrap().mode() & 0o777, 0o644);
    fs::remove_dir_all(output).expect("remove output");
}

#[test]
fn base_context_stages_dashboard_policy_sources() {
    let root = std::env::temp_dir().join(format!(
        "soda-release-prepare-policy-{}",
        std::process::id()
    ));
    let source = root.join("source");
    let output = root.join("output");
    fs::create_dir_all(source.join("system/host/selinux")).expect("source tree");
    fs::write(source.join("system/host/Containerfile"), b"FROM base\n").expect("Containerfile");
    fs::write(
        source.join("system/host/selinux/soda_dashboard.te"),
        b"policy_module(soda_dashboard, 1.0)\n",
    )
    .expect("policy source");
    fs::write(
        source.join("system/host/selinux/soda_dashboard.fc"),
        b"/run/soda/host\\.sock -s system_u:object_r:soda_host_socket_t:s0\n",
    )
    .expect("file contexts");

    PreparedWriter {
        source: source.to_string_lossy().into_owned(),
        out: output.to_string_lossy().into_owned(),
    }
    .write_base_files(&[], "https://repo.invalid/repo")
    .expect("base context");

    assert_eq!(
        fs::read(output.join("selinux/soda_dashboard.te")).expect("staged policy"),
        b"policy_module(soda_dashboard, 1.0)\n"
    );
    assert_eq!(
        fs::read(output.join("selinux/soda_dashboard.fc")).expect("staged file contexts"),
        b"/run/soda/host\\.sock -s system_u:object_r:soda_host_socket_t:s0\n"
    );
    fs::remove_dir_all(root).expect("remove test tree");
}

#[test]
fn oracle_base_inputs_require_exact_revision() {
    // Oracle: Go finishBaseInputs revision gate.
    struct Stub;
    impl Production for Stub {
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
            Ok(())
        }
        fn capture(&self, _: &str, _: &str, _: &[String]) -> Result<String, Error> {
            Ok(String::new())
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
        fn images(
            &self,
            _: &str,
        ) -> Result<std::collections::HashMap<String, model::ProducedImage>, Error> {
            Ok(Default::default())
        }
        fn inspect_oci(&self, _: &str, _: &str, _: &str) -> Result<model::Image, Error> {
            Ok(Default::default())
        }
        fn verify_content(
            &self,
            _: &model::Payload,
            _: &str,
        ) -> Result<(std::collections::HashMap<String, String>, u64), Error> {
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
    let stub = Stub;
    assert_eq!(
        load_base(&stub, "not-an-arch").unwrap_err().0,
        "expected x86_64 or aarch64"
    );
    assert_eq!(
        finish_base_inputs("/nonexistent", "short", Base::default())
            .unwrap_err()
            .0,
        "exact source revision required"
    );
}

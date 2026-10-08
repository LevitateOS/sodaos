use super::*;

#[test]
fn oracle_extension_asset_names() {
    assert!(extension::safe_extension_asset_name(r"app.js"), "H-asset-0");
    assert!(
        extension::safe_extension_asset_name(r"css/main.css"),
        "H-asset-1"
    );
    assert!(
        !(extension::safe_extension_asset_name(r"../x.js")),
        "H-asset-2"
    );
    assert!(
        !(extension::safe_extension_asset_name(r"/abs.js")),
        "H-asset-3"
    );
    assert!(
        !(extension::safe_extension_asset_name(r"a/../b.js")),
        "H-asset-4"
    );
    assert!(
        !(extension::safe_extension_asset_name(r"b\c.js")),
        "H-asset-5"
    );
    assert!(!(extension::safe_extension_asset_name(r"")), "H-asset-6");
}

#[test]
fn oracle_qualify_reason() {
    check_ok(
        "I-reason-0",
        r"Ym9vbQ==",
        recall::qualify_reason(r"  boom  ").as_bytes(),
    );
    check_ok(
        "I-reason-1",
        r"",
        recall::qualify_reason(r"COMMAND podman").as_bytes(),
    );
    check_ok(
        "I-reason-2",
        r"",
        recall::qualify_reason(r"$ echo x").as_bytes(),
    );
    check_ok("I-reason-3", r"", recall::qualify_reason(r"").as_bytes());
    check_ok(
        "I-reason-4",
        r"",
        recall::qualify_reason(r"  $ x").as_bytes(),
    );
}

struct Stub;
impl soda_release_image::foreign::Production for Stub {
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
    fn execute(
        &self,
        _: &str,
        _: &str,
        _: &[String],
    ) -> Result<(), soda_release_image::error::Error> {
        unreachable!()
    }
    fn capture(
        &self,
        _: &str,
        _: &str,
        _: &[String],
    ) -> Result<String, soda_release_image::error::Error> {
        unreachable!()
    }
    fn resolve_inputs(&mut self) -> Result<(), soda_release_image::error::Error> {
        unreachable!()
    }
    fn dependencies(&self) -> Result<(), soda_release_image::error::Error> {
        unreachable!()
    }
    fn compile(&self, _: &str, _: &str, _: &str) -> Result<(), soda_release_image::error::Error> {
        unreachable!()
    }
    fn compile_rust(
        &self,
        _: &str,
        _: &str,
        _: &str,
    ) -> Result<(), soda_release_image::error::Error> {
        unreachable!()
    }
    fn stage_fork_binary(&self, _: &str) -> Result<(), soda_release_image::error::Error> {
        unreachable!()
    }
    fn assets(&self, _: &str, _: &str) -> Result<(), soda_release_image::error::Error> {
        unreachable!()
    }
    fn images(
        &self,
        _: &str,
    ) -> Result<
        std::collections::HashMap<String, model::ProducedImage>,
        soda_release_image::error::Error,
    > {
        unreachable!()
    }
    fn inspect_oci(
        &self,
        _: &str,
        _: &str,
        _: &str,
    ) -> Result<model::Image, soda_release_image::error::Error> {
        unreachable!()
    }
    fn verify_content(
        &self,
        _: &model::Payload,
        _: &str,
    ) -> Result<(std::collections::HashMap<String, String>, u64), soda_release_image::error::Error>
    {
        unreachable!()
    }
    fn resolve_core_os(
        &self,
    ) -> Result<soda_build_tools::reader::stream::ResolvedCoreOS, soda_release_image::error::Error>
    {
        unreachable!()
    }
    fn read_live_inputs(
        &self,
        _: &str,
    ) -> Result<soda_build_tools::reader::stream::LiveInputs, soda_release_image::error::Error>
    {
        unreachable!()
    }
    fn check_native(&self, _: &str) -> Result<(), soda_release_image::error::Error> {
        unreachable!()
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
    ) -> Result<(), soda_release_image::error::Error> {
        unreachable!()
    }
    fn verify_copy(
        &self,
        _: &model::Trust,
        _: &str,
        _: &str,
        _: &str,
        _: &str,
    ) -> Result<(), soda_release_image::error::Error> {
        unreachable!()
    }
    fn write_document(
        &self,
        _: &str,
        _: &soda_release_image::foreign::PackagingInputs,
    ) -> Result<String, soda_release_image::error::Error> {
        unreachable!()
    }
}

#[test]
fn oracle_stage_layout() {
    let stub = Stub;
    let p3 = model::Payload {
        format: 3,
        ..model::Payload::default()
    };
    let p2 = model::Payload {
        format: 2,
        ..model::Payload::default()
    };
    assert!(
        layout::valid_stage_layout("/a", "/b", &p3, Some(&stub)),
        "J-layout-ok"
    );
    assert!(
        !(layout::valid_stage_layout("/a", "/b", &p3, None)),
        "J-layout-nil"
    );
    assert!(
        !(layout::valid_stage_layout("/a", "/b", &p2, Some(&stub))),
        "J-layout-fmt"
    );
    assert!(
        !(layout::valid_stage_layout("a", "/b", &p3, Some(&stub))),
        "J-layout-rel"
    );
    assert!(
        !(layout::valid_stage_layout("/a:x", "/b", &p3, Some(&stub))),
        "J-layout-colon"
    );
}

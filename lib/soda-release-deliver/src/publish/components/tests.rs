use super::*;
use std::cell::RefCell;

struct Observations {
    raw: Vec<u8>,
    digest: String,
    architecture: String,
    calls: RefCell<Vec<Vec<String>>>,
}
impl Registry for Observations {
    fn run(&self, args: &[String]) -> Result<Vec<u8>, Error> {
        self.calls.borrow_mut().push(args.to_vec());
        assert_eq!(
            args[0], "inspect",
            "readback must never publish an upstream image"
        );
        assert!(args.contains(&"--no-creds".into()));
        if args.contains(&"--raw".into()) {
            return Ok(self.raw.clone());
        }
        Ok(serde_json::to_vec(&serde_json::json!({"Digest":self.digest,"Architecture":self.architecture,"Os":"linux"})).unwrap())
    }
}
fn fixture() -> (Observations, Image) {
    let config = format!("sha256:{}", "a".repeat(64));
    let raw = serde_json::to_vec(&serde_json::json!({"config":{"digest":config}})).unwrap();
    let digest = hash_bytes(&raw);
    (
        Observations {
            raw,
            digest: digest.clone(),
            architecture: "amd64".into(),
            calls: RefCell::new(vec![]),
        },
        Image {
            reference: format!("ghcr.io/example/input@{digest}"),
            manifest: digest,
            config,
            ..Image::default()
        },
    )
}
#[test]
fn anonymous_component_readback_binds_manifest_config_and_native_platform() {
    let (mut registry, mut image) = fixture();
    observe_binding(&registry, &image).unwrap();
    image.config = format!("sha256:{}", "b".repeat(64));
    assert!(observe_binding(&registry, &image).is_err());
    let (_, correct) = fixture();
    image = correct;
    registry.architecture = "arm64".into();
    assert!(observe_binding(&registry, &image).is_err());
    registry.architecture = "amd64".into();
    registry.raw.push(b' ');
    assert!(observe_binding(&registry, &image).is_err());
}
#[test]
fn complete_readback_includes_six_roles_fountain_and_disabled_tailnet_without_writes() {
    let (registry, image) = fixture();
    let mut payload = Payload::default();
    payload.release_tag = "v0.1.0".into();
    for role in Role::ALL {
        *role.binding_mut(&mut payload.images) = image.clone();
    }
    payload.fountain.image = image.reference.clone();
    payload.fountain.image_config = image.config.clone();
    payload.optional_extensions.tailnet.images.tailscale = image.clone();
    observe_payload(&registry, &payload).unwrap();
    let calls = registry.calls.borrow();
    assert_eq!(calls.len(), 24); // Eight immutable inputs plus four immutable Soda tags.
    assert_eq!(
        calls
            .iter()
            .filter(|args| args.last().unwrap().ends_with(":v0.1.0"))
            .count(),
        8
    );
}

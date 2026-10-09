use super::*;

use crate::json;
use crate::terminal;

// ---------- daemon config goldens ----------

const GOLDENS: &str = "tests/data/iconfig";
const PROJECT_IMAGE: &str =
    "sha256:1111111111111111111111111111111111111111111111111111111111111111";
const TAILNET_IMAGE: &str =
    "sha256:2222222222222222222222222222222222222222222222222222222222222222";

fn golden(name: &str) -> String {
    format!("{GOLDENS}/{name}")
}

#[test]
fn golden_valid_configs() {
    let c = load_config(&golden("valid-full.json"), "").unwrap();
    assert_eq!(
        c.muse_sha256,
        "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
    );
    assert_eq!(c.muse_version, "1.2.3");
    assert_eq!(c.muse_socket, "/run/soda-muse/launch.sock");
    assert_eq!(c.identity_socket, "/run/soda/identity.sock");
    assert_eq!(c.codex_harness, "/usr/libexec/soda/codex");
    assert_eq!(
        c.codex_harness_sha256,
        "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
    );
    assert_eq!(c.codex_harness_version, "0.153.4");
    assert!(c.tailnet_management);
    assert_eq!(
        c.tailnet_image,
        "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"
    );
    assert_eq!(
        c.image,
        "dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd"
    );
    assert_eq!(c.network, "soda-net");
    assert_eq!(c.subnet, "10.89.0.0/24");
    assert_eq!(c.bridge, "soda-br0");
    let c = load_config(&golden("valid-minimal.json"), "").unwrap();
    assert_eq!(c.image, "img");
    assert_eq!(c.network, "soda-net");
    assert_eq!(c.subnet, "10.89.0.0/24");
    assert_eq!(c.bridge, "soda-br0");
    assert!(!c.tailnet_management);
    assert!(c.muse_sha256.is_empty());
    assert!(c.codex_harness.is_empty());
}

#[test]
fn golden_decode_errors() {
    // The config reader rejects unknown fields and malformed typed values.
    let err = load_config(&golden("unknown-field.json"), "").unwrap_err();
    assert!(!err.is_empty());
    assert!(load_config(&golden("bad-type-bool.json"), "").is_err());
    assert!(load_config(&golden("bad-type-string.json"), "").is_err());
    // Empty input is rejected at the config read boundary.
    assert!(!load_config(&golden("empty.json"), "")
        .unwrap_err()
        .is_empty());
    // Missing files fail at the read, carrying the path.
    let err = load_config(&golden("does-not-exist.json"), "").unwrap_err();
    assert!(
        err.starts_with("read tests/data/iconfig/does-not-exist.json: "),
        "{err}"
    );
}

#[test]
fn golden_go_framing_parity() {
    // Go's single `Decode` ignores trailing data: accepted, like Go.
    let c = load_config(&golden("trailing.json"), "").unwrap();
    assert_eq!(c.image, "img");
    // Duplicate fields resolve last-wins, like `encoding/json`.
    let c = load_config(&golden("duplicates.json"), "").unwrap();
    assert_eq!(c.image, "img");
    // A literal `null` decodes as a no-op and reaches validation, like Go
    // (whose subnet error differs only in stdlib text).
    let err = load_config(&golden("null.json"), "").unwrap_err();
    assert_eq!(err, net::parse_prefix("").err().unwrap());
}

#[test]
fn golden_release_overlay() {
    let release = golden("release.json");
    let c = load_config(&golden("overlay-base.json"), &release).unwrap();
    assert_eq!(c.image, PROJECT_IMAGE);
    assert_eq!(c.tailnet_image, TAILNET_IMAGE);
    assert!(c.tailnet_management);
    // Without management the companion image stays empty.
    let c = load_config(&golden("overlay-nomgmt.json"), &release).unwrap();
    assert_eq!(c.image, PROJECT_IMAGE);
    assert!(c.tailnet_image.is_empty());
    // Saved selections conflicting with the release fail loudly.
    for name in ["conflict-image.json", "conflict-tailnet.json"] {
        assert_eq!(
            load_config(&golden(name), &release).unwrap_err(),
            "saved image selection conflicts with appliance release; explicit migration required",
            "{name}"
        );
    }
    // Unreadable or invalid payloads fail alike.
    for release in [golden("bad-release.json"), golden("does-not-exist.json")] {
        assert_eq!(
            load_config(&golden("overlay-base.json"), &release).unwrap_err(),
            "immutable appliance image defaults unavailable",
            "{release}"
        );
    }
    // An empty release path disables the overlay entirely.
    let c = load_config(&golden("valid-minimal.json"), "").unwrap();
    assert_eq!(c.image, "img");
}

#[test]
fn host_payload_architecture_matches_native_policy() {
    assert!(host_payload_architecture_is_native("x86_64"));
    assert!(!host_payload_architecture_is_native(""));
    assert!(!host_payload_architecture_is_native("X86_64"));
}

#[test]
fn golden_validators() {
    const MUSE: &str = "explicit muse socket, broker socket and release digest required";
    const IDENTITY: &str = "explicit identity runtime socket and verified harness required";
    const STAGED: &str = "invalid staged harness version";
    const TAILNET: &str = "invalid immutable Tailnet companion configuration";
    const NATIVE: &str = "invalid native runtime configuration";
    // (golden, expected error); every message matches Go byte for byte
    // except the subnet case, which carries this crate's prefix message.
    let cases = [
        ("muse-relative-socket.json", MUSE),
        ("muse-bad-digest.json", MUSE),
        ("muse-no-version.json", MUSE),
        ("muse-bad-base.json", MUSE),
        ("muse-relative-identity.json", MUSE),
        ("identity-relative-harness.json", IDENTITY),
        ("identity-bad-sha.json", IDENTITY),
        ("identity-bad-version.json", STAGED),
        ("tailnet-no-mgmt.json", TAILNET),
        ("tailnet-bad-ref.json", TAILNET),
        ("bad-network.json", NATIVE),
        ("bad-bridge.json", NATIVE),
        ("empty-image.json", NATIVE),
        ("dash-image.json", NATIVE),
    ];
    for (name, expected) in cases {
        assert_eq!(
            load_config(&golden(name), "").unwrap_err(),
            expected,
            "{name}"
        );
    }
    assert_eq!(
        load_config(&golden("bad-subnet.json"), "").unwrap_err(),
        net::parse_prefix("nope").err().unwrap()
    );
    // Empty selectors skip their whole stage, like Go.
    load_config(&golden("muse-skipped.json"), "").unwrap();
    load_config(&golden("identity-skipped.json"), "").unwrap();
}

#[test]
fn golden_validation_order() {
    // Each golden fails two stages; the earlier stage's error must win,
    // proving muse < identity < subnet < tailnet < network order.
    assert_eq!(
        load_config(&golden("order-muse-before-subnet.json"), "").unwrap_err(),
        "explicit muse socket, broker socket and release digest required"
    );
    assert_eq!(
        load_config(&golden("order-identity-before-tailnet.json"), "").unwrap_err(),
        "explicit identity runtime socket and verified harness required"
    );
    assert_eq!(
        load_config(&golden("order-subnet-before-network.json"), "").unwrap_err(),
        net::parse_prefix("nope").err().unwrap()
    );
    assert_eq!(
        load_config(&golden("order-tailnet-before-network.json"), "").unwrap_err(),
        "invalid immutable Tailnet companion configuration"
    );
}

#[test]
fn crate_helpers_match_go_validators() {
    // The validators host configuration reuses decide exactly
    // like their Go regexes on boundary inputs.
    assert!(domain::valid_image_ref(&format!(
        "sha256:{}",
        "c".repeat(64)
    )));
    assert!(domain::valid_image_ref(&"c".repeat(64)));
    assert!(!domain::valid_image_ref("sha256:xyz"));
    assert!(domain::valid_container_id(&"a".repeat(64)));
    assert!(!domain::valid_container_id(&"A".repeat(64)));
    assert!(domain::valid_login("soda-net"));
    assert!(!domain::valid_login("9bad"));
    assert!(!domain::valid_login(&"a".repeat(32)));
    assert!(factory::valid_harness_version("0.153.4"));
    assert!(!factory::valid_harness_version("!!!"));
    assert!(net::parse_prefix("10.89.0.0/24").is_ok());
    assert!(net::parse_prefix("nope").is_err());
    assert!(net::parse_prefix("10.0.0.1/24").is_ok());
    // `,string` decoding behind the connection projection.
    assert_eq!(
        terminal::parse_string_i64("9007199254740993"),
        Some(9007199254740993)
    );
    assert_eq!(terminal::parse_string_i64(""), None);
    // JSON quoting behind request encoding.
    assert_eq!(json::quote("a&b"), "\"a\\u0026b\"");
}

#[test]
fn configured_muse_socket_requires_clean_absolute_spelling() {
    let mut c = Config::default();
    c.muse_sha256 = "a".repeat(64);
    c.muse_version = "1".into();
    c.identity_socket = "/run/soda/identity.sock".into();
    for path in [
        "/run/soda-muse/launch.sock/",
        "/run//soda-muse/launch.sock",
        "/run/soda-muse/./launch.sock",
        "/run/other/../soda-muse/launch.sock",
    ] {
        c.muse_socket = path.into();
        assert!(validate_muse_runtime(&c).is_err(), "{path}");
    }
    c.muse_socket = "/run/soda-muse/launch.sock".into();
    assert!(validate_muse_runtime(&c).is_ok());
}

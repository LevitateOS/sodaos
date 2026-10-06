use crate::test_support::{hex_string, PROJECT};

use super::{decode_observation, validate_observation, Observation};

#[test]
fn observation_decode_and_validate() {
    let id = hex_string('a', 64);
    let body = format!(
        "{{\"id\":\"{id}\",\"project\":\"{PROJECT}\",\"owner\":\"42\",\"pid\":123,\"running\":true}}"
    );
    let o = decode_observation(body.as_bytes()).unwrap();
    assert_eq!(o.pid, 123);
    assert!(o.running);
    validate_observation(&o, PROJECT).unwrap();
    // Identity failures.
    let mut foreign = Observation {
        id: id.clone(),
        project: String::from("pabcdef0123456789abcdef01"),
        owner: String::from("42"),
        pid: 1,
        running: true,
    };
    assert_eq!(
        validate_observation(&foreign, PROJECT).unwrap_err(),
        "project container identity differs"
    );
    foreign.project = PROJECT.to_string();
    foreign.id = hex_string('A', 64);
    assert_eq!(
        validate_observation(&foreign, PROJECT).unwrap_err(),
        "project container identity differs"
    );
    foreign.id = hex_string('a', 63);
    assert_eq!(
        validate_observation(&foreign, PROJECT).unwrap_err(),
        "project container identity differs"
    );
    // Owner failures and edge acceptances.
    foreign.id = id.clone();
    for owner in ["0", "-5", "abc", "", "42 ", "99999999999999999999999"] {
        foreign.owner = owner.to_string();
        assert_eq!(
            validate_observation(&foreign, PROJECT).unwrap_err(),
            "project owner label is invalid",
            "owner {owner:?}"
        );
    }
    foreign.owner = String::from("+42");
    validate_observation(&foreign, PROJECT).unwrap();
    foreign.owner = String::from("42");
    foreign.pid = -5;
    validate_observation(&foreign, PROJECT).unwrap();
    // Decode failures collapse to the observation message.
    for bad in [
        format!("{{\"id\":\"{id}\",\"project\":\"{PROJECT}\",\"owner\":\"42\",\"pid\":1.5,\"running\":true}}"),
        format!("{{\"id\":\"{id}\",\"project\":\"{PROJECT}\",\"owner\":\"42\",\"pid\":\"12\",\"running\":true}}"),
        format!("{{\"id\":\"{id}\",\"project\":\"{PROJECT}\",\"owner\":\"42\",\"pid\":1,\"running\":1}}"),
        format!("{{\"id\":\"{id}\",\"project\":\"{PROJECT}\",\"owner\":\"42\",\"pid\":1,\"running\":\"true\"}}"),
        format!("{{\"id\":\"{id}\",\"id\":\"{id}\",\"project\":\"{PROJECT}\",\"owner\":\"42\",\"pid\":1,\"running\":true}}"),
        format!("{{\"id\":\"{id}\",\"project\":\"{PROJECT}\",\"owner\":\"42\",\"pid\":1,\"running\":true,\"bogus\":1}}"),
        format!("{{\"id\":\"{id}\",\"project\":\"{PROJECT}\",\"owner\":\"42\",\"pid\":1,\"running\":true,}}"),
        format!("{{\"id\":\"{id}\",\"project\":\"{PROJECT}\",\"owner\":\"42\",\"pid\":1,\"running\":true}} {{}}"),
        String::from("[]"),
        String::from("null"),
        String::from("{oops"),
    ] {
        assert_eq!(
            decode_observation(bad.as_bytes()).unwrap_err(),
            "invalid project observation",
            "input {bad:?}"
        );
    }
    assert_eq!(
        decode_observation(&vec![b'{'; 2 << 20]).unwrap_err(),
        "invalid project observation"
    );
    assert_eq!(
        decode_observation(b"{\"id\":\"\xff\"}").unwrap_err(),
        "invalid project observation"
    );
    // Nulls decode to zero values, then fail validation.
    let nulls = decode_observation(
        format!("{{\"id\":null,\"project\":\"{PROJECT}\",\"owner\":null,\"pid\":null,\"running\":null}}")
            .as_bytes(),
    )
    .unwrap();
    assert_eq!(
        validate_observation(&nulls, PROJECT).unwrap_err(),
        "project container identity differs"
    );
    // Exact keys win over earlier case variants in sorted order.
    let other = hex_string('b', 64);
    let o = decode_observation(
        format!("{{\"ID\":\"{other}\",\"id\":\"{id}\",\"project\":\"{PROJECT}\",\"owner\":\"42\",\"pid\":1,\"running\":true}}")
            .as_bytes(),
    )
    .unwrap();
    assert_eq!(o.id, id);
    validate_observation(&o, PROJECT).unwrap();
}

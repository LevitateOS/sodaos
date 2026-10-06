use super::*;

#[test]
fn hostnames_follow_the_script_regexes() {
    assert!(is_appliance_hostname("factory-01.lab.example"));
    assert!(is_appliance_hostname("a"));
    assert!(!is_appliance_hostname(""));
    assert!(!is_appliance_hostname("-lead.example"));
    assert!(!is_appliance_hostname("trail-.example"));
    assert!(!is_appliance_hostname("UPPER.example"));
    assert!(!is_appliance_hostname("under_score.example"));
    assert!(!is_appliance_hostname(".leading.example"));
    assert!(!is_appliance_hostname("trailing.example."));
    assert!(!is_appliance_hostname("double..dot"));
    assert!(!is_appliance_hostname(&"a".repeat(64)));
    assert!(is_appliance_hostname(&("a".repeat(63) + ".example")));
    assert!(!is_appliance_hostname(&"a".repeat(254)));
    assert!(is_fixture_hostname("soda-native-fixture"));
    assert!(is_fixture_hostname("soda-native-a"));
    assert!(!is_fixture_hostname("soda-native-"));
    assert!(!is_fixture_hostname("soda-native-UPPER"));
    assert!(!is_fixture_hostname("other-fixture"));
    assert!(!is_fixture_hostname(
        "soda-native-a-very-long-tail-that-keeps-going-past-forty-two"
    ));
}

#[test]
fn dump_matches_json_indent_two() {
    let value = JsonValue::parse(
        "{\"b\": [1, {\"x\": true}, [], {}], \"a\": \"q\\\"\\n\\u0001~/\\u007f\\u00e9😀\", \"e\": {}, \"n\": null}",
    )
    .expect("parse");
    assert_eq!(
        dump_python(&value),
        "{\n  \"b\": [\n    1,\n    {\n      \"x\": true\n    },\n    [],\n    {}\n  ],\n  \"a\": \"q\\\"\\n\\u0001~/\\u007f\\u00e9\\ud83d\\ude00\",\n  \"e\": {},\n  \"n\": null\n}"
    );
}

#[test]
fn dump_keeps_number_literals_and_key_order() {
    let value = JsonValue::parse("{\"mode\": 420, \"ratio\": 1e2}").expect("parse");
    assert_eq!(
        dump_python(&value),
        "{\n  \"mode\": 420,\n  \"ratio\": 1e2\n}"
    );
}

use super::*;
use crate::json_emit::{marshal_indent, Emit};

#[test]
fn oracle_marshal_indent_vectors() {
    // Oracle: Go json.MarshalIndent outputs (no trailing newline).
    let value = Emit::Object(vec![
        (
            "URL".to_string(),
            Emit::Str("https://x.test/a.iso".to_string()),
        ),
        ("n".to_string(), Emit::Int(-3)),
        ("ok".to_string(), Emit::Bool(true)),
        (
            "list".to_string(),
            Emit::List(vec![Emit::Str("a<b".to_string()), Emit::UInt(7)]),
        ),
        ("empty".to_string(), Emit::List(vec![])),
        (
            "nested".to_string(),
            Emit::Object(vec![("k".to_string(), Emit::Str("v&v".to_string()))]),
        ),
    ]);
    assert_eq!(
        marshal_indent(&value),
        "{\n  \"URL\": \"https://x.test/a.iso\",\n  \"n\": -3,\n  \"ok\": true,\n  \"list\": [\n    \"a\\u003cb\",\n    7\n  ],\n  \"empty\": [],\n  \"nested\": {\n    \"k\": \"v\\u0026v\"\n  }\n}"
    );
    assert_eq!(marshal_indent(&Emit::Object(vec![])), "{}");
}

#[test]
fn strict_rejects_unknown_fields_go_style() {
    let value = JsonValue::parse("{\"CompilerImage\":\"x\",\"bogus\":1}").unwrap();
    let mut binder = Strict::bind(&value, "build.ForgejoToolchain").unwrap();
    binder.string("CompilerImage").unwrap();
    assert_eq!(
        binder.finish().unwrap_err(),
        "json: unknown field \"bogus\""
    );
}

#[test]
fn strict_folds_names_and_skips_null() {
    let value = JsonValue::parse("{\"compilerimage\":null,\"APKPackages\":[\"a\"]}").unwrap();
    let mut binder = Strict::bind(&value, "build.ForgejoToolchain").unwrap();
    assert_eq!(binder.string("CompilerImage").unwrap(), "");
    assert_eq!(binder.string_list("APKPackages").unwrap(), vec!["a"]);
    binder.finish().unwrap();
}

#[test]
fn lenient_lookup_folds_and_last_wins() {
    let value = JsonValue::parse("{\"mediatype\":\"a\",\"mediaType\":\"b\",\"size\":7}").unwrap();
    let fields = Fields::of(&value).unwrap();
    assert_eq!(fields.string("mediaType").unwrap(), "b");
    assert_eq!(fields.int("size").unwrap(), 7);
    assert_eq!(fields.string("missing").unwrap(), "");
    assert!(fields.int("mediaType").is_err());
}

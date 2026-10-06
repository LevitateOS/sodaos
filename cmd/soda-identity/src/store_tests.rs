use super::*;

#[test]
fn bind_rewrites_placeholders_outside_literals() {
    assert_eq!(bind("SELECT ?, ?",), "SELECT $1, $2");
    assert_eq!(
        bind("WHERE state='ready' AND id=?"),
        "WHERE state='ready' AND id=$1"
    );
    assert_eq!(bind("VALUES('it''s ?')"), "VALUES('it''s ?')");
    // The revoked-credential CASE keeps its quoted literal intact.
    let query = "credential=CASE WHEN ?='revoked' THEN '\\x'::bytea ELSE credential END WHERE id=?";
    assert_eq!(
        bind(query),
        "credential=CASE WHEN $1='revoked' THEN '\\x'::bytea ELSE credential END WHERE id=$2"
    );
}

#[test]
fn bytea_params_use_hex_text_form() {
    assert_eq!(Param::bytea(&[0xde, 0xad]).encode().unwrap(), b"\\xdead");
}

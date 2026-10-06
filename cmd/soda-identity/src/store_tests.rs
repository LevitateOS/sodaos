use super::*;

#[test]
fn bytea_params_use_hex_text_form() {
    assert_eq!(Param::bytea(&[0xde, 0xad]).encode().unwrap(), b"\\xdead");
}

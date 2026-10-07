use super::*;

#[test]
fn parameters_keep_postgres_value_types() {
    assert!(matches!(Param::text("x"), Param::Text(_)));
    assert!(matches!(Param::json("{}"), Param::Json(_)));
    assert!(matches!(Param::int(3), Param::Int(3)));
    assert!(matches!(Param::int64(3), Param::Int64(3)));
    assert!(matches!(Param::boolean(true), Param::Boolean(true)));
    assert!(
        matches!(Param::bytea(&[0xde, 0xad]), Param::Bytea(v) if v.as_slice() == &[0xde, 0xad])
    );
}

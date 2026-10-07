use super::*;

#[test]
fn strict_wire_deadlines_decode_to_unix_seconds() {
    assert_eq!(parse_iso_deadline("1970-01-01T00:00:00Z"), Some(0));
    assert_eq!(parse_iso_deadline("2024-02-29T00:00:00Z"), Some(1709164800));
    assert_eq!(
        parse_iso_deadline("2030-06-15T12:34:56.123456+02:00"),
        Some(1907750096)
    );
    assert_eq!(parse_iso_deadline("0001-01-01T00:00:00Z"), None);
    assert_eq!(parse_iso_deadline("1969-12-31T23:59:59Z"), None);
}

#[test]
fn strict_wire_deadlines_reject_permissive_iso_shapes() {
    for input in [
        "2025-07-04 12:00:00+00:00",
        "2025-07-04T12:00+00:00",
        "2025-07-04T12+00:00",
        "20250704T120000+00:00",
        "2025-07-04T12:00:00+0000",
        "2025-07-04T12:00:00+00",
        "2025-07-04T12:00:00+00:00:01",
        "2025-07-04T12:00:00.1234567890Z",
        "2025-07-04T12:00:60Z",
        "2025-02-29T12:00:00Z",
        "2025-07-04T12:00:00z",
    ] {
        assert_eq!(parse_iso_deadline(input), None, "input {input:?}");
    }
}

#[test]
fn now_secs_tracks_system_wall_clock() {
    let sys_now = std::time::SystemTime::now()
        .duration_since(std::time::UNIX_EPOCH)
        .unwrap()
        .as_secs() as i64;
    assert!((now_secs() - sys_now).abs() <= 2);
}

use super::*;

#[test]
fn current_timestamp_uses_canonical_shared_format() {
    let text = now_rfc3339_nano();
    assert!(validate_rfc3339(&text).is_ok());
    let (seconds, nanos) = soda_wire_time::parse(&text).unwrap();
    assert_eq!(
        soda_wire_time::format(seconds, nanos).as_deref(),
        Some(text.as_str())
    );
}

#[test]
fn timestamp_validation_uses_strict_wire_profile() {
    for good in [
        "2026-10-04T18:05:00Z",
        "2026-10-04T18:05:00.123456789Z",
        "2024-02-29T00:00:00+00:00",
        "0001-01-01T00:00:00Z",
    ] {
        assert!(validate_rfc3339(good).is_ok(), "{good}");
    }
    for bad in [
        "2026-10-04 18:05:00Z",
        "2026-13-01T00:00:00Z",
        "2023-02-29T00:00:00Z",
        "2026-10-04T24:00:00Z",
        "2026-10-04T00:00:00",
        "2026-10-04T00:00:00.",
        "2026-10-04T00:00:00.1234567890Z",
        "2026-10-04T00:00:61Z",
        "0000-01-01T00:00:00Z",
        "not-a-time",
    ] {
        assert!(validate_rfc3339(bad).is_err(), "{bad}");
    }
}

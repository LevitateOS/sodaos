use super::*;

#[test]
fn civil_dates_cover_edges() {
    assert_eq!(format_unix_nano(0, 0), "1970-01-01T00:00:00Z");
    assert_eq!(format_unix_nano(0, 120_000_000), "1970-01-01T00:00:00.12Z");
    assert_eq!(format_unix_nano(1_704_067_200, 0), "2024-01-01T00:00:00Z");
    assert_eq!(
        format_unix_nano(1_893_456_000, 1),
        "2030-01-01T00:00:00.000000001Z"
    );
    assert_eq!(format_unix_nano(-1, 0), "1969-12-31T23:59:59Z");
    assert!(validate_rfc3339(&now_rfc3339_nano()).is_ok());
}

#[test]
fn timestamps_validate_strictly() {
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
        "not-a-time",
    ] {
        assert!(validate_rfc3339(bad).is_err(), "{bad}");
    }
}

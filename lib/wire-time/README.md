# soda-wire-time

Parse and format timestamps exchanged by Soda's Rust and Go components. This
library gives consumers one strict RFC3339 wire format and canonical UTC output.
The Cargo package is `soda-wire-time`; Rust imports use `soda_wire_time`.

## Use

Use this workspace library when reading protocol timestamps or producing UTC
timestamps and filename stems. It has no executable or configuration file.

```rust
let (seconds, nanos) =
    soda_wire_time::parse("2026-10-04T20:30:05.1200+02:00").unwrap();
assert_eq!(
    soda_wire_time::format(seconds, nanos).as_deref(),
    Some("2026-10-04T18:30:05.12Z")
);
assert_eq!(
    soda_wire_time::compact_utc(seconds).as_deref(),
    Some("20261004T183005Z")
);
```

The [public functions](src/lib.rs) are:

- `parse(&str)`: Unix seconds and normalized nanoseconds.
- `parse_nanos(&str)`: signed Unix nanoseconds as `i128`.
- `format(i64, u32)`: UTC RFC3339, carrying overfull nanoseconds into seconds.
- `compact_utc(i64)`: a UTC `YYYYMMDDTHHMMSSZ` filename stem.

## Input limits

Parsing requires uppercase `T`, uppercase `Z` or a numeric `±HH:MM` offset,
and at most nine fractional digits. Calendar validity is checked. Leap seconds,
year zero, whitespace and incomplete timestamps are refused. The UTC result
must stay within years 1–9999. Invalid or out-of-range values return `None`;
callers decide how to report that failure. These functions do no IO.

For the interfaces that carry these values, see the
[factory interfaces](../../docs/architecture/factory-interfaces.md) and
[terminal reference](../../docs/reference/terminal.md).

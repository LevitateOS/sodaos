# Installation input parsers

[Responsibility map index](README.md) · [Coverage scope](../README.md).

Current responsibility accounting at `519b76bd` (2026-10-07).
One slice owns each named duty; disjoint complete symbols may share an owner.
Compound fields/clauses may share a physical line with distinct selectors.
Disposition concerns the duty, not source validity or installed qualification.

<a id="coverage-6e31ec2142a2"></a>

## [internal/strictjson/decode.go](../../../../../internal/strictjson/decode.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–188; file scaffold; maximumRequestBytes; Decode; decodeUniqueObject; requireObject; decodeFields; decodeFieldName; rejectDuplicateKeys; rejectObjectDuplicates; finishObject | [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) | retained | Current scaffold duty: file scaffold; 10 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-8ce2fde29edc"></a>

## [internal/strictjson/decode_test.go](../../../../../internal/strictjson/decode_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–76; file scaffold; testRequest; TestDecodeAcceptsOneKnownObject; TestDecodeRejectsInvalidRequestShapes; nestedRequest; listedRequest; TestDecodeRejectsNestedDuplicateFields; TestDecodeAcceptsUniqueNestedFields; TestDecodeRejectsInvalidUTF8AndOversizedRequests | [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) | retained | Current scaffold duty: file scaffold; 9 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-96556299160e"></a>

## [lib/wire-time/src/lib.rs](../../../../../lib/wire-time/src/lib.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–155; current module/import/attribute shell; declaration parse; declaration parse_nanos; declaration format; declaration compact_utc; declaration wire_shape; declaration digits; declaration tests; declaration strict_profile_and_calendar_boundaries; declaration canonical_format_normalizes_and_trims | [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) | retained | Imports and module declarations wire lib/wire-time/src/lib.rs into its current native target.; 10 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

## Removed predecessor navigation

These anchors identify earlier mapped source. Their paths no longer define this current map;
the current inventory and normalized predecessor ledger preserve replacement/retirement evidence.
No historical source body is counted as current coverage.

<a id="coverage-d643d24a4217"></a>
<a id="rustsoda-installsrcnetiprs-1"></a>

Former source `rust/soda-install/src/netip.rs`; consult its pinned earlier Git source and the current coverage disposition.

<a id="coverage-c717e6a318a9"></a>
<a id="rustsoda-installsrcsshkeyrs-1"></a>

Former source `rust/soda-install/src/sshkey.rs`; consult its pinned earlier Git source and the current coverage disposition.

<a id="coverage-53436d09cc68"></a>
<a id="rustsoda-installsrcurlxrs-1"></a>

Former source `rust/soda-install/src/urlx.rs`; consult its pinned earlier Git source and the current coverage disposition.

<a id="coverage-2ee020b0bd1c"></a>
<a id="rustsoda-installsrcx509rs-1"></a>

Former source `rust/soda-install/src/x509.rs`; consult its pinned earlier Git source and the current coverage disposition.

# Soda forgejo migrate

[Responsibility map index](README.md) · [Coverage snapshot and limits](../README.md).
Page grouping is navigation; the slice IDs retain their individual review ownership.

<a id="coverage-07af1cdebbc5"></a>

## [cmd/soda-forgejo-migrate/src/main.rs](../../../../../cmd/soda-forgejo-migrate/src/main.rs)

Scoped R02 reconciliation at `eaed66a9` (2026-10-07). This source is staged by
release-image's current compiler/payload selection and
`system/host/services/soda-forgejo-migrate.service`. It removes only conflicting
inline database credentials before Forgejo; no generic rotation or new
publication contract follows. L18/TMP02 deleted `create_tmp`, its entropy/name
allocation, `remove_tmp` and the unused empty file. The original inode write,
mode/ownership, service UMask0077 and byte scrub remain. Two unit tests, three
real-command/wiring script tests and eight byte/inode/mode fixtures pass; source
and development-build evidence is distinct from installed qualification.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [O03](../../slices/operator-administration.md#o03-existing-install-credential-maintenance) / active | 1–45 | Existing config conflict repair: `main`, `run`, explicit/default path, regular-file skip, read-before-write scrub, diagnostic and exit handling; in-place original-inode write |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 47–75 | Byte-wise records/newline handling and exact assignment admission: `records`, `is_passwd_assignment` |
| [O03](../../slices/operator-administration.md#o03-existing-install-credential-maintenance) / active | 77–97 | Exact database-section `migrate`, `in_db`, scrubbed count and unchanged other bytes |
| [O03](../../slices/operator-administration.md#o03-existing-install-credential-maintenance) / active tests | 99–126 | `tests`, `passwd_shape_matches_awk`, `record_split_matches_awk_print`; command/mode/wiring fixtures remain in scripts |
| [H04](../../slices/shared-supporting-slices.md#h04-configuration-and-filesystem-primitives) / retired by TMP02 | removed | Unused empty temporary allocation/name/cleanup; no replacement owner or library required |

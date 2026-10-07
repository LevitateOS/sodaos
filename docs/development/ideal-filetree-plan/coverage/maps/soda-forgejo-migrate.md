# Soda forgejo migrate

[Responsibility map index](README.md) · [Coverage scope](../README.md).

Current responsibility accounting at `519b76bd` (2026-10-07).
One slice owns each named duty; disjoint complete symbols may share an owner.
Compound fields/clauses may share a physical line with distinct selectors.
Disposition concerns the duty, not source validity or installed qualification.

<a id="coverage-07af1cdebbc5"></a>
<a id="coverage-151e193d39ae"></a>

## [cmd/soda-forgejo-migrate/src/main.rs](../../../../../cmd/soda-forgejo-migrate/src/main.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–49, 78–126; current module/import/attribute shell; declaration main; declaration run; declaration migrate; declaration tests; declaration passwd_shape_matches_awk; declaration record_split_matches_awk_print | [O03](../../slices/operator-administration.md#o03-existing-install-credential-maintenance) | retained | Imports and module declarations wire cmd/soda-forgejo-migrate/src/main.rs into its current native target.; 7 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |
| 50–77; declaration records; declaration is_passwd_assignment | [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) | retained | records: records: current source retains the mapped byte-wise records/newline handling and exact assignment admission: `records`, `is_passwd_assignment` duty.; is_passwd_assignment: is_passwd_assignment: current source retains the mapped byte-wise records/newline handling and exact assignment admission: `records`, `is_passwd_assignment` duty. — current source cmd/soda-forgejo-migrate/src/main.rs; lines 50-58; module/caller wiring inspected; current source cmd/soda-forgejo-migrate/src/main.rs; lines 59-77; module/caller wiring inspected |

# Soda forgejo migrate

[Responsibility map index](README.md) · [Coverage snapshot and limits](../README.md).
Page grouping is navigation; the slice IDs retain their individual review ownership.

<a id="coverage-07af1cdebbc5"></a>

## [rust/soda-forgejo-migrate/src/main.rs](../../../../../rust/soda-forgejo-migrate/src/main.rs)

Shipped via appliance/services/soda-forgejo-migrate.service:9 and soda-release-image/src/build.rs:621-623. Existing-install config repair; no generic rotation contract inferred. Source assertions/fixtures were inspected for mapping only; no test execution, runtime or installed proof claimed.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [O03](../../slices/operator-administration.md#o03-existing-install-credential-maintenance) / active | 1–57 | Native existing config credential conflict repair before Forgejo; declarations/fields: `main`, `run` |
| [H04](../../slices/shared-supporting-slices.md#h04-configuration-and-filesystem-primitives) / active | 58–101 | Private staging file and no-broad-migration write mechanics; declarations/fields: `create_tmp`, `ALPHABET`, `remove_tmp`, `records` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 102–137 | Byte-wise AWK-equivalent record/password assignment matching; declarations/fields: `is_passwd_assignment`, `migrate` |
| [O03](../../slices/operator-administration.md#o03-existing-install-credential-maintenance) / active | 138–162 | Exact database-section credential config migration; declarations/fields: `tests`, `passwd_shape_matches_awk` |
| [O03](../../slices/operator-administration.md#o03-existing-install-credential-maintenance) / active | 163–173 | Section/password shape source assertions; declarations/fields: `record_split_matches_awk_print` |


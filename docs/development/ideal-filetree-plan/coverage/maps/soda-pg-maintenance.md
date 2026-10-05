# Soda pg maintenance

[Responsibility map index](README.md) · [Coverage snapshot and limits](../README.md).
Page grouping is navigation; the slice IDs retain their individual review ownership.

<a id="coverage-49cb1c9a600f"></a>

## [rust/soda-pg-maintenance/src/lib.rs](../../../../../rust/soda-pg-maintenance/src/lib.rs)

Source assertions/fixtures were inspected for mapping only; no test execution, runtime or installed proof claimed. Public module wiring is mapped separately; module exposure does not create a new process boundary.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [H02](../../slices/shared-supporting-slices.md#h02-storage-mechanics) / active | 1–24 | Bounded native database identifier/SQL mechanics; declarations/fields: `valid_db_name`, `escape_literal` |
| [O01](../../slices/operator-administration.md#o01-first-boot-database-provisioning) / active | 25–33 | Initial database role SQL; declarations/fields: `init_role_sql` |
| [O06](../../slices/operator-administration.md#o06-database-restore) / active | 34–52 | Fresh database restore existence SQL; declarations/fields: `db_exists_sql`, `civil_from_days` |
| [O05](../../slices/operator-administration.md#o05-database-backup-and-retention) / active | 53–76 | Native backup timestamp/retention helper; declarations/fields: `stamp_from_unix`, `utc_stamp`, `tests` |
| [O01](../../slices/operator-administration.md#o01-first-boot-database-provisioning) / active | 77–111 | Native SQL/date source assertions; declarations/fields: `db_names_match_shell_case`, `literal_escapes_quotes_only`, `init_sql_shape`, `stamps_match_date_utc` |


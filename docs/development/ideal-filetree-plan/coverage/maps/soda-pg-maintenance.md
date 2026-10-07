# Soda pg maintenance

[Responsibility map index](README.md) · [Coverage scope](../README.md).

Current responsibility accounting at `519b76bd` (2026-10-07).
One slice owns each named duty; disjoint complete symbols may share an owner.
Compound fields/clauses may share a physical line with distinct selectors.
Disposition concerns the duty, not source validity or installed qualification.

<a id="coverage-ea796879e80a"></a>

## [cmd/soda-pg-maintenance/src/bin/soda-pg-backup.rs](../../../../../cmd/soda-pg-maintenance/src/bin/soda-pg-backup.rs)

Inherited prior inventory row; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–259; lines 1–16: use soda_pg_maintenance and attached body; lines 17–17: use std and attached body; lines 18–18: use std and attached body; lines 19–19: use std and attached body; lines 20–20: use std and attached body; lines 21–21: use std and attached body; lines 22–22: use std and attached body; lines 23–26: fn main and attached body; lines 27–142: fn run and attached body; lines 143–166: fn rename_noreplace and attached body; lines 167–184: fn dump_to_file and attached body; lines 185–186: fn has_pgdmp_magic and attached body; lines 187–196: use std and attached body; lines 197–214: fn rotate and attached body; lines 215–224: fn is_run_name and attached body; lines 225–225: mod tests and attached body; lines 226–228: use super and attached body; lines 229–245: fn run_names_match_shell_grep and attached body; lines 246–259: fn publication_does_not_replace_existing_timestamp_directory and attached body | [O05](../../slices/operator-administration.md#o05-database-backup-and-retention) | retained | Declaration block for use soda_pg_maintenance in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 19 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-a796f9dfa694"></a>

## [cmd/soda-pg-maintenance/src/bin/soda-pg-init-roles.rs](../../../../../cmd/soda-pg-maintenance/src/bin/soda-pg-init-roles.rs)

Inherited prior inventory row; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–117; lines 1–16: use soda_pg_maintenance and attached body; lines 17–17: use std and attached body; lines 18–18: use std and attached body; lines 19–19: use std and attached body; lines 20–20: use std and attached body; lines 21–21: use std and attached body; lines 22–22: use std and attached body; lines 23–26: fn main and attached body; lines 27–117: fn run and attached body | [O01](../../slices/operator-administration.md#o01-first-boot-database-provisioning) | retained | Declaration block for use soda_pg_maintenance in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 9 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-0a86156af60b"></a>

## [cmd/soda-pg-maintenance/src/bin/soda-pg-restore.rs](../../../../../cmd/soda-pg-maintenance/src/bin/soda-pg-restore.rs)

Current cohesive body inspected; single defining duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–430; whole file: explicit postgresql restore command: admission, safe remote stage creation, dump transfer, database restoration and cleanup | [O06](../../slices/operator-administration.md#o06-database-restore) | retained | Explicit PostgreSQL restore command: admission, safe remote stage creation, dump transfer, database restoration and cleanup. — cmd/soda-pg-maintenance/src/bin/soda-pg-restore.rs; operator invoked with --yes |

<a id="coverage-49cb1c9a600f"></a>
<a id="coverage-9a830b9df368"></a>

## [cmd/soda-pg-maintenance/src/lib.rs](../../../../../cmd/soda-pg-maintenance/src/lib.rs)

Current helper and direct command consumers inspected

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–37, 65–94; database name admission, SQL quoting and role/database setup query generation; embedded database-name, literal escaping, and role/database SQL assertions | [O01](../../slices/operator-administration.md#o01-first-boot-database-provisioning) | retained | Validates database identifiers and builds idempotent PostgreSQL role/database setup SQL consumed by first-boot provisioning.; Tests identifier admission and SQL-generation contracts used by PostgreSQL role/database provisioning. — cmd/soda-pg-maintenance/src/lib.rs:9-37; soda-pg-init-roles and soda-setup provisioning callers; current source cmd/soda-pg-maintenance/src/lib.rs:65-94; tests pin O01 helpers |
| 38–43, 55–64, 112–129; UTC backup naming and timestamp generation; embedded UTC timestamp conversion assertions | [O05](../../slices/operator-administration.md#o05-database-backup-and-retention) | retained | Formats UTC backup timestamps for stable backup naming and retention records.; Tests UTC backup timestamp conversion and boundary behavior used by backup naming and retention records. — cmd/soda-pg-maintenance/src/lib.rs:39-42,56-63; backup command consumer; current source cmd/soda-pg-maintenance/src/lib.rs:112-129; tests pin timestamp helper |
| 44–54, 95–111; SQL child delivery exit mapping shared by provision and restore; delivery_exit_maps_failed_write_and_child_status | [H04](../../slices/shared-supporting-slices.md#h04-configuration-and-filesystem-primitives) | retained | Maps input-delivery success and the already-reaped child ExitStatus to the process outcome; command callers retain child creation, stdin write, exactly-once reap, and O01/O06 domain admission/state.; Focused contract test for shared process delivery and reaped-child outcome mapping; O01/O06 callers retain command lifecycle and domain state. — current source cmd/soda-pg-maintenance/src/lib.rs:44-54; callers soda-pg-init-roles.rs and soda-pg-restore.rs each own delivery/reap and call the shared mapper; current source cmd/soda-pg-maintenance/src/lib.rs:95-111; directly exercises delivery_exit outcome mapping |

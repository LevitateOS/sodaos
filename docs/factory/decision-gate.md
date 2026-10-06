# Factory storage decision gate

> **Superseded for the Soda store.** The owner retains PostgreSQL for the
> Soda factory/product store. The SQLite gate, measurements and
> single-writer proof below are preserved as historical evidence for the
> retired backend only; they do not qualify the current PostgreSQL store.
> The intake fail-closed contract (B3) is backend-independent and still
> holds.

This note records the storage-backend decision for the factory run/command
ledger and the intake fail-closed contract. It closes review items A10
(SQLite-vs-PostgreSQL gate), A11 (keep Soda on SQLite), N-ST1 (reservation
race under the selected backend) and B3 (unconfigured intake stays
fail-closed). No PostgreSQL backend, dialect layer, or migration was
adopted.

## Measured load (A10)

Representative single-appliance factory load was measured against the
production store path (`internal/store`, SQLite via `modernc.org/sqlite`,
WAL, `busy_timeout(5000)`, single-writer pool) with pinned Go 1.26.7:

- Intake: 500 deliveries + dedup reads at 94/s.
- Readiness sweeps: 500 save+read pairs at 105/s.
- Listings: 200 policy/control listings in ~10 ms total.
- Size: ~380 KiB for 500 deliveries; restore is a single-file copy with
  `IntegrityCheck` after open.

Steady-state factory traffic is webhook bursts of a few deliveries per
minute plus dispatch passes visiting at most 64 queued issues each. The
measured throughput exceeds that by two orders of magnitude, and the
bounded listings (`MaxQueuedDispatch`, `MaxHeldReservations`) cap growth.
SQLite is sufficient; no alternate backend is selected.

## Decision (A10, A11)

- Soda keeps its SQLite store. No PostgreSQL dependency, no dialect
  abstraction, no migration.
- Revisit only on evidence: sustained intake/dispatch write latency above
  the busy timeout, database growth that breaks single-file backup/
  restore windows, or a second writer that cannot join the coordinator's
  exclusive ownership (`StartCoordinator`). A revisit must re-measure
  first and, per the gate, prove and fix the native reservation race on
  the candidate backend before any switch.

## Reservation race (N-ST1)

On the selected backend the dispatch reservation race is fenced without
a new mechanism:

- Single writer: `SetMaxOpenConns(1)` serializes all store access, so no
  two transactions interleave on one appliance.
- Atomic transitions: held-to-terminal moves are single conditional
  `UPDATE ... WHERE state='held'` statements that succeed only on
  exactly one row (`transitionReservation`); concurrent settlers cannot
  consume capacity twice. Dispatch packets commit assignment,
  reservation, run and view atomically (`RecordDispatchPacket`), and a
  second unfinished assignment for an issue refuses with
  `ErrAssignmentActive`.
- Verified: `TestRecordDispatchPacket`,
  `TestDispatchAttemptAndFinish`, `TestReservationTransitions` and
  `TestRunUsageFirstWriteWins` pass under `-race`.

No READ-COMMITTED anomaly applies: there is no PostgreSQL backend, and
no code path assumes multi-writer semantics.

## Intake fail-closed (B3)

Unconfigured factory intake refuses as unavailable; it never bypasses
authentication:

- `loadIntakeSecret` returns nil when `factory_intake_secret_file` is
  unset or unreadable (`internal/web/server.go`).
- A nil secret or missing coordinator answers `503 intake_unavailable`
  (`internal/web/api/factory_intake.go`); assessment failures also
  answer 503 so the native side retries.
- Covered by `TestFactoryIntakeFailures` ("unconfigured intake served")
  and `TestFactoryIntakeSecretFile` (absolute-path validation).

Authority for factory intake and grants stays with the
[trust model](../architecture/trust.md); the operator surface is the
[factory reference](../reference/factory.md).

# Go package boundary checks

Developers use this test-only package to check Soda's internal package topology.
It has no production API or runnable command. The source of the checks is
[`arch_test.go`](arch_test.go).

## Run

From the repository root with the pinned Go toolchain:

```sh
go test ./internal/archcheck/
```

For a focused check, select an existing test:

```sh
go test ./internal/archcheck/ -run TestDependencyDirection
```

The checks read source files and directories. They do not start Soda, connect
to its database or exercise installed host behavior; Go may create its normal
test/build cache.

## What the checks cover

- Retired package paths and the removed web alias facade stay absent.
- Production Go package declarations match their directory names.
- Production imports follow the encoded ownership and dependency boundaries.
- Required current owners exist, and superseded native executor packages remain
  absent after their Rust cutover.

The import scan excludes test files so test fakes can cross boundaries. These
tests protect topology rather than proving authorization, product correctness
or native deployment behavior.

When a check fails, inspect the reported package/import and its owning source.
Follow [Go ownership](../../docs/development/go.md) and
[package conventions](../../docs/development/go-packages.md), reconciled with
the [recorded ownership decisions](../../docs/development/ideal-filetree-plan/package-ownership.md).
An intended ownership change needs its applicable decision; do not weaken a
boundary merely to silence a failing import.

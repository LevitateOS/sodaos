# Soda source ownership and desired tree

Use these documents to locate responsibilities and concrete refactoring options.
The [backlog](../factory-remaining-tasks.md) records unfinished product work;
the owner's current request determines what to implement.

## Retained Forgejo presentation

Soda deliberately overrides all Forgejo pages. Preserve that design scope and
check native integration when changing it.

## Owner language policy

Network-facing servers use Go. System and privileged applications use Rust.
Do not add authored Python programs. Preserve the existing Rust host and release
crates and their installed executable identities.

Installed commands belong in `cmd/`, libraries in `lib/`, support tools in
`tools/`, browser surfaces in `frontend/`, and image/service definitions in
`system/`. Canonical assets stay in `assets/`; the public handbook lives in
`docs/public/`.

## Technical references

| Document | Purpose |
| --- | --- |
| [Package ownership](package-ownership.md) | Packages and process boundaries |
| [Port assessment](port-assessment.md) | Language owners and remaining cutovers |
| [Placement](placement.md) | File and package placement |
| [Proposed tree](proposed-tree.md) | Proposed implementation destinations |
| [Decomposition](decomposition/README.md) | Concrete responsibility seams |
| [Integration](integration.md) | Build, import and installation wiring |
| [Library contracts](library-adoption.md) | Library boundaries and callers |

Update technical guidance when a change makes it misleading. No audit inventory,
review ledger, consultation receipt or coverage reconciliation is required.

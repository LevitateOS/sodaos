# Product scope

This document owns deferred and excluded product work for the target architecture.
It is a scope boundary, not a backlog tracker and not permission to strip working
validation or error handling.

The automatic issue-to-merge loop, including dependency and requirements blockers,
environment reuse, visible CLI execution, review/fix and reconsideration of waiting
issues, is core scope. Its behavior is owned by the
[factory product contract](overview.md#software-factory-workflow).

## Deferred

1. **Private toolchain and service branches** — temporary personal branches of shared
   tools or services that return to the shared environment after integration.
2. **Access-lifecycle edge cases** — automatic synchronization of key rotation,
   Linux offboarding and unrelated active-session termination across systems.
3. **Identity and ownership remapping** — silent remapping of Linux users when
   Forgejo usernames or repository ownership change.
4. **General operation-recovery machinery** — fleet-wide reconciliation, repair
   orchestrators and automatic destructive recovery.
5. **Broader recovery and lifecycle management** — project archival/deletion
   programmes and general fleet orchestration beyond the selected release model.
6. **Factory expansion** — production deployment after merge, multihost execution,
   general-purpose orchestration beyond the factory loop, workspace snapshots or
   conversation resume, and execution of hostile external contributions. The
   factory targets a trusted team on one operator-managed appliance.

## Not pursuing

- Public Internet application hosting as a Soda product surface
- Mutually untrusted tenant hosting
- Domain ownership or preprovisioned per-app public certificates as a prerequisite
  for trying Soda or using baseline services
- A second standalone Soda web frontend
- Predecessor host developer accounts, Cockpit Projects, managed checkouts or the
  predecessor Updates platform

## Still required

Deferral does not remove:

- authorized factory operation, distinct execution identities, time and resource
  limits, cancellation and reconciliation of recorded run resources
- exact-candidate verification and review, native merge authorization and an
  understandable intervention outcome when work cannot finish
- ordinary authorization, validation and honest error handling
- persistence and no-destructive-repair rules for project roots
- native installation, update and recovery qualification for the release candidate
- usable project-local accounts, SSH, shared tools, reachable project addresses and
  persistent project state

Release and update architecture: [Release](../architecture/release.md).
Host capability strategy (non-normative): [Host strategy](../research/host-strategy.md).

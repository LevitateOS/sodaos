# Product scope

This document owns deferred and excluded product work for the current architecture.
It is a scope boundary, not a backlog tracker and not permission to strip working
validation or error handling.

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

## Not pursuing

- Public Internet application hosting as a Soda product surface
- Domain ownership or preprovisioned per-app public certificates as a prerequisite
  for trying Soda or using baseline services
- A second standalone Soda web frontend (the same-origin workspace shell that
  frames Forgejo is not this)
- A downstream Forgejo fork for convenience features
- Predecessor host developer accounts, Cockpit Projects, managed checkouts or the
  predecessor Updates platform

## Still required

Deferral does not remove:

- ordinary authorization, validation and honest error handling
- persistence and no-destructive-repair rules for project roots
- native installation, update and recovery qualification for the release candidate
- usable project-local accounts, SSH, shared tools, reachable project addresses and
  persistent project state

Release and update architecture: [Release](../architecture/release.md).
Host capability strategy (non-normative): [Host strategy](../research/host-strategy.md).

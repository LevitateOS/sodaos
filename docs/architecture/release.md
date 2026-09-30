# Release architecture

Soda ships as a **minimum-deviation Fedora CoreOS** appliance: one immutable
install/update candidate that carries the Soda payload, then separate commissioning
for public delivery and unattended scheduling.

This document owns the durable release model. How to run tools day-to-day lives in
[development release workflow](../development/release.md) and
[installation](../guides/installation.md).

## Candidate model

1. **Build** produces architecture-specific candidate artifacts.
2. **Image/media** assembles host content and installation media that consume that
   candidate.
3. **Qualify** admits artifacts against native installation and guest-state checks.
4. **Deliver** signs and publishes only admitted payloads.

One candidate is the unit of install and update. Experimental host-image tooling
does not replace native FCOS installation and updating.

## Host update ownership

The appliance follows Fedora CoreOS update mechanics (rpm-ostree / Zincati-aligned
operation). Soda does not invent a parallel host updater for ordinary maintenance.
Signed CoreOS-aligned publication and an independent emergency train follow the
working delivery lane; native installation, update and recovery must be qualified
with the candidate.

## Architectures

SodaOS supports **x86_64 only**. AArch64/ARM64 is unsupported and excluded from
current implementation, build, delivery and qualification scope. Supporting it
requires a future explicit product decision; upstream platform support or existing
experimental selectors do not establish Soda support.

Qualification requires native Linux x86_64 evidence. Cross-compilation, emulation
and development checks on another host architecture do not qualify the appliance.

## Media

ISO is the primary installation medium. Prepared cloud disk images may follow the
same candidate. Exact artifact names, trust bootstrap and publication endpoints
belong to the delivery implementation and operator guides, not duplicated version
tables here.

## Separation from product features

Release engineering owns shipping bytes and qualification. It does not redefine
project, Spaces, Tailnet or runner product contracts. Those remain in their
canonical documents.

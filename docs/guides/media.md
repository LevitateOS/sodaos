# CoreOS installation media

Soda installation media is a Fedora CoreOS-derived installer that consumes a signed
Soda candidate. It is not Anaconda/Kickstart and not a predecessor-branded installer.

Release model: [Release architecture](../architecture/release.md).
Build/install procedures: [Installation](installation.md).

## Owners

| Owner | Responsibility |
| --- | --- |
| `soda-candidate`, `soda-build` (`lib/soda-release-tools`) | Development candidate/media producer (no release qualification) |
| Stable CoreOS stream (resolved live per build) | Upstream ISO metadata per architecture; observed values recorded, never pinned |
| `rust/soda-install` | Interactive disk adapter and installed-host continuation |
| `soda-render-provisioning` (`tools/release-assets`) | Public bootstrap Butane/Ignition template |

Python and Butane are build-time tools. They are not required on the live OS.

## Branding

ISO boot menus display SodaOS installer identity through same-length text
substitutions that preserve native kernel-argument embed offsets. The signed EFI
executable, kernel, initramfs and OS payload are not patched.

Public live and destination provisioning may share canonical SVG artwork while
leaving `/usr/lib/os-release` metadata intact. Do not use OS metadata as a
display-only branding overlay. Soda branding belongs in the installer, console
artwork and supported Cockpit hooks.

## Console on media

The interactive installer console ships on the ISO. The live launcher loads it from
media; media build is not an application-build prerequisite for a running host.

## Install interaction

Manual install uses a password-only text flow with correction/Back/pre-write restart,
then key enrollment and private setup on the installed host. The installer writes
disks only after explicit confirmation. Interrupted writes must leave a recoverable
or clearly failed machine state rather than a silently partial appliance.

After a disk-install outcome, the live console offers an explicit reboot or
poweroff choice. If terminal input ends or fails, the installer attempts one
reboot with a two-minute bound. A failed reboot is reported; an existing install
failure remains the returned primary error.

## After first boot

1. Import a laptop SSH key after local password login when that path is selected ([optional key enrollment](../operator/enroll-key.md)).
2. Configure private browser access with explicit client certificate trust.
3. Complete [Operator setup](operator-setup.md).

## Validation

Media may select any authenticated `coreos-installer` version. The producer records
the actual version from the candidate OS image in the authenticated live identity.
The live installer must match that identity before installation. The Assembler's
customization tool has its own recorded version, which customization must match;
it can differ from the installer delivered in the candidate OS. A missing,
malformed or different version refuses the corresponding handoff.

Validate media and installation on native `x86_64` under the
[platform scope](../architecture/release.md#architectures). Diskless boot evidence does
not substitute for fresh-disk installation proof. Candidate identity, download
integrity and complete local payload availability must hold before installation
finishes for network-install media.

# Operator Cockpit

Soda ships **stock Cockpit pages with Soda branding**. Custom Soda Cockpit extension
pages are retired from source. Tailnet and Runners operator controls live in native
Forgejo administration views (`/admin?soda-view=tailnet` and `/admin?soda-view=runners`).

Branding assets: `assets/branding/cockpit/`. Provenance:
`assets/branding/cockpit/provenance/`.

## Role

Cockpit is the root administrator's host console, not a restricted copy of the
project dashboard. Preserve the root-only / private-access boundary. Ability to
change Soda-managed resources is not by itself a reason to hide a stock page.

## Root-administration baseline

| Page | Purpose |
| --- | --- |
| Overview | Host health, clock, hardware, shutdown/reboot |
| Logs | Native journal diagnosis |
| Storage | Disks, filesystems, mounts, capacity |
| Networking | Host interfaces and firewall; keep a recovery path |
| Services | Host units including appliance Quadlets and project units |
| Podman Containers | Host engine administration and troubleshooting |
| Terminal | Root diagnosis/recovery |
| Files | Host file inspection and deliberate configuration maintenance |
| SELinux | Diagnose policy denials on the enforcing appliance |
| Diagnostic Reports | Manual support-bundle collection; review before sharing |
| Accounts | Host Linux accounts for root administration, not Forgejo or project identities |

## Branding rules

Use supported upstream branding/theme mechanisms for login and shell. Follow current
Soda design tokens; do not freeze older identity-kit presentation. Do not keep a
Soda extension merely to supply branding. Validate against the installed Cockpit
version.

## Related

- [Networking](../architecture/networking.md)
- [Branding](../design/branding.md)
- [Installation](../guides/installation.md)

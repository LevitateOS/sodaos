# Selected predecessor reuse

Reference: `soda-os` commit `bc1d3e0dbec48dfaa6a20f9d0453ad3e4cdf353c`. The predecessor repository is unchanged. This is a compatibility inventory, **not evidence that its old deployment works in SodaOS**. All new source and authored checks remain unbuilt/unexecuted.

## Already ported with M01–M14

- Canonical branding and palette, with existing attribution/licenses.
- Tailnet and Runners Cockpit pages, stores, native bridges, shared components and focused tests.
- Their Go commands/native logic and narrow shared process, locking and JSON helpers where needed.
- Runner client source inputs and relevant native service wiring, adapted to operator-only access and configured Forgejo origins.

## Compatible follow-ups

| Predecessor source | SodaOS destination/adaptation |
| --- | --- |
| `AGENTS.md` | Tailored engineering guidance; actual SodaOS scope, commands, source-only hold and evidence distinctions |
| `scripts/check-forgejo-branding.mjs` | Same native component/control/color checks, with explicit target/native execution and fresh-evidence guards |
| `scripts/render-forgejo-branding.sh`, `tools/png-equal/`, focused tests | Forgejo raster consistency/regeneration source; real renderer test opt-in, canonical artwork unchanged |
| `packaging/rpm/forgejo/sources/app.ini.tmpl` | Selected app metadata, stock/accessibility themes and cache policy in `appliance/config/forgejo.env`; no copied host paths, PAM identity or secrets |
| Native console welcome and profile hook | `appliance/bin/soda-console-welcome`, interactive-only host hook, actual configured HTTPS origins and loopback Cockpit guidance |
| `docs/public/40-Develop/10-connect-and-develop.md` | Reworked editor, SSH/SCP/SFTP, Git, tool and service guidance for project-local users/IPs and shared installations |
| `docs/screenshot-capture.md` | Current-page capture/redaction rules, with no fabricated images or published placeholders |
| Tea source lock/license/fetch/build inputs | `project-os/locks/`, `project-os/licenses/`, source fetch and native CLI build feeding the Rocky image instead of a Fedora host RPM |
| GitHub CLI baseline and user guidance | Same version baseline via GitHub's signed RPM repository inside Rocky; personal native authentication, not host runner credentials |

Relevant operating instructions are linked from the README. Native application configuration uses current container paths; image/build recipes and tests carry their own callers rather than leaving copied files unused.

## Deliberately not copied

- Host developer-account/workspace provisioning, managed checkout/catalog machinery and old Cockpit Projects/People pages.
- The predecessor's release/acceptance framework, broad image orchestrator, bootc/Anaconda installer and automatic publication paths.
- Separately reserved Updates work, signing/release ceremonies, coordinated recovery/removal and other [deferred scope](deferred.md).
- Git inspection/provisioning code with no current Soda-owned checkout caller.
- Old PAM-backed Forgejo human identity, fixed `:30000` browser links and the assumption that workspaces share the host network.
- Unrelated renderer/tooling directories solely to increase copied line count. Their source can be reassessed for a concrete future caller without modifying the predecessor.

## Evidence boundary

The work copied/adapted source and inspected upstream configuration/release metadata. No dependency resolution, source archive/client binary downloads, builds, renderer/browser checks, native service mutation, account/provider registration or screenshots ran. Focused tests are authored, not passed. Follow [implementation status](implementation-status.md) and [later native validation](native-validation.md) before making any installed capability claim.

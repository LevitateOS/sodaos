# Public handbook editorial handoff

The source under [public/](public/README.md) is **release-day user documentation**,
not an assertion that the current checkout is released or every described journey
has installed evidence. Keep readiness limitations here and in the implementation
handoff, not as banners or missing-feature caveats in public pages.

## Scope and research

Studied the predecessor's five-section/17-page handbook and the website's actual
`scripts/docs-sync.mjs`, `src/app/docs/content.ts`, route/page tests and navigation.
The predecessor checkout has unrelated unresolved/dirty work; it was read only,
not changed or synchronized. Its account, workspace, media/signing and Updates
instructions are not a governing contract for the new architecture.

The new handbook has five sections and 22 published pages: Start here, Deploy,
Use Soda, Develop, Operate. All 17 preceding route slugs are retained for topic
continuity; five pages add the dashboard, operator setup, collaboration, shared
tools/files and project services. The former updates/removal slugs now describe
native CoreOS maintenance and data safety, not predecessor destructive controls.
No old prose or screenshot assets were copied wholesale.

Source references for exact behavior include `docs/architecture.md`, the leading
U plan and `docs/deferred.md`; `docs/installation.md`, `docs/operator-setup.md`,
`docs/development-environment.md`, `docs/project-services.md`, `docs/project-clis.md`,
`docs/cockpit-port.md` and `docs/runners-port.md`; production project profiles,
account setup, service definitions and dashboard environment/repository controls.
The explicitly requested existing-workspace browser terminal is described as a
release-day outcome without inventing transport, implicit join or automatic start.

## Publication review still required

| Area | Internal follow-up, not public readiness copy |
| --- | --- |
| ISO/QCOW2/Scaleway | Website launch paths are preserved. P09/P10 media remain unselected in the engineering plan. Bind the public deployment guides to real release recipes, exact artifact forms, target firmware/disk semantics and platform-specific Ignition delivery before launch. Do not call a CoreOS base/kit a preinstalled Soda disk or infer cloud-init/Anaconda from the predecessor. |
| Artifact trust | The new repository has no selected predecessor-equivalent release signing/publishing contract. Verification prose explains trusted digests without inventing a Sigstore workflow, filenames, OCI host image or signed record. Add exact trust bootstrap and commands once selected; do not copy the old production certificate identity. |
| Product acceptance | U20 owns release proof, native architectures, fresh install, upgrade and lifecycle. These docs create no execution grant, new implementation milestone or evidence of a working terminal/complete dashboard. |
| Interface wording | Confirm actual release labels and native fallback destinations against each completed page; do not couple public instructions to `/app/` preview routes. |
| Existing-workspace terminal | Test the requested own-user/selected-project experience and session/security behavior before taking screenshots or accepting the corresponding instructions. No mechanism is selected here. |
| Backup/maintenance | Native operator procedures are guidance, not a new coordinated Soda backup/restore/updater feature. Match application upgrade steps and database/grant-key compatibility to the release. |
| Screenshots | Capture real dashboard/Forgejo/operator views per `docs/screenshot-capture.md`. No image placeholders, generated UI or private test evidence are published. |
| Website marketing alignment | Its existing design brief/home/team copy still contains predecessor `Set up for me`, per-person dependency and repository-catalog language. The new governing product uses explicit `Add me to this project`, project-local accounts and genuine shared installed tools. This task does not rewrite marketing or cancel personally owned hardware, launch media, Scaleway or future WSL2. Coordinate a focused copy/brief alignment separately. |
| Publication provenance | Sync only clean committed source through the website CLI. Source links require that commit to be published in `LevitateOS/sodaos`; local generation is not deployment or GitHub publication. |

## Editorial validation

The website synchronizer owns syntax/structure, relative page/heading links,
image paths/signatures, source hashes and deterministic generated HTML. Its
existing tests and per-page rendering tests cover ingestion without a parallel
Markdown renderer in SodaOS. Source freshness and snapshot integrity are separate
checks. Run website rendering/architecture checks after sync, and perform real
responsive/browser/visual review before launch. No image or browser runtime
claims follow from Markdown rendering alone.

Keep source and generated changes in separate repository commits; never modify
`src/app/docs/generated/` by hand. See the authoring README for exact commands.

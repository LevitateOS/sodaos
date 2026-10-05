# Responsibility map index

### Responsibilities inside mixed and oversized files

Every responsibility has a candidate review owner. Disjoint intervals can share a role; their exact spans and declaration/field names are preserved. Dense SQL declarations, shared struct fields and composite expressions can have several named responsibilities on one physical line; the field/column selectors distinguish them and do not claim competing ownership of the same field. Imports, module scaffolding and aggregate DTO composition have a review owner for bookkeeping, while domain fields, branches, SQL tables and assertions map to their own slices. This does not imply physically extracting each interval, splitting coherent functions arbitrarily, or adding packages/processes.


Snapshot and limits: [coverage index](../README.md). The page groups below
organize current source components and filename families for lookup; slice IDs
in the rows remain the review owners. A source file's full map stays on one
page. These groups do not establish target packages, services or sidecars.

| Current source component / family | File maps |
| --- | ---: |
| [Acceptance execution and evidence](acceptance-execution-and-evidence.md) | 6 |
| [Acceptance native qualification](acceptance-native-qualification.md) | 7 |
| [Appliance definitions](appliance-definitions.md) | 11 |
| [Assets](assets.md) | 2 |
| [Backend acceptance](backend-acceptance.md) | 4 |
| [Backend config](backend-config.md) | 2 |
| [Backend factory](backend-factory.md) | 9 |
| [Backend forgejo](backend-forgejo.md) | 5 |
| [Backend host](backend-host.md) | 9 |
| [Backend identity](backend-identity.md) | 9 |
| [Backend project](backend-project.md) | 8 |
| [Backend store](backend-store.md) | 18 |
| [Backend tailnet](backend-tailnet.md) | 10 |
| [Backend web api](backend-web-api.md) | 21 |
| [Backend web auth and composition](backend-web-auth-and-composition.md) | 11 |
| [Browser spaces](browser-spaces.md) | 11 |
| [Browser tailnet](browser-tailnet.md) | 2 |
| [Developer scripts](developer-scripts.md) | 15 |
| [Developer tools](developer-tools.md) | 3 |
| [Documentation contracts](documentation-contracts.md) | 11 |
| [Documentation development and design](documentation-development-and-design.md) | 16 |
| [Factory control admission and dispatch](factory-control-admission-and-dispatch.md) | 16 |
| [Factory control native fixtures](factory-control-native-fixtures.md) | 5 |
| [Factory control publication and review](factory-control-publication-and-review.md) | 10 |
| [Host factory runs](host-factory-runs.md) | 1 |
| [Host muse execution](host-muse-execution.md) | 3 |
| [Host preparation](host-preparation.md) | 2 |
| [Host projects](host-projects.md) | 6 |
| [Host protocols and clients](host-protocols-and-clients.md) | 4 |
| [Host provider execution](host-provider-execution.md) | 4 |
| [Host runtime composition](host-runtime-composition.md) | 4 |
| [Host service admission](host-service-admission.md) | 5 |
| [Host tailnet companions](host-tailnet-companions.md) | 4 |
| [Host tailnet control](host-tailnet-control.md) | 6 |
| [Host terminals](host-terminals.md) | 1 |
| [Identity broker protocol and policy](identity-broker-protocol-and-policy.md) | 9 |
| [Identity broker state and entrypoints](identity-broker-state-and-entrypoints.md) | 3 |
| [Identity providers](identity-providers.md) | 3 |
| [Installation input parsers](installation-input-parsers.md) | 4 |
| [Installation native workflow](installation-native-workflow.md) | 10 |
| [Installation operator enrollment](installation-operator-enrollment.md) | 3 |
| [Project system definitions](project-system-definitions.md) | 2 |
| [Public handbook](public-handbook.md) | 12 |
| [Root and retired definitions](root-and-retired-definitions.md) | 3 |
| [Server entrypoints](server-entrypoints.md) | 7 |
| [Soda activate](soda-activate.md) | 1 |
| [Soda asset fetchers](soda-asset-fetchers.md) | 2 |
| [Soda candidate setup](soda-candidate-setup.md) | 2 |
| [Soda console welcome](soda-console-welcome.md) | 1 |
| [Soda factory](soda-factory.md) | 1 |
| [Soda forgejo domain](soda-forgejo-domain.md) | 1 |
| [Soda forgejo locales](soda-forgejo-locales.md) | 3 |
| [Soda forgejo migrate](soda-forgejo-migrate.md) | 1 |
| [Soda identity compose](soda-identity-compose.md) | 1 |
| [Soda image import](soda-image-import.md) | 1 |
| [Soda muse](soda-muse.md) | 1 |
| [Soda muse maintain](soda-muse-maintain.md) | 1 |
| [Soda pg maintenance](soda-pg-maintenance.md) | 1 |
| [Soda project account](soda-project-account.md) | 2 |
| [Soda project factory roles](soda-project-factory-roles.md) | 9 |
| [Soda project terminal](soda-project-terminal.md) | 10 |
| [Soda release build implementation](soda-release-build-implementation.md) | 8 |
| [Soda release build verification](soda-release-build-verification.md) | 1 |
| [Soda release deliver implementation](soda-release-deliver-implementation.md) | 11 |
| [Soda release deliver verification](soda-release-deliver-verification.md) | 1 |
| [Soda release image implementation](soda-release-image-implementation.md) | 10 |
| [Soda release image verification](soda-release-image-verification.md) | 1 |
| [Soda release tools implementation](soda-release-tools-implementation.md) | 10 |
| [Soda release tools verification](soda-release-tools-verification.md) | 1 |
| [Soda rotate lab creds](soda-rotate-lab-creds.md) | 1 |
| [Soda setup](soda-setup.md) | 1 |
| [Soda stage render](soda-stage-render.md) | 5 |
| [Soda test vm](soda-test-vm.md) | 2 |
| [Tests build](tests-build.md) | 14 |
| [Tests forgejo](tests-forgejo.md) | 10 |
| [Tests frontend](tests-frontend.md) | 14 |
| [Tests installed](tests-installed.md) | 13 |

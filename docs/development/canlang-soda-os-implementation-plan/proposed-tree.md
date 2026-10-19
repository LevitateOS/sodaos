# Scoped production-workerd target tree

This is the source allocation for the [replacement chapter](production-workerd.md#defining-source-map-and-target-allocation). It includes its retained defining/consumer files, proposed leaves and verification joins across both canonical repositories. It is not a complete tree of either repository, and proposed leaves do not exist until implementation. The chapter owns task dependencies and each file’s disposition; no separate task list is defined here.

Generated TypeScript distributions, worker bundles, the native exec-guard binary/manifest and installed production subtree are produced by the named build/packaging owners and validated by P16.1. Dependencies and native licenses are inventoried by the existing archive producer; they are not copied into authored source. Existing development files remain in their owning package. `soda-lifetime.ts` changes responsibility in place; its Miniflare-specific supervisor behavior retires once direct lifetime acceptance passes. No existing source file is unconditionally deleted by this plan.

```text
├── canlang/
│   ├── bun.lock
│   ├── docs/
│   │   ├── ideal-filetree-plan.md
│   │   ├── soda-runtime.md
│   │   └── specification/
│   │       └── DECISIONS.md
│   ├── package.json
│   ├── packages/
│   │   ├── cloudflare/
│   │   │   ├── host/
│   │   │   │   ├── Cargo.lock
│   │   │   │   ├── Cargo.toml
│   │   │   │   ├── rust-toolchain.toml
│   │   │   │   ├── src/
│   │   │   │   │   └── main.rs
│   │   │   │   └── tests/
│   │   │   │       └── lifetime.rs
│   │   │   ├── package.json
│   │   │   ├── scripts/
│   │   │   │   └── build-workerd-exec.mjs
│   │   │   ├── src/
│   │   │   │   ├── cli/
│   │   │   │   │   ├── soda-runtime.ts
│   │   │   │   │   └── soda.ts
│   │   │   │   ├── deploy/
│   │   │   │   │   ├── bundle.ts
│   │   │   │   │   ├── producer-files.ts
│   │   │   │   │   ├── soda-contract.ts
│   │   │   │   │   ├── soda-lifetime.ts
│   │   │   │   │   ├── soda-run.ts
│   │   │   │   │   ├── soda.ts
│   │   │   │   │   └── workerd-config.ts
│   │   │   │   ├── dev/
│   │   │   │   │   └── local-run.ts
│   │   │   │   ├── distribution.ts
│   │   │   │   ├── runtime/
│   │   │   │   │   ├── auth-rate-limiter.ts
│   │   │   │   │   ├── env-assembly.ts
│   │   │   │   │   ├── export-downloads.ts
│   │   │   │   │   ├── page-preferences.ts
│   │   │   │   │   ├── soda-admin.ts
│   │   │   │   │   ├── soda-application.ts
│   │   │   │   │   ├── soda-d1.ts
│   │   │   │   │   └── soda-gateway.ts
│   │   │   │   └── worker/
│   │   │   │       └── main.ts
│   │   │   └── test/
│   │   │       ├── durable-files-sqlite.integration.test.ts
│   │   │       ├── soda-bundle.test.ts
│   │   │       ├── soda-d1.integration.test.ts
│   │   │       ├── soda-lifetime.test.ts
│   │   │       ├── soda-transport.integration.test.ts
│   │   │       ├── soda-writer.integration.test.ts
│   │   │       ├── worker-boundary.test.ts
│   │   │       └── workerd-config.test.ts
│   │   ├── files/
│   │   │   └── src/
│   │   │       └── host/
│   │   │           ├── durable-sqlite.ts
│   │   │           ├── sqlite-core.ts
│   │   │           └── sqlite.ts
│   │   ├── identity/
│   │   │   ├── package.json
│   │   │   └── src/
│   │   │       ├── accounts/
│   │   │       │   └── passwords.ts
│   │   │       ├── sessions/
│   │   │       │   ├── comparison-worker.ts
│   │   │       │   └── comparison.ts
│   │   │       └── storage/
│   │   │           ├── bootstrap.ts
│   │   │           └── d1.ts
│   │   └── state/
│   │       └── src/
│   │           └── storage/
│   │               ├── d1.ts
│   │               ├── owner-router.ts
│   │               ├── schema.ts
│   │               └── sqlite-codecs.ts
│   ├── scripts/
│   │   ├── build-package.mjs
│   │   ├── distribution/
│   │   │   └── licenses/
│   │   │       ├── miniflare-provider-LICENSE
│   │   │       └── workerd-LICENSE
│   │   ├── fixtures/
│   │   │   ├── native-runtime.can
│   │   │   └── soda-runtime.can
│   │   ├── pack-install.mjs
│   │   ├── pack-release.mjs
│   │   ├── pack-tools.mjs
│   │   ├── verify-install.mjs
│   │   ├── verify-native-runtime.mjs
│   │   ├── verify-soda-host.mjs
│   │   ├── verify-soda-recovery.mjs
│   │   ├── verify-soda-runtime-closure.mjs
│   │   ├── verify-soda-runtime.mjs
│   │   └── verify-turbo-cache.mjs
│   └── turbo.json
└── sodaos/
    ├── cmd/
    │   └── soda-project-maintain/
    │       └── src/
    │           └── canlang.rs
    ├── docs/
    │   ├── development/
    │   │   └── canlang-soda-os-implementation-plan/
    │   │       ├── production-workerd.md
    │   │       └── proposed-tree.md
    │   └── guides/
    │       └── canlang-applications.md
    ├── lib/
    │   ├── host/
    │   │   └── src/
    │   │       ├── application/
    │   │       │   ├── ingress.rs
    │   │       │   ├── runtime.rs
    │   │       │   ├── snapshot.rs
    │   │       │   └── wire.rs
    │   │       └── bin/
    │   │           └── soda-canlang-storage.rs
    │   ├── release-inputs/
    │   │   └── src/
    │   │       ├── payload.rs
    │   │       └── reader/
    │   │           └── candidate.rs
    │   ├── soda-release-build/
    │   │   └── src/
    │   │       ├── production_assets.rs
    │   │       ├── production_images.rs
    │   │       └── production_inputs.rs
    │   └── soda-release-image/
    │       └── src/
    │           ├── canlang.rs
    │           ├── payload_stage.rs
    │           └── record.rs
    ├── system/
    │   ├── containers/
    │   │   └── canlang/
    │   │       ├── Application.Containerfile
    │   │       ├── Containerfile
    │   │       └── verify-runtime-image.mjs
    │   ├── host/
    │   │   └── services/
    │   │       └── soda-canlang@.service
    │   └── project/
    │       ├── canlang-release.json
    │       └── Containerfile
    └── tools/
        └── release-assets/
            └── src/
                └── fetch/
                    └── canlang.rs
```

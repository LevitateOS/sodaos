//! Selected current behavior expectations and retained fixture values.
//!
//! Some values originated in the former Go owner. They remain only where a
//! current producer or consumer assertion uses them; they do not require
//! general byte-for-byte compatibility with that implementation.

#![allow(dead_code)]

pub const OCI_MANIFEST: &str =
    "sha256:098b60ba449c4b81d38cca87e08b16ff83522b9edb36bdb36025d9d370a99295";
pub const OCI_CONFIG: &str =
    "sha256:9265b4ccf05645aeded07db50e8b29fd99e83b6b4994a4ae9e952aca54c3f691";
pub const OCI_ARCH: &str = "amd64";
pub const OCI_REVISION: &str = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa";
pub const OCI_SOURCE: &str = "https://github.com/LevitateOS/sodaos";
pub const OCI_BASENAME: &str = "synthetic-base";
pub const OCI_BASEDIGEST: &str =
    "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb";
pub const OCI_FIXTURE_HASH: &str =
    "0ab43d832955b51125a34a1069f654d5b67125d776f9009738e126417984103c";

// The Fountain compile argv/script vectors are removed with the producer's
// Fountain checkout/toolchain/compile/stage-binary dependency: admission
// consumes the independently released Fountain image and metadata instead.
pub const PRODUCTION_SEQUENCE: &str = r###"bun --version
go mod verify
bun install --frozen-lockfile
podman --remote=false pull --quiet --platform=linux/amd64 docker.io/rockylinux/rockylinux:10.2
podman --remote=false image inspect --format {{.Digest}} sha256:9265b4ccf05645aeded07db50e8b29fd99e83b6b4994a4ae9e952aca54c3f691
podman --remote=false pull --quiet --platform=linux/amd64 codeberg.org/forgejo/forgejo:15.0.10
podman --remote=false image inspect --format {{.Digest}} sha256:9265b4ccf05645aeded07db50e8b29fd99e83b6b4994a4ae9e952aca54c3f691
podman --remote=false pull --quiet --platform=linux/amd64 docker.io/library/caddy:2
podman --remote=false image inspect --format {{.Digest}} sha256:9265b4ccf05645aeded07db50e8b29fd99e83b6b4994a4ae9e952aca54c3f691
podman --remote=false pull --quiet --platform=linux/amd64 docker.io/library/postgres:17
podman --remote=false image inspect --format {{.Digest}} sha256:9265b4ccf05645aeded07db50e8b29fd99e83b6b4994a4ae9e952aca54c3f691
podman --remote=false pull --quiet --platform=linux/amd64 docker.io/tailscale/tailscale@sha256:99691662a2e82fbfd2bf55db0fd52bacd2f4664ce6fd1deca3746a024144a2b7
podman --remote=false image inspect --format {{.Digest}} sha256:9265b4ccf05645aeded07db50e8b29fd99e83b6b4994a4ae9e952aca54c3f691
cargo build --release --locked --offline --config profile.release.strip="symbols" --manifest-path $ROOT/Cargo.toml -p soda-muse --bin soda-muse
cargo build --release --locked --offline --config profile.release.strip="symbols" --manifest-path $ROOT/Cargo.toml -p soda-identity-compose --bin soda-identity-compose
cargo build --release --locked --offline --config profile.release.strip="symbols" --manifest-path $ROOT/Cargo.toml -p soda-project-terminal --bin project-terminal
cargo build --release --locked --offline --config profile.release.strip="symbols" --manifest-path $ROOT/Cargo.toml -p soda-project-terminal --bin project-account
cargo build --release --locked --offline --config profile.release.strip="symbols" --manifest-path $ROOT/Cargo.toml -p soda-project-terminal --bin project-factory-roles
bun scripts/build-forgejo.ts --out $ROOT/.artifacts/native/x86_64/forgejo-js
cargo run --release --locked -p soda-release-assets --bin soda-fetch-terminal -- --out $ROOT/.artifacts/native/x86_64/terminal-assets
bun scripts/build-soda-extension.ts --package=soda --out $ROOT/.artifacts/native/x86_64/soda-extension-assets --terminal-assets $ROOT/.artifacts/native/x86_64/terminal-assets
bun scripts/build-soda-extension.ts --package=tailnet --out $ROOT/.artifacts/native/x86_64/tailnet-extension-assets --terminal-assets $ROOT/.artifacts/native/x86_64/terminal-assets
cargo run --release --locked -p soda-release-assets --bin soda-forgejo-locales -- --lock frontend/forgejo/locale.lock.json --out $ROOT/.artifacts/native/x86_64/forgejo-locales/locale_en-US.ini
cargo run --release --locked -p soda-release-assets --bin soda-fetch-muse -- --arch x86_64 --out $ROOT/.artifacts/native/x86_64/project-tools/bin/muse-native
cargo run --release --locked -p soda-release-assets --bin soda-stage-canlang -- --archive $ROOT/.artifacts/canlang/canlang-tools-linux-x86_64.tar.gz --out $ROOT/.artifacts/native/x86_64/project-tools
cargo run --release --locked -p soda-release-assets --bin soda-fetch-tea -- --arch x86_64 --out $ROOT/.artifacts/native/x86_64/project-tools
cargo run --release --locked -p soda-release-assets --bin soda-stage -- --arch x86_64 --host-context $ROOT/.artifacts/native/x86_64/context --forgejo-context $ROOT/.artifacts/native/x86_64/forgejo-context
podman --remote=false build --pull=never --rm=true --force-rm=false --platform=linux/amd64 --build-arg=BASE_IMAGE=docker.io/rockylinux/rockylinux@sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb --label=org.opencontainers.image.revision=aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa --label=org.opencontainers.image.source=https://github.com/LevitateOS/sodaos --label=org.opencontainers.image.base.name=docker.io/rockylinux/rockylinux@sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb --label=org.opencontainers.image.base.digest=sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb --iidfile $ROOT/.artifacts/native/x86_64/dashboard.iid --file system/containers/dashboard/Containerfile --build-arg=ARTIFACT_DIR=.artifacts/native/x86_64 .
skopeo copy --format oci --dest-compress-format gzip --dest-compress-level 1 containers-storage:sha256:9265b4ccf05645aeded07db50e8b29fd99e83b6b4994a4ae9e952aca54c3f691 oci-archive:$ROOT/.artifacts/native/x86_64/images/dashboard.oci
podman --remote=false build --pull=never --rm=true --force-rm=false --platform=linux/amd64 --build-arg=BASE_IMAGE=docker.io/rockylinux/rockylinux@sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb --label=org.opencontainers.image.revision=aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa --label=org.opencontainers.image.source=https://github.com/LevitateOS/sodaos --label=org.opencontainers.image.base.name=docker.io/rockylinux/rockylinux@sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb --label=org.opencontainers.image.base.digest=sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb --iidfile $ROOT/.artifacts/native/x86_64/project-os.iid --file system/project/Containerfile --build-arg=ARTIFACT_DIR=.artifacts/native/x86_64 .
skopeo copy --format oci --dest-compress-format gzip --dest-compress-level 1 containers-storage:sha256:9265b4ccf05645aeded07db50e8b29fd99e83b6b4994a4ae9e952aca54c3f691 oci-archive:$ROOT/.artifacts/native/x86_64/images/project-os.oci
podman --remote=false build --pull=never --rm=true --force-rm=false --platform=linux/amd64 --build-arg=BASE_IMAGE=docker.io/rockylinux/rockylinux@sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb --label=org.opencontainers.image.revision=aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa --label=org.opencontainers.image.source=https://github.com/LevitateOS/sodaos --label=org.opencontainers.image.base.name=docker.io/rockylinux/rockylinux@sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb --label=org.opencontainers.image.base.digest=sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb --iidfile $ROOT/.artifacts/native/x86_64/canlang-runtime.iid --file system/containers/canlang/Containerfile --build-arg=ARTIFACT_DIR=.artifacts/native/x86_64 .
skopeo copy --format oci --dest-compress-format gzip --dest-compress-level 1 containers-storage:sha256:9265b4ccf05645aeded07db50e8b29fd99e83b6b4994a4ae9e952aca54c3f691 oci-archive:$ROOT/.artifacts/native/x86_64/images/canlang-runtime.oci
podman --remote=false build --pull=never --rm=true --force-rm=false --platform=linux/amd64 --build-arg=BASE_IMAGE=codeberg.org/forgejo/forgejo@sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb --label=org.opencontainers.image.revision=aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa --label=org.opencontainers.image.source=https://github.com/LevitateOS/sodaos --label=org.opencontainers.image.base.name=codeberg.org/forgejo/forgejo@sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb --label=org.opencontainers.image.base.digest=sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb --iidfile $ROOT/.artifacts/native/x86_64/forgejo.iid --file Containerfile .
skopeo copy --format oci --dest-compress-format gzip --dest-compress-level 1 containers-storage:sha256:9265b4ccf05645aeded07db50e8b29fd99e83b6b4994a4ae9e952aca54c3f691 oci-archive:$ROOT/.artifacts/native/x86_64/images/forgejo.oci
"###;
pub const MISC_HTTPS_HTTP: bool = false;
pub const READ_OK_MODE: u32 = 420;

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

pub const FORGEJO_ARGV: &str = r###"--remote=false
run
--rm
--pull=always
--platform=linux/amd64
--volume=$ROOT/frozen-fork:/work/source:Z
--volume=$ROOT/native/forgejo-build:/work/out:Z
--workdir=/work/source
--env=GOCACHE=/work/out/gocache
--env=GOPATH=/work/out/gopath
--env=TMPDIR=/work/out/tmp
--env=FORGEJO_VERSION=15.0.9-soda.aaaaaaaaaaaa+gitea-1.22.0
docker.io/library/golang@sha256:28d89ee9cc0ff9fec75c82ca201e6bf7fdf9a679d4b7b24dfa04f2bb766bb468
sh
-ec
"###;
pub const FORGEJO_SCRIPT: &str = r###"set -eu
apk add --no-cache build-base=0.5-r4
wget -O /tmp/bun.zip "https://github.com/oven-sh/bun/releases/download/bun-v1.4.2/bun-linux-x64-musl.zip"
echo "4835eca59d6da70f4674f5642f6e459dcadab773695b2ed9922d131057989742  /tmp/bun.zip" | sha256sum -c -
rm -rf /tmp/bun-extract && mkdir -p /tmp/bun-extract
unzip -q -o -d /tmp/bun-extract /tmp/bun.zip
install -m 755 /tmp/bun-extract/bun-linux-x64-musl/bun /usr/local/bin/bun
test "$(bun --version)" = "1.4.2"
mkdir -p /work/out/gocache /work/out/gopath /work/out/tmp
LC_ALL=C apk info -v | LC_ALL=C sort > /work/out/apk-packages.txt
# The git archive never contains built frontend outputs; generate them with
# the pinned Bun toolchain before bindata embeds public/. The frozen install
# resolves from the archived package lock; Bun is the only JS toolchain.
bun install --frozen-lockfile --no-progress
BROWSERSLIST_IGNORE_OLD_DATA=true bun ./node_modules/.bin/webpack
test -s public/assets/js/index.js
test -s public/assets/css/index.css
for package in options public templates migration; do
  (cd modules/$package && go generate -tags bindata .)
done
LDFLAGS=""
if [ -n "${FORGEJO_VERSION:-}" ]; then
  LDFLAGS="-X main.Version=${FORGEJO_VERSION} -X main.ForgejoVersion=${FORGEJO_VERSION} -X main.ReleaseVersion=${FORGEJO_VERSION} -X \"main.Tags=bindata sqlite sqlite_unlock_notify\""
fi
go build -buildvcs=false -tags 'bindata sqlite sqlite_unlock_notify' -ldflags "${LDFLAGS}" -trimpath -o /work/out/forgejo-bin .
"###;
pub const PRODUCTION_SEQUENCE: &str = r###"bun --version
go mod verify
bun install --frozen-lockfile
podman --remote=false pull --quiet --platform=linux/amd64 docker.io/rockylinux/rockylinux:10.2
podman --remote=false image inspect --format {{.Digest}} sha256:9265b4ccf05645aeded07db50e8b29fd99e83b6b4994a4ae9e952aca54c3f691
podman --remote=false pull --quiet --platform=linux/amd64 codeberg.org/forgejo/forgejo:15.0.9
podman --remote=false image inspect --format {{.Digest}} sha256:9265b4ccf05645aeded07db50e8b29fd99e83b6b4994a4ae9e952aca54c3f691
podman --remote=false pull --quiet --platform=linux/amd64 docker.io/library/caddy:2
podman --remote=false image inspect --format {{.Digest}} sha256:9265b4ccf05645aeded07db50e8b29fd99e83b6b4994a4ae9e952aca54c3f691
podman --remote=false pull --quiet --platform=linux/amd64 docker.io/tailscale/alpine-base:3.22
podman --remote=false image inspect --format {{.Digest}} sha256:9265b4ccf05645aeded07db50e8b29fd99e83b6b4994a4ae9e952aca54c3f691
cargo build --release --locked --manifest-path $ROOT/Cargo.toml -p soda-muse --bin soda-muse
cargo build --release --locked --manifest-path $ROOT/Cargo.toml -p soda-identity-compose --bin soda-identity-compose
cargo build --release --locked --manifest-path $ROOT/Cargo.toml -p soda-project-terminal --bin project-terminal
cargo build --release --locked --manifest-path $ROOT/Cargo.toml -p soda-project-terminal --bin project-account
cargo build --release --locked --manifest-path $ROOT/Cargo.toml -p soda-project-terminal --bin project-factory-roles
bun scripts/build-forgejo.ts --out $ROOT/.artifacts/native/x86_64/forgejo-js
cargo run --release --locked -p soda-release-assets --bin soda-fetch-terminal -- --out $ROOT/.artifacts/native/x86_64/terminal-assets
bun scripts/build-soda-extension.ts --out $ROOT/.artifacts/native/x86_64/soda-extension-assets --terminal-assets $ROOT/.artifacts/native/x86_64/terminal-assets
cargo run --release --locked -p soda-release-assets --bin soda-forgejo-locales -- --lock frontend/forgejo/locale.lock.json --out $ROOT/.artifacts/native/x86_64/forgejo-locales/locale_en-US.ini
cargo run --release --locked -p soda-release-assets --bin soda-fetch-muse -- --arch x86_64 --out $ROOT/.artifacts/native/x86_64/project-tools/bin/muse-native
cargo run --release --locked -p soda-release-assets --bin soda-fetch-tea -- --arch x86_64 --out $ROOT/.artifacts/native/x86_64/project-tools
cargo run --release --locked -p soda-release-assets --bin soda-stage -- --arch x86_64 --host-context $ROOT/.artifacts/native/x86_64/context --forgejo-context $ROOT/.artifacts/native/x86_64/forgejo-context
podman --remote=false build --pull=never --rm=false --platform=linux/amd64 --build-arg=BASE_IMAGE=docker.io/rockylinux/rockylinux@sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb --label=org.opencontainers.image.revision=aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa --label=org.opencontainers.image.source=https://github.com/LevitateOS/sodaos --label=org.opencontainers.image.base.name=docker.io/rockylinux/rockylinux@sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb --label=org.opencontainers.image.base.digest=sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb --iidfile $ROOT/.artifacts/native/x86_64/dashboard.iid --file system/containers/dashboard/Containerfile --build-arg=ARTIFACT_DIR=.artifacts/native/x86_64 .
podman --remote=false save --format=oci-archive --output $ROOT/.artifacts/native/x86_64/images/dashboard.oci sha256:9265b4ccf05645aeded07db50e8b29fd99e83b6b4994a4ae9e952aca54c3f691
podman --remote=false build --pull=never --rm=false --platform=linux/amd64 --build-arg=BASE_IMAGE=docker.io/rockylinux/rockylinux@sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb --label=org.opencontainers.image.revision=aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa --label=org.opencontainers.image.source=https://github.com/LevitateOS/sodaos --label=org.opencontainers.image.base.name=docker.io/rockylinux/rockylinux@sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb --label=org.opencontainers.image.base.digest=sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb --iidfile $ROOT/.artifacts/native/x86_64/project-os.iid --file system/project/Containerfile --build-arg=ARTIFACT_DIR=.artifacts/native/x86_64 .
podman --remote=false save --format=oci-archive --output $ROOT/.artifacts/native/x86_64/images/project-os.oci sha256:9265b4ccf05645aeded07db50e8b29fd99e83b6b4994a4ae9e952aca54c3f691
podman --remote=false build --pull=never --rm=false --platform=linux/amd64 --build-arg=BASE_IMAGE=codeberg.org/forgejo/forgejo@sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb --label=org.opencontainers.image.revision=aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa --label=org.opencontainers.image.source=https://github.com/LevitateOS/sodaos --label=org.opencontainers.image.base.name=codeberg.org/forgejo/forgejo@sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb --label=org.opencontainers.image.base.digest=sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb --iidfile $ROOT/.artifacts/native/x86_64/forgejo.iid --file Containerfile .
podman --remote=false save --format=oci-archive --output $ROOT/.artifacts/native/x86_64/images/forgejo.oci sha256:9265b4ccf05645aeded07db50e8b29fd99e83b6b4994a4ae9e952aca54c3f691
podman --remote=false build --pull=never --rm=false --platform=linux/amd64 --build-arg=BASE_IMAGE=sha256:9265b4ccf05645aeded07db50e8b29fd99e83b6b4994a4ae9e952aca54c3f691 --label=org.opencontainers.image.revision=aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa --label=org.opencontainers.image.source=https://github.com/LevitateOS/sodaos --label=org.opencontainers.image.base.name=sha256:9265b4ccf05645aeded07db50e8b29fd99e83b6b4994a4ae9e952aca54c3f691 --label=org.opencontainers.image.base.digest=sha256:9265b4ccf05645aeded07db50e8b29fd99e83b6b4994a4ae9e952aca54c3f691 --iidfile $ROOT/.artifacts/native/x86_64/extension.iid --file Containerfile .
podman --remote=false save --format=oci-archive --output $ROOT/.artifacts/native/x86_64/images/extension.oci sha256:9265b4ccf05645aeded07db50e8b29fd99e83b6b4994a4ae9e952aca54c3f691
podman --remote=false save --format=oci-archive --output $ROOT/.artifacts/native/x86_64/images/proxy.oci sha256:9265b4ccf05645aeded07db50e8b29fd99e83b6b4994a4ae9e952aca54c3f691
podman --remote=false build --pull=never --rm=false --platform=linux/amd64 --build-arg=BASE_IMAGE=docker.io/tailscale/alpine-base@sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb --label=org.opencontainers.image.revision=aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa --label=org.opencontainers.image.source=https://github.com/LevitateOS/sodaos --label=org.opencontainers.image.base.name=docker.io/tailscale/alpine-base@sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb --label=org.opencontainers.image.base.digest=sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb --iidfile $ROOT/.artifacts/native/x86_64/tailnet.iid --file system/containers/tailnet/Containerfile --build-arg=TAILSCALE_VERSION=1.98.2 --build-arg=TARGETARCH=amd64 --build-arg=ARCHIVE_SHA256=eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee .
podman --remote=false save --format=oci-archive --output $ROOT/.artifacts/native/x86_64/images/tailnet.oci sha256:9265b4ccf05645aeded07db50e8b29fd99e83b6b4994a4ae9e952aca54c3f691
"###;
pub const MISC_HTTPS_HTTP: bool = false;
pub const READ_OK_MODE: u32 = 420;

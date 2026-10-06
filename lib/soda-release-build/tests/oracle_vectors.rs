//! Oracle vectors generated from the Go owner (`internal/release/build`).
//!
//! Produced by the throwaway `zz-oracle-tmp` driver (deleted after
//! generation; see tests/oracle.rs). Every constant below is Go ground
//! truth: byte outputs or error texts the Rust port must reproduce.

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
pub const OCI_ERR_ARCH: &str = "expected x86_64";
pub const OCI_ERR_REVISION: &str = "OCI source revision mismatch";
pub const OCI_ERR_TAMPERED: &str = "OCI blob checksum mismatch";
pub const OCI_FIXTURE_HASH: &str =
    "0ab43d832955b51125a34a1069f654d5b67125d776f9009738e126417984103c";
pub const OCI_ERR_MISSING: &str = "requested OCI member missing";
pub const LAYOUT_IMAGES: usize = 1;
pub const LAYOUT_FILES: usize = 5;
pub const LAYOUT_BYTES: u64 = 3222;
pub const LAYOUT_INDEXHASH: &str =
    "da95ff130c78dc61e49e1588584f917244d46d747c5a4c2f46ae8d05fbc62371";
pub const LAYOUT_ERR_ARCH: &str = "expected x86_64";

pub const LIVE_INPUTS_JSON: &str = r###"{
  "CoreOS": {
    "Release": "44.20260901.1.0",
    "MetadataURL": "https://builds.test/prod/streams/stable/builds/44.20260901.1.0/release.json",
    "Container": {
      "x86_64": "quay.io/fedora/fedora-coreos@sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"
    },
    "ISO": {
      "x86_64": {
        "URL": "https://builds.test/fedora-coreos-44.20260901.1.0-live.x86_64.iso",
        "SignatureURL": "https://builds.test/fedora-coreos-44.20260901.1.0-live.x86_64.iso.sig",
        "SHA256": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
        "UncompressedSHA256": "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
      }
    },
    "QEMU": {
      "x86_64": {
        "URL": "https://builds.test/fedora-coreos-44.20260901.1.0-live.x86_64.iso",
        "SignatureURL": "https://builds.test/fedora-coreos-44.20260901.1.0-live.x86_64.iso.sig",
        "SHA256": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
        "UncompressedSHA256": "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
      }
    }
  },
  "Tailnet": {
    "Version": "1.98.2",
    "SHA256": "eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee",
    "Base": "docker.io/tailscale/alpine-base:3.22"
  }
}
"###;
pub const LIVE_ERR_TAILNET: &str = "invalid Tailnet version";
pub const LIVE_ERR_RELEASE: &str = "stable stream release is malformed";
pub const LIVE_ERR_CONTAINER: &str = "x86_64 base digest required";

pub const VERIFIED_BASE_JSON: &str = r###"{
  "Path": "$OUT/coreos.qcow2",
  "SHA256": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
  "Architecture": "x86_64",
  "Release": "44.20260901.1.0",
  "Signer": "ABC"
}
"###;
pub const RESOLVED_INPUTS_JSON: &str = r###"[
  {
    "Requested": "docker.io/rockylinux/rockylinux:10.2",
    "Reference": "docker.io/rockylinux/rockylinux@sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
    "Config": "sha256:9265b4ccf05645aeded07db50e8b29fd99e83b6b4994a4ae9e952aca54c3f691"
  }
]
"###;

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
pub const FORGEJO_TOOLCHAIN_JSON: &str = r###"{
  "CompilerImage": "docker.io/library/golang@sha256:28d89ee9cc0ff9fec75c82ca201e6bf7fdf9a679d4b7b24dfa04f2bb766bb468",
  "APKPackages": [
    "build-base-0.5-r4",
    "gcc-14.2.0-r6",
    "musl-dev-1.2.5-r10"
  ]
}
"###;

pub const PRODUCTION_SEQUENCE: &str = r###"STEP Check frontend toolchain
bun --version
STEP Verify Go dependencies
go mod verify
STEP Install frontend dependencies
bun install --frozen-lockfile
STEP Pull and resolve docker.io/rockylinux/rockylinux:10.2
podman --remote=false pull --quiet --platform=linux/amd64 docker.io/rockylinux/rockylinux:10.2
podman --remote=false image inspect --format {{.Digest}} sha256:9265b4ccf05645aeded07db50e8b29fd99e83b6b4994a4ae9e952aca54c3f691
STEP Pull and resolve codeberg.org/forgejo/forgejo:15.0.9
podman --remote=false pull --quiet --platform=linux/amd64 codeberg.org/forgejo/forgejo:15.0.9
podman --remote=false image inspect --format {{.Digest}} sha256:9265b4ccf05645aeded07db50e8b29fd99e83b6b4994a4ae9e952aca54c3f691
STEP Pull and resolve docker.io/library/caddy:2
podman --remote=false pull --quiet --platform=linux/amd64 docker.io/library/caddy:2
podman --remote=false image inspect --format {{.Digest}} sha256:9265b4ccf05645aeded07db50e8b29fd99e83b6b4994a4ae9e952aca54c3f691
STEP Pull and resolve docker.io/tailscale/alpine-base:3.22
podman --remote=false pull --quiet --platform=linux/amd64 docker.io/tailscale/alpine-base:3.22
podman --remote=false image inspect --format {{.Digest}} sha256:9265b4ccf05645aeded07db50e8b29fd99e83b6b4994a4ae9e952aca54c3f691
STEP Compile soda-muse
cargo build --release --locked --manifest-path $ROOT/Cargo.toml -p soda-muse
STEP Compile soda-identity-compose
cargo build --release --locked --manifest-path $ROOT/Cargo.toml -p soda-identity-compose
STEP Compile project-terminal
cargo build --release --locked --manifest-path $ROOT/Cargo.toml -p soda-project-terminal
STEP Compile project-account
cargo build --release --locked --manifest-path $ROOT/Cargo.toml -p soda-project-account
STEP Compile project-factory-roles
cargo build --release --locked --manifest-path $ROOT/Cargo.toml -p soda-project-factory-roles
STEP Build frontend assets
bun scripts/build-forgejo.ts --out $ROOT/.artifacts/native/x86_64/forgejo-js
STEP Fetch terminal assets
cargo run --release --locked -p soda-asset-fetchers --bin soda-fetch-terminal -- --out $ROOT/.artifacts/native/x86_64/terminal-assets
STEP Build Soda extension browser assets
bun scripts/build-soda-extension.ts --out $ROOT/.artifacts/native/x86_64/soda-extension-assets --terminal-assets $ROOT/.artifacts/native/x86_64/terminal-assets
STEP Prepare Forgejo translations
cargo run --release --locked -p soda-forgejo-locales --bin soda-forgejo-locales -- --lock appliance/forgejo/locale.lock.json --out $ROOT/.artifacts/native/x86_64/forgejo-locales/locale_en-US.ini
STEP Fetch upstream Muse binary
cargo run --release --locked -p soda-asset-fetchers --bin soda-fetch-muse -- --arch x86_64 --out $ROOT/.artifacts/native/x86_64/project-tools/bin/muse-native
STEP Fetch upstream Tea binary
cargo run --release --locked -p soda-asset-fetchers --bin soda-fetch-tea -- --arch x86_64 --out $ROOT/.artifacts/native/x86_64/project-tools
STEP Stage appliance files
cargo run --release --locked -p soda-stage-render --bin soda-stage -- --arch x86_64 --host-context $ROOT/.artifacts/native/x86_64/context --forgejo-context $ROOT/.artifacts/native/x86_64/forgejo-context
STEP Select frozen Rocky base
STEP Build image: dashboard
podman --remote=false build --pull=never --rm=false --platform=linux/amd64 --build-arg=BASE_IMAGE=docker.io/rockylinux/rockylinux@sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb --label=org.opencontainers.image.revision=aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa --label=org.opencontainers.image.source=https://github.com/LevitateOS/sodaos --label=org.opencontainers.image.base.name=docker.io/rockylinux/rockylinux@sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb --label=org.opencontainers.image.base.digest=sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb --iidfile $ROOT/.artifacts/native/x86_64/dashboard.iid --file system/containers/dashboard/Containerfile --build-arg=ARTIFACT_DIR=.artifacts/native/x86_64 .
STEP Export and verify image: dashboard
podman --remote=false save --format=oci-archive --output $ROOT/.artifacts/native/x86_64/images/dashboard.oci sha256:9265b4ccf05645aeded07db50e8b29fd99e83b6b4994a4ae9e952aca54c3f691
STEP Build image: project-os
podman --remote=false build --pull=never --rm=false --platform=linux/amd64 --build-arg=BASE_IMAGE=docker.io/rockylinux/rockylinux@sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb --label=org.opencontainers.image.revision=aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa --label=org.opencontainers.image.source=https://github.com/LevitateOS/sodaos --label=org.opencontainers.image.base.name=docker.io/rockylinux/rockylinux@sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb --label=org.opencontainers.image.base.digest=sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb --iidfile $ROOT/.artifacts/native/x86_64/project-os.iid --file system/project/Containerfile --build-arg=ARTIFACT_DIR=.artifacts/native/x86_64 .
STEP Export and verify image: project-os
podman --remote=false save --format=oci-archive --output $ROOT/.artifacts/native/x86_64/images/project-os.oci sha256:9265b4ccf05645aeded07db50e8b29fd99e83b6b4994a4ae9e952aca54c3f691
STEP Select frozen Forgejo
STEP Build image: forgejo
podman --remote=false build --pull=never --rm=false --platform=linux/amd64 --build-arg=BASE_IMAGE=codeberg.org/forgejo/forgejo@sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb --label=org.opencontainers.image.revision=aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa --label=org.opencontainers.image.source=https://github.com/LevitateOS/sodaos --label=org.opencontainers.image.base.name=codeberg.org/forgejo/forgejo@sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb --label=org.opencontainers.image.base.digest=sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb --iidfile $ROOT/.artifacts/native/x86_64/forgejo.iid --file Containerfile .
STEP Export and verify image: forgejo
podman --remote=false save --format=oci-archive --output $ROOT/.artifacts/native/x86_64/images/forgejo.oci sha256:9265b4ccf05645aeded07db50e8b29fd99e83b6b4994a4ae9e952aca54c3f691
STEP Build image: extension
podman --remote=false build --pull=never --rm=false --platform=linux/amd64 --build-arg=BASE_IMAGE=sha256:9265b4ccf05645aeded07db50e8b29fd99e83b6b4994a4ae9e952aca54c3f691 --label=org.opencontainers.image.revision=aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa --label=org.opencontainers.image.source=https://github.com/LevitateOS/sodaos --label=org.opencontainers.image.base.name=sha256:9265b4ccf05645aeded07db50e8b29fd99e83b6b4994a4ae9e952aca54c3f691 --label=org.opencontainers.image.base.digest=sha256:9265b4ccf05645aeded07db50e8b29fd99e83b6b4994a4ae9e952aca54c3f691 --iidfile $ROOT/.artifacts/native/x86_64/extension.iid --file Containerfile .
STEP Export and verify image: extension
podman --remote=false save --format=oci-archive --output $ROOT/.artifacts/native/x86_64/images/extension.oci sha256:9265b4ccf05645aeded07db50e8b29fd99e83b6b4994a4ae9e952aca54c3f691
STEP Select frozen Proxy
STEP Export and verify image: proxy
podman --remote=false save --format=oci-archive --output $ROOT/.artifacts/native/x86_64/images/proxy.oci sha256:9265b4ccf05645aeded07db50e8b29fd99e83b6b4994a4ae9e952aca54c3f691
STEP Select frozen Tailnet base
STEP Build image: tailnet
podman --remote=false build --pull=never --rm=false --platform=linux/amd64 --build-arg=BASE_IMAGE=docker.io/tailscale/alpine-base@sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb --label=org.opencontainers.image.revision=aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa --label=org.opencontainers.image.source=https://github.com/LevitateOS/sodaos --label=org.opencontainers.image.base.name=docker.io/tailscale/alpine-base@sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb --label=org.opencontainers.image.base.digest=sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb --iidfile $ROOT/.artifacts/native/x86_64/tailnet.iid --file system/containers/tailnet/Containerfile --build-arg=TAILSCALE_VERSION=1.98.2 --build-arg=TARGETARCH=amd64 --build-arg=ARCHIVE_SHA256=eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee .
STEP Export and verify image: tailnet
podman --remote=false save --format=oci-archive --output $ROOT/.artifacts/native/x86_64/images/tailnet.oci sha256:9265b4ccf05645aeded07db50e8b29fd99e83b6b4994a4ae9e952aca54c3f691
"###;
pub const APP_INPUTS_JSON: &str = r###"[
  {
    "Requested": "docker.io/rockylinux/rockylinux:10.2",
    "Reference": "docker.io/rockylinux/rockylinux@sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
    "Config": "sha256:9265b4ccf05645aeded07db50e8b29fd99e83b6b4994a4ae9e952aca54c3f691"
  },
  {
    "Requested": "codeberg.org/forgejo/forgejo:15.0.9",
    "Reference": "codeberg.org/forgejo/forgejo@sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
    "Config": "sha256:9265b4ccf05645aeded07db50e8b29fd99e83b6b4994a4ae9e952aca54c3f691"
  },
  {
    "Requested": "docker.io/library/caddy:2",
    "Reference": "docker.io/library/caddy@sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
    "Config": "sha256:9265b4ccf05645aeded07db50e8b29fd99e83b6b4994a4ae9e952aca54c3f691"
  },
  {
    "Requested": "docker.io/tailscale/alpine-base:3.22",
    "Reference": "docker.io/tailscale/alpine-base@sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
    "Config": "sha256:9265b4ccf05645aeded07db50e8b29fd99e83b6b4994a4ae9e952aca54c3f691"
  }
]
"###;

pub const PROGRESS_BYTES: &str = r###"LOG      $TMP/timing.log
START    P3
START    Compile once
DONE     Compile once | section 00:00:03 | total 00:00:00
DONE     P3 | phase 00:00:05 | total 00:00:00
START    P4
FAILED   P4 | phase 00:00:04 | total 00:00:00

SECTION SUMMARY
  DONE     Compile once | section 00:00:03 | total 00:00:00
  DONE     P3 | phase 00:00:05 | total 00:00:00
  FAILED   P4 | phase 00:00:04 | total 00:00:00
FAILED   Release fixture | total 00:00:00 | exit 1
"###;

pub const EXIT_NIL: i32 = 0;
pub const EXIT_7: i32 = 7;
pub const EXIT_SIGTERM: i32 = 143;
pub const EXIT_CANCELED: i32 = 130;
pub const EXIT_GENERIC: i32 = 1;
pub const MISC_OCI_ARCH: &str = "expected x86_64";
pub const MISC_HTTPS_HTTP: bool = false;
pub const MISC_TAILNET: &str = "invalid Tailnet version";
pub const READ_UNKNOWN_FIELD: &str = r###"json: unknown field "bogus""###;
pub const READ_TRAILING: &str = "trailing JSON data";
pub const READ_OK_MODE: u32 = 420;

# Local host-content candidate, not yet an installable Soda release. The caller
# supplies the architecture-specific digest from locks/coreos-host.json.
ARG BASE_IMAGE
FROM ${BASE_IMAGE}

# Keep the existing provisioning package list as the source of authority. The
# preparation command validates it and emits this argument file and repository URL.
COPY packages.list tailscale-repo.url /run/soda-build/
RUN curl --fail --show-error --location "$(cat /run/soda-build/tailscale-repo.url)" \
      --output /etc/yum.repos.d/tailscale.repo && \
    rpm-ostree install $(cat /run/soda-build/packages.list) && \
    mkdir -p /usr/share/soda/host-image && \
    rpm -qa --qf '%{NAME} %{EPOCHNUM}:%{VERSION}-%{RELEASE}.%{ARCH}\n' > /run/soda-build/packages.unsorted && \
    LC_ALL=C sort /run/soda-build/packages.unsorted > /usr/share/soda/host-image/packages.txt

COPY rootfs/ /
# forgejo-runner-12.13.2-1.fc44 ships whitespace-only tab lines that bootc
# 1.16.7's sysusers parser rejects. Drop only blank lines, preserving every
# account field and directive. The original package remains in build evidence.
RUN sed -i '/^[[:space:]]*$/d' /usr/lib/sysusers.d/forgejo-runner.conf
# Do not enable a competing automatic updater in the candidate. These changes
# affect only the built image, never the builder's services or trust configuration.
# Lint resolves tmpfiles paths, including /sys entries. Give this rootless build
# step an empty /sys rather than the host's protected IMA filesystem. This is a
# build-only mount, not a shipped mount or a skipped lint/privileged build.
RUN --mount=type=tmpfs,target=/sys systemctl mask bootc-fetch-apply-updates.timer && \
    ostree container commit && \
    bootc container lint

LABEL org.opencontainers.image.title="SodaOS host-content candidate" \
      org.opencontainers.image.source="https://github.com/LevitateOS/sodaos" \
      io.soda.host-image.scope="host-content-only"

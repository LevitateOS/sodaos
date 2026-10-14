# The release builder supplies the complete, verified appliance image. Soda
# binaries and bundled application images are reused, never rebuilt here.
ARG SODA_BASE_IMAGE=soda-appliance:build
ARG BASE_IMAGE=registry.fedoraproject.org/fedora:44
FROM ${SODA_BASE_IMAGE} AS appliance

FROM ${BASE_IMAGE}
SHELL ["/bin/bash", "-o", "pipefail", "-c"]
COPY --from=appliance /etc/yum.repos.d/tailscale.repo /etc/yum.repos.d/tailscale.repo
RUN dnf -y install systemd podman fuse-overlayfs iproute procps-ng \
    openssh-clients ca-certificates curl git gh nodejs tar gzip vim-minimal \
    cockpit-system cockpit-ws cockpit-bridge cockpit-storaged \
    cockpit-networkmanager cockpit-podman cockpit-files cockpit-selinux \
    cockpit-sosreport tailscale && dnf clean all
# The stock Cockpit storage dependency set includes jq; admit the actual tool.
RUN rpm -q jq

COPY --from=appliance /usr/libexec/soda/ /usr/libexec/soda/
COPY --from=appliance /usr/bin/soda-* /usr/bin/
COPY --from=appliance /usr/share/soda/ /usr/share/soda/
COPY --from=appliance /usr/share/licenses/soda/ /usr/share/licenses/soda/
COPY --from=appliance /usr/share/containers/systemd/ /usr/share/containers/systemd/
COPY --from=appliance /usr/lib/systemd/system/soda-*.service /usr/lib/systemd/system/
COPY --from=appliance /usr/lib/systemd/system/soda-*.socket /usr/lib/systemd/system/
COPY --from=appliance /usr/lib/systemd/system/soda-*.timer /usr/lib/systemd/system/
COPY --from=appliance /usr/lib/systemd/system/soda-host.service.d/ /usr/lib/systemd/system/soda-host.service.d/
COPY --from=appliance /usr/lib/sysusers.d/soda.conf /usr/lib/sysusers.d/soda.conf
COPY --from=appliance /usr/lib/tmpfiles.d/soda.conf /usr/lib/tmpfiles.d/soda.conf
COPY --from=appliance /usr/lib/tmpfiles.d/soda-host-packages.conf /usr/lib/tmpfiles.d/soda-host-packages.conf
COPY --from=appliance /etc/soda/ /etc/soda/
COPY --from=appliance /etc/cockpit/ /etc/cockpit/
COPY soda-docker-init.service /usr/lib/systemd/system/soda-docker-init.service
COPY image-import.conf /usr/lib/systemd/system/soda-image-import.service.d/docker.conf
COPY project-scope.conf /usr/lib/systemd/system/libpod-.scope.d/soda-docker.conf
COPY network-scope.conf /usr/lib/systemd/system/run-.scope.d/soda-docker.conf

RUN test "$(uname -m)" = x86_64 && \
    test -f /usr/share/soda/release.json && \
    test -f /usr/share/soda/images/index.json && \
    systemd-sysusers && systemd-tmpfiles --create && \
    test ! -s /etc/subuid && test ! -s /etc/subgid && \
    printf 'containers:1000000:268435456\n' > /etc/subuid && \
    cp /etc/subuid /etc/subgid && \
    touch /etc/soda/installed && \
    systemctl mask auditd.service audit-rules.service NetworkManager.service \
      NetworkManager-wait-online.service systemd-resolved.service systemd-sysctl.service && \
    systemctl enable soda-host.socket soda-postgres-backup.timer cockpit.socket tailscaled.service && \
    systemctl set-default multi-user.target && \
    rm -f /etc/machine-id && touch /etc/machine-id && \
    mkdir -p /usr/share/soda/docker-runtime && \
    rpm -qa --qf '%{NAME} %{EPOCHNUM}:%{VERSION}-%{RELEASE}.%{ARCH}\n' | \
      LC_ALL=C sort > /usr/share/soda/docker-runtime/packages.txt

ENV container=docker SODA_PROJECT_SUBNET=10.89.0.0/24
STOPSIGNAL SIGRTMIN+3
CMD ["/usr/lib/systemd/systemd"]
LABEL org.opencontainers.image.title="SodaOS Docker runtime" \
    org.opencontainers.image.source="https://github.com/LevitateOS/sodaos" \
    io.soda.runtime="docker"

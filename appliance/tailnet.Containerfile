# Upstream's container base plus its checksum-locked release binaries.
# No containerboot wrapper: the trusted helper invokes tailscaled directly.
ARG BASE_IMAGE
FROM ${BASE_IMAGE}
COPY appliance/licenses/tailscale-LICENSE /usr/share/licenses/tailscale/LICENSE
ARG TAILSCALE_VERSION
ARG TARGETARCH
ARG ARCHIVE_SHA256
ADD --checksum=sha256:${ARCHIVE_SHA256} https://pkgs.tailscale.com/stable/tailscale_${TAILSCALE_VERSION}_${TARGETARCH}.tgz /tmp/tailscale.tgz
RUN tar -xzf /tmp/tailscale.tgz -C /usr/local/bin --strip-components=1 \
        "tailscale_${TAILSCALE_VERSION}_${TARGETARCH}/tailscale" \
        "tailscale_${TAILSCALE_VERSION}_${TARGETARCH}/tailscaled" && \
    rm /tmp/tailscale.tgz
ENTRYPOINT ["/usr/local/bin/tailscaled"]

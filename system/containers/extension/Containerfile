# Built separately from the host image. BASE_IMAGE is the patched Forgejo
# image, so its stopped-host extension CLI can validate and install this bundle.
ARG BASE_IMAGE
FROM ${BASE_IMAGE}
COPY --chmod=0444 extension/extension.json /usr/share/soda/extension/extension.json
COPY --chmod=0755 extension/backend /usr/share/soda/extension/backend
COPY --chmod=0755 extension/run /usr/share/soda/extension/run
COPY extension/assets/ /usr/share/soda/extension/assets/
RUN find /usr/share/soda/extension/assets -type d -exec chmod 0555 {} + && \
    find /usr/share/soda/extension/assets -type f -exec chmod 0444 {} +

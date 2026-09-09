ARG BASE_IMAGE=quay.io/rockylinux/rockylinux:10.2
FROM ${BASE_IMAGE}
ARG ARTIFACT_DIR
RUN dnf -y install ca-certificates && dnf clean all
COPY --chmod=0755 ${ARTIFACT_DIR}/bin/soda-dashboard /usr/local/bin/soda-dashboard
USER 2000:2000
ENTRYPOINT ["/usr/local/bin/soda-dashboard"]

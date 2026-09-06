FROM quay.io/rockylinux/rockylinux:9.6
ARG ARTIFACT_DIR
RUN dnf -y install ca-certificates && dnf clean all
COPY ${ARTIFACT_DIR}/bin/soda-dashboard /usr/local/bin/soda-dashboard
USER 2000:2000
ENTRYPOINT ["/usr/local/bin/soda-dashboard"]

# Per-app bytes are produced by can-soda build, not during application startup.
ARG RUNTIME_IMAGE
FROM ${RUNTIME_IMAGE}
ARG APP_BUNDLE
ARG APP_NAME
ARG ARTIFACT_DIGEST
ARG BUNDLE_DIGEST
ARG STORAGE_DIGEST
ARG TOOLS_DIGEST
ARG RUNTIME_DIGEST
ARG APP_ORIGIN
LABEL io.canlang.app="${APP_NAME}" \
      io.canlang.artifact="${ARTIFACT_DIGEST}" \
      io.canlang.bundle="${BUNDLE_DIGEST}" \
      io.canlang.storage="${STORAGE_DIGEST}" \
      io.canlang.tools="${TOOLS_DIGEST}" \
      io.soda.canlang.runtime="${RUNTIME_DIGEST}" \
      io.canlang.origin="${APP_ORIGIN}"
COPY --chmod=0444 ${APP_BUNDLE} /opt/application/bundle.json

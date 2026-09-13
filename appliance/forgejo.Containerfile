# Exact upstream Forgejo image/wrapper, with release-owned custom resources.
ARG BASE_IMAGE
FROM ${BASE_IMAGE}
COPY forgejo/ /usr/share/soda/forgejo/
COPY presentation.json /usr/share/soda/presentation.json
# The upstream s6 setup and environment-to-ini keep writing their normal app.ini
# through this link. No /data tree is copied into the image. The git service user
# can read, but cannot rewrite, release-owned templates/static resources.
RUN find /usr/share/soda/forgejo -type d -exec chmod 0555 {} + && \
    find /usr/share/soda/forgejo -type f -exec chmod 0444 {} + && \
    ln -s /data/gitea/conf /usr/share/soda/forgejo/conf
ENV GITEA_CUSTOM=/usr/share/soda/forgejo \
    FORGEJO_CUSTOM=/usr/share/soda/forgejo

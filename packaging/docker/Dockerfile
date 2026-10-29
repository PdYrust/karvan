# Karvan's image: the release binary on distroless static, nothing else
# (D26). The build context holds linux_amd64/karvan and linux_arm64/karvan,
# the same static binaries as the release archives. It runs as root and
# reaches Docker through the mounted socket (D29, docs/docker.md).
FROM gcr.io/distroless/static-debian12@sha256:d75cdd72874d4790092fcb1b058493ecf6bb5bf2b2b897045b00ff01d91843f2
ARG TARGETARCH
ARG VERSION=dev
LABEL org.opencontainers.image.title="Karvan" \
	org.opencontainers.image.source="https://github.com/PdYrust/karvan" \
	org.opencontainers.image.version="${VERSION}"
COPY --chmod=0755 linux_${TARGETARCH}/karvan /usr/local/bin/karvan
USER 0:0
HEALTHCHECK --interval=30s --timeout=5s --start-period=60s --start-interval=2s --retries=3 CMD ["karvan", "health"]
# karvan serve by default; `docker compose run --rm karvan init` runs init.
ENTRYPOINT ["karvan"]
CMD ["serve"]

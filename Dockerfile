# syntax=docker/dockerfile:1.7

FROM --platform=$BUILDPLATFORM golang:1.26-bookworm AS unifiedproxy-build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG UNIFIEDPROXY_VERSION=dev

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build \
      -buildvcs=false \
      -trimpath \
      -ldflags="-s -w -X main.version=${UNIFIEDPROXY_VERSION}" \
      -o /out/unifiedproxy \
      .


FROM debian:bookworm-slim AS mihomo-download

ARG MIHOMO_VERSION=1.19.33
ARG MIHOMO_SHA256=386508bfb1c72c43b6a0eda73708ecdce82546581b0ebcb16366c53757c622b6

RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates curl gzip \
    && rm -rf /var/lib/apt/lists/*

RUN set -eux; \
    archive="mihomo-linux-amd64-compatible-v${MIHOMO_VERSION}.gz"; \
    curl --fail --location --retry 3 \
      "https://github.com/MetaCubeX/mihomo/releases/download/v${MIHOMO_VERSION}/${archive}" \
      --output "/tmp/${archive}"; \
    echo "${MIHOMO_SHA256}  /tmp/${archive}" | sha256sum --check --strict; \
    gzip --decompress "/tmp/${archive}"; \
    mv "/tmp/${archive%.gz}" /tmp/mihomo; \
    chmod 0755 /tmp/mihomo


FROM debian:bookworm-slim

ARG UNIFIEDPROXY_VERSION=dev

LABEL org.opencontainers.image.title="UnifiedProxy" \
      org.opencontainers.image.description="A server management panel powered by Mihomo" \
      org.opencontainers.image.source="https://github.com/HlONGlin/UnifiedProxy" \
      org.opencontainers.image.licenses="GPL-3.0-only" \
      org.opencontainers.image.version="${UNIFIEDPROXY_VERSION}"

RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates tzdata \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /opt/unifiedproxy

RUN mkdir -p /opt/unifiedproxy/bin /opt/unifiedproxy/bootstrap /opt/unifiedproxy/core /opt/unifiedproxy/data

COPY --from=unifiedproxy-build /out/unifiedproxy /opt/unifiedproxy/bin/unifiedproxy
COPY --from=mihomo-download /tmp/mihomo /opt/unifiedproxy/bootstrap/mihomo
COPY docker/entrypoint.sh /usr/local/bin/unifiedproxy-entrypoint

RUN chmod 0755 \
      /opt/unifiedproxy/bin/unifiedproxy \
      /opt/unifiedproxy/bootstrap/mihomo \
      /usr/local/bin/unifiedproxy-entrypoint

ENV TZ=UTC \
    UNIFIEDPROXY_DATA_DIR=/opt/unifiedproxy/data \
    UNIFIEDPROXY_CORE_PATH=/opt/unifiedproxy/core/mihomo

VOLUME ["/opt/unifiedproxy/data", "/opt/unifiedproxy/core"]

EXPOSE 2053/tcp 12080/tcp 12080/udp 2096/tcp

STOPSIGNAL SIGTERM

ENTRYPOINT ["/usr/local/bin/unifiedproxy-entrypoint"]

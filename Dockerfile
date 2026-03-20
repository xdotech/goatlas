# syntax=docker/dockerfile:1

# ── Stage 1: Build ──
FROM golang:1.25-bookworm AS builder

WORKDIR /src

# Cache dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source
COPY . .

ARG VERSION=dev

# Build with CGO enabled (required by go-tree-sitter)
RUN CGO_ENABLED=1 GOOS=linux \
    go build -ldflags="-s -w -X github.com/xdotech/goatlas/cmd.Version=${VERSION}" \
    -o /out/goatlas .

# ── Stage 2: Runtime ──
# Use debian-slim (glibc) — required by CGO/tree-sitter; alpine (musl) is incompatible
FROM debian:bookworm-slim

# ca-certificates for TLS, git for repo detection in hooks
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates git gosu \
    && rm -rf /var/lib/apt/lists/*

# Non-root user
RUN useradd -m -u 1000 -d /app goatlas
WORKDIR /app

COPY --from=builder /out/goatlas /app/goatlas
COPY --chown=goatlas:goatlas goatlas.yaml /app/goatlas.yaml
COPY docker-entrypoint.sh /app/docker-entrypoint.sh
RUN chmod +x /app/docker-entrypoint.sh

# Data directory for any runtime artifacts
RUN mkdir -p /app/data && chown -R goatlas:goatlas /app/data

ENV GOATLAS_CONFIG=/app/goatlas.yaml

ENTRYPOINT ["/app/docker-entrypoint.sh"]
CMD ["serve"]

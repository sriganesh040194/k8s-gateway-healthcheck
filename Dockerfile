# ── Build stage ──────────────────────────────────────────────────────────────
FROM mirror.gcr.io/library/golang:1.23-alpine AS builder

WORKDIR /build

# Leverage layer cache: download deps before copying source
COPY go.mod ./
RUN go mod download

# Copy source and build a static binary
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s" \
    -o k8s-gateway-healthcheck \
    ./cmd/server

# ── Final stage ──────────────────────────────────────────────────────────────
# Distroless: no shell, no package manager — minimal attack surface
FROM gcr.io/distroless/static:nonroot

WORKDIR /app
COPY --from=builder /build/k8s-gateway-healthcheck .

# Run as non-root (uid 65532 in distroless/nonroot)
USER nonroot:nonroot

EXPOSE 8080

ENTRYPOINT ["/app/k8s-gateway-healthcheck"]

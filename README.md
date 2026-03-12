<div align="center">

# k8s-gateway-healthcheck

**Enterprise-grade Go healthcheck service for Kubernetes & Azure App Gateway**

[![Build & Release](https://github.com/sriganeshk/k8s-gateway-healthcheck/actions/workflows/ci.yml/badge.svg)](https://github.com/sriganeshk/k8s-gateway-healthcheck/actions/workflows/ci.yml)
[![Go 1.23](https://img.shields.io/badge/Go-1.23-00ADD8?logo=go)](https://golang.org)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![ghcr.io](https://img.shields.io/badge/ghcr.io-k8s--gateway--healthcheck-blue?logo=github)](https://ghcr.io/sriganeshk/k8s-gateway-healthcheck)

*Vibe coded by [Sriganesh Karuppannan](https://github.com/sriganeshk)*

</div>

---

## Overview

A lightweight, zero-dependency Go service that exposes Kubernetes-native health probe endpoints and a full diagnostic endpoint for Azure App Gateway / AGIC. Designed for AKS workloads requiring liveness, readiness, and startup probes with minimal resource footprint.

**Highlights**

- Distroless container image — no shell, no package manager
- Runs as non-root (`uid 65532`) with read-only filesystem
- ~12 MB heap at idle, 25m CPU request
- Structured JSON logging
- CORS + security headers built-in
- Helm chart with dev/prod value sets included

---

## Endpoints

| Path | Probe | Used by |
|------|-------|---------|
| `GET /healthz` | Liveness | Kubernetes liveness probe |
| `GET /readyz` | Readiness | Kubernetes readiness probe |
| `GET /startupz` | Startup | Kubernetes startup probe |
| `GET /health` | Full status | App Gateway, Grafana, dashboards |
| `GET /` | Info | Quick version check |

---

## Response — `/health`

```json
{
  "status": "healthy",
  "service": "k8s-gateway-healthcheck",
  "version": "1.0.0",
  "environment": "production",
  "timestamp": "2026-03-10T10:00:00Z",
  "uptime": "3h45m12s",
  "checks": {
    "liveness":   { "status": "healthy", "message": "Process alive" },
    "memory":     { "status": "healthy", "message": "12MB heap allocated" },
    "goroutines": { "status": "healthy", "message": "8 goroutines running" },
    "startup":    { "status": "healthy", "message": "Initialization complete" }
  }
}
```

> `status` is `healthy` (200) or `unhealthy` (503). `degraded` returns 200 so App Gateway keeps routing while an alert fires.

---

## Getting Started

### Run locally

```bash
docker pull ghcr.io/sriganeshk/k8s-gateway-healthcheck:latest

docker run -p 8080:8080 ghcr.io/sriganeshk/k8s-gateway-healthcheck:latest

curl http://localhost:8080/health
```

### Build from source

```bash
git clone https://github.com/sriganeshk/k8s-gateway-healthcheck.git
cd k8s-gateway-healthcheck

docker build -t k8s-gateway-healthcheck:latest .
docker run -p 8080:8080 k8s-gateway-healthcheck:latest
```

---

## Deploy to AKS

### Helm (recommended)

```bash
# Dev — single replica, no TLS, relaxed resources
helm upgrade --install k8s-gateway-healthcheck ./helm \
  -f helm/values.yaml -f helm/values-dev.yaml \
  --namespace dev --create-namespace

# Prod — HA, TLS, cert-manager, explicit image SHA
helm upgrade --install k8s-gateway-healthcheck ./helm \
  -f helm/values.yaml -f helm/values-prod.yaml \
  --set image.tag=$IMAGE_SHA \
  --namespace production

# Validate before applying
helm lint ./helm -f helm/values.yaml -f helm/values-prod.yaml
helm template k8s-gateway-healthcheck ./helm -f helm/values.yaml \
  | kubectl apply --dry-run=server -f -
```

**TLS:** set `ingress.tls.enabled=true` and `ingress.tls.certManagerIssuer` in your values file.
See [helm/values-prod.yaml](helm/values-prod.yaml) for a full example.

### Plain manifests

```bash
kubectl apply -f k8s/deployment.yaml
kubectl apply -f k8s/service.yaml
kubectl apply -f k8s/ingress.yaml

kubectl rollout status deployment/k8s-gateway-healthcheck -n <namespace>
```

---

## CI/CD

GitHub Actions builds, tests, and pushes to [ghcr.io](https://ghcr.io) on every push to `main`.
A GitHub Release with changelog is created automatically on `v*` tags.

| Event | Tags published |
|-------|---------------|
| Push to `main` | `edge`, `sha-<commit>` |
| Tag `v1.2.3` | `1.2.3`, `1.2`, `1`, `latest`, `sha-<commit>` |
| Pull request | Build only (no push) |

```bash
# Trigger a release
git tag v1.0.0 && git push origin v1.0.0
```

---

## Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8080` | HTTP listen port |
| `APP_NAME` | `gateway-healthcheck` | Service name in responses |
| `APP_VERSION` | *(image tag)* | Version in responses |
| `ENVIRONMENT` | `production` | Env label (auto-set from namespace in K8s) |

---

## Azure App Gateway — Backend Health Probe

In **Azure Portal → App Gateway → Health Probes**, add:

| Field | Value |
|-------|-------|
| Protocol | HTTP |
| Path | `/healthz` |
| Interval | 30s |
| Timeout | 10s |
| Match codes | `200` |

---

## Extending

Add custom checks by extending [internal/health/checker.go](internal/health/checker.go):

```go
func (c *Checker) checkDatabase(ctx context.Context) Check {
    if err := db.PingContext(ctx); err != nil {
        return Check{Status: StatusUnhealthy, Message: err.Error()}
    }
    return Check{Status: StatusHealthy, Message: "Connected"}
}
```

---

## License

[MIT](LICENSE) © [Sriganesh Karuppannan](https://github.com/sriganeshk)

# Camper Vane — multi-stage production image
# Targets:
#   api   — Go API binary (default)
#   proxy — Caddy + baked frontend/dist (same-origin /api → api:8080)

# --- Frontend build ---
FROM node:22-alpine AS frontend
WORKDIR /src
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci
COPY frontend/ ./
RUN npm run build

# --- Go API build ---
FROM golang:1.25-alpine AS gobuild
WORKDIR /src
RUN apk add --no-cache git ca-certificates
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/server ./cmd/server

# --- API runtime ---
FROM alpine:3.21 AS api
RUN apk add --no-cache ca-certificates tzdata \
	&& mkdir -p /data
WORKDIR /app
COPY --from=gobuild /out/server /app/server
# Runs as root so the Compose named volume at /data is writable for SQLite.
# Harden (non-root + chown entrypoint) when you terminate TLS and mount secrets read-only.
ENV DATABASE_PATH=/data/camper_vane.db
EXPOSE 8080
ENTRYPOINT ["/app/server"]

# --- Reverse proxy + static UI ---
FROM caddy:2-alpine AS proxy
COPY deploy/Caddyfile.docker /etc/caddy/Caddyfile
COPY --from=frontend /src/dist /srv
EXPOSE 80 443

# Reduit — multi-stage Docker build.
#
# Build stage compiles the reduit binary statically; runtime stage
# ships only the binary, embedded migrations, and CA certificates.
# Image is roughly 25–35 MiB depending on Go version.

ARG GO_VERSION=1.26

# ---- build -----------------------------------------------------------------
FROM golang:${GO_VERSION}-alpine AS build

WORKDIR /src

# Install build tools for static cgo-free build.
RUN apk add --no-cache git ca-certificates

# Cache go mod downloads.
COPY go.mod go.sum ./
RUN go mod download

# Copy source and build.
COPY . .

ARG VERSION=docker
RUN CGO_ENABLED=0 \
    go build \
        -trimpath \
        -ldflags "-s -w -X github.com/joestump/reduit/internal/cli.Version=${VERSION}" \
        -o /out/reduit \
        ./cmd/reduit

# ---- runtime ---------------------------------------------------------------
FROM alpine:3.24

# Non-root user. UID/GID 10000 to avoid colliding with host UIDs that
# self-hosters often use.
RUN addgroup -S -g 10000 reduit \
 && adduser  -S -u 10000 -G reduit reduit \
 && apk upgrade --no-cache \
 && apk add --no-cache ca-certificates tzdata \
 && mkdir -p /var/lib/reduit /etc/reduit \
 && chown -R reduit:reduit /var/lib/reduit \
 && chmod 0700 /var/lib/reduit

COPY --from=build /out/reduit /usr/local/bin/reduit

# Standard volume mounts:
#   /var/lib/reduit  — persistent data (sqlite cache; point the config's
#                      data_dir here — the default is ~/.local/share/reduit)
#   /etc/reduit      — config (reduit.yaml)
VOLUME ["/var/lib/reduit", "/etc/reduit"]

# No REDUIT_CONFIG here: the default resolution order already includes
# /etc/reduit/reduit.yaml, and setting the env var explicitly turns an
# absent config file into a hard startup error instead of falling back
# to defaults. Mount a config at /etc/reduit/reduit.yaml to override.

# NOTE: Reduit keeps Proton credentials in the OS keychain (ADR-0013), so
# in-container use requires a Secret Service reachable over D-Bus — on a
# headless host that means a libsecret collection unlocked out of band at
# boot, per ADR-0013's headless guidance. There is no file-based secrets
# fallback by design.

USER reduit
ENTRYPOINT ["/usr/local/bin/reduit"]
# Default to the stdio MCP server: the container is meant to be exec'd by an
# MCP client (`docker run -i … reduit mcp`). `tui` and `sync` are the other
# useful entrypoints (`docker run -it … reduit tui`,
# `… reduit sync --watch 15m`).
CMD ["mcp"]

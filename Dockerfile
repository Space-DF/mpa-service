# syntax=docker/dockerfile:1.7
# Use the official golang image to create a binary.
# This is based on Debian and sets the GOPATH to /go.
# https://hub.docker.com/_/golang
FROM --platform=$BUILDPLATFORM golang:1.24-bookworm AS builder

# Create and change to the app directory.
WORKDIR /app

# Cache deps
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

# Copy source
COPY . .

# Build args from buildx
ARG TARGETOS
ARG TARGETARCH

# Build static binary for correct platform
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 \
    GOOS=$TARGETOS \
    GOARCH=$TARGETARCH \
    go build \
      -trimpath \
      -ldflags="-s -w" \
      -o mpa-service \
      ./cmd/mpa

RUN chmod 755 /app/mpa-service

# Use scratch image for smallest possible container
FROM scratch

WORKDIR /app

# CA certs
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# App
COPY --from=builder /app/mpa-service /app/mpa-service
COPY --from=builder /app/configs /app/configs

USER 1000
EXPOSE 80

LABEL maintainer="Space-DF" \
  description="Multi-Protocol Agent (MPA) service" \
  version="1.0"

CMD ["/app/mpa-service", "serve"]

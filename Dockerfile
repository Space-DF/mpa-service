# Use the official golang image to create a binary.
# This is based on Debian and sets the GOPATH to /go.
# https://hub.docker.com/_/golang
FROM --platform=$BUILDPLATFORM golang:1.24-bookworm AS builder

# Create and change to the app directory.
WORKDIR /app

# Retrieve application dependencies.
# This allows the container build to reuse cached dependencies.
# Expecting to copy go.mod and if present go.sum.
COPY go.mod go.sum ./
RUN go mod download

# Copy source
COPY . .

# Build args from buildx
ARG TARGETOS
ARG TARGETARCH

# Build static binary for correct platform
RUN CGO_ENABLED=0 \
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

# Copy CA certificates for HTTPS requests
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Create and change to the app directory.
WORKDIR /app

# Copy the binary to the production image from the builder stage.
COPY --from=builder /app/mpa-service /app/mpa-service
COPY --from=builder /app/configs /app/configs

# Create non-root user (ID 1000)
USER 1000

# Expose the default port
EXPOSE 80

# Add metadata labels
LABEL maintainer="Space-DF" \
  description="Multi-Protocol Agent (MPA) service" \
  version="1.0"

# Run the web service on container startup.
CMD ["/app/mpa-service", "serve"]

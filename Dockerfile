# Use the official golang image to create a binary.
# This is based on Debian and sets the GOPATH to /go.
# https://hub.docker.com/_/golang
FROM golang:1.24-bookworm as builder

# Create and change to the app directory.
WORKDIR /app

# Retrieve application dependencies.
# This allows the container build to reuse cached dependencies.
# Expecting to copy go.mod and if present go.sum.
COPY go.mod go.sum ./
RUN go mod download

# Copy local code to the container image.
COPY . ./

# Build the binary with optimizations.
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags '-extldflags "-static"' -o mpa-service ./cmd/mpa
# Ensure binary is world-executable BEFORE copying to scratch
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

# Add health check
# HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
#   CMD ["/app/mpa-service", "health"] || exit 1

# Add metadata labels
LABEL maintainer="Space-DF" \
  description="Multi-Protocol Agent (MPA) service" \
  version="1.0"

# Run the web service on container startup.
CMD ["/app/mpa-service", "serve"]

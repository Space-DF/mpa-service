# Use the official golang image to create a binary.
# This is based on Debian and sets the GOPATH to /go.
# https://hub.docker.com/_/golang
FROM golang:1.23-bookworm as builder

# Create and change to the app directory.
WORKDIR /app

# Retrieve application dependencies.
# This allows the container build to reuse cached dependencies.
# Expecting to copy go.mod and if present go.sum.
COPY go.* ./
RUN go mod download

# Copy local code to the container image.
COPY . ./

# Build the binary.
RUN go build -v -o mpa-service ./cmd/mpa

# Use the official Debian slim image for a lean production container.
# https://hub.docker.com/_/debian
# https://docs.docker.com/develop/develop-images/multistage-build/#use-multi-stage-builds
FROM debian:bookworm-slim
RUN set -x && apt-get update && DEBIAN_FRONTEND=noninteractive apt-get install -y \
    ca-certificates && \
    rm -rf /var/lib/apt/lists/*

# Create and change to the app directory.
WORKDIR /app

# Copy the binary to the production image from the builder stage.
COPY --from=builder /app/mpa-service /app/mpa-service
COPY --from=builder /app/configs /app/configs

# Expose the default port
EXPOSE 8080

# Run the web service on container startup.
CMD ["/app/mpa-service", "serve"]
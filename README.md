# MPA - Multi-Protocol Agent

A Golang service that receives device payloads from multiple IoT protocols via HTTP and forwards them to an MQTT broker. Designed as a Multi-Protocol Agent (MPA) that can handle various IoT data sources with a unified interface.

## Features

- 🔄 Multi-protocol support (ChirpStack, and extensible for other protocols)
- 📡 Publishes device data to MQTT broker in unified JSON format
- ⚡ Graceful shutdown with signal handling
- 🔧 Configurable via YAML config file and environment variables
- 🏥 Health check endpoints for each protocol
- 📝 Structured logging
- 🚀 Extensible architecture for adding new protocols

## Architecture

```
┌─────────────────┐    ┌──────────────┐    ┌─────────────┐    ┌──────────────┐
│   ChirpStack    │───▶│     MPA      │───▶│    MQTT     │───▶│ MQTT Broker  │
│   (LoRaWAN)     │    │   Service    │    │  Publisher  │    │              │
└─────────────────┘    └──────────────┘    └─────────────┘    └──────────────┘
                                │
                                ▼
                       ┌──────────────┐
                       │  Additional  │
                       │  Protocols   │
                       │  (Future)    │
                       └──────────────┘
```

## Quick Start

### 1. Configuration

Copy the example configuration file:
```bash
cp .env.example .env
```

Edit the configuration files:
- `configs/config.yaml` - Main configuration
- `.env` - Environment variables (optional)

### 2. Build and Run

```bash
# Install dependencies
make deps

# Build the binary
make build

# Run the service
make run

# Or run in development mode
make dev
```

### 3. Docker (Optional)

```dockerfile
FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main ./cmd/mpa

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/main .
COPY --from=builder /app/configs ./configs
CMD ["./main"]
```

## Configuration

### HTTP Server Settings

```yaml
http:
  port: 8080                    # Server port
  read_timeout: 30              # Request read timeout (seconds)
  write_timeout: 30             # Response write timeout (seconds)
  idle_timeout: 60              # Idle connection timeout (seconds)
```

### MQTT Settings

```yaml
mqtt:
  broker: "localhost"           # MQTT broker host
  port: 1883                    # MQTT broker port
  client_id: "mpa-service"      # MQTT client ID
  username: ""                  # MQTT username (optional)
  password: ""                  # MQTT password (optional)
  topic: "mpa/devices/data"     # MQTT topic to publish to
  qos: 0                        # Quality of Service (0, 1, or 2)
  retained: false               # Retain messages flag
```

### Environment Variables

All configuration can also be set via environment variables:

```bash
export HTTP_PORT=8080
export MQTT_BROKER=localhost
export MQTT_USERNAME=myuser
export MQTT_PASSWORD=mypass
```

### Protocol Configuration

```yaml
protocols:
  chirpstack:
    enabled: true
    path: "/chirpstack"          # Endpoint path for ChirpStack
```

## Usage

### ChirpStack Integration

1. In your ChirpStack Application Server:
2. Go to Applications → [Your App] → Integrations
3. Add HTTP Integration
4. Set URL to: `http://your-service:8080/chirpstack`
5. Set Method to: `POST`
6. Add headers if needed (optional)

### Data Format

The service expects JSON payloads from ChirpStack in the following format:

```json
{
  "applicationID": 1,
  "applicationName": "MyApp",
  "deviceName": "Device-001",
  "devEUI": "aa555a0026012345",
  "data": "SGVsbG8gV29ybGQ=",
  "fCnt": 123,
  "fPort": 1,
  "rxInfo": [
    {
      "gatewayID": "aa555a0000000000",
      "rssi": -45,
      "snr": 7.2,
      "location": {
        "latitude": 52.1234,
        "longitude": 4.5678,
        "altitude": 10
      }
    }
  ],
  "publishedAt": "2023-01-01T12:00:00Z"
}
```

### Output Format

The service publishes the following format to MQTT:

```json
{
  "device_id": "aa555a0026012345",
  "device_name": "Device-001",
  "timestamp": "2023-01-01T12:00:00Z",
  "raw_data": "SGVsbG8gV29ybGQ=",
  "decoded_data": {
    "temperature": 23.5,
    "humidity": 65
  },
  "rssi": -45,
  "snr": 7.2,
  "location": {
    "latitude": 52.1234,
    "longitude": 4.5678,
    "altitude": 10
  },
  "port": 1,
  "frame_counter": 123
}
```

## API Endpoints

### ChirpStack Protocol
- `POST /chirpstack` - Main webhook endpoint for ChirpStack
- `GET /health/chirpstack` - ChirpStack handler health check

### Global Endpoints
- `GET /health` - Global health check (shows all active protocols)

## Extending with New Protocols

The MPA architecture makes it easy to add new protocol handlers:

1. Create a new handler package in `internal/handlers/[protocol-name]/`
2. Implement the `ProtocolHandler` interface:
   - `Name() string` - Protocol name
   - `Path() string` - HTTP endpoint path
   - `Method() string` - HTTP method
   - `Handle(c echo.Context) error` - Request handler
   - `HealthCheck(c echo.Context) error` - Health check
3. Register the handler in the configuration
4. Update the config file to enable the new protocol

## Build Targets

```bash
make build          # Build for current platform
make build-linux    # Build for Linux AMD64
make build-windows  # Build for Windows AMD64
make build-arm      # Build for ARM
make test           # Run tests
make clean          # Clean build artifacts
```

## Testing

### Unit Tests
```bash
make test
```

### Manual Testing

1. Start the service:
```bash
make dev
```

2. Send a test payload:
```bash
curl -X POST http://localhost:8080/chirpstack \
  -H "Content-Type: application/json" \
  -d @test_payload.json
```

3. Check health:
```bash
curl http://localhost:8080/health
```

## Logging

The service provides structured logging with different levels:
- INFO: General operational messages
- WARN: Warning messages
- ERROR: Error messages

## Error Handling

- Invalid JSON payloads return HTTP 400
- MQTT connection issues are logged and retried
- Graceful shutdown on SIGINT/SIGTERM
- Health check fails if MQTT is disconnected

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Submit a pull request

## License

MIT License - see LICENSE file for details.
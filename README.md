# MPA - Multi-Protocol Agent

A comprehensive IoT protocol gateway that automatically detects device types and normalizes data from multiple transport protocols into a unified MQTT stream. The MPA service acts as a smart intermediary that can handle various IoT devices and communication protocols while providing a standardized output format.

## ✨ Features

- 🌐 **Multi-Transport Support** - HTTP, MQTT Subscriber, WebSocket, SocketIO
- 🔍 **Automatic Device Detection** - Path, header, payload, and rule-based detection
- 🏭 **Device-Specific Parsing** - RAK2270, RAK7200, Dragino LHT65, and extensible
- 📡 **Unified MQTT Output** - All devices publish to standardized MQTT format
- 🔄 **Backward Compatible** - Existing ChirpStack integrations work unchanged
- ⚡ **Real-time Communication** - WebSocket and SocketIO for live data
- 🔧 **Runtime Management** - Dynamic device profile and parser registration
- 🏥 **Comprehensive Health Monitoring** - Per-transport and global health checks
- 📝 **Structured Logging** - Detailed operational insights
- 🚀 **Horizontally Scalable** - Multiple concurrent connections and protocols

## 🏗️ Architecture

```
IoT Devices (Various Makes/Models)
     ↓
┌───────────────────────────────────────────────────────────┐
│                    Transport Protocols                    │
├─────────────┬─────────────┬─────────────┬─────────────────┤
│    HTTP     │    MQTT     │  WebSocket  │    SocketIO     │
│  POST /http │ Subscriber  │   /ws       │  /socket.io/    │
└─────────────┴─────────────┴─────────────┴─────────────────┘
     ↓
┌───────────────────────────────────────────────────────────┐
│                   Device Detection                        │
├─────────────┬─────────────┬─────────────┬─────────────────┤
│    Path     │   Header    │   Payload   │     Rules       │
│   Based     │    Based    │   Analysis  │  Conditional    │
└─────────────┴─────────────┴─────────────┴─────────────────┘
     ↓
┌───────────────────────────────────────────────────────────┐
│                Device-Specific Parsers                    │
├─────────────┬─────────────┬─────────────┬─────────────────┤
│   RAK2270   │   RAK7200   │ Dragino LHT │ Generic LoRaWAN │
│   Tracker   │   EnvSensor │    65       │   (fallback)    │
└─────────────┴─────────────┴─────────────┴─────────────────┘
     ↓
┌───────────────────────────────────────────────────────────┐
│                 Unified MQTT Output                       │
│            (Standardized Message Format)                  │
└───────────────────────────────────────────────────────────┘
     ↓
┌───────────────────────────────────────────────────────────┐
│                   MQTT Broker                             │
│              (Your Main Data Stream)                      │
└───────────────────────────────────────────────────────────┘
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
Create a `.env` file with the following example settings:

```bash
# Server Configuration
SERVER_LOG_LEVEL=info
SERVER_API_PORT=8080
SERVER_READ_TIMEOUT=30
SERVER_WRITE_TIMEOUT=30

# MQTT Output Configuration (where to publish normalized data)
MQTT_BROKER=localhost
MQTT_PORT=1883
MQTT_CLIENT_ID=mpa-service
MQTT_USERNAME=admin
MQTT_PASSWORD=public
MQTT_TOPIC=mpa/devices/data
MQTT_QOS=0
MQTT_RETAINED=false

# HTTP Protocol Configuration
PROTOCOLS_HTTP_ENABLED=true
PROTOCOLS_HTTP_PATH=/http

# SMS Protocol Configuration
PROTOCOLS_SMS_ENABLED=false
PROTOCOLS_SMS_PROVIDER=twilio
PROTOCOLS_SMS_API_KEY=
PROTOCOLS_SMS_API_SECRET=
PROTOCOLS_SMS_WEBHOOK_URL=
PROTOCOLS_SMS_PORT=8081

# WebSocket Protocol Configuration
PROTOCOLS_WEBSOCKET_ENABLED=false
PROTOCOLS_WEBSOCKET_PATH=/ws
PROTOCOLS_WEBSOCKET_PORT=8082

# MQTT Protocol Configuration
PROTOCOLS_MQTT_PROTOCOL_ENABLED=false
PROTOCOLS_MQTT_PROTOCOL_PORT=1884
```

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

### Transport Protocol Configuration

```yaml
protocols:
  # Generic HTTP transport (recommended)
  http:
    enabled: true
    path: "/http"                # Universal endpoint for all HTTP devices
  
  # WebSocket transport
  websocket:
    enabled: true
    path: "/ws"
    port: 8082
  
  # MQTT subscriber transport  
  mqtt_protocol:
    enabled: true
    port: 1884                   # MQTT broker port for incoming devices
  
  # Backward compatibility
  chirpstack:
    enabled: false               # Legacy endpoint (optional)
    path: "/chirpstack"
```

### Supported Device Types

The service automatically detects these device types:

- **RAK2270** - RAK Sticker Tracker (GPS/LoRaWAN)
- **RAK7200** - RAK WisNode Track Lite (Environmental)
- **Dragino LHT65** - Temperature & Humidity Sensor
- **Generic LoRaWAN** - Any unidentified LoRaWAN device
- **Generic HTTP** - Custom HTTP IoT devices
- **Extensible** - Easy to add new device profiles

## Usage

### ChirpStack Integration

1. In your ChirpStack Application Server:
2. Go to Applications → [Your App] → Integrations
3. Add HTTP Integration
4. Set URL to: `http://your-mpa-service:8080/http`
5. Set Method to: `POST`
6. Enable events: up, join, ack, txack, status, location, log, integration

### Data Format

The service accepts standard ChirpStack webhook payloads and automatically detects device types:

#### RAK2270 Tracker Example:
```json
{
  "deduplicationId": "12345-abc-def",
  "time": "2024-01-15T14:30:00.123456Z",
  "event": "up",
  "deviceInfo": {
    "tenantId": "52f14cd4-c6f1-4fbd-8f87-4025e1d49242",
    "tenantName": "SpaceDF",
    "applicationId": "ca739e26-7b67-4f14-b95e-c4ca0e4c8c9e",
    "applicationName": "Asset Trackers",
    "deviceName": "RAK2270-Tracker-001",
    "devEui": "1122334455667788",
    "devAddr": "aabbccdd"
  },
  "uplinkEvent": {
    "fCnt": 1543,
    "fPort": 2,
    "data": "AWcB6AIBAWE=",
    "object": {
      "latitude": 52.520008,
      "longitude": 13.404954,
      "altitude": 100.5,
      "battery": 85,
      "temperature": 23.4,
      "motion": true
    },
    "rxInfo": [
      {
        "gatewayId": "gateway-001-berlin",
        "rssi": -85,
        "snr": 12.3,
        "location": {
          "latitude": 52.51945,
          "longitude": 13.40443,
          "altitude": 65.0
        }
      }
    ]
  }
}
```

#### RAK7200 Environmental Sensor Example:
```json
{
  "time": "2024-01-15T14:25:12.654321Z",
  "event": "up",
  "deviceInfo": {
    "tenantName": "SpaceDF",
    "applicationName": "Environmental Monitoring",
    "deviceName": "RAK7200-Office-A12",
    "devEui": "9988776655443322"
  },
  "uplinkEvent": {
    "fCnt": 89,
    "fPort": 1,
    "object": {
      "temperature": 25.3,
      "humidity": 65.2,
      "pressure": 1013.25,
      "battery": 3.6
    }
  }
}
```

#### Generic LoRaWAN Device Example:
```json
{
  "time": "2024-01-15T14:35:45.789012Z",
  "event": "up", 
  "deviceInfo": {
    "deviceName": "Custom-Sensor-456",
    "devEui": "5566778899aabbcc"
  },
  "uplinkEvent": {
    "object": {
      "temperature": 25.3,
      "humidity": 65.2
    }
  }
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

## 🌐 API Endpoints

### Transport Endpoints
- `POST /http` - Universal HTTP endpoint (auto-detects device types)
- `GET /ws` - WebSocket upgrade endpoint
- `GET /socket.io/` - SocketIO endpoint
- MQTT Subscriber - Subscribes to configured topics automatically

### Health & Monitoring
- `GET /health` - Global service health (all transports + device profiles)
- `GET /health/http` - HTTP transport health
- `GET /health/websocket` - WebSocket transport health  
- `GET /health/mqtt-subscriber` - MQTT subscriber health
- `GET /health/socketio` - SocketIO transport health
- `GET /device-profiles` - List all supported device types and detection rules

### Backward Compatibility
- `POST /chirpstack` - Legacy ChirpStack endpoint (if enabled)

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

### Local Linting & Security Checks
- Tooling (once):
  ```bash
  go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.4.0
  go install github.com/securego/gosec/v2/cmd/gosec@master
  ```
- MPA service:
  ```bash
  cd mpa-service (optional)
  golangci-lint run
  gosec ./...
  ```

### Manual Testing

1. Start the service:
```bash
make dev
```

2. Test RAK2270 device:
```bash
curl -X POST http://localhost:8080/http \
  -H "Content-Type: application/json" \
  -d '{
    "event": "up",
    "deviceInfo": {
      "deviceName": "RAK2270-Tracker-001",
      "devEui": "1122334455667788"
    },
    "uplinkEvent": {
      "object": {
        "latitude": 52.520008,
        "longitude": 13.404954,
        "battery": 85
      }
    }
  }'
```

3. Test generic device:
```bash
curl -X POST http://localhost:8080/http \
  -H "Content-Type: application/json" \
  -d '{
    "id": "sensor123",
    "name": "Temperature Sensor",
    "temperature": 25.5
  }'
```

4. Check health and device profiles:
```bash
curl http://localhost:8080/health
curl http://localhost:8080/device-profiles
```

5. Test WebSocket (requires a WebSocket client):
```javascript
const ws = new WebSocket('ws://localhost:8080/ws');
ws.send(JSON.stringify({
  device_id: "ws_sensor01",
  temperature: 22.1
}));
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
Licensed under the Apache License, Version 2.0  
See the LICENSE file for details.

[![SpaceDF - A project from Digital Fortress](https://df.technology/images/SpaceDF.png)](https://df.technology/)
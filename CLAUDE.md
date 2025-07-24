# MPA - Multi-Protocol Agent - Development Guide

## Project Overview
A Multi-Protocol Agent (MPA) service that receives device payloads from multiple IoT protocols via HTTP and forwards them to an MQTT broker. Designed with extensible architecture for adding new protocol handlers.

## Architecture
- **CLI Framework**: Cobra with `serve` command
- **Web Framework**: Echo (high-performance, middleware support)
- **Protocol Handlers**: Extensible interface for multiple IoT protocols
- **MQTT Integration**: Publishes unified message format across all protocols

## Key Features

### CLI Commands
```bash
# Build and run server
make build
./bin/mpa-service serve

# Development mode
make dev

# CLI options
./bin/mpa-service serve --port 8080 --log-level debug
./bin/mpa-service serve --help
```

### Supported Event Types (ChirpStack Webhooks)
- **up**: Uplink data from devices
- **join**: Device join events
- **ack**: Downlink acknowledgments
- **txack**: Transmission acknowledgments
- **status**: Device status reports
- **location**: Device location events
- **log**: Device log messages
- **integration**: Integration events

### MQTT Message Structure
```json
{
  "device_id": "string",
  "device_name": "string",
  "timestamp": "2024-01-01T00:00:00Z",
  "event_type": "up|join|ack|txack|status|location|log|integration",
  "raw_data": "base64",           // uplink only
  "decoded_data": {},             // uplink decoded payload
  "port": 1,                      // uplink port
  "frame_counter": 123,           // uplink frame counter
  "rssi": -85.5,                  // signal strength (uplink)
  "snr": 12.3,                    // signal-to-noise ratio (uplink)
  "location": {                   // GPS coordinates when available
    "latitude": 0.0,
    "longitude": 0.0,
    "altitude": 0.0
  },
  "join_info": {},                // join event specific
  "ack_info": {},                 // ack event specific
  "tx_ack_info": {},              // txack event specific
  "status_info": {},              // status event specific
  "log_info": {},                 // log event specific
  "integration_info": {}          // integration event specific
}
```

## Project Structure
```
cmd/mpa/
├── main.go           # Entry point
├── cmd/
│   ├── root.go       # CLI root command
│   └── serve.go      # Serve command implementation

internal/
├── models/
│   ├── events.go     # MQTT message structure
│   └── chirpstack.go # ChirpStack event models
├── handlers/
│   ├── handler.go    # ProtocolHandler interface
│   └── chirpstack/   # ChirpStack protocol handler
├── config/           # Configuration management
├── logger/           # Logging utilities
└── mqtt/             # MQTT client interface
```

## Development Commands
```bash
# Build
make build

# Run tests
make test

# Development mode
make dev

# Clean build artifacts
make clean

# Build for different platforms
make build-linux
make build-windows
make build-arm
```

## Configuration
Configuration is loaded from `configs/config.yaml` and can be overridden via CLI flags:
- `--config`: Config file path
- `--port`: HTTP server port
- `--log-level`: Logging level (debug, info, warn, error)

## Environment Setup
```bash
# Install dependencies
go mod tidy

# Run in development
go run ./cmd/mpa serve

# Build binary
make build
```

## Integration Notes
- Webhook endpoints: POST to configured protocol paths (default: `/chirpstack`)
- Each protocol has dedicated health check endpoint: `/health/[protocol-name]`
- Global health check at `/health` shows all active protocols
- Requires `event` field in JSON payload for ChirpStack events
- All protocols publish to MQTT with unified format

## Adding New Protocols
1. Create new handler package in `internal/handlers/[protocol]/`
2. Implement `ProtocolHandler` interface
3. Add configuration to `config.yaml` under `protocols.[protocol]`
4. Register handler in serve command
package chirpstack

import (
	"github.com/Space-DF/mpa-service/internal/handlers"
	"github.com/Space-DF/mpa-service/internal/logger"
	"github.com/Space-DF/mpa-service/internal/services"
)

// Factory creates ChirpStack HTTP handlers
type Factory struct {
	deviceService *services.DeviceService
	config        Config
	logger        *logger.Logger
}

// NewFactory creates a new ChirpStack handler factory
func NewFactory(deviceService *services.DeviceService, config Config, logger *logger.Logger) handlers.ProtocolHandlerFactory {
	return &Factory{
		deviceService: deviceService,
		config:        config,
		logger:        logger,
	}
}

// Name returns the factory name
func (f *Factory) Name() string {
	return "chirpstack"
}

// CreateHandler creates a ChirpStack handler
func (f *Factory) CreateHandler() handlers.ProtocolHandler {
	return NewHandler(f.deviceService, f.config, f.logger)
}
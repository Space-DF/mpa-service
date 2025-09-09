package ttn

import (
	"github.com/Space-DF/mpa-service/internal/handlers"
	"github.com/Space-DF/mpa-service/internal/logger"
	"github.com/Space-DF/mpa-service/internal/services"
)

// Factory creates TTN HTTP handlers
type Factory struct {
	deviceService *services.DeviceService
	config        Config
	logger        *logger.Logger
}

// NewFactory creates a new TTN handler factory
func NewFactory(deviceService *services.DeviceService, config Config, logger *logger.Logger) handlers.ProtocolHandlerFactory {
	return &Factory{
		deviceService: deviceService,
		config:        config,
		logger:        logger,
	}
}

// Name returns the factory name
func (f *Factory) Name() string {
	return "ttn"
}

// CreateHandler creates a TTN handler
func (f *Factory) CreateHandler() handlers.ProtocolHandler {
	return NewHandler(f.deviceService, f.config, f.logger)
}
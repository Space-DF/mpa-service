package helium

import (
	"github.com/Space-DF/mpa-service/internal/handlers"
	"github.com/Space-DF/mpa-service/internal/logger"
	"github.com/Space-DF/mpa-service/internal/services"
)

// Factory creates Helium HTTP handlers
type Factory struct {
	deviceService *services.DeviceService
	config        Config
	logger        *logger.Logger
}

// NewFactory creates a new Helium handler factory
func NewFactory(deviceService *services.DeviceService, config Config, logger *logger.Logger) handlers.ProtocolHandlerFactory {
	return &Factory{
		deviceService: deviceService,
		config:        config,
		logger:        logger,
	}
}

// Name returns the factory name
func (f *Factory) Name() string {
	return "helium"
}

// CreateHandler creates a Helium handler
func (f *Factory) CreateHandler() handlers.ProtocolHandler {
	return NewHandler(f.deviceService, f.config, f.logger)
}
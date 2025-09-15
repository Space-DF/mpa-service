package lorawan

import (
	"fmt"
	
	"github.com/Space-DF/mpa-service/internal/logger"
	"github.com/Space-DF/mpa-service/internal/protocols/handlers"
	"github.com/Space-DF/mpa-service/internal/protocols/lorawan/base"
	"github.com/Space-DF/mpa-service/internal/services"
)

// LoRaWANHandlerFactory creates LoRaWAN protocol handlers
type LoRaWANHandlerFactory struct {
	providerName  string
	deviceService *services.DeviceService
	logger        *logger.Logger
}

// NewLoRaWANHandlerFactory creates a generic factory for LoRaWAN handlers
func NewLoRaWANHandlerFactory(providerName string, deviceService *services.DeviceService, log *logger.Logger) handlers.ProtocolHandlerFactory {
	return &LoRaWANHandlerFactory{
		providerName:  providerName,
		deviceService: deviceService,
		logger:        log,
	}
}

// Convenience functions for backward compatibility
func NewChirpStackFactory(deviceService *services.DeviceService, log *logger.Logger) handlers.ProtocolHandlerFactory {
	return NewLoRaWANHandlerFactory("chirpstack", deviceService, log)
}

func NewTTNFactory(deviceService *services.DeviceService, log *logger.Logger) handlers.ProtocolHandlerFactory {
	return NewLoRaWANHandlerFactory("ttn", deviceService, log)
}

func NewHeliumFactory(deviceService *services.DeviceService, log *logger.Logger) handlers.ProtocolHandlerFactory {
	return NewLoRaWANHandlerFactory("helium", deviceService, log)
}

// Name returns the LoRaWAN provider name
func (f LoRaWANHandlerFactory) Name() string {
	return f.providerName
}

// CreateHandler creates a LoRaWAN protocol handler
func (f *LoRaWANHandlerFactory) CreateHandler() handlers.ProtocolHandler {
	// Check if dependencies are properly initialized
	if f.deviceService == nil || f.logger == nil {
		panic(fmt.Sprintf("LoRaWANHandlerFactory for %s not properly initialized - missing dependencies. Use SetupFactory() or direct constructors like NewChirpStackFactory()", f.providerName))
	}
	
	baseConfig := base.Config{
		Provider: f.providerName,
	}
	return base.NewLoRaWANHandler(f.deviceService, baseConfig, f.logger)
}

// init registers all LoRaWAN protocol factories with the default registry
func init() {
	RegisterLoRaWANFactories()
}

// RegisterLoRaWANFactories registers all LoRaWAN provider factories
func RegisterLoRaWANFactories() {
	// Register ChirpStack - factory needs dependencies set via SetupFactory before use
	handlers.Register("chirpstack", func() handlers.ProtocolHandlerFactory {
		return &LoRaWANHandlerFactory{providerName: "chirpstack"}
	})

	// Register TTN - factory needs dependencies set via SetupFactory before use  
	handlers.Register("ttn", func() handlers.ProtocolHandlerFactory {
		return &LoRaWANHandlerFactory{providerName: "ttn"}
	})

	// Register Helium - factory needs dependencies set via SetupFactory before use
	handlers.Register("helium", func() handlers.ProtocolHandlerFactory {
		return &LoRaWANHandlerFactory{providerName: "helium"}
	})
}

// SetupFactory configures a factory with dependencies
func (f *LoRaWANHandlerFactory) SetupFactory(deviceService *services.DeviceService, log *logger.Logger) {
	f.deviceService = deviceService
	f.logger = log
}

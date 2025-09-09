package lorawan

import (
	"github.com/Space-DF/mpa-service/internal/logger"
	"github.com/Space-DF/mpa-service/internal/protocols/common"
	"github.com/Space-DF/mpa-service/internal/protocols/lorawan/chirpstack"
	"github.com/Space-DF/mpa-service/internal/protocols/lorawan/helium"
	"github.com/Space-DF/mpa-service/internal/protocols/lorawan/ttn"
	"github.com/Space-DF/mpa-service/internal/services"
)

// LoRaWANHandlerFactory creates LoRaWAN protocol handlers
type LoRaWANHandlerFactory struct {
	providerName  string
	deviceService *services.DeviceService
	logger        *logger.Logger
	handlerFunc   func(*services.DeviceService, *logger.Logger) handlers.ProtocolHandler
}

// NewChirpStackFactory creates a factory for ChirpStack handlers
func NewChirpStackFactory(deviceService *services.DeviceService, log *logger.Logger) handlers.ProtocolHandlerFactory {
	return &LoRaWANHandlerFactory{
		providerName:  "chirpstack",
		deviceService: deviceService,
		logger:        log,
		handlerFunc: func(ds *services.DeviceService, logger *logger.Logger) handlers.ProtocolHandler {
			return chirpstack.NewHandler(ds, chirpstack.Config{}, logger)
		},
	}
}

// NewTTNFactory creates a factory for TTN handlers
func NewTTNFactory(deviceService *services.DeviceService, log *logger.Logger) handlers.ProtocolHandlerFactory {
	return &LoRaWANHandlerFactory{
		providerName:  "ttn",
		deviceService: deviceService,
		logger:        log,
		handlerFunc: func(ds *services.DeviceService, logger *logger.Logger) handlers.ProtocolHandler {
			return ttn.NewHandler(ds, ttn.Config{}, logger)
		},
	}
}

// NewHeliumFactory creates a factory for Helium handlers
func NewHeliumFactory(deviceService *services.DeviceService, log *logger.Logger) handlers.ProtocolHandlerFactory {
	return &LoRaWANHandlerFactory{
		providerName:  "helium",
		deviceService: deviceService,
		logger:        log,
		handlerFunc: func(ds *services.DeviceService, logger *logger.Logger) handlers.ProtocolHandler {
			return helium.NewHandler(ds, helium.Config{}, logger)
		},
	}
}

// Name returns the LoRaWAN provider name
func (f *LoRaWANHandlerFactory) Name() string {
	return f.providerName
}

// CreateHandler creates a LoRaWAN protocol handler
func (f *LoRaWANHandlerFactory) CreateHandler() handlers.ProtocolHandler {
	return f.handlerFunc(f.deviceService, f.logger)
}
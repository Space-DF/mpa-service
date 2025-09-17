package helium

import (
	"github.com/Space-DF/mpa-service/internal/protocols/handlers"
	"github.com/Space-DF/mpa-service/internal/protocols/lorawan/base"
	"github.com/Space-DF/mpa-service/internal/logger"
	"github.com/Space-DF/mpa-service/internal/services"
)

// Config holds Helium handler configuration
type Config struct {
	// No configuration needed - path is static
}

// NewHandler creates a new Helium handler using the generic LoRaWAN handler
func NewHandler(deviceService *services.DeviceService, config Config, logger *logger.Logger) handlers.ProtocolHandler {
	baseConfig := base.Config{
		Provider: "helium",
	}
	return base.NewLoRaWANHandler(deviceService, baseConfig, logger)
}

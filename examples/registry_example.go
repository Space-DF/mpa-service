package examples

import (
	"fmt"

	"github.com/Space-DF/mpa-service/internal/logger"
	"github.com/Space-DF/mpa-service/internal/protocols/handlers"
	"github.com/Space-DF/mpa-service/internal/services"
	"github.com/labstack/echo/v4"
)

// GetProtocolHandler demonstrates the registry pattern usage
// This replaces the old factory pattern like getGun() function
func GetProtocolHandler(protocolType string, deviceService *services.DeviceService, log *logger.Logger) (handlers.ProtocolHandler, error) {
	// Create factory using registry
	factory, err := handlers.CreateFactory(protocolType)
	if err != nil {
		return nil, fmt.Errorf("unsupported protocol type: %s", protocolType)
	}

	// Configure factory with dependencies (if needed)
	if setupFactory, ok := factory.(interface {
		SetupFactory(*services.DeviceService, *logger.Logger)
	}); ok {
		setupFactory.SetupFactory(deviceService, log)
	}

	// Create handler
	return factory.CreateHandler(), nil
}

// ExampleUsage demonstrates how to use the registry pattern
func ExampleUsage() {
	var deviceService *services.DeviceService
	var log *logger.Logger

	// This replaces the old if-else chain in getGun()
	chirpstackHandler, err := GetProtocolHandler("chirpstack", deviceService, log)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	ttnHandler, err := GetProtocolHandler("ttn", deviceService, log)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	// List all available protocols
	protocols := handlers.GetRegisteredProtocols()
	fmt.Printf("Available protocols: %v\n", protocols)

	// Use handlers
	_ = chirpstackHandler
	_ = ttnHandler
}

// RegisterCustomProtocol shows how to dynamically register new protocols
func RegisterCustomProtocol() {
	// Register a custom protocol at runtime
	err := handlers.Register("custom", func() handlers.ProtocolHandlerFactory {
		return &CustomProtocolFactory{}
	})
	if err != nil {
		fmt.Printf("Failed to register custom protocol: %v\n", err)
	}
}

// CustomProtocolFactory example implementation
type CustomProtocolFactory struct{}

func (f *CustomProtocolFactory) Name() string {
	return "custom"
}

func (f *CustomProtocolFactory) CreateHandler() handlers.ProtocolHandler {
	return &CustomProtocolHandler{}
}

// CustomProtocolHandler example implementation
type CustomProtocolHandler struct{}

func (h *CustomProtocolHandler) Name() string {
	return "custom"
}

func (h *CustomProtocolHandler) Path() string {
	return "/custom"
}

func (h *CustomProtocolHandler) Method() string {
	return "POST"
}

func (h *CustomProtocolHandler) Handle(c echo.Context) error {
	return nil
}

func (h *CustomProtocolHandler) HealthCheck(c echo.Context) error {
	return c.JSON(200, map[string]interface{}{
		"status":   "healthy",
		"protocol": "custom",
	})
}

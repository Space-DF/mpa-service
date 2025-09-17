package handlers

import (
	"github.com/labstack/echo/v4"
	"github.com/Space-DF/mpa-service/internal/mqtt"
)

// ProtocolHandler defines the interface for all protocol handlers
type ProtocolHandler interface {
	// Name returns the protocol name
	Name() string
	
	// Path returns the HTTP endpoint path for this protocol
	Path() string
	
	// Method returns the HTTP method this handler expects
	Method() string
	
	// Handle processes the incoming request
	Handle(c echo.Context) error
	
	// HealthCheck returns health status for this protocol handler
	HealthCheck(c echo.Context) error
}

// HandlerManager manages all protocol handlers
type HandlerManager struct {
	handlers map[string]ProtocolHandler
	mqttClient mqtt.ClientInterface
}

// NewHandlerManager creates a new handler manager
func NewHandlerManager(mqttClient mqtt.ClientInterface) *HandlerManager {
	return &HandlerManager{
		handlers:   make(map[string]ProtocolHandler),
		mqttClient: mqttClient,
	}
}

// Register adds a new protocol handler
func (hm *HandlerManager) Register(handler ProtocolHandler) {
	hm.handlers[handler.Name()] = handler
}

// GetHandlers returns all registered handlers
func (hm *HandlerManager) GetHandlers() map[string]ProtocolHandler {
	return hm.handlers
}

// GetHandler returns a specific handler by name
func (hm *HandlerManager) GetHandler(name string) (ProtocolHandler, bool) {
	handler, exists := hm.handlers[name]
	return handler, exists
}

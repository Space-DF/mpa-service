package chirpstack

import (
	"io"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/Space-DF/mpa-service/internal/handlers"
	"github.com/Space-DF/mpa-service/internal/logger"
	"github.com/Space-DF/mpa-service/internal/services"
)

// Handler implements HTTP handler for ChirpStack webhooks
type Handler struct {
	deviceService *services.DeviceService
	config        Config
	logger        *logger.Logger
}

// Config holds ChirpStack handler configuration
type Config struct {
	// No configuration needed - path is static
}

// NewHandler creates a new ChirpStack handler
func NewHandler(deviceService *services.DeviceService, config Config, logger *logger.Logger) handlers.ProtocolHandler {
	return &Handler{
		deviceService: deviceService,
		config:        config,
		logger:        logger,
	}
}

// Name returns the handler name
func (h *Handler) Name() string {
	return "chirpstack"
}

// Path returns the HTTP endpoint path
func (h *Handler) Path() string {
	return "/lorawan/chirpstack/http"
}

// Method returns the HTTP method
func (h *Handler) Method() string {
	return "POST"
}

// Handle processes ChirpStack webhook requests
func (h *Handler) Handle(c echo.Context) error {
	startTime := time.Now()
	
	// Read request body
	body, err := io.ReadAll(c.Request().Body)
	if err != nil {
		h.logger.Errorf("ChirpStack: Error reading request body: %v", err)
		return echo.NewHTTPError(400, "Failed to read request body")
	}
	
	h.logger.Infof("ChirpStack: Received request from %s (body: %d bytes)", 
		c.Request().RemoteAddr, len(body))
	
	// Add ChirpStack source to metadata
	transportMetadata := map[string]interface{}{
		"transport":      "http",
		"lorawan_source": "chirpstack",
		"path":           c.Request().URL.Path,
		"received_at":    startTime.UTC().Format(time.RFC3339),
	}
	
	// Process through device service
	if err := h.deviceService.ProcessHTTPMessage(c.Request(), body, transportMetadata); err != nil {
		h.logger.Errorf("ChirpStack: Error processing message: %v", err)
		return echo.NewHTTPError(500, "Failed to process message")
	}
	
	h.logger.Infof("ChirpStack: Successfully processed message in %v", time.Since(startTime))
	
	return c.JSON(200, map[string]interface{}{
		"status":  "success",
		"source":  "chirpstack",
		"message": "Message processed successfully",
	})
}

// HealthCheck returns health status
func (h *Handler) HealthCheck(c echo.Context) error {
	healthStatus := h.deviceService.GetHealthStatus()
	
	if !healthStatus["mqtt_connected"].(bool) {
		return echo.NewHTTPError(503, "MQTT client not connected")
	}
	
	return c.JSON(200, map[string]interface{}{
		"source":    "chirpstack",
		"status":    "healthy",
		"endpoint":  "/lorawan/chirpstack/http",
		"mqtt":      "connected",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}
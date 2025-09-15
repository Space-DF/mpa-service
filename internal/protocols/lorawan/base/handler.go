package base

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/Space-DF/mpa-service/internal/protocols/handlers"
	"github.com/Space-DF/mpa-service/internal/logger"
	"github.com/Space-DF/mpa-service/internal/services"
)

// LoRaWANHandler implements a generic HTTP handler for LoRaWAN webhook providers
type LoRaWANHandler struct {
	deviceService *services.DeviceService
	config        Config
	logger        *logger.Logger
	provider      string
}

// Config holds LoRaWAN handler configuration
type Config struct {
	Provider        string `yaml:"provider"`        // Provider name (chirpstack, ttn, helium, etc.)
	MaxRequestSize  int64  `yaml:"max_request_size"` // Maximum request body size in bytes (default: 1MB)
	RequestTimeout  int    `yaml:"request_timeout"`  // Request timeout in seconds (default: 30)
}

// NewLoRaWANHandler creates a new generic LoRaWAN handler
func NewLoRaWANHandler(deviceService *services.DeviceService, config Config, logger *logger.Logger) handlers.ProtocolHandler {
	return &LoRaWANHandler{
		deviceService: deviceService,
		config:        config,
		logger:        logger,
		provider:      config.Provider,
	}
}

// Name returns the handler name
func (h LoRaWANHandler) Name() string {
	return h.provider
}

// Path returns the HTTP endpoint path
func (h LoRaWANHandler) Path() string {
	return fmt.Sprintf("/lorawan/%s/http", h.provider)
}

// Method returns the HTTP method
func (h LoRaWANHandler) Method() string {
	return "POST"
}

// Handle processes LoRaWAN webhook requests
func (h *LoRaWANHandler) Handle(c echo.Context) error {
	startTime := time.Now()
	
	// Validate Content-Type
	contentType := c.Request().Header.Get("Content-Type")
	if contentType != "" && !strings.HasPrefix(contentType, "application/json") {
		h.logger.Warnf("%s: Invalid content type: %s", h.provider, contentType)
		return echo.NewHTTPError(400, "Content-Type must be application/json")
	}
	
	// Set maximum request body size (default 1MB)
	maxSize := h.config.MaxRequestSize
	if maxSize == 0 {
		maxSize = 1024 * 1024 // 1MB default
	}
	
	// Limit request body size to prevent memory exhaustion attacks
	c.Request().Body = http.MaxBytesReader(c.Response().Writer, c.Request().Body, maxSize)
	
	// Read request body with size limit
	body, err := io.ReadAll(c.Request().Body)
	if err != nil {
		h.logger.Errorf("%s: Error reading request body: %v", h.provider, err)
		// Check if it's a size limit error
		if strings.Contains(err.Error(), "http: request body too large") {
			return echo.NewHTTPError(413, "Request body too large")
		}
		return echo.NewHTTPError(400, "Failed to read request body")
	}
	
	h.logger.Infof("%s: Received request from %s (body: %d bytes)", 
		h.provider, c.Request().RemoteAddr, len(body))
	
	// Add provider-specific source to metadata
	transportMetadata := map[string]interface{}{
		"transport":      "http",
		"lorawan_source": h.provider,
		"path":           c.Request().URL.Path,
		"received_at":    startTime.UTC().Format(time.RFC3339),
	}
	
	// Process through device service
	if err := h.deviceService.ProcessHTTPMessage(c.Request(), body, transportMetadata); err != nil {
		h.logger.Errorf("%s: Error processing message: %v", h.provider, err)
		return echo.NewHTTPError(500, "Failed to process message")
	}
	
	h.logger.Infof("%s: Successfully processed message in %v", h.provider, time.Since(startTime))
	
	return c.JSON(200, map[string]interface{}{
		"status":  "success",
		"source":  h.provider,
		"message": "Message processed successfully",
	})
}

// HealthCheck returns health status
func (h *LoRaWANHandler) HealthCheck(c echo.Context) error {
	healthStatus := h.deviceService.GetHealthStatus()
	
	if !healthStatus["mqtt_connected"].(bool) {
		return echo.NewHTTPError(503, "MQTT client not connected")
	}
	
	return c.JSON(200, map[string]interface{}{
		"source":    h.provider,
		"status":    "healthy",
		"endpoint":  h.Path(),
		"mqtt":      "connected",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

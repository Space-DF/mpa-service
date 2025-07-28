package http

import (
	"fmt"
	"io"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/Space-DF/mpa-service/internal/handlers"
	"github.com/Space-DF/mpa-service/internal/logger"
	"github.com/Space-DF/mpa-service/internal/services"
)

// Handler implements HTTP transport handler for multi-device support
type Handler struct {
	deviceService *services.DeviceService
	config        Config
	logger        *logger.Logger
}

// Config holds HTTP transport handler configuration
type Config struct {
	Path string `yaml:"path"`
}

// NewHandler creates a new HTTP transport handler
func NewHandler(deviceService *services.DeviceService, config Config, logger *logger.Logger) handlers.ProtocolHandler {
	return &Handler{
		deviceService: deviceService,
		config:        config,
		logger:        logger,
	}
}

// Name returns the transport protocol name
func (h *Handler) Name() string {
	return "http"
}

// Path returns the HTTP endpoint path
func (h *Handler) Path() string {
	return h.config.Path
}

// Method returns the HTTP method this handler expects
func (h *Handler) Method() string {
	return "POST"
}

// Handle processes incoming HTTP requests from various device types
func (h *Handler) Handle(c echo.Context) error {
	startTime := time.Now()
	
	// Read request body
	body, err := io.ReadAll(c.Request().Body)
	if err != nil {
		h.logger.Errorf("HTTP Transport: Error reading request body: %v", err)
		return echo.NewHTTPError(400, "Failed to read request body")
	}
	
	// Log incoming request
	h.logger.Infof("HTTP Transport: Received %s request to %s from %s (body size: %d bytes)", 
		c.Request().Method, c.Request().URL.Path, c.Request().RemoteAddr, len(body))
	
	// Debug log: Print original message content before any processing
	h.logger.Debugf("HTTP Transport: Original message payload:\n%s", string(body))
	
	// Prepare transport metadata
	transportMetadata := map[string]interface{}{
		"transport":    "http",
		"method":       c.Request().Method,
		"path":         c.Request().URL.Path,
		"remote_addr":  c.Request().RemoteAddr,
		"user_agent":   c.Request().Header.Get("User-Agent"),
		"content_type": c.Request().Header.Get("Content-Type"),
		"received_at":  startTime.UTC().Format(time.RFC3339),
	}
	
	// Add query parameters if present
	if len(c.Request().URL.RawQuery) > 0 {
		transportMetadata["query_params"] = c.Request().URL.RawQuery
	}
	
	// Add custom headers (X- headers often contain device information)
	for name, values := range c.Request().Header {
		if len(name) > 2 && (name[:2] == "X-" || name[:2] == "x-") {
			if len(values) > 0 {
				transportMetadata["header_"+name] = values[0]
			}
		}
	}
	
	// Process message through device service
	if err := h.deviceService.ProcessHTTPMessage(c.Request(), body, transportMetadata); err != nil {
		h.logger.Errorf("HTTP Transport: Error processing message: %v", err)
		
		// Return appropriate error response based on error type
		if fmt.Sprintf("%v", err)[:19] == "device detection failed" {
			return echo.NewHTTPError(400, fmt.Sprintf("Device detection failed: %v", err))
		} else if fmt.Sprintf("%v", err)[:21] == "message parsing failed" {
			return echo.NewHTTPError(400, fmt.Sprintf("Message parsing failed: %v", err))
		} else {
			return echo.NewHTTPError(500, "Internal processing error")
		}
	}
	
	processingTime := time.Since(startTime)
	h.logger.Infof("HTTP Transport: Successfully processed message in %v", processingTime)
	
	// Return success response
	return c.JSON(200, map[string]interface{}{
		"status":         "success",
		"message":        "Message processed successfully",
		"transport":      "http",
		"processing_time_ms": processingTime.Milliseconds(),
		"timestamp":      time.Now().UTC().Format(time.RFC3339),
	})
}

// HealthCheck returns health status for HTTP transport handler
func (h *Handler) HealthCheck(c echo.Context) error {
	healthStatus := h.deviceService.GetHealthStatus()
	
	if !healthStatus["mqtt_connected"].(bool) {
		return echo.NewHTTPError(503, "MQTT client not connected")
	}
	
	return c.JSON(200, map[string]interface{}{
		"transport": "http",
		"status":    "healthy", 
		"message":   "HTTP transport handler is running",
		"mqtt":      "connected",
		"parsers":   healthStatus["parsers"],
		"endpoint":  h.config.Path,
		"method":    "POST",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}


package http

import (
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/Space-DF/mpa-service/internal/logger"
	"github.com/Space-DF/mpa-service/internal/protocols/handlers"
	"github.com/Space-DF/mpa-service/internal/services"
	"github.com/labstack/echo/v4"
)

// Handler implements HTTP transport handler for multi-device support
type Handler struct {
	deviceService *services.DeviceService
	config        Config
	logger        *logger.Logger
}

// Config holds HTTP transport handler configuration
type Config struct {
	Path           string `yaml:"path"`
	MaxRequestSize int64  `yaml:"max_request_size"` // Maximum request body size in bytes (default: 1MB)
	RequestTimeout int    `yaml:"request_timeout"`  // Request timeout in seconds (default: 30)
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
func (h Handler) Name() string {
	return "http"
}

// Path returns the HTTP endpoint path
func (h Handler) Path() string {
	return h.config.Path
}

// Method returns the HTTP method this handler expects
func (h Handler) Method() string {
	return "POST"
}

// Handle processes incoming HTTP requests from various device types
func (h *Handler) Handle(c echo.Context) error {
	startTime := time.Now()

	// HTTP Handler - Strict Tenant Validation
	h.logger.Infof("HTTP Handler: Processing request with strict tenant validation")

	// Validate Content-Type for JSON payloads
	contentType := c.Request().Header.Get("Content-Type")
	if contentType != "" && !strings.HasPrefix(contentType, "application/json") && !strings.HasPrefix(contentType, "text/plain") {
		h.logger.Warnf("HTTP Transport: Invalid content type: %s", contentType)
		return echo.NewHTTPError(400, "Content-Type must be application/json or text/plain")
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
		h.logger.Errorf("HTTP Transport: Error reading request body: %v", err)
		// Check if it's a size limit error
		if strings.Contains(err.Error(), "http: request body too large") {
			return echo.NewHTTPError(413, "Request body too large")
		}
		return echo.NewHTTPError(400, "Failed to read request body")
	}

	// Log incoming request
	h.logger.Infof("HTTP Transport: Received %s request to %s from %s (body size: %d bytes)",
		c.Request().Method, c.Request().URL.Path, c.Request().RemoteAddr, len(body))

	// Log original message content for debugging
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

	// Extract organization slug from subdomain
	tenantID := h.extractTenantFromSubdomain(c.Request().Host)
	if tenantID == "" {
		h.logger.Errorf("HTTP Transport: Organization slug not found in subdomain")
		return echo.NewHTTPError(400, "Invalid subdomain: must include organization slug (e.g., {org-slug}.localhost)")
	}

	// Sanitize tenant ID for security
	tenantID = h.sanitizeTenantID(tenantID)
	if tenantID == "" {
		h.logger.Errorf("HTTP Transport: Invalid organization slug name")

		return echo.NewHTTPError(400, "Invalid organization slug name")
	}

	transportMetadata["tenant_id"] = tenantID
	h.logger.Infof("HTTP Transport: Tenant extraction successful from subdomain, tenant_id = %s", tenantID)

	// Call service with tenant information
	if err := h.deviceService.ProcessHTTPMessage(c.Request(), body, transportMetadata); err != nil {
		h.logger.Errorf("HTTP Transport: Error processing message: %v", err)
		return echo.NewHTTPError(500, "Internal processing error")
	}

	h.logger.Infof("HTTP Transport: ✅ Message processed successfully with tenant-based routing")

	processingTime := time.Since(startTime)
	h.logger.Infof("HTTP Transport: Successfully processed message in %v", processingTime)

	// Return success response
	return c.JSON(200, map[string]interface{}{
		"status":             "success",
		"message":            "Message processed successfully",
		"transport":          "http",
		"processing_time_ms": processingTime.Milliseconds(),
		"timestamp":          time.Now().UTC().Format(time.RFC3339),
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

// sanitizeTenantID sanitizes organization slug name for security
func (h *Handler) sanitizeTenantID(tenantID string) string {
	if tenantID == "" {
		return ""
	}

	// Remove MQTT wildcards and dangerous characters
	tenantID = strings.ReplaceAll(tenantID, "+", "")
	tenantID = strings.ReplaceAll(tenantID, "#", "")
	tenantID = strings.ReplaceAll(tenantID, "../", "")
	tenantID = strings.ReplaceAll(tenantID, "..", "")
	tenantID = strings.ReplaceAll(tenantID, "/", "")
	tenantID = strings.ReplaceAll(tenantID, "\\", "")
	tenantID = strings.ReplaceAll(tenantID, " ", "")
	tenantID = strings.ReplaceAll(tenantID, "\t", "")
	tenantID = strings.ReplaceAll(tenantID, "\n", "")
	tenantID = strings.ReplaceAll(tenantID, "\r", "")
	tenantID = strings.TrimSpace(tenantID)

	// Validate format (alphanumeric, hyphens, underscores only)
	if len(tenantID) > 0 && len(tenantID) <= 64 {
		valid := true
		for _, char := range tenantID {
			if (char < 'a' || char > 'z') && (char < 'A' || char > 'Z') &&
				(char < '0' || char > '9') && char != '-' && char != '_' {
				valid = false
				break
			}
		}
		if valid {
			return tenantID
		}
	}

	return ""
}

// extractTenantFromSubdomain extracts organization slug from subdomain
func (h *Handler) extractTenantFromSubdomain(host string) string {
	if host == "" {
		return ""
	}

	// Remove port if present (e.g., "spacedf.localhost:3000" -> "spacedf.localhost")
	if colonIndex := strings.Index(host, ":"); colonIndex != -1 {
		host = host[:colonIndex]
	}

	// Split by dots to get subdomain parts
	parts := strings.Split(host, ".")
	if len(parts) < 2 {
		return ""
	}

	// The first part should be the organization slug
	// e.g., "spacedf.localhost" -> ["spacedf", "localhost"] -> "spacedf"
	return parts[0]
}
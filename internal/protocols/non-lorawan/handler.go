package nonlorawan

import (
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/Space-DF/mpa-service/internal/logger"
	"github.com/Space-DF/mpa-service/internal/protocols/handlers"
	"github.com/Space-DF/mpa-service/internal/services"
	"github.com/labstack/echo/v4"
)

// Handler implements the HTTP API ingress for non-LoRaWAN devices.
type Handler struct {
	deviceService *services.DeviceService
	logger        *logger.Logger
}

const maxRequestSize = 1024 * 1024

var tenantIDPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,64}$`)

type RequestPayload struct {
	SerialNumber string `json:"serialNumber"`
	Payload      string `json:"payload"`
	ReceivedAt   string `json:"received_at,omitempty"`
}

func NewHandler(deviceService *services.DeviceService, logger *logger.Logger) handlers.ProtocolHandler {
	return &Handler{
		deviceService: deviceService,
		logger:        logger,
	}
}

func (h Handler) Name() string {
	return "api"
}

func (h Handler) Path() string {
	return "/api/http"
}

func (h Handler) Method() string {
	return "POST"
}

func (h *Handler) Handle(c echo.Context) error {
	startTime := time.Now()

	contentType := c.Request().Header.Get("Content-Type")
	if contentType != "" && !strings.HasPrefix(contentType, "application/json") {
		h.logger.Warnf("api: invalid content type: %s", contentType)
		return echo.NewHTTPError(http.StatusBadRequest, "Content-Type must be application/json")
	}

	c.Request().Body = http.MaxBytesReader(c.Response().Writer, c.Request().Body, maxRequestSize)
	body, err := io.ReadAll(c.Request().Body)
	if err != nil {
		h.logger.Errorf("api: error reading request body: %v", err)
		if strings.Contains(err.Error(), "http: request body too large") {
			return echo.NewHTTPError(http.StatusRequestEntityTooLarge, "Request body too large")
		}
		return echo.NewHTTPError(http.StatusBadRequest, "Failed to read request body")
	}

	var requestPayload RequestPayload
	if err := json.Unmarshal(body, &requestPayload); err != nil {
		h.logger.Warnf("api: invalid JSON payload: %v", err)
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid JSON payload")
	}

	serialNumber := strings.TrimSpace(requestPayload.SerialNumber)
	if serialNumber == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "serialNumber is required")
	}

	encodedPayload := strings.TrimSpace(requestPayload.Payload)
	if encodedPayload == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "payload is required")
	}
	if _, err := base64.StdEncoding.DecodeString(encodedPayload); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "payload must be a valid base64 string")
	}

	tenantID := extractTenantID(c.Request().Host)
	if tenantID == "" {
		h.logger.Errorf("api: invalid organization slug")
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid subdomain: must include organization slug (e.g., {org-slug}.localhost)")
	}

	receivedAt := strings.TrimSpace(requestPayload.ReceivedAt)
	if receivedAt == "" {
		receivedAt = startTime.UTC().Format(time.RFC3339)
	}

	transportMetadata := map[string]interface{}{
		"transport":     "http",
		"api_source":    "api",
		"serial_number": serialNumber,
		"tenant_id":     tenantID,
		"received_at":   receivedAt,
	}

	if err := h.deviceService.ProcessHTTPMessage(c.Request(), body, transportMetadata); err != nil {
		h.logger.Errorf("api: error processing message: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to process message")
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"status":             "success",
		"source":             "api",
		"serialNumber":       serialNumber,
		"message":            "Message processed successfully",
		"processing_time_ms": time.Since(startTime).Milliseconds(),
	})
}

func (h *Handler) HealthCheck(c echo.Context) error {
	healthStatus := h.deviceService.GetHealthStatus()

	if !healthStatus["mqtt_connected"].(bool) {
		return echo.NewHTTPError(http.StatusServiceUnavailable, "MQTT client not connected")
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"source":    "api",
		"status":    "healthy",
		"endpoint":  h.Path(),
		"mqtt":      "connected",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

func extractTenantID(host string) string {
	host = strings.TrimSpace(host)
	if host == "" {
		return ""
	}

	if colonIndex := strings.Index(host, ":"); colonIndex != -1 {
		host = host[:colonIndex]
	}

	parts := strings.Split(host, ".")
	if len(parts) < 2 {
		return ""
	}

	tenantID := strings.TrimSpace(parts[0])
	if tenantIDPattern.MatchString(tenantID) {
		return tenantID
	}
	return ""
}

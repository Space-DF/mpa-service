package chirpstack

import (
	"fmt"
	"log"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/Space-DF/mpa-service/internal/handlers"
	"github.com/Space-DF/mpa-service/internal/models"
	"github.com/Space-DF/mpa-service/internal/mqtt"
)

// Handler implements the ChirpStack protocol handler
type Handler struct {
	mqttClient mqtt.ClientInterface
	config     Config
}

// Config holds ChirpStack-specific configuration
type Config struct {
	Path string `yaml:"path"`
}

// NewHandler creates a new ChirpStack handler
func NewHandler(mqttClient mqtt.ClientInterface, config Config) handlers.ProtocolHandler {
	return &Handler{
		mqttClient: mqttClient,
		config:     config,
	}
}

// Name returns the protocol name
func (h *Handler) Name() string {
	return "chirpstack"
}

// Path returns the HTTP endpoint path
func (h *Handler) Path() string {
	return h.config.Path
}

// Method returns the HTTP method this handler expects
func (h *Handler) Method() string {
	return "POST"
}

// Handle processes ChirpStack webhook events
func (h *Handler) Handle(c echo.Context) error {
	var eventWrapper models.EventWrapper
	if err := c.Bind(&eventWrapper); err != nil {
		log.Printf("Error binding ChirpStack event: %v", err)
		return echo.NewHTTPError(400, "Invalid JSON")
	}

	if eventWrapper.Event == "" {
		return echo.NewHTTPError(400, "Event type is required")
	}

	if err := h.processEvent(&eventWrapper); err != nil {
		log.Printf("Error processing ChirpStack event %s: %v", eventWrapper.Event, err)
		return echo.NewHTTPError(500, "Internal server error")
	}

	return c.JSON(200, map[string]string{
		"status":  "success",
		"message": fmt.Sprintf("ChirpStack event %s processed", eventWrapper.Event),
	})
}

// HealthCheck returns health status for ChirpStack handler
func (h *Handler) HealthCheck(c echo.Context) error {
	if h.mqttClient == nil || !h.mqttClient.IsConnected() {
		return echo.NewHTTPError(503, "MQTT client not connected")
	}

	return c.JSON(200, map[string]string{
		"protocol":  "chirpstack",
		"status":    "healthy",
		"message":   "ChirpStack handler is running",
		"mqtt":      "connected",
	})
}

// processEvent handles all ChirpStack event types
func (h *Handler) processEvent(eventWrapper *models.EventWrapper) error {
	mqttMessage := &models.MQTTMessage{
		DeviceID:        "",
		DeviceName:      "",
		Timestamp:       time.Now(),
		EventType:       eventWrapper.Event,
	}

	switch eventWrapper.Event {
	case string(models.EventUplink):
		return h.processUplinkEvent(eventWrapper.UplinkEvent, mqttMessage)
	case string(models.EventJoin):
		return h.processJoinEvent(eventWrapper.JoinEvent, mqttMessage)
	case string(models.EventAck):
		return h.processAckEvent(eventWrapper.AckEvent, mqttMessage)
	case string(models.EventTxAck):
		return h.processTxAckEvent(eventWrapper.TxAckEvent, mqttMessage)
	case string(models.EventStatus):
		return h.processStatusEvent(eventWrapper.StatusEvent, mqttMessage)
	case string(models.EventLocation):
		return h.processLocationEvent(eventWrapper.LocationEvent, mqttMessage)
	case string(models.EventLog):
		return h.processLogEvent(eventWrapper.LogEvent, mqttMessage)
	case string(models.EventIntegration):
		return h.processIntegrationEvent(eventWrapper.IntegrationEvent, mqttMessage)
	default:
		return fmt.Errorf("unknown ChirpStack event type: %s", eventWrapper.Event)
	}
}

// Event processing methods remain the same as before...
func (h *Handler) processUplinkEvent(event *models.UplinkEvent, mqttMsg *models.MQTTMessage) error {
	if event == nil {
		return fmt.Errorf("uplink event is nil")
	}
	
	mqttMsg.DeviceID = event.DeviceInfo.DevEUI
	mqttMsg.DeviceName = event.DeviceInfo.DeviceName
	mqttMsg.Data = event.Data
	mqttMsg.DecodedData = event.Object
	mqttMsg.Port = int(event.FPort)
	mqttMsg.FCnt = int(event.FCnt)
	
	if len(event.RxInfo) > 0 {
		mqttMsg.RSSI = event.RxInfo[0].RSSI
		mqttMsg.SNR = event.RxInfo[0].SNR
		
		if event.RxInfo[0].Location.Latitude != 0 || event.RxInfo[0].Location.Longitude != 0 {
			mqttMsg.Location = &event.RxInfo[0].Location
		}
	}
	
	return h.mqttClient.Publish(mqttMsg)
}

func (h *Handler) processJoinEvent(event *models.JoinEvent, mqttMsg *models.MQTTMessage) error {
	if event == nil {
		return fmt.Errorf("join event is nil")
	}
	
	mqttMsg.DeviceID = event.DeviceInfo.DevEUI
	mqttMsg.DeviceName = event.DeviceInfo.DeviceName
	mqttMsg.JoinInfo = map[string]interface{}{
		"dev_addr": event.DevAddr,
	}
	
	return h.mqttClient.Publish(mqttMsg)
}

func (h *Handler) processAckEvent(event *models.AckEvent, mqttMsg *models.MQTTMessage) error {
	if event == nil {
		return fmt.Errorf("ack event is nil")
	}
	
	mqttMsg.DeviceID = event.DeviceInfo.DevEUI
	mqttMsg.DeviceName = event.DeviceInfo.DeviceName
	mqttMsg.AckInfo = map[string]interface{}{
		"queue_item_id": event.QueueItemID,
		"acknowledged":  event.Acknowledged,
		"f_cnt_down":    event.FCntDown,
	}
	
	return h.mqttClient.Publish(mqttMsg)
}

func (h *Handler) processTxAckEvent(event *models.TxAckEvent, mqttMsg *models.MQTTMessage) error {
	if event == nil {
		return fmt.Errorf("tx ack event is nil")
	}
	
	mqttMsg.DeviceID = event.DeviceInfo.DevEUI
	mqttMsg.DeviceName = event.DeviceInfo.DeviceName
	mqttMsg.TxAckInfo = map[string]interface{}{
		"downlink_id":  event.DownlinkID,
		"queue_item_id": event.QueueItemID,
		"f_cnt_down":    event.FCntDown,
		"gateway_id":    event.GatewayID,
	}
	
	return h.mqttClient.Publish(mqttMsg)
}

func (h *Handler) processStatusEvent(event *models.StatusEvent, mqttMsg *models.MQTTMessage) error {
	if event == nil {
		return fmt.Errorf("status event is nil")
	}
	
	mqttMsg.DeviceID = event.DeviceInfo.DevEUI
	mqttMsg.DeviceName = event.DeviceInfo.DeviceName
	mqttMsg.StatusInfo = map[string]interface{}{
		"battery_level": event.BatteryLevel,
		"margin":        event.Margin,
	}
	
	return h.mqttClient.Publish(mqttMsg)
}

func (h *Handler) processLocationEvent(event *models.LocationEvent, mqttMsg *models.MQTTMessage) error {
	if event == nil {
		return fmt.Errorf("location event is nil")
	}
	
	mqttMsg.DeviceID = event.DeviceInfo.DevEUI
	mqttMsg.DeviceName = event.DeviceInfo.DeviceName
	mqttMsg.Location = &event.Location
	
	return h.mqttClient.Publish(mqttMsg)
}

func (h *Handler) processLogEvent(event *models.LogEvent, mqttMsg *models.MQTTMessage) error {
	if event == nil {
		return fmt.Errorf("log event is nil")
	}
	
	mqttMsg.DeviceID = event.DeviceInfo.DevEUI
	mqttMsg.DeviceName = event.DeviceInfo.DeviceName
	mqttMsg.LogInfo = map[string]interface{}{
		"level":       event.Level,
		"code":        event.Code,
		"description": event.Description,
		"context":     event.Context,
	}
	
	return h.mqttClient.Publish(mqttMsg)
}

func (h *Handler) processIntegrationEvent(event *models.IntegrationEvent, mqttMsg *models.MQTTMessage) error {
	if event == nil {
		return fmt.Errorf("integration event is nil")
	}
	
	mqttMsg.DeviceID = event.DeviceInfo.DevEUI
	mqttMsg.DeviceName = event.DeviceInfo.DeviceName
	mqttMsg.IntegrationInfo = map[string]interface{}{
		"integration_name": event.IntegrationName,
		"event_type":       event.EventType,
		"object":           event.Object,
	}
	
	return h.mqttClient.Publish(mqttMsg)
}
package services

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/Space-DF/mpa-service/internal/models"
	"github.com/Space-DF/mpa-service/internal/mqtt"
)

// DeviceService provides transport-agnostic device detection, parsing, and message publishing
type DeviceService struct {
	deviceRegistry *models.DeviceRegistry
	parserRegistry *models.ParserRegistry
	mqttClient     mqtt.ClientInterface
}

// NewDeviceService creates a new device service
func NewDeviceService(mqttClient mqtt.ClientInterface) *DeviceService {
	service := &DeviceService{
		deviceRegistry: models.NewDeviceRegistry(),
		parserRegistry: models.NewParserRegistry(),
		mqttClient:     mqttClient,
	}
	
	// No device profile initialization needed - service just forwards messages
	log.Printf("DeviceService: Initialized as message forwarding service (no device detection)")
	return service
}

// createDefaultProfiles - REMOVED: Device detection not needed for forwarding service

// initializeDetectors - REMOVED: Device detection not needed for forwarding service

// ProcessHTTPMessage processes an HTTP request and publishes to MQTT
func (ds *DeviceService) ProcessHTTPMessage(request *http.Request, body []byte, transportMetadata map[string]interface{}) error {
	log.Printf("DeviceService: Processing HTTP message (body size: %d bytes)", len(body))
	
	// Forward message directly to MQTT without device detection
	return ds.forwardToMQTT(body, request, transportMetadata)
}

// ProcessMQTTMessage processes an MQTT message and republishes to main broker
func (ds *DeviceService) ProcessMQTTMessage(topic string, payload []byte, transportMetadata map[string]interface{}) error {
	log.Printf("DeviceService: Processing MQTT message from topic: %s (payload size: %d bytes)", topic, len(payload))
	
	// Add topic information to metadata
	transportMetadata["mqtt_topic"] = topic
	
	// Create mock HTTP request for consistency
	mockRequest := &http.Request{
		Header: make(http.Header),
	}
	
	// Forward message directly to MQTT without device detection
	return ds.forwardToMQTT(payload, mockRequest, transportMetadata)
}

// ProcessWebSocketMessage processes a WebSocket message and publishes to MQTT
func (ds *DeviceService) ProcessWebSocketMessage(message []byte, connectionMetadata map[string]interface{}) error {
	log.Printf("DeviceService: Processing WebSocket message (message size: %d bytes)", len(message))
	
	// Create mock HTTP request for consistency
	mockRequest := &http.Request{
		Header: make(http.Header),
	}
	
	connectionMetadata["transport"] = "websocket"
	
	// Forward message directly to MQTT without device detection
	return ds.forwardToMQTT(message, mockRequest, connectionMetadata)
}

// detectDevice - REMOVED: Device detection not needed for forwarding service

// detectDeviceFromMQTT - REMOVED: Device detection not needed for forwarding service

// detectDeviceFromWebSocket - REMOVED: Device detection not needed for forwarding service

// parseAndPublish - REMOVED: Parsing not needed for forwarding service

// forwardToMQTT forwards raw message directly to MQTT without device detection or parsing
func (ds *DeviceService) forwardToMQTT(payload []byte, request *http.Request, metadata map[string]interface{}) error {
	// Create a simple MQTT message with raw payload
	mqttMessage := &models.MQTTMessage{
		DeviceID:    "unknown",
		DeviceName:  "unknown", 
		EventType:   "raw",
		Data:        payload,
		DecodedData: nil,
		Timestamp:   time.Now(),
		Metadata:    metadata,
	}
	
	// Add request information to metadata if available
	if request != nil {
		if mqttMessage.Metadata == nil {
			mqttMessage.Metadata = make(map[string]interface{})
		}
		mqttMessage.Metadata["request_path"] = request.URL.Path
		mqttMessage.Metadata["request_method"] = request.Method
		if userAgent := request.Header.Get("User-Agent"); userAgent != "" {
			mqttMessage.Metadata["user_agent"] = userAgent
		}
	}
	
	// Check if MQTT client is available and connected before publishing
	if ds.mqttClient != nil && ds.mqttClient.IsConnected() {
		if err := ds.mqttClient.Publish(mqttMessage); err != nil {
			return fmt.Errorf("failed to publish to MQTT: %w", err)
		}
		log.Printf("DeviceService: Forwarded raw message to MQTT (size: %d bytes)", len(payload))
	} else {
		log.Printf("DeviceService: MQTT client not available, skipping forward (size: %d bytes)", len(payload))
	}
	
	return nil
}

// AddDeviceProfile - REMOVED: Device profiles not needed for forwarding service
func (ds *DeviceService) AddDeviceProfile(profile *models.DeviceProfile) error {
	return fmt.Errorf("device profiles not supported in forwarding mode")
}

// AddParser - REMOVED: Parsers not needed for forwarding service
func (ds *DeviceService) AddParser(name string, parser models.DevicePayloadParser) {
	log.Printf("DeviceService: Parsers not supported in forwarding mode")
}

// GetDeviceProfiles - REMOVED: Device profiles not needed for forwarding service
func (ds *DeviceService) GetDeviceProfiles() map[string]*models.DeviceProfile {
	return make(map[string]*models.DeviceProfile) // Empty map for forwarding mode
}

// GetHealthStatus returns service health information
func (ds *DeviceService) GetHealthStatus() map[string]interface{} {
	return map[string]interface{}{
		"mode":           "forwarding",  
		"parsers":        0,
		"mqtt_connected": ds.mqttClient != nil && ds.mqttClient.IsConnected(),
	}
}

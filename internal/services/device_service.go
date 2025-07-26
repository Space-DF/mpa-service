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
	
	service.initializeDefaults()
	return service
}

// initializeDefaults sets up default device profiles and parsers
func (ds *DeviceService) initializeDefaults() {
	// Register parsers
	ds.parserRegistry.RegisterParser("generic", models.NewGenericJSONParser())
	ds.parserRegistry.RegisterParser("chirpstack", models.NewChirpStackParser())
	
	// Create and register default device profiles
	ds.createDefaultProfiles()
	
	// Initialize detectors
	ds.initializeDetectors()
}

// createDefaultProfiles creates built-in device profiles
func (ds *DeviceService) createDefaultProfiles() {
	// ChirpStack profile - backward compatibility
	chirpStackProfile := &models.DeviceProfile{
		ID:          "chirpstack",
		Make:        "chirpstack",
		Model:       "lorawan",
		Version:     "v1.0",
		Description: "ChirpStack LoRaWAN Network Server",
		Detection: models.DetectionConfig{
			Method:   models.DetectionMethodPayload,
			Priority: 100,
		},
		Fields: map[string]string{
			"device_id":   "deviceInfo.devEui",
			"device_name": "deviceInfo.deviceName",
			"event_type":  "event",
		},
		Metadata: map[string]interface{}{
			"protocol": "chirpstack",
			"type":     "lorawan",
		},
	}
	ds.deviceRegistry.RegisterProfile(chirpStackProfile)
	
	// Generic HTTP profile
	genericHTTPProfile := &models.DeviceProfile{
		ID:          "generic-http",
		Make:        "generic",
		Model:       "http",
		Version:     "v1.0",
		Description: "Generic HTTP device",
		Detection: models.DetectionConfig{
			Method:   models.DetectionMethodPath,
			Pattern:  "/http",
			Priority: 60,
		},
		Fields: map[string]string{
			"device_id":    "id",
			"device_name":  "name",
			"message_type": "type",
		},
		Metadata: map[string]interface{}{
			"protocol": "http",
			"type":     "generic",
		},
	}
	ds.deviceRegistry.RegisterProfile(genericHTTPProfile)
	
	// Generic MQTT profile
	genericMQTTProfile := &models.DeviceProfile{
		ID:          "generic-mqtt",
		Make:        "generic",
		Model:       "mqtt",
		Version:     "v1.0",
		Description: "Generic MQTT device",
		Detection: models.DetectionConfig{
			Method:   models.DetectionMethodRules,
			Priority: 50,
		},
		Fields: map[string]string{
			"device_id":    "device_id",
			"device_name":  "device_name",
			"message_type": "message_type",
		},
		Metadata: map[string]interface{}{
			"protocol": "mqtt",
			"type":     "generic",
		},
	}
	ds.deviceRegistry.RegisterProfile(genericMQTTProfile)
	
	// Set generic HTTP as default fallback
	ds.deviceRegistry.SetDefaultProfile(genericHTTPProfile)
}

// initializeDetectors sets up and registers all detectors
func (ds *DeviceService) initializeDetectors() {
	pathDetector := models.NewPathDetector()
	headerDetector := models.NewHeaderDetector()
	payloadDetector := models.NewPayloadDetector()
	ruleDetector := models.NewRuleBasedDetector()
	
	// Register profiles with appropriate detectors
	for _, profile := range ds.deviceRegistry.GetAllProfiles() {
		switch profile.Detection.Method {
		case models.DetectionMethodPath:
			pathDetector.RegisterProfile(profile)
		case models.DetectionMethodHeader, models.DetectionMethodHeaders:
			headerDetector.RegisterProfile(profile)
		case models.DetectionMethodPayload:
			payloadDetector.RegisterProfile(profile)
		case models.DetectionMethodRules:
			ruleDetector.RegisterProfile(profile)
		}
	}
	
	// Register detectors with registry (in priority order)
	ds.deviceRegistry.RegisterDetector(pathDetector)
	ds.deviceRegistry.RegisterDetector(headerDetector)
	ds.deviceRegistry.RegisterDetector(ruleDetector)
	ds.deviceRegistry.RegisterDetector(payloadDetector)
}

// ProcessHTTPMessage processes an HTTP request and publishes to MQTT
func (ds *DeviceService) ProcessHTTPMessage(request *http.Request, body []byte, transportMetadata map[string]interface{}) error {
	// Detect device type
	deviceProfile, err := ds.detectDevice(request, body, transportMetadata)
	if err != nil {
		return fmt.Errorf("device detection failed: %w", err)
	}
	
	log.Printf("DeviceService: Detected device profile: %s (%s-%s)", 
		deviceProfile.ID, deviceProfile.Make, deviceProfile.Model)
	
	// Parse and publish
	return ds.parseAndPublish(body, deviceProfile, request, transportMetadata)
}

// ProcessMQTTMessage processes an MQTT message and republishes to main broker
func (ds *DeviceService) ProcessMQTTMessage(topic string, payload []byte, transportMetadata map[string]interface{}) error {
	// For MQTT, we need to create a mock HTTP request for detection
	// or implement MQTT-specific detection
	mockRequest := &http.Request{
		Header: make(http.Header),
	}
	
	// Add topic information for detection
	transportMetadata["mqtt_topic"] = topic
	
	// Detect device type
	deviceProfile, err := ds.detectDeviceFromMQTT(topic, payload, transportMetadata)
	if err != nil {
		return fmt.Errorf("MQTT device detection failed: %w", err)
	}
	
	log.Printf("DeviceService: Detected MQTT device profile: %s (%s-%s)", 
		deviceProfile.ID, deviceProfile.Make, deviceProfile.Model)
	
	// Parse and publish
	return ds.parseAndPublish(payload, deviceProfile, mockRequest, transportMetadata)
}

// ProcessWebSocketMessage processes a WebSocket message and publishes to MQTT
func (ds *DeviceService) ProcessWebSocketMessage(message []byte, connectionMetadata map[string]interface{}) error {
	// Create mock HTTP request for WebSocket
	mockRequest := &http.Request{
		Header: make(http.Header),
	}
	
	connectionMetadata["transport"] = "websocket"
	
	// Detect device type
	deviceProfile, err := ds.detectDeviceFromWebSocket(message, connectionMetadata)
	if err != nil {
		return fmt.Errorf("WebSocket device detection failed: %w", err)
	}
	
	log.Printf("DeviceService: Detected WebSocket device profile: %s (%s-%s)", 
		deviceProfile.ID, deviceProfile.Make, deviceProfile.Model)
	
	// Parse and publish
	return ds.parseAndPublish(message, deviceProfile, mockRequest, connectionMetadata)
}

// detectDevice uses HTTP request for device detection
func (ds *DeviceService) detectDevice(request *http.Request, body []byte, metadata map[string]interface{}) (*models.DeviceProfile, error) {
	return ds.deviceRegistry.DetectDevice(request, body)
}

// detectDeviceFromMQTT implements MQTT-specific device detection
func (ds *DeviceService) detectDeviceFromMQTT(topic string, payload []byte, metadata map[string]interface{}) (*models.DeviceProfile, error) {
	// Try payload-based detection first
	payloadDetector := models.NewPayloadDetector()
	for _, profile := range ds.deviceRegistry.GetAllProfiles() {
		if profile.Detection.Method == models.DetectionMethodPayload {
			payloadDetector.RegisterProfile(profile)
		}
	}
	
	mockRequest := &http.Request{Header: make(http.Header)}
	if profile, err := payloadDetector.DetectDevice(mockRequest, payload); err == nil {
		return profile, nil
	}
	
	// Fall back to generic MQTT profile
	if profile, exists := ds.deviceRegistry.GetProfile("generic-mqtt"); exists {
		return profile, nil
	}
	
	return nil, fmt.Errorf("no device profile found for MQTT topic: %s", topic)
}

// detectDeviceFromWebSocket implements WebSocket-specific device detection
func (ds *DeviceService) detectDeviceFromWebSocket(message []byte, metadata map[string]interface{}) (*models.DeviceProfile, error) {
	// For now, use payload-based detection
	// This can be enhanced with connection-specific metadata
	payloadDetector := models.NewPayloadDetector()
	for _, profile := range ds.deviceRegistry.GetAllProfiles() {
		if profile.Detection.Method == models.DetectionMethodPayload {
			payloadDetector.RegisterProfile(profile)
		}
	}
	
	mockRequest := &http.Request{Header: make(http.Header)}
	if profile, err := payloadDetector.DetectDevice(mockRequest, message); err == nil {
		return profile, nil
	}
	
	// Fall back to generic profile
	if profile, exists := ds.deviceRegistry.GetProfile("generic-http"); exists {
		return profile, nil
	}
	
	return nil, fmt.Errorf("no device profile found for WebSocket message")
}

// parseAndPublish handles parsing and MQTT publishing
func (ds *DeviceService) parseAndPublish(payload []byte, profile *models.DeviceProfile, request *http.Request, metadata map[string]interface{}) error {
	// Get appropriate parser
	parser, err := ds.parserRegistry.GetParserForDevice(profile)
	if err != nil {
		return fmt.Errorf("parser not found for device %s: %w", profile.ID, err)
	}
	
	// Parse the message
	deviceMessage, err := parser.Parse(payload, profile, request)
	if err != nil {
		return fmt.Errorf("message parsing failed with %s parser: %w", parser.GetParserName(), err)
	}
	
	// Add transport metadata
	if deviceMessage.Metadata == nil {
		deviceMessage.Metadata = make(map[string]interface{})
	}
	for k, v := range metadata {
		deviceMessage.Metadata[k] = v
	}
	
	// Convert to MQTT message and publish
	mqttMessage := deviceMessage.ToMQTTMessage()
	mqttMessage.Timestamp = time.Now()
	
	if err := ds.mqttClient.Publish(mqttMessage); err != nil {
		return fmt.Errorf("failed to publish to MQTT: %w", err)
	}
	
	log.Printf("DeviceService: Successfully processed message from device %s (%s) via %s parser", 
		deviceMessage.DeviceID, deviceMessage.Profile.ID, parser.GetParserName())
	
	return nil
}

// AddDeviceProfile allows runtime addition of device profiles
func (ds *DeviceService) AddDeviceProfile(profile *models.DeviceProfile) error {
	if err := ds.deviceRegistry.RegisterProfile(profile); err != nil {
		return err
	}
	
	// Re-initialize detectors to include new profile
	ds.initializeDetectors()
	return nil
}

// AddParser allows runtime addition of custom parsers
func (ds *DeviceService) AddParser(name string, parser models.DevicePayloadParser) {
	ds.parserRegistry.RegisterParser(name, parser)
}

// GetDeviceProfiles returns all registered device profiles
func (ds *DeviceService) GetDeviceProfiles() map[string]*models.DeviceProfile {
	return ds.deviceRegistry.GetAllProfiles()
}

// GetHealthStatus returns service health information
func (ds *DeviceService) GetHealthStatus() map[string]interface{} {
	return map[string]interface{}{
		"device_profiles": len(ds.deviceRegistry.GetAllProfiles()),
		"parsers":        len(ds.parserRegistry.GetAllParsers()),
		"mqtt_connected": ds.mqttClient != nil && ds.mqttClient.IsConnected(),
	}
}
package models

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// DevicePayloadParser defines interface for parsing device-specific payloads
type DevicePayloadParser interface {
	// Parse converts raw payload data to DeviceMessage
	Parse(rawData []byte, profile *DeviceProfile, request *http.Request) (*DeviceMessage, error)
	
	// Validate checks if the payload is valid for this parser
	Validate(rawData []byte, profile *DeviceProfile) error
	
	// GetSupportedDeviceTypes returns device types this parser can handle
	GetSupportedDeviceTypes() []string
	
	// GetParserName returns the name of this parser
	GetParserName() string
}

// ParserRegistry manages device payload parsers
type ParserRegistry struct {
	parsers map[string]DevicePayloadParser
}

// NewParserRegistry creates a new parser registry
func NewParserRegistry() *ParserRegistry {
	return &ParserRegistry{
		parsers: make(map[string]DevicePayloadParser),
	}
}

// RegisterParser adds a parser to the registry
func (pr *ParserRegistry) RegisterParser(name string, parser DevicePayloadParser) {
	pr.parsers[name] = parser
}

// GetParser retrieves a parser by name
func (pr *ParserRegistry) GetParser(name string) (DevicePayloadParser, bool) {
	parser, exists := pr.parsers[name]
	return parser, exists
}

// GetParserForDevice finds the appropriate parser for a device profile
func (pr *ParserRegistry) GetParserForDevice(profile *DeviceProfile) (DevicePayloadParser, error) {
	// First try to get parser by device make/model
	parserName := fmt.Sprintf("%s-%s", profile.Make, profile.Model)
	if parser, exists := pr.parsers[parserName]; exists {
		return parser, nil
	}
	
	// Try by make only
	if parser, exists := pr.parsers[profile.Make]; exists {
		return parser, nil
	}
	
	// Fall back to generic parser
	if parser, exists := pr.parsers["generic"]; exists {
		return parser, nil
	}
	
	return nil, fmt.Errorf("no parser found for device profile: %s", profile.ID)
}

// GetAllParsers returns all registered parsers
func (pr *ParserRegistry) GetAllParsers() map[string]DevicePayloadParser {
	return pr.parsers
}

// BaseParser provides common functionality for all parsers
type BaseParser struct {
	Name               string
	SupportedDevices   []string
}

// GetParserName returns the parser name
func (bp *BaseParser) GetParserName() string {
	return bp.Name
}

// GetSupportedDeviceTypes returns supported device types
func (bp *BaseParser) GetSupportedDeviceTypes() []string {
	return bp.SupportedDevices
}

// Validate performs basic validation
func (bp *BaseParser) Validate(rawData []byte, profile *DeviceProfile) error {
	if len(rawData) == 0 {
		return fmt.Errorf("empty payload")
	}
	return nil
}

// GenericJSONParser handles generic JSON payloads using field mappings
type GenericJSONParser struct {
	BaseParser
}

// NewGenericJSONParser creates a new generic JSON parser
func NewGenericJSONParser() *GenericJSONParser {
	return &GenericJSONParser{
		BaseParser: BaseParser{
			Name:             "generic",
			SupportedDevices: []string{"generic", "http", "custom"},
		},
	}
}

// Parse processes generic JSON payload using device profile field mappings
func (gjp *GenericJSONParser) Parse(rawData []byte, profile *DeviceProfile, request *http.Request) (*DeviceMessage, error) {
	if err := gjp.Validate(rawData, profile); err != nil {
		return nil, err
	}
	
	var payload map[string]interface{}
	if err := json.Unmarshal(rawData, &payload); err != nil {
		return nil, fmt.Errorf("failed to parse JSON payload: %w", err)
	}
	
	// Extract fields using mappings from device profile
	deviceID := gjp.extractField(payload, profile.Fields["device_id"])
	deviceName := gjp.extractField(payload, profile.Fields["device_name"])
	messageType := gjp.extractField(payload, profile.Fields["message_type"])
	
	// If message_type is not specified, use default
	if messageType == "" {
		messageType = "data"
	}
	
	// Build metadata from request
	metadata := make(map[string]interface{})
	metadata["http_method"] = request.Method
	metadata["http_path"] = request.URL.Path
	metadata["user_agent"] = request.Header.Get("User-Agent")
	metadata["remote_addr"] = request.RemoteAddr
	
	// Add device profile metadata
	for k, v := range profile.Metadata {
		metadata[k] = v
	}
	
	return &DeviceMessage{
		Profile:     profile,
		DeviceID:    deviceID,
		DeviceName:  deviceName,
		MessageType: messageType,
		RawData:     rawData,
		ParsedData:  payload,
		Metadata:    metadata,
	}, nil
}

// extractField extracts a field value from JSON payload using dot notation
func (gjp *GenericJSONParser) extractField(payload map[string]interface{}, fieldPath string) string {
	if fieldPath == "" {
		return ""
	}
	
	// Simple implementation - can be enhanced with proper JSON path parsing
	if value, exists := payload[fieldPath]; exists {
		if str, ok := value.(string); ok {
			return str
		}
		return fmt.Sprintf("%v", value)
	}
	
	return ""
}

// ChirpStackParser handles ChirpStack-specific webhook payloads
type ChirpStackParser struct {
	BaseParser
}

// NewChirpStackParser creates a new ChirpStack parser
func NewChirpStackParser() *ChirpStackParser {
	return &ChirpStackParser{
		BaseParser: BaseParser{
			Name:             "chirpstack",
			SupportedDevices: []string{"chirpstack", "lorawan"},
		},
	}
}

// Parse processes ChirpStack webhook payload
func (csp *ChirpStackParser) Parse(rawData []byte, profile *DeviceProfile, request *http.Request) (*DeviceMessage, error) {
	if err := csp.Validate(rawData, profile); err != nil {
		return nil, err
	}
	
	var eventWrapper EventWrapper
	if err := json.Unmarshal(rawData, &eventWrapper); err != nil {
		return nil, fmt.Errorf("failed to parse ChirpStack payload: %w", err)
	}
	
	if eventWrapper.Event == "" {
		return nil, fmt.Errorf("missing event type in ChirpStack payload")
	}
	
	// Extract device information based on event type
	var deviceID, deviceName string
	var decodedData map[string]interface{}
	
	switch eventWrapper.Event {
	case string(EventUplink):
		if eventWrapper.UplinkEvent != nil {
			deviceID = eventWrapper.UplinkEvent.DeviceInfo.DevEUI
			deviceName = eventWrapper.UplinkEvent.DeviceInfo.DeviceName
			decodedData = eventWrapper.UplinkEvent.Object
		}
	case string(EventJoin):
		if eventWrapper.JoinEvent != nil {
			deviceID = eventWrapper.JoinEvent.DeviceInfo.DevEUI
			deviceName = eventWrapper.JoinEvent.DeviceInfo.DeviceName
		}
	case string(EventAck):
		if eventWrapper.AckEvent != nil {
			deviceID = eventWrapper.AckEvent.DeviceInfo.DevEUI
			deviceName = eventWrapper.AckEvent.DeviceInfo.DeviceName
		}
	case string(EventTxAck):
		if eventWrapper.TxAckEvent != nil {
			deviceID = eventWrapper.TxAckEvent.DeviceInfo.DevEUI
			deviceName = eventWrapper.TxAckEvent.DeviceInfo.DeviceName
		}
	case string(EventStatus):
		if eventWrapper.StatusEvent != nil {
			deviceID = eventWrapper.StatusEvent.DeviceInfo.DevEUI
			deviceName = eventWrapper.StatusEvent.DeviceInfo.DeviceName
		}
	case string(EventLocation):
		if eventWrapper.LocationEvent != nil {
			deviceID = eventWrapper.LocationEvent.DeviceInfo.DevEUI
			deviceName = eventWrapper.LocationEvent.DeviceInfo.DeviceName
		}
	case string(EventLog):
		if eventWrapper.LogEvent != nil {
			deviceID = eventWrapper.LogEvent.DeviceInfo.DevEUI
			deviceName = eventWrapper.LogEvent.DeviceInfo.DeviceName
		}
	case string(EventIntegration):
		if eventWrapper.IntegrationEvent != nil {
			deviceID = eventWrapper.IntegrationEvent.DeviceInfo.DevEUI
			deviceName = eventWrapper.IntegrationEvent.DeviceInfo.DeviceName
		}
	}
	
	// Build parsed data from the entire event wrapper
	parsedData := make(map[string]interface{})
	eventData, _ := json.Marshal(eventWrapper)
	json.Unmarshal(eventData, &parsedData)
	
	if decodedData != nil {
		parsedData["decoded_object"] = decodedData
	}
	
	// Build metadata
	metadata := make(map[string]interface{})
	metadata["protocol"] = "chirpstack"
	metadata["event_type"] = eventWrapper.Event
	metadata["http_method"] = request.Method
	metadata["http_path"] = request.URL.Path
	
	return &DeviceMessage{
		Profile:     profile,
		DeviceID:    deviceID,
		DeviceName:  deviceName,
		MessageType: eventWrapper.Event,
		RawData:     rawData,
		ParsedData:  parsedData,
		Metadata:    metadata,
	}, nil
}

// Validate checks ChirpStack payload structure
func (csp *ChirpStackParser) Validate(rawData []byte, profile *DeviceProfile) error {
	if err := csp.BaseParser.Validate(rawData, profile); err != nil {
		return err
	}
	
	var eventWrapper EventWrapper
	if err := json.Unmarshal(rawData, &eventWrapper); err != nil {
		return fmt.Errorf("invalid ChirpStack JSON format: %w", err)
	}
	
	if eventWrapper.Event == "" {
		return fmt.Errorf("missing required 'event' field in ChirpStack payload")
	}
	
	return nil
}
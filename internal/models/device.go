package models

import (
	"fmt"
	"net/http"
)

// DeviceProfile defines how to detect and parse messages from a specific device type
type DeviceProfile struct {
	ID          string                 `yaml:"id" json:"id"`
	Make        string                 `yaml:"make" json:"make"`
	Model       string                 `yaml:"model" json:"model"`
	Version     string                 `yaml:"version" json:"version"`
	Description string                 `yaml:"description" json:"description"`
	Detection   DetectionConfig        `yaml:"detection" json:"detection"`
	Fields      map[string]string      `yaml:"fields" json:"fields"`
	Metadata    map[string]interface{} `yaml:"metadata" json:"metadata"`
}

// DetectionConfig defines how to detect if a request matches this device profile
type DetectionConfig struct {
	Method   DetectionMethod           `yaml:"method" json:"method"`
	Pattern  string                    `yaml:"pattern,omitempty" json:"pattern,omitempty"`
	Header   string                    `yaml:"header,omitempty" json:"header,omitempty"`
	Value    string                    `yaml:"value,omitempty" json:"value,omitempty"`
	Headers  map[string]string         `yaml:"headers,omitempty" json:"headers,omitempty"`
	Rules    []DetectionRule           `yaml:"rules,omitempty" json:"rules,omitempty"`
	Priority int                       `yaml:"priority" json:"priority"` // Higher priority checked first
}

// DetectionMethod defines different ways to detect device types
type DetectionMethod string

const (
	DetectionMethodPath    DetectionMethod = "path"      // Match URL path pattern
	DetectionMethodHeader  DetectionMethod = "header"    // Match single header value
	DetectionMethodHeaders DetectionMethod = "headers"   // Match multiple headers
	DetectionMethodPayload DetectionMethod = "payload"   // Analyze payload structure
	DetectionMethodRules   DetectionMethod = "rules"     // Complex rule-based detection
)

// DetectionRule allows complex detection logic
type DetectionRule struct {
	Type      string `yaml:"type" json:"type"`           // "header", "path", "payload"
	Field     string `yaml:"field" json:"field"`         // Header name or JSON path
	Operator  string `yaml:"operator" json:"operator"`   // "equals", "contains", "exists", "regex"
	Value     string `yaml:"value" json:"value"`         // Expected value
	Required  bool   `yaml:"required" json:"required"`   // Must match for device to be detected
}

// DeviceRegistry manages device profiles and detection
type DeviceRegistry struct {
	profiles    map[string]*DeviceProfile
	detectors   []DeviceDetector
	defaultProfile *DeviceProfile
}

// DeviceDetector defines interface for device detection strategies
type DeviceDetector interface {
	DetectDevice(request *http.Request, body []byte) (*DeviceProfile, error)
	GetSupportedMethods() []DetectionMethod
	GetPriority() int
}

// NewDeviceRegistry creates a new device registry
func NewDeviceRegistry() *DeviceRegistry {
	return &DeviceRegistry{
		profiles:  make(map[string]*DeviceProfile),
		detectors: make([]DeviceDetector, 0),
	}
}

// RegisterProfile adds a device profile to the registry
func (dr *DeviceRegistry) RegisterProfile(profile *DeviceProfile) error {
	if profile.ID == "" {
		return fmt.Errorf("device profile ID cannot be empty")
	}
	dr.profiles[profile.ID] = profile
	return nil
}

// GetProfile retrieves a device profile by ID
func (dr *DeviceRegistry) GetProfile(id string) (*DeviceProfile, bool) {
	profile, exists := dr.profiles[id]
	return profile, exists
}

// GetAllProfiles returns all registered device profiles
func (dr *DeviceRegistry) GetAllProfiles() map[string]*DeviceProfile {
	return dr.profiles
}

// RegisterDetector adds a device detector to the registry
func (dr *DeviceRegistry) RegisterDetector(detector DeviceDetector) {
	dr.detectors = append(dr.detectors, detector)
}

// SetDefaultProfile sets the fallback profile when no specific device is detected
func (dr *DeviceRegistry) SetDefaultProfile(profile *DeviceProfile) {
	dr.defaultProfile = profile
}

// DetectDevice attempts to identify the device type from an HTTP request
func (dr *DeviceRegistry) DetectDevice(request *http.Request, body []byte) (*DeviceProfile, error) {
	// Try each detector in priority order
	for _, detector := range dr.detectors {
		if profile, err := detector.DetectDevice(request, body); err == nil && profile != nil {
			return profile, nil
		}
	}
	
	// If no detector matches, return default profile
	if dr.defaultProfile != nil {
		return dr.defaultProfile, nil
	}
	
	return nil, fmt.Errorf("no device profile detected and no default profile configured")
}

// DeviceMessage represents a parsed message from a detected device
type DeviceMessage struct {
	Profile     *DeviceProfile         `json:"profile"`
	DeviceID    string                 `json:"device_id"`
	DeviceName  string                 `json:"device_name"`
	MessageType string                 `json:"message_type"`
	RawData     []byte                 `json:"raw_data"`
	ParsedData  map[string]interface{} `json:"parsed_data"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// ToMQTTMessage converts device message to unified MQTT format
func (dm *DeviceMessage) ToMQTTMessage() *MQTTMessage {
	return &MQTTMessage{
		DeviceID:    dm.DeviceID,
		DeviceName:  dm.DeviceName,
		EventType:   dm.MessageType,
		Data:        dm.RawData,
		DecodedData: dm.ParsedData,
	}
}

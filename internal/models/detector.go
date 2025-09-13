package models

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"strings"
)

// PathDetector detects devices based on URL path patterns
type PathDetector struct {
	profiles map[string]*DeviceProfile
}

// NewPathDetector creates a new path-based detector
func NewPathDetector() *PathDetector {
	return &PathDetector{
		profiles: make(map[string]*DeviceProfile),
	}
}

// RegisterProfile adds a device profile for path detection
func (pd *PathDetector) RegisterProfile(profile *DeviceProfile) {
	if profile.Detection.Method == DetectionMethodPath {
		pd.profiles[profile.Detection.Pattern] = profile
	}
}

// DetectDevice attempts to detect device type from URL path
func (pd *PathDetector) DetectDevice(request *http.Request, body []byte) (*DeviceProfile, error) {
	path := request.URL.Path
	
	// Try exact path matches first
	for pattern, profile := range pd.profiles {
		if pattern == path {
			return profile, nil
		}
	}
	
	// Try pattern matching (simple wildcard support)
	for pattern, profile := range pd.profiles {
		if pd.matchesPattern(path, pattern) {
			return profile, nil
		}
	}
	
	return nil, fmt.Errorf("no device profile matches path: %s", path)
}

// matchesPattern performs simple wildcard pattern matching
func (pd *PathDetector) matchesPattern(path, pattern string) bool {
	// Convert wildcard pattern to regex
	escaped := regexp.QuoteMeta(pattern)
	regex := strings.ReplaceAll(escaped, "\\*", ".*")
	regex = "^" + regex + "$"
	
	matched, _ := regexp.MatchString(regex, path)
	return matched
}

// GetSupportedMethods returns supported detection methods
func (pd *PathDetector) GetSupportedMethods() []DetectionMethod {
	return []DetectionMethod{DetectionMethodPath}
}

// GetPriority returns detector priority
func (pd *PathDetector) GetPriority() int {
	return 100 // High priority for path detection
}

// HeaderDetector detects devices based on HTTP headers
type HeaderDetector struct {
	profiles []*DeviceProfile
}

// NewHeaderDetector creates a new header-based detector
func NewHeaderDetector() *HeaderDetector {
	return &HeaderDetector{
		profiles: make([]*DeviceProfile, 0),
	}
}

// RegisterProfile adds a device profile for header detection
func (hd *HeaderDetector) RegisterProfile(profile *DeviceProfile) {
	if profile.Detection.Method == DetectionMethodHeader || profile.Detection.Method == DetectionMethodHeaders {
		hd.profiles = append(hd.profiles, profile)
	}
}

// DetectDevice attempts to detect device type from HTTP headers
func (hd *HeaderDetector) DetectDevice(request *http.Request, body []byte) (*DeviceProfile, error) {
	// Sort profiles by priority (higher first)
	sort.Slice(hd.profiles, func(i, j int) bool {
		return hd.profiles[i].Detection.Priority > hd.profiles[j].Detection.Priority
	})
	
	for _, profile := range hd.profiles {
		if hd.matchesProfile(request, profile) {
			return profile, nil
		}
	}
	
	return nil, fmt.Errorf("no device profile matches headers")
}

// matchesProfile checks if request headers match a device profile
func (hd *HeaderDetector) matchesProfile(request *http.Request, profile *DeviceProfile) bool {
	detection := profile.Detection
	
	switch detection.Method {
	case DetectionMethodHeader:
		// Single header matching
		headerValue := request.Header.Get(detection.Header)
		return headerValue == detection.Value
		
	case DetectionMethodHeaders:
		// Multiple headers matching
		for headerName, expectedValue := range detection.Headers {
			actualValue := request.Header.Get(headerName)
			if actualValue != expectedValue {
				return false
			}
		}
		return true
	}
	
	return false
}

// GetSupportedMethods returns supported detection methods
func (hd *HeaderDetector) GetSupportedMethods() []DetectionMethod {
	return []DetectionMethod{DetectionMethodHeader, DetectionMethodHeaders}
}

// GetPriority returns detector priority
func (hd *HeaderDetector) GetPriority() int {
	return 90 // Medium-high priority for header detection
}

// PayloadDetector detects devices based on payload structure
type PayloadDetector struct {
	profiles []*DeviceProfile
}

// NewPayloadDetector creates a new payload-based detector
func NewPayloadDetector() *PayloadDetector {
	return &PayloadDetector{
		profiles: make([]*DeviceProfile, 0),
	}
}

// RegisterProfile adds a device profile for payload detection
func (pd *PayloadDetector) RegisterProfile(profile *DeviceProfile) {
	if profile.Detection.Method == DetectionMethodPayload {
		pd.profiles = append(pd.profiles, profile)
	}
}

// DetectDevice attempts to detect device type from payload structure
func (pd *PayloadDetector) DetectDevice(request *http.Request, body []byte) (*DeviceProfile, error) {
	if len(body) == 0 {
		return nil, fmt.Errorf("empty payload for payload detection")
	}
	
	// Sort profiles by priority (higher first)
	sort.Slice(pd.profiles, func(i, j int) bool {
		return pd.profiles[i].Detection.Priority > pd.profiles[j].Detection.Priority
	})
	
	for _, profile := range pd.profiles {
		if pd.matchesPayload(body, profile) {
			return profile, nil
		}
	}
	
	return nil, fmt.Errorf("no device profile matches payload structure")
}

// matchesPayload checks if payload structure matches a device profile
func (pd *PayloadDetector) matchesPayload(body []byte, profile *DeviceProfile) bool {
	// Try to parse as JSON
	var payload map[string]interface{}
	if err := json.Unmarshal(body, &payload); err != nil {
		return false
	}
	
	// Check for ChirpStack-specific structure
	if profile.Make == "chirpstack" {
		_, hasEvent := payload["event"]
		return hasEvent
	}
	
	// For other profiles, check if required fields exist
	for fieldName := range profile.Fields {
		if _, exists := payload[fieldName]; !exists {
			// Field not found, this profile doesn't match
			return false
		}
	}
	
	return true
}

// GetSupportedMethods returns supported detection methods
func (pd *PayloadDetector) GetSupportedMethods() []DetectionMethod {
	return []DetectionMethod{DetectionMethodPayload}
}

// GetPriority returns detector priority
func (pd *PayloadDetector) GetPriority() int {
	return 50 // Lower priority for payload detection (more expensive)
}

// RuleBasedDetector detects devices based on complex rules
type RuleBasedDetector struct {
	profiles []*DeviceProfile
}

// NewRuleBasedDetector creates a new rule-based detector
func NewRuleBasedDetector() *RuleBasedDetector {
	return &RuleBasedDetector{
		profiles: make([]*DeviceProfile, 0),
	}
}

// RegisterProfile adds a device profile for rule-based detection
func (rbd *RuleBasedDetector) RegisterProfile(profile *DeviceProfile) {
	if profile.Detection.Method == DetectionMethodRules {
		rbd.profiles = append(rbd.profiles, profile)
	}
}

// DetectDevice attempts to detect device type using complex rules
func (rbd *RuleBasedDetector) DetectDevice(request *http.Request, body []byte) (*DeviceProfile, error) {
	// Sort profiles by priority (higher first)
	sort.Slice(rbd.profiles, func(i, j int) bool {
		return rbd.profiles[i].Detection.Priority > rbd.profiles[j].Detection.Priority
	})
	
	for _, profile := range rbd.profiles {
		if rbd.matchesRules(request, body, profile) {
			return profile, nil
		}
	}
	
	return nil, fmt.Errorf("no device profile matches detection rules")
}

// matchesRules checks if request matches all rules in a device profile
func (rbd *RuleBasedDetector) matchesRules(request *http.Request, body []byte, profile *DeviceProfile) bool {
	for _, rule := range profile.Detection.Rules {
		if !rbd.matchesRule(request, body, rule) {
			if rule.Required {
				return false // Required rule failed
			}
		}
	}
	return true
}

// matchesRule checks if a single rule matches
func (rbd *RuleBasedDetector) matchesRule(request *http.Request, body []byte, rule DetectionRule) bool {
	switch rule.Type {
	case "header":
		return rbd.matchesHeaderRule(request, rule)
	case "path":
		return rbd.matchesPathRule(request, rule)
	case "payload":
		return rbd.matchesPayloadRule(body, rule)
	default:
		return false
	}
}

// matchesHeaderRule checks if header rule matches
func (rbd *RuleBasedDetector) matchesHeaderRule(request *http.Request, rule DetectionRule) bool {
	headerValue := request.Header.Get(rule.Field)
	
	switch rule.Operator {
	case "equals":
		return headerValue == rule.Value
	case "contains":
		return strings.Contains(headerValue, rule.Value)
	case "exists":
		return headerValue != ""
	case "regex":
		matched, _ := regexp.MatchString(rule.Value, headerValue)
		return matched
	default:
		return false
	}
}

// matchesPathRule checks if path rule matches
func (rbd *RuleBasedDetector) matchesPathRule(request *http.Request, rule DetectionRule) bool {
	path := request.URL.Path
	
	switch rule.Operator {
	case "equals":
		return path == rule.Value
	case "contains":
		return strings.Contains(path, rule.Value)
	case "regex":
		matched, _ := regexp.MatchString(rule.Value, path)
		return matched
	default:
		return false
	}
}

// matchesPayloadRule checks if payload rule matches
func (rbd *RuleBasedDetector) matchesPayloadRule(body []byte, rule DetectionRule) bool {
	var payload map[string]interface{}
	if err := json.Unmarshal(body, &payload); err != nil {
		return false
	}
	
	// Simple field existence check (can be enhanced with JSON path)
	value, exists := payload[rule.Field]
	
	switch rule.Operator {
	case "exists":
		return exists
	case "equals":
		if !exists {
			return false
		}
		return fmt.Sprintf("%v", value) == rule.Value
	case "contains":
		if !exists {
			return false
		}
		return strings.Contains(fmt.Sprintf("%v", value), rule.Value)
	default:
		return false
	}
}

// GetSupportedMethods returns supported detection methods
func (rbd *RuleBasedDetector) GetSupportedMethods() []DetectionMethod {
	return []DetectionMethod{DetectionMethodRules}
}

// GetPriority returns detector priority
func (rbd *RuleBasedDetector) GetPriority() int {
	return 80 // Medium priority for rule-based detection
}

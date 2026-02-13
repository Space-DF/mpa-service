package models

import "time"

// Location represents GPS coordinates
type Location struct {
	Latitude  float64 `json:"latitude,omitempty"`
	Longitude float64 `json:"longitude,omitempty"`
	Altitude  float64 `json:"altitude,omitempty"`
}

// MQTTMessage represents the unified message structure to be sent to MQTT
type MQTTMessage struct {
	DeviceID   string    `json:"device_id"`
	DeviceName string    `json:"device_name"`
	Timestamp  time.Time `json:"timestamp"`
	EventType  string    `json:"event_type"`

	// Raw message data (preserved from original)
	Data        []byte                 `json:"raw_data,omitempty"`
	DecodedData map[string]interface{} `json:"decoded_data,omitempty"`
	Location    *Location              `json:"location,omitempty"`

	// Transport metadata (for forwarding mode)
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

package models

import "time"

// DeviceInfo contains common device information across all events
type DeviceInfo struct {
	TenantID        string            `json:"tenantId,omitempty"`
	TenantName      string            `json:"tenantName,omitempty"`
	ApplicationID   string            `json:"applicationId,omitempty"`
	ApplicationName string            `json:"applicationName,omitempty"`
	DeviceProfileID string            `json:"deviceProfileId,omitempty"`
	DeviceProfile   DeviceProfile     `json:"deviceProfile,omitempty"`
	DeviceName      string            `json:"deviceName,omitempty"`
	DevEUI          string            `json:"devEui,omitempty"`
	DevAddr         string            `json:"devAddr,omitempty"`
	Tags            map[string]string `json:"tags,omitempty"`
	Variables       map[string]string `json:"variables,omitempty"`
}

// BaseEvent contains common fields for all event types
type BaseEvent struct {
	DeduplicationID string    `json:"deduplicationId,omitempty"`
	Time            time.Time `json:"time,omitempty"`
	DeviceInfo      DeviceInfo `json:"deviceInfo,omitempty"`
}

// UplinkEvent represents uplink data from devices
type UplinkEvent struct {
	BaseEvent
	DevAddr         string                 `json:"devAddr,omitempty"`
	ADR             bool                   `json:"adr,omitempty"`
	DR              int                    `json:"dr,omitempty"`
	FCnt            uint32                 `json:"fCnt,omitempty"`
	FPort           uint32                 `json:"fPort,omitempty"`
	Data            []byte                 `json:"data,omitempty"`
	Object          map[string]interface{} `json:"object,omitempty"`
	ConfirmedUplink bool                   `json:"confirmedUplink,omitempty"`
	TxInfo          TxInfo                 `json:"txInfo,omitempty"`
	RxInfo          []RxInfo               `json:"rxInfo,omitempty"`
}

// JoinEvent represents device join events
type JoinEvent struct {
	BaseEvent
	DevAddr string `json:"devAddr,omitempty"`
}

// AckEvent represents downlink acknowledgment events
type AckEvent struct {
	BaseEvent
	QueueItemID string `json:"queueItemId,omitempty"`
	Acknowledged bool   `json:"acknowledged,omitempty"`
	FCntDown    uint32 `json:"fCntDown,omitempty"`
}

// TxAckEvent represents downlink transmission acknowledgment
type TxAckEvent struct {
	BaseEvent
	DownlinkID string `json:"downlinkId,omitempty"`
	QueueItemID string `json:"queueItemId,omitempty"`
	FCntDown   uint32 `json:"fCntDown,omitempty"`
	GatewayID  string `json:"gatewayId,omitempty"`
	TxInfo     TxInfo `json:"txInfo,omitempty"`
}

// StatusEvent represents device status events
type StatusEvent struct {
	BaseEvent
	BatteryLevel *float32 `json:"batteryLevel,omitempty"`
	Margin       int      `json:"margin,omitempty"`
}

// LocationEvent represents device location events
type LocationEvent struct {
	BaseEvent
	Location Location `json:"location,omitempty"`
}

// LogEvent represents device log events
type LogEvent struct {
	BaseEvent
	Level       string            `json:"level,omitempty"`
	Code        string            `json:"code,omitempty"`
	Description string            `json:"description,omitempty"`
	Context     map[string]string `json:"context,omitempty"`
}

// IntegrationEvent represents integration events
type IntegrationEvent struct {
	BaseEvent
	IntegrationName string                 `json:"integrationName,omitempty"`
	EventType       string                 `json:"eventType,omitempty"`
	Object          map[string]interface{} `json:"object,omitempty"`
}

// EventWrapper represents the complete webhook payload from ChirpStack
type EventWrapper struct {
	Event string `json:"event"`
	
	// Event-specific data
	UplinkEvent      *UplinkEvent      `json:"uplinkEvent,omitempty"`
	JoinEvent        *JoinEvent        `json:"joinEvent,omitempty"`
	AckEvent         *AckEvent         `json:"ackEvent,omitempty"`
	TxAckEvent       *TxAckEvent       `json:"txAckEvent,omitempty"`
	StatusEvent      *StatusEvent      `json:"statusEvent,omitempty"`
	LocationEvent    *LocationEvent    `json:"locationEvent,omitempty"`
	LogEvent         *LogEvent         `json:"logEvent,omitempty"`
	IntegrationEvent *IntegrationEvent `json:"integrationEvent,omitempty"`
}



type DeviceProfile struct {
	ID            string            `json:"id,omitempty"`
	Name          string            `json:"name,omitempty"`
	Vendor        string            `json:"vendor,omitempty"`
	Description   string            `json:"description,omitempty"`
	Region        string            `json:"region,omitempty"`
	MACVersion    string            `json:"macVersion,omitempty"`
	RegParamsRev  string            `json:"regParamsRevision,omitempty"`
	ADRAlgorithm  map[string]string `json:"adrAlgorithm,omitempty"`
}

type TxInfo struct {
	Frequency  int                    `json:"frequency,omitempty"`
	Modulation string                 `json:"modulation,omitempty"`
	LoRaModulationInfo *LoRaModulationInfo `json:"loRaModulationInfo,omitempty"`
}

type LoRaModulationInfo struct {
	Bandwidth       int    `json:"bandwidth,omitempty"`
	SpreadingFactor int    `json:"spreadingFactor,omitempty"`
	CodeRate        string `json:"codeRate,omitempty"`
}

type RxInfo struct {
	GatewayID string  `json:"gatewayId,omitempty"`
	UplinkID  string  `json:"uplinkId,omitempty"`
	RSSI      float64 `json:"rssi,omitempty"`
	SNR       float64 `json:"snr,omitempty"`
	Channel   int     `json:"channel,omitempty"`
	RFChain   int     `json:"rfChain,omitempty"`
	Location  Location `json:"location,omitempty"`
	Antenna   int     `json:"antenna,omitempty"`
}

type Location struct {
	Latitude  float64 `json:"latitude,omitempty"`
	Longitude float64 `json:"longitude,omitempty"`
	Altitude  float64 `json:"altitude,omitempty"`
}

// MQTTMessage represents the unified message structure to be sent to MQTT
type MQTTMessage struct {
	DeviceID        string                 `json:"device_id"`
	DeviceName      string                 `json:"device_name"`
	Timestamp       time.Time              `json:"timestamp"`
	EventType       string                 `json:"event_type"`
	
	// Uplink-specific fields
	Data           []byte                 `json:"raw_data,omitempty"`
	DecodedData    map[string]interface{} `json:"decoded_data,omitempty"`
	Port           int                    `json:"port,omitempty"`
	FCnt           int                    `json:"frame_counter,omitempty"`
	RSSI           float64                `json:"rssi,omitempty"`
	SNR            float64                `json:"snr,omitempty"`
	Location       *Location              `json:"location,omitempty"`
	
	// Join event fields
	JoinInfo       map[string]interface{} `json:"join_info,omitempty"`
	
	// Ack event fields
	AckInfo        map[string]interface{} `json:"ack_info,omitempty"`
	
	// TxAck event fields
	TxAckInfo      map[string]interface{} `json:"tx_ack_info,omitempty"`
	
	// Status event fields
	StatusInfo     map[string]interface{} `json:"status_info,omitempty"`
	
	// Log event fields
	LogInfo        map[string]interface{} `json:"log_info,omitempty"`
	
	// Integration event fields
	IntegrationInfo map[string]interface{} `json:"integration_info,omitempty"`
}

// EventType represents the type of ChirpStack event
type EventType string

const (
	EventUplink      EventType = "up"
	EventJoin        EventType = "join"
	EventAck         EventType = "ack"
	EventTxAck       EventType = "txack"
	EventStatus      EventType = "status"
	EventLocation    EventType = "location"
	EventLog         EventType = "log"
	EventIntegration EventType = "integration"
)
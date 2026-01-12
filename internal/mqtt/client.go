package mqtt

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/Space-DF/mpa-service/internal/config"
	"github.com/Space-DF/mpa-service/internal/models"
	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type Client struct {
	client mqtt.Client
	config config.MQTTConfig
}

type ClientInterface interface {
	Connect() error
	Disconnect()
	Publish(deviceData *models.MQTTMessage) error
	IsConnected() bool
}

func NewClient(config config.MQTTConfig) ClientInterface {
	// Validate configuration at startup
	if err := validateConfig(config); err != nil {
		log.Printf("Warning: MQTT configuration validation failed: %v", err)
	}

	return &Client{
		config: config,
	}
}

// validateConfig ensures the MQTT configuration is valid
func validateConfig(cfg config.MQTTConfig) error {
	// Skip validation if MQTT is disabled (broker is empty)
	if cfg.Broker == "" {
		return nil
	}

	if cfg.Port <= 0 {
		return fmt.Errorf("port must be positive")
	}
	if cfg.ClientID == "" {
		return fmt.Errorf("client ID is required")
	}
	if cfg.TopicTemplate != "" {
		if !strings.Contains(cfg.TopicTemplate, "{tenant_id}") {
			return fmt.Errorf("topic template must contain {tenant_id}")
		}
	}
	return nil
}

func (c *Client) Connect() error {
	broker := fmt.Sprintf("tcp://%s:%d", c.config.Broker, c.config.Port)

	opts := mqtt.NewClientOptions()
	opts.AddBroker(broker)
	opts.SetClientID(c.config.ClientID)
	opts.SetUsername(c.config.Username)
	opts.SetPassword(c.config.Password)
	opts.SetAutoReconnect(true)
	opts.SetConnectRetry(true)
	opts.SetConnectRetryInterval(5 * time.Second)
	opts.SetConnectTimeout(60 * time.Second)       // Increase connection timeout to 60 seconds
	opts.SetKeepAlive(60 * time.Second)            // Set keep alive interval
	opts.SetPingTimeout(10 * time.Second)          // Set ping timeout
	opts.SetMaxReconnectInterval(30 * time.Second) // Max reconnect interval

	opts.SetDefaultPublishHandler(func(client mqtt.Client, msg mqtt.Message) {
		log.Printf("Received message on topic: %s", msg.Topic())
	})

	opts.SetOnConnectHandler(func(client mqtt.Client) {
		log.Printf("Connected to MQTT broker at %s", broker)
	})

	opts.SetConnectionLostHandler(func(client mqtt.Client, err error) {
		log.Printf("Connection lost to MQTT broker: %v", err)
	})

	c.client = mqtt.NewClient(opts)

	if token := c.client.Connect(); token.Wait() && token.Error() != nil {
		return fmt.Errorf("failed to connect to MQTT broker: %w", token.Error())
	}

	return nil
}

func (c *Client) Disconnect() {
	if c.client != nil && c.client.IsConnected() {
		c.client.Disconnect(250)
		log.Println("Disconnected from MQTT broker")
	}
}

func (c *Client) Publish(deviceData *models.MQTTMessage) error {
	if !c.client.IsConnected() {
		return fmt.Errorf("MQTT client is not connected")
	}

	// 1. Determine topic first (cheap operation)
	topic, err := c.determineTopic(deviceData)
	if err != nil {
		return fmt.Errorf("topic determination failed: %w", err)
	}

	// 2. Validate topic
	if err := c.validateTopic(topic); err != nil {
		return err
	}

	// 3. Marshal payload (expensive operation - only after validation)
	payload, err := json.Marshal(deviceData)
	if err != nil {
		return fmt.Errorf("failed to marshal device data: %w", err)
	}

	// 4. Publish to MQTT
	return c.publishToTopic(topic, payload, deviceData.DeviceID)
}

// determineTopic extracts or generates the appropriate topic for publishing
func (c *Client) determineTopic(deviceData *models.MQTTMessage) (string, error) {
	// Check for forwarding topic first
	if deviceData.Metadata != nil {
		if topic, ok := deviceData.Metadata["mqtt_topic"].(string); ok && topic != "" {
			return topic, nil
		}
	}

	// Generate tenant-based topic
	return c.generateTenantTopic(deviceData)
}

// validateTopic ensures the topic meets our simplified requirements
func (c *Client) validateTopic(topic string) error {
	if topic == "" {
		return fmt.Errorf("topic cannot be empty")
	}

	// Only accept simplified tenant-based pattern: tenant/{tenant}/device/data
	if !strings.Contains(topic, "/device/") || !strings.HasSuffix(topic, "/data") || !strings.HasPrefix(topic, "tenant/") {
		return fmt.Errorf("topic must follow pattern tenant/{tenant}/device/data: %s", topic)
	}

	// Ensure it has exactly 4 parts: tenant/{tenant}/device/data
	parts := strings.Split(topic, "/")
	if len(parts) != 4 {
		return fmt.Errorf("topic must have exactly 4 parts: tenant/{tenant}/device/data: %s", topic)
	}

	return nil
}

// publishToTopic handles the actual MQTT publishing
func (c *Client) publishToTopic(topic string, payload []byte, deviceID string) error {
	token := c.client.Publish(topic, c.config.QOS, c.config.Retained, payload)
	token.Wait()

	if token.Error() != nil {
		return fmt.Errorf("failed to publish message: %w", token.Error())
	}

	log.Printf("Published to %s for device %s", topic, deviceID)
	return nil
}

func (c *Client) IsConnected() bool {
	return c.client != nil && c.client.IsConnected()
}

// generateTenantTopic creates a topic using tenant information
func (c *Client) generateTenantTopic(deviceData *models.MQTTMessage) (string, error) {
	if c.config.TopicTemplate == "" {
		return "", fmt.Errorf("topic template not configured")
	}

	// Check if we have tenant information in metadata - REQUIRED
	if deviceData == nil || deviceData.Metadata == nil {
		return "", fmt.Errorf("device data and metadata are required")
	}

	tenantID, ok := deviceData.Metadata["tenant_id"].(string)
	if !ok || tenantID == "" {
		return "", fmt.Errorf("tenant_id is required in metadata")
	}

	// Generate simplified tenant-based topic: {tenant}/device/data
	topic := fmt.Sprintf("tenant/%s/device/data", tenantID)
	return topic, nil
}

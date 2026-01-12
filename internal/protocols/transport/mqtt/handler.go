package mqttprotocol

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/Space-DF/mpa-service/internal/protocols/handlers"
	"github.com/Space-DF/mpa-service/internal/services"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/labstack/echo/v4"
)

// Handler implements MQTT subscriber transport handler for multi-device support
type Handler struct {
	deviceService *services.DeviceService
	config        Config
	mqttClient    mqtt.Client
	isRunning     bool
	mutex         sync.RWMutex
	messageCount  int64
	lastMessage   time.Time
}

// Config holds MQTT transport handler configuration
type Config struct {
	Broker          string   `yaml:"broker"`
	Port            int      `yaml:"port"`
	ClientID        string   `yaml:"client_id"`
	Username        string   `yaml:"username"`
	Password        string   `yaml:"password"`
	SubscribeTopics []string `yaml:"subscribe_topics"`
	QOS             byte     `yaml:"qos"`
}

// NewHandler creates a new MQTT subscriber transport handler
func NewHandler(deviceService *services.DeviceService, config Config) handlers.ProtocolHandler {
	return &Handler{
		deviceService: deviceService,
		config:        config,
		isRunning:     false,
		messageCount:  0,
	}
}

// Name returns the transport protocol name
func (h *Handler) Name() string {
	return "mqtt-subscriber"
}

// Path returns empty string as MQTT doesn't use HTTP paths
func (h *Handler) Path() string {
	return "" // MQTT doesn't use HTTP paths
}

// Method returns empty string as MQTT doesn't use HTTP methods
func (h *Handler) Method() string {
	return "" // MQTT doesn't use HTTP methods
}

// Handle is not used for MQTT (non-HTTP protocol)
func (h *Handler) Handle(c echo.Context) error {
	return echo.NewHTTPError(404, "MQTT transport doesn't support HTTP requests")
}

// HealthCheck returns health status for MQTT transport handler
func (h *Handler) HealthCheck(c echo.Context) error {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	isConnected := h.mqttClient != nil && h.mqttClient.IsConnected()
	healthStatus := h.deviceService.GetHealthStatus()

	status := "healthy"
	if !isConnected || !h.isRunning {
		status = "unhealthy"
	}

	return c.JSON(200, map[string]interface{}{
		"transport":          "mqtt-subscriber",
		"status":             status,
		"message":            "MQTT subscriber transport handler status",
		"mqtt_connected":     isConnected,
		"subscriber_running": h.isRunning,
		"parsers":            healthStatus["parsers"],
		"broker":             fmt.Sprintf("%s:%d", h.config.Broker, h.config.Port),
		"subscribe_topics":   h.config.SubscribeTopics,
		"messages_received":  h.messageCount,
		"last_message":       h.lastMessage.Format(time.RFC3339),
		"timestamp":          time.Now().UTC().Format(time.RFC3339),
	})
}

// Start initializes and starts the MQTT subscriber
func (h *Handler) Start() error {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	if h.isRunning {
		return fmt.Errorf("MQTT subscriber already running")
	}

	// Create MQTT client options
	broker := fmt.Sprintf("tcp://%s:%d", h.config.Broker, h.config.Port)
	opts := mqtt.NewClientOptions()
	opts.AddBroker(broker)
	opts.SetClientID(h.config.ClientID + "-subscriber")
	opts.SetUsername(h.config.Username)
	opts.SetPassword(h.config.Password)
	opts.SetAutoReconnect(true)
	opts.SetConnectRetry(true)
	opts.SetConnectRetryInterval(5 * time.Second)
	opts.SetKeepAlive(60 * time.Second)

	// Set connection handlers
	opts.SetOnConnectHandler(func(client mqtt.Client) {
		log.Printf("MQTT Subscriber: Connected to broker at %s", broker)
		h.subscribeToTopics()
	})

	opts.SetConnectionLostHandler(func(client mqtt.Client, err error) {
		log.Printf("MQTT Subscriber: Connection lost to broker: %v", err)
	})

	opts.SetReconnectingHandler(func(client mqtt.Client, options *mqtt.ClientOptions) {
		log.Printf("MQTT Subscriber: Attempting to reconnect to broker...")
	})

	// Create and connect client
	h.mqttClient = mqtt.NewClient(opts)

	if token := h.mqttClient.Connect(); token.Wait() && token.Error() != nil {
		return fmt.Errorf("failed to connect to MQTT broker: %w", token.Error())
	}

	h.isRunning = true
	log.Printf("MQTT Subscriber: Started and connected to %s", broker)

	return nil
}

// Stop gracefully stops the MQTT subscriber
func (h *Handler) Stop() error {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	if !h.isRunning {
		return fmt.Errorf("MQTT subscriber not running")
	}

	if h.mqttClient != nil && h.mqttClient.IsConnected() {
		// Unsubscribe from all topics
		for _, topic := range h.config.SubscribeTopics {
			if token := h.mqttClient.Unsubscribe(topic); token.Wait() && token.Error() != nil {
				log.Printf("MQTT Subscriber: Error unsubscribing from topic %s: %v", topic, token.Error())
			}
		}

		h.mqttClient.Disconnect(250)
		log.Printf("MQTT Subscriber: Disconnected from broker")
	}

	h.isRunning = false
	return nil
}

// subscribeToTopics subscribes to configured topics
func (h *Handler) subscribeToTopics() {
	for _, topic := range h.config.SubscribeTopics {
		token := h.mqttClient.Subscribe(topic, h.config.QOS, h.messageHandler)
		if token.Wait() && token.Error() != nil {
			log.Printf("MQTT Subscriber: Error subscribing to topic %s: %v", topic, token.Error())
		} else {
			log.Printf("MQTT Subscriber: Subscribed to topic: %s", topic)
		}
	}
}

// messageHandler processes incoming MQTT messages
func (h *Handler) messageHandler(client mqtt.Client, msg mqtt.Message) {
	h.mutex.Lock()
	h.messageCount++
	h.lastMessage = time.Now()
	h.mutex.Unlock()

	log.Printf("MQTT Subscriber: Received message on topic: %s (payload size: %d bytes)",
		msg.Topic(), len(msg.Payload()))

	// Prepare transport metadata
	transportMetadata := map[string]interface{}{
		"transport":   "mqtt-subscriber",
		"topic":       msg.Topic(),
		"qos":         msg.Qos(),
		"retained":    msg.Retained(),
		"duplicate":   msg.Duplicate(),
		"message_id":  msg.MessageID(),
		"received_at": h.lastMessage.UTC().Format(time.RFC3339),
	}

	// Process message through device service
	if err := h.deviceService.ProcessMQTTMessage(msg.Topic(), msg.Payload(), transportMetadata); err != nil {
		log.Printf("MQTT Subscriber: Error processing message from topic %s: %v", msg.Topic(), err)
		return
	}

	log.Printf("MQTT Subscriber: Successfully processed message from topic: %s", msg.Topic())
}

// IsRunning returns whether the MQTT subscriber is currently running
func (h *Handler) IsRunning() bool {
	h.mutex.RLock()
	defer h.mutex.RUnlock()
	return h.isRunning
}

// GetMessageCount returns the total number of messages received
func (h *Handler) GetMessageCount() int64 {
	h.mutex.RLock()
	defer h.mutex.RUnlock()
	return h.messageCount
}

// GetLastMessageTime returns the timestamp of the last received message
func (h *Handler) GetLastMessageTime() time.Time {
	h.mutex.RLock()
	defer h.mutex.RUnlock()
	return h.lastMessage
}

// GetSubscribedTopics returns the list of subscribed topics
func (h *Handler) GetSubscribedTopics() []string {
	return h.config.SubscribeTopics
}

// AddSubscription adds a new topic subscription at runtime
func (h *Handler) AddSubscription(topic string) error {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	if !h.isRunning || h.mqttClient == nil || !h.mqttClient.IsConnected() {
		return fmt.Errorf("MQTT subscriber not running or not connected")
	}

	token := h.mqttClient.Subscribe(topic, h.config.QOS, h.messageHandler)
	if token.Wait() && token.Error() != nil {
		return fmt.Errorf("failed to subscribe to topic %s: %w", topic, token.Error())
	}

	// Add to config for tracking
	h.config.SubscribeTopics = append(h.config.SubscribeTopics, topic)
	log.Printf("MQTT Subscriber: Added subscription to topic: %s", topic)

	return nil
}

// RemoveSubscription removes a topic subscription at runtime
func (h *Handler) RemoveSubscription(topic string) error {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	if !h.isRunning || h.mqttClient == nil || !h.mqttClient.IsConnected() {
		return fmt.Errorf("MQTT subscriber not running or not connected")
	}

	token := h.mqttClient.Unsubscribe(topic)
	if token.Wait() && token.Error() != nil {
		return fmt.Errorf("failed to unsubscribe from topic %s: %w", topic, token.Error())
	}

	// Remove from config
	for i, t := range h.config.SubscribeTopics {
		if t == topic {
			h.config.SubscribeTopics = append(h.config.SubscribeTopics[:i], h.config.SubscribeTopics[i+1:]...)
			break
		}
	}

	log.Printf("MQTT Subscriber: Removed subscription from topic: %s", topic)
	return nil
}

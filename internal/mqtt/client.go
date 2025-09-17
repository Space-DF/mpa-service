package mqtt

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/Space-DF/mpa-service/internal/config"
	"github.com/Space-DF/mpa-service/internal/models"
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
	return &Client{
		config: config,
	}
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
	
	payload, err := json.Marshal(deviceData)
	if err != nil {
		return fmt.Errorf("failed to marshal device data: %w", err)
	}
	
	token := c.client.Publish(c.config.Topic, c.config.QOS, c.config.Retained, payload)
	token.Wait()
	
	if token.Error() != nil {
		return fmt.Errorf("failed to publish message: %w", token.Error())
	}
	
	log.Printf("Published message to topic %s for device %s", c.config.Topic, deviceData.DeviceID)
	return nil
}

func (c *Client) IsConnected() bool {
	return c.client != nil && c.client.IsConnected()
}

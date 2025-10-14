package config

import (
	"log"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	MQTT     MQTTConfig     `mapstructure:"mqtt"`
	Protocols ProtocolsConfig `mapstructure:"protocols"`
}

type ServerConfig struct {
	Port         int    `mapstructure:"port" env:"SERVER_PORT"`
	LogLevel     string `mapstructure:"log_level" env:"SERVER_LOG_LEVEL"`
	ReadTimeout  int    `mapstructure:"read_timeout" env:"SERVER_READ_TIMEOUT"`
	WriteTimeout int    `mapstructure:"write_timeout" env:"SERVER_WRITE_TIMEOUT"`
	IdleTimeout  int    `mapstructure:"idle_timeout" env:"SERVER_IDLE_TIMEOUT"`
}

type ProtocolsConfig struct {
	HTTP       HTTPConfig         `mapstructure:"http"`
	SMS        SMSConfig          `mapstructure:"sms"`
	WebSocket  WebSocketConfig    `mapstructure:"websocket"`
	MQTT       MQTTProtocolConfig `mapstructure:"mqtt_protocol"`
	LoRaWAN    LoRaWANConfig      `mapstructure:"lorawan"`
	// Backward compatibility - these will be deprecated
	ChirpStack ChirpStackConfig   `mapstructure:"chirpstack"`
	TTN        TTNConfig          `mapstructure:"ttn"`
	Helium     HeliumConfig       `mapstructure:"helium"`
}

// Protocol-specific configurations
type HTTPConfig struct {
	Enabled bool   `mapstructure:"enabled" env:"PROTOCOLS_HTTP_ENABLED"`
	Path    string `mapstructure:"path" env:"PROTOCOLS_HTTP_PATH"`
}

type SMSConfig struct {
	Enabled    bool   `mapstructure:"enabled" env:"PROTOCOLS_SMS_ENABLED"`
	Provider   string `mapstructure:"provider" env:"PROTOCOLS_SMS_PROVIDER"`
	APIKey     string `mapstructure:"api_key" env:"PROTOCOLS_SMS_API_KEY"`
	APISecret  string `mapstructure:"api_secret" env:"PROTOCOLS_SMS_API_SECRET"`
	WebhookURL string `mapstructure:"webhook_url" env:"PROTOCOLS_SMS_WEBHOOK_URL"`
	Port       int    `mapstructure:"port" env:"PROTOCOLS_SMS_PORT"`
}

type WebSocketConfig struct {
	Enabled bool   `mapstructure:"enabled" env:"PROTOCOLS_WEBSOCKET_ENABLED"`
	Path    string `mapstructure:"path" env:"PROTOCOLS_WEBSOCKET_PATH"`
	Port    int    `mapstructure:"port" env:"PROTOCOLS_WEBSOCKET_PORT"`
}

type MQTTProtocolConfig struct {
	Enabled bool `mapstructure:"enabled" env:"PROTOCOLS_MQTT_PROTOCOL_ENABLED"`
	Port    int  `mapstructure:"port" env:"PROTOCOLS_MQTT_PROTOCOL_PORT"`
}

type ChirpStackConfig struct {
	Enabled bool `mapstructure:"enabled" env:"PROTOCOLS_CHIRPSTACK_ENABLED"`
}

type TTNConfig struct {
	Enabled bool `mapstructure:"enabled" env:"PROTOCOLS_TTN_ENABLED"`
}

type HeliumConfig struct {
	Enabled bool `mapstructure:"enabled" env:"PROTOCOLS_HELIUM_ENABLED"`
}

// LoRaWANConfig defines configuration for LoRaWAN providers
type LoRaWANConfig struct {
	Providers map[string]LoRaWANProviderConfig `mapstructure:"providers"`
}

// LoRaWANProviderConfig defines configuration for a single LoRaWAN provider
type LoRaWANProviderConfig struct {
	Enabled bool   `mapstructure:"enabled"`
	Name    string `mapstructure:"name"` // Display name (optional)
}

type MQTTConfig struct {
	Broker          string   `mapstructure:"broker" env:"MQTT_BROKER"`
	Port            int      `mapstructure:"port" env:"MQTT_PORT"`
	ClientID        string   `mapstructure:"client_id" env:"MQTT_CLIENT_ID"`
	Username        string   `mapstructure:"username" env:"MQTT_USERNAME"`
	Password        string   `mapstructure:"password" env:"MQTT_PASSWORD"`
	Topic           string   `mapstructure:"topic" env:"MQTT_TOPIC"`
	TopicTemplate   string   `mapstructure:"topic_template" env:"MQTT_TOPIC_TEMPLATE"`
	SubscribeTopics []string `mapstructure:"subscribe_topics" env:"MQTT_SUBSCRIBE_TOPICS"`
	QOS             byte     `mapstructure:"qos" env:"MQTT_QOS"`
	Retained        bool     `mapstructure:"retained" env:"MQTT_RETAINED"`
}


func New() (Config, error) {
	var config Config

	vp := viper.New()
	
	// Set defaults first (lowest priority)
	setDefaults(vp)
	log.Printf("DEBUG: Default ChirpStack enabled: %v", vp.Get("protocols.chirpstack.enabled"))

	// Load config file (medium priority) 
	vp.SetConfigFile("configs/config.yaml")
	if err := vp.ReadInConfig(); err != nil {
		log.Printf("Config file not found, using defaults and environment variables")
	} else {
		log.Printf("DEBUG: Config file loaded, ChirpStack enabled: %v", vp.Get("protocols.chirpstack.enabled"))
	}

	// Load .env file (higher priority)
	if err := godotenv.Load(".env"); err != nil {
		log.Printf("No .env file found")
	} else {
		log.Printf("DEBUG: .env file loaded")
	}

	// Enable OS environment variables (highest priority)
	vp.AutomaticEnv()
	vp.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	vp.SetEnvPrefix("")
	
	log.Printf("DEBUG: Environment variable PROTOCOLS_CHIRPSTACK_ENABLED: %s", os.Getenv("PROTOCOLS_CHIRPSTACK_ENABLED"))
	log.Printf("DEBUG: Viper value for protocols.chirpstack.enabled: %v", vp.Get("protocols.chirpstack.enabled"))

	err := vp.Unmarshal(&config)
	if err != nil {
		return config, err
	}
	
	log.Printf("DEBUG: Final config ChirpStack enabled: %v", config.Protocols.ChirpStack.Enabled)
	return config, nil
}

func setDefaults(vp *viper.Viper) {
	vp.SetDefault("server.port", 8080)
	vp.SetDefault("server.log_level", "info")
	vp.SetDefault("server.read_timeout", 30)
	vp.SetDefault("server.write_timeout", 30)
	vp.SetDefault("server.idle_timeout", 60)
	vp.SetDefault("mqtt.broker", "localhost")
	vp.SetDefault("mqtt.port", 1883)
	vp.SetDefault("mqtt.client_id", "mpa-service")
	vp.SetDefault("mqtt.topic_template", "tenant/{tenant_id}/device/data")
	vp.SetDefault("mqtt.subscribe_topics", []string{
		"*/device/data",
		"*/device/status", 
		"*/device/telemetry",
	})
	vp.SetDefault("mqtt.qos", 0)
	vp.SetDefault("mqtt.retained", false)
	vp.SetDefault("protocols.http.enabled", true)
	vp.SetDefault("protocols.http.path", "/http")
	vp.SetDefault("protocols.sms.enabled", false)
	vp.SetDefault("protocols.sms.provider", "twilio")
	vp.SetDefault("protocols.sms.port", 8081)
	vp.SetDefault("protocols.websocket.enabled", false)
	vp.SetDefault("protocols.websocket.path", "/ws")
	vp.SetDefault("protocols.websocket.port", 8082)
	vp.SetDefault("protocols.mqtt_protocol.enabled", false)
	vp.SetDefault("protocols.mqtt_protocol.port", 1884)
	vp.SetDefault("protocols.chirpstack.enabled", false)
	vp.SetDefault("protocols.ttn.enabled", false)
	vp.SetDefault("protocols.helium.enabled", false)
}

func (c Config) ReadTimeout() time.Duration {
	return time.Duration(c.Server.ReadTimeout) * time.Second
}

func (c Config) WriteTimeout() time.Duration {
	return time.Duration(c.Server.WriteTimeout) * time.Second
}

func (c Config) IdleTimeout() time.Duration {
	return time.Duration(c.Server.IdleTimeout) * time.Second
}

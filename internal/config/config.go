package config

import (
	"strings"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	MQTT     MQTTConfig     `mapstructure:"mqtt"`
	Protocols ProtocolsConfig `mapstructure:"protocols"`
}

type ServerConfig struct {
	Port         int    `mapstructure:"port"`
	LogLevel     string `mapstructure:"log_level"`
	ReadTimeout  int    `mapstructure:"read_timeout"`
	WriteTimeout int    `mapstructure:"write_timeout"`
	IdleTimeout  int    `mapstructure:"idle_timeout"`
}

type ProtocolsConfig struct {
	ChirpStack ChirpStackConfig `mapstructure:"chirpstack"`
	HTTP       HTTPConfig       `mapstructure:"http"`
	SMS        SMSConfig        `mapstructure:"sms"`
	WebSocket  WebSocketConfig  `mapstructure:"websocket"`
	MQTT       MQTTProtocolConfig `mapstructure:"mqtt_protocol"`
}

// Protocol-specific configurations
type ChirpStackConfig struct {
	Enabled bool   `mapstructure:"enabled"`
	Path    string `mapstructure:"path"`
}

type HTTPConfig struct {
	Enabled bool   `mapstructure:"enabled"`
	Path    string `mapstructure:"path"`
}

type SMSConfig struct {
	Enabled    bool   `mapstructure:"enabled"`
	Provider   string `mapstructure:"provider"`
	APIKey     string `mapstructure:"api_key"`
	APISecret  string `mapstructure:"api_secret"`
	WebhookURL string `mapstructure:"webhook_url"`
	Port       int    `mapstructure:"port"`
}

type WebSocketConfig struct {
	Enabled bool   `mapstructure:"enabled"`
	Path    string `mapstructure:"path"`
	Port    int    `mapstructure:"port"`
}

type MQTTProtocolConfig struct {
	Enabled bool `mapstructure:"enabled"`
	Port    int  `mapstructure:"port"`
}

type MQTTConfig struct {
	Broker   string `mapstructure:"broker"`
	Port     int    `mapstructure:"port"`
	ClientID string `mapstructure:"client_id"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
	Topic    string `mapstructure:"topic"`
	QOS      byte   `mapstructure:"qos"`
	Retained bool   `mapstructure:"retained"`
}


func New() (Config, error) {
	var config Config

	vp := viper.New()
	vp.SetConfigFile("configs/config.yaml")

	vp.AutomaticEnv()
	vp.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	vp.SetDefault("server.port", 8080)
	vp.SetDefault("server.log_level", "info")
	vp.SetDefault("server.read_timeout", 30)
	vp.SetDefault("server.write_timeout", 30)
	vp.SetDefault("server.idle_timeout", 60)
	vp.SetDefault("mqtt.broker", "localhost")
	vp.SetDefault("mqtt.port", 1883)
	vp.SetDefault("mqtt.client_id", "mpa-service")
	vp.SetDefault("mqtt.topic", "mpa/devices/data")
	vp.SetDefault("mqtt.qos", 0)
	vp.SetDefault("mqtt.retained", false)
	vp.SetDefault("protocols.chirpstack.enabled", true)
	vp.SetDefault("protocols.chirpstack.path", "/chirpstack")
	vp.SetDefault("protocols.http.enabled", false)
	vp.SetDefault("protocols.http.path", "/http")
	vp.SetDefault("protocols.sms.enabled", false)
	vp.SetDefault("protocols.sms.provider", "twilio")
	vp.SetDefault("protocols.sms.port", 8081)
	vp.SetDefault("protocols.websocket.enabled", false)
	vp.SetDefault("protocols.websocket.path", "/ws")
	vp.SetDefault("protocols.websocket.port", 8082)
	vp.SetDefault("protocols.mqtt_protocol.enabled", false)
	vp.SetDefault("protocols.mqtt_protocol.port", 1884)

	if err := vp.ReadInConfig(); err != nil {
		return config, err
	}

	vp.SetConfigFile(".env")
	_ = vp.MergeInConfig()

	err := vp.Unmarshal(&config)
	return config, err
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
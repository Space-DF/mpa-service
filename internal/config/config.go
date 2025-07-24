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
}

type ChirpStackConfig struct {
	Enabled bool   `mapstructure:"enabled"`
	Path    string `mapstructure:"path"`
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

type ServerConfig struct {
	LogLevel string `mapstructure:"log_level"`
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
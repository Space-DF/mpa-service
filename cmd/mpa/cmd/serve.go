package cmd

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/spf13/cobra"
	"github.com/Space-DF/mpa-service/internal/config"
	"github.com/Space-DF/mpa-service/internal/handlers"
	"github.com/Space-DF/mpa-service/internal/handlers/chirpstack"
	"github.com/Space-DF/mpa-service/internal/handlers/http"
	"github.com/Space-DF/mpa-service/internal/handlers/mqttprotocol"
	"github.com/Space-DF/mpa-service/internal/handlers/websocket"
	"github.com/Space-DF/mpa-service/internal/handlers/socketio"
	"github.com/Space-DF/mpa-service/internal/logger"
	"github.com/Space-DF/mpa-service/internal/mqtt"
	"github.com/Space-DF/mpa-service/internal/services"
)

var (
	port    int
	logLevel string
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the Multi-Protocol Agent service",
	Long: `Start the Multi-Protocol Agent (MPA) service that supports multiple
IoT transport protocols including HTTP, MQTT, WebSocket, and SocketIO.

The service automatically detects device types and formats, then publishes
unified messages to the configured MQTT broker. Supports ChirpStack, 
generic devices, and custom device profiles.`,
	Run: runServe,
}

func init() {
	rootCmd.AddCommand(serveCmd)

	serveCmd.Flags().IntVarP(&port, "port", "p", 0, "HTTP server port (overrides config)")
	serveCmd.Flags().StringVarP(&logLevel, "log-level", "l", "", "Log level (debug, info, warn, error)")
}

func runServe(cmd *cobra.Command, args []string) {
	cfg, err := config.New()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Override config with CLI flags if provided
	if port != 0 {
		cfg.Server.Port = port
	}
	if logLevel != "" {
		cfg.Server.LogLevel = logLevel
	}

	logger := logger.New(cfg.Server.LogLevel)
	
	// Initialize MQTT client for output
	mqttClient := mqtt.NewClient(cfg.MQTT)
	if err := mqttClient.Connect(); err != nil {
		logger.Errorf("Failed to connect to MQTT broker: %v", err)
		os.Exit(1)
	}
	defer mqttClient.Disconnect()

	// Initialize device service (shared across all transports)
	deviceService := services.NewDeviceService(mqttClient)

	// Create handler manager for HTTP-based handlers
	handlerManager := handlers.NewHandlerManager(mqttClient)

	// Store non-HTTP handlers for lifecycle management
	var mqttHandler interface{}
	var socketIOHandler interface{}

	// Register transport handlers based on configuration
	transportCount := 0

	// 1. HTTP Transport (replaces ChirpStack-specific handler)
	if cfg.Protocols.HTTP.Enabled {
		httpHandler := http.NewHandler(deviceService, http.Config{
			Path: cfg.Protocols.HTTP.Path,
		})
		handlerManager.Register(httpHandler)
		logger.Infof("Registered HTTP transport handler at path: %s", cfg.Protocols.HTTP.Path)
		transportCount++
	}

	// Backward compatibility: ChirpStack as HTTP handler
	if cfg.Protocols.ChirpStack.Enabled {
		chirpHandler := chirpstack.NewHandler(mqttClient, chirpstack.Config{
			Path: cfg.Protocols.ChirpStack.Path,
		})
		handlerManager.Register(chirpHandler)
		logger.Infof("Registered ChirpStack handler at path: %s (backward compatibility)", cfg.Protocols.ChirpStack.Path)
		transportCount++
	}

	// 2. WebSocket Transport
	if cfg.Protocols.WebSocket.Enabled {
		wsHandler := websocket.NewHandler(deviceService, websocket.Config{
			Path:               cfg.Protocols.WebSocket.Path,
			Port:               cfg.Protocols.WebSocket.Port,
			ReadBufferSize:     1024,
			WriteBufferSize:    1024,
			CheckOrigin:        false,
			HandshakeTimeout:   10,
			MaxMessageSize:     512 * 1024, // 512KB
			PingPeriod:         54,
			PongWait:           60,
			WriteWait:          10,
		})
		handlerManager.Register(wsHandler)
		logger.Infof("Registered WebSocket transport handler at path: %s", cfg.Protocols.WebSocket.Path)
		transportCount++
	}

	// 3. MQTT Subscriber Transport (non-HTTP)
	if cfg.Protocols.MQTT.Enabled {
		mqttConfig := mqttprotocol.Config{
			Broker:          cfg.MQTT.Broker,
			Port:            cfg.MQTT.Port,
			ClientID:        cfg.MQTT.ClientID + "-subscriber",
			Username:        cfg.MQTT.Username,
			Password:        cfg.MQTT.Password,
			SubscribeTopics: []string{"devices/+/data", "sensors/+/telemetry", "iot/+/messages"},
			QOS:             1,
		}
		
		mqttTransportHandler := mqttprotocol.NewHandler(deviceService, mqttConfig)
		mqttHandler = mqttTransportHandler
		
		// Start MQTT subscriber
		if startable, ok := mqttHandler.(interface{ Start() error }); ok {
			if err := startable.Start(); err != nil {
				logger.Errorf("Failed to start MQTT subscriber: %v", err)
			} else {
				logger.Infof("Started MQTT subscriber transport (topics: %v)", mqttConfig.SubscribeTopics)
				transportCount++
			}
		}
		
		// Add health check endpoint for MQTT
		handlerManager.Register(mqttTransportHandler.(handlers.ProtocolHandler))
	}

	// 4. SocketIO Transport (non-HTTP initially, but needs HTTP for upgrade)
	if cfg.Protocols.WebSocket.Enabled { // Reuse WebSocket config for SocketIO
		sioHandler := socketio.NewHandler(deviceService, socketio.Config{
			Path:           "/socket.io/",
			Port:           cfg.Protocols.WebSocket.Port,
			Namespace:      "/",
			PingTimeout:    20,
			PingInterval:   25,
			MaxConnections: 1000,
			AllowOrigins:   "*",
			AllowMethods:   "GET,POST",
			MaxMessageSize: 1024 * 1024, // 1MB
		})
		
		socketIOHandler = sioHandler
		
		// Start SocketIO server
		if startable, ok := socketIOHandler.(interface{ Start() error }); ok {
			if err := startable.Start(); err != nil {
				logger.Errorf("Failed to start SocketIO server: %v", err)
			} else {
				logger.Infof("Started SocketIO transport at path: /socket.io/")
				transportCount++
			}
		}
		
		// Register SocketIO with handler manager for HTTP routes
		handlerManager.Register(sioHandler.(handlers.ProtocolHandler))
	}

	// Create Echo instance
	e := echo.New()
	e.HideBanner = true

	// Middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORS())

	// Setup routes for all registered HTTP-based handlers
	httpHandlerCount := 0
	for name, handler := range handlerManager.GetHandlers() {
		if handler.Path() != "" && handler.Method() != "" {
			e.Add(handler.Method(), handler.Path(), handler.Handle)
			logger.Infof("Registered %s transport at: %s [%s]", name, handler.Path(), handler.Method())
			httpHandlerCount++
		}
		
		// Health check for each handler
		e.GET(fmt.Sprintf("/health/%s", name), handler.HealthCheck)
	}

	// Global health endpoint
	e.GET("/health", func(c echo.Context) error {
		healthData := map[string]interface{}{
			"status":       "healthy",
			"service":      "mpa-service",
			"version":      "v2.0.0",
			"transports":   transportCount,
			"http_handlers": httpHandlerCount,
			"message":      "Multi-Protocol Agent is running",
			"mqtt_broker":  fmt.Sprintf("%s:%d", cfg.MQTT.Broker, cfg.MQTT.Port),
			"mqtt_topic":   cfg.MQTT.Topic,
			"timestamp":    time.Now().UTC().Format(time.RFC3339),
		}
		
		// Add device service health
		deviceHealth := deviceService.GetHealthStatus()
		healthData["device_profiles"] = deviceHealth["device_profiles"]
		healthData["parsers"] = deviceHealth["parsers"]
		healthData["mqtt_connected"] = deviceHealth["mqtt_connected"]
		
		return c.JSON(200, healthData)
	})

	// Device profiles management endpoint
	e.GET("/device-profiles", func(c echo.Context) error {
		profiles := deviceService.GetDeviceProfiles()
		result := make(map[string]interface{})
		
		for id, profile := range profiles {
			result[id] = map[string]interface{}{
				"make":        profile.Make,
				"model":       profile.Model,
				"version":     profile.Version,
				"description": profile.Description,
				"detection":   profile.Detection.Method,
			}
		}
		
		return c.JSON(200, map[string]interface{}{
			"device_profiles": result,
			"count":          len(profiles),
		})
	})

	// Configure server
	e.Server.Addr = fmt.Sprintf(":%d", cfg.Server.Port)
	e.Server.ReadTimeout = cfg.ReadTimeout()
	e.Server.WriteTimeout = cfg.WriteTimeout()
	e.Server.IdleTimeout = cfg.IdleTimeout()

	// Start HTTP server in a goroutine
	go func() {
		logger.Infof("🚀 Starting Multi-Protocol Agent (MPA) service")
		logger.Infof("📡 HTTP server on port %d", cfg.Server.Port)
		logger.Infof("🔌 Registered %d transport handlers (%d HTTP-based)", transportCount, httpHandlerCount)
		logger.Infof("📨 MQTT output broker: %s:%d", cfg.MQTT.Broker, cfg.MQTT.Port)
		logger.Infof("📋 MQTT output topic: %s", cfg.MQTT.Topic)
		logger.Infof("🔍 Device profiles: %d", len(deviceService.GetDeviceProfiles()))
		
		if err := e.Start(e.Server.Addr); err != nil {
			logger.Errorf("HTTP server error: %v", err)
			os.Exit(1)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("🛑 Shutting down Multi-Protocol Agent service...")
	
	// Stop non-HTTP transports first
	if mqttHandler != nil {
		if stoppable, ok := mqttHandler.(interface{ Stop() error }); ok {
			if err := stoppable.Stop(); err != nil {
				logger.Errorf("Error stopping MQTT subscriber: %v", err)
			} else {
				logger.Info("✅ MQTT subscriber stopped")
			}
		}
	}
	
	if socketIOHandler != nil {
		if stoppable, ok := socketIOHandler.(interface{ Stop() error }); ok {
			if err := stoppable.Stop(); err != nil {
				logger.Errorf("Error stopping SocketIO server: %v", err)
			} else {
				logger.Info("✅ SocketIO server stopped")
			}
		}
	}
	
	// Shutdown HTTP server
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	
	if err := e.Shutdown(ctx); err != nil {
		logger.Errorf("HTTP server forced to shutdown: %v", err)
		os.Exit(1)
	}
	
	logger.Info("✅ Multi-Protocol Agent service exited gracefully")
}
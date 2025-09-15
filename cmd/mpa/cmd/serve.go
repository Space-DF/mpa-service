package cmd

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Space-DF/mpa-service/internal/config"
	"github.com/Space-DF/mpa-service/internal/protocols/handlers"
	"github.com/Space-DF/mpa-service/internal/protocols/transport/http"
	mqttprotocol "github.com/Space-DF/mpa-service/internal/protocols/transport/mqtt"
	"github.com/Space-DF/mpa-service/internal/protocols/transport/socketio"
	"github.com/Space-DF/mpa-service/internal/protocols/transport/websocket"
	"github.com/Space-DF/mpa-service/internal/protocols/lorawan"
	"github.com/Space-DF/mpa-service/internal/logger"
	"github.com/Space-DF/mpa-service/internal/mqtt"
	"github.com/Space-DF/mpa-service/internal/services"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/spf13/cobra"
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

The service forwards raw messages to the configured MQTT broker while
preserving the original message content. Supports multiple device types
and custom protocols.`,
	Run: runServe,
}

func init() {
	rootCmd.AddCommand(serveCmd)

	serveCmd.Flags().IntVarP(&port, "port", "p", 0, "HTTP server port (overrides config)")
	serveCmd.Flags().StringVarP(&logLevel, "log-level", "l", "", "Log level (debug, info, warn, error)")
}

func runServe(cmd *cobra.Command, args []string) {
	fmt.Println("Starting MPA Service...")
	cfg, err := config.New()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}
	fmt.Println("Configuration loaded")

	// Override config with CLI flags if provided
	if port != 0 {
		cfg.Server.Port = port
	}
	if logLevel != "" {
		cfg.Server.LogLevel = logLevel
	}

	logger := logger.New(cfg.Server.LogLevel)
	fmt.Println("Logger initialized")
	
	// Initialize MQTT client for output (optional for HTTP-only mode)
	var mqttClient mqtt.ClientInterface
	fmt.Printf("MQTT broker config: %s\n", cfg.MQTT.Broker)
	if cfg.MQTT.Broker != "" {
		fmt.Println("Attempting MQTT connection...")
		mqttClientImpl := mqtt.NewClient(cfg.MQTT)
		if err := mqttClientImpl.Connect(); err != nil {
			fmt.Printf("MQTT connection failed: %v (continuing without MQTT)\n", err)
			mqttClient = nil
		} else {
			fmt.Println("MQTT connected successfully")
			mqttClient = mqttClientImpl
			defer mqttClientImpl.Disconnect()
		}
	} else {
		fmt.Println("No MQTT broker configured, skipping")
	}

	// Initialize device service (shared across all transports)
	deviceService := services.NewDeviceService(mqttClient)

	// Create handler manager for HTTP-based handlers
	handlerManager := handlers.NewHandlerManager(mqttClient)

	// Store non-HTTP handlers for lifecycle management
	var mqttHandler interface{}
	var socketIOHandler interface{}

	// Register transport handlers based on configuration
	transportCount := 0

	// 1. HTTP Transport (generic HTTP handler)
	if cfg.Protocols.HTTP.Enabled {
		httpHandler := http.NewHandler(deviceService, http.Config{
			Path: cfg.Protocols.HTTP.Path,
		}, logger)
		handlerManager.Register(httpHandler)
		logger.Infof("Registered HTTP transport handler at path: %s", cfg.Protocols.HTTP.Path)
		transportCount++
	}

	// 2. ChirpStack HTTP Transport
	if cfg.Protocols.ChirpStack.Enabled {
		chirpstackFactory := lorawan.NewChirpStackFactory(deviceService, logger)
		chirpstackHandler := chirpstackFactory.CreateHandler()
		handlerManager.Register(chirpstackHandler)
		logger.Infof("Registered ChirpStack transport handler at path: /lorawan/chirpstack/http")
		transportCount++
	}

	// 3. TTN HTTP Transport
	if cfg.Protocols.TTN.Enabled {
		ttnFactory := lorawan.NewTTNFactory(deviceService, logger)
		ttnHandler := ttnFactory.CreateHandler()
		handlerManager.Register(ttnHandler)
		logger.Infof("Registered TTN transport handler at path: /lorawan/ttn/http")
		transportCount++
	}

	// 4. Helium HTTP Transport
	if cfg.Protocols.Helium.Enabled {
		heliumFactory := lorawan.NewHeliumFactory(deviceService, logger)
		heliumHandler := heliumFactory.CreateHandler()
		handlerManager.Register(heliumHandler)
		logger.Infof("Registered Helium transport handler at path: /lorawan/helium/http")
		transportCount++
	}


	// 5. WebSocket Transport
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

	// 6. MQTT Subscriber Transport (non-HTTP)
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
		handlerManager.Register(mqttTransportHandler)
	}

	// 7. SocketIO Transport (non-HTTP initially, but needs HTTP for upgrade)
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
		handlerManager.Register(sioHandler)
	}

	// Create Echo instance
	e := echo.New()
	e.HideBanner = true

	// Security and operational middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORS())
	
	// Request timeout middleware (prevent slow loris attacks)
	e.Use(middleware.TimeoutWithConfig(middleware.TimeoutConfig{
		Timeout: 30 * time.Second,
	}))
	
	// Body limit middleware (global fallback, handlers have their own limits)
	e.Use(middleware.BodyLimit("2M"))
	
	// Rate limiting middleware (basic protection)
	e.Use(middleware.RateLimiter(middleware.NewRateLimiterMemoryStore(100)))

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
		healthData["parsers"] = deviceHealth["parsers"] 
		healthData["mqtt_connected"] = deviceHealth["mqtt_connected"]
		
		return c.JSON(200, healthData)
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

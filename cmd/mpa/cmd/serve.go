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
	"github.com/Space-DF/mpa-service/internal/logger"
	"github.com/Space-DF/mpa-service/internal/mqtt"
)

var (
	port    int
	logLevel string
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the HTTP server",
	Long: `Start the HTTP server that listens for ChirpStack webhooks
and forwards data to the configured MQTT broker.

The server will listen on the configured port and endpoint,
handling incoming webhook payloads from ChirpStack and
publishing them to the MQTT broker.`,
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
		cfg.HTTP.Port = port
	}
	if logLevel != "" {
		cfg.Server.LogLevel = logLevel
	}

	logger := logger.New(cfg.Server.LogLevel)
	
	// Initialize MQTT client
	mqttClient := mqtt.NewClient(cfg.MQTT)
	if err := mqttClient.Connect(); err != nil {
		logger.Errorf("Failed to connect to MQTT broker: %v", err)
		os.Exit(1)
	}
	defer mqttClient.Disconnect()

	// Initialize HTTP handlers
	chirpStackHandler := handlers.NewChirpStackHandler(mqttClient)

	// Create handler manager
	handlerManager := handlers.NewHandlerManager(mqttClient)

	// Register protocol handlers
	if cfg.Protocols.ChirpStack.Enabled {
		chirpHandler := chirpstack.NewHandler(mqttClient, chirpstack.Config{
			Path: cfg.Protocols.ChirpStack.Path,
		})
		handlerManager.Register(chirpHandler)
		logger.Infof("Registered ChirpStack handler at path: %s", cfg.Protocols.ChirpStack.Path)
	}

	// Create Echo instance
	e := echo.New()

	// Middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORS())

	// Setup routes for all registered handlers
	for name, handler := range handlerManager.GetHandlers() {
		e.Add(handler.Method(), handler.Path(), handler.Handle)
		e.GET(fmt.Sprintf("/health/%s", name), handler.HealthCheck)
		logger.Infof("Registered %s handler at: %s [%s]", name, handler.Path(), handler.Method())
	}

	// Global health endpoint
	e.GET("/health", func(c echo.Context) error {
		return c.JSON(200, map[string]interface{}{
			"status":    "healthy",
			"service":   "mpa-service",
			"protocols": len(handlerManager.GetHandlers()),
			"message":   "Multi-Protocol Agent is running",
		})
	})

	// Configure server
	e.Server.Addr = fmt.Sprintf(":%d", cfg.Server.Port)
	e.Server.ReadTimeout = cfg.ReadTimeout()
	e.Server.WriteTimeout = cfg.WriteTimeout()
	e.Server.IdleTimeout = cfg.IdleTimeout()

	// Start server in a goroutine
	go func() {
		logger.Infof("Starting MPA service on port %d", cfg.Server.Port)
		logger.Infof("Registered %d protocol handlers", len(handlerManager.GetHandlers()))
		logger.Infof("MQTT broker: %s:%d", cfg.MQTT.Broker, cfg.MQTT.Port)
		logger.Infof("MQTT topic: %s", cfg.MQTT.Topic)
		
		if err := e.Start(e.Server.Addr); err != nil {
			logger.Errorf("MPA service error: %v", err)
			os.Exit(1)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down MPA service...")
	
	// Create a context with timeout for graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	
	if err := e.Shutdown(ctx); err != nil {
		logger.Errorf("MPA service forced to shutdown: %v", err)
		os.Exit(1)
	}
	
	logger.Info("MPA service exited")
}
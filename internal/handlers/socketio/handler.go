package socketio

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	socketio "github.com/googollee/go-socket.io"
	"github.com/labstack/echo/v4"
	"github.com/Space-DF/mpa-service/internal/handlers"
	"github.com/Space-DF/mpa-service/internal/services"
)

// Handler implements SocketIO transport handler for multi-device support
type Handler struct {
	deviceService *services.DeviceService
	config        Config
	server        *socketio.Server
	isRunning     bool
	mutex         sync.RWMutex
	connections   map[string]*Connection
	messageCount  int64
}

// Config holds SocketIO transport handler configuration
type Config struct {
	Path            string `yaml:"path"`
	Port            int    `yaml:"port"`
	Namespace       string `yaml:"namespace"`
	PingTimeout     int    `yaml:"ping_timeout"`
	PingInterval    int    `yaml:"ping_interval"`
	MaxConnections  int    `yaml:"max_connections"`
	AllowOrigins    string `yaml:"allow_origins"`
	AllowMethods    string `yaml:"allow_methods"`
	MaxMessageSize  int64  `yaml:"max_message_size"`
}

// Connection represents a SocketIO connection
type Connection struct {
	ID           string
	SessionID    string
	ConnectedAt  time.Time
	LastMessage  time.Time
	MessageCount int64
	RemoteAddr   string
	UserAgent    string
	Namespace    string
}

// NewHandler creates a new SocketIO transport handler
func NewHandler(deviceService *services.DeviceService, config Config) handlers.ProtocolHandler {
	handler := &Handler{
		deviceService: deviceService,
		config:        config,
		connections:   make(map[string]*Connection),
		messageCount:  0,
		isRunning:     false,
	}
	
	return handler
}

// Name returns the transport protocol name
func (h *Handler) Name() string {
	return "socketio"
}

// Path returns the SocketIO endpoint path
func (h *Handler) Path() string {
	return h.config.Path
}

// Method returns the HTTP method for SocketIO
func (h *Handler) Method() string {
	return "GET" // SocketIO starts with GET for upgrade
}

// Handle serves the SocketIO endpoint
func (h *Handler) Handle(c echo.Context) error {
	if h.server == nil {
		return echo.NewHTTPError(503, "SocketIO server not initialized")
	}
	
	// SocketIO server handles the HTTP request
	h.server.ServeHTTP(c.Response(), c.Request())
	return nil
}

// Start initializes and starts the SocketIO server
func (h *Handler) Start() error {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	
	if h.isRunning {
		return fmt.Errorf("SocketIO server already running")
	}
	
	// Create SocketIO server
	server := socketio.NewServer(nil)
	
	h.server = server
	
	// Set up connection handlers
	h.setupHandlers()
	
	h.isRunning = true
	log.Printf("SocketIO: Server started on path %s", h.config.Path)
	
	return nil
}

// Stop gracefully stops the SocketIO server
func (h *Handler) Stop() error {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	
	if !h.isRunning {
		return fmt.Errorf("SocketIO server not running")
	}
	
	if h.server != nil {
		// Close all connections
		for _, conn := range h.connections {
			log.Printf("SocketIO: Closing connection: %s", conn.ID)
		}
		
		h.server.Close()
		h.server = nil
	}
	
	h.connections = make(map[string]*Connection)
	h.isRunning = false
	log.Printf("SocketIO: Server stopped")
	
	return nil
}

// setupHandlers configures SocketIO event handlers
func (h *Handler) setupHandlers() {
	namespace := h.config.Namespace
	if namespace == "" {
		namespace = "/"
	}
	
	// Connection handler
	h.server.OnConnect(namespace, func(so socketio.Conn) error {
		connectionID := fmt.Sprintf("sio_%d_%s", time.Now().UnixNano(), so.RemoteAddr().String())
		
		connection := &Connection{
			ID:          connectionID,
			SessionID:   so.ID(),
			ConnectedAt: time.Now(),
			RemoteAddr:  so.RemoteAddr().String(),
			Namespace:   namespace,
		}
		
		h.mutex.Lock()
		h.connections[connectionID] = connection
		h.mutex.Unlock()
		
		// Store connection ID in context for later use
		so.SetContext(connectionID)
		
		log.Printf("SocketIO: New connection established: %s (session: %s) from %s", 
			connectionID, so.ID(), so.RemoteAddr().String())
		
		// Join a room for broadcasting
		so.Join("devices")
		
		// Send welcome message
		so.Emit("connected", map[string]interface{}{
			"status":        "connected",
			"connection_id": connectionID,
			"session_id":    so.ID(),
			"timestamp":     time.Now().UTC().Format(time.RFC3339),
		})
		
		return nil
	})
	
	// Disconnection handler
	h.server.OnDisconnect(namespace, func(so socketio.Conn, reason string) {
		connectionID := so.Context().(string)
		
		h.mutex.Lock()
		delete(h.connections, connectionID)
		h.mutex.Unlock()
		
		log.Printf("SocketIO: Connection disconnected: %s (reason: %s)", connectionID, reason)
	})
	
	// Message handler for device data
	h.server.OnEvent(namespace, "device_data", func(so socketio.Conn, data interface{}) {
		connectionID := so.Context().(string)
		
		h.mutex.Lock()
		if conn, exists := h.connections[connectionID]; exists {
			conn.LastMessage = time.Now()
			conn.MessageCount++
		}
		h.messageCount++
		h.mutex.Unlock()
		
		log.Printf("SocketIO: Received device_data from %s", connectionID)
		
		// Process the message
		h.processDeviceData(so, connectionID, data)
	})
	
	// Generic message handler
	h.server.OnEvent(namespace, "message", func(so socketio.Conn, data interface{}) {
		connectionID := so.Context().(string)
		
		h.mutex.Lock()
		if conn, exists := h.connections[connectionID]; exists {
			conn.LastMessage = time.Now()
			conn.MessageCount++
		}
		h.messageCount++
		h.mutex.Unlock()
		
		log.Printf("SocketIO: Received message from %s", connectionID)
		
		// Process the message
		h.processMessage(so, connectionID, data)
	})
	
	// Error handler
	h.server.OnError(namespace, func(so socketio.Conn, err error) {
		connectionID := so.Context().(string)
		log.Printf("SocketIO: Error on connection %s: %v", connectionID, err)
	})
}

// processDeviceData processes device data messages
func (h *Handler) processDeviceData(so socketio.Conn, connectionID string, data interface{}) {
	// Convert data to bytes for processing
	var message []byte
	var err error
	
	switch v := data.(type) {
	case string:
		message = []byte(v)
	case []byte:
		message = v
	case map[string]interface{}:
		// For JSON objects, marshal to bytes
		message, err = json.Marshal(v)
		if err != nil {
			log.Printf("SocketIO: Error marshaling data from %s: %v", connectionID, err)
			h.sendError(so, "Invalid data format")
			return
		}
	default:
		log.Printf("SocketIO: Unsupported data type from %s: %T", connectionID, data)
		h.sendError(so, "Unsupported data type")
		return
	}
	
	// Prepare transport metadata
	transportMetadata := h.getTransportMetadata(connectionID, "device_data")
	
	// Process message through device service
	if err := h.deviceService.ProcessWebSocketMessage(message, transportMetadata); err != nil {
		log.Printf("SocketIO: Error processing device_data from %s: %v", connectionID, err)
		h.sendError(so, fmt.Sprintf("Processing failed: %v", err))
		return
	}
	
	log.Printf("SocketIO: Successfully processed device_data from %s", connectionID)
	
	// Send success response
	so.Emit("device_data_response", map[string]interface{}{
		"status":    "success",
		"message":   "Device data processed successfully",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

// processMessage processes generic messages
func (h *Handler) processMessage(so socketio.Conn, connectionID string, data interface{}) {
	// Similar to processDeviceData but for generic messages
	var message []byte
	var err error
	
	switch v := data.(type) {
	case string:
		message = []byte(v)
	case []byte:
		message = v
	case map[string]interface{}:
		message, err = json.Marshal(v)
		if err != nil {
			log.Printf("SocketIO: Error marshaling message from %s: %v", connectionID, err)
			h.sendError(so, "Invalid message format")
			return
		}
	default:
		log.Printf("SocketIO: Unsupported message type from %s: %T", connectionID, data)
		h.sendError(so, "Unsupported message type")
		return
	}
	
	// Prepare transport metadata
	transportMetadata := h.getTransportMetadata(connectionID, "message")
	
	// Process message through device service
	if err := h.deviceService.ProcessWebSocketMessage(message, transportMetadata); err != nil {
		log.Printf("SocketIO: Error processing message from %s: %v", connectionID, err)
		h.sendError(so, fmt.Sprintf("Processing failed: %v", err))
		return
	}
	
	log.Printf("SocketIO: Successfully processed message from %s", connectionID)
	
	// Send success response
	so.Emit("message_response", map[string]interface{}{
		"status":    "success",
		"message":   "Message processed successfully",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

// getTransportMetadata creates transport metadata for message processing
func (h *Handler) getTransportMetadata(connectionID, eventType string) map[string]interface{} {
	h.mutex.RLock()
	conn, exists := h.connections[connectionID]
	h.mutex.RUnlock()
	
	metadata := map[string]interface{}{
		"transport":      "socketio",
		"connection_id":  connectionID,
		"event_type":     eventType,
		"received_at":    time.Now().UTC().Format(time.RFC3339),
	}
	
	if exists {
		metadata["session_id"] = conn.SessionID
		metadata["remote_addr"] = conn.RemoteAddr
		metadata["connected_at"] = conn.ConnectedAt.UTC().Format(time.RFC3339)
		metadata["message_count"] = conn.MessageCount
		metadata["namespace"] = conn.Namespace
	}
	
	return metadata
}

// sendError sends an error response to the client
func (h *Handler) sendError(so socketio.Conn, message string) {
	so.Emit("error", map[string]interface{}{
		"status":    "error",
		"message":   message,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

// HealthCheck returns health status for SocketIO transport handler
func (h *Handler) HealthCheck(c echo.Context) error {
	h.mutex.RLock()
	connectionCount := len(h.connections)
	messageCount := h.messageCount
	isRunning := h.isRunning
	h.mutex.RUnlock()
	
	healthStatus := h.deviceService.GetHealthStatus()
	
	status := "healthy"
	if !isRunning {
		status = "not_running"
	}
	
	return c.JSON(200, map[string]interface{}{
		"transport":         "socketio",
		"status":            status,
		"message":           "SocketIO transport handler status",
		"server_running":    isRunning,
		"mqtt_connected":    healthStatus["mqtt_connected"],
		"parsers":           healthStatus["parsers"],
		"endpoint":          h.config.Path,
		"namespace":         h.config.Namespace,
		"active_connections": connectionCount,
		"total_messages":    messageCount,
		"max_connections":   h.config.MaxConnections,
		"timestamp":         time.Now().UTC().Format(time.RFC3339),
	})
}

// GetConnections returns information about active connections
func (h *Handler) GetConnections() map[string]interface{} {
	h.mutex.RLock()
	defer h.mutex.RUnlock()
	
	connections := make(map[string]interface{})
	for id, conn := range h.connections {
		connections[id] = map[string]interface{}{
			"session_id":     conn.SessionID,
			"remote_addr":    conn.RemoteAddr,
			"connected_at":   conn.ConnectedAt.UTC().Format(time.RFC3339),
			"last_message":   conn.LastMessage.UTC().Format(time.RFC3339),
			"message_count":  conn.MessageCount,
			"namespace":      conn.Namespace,
		}
	}
	
	return connections
}

// BroadcastToRoom sends a message to all clients in a specific room
func (h *Handler) BroadcastToRoom(room string, event string, data interface{}) error {
	if h.server == nil {
		return fmt.Errorf("SocketIO server not initialized")
	}
	
	namespace := h.config.Namespace
	if namespace == "" {
		namespace = "/"
	}
	
	h.server.BroadcastToRoom(namespace, room, event, data)
	log.Printf("SocketIO: Broadcasted %s event to room %s", event, room)
	
	return nil
}

// IsRunning returns whether the SocketIO server is running
func (h *Handler) IsRunning() bool {
	h.mutex.RLock()
	defer h.mutex.RUnlock()
	return h.isRunning
}
package websocket

import (
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
	"github.com/Space-DF/mpa-service/internal/handlers"
	"github.com/Space-DF/mpa-service/internal/services"
)

// Handler implements WebSocket transport handler for multi-device support
type Handler struct {
	deviceService *services.DeviceService
	config        Config
	upgrader      websocket.Upgrader
	connections   map[string]*Connection
	mutex         sync.RWMutex
	messageCount  int64
}

// Config holds WebSocket transport handler configuration
type Config struct {
	Path               string `yaml:"path"`
	Port               int    `yaml:"port"`
	ReadBufferSize     int    `yaml:"read_buffer_size"`
	WriteBufferSize    int    `yaml:"write_buffer_size"`
	CheckOrigin        bool   `yaml:"check_origin"`
	HandshakeTimeout   int    `yaml:"handshake_timeout"`
	MaxMessageSize     int64  `yaml:"max_message_size"`
	PingPeriod         int    `yaml:"ping_period"`
	PongWait           int    `yaml:"pong_wait"`
	WriteWait          int    `yaml:"write_wait"`
}

// Connection represents a WebSocket connection
type Connection struct {
	ID         string
	Conn       *websocket.Conn
	ConnectedAt time.Time
	LastMessage time.Time
	MessageCount int64
	RemoteAddr  string
	UserAgent   string
	Headers     http.Header
}

// NewHandler creates a new WebSocket transport handler
func NewHandler(deviceService *services.DeviceService, config Config) handlers.ProtocolHandler {
	handler := &Handler{
		deviceService: deviceService,
		config:        config,
		connections:   make(map[string]*Connection),
		messageCount:  0,
	}
	
	// Configure WebSocket upgrader
	handler.upgrader = websocket.Upgrader{
		ReadBufferSize:   config.ReadBufferSize,
		WriteBufferSize:  config.WriteBufferSize,
		HandshakeTimeout: time.Duration(config.HandshakeTimeout) * time.Second,
		CheckOrigin: func(r *http.Request) bool {
			if config.CheckOrigin {
				// Add your origin checking logic here
				return true
			}
			return true // Allow all origins for now
		},
	}
	
	return handler
}

// Name returns the transport protocol name
func (h *Handler) Name() string {
	return "websocket"
}

// Path returns the WebSocket endpoint path
func (h *Handler) Path() string {
	return h.config.Path
}

// Method returns the HTTP method for WebSocket upgrade
func (h *Handler) Method() string {
	return "GET" // WebSocket upgrade uses GET
}

// Handle processes WebSocket upgrade requests
func (h *Handler) Handle(c echo.Context) error {
	// Upgrade HTTP connection to WebSocket
	conn, err := h.upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		log.Printf("WebSocket: Failed to upgrade connection: %v", err)
		return echo.NewHTTPError(400, "Failed to upgrade to WebSocket")
	}
	
	// Create connection tracking
	connectionID := fmt.Sprintf("ws_%d_%s", time.Now().UnixNano(), c.Request().RemoteAddr)
	connection := &Connection{
		ID:          connectionID,
		Conn:        conn,
		ConnectedAt: time.Now(),
		RemoteAddr:  c.Request().RemoteAddr,
		UserAgent:   c.Request().Header.Get("User-Agent"),
		Headers:     c.Request().Header,
	}
	
	h.mutex.Lock()
	h.connections[connectionID] = connection
	h.mutex.Unlock()
	
	log.Printf("WebSocket: New connection established: %s from %s", connectionID, c.Request().RemoteAddr)
	
	// Handle the connection
	go h.handleConnection(connection)
	
	return nil
}

// handleConnection manages a WebSocket connection
func (h *Handler) handleConnection(conn *Connection) {
	defer func() {
		// Remove connection from tracking
		h.mutex.Lock()
		delete(h.connections, conn.ID)
		h.mutex.Unlock()
		
		conn.Conn.Close()
		log.Printf("WebSocket: Connection closed: %s", conn.ID)
	}()
	
	// Set connection limits
	conn.Conn.SetReadLimit(h.config.MaxMessageSize)
	conn.Conn.SetReadDeadline(time.Now().Add(time.Duration(h.config.PongWait) * time.Second))
	conn.Conn.SetPongHandler(func(string) error {
		conn.Conn.SetReadDeadline(time.Now().Add(time.Duration(h.config.PongWait) * time.Second))
		return nil
	})
	
	// Start ping ticker
	ticker := time.NewTicker(time.Duration(h.config.PingPeriod) * time.Second)
	defer ticker.Stop()
	
	// Handle ping in a separate goroutine
	go func() {
		for {
			select {
			case <-ticker.C:
				conn.Conn.SetWriteDeadline(time.Now().Add(time.Duration(h.config.WriteWait) * time.Second))
				if err := conn.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
					log.Printf("WebSocket: Error sending ping to %s: %v", conn.ID, err)
					return
				}
			}
		}
	}()
	
	// Read messages from the connection
	for {
		messageType, message, err := conn.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket: Unexpected close error for %s: %v", conn.ID, err)
			}
			break
		}
		
		if messageType == websocket.TextMessage || messageType == websocket.BinaryMessage {
			conn.LastMessage = time.Now()
			conn.MessageCount++
			
			h.mutex.Lock()
			h.messageCount++
			h.mutex.Unlock()
			
			log.Printf("WebSocket: Received message from %s (type: %d, size: %d bytes)", 
				conn.ID, messageType, len(message))
			
			// Process the message
			h.processMessage(conn, message)
		}
	}
}

// processMessage processes incoming WebSocket messages
func (h *Handler) processMessage(conn *Connection, message []byte) {
	// Prepare transport metadata
	transportMetadata := map[string]interface{}{
		"transport":       "websocket",
		"connection_id":   conn.ID,
		"remote_addr":     conn.RemoteAddr,
		"user_agent":      conn.UserAgent,
		"connected_at":    conn.ConnectedAt.UTC().Format(time.RFC3339),
		"message_count":   conn.MessageCount,
		"received_at":     time.Now().UTC().Format(time.RFC3339),
	}
	
	// Add WebSocket-specific headers
	for name, values := range conn.Headers {
		if len(name) > 3 && (name[:3] == "Sec" || name[:2] == "X-") {
			if len(values) > 0 {
				transportMetadata["header_"+name] = values[0]
			}
		}
	}
	
	// Process message through device service
	if err := h.deviceService.ProcessWebSocketMessage(message, transportMetadata); err != nil {
		log.Printf("WebSocket: Error processing message from %s: %v", conn.ID, err)
		
		// Send error response back to client
		errorResponse := map[string]interface{}{
			"status":    "error",
			"message":   "Message processing failed",
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		}
		
		h.sendResponse(conn, errorResponse)
		return
	}
	
	log.Printf("WebSocket: Successfully processed message from %s", conn.ID)
	
	// Send success response back to client
	successResponse := map[string]interface{}{
		"status":    "success",
		"message":   "Message processed successfully",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}
	
	h.sendResponse(conn, successResponse)
}

// sendResponse sends a response back to the WebSocket client
func (h *Handler) sendResponse(conn *Connection, response interface{}) {
	conn.Conn.SetWriteDeadline(time.Now().Add(time.Duration(h.config.WriteWait) * time.Second))
	if err := conn.Conn.WriteJSON(response); err != nil {
		log.Printf("WebSocket: Error sending response to %s: %v", conn.ID, err)
	}
}

// HealthCheck returns health status for WebSocket transport handler
func (h *Handler) HealthCheck(c echo.Context) error {
	h.mutex.RLock()
	connectionCount := len(h.connections)
	messageCount := h.messageCount
	h.mutex.RUnlock()
	
	healthStatus := h.deviceService.GetHealthStatus()
	
	return c.JSON(200, map[string]interface{}{
		"transport":         "websocket",
		"status":            "healthy",
		"message":           "WebSocket transport handler is running",
		"mqtt_connected":    healthStatus["mqtt_connected"],
		"device_profiles":   healthStatus["device_profiles"],
		"parsers":           healthStatus["parsers"],
		"endpoint":          h.config.Path,
		"active_connections": connectionCount,
		"total_messages":    messageCount,
		"max_message_size":  h.config.MaxMessageSize,
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
			"remote_addr":    conn.RemoteAddr,
			"connected_at":   conn.ConnectedAt.UTC().Format(time.RFC3339),
			"last_message":   conn.LastMessage.UTC().Format(time.RFC3339),
			"message_count":  conn.MessageCount,
			"user_agent":     conn.UserAgent,
		}
	}
	
	return connections
}

// GetConnectionCount returns the number of active connections
func (h *Handler) GetConnectionCount() int {
	h.mutex.RLock()
	defer h.mutex.RUnlock()
	return len(h.connections)
}

// GetMessageCount returns the total number of messages received
func (h *Handler) GetMessageCount() int64 {
	h.mutex.RLock()
	defer h.mutex.RUnlock()
	return h.messageCount
}

// BroadcastMessage sends a message to all connected clients
func (h *Handler) BroadcastMessage(message interface{}) error {
	h.mutex.RLock()
	connections := make([]*Connection, 0, len(h.connections))
	for _, conn := range h.connections {
		connections = append(connections, conn)
	}
	h.mutex.RUnlock()
	
	for _, conn := range connections {
		go h.sendResponse(conn, message)
	}
	
	return nil
}

// CloseConnection closes a specific connection
func (h *Handler) CloseConnection(connectionID string) error {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	
	if conn, exists := h.connections[connectionID]; exists {
		conn.Conn.Close()
		delete(h.connections, connectionID)
		log.Printf("WebSocket: Manually closed connection: %s", connectionID)
		return nil
	}
	
	return fmt.Errorf("connection %s not found", connectionID)
}
package handlers

// ProtocolHandlerFactory defines interface to create ProtocolHandlers
type ProtocolHandlerFactory interface {
    // Name returns the protocol name
    Name() string

    // CreateHandler creates a ProtocolHandler instance based on config
    CreateHandler() ProtocolHandler
}

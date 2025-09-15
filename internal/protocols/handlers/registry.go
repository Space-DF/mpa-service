package handlers

import (
	"fmt"
	"sync"
)

// FactoryConstructor defines a function type that creates a ProtocolHandlerFactory
type FactoryConstructor func() ProtocolHandlerFactory

// HandlerRegistry manages dynamic registration of protocol handler factories
type HandlerRegistry struct {
	mu           sync.RWMutex
	constructors map[string]FactoryConstructor
}

// NewHandlerRegistry creates a new handler registry
func NewHandlerRegistry() *HandlerRegistry {
	return &HandlerRegistry{
		constructors: make(map[string]FactoryConstructor),
	}
}

// Register registers a factory constructor for a protocol
func (r *HandlerRegistry) Register(protocolName string, constructor FactoryConstructor) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.constructors[protocolName]; exists {
		return fmt.Errorf("protocol '%s' is already registered", protocolName)
	}

	r.constructors[protocolName] = constructor
	return nil
}

// CreateFactory creates a factory for the specified protocol
func (r *HandlerRegistry) CreateFactory(protocolName string) (ProtocolHandlerFactory, error) {
	r.mu.RLock()
	constructor, exists := r.constructors[protocolName]
	r.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("protocol '%s' not found in registry", protocolName)
	}

	return constructor(), nil
}

// GetRegisteredProtocols returns a list of all registered protocol names
func (r *HandlerRegistry) GetRegisteredProtocols() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	protocols := make([]string, 0, len(r.constructors))
	for name := range r.constructors {
		protocols = append(protocols, name)
	}
	return protocols
}

// IsRegistered checks if a protocol is registered
func (r *HandlerRegistry) IsRegistered(protocolName string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	_, exists := r.constructors[protocolName]
	return exists
}

// Unregister removes a protocol from the registry
func (r *HandlerRegistry) Unregister(protocolName string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.constructors[protocolName]; !exists {
		return fmt.Errorf("protocol '%s' not found in registry", protocolName)
	}

	delete(r.constructors, protocolName)
	return nil
}

// DefaultRegistry is a global registry instance
var DefaultRegistry = NewHandlerRegistry()

// Register is a convenience function to register with the default registry
func Register(protocolName string, constructor FactoryConstructor) error {
	return DefaultRegistry.Register(protocolName, constructor)
}

// CreateFactory is a convenience function to create factory from the default registry
func CreateFactory(protocolName string) (ProtocolHandlerFactory, error) {
	return DefaultRegistry.CreateFactory(protocolName)
}

// GetRegisteredProtocols is a convenience function to get protocols from the default registry
func GetRegisteredProtocols() []string {
	return DefaultRegistry.GetRegisteredProtocols()
}
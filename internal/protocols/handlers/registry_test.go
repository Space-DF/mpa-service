package handlers

import (
	"testing"

	"github.com/labstack/echo/v4"
)

type mockFactory struct {
	name string
}

func (f *mockFactory) Name() string {
	return f.name
}

func (f *mockFactory) CreateHandler() ProtocolHandler {
	return &mockHandler{protocol: f.name}
}

type mockHandler struct {
	protocol string
}

func (h *mockHandler) Name() string {
	return h.protocol
}

func (h *mockHandler) Path() string {
	return "/" + h.protocol
}

func (h *mockHandler) Method() string {
	return "POST"
}

func (h *mockHandler) Handle(c echo.Context) error {
	return nil
}

func (h *mockHandler) HealthCheck(c echo.Context) error {
	return nil
}

func TestHandlerRegistry_Register(t *testing.T) {
	registry := NewHandlerRegistry()

	err := registry.Register("test", func() ProtocolHandlerFactory {
		return &mockFactory{name: "test"}
	})

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if !registry.IsRegistered("test") {
		t.Error("Expected protocol to be registered")
	}
}

func TestHandlerRegistry_RegisterDuplicate(t *testing.T) {
	registry := NewHandlerRegistry()

	err := registry.Register("test", func() ProtocolHandlerFactory {
		return &mockFactory{name: "test"}
	})
	if err != nil {
		t.Fatalf("Expected no error on first registration, got %v", err)
	}

	err = registry.Register("test", func() ProtocolHandlerFactory {
		return &mockFactory{name: "test2"}
	})

	if err == nil {
		t.Error("Expected error when registering duplicate protocol")
	}
}

func TestHandlerRegistry_CreateFactory(t *testing.T) {
	registry := NewHandlerRegistry()

	err := registry.Register("test", func() ProtocolHandlerFactory {
		return &mockFactory{name: "test"}
	})
	if err != nil {
		t.Fatalf("Expected no error on registration, got %v", err)
	}

	factory, err := registry.CreateFactory("test")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if factory.Name() != "test" {
		t.Errorf("Expected factory name 'test', got %s", factory.Name())
	}
}

func TestHandlerRegistry_CreateFactoryNotFound(t *testing.T) {
	registry := NewHandlerRegistry()

	_, err := registry.CreateFactory("nonexistent")
	if err == nil {
		t.Error("Expected error for nonexistent protocol")
	}
}

func TestHandlerRegistry_GetRegisteredProtocols(t *testing.T) {
	registry := NewHandlerRegistry()

	err := registry.Register("protocol1", func() ProtocolHandlerFactory {
		return &mockFactory{name: "protocol1"}
	})
	if err != nil {
		t.Fatalf("Expected no error on first registration, got %v", err)
	}
	err = registry.Register("protocol2", func() ProtocolHandlerFactory {
		return &mockFactory{name: "protocol2"}
	})
	if err != nil {
		t.Fatalf("Expected no error on second registration, got %v", err)
	}

	protocols := registry.GetRegisteredProtocols()
	if len(protocols) != 2 {
		t.Errorf("Expected 2 protocols, got %d", len(protocols))
	}
}

func TestHandlerRegistry_Unregister(t *testing.T) {
	registry := NewHandlerRegistry()

	err := registry.Register("test", func() ProtocolHandlerFactory {
		return &mockFactory{name: "test"}
	})
	if err != nil {
		t.Fatalf("Expected no error on registration, got %v", err)
	}

	err = registry.Unregister("test")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if registry.IsRegistered("test") {
		t.Error("Expected protocol to be unregistered")
	}
}

func TestHandlerRegistry_UnregisterNotFound(t *testing.T) {
	registry := NewHandlerRegistry()

	err := registry.Unregister("nonexistent")
	if err == nil {
		t.Error("Expected error when unregistering nonexistent protocol")
	}
}

func TestDefaultRegistry(t *testing.T) {
	err := Register("test-default", func() ProtocolHandlerFactory {
		return &mockFactory{name: "test-default"}
	})

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	factory, err := CreateFactory("test-default")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if factory.Name() != "test-default" {
		t.Errorf("Expected factory name 'test-default', got %s", factory.Name())
	}

	protocols := GetRegisteredProtocols()
	found := false
	for _, p := range protocols {
		if p == "test-default" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected to find 'test-default' in registered protocols")
	}
}

func TestConcurrentAccess(t *testing.T) {
	registry := NewHandlerRegistry()

	done := make(chan bool)
	go func() {
		for i := 0; i < 100; i++ {
			_ = registry.Register("concurrent1", func() ProtocolHandlerFactory {
				return &mockFactory{name: "concurrent1"}
			})
			_ = registry.Unregister("concurrent1")
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 100; i++ {
			registry.IsRegistered("concurrent1")
			registry.GetRegisteredProtocols()
		}
		done <- true
	}()

	<-done
	<-done
}

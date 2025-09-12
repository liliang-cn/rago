package store

import (
	"testing"
)

func TestNewStoreFactory(t *testing.T) {
	factory := NewStoreFactory()
	if factory == nil {
		t.Fatal("NewStoreFactory returned nil")
	}

	if factory.registeredTypes == nil {
		t.Fatal("registeredTypes map not initialized")
	}

	// Should have default stores registered
	supportedTypes := factory.SupportedTypes()
	if len(supportedTypes) == 0 {
		t.Fatal("No supported store types found")
	}

	// Should have sqvect/sqlite registered
	found := false
	for _, storeType := range supportedTypes {
		if storeType == "sqvect" || storeType == "sqlite" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Default sqvect store type not registered")
	}
}

func TestStoreFactory_Register(t *testing.T) {
	factory := NewStoreFactory()

	// Register a mock store creator
	mockCreator := func(config StoreConfig) (VectorStore, error) {
		return nil, nil
	}

	factory.Register("MockStore", mockCreator)

	// Check if it's registered (case-insensitive)
	supportedTypes := factory.SupportedTypes()
	found := false
	for _, storeType := range supportedTypes {
		if storeType == "mockstore" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Registered store type not found in supported types")
	}
}

func TestStoreFactory_CreateStore_UnsupportedType(t *testing.T) {
	factory := NewStoreFactory()

	config := StoreConfig{
		Type:       "unsupported_type",
		Parameters: map[string]interface{}{},
	}

	store, err := factory.CreateStore(config)
	if err == nil {
		t.Error("Expected error for unsupported store type")
	}
	if store != nil {
		t.Error("Expected nil store for unsupported type")
	}

	if err.Error() != "unsupported store type: unsupported_type" {
		t.Errorf("Unexpected error message: %v", err)
	}
}

func TestStoreFactory_SupportedTypes(t *testing.T) {
	factory := NewStoreFactory()

	types := factory.SupportedTypes()
	if len(types) == 0 {
		t.Error("Expected at least one supported type")
	}

	// All types should be lowercase
	for _, storeType := range types {
		if storeType != string(storeType) {
			t.Errorf("Store type should be lowercase: %s", storeType)
		}
	}
}

func TestNewDefaultStore_InvalidType(t *testing.T) {
	// Test with invalid store type
	store, err := NewDefaultStore("invalid")
	if err == nil {
		t.Error("Expected error for invalid store type")
	}
	if store != nil {
		t.Error("Expected nil store for invalid type")
	}
}

func TestStoreConfig(t *testing.T) {
	config := StoreConfig{
		Type: "test",
		Parameters: map[string]interface{}{
			"key": "value",
		},
	}

	if config.Type != "test" {
		t.Error("Config type not set correctly")
	}

	if config.Parameters["key"] != "value" {
		t.Error("Config parameters not set correctly")
	}
}

func TestDistanceMetric(t *testing.T) {
	metrics := []DistanceMetric{
		DistanceCosine,
		DistanceEuclidean,
		DistanceDotProduct,
	}

	expectedValues := []string{"cosine", "euclidean", "dot_product"}

	for i, metric := range metrics {
		if string(metric) != expectedValues[i] {
			t.Errorf("Distance metric %d: expected %s, got %s", i, expectedValues[i], string(metric))
		}
	}
}

func TestErrDocumentNotFound(t *testing.T) {
	err := ErrDocumentNotFound{ID: "test123"}
	expected := "document not found: test123"

	if err.Error() != expected {
		t.Errorf("Expected error message %q, got %q", expected, err.Error())
	}
}

func TestErrIndexAlreadyExists(t *testing.T) {
	err := ErrIndexAlreadyExists{Name: "test_index"}
	expected := "index already exists: test_index"

	if err.Error() != expected {
		t.Errorf("Expected error message %q, got %q", expected, err.Error())
	}
}

func TestErrIndexNotFound(t *testing.T) {
	err := ErrIndexNotFound{Name: "missing_index"}
	expected := "index not found: missing_index"

	if err.Error() != expected {
		t.Errorf("Expected error message %q, got %q", expected, err.Error())
	}
}

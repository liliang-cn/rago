package store

import (
	"encoding/json"
	"testing"
)

// The retry policy must parse: an invalid service config makes grpc.NewClient
// fail, and every recall would then fail with it.
func TestCortexRemoteRetryConfigBuildsAClient(t *testing.T) {
	if !json.Valid([]byte(cortexRemoteServiceConfig)) {
		t.Fatal("service config is not valid JSON")
	}
	s, err := NewCortexRemoteMemoryStore(CortexRemoteConfig{Endpoint: "127.0.0.1:1"})
	if err != nil {
		t.Fatalf("client with retry policy: %v", err)
	}
	if s.owned != nil {
		_ = s.owned.Close()
	}
}

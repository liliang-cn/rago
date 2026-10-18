package mcp

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// A server that never answers its handshake must not hold up anyone else:
// every model round reads the tool list under the manager's lock, so a
// connect made while holding it stalls every run in the process.
func TestManager_AStuckConnectDoesNotBlockTheToolList(t *testing.T) {
	good := httptest.NewServer(newEchoHandler("ok:", false))
	defer good.Close()

	release := make(chan struct{})
	stuck := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-release:
		case <-r.Context().Done():
		}
	}))
	defer stuck.Close()
	defer close(release)

	m := NewManager(&Config{LoadedServers: []ServerConfig{
		{Name: "echo", Type: ServerTypeHTTP, URL: good.URL, DefaultTimeout: 5 * time.Second},
		{Name: "stuck", Type: ServerTypeHTTP, URL: stuck.URL, DefaultTimeout: 5 * time.Second},
	}})
	defer m.Close()

	ctx := context.Background()
	_, err := m.StartServer(ctx, "echo")
	require.NoError(t, err)

	connecting := make(chan error, 1)
	go func() {
		_, err := m.StartServer(ctx, "stuck")
		connecting <- err
	}()
	time.Sleep(100 * time.Millisecond) // let it get into the handshake

	listed := make(chan []AgentToolInfo, 1)
	go func() { listed <- m.GetAvailableTools(ctx) }()
	select {
	case tools := <-listed:
		require.NotEmpty(t, tools, "the connected server's tools are missing")
	case <-time.After(time.Second):
		t.Fatal("listing tools waited for another server's connect")
	}

	select {
	case err := <-connecting:
		t.Fatalf("the stuck server connected or gave up too early: %v", err)
	default:
	}
}

// Two starts of the same server at once make one connection, not two.
func TestManager_ConcurrentStartsShareOneConnect(t *testing.T) {
	good := httptest.NewServer(newEchoHandler("ok:", false))
	defer good.Close()

	m := NewManager(&Config{LoadedServers: []ServerConfig{
		{Name: "echo", Type: ServerTypeHTTP, URL: good.URL, DefaultTimeout: 5 * time.Second},
	}})
	defer m.Close()

	got := make(chan *Client, 4)
	for range 4 {
		go func() {
			c, err := m.StartServer(context.Background(), "echo")
			require.NoError(t, err)
			got <- c
		}()
	}
	first := <-got
	for range 3 {
		require.Same(t, first, <-got, "a second client was connected for the same server")
	}
}

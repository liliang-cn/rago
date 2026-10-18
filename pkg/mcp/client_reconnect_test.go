package mcp

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type echoArgs struct {
	Text string `json:"text"`
}

// newEchoHandler serves one "echo" tool whose reply starts with prefix, so a
// test can tell which server generation answered.
func newEchoHandler(prefix string, extraTool bool) http.Handler {
	srv := mcp.NewServer(&mcp.Implementation{Name: "echo", Version: "1"}, nil)
	mcp.AddTool(srv, &mcp.Tool{Name: "echo", Description: "echo " + prefix},
		func(ctx context.Context, req *mcp.CallToolRequest, in echoArgs) (*mcp.CallToolResult, any, error) {
			return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: prefix + in.Text}}}, nil, nil
		})
	if extraTool {
		mcp.AddTool(srv, &mcp.Tool{Name: "extra", Description: "extra"},
			func(ctx context.Context, req *mcp.CallToolRequest, in echoArgs) (*mcp.CallToolResult, any, error) {
				return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "extra"}}}, nil, nil
			})
	}
	return mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server { return srv }, nil)
}

func startServerAt(t *testing.T, addr string, h http.Handler) *httptest.Server {
	t.Helper()
	var ln net.Listener
	var err error
	for i := 0; i < 50; i++ { // the port may linger briefly after a close
		if ln, err = net.Listen("tcp", addr); err == nil {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	require.NoError(t, err)
	s := httptest.NewUnstartedServer(h)
	s.Listener.Close()
	s.Listener = ln
	s.Start()
	return s
}

func eventually(t *testing.T, cond func() bool, msg string) {
	t.Helper()
	require.Eventually(t, cond, 10*time.Second, 20*time.Millisecond, msg)
}

func callEcho(t *testing.T, c *Client) string {
	t.Helper()
	res, err := c.CallTool(context.Background(), "echo", map[string]interface{}{"text": "hi"})
	require.NoError(t, err)
	require.True(t, res.Success, res.Error)
	return res.Data.(string)
}

func newTestHTTPClient(t *testing.T, url string) *Client {
	t.Helper()
	c, err := NewClient(&ServerConfig{Name: "echo", Type: ServerTypeHTTP, URL: url, DefaultTimeout: 3 * time.Second}, nil)
	require.NoError(t, err)
	c.pingInterval = 50 * time.Millisecond
	c.reconnectBackoff = 30 * time.Millisecond
	c.reconnectMax = 100 * time.Millisecond
	return c
}

func TestClient_HTTPReconnectsAfterServerRestart(t *testing.T) {
	s1 := httptest.NewServer(newEchoHandler("one:", false))
	addr := s1.Listener.Addr().String()

	c := newTestHTTPClient(t, "http://"+addr)
	require.NoError(t, c.Connect(context.Background()))
	t.Cleanup(func() { c.Close() })
	assert.Equal(t, "one:hi", callEcho(t, c))
	assert.Len(t, c.GetTools(), 1)

	// Server goes away: the heartbeat notices and the client reports disconnected.
	s1.CloseClientConnections()
	s1.Close()
	eventually(t, func() bool { return !c.IsConnected() }, "client should notice the server is gone")

	// A call while it is down fails fast and clearly.
	_, err := c.CallTool(context.Background(), "echo", map[string]interface{}{"text": "hi"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not connected")

	// Server returns at the same URL with a changed tool set.
	s2 := startServerAt(t, addr, newEchoHandler("two:", true))
	t.Cleanup(s2.Close)
	eventually(t, c.IsConnected, "client should reconnect")

	assert.Equal(t, "two:hi", callEcho(t, c))
	assert.Len(t, c.GetTools(), 2, "tool cache is refreshed on reconnect")
	assert.NotNil(t, c.toolByName("extra"))
	require.NoError(t, c.Close()) // before the server: its open stream would stall s2.Close
}

func TestClient_HTTPCallTriggersImmediateReconnect(t *testing.T) {
	s1 := httptest.NewServer(newEchoHandler("one:", false))
	addr := s1.Listener.Addr().String()

	c := newTestHTTPClient(t, "http://"+addr)
	c.pingInterval = time.Hour // only a tool call can notice
	require.NoError(t, c.Connect(context.Background()))
	t.Cleanup(func() { c.Close() })

	// Drop the session first (as a failed ping would), then take the server down.
	c.mu.Lock()
	old := c.session
	c.mu.Unlock()
	c.markDisconnected()
	old.Close()
	s1.CloseClientConnections()
	s1.Close()
	assert.False(t, c.IsConnected())

	s2 := startServerAt(t, addr, newEchoHandler("two:", false))
	t.Cleanup(s2.Close)

	assert.Equal(t, "two:hi", callEcho(t, c))
	assert.True(t, c.IsConnected())
	require.NoError(t, c.Close())
}

func TestClient_HTTPCloseStopsReconnecting(t *testing.T) {
	s1 := httptest.NewServer(newEchoHandler("one:", false))
	addr := s1.Listener.Addr().String()

	c := newTestHTTPClient(t, "http://"+addr)
	require.NoError(t, c.Connect(context.Background()))

	s1.CloseClientConnections()
	s1.Close()
	eventually(t, func() bool { return !c.IsConnected() }, "disconnected")

	require.NoError(t, c.Close())

	// A server coming back must not revive a closed client.
	s2 := startServerAt(t, addr, newEchoHandler("two:", false))
	t.Cleanup(s2.Close)
	time.Sleep(300 * time.Millisecond)
	assert.False(t, c.IsConnected())
	_, err := c.CallTool(context.Background(), "echo", nil)
	assert.Error(t, err)
}

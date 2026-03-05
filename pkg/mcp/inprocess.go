package mcp

import (
	"context"
	"fmt"
	"io"
	"log"

	"github.com/mark3labs/mcp-filesystem-server/filesystemserver"
	mcpgo_server "github.com/mark3labs/mcp-go/server"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// createInProcessTransport creates an in-process transport for supported servers (e.g. filesystem)
// It uses io.Pipe to bridge the mark3labs MCP Server and the official SDK MCP Client in-memory.
func (c *Client) createInProcessTransport(ctx context.Context) (mcp.Transport, error) {
	// For now, we only support the filesystem server as an in-process server
	if c.config.Name == "filesystem" || c.config.Name == "builtin_filesystem" {
		// Create pipes for bidirectional communication
		// clientRead (rago reads from here) <- serverWrite (mark3labs writes to here)
		// serverRead (mark3labs reads from here) <- clientWrite (rago writes to here)
		clientRead, serverWrite := io.Pipe()
		serverRead, clientWrite := io.Pipe()

		// Allowed directories - default to root or workspace
		allowedDirs := []string{"/"}
		if len(c.config.Args) > 0 {
			allowedDirs = c.config.Args
		}

		fss, err := filesystemserver.NewFilesystemServer(allowedDirs)
		if err != nil {
			return nil, fmt.Errorf("failed to create in-process filesystem server: %w", err)
		}

		// Run the server in a goroutine
		go func() {
			// StdioServer.Listen(ctx, stdin, stdout) expects io.Reader and io.Writer
			stdioServer := mcpgo_server.NewStdioServer(fss)
			err := stdioServer.Listen(ctx, serverRead, serverWrite)
			if err != nil && err != io.EOF && err != io.ErrClosedPipe {
				log.Printf("[WARN] In-process filesystem server error: %v", err)
			}
			// Close pipes when the server stops
			clientRead.Close()
			clientWrite.Close()
		}()

		// Provide the other ends of the pipes to the official MCP SDK transport
		transport := &mcp.IOTransport{
			Reader: clientRead,
			Writer: clientWrite,
		}

		return transport, nil
	}

	return nil, fmt.Errorf("unsupported in-process server name: %s", c.config.Name)
}

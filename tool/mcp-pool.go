package tool

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
)

type mcpClientPool struct {
	clients map[string]MCPClient
	configs map[string]MCPServer
	mu      sync.RWMutex
}

var pool = &mcpClientPool{
	clients: make(map[string]MCPClient),
	configs: make(map[string]MCPServer),
}

func (p *mcpClientPool) getClient(
	ctx context.Context,
	name string,
	config MCPServer,
) (MCPClient, error) {
	p.mu.RLock()
	if client, exists := p.clients[name]; exists {
		p.mu.RUnlock()
		return client, nil
	}
	p.mu.RUnlock()

	p.mu.Lock()
	defer p.mu.Unlock()

	if client, exists := p.clients[name]; exists {
		return client, nil
	}

	var c MCPClient
	var err error

	switch config.Type {
	case MCPStdio:
		c, err = client.NewStdioMCPClient(
			config.Command,
			config.Env,
			config.Args...)
	case MCPSse:
		c, err = client.NewSSEMCPClient(
			config.URL,
			client.WithHeaders(config.Headers),
		)
	default:
		return nil, fmt.Errorf("invalid MCP type: %s", config.Type)
	}

	if err != nil {
		return nil, err
	}

	initRequest := mcp.InitializeRequest{}
	initRequest.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initRequest.Params.ClientInfo = mcp.Implementation{
		Name:    "llm",
		Version: "0.0.1",
	}

	_, err = c.Initialize(ctx, initRequest)
	if err != nil {
		c.Close()
		return nil, err
	}

	p.clients[name] = c
	p.configs[name] = config
	return c, nil
}

func (p *mcpClientPool) closeAll() {
	p.mu.Lock()
	defer p.mu.Unlock()

	for name, client := range p.clients {
		if err := client.Close(); err != nil {
			slog.Error("error closing MCP client", "name", name, "error", err)
		}
	}
	p.clients = make(map[string]MCPClient)
	p.configs = make(map[string]MCPServer)
}

func CloseMCPPool() {
	pool.closeAll()
}

package tool

import (
	"context"
	"fmt"
	"net/http"
	"os/exec"
	"sync"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type sessionWrapper struct {
	session *mcp.ClientSession
}

func (s *sessionWrapper) ListTools(ctx context.Context, params *mcp.ListToolsParams) (*mcp.ListToolsResult, error) {
	return s.session.ListTools(ctx, params)
}

func (s *sessionWrapper) CallTool(ctx context.Context, params *mcp.CallToolParams) (*mcp.CallToolResult, error) {
	return s.session.CallTool(ctx, params)
}

func (s *sessionWrapper) Close() error {
	return s.session.Close()
}

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

	client := mcp.NewClient(&mcp.Implementation{
		Name:    "llm",
		Version: "1.0.0",
	}, nil)

	var transport mcp.Transport
	var err error

	switch config.Type {
	case MCPStdio:
		cmd := exec.Command(config.Command, config.Args...)
		if len(config.Env) > 0 {
			cmd.Env = config.Env
		}
		transport = &mcp.CommandTransport{Command: cmd}
	case MCPSse:
		httpClient := &http.Client{}
		if len(config.Headers) > 0 {
			transport := http.DefaultTransport.(*http.Transport).Clone()
			httpClient.Transport = &headerTransport{
				base:    transport,
				headers: config.Headers,
			}
		}
		transport = &mcp.SSEClientTransport{
			Endpoint:   config.URL,
			HTTPClient: httpClient,
		}
	case MCPStreamableHTTP:
		httpClient := &http.Client{}
		if len(config.Headers) > 0 {
			transport := http.DefaultTransport.(*http.Transport).Clone()
			httpClient.Transport = &headerTransport{
				base:    transport,
				headers: config.Headers,
			}
		}
		transport = &mcp.StreamableClientTransport{
			Endpoint:   config.URL,
			HTTPClient: httpClient,
		}
	default:
		return nil, fmt.Errorf("invalid MCP type: %s", config.Type)
	}

	session, err := client.Connect(ctx, transport, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to connect MCP client: %w", err)
	}

	wrapper := &sessionWrapper{session: session}
	p.clients[name] = wrapper
	p.configs[name] = config
	return wrapper, nil
}

type headerTransport struct {
	base    http.RoundTripper
	headers map[string]string
}

func (h *headerTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	for key, value := range h.headers {
		req.Header.Set(key, value)
	}
	return h.base.RoundTrip(req)
}

func (p *mcpClientPool) closeAll() {
	p.mu.Lock()
	defer p.mu.Unlock()

	for _, client := range p.clients {
		client.Close()
	}
	p.clients = make(map[string]MCPClient)
	p.configs = make(map[string]MCPServer)
}

func CloseMCPPool() {
	pool.closeAll()
}

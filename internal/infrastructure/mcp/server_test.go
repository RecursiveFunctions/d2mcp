package mcp

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	mcpapi "github.com/mark3labs/mcp-go/mcp"
)

func TestRegisterResourceDelegatesToMCPServer(t *testing.T) {
	wrapped, err := NewServer("test", "0.0.0")
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}

	const uri = "d2://reference/test"
	resource := mcpapi.NewResource(uri, "Test reference", mcpapi.WithMIMEType("text/plain"))
	if err := wrapped.RegisterResource(resource, func(_ context.Context, request mcpapi.ReadResourceRequest) ([]mcpapi.ResourceContents, error) {
		return []mcpapi.ResourceContents{mcpapi.TextResourceContents{
			URI: request.Params.URI, MIMEType: "text/plain", Text: "registered",
		}}, nil
	}); err != nil {
		t.Fatalf("RegisterResource() error = %v", err)
	}

	request := json.RawMessage(`{"jsonrpc":"2.0","id":1,"method":"resources/read","params":{"uri":"` + uri + `"}}`)
	response := wrapped.GetMCPServer().HandleMessage(context.Background(), request)
	encoded, err := json.Marshal(response)
	if err != nil {
		t.Fatalf("failed to encode response: %v", err)
	}
	if !strings.Contains(string(encoded), `"text":"registered"`) || !strings.Contains(string(encoded), uri) {
		t.Fatalf("resources/read response = %s", encoded)
	}
}

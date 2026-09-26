package handler

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/i2y/d2mcp/internal/catalog"
)

// CatalogResourceHandler returns embedded D2 reference content for a catalog URI.
func CatalogResourceHandler(_ context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	resource, ok := catalog.ReadResource(request.Params.URI)
	if !ok {
		return nil, fmt.Errorf("catalog resource not found: %s", request.Params.URI)
	}
	return []mcp.ResourceContents{mcp.TextResourceContents{
		URI:      resource.URI,
		MIMEType: resource.MIMEType,
		Text:     resource.Text,
	}}, nil
}

var _ server.ResourceHandlerFunc = CatalogResourceHandler

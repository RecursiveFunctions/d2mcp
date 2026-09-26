package handler

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/i2y/d2mcp/internal/catalog"
)

// CapabilitiesHandler handles the d2_capabilities tool.
type CapabilitiesHandler struct{}

// NewCapabilitiesHandler creates a D2 catalog listing handler.
func NewCapabilitiesHandler() *CapabilitiesHandler {
	return &CapabilitiesHandler{}
}

// GetTool returns the MCP tool definition.
func (h *CapabilitiesHandler) GetTool() mcp.Tool {
	categoryIDs := make([]string, 0, len(catalog.ListCategories()))
	for _, category := range catalog.ListCategories() {
		categoryIDs = append(categoryIDs, category.ID)
	}
	return mcp.NewTool(
		"d2_capabilities",
		mcp.WithDescription("List the embedded D2 v0.9.0 capability catalog, grouped by category. Use the returned IDs with d2_help or the returned URIs as MCP resources."),
		mcp.WithString("category", mcp.Description("Optional exact category ID to list."), mcp.Enum(categoryIDs...)),
	)
}

// GetHandler returns the tool handler function.
func (h *CapabilitiesHandler) GetHandler() server.ToolHandlerFunc {
	return h.Handle
}

type capabilityCategory struct {
	catalog.Category
	Entries []catalog.Entry `json:"entries"`
}

type capabilitiesResponse struct {
	Version               string               `json:"version"`
	Exports               []string             `json:"exports"`
	Layouts               []string             `json:"layouts"`
	SourceAuthoring       []string             `json:"source_authoring"`
	ExternalLayoutPlugins bool                 `json:"external_layout_plugins"`
	WorkspaceSecurity     bool                 `json:"workspace_security"`
	BoundedHTTPSAssets    bool                 `json:"bounded_https_assets"`
	Categories            []capabilityCategory `json:"categories"`
}

// Handle lists all catalog categories or one exact category.
func (h *CapabilitiesHandler) Handle(_ context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	requestedCategory, err := optionalStringArgument(request.GetArguments(), "category")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	response := capabilitiesResponse{
		Version:               catalog.Version,
		Exports:               []string{"svg", "png", "pdf", "pptx", "gif", "ascii"},
		Layouts:               []string{"dagre", "elk", "tala"},
		SourceAuthoring:       []string{"create", "get", "update", "format", "validate", "oracle"},
		ExternalLayoutPlugins: false,
		WorkspaceSecurity:     true,
		BoundedHTTPSAssets:    true,
		Categories:            []capabilityCategory{},
	}
	for _, category := range catalog.ListCategories() {
		if requestedCategory != nil && *requestedCategory != category.ID {
			continue
		}
		response.Categories = append(response.Categories, capabilityCategory{
			Category: category,
			Entries:  catalog.ListCategory(category.ID),
		})
	}
	if requestedCategory != nil && len(response.Categories) == 0 {
		return mcp.NewToolResultError(fmt.Sprintf("unknown category %q", *requestedCategory)), nil
	}

	output, err := json.Marshal(response)
	if err != nil {
		return nil, fmt.Errorf("failed to encode capabilities response: %w", err)
	}
	return mcp.NewToolResultText(string(output)), nil
}

package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"math"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/i2y/d2mcp/internal/catalog"
)

// HelpHandler handles the d2_help tool.
type HelpHandler struct{}

// NewHelpHandler creates a D2 reference search handler.
func NewHelpHandler() *HelpHandler {
	return &HelpHandler{}
}

// GetTool returns the MCP tool definition.
func (h *HelpHandler) GetTool() mcp.Tool {
	return mcp.NewTool(
		"d2_help",
		mcp.WithDescription("Look up an exact D2 v0.9.0 catalog topic by ID or search the embedded reference. Exact IDs return the full reference document; other queries return ranked topic summaries."),
		mcp.WithString("query", mcp.Description("Exact topic ID or search terms, such as 'layout-engines', 'sequence diagram', or 'TALA'."), mcp.Required()),
		mcp.WithNumber("limit", mcp.Description("Maximum search results when query is not an exact topic ID."), mcp.DefaultNumber(catalog.DefaultSearchLimit), mcp.Min(1), mcp.Max(catalog.MaxSearchLimit)),
	)
}

// GetHandler returns the tool handler function.
func (h *HelpHandler) GetHandler() server.ToolHandlerFunc {
	return h.Handle
}

type helpResponse struct {
	Version  string                   `json:"version"`
	Query    string                   `json:"query"`
	Entry    *catalog.Entry           `json:"entry,omitempty"`
	Resource *catalog.ResourceContent `json:"resource,omitempty"`
	Results  []catalog.Entry          `json:"results,omitempty"`
}

// Handle processes a D2 reference lookup or search.
func (h *HelpHandler) Handle(_ context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	arguments := request.GetArguments()
	query, err := requiredHelpStringArgument(arguments, "query")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	limit, err := searchLimitArgument(arguments)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	response := helpResponse{Version: catalog.Version, Query: query}
	if entry, ok := catalog.Lookup(query); ok {
		resource, ok := catalog.ReadResource(entry.URI)
		if !ok {
			return nil, fmt.Errorf("catalog resource %q is unavailable", entry.URI)
		}
		response.Entry = &entry
		response.Resource = &resource
	} else {
		response.Results = catalog.Search(query, limit)
	}

	output, err := json.Marshal(response)
	if err != nil {
		return nil, fmt.Errorf("failed to encode help response: %w", err)
	}
	return mcp.NewToolResultText(string(output)), nil
}

func requiredHelpStringArgument(arguments map[string]any, name string) (string, error) {
	value, exists := arguments[name]
	if !exists {
		return "", fmt.Errorf("%s is required", name)
	}
	stringValue, ok := value.(string)
	if !ok {
		return "", fmt.Errorf("%s must be a string", name)
	}
	if stringValue == "" {
		return "", fmt.Errorf("%s must not be empty", name)
	}
	return stringValue, nil
}

func searchLimitArgument(arguments map[string]any) (int, error) {
	value, exists := arguments["limit"]
	if !exists {
		return catalog.DefaultSearchLimit, nil
	}

	var limit int
	switch number := value.(type) {
	case float64:
		if math.Trunc(number) != number {
			return 0, fmt.Errorf("limit must be an integer")
		}
		limit = int(number)
	case int:
		limit = number
	default:
		return 0, fmt.Errorf("limit must be a number")
	}
	if limit < 1 || limit > catalog.MaxSearchLimit {
		return 0, fmt.Errorf("limit must be between 1 and %d", catalog.MaxSearchLimit)
	}
	return limit, nil
}

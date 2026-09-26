package handler

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/i2y/d2mcp/internal/catalog"
)

func TestHelpHandlerReturnsExactResource(t *testing.T) {
	h := NewHelpHandler()
	request := toolRequest(map[string]any{"query": "layout-engines"})

	result, err := h.Handle(context.Background(), request)
	if err != nil || result.IsError {
		t.Fatalf("Handle() result = %+v, error = %v", result, err)
	}
	var response helpResponse
	decodeToolResult(t, result, &response)
	if response.Version != catalog.Version || response.Entry == nil || response.Entry.ID != "layout-engines" {
		t.Fatalf("Handle() response = %#v", response)
	}
	if response.Resource == nil || response.Resource.URI != response.Entry.URI || response.Resource.Text == "" {
		t.Fatalf("Handle() resource = %#v", response.Resource)
	}
}

func TestHelpHandlerSearchesAndValidatesLimit(t *testing.T) {
	h := NewHelpHandler()
	request := toolRequest(map[string]any{"query": "diagram", "limit": float64(2)})
	result, err := h.Handle(context.Background(), request)
	if err != nil || result.IsError {
		t.Fatalf("Handle() result = %+v, error = %v", result, err)
	}
	var response helpResponse
	decodeToolResult(t, result, &response)
	if len(response.Results) != 2 {
		t.Fatalf("Handle() returned %d results, want 2", len(response.Results))
	}

	for _, arguments := range []map[string]any{
		{},
		{"query": 42},
		{"query": "diagram", "limit": 1.5},
		{"query": "diagram", "limit": float64(catalog.MaxSearchLimit + 1)},
	} {
		result, err := h.Handle(context.Background(), toolRequest(arguments))
		if err != nil {
			t.Fatalf("Handle() error = %v", err)
		}
		if !result.IsError {
			t.Fatalf("Handle(%#v) did not return a tool error", arguments)
		}
	}
}

func TestCapabilitiesHandlerListsAndFiltersCatalog(t *testing.T) {
	h := NewCapabilitiesHandler()
	result, err := h.Handle(context.Background(), toolRequest(nil))
	if err != nil || result.IsError {
		t.Fatalf("Handle() result = %+v, error = %v", result, err)
	}
	var all capabilitiesResponse
	decodeToolResult(t, result, &all)
	if all.Version != catalog.Version || len(all.Categories) != len(catalog.ListCategories()) {
		t.Fatalf("Handle() response = %#v", all)
	}
	if len(all.Exports) != 6 || len(all.Layouts) != 3 || all.ExternalLayoutPlugins || !all.WorkspaceSecurity || !all.BoundedHTTPSAssets {
		t.Fatalf("machine-readable capabilities = %#v", all)
	}

	result, err = h.Handle(context.Background(), toolRequest(map[string]any{"category": "layout"}))
	if err != nil || result.IsError {
		t.Fatalf("filtered Handle() result = %+v, error = %v", result, err)
	}
	var filtered capabilitiesResponse
	decodeToolResult(t, result, &filtered)
	if len(filtered.Categories) != 1 || filtered.Categories[0].ID != "layout" || len(filtered.Categories[0].Entries) == 0 {
		t.Fatalf("filtered Handle() response = %#v", filtered)
	}

	result, err = h.Handle(context.Background(), toolRequest(map[string]any{"category": "unknown"}))
	if err != nil || !result.IsError {
		t.Fatalf("unknown category result = %+v, error = %v", result, err)
	}
}

func TestCatalogResourceHandler(t *testing.T) {
	entry, ok := catalog.Lookup("connections")
	if !ok {
		t.Fatal("catalog entry is missing")
	}
	request := mcp.ReadResourceRequest{}
	request.Params.URI = entry.URI
	contents, err := CatalogResourceHandler(context.Background(), request)
	if err != nil || len(contents) != 1 {
		t.Fatalf("CatalogResourceHandler() contents = %#v, error = %v", contents, err)
	}
	text, ok := contents[0].(mcp.TextResourceContents)
	if !ok || text.URI != entry.URI || text.MIMEType != catalog.ResourceMIMEType || text.Text == "" {
		t.Fatalf("CatalogResourceHandler() content = %#v", contents[0])
	}

	request.Params.URI = "d2://reference/v0.9.0/missing"
	if _, err := CatalogResourceHandler(context.Background(), request); err == nil {
		t.Fatal("CatalogResourceHandler() accepted an unknown URI")
	}
}

func toolRequest(arguments map[string]any) mcp.CallToolRequest {
	request := mcp.CallToolRequest{}
	request.Params.Arguments = arguments
	return request
}

func decodeToolResult(t *testing.T, result *mcp.CallToolResult, target any) {
	t.Helper()
	if len(result.Content) != 1 {
		t.Fatalf("tool result content count = %d", len(result.Content))
	}
	text, ok := result.Content[0].(mcp.TextContent)
	if !ok {
		t.Fatalf("tool result content type = %T", result.Content[0])
	}
	if err := json.Unmarshal([]byte(text.Text), target); err != nil {
		t.Fatalf("tool result is not JSON: %v", err)
	}
}

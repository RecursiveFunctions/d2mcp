package handler

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/i2y/d2mcp/internal/infrastructure/d2"
	"github.com/i2y/d2mcp/internal/usecase"
)

func TestSourceHandlersUpdateAndPreserveInvalidDraft(t *testing.T) {
	ctx := context.Background()
	uc := usecase.NewDiagramUseCase(d2.NewD2OracleRepository())
	if err := uc.Create(ctx, "diagram", "a -> b"); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	update := NewUpdateSourceHandler(uc)

	request := mcp.CallToolRequest{}
	request.Params.Arguments = map[string]any{"diagram_id": "diagram", "content": "a: {"}
	result, err := update.Handle(ctx, request)
	if err != nil {
		t.Fatalf("Handle() error = %v", err)
	}
	if result.IsError {
		t.Fatalf("invalid draft returned tool error: %+v", result)
	}
	text := result.Content[0].(mcp.TextContent)
	var response struct {
		Updated bool `json:"updated"`
		Valid   bool `json:"valid"`
	}
	if err := json.Unmarshal([]byte(text.Text), &response); err != nil {
		t.Fatalf("response is not JSON: %v", err)
	}
	if response.Valid || response.Updated {
		t.Fatalf("invalid update response = %+v", response)
	}
	if source, err := uc.GetSource(ctx, "diagram"); err != nil || source != "a -> b\n" {
		t.Fatalf("invalid draft changed source to %q, error %v", source, err)
	}

	request.Params.Arguments = map[string]any{"diagram_id": "diagram", "content": "x -> y"}
	result, err = update.Handle(ctx, request)
	if err != nil || result.IsError {
		t.Fatalf("valid update result = %+v, error %v", result, err)
	}
	if source, err := uc.GetSource(ctx, "diagram"); err != nil || source != "x -> y\n" {
		t.Fatalf("valid update source = %q, error %v", source, err)
	}
}

func TestFormatHandlerDoesNotMutateStoredSource(t *testing.T) {
	ctx := context.Background()
	uc := usecase.NewDiagramUseCase(d2.NewD2OracleRepository())
	if err := uc.Create(ctx, "diagram", "a -> b"); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	request := mcp.CallToolRequest{}
	request.Params.Arguments = map[string]any{"content": "x:{y}"}
	result, err := NewFormatHandler(uc).Handle(ctx, request)
	if err != nil || result.IsError {
		t.Fatalf("format result = %+v, error %v", result, err)
	}
	if got := result.Content[0].(mcp.TextContent).Text; got != "x: {y}\n" {
		t.Fatalf("formatted source = %q", got)
	}
	if source, err := uc.GetSource(ctx, "diagram"); err != nil || source != "a -> b\n" {
		t.Fatalf("format changed source to %q, error %v", source, err)
	}
}

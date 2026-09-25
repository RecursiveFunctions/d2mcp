package handler

import (
	"context"
	"encoding/json"
	"io"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/i2y/d2mcp/internal/domain/entity"
	"github.com/i2y/d2mcp/internal/usecase"
)

type handlerValidationRepository struct{}

func (*handlerValidationRepository) Render(context.Context, string, entity.ExportFormat, *entity.Theme) (io.Reader, error) {
	return nil, nil
}

func (*handlerValidationRepository) Create(context.Context, *entity.Diagram) error {
	return nil
}

func (*handlerValidationRepository) Export(context.Context, string, entity.ExportFormat) (io.Reader, error) {
	return nil, nil
}

func (*handlerValidationRepository) GetContent(context.Context, string) (string, error) {
	return "a -> b", nil
}

func (*handlerValidationRepository) Validate(_ context.Context, content string) (*entity.DiagramValidationResult, error) {
	if content == "a -> b" {
		return &entity.DiagramValidationResult{Valid: true, Diagnostics: []entity.DiagramDiagnostic{}}, nil
	}
	return &entity.DiagramValidationResult{
		Valid: false,
		Diagnostics: []entity.DiagramDiagnostic{{
			Severity: "error",
			Message:  "connection missing destination",
		}},
	}, nil
}

func TestValidateHandlerReturnsStructuredInvalidResult(t *testing.T) {
	handler := NewValidateHandler(usecase.NewDiagramUseCase(&handlerValidationRepository{}))
	request := mcp.CallToolRequest{}
	request.Params.Arguments = map[string]any{"content": "a ->"}

	result, err := handler.Handle(context.Background(), request)
	if err != nil {
		t.Fatalf("Handle() error = %v", err)
	}
	if result.IsError {
		t.Fatalf("Handle() returned tool error: %+v", result)
	}
	if len(result.Content) != 1 {
		t.Fatalf("Handle() content count = %d", len(result.Content))
	}
	text, ok := result.Content[0].(mcp.TextContent)
	if !ok {
		t.Fatalf("Handle() content type = %T", result.Content[0])
	}

	var response entity.DiagramValidationResult
	if err := json.Unmarshal([]byte(text.Text), &response); err != nil {
		t.Fatalf("Handle() response is not JSON: %v", err)
	}
	if response.Valid || len(response.Diagnostics) != 1 {
		t.Fatalf("Handle() response = %+v", response)
	}
}

func TestValidateHandlerRejectsInvalidSourceSelection(t *testing.T) {
	handler := NewValidateHandler(usecase.NewDiagramUseCase(&handlerValidationRepository{}))

	tests := []struct {
		name      string
		arguments map[string]any
	}{
		{name: "neither", arguments: map[string]any{}},
		{name: "both", arguments: map[string]any{"content": "a", "diagram_id": "diagram"}},
		{name: "wrong content type", arguments: map[string]any{"content": 42}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := mcp.CallToolRequest{}
			request.Params.Arguments = test.arguments
			result, err := handler.Handle(context.Background(), request)
			if err != nil {
				t.Fatalf("Handle() error = %v", err)
			}
			if !result.IsError {
				t.Fatalf("Handle() result = %+v, want tool error", result)
			}
		})
	}
}

func TestValidateHandlerAcceptsStoredDiagram(t *testing.T) {
	handler := NewValidateHandler(usecase.NewDiagramUseCase(&handlerValidationRepository{}))
	request := mcp.CallToolRequest{}
	request.Params.Arguments = map[string]any{"diagram_id": "diagram"}

	result, err := handler.Handle(context.Background(), request)
	if err != nil {
		t.Fatalf("Handle() error = %v", err)
	}
	if result.IsError {
		t.Fatalf("Handle() returned tool error: %+v", result)
	}
}

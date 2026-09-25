package handler

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/i2y/d2mcp/internal/usecase"
)

// ValidateHandler handles the d2_validate tool.
type ValidateHandler struct {
	useCase *usecase.DiagramUseCase
}

// NewValidateHandler creates a new validation handler.
func NewValidateHandler(useCase *usecase.DiagramUseCase) *ValidateHandler {
	return &ValidateHandler{useCase: useCase}
}

// GetTool returns the MCP tool definition.
func (h *ValidateHandler) GetTool() mcp.Tool {
	return mcp.NewTool(
		"d2_validate",
		mcp.WithDescription("Validate D2 source and report structured syntax or semantic diagnostics. Provide exactly one of content or diagram_id. Invalid D2 returns valid=false instead of a tool error. For simple, unambiguous errors, the result may include repaired_content that has been revalidated without changing the stored diagram."),
		mcp.WithString("content", mcp.Description("Raw D2 source to validate. Mutually exclusive with diagram_id.")),
		mcp.WithString("diagram_id", mcp.Description("ID of a stored diagram whose current source should be validated. Mutually exclusive with content.")),
	)
}

// GetHandler returns the tool handler function.
func (h *ValidateHandler) GetHandler() server.ToolHandlerFunc {
	return h.Handle
}

// Handle processes the validation request.
func (h *ValidateHandler) Handle(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	arguments := request.GetArguments()
	content, err := optionalStringArgument(arguments, "content")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	diagramID, err := optionalStringArgument(arguments, "diagram_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	result, err := h.useCase.ValidateDiagram(ctx, content, diagramID)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to validate diagram", err), nil
	}

	output, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to encode validation result: %w", err)
	}
	return mcp.NewToolResultText(string(output)), nil
}

func optionalStringArgument(arguments map[string]any, name string) (*string, error) {
	value, exists := arguments[name]
	if !exists {
		return nil, nil
	}
	stringValue, ok := value.(string)
	if !ok {
		return nil, fmt.Errorf("%s must be a string", name)
	}
	return &stringValue, nil
}

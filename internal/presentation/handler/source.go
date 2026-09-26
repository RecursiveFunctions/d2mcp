package handler

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/i2y/d2mcp/internal/usecase"
)

// GetSourceHandler handles the d2_get_source tool.
type GetSourceHandler struct{ useCase *usecase.DiagramUseCase }

func NewGetSourceHandler(useCase *usecase.DiagramUseCase) *GetSourceHandler {
	return &GetSourceHandler{useCase: useCase}
}

func (h *GetSourceHandler) GetTool() mcp.Tool {
	return mcp.NewTool(
		"d2_get_source",
		mcp.WithDescription("Return the complete canonical D2 source for a stored diagram, including Oracle edits."),
		mcp.WithString("diagram_id", mcp.Description("Stored diagram ID"), mcp.Required()),
	)
}

func (h *GetSourceHandler) GetHandler() server.ToolHandlerFunc { return h.Handle }

func (h *GetSourceHandler) Handle(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	diagramID := mcp.ParseString(request, "diagram_id", "")
	content, err := h.useCase.GetSource(ctx, diagramID)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to get diagram source", err), nil
	}
	return mcp.NewToolResultText(content), nil
}

// UpdateSourceHandler handles the d2_update_source tool.
type UpdateSourceHandler struct{ useCase *usecase.DiagramUseCase }

func NewUpdateSourceHandler(useCase *usecase.DiagramUseCase) *UpdateSourceHandler {
	return &UpdateSourceHandler{useCase: useCase}
}

func (h *UpdateSourceHandler) GetTool() mcp.Tool {
	return mcp.NewTool(
		"d2_update_source",
		mcp.WithDescription("Validate complete D2 source and atomically replace a stored diagram. Invalid source returns diagnostics and leaves the diagram unchanged."),
		mcp.WithString("diagram_id", mcp.Description("Stored diagram ID"), mcp.Required()),
		mcp.WithString("content", mcp.Description("Complete replacement D2 source"), mcp.Required()),
	)
}

func (h *UpdateSourceHandler) GetHandler() server.ToolHandlerFunc { return h.Handle }

func (h *UpdateSourceHandler) Handle(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	diagramID := mcp.ParseString(request, "diagram_id", "")
	content, err := stringArgumentAllowEmpty(request.GetArguments(), "content")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	validation, err := h.useCase.UpdateSource(ctx, diagramID, content)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to update diagram source", err), nil
	}
	output, err := json.Marshal(struct {
		Updated     bool `json:"updated"`
		Valid       bool `json:"valid"`
		Diagnostics any  `json:"diagnostics"`
	}{Updated: validation.Valid, Valid: validation.Valid, Diagnostics: validation.Diagnostics})
	if err != nil {
		return nil, fmt.Errorf("failed to encode source update result: %w", err)
	}
	return mcp.NewToolResultText(string(output)), nil
}

// FormatHandler handles the d2_format tool.
type FormatHandler struct{ useCase *usecase.DiagramUseCase }

func NewFormatHandler(useCase *usecase.DiagramUseCase) *FormatHandler {
	return &FormatHandler{useCase: useCase}
}

func (h *FormatHandler) GetTool() mcp.Tool {
	return mcp.NewTool(
		"d2_format",
		mcp.WithDescription("Format valid D2 source without storing or rendering it."),
		mcp.WithString("content", mcp.Description("D2 source to format"), mcp.Required()),
	)
}

func (h *FormatHandler) GetHandler() server.ToolHandlerFunc { return h.Handle }

func (h *FormatHandler) Handle(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	content, err := stringArgumentAllowEmpty(request.GetArguments(), "content")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	formatted, err := h.useCase.FormatSource(ctx, content)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to format D2 source", err), nil
	}
	return mcp.NewToolResultText(formatted), nil
}

func stringArgumentAllowEmpty(arguments map[string]any, name string) (string, error) {
	value, exists := arguments[name]
	if !exists {
		return "", fmt.Errorf("%s is required", name)
	}
	stringValue, ok := value.(string)
	if !ok {
		return "", fmt.Errorf("%s must be a string", name)
	}
	return stringValue, nil
}

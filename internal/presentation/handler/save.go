package handler

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/i2y/d2mcp/internal/domain/entity"
	"github.com/i2y/d2mcp/internal/security/workspace"
	"github.com/i2y/d2mcp/internal/usecase"
)

// SaveHandler handles the d2_save tool.
type SaveHandler struct {
	useCase    *usecase.DiagramUseCase
	workspaces *workspace.Roots
}

// NewSaveHandler creates a new save handler.
func NewSaveHandler(useCase *usecase.DiagramUseCase, workspaces *workspace.Roots) *SaveHandler {
	return &SaveHandler{
		useCase:    useCase,
		workspaces: workspaces,
	}
}

// GetTool returns the MCP tool definition.
func (h *SaveHandler) GetTool() mcp.Tool {
	options := []mcp.ToolOption{
		mcp.WithDescription("Save an existing diagram within a configured workspace root using a bundled D2 renderer."),
		mcp.WithString("diagramId", mcp.Description("ID of the diagram to save"), mcp.Required()),
		mcp.WithString("format", mcp.Description("Export format"), mcp.Enum("svg", "png", "pdf", "pptx", "gif", "ascii", "txt"), mcp.DefaultString("svg")),
		mcp.WithString("path", mcp.Description("Output path relative to the selected workspace root")),
		mcp.WithString("workspace_root", mcp.Description("Configured workspace root for the output"), mcp.DefaultString("default")),
	}
	options = append(options, renderToolOptions()...)
	return mcp.NewTool("d2_save", options...)
}

// GetHandler returns the tool handler function.
func (h *SaveHandler) GetHandler() server.ToolHandlerFunc {
	return h.Handle
}

// Handle processes the save request.
func (h *SaveHandler) Handle(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Extract arguments.
	diagramID := mcp.ParseString(request, "diagramId", "")
	if diagramID == "" {
		return mcp.NewToolResultError("diagramId is required"), nil
	}

	formatStr := mcp.ParseString(request, "format", "svg")
	format := entity.ExportFormat(formatStr)
	options, err := parseRenderOptions(request)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	outputPath := mcp.ParseString(request, "path", "")
	rootName := mcp.ParseString(request, "workspace_root", workspace.DefaultRoot)

	// Export the diagram.
	reader, err := h.useCase.ExportDiagramWithOptions(ctx, diagramID, format, options)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to export diagram", err), nil
	}

	// Read the output.
	data, err := io.ReadAll(reader)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to read exported output", err), nil
	}

	// Determine and confine the output path.
	if outputPath == "" {
		filename := fmt.Sprintf("%s_%d.%s", diagramID, time.Now().Unix(), formatStr)
		outputPath = filepath.Join("d2mcp_output", filename)
	}
	outputPath, err = h.workspaces.ResolveWrite(rootName, outputPath)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("Output path is outside the workspace root", err), nil
	}
	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to create output directory", err), nil
	}
	outputPath, err = h.workspaces.ResolveWrite(rootName, outputPath)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("Output path is outside the workspace root", err), nil
	}

	// Write to file.
	if err := os.WriteFile(outputPath, data, 0o644); err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to write output file", err), nil
	}

	// Return result.
	result := fmt.Sprintf("Diagram saved to: %s\n", outputPath)
	result += fmt.Sprintf("Format: %s\n", formatStr)
	result += fmt.Sprintf("Size: %d bytes", len(data))

	return mcp.NewToolResultText(result), nil
}

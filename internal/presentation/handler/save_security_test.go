package handler

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/i2y/d2mcp/internal/domain/entity"
	d2infra "github.com/i2y/d2mcp/internal/infrastructure/d2"
	"github.com/i2y/d2mcp/internal/security/workspace"
	"github.com/i2y/d2mcp/internal/usecase"
)

func TestSaveHandlerConfinesOutputToSelectedWorkspace(t *testing.T) {
	root := saveScratchDir(t)
	outside := saveScratchDir(t)
	workspaces, err := workspace.New(root, map[string]string{"output": root})
	if err != nil {
		t.Fatal(err)
	}
	created, err := d2infra.NewD2RepositoryWithSecurity(d2infra.SecurityConfig{Workspaces: workspaces})
	if err != nil {
		t.Fatal(err)
	}
	if err := created.Create(context.Background(), &entity.Diagram{ID: "diagram", Content: "a -> b", WorkspaceRoot: "output"}); err != nil {
		t.Fatal(err)
	}
	handler := NewSaveHandler(usecase.NewDiagramUseCase(created), workspaces)

	request := mcp.CallToolRequest{}
	arguments := map[string]any{
		"diagramId": "diagram", "format": "svg", "path": "exports/diagram.svg", "workspace_root": "output",
	}
	request.Params.Arguments = arguments
	result, err := handler.Handle(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatalf("confined save returned error: %+v", result)
	}
	if _, err := os.Stat(filepath.Join(root, "exports", "diagram.svg")); err != nil {
		t.Fatalf("saved output: %v", err)
	}

	arguments["path"] = filepath.Join(outside, "escape.svg")
	result, err = handler.Handle(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError {
		t.Fatal("outside save path succeeded")
	}
}

func saveScratchDir(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp(".", ".save-security-")
	if err != nil {
		t.Fatal(err)
	}
	absolute, err := filepath.Abs(dir)
	if err != nil {
		os.RemoveAll(dir)
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(absolute) })
	return absolute
}

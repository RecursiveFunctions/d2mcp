package repository

import (
	"context"
	"io"

	"github.com/i2y/d2mcp/internal/domain/entity"
)

// DiagramRepository defines the interface for diagram operations.
type DiagramRepository interface {
	// Render renders D2 text into a diagram with specified format.
	Render(ctx context.Context, content string, format entity.ExportFormat, theme *entity.Theme) (io.Reader, error)

	// Create creates a new diagram programmatically.
	Create(ctx context.Context, diagram *entity.Diagram) error

	// Export exports the diagram to the specified format.
	Export(ctx context.Context, diagramID string, format entity.ExportFormat) (io.Reader, error)

	// GetContent returns the current D2 source for a stored diagram.
	GetContent(ctx context.Context, diagramID string) (string, error)

	// Validate validates D2 source without changing repository state.
	Validate(ctx context.Context, content string) (*entity.DiagramValidationResult, error)
}

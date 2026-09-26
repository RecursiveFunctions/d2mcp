package d2

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"

	"github.com/d2lang/d2/d2compiler"
	"github.com/d2lang/d2/d2format"
	"github.com/d2lang/d2/d2graph"
	"github.com/d2lang/d2/d2parser"
	"github.com/d2lang/d2/lib/log"

	"github.com/i2y/d2mcp/internal/domain/entity"
	"github.com/i2y/d2mcp/internal/domain/repository"
	"github.com/i2y/d2mcp/internal/security/assets"
	"github.com/i2y/d2mcp/internal/security/remote"
	"github.com/i2y/d2mcp/internal/security/workspace"
)

// D2Repository implements the DiagramRepository interface using D2.
type D2Repository struct {
	diagrams      map[string]*diagramData
	workspaces    *workspace.Roots
	assetResolver *assets.Resolver
	httpClient    *http.Client
	maxAssetBytes int64
	mu            sync.RWMutex
}

// SecurityConfig supplies the confined workspaces and remote client used during compilation and rendering.
type SecurityConfig struct {
	Workspaces    *workspace.Roots
	HTTPClient    *http.Client
	MaxAssetBytes int64
}

// diagramData holds the D2 graph and related data.
type diagramData struct {
	content       string
	graph         *d2graph.Graph
	workspaceRoot string
}

// NewD2Repository creates a D2 repository secured to the current working directory.
func NewD2Repository() repository.DiagramRepository {
	repository, err := newD2Repository(SecurityConfig{})
	if err != nil {
		panic(err)
	}
	return repository
}

// NewD2RepositoryWithSecurity creates a D2 repository with explicit security dependencies.
func NewD2RepositoryWithSecurity(config SecurityConfig) (repository.DiagramRepository, error) {
	return newD2Repository(config)
}

func newD2Repository(config SecurityConfig) (*D2Repository, error) {
	if config.MaxAssetBytes > 64<<20 {
		return nil, errors.New("max asset bytes cannot exceed 67108864")
	}
	workspaces := config.Workspaces
	if workspaces == nil {
		var err error
		workspaces, err = workspace.New("", nil)
		if err != nil {
			return nil, err
		}
	}
	client := config.HTTPClient
	if client == nil {
		var err error
		client, err = remote.NewClient(remote.Config{MaxResponseBytes: config.MaxAssetBytes})
		if err != nil {
			return nil, err
		}
	}
	assetResolver, err := assets.New(workspaces, client, config.MaxAssetBytes)
	if err != nil {
		return nil, err
	}
	maxAssetBytes := config.MaxAssetBytes
	if maxAssetBytes == 0 {
		maxAssetBytes = assets.DefaultMaxBytes
	}
	return &D2Repository{
		diagrams:      make(map[string]*diagramData),
		workspaces:    workspaces,
		assetResolver: assetResolver,
		httpClient:    client,
		maxAssetBytes: maxAssetBytes,
	}, nil
}

// withSilentD2 executes a function with D2 logging disabled.
func withSilentD2(ctx context.Context, fn func(context.Context) error) error {
	nullLogger := slog.New(slog.NewTextHandler(io.Discard, nil))
	return fn(log.With(ctx, nullLogger))
}

// Render renders D2 text with backward-compatible theme handling.
func (r *D2Repository) Render(ctx context.Context, content string, format entity.ExportFormat, theme *entity.Theme) (io.Reader, error) {
	options := entity.RenderOptions{}
	if theme != nil {
		themeID := int64(theme.ID)
		options.Theme.LightID = &themeID
	}
	return r.RenderWithOptions(ctx, content, format, options)
}

func (r *D2Repository) compileGraph(rootName, content string) (*d2graph.Graph, error) {
	workspaces := r.workspaces
	if workspaces == nil {
		var err error
		workspaces, err = workspace.New("", nil)
		if err != nil {
			return nil, err
		}
	}
	filesystem, err := workspaces.FS(rootName)
	if err != nil {
		return nil, fmt.Errorf("select workspace root: %w", err)
	}
	graph, _, err := d2compiler.Compile("index.d2", strings.NewReader(content), &d2compiler.CompileOptions{
		UTF16Pos: false,
		FS:       filesystem,
	})
	return graph, err
}

// Create creates a new diagram programmatically.
func (r *D2Repository) Create(ctx context.Context, diagram *entity.Diagram) error {
	rootName := diagram.WorkspaceRoot
	if rootName == "" {
		rootName = workspace.DefaultRoot
	}
	graph, err := r.compileGraph(rootName, diagram.Content)
	if err != nil {
		return fmt.Errorf("failed to compile diagram: %w", err)
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	r.diagrams[diagram.ID] = &diagramData{
		content:       diagram.Content,
		graph:         graph,
		workspaceRoot: rootName,
	}
	return nil
}

// Export exports the diagram to the specified format.
func (r *D2Repository) Export(ctx context.Context, diagramID string, format entity.ExportFormat) (io.Reader, error) {
	return r.ExportWithOptions(ctx, diagramID, format, entity.RenderOptions{})
}

// ExportWithOptions exports a stored diagram using typed D2 v0.9 options.
func (r *D2Repository) ExportWithOptions(ctx context.Context, diagramID string, format entity.ExportFormat, options entity.RenderOptions) (io.Reader, error) {
	r.mu.RLock()
	data, exists := r.diagrams[diagramID]
	if !exists {
		r.mu.RUnlock()
		return nil, fmt.Errorf("diagram %s not found", diagramID)
	}
	content := data.content
	rootName := data.workspaceRoot
	r.mu.RUnlock()
	if rootName == "" {
		rootName = workspace.DefaultRoot
	}
	return r.renderWithOptionsInRoot(ctx, rootName, content, format, options)
}

// GetContent returns the stored D2 source.
func (r *D2Repository) GetContent(ctx context.Context, diagramID string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	data, exists := r.diagrams[diagramID]
	if !exists {
		return "", fmt.Errorf("diagram %s not found", diagramID)
	}
	return data.content, nil
}

// ReplaceContent validates content before atomically replacing stored source.
func (r *D2Repository) ReplaceContent(ctx context.Context, diagramID, content string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if diagramID == "" {
		return errors.New("diagram ID is required")
	}

	r.mu.RLock()
	data, exists := r.diagrams[diagramID]
	if !exists {
		r.mu.RUnlock()
		return fmt.Errorf("diagram %s not found", diagramID)
	}
	rootName := data.workspaceRoot
	r.mu.RUnlock()
	if rootName == "" {
		rootName = workspace.DefaultRoot
	}
	graph, err := r.compileGraph(rootName, content)
	if err != nil {
		return fmt.Errorf("failed to compile diagram: %w", err)
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.diagrams[diagramID]; !exists {
		return fmt.Errorf("diagram %s not found", diagramID)
	}
	r.diagrams[diagramID] = &diagramData{content: content, graph: graph, workspaceRoot: rootName}
	return nil
}

// FormatContent formats valid D2 source without storing it.
func (r *D2Repository) FormatContent(ctx context.Context, content string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	ast, err := d2parser.Parse("", strings.NewReader(content), &d2parser.ParseOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to parse diagram: %w", err)
	}
	return d2format.Format(ast), nil
}

// Validate validates D2 source and normalizes compiler diagnostics.
func (r *D2Repository) Validate(ctx context.Context, content string) (*entity.DiagramValidationResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	_, err := r.compileGraph(workspace.DefaultRoot, content)
	if err == nil {
		return &entity.DiagramValidationResult{
			Valid:       true,
			Diagnostics: []entity.DiagramDiagnostic{},
		}, nil
	}

	result := &entity.DiagramValidationResult{
		Valid:       false,
		Diagnostics: []entity.DiagramDiagnostic{},
	}

	var parseErr *d2parser.ParseError
	if !errors.As(err, &parseErr) {
		result.Diagnostics = append(result.Diagnostics, entity.DiagramDiagnostic{
			Severity: "error",
			Message:  err.Error(),
		})
		return result, nil
	}

	for _, compilerErr := range parseErr.Errors {
		message := strings.TrimPrefix(compilerErr.Message, compilerErr.Range.String()+": ")
		result.Diagnostics = append(result.Diagnostics, entity.DiagramDiagnostic{
			Severity: "error",
			Message:  message,
			Range: &entity.SourceRange{
				Start: entity.SourcePosition{
					Line:   compilerErr.Range.Start.Line + 1,
					Column: compilerErr.Range.Start.Column + 1,
					Byte:   compilerErr.Range.Start.Byte,
				},
				End: entity.SourcePosition{
					Line:   compilerErr.Range.End.Line + 1,
					Column: compilerErr.Range.End.Column + 1,
					Byte:   compilerErr.Range.End.Byte,
				},
			},
		})
	}

	return result, nil
}

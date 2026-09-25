package usecase

import (
	"context"
	"errors"
	"io"
	"testing"

	"github.com/i2y/d2mcp/internal/domain/entity"
	d2infra "github.com/i2y/d2mcp/internal/infrastructure/d2"
)

type validationRepository struct {
	content       map[string]string
	getContentErr error
	validateErr   error
	validate      func(string) *entity.DiagramValidationResult
}

func (r *validationRepository) Render(context.Context, string, entity.ExportFormat, *entity.Theme) (io.Reader, error) {
	return nil, nil
}

func (r *validationRepository) Create(context.Context, *entity.Diagram) error {
	return nil
}

func (r *validationRepository) Export(context.Context, string, entity.ExportFormat) (io.Reader, error) {
	return nil, nil
}

func (r *validationRepository) GetContent(_ context.Context, diagramID string) (string, error) {
	if r.getContentErr != nil {
		return "", r.getContentErr
	}
	content, ok := r.content[diagramID]
	if !ok {
		return "", errors.New("diagram not found")
	}
	return content, nil
}

func (r *validationRepository) Validate(_ context.Context, content string) (*entity.DiagramValidationResult, error) {
	if r.validateErr != nil {
		return nil, r.validateErr
	}
	if r.validate != nil {
		return r.validate(content), nil
	}
	return &entity.DiagramValidationResult{Valid: true, Diagnostics: []entity.DiagramDiagnostic{}}, nil
}

func stringPointer(value string) *string {
	return &value
}

func TestValidateDiagramSourceSelection(t *testing.T) {
	repo := &validationRepository{content: map[string]string{"stored": "a -> b"}}
	useCase := NewDiagramUseCase(repo)

	tests := []struct {
		name      string
		content   *string
		diagramID *string
		wantErr   bool
	}{
		{name: "raw content", content: stringPointer("a -> b")},
		{name: "empty raw content", content: stringPointer("")},
		{name: "stored diagram", diagramID: stringPointer("stored")},
		{name: "both sources", content: stringPointer("a"), diagramID: stringPointer("stored"), wantErr: true},
		{name: "neither source", wantErr: true},
		{name: "empty diagram ID", diagramID: stringPointer(""), wantErr: true},
		{name: "missing diagram", diagramID: stringPointer("missing"), wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := useCase.ValidateDiagram(context.Background(), test.content, test.diagramID)
			if (err != nil) != test.wantErr {
				t.Fatalf("ValidateDiagram() error = %v, wantErr %v", err, test.wantErr)
			}
			if !test.wantErr && (result == nil || !result.Valid) {
				t.Fatalf("ValidateDiagram() result = %+v", result)
			}
		})
	}
}

func TestValidateDiagramRepairsMissingTerminators(t *testing.T) {
	const source = `a: { label: "value`
	const quoteRepaired = `a: { label: "value"`
	const fullyRepaired = `a: { label: "value"}`

	repo := &validationRepository{
		validate: func(content string) *entity.DiagramValidationResult {
			switch content {
			case source:
				return invalidResult(`double quoted strings must be terminated with "`)
			case quoteRepaired:
				return invalidResult("maps must be terminated with }")
			case fullyRepaired:
				return &entity.DiagramValidationResult{Valid: true, Diagnostics: []entity.DiagramDiagnostic{}}
			default:
				t.Fatalf("unexpected validation candidate %q", content)
				return nil
			}
		},
	}

	result, err := NewDiagramUseCase(repo).ValidateDiagram(context.Background(), stringPointer(source), nil)
	if err != nil {
		t.Fatalf("ValidateDiagram() error = %v", err)
	}
	if result.Valid || !result.RepairAvailable {
		t.Fatalf("ValidateDiagram() result = %+v", result)
	}
	if result.RepairedContent != fullyRepaired {
		t.Errorf("repaired content = %q, want %q", result.RepairedContent, fullyRepaired)
	}
	if len(result.Repairs) != 2 {
		t.Errorf("repairs = %#v", result.Repairs)
	}
}

func TestValidateDiagramDoesNotRepairAmbiguousError(t *testing.T) {
	repo := &validationRepository{
		validate: func(string) *entity.DiagramValidationResult {
			return invalidResult("connection missing destination")
		},
	}

	result, err := NewDiagramUseCase(repo).ValidateDiagram(context.Background(), stringPointer("a ->"), nil)
	if err != nil {
		t.Fatalf("ValidateDiagram() error = %v", err)
	}
	if result.RepairAvailable || result.RepairedContent != "" || len(result.Repairs) != 0 {
		t.Fatalf("ValidateDiagram() unexpectedly repaired source: %+v", result)
	}
}

func TestRemoveUnexpectedMapTerminators(t *testing.T) {
	diagnostics := []entity.DiagramDiagnostic{
		unexpectedMapTerminator(10, 11),
		unexpectedMapTerminator(9, 10),
	}
	candidate, ok := removeUnexpectedMapTerminators("a: { b } }}", diagnostics)
	if !ok {
		t.Fatal("removeUnexpectedMapTerminators() rejected repairable input")
	}
	if candidate.content != "a: { b } " {
		t.Errorf("candidate content = %q", candidate.content)
	}
}

func TestValidateDiagramPropagatesRepositoryError(t *testing.T) {
	wantErr := errors.New("validation failed")
	repo := &validationRepository{validateErr: wantErr}

	_, err := NewDiagramUseCase(repo).ValidateDiagram(context.Background(), stringPointer("a"), nil)
	if !errors.Is(err, wantErr) {
		t.Fatalf("ValidateDiagram() error = %v, want %v", err, wantErr)
	}
}

func TestValidateDiagramRepairsRealD2Syntax(t *testing.T) {
	useCase := NewDiagramUseCase(d2infra.NewD2Repository())
	tests := []struct {
		name   string
		source string
		want   string
	}{
		{name: "missing map terminator", source: "a: {\n b", want: "a: {\n b}"},
		{name: "extra map terminator", source: "a: { b } }", want: "a: { b } "},
		{name: "missing quote", source: `a: "label`, want: `a: "label"`},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := useCase.ValidateDiagram(context.Background(), stringPointer(test.source), nil)
			if err != nil {
				t.Fatalf("ValidateDiagram() error = %v", err)
			}
			if !result.RepairAvailable || result.RepairedContent != test.want {
				t.Fatalf("ValidateDiagram() result = %+v, want repaired content %q", result, test.want)
			}
		})
	}
}

func invalidResult(message string) *entity.DiagramValidationResult {
	return &entity.DiagramValidationResult{
		Valid: false,
		Diagnostics: []entity.DiagramDiagnostic{{
			Severity: "error",
			Message:  message,
		}},
	}
}

func unexpectedMapTerminator(start, end int) entity.DiagramDiagnostic {
	return entity.DiagramDiagnostic{
		Severity: "error",
		Message:  "unexpected map termination character } in file map",
		Range: &entity.SourceRange{
			Start: entity.SourcePosition{Byte: start},
			End:   entity.SourcePosition{Byte: end},
		},
	}
}

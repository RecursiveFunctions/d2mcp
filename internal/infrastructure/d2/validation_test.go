package d2

import (
	"context"
	"testing"

	"github.com/i2y/d2mcp/internal/domain/entity"
)

func TestD2RepositoryValidate(t *testing.T) {
	repo := NewD2Repository()
	ctx := context.Background()

	valid, err := repo.Validate(ctx, "a -> b")
	if err != nil {
		t.Fatalf("Validate() valid source error = %v", err)
	}
	if !valid.Valid || len(valid.Diagnostics) != 0 {
		t.Fatalf("Validate() valid source = %+v", valid)
	}

	invalid, err := repo.Validate(ctx, "a: {\n b")
	if err != nil {
		t.Fatalf("Validate() invalid source error = %v", err)
	}
	if invalid.Valid || len(invalid.Diagnostics) != 1 {
		t.Fatalf("Validate() invalid source = %+v", invalid)
	}

	diagnostic := invalid.Diagnostics[0]
	if diagnostic.Message != "maps must be terminated with }" {
		t.Errorf("diagnostic message = %q", diagnostic.Message)
	}
	if diagnostic.Severity != "error" {
		t.Errorf("diagnostic severity = %q", diagnostic.Severity)
	}
	if diagnostic.Range == nil {
		t.Fatal("diagnostic range is nil")
	}
	if diagnostic.Range.Start.Line != 1 || diagnostic.Range.Start.Column != 4 {
		t.Errorf("diagnostic start = %+v", diagnostic.Range.Start)
	}
}

func TestD2RepositoryValidateHonorsCanceledContext(t *testing.T) {
	repo := NewD2Repository()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if _, err := repo.Validate(ctx, "a -> b"); err != context.Canceled {
		t.Fatalf("Validate() error = %v, want context.Canceled", err)
	}
}

func TestD2RepositoryGetContent(t *testing.T) {
	repo := NewD2Repository()
	ctx := context.Background()
	want := "a -> b"

	if err := repo.Create(ctx, &entity.Diagram{ID: "diagram", Content: want}); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	got, err := repo.GetContent(ctx, "diagram")
	if err != nil {
		t.Fatalf("GetContent() error = %v", err)
	}
	if got != want {
		t.Errorf("GetContent() = %q, want %q", got, want)
	}

	if _, err := repo.GetContent(ctx, "missing"); err == nil {
		t.Fatal("GetContent() missing diagram returned nil error")
	}
}

func TestD2OracleRepositoryGetContentIncludesEdits(t *testing.T) {
	repo := NewD2OracleRepository()
	ctx := context.Background()

	if err := repo.Create(ctx, &entity.Diagram{ID: "diagram", Content: "a"}); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if _, err := repo.CreateElement(ctx, "diagram", nil, "b"); err != nil {
		t.Fatalf("CreateElement() error = %v", err)
	}

	content, err := repo.GetContent(ctx, "diagram")
	if err != nil {
		t.Fatalf("GetContent() error = %v", err)
	}
	if content == "a" {
		t.Fatalf("GetContent() did not include Oracle edit: %q", content)
	}
}

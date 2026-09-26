package d2

import (
	"context"
	"testing"
)

func TestReplaceContentIsAtomic(t *testing.T) {
	ctx := context.Background()
	repo := NewD2OracleRepository().(*D2OracleRepository)
	if err := repo.LoadDiagram(ctx, "diagram", "a -> b"); err != nil {
		t.Fatalf("LoadDiagram() error = %v", err)
	}

	before, err := repo.GetContent(ctx, "diagram")
	if err != nil {
		t.Fatalf("GetContent() error = %v", err)
	}
	if err := repo.ReplaceContent(ctx, "diagram", "a: {"); err == nil {
		t.Fatal("ReplaceContent() invalid source returned nil error")
	}
	content, err := repo.GetContent(ctx, "diagram")
	if err != nil {
		t.Fatalf("GetContent() error = %v", err)
	}
	if content != before {
		t.Fatalf("invalid replacement changed content from %q to %q", before, content)
	}

	if err := repo.ReplaceContent(ctx, "diagram", "x -> y"); err != nil {
		t.Fatalf("ReplaceContent() valid source error = %v", err)
	}
	content, err = repo.GetContent(ctx, "diagram")
	if err != nil {
		t.Fatalf("GetContent() error = %v", err)
	}
	if content != "x -> y\n" {
		t.Fatalf("GetContent() = %q, want %q", content, "x -> y\\n")
	}
}

func TestFormatContent(t *testing.T) {
	repo := NewD2OracleRepository().(*D2OracleRepository)
	formatted, err := repo.FormatContent(context.Background(), "a:{b}")
	if err != nil {
		t.Fatalf("FormatContent() error = %v", err)
	}
	if formatted != "a: {b}\n" {
		t.Fatalf("FormatContent() = %q", formatted)
	}
	if _, err := repo.FormatContent(context.Background(), "a: {"); err == nil {
		t.Fatal("FormatContent() invalid source returned nil error")
	}
}

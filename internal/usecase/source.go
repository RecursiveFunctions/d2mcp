package usecase

import (
	"context"
	"fmt"

	"github.com/i2y/d2mcp/internal/domain/entity"
	"github.com/i2y/d2mcp/internal/domain/repository"
)

func (uc *DiagramUseCase) sourceRepository() (repository.SourceRepository, error) {
	repo, ok := uc.repo.(repository.SourceRepository)
	if !ok {
		return nil, fmt.Errorf("source authoring is not supported by this repository")
	}
	return repo, nil
}

// GetSource returns the canonical source for a stored diagram.
func (uc *DiagramUseCase) GetSource(ctx context.Context, diagramID string) (string, error) {
	if diagramID == "" {
		return "", &ValidationError{Message: "diagram ID is required"}
	}
	return uc.repo.GetContent(ctx, diagramID)
}

// UpdateSource validates and atomically replaces a stored diagram's source.
func (uc *DiagramUseCase) UpdateSource(ctx context.Context, diagramID, content string) (*entity.DiagramValidationResult, error) {
	if diagramID == "" {
		return nil, &ValidationError{Message: "diagram ID is required"}
	}
	repo, err := uc.sourceRepository()
	if err != nil {
		return nil, err
	}
	validation, err := repo.Validate(ctx, content)
	if err != nil || !validation.Valid {
		return validation, err
	}
	if err := repo.ReplaceContent(ctx, diagramID, content); err != nil {
		return nil, err
	}
	return validation, nil
}

// FormatSource formats valid source without changing stored diagrams.
func (uc *DiagramUseCase) FormatSource(ctx context.Context, content string) (string, error) {
	repo, err := uc.sourceRepository()
	if err != nil {
		return "", err
	}
	return repo.FormatContent(ctx, content)
}

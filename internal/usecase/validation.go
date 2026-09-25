package usecase

import (
	"context"
	"fmt"
	"sort"

	"github.com/i2y/d2mcp/internal/domain/entity"
)

const maxRepairPasses = 4

type repairCandidate struct {
	content string
	repairs []string
}

// ValidateDiagram validates raw source or the current source of a stored diagram.
func (uc *DiagramUseCase) ValidateDiagram(ctx context.Context, content, diagramID *string) (*entity.DiagramValidationResult, error) {
	if (content == nil) == (diagramID == nil) {
		return nil, &ValidationError{Message: "exactly one of content or diagram_id is required"}
	}

	source := ""
	if content != nil {
		source = *content
	} else {
		if *diagramID == "" {
			return nil, &ValidationError{Message: "diagram_id cannot be empty"}
		}
		var err error
		source, err = uc.repo.GetContent(ctx, *diagramID)
		if err != nil {
			return nil, err
		}
	}

	result, err := uc.repo.Validate(ctx, source)
	if err != nil || result.Valid {
		return result, err
	}

	repairedContent, repairs, repaired, err := uc.tryRepair(ctx, source, result.Diagnostics)
	if err != nil {
		return nil, err
	}
	if repaired {
		result.RepairAvailable = true
		result.RepairedContent = repairedContent
		result.Repairs = repairs
	}
	return result, nil
}

func (uc *DiagramUseCase) tryRepair(ctx context.Context, source string, diagnostics []entity.DiagramDiagnostic) (string, []string, bool, error) {
	current := source
	currentDiagnostics := diagnostics
	var repairs []string

	for range maxRepairPasses {
		candidate, ok := buildRepairCandidate(current, currentDiagnostics)
		if !ok || candidate.content == current {
			return "", nil, false, nil
		}

		validation, err := uc.repo.Validate(ctx, candidate.content)
		if err != nil {
			return "", nil, false, err
		}
		repairs = append(repairs, candidate.repairs...)
		if validation.Valid {
			return candidate.content, repairs, true, nil
		}

		current = candidate.content
		currentDiagnostics = validation.Diagnostics
	}

	return "", nil, false, nil
}

func buildRepairCandidate(content string, diagnostics []entity.DiagramDiagnostic) (repairCandidate, bool) {
	if candidate, ok := appendMissingTerminators(content, diagnostics); ok {
		return candidate, true
	}
	return removeUnexpectedMapTerminators(content, diagnostics)
}

func appendMissingTerminators(content string, diagnostics []entity.DiagramDiagnostic) (repairCandidate, bool) {
	terminators := map[string]string{
		"maps must be terminated with }":                  "}",
		"edge groups must be terminated with )":           ")",
		`double quoted strings must be terminated with "`: `"`,
		"single quoted strings must be terminated with '": "'",
		"arrays must be terminated with ]":                "]",
		"substitutions must be terminated by }":           "}",
		`block comments must be terminated with """`:      `"""`,
	}

	if len(diagnostics) == 0 {
		return repairCandidate{}, false
	}

	candidate := repairCandidate{content: content}
	for _, diagnostic := range diagnostics {
		terminator, ok := terminators[diagnostic.Message]
		if !ok {
			return repairCandidate{}, false
		}
		candidate.content += terminator
		candidate.repairs = append(candidate.repairs, fmt.Sprintf("appended missing %q terminator", terminator))
	}
	return candidate, true
}

func removeUnexpectedMapTerminators(content string, diagnostics []entity.DiagramDiagnostic) (repairCandidate, bool) {
	const message = "unexpected map termination character } in file map"
	if len(diagnostics) == 0 {
		return repairCandidate{}, false
	}

	ranges := make([]entity.SourceRange, 0, len(diagnostics))
	for _, diagnostic := range diagnostics {
		if diagnostic.Message != message || diagnostic.Range == nil {
			return repairCandidate{}, false
		}
		ranges = append(ranges, *diagnostic.Range)
	}

	sort.Slice(ranges, func(i, j int) bool {
		return ranges[i].Start.Byte > ranges[j].Start.Byte
	})

	candidate := []byte(content)
	for _, sourceRange := range ranges {
		start := sourceRange.Start.Byte
		end := sourceRange.End.Byte
		if start < 0 || end > len(candidate) || end != start+1 || candidate[start] != '}' {
			return repairCandidate{}, false
		}
		candidate = append(candidate[:start], candidate[end:]...)
	}

	return repairCandidate{
		content: string(candidate),
		repairs: []string{fmt.Sprintf("removed %d unexpected top-level map terminator(s)", len(ranges))},
	}, true
}

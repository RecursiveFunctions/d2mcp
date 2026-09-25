package entity

// SourcePosition identifies a position in D2 source.
type SourcePosition struct {
	Line   int `json:"line"`
	Column int `json:"column"`
	Byte   int `json:"-"`
}

// SourceRange identifies a half-open range in D2 source.
type SourceRange struct {
	Start SourcePosition `json:"start"`
	End   SourcePosition `json:"end"`
}

// DiagramDiagnostic describes a D2 validation problem.
type DiagramDiagnostic struct {
	Severity string       `json:"severity"`
	Message  string       `json:"message"`
	Range    *SourceRange `json:"range,omitempty"`
}

// DiagramValidationResult describes source validity and any conservative repair.
type DiagramValidationResult struct {
	Valid           bool                `json:"valid"`
	Diagnostics     []DiagramDiagnostic `json:"diagnostics"`
	RepairAvailable bool                `json:"repair_available"`
	RepairedContent string              `json:"repaired_content,omitempty"`
	Repairs         []string            `json:"repairs,omitempty"`
}

package validate

import (
	"fmt"

	"opcodeoracle/internal/analysis"
	"opcodeoracle/internal/state"
)

// Issue represents a validation warning.
type Issue struct {
	Address uint16
	Message string
}

func (i Issue) String() string {
	return fmt.Sprintf("$%04X: %s", i.Address, i.Message)
}

// Validate checks for potential issues in the state.
// It requires InstructionBoundaries from analysis to check addresses.
func Validate(s *state.State, boundaries analysis.InstructionBoundaries) []Issue {
	var issues []Issue

	// Check annotations
	for addr := range s.Annotations.All() {
		if boundaries.IsInstructionDataAt(addr) {
			issues = append(issues, Issue{
				Address: addr,
				Message: "annotation on instruction data (operand byte)",
			})
		}
	}

	// Check headlines
	for addr := range s.Headlines.All() {
		if boundaries.IsInstructionDataAt(addr) {
			issues = append(issues, Issue{
				Address: addr,
				Message: "headline on instruction data (operand byte)",
			})
		}
	}

	return issues
}

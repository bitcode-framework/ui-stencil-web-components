package runtime

import (
	"time"

	"github.com/bitcode-framework/go-json/lang"
)

type Limits struct {
	MaxDepth          int
	MaxSteps          int
	MaxLoopIterations int
	MaxNodes          int
	MaxVariables      int
	MaxVariableSize   int
	MaxOutputSize     int
	Timeout           time.Duration
}

func DefaultLimits() Limits {
	d := lang.DefaultLimits()
	return Limits{
		MaxDepth:          d.MaxDepth,
		MaxSteps:          d.MaxSteps,
		MaxLoopIterations: d.MaxLoopIterations,
		MaxNodes:          d.MaxNodes,
		MaxVariables:      d.MaxVariables,
		MaxVariableSize:   d.MaxVariableSize,
		MaxOutputSize:     d.MaxOutputSize,
		Timeout:           d.Timeout,
	}
}

func HardLimits() Limits {
	h := lang.HardLimits()
	return Limits{
		MaxDepth:          h.MaxDepth,
		MaxSteps:          h.MaxSteps,
		MaxLoopIterations: h.MaxLoopIterations,
		MaxNodes:          h.MaxNodes,
		MaxVariables:      h.MaxVariables,
		MaxVariableSize:   h.MaxVariableSize,
		MaxOutputSize:     h.MaxOutputSize,
		Timeout:           h.Timeout,
	}
}

func (l Limits) ToResolved() lang.ResolvedLimits {
	return lang.ResolvedLimits{
		MaxDepth:          l.MaxDepth,
		MaxSteps:          l.MaxSteps,
		MaxLoopIterations: l.MaxLoopIterations,
		MaxNodes:          l.MaxNodes,
		MaxVariables:      l.MaxVariables,
		MaxVariableSize:   l.MaxVariableSize,
		MaxOutputSize:     l.MaxOutputSize,
		Timeout:           l.Timeout,
	}
}

// Resolve merges engine-level and program-level limits, picking the most
// restrictive (minimum) non-zero value for each field and clamping the
// result to hard limits.
func Resolve(engine, program Limits) Limits {
	result := engine
	if program.MaxDepth > 0 && program.MaxDepth < result.MaxDepth {
		result.MaxDepth = program.MaxDepth
	}
	if program.MaxSteps > 0 && program.MaxSteps < result.MaxSteps {
		result.MaxSteps = program.MaxSteps
	}
	if program.MaxLoopIterations > 0 && program.MaxLoopIterations < result.MaxLoopIterations {
		result.MaxLoopIterations = program.MaxLoopIterations
	}
	if program.MaxNodes > 0 && program.MaxNodes < result.MaxNodes {
		result.MaxNodes = program.MaxNodes
	}
	if program.MaxVariables > 0 && program.MaxVariables < result.MaxVariables {
		result.MaxVariables = program.MaxVariables
	}
	if program.MaxVariableSize > 0 && program.MaxVariableSize < result.MaxVariableSize {
		result.MaxVariableSize = program.MaxVariableSize
	}
	if program.MaxOutputSize > 0 && program.MaxOutputSize < result.MaxOutputSize {
		result.MaxOutputSize = program.MaxOutputSize
	}
	if program.Timeout > 0 && program.Timeout < result.Timeout {
		result.Timeout = program.Timeout
	}
	hard := HardLimits()
	if result.MaxDepth > hard.MaxDepth {
		result.MaxDepth = hard.MaxDepth
	}
	if result.MaxSteps > hard.MaxSteps {
		result.MaxSteps = hard.MaxSteps
	}
	if result.MaxLoopIterations > hard.MaxLoopIterations {
		result.MaxLoopIterations = hard.MaxLoopIterations
	}
	if result.MaxNodes > hard.MaxNodes {
		result.MaxNodes = hard.MaxNodes
	}
	if result.MaxVariables > hard.MaxVariables {
		result.MaxVariables = hard.MaxVariables
	}
	if result.MaxVariableSize > hard.MaxVariableSize {
		result.MaxVariableSize = hard.MaxVariableSize
	}
	if result.MaxOutputSize > hard.MaxOutputSize {
		result.MaxOutputSize = hard.MaxOutputSize
	}
	if result.Timeout > hard.Timeout {
		result.Timeout = hard.Timeout
	}
	return result
}

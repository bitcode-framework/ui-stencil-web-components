package steps

import (
	"context"
	"fmt"
	"strings"

	"github.com/bitcode-framework/bitcode/internal/compiler/parser"
	"github.com/bitcode-framework/bitcode/internal/runtime/executor"
)

type StepExecutor interface {
	ExecuteSteps(ctx context.Context, execCtx *executor.Context, steps []parser.StepDefinition) error
}

type IfHandler struct {
	Executor StepExecutor
}

func (h *IfHandler) Execute(ctx context.Context, execCtx *executor.Context, step parser.StepDefinition) error {
	result := executor.EvaluateCondition(step.Condition, execCtx)

	if len(step.ThenSteps) > 0 || len(step.ElseSteps) > 0 {
		var steps []parser.StepDefinition
		if result {
			steps = step.ThenSteps
		} else {
			steps = step.ElseSteps
		}
		if len(steps) > 0 {
			return h.Executor.ExecuteSteps(ctx, execCtx, steps)
		}
		return nil
	}

	target := step.Else
	if result {
		target = step.Then
	}
	if target != "" {
		execCtx.Variables["_goto"] = target
	}
	return nil
}

type SwitchHandler struct {
	Executor StepExecutor
}

func (h *SwitchHandler) Execute(ctx context.Context, execCtx *executor.Context, step parser.StepDefinition) error {
	val := resolveVariable(step.Field, execCtx)
	valStr := fmt.Sprintf("%v", val)

	if len(step.CaseSteps) > 0 {
		steps, ok := step.CaseSteps[valStr]
		if !ok {
			steps = step.CaseSteps["default"]
		}
		if len(steps) > 0 {
			return h.Executor.ExecuteSteps(ctx, execCtx, steps)
		}
		return nil
	}

	target, ok := step.Cases[valStr]
	if !ok {
		target = step.Cases["default"]
	}
	if target != "" {
		execCtx.Variables["_goto"] = target
	}
	return nil
}

type LoopHandler struct {
	Executor StepExecutor
}

func (h *LoopHandler) Execute(ctx context.Context, execCtx *executor.Context, step parser.StepDefinition) error {
	items := resolveVariable(step.Over, execCtx)
	list, ok := items.([]any)
	if !ok {
		if mapList, ok2 := items.([]map[string]any); ok2 {
			list = make([]any, len(mapList))
			for i, m := range mapList {
				list[i] = m
			}
		} else {
			return fmt.Errorf("loop over %q is not iterable", step.Over)
		}
	}

	for i, item := range list {
		execCtx.Variables["_index"] = i
		execCtx.Variables["_item"] = item

		if len(step.Steps) > 0 {
			if err := h.Executor.ExecuteSteps(ctx, execCtx, step.Steps); err != nil {
				return fmt.Errorf("loop iteration %d failed: %w", i, err)
			}
		}
	}

	return nil
}

func resolveVariable(name string, execCtx *executor.Context) any {
	name = strings.TrimPrefix(name, "{{")
	name = strings.TrimSuffix(name, "}}")
	name = strings.TrimSpace(name)

	if strings.HasPrefix(name, "input.") {
		key := strings.TrimPrefix(name, "input.")
		return execCtx.Input[key]
	}
	if val, ok := execCtx.Variables[name]; ok {
		return val
	}
	return nil
}

func interpolate(s string, execCtx *executor.Context) string {
	return executor.InterpolateString(s, execCtx)
}

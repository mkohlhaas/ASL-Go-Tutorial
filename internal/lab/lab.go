// Package lab contains the small amount of glue code shared by the tutorial
// steps: loading a definition, registering Go handlers, running the state
// loop, and printing a readable execution trace.
//
// The lab runner is intentionally tiny. Version 1.2.22 of the library has a
// quirk in its non-persistent top-level runner (StateMachine.Execute): every
// state receives the original workflow input instead of the previous state's
// output. The persistent runner (used in step 7) does thread data forward.
// To keep steps 2-6 database-free, Run below implements the standard ASL loop
// directly on top of the library's parsed state objects.
package lab

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/hussainpithawala/state-machine-amz-go/pkg/execution"
	"github.com/hussainpithawala/state-machine-amz-go/pkg/executor"
	"github.com/hussainpithawala/state-machine-amz-go/pkg/statemachine"
	"github.com/hussainpithawala/state-machine-amz-go/pkg/types"
)

// Handler is the signature every Go task handler must have.
type Handler func(context.Context, any) (any, error)

// Run loads an ASL workflow from disk, registers the Go task handlers, and
// runs the workflow once with the supplied input.
func Run(ctx context.Context, definitionPath string, input any, handlers map[string]Handler) (*execution.Execution, error) {
	definition, err := os.ReadFile(definitionPath)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", definitionPath, err)
	}

	sm, err := statemachine.New(definition, false) // false => YAML
	if err != nil {
		return nil, fmt.Errorf("parse state machine: %w", err)
	}

	exec := executor.NewBaseExecutor()
	for resource, handler := range handlers {
		exec.RegisterGoFunction(resource, handler)
	}
	ctx = context.WithValue(ctx, types.ExecutionContextKey, executor.NewExecutionContextAdapter(exec))

	return runLoop(ctx, sm, input)
}

// runLoop is the standard ASL execution loop: execute the current state, make
// its output the next state's input, repeat until an End state or an error.
func runLoop(ctx context.Context, sm *statemachine.StateMachine, input any) (*execution.Execution, error) {
	execCtx := execution.NewContext("lab-run", sm.StartAt, input)
	execCtx.Status = "RUNNING"
	current := sm.StartAt

	for {
		select {
		case <-ctx.Done():
			execCtx.Status = "CANCELLED"
			execCtx.Error = ctx.Err()
			execCtx.EndTime = time.Now()
			return execCtx, ctx.Err()
		default:
		}

		state, err := sm.GetState(current)
		if err != nil {
			execCtx.Status = "FAILED"
			execCtx.Error = err
			execCtx.EndTime = time.Now()
			return execCtx, err
		}

		execCtx.CurrentState = current
		output, next, stateErr := state.Execute(ctx, execCtx.Input)

		execCtx.AddStateHistory(current, execCtx.Input, output, statusFor(stateErr))
		if stateErr != nil {
			execCtx.History[len(execCtx.History)-1].Error = stateErr
			execCtx.Status = "FAILED"
			execCtx.Error = stateErr
			execCtx.Output = output
			execCtx.EndTime = time.Now()
			return execCtx, stateErr
		}

		execCtx.Input = output

		if state.IsEnd() {
			execCtx.Status = "SUCCEEDED"
			execCtx.Output = output
			execCtx.EndTime = time.Now()
			return execCtx, nil
		}
		if next == nil || *next == "" {
			err := fmt.Errorf("non-terminal state %s did not provide next state", current)
			execCtx.Status = "FAILED"
			execCtx.Error = err
			execCtx.EndTime = time.Now()
			return execCtx, err
		}
		current = *next
	}
}

func statusFor(err error) string {
	if err != nil {
		return "FAILED"
	}
	return "SUCCEEDED"
}

// PrintExecution prints the final result plus a compact state-by-state trace.
func PrintExecution(result *execution.Execution, err error) {
	fmt.Println()
	if result == nil {
		fmt.Printf("execution error: %v\n", err)
		return
	}

	status := result.Status
	if err != nil {
		status = fmt.Sprintf("%s (%v)", result.Status, err)
	}
	fmt.Printf("status:   %s\n", status)
	fmt.Printf("duration: %s\n", result.EndTime.Sub(result.StartTime).Round(time.Millisecond))
	PrintJSON("output", result.Output)

	fmt.Println("\nstate trace (each line shows the input and output of one state):")
	for _, h := range result.History {
		fmt.Printf("  [%d] %-18s %-8s\n", h.SequenceNumber, h.StateName, h.Status)
		fmt.Printf("        in:  %s\n", compact(h.Input))
		if h.Error != nil {
			fmt.Printf("        err: %v\n", h.Error)
		} else {
			fmt.Printf("        out: %s\n", compact(h.Output))
		}
	}
}

// PrintJSON pretty-prints any value.
func PrintJSON(label string, value any) {
	encoded, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		fmt.Printf("%s: %v\n", label, value)
		return
	}
	fmt.Printf("%s: %s\n", label, encoded)
}

func compact(value any) string {
	encoded, err := json.Marshal(value)
	if err != nil {
		return fmt.Sprintf("%v", value)
	}
	const max = 220
	if len(encoded) > max {
		return string(encoded[:max]) + "..."
	}
	return string(encoded)
}

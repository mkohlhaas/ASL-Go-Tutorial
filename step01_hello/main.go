// Step 1: the smallest useful state machine.
//
// Run from the repository root:
//
//	go run ./step01_hello
//
// This step deliberately uses the library's own Execute method. Because the
// workflow has exactly one state, it sidesteps a bug in v1.2.22's in-memory
// runner (see the README); later steps use the tutorial's small lab runner.
//
// What to notice:
//   - A workflow is just an ASL document (YAML here).
//   - Task states call Go functions registered under a "Resource" name.
//   - The executor reaches the Task state through context.Context.
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/hussainpithawala/state-machine-amz-go/pkg/executor"
	"github.com/hussainpithawala/state-machine-amz-go/pkg/statemachine"
	"github.com/hussainpithawala/state-machine-amz-go/pkg/types"

	"asl-tutorial/internal/lab"
)

func main() {
	ctx := context.Background()

	// 1. Parse and validate the ASL definition.
	//    The second argument is isJSON=false, because our file is YAML.
	definition, err := os.ReadFile("workflows/01_hello.yaml")
	if err != nil {
		fail(err)
	}
	sm, err := statemachine.New(definition, false)
	if err != nil {
		fail(err)
	}
	fmt.Printf("loaded %q (start state: %s)\n", sm.Comment, sm.StartAt)

	// 2. Register the Go function that implements the Task's Resource.
	exec := executor.NewBaseExecutor()
	exec.RegisterGoFunction("demo:greeting", func(_ context.Context, input any) (any, error) {
		// "Parameters" already ran: the handler receives {"name": ..., "language": ...}.
		params := input.(map[string]any)
		fmt.Printf("handler %q received: %v\n", "demo:greeting", params)

		return map[string]any{
			"message": fmt.Sprintf("Hello, %s!", params["name"]),
		}, nil
	})

	// 3. Put the executor on the context. The Task state looks it up through
	//    types.ExecutionContextKey. Without this line a Task silently behaves
	//    like a Pass state and returns its input unchanged.
	ctx = context.WithValue(ctx, types.ExecutionContextKey, executor.NewExecutionContextAdapter(exec))

	// 4. Execute. The input is an ordinary Go value; it is normally a
	//    map[string]any after JSON/YAML parsing.
	result, err := sm.Execute(ctx, map[string]any{
		"name": "Ada",
		"role": "engineer",
	}, statemachine.WithExecutionName("step01-hello"))

	lab.PrintExecution(result, err)
	if err != nil {
		os.Exit(1)
	}
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}

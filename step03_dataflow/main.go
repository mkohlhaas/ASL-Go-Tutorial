// Step 3: shaping data between states.
//
//	go run ./step03_dataflow
//
// ASL gives every state four data knobs:
//
//	InputPath     filter what this state sees
//	Parameters    build a custom input for a Task (see the README gotcha)
//	ResultSelector pick/rename fields from a Task's result
//	ResultPath    merge the result back into the state's input
//	OutputPath    filter what the next state sees
//
// The trace makes the effect of each one visible.
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"asl-tutorial/internal/lab"
)

func main() {
	ctx := context.Background()

	handlers := map[string]lab.Handler{
		"demo:summary": func(_ context.Context, input any) (any, error) {
			// InputPath already reduced the document to the customer object.
			customer := input.(map[string]any)
			fmt.Printf("demo:summary received: %v\n", customer)

			return map[string]any{
				"name":  customer["name"],
				"tier":  customer["tier"],
				"score": 0.92, // not selected by ResultSelector
				"history": map[string]any{
					"orders": 12,
				},
			}, nil
		},
		"demo:metadata": func(_ context.Context, input any) (any, error) {
			// OutputPath on the previous state means we only see the summary.
			fmt.Printf("demo:metadata received: %v\n", input)
			return map[string]any{
				"schema":      3,
				"generatedAt": time.Now().UTC().Format(time.RFC3339),
			}, nil
		},
	}

	input := map[string]any{
		"orderId":  "ORD-9",
		"amount":   42.5,
		"currency": "USD",
		"customer": map[string]any{
			"name": "Ada",
			"tier": "gold",
			"history": map[string]any{
				"orders": 12,
			},
		},
	}

	result, err := lab.Run(ctx, "workflows/03_dataflow.yaml", input, handlers)
	lab.PrintExecution(result, err)
	if err != nil {
		os.Exit(1)
	}

	fmt.Println(`
Try it:
  - remove OutputPath from BuildSummary and watch the full customer/order
    document survive into AddMetadata;
  - change ResultSelector to include "$.score" or add a literal field.`)
}

// Step 2: branching with Choice.
//
//	go run ./step02_choice
//
// A Choice state is the ASL "if/else". It does not call a handler; it
// evaluates rules against the current state input and returns the name of the
// next state. The first matching rule wins, otherwise Default is used.
package main

import (
	"context"
	"fmt"
	"os"

	"asl-tutorial/internal/lab"
)

func main() {
	ctx := context.Background()

	handlers := map[string]lab.Handler{
		"demo:route": func(_ context.Context, input any) (any, error) {
			data := input.(map[string]any)

			// The Choice state already decided which branch we are in, so the
			// handler only needs to do the branch's work.
			route := "standard"
			if data["tier"] == "gold" {
				route = "premium"
			}

			return map[string]any{
				"route":   route,
				"orderId": data["orderId"],
				"queue":   route + "-queue",
			}, nil
		},
	}

	scenarios := []struct {
		name  string
		input map[string]any
	}{
		{
			name:  "gold customer over 200 -> Premium (compound And)",
			input: map[string]any{"orderId": "ORD-1", "total": 249.0, "tier": "gold"},
		},
		{
			name:  "silver customer over 200 -> Standard",
			input: map[string]any{"orderId": "ORD-2", "total": 249.0, "tier": "silver"},
		},
		{
			name:  "small order -> Small (Pass branch)",
			input: map[string]any{"orderId": "ORD-3", "total": 19.99, "tier": "gold"},
		},
		{
			name:  "no rule matches -> Default",
			input: map[string]any{"orderId": "ORD-4", "total": 120.0, "tier": "silver"},
		},
	}

	for i, scenario := range scenarios {
		fmt.Printf("\n=== scenario %d: %s ===\n", i+1, scenario.name)
		result, err := lab.Run(ctx, "workflows/02_choice.yaml", scenario.input, handlers)
		lab.PrintExecution(result, err)
		if err != nil {
			os.Exit(1)
		}
	}

	fmt.Println(`
Try it: add a "NumericGreaterThan: 500" rule with a FraudReview branch, or
change the order of the rules and see which branch wins.`)
}

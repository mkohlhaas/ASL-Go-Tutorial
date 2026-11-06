// Step 4: retries, catch blocks, and terminal failures.
//
//	go run ./step04_errors
//
// The library matches `ErrorEquals` against the handler's error *text*:
//
//	return nil, errors.New("ServiceUnavailable")
//
// matches `ErrorEquals: ["ServiceUnavailable"]`, and `States.ALL` catches
// everything. Wrapped/typed errors do not match a bare name.
package main

import (
	"context"
	"errors"
	"fmt"
	"os"

	"asl-tutorial/internal/lab"
)

func main() {
	ctx := context.Background()

	// --- Scenario 1: transient failure, retried until it succeeds ----------
	fmt.Println("=== scenario 1: flaky service is retried ===")
	attempts := 0
	handlers := map[string]lab.Handler{
		"demo:charge": func(_ context.Context, input any) (any, error) {
			attempts++
			fmt.Printf("  charge attempt %d\n", attempts)
			if attempts < 3 {
				return nil, errors.New("ServiceUnavailable")
			}
			return map[string]any{"transactionId": "TX-123", "attempts": attempts}, nil
		},
		"demo:receipt": func(_ context.Context, input any) (any, error) {
			return map[string]any{"sent": true}, nil
		},
	}
	result, err := lab.Run(ctx, "workflows/04_retry_catch.yaml", map[string]any{
		"orderId": "ORD-1", "amount": 100.0, "mode": "flaky",
	}, handlers)
	lab.PrintExecution(result, err)
	if err != nil {
		os.Exit(1)
	}

	// --- Scenario 2: a business error is caught and handled ----------------
	fmt.Println("\n=== scenario 2: declined payment is caught ===")
	handlers = map[string]lab.Handler{
		"demo:charge": func(_ context.Context, input any) (any, error) {
			return nil, errors.New("PaymentDeclined")
		},
		"demo:receipt": func(_ context.Context, input any) (any, error) {
			return map[string]any{"sent": true}, nil
		},
	}
	result, err = lab.Run(ctx, "workflows/04_retry_catch.yaml", map[string]any{
		"orderId": "ORD-2", "amount": 900.0, "mode": "declined",
	}, handlers)
	lab.PrintExecution(result, err)
	if err != nil {
		os.Exit(1)
	}

	// --- Scenario 3: no Catch -> the execution fails -----------------------
	fmt.Println("\n=== scenario 3: uncaught error fails the execution ===")
	handlers = map[string]lab.Handler{
		"demo:reserve": func(_ context.Context, input any) (any, error) {
			return nil, errors.New("InventoryUnavailable")
		},
	}
	result, err = lab.Run(ctx, "workflows/04_uncaught.yaml", map[string]any{
		"orderId": "ORD-3",
	}, handlers)
	lab.PrintExecution(result, err)
	// We EXPECT this one to fail, so a non-nil err is not a tutorial failure.

	fmt.Println(`
Try it:
  - change ErrorEquals to ["States.ALL"] and see PaymentDeclined get caught;
  - make the handler return fmt.Errorf("ServiceUnavailable: %w", err) and
    notice that it no longer matches the exact string "ServiceUnavailable".`)
}

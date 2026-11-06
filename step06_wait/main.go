// Step 6: Wait states and context cancellation.
//
//	go run ./step06_wait
//
// A Wait state parks the execution. It is free (no worker is busy while
// waiting). The wait is interruptible through the context.Context, which is
// exactly what you want for shutdowns and request timeouts.
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"asl-tutorial/internal/lab"
)

func main() {
	handlers := map[string]lab.Handler{
		"demo:finish": func(_ context.Context, input any) (any, error) {
			return map[string]any{"ok": true}, nil
		},
	}

	// --- Scenario 1: wait one second and finish ----------------------------
	fmt.Println("=== scenario 1: wait 1s (from SecondsPath) ===")
	start := time.Now()
	result, err := lab.Run(context.Background(), "workflows/06_wait.yaml", map[string]any{
		"orderId":      "ORD-6",
		"delaySeconds": 1,
	}, handlers)
	lab.PrintExecution(result, err)
	fmt.Printf("wall clock: %s\n", time.Since(start).Round(time.Millisecond))
	if err != nil {
		os.Exit(1)
	}

	// --- Scenario 2: cancel while waiting ----------------------------------
	fmt.Println("\n=== scenario 2: a 300ms timeout cancels a 2s wait ===")
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()

	result, err = lab.Run(ctx, "workflows/06_wait.yaml", map[string]any{
		"orderId":      "ORD-7",
		"delaySeconds": 2,
	}, handlers)
	lab.PrintExecution(result, err)
	// We EXPECT this one to be cancelled, so a non-nil err is fine here.

	fmt.Println(`
Try it:
  - use Seconds: 2 instead of SecondsPath and compare the trace;
  - set Timestamp to a moment in the past. A past timestamp does not wait.`)
}

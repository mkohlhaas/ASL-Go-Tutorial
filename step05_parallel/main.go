// Step 5: concurrent branches with Parallel.
//
//	go run ./step05_parallel
//
// Every branch starts from the same input and runs concurrently. The Parallel
// state waits for all branches and returns their results as an array, in
// branch order. If one branch fails, the whole Parallel state fails.
package main

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"asl-tutorial/internal/lab"
)

func main() {
	ctx := context.Background()

	var mu sync.Mutex
	started := time.Now()

	// Each handler sleeps for the same amount of time. If the branches are
	// truly concurrent the whole workflow takes ~400ms, not ~1200ms.
	branch := func(name string, work time.Duration) lab.Handler {
		return func(_ context.Context, input any) (any, error) {
			mu.Lock()
			offset := time.Since(started).Round(10 * time.Millisecond)
			mu.Unlock()
			fmt.Printf("  %-9s started at +%s\n", name, offset)

			time.Sleep(work)
			return map[string]any{
				"ok":    true,
				"after": work.String(),
			}, nil
		}
	}

	handlers := map[string]lab.Handler{
		"demo:price":    branch("price", 400*time.Millisecond),
		"demo:stock":    branch("stock", 400*time.Millisecond),
		"demo:shipping": branch("shipping", 400*time.Millisecond),
	}

	result, err := lab.Run(ctx, "workflows/05_parallel.yaml", map[string]any{
		"orderId": "ORD-5",
		"amount":  149.0,
	}, handlers)
	lab.PrintExecution(result, err)
	if err != nil {
		os.Exit(1)
	}

	fmt.Println(`
Try it:
  - change one branch to return an error and observe that the whole Parallel
    state fails;
  - add a fourth branch. The results array always follows branch order, not
    completion order.`)
}

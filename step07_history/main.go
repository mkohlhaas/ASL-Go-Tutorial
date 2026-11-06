// Step 7: the production path - PostgreSQL persistence.
//
// Start the database first:
//
//	docker compose up -d
//	go run ./step07_history
//
// Unlike steps 2-6, this uses the library's real persistent engine:
//
//	repository.NewPersistenceManager(...)
//	persistent.New(...)
//	sm.SaveDefinition(ctx)
//	sm.Execute(ctx, input)
//	sm.GetExecutionHistory(ctx, executionID)
//
// Everything survives a process restart: run this program twice and the list
// at the end will contain executions from both runs.
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/hussainpithawala/state-machine-amz-go/pkg/executor"
	"github.com/hussainpithawala/state-machine-amz-go/pkg/repository"
	"github.com/hussainpithawala/state-machine-amz-go/pkg/statemachine"
	"github.com/hussainpithawala/state-machine-amz-go/pkg/statemachine/persistent"
	"github.com/hussainpithawala/state-machine-amz-go/pkg/types"
)

const stateMachineID = "tutorial-step07"

func main() {
	ctx := context.Background()

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://tutorial:tutorial@localhost:5432/tutorial?sslmode=disable"
	}

	// 1. Connect and let the library create/migrate its tables.
	manager, err := repository.NewPersistenceManager(&repository.Config{
		Strategy:      "gorm-postgres",
		ConnectionURL: dsn,
		Options: map[string]any{
			"log_level": "silent",
		},
	})
	if err != nil {
		fail("create persistence manager", err)
	}
	defer manager.Close()

	if err := manager.Initialize(ctx); err != nil {
		fail("initialize database (is Postgres running? try: docker compose up -d)", err)
	}
	fmt.Println("connected to Postgres")

	// 2. Save the definition once. SaveDefinition is an upsert, so running
	//    this program repeatedly is safe.
	definition, err := os.ReadFile("workflows/07_persistent.yaml")
	if err != nil {
		fail("read workflow", err)
	}
	sm, err := persistent.New(definition, false, stateMachineID, manager)
	if err != nil {
		fail("create persistent state machine", err)
	}
	if err := sm.SaveDefinition(ctx); err != nil {
		fail("save definition", err)
	}

	// 3. Register the Go handlers. The executor still reaches state execution
	//    through context, exactly as in step 1.
	exec := executor.NewBaseExecutor()
	exec.RegisterGoFunction("demo:validate", func(_ context.Context, input any) (any, error) {
		data := input.(map[string]any)
		return map[string]any{
			"valid": data["amount"].(float64) > 0,
		}, nil
	})
	exec.RegisterGoFunction("demo:store", func(_ context.Context, input any) (any, error) {
		return map[string]any{
			"table": "orders",
			"at":    time.Now().UTC().Format(time.RFC3339),
		}, nil
	})
	ctx = context.WithValue(ctx, types.ExecutionContextKey, executor.NewExecutionContextAdapter(exec))

	// 4. Execute twice with explicit names. Names must be unique per state
	//    machine; without WithExecutionName the library generates one.
	for i := 1; i <= 2; i++ {
		name := fmt.Sprintf("step07-run-%d-%d", time.Now().UnixNano(), i)
		fmt.Printf("\n--- execution %s ---\n", name)

		result, err := sm.Execute(ctx, map[string]any{
			"orderId": fmt.Sprintf("ORD-%d", i),
			"amount":  75.0 * float64(i),
		}, statemachine.WithExecutionName(name))
		if err != nil {
			fail("execute", err)
		}
		fmt.Printf("status: %s\n", result.Status)
		fmt.Printf("output: %v\n", result.Output)

		// History is read back from the database, not from memory.
		history, err := sm.GetExecutionHistory(ctx, result.ID)
		if err != nil {
			fail("get history", err)
		}
		fmt.Println("stored history:")
		for _, h := range history {
			fmt.Printf("  #%d %-9s %-8s input=%v output=%v\n",
				h.SequenceNumber, h.StateName, h.Status, h.Input, h.Output)
		}
	}

	// 5. The definition itself is in the database too: reload it by ID,
	//    without reading the YAML file again.
	loaded, err := persistent.NewFromDefnId(ctx, stateMachineID, manager)
	if err != nil {
		fail("load definition from database", err)
	}
	fmt.Printf("\nreloaded state machine %q (start state: %s)\n", stateMachineID, loaded.GetStartAt())

	// 6. Query executions with filtering and pagination.
	filter := &repository.ExecutionFilter{
		Status: repository.StatusSucceeded,
		Limit:  10,
	}
	executions, err := sm.ListExecutions(ctx, filter)
	if err != nil {
		fail("list executions", err)
	}
	count, err := sm.CountExecutions(ctx, &repository.ExecutionFilter{Status: repository.StatusSucceeded})
	if err != nil {
		fail("count executions", err)
	}

	fmt.Printf("\n%d succeeded executions for %s (showing %d):\n", count, stateMachineID, len(executions))
	for _, e := range executions {
		fmt.Printf("  %-38s %-28s %s\n", e.ExecutionID, e.Name, e.Status)
	}

	fmt.Println(`
Run this program again: the rows above persist across process restarts.
Reset the database with: docker compose down -v`)
}

func fail(what string, err error) {
	fmt.Fprintf(os.Stderr, "%s: %v\n", what, err)
	os.Exit(1)
}

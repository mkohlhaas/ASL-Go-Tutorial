DOCKER ?= docker
COMPOSE := $(DOCKER) compose

.DEFAULT_GOAL := demo

.PHONY: help demo demo-all step01 step02 step03 step04 step05 step06 step07 \
        db-up db-wait db-down db-reset fmt vet test check

help:
	@echo "Targets:"
	@echo "  demo       run steps 1-6 (no database)"
	@echo "  demo-all   run steps 1-6, then step 7 (needs Postgres)"
	@echo "  step01..06 run one step"
	@echo "  step07     start/await Postgres, then run the persistence demo"
	@echo "  db-up      start Postgres"
	@echo "  db-down    stop Postgres (keep data)"
	@echo "  db-reset   stop Postgres and delete its volume"
	@echo "  fmt        gofmt all packages"
	@echo "  vet        go vet all packages"
	@echo "  test       go test all packages"
	@echo "  check      fmt + vet + test"
	@echo ""
	@echo "If Docker needs root: make db-up DOCKER=\"sudo docker\""

demo: step01 step02 step03 step04 step05 step06

demo-all: demo step07

step01:
	@echo "==> Step 1: one Task, one Go handler"
	@go run ./step01_hello

step02:
	@echo "==> Step 2: branching with Choice"
	@go run ./step02_choice

step03:
	@echo "==> Step 3: shaping data with paths"
	@go run ./step03_dataflow

step04:
	@echo "==> Step 4: Retry, Catch, and failure"
	@go run ./step04_errors

step05:
	@echo "==> Step 5: concurrent branches with Parallel"
	@go run ./step05_parallel

step06:
	@echo "==> Step 6: Wait and context cancellation"
	@go run ./step06_wait

step07: db-wait
	@echo "==> Step 7: persistence with PostgreSQL"
	@go run ./step07_history

db-up:
	$(COMPOSE) up -d

db-wait: db-up
	@printf "waiting for Postgres"
	@until $(COMPOSE) exec -T postgres pg_isready -U tutorial -d tutorial >/dev/null 2>&1; do \
		printf "."; \
		sleep 1; \
	done; \
	echo " ready"

db-down:
	$(COMPOSE) down

db-reset:
	$(COMPOSE) down -v

fmt:
	go fmt ./...

vet:
	go vet ./...

test:
	go test ./...

check: fmt vet test

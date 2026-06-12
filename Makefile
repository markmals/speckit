# Local mirror of the `go` CI workflow (.github/workflows/ci-go.yml).
# Run `make ci` before pushing so a remote machine never diagnoses what a local
# one could have.
.PHONY: ci build vet test fmt fmt-check

ci: fmt-check build vet test ## everything CI runs, plus a gofmt gate

build: ## compile all packages
	go build ./...

vet: ## go vet
	go vet ./...

test: ## run the test suite
	go test ./...

fmt: ## format the tree in place
	gofmt -w cmd internal

fmt-check: ## fail if anything needs gofmt
	@unformatted=$$(gofmt -l cmd internal); \
	if [ -n "$$unformatted" ]; then \
		echo "gofmt needed (run 'make fmt'):"; echo "$$unformatted"; exit 1; \
	fi

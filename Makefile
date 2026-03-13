BINARY := marp2pptx
MODULE := github.com/pspoerri/marp2pptx

.PHONY: build test test-verbose test-integration lint fmt vet clean run install-hooks eval eval-lint

build:
	go build -o $(BINARY) .

test:
	go test ./...

test-verbose:
	go test -v ./...

test-integration:
	go test -v -run TestIntegration ./...

test-run:
	go test -v -run $(RUN) ./...

lint: vet fmt-check

vet:
	go vet ./...

fmt:
	gofmt -w .

fmt-check:
	@test -z "$$(gofmt -l .)" || (echo "Files need formatting:" && gofmt -l . && exit 1)

clean:
	rm -f $(BINARY)
	rm -f testdata/*.pptx

run: build
	./$(BINARY) $(ARGS)

# Evaluate PPTX output: generate and inspect
# Usage: make eval ARGS=testdata/sample.pptx
#        make eval-all
eval:
	go run ./cmd/pptxeval $(ARGS)

# Lint PPTX output for structural issues
# Usage: make eval-lint ARGS=testdata/sample.pptx
#        make eval-lint-all
eval-lint:
	go run ./cmd/pptxeval -lint $(ARGS)

# Generate and lint all test outputs
eval-all: build
	./$(BINARY) -o testdata/sample.pptx testdata/sample.md
	./$(BINARY) -o testdata/extensions.pptx testdata/extensions.md
	./$(BINARY) -o testdata/mermaid.pptx testdata/mermaid.md
	@echo "--- sample.pptx ---"
	@go run ./cmd/pptxeval testdata/sample.pptx
	@echo "--- extensions.pptx ---"
	@go run ./cmd/pptxeval testdata/extensions.pptx
	@echo "--- mermaid.pptx ---"
	@go run ./cmd/pptxeval testdata/mermaid.pptx

eval-lint-all: build
	./$(BINARY) -o testdata/sample.pptx testdata/sample.md
	./$(BINARY) -o testdata/extensions.pptx testdata/extensions.md
	./$(BINARY) -o testdata/mermaid.pptx testdata/mermaid.md
	go run ./cmd/pptxeval -lint testdata/sample.pptx
	go run ./cmd/pptxeval -lint testdata/extensions.pptx
	go run ./cmd/pptxeval -lint testdata/mermaid.pptx

install-hooks:
	git config core.hooksPath githooks

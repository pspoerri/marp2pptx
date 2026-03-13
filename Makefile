BINARY := marp2pptx
MODULE := github.com/pascal/marp2pptx

.PHONY: build test test-verbose lint fmt vet clean run

build:
	go build -o $(BINARY) .

test:
	go test ./...

test-verbose:
	go test -v ./...

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

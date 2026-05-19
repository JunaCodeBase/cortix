BIN      := cortix
CMD      := ./cmd/cortix
GOFLAGS  :=

.PHONY: build build-windows run test vet clean help

build:
	go build $(GOFLAGS) -o $(BIN) $(CMD)

build-windows:
	go build $(GOFLAGS) -o $(BIN).exe $(CMD)

run:
	go run $(CMD) $(ARGS)

test:
	go test ./...

vet:
	go vet ./...

clean:
	rm -f $(BIN) $(BIN).exe

help:
	@echo "make build          Build for Linux/macOS  → ./cortix"
	@echo "make build-windows  Build for Windows      → cortix.exe"
	@echo "make run ARGS='...' Run without building   → go run ./cmd/cortix ..."
	@echo "make test           Run all tests"
	@echo "make vet            Run go vet"
	@echo "make clean          Remove built binaries"

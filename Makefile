GO ?= go
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "0.1.0-dev")
LDFLAGS := -ldflags="-X 'main.version=$(VERSION)'"
BINARY := govm

.PHONY: all build test vet lint clean run

all: build

build:
	GOOS=$(GOOS) GOARCH=$(GOARCH) $(GO) build $(LDFLAGS) -o $(BINARY) .

run: build
	./$(BINARY)

test:
	$(GO) test -short -count=1 ./...

test-full:
	$(GO) test -count=1 ./...

vet:
	$(GO) vet ./...

lint:
	golangci-lint run --timeout=5m ./...

clean:
	rm -f $(BINARY) $(BINARY).exe
	go clean ./...

# Cross-compile all release targets
release: clean
	GOOS=linux   GOARCH=amd64 $(GO) build $(LDFLAGS) -o $(BINARY)_$(VERSION)_linux_amd64 .
	GOOS=linux   GOARCH=arm64 $(GO) build $(LDFLAGS) -o $(BINARY)_$(VERSION)_linux_arm64 .
	GOOS=darwin  GOARCH=amd64 $(GO) build $(LDFLAGS) -o $(BINARY)_$(VERSION)_darwin_amd64 .
	GOOS=darwin  GOARCH=arm64 $(GO) build $(LDFLAGS) -o $(BINARY)_$(VERSION)_darwin_arm64 .
	GOOS=windows GOARCH=amd64 $(GO) build $(LDFLAGS) -o $(BINARY)_$(VERSION)_windows_amd64.exe .

# Every folder in cmd/ is an app; they are found automatically.
APPS      := $(notdir $(wildcard cmd/*))
VERSION   ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS   := -s -w -X main.version=$(VERSION)
HOST      ?= batocera.local
REMOTEDIR := /userdata/system/batox-apps

.PHONY: build build-arm64 build-amd64 test vet fmt fmt-check deploy clean

build: build-arm64 build-amd64

build-arm64:
	@mkdir -p build
	@for app in $(APPS); do \
		echo "building $$app (linux/arm64)"; \
		CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -trimpath -ldflags "$(LDFLAGS)" -o build/$$app-linux-arm64 ./cmd/$$app || exit 1; \
	done

build-amd64:
	@mkdir -p build
	@for app in $(APPS); do \
		echo "building $$app (linux/amd64)"; \
		CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags "$(LDFLAGS)" -o build/$$app-linux-amd64 ./cmd/$$app || exit 1; \
	done

test:
	go test -race ./...

vet:
	go vet ./...

fmt:
	gofmt -w .

fmt-check:
	@test -z "$$(gofmt -l .)" || (echo "run make fmt:"; gofmt -l .; exit 1)

# Copies the Raspberry Pi builds to the console: make deploy HOST=192.168.1.50
deploy: build-arm64
	ssh root@$(HOST) "mkdir -p $(REMOTEDIR)"
	@for app in $(APPS); do \
		scp build/$$app-linux-arm64 root@$(HOST):$(REMOTEDIR)/$$app || exit 1; \
	done
	@echo "installed in $(REMOTEDIR) on $(HOST)"

clean:
	rm -rf build

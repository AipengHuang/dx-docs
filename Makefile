.PHONY: build build-prod clean run test

BINARY_NAME := dixian-knowledge
MAIN_PATH := ./cmd/server

build:
	go build -o $(BINARY_NAME) $(MAIN_PATH)

build-prod:
	@VERSION="$${VERSION:-$$(cat VERSION 2>/dev/null || echo dev)}"; \
	LDFLAGS="-X 'github.com/Tencent/WeKnora/internal/handler.Version=$$VERSION' -X 'github.com/Tencent/WeKnora/internal/handler.Edition=internal' -X 'github.com/Tencent/WeKnora/internal/handler.CommitID=$${COMMIT_ID:-unknown}' -X 'github.com/Tencent/WeKnora/internal/handler.BuildTime=$${BUILD_TIME:-unknown}' -X 'github.com/Tencent/WeKnora/internal/handler.GoVersion=$${GO_VERSION:-unknown}' -X 'google.golang.org/protobuf/reflect/protoregistry.conflictPolicy=warn'"; \
	go build -ldflags="-w -s $$LDFLAGS" -o $(BINARY_NAME) $(MAIN_PATH)

run: build
	./$(BINARY_NAME)

test:
	go test ./...

clean:
	rm -f $(BINARY_NAME)

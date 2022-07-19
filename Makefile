ifndef config
  config=release
endif

ifeq ($(config),debug)
	GO_TEST_FLAGS += -v
endif

PROJECTS := mktree docs
TARGET := ./bin/mktree
VERSION := $(shell cat VERSION)

.PHONY: all bump-patch bump-minor bump-major clean docs docs-preview format release run test help

all: $(PROJECTS)

bump-patch:
	@echo "=== Bumping patch number ==="
	go run ./cmd/release-tool bump-version -patch VERSION

bump-minor:
	@echo "=== Bumping minor number ==="
	go run ./cmd/release-tool bump-version -minor VERSION

bump-major:
	@echo "=== Bumping major number ==="
	go run ./cmd/release-tool bump-version -major VERSION

clean:
	@echo "==== Removing mktree ===="
	rm -rf ./bin/*

docs: mktree
	@echo "=== Regenerating documentation ==="
	tools/docs.sh -b

docs-preview: mktree
	@echo "=== Serving documentation ==="
	tools/docs.sh -s

format: 
	@echo "==== Formatting mktree source code ===="
	gofmt -w .

mktree: format clean
	@echo "==== Building mktree ($(config)) ===="
	go build -o $(TARGET) ./cmd/mktree

release: mktree
	@echo "==== Building mktree ($(VERSION)) ===="
	env GOOS=darwin GOARCH=amd64 go build -o bin/mktree-$(VERSION)-darwin-amd64 ./cmd/mktree
	env GOOS=linux  GOARCH=amd64 go build -o bin/mktree-$(VERSION)-linux-amd64  ./cmd/mktree
	chmod -R a+x ./bin/

run: mktree
	@echo "==== Running mktree ($(config)) ===="
	$(TARGET)

test: mktree
	@echo "==== Testing mktree (test) ===="
	go test $(GO_TEST_FLAGS) ./...

help:
	@echo "Usage: make [target]"
	@echo ""
	@echo "TARGETS:"
	@echo "   all (default)"
	@echo "   bump-patch"
	@echo "   bump-minor"
	@echo "   bump-major"
	@echo "   clean"
	@echo "   docs"
	@echo "   docs-preview"
	@echo "   run"
	@echo "   test"
	@echo ""
	@echo "For more information, see https://github.com/premake/premake-core/wiki"

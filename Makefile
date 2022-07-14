ifndef config
  config=release
endif

ifeq ($(config),debug)
	GO_TEST_FLAGS += -v
endif

PROJECTS := mktree docs
TARGET := ./bin/mktree

.PHONY: all bump-patch clean docs docs-serve format run test help

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
	rm -rf $(TARGET)

docs: mktree
	@echo "=== Regenerating documentation ==="
	tools/docs.sh -b

docs-serve: mktree
	@echo "=== Serving documentation ==="
	tools/docs.sh -s

format: 
	@echo "==== Formatting mktree source code ===="
	gofmt -w .

mktree: format clean
	@echo "==== Building mktree ($(config)) ===="
	go build -o $(TARGET) ./cmd/mktree

run: mktree
	@echo "==== Running mktree ($(config)) ===="
	$(TARGET)

test:
	@echo "==== Testing mktree (test) ===="
	go test $(GO_TEST_FLAGS) ./...

help:
	@echo "Usage: make [target]"
	@echo ""
	@echo "TARGETS:"
	@echo "   all (default)"
	@echo "   clean"
	@echo "   docs"
	@echo "   docs-serve"
	@echo "   run"
	@echo "   test"
	@echo ""
	@echo "For more information, see https://github.com/premake/premake-core/wiki"

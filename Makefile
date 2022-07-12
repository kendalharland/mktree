ifndef config
  config=release
endif

ifeq ($(config),debug)
			 ALL_CFLAGS += -g -DDEBUG_MODE -DDEBUG_TRACE_EXECUTION
else ifeq ($(config),debuggc)
			 ALL_CFLAGS += -g -DDEBUG_STRESS_GC -DDEBUG_LOG_GC
else ifeq ($(config),optimize)
			 ALL_CFLAGS += -DOBA_COMPUTED_GOTO
else ifeq ($(config),test)
			 # Disable stack traces to simplify comparing error output.
			 ALL_CFLAGS += -DDISABLE_STACK_TRACES -DDEBUG_STRESS_GC
else ifneq ($(config),release)
		$(error "invalid configuration $(config)")
endif

PROJECTS := mktree
TARGET := ./bin/mktree

.PHONY: all clean docs format run test help

all: $(PROJECTS)

clean:
	@echo "==== Removing mktree ===="
	rm -rf $(TARGET)

docs:
	@echo "=== Regenerating documentation ==="
	tools/docs.sh -b

docs-serve:
	@echo "=== Serving documentation ==="
	tools/docs.sh -s

format:
	@echo "==== Formatting mktree source code ===="
	gofmt -w ./...

mktree: clean
	@echo "==== Building mktree ($(config)) ===="
	go build -o $(TARGET) ./cmd

run: mktree
	@echo "==== Running mktree ($(config)) ===="
	$(TARGET)

test:
	@echo "==== Testing mktree (test) ===="
	go test -v ./...

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

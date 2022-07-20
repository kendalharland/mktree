PROJECTS := hello
TARGET := hello

INCLUDES += -I ./src/include
ALL_CFLAGS += $(INCLUDES) -o $(TARGET)

.PHONY: all clean format run help

all: $(PROJECTS)

clean:
	@echo "==== Removing hello ===="
	rm -rf $(TARGET)

format:
	@echo "==== Formatting hello source code ===="
	find . -regex '.*\.\(c\|h\)' -exec clang-format -style=file -i {} \;

hello: clean
	@echo "==== Building hello ===="
	$(CC) $(ALL_CFLAGS) ./src/main.c

run: hello
	@echo "==== Running hello ===="
	./hello 

help:
	@echo "Usage: make [target]"
	@echo ""
	@echo "TARGETS:"
	@echo "   all (default)"
	@echo "   clean"
	@echo "   format"
	@echo "   hello"
	@echo "   run"
	@echo ""